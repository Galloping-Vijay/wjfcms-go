package model

type Category struct {
	BaseModel
	Name        string `json:"name"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	Sort        int8   `json:"sort"`
	Pid         int8   `json:"pid"`
}
