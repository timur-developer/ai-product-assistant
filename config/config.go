package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultAppEnv          = "dev"
	defaultHTTPPort        = "8080"
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultDBPingTimeout   = 5 * time.Second
	defaultLLMTimeout      = 15 * time.Second
	defaultLLMMaxRetries   = 2
	defaultLLMRetryDelay   = 200 * time.Millisecond
)

type Config struct {
	AppEnv   string
	HTTPPort string

	DatabaseURL string
	LLM         LLMConfig

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DBPingTimeout   time.Duration
}

type LLMConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
}

func Load() (Config, error) {
	llmTimeout, err := getEnvDurationSecondsStrict("LLM_TIMEOUT_SEC", defaultLLMTimeout)
	if err != nil {
		return Config{}, err
	}
	llmMaxRetries, err := getEnvIntStrict("LLM_MAX_RETRIES", defaultLLMMaxRetries)
	if err != nil {
		return Config{}, err
	}
	llmRetryDelay, err := getEnvDurationMillisecondsStrict("LLM_RETRY_BASE_DELAY_MS", defaultLLMRetryDelay)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:          getEnv("APP_ENV", defaultAppEnv),
		HTTPPort:        getEnv("HTTP_PORT", defaultHTTPPort),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LLM: LLMConfig{
			BaseURL:        os.Getenv("LLM_BASE_URL"),
			APIKey:         os.Getenv("LLM_API_KEY"),
			Model:          os.Getenv("LLM_MODEL"),
			Timeout:        llmTimeout,
			MaxRetries:     llmMaxRetries,
			RetryBaseDelay: llmRetryDelay,
		},
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
	if cfg.LLM.BaseURL == "" {
		return Config{}, errors.New("LLM_BASE_URL is required")
	}
	if cfg.LLM.APIKey == "" {
		return Config{}, errors.New("LLM_API_KEY is required")
	}
	if cfg.LLM.Model == "" {
		return Config{}, errors.New("LLM_MODEL is required")
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

func getEnvDurationMilliseconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return fallback
	}

	return time.Duration(ms) * time.Millisecond
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}

	return n
}

func getEnvDurationSecondsStrict(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return time.Duration(seconds) * time.Second, nil
}

func getEnvDurationMillisecondsStrict(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return time.Duration(ms) * time.Millisecond, nil
}

func getEnvIntStrict(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}

	return n, nil
}
