package router

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/handler"
	"wjfcm-go/internal/middleware"
	"wjfcm-go/internal/requestlog"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB) *gin.Engine {
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	if cfg.App.ConsoleColor {
		gin.ForceConsoleColor()
	} else {
		gin.DisableConsoleColor()
	}

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"safeHTML": handler.SafeHTML,
		"splitTags": func(value string) []string {
			parts := strings.Split(value, ",")
			tags := make([]string, 0, len(parts))
			for _, part := range parts {
				if tag := strings.TrimSpace(part); tag != "" {
					tags = append(tags, tag)
				}
			}
			return tags
		},
		"config": func(configs map[string]string, key string, fallback string) string {
			if value := strings.TrimSpace(configs[key]); value != "" {
				return value
			}
			return fallback
		},
	})
	r.LoadHTMLGlob("templates/*.tmpl")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", requestlog.HeaderName},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	requestLogStore := requestlog.NewStore(cfg.Log)
	requestlog.SetDefault(requestLogStore)
	r.Use(requestlog.Middleware(requestLogStore))
	r.Use(installGuard(db))

	auth := handler.NewAuthHandler(cfg, db)
	homeAuth := handler.NewHomeAuthHandler(cfg, db)
	articles := handler.NewArticleHandler(cfg, db)
	categories := handler.NewCategoryHandler(db)
	tags := handler.NewTagHandler(db)
	comments := handler.NewCommentHandler(db)
	users := handler.NewUserHandler(db)
	admins := handler.NewAdminHandler(db)
	roles := handler.NewRoleHandler(db)
	permissions := handler.NewPermissionHandler(db)
	content := handler.NewContentHandler(cfg, db)
	uploads := handler.NewUploadHandler(cfg)
	seoPages := handler.NewSEOPageHandler(db)
	wechat := handler.NewWechatHandler(cfg, db)
	baidu := handler.NewBaiduHandler(cfg)
	installer := handler.NewInstallHandler(cfg, db)
	requestLogs := handler.NewRequestLogHandler()

	r.Static("/uploads", cfg.Upload.PublicDir+"/"+cfg.Upload.BasePath)
	r.Static("/images", cfg.Upload.PublicDir+"/images")
	r.StaticFile("/favicon.ico", cfg.Upload.PublicDir+"/favicon.ico")
	r.GET("/install", installer.Show)
	r.POST("/install", installer.Store)
	r.GET("/", seoPages.Index)
	r.GET("/category/:id", seoPages.Category)
	r.GET("/tag/:id", seoPages.Tag)
	r.GET("/search", seoPages.Search)
	r.GET("/archive", seoPages.Archive)
	r.GET("/chat", seoPages.Chat)
	r.GET("/login", seoPages.Login)
	r.GET("/register", seoPages.Register)
	r.GET("/forgot-password", seoPages.ForgotPassword)
	r.GET("/user", seoPages.User)
	r.GET("/blank", seoPages.Blank)
	r.GET("/article/:id", seoPages.Article)
	r.GET("/robots.txt", seoPages.Robots)
	r.GET("/sitemap.xml", seoPages.Sitemap)
	r.GET("/tools/linkSubmit", content.SubmitBaiduLinks)
	r.Any("/baidu/serve", baidu.Serve)
	r.GET("/wechat", wechat.Verify)
	r.POST("/wechat", wechat.Serve)
	api := r.Group("/api")
	api.GET("/health", handler.Health)

	home := api.Group("/home")
	home.GET("/articles", articles.PublicIndex)
	home.GET("/archive", articles.PublicArchive)
	home.GET("/articles/:id", articles.PublicShow)
	home.GET("/categories", categories.Index)
	home.GET("/tags", tags.Index)
	home.GET("/navs", content.Navs)
	home.GET("/friend-links", content.FriendLinks)
	home.POST("/friend-links", content.ApplyFriendLink)
	home.GET("/chats", content.Chats)
	home.GET("/system-configs", content.SystemConfigs)
	home.GET("/comments", comments.PublicIndex)
	home.GET("/auth/captcha", homeAuth.Captcha)
	home.POST("/auth/register", homeAuth.Register)
	home.POST("/auth/login", homeAuth.Login)
	home.POST("/auth/email-code", homeAuth.SendEmailCode)
	home.POST("/auth/reset-password", homeAuth.ResetPassword)
	home.GET("/auth/:provider", homeAuth.OAuthRedirect)
	home.GET("/auth/:provider/callback", homeAuth.OAuthCallback)

	homeUser := home.Group("/")
	homeUser.Use(middleware.UserAuth(cfg))
	homeUser.GET("/profile", homeAuth.Profile)
	homeUser.PUT("/profile", homeAuth.UpdateProfile)
	homeUser.POST("/upload/image", uploads.Image)
	homeUser.POST("/comments", comments.PublicStore)
	homeUser.POST("/comments/:id/action", comments.PublicAction)

	admin := api.Group("/admin")
	admin.POST("/auth/login", auth.Login)
	admin.POST("/auth/refresh", auth.Refresh)
	admin.Use(middleware.AdminAuth(cfg))
	admin.Use(middleware.AdminPermission(db))
	admin.GET("/profile", auth.Profile)
	admin.GET("/request-logs/:request_id", requestLogs.Show)
	admin.PUT("/profile", auth.UpdateProfile)
	admin.PUT("/password", auth.UpdatePassword)
	admin.POST("/auth/logout", auth.Logout)
	admin.POST("/upload/image", uploads.Image)
	admin.POST("/tools/baidu-submit", content.SubmitBaiduLinks)
	admin.GET("/menus", permissions.Menu)
	admin.GET("/articles", articles.Index)
	admin.POST("/articles", articles.Store)
	admin.POST("/articles/replace", articles.Replace)
	admin.GET("/articles/:id", articles.Show)
	admin.PUT("/articles/:id", articles.Update)
	admin.POST("/articles/:id/baijiahao", articles.PublishBaijiahao)
	admin.DELETE("/articles/:id", articles.Destroy)
	admin.POST("/articles/:id/restore", articles.Restore)
	admin.DELETE("/articles/:id/force", articles.ForceDelete)
	admin.GET("/categories", categories.Index)
	admin.POST("/categories", categories.Store)
	admin.PUT("/categories/:id", categories.Update)
	admin.DELETE("/categories/:id", categories.Destroy)
	admin.POST("/categories/:id/restore", categories.Restore)
	admin.DELETE("/categories/:id/force", categories.ForceDelete)
	admin.GET("/tags", tags.Index)
	admin.POST("/tags", tags.Store)
	admin.PUT("/tags/:id", tags.Update)
	admin.DELETE("/tags/:id", tags.Destroy)
	admin.POST("/tags/:id/restore", tags.Restore)
	admin.DELETE("/tags/:id/force", tags.ForceDelete)
	admin.GET("/comments", comments.Index)
	admin.POST("/comments/replace", comments.Replace)
	admin.PUT("/comments/:id", comments.Update)
	admin.DELETE("/comments/:id", comments.Destroy)
	admin.POST("/comments/:id/restore", comments.Restore)
	admin.DELETE("/comments/:id/force", comments.ForceDelete)
	admin.GET("/users", users.Index)
	admin.POST("/users", users.Store)
	admin.PUT("/users/:id", users.Update)
	admin.PUT("/users/:id/password", users.UpdatePassword)
	admin.DELETE("/users/:id", users.Destroy)
	admin.POST("/users/:id/restore", users.Restore)
	admin.DELETE("/users/:id/force", users.ForceDelete)
	admin.GET("/admins", admins.Index)
	admin.POST("/admins", admins.Store)
	admin.PUT("/admins/:id", admins.Update)
	admin.PUT("/admins/:id/password", admins.UpdatePassword)
	admin.DELETE("/admins/:id", admins.Destroy)
	admin.POST("/admins/:id/restore", admins.Restore)
	admin.DELETE("/admins/:id/force", admins.ForceDelete)
	admin.GET("/roles", roles.Index)
	admin.POST("/roles", roles.Store)
	admin.GET("/roles/permission-tree", roles.PermissionTree)
	admin.GET("/roles/:id", roles.Show)
	admin.PUT("/roles/:id", roles.Update)
	admin.DELETE("/roles/:id", roles.Destroy)
	admin.POST("/roles/:id/restore", roles.Restore)
	admin.DELETE("/roles/:id/force", roles.ForceDelete)
	admin.GET("/permissions", permissions.Index)
	admin.GET("/permissions/menu", permissions.Menu)
	admin.POST("/permissions", permissions.Store)
	admin.PUT("/permissions/:id", permissions.Update)
	admin.DELETE("/permissions/:id", permissions.Destroy)
	admin.POST("/permissions/:id/restore", permissions.Restore)
	admin.DELETE("/permissions/:id/force", permissions.ForceDelete)
	admin.GET("/navs", content.Navs)
	admin.POST("/navs", content.StoreNav)
	admin.PUT("/navs/:id", content.UpdateNav)
	admin.DELETE("/navs/:id", content.DeleteNav)
	admin.POST("/navs/:id/restore", content.RestoreNav)
	admin.DELETE("/navs/:id/force", content.ForceDeleteNav)
	admin.GET("/friend-links", content.FriendLinks)
	admin.POST("/friend-links", content.StoreFriendLink)
	admin.PUT("/friend-links/:id", content.UpdateFriendLink)
	admin.DELETE("/friend-links/:id", content.DeleteFriendLink)
	admin.POST("/friend-links/:id/restore", content.RestoreFriendLink)
	admin.DELETE("/friend-links/:id/force", content.ForceDeleteFriendLink)
	admin.GET("/chats", content.Chats)
	admin.POST("/chats", content.StoreChat)
	admin.PUT("/chats/:id", content.UpdateChat)
	admin.DELETE("/chats/:id", content.DeleteChat)
	admin.POST("/chats/:id/restore", content.RestoreChat)
	admin.DELETE("/chats/:id/force", content.ForceDeleteChat)
	admin.GET("/system-configs", content.SystemConfigs)
	admin.PUT("/system-configs/:id", content.UpdateSystemConfig)
	admin.GET("/wx-keywords", content.WxKeywords)
	admin.POST("/wx-keywords", content.StoreWxKeyword)
	admin.PUT("/wx-keywords/:id", content.UpdateWxKeyword)
	admin.DELETE("/wx-keywords/:id", content.DeleteWxKeyword)
	admin.POST("/wx-keywords/:id/restore", content.RestoreWxKeyword)
	admin.DELETE("/wx-keywords/:id/force", content.ForceDeleteWxKeyword)

	r.NoRoute(servePublicRootFile(cfg.Upload.PublicDir))

	return r
}

func installGuard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/install" || strings.HasPrefix(path, "/images/") || path == "/favicon.ico" {
			c.Next()
			return
		}
		if handler.IsInstalled(db) {
			if db == nil {
				c.String(http.StatusServiceUnavailable, "database unavailable, please check .env and restart service")
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusFound, "/install")
			c.Abort()
			return
		}
		c.String(http.StatusServiceUnavailable, "system is not installed")
		c.Abort()
	}
}

func servePublicRootFile(publicDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if name == "" || strings.ContainsAny(name, `/\`) {
			c.Status(http.StatusNotFound)
			return
		}

		clean := filepath.Clean(name)
		if clean != name || clean == "." || strings.HasPrefix(clean, "..") {
			c.Status(http.StatusNotFound)
			return
		}

		fullPath := filepath.Join(publicDir, clean)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}

		c.File(fullPath)
	}
}
