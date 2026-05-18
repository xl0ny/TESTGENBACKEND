package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr           string
	KafkaBrokers       []string
	KafkaTopicSegments string
	AssemblyInterval   time.Duration
	TimeoutCycles      int
	SegmentLossRate    float64
	AgentBaseURL       string
}

func Load() Config {
	intervalSec := envInt("ASSEMBLY_INTERVAL_SEC", 2)
	timeoutCycles := envInt("TIMEOUT_CYCLES", 3)
	lossPct := envInt("SEGMENT_LOSS_PERCENT", 8)

	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8081"),
		KafkaBrokers:       []string{env("KAFKA_BROKERS", "localhost:9092")},
		KafkaTopicSegments: env("KAFKA_TOPIC_SEGMENTS", "transport-segments"),
		AssemblyInterval:   time.Duration(intervalSec) * time.Second,
		TimeoutCycles:      timeoutCycles,
		SegmentLossRate:    float64(lossPct) / 100.0,
		AgentBaseURL:       env("AGENT_BASE_URL", "http://localhost:8082"),
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
