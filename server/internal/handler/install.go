package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/database"
	"wjfcm-go/internal/model"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type InstallHandler struct {
	cfg config.Config
	db  *gorm.DB
}

type installPageData struct {
	Title       string
	SiteName    string
	Year        int
	Installed   bool
	Error       string
	Success     string
	Form        installRequest
	RequireNote template.HTML
}

type installRequest struct {
	AppName        string
	AppURL         string
	AppPort        string
	DBHost         string
	DBPort         string
	DBDatabase     string
	DBUsername     string
	DBPassword     string
	DBPrefix       string
	PublicDir      string
	UploadBasePath string
	AdminAccount   string
	AdminUsername  string
	AdminPassword  string
	ConfirmPass    string
}

func NewInstallHandler(cfg config.Config, db *gorm.DB) *InstallHandler {
	return &InstallHandler{cfg: cfg, db: db}
}

func (h *InstallHandler) Show(c *gin.Context) {
	c.HTML(http.StatusOK, "seo_install.tmpl", h.pageData(""))
}

func (h *InstallHandler) Store(c *gin.Context) {
	if h.installed() {
		c.HTML(http.StatusOK, "seo_install.tmpl", h.pageData("系统已经安装，如需重新安装请先备份数据并删除 server/.install.lock"))
		return
	}

	req := installRequest{
		AppName:        strings.TrimSpace(c.PostForm("app_name")),
		AppURL:         strings.TrimSpace(c.PostForm("app_url")),
		AppPort:        strings.TrimSpace(c.PostForm("app_port")),
		DBHost:         strings.TrimSpace(c.PostForm("db_host")),
		DBPort:         strings.TrimSpace(c.PostForm("db_port")),
		DBDatabase:     strings.TrimSpace(c.PostForm("db_database")),
		DBUsername:     strings.TrimSpace(c.PostForm("db_username")),
		DBPassword:     c.PostForm("db_password"),
		DBPrefix:       strings.TrimSpace(c.PostForm("db_prefix")),
		PublicDir:      strings.TrimSpace(c.PostForm("public_dir")),
		UploadBasePath: strings.TrimSpace(c.PostForm("upload_base_path")),
		AdminAccount:   strings.TrimSpace(c.PostForm("admin_account")),
		AdminUsername:  strings.TrimSpace(c.PostForm("admin_username")),
		AdminPassword:  c.PostForm("admin_password"),
		ConfirmPass:    c.PostForm("confirm_password"),
	}
	if err := validateInstallRequest(req); err != nil {
		data := h.pageData(err.Error())
		data.Form = req
		c.HTML(http.StatusBadRequest, "seo_install.tmpl", data)
		return
	}

	cfg := h.installConfig(req)
	if err := createInstallDatabase(cfg); err != nil {
		data := h.pageData("创建数据库失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusBadRequest, "seo_install.tmpl", data)
		return
	}

	db, err := database.Open(cfg)
	if err != nil {
		data := h.pageData("连接数据库失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusBadRequest, "seo_install.tmpl", data)
		return
	}
	if err := migrateInstallTables(db); err != nil {
		data := h.pageData("创建数据表失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusInternalServerError, "seo_install.tmpl", data)
		return
	}
	if err := seedInstallData(db, cfg, req); err != nil {
		data := h.pageData("写入初始数据失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusInternalServerError, "seo_install.tmpl", data)
		return
	}
	if err := ensureInstallDirs(cfg); err != nil {
		data := h.pageData("创建资源目录失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusInternalServerError, "seo_install.tmpl", data)
		return
	}
	if err := writeInstallEnv(cfg); err != nil {
		data := h.pageData("写入 .env 失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusInternalServerError, "seo_install.tmpl", data)
		return
	}
	if err := os.WriteFile(".install.lock", []byte(time.Now().Format(time.RFC3339)+"\n"), 0600); err != nil {
		data := h.pageData("写入安装锁失败：" + err.Error())
		data.Form = req
		c.HTML(http.StatusInternalServerError, "seo_install.tmpl", data)
		return
	}

	data := h.pageData("")
	data.Success = "安装完成，请重启 wjfcm-go 服务后访问后台登录。后台地址：/admin/login"
	data.Installed = true
	c.HTML(http.StatusOK, "seo_install.tmpl", data)
}

func (h *InstallHandler) installed() bool {
	return IsInstalled(h.db)
}

func (h *InstallHandler) pageData(message string) installPageData {
	data := installPageData{
		Title:     "安装 wjfcm-go",
		SiteName:  "wjfcm-go",
		Year:      time.Now().Year(),
		Installed: h.installed(),
		Error:     message,
		Form: installRequest{
			AppName:        firstNonEmpty(h.cfg.App.Name, "wjfcm-go"),
			AppURL:         firstNonEmpty(h.cfg.App.URL, "http://localhost:8080"),
			AppPort:        firstNonEmpty(h.cfg.App.Port, "8080"),
			DBHost:         firstNonEmpty(h.cfg.DB.Host, "127.0.0.1"),
			DBPort:         firstNonEmpty(h.cfg.DB.Port, "3306"),
			DBDatabase:     h.cfg.DB.Database,
			DBUsername:     h.cfg.DB.Username,
			DBPrefix:       firstNonEmpty(h.cfg.DB.Prefix, "wjf_"),
			PublicDir:      firstNonEmpty(h.cfg.Upload.PublicDir, "../public"),
			UploadBasePath: firstNonEmpty(h.cfg.Upload.BasePath, "uploads"),
			AdminAccount:   "13000000000",
			AdminUsername:  "Vijay",
		},
		RequireNote: "安装会创建数据表并写入初始管理员。生产环境请先准备空数据库或确认数据已备份。",
	}
	return data
}

func IsInstalled(db *gorm.DB) bool {
	if _, err := os.Stat(".install.lock"); err == nil {
		return true
	}
	if db == nil {
		return false
	}
	if !db.Migrator().HasTable(&model.Admin{}) {
		return false
	}
	var count int64
	return db.Model(&model.Admin{}).Count(&count).Error == nil && count > 0
}

func validateInstallRequest(req installRequest) error {
	if req.AppName == "" || req.AppURL == "" || req.AppPort == "" {
		return fmt.Errorf("请填写站点名称、站点地址和服务端口")
	}
	if req.DBHost == "" || req.DBPort == "" || req.DBDatabase == "" || req.DBUsername == "" {
		return fmt.Errorf("请填写完整数据库配置")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(req.DBDatabase) {
		return fmt.Errorf("数据库名只能包含字母、数字和下划线")
	}
	if req.DBPrefix != "" && !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(req.DBPrefix) {
		return fmt.Errorf("数据表前缀只能包含字母、数字和下划线")
	}
	if req.AdminAccount == "" || req.AdminUsername == "" {
		return fmt.Errorf("请填写管理员账号和昵称")
	}
	if len(req.AdminPassword) < 6 {
		return fmt.Errorf("管理员密码至少 6 位")
	}
	if req.AdminPassword != req.ConfirmPass {
		return fmt.Errorf("两次输入的管理员密码不一致")
	}
	return nil
}

func (h *InstallHandler) installConfig(req installRequest) config.Config {
	cfg := h.cfg
	cfg.App.Name = req.AppName
	cfg.App.Env = "production"
	cfg.App.Debug = false
	cfg.App.Port = req.AppPort
	cfg.App.URL = strings.TrimRight(req.AppURL, "/")
	cfg.App.Key = randomInstallSecret(32)
	cfg.App.ConsoleColor = true
	cfg.JWT.Secret = randomInstallSecret(48)
	cfg.DB.Host = req.DBHost
	cfg.DB.Port = req.DBPort
	cfg.DB.Database = req.DBDatabase
	cfg.DB.Username = req.DBUsername
	cfg.DB.Password = req.DBPassword
	cfg.DB.Prefix = req.DBPrefix
	cfg.Upload.PublicDir = req.PublicDir
	cfg.Upload.BasePath = firstNonEmpty(req.UploadBasePath, "uploads")
	cfg.CORS.AllowOrigins = []string{cfg.App.URL}
	return cfg
}

func createInstallDatabase(cfg config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", cfg.DB.Username, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + cfg.DB.Database + "` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	return err
}

func migrateInstallTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Admin{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.ModelHasRole{},
		&model.ModelHasPermission{},
		&model.RoleHasPermission{},
		&model.SystemConfig{},
		&model.Category{},
		&model.Article{},
		&model.Tag{},
		&model.Comment{},
		&model.FriendLink{},
		&model.Nav{},
		&model.Chat{},
		&model.WxKeyword{},
	)
}

func seedInstallData(db *gorm.DB, cfg config.Config, req installRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedInstallAdmin(tx, req); err != nil {
			return err
		}
		if err := seedInstallPermissions(tx); err != nil {
			return err
		}
		if err := seedInstallSystemConfigs(tx, cfg); err != nil {
			return err
		}
		if err := seedInstallContent(tx); err != nil {
			return err
		}
		return nil
	})
}

func seedInstallAdmin(tx *gorm.DB, req installRequest) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	role := model.Role{BaseModel: model.BaseModel{ID: 1}, Name: "admin", Description: "超级管理员", GuardName: "admin", Status: 1}
	if err := tx.Where("id = ?", 1).FirstOrCreate(&role).Error; err != nil {
		return err
	}
	admin := model.Admin{
		BaseBigModel: model.BaseBigModel{ID: 1},
		Account:      req.AdminAccount,
		Username:     req.AdminUsername,
		Password:     string(hash),
		RoleNames:    "admin",
		Status:       1,
	}
	if err := tx.Where("id = ?", 1).FirstOrCreate(&admin).Error; err != nil {
		return err
	}
	adminRole := model.ModelHasRole{RoleID: 1, ModelType: "App\\Models\\Admin", ModelID: 1}
	return tx.Where("role_id = ? AND model_type = ? AND model_id = ?", adminRole.RoleID, adminRole.ModelType, adminRole.ModelID).FirstOrCreate(&adminRole).Error
}

func seedInstallPermissions(tx *gorm.DB) error {
	items := []model.Permission{
		{BaseModel: model.BaseModel{ID: 1}, Name: "后台管理", GuardName: "admin", URL: "/admin/index", Level: 0, ParentID: 0, DisplayMenu: 1, Icon: "layout-dashboard"},
		{BaseModel: model.BaseModel{ID: 2}, Name: "系统管理", GuardName: "admin", URL: "/admin/system", Level: 0, ParentID: 0, DisplayMenu: 1, Icon: "settings"},
		{BaseModel: model.BaseModel{ID: 3}, Name: "控制台", GuardName: "admin", URL: "/admin/index/main", Level: 1, ParentID: 1, DisplayMenu: 1, Icon: "home"},
		{BaseModel: model.BaseModel{ID: 4}, Name: "角色管理", GuardName: "admin", URL: "/admin/role/index", Level: 1, ParentID: 2, DisplayMenu: 1, Icon: "shield"},
		{BaseModel: model.BaseModel{ID: 5}, Name: "权限管理", GuardName: "admin", URL: "/admin/permission/index", Level: 1, ParentID: 2, DisplayMenu: 1, Icon: "key"},
		{BaseModel: model.BaseModel{ID: 6}, Name: "用户管理", GuardName: "admin", URL: "/admin/admin/index", Level: 1, ParentID: 2, DisplayMenu: 1, Icon: "users"},
		{BaseModel: model.BaseModel{ID: 7}, Name: "管理员列表", GuardName: "admin", URL: "/admin/admin/index", Level: 2, ParentID: 6, DisplayMenu: 1},
		{BaseModel: model.BaseModel{ID: 8}, Name: "用户列表", GuardName: "admin", URL: "/admin/user/index", Level: 2, ParentID: 6, DisplayMenu: 1},
		{BaseModel: model.BaseModel{ID: 9}, Name: "网站设置", GuardName: "admin", URL: "/admin/systemConfig/basal", Level: 1, ParentID: 2, DisplayMenu: 1, Icon: "sliders"},
		{BaseModel: model.BaseModel{ID: 10}, Name: "内容管理", GuardName: "admin", URL: "/admin/article", Level: 0, ParentID: 0, DisplayMenu: 1, Icon: "book-open"},
		{BaseModel: model.BaseModel{ID: 11}, Name: "文章列表", GuardName: "admin", URL: "/admin/article/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "file-text"},
		{BaseModel: model.BaseModel{ID: 12}, Name: "文章分类", GuardName: "admin", URL: "/admin/category/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "folder"},
		{BaseModel: model.BaseModel{ID: 13}, Name: "评论列表", GuardName: "admin", URL: "/admin/comment/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "message-circle"},
		{BaseModel: model.BaseModel{ID: 14}, Name: "有些话", GuardName: "admin", URL: "/admin/chat/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "smile"},
		{BaseModel: model.BaseModel{ID: 15}, Name: "标签管理", GuardName: "admin", URL: "/admin/tag/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "tag"},
		{BaseModel: model.BaseModel{ID: 16}, Name: "前台菜单", GuardName: "admin", URL: "/admin/nav/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "menu"},
		{BaseModel: model.BaseModel{ID: 17}, Name: "友情链接", GuardName: "admin", URL: "/admin/friendLinks/index", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "link"},
		{BaseModel: model.BaseModel{ID: 18}, Name: "微信配置", GuardName: "admin", URL: "/admin/wechat", Level: 1, ParentID: 10, DisplayMenu: 1, Icon: "message-square"},
		{BaseModel: model.BaseModel{ID: 19}, Name: "关键词回复", GuardName: "admin", URL: "/admin/weChat/keyword/index", Level: 2, ParentID: 18, DisplayMenu: 1},
	}
	for _, item := range items {
		permission := item
		if err := tx.Where("id = ?", item.ID).FirstOrCreate(&permission).Error; err != nil {
			return err
		}
		if item.ID > 0 {
			link := model.RoleHasPermission{RoleID: 1, PermissionID: item.ID}
			if err := tx.Where("role_id = ? AND permission_id = ?", link.RoleID, link.PermissionID).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedInstallSystemConfigs(tx *gorm.DB, cfg config.Config) error {
	items := []model.SystemConfig{
		{Title: "网站名称", Key: "site_name", Value: cfg.App.Name, Type: "text", ConfigType: 1, Status: 1},
		{Title: "网站地址", Key: "site_url", Value: cfg.App.URL, Type: "text", ConfigType: 1, Status: 1},
		{Title: "网站 LOGO", Key: "site_logo", Value: "/images/config/logo.png", Type: "image", ConfigType: 1, Status: 1},
		{Title: "默认头像", Key: "site_avatar", Value: "/images/config/avatar.jpg", Type: "image", ConfigType: 1, Status: 1},
		{Title: "SEO 标题", Key: "site_seo_title", Value: cfg.App.Name, Type: "text", ConfigType: 2, Status: 1},
		{Title: "SEO 关键词", Key: "site_keywords", Value: "wjfcm-go,CMS,Gin,Vue", Type: "text", ConfigType: 2, Status: 1},
		{Title: "SEO 描述", Key: "site_description", Value: "wjfcm-go CMS", Type: "textarea", ConfigType: 2, Status: 1},
		{Title: "举报邮箱", Key: "report_email", Value: "", Type: "text", ConfigType: 1, Status: 1},
		{Title: "ICP备案号", Key: "site_icp", Value: "", Type: "text", ConfigType: 1, Status: 1},
		{Title: "百度站长验证", Key: "baidu_site_verification", Value: "9jxVRatXIs", Type: "text", ConfigType: 1, Status: 1},
		{Title: "百度联盟验证", Key: "baidu_union_verify", Value: "285b65cf325abe072bde0437c133e008", Type: "text", ConfigType: 1, Status: 1},
		{Title: "360 站点验证", Key: "360_site_verification", Value: "6f7a678e74c316eb393d9fd80d103ca2", Type: "text", ConfigType: 1, Status: 1},
		{Title: "Google 站点验证", Key: "google_site_verification", Value: "", Type: "text", ConfigType: 1, Status: 1},
		{Title: "统计代码", Key: "site_tongji", Value: "", Type: "textarea", ConfigType: 1, Status: 1},
		{Title: "Google AdSense Client", Key: "site_google_adsense_client", Value: "ca-pub-4281894096969033", Type: "text", ConfigType: 1, Status: 1},
		{Title: "Google AdSense 完整代码", Key: "site_google_adsense_html", Value: "", Type: "textarea", ConfigType: 1, Status: 1},
		{Title: "微信号", Key: "site_wechat", Value: "", Type: "text", ConfigType: 1, Status: 1},
		{Title: "公众号二维码", Key: "site_wechat_public", Value: "/images/config/wx_public.jpg", Type: "image", ConfigType: 1, Status: 1},
	}
	for _, item := range items {
		configItem := item
		if err := tx.Where("`key` = ?", item.Key).FirstOrCreate(&configItem).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedInstallContent(tx *gorm.DB) error {
	category := model.Category{BaseModel: model.BaseModel{ID: 1}, Name: "默认分类", Keywords: "默认分类", Description: "默认分类", Sort: 0}
	if err := tx.Where("id = ?", 1).FirstOrCreate(&category).Error; err != nil {
		return err
	}
	navs := []model.Nav{
		{BaseModel: model.BaseModel{ID: 1}, Name: "首页", URL: "/", Sort: 0, Target: "_self"},
		{BaseModel: model.BaseModel{ID: 2}, Name: "文章归档", URL: "/archive", Sort: 10, Target: "_self"},
		{BaseModel: model.BaseModel{ID: 3}, Name: "有些话", URL: "/chat", Sort: 20, Target: "_self"},
	}
	for _, item := range navs {
		nav := item
		if err := tx.Where("id = ?", item.ID).FirstOrCreate(&nav).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureInstallDirs(cfg config.Config) error {
	paths := []string{
		cfg.Upload.PublicDir,
		filepath.Join(cfg.Upload.PublicDir, cfg.Upload.BasePath),
		filepath.Join(cfg.Upload.PublicDir, "images", "config"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

func writeInstallEnv(cfg config.Config) error {
	if _, err := os.Stat(".env"); err == nil {
		backup := ".env.bak." + time.Now().Format("20060102150405")
		if err := os.Rename(".env", backup); err != nil {
			return err
		}
	}
	lines := []string{
		"APP_NAME=" + quoteEnv(cfg.App.Name),
		"APP_ENV=production",
		"APP_KEY=" + quoteEnv(cfg.App.Key),
		"APP_DEBUG=false",
		"APP_PORT=" + quoteEnv(cfg.App.Port),
		"APP_URL=" + quoteEnv(cfg.App.URL),
		"APP_CONSOLE_COLOR=true",
		"",
		"LOG_CHANNEL=stack",
		"LOG_PATH=storage/logs",
		"LOG_MAX_SIZE_MB=50",
		"REQUEST_LOG_ENABLED=false",
		"REQUEST_LOG_TYPE=json",
		"REQUEST_LOG_PATH=storage/request-logs",
		"REQUEST_LOG_OUTPUT=file",
		"REQUEST_LOG_LEVEL=info",
		"REQUEST_LOG_ONLY_API=true",
		"REQUEST_LOG_MAX_BODY_KB=256",
		"REQUEST_LOG_MAX_RESPONSE_KB=64",
		"REQUEST_LOG_MAX_FILE_MB=20",
		"REQUEST_LOG_KEEP_DAYS=14",
		"",
		"DB_HOST=" + quoteEnv(cfg.DB.Host),
		"DB_PORT=" + quoteEnv(cfg.DB.Port),
		"DB_DATABASE=" + quoteEnv(cfg.DB.Database),
		"DB_USERNAME=" + quoteEnv(cfg.DB.Username),
		"DB_PASSWORD=" + quoteEnv(cfg.DB.Password),
		"DB_PREFIX=" + quoteEnv(cfg.DB.Prefix),
		"DB_LOG_SQL=false",
		"DB_LOG_SLOW_SQL=true",
		"DB_LOG_ERROR_SQL=true",
		"DB_LOG_LEVEL=info",
		"DB_SLOW_THRESHOLD_MS=200",
		"",
		"JWT_SECRET=" + quoteEnv(cfg.JWT.Secret),
		"JWT_EXPIRES_MINUTES=120",
		"JWT_REFRESH_EXPIRES_MINUTES=10080",
		"",
		"CORS_ALLOW_ORIGINS=" + quoteEnv(strings.Join(cfg.CORS.AllowOrigins, ",")),
		"PUBLIC_DIR=" + quoteEnv(cfg.Upload.PublicDir),
		"UPLOAD_BASE_PATH=" + quoteEnv(cfg.Upload.BasePath),
	}
	return os.WriteFile(".env", []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func quoteEnv(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func randomInstallSecret(length int) string {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}
