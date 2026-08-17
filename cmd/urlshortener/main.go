package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ocularminds/url-shortener/config"
	mysqlrepo "github.com/ocularminds/url-shortener/core/repository/mysql"
	"github.com/ocularminds/url-shortener/core/service"
	"github.com/ocularminds/url-shortener/web"
)

func main() {
	if err := run(); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := os.Getenv("URL_SHORTENER_CONFIG")
	if configPath == "" {
		configPath = "configs/config.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := mysqlrepo.New(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer store.Close()

	shortener, err := service.NewURLShortener(
		store,
		service.CryptoSlugGenerator{Length: service.DefaultSlugLength},
	)
	if err != nil {
		return err
	}
	handler, err := web.NewHandler(shortener, cfg.PublicBaseURL, log.Default())
	if err != nil {
		return err
	}
	limiter := web.NewTokenBucketLimiter(cfg.RateLimit)
	app, err := web.NewApplication(cfg.Port, cfg.Server, handler.Routes(limiter))
	if err != nil {
		return err
	}

	log.Printf("listening on %s", app.Address())
	return app.Run(ctx)
}
