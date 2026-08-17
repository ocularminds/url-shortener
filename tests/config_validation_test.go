package tests

import (
	"testing"
	"time"

	appconfig "github.com/ocularminds/url-shortener/config"
)

func validConfig() appconfig.Config {
	cfg := appconfig.Default()
	cfg.PublicBaseURL = "https://sho.rt"
	cfg.Database.Name = "shortener"
	cfg.Database.Username = "app"
	return cfg
}

func TestConfigValidationRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*appconfig.Config)
	}{
		{name: "port below range", mutate: func(cfg *appconfig.Config) { cfg.Port = 0 }},
		{name: "port above range", mutate: func(cfg *appconfig.Config) { cfg.Port = 65536 }},
		{name: "relative public URL", mutate: func(cfg *appconfig.Config) { cfg.PublicBaseURL = "sho.rt" }},
		{name: "invalid public port", mutate: func(cfg *appconfig.Config) { cfg.PublicBaseURL = "https://sho.rt:65536" }},
		{name: "public URL fragment", mutate: func(cfg *appconfig.Config) { cfg.PublicBaseURL = "https://sho.rt/#fragment" }},
		{name: "missing database name", mutate: func(cfg *appconfig.Config) { cfg.Database.Name = "" }},
		{name: "missing database username", mutate: func(cfg *appconfig.Config) { cfg.Database.Username = "" }},
		{name: "missing database host", mutate: func(cfg *appconfig.Config) { cfg.Database.Host = "" }},
		{name: "database port below range", mutate: func(cfg *appconfig.Config) { cfg.Database.Port = 0 }},
		{name: "database port above range", mutate: func(cfg *appconfig.Config) { cfg.Database.Port = 65536 }},
		{name: "unsupported database TLS mode", mutate: func(cfg *appconfig.Config) { cfg.Database.TLSMode = "skip-verify" }},
		{name: "zero open connections", mutate: func(cfg *appconfig.Config) { cfg.Database.MaxOpenConnections = 0 }},
		{name: "negative idle connections", mutate: func(cfg *appconfig.Config) { cfg.Database.MaxIdleConnections = -1 }},
		{name: "too many idle connections", mutate: func(cfg *appconfig.Config) { cfg.Database.MaxIdleConnections = cfg.Database.MaxOpenConnections + 1 }},
		{name: "zero rate", mutate: func(cfg *appconfig.Config) { cfg.RateLimit.RequestsPerMinute = 0 }},
		{name: "zero burst", mutate: func(cfg *appconfig.Config) { cfg.RateLimit.Burst = 0 }},
		{name: "zero max clients", mutate: func(cfg *appconfig.Config) { cfg.RateLimit.MaxClients = 0 }},
		{name: "small header limit", mutate: func(cfg *appconfig.Config) { cfg.Server.MaxHeaderBytes = 4095 }},
		{name: "zero read header timeout", mutate: func(cfg *appconfig.Config) { cfg.Server.ReadHeaderTimeout = 0 }},
		{name: "zero read timeout", mutate: func(cfg *appconfig.Config) { cfg.Server.ReadTimeout = 0 }},
		{name: "zero write timeout", mutate: func(cfg *appconfig.Config) { cfg.Server.WriteTimeout = 0 }},
		{name: "zero idle timeout", mutate: func(cfg *appconfig.Config) { cfg.Server.IdleTimeout = 0 }},
		{name: "zero shutdown timeout", mutate: func(cfg *appconfig.Config) { cfg.Server.ShutdownTimeout = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid configuration")
			}
		})
	}
}

func TestConfigAcceptsLoopbackDatabaseHosts(t *testing.T) {
	for _, host := range []string{"localhost", "LOCALHOST", "127.0.0.1", "::1"} {
		t.Run(host, func(t *testing.T) {
			cfg := validConfig()
			cfg.Database.Host = host
			cfg.Database.TLSMode = ""
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() rejected loopback host %q: %v", host, err)
			}
		})
	}
}

func TestDatabaseAddress(t *testing.T) {
	cfg := appconfig.DatabaseConfig{Host: "::1", Port: 3306}
	if got := cfg.Address(); got != "[::1]:3306" {
		t.Fatalf("Address() = %q", got)
	}
}

func TestDefaultConfigUsesPositiveDurations(t *testing.T) {
	cfg := appconfig.Default()
	for name, duration := range map[string]time.Duration{
		"connect":  cfg.Database.ConnectTimeout,
		"read":     cfg.Database.ReadTimeout,
		"write":    cfg.Database.WriteTimeout,
		"lifetime": cfg.Database.ConnectionLifetime,
		"idle":     cfg.Database.ConnectionIdleTime,
	} {
		if duration <= 0 {
			t.Errorf("%s duration = %v", name, duration)
		}
	}
}
