package model

type Comment struct {
	BaseModel
	UserID    uint     `json:"user_id"`
	Type      int8     `json:"type"`
	Pid       uint     `json:"pid"`
	OriginID  uint     `json:"origin_id"`
	ArticleID uint     `json:"article_id"`
	Content   string   `json:"content"`
	Status    int8     `json:"status"`
	Cai       uint     `json:"cai"`
	Zan       uint     `json:"zan"`
	User      *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Article   *Article `json:"article,omitempty" gorm:"foreignKey:ArticleID"`
}
