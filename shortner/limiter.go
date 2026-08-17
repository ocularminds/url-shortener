package shortner

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type clientBucket struct {
	tokens   float64
	lastSeen time.Time
}

// TokenBucketLimiter limits create requests per direct peer. Proxy headers are
// deliberately ignored unless a trusted proxy layer normalizes RemoteAddr.
type TokenBucketLimiter struct {
	mu         sync.Mutex
	clients    map[string]clientBucket
	rate       float64
	burst      float64
	maxClients int
	now        func() time.Time
}

func NewTokenBucketLimiter(config RateLimitConfig) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		clients:    make(map[string]clientBucket),
		rate:       float64(config.RequestsPerMinute) / 60,
		burst:      float64(config.Burst),
		maxClients: config.MaxClients,
		now:        time.Now,
	}
}

func (limiter *TokenBucketLimiter) Allow(key string) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	bucket, exists := limiter.clients[key]
	if !exists {
		if len(limiter.clients) >= limiter.maxClients {
			limiter.evictStale(now)
			if len(limiter.clients) >= limiter.maxClients {
				return false, time.Minute
			}
		}
		bucket = clientBucket{tokens: limiter.burst, lastSeen: now}
	}

	elapsed := now.Sub(bucket.lastSeen).Seconds()
	bucket.tokens = math.Min(limiter.burst, bucket.tokens+elapsed*limiter.rate)
	bucket.lastSeen = now
	if bucket.tokens < 1 {
		limiter.clients[key] = bucket
		wait := time.Duration(math.Ceil((1-bucket.tokens)/limiter.rate*1000)) * time.Millisecond
		return false, wait
	}
	bucket.tokens--
	limiter.clients[key] = bucket
	return true, 0
}

func (limiter *TokenBucketLimiter) evictStale(now time.Time) {
	staleAfter := time.Duration(math.Ceil(limiter.burst/limiter.rate)) * time.Second
	for key, bucket := range limiter.clients {
		if now.Sub(bucket.lastSeen) > staleAfter {
			delete(limiter.clients, key)
		}
	}
}

func limitCreateRequests(next http.Handler, limiter *TokenBucketLimiter) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.URL.Path == "/" {
			allowed, retryAfter := limiter.Allow(clientAddress(request.RemoteAddr))
			if !allowed {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				response.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
				response.Header().Set("Content-Type", "application/json; charset=utf-8")
				response.Header().Set("Cache-Control", "no-store")
				response.WriteHeader(http.StatusTooManyRequests)
				_, _ = response.Write([]byte(`{"Code":429,"Message":"Rate limit exceeded"}` + "\n"))
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}

func clientAddress(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	if remoteAddress == "" {
		return "unknown"
	}
	return remoteAddress
}
