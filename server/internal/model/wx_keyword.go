package model

import "time"

type WxKeyword struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	KeyName   string    `json:"key_name"`
	KeyValue  string    `json:"key_value"`
	Sort      int       `json:"sort"`
	Status    int8      `json:"status"`
}
