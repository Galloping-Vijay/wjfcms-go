package model

type App struct {
	BaseModel
	Title       string `json:"title" gorm:"size:120;not null"`
	Author      string `json:"author" gorm:"size:60"`
	Description string `json:"description" gorm:"type:text"`
	URL         string `json:"url" gorm:"size:255;not null"`
	Icon        string `json:"icon" gorm:"size:255"`
	Target      string `json:"target" gorm:"size:20;default:_blank"`
	Sort        int    `json:"sort" gorm:"default:0"`
	Status      int8   `json:"status" gorm:"default:1"`
}
