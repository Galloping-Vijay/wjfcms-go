package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"
	"wjfcm-go/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AdminAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, 401, "请先登录")
			c.Abort()
			return
		}

		claims, err := service.ParseAdminToken(strings.TrimPrefix(header, "Bearer "), cfg)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 401, "登录状态已失效")
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("admin_account", claims.Account)
		c.Next()
	}
}

func UserAuth(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, 401, "请先登录")
			c.Abort()
			return
		}

		claims, err := service.ParseUserToken(strings.TrimPrefix(header, "Bearer "), cfg)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, 401, "登录状态已失效")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

func AdminPermission(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetUint64("admin_id")
		if service.IsSuperAdmin(adminID) {
			c.Next()
			return
		}

		candidates := legacyPermissionCandidates(c.Request.Method, c.FullPath())
		if len(candidates) == 0 {
			c.Next()
			return
		}

		var configuredCount int64
		if err := db.Model(&model.Permission{}).
			Where("guard_name = ? AND url IN ?", "admin", candidates).
			Count(&configuredCount).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			c.Abort()
			return
		}
		if configuredCount == 0 {
			c.Next()
			return
		}

		urls, err := service.AdminPermissionURLs(db, adminID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			c.Abort()
			return
		}
		allowed := make(map[string]bool, len(urls))
		for _, url := range urls {
			allowed[strings.TrimRight(url, "/")] = true
		}
		for _, candidate := range candidates {
			if allowed[candidate] {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, 403, "没有操作权限")
		c.Abort()
	}
}

