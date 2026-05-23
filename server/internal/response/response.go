package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Data       any    `json:"data,omitempty"`
	Count      int64  `json:"count,omitempty"`
	Meta       any    `json:"meta,omitempty"`
	CreateTime string `json:"create_time"`
}

func OK(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, Body{
		Code:       0,
		Msg:        msg,
		Data:       data,
		CreateTime: now(),
	})
}

func Page(c *gin.Context, msg string, data any, count int64) {
	c.JSON(http.StatusOK, Body{
		Code:       0,
		Msg:        msg,
		Data:       data,
		Count:      count,
		CreateTime: now(),
	})
}

func Error(c *gin.Context, status int, code int, msg string) {
	c.JSON(status, Body{
		Code:       code,
		Msg:        msg,
		CreateTime: now(),
	})
}

func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
