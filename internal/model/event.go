package model

import "time"

type Event struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
	ReceivedAt time.Time `json:"-"`
}
