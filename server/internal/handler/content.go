package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"
	"wjfcm-go/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ContentHandler struct {
	db  *gorm.DB
	cfg config.Config
}

func NewContentHandler(cfg config.Config, db *gorm.DB) *ContentHandler {
	return &ContentHandler{cfg: cfg, db: db}
}

func (h *ContentHandler) SubmitBaiduLinks(c *gin.Context) {
	api := strings.TrimSpace(h.cfg.BaiduSite.API)
	if api == "" {
		api = strings.TrimSpace(h.systemConfigValue("baidu_site_api"))
	}
	if api == "" {
		response.Error(c, http.StatusBadRequest, 1, "请先配置 BAIDU_SITE_API")
		return
	}

	base := strings.TrimSpace(h.cfg.BaiduSite.Base)
	if base == "" {
		base = strings.TrimSpace(h.systemConfigValue("site_url"))
	}
	if base == "" {
		base = strings.TrimSpace(h.cfg.App.URL)
	}
	base = strings.TrimRight(base, "/") + "/"

	urls := []string{
		base,
		base + "register",
		base + "login",
		base + "chat",
		base + "admin/login",
	}

	var articleIDs []uint64
	if err := h.db.Model(&model.Article{}).Where("status = ?", 1).Order("id ASC").Pluck("id", &articleIDs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	for _, id := range articleIDs {
		urls = append(urls, base+"article/"+strconv.FormatUint(id, 10))
	}

	var categoryIDs []uint
	if err := h.db.Model(&model.Category{}).Order("id ASC").Pluck("id", &categoryIDs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	for _, id := range categoryIDs {
		urls = append(urls, base+"category/"+strconv.FormatUint(uint64(id), 10))
	}

	var tagIDs []uint
	if err := h.db.Model(&model.Tag{}).Order("id ASC").Pluck("id", &tagIDs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	for _, id := range tagIDs {
		urls = append(urls, base+"tag/"+strconv.FormatUint(uint64(id), 10))
	}

	client := http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodPost, api, strings.NewReader(strings.Join(urls, "\n")))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 1, err.Error())
		return
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, "百度主动推送失败："+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, "读取百度响应失败")
		return
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		response.Error(c, http.StatusBadGateway, 1, "百度响应异常："+string(body))
		return
	}
	if _, ok := result["error"]; ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"code": 1,
			"msg":  "百度主动推送失败",
			"data": result,
		})
		return
	}
	result["submitted_count"] = len(urls)
	result["submitted_urls"] = urls
	response.OK(c, "百度主动推送完成", result)
}

func (h *ContentHandler) HistoryToday(c *gin.Context) {
	now := time.Now()
	apiURL := "http://www.jiahengfei.cn:33550/port/history?dispose=detail&key=jiahengfei&month=" +
		now.Format("01") + "&day=" + now.Format("02")

	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, "历史上的今天接口暂不可用")
		return
	}
	defer resp.Body.Close()

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		response.Error(c, http.StatusBadGateway, 1, "历史上的今天响应异常")
		return
	}
	response.OK(c, "获取成功", payload)
}

func (h *ContentHandler) systemConfigValue(key string) string {
	var item model.SystemConfig
	if err := h.db.Where("`key` = ? AND status = ?", key, 1).First(&item).Error; err != nil {
		return ""
	}
	return item.Value
}

