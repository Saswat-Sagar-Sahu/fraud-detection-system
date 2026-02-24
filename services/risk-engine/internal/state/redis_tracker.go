package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
)

type RedisTracker struct {
	client redis.UniversalClient
	cfg    TrackerConfig
}

type locationState struct {
	EventID    string   `json:"event_id"`
	LoginID    string   `json:"login_id,omitempty"`
	SourceIP   string   `json:"source_ip,omitempty"`
	DeviceID   string   `json:"device_id,omitempty"`
	Country    string   `json:"country,omitempty"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	OccurredAt string   `json:"occurred_at"`
	RecordedAt string   `json:"recorded_at"`
}

type transactionState struct {
	EventID    string   `json:"event_id"`
	SourceIP   string   `json:"source_ip,omitempty"`
	DeviceID   string   `json:"device_id,omitempty"`
	Country    string   `json:"country,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Amount     *float64 `json:"amount,omitempty"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
	OccurredAt string   `json:"occurred_at"`
	RecordedAt string   `json:"recorded_at"`
}

func NewRedisTracker(client redis.UniversalClient, cfg TrackerConfig) (*RedisTracker, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if err := validateKeyPattern(cfg.FailedLoginKeyPattern); err != nil {
		return nil, fmt.Errorf("invalid failed login key pattern: %w", err)
	}
	if err := validateKeyPattern(cfg.LocationsKeyPattern); err != nil {
		return nil, fmt.Errorf("invalid locations key pattern: %w", err)
	}
	if err := validateKeyPattern(cfg.TransactionsKeyPattern); err != nil {
		return nil, fmt.Errorf("invalid transactions key pattern: %w", err)
	}
	if cfg.FailedLoginTTL <= 0 {
		return nil, fmt.Errorf("failed login ttl must be > 0")
	}
	if cfg.LocationTTL <= 0 {
		return nil, fmt.Errorf("location ttl must be > 0")
	}
	if cfg.TransactionsTTL <= 0 {
		return nil, fmt.Errorf("transactions ttl must be > 0")
	}
	if cfg.RecentTransactionsMaxSize <= 0 {
		return nil, fmt.Errorf("recent transactions max size must be > 0")
	}

	return &RedisTracker{client: client, cfg: cfg}, nil
}

func (t *RedisTracker) TrackEvent(ctx context.Context, event model.Event) error {
	switch event.EventType {
	case model.EventTypeLogin:
		return t.trackLogin(ctx, event)
	case model.EventTypeTransaction:
		return t.trackTransaction(ctx, event)
	default:
		return nil
	}
}

func (t *RedisTracker) GetUserState(ctx context.Context, userID string) (model.UserState, error) {
	failedLoginKey, err := t.userKey(t.cfg.FailedLoginKeyPattern, userID)
	if err != nil {
		return model.UserState{}, err
	}
	locationKey, err := t.userKey(t.cfg.LocationsKeyPattern, userID)
	if err != nil {
		return model.UserState{}, err
	}
	transactionsKey, err := t.userKey(t.cfg.TransactionsKeyPattern, userID)
	if err != nil {
		return model.UserState{}, err
	}

	state := model.UserState{}

	failedLoginCount, err := t.client.Get(ctx, failedLoginKey).Int64()
	if err == nil {
		state.FailedLoginCount = failedLoginCount
	} else if err != redis.Nil {
		return model.UserState{}, fmt.Errorf("read failed login count key=%s: %w", failedLoginKey, err)
	}

	locationRaw, err := t.client.Get(ctx, locationKey).Result()
	if err == nil {
		var stored locationState
		if unmarshalErr := json.Unmarshal([]byte(locationRaw), &stored); unmarshalErr != nil {
			return model.UserState{}, fmt.Errorf("decode location state key=%s: %w", locationKey, unmarshalErr)
		}

		location := model.LoginLocation{
			EventID:   stored.EventID,
			LoginID:   stored.LoginID,
			SourceIP:  stored.SourceIP,
			DeviceID:  stored.DeviceID,
			Country:   stored.Country,
			Latitude:  stored.Latitude,
			Longitude: stored.Longitude,
		}
		location.OccurredAt, err = parseTimestamp(stored.OccurredAt)
		if err != nil {
			return model.UserState{}, fmt.Errorf("parse location occurred_at key=%s: %w", locationKey, err)
		}
		location.RecordedAt, err = parseTimestamp(stored.RecordedAt)
		if err != nil {
			return model.UserState{}, fmt.Errorf("parse location recorded_at key=%s: %w", locationKey, err)
		}
		state.LastLoginLocation = &location
	} else if err != redis.Nil {
		return model.UserState{}, fmt.Errorf("read location state key=%s: %w", locationKey, err)
	}

	records, err := t.client.LRange(ctx, transactionsKey, 0, t.cfg.RecentTransactionsMaxSize-1).Result()
	if err != nil && err != redis.Nil {
		return model.UserState{}, fmt.Errorf("read transactions key=%s: %w", transactionsKey, err)
	}
	state.RecentTransactions = make([]model.TransactionRecord, 0, len(records))
	for idx, raw := range records {
		var stored transactionState
		if unmarshalErr := json.Unmarshal([]byte(raw), &stored); unmarshalErr != nil {
			return model.UserState{}, fmt.Errorf("decode transaction state key=%s index=%d: %w", transactionsKey, idx, unmarshalErr)
		}

		tx := model.TransactionRecord{
			EventID:   stored.EventID,
			SourceIP:  stored.SourceIP,
			DeviceID:  stored.DeviceID,
			Country:   stored.Country,
			Currency:  stored.Currency,
			Amount:    stored.Amount,
			Latitude:  stored.Latitude,
			Longitude: stored.Longitude,
		}
		tx.OccurredAt, err = parseTimestamp(stored.OccurredAt)
		if err != nil {
			return model.UserState{}, fmt.Errorf("parse transaction occurred_at key=%s index=%d: %w", transactionsKey, idx, err)
		}
		tx.RecordedAt, err = parseTimestamp(stored.RecordedAt)
		if err != nil {
			return model.UserState{}, fmt.Errorf("parse transaction recorded_at key=%s index=%d: %w", transactionsKey, idx, err)
		}
		state.RecentTransactions = append(state.RecentTransactions, tx)
	}

	return state, nil
}

