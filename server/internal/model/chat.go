package model

type Chat struct {
	BaseModel
	Content string `json:"content"`
}