func (h *ContentHandler) Navs(c *gin.Context) {
	var items []model.Nav
	query := h.db.Model(&model.Nav{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)
	if err := query.Order("sort DESC, id DESC").Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", items, total)
}

func (h *ContentHandler) StoreNav(c *gin.Context) {
	var item model.Nav
	if err := c.ShouldBindJSON(&item); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写导航信息")
		return
	}
	if item.Target == "" {
		item.Target = "_self"
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) UpdateNav(c *gin.Context) {
	var item model.Nav
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "导航不存在")
		return
	}
	if err := c.ShouldBindJSON(&item); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写导航信息")
		return
	}
	if item.Target == "" {
		item.Target = "_self"
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) DeleteNav(c *gin.Context) {
	if err := h.db.Delete(&model.Nav{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ContentHandler) RestoreNav(c *gin.Context) {
	restoreByID(c, h.db, &model.Nav{})
}

func (h *ContentHandler) ForceDeleteNav(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Nav{})
}

func (h *ContentHandler) FriendLinks(c *gin.Context) {
	var items []model.FriendLink
	query := h.db.Model(&model.FriendLink{})
	if c.FullPath() == "/api/home/friend-links" {
		query = query.Where("status = ?", 1)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR url LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)
	if err := query.Order("sort DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", items, total)
}

func (h *ContentHandler) ApplyFriendLink(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写友情链接信息")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.URL == "" || req.Email == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写网站名称、链接和联系邮箱")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "联系邮箱格式不正确")
		return
	}
	parsed, err := url.ParseRequestURI(req.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		response.Error(c, http.StatusBadRequest, 1, "网站链接必须以 http 或 https 开头")
		return
	}

	clientIP := c.ClientIP()
	var count int64
	h.db.Model(&model.FriendLink{}).Where("client_ip = ? AND created_at >= ?", clientIP, time.Now().Add(-5*time.Minute)).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "您已提交过了，不要重复提交哦")
		return
	}
	h.db.Model(&model.FriendLink{}).Where("email = ? OR url = ?", req.Email, req.URL).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "该邮箱或网站链接已提交过")
		return
	}

	item := model.FriendLink{
		Name:     req.Name,
		URL:      req.URL,
		Email:    req.Email,
		ClientIP: clientIP,
		Status:   0,
		Sort:     0,
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	go func() {
		body := "有新的友情链接申请，请及时处理。<br>" +
			"网站名称：" + item.Name + "<br>" +
			"网站链接：" + item.URL + "<br>" +
			"联系邮箱：" + item.Email
		if err := service.SendMail(h.cfg.Mail, "友情链接申请，请及时处理", body); err != nil {
			log.Printf("[mail] friend link notification failed: %v", err)
		}
	}()
	response.OK(c, "申请已提交，请等待审核", item)
}

