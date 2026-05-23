package model

type Nav struct {
	BaseModel
	Pid    int    `json:"pid"`
	Sort   int    `json:"sort"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Target string `json:"target"`
	Icon   string `json:"icon"`
}
