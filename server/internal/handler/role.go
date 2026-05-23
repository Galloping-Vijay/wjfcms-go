package handler

import (
	"net/http"
	"strconv"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RoleHandler struct {
	db *gorm.DB
}

type rolePayload struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Status        int8   `json:"status"`
	GuardName     string `json:"guard_name"`
	PermissionIDs []uint `json:"permission_ids"`
}

func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{db: db}
}

func (h *RoleHandler) Index(c *gin.Context) {
	var roles []model.Role
	query := h.db.Model(&model.Role{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if description := c.Query("description"); description != "" {
		query = query.Where("description LIKE ?", "%"+description+"%")
	}
	if guardName := c.Query("guard_name"); guardName != "" {
		query = query.Where("guard_name = ?", guardName)
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)
	if err := query.Order("id DESC").Find(&roles).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", roles, total)
}

func (h *RoleHandler) Show(c *gin.Context) {
	var role model.Role
	if err := h.db.First(&role, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "角色不存在")
		return
	}
	var permissionIDs []uint
	if err := h.db.Model(&model.RoleHasPermission{}).
		Where("role_id = ?", role.ID).
		Pluck("permission_id", &permissionIDs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "获取成功", gin.H{
		"role":           role,
		"permission_ids": permissionIDs,
	})
}

func (h *RoleHandler) Store(c *gin.Context) {
	var req rolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写角色信息")
		return
	}
	role := model.Role{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		GuardName:   req.GuardName,
	}
	if role.GuardName == "" {
		role.GuardName = "admin"
	}
	if role.Status == 0 {
		role.Status = 1
	}
	if err := h.db.Create(&role).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.syncPermissions(role.ID, req.PermissionIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	var role model.Role
	if err := h.db.First(&role, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "角色不存在")
		return
	}
	var req rolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写角色信息")
		return
	}
	updates := map[string]any{
		"name": req.Name, "description": req.Description, "status": req.Status, "guard_name": req.GuardName,
	}
	if updates["guard_name"] == "" {
		updates["guard_name"] = "admin"
	}
	if err := h.db.Model(&role).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.syncPermissions(role.ID, req.PermissionIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", role)
}

func (h *RoleHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.Role{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *RoleHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.Role{})
}

func (h *RoleHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Role{})
}

func (h *RoleHandler) PermissionTree(c *gin.Context) {
	guard := c.DefaultQuery("guard_name", "admin")
	roleID, _ := strconv.Atoi(c.DefaultQuery("role_id", "0"))
	var permissions []model.Permission
	if err := h.db.Where("guard_name = ?", guard).
		Order("level ASC, sort_order DESC, id ASC").
		Find(&permissions).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	checked := make(map[uint]bool)
	if roleID > 0 {
		var ids []uint
		if err := h.db.Model(&model.RoleHasPermission{}).
			Where("role_id = ?", roleID).
			Pluck("permission_id", &ids).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			return
		}
		for _, id := range ids {
			checked[id] = true
		}
	}

	response.OK(c, "获取成功", buildRolePermissionTree(permissions, 0, checked))
}

func (h *RoleHandler) syncPermissions(roleID uint, permissionIDs []uint) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleHasPermission{}).Error; err != nil {
			return err
		}
		for _, permissionID := range permissionIDs {
			if err := tx.Create(&model.RoleHasPermission{RoleID: roleID, PermissionID: permissionID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type rolePermissionNode struct {
	ID       uint                 `json:"id"`
	Name     string               `json:"name"`
	Level    int                  `json:"level"`
	ParentID int                  `json:"parent_id"`
	Checked  bool                 `json:"checked"`
	Children []rolePermissionNode `json:"children"`
}

func buildRolePermissionTree(items []model.Permission, parentID int, checked map[uint]bool) []rolePermissionNode {
	nodes := make([]rolePermissionNode, 0)
	for _, item := range items {
		if item.ParentID == parentID {
			nodes = append(nodes, rolePermissionNode{
				ID: item.ID, Name: item.Name, Level: item.Level, ParentID: item.ParentID,
				Checked:  checked[item.ID],
				Children: buildRolePermissionTree(items, int(item.ID), checked),
			})
		}
	}
	return nodes
}
