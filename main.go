package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ocularminds/url-shortener/shortner"
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
		configPath = "config.json"
	}

	cfg, err := shortner.LoadConfig(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := shortner.NewApplication(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Close()

	log.Printf("listening on %s", app.Address())
	return app.Run(ctx)
}
