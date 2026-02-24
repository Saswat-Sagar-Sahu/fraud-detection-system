# Real-Time Fraud Detection System

Production-style, event-driven fraud/risk detection pipeline built with Go, Kafka, and Redis.

## What Is Implemented
- HTTP event ingestion service (`event-generator`)
- Kafka event streaming (`fraud.events.v1`)
- Kafka consumer group with parallel processing (`risk-engine`)
- Redis-backed user state tracking
  - failed login counter
  - last login location
  - recent transactions list
- Rule engine for risk scoring
- Kafka risk alert publishing (`fraud.risk-alerts.v1`)

## Event Types
- `LOGIN`
- `TRANSACTION`
- `DEVICE_CHANGE`
- `PASSWORD_RESET`

## Rule Engine
Rules currently active:
- Failed login burst: `+30`
- New device: `+20`
- Impossible travel: `+50`
- Transaction spike: `+40`

Default fraud alert condition:
- Publish alert when `risk_score >= 50`

## System Components
- `services/event-generator`: REST API producer
- `services/risk-engine`: Kafka consumer + rules + Redis state + alert publisher
- `docker-compose.yml`: Kafka, Zookeeper, Redis, Kafka UI, RedisInsight

## Topics
- Events topic: `fraud.events.v1`
- Alerts topic: `fraud.risk-alerts.v1`

## Quick Start
1. Start infrastructure:
```bash
make infra-up-monitoring
```

2. Start risk engine:
```bash
make run-consumer
```

3. Start event generator:
```bash
make run-generator
```

4. Send a test event:
```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "LOGIN",
    "user_id": "user-123",
    "login_id": "john.doe@example.com",
    "occurred_at": "2026-02-25T10:00:00Z",
    "source_ip": "103.21.244.1",
    "device_id": "device-abc",
    "country": "US",
    "latitude": 37.7749,
    "longitude": -122.4194,
    "metadata": {"login_success": false}
  }'
```

## How Fraud Reaction Works
For each event, `risk-engine`:
1. Reads historical user state from Redis.
2. Evaluates all rules and computes `risk_score`.
3. Updates Redis state with current event.
4. If score crosses threshold, publishes alert to `fraud.risk-alerts.v1`.

You will see logs with:
- `risk_score`
- `rule_breakdown`
- `risk_alert_raised=true|false`

## Validate Alerts
Consume alert topic:
```bash
docker-compose exec -T kafka kafka-console-consumer \
  --bootstrap-server kafka:29092 \
  --topic fraud.risk-alerts.v1 \
  --from-beginning
```

## Dashboards
- Kafka UI: http://localhost:8081
- RedisInsight: http://localhost:5540

RedisInsight connection (when started via Compose):
- host: `redis`
- port: `6379`
- username/password: empty

## Important Config (risk-engine)
See `services/risk-engine/.env.example`.
Most important settings:
- `RISK_ALERTS_TOPIC`
- `RISK_ALERT_THRESHOLD`
- `ENABLE_RISK_ALERTS`
- `REDIS_KEY_PREFIX`
- `REDIS_FAILED_LOGIN_KEY`
- `REDIS_LOCATIONS_KEY`
- `REDIS_TRANSACTIONS_KEY`
