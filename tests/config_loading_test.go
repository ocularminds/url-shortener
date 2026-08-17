package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/ocularminds/url-shortener/config"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigRetainsDefaultsAndBuildsPublicURL(t *testing.T) {
	path := writeConfigFile(t, `{
		"port": 9090,
		"database": {"name":"shortener","username":"app","max_idle_connections":0}
	}`)
	cfg, err := appconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicBaseURL != "http://localhost:9090" {
		t.Fatalf("PublicBaseURL = %q, want dynamic localhost URL", cfg.PublicBaseURL)
	}
	if cfg.Database.MaxIdleConnections != 0 {
		t.Fatalf("MaxIdleConnections = %d, want explicit zero", cfg.Database.MaxIdleConnections)
	}
	if cfg.Database.MaxOpenConnections != 25 || cfg.Server.MaxHeaderBytes != 1<<20 || cfg.RateLimit.Burst != 10 {
		t.Fatalf("defaults were not retained: %+v", cfg)
	}
}

func TestLoadConfigAppliesEveryEnvironmentOverride(t *testing.T) {
	path := writeConfigFile(t, `{"database":{"name":"file-name","username":"file-user"}}`)
	overrides := map[string]string{
		"URL_SHORTENER_PORT":                    "9443",
		"URL_SHORTENER_PUBLIC_BASE_URL":         "https://sho.rt",
		"URL_SHORTENER_DB_NAME":                 "environment-name",
		"URL_SHORTENER_DB_USERNAME":             "environment-user",
		"URL_SHORTENER_DB_PASSWORD":             "environment-password",
		"URL_SHORTENER_DB_HOST":                 "database.example.com",
		"URL_SHORTENER_DB_PORT":                 "3307",
		"URL_SHORTENER_DB_TLS_MODE":             "true",
		"URL_SHORTENER_DB_MAX_OPEN_CONNECTIONS": "40",
		"URL_SHORTENER_DB_MAX_IDLE_CONNECTIONS": "20",
		"URL_SHORTENER_RATE_PER_MINUTE":         "120",
		"URL_SHORTENER_RATE_BURST":              "15",
		"URL_SHORTENER_RATE_MAX_CLIENTS":        "500",
	}
	for name, value := range overrides {
		t.Setenv(name, value)
	}
	cfg, err := appconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9443 || cfg.PublicBaseURL != "https://sho.rt" {
		t.Fatalf("server overrides were not applied: %+v", cfg)
	}
	wantDatabase := appconfig.DatabaseConfig{
		Name:               "environment-name",
		Username:           "environment-user",
		Password:           "environment-password",
		Host:               "database.example.com",
		Port:               3307,
		TLSMode:            "true",
		ConnectTimeout:     cfg.Database.ConnectTimeout,
		ReadTimeout:        cfg.Database.ReadTimeout,
		WriteTimeout:       cfg.Database.WriteTimeout,
		MaxOpenConnections: 40,
		MaxIdleConnections: 20,
		ConnectionLifetime: cfg.Database.ConnectionLifetime,
		ConnectionIdleTime: cfg.Database.ConnectionIdleTime,
	}
	if cfg.Database != wantDatabase {
		t.Fatalf("database overrides = %+v, want %+v", cfg.Database, wantDatabase)
	}
	if cfg.RateLimit != (appconfig.RateLimit{RequestsPerMinute: 120, Burst: 15, MaxClients: 500}) {
		t.Fatalf("rate limit overrides = %+v", cfg.RateLimit)
	}
}

func TestLoadConfigRejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		envName  string
		envValue string
	}{
		{name: "malformed JSON", contents: `{`},
		{name: "unknown field", contents: `{"database":{"name":"shortener","username":"app"},"unexpected":true}`},
		{name: "multiple values", contents: `{"database":{"name":"shortener","username":"app"}} {}`},
		{name: "invalid trailing JSON", contents: `{"database":{"name":"shortener","username":"app"}} !`},
		{name: "invalid values", contents: `{}`},
		{name: "invalid integer override", contents: `{"database":{"name":"shortener","username":"app"}}`, envName: "URL_SHORTENER_PORT", envValue: "not-a-number"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfigFile(t, test.contents)
			if test.envName != "" {
				t.Setenv(test.envName, test.envValue)
			}
			if _, err := appconfig.Load(path); err == nil {
				t.Fatal("Load() accepted invalid configuration")
			}
		})
	}

	missing := filepath.Join(t.TempDir(), "missing.json")
	if _, err := appconfig.Load(missing); err == nil || !strings.Contains(err.Error(), "open config") {
		t.Fatalf("Load(missing) error = %v", err)
	}
}
