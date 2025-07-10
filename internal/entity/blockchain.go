package entity

import "time"

// BlockchainState хранит состояние синхронизации с блокчейном
type BlockchainState struct {
	ID                 string    `bson:"_id" json:"id"`
	LastProcessedBlock uint64    `bson:"last_processed_block" json:"last_processed_block"`
	UpdatedAt          time.Time `bson:"updated_at" json:"updated_at"`
}

// BlockchainEvent представляет событие блокчейна для сохранения в БД
type BlockchainEvent struct {
	ID          string    `bson:"_id" json:"id"` // уникальный идентификатор события
	BlockNumber uint64    `bson:"block_number" json:"block_number"`
	TxHash      string    `bson:"tx_hash" json:"tx_hash"`
	EventType   string    `bson:"event_type" json:"event_type"` // WishAdded, WishCompleted, WishDeleted
	UserUUID    string    `bson:"user_uuid" json:"user_uuid"`
	WishUUID    string    `bson:"wish_uuid" json:"wish_uuid"`
	ProcessedAt time.Time `bson:"processed_at" json:"processed_at"`
}
