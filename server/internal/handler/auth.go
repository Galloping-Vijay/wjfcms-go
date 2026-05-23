package handler

import (
	"net/http"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"
	"wjfcm-go/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	cfg config.Config
	db  *gorm.DB
}

type loginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(cfg config.Config, db *gorm.DB) *AuthHandler {
	return &AuthHandler{cfg: cfg, db: db}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请输入账号和密码")
		return
	}

	result, err := service.LoginAdmin(h.db, h.cfg, req.Account, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 1, err.Error())
		return
	}

	response.OK(c, "登录成功", result)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "缺少刷新令牌")
		return
	}
	claims, err := service.ParseAdminRefreshToken(req.RefreshToken, h.cfg)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 401, "刷新令牌已失效")
		return
	}
	var admin model.Admin
	if err := h.db.First(&admin, claims.AdminID).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, 401, "管理员不存在")
		return
	}
	if admin.Status != 1 {
		response.Error(c, http.StatusForbidden, 403, "账号已被禁用")
		return
	}
	token, err := service.MakeAdminToken(admin, h.cfg)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	refreshToken, err := service.MakeAdminRefreshToken(admin, h.cfg)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	permissions, err := service.AdminPermissionURLs(h.db, admin.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "刷新成功", service.LoginResult{
		Token:        token,
		RefreshToken: refreshToken,
		Admin:        admin,
		Permissions:  permissions,
	})
}

func (h *AuthHandler) Profile(c *gin.Context) {
	adminID := c.GetUint64("admin_id")
	var admin model.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "管理员不存在")
		return
	}
	permissions, err := service.AdminPermissionURLs(h.db, admin.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	response.OK(c, "获取成功", gin.H{
		"admin":       admin,
		"permissions": permissions,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	adminID := c.GetUint64("admin_id")
	var admin model.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "管理员不存在")
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
		Tel      string `json:"tel"`
		Email    string `json:"email"`
		Sex      int8   `json:"sex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写管理员资料")
		return
	}
	if err := h.db.Model(&admin).Updates(map[string]any{
		"username": req.Username,
		"tel":      req.Tel,
		"email":    req.Email,
		"sex":      req.Sex,
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	h.db.First(&admin, adminID)
	response.OK(c, "资料已保存", admin)
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	adminID := c.GetUint64("admin_id")
	var admin model.Admin
	if err := h.db.First(&admin, adminID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "管理员不存在")
		return
	}
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		Password    string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写原密码和新密码")
		return
	}
	if len(req.Password) < 6 {
		response.Error(c, http.StatusBadRequest, 1, "新密码至少 6 位")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.OldPassword)); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "原密码不正确")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.db.Model(&admin).Update("password", string(hash)).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "密码已修改，请重新登录", nil)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	response.OK(c, "退出成功", nil)
}
