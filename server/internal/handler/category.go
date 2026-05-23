package handler

import (
	"net/http"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	db *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

func (h *CategoryHandler) Index(c *gin.Context) {
	var categories []model.Category
	query := h.db.Model(&model.Category{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)

	err := query.Order("sort DESC, id DESC").Find(&categories).Error
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", categories, total)
}

func (h *CategoryHandler) Store(c *gin.Context) {
	var category model.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写分类信息")
		return
	}
	if err := h.db.Create(&category).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", category)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	var category model.Category
	if err := h.db.First(&category, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "分类不存在")
		return
	}
	if err := c.ShouldBindJSON(&category); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写分类信息")
		return
	}
	if err := h.db.Save(&category).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", category)
}

func (h *CategoryHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.Category{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *CategoryHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.Category{})
}

func (h *CategoryHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Category{})
}
