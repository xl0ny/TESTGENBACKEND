package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr              string
	TransportBaseURL      string
	ReceiveWaitMS         int
	ReceiveMaxAttempts    int
	ReceivePollIntervalMS int
}

func Load() Config {
	port := envInt("APP_PORT", 8080)
	addr := env("HTTP_ADDR", "")
	if addr == "" {
		addr = fmt.Sprintf(":%d", port)
	}

	return Config{
		HTTPAddr:              addr,
		TransportBaseURL:      env("TRANSPORT_BASE_URL", "http://localhost:8081"),
		ReceiveWaitMS:         envInt("RECEIVE_WAIT_MS", 3000),
		ReceiveMaxAttempts:    envInt("RECEIVE_MAX_ATTEMPTS", 40),
		ReceivePollIntervalMS: envInt("RECEIVE_POLL_INTERVAL_MS", 500),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
