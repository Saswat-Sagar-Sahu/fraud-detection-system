package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr     string
	KafkaBrokers []string
	KafkaTopic   string
}

func Load() Config {
	return Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		KafkaBrokers: parseBrokers(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "fraud.events.v1"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseBrokers(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"localhost:9092"}
	}
	return out
}
