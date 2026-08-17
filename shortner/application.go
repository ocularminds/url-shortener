package shortner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type repositoryCloser interface {
	Close() error
}

type Application struct {
	server          *http.Server
	repository      repositoryCloser
	shutdownTimeout time.Duration
}

func NewApplication(ctx context.Context, cfg Config) (*Application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	repository, err := NewMySQLRepository(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	app, err := newApplication(cfg, repository, repository)
	if err != nil {
		repository.Close()
		return nil, err
	}
	return app, nil
}

func newApplication(cfg Config, repository LinkRepository, closer repositoryCloser) (*Application, error) {
	service, err := NewURLShortener(repository, CryptoSlugGenerator{Length: DefaultSlugLength})
	if err != nil {
		return nil, err
	}
	handler, err := NewHandler(service, cfg.PublicBaseURL, cfg.Server.ViewsDirectory, log.Default())
	if err != nil {
		return nil, err
	}
	limiter := NewTokenBucketLimiter(cfg.RateLimit)
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           handler.Routes(limiter),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}
	return &Application{
		server:          server,
		repository:      closer,
		shutdownTimeout: cfg.Server.ShutdownTimeout,
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

func (application *Application) Close() error {
	if application.repository == nil {
		return nil
	}
	return application.repository.Close()
}
