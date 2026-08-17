package tests

import (
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/ocularminds/url-shortener/config"
)

func TestLoadConfigAppliesEnvironmentSecrets(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	contents := `{
		"port": 8080,
		"public_base_url": "https://sho.rt",
		"database": {"name":"shortener","host":"127.0.0.1","port":3306,"username":"app"},
		"server": {},
		"rate_limit": {}
	}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("URL_SHORTENER_DB_PASSWORD", "environment-secret")

	cfg, err := appconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Password != "environment-secret" {
		t.Fatal("database password environment override was not applied")
	}
	if cfg.Server.ReadHeaderTimeout == 0 || cfg.Database.MaxOpenConnections == 0 {
		t.Fatal("safe defaults were not applied")
	}
}

func TestConfigRejectsHostilePublicURL(t *testing.T) {
	cfg := appconfig.Default()
	cfg.Database.Name = "shortener"
	cfg.Database.Username = "app"
	for _, candidate := range []string{
		"javascript://example.com",
		"https://user:pass@example.com",
		"https://example.com/hidden/path",
		"https://example.com/?override=true",
		"https://example.com:999999",
	} {
		cfg.PublicBaseURL = candidate
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() accepted public URL %q", candidate)
		}
	}
}

func TestConfigRequiresTLSForRemoteDatabase(t *testing.T) {
	cfg := appconfig.Default()
	cfg.PublicBaseURL = "https://sho.rt"
	cfg.Database.Name = "shortener"
	cfg.Database.Username = "app"
	cfg.Database.Host = "database.example.com"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted a remote database without TLS")
	}
	cfg.Database.TLSMode = "true"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() rejected remote database TLS: %v", err)
	}
}
