package entity

import "time"

type History struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	StreamerUUID string    `bson:"streamer_uuid" json:"streamer_uuid"`
	Type         string    `bson:"type" json:"type"` // donate/withdraw
	Username     *string   `bson:"username,omitempty" json:"username,omitempty"`
	Datetime     time.Time `bson:"datetime" json:"datetime"`
	Amount       float64   `bson:"amount" json:"amount"`
	WishUUID     *string   `bson:"wish_uuid,omitempty" json:"wish_uuid,omitempty"`
	Message      *string   `bson:"message,omitempty" json:"message,omitempty"`
}

type HistoryItem struct {
	Type     string  `json:"type"`
	Username *string `json:"username,omitempty"`
	Datetime string  `json:"datetime"`
	Amount   float64 `json:"amount"`
	WishUUID *string `json:"wish_uuid,omitempty"`
	Message  *string `json:"message,omitempty"`
}

type UserHistoryResponse struct {
	Page    int           `json:"page"`
	History []HistoryItem `json:"history"`
}
