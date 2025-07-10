package entity

import (
	"time"
)

type User struct {
	UUID                  string    `bson:"uuid" json:"uuid"`
	PolygonWallet         string    `bson:"polygon_wallet" json:"polygon_wallet"`
	Name                  string    `bson:"name" json:"name"`
	Topics                []string  `bson:"topics" json:"topics"`
	Banner                string    `bson:"banner" json:"banner"`
	Avatar                string    `bson:"avatar" json:"avatar"`
	BackgroundColor       *string   `bson:"background_color,omitempty" json:"background_color,omitempty"`
	BackgroundImage       *string   `bson:"background_image,omitempty" json:"background_image,omitempty"`
	ButtonBackgroundColor string    `bson:"button_background_color" json:"button_background_color"`
	ButtonTextColor       string    `bson:"button_text_color" json:"button_text_color"`
	CreatedAt             time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt             time.Time `bson:"updated_at" json:"updated_at"`
	TelegramID            string    `bson:"telegram_id" json:"telegram_id"`
}

type RegisterUserRequest struct {
	PolygonWallet string   `json:"polygon_wallet"`
	Topics        []string `json:"topics"`
	Name          string   `json:"name"`
	TelegramID    string   `json:"telegram_id"`
}

type RegisterUserResponse struct {
	StreamerUUID string `json:"streamer_uuid"`
}

type UpdateUserRequest struct {
	Banner                string  `json:"banner"`
	Name                  string  `json:"name"`
	BackgroundColor       *string `json:"background_color,omitempty"`
	BackgroundImage       *string `json:"background_image,omitempty"`
	ButtonBackgroundColor string  `json:"button_background_color"`
	ButtonTextColor       string  `json:"button_text_color"`
	Avatar                string  `json:"avatar"`
	UUID                  string  `json:"-"`
}

type UserProfileResponse struct {
	Banner                string   `json:"banner"`
	Name                  string   `json:"name"`
	BackgroundColor       *string  `json:"background_color,omitempty"`
	BackgroundImage       *string  `json:"background_image,omitempty"`
	ButtonBackgroundColor string   `json:"button_background_color"`
	ButtonTextColor       string   `json:"button_text_color"`
	Avatar                string   `json:"avatar"`
	Topics                []string `json:"topics"`
}
