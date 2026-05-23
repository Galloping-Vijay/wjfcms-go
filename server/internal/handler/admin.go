package handler

import (
	"net/http"
	"strings"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

type adminRequest struct {
	Account   string `json:"account" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password"`
	RoleNames string `json:"role_names"`
	RoleIDs   []uint `json:"role_ids"`
	Tel       string `json:"tel"`
	Email     string `json:"email"`
	Sex       int8   `json:"sex"`
	Status    int8   `json:"status"`
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) Index(c *gin.Context) {
	var admins []model.Admin
	query := h.db.Model(&model.Admin{})
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("account LIKE ? OR username LIKE ? OR tel LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if account := c.Query("account"); account != "" {
		query = query.Where("account LIKE ?", "%"+account+"%")
	}
	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if sex := c.Query("sex"); sex != "" {
		query = query.Where("sex = ?", sex)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&admins).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.fillRoleIDs(admins); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", admins, total)
}

func (h *AdminHandler) Store(c *gin.Context) {
	var req adminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写管理员账号、昵称和密码")
		return
	}
	if req.Password == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写密码")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	roleNames, err := h.roleNames(req.RoleIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if len(roleNames) > 0 {
		req.RoleNames = strings.Join(roleNames, ",")
	}

	admin := model.Admin{
		Account: req.Account, Username: req.Username, Password: string(hash), RoleNames: req.RoleNames,
		Tel: req.Tel, Email: req.Email, Sex: req.Sex, Status: req.Status,
	}
	if admin.Status == 0 {
		admin.Status = 1
	}
	if err := h.db.Create(&admin).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.syncRoles(admin.ID, req.RoleIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", admin)
}

func (h *AdminHandler) Update(c *gin.Context) {
	var admin model.Admin
	if err := h.db.First(&admin, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "管理员不存在")
		return
	}
	var req adminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写管理员信息")
		return
	}
	roleNames, err := h.roleNames(req.RoleIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if len(roleNames) > 0 {
		req.RoleNames = strings.Join(roleNames, ",")
	}
	updates := map[string]any{
		"account": req.Account, "username": req.Username, "role_names": req.RoleNames,
		"tel": req.Tel, "email": req.Email, "sex": req.Sex, "status": req.Status,
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			return
		}
		updates["password"] = string(hash)
	}
	if err := h.db.Model(&admin).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.syncRoles(admin.ID, req.RoleIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", admin)
}

func (h *AdminHandler) UpdatePassword(c *gin.Context) {
	var admin model.Admin
	if err := h.db.First(&admin, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "管理员不存在")
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
	if err := h.db.Model(&admin).Update("password", string(hash)).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "密码修改成功", nil)
}

func (h *AdminHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.Admin{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *AdminHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.Admin{})
}

func (h *AdminHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Admin{})
}

func (h *AdminHandler) fillRoleIDs(admins []model.Admin) error {
	for index := range admins {
		var roleIDs []uint
		if err := h.db.Model(&model.ModelHasRole{}).
			Where("model_type = ? AND model_id = ?", "App\\Models\\Admin", admins[index].ID).
			Pluck("role_id", &roleIDs).Error; err != nil {
			return err
		}
		admins[index].RoleIDs = roleIDs
	}
	return nil
}

func (h *AdminHandler) roleNames(roleIDs []uint) ([]string, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var roles []model.Role
	if err := h.db.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name)
	}
	return names, nil
}

func (h *AdminHandler) syncRoles(adminID uint64, roleIDs []uint) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("model_type = ? AND model_id = ?", "App\\Models\\Admin", adminID).
			Delete(&model.ModelHasRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if err := tx.Create(&model.ModelHasRole{
				RoleID:    roleID,
				ModelType: "App\\Models\\Admin",
				ModelID:   adminID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
