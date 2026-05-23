package model

import "time"

type SystemConfig struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	Title      string    `json:"title"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Type       string    `json:"type"`
	ConfigType int8      `json:"config_type"`
	Status     int8      `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
