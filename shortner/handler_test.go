package shortner

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(t *testing.T) (http.Handler, *memoryRepository) {
	t.Helper()
	repository := newMemoryRepository()
	service := newTestService(t, repository, "AbCd1234", "EfGh5678")
	handler, err := NewHandler(service, "https://sho.rt", "../views", log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	return handler.Routes(nil), repository
}

func performJSONRequest(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "http://attacker.example/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestHandlerCreatesLinkWithoutTrustingHostHeader(t *testing.T) {
	handler, _ := newTestHandler(t)
	response := performJSONRequest(handler, `{"url":"https://example.com/long"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	var payload linkResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ShortLink != "https://sho.rt/AbCd1234" {
		t.Fatalf("shortLink = %q, want configured origin", payload.ShortLink)
	}

	response = performJSONRequest(handler, `{"url":"https://example.com/long"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("repeat status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRejectsMalformedRequests(t *testing.T) {
	handler, _ := newTestHandler(t)
	tests := []struct {
		name        string
		contentType string
		body        string
		status      int
	}{
		{name: "wrong content type", contentType: "text/plain", body: `{}`, status: http.StatusUnsupportedMediaType},
		{name: "unknown field", contentType: "application/json", body: `{"url":"https://example.com","admin":true}`, status: http.StatusBadRequest},
		{name: "multiple values", contentType: "application/json", body: `{"url":"https://example.com"}{}`, status: http.StatusBadRequest},
		{name: "unsafe scheme", contentType: "application/json", body: `{"url":"javascript://example.com"}`, status: http.StatusBadRequest},
		{name: "oversized", contentType: "application/json", body: `{"url":"https://example.com/` + strings.Repeat("a", maxRequestBodyBytes) + `"}`, status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func TestHandlerRedirectsAndReturnsNotFound(t *testing.T) {
	handler, _ := newTestHandler(t)
	created := performJSONRequest(handler, `{"url":"https://example.com/target"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d", created.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/AbCd1234", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "https://example.com/target" {
		t.Fatalf("redirect = (%d, %q), want target", response.Code, response.Header().Get("Location"))
	}

	request = httptest.NewRequest(http.MethodGet, "/NotFound", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestHandlerAddsSecurityHeaders(t *testing.T) {
	handler, _ := newTestHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/missing1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	for _, header := range []string{
		"Content-Security-Policy",
		"Permissions-Policy",
		"Referrer-Policy",
		"X-Content-Type-Options",
		"X-Frame-Options",
	} {
		if response.Header().Get(header) == "" {
			t.Errorf("security header %s is missing", header)
		}
	}
}

func TestCreateRouteIsRateLimited(t *testing.T) {
	repository := newMemoryRepository()
	service := newTestService(t, repository, "AbCd1234", "EfGh5678")
	handler, err := NewHandler(service, "https://sho.rt", "../views", log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	limiter := NewTokenBucketLimiter(RateLimitConfig{RequestsPerMinute: 1, Burst: 1, MaxClients: 10})
	router := handler.Routes(limiter)

	first := performJSONRequest(router, `{"url":"https://example.com/one"}`)
	second := performJSONRequest(router, `{"url":"https://example.com/two"}`)
	if first.Code != http.StatusCreated || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = (%d, %d), want (%d, %d)", first.Code, second.Code, http.StatusCreated, http.StatusTooManyRequests)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("rate-limited response is missing Retry-After")
	}
}
