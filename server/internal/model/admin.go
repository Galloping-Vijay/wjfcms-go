package model

type Admin struct {
	BaseBigModel
	Account   string `json:"account"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	RoleNames string `json:"role_names"`
	Tel       string `json:"tel"`
	Email     string `json:"email"`
	Sex       int8   `json:"sex"`
	Status    int8   `json:"status"`
	RoleIDs   []uint `json:"role_ids,omitempty" gorm:"-"`
}
