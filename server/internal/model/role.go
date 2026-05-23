package model

type Role struct {
	BaseModel
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int8   `json:"status"`
	GuardName   string `json:"guard_name"`
}
