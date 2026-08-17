package shortner

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
)

const maxRequestBodyBytes = 4 << 10

type Handler struct {
	service       LinkService
	publicBaseURL *url.URL
	viewsDir      string
	logger        *log.Logger
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type linkResponse struct {
	Code      int    `json:"Code"`
	Message   string `json:"Message"`
	Link      string `json:"Link,omitempty"`
	ShortLink string `json:"ShortLink,omitempty"`
}

func NewHandler(service LinkService, publicBaseURL string, viewsDir string, logger *log.Logger) (*Handler, error) {
	if service == nil {
		return nil, errors.New("link service is required")
	}
	baseURL, err := url.Parse(publicBaseURL)
	if err != nil || baseURL.Host == "" {
		return nil, errors.New("valid public base URL is required")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{
		service:       service,
		publicBaseURL: baseURL,
		viewsDir:      viewsDir,
		logger:        logger,
	}, nil
}

func (handler *Handler) Routes(limiter *TokenBucketLimiter) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", handler.home)
	router.HandleFunc("POST /{$}", handler.create)
	router.HandleFunc("GET /app.js", handler.javascript)
	router.HandleFunc("GET /style.css", handler.stylesheet)
	router.HandleFunc("GET /{slug}", handler.redirect)

	var result http.Handler = router
	if limiter != nil {
		result = limitCreateRequests(result, limiter)
	}
	return handler.securityHeaders(result)
}

func (handler *Handler) home(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(response, request, filepath.Join(handler.viewsDir, "index.html"))
}

func (handler *Handler) javascript(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(response, request, filepath.Join(handler.viewsDir, "app.js"))
}

func (handler *Handler) stylesheet(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/css; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(response, request, filepath.Join(handler.viewsDir, "style.css"))
}

func (handler *Handler) create(response http.ResponseWriter, request *http.Request) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		handler.writeError(response, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input createLinkRequest
	if err := decoder.Decode(&input); err != nil {
		handler.writeError(response, http.StatusBadRequest, "Request body must contain a valid URL")
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		handler.writeError(response, http.StatusBadRequest, "Request body must contain one JSON object")
		return
	}

	link, created, err := handler.service.Create(request.Context(), input.URL)
	if err != nil {
		if errors.Is(err, ErrInvalidURL) {
			handler.writeError(response, http.StatusBadRequest, "Provide a valid HTTP or HTTPS URL")
			return
		}
		handler.logger.Printf("create short link: %v", err)
		handler.writeError(response, http.StatusInternalServerError, "Unable to create short link")
		return
	}

	status := http.StatusOK
	message := "Short URL already exists"
	if created {
		status = http.StatusCreated
		message = "Short URL generated"
	}
	shortURL := handler.publicBaseURL.JoinPath(link.Shortened).String()
	handler.writeJSON(response, status, linkResponse{
		Code:      status,
		Message:   message,
		Link:      link.Shortened,
		ShortLink: shortURL,
	})
}

func (handler *Handler) redirect(response http.ResponseWriter, request *http.Request) {
	link, err := handler.service.Resolve(request.Context(), request.PathValue("slug"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			handler.writeError(response, http.StatusNotFound, "Short URL was not found or has expired")
			return
		}
		handler.logger.Printf("resolve short link: %v", err)
		handler.writeError(response, http.StatusInternalServerError, "Unable to resolve short link")
		return
	}
	http.Redirect(response, request, link.Original, http.StatusFound)
}

func (handler *Handler) writeError(response http.ResponseWriter, status int, message string) {
	handler.writeJSON(response, status, linkResponse{Code: status, Message: message})
}

func (handler *Handler) writeJSON(response http.ResponseWriter, status int, value linkResponse) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		handler.logger.Printf("write JSON response: %v", err)
	}
}

func (handler *Handler) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'")
		response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		response.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		response.Header().Set("Origin-Agent-Cluster", "?1")
		response.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		if request.TLS != nil || handler.publicBaseURL.Scheme == "https" {
			response.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(response, request)
	})
}
