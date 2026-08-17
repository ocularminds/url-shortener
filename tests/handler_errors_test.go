package tests

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ocularminds/url-shortener/core/models"
	"github.com/ocularminds/url-shortener/core/service"
	"github.com/ocularminds/url-shortener/web"
)

type stubLinkService struct {
	create  func(context.Context, string) (models.ShortLink, bool, error)
	resolve func(context.Context, string) (models.ShortLink, error)
}

func (stub stubLinkService) Create(ctx context.Context, original string) (models.ShortLink, bool, error) {
	return stub.create(ctx, original)
}

func (stub stubLinkService) Resolve(ctx context.Context, slug string) (models.ShortLink, error) {
	return stub.resolve(ctx, slug)
}

type failingResponseWriter struct {
	header http.Header
	status int
}

func (writer *failingResponseWriter) Header() http.Header    { return writer.header }
func (writer *failingResponseWriter) WriteHeader(status int) { writer.status = status }
func (writer *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func completeStub() stubLinkService {
	return stubLinkService{
		create: func(_ context.Context, original string) (models.ShortLink, bool, error) {
			return models.ShortLink{Shortened: "AbCd1234", Original: original}, true, nil
		},
		resolve: func(_ context.Context, _ string) (models.ShortLink, error) {
			return models.ShortLink{Original: "https://example.com"}, nil
		},
	}
}

func TestHandlerConstructorValidation(t *testing.T) {
	if _, err := web.NewHandler(nil, "https://sho.rt", nil); err == nil {
		t.Fatal("NewHandler() accepted a nil service")
	}
	for _, baseURL := range []string{"not-a-url", "://invalid"} {
		if _, err := web.NewHandler(completeStub(), baseURL, nil); err == nil {
			t.Fatalf("NewHandler() accepted base URL %q", baseURL)
		}
	}
	if _, err := web.NewHandler(completeStub(), "http://sho.rt", nil); err != nil {
		t.Fatalf("NewHandler() rejected nil logger: %v", err)
	}
}

func TestHandlerReportsServiceFailures(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		body    string
		service stubLinkService
	}{
		{name: "create", method: http.MethodPost, body: `{"url":"https://example.com"}`, service: stubLinkService{
			create: func(context.Context, string) (models.ShortLink, bool, error) {
				return models.ShortLink{}, false, errors.New("database unavailable")
			},
			resolve: completeStub().resolve,
		}},
		{name: "resolve", method: http.MethodGet, service: stubLinkService{
			create: completeStub().create,
			resolve: func(context.Context, string) (models.ShortLink, error) {
				return models.ShortLink{}, errors.New("database unavailable")
			},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			handler, err := web.NewHandler(test.service, "https://sho.rt", log.New(&logs, "", 0))
			if err != nil {
				t.Fatal(err)
			}
			path := "/AbCd1234"
			if test.method == http.MethodPost {
				path = "/"
			}
			request := httptest.NewRequest(test.method, path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.Routes(nil).ServeHTTP(response, request)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", response.Code)
			}
			if !strings.Contains(logs.String(), "database unavailable") {
				t.Fatalf("dependency failure was not logged: %q", logs.String())
			}
		})
	}
}

func TestHandlerLogsResponseWriteFailure(t *testing.T) {
	var logs bytes.Buffer
	handler, err := web.NewHandler(completeStub(), "https://sho.rt", log.New(&logs, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := &failingResponseWriter{header: make(http.Header)}
	handler.Routes(nil).ServeHTTP(response, request)
	if response.status != http.StatusCreated || !strings.Contains(logs.String(), "write JSON response") {
		t.Fatalf("status/logs = (%d, %q)", response.status, logs.String())
	}
}

func TestHandlerSecurityHeadersFollowRequestTransport(t *testing.T) {
	handler, err := web.NewHandler(completeStub(), "http://sho.rt", nil)
	if err != nil {
		t.Fatal(err)
	}
	router := handler.Routes(nil)

	httpRequest := httptest.NewRequest(http.MethodGet, "http://sho.rt/missing1", nil)
	httpResponse := httptest.NewRecorder()
	router.ServeHTTP(httpResponse, httpRequest)
	if httpResponse.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("plain HTTP response unexpectedly included HSTS")
	}

	httpsRequest := httptest.NewRequest(http.MethodGet, "https://sho.rt/missing1", nil)
	httpsResponse := httptest.NewRecorder()
	router.ServeHTTP(httpsResponse, httpsRequest)
	if httpsResponse.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("TLS response did not include HSTS")
	}
}

func TestHandlerReturnsInvalidURLFromService(t *testing.T) {
	stub := completeStub()
	stub.create = func(context.Context, string) (models.ShortLink, bool, error) {
		return models.ShortLink{}, false, service.ErrInvalidURL
	}
	handler, err := web.NewHandler(stub, "https://sho.rt", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := performJSONRequest(handler.Routes(nil), `{"url":"https://example.com"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
