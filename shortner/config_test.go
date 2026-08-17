package shortner

import (
	"os"
	"path/filepath"
	"testing"
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

	cfg, err := LoadConfig(path)
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
	cfg := DefaultConfig()
	cfg.Database.Name = "shortener"
	cfg.Database.Username = "app"
	for _, candidate := range []string{
		"javascript://example.com",
		"https://user:pass@example.com",
		"https://example.com/hidden/path",
		"https://example.com/?override=true",
	} {
		cfg.PublicBaseURL = candidate
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() accepted public URL %q", candidate)
		}
	}
}
