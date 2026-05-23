package handler

import (
	"net/http"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TagHandler struct {
	db *gorm.DB
}

func NewTagHandler(db *gorm.DB) *TagHandler {
	return &TagHandler{db: db}
}

func (h *TagHandler) Index(c *gin.Context) {
	var tags []model.Tag
	query := h.db.Model(&model.Tag{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)

	if err := query.Order("id DESC").Find(&tags).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", tags, total)
}

func (h *TagHandler) Store(c *gin.Context) {
	var tag model.Tag
	if err := c.ShouldBindJSON(&tag); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写标签名称")
		return
	}
	if err := h.db.Create(&tag).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", tag)
}

func (h *TagHandler) Update(c *gin.Context) {
	var tag model.Tag
	if err := h.db.First(&tag, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "标签不存在")
		return
	}
	if err := c.ShouldBindJSON(&tag); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写标签名称")
		return
	}
	if err := h.db.Save(&tag).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", tag)
}

func (h *TagHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.Tag{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *TagHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.Tag{})
}

func (h *TagHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Tag{})
}
