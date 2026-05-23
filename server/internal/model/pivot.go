package model

type ModelHasRole struct {
	RoleID    uint   `json:"role_id" gorm:"column:role_id"`
	ModelType string `json:"model_type" gorm:"column:model_type"`
	ModelID   uint64 `json:"model_id" gorm:"column:model_id"`
}

type RoleHasPermission struct {
	PermissionID uint `json:"permission_id" gorm:"column:permission_id"`
	RoleID       uint `json:"role_id" gorm:"column:role_id"`
}

type ModelHasPermission struct {
	PermissionID uint   `json:"permission_id" gorm:"column:permission_id"`
	ModelType    string `json:"model_type" gorm:"column:model_type"`
	ModelID      uint64 `json:"model_id" gorm:"column:model_id"`
}
