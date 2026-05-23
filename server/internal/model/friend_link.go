package model

type FriendLink struct {
	BaseModel
	Name     string `json:"name"`
	URL      string `json:"url"`
	Email    string `json:"email"`
	ClientIP string `json:"client_ip"`
	Sort     int8   `json:"sort"`
	Status   int8   `json:"status"`
}
