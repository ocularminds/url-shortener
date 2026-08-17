package tests

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ocularminds/url-shortener/config"
	"github.com/ocularminds/url-shortener/web"
)

func TestApplicationConstructorAndAddress(t *testing.T) {
	cfg := config.Default().Server
	if _, err := web.NewApplication(8080, cfg, nil); err == nil {
		t.Fatal("NewApplication() accepted a nil handler")
	}
	application, err := web.NewApplication(9090, cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if application.Address() != ":9090" {
		t.Fatalf("Address() = %q", application.Address())
	}
	if err := application.Serve(context.Background(), nil); err == nil {
		t.Fatal("Serve() accepted a nil listener")
	}
}

func TestApplicationServesAndShutsDownGracefully(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	application, err := web.NewApplication(0, config.Default().Server, handler)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- application.Serve(ctx, listener) }()

	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", response.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() shutdown error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not shut down")
	}
}

func TestApplicationServeReturnsListenerError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	application, err := web.NewApplication(0, config.Default().Server, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Serve(context.Background(), listener); err == nil {
		t.Fatal("Serve() swallowed a listener error")
	}
}

func TestApplicationRunPaths(t *testing.T) {
	t.Run("cancelled context", func(t *testing.T) {
		application, err := web.NewApplication(0, config.Default().Server, http.NotFoundHandler())
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := application.Run(ctx); err != nil {
			t.Fatalf("Run() with cancelled context = %v", err)
		}
	})

	t.Run("occupied port", func(t *testing.T) {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		_, rawPort, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			t.Fatal(err)
		}
		application, err := web.NewApplication(port, config.Default().Server, http.NotFoundHandler())
		if err != nil {
			t.Fatal(err)
		}
		if err := application.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "listen") {
			t.Fatalf("Run() error = %v, want bind failure", err)
		}
	})
}

func TestApplicationReportsShutdownTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		response.WriteHeader(http.StatusNoContent)
	})
	cfg := config.Default().Server
	cfg.ShutdownTimeout = time.Nanosecond
	application, err := web.NewApplication(0, cfg, handler)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- application.Serve(ctx, listener) }()
	requestDone := make(chan struct{})
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
		}
		close(requestDone)
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "graceful shutdown") {
			t.Fatalf("Serve() error = %v, want shutdown timeout", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not report shutdown timeout")
	}
	close(release)
	<-requestDone
}
