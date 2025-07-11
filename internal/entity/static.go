package entity

import "time"

type StaticFile struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	Type         string    `bson:"type" json:"type"` // avatar, banner, background, wish
	UploaderUUID string    `bson:"uploader_uuid" json:"uploader_uuid"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
}

type UploadStaticResponse struct {
	UUID string `json:"uuid"`
}

type StaticFileResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}
