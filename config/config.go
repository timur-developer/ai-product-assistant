package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultAppEnv         = "dev"
	defaultHTTPPort       = "8080"
	defaultReadTimeout    = 10 * time.Second
	defaultWriteTimeout   = 10 * time.Second
	defaultIdleTimeout    = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultDBPingTimeout  = 5 * time.Second
)

type Config struct {
	AppEnv   string
	HTTPPort string

	DatabaseURL string

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DBPingTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:          getEnv("APP_ENV", defaultAppEnv),
		HTTPPort:        getEnv("HTTP_PORT", defaultHTTPPort),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		ReadTimeout:     getEnvDurationSeconds("HTTP_READ_TIMEOUT_SEC", defaultReadTimeout),
		WriteTimeout:    getEnvDurationSeconds("HTTP_WRITE_TIMEOUT_SEC", defaultWriteTimeout),
		IdleTimeout:     getEnvDurationSeconds("HTTP_IDLE_TIMEOUT_SEC", defaultIdleTimeout),
		ShutdownTimeout: getEnvDurationSeconds("HTTP_SHUTDOWN_TIMEOUT_SEC", defaultShutdownTimeout),
		DBPingTimeout:   getEnvDurationSeconds("DB_PING_TIMEOUT_SEC", defaultDBPingTimeout),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.HTTPPort == "" {
		return Config{}, errors.New("HTTP_PORT is required")
	}

	return cfg, nil
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
