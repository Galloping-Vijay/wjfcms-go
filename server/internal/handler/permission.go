package handler

import (
	"net/http"
	"sort"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"
	"wjfcm-go/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	db *gorm.DB
}

func NewPermissionHandler(db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{db: db}
}

func (h *PermissionHandler) Index(c *gin.Context) {
	var permissions []model.Permission
	query := h.db.Model(&model.Permission{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if guard := c.Query("guard_name"); guard != "" {
		query = query.Where("guard_name = ?", guard)
	}
	if displayMenu := c.Query("display_menu"); displayMenu != "" {
		query = query.Where("display_menu = ?", displayMenu)
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)
	if err := query.Order("level ASC, sort_order DESC, id ASC").Find(&permissions).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", permissions, total)
}

func (h *PermissionHandler) Menu(c *gin.Context) {
	var permissions []model.Permission
	guard := c.DefaultQuery("guard_name", "admin")
	query := h.db.Where("guard_name = ? AND display_menu = 1", guard)

	if adminID := c.GetUint64("admin_id"); adminID > 0 {
		if !service.IsSuperAdmin(adminID) {
			permissionIDs, err := service.AdminPermissionIDs(h.db, adminID, true)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, 1, err.Error())
				return
			}
			if len(permissionIDs) == 0 {
				response.OK(c, "获取成功", []model.Permission{})
				return
			}
			query = query.Where("id IN ?", permissionIDs)
		}
	}

	if err := query.
		Order("sort_order DESC, id ASC").
		Find(&permissions).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "获取成功", buildPermissionTree(permissions, 0))
}

func (h *PermissionHandler) Store(c *gin.Context) {
	var permission model.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写权限信息")
		return
	}
	if permission.GuardName == "" {
		permission.GuardName = "admin"
	}
	level, err := h.permissionLevel(permission.ParentID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 1, err.Error())
		return
	}
	permission.Level = level
	if err := h.db.Create(&permission).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", permission)
}

func (h *PermissionHandler) Update(c *gin.Context) {
	var permission model.Permission
	if err := h.db.First(&permission, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "权限不存在")
		return
	}
	var req model.Permission
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写权限信息")
		return
	}
	if req.GuardName == "" {
		req.GuardName = "admin"
	}
	if req.ParentID == int(permission.ID) {
		response.Error(c, http.StatusBadRequest, 1, "上级权限不能选择自己")
		return
	}
	level, err := h.permissionLevel(req.ParentID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 1, err.Error())
		return
	}
	if permission.Level != level {
		var childCount int64
		h.db.Model(&model.Permission{}).Where("parent_id = ?", permission.ID).Count(&childCount)
		if childCount > 0 {
			response.Error(c, http.StatusBadRequest, 1, "该权限存在子菜单，暂不支持修改层级")
			return
		}
	}
	updates := map[string]any{
		"name": req.Name, "guard_name": req.GuardName, "sort_order": req.SortOrder,
		"url": req.URL, "level": level, "icon": req.Icon, "parent_id": req.ParentID,
		"display_menu": req.DisplayMenu,
	}
	if err := h.db.Model(&permission).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", permission)
}

func (h *PermissionHandler) Destroy(c *gin.Context) {
	var count int64
	h.db.Model(&model.Permission{}).Where("parent_id = ?", c.Param("id")).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "存在子菜单，不能删除")
		return
	}
	h.db.Model(&model.RoleHasPermission{}).Where("permission_id = ?", c.Param("id")).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "存在使用该权限的角色，请先解除角色授权")
		return
	}
	h.db.Model(&model.ModelHasPermission{}).Where("permission_id = ?", c.Param("id")).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "存在使用该权限的管理员，请先解除管理员授权")
		return
	}
	if err := h.db.Delete(&model.Permission{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *PermissionHandler) Restore(c *gin.Context) {
	restoreByID(c, h.db, &model.Permission{})
}

func (h *PermissionHandler) ForceDelete(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Permission{})
}

func (h *PermissionHandler) permissionLevel(parentID int) (int, error) {
	if parentID == 0 {
		return 0, nil
	}
	var parent model.Permission
	if err := h.db.First(&parent, parentID).Error; err != nil {
		return 0, err
	}
	return parent.Level + 1, nil
}

func buildPermissionTree(items []model.Permission, parentID int) []model.Permission {
	children := make([]model.Permission, 0)
	for _, item := range items {
		if item.ParentID == parentID {
			item.Children = buildPermissionTree(items, int(item.ID))
			children = append(children, item)
		}
	}
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].SortOrder == children[j].SortOrder {
			return children[i].ID < children[j].ID
		}
		return children[i].SortOrder > children[j].SortOrder
	})
	return children
}
