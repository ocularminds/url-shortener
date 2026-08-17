// Package web owns HTTP transport and application lifecycle concerns.
package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ocularminds/url-shortener/config"
)

type Application struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

func NewApplication(port int, cfg config.ServerConfig, handler http.Handler) (*Application, error) {
	if handler == nil {
		return nil, errors.New("HTTP handler is required")
	}
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
	return &Application{
		server:          server,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (application *Application) Address() string {
	return application.server.Addr
}

func (application *Application) Run(ctx context.Context) error {
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- application.server.ListenAndServe()
	}()

	select {
	case err := <-errorChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), application.shutdownTimeout)
		defer cancel()
		if err := application.server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		err := <-errorChannel
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
