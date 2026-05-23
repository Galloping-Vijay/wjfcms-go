package handler

import (
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	response.OK(c, "OK", gin.H{"status": "up"})
}
