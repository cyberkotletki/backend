package entity

import "time"

type Wish struct {
	UUID         string    `bson:"uuid" json:"uuid"`
	StreamerUUID string    `bson:"streamer_uuid" json:"streamer_uuid"`
	WishURL      *string   `bson:"wish_url,omitempty" json:"wish_url,omitempty"`
	Name         string    `bson:"name" json:"name"`
	Description  *string   `bson:"description,omitempty" json:"description,omitempty"`
	Image        string    `bson:"image" json:"image"`
	PolTarget    float64   `bson:"pol_target" json:"pol_target"`
	PolAmount    float64   `bson:"pol_amount" json:"pol_amount"`
	IsPriority   bool      `bson:"is_priority" json:"is_priority"`
	Status       string    `bson:"status" json:"status"` // pending, active, complete, deleted
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
}

type AddWishRequest struct {
	WishURL     *string `json:"wish_url,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Image       string  `json:"image"`
	PolTarget   float64 `json:"pol_target"`
	IsPriority  bool    `json:"is_priority"`
	UserUUID    string  `json:"-"`
}

type AddWishResponse struct {
	WishUUID string `json:"wish_uuid"`
}

type UpdateWishRequest struct {
	WishUUID   string `json:"wish_uuid"`
	Image      string `json:"image"`
	IsPriority bool   `json:"is_priority"`
	UserUUID   string `json:"-"`
}

type WishResponse struct {
	UUID        string  `json:"uuid"`
	WishURL     *string `json:"wish_url,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Image       string  `json:"image"`
	PolTarget   float64 `json:"pol_target"`
	PolAmount   float64 `json:"pol_amount"`
	IsPriority  bool    `json:"is_priority"`
}

type GetWishesResponse struct {
	Wishes []WishResponse `json:"wishes"`
}