func (h *ContentHandler) StoreFriendLink(c *gin.Context) {
	var item model.FriendLink
	if err := c.ShouldBindJSON(&item); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写友情链接信息")
		return
	}
	if item.ClientIP == "" {
		item.ClientIP = c.ClientIP()
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) UpdateFriendLink(c *gin.Context) {
	var item model.FriendLink
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "友情链接不存在")
		return
	}
	if err := c.ShouldBindJSON(&item); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写友情链接信息")
		return
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) DeleteFriendLink(c *gin.Context) {
	if err := h.db.Delete(&model.FriendLink{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ContentHandler) RestoreFriendLink(c *gin.Context) {
	restoreByID(c, h.db, &model.FriendLink{})
}

func (h *ContentHandler) ForceDeleteFriendLink(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.FriendLink{})
}

func (h *ContentHandler) Chats(c *gin.Context) {
	var items []model.Chat
	page, pageSize := pageParams(c)
	query := h.db.Model(&model.Chat{})
	if content := c.Query("content"); content != "" {
		query = query.Where("content LIKE ?", "%"+content+"%")
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	var total int64
	query.Count(&total)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", items, total)
}

func (h *ContentHandler) StoreChat(c *gin.Context) {
	var item model.Chat
	if err := c.ShouldBindJSON(&item); err != nil || item.Content == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写内容")
		return
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) UpdateChat(c *gin.Context) {
	var item model.Chat
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "内容不存在")
		return
	}
	if err := c.ShouldBindJSON(&item); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写内容")
		return
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) DeleteChat(c *gin.Context) {
	if err := h.db.Delete(&model.Chat{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ContentHandler) RestoreChat(c *gin.Context) {
	restoreByID(c, h.db, &model.Chat{})
}

func (h *ContentHandler) ForceDeleteChat(c *gin.Context) {
	forceDeleteByID(c, h.db, &model.Chat{})
}

func (h *ContentHandler) SystemConfigs(c *gin.Context) {
	var items []model.SystemConfig
	group := c.Query("group")
	if group != "" {
		h.ensureSystemConfigDefaults(group)
	}
	query := h.db.Model(&model.SystemConfig{})
	if group != "" {
		query = applySystemConfigGroup(query, group)
	}
	if configType := c.Query("config_type"); configType != "" {
		query = query.Where("config_type = ?", configType)
	}
	if title := c.Query("title"); title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}
	if key := c.Query("key"); key != "" {
		query = query.Where("`key` LIKE ?", "%"+key+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	if err := query.Order("id ASC").Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", items, total)
}

func applySystemConfigGroup(query *gorm.DB, group string) *gorm.DB {
	switch group {
	case "basic":
		return query.Where("`key` IN ?", []string{"site_name", "site_url", "site_logo", "site_icp", "site_tongji", "site_copyright"})
	case "contact":
		return query.Where("`key` IN ?", []string{"site_co_name", "address", "map_lat", "map_lng", "site_phone", "site_email", "site_qq", "site_wechat"})
	case "seo":
		return query.Where("`key` IN ?", []string{"seo_title", "site_seo_keywords", "site_seo_description"})
	case "wechat":
		return query.Where("`key` IN ?", systemConfigSeedKeys(wechatConfigSeeds()))
	case "mini":
		return query.Where("`key` IN ?", systemConfigSeedKeys(miniProgramConfigSeeds()))
	default:
		return query
	}
}

type systemConfigSeed struct {
	Title      string
	Key        string
	Value      string
	Type       string
	ConfigType int8
	Status     int8
}

func (h *ContentHandler) ensureSystemConfigDefaults(group string) {
	var seeds []systemConfigSeed
	switch group {
	case "wechat":
		seeds = wechatConfigSeeds()
	case "mini":
		seeds = miniProgramConfigSeeds()
	default:
		return
	}
	for _, seed := range seeds {
		var count int64
		h.db.Model(&model.SystemConfig{}).Where("`key` = ?", seed.Key).Count(&count)
		if count > 0 {
			continue
		}
		_ = h.db.Create(&model.SystemConfig{
			Title:      seed.Title,
			Key:        seed.Key,
			Value:      seed.Value,
			Type:       seed.Type,
			ConfigType: seed.ConfigType,
			Status:     seed.Status,
		}).Error
	}
}

func wechatConfigSeeds() []systemConfigSeed {
	return []systemConfigSeed{
		{Title: "微信号", Key: "site_wechat", Type: "text", ConfigType: 1, Status: 1},
		{Title: "公众号二维码", Key: "site_wechat_public", Value: "/images/config/wx_public.jpg", Type: "image", ConfigType: 1, Status: 1},
		{Title: "公众号标题", Key: "site_wechat_public_title", Value: "公众号", Type: "text", ConfigType: 1, Status: 1},
		{Title: "公众号描述", Key: "site_wechat_public_desc", Type: "textarea", ConfigType: 1, Status: 1},
		{Title: "微信客服二维码", Key: "site_wechat_qrcode", Value: "/images/config/wx.jpg", Type: "image", ConfigType: 1, Status: 1},
		{Title: "微信支付收款码", Key: "site_wechat_pay_qrcode", Value: "/images/config/weixin_pay.jpg", Type: "image", ConfigType: 1, Status: 1},
		{Title: "公众号 AppID", Key: "wechat_official_account_appid", Type: "text", ConfigType: 1, Status: 1},
		{Title: "公众号 AppSecret", Key: "wechat_official_account_secret", Type: "password", ConfigType: 1, Status: 1},
		{Title: "公众号 Token", Key: "wechat_official_account_token", Type: "text", ConfigType: 1, Status: 1},
		{Title: "公众号 EncodingAESKey", Key: "wechat_official_account_aes_key", Type: "text", ConfigType: 1, Status: 1},
		{Title: "微信网页授权 Client ID", Key: "wechatweb_client_id", Type: "text", ConfigType: 1, Status: 1},
		{Title: "微信网页授权 Secret", Key: "wechatweb_client_secret", Type: "password", ConfigType: 1, Status: 1},
		{Title: "微信网页授权回调", Key: "wechatweb_redirect_uri", Type: "text", ConfigType: 1, Status: 1},
		{Title: "图灵 API Key", Key: "tuling_api_key", Type: "password", ConfigType: 1, Status: 1},
		{Title: "图灵 API URL", Key: "tuling_api_url", Type: "text", ConfigType: 1, Status: 1},
		{Title: "QQ AI AppID", Key: "qq_ai_appid", Type: "text", ConfigType: 1, Status: 1},
		{Title: "QQ AI AppKey", Key: "qq_ai_appkey", Type: "password", ConfigType: 1, Status: 1},
		{Title: "QQ AI URL", Key: "qq_ai_url", Type: "text", ConfigType: 1, Status: 1},
	}
}

func miniProgramConfigSeeds() []systemConfigSeed {
	return []systemConfigSeed{
		{Title: "小程序 AppID", Key: "mini_program_appid", Type: "text", ConfigType: 2, Status: 1},
		{Title: "小程序 AppSecret", Key: "mini_program_secret", Type: "password", ConfigType: 2, Status: 1},
		{Title: "小程序二维码", Key: "mini_program_qrcode", Value: "/images/config/wx_public.jpg", Type: "image", ConfigType: 2, Status: 1},
		{Title: "小程序标题", Key: "mini_program_title", Value: "小程序", Type: "text", ConfigType: 2, Status: 1},
		{Title: "小程序描述", Key: "mini_program_desc", Type: "textarea", ConfigType: 2, Status: 1},
	}
}

func systemConfigSeedKeys(seeds []systemConfigSeed) []string {
	keys := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		keys = append(keys, seed.Key)
	}
	sort.Strings(keys)
	return keys
}

func (h *ContentHandler) UpdateSystemConfig(c *gin.Context) {
	var item model.SystemConfig
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "配置不存在")
		return
	}
	var req model.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写配置")
		return
	}
	if err := h.db.Model(&item).Updates(map[string]any{
		"title": req.Title, "key": req.Key, "value": req.Value, "type": req.Type,
		"config_type": req.ConfigType, "status": req.Status,
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) WxKeywords(c *gin.Context) {
	var items []model.WxKeyword
	query := h.db.Model(&model.WxKeyword{})
	if name := c.Query("name"); name != "" {
		query = query.Where("key_name LIKE ?", "%"+name+"%")
	}
	if value := c.Query("key_value"); value != "" {
		query = query.Where("key_value LIKE ?", "%"+value+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query = applyDeleteFilter(query, c.Query("delete"))
	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)
	if err := query.Order("sort DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.Page(c, "获取成功", items, total)
}

func (h *ContentHandler) StoreWxKeyword(c *gin.Context) {
	var item model.WxKeyword
	if err := c.ShouldBindJSON(&item); err != nil || strings.TrimSpace(item.KeyName) == "" || strings.TrimSpace(item.KeyValue) == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写关键词和回复内容")
		return
	}
	item.KeyName = strings.TrimSpace(item.KeyName)
	item.KeyValue = strings.TrimSpace(item.KeyValue)
	if item.Status != 0 {
		item.Status = 1
	}
	var count int64
	h.db.Model(&model.WxKeyword{}).Where("LOWER(key_name) = ?", strings.ToLower(item.KeyName)).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "该关键词已存在")
		return
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) UpdateWxKeyword(c *gin.Context) {
	var item model.WxKeyword
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "关键词不存在")
		return
	}
	var req model.WxKeyword
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写关键词信息")
		return
	}
	if strings.TrimSpace(req.KeyName) == "" && strings.TrimSpace(req.KeyValue) == "" && req.Sort == 0 && req.Status == 0 {
		response.Error(c, http.StatusBadRequest, 1, "请填写关键词信息")
		return
	}
	updates := map[string]any{}
	if strings.TrimSpace(req.KeyName) != "" {
		keyName := strings.TrimSpace(req.KeyName)
		var count int64
		h.db.Model(&model.WxKeyword{}).Where("LOWER(key_name) = ? AND id <> ?", strings.ToLower(keyName), item.ID).Count(&count)
		if count > 0 {
			response.Error(c, http.StatusBadRequest, 1, "该关键词已存在")
			return
		}
		updates["key_name"] = keyName
	}
	if strings.TrimSpace(req.KeyValue) != "" {
		updates["key_value"] = strings.TrimSpace(req.KeyValue)
	}
	updates["sort"] = req.Sort
	updates["status"] = req.Status
	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", item)
}

func (h *ContentHandler) DeleteWxKeyword(c *gin.Context) {
	if err := h.db.Delete(&model.WxKeyword{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ContentHandler) RestoreWxKeyword(c *gin.Context) {
	response.Error(c, http.StatusBadRequest, 1, "关键词回复不支持回收站恢复")
}

func (h *ContentHandler) ForceDeleteWxKeyword(c *gin.Context) {
	if err := h.db.Delete(&model.WxKeyword{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}
