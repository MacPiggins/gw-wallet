package model

import "time"

type Transfer struct {
	TransactionID string    `json:"transaction_id" bson:"transaction_id"`
	Sender        string    `json:"sender" bson:"sender"`
	Receiver      string    `json:"receiver" bson:"receiver"`
	Amount        int64     `json:"amount" bson:"amount"`
	Currency      string    `json:"currency" bson:"currency"`
	OccurredAt    time.Time `json:"occurred_at" bson:"occurred_at"`
	ReceivedAt    time.Time `json:"-" bson:"received_at"`
}
