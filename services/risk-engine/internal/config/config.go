package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaGroupID       string
	WorkerBuffer       int
	ProcessTimeout     time.Duration
	RiskAlertsTopic    string
	RiskAlertThreshold int
	EnableRiskAlerts   bool

	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	RedisKeyPrefix            string
	FailedLoginKeyPattern     string
	LocationsKeyPattern       string
	TransactionsKeyPattern    string
	FailedLoginTTL            time.Duration
	LocationTTL               time.Duration
	TransactionsTTL           time.Duration
	RecentTransactionsMaxSize int64

	RuleFailedLoginBurstThreshold  int64
	RuleFailedLoginBurstScore      int
	RuleNewDeviceScore             int
	RuleImpossibleTravelScore      int
	RuleImpossibleTravelMaxSpeed   float64
	RuleImpossibleTravelMinKM      float64
	RuleTransactionSpikeScore      int
	RuleTransactionSpikeMinSamples int
	RuleTransactionSpikeMultiplier float64
}

func Load() Config {
	return Config{
		KafkaBrokers:       parseCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "fraud.events.v1"),
		KafkaGroupID:       getEnv("KAFKA_GROUP_ID", "risk-engine-v1"),
		WorkerBuffer:       getEnvInt("WORKER_BUFFER", 256),
		ProcessTimeout:     getEnvDuration("PROCESS_TIMEOUT", 5*time.Second),
		RiskAlertsTopic:    getEnv("RISK_ALERTS_TOPIC", "fraud.risk-alerts.v1"),
		RiskAlertThreshold: getEnvInt("RISK_ALERT_THRESHOLD", 50),
		EnableRiskAlerts:   getEnvBool("ENABLE_RISK_ALERTS", true),

		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		RedisKeyPrefix:            getEnv("REDIS_KEY_PREFIX", "fraud"),
		FailedLoginKeyPattern:     getEnv("REDIS_FAILED_LOGIN_KEY", "user:{id}:failed_login"),
		LocationsKeyPattern:       getEnv("REDIS_LOCATIONS_KEY", "user:{id}:locations"),
		TransactionsKeyPattern:    getEnv("REDIS_TRANSACTIONS_KEY", "user:{id}:transactions"),
		FailedLoginTTL:            getEnvDuration("FAILED_LOGIN_TTL", 24*time.Hour),
		LocationTTL:               getEnvDuration("LOCATION_TTL", 30*24*time.Hour),
		TransactionsTTL:           getEnvDuration("TRANSACTIONS_TTL", 30*24*time.Hour),
		RecentTransactionsMaxSize: getEnvInt64("RECENT_TRANSACTIONS_MAX_SIZE", 50),

		RuleFailedLoginBurstThreshold:  getEnvInt64("RULE_FAILED_LOGIN_BURST_THRESHOLD", 3),
		RuleFailedLoginBurstScore:      getEnvInt("RULE_FAILED_LOGIN_BURST_SCORE", 30),
		RuleNewDeviceScore:             getEnvInt("RULE_NEW_DEVICE_SCORE", 20),
		RuleImpossibleTravelScore:      getEnvInt("RULE_IMPOSSIBLE_TRAVEL_SCORE", 50),
		RuleImpossibleTravelMaxSpeed:   getEnvFloat64("RULE_IMPOSSIBLE_TRAVEL_MAX_SPEED_KMPH", 900),
		RuleImpossibleTravelMinKM:      getEnvFloat64("RULE_IMPOSSIBLE_TRAVEL_MIN_DISTANCE_KM", 500),
		RuleTransactionSpikeScore:      getEnvInt("RULE_TRANSACTION_SPIKE_SCORE", 40),
		RuleTransactionSpikeMinSamples: getEnvInt("RULE_TRANSACTION_SPIKE_MIN_SAMPLES", 3),
		RuleTransactionSpikeMultiplier: getEnvFloat64("RULE_TRANSACTION_SPIKE_MULTIPLIER", 3.0),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed < 0 {
		return defaultVal
	}
	return parsed
}

func getEnvInt64(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil || parsed <= 0 {
		return defaultVal
	}
	return parsed
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := time.ParseDuration(v)
	if err != nil || parsed <= 0 {
		return defaultVal
	}
	return parsed
}

func getEnvFloat64(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil || parsed <= 0 {
		return defaultVal
	}
	return parsed
}

func getEnvBool(key string, defaultVal bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func parseCSV(v string) []string {
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
