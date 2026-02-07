package config

import (
	"fmt"
	"math/bits"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration
type Config struct {
	// MCP Server Configuration
	MCPPort        int           `env:"MCP_PORT" envDefault:"8080"`
	LogLevel       string        `env:"LOG_LEVEL" envDefault:"info"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`

	// Web UI Configuration
	WebPort    int  `env:"WEB_PORT" envDefault:"8081"`
	WebEnabled bool `env:"WEB_ENABLED" envDefault:"true"`

	// Database Configuration
	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432"`
	DBName     string `env:"DB_NAME" envDefault:"guardrails"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBSSLMode  string `env:"DB_SSLMODE" envDefault:"prefer"`

	// Redis Configuration
	RedisHost     string `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort     int    `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisUseTLS   bool   `env:"REDIS_USE_TLS" envDefault:"false"`

	// Security Configuration
	MCPAPIKey string `env:"MCP_API_KEY,required"`
	IDEAPIKey string `env:"IDE_API_KEY,required"`

	// JWT Configuration
	JWTSecret        string        `env:"JWT_SECRET,required"`
	JWTIssuer        string        `env:"JWT_ISSUER" envDefault:"guardrail-mcp"`
	JWTExpiry        time.Duration `env:"JWT_EXPIRY" envDefault:"15m"`
	JWTRotationHours time.Duration `env:"JWT_ROTATION_HOURS" envDefault:"168h"` // 7 days

	// Rate Limiting Configuration
	MCPRateLimit     int `env:"MCP_RATE_LIMIT" envDefault:"1000"`
	IDERateLimit     int `env:"IDE_RATE_LIMIT" envDefault:"500"`
	SessionRateLimit int `env:"SESSION_RATE_LIMIT" envDefault:"100"`

	// Cache TTL Configuration
	CacheTTLRules  time.Duration `env:"CACHE_TTL_RULES" envDefault:"5m"`
	CacheTTLDocs   time.Duration `env:"CACHE_TTL_DOCS" envDefault:"10m"`
	CacheTTLSearch time.Duration `env:"CACHE_TTL_SEARCH" envDefault:"2m"`

	// Feature Flags
	EnableValidation bool `env:"ENABLE_VALIDATION" envDefault:"true"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate JWT secret
	if err := ValidateJWTSecret(cfg.JWTSecret); err != nil {
		return nil, fmt.Errorf("JWT_SECRET validation failed: %w", err)
	}

	return &cfg, nil
}

// ValidateJWTSecret ensures the JWT secret meets security requirements
func ValidateJWTSecret(secret string) error {
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(secret))
	}

	// Check entropy
	var entropy float64
	for _, b := range []byte(secret) {
		entropy += float64(bits.OnesCount8(uint8(b)))
	}
	if entropy/float64(len(secret)) < 3.5 {
		return fmt.Errorf("JWT_SECRET has insufficient entropy")
	}

	return nil
}

// DatabaseURL returns the PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

// RedisAddr returns the Redis connection address
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}
