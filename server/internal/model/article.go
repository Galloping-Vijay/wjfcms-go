package model

type Article struct {
	BaseBigModel
	CategoryID  uint      `json:"category_id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Content     string    `json:"content"`
	Markdown    string    `json:"markdown"`
	Description string    `json:"description"`
	Keywords    string    `json:"keywords"`
	Cover       string    `json:"cover"`
	IsTop       bool      `json:"is_top"`
	Status      int8      `json:"status"`
	IsBaijiahao bool      `json:"is_baijiahao"`
	Click       uint      `json:"click"`
	Category    *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}
