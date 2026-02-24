package model

import "time"

type UserState struct {
	FailedLoginCount   int64
	LastLoginLocation  *LoginLocation
	RecentTransactions []TransactionRecord
}

type LoginLocation struct {
	EventID    string
	LoginID    string
	SourceIP   string
	DeviceID   string
	Country    string
	Latitude   *float64
	Longitude  *float64
	OccurredAt time.Time
	RecordedAt time.Time
}

type TransactionRecord struct {
	EventID    string
	SourceIP   string
	DeviceID   string
	Country    string
	Currency   string
	Amount     *float64
	Latitude   *float64
	Longitude  *float64
	OccurredAt time.Time
	RecordedAt time.Time
}
