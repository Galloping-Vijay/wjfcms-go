package model

type Permission struct {
	BaseModel
	Name        string       `json:"name"`
	GuardName   string       `json:"guard_name"`
	SortOrder   int8         `json:"sort_order"`
	URL         string       `json:"url"`
	Level       int          `json:"level"`
	Icon        string       `json:"icon"`
	ParentID    int          `json:"parent_id"`
	DisplayMenu int8         `json:"display_menu"`
	Children    []Permission `json:"children,omitempty" gorm:"-"`
}
