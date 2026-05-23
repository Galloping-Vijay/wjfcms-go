package handler

import (
	"net/http"
	"time"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) Index(c *gin.Context) {
	var users []model.User
	query := h.db.Model(&model.User{})
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ? OR tel LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", users, total)
}

func (h *UserHandler) Store(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Tel      string `json:"tel"`
		Sex      int8   `json:"sex"`
		City     string `json:"city"`
		Intro    string `json:"intro"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写用户昵称、邮箱和密码")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	now := time.Now()
	user := model.User{
		Name: req.Name, Email: req.Email, Password: string(hash), Tel: req.Tel,
		Sex: req.Sex, City: req.City, Intro: req.Intro, Avatar: req.Avatar,
		EmailVerifiedAt: &now,
	}
	if user.Avatar == "" {
		user.Avatar = "/images/config/avatar_l.jpg"
	}
	if err := h.db.Create(&user).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", user)
}

func (h *UserHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.User{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *UserHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.User{})
}

func (h *UserHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.User{})
}

func (h *UserHandler) Update(c *gin.Context) {
	var user model.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "用户不存在")
		return
	}
	var req struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		Sex    int8   `json:"sex"`
		Tel    string `json:"tel"`
		City   string `json:"city"`
		Intro  string `json:"intro"`
		Avatar string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写用户信息")
		return
	}
	if err := h.db.Model(&user).Updates(map[string]any{
		"name": req.Name, "email": req.Email, "sex": req.Sex, "tel": req.Tel,
		"city": req.City, "intro": req.Intro, "avatar": req.Avatar,
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", user)
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var user model.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "用户不存在")
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写新密码")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.db.Model(&user).Update("password", string(hash)).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "密码修改成功", nil)
}
