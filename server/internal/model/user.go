package model

import "time"

type User struct {
	BaseBigModel
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Sex             int8       `json:"sex"`
	Tel             string     `json:"tel"`
	City            string     `json:"city"`
	Intro           string     `json:"intro"`
	Avatar          string     `json:"avatar"`
	ProviderID      string     `json:"provider_id"`
	Provider        string     `json:"provider"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	Password        string     `json:"-"`
	RememberToken   string     `json:"-"`
}