func (t *RedisTracker) trackLogin(ctx context.Context, event model.Event) error {
	failedLoginKey, err := t.userKey(t.cfg.FailedLoginKeyPattern, event.UserID)
	if err != nil {
		return err
	}
	locationKey, err := t.userKey(t.cfg.LocationsKeyPattern, event.UserID)
	if err != nil {
		return err
	}

	loginSuccess, hasLoginSuccess, err := parseLoginSuccess(event.Metadata)
	if err != nil {
		return fmt.Errorf("parse login_success metadata for user_id=%s: %w", event.UserID, err)
	}

	pipe := t.client.TxPipeline()
	hasMutations := false

	if hasLoginSuccess {
		hasMutations = true
		if loginSuccess {
			pipe.Del(ctx, failedLoginKey)
		} else {
			pipe.Incr(ctx, failedLoginKey)
			pipe.Expire(ctx, failedLoginKey, t.cfg.FailedLoginTTL)
		}
	}

	if hasLocationData(event) {
		hasMutations = true
		state, err := json.Marshal(locationState{
			EventID:    event.EventID,
			LoginID:    event.LoginID,
			SourceIP:   event.SourceIP,
			DeviceID:   event.DeviceID,
			Country:    event.Country,
			Latitude:   event.Latitude,
			Longitude:  event.Longitude,
			OccurredAt: event.OccurredAt.UTC().Format(time.RFC3339Nano),
			RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return fmt.Errorf("marshal location state event_id=%s: %w", event.EventID, err)
		}
		pipe.Set(ctx, locationKey, state, t.cfg.LocationTTL)
	}

	if !hasMutations {
		return nil
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("apply login state update user_id=%s: %w", event.UserID, err)
	}
	return nil
}

func (t *RedisTracker) trackTransaction(ctx context.Context, event model.Event) error {
	if event.Amount == nil {
		return fmt.Errorf("transaction amount missing event_id=%s", event.EventID)
	}
	if *event.Amount < 0 {
		return fmt.Errorf("transaction amount must be >= 0 event_id=%s", event.EventID)
	}
	if strings.TrimSpace(event.Currency) == "" {
		return fmt.Errorf("transaction currency missing event_id=%s", event.EventID)
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("transaction occurred_at missing event_id=%s", event.EventID)
	}

	transactionsKey, err := t.userKey(t.cfg.TransactionsKeyPattern, event.UserID)
	if err != nil {
		return err
	}

	state, err := json.Marshal(transactionState{
		EventID:    event.EventID,
		SourceIP:   event.SourceIP,
		DeviceID:   event.DeviceID,
		Country:    event.Country,
		Currency:   event.Currency,
		Amount:     event.Amount,
		Latitude:   event.Latitude,
		Longitude:  event.Longitude,
		OccurredAt: event.OccurredAt.UTC().Format(time.RFC3339Nano),
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("marshal transaction state event_id=%s: %w", event.EventID, err)
	}

	pipe := t.client.TxPipeline()
	pipe.LPush(ctx, transactionsKey, state)
	pipe.LTrim(ctx, transactionsKey, 0, t.cfg.RecentTransactionsMaxSize-1)
	pipe.Expire(ctx, transactionsKey, t.cfg.TransactionsTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("apply transaction state update user_id=%s: %w", event.UserID, err)
	}
	return nil
}

func (t *RedisTracker) userKey(pattern, userID string) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", fmt.Errorf("user_id is empty")
	}

	base := strings.ReplaceAll(pattern, "{id}", trimmedUserID)
	if strings.Contains(base, "{id}") {
		return "", fmt.Errorf("key pattern unresolved: %s", pattern)
	}

	prefix := strings.Trim(strings.TrimSpace(t.cfg.KeyPrefix), ":")
	if prefix == "" {
		return base, nil
	}
	return prefix + ":" + base, nil
}

func validateKeyPattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("pattern is empty")
	}
	if !strings.Contains(pattern, "{id}") {
		return fmt.Errorf("pattern must include {id}")
	}
	return nil
}

func hasLocationData(event model.Event) bool {
	if event.Latitude != nil && event.Longitude != nil {
		return true
	}
	if strings.TrimSpace(event.Country) != "" {
		return true
	}
	if strings.TrimSpace(event.SourceIP) != "" {
		return true
	}
	return false
}

func parseLoginSuccess(metadata map[string]interface{}) (value bool, exists bool, err error) {
	raw, ok := metadata["login_success"]
	if !ok {
		return false, false, nil
	}

	switch v := raw.(type) {
	case bool:
		return v, true, nil
	case float64:
		switch v {
		case 1:
			return true, true, nil
		case 0:
			return false, true, nil
		default:
			return false, true, fmt.Errorf("invalid login_success numeric value %v", v)
		}
	case string:
		parsed, parseErr := strconv.ParseBool(strings.TrimSpace(v))
		if parseErr != nil {
			return false, true, fmt.Errorf("invalid login_success string %q", v)
		}
		return parsed, true, nil
	default:
		return false, true, fmt.Errorf("unsupported login_success type %T", raw)
	}
}

func parseTimestamp(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
