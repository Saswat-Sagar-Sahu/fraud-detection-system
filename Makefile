.PHONY: infra-up infra-up-monitoring infra-down monitoring-urls run-generator run-consumer test-generator test-consumer topics-create wait-kafka

COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; fi)

ifeq ($(strip $(COMPOSE)),)
$(error Docker Compose not found. Install Docker Compose v2 plugin or docker-compose binary)
endif

KAFKA_CONTAINER ?= kafka
KAFKA_BOOTSTRAP ?= kafka:29092
EVENTS_TOPIC ?= fraud.events.v1
RISK_ALERTS_TOPIC ?= fraud.risk-alerts.v1

infra-up:
	$(COMPOSE) up -d zookeeper kafka redis
	$(MAKE) topics-create

infra-up-monitoring:
	$(COMPOSE) --profile monitoring up -d
	$(MAKE) topics-create
	$(MAKE) monitoring-urls

infra-down:
	$(COMPOSE) --profile monitoring down -v

monitoring-urls:
	@echo "Kafka UI: http://localhost:8081"
	@echo "RedisInsight: http://localhost:5540"
	@echo "RedisInsight connection (Compose): host=redis port=6379 username=<empty> password=<empty>"

wait-kafka:
	@echo "Waiting for Kafka to be ready..."
	@until $(COMPOSE) exec -T $(KAFKA_CONTAINER) kafka-topics --bootstrap-server $(KAFKA_BOOTSTRAP) --list >/dev/null 2>&1; do sleep 2; done
	@echo "Kafka is ready."

topics-create: wait-kafka
	$(COMPOSE) exec -T $(KAFKA_CONTAINER) kafka-topics --bootstrap-server $(KAFKA_BOOTSTRAP) --create --if-not-exists --topic $(EVENTS_TOPIC) --partitions 3 --replication-factor 1
	$(COMPOSE) exec -T $(KAFKA_CONTAINER) kafka-topics --bootstrap-server $(KAFKA_BOOTSTRAP) --create --if-not-exists --topic $(RISK_ALERTS_TOPIC) --partitions 3 --replication-factor 1
	@echo "Available Kafka topics:"
	$(COMPOSE) exec -T $(KAFKA_CONTAINER) kafka-topics --bootstrap-server $(KAFKA_BOOTSTRAP) --list

run-generator:
	cd services/event-generator && go run ./cmd/event-generator

run-consumer:
	cd services/risk-engine && go run ./cmd/risk-engine

test-generator:
	cd services/event-generator && go test ./...

test-consumer:
	cd services/risk-engine && go test ./...
