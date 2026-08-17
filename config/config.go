// Package config loads and validates runtime configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxConfigBytes = 64 << 10

type Config struct {
	Port          int            `json:"port"`
	PublicBaseURL string         `json:"public_base_url"`
	Database      DatabaseConfig `json:"database"`
	Server        ServerConfig   `json:"server"`
	RateLimit     RateLimit      `json:"rate_limit"`
}

type DatabaseConfig struct {
	Name               string        `json:"name"`
	Username           string        `json:"username"`
	Password           string        `json:"-"`
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	TLSMode            string        `json:"tls_mode"`
	ConnectTimeout     time.Duration `json:"-"`
	ReadTimeout        time.Duration `json:"-"`
	WriteTimeout       time.Duration `json:"-"`
	MaxOpenConnections int           `json:"max_open_connections"`
	MaxIdleConnections int           `json:"max_idle_connections"`
	ConnectionLifetime time.Duration `json:"-"`
	ConnectionIdleTime time.Duration `json:"-"`
}

type ServerConfig struct {
	ReadHeaderTimeout time.Duration `json:"-"`
	ReadTimeout       time.Duration `json:"-"`
	WriteTimeout      time.Duration `json:"-"`
	IdleTimeout       time.Duration `json:"-"`
	ShutdownTimeout   time.Duration `json:"-"`
	MaxHeaderBytes    int           `json:"max_header_bytes"`
}

type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst             int `json:"burst"`
	MaxClients        int `json:"max_clients"`
}

func Default() Config {
	return Config{
		Port: 8080,
		Database: DatabaseConfig{
			Host:               "127.0.0.1",
			Port:               3306,
			ConnectTimeout:     3 * time.Second,
			ReadTimeout:        3 * time.Second,
			WriteTimeout:       3 * time.Second,
			MaxOpenConnections: 25,
			MaxIdleConnections: 10,
			ConnectionLifetime: 5 * time.Minute,
			ConnectionIdleTime: time.Minute,
		},
		Server: ServerConfig{
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   10 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
		RateLimit: RateLimit{
			RequestsPerMinute: 60,
			Burst:             10,
			MaxClients:        10_000,
		},
	}
}

// Load reads non-secret JSON settings and applies URL_SHORTENER_* environment
// overrides. The database password is intentionally environment-only.
func Load(path string) (Config, error) {
	cfg := Default()
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, maxConfigBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func applyEnvironment(cfg *Config) error {
	stringOverrides := map[string]*string{
		"URL_SHORTENER_PUBLIC_BASE_URL": &cfg.PublicBaseURL,
		"URL_SHORTENER_DB_NAME":         &cfg.Database.Name,
		"URL_SHORTENER_DB_USERNAME":     &cfg.Database.Username,
		"URL_SHORTENER_DB_PASSWORD":     &cfg.Database.Password,
		"URL_SHORTENER_DB_HOST":         &cfg.Database.Host,
		"URL_SHORTENER_DB_TLS_MODE":     &cfg.Database.TLSMode,
	}
	for name, destination := range stringOverrides {
		if value, ok := os.LookupEnv(name); ok {
			*destination = value
		}
	}

	integerOverrides := map[string]*int{
		"URL_SHORTENER_PORT":                    &cfg.Port,
		"URL_SHORTENER_DB_PORT":                 &cfg.Database.Port,
		"URL_SHORTENER_DB_MAX_OPEN_CONNECTIONS": &cfg.Database.MaxOpenConnections,
		"URL_SHORTENER_DB_MAX_IDLE_CONNECTIONS": &cfg.Database.MaxIdleConnections,
		"URL_SHORTENER_RATE_PER_MINUTE":         &cfg.RateLimit.RequestsPerMinute,
		"URL_SHORTENER_RATE_BURST":              &cfg.RateLimit.Burst,
		"URL_SHORTENER_RATE_MAX_CLIENTS":        &cfg.RateLimit.MaxClients,
	}
	for name, destination := range integerOverrides {
		value, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be an integer: %w", name, err)
		}
		*destination = parsed
	}
	return nil
}

func (cfg Config) Validate() error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	base, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.Hostname() == "" {
		return errors.New("public_base_url must be an absolute http or https URL")
	}
	if port := base.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return errors.New("public_base_url contains an invalid port")
		}
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return errors.New("public_base_url must not contain credentials, a path, query, or fragment")
	}
	if cfg.Database.Name == "" || cfg.Database.Username == "" {
		return errors.New("database name and username are required")
	}
	if cfg.Database.Host == "" || cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return errors.New("database host and a valid port are required")
	}
	if cfg.Database.TLSMode != "" && cfg.Database.TLSMode != "true" {
		return errors.New("database tls_mode must be true or omitted for a loopback connection")
	}
	if !isLoopbackHost(cfg.Database.Host) && cfg.Database.TLSMode != "true" {
		return errors.New("database TLS is required for non-loopback connections")
	}
	if cfg.Database.MaxOpenConnections < 1 || cfg.Database.MaxIdleConnections < 0 || cfg.Database.MaxIdleConnections > cfg.Database.MaxOpenConnections {
		return errors.New("invalid database connection pool limits")
	}
	if cfg.RateLimit.RequestsPerMinute < 1 || cfg.RateLimit.Burst < 1 || cfg.RateLimit.MaxClients < 1 {
		return errors.New("rate-limit values must be positive")
	}
	if cfg.Server.MaxHeaderBytes < 4096 {
		return errors.New("max_header_bytes must be at least 4096")
	}
	if cfg.Server.ReadHeaderTimeout <= 0 || cfg.Server.ReadTimeout <= 0 || cfg.Server.WriteTimeout <= 0 || cfg.Server.IdleTimeout <= 0 || cfg.Server.ShutdownTimeout <= 0 {
		return errors.New("server timeouts must be positive")
	}
	return nil
}

func (cfg DatabaseConfig) Address() string {
	return net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}
