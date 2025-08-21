package config

import (
        "fmt"
        "os"
        "strconv"
        "time"
)

// Config holds all configuration for the application
type Config struct {
        Server   ServerConfig   `yaml:"server"`
        Database DatabaseConfig `yaml:"database"`
        JWT      JWTConfig      `yaml:"jwt"`
        Metrics  MetricsConfig  `yaml:"metrics"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
        Host         string        `yaml:"host"`
        Port         int           `yaml:"port"`
        ReadTimeout  time.Duration `yaml:"read_timeout"`
        WriteTimeout time.Duration `yaml:"write_timeout"`
        IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
        Host            string `yaml:"host"`
        Port            int    `yaml:"port"`
        User            string `yaml:"user"`
        Password        string `yaml:"password"`
        Name            string `yaml:"name"`
        SSLMode         string `yaml:"ssl_mode"`
        MaxOpenConns    int    `yaml:"max_open_conns"`
        MaxIdleConns    int    `yaml:"max_idle_conns"`
        ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
        SecretKey       string        `yaml:"secret_key"`
        TokenExpiration time.Duration `yaml:"token_expiration"`
        RefreshExpiration time.Duration `yaml:"refresh_expiration"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
        Enabled bool   `yaml:"enabled"`
        Path    string `yaml:"path"`
        Port    int    `yaml:"port"`
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
        config := &Config{
                Server: ServerConfig{
                        Host:         getEnv("SERVER_HOST", "0.0.0.0"),
                        Port:         getEnvAsInt("SERVER_PORT", 8080),
                        ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
                        WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
                        IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
                },
                Database: DatabaseConfig{
                        Host:            getEnv("DB_HOST", "localhost"),
                        Port:            getEnvAsInt("DB_PORT", 5432),
                        User:            getEnv("DB_USER", "postgres"),
                        Password:        getEnv("DB_PASSWORD", "postgres"),
                        Name:            getEnv("DB_NAME", "taskdb"),
                        SSLMode:         getEnv("DB_SSL_MODE", "disable"),
                        MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
                        MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
                        ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
                },
                JWT: JWTConfig{
                        SecretKey:         getEnv("JWT_SECRET_KEY", "your-secret-key-change-this-in-production"),
                        TokenExpiration:   getEnvAsDuration("JWT_TOKEN_EXPIRATION", 24*time.Hour),
                        RefreshExpiration: getEnvAsDuration("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
                },
                Metrics: MetricsConfig{
                        Enabled: getEnvAsBool("METRICS_ENABLED", true),
                        Path:    getEnv("METRICS_PATH", "/metrics"),
                        Port:    getEnvAsInt("METRICS_PORT", 8081),
                },
        }

        return config, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
        if value := os.Getenv(key); value != "" {
                return value
        }
        return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
        if value := os.Getenv(key); value != "" {
                if intValue, err := strconv.Atoi(value); err == nil {
                        return intValue
                }
        }
        return defaultValue
}

// getEnvAsBool gets an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
        if value := os.Getenv(key); value != "" {
                if boolValue, err := strconv.ParseBool(value); err == nil {
                        return boolValue
                }
        }
        return defaultValue
}

// getEnvAsDuration gets an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
        if value := os.Getenv(key); value != "" {
                if duration, err := time.ParseDuration(value); err == nil {
                        return duration
                }
        }
        return defaultValue
}

// GetDSN returns the database connection string
func (d *DatabaseConfig) GetDSN() string {
        return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
                d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}