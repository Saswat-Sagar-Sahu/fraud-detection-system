package main

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/config"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/kafka"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/processor"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/rules"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/state"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)

	cfg := config.Load()
	log.Printf(
		"configuration loaded kafka_brokers=%s kafka_topic=%s kafka_group_id=%s worker_buffer=%d process_timeout=%s risk_alerts_topic=%s risk_alert_threshold=%d enable_risk_alerts=%t redis_addr=%s redis_db=%d redis_key_prefix=%s",
		strings.Join(cfg.KafkaBrokers, ","),
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
		cfg.WorkerBuffer,
		cfg.ProcessTimeout,
		cfg.RiskAlertsTopic,
		cfg.RiskAlertThreshold,
		cfg.EnableRiskAlerts,
		cfg.RedisAddr,
		cfg.RedisDB,
		cfg.RedisKeyPrefix,
	)

	redisClient, err := state.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis client initialization failed: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("redis close error: %v", err)
		}
	}()

	tracker, err := state.NewRedisTracker(redisClient, state.TrackerConfig{
		KeyPrefix:                 cfg.RedisKeyPrefix,
		FailedLoginKeyPattern:     cfg.FailedLoginKeyPattern,
		LocationsKeyPattern:       cfg.LocationsKeyPattern,
		TransactionsKeyPattern:    cfg.TransactionsKeyPattern,
		FailedLoginTTL:            cfg.FailedLoginTTL,
		LocationTTL:               cfg.LocationTTL,
		TransactionsTTL:           cfg.TransactionsTTL,
		RecentTransactionsMaxSize: cfg.RecentTransactionsMaxSize,
	})
	if err != nil {
		log.Fatalf("redis state tracker initialization failed: %v", err)
	}

	ruleEngine := rules.NewEngine([]rules.NamedRule{
		{Name: "failed_login_burst", Rule: rules.NewFailedLoginBurstRule(cfg.RuleFailedLoginBurstThreshold, cfg.RuleFailedLoginBurstScore)},
		{Name: "new_device", Rule: rules.NewNewDeviceRule(cfg.RuleNewDeviceScore)},
		{Name: "impossible_travel", Rule: rules.NewImpossibleTravelRule(cfg.RuleImpossibleTravelScore, cfg.RuleImpossibleTravelMaxSpeed, cfg.RuleImpossibleTravelMinKM)},
		{Name: "transaction_spike", Rule: rules.NewTransactionSpikeRule(cfg.RuleTransactionSpikeScore, cfg.RuleTransactionSpikeMinSamples, cfg.RuleTransactionSpikeMultiplier)},
	})

	log.Printf(
		"rules configured failed_login_burst_threshold=%d failed_login_burst_score=%d new_device_score=%d impossible_travel_score=%d impossible_travel_max_speed_kmph=%.2f impossible_travel_min_km=%.2f transaction_spike_score=%d transaction_spike_min_samples=%d transaction_spike_multiplier=%.2f",
		cfg.RuleFailedLoginBurstThreshold,
		cfg.RuleFailedLoginBurstScore,
		cfg.RuleNewDeviceScore,
		cfg.RuleImpossibleTravelScore,
		cfg.RuleImpossibleTravelMaxSpeed,
		cfg.RuleImpossibleTravelMinKM,
		cfg.RuleTransactionSpikeScore,
		cfg.RuleTransactionSpikeMinSamples,
		cfg.RuleTransactionSpikeMultiplier,
	)

	var alertPublisher *kafka.RiskAlertPublisher
	if cfg.EnableRiskAlerts {
		if err := kafka.ValidateTopic(cfg.RiskAlertsTopic); err != nil {
			log.Fatalf("invalid risk alerts topic: %v", err)
		}
		alertPublisher = kafka.NewRiskAlertPublisher(cfg.KafkaBrokers, cfg.RiskAlertsTopic)
		defer func() {
			if err := alertPublisher.Close(); err != nil {
				log.Printf("risk alert publisher close error: %v", err)
			}
		}()
	}

	eventProcessor := processor.NewRiskProcessor(
		tracker,
		tracker,
		ruleEngine,
		alertPublisher,
		cfg.RiskAlertThreshold,
		cfg.EnableRiskAlerts,
	)
	consumer := kafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
		cfg.WorkerBuffer,
		cfg.ProcessTimeout,
		eventProcessor,
	)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("consumer close error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("consumer stopped with error: %v", err)
	}

	log.Printf("consumer stopped")
}
