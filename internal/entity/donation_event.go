package entity

import "time"

// DonationEvent описывает событие доната для отправки в брокере сообщений
// UUID — идентификатор доната, StreamerUUID — получатель, DonorUsername — имя донатера (может быть пустым),
// Amount — сумма, WishUUID — цель доната (может быть пустым), Message — сообщение (может быть пустым),
// Datetime — время события

type DonationEvent struct {
	UUID          string    `json:"uuid"`
	StreamerUUID  string    `json:"streamer_uuid"`
	DonorUsername string    `json:"donor_username,omitempty"`
	Amount        float64   `json:"amount"`
	WishUUID      string    `json:"wish_uuid,omitempty"`
	Message       string    `json:"message,omitempty"`
	Datetime      time.Time `json:"datetime"`
}