func legacyPermissionCandidates(method string, fullPath string) []string {
	path := strings.TrimRight(fullPath, "/")
	method = strings.ToUpper(method)
	exact := map[string]string{
		"GET /api/admin/profile":                   "/admin/index/main",
		"PUT /api/admin/profile":                   "",
		"PUT /api/admin/password":                  "",
		"POST /api/admin/auth/logout":              "",
		"GET /api/admin/menus":                     "/admin/index/index",
		"POST /api/admin/upload/image":             "/admin/article/uploadImage",
		"GET /api/admin/articles":                  "/admin/article/index",
		"POST /api/admin/articles":                 "/admin/article/store",
		"POST /api/admin/articles/replace":         "/admin/article/replace",
		"GET /api/admin/articles/:id":              "/admin/article/show",
		"PUT /api/admin/articles/:id":              "/admin/article/update",
		"POST /api/admin/articles/:id/baijiahao":   "/admin/article/update",
		"DELETE /api/admin/articles/:id":           "/admin/article/destroy",
		"POST /api/admin/articles/:id/restore":     "/admin/article/update",
		"DELETE /api/admin/articles/:id/force":     "/admin/article/destroy",
		"GET /api/admin/categories":                "/admin/category/index",
		"POST /api/admin/categories":               "/admin/category/store",
		"PUT /api/admin/categories/:id":            "/admin/category/update",
		"DELETE /api/admin/categories/:id":         "/admin/category/destroy",
		"POST /api/admin/categories/:id/restore":   "/admin/category/update",
		"DELETE /api/admin/categories/:id/force":   "/admin/category/destroy",
		"GET /api/admin/tags":                      "/admin/tag/index",
		"POST /api/admin/tags":                     "/admin/tag/store",
		"PUT /api/admin/tags/:id":                  "/admin/tag/update",
		"DELETE /api/admin/tags/:id":               "/admin/tag/destroy",
		"POST /api/admin/tags/:id/restore":         "/admin/tag/update",
		"DELETE /api/admin/tags/:id/force":         "/admin/tag/destroy",
		"GET /api/admin/comments":                  "/admin/comment/index",
		"POST /api/admin/comments/replace":         "/admin/comment/replace",
		"PUT /api/admin/comments/:id":              "/admin/comment/update",
		"DELETE /api/admin/comments/:id":           "/admin/comment/destroy",
		"POST /api/admin/comments/:id/restore":     "/admin/comment/update",
		"DELETE /api/admin/comments/:id/force":     "/admin/comment/destroy",
		"GET /api/admin/users":                     "/admin/user/index",
		"POST /api/admin/users":                    "/admin/user/store",
		"PUT /api/admin/users/:id":                 "/admin/user/update",
		"PUT /api/admin/users/:id/password":        "/admin/user/update",
		"DELETE /api/admin/users/:id":              "/admin/user/destroy",
		"POST /api/admin/users/:id/restore":        "/admin/user/update",
		"DELETE /api/admin/users/:id/force":        "/admin/user/destroy",
		"GET /api/admin/admins":                    "/admin/admin/index",
		"POST /api/admin/admins":                   "/admin/admin/store",
		"PUT /api/admin/admins/:id":                "/admin/admin/update",
		"PUT /api/admin/admins/:id/password":       "/admin/admin/password",
		"DELETE /api/admin/admins/:id":             "/admin/admin/destroy",
		"POST /api/admin/admins/:id/restore":       "/admin/admin/update",
		"DELETE /api/admin/admins/:id/force":       "/admin/admin/destroy",
		"GET /api/admin/roles":                     "/admin/role/index",
		"POST /api/admin/roles":                    "/admin/role/store",
		"GET /api/admin/roles/:id":                 "/admin/role/show",
		"PUT /api/admin/roles/:id":                 "/admin/role/update",
		"DELETE /api/admin/roles/:id":              "/admin/role/destroy",
		"POST /api/admin/roles/:id/restore":        "/admin/role/update",
		"DELETE /api/admin/roles/:id/force":        "/admin/role/destroy",
		"GET /api/admin/permissions":               "/admin/permission/index",
		"GET /api/admin/permissions/menu":          "/admin/permission/index",
		"POST /api/admin/permissions":              "/admin/permission/store",
		"PUT /api/admin/permissions/:id":           "/admin/permission/update",
		"DELETE /api/admin/permissions/:id":        "/admin/permission/destroy",
		"POST /api/admin/permissions/:id/restore":  "/admin/permission/update",
		"DELETE /api/admin/permissions/:id/force":  "/admin/permission/destroy",
		"GET /api/admin/navs":                      "/admin/nav/index",
		"POST /api/admin/navs":                     "/admin/nav/store",
		"PUT /api/admin/navs/:id":                  "/admin/nav/update",
		"DELETE /api/admin/navs/:id":               "/admin/nav/destroy",
		"POST /api/admin/navs/:id/restore":         "/admin/nav/update",
		"DELETE /api/admin/navs/:id/force":         "/admin/nav/destroy",
		"GET /api/admin/friend-links":              "/admin/friendLinks/index",
		"POST /api/admin/friend-links":             "/admin/friendLinks/store",
		"PUT /api/admin/friend-links/:id":          "/admin/friendLinks/update",
		"DELETE /api/admin/friend-links/:id":       "/admin/friendLinks/destroy",
		"POST /api/admin/friend-links/:id/restore": "/admin/friendLinks/update",
		"DELETE /api/admin/friend-links/:id/force": "/admin/friendLinks/destroy",
		"GET /api/admin/chats":                     "/admin/chat/index",
		"POST /api/admin/chats":                    "/admin/chat/store",
		"PUT /api/admin/chats/:id":                 "/admin/chat/update",
		"DELETE /api/admin/chats/:id":              "/admin/chat/destroy",
		"POST /api/admin/chats/:id/restore":        "/admin/chat/update",
		"DELETE /api/admin/chats/:id/force":        "/admin/chat/destroy",
		"GET /api/admin/wx-keywords":               "/admin/weChat/keyword/index",
		"POST /api/admin/wx-keywords":              "/admin/weChat/keyword/store",
		"PUT /api/admin/wx-keywords/:id":           "/admin/weChat/keyword/update",
		"DELETE /api/admin/wx-keywords/:id":        "/admin/weChat/keyword/destroy",
		"POST /api/admin/wx-keywords/:id/restore":  "/admin/weChat/keyword/update",
		"DELETE /api/admin/wx-keywords/:id/force":  "/admin/weChat/keyword/destroy",
		"GET /api/admin/system-configs":            "/admin/systemConfig/basal",
		"PUT /api/admin/system-configs/:id":        "/admin/systemConfig/update",
		"GET /api/admin/roles/permission-tree":     "/admin/role/update",
	}
	primary := exact[method+" "+path]
	if primary == "" {
		return nil
	}
	return permissionURLVariants(primary)
}

func permissionURLVariants(url string) []string {
	url = strings.TrimRight(url, "/")
	items := []string{url}
	if strings.HasSuffix(url, "/index") {
		items = append(items, strings.TrimSuffix(url, "/index"))
	}
	aliases := map[string][]string{
		"/admin/comment/update": {"/admin/comment/replace"},
		"/admin/tag/update":     {"/admin/tag/upda"},
	}
	items = append(items, aliases[url]...)
	return append(items, regexp.MustCompile(`/+`).ReplaceAllString(url, "/"))
}
