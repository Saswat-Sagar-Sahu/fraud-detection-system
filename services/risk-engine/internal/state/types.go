package state

import (
	"context"
	"time"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
)

type Tracker interface {
	TrackEvent(ctx context.Context, event model.Event) error
}

type Reader interface {
	GetUserState(ctx context.Context, userID string) (model.UserState, error)
}

type TrackerConfig struct {
	KeyPrefix                 string
	FailedLoginKeyPattern     string
	LocationsKeyPattern       string
	TransactionsKeyPattern    string
	FailedLoginTTL            time.Duration
	LocationTTL               time.Duration
	TransactionsTTL           time.Duration
	RecentTransactionsMaxSize int64
}
