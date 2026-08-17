package shortner

import "time"

const (
	DefaultExpiryDays = 30
	DefaultSlugLength = 8
	MaxURLLength      = 2048
)

// Config contains application configuration. Secrets should be supplied through
// environment variables rather than committed configuration files.
type Config struct {
	Port          int             `json:"port"`
	PublicBaseURL string          `json:"public_base_url"`
	Database      DatabaseConfig  `json:"database"`
	Server        ServerConfig    `json:"server"`
	RateLimit     RateLimitConfig `json:"rate_limit"`
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
	ViewsDirectory    string        `json:"views_directory"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst             int `json:"burst"`
	MaxClients        int `json:"max_clients"`
}

// ShortLink represents a stored redirect.
type ShortLink struct {
	Shortened string    `json:"slug"`
	Original  string    `json:"url"`
	Expiry    int       `json:"expiryDays"`
	Created   time.Time `json:"createdAt"`
	Hits      uint64    `json:"hits"`
}

func (link ShortLink) Expired(now time.Time) bool {
	return link.Expiry > 0 && !now.Before(link.Created.AddDate(0, 0, link.Expiry))
}
