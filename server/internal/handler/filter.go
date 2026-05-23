package handler

import (
	"net/http"

	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func applyDeleteFilter(query *gorm.DB, value string) *gorm.DB {
	switch value {
	case "1":
		return query.Unscoped().Where("deleted_at IS NOT NULL")
	case "2":
		return query.Unscoped()
	default:
		return query
	}
}

func restoreByID(c *gin.Context, db *gorm.DB, model any) {
	if err := db.Unscoped().Model(model).Where("id = ?", c.Param("id")).Update("deleted_at", nil).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func forceDeleteByID(c *gin.Context, db *gorm.DB, model any) {
	if err := db.Unscoped().Delete(model, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}
