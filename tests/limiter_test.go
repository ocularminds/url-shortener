package tests

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ocularminds/url-shortener/config"
	"github.com/ocularminds/url-shortener/web"
)

func TestTokenBucketLimiterRefillsDeterministically(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	limiter := web.NewTokenBucketLimiter(
		config.RateLimit{RequestsPerMinute: 60, Burst: 2, MaxClients: 10},
		web.WithLimiterClock(func() time.Time { return now }),
	)
	for attempt := 0; attempt < 2; attempt++ {
		if allowed, wait := limiter.Allow("client"); !allowed || wait != 0 {
			t.Fatalf("attempt %d = (%v, %v), want allowed", attempt, allowed, wait)
		}
	}
	if allowed, wait := limiter.Allow("client"); allowed || wait != time.Second {
		t.Fatalf("empty bucket = (%v, %v), want one-second wait", allowed, wait)
	}
	now = now.Add(500 * time.Millisecond)
	if allowed, wait := limiter.Allow("client"); allowed || wait != 500*time.Millisecond {
		t.Fatalf("half-refilled bucket = (%v, %v)", allowed, wait)
	}
	now = now.Add(500 * time.Millisecond)
	if allowed, wait := limiter.Allow("client"); !allowed || wait != 0 {
		t.Fatalf("refilled bucket = (%v, %v), want allowed", allowed, wait)
	}
}

func TestTokenBucketLimiterBoundsClientMemory(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	limiter := web.NewTokenBucketLimiter(
		config.RateLimit{RequestsPerMinute: 60, Burst: 1, MaxClients: 1},
		web.WithLimiterClock(func() time.Time { return now }),
	)
	if allowed, _ := limiter.Allow("first"); !allowed {
		t.Fatal("first client was rejected")
	}
	if allowed, wait := limiter.Allow("second"); allowed || wait != time.Minute {
		t.Fatalf("second client = (%v, %v), want capacity rejection", allowed, wait)
	}
	now = now.Add(2 * time.Second)
	if allowed, wait := limiter.Allow("second"); !allowed || wait != 0 {
		t.Fatalf("second client after eviction = (%v, %v)", allowed, wait)
	}
	defaultClock := web.NewTokenBucketLimiter(
		config.RateLimit{RequestsPerMinute: 60, Burst: 1, MaxClients: 1},
		web.WithLimiterClock(nil),
	)
	if allowed, _ := defaultClock.Allow("client"); !allowed {
		t.Fatal("nil clock option disabled the limiter")
	}
}

func TestRateLimiterNormalizesAndFallsBackFromRemoteAddress(t *testing.T) {
	store := newMemoryRepository()
	shortener := newTestService(t, store, "AbCd1234", "EfGh5678")
	handler, err := web.NewHandler(shortener, "https://sho.rt", log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		addresses  [2]string
		wantSecond int
	}{
		{name: "host and ports", addresses: [2]string{"192.0.2.10:1000", "192.0.2.10:2000"}, wantSecond: http.StatusTooManyRequests},
		{name: "empty address", addresses: [2]string{"", ""}, wantSecond: http.StatusTooManyRequests},
		{name: "unparseable addresses", addresses: [2]string{"peer-one", "peer-two"}, wantSecond: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limiter := web.NewTokenBucketLimiter(config.RateLimit{RequestsPerMinute: 1, Burst: 1, MaxClients: 10})
			router := handler.Routes(limiter)
			for index, remoteAddress := range test.addresses {
				request := httptest.NewRequest(http.MethodPost, "http://sho.rt/", strings.NewReader(`{}`))
				request.RemoteAddr = remoteAddress
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if index == 1 && response.Code != test.wantSecond {
					t.Fatalf("second status = %d, want %d", response.Code, test.wantSecond)
				}
			}
		})
	}
}
