package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"
	"wjfcm-go/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type HomeAuthHandler struct {
	cfg        config.Config
	db         *gorm.DB
	codeMu     sync.Mutex
	emailCodes map[string]emailCode
	captchas   map[string]captchaCode
}

type emailCode struct {
	Code      string
	Purpose   string
	ExpiresAt time.Time
	SentAt    time.Time
}

type captchaCode struct {
	Code      string
	ExpiresAt time.Time
}

type homeLoginRequest struct {
	Account     string `json:"account" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaKey  string `json:"captcha_key" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

type homeRegisterRequest struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Code        string `json:"code" binding:"required"`
	CaptchaKey  string `json:"captcha_key" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

type oauthProfile struct {
	ID       string
	Name     string
	Email    string
	Avatar   string
	Provider string
}

func NewHomeAuthHandler(cfg config.Config, db *gorm.DB) *HomeAuthHandler {
	return &HomeAuthHandler{cfg: cfg, db: db, emailCodes: map[string]emailCode{}, captchas: map[string]captchaCode{}}
}

func (h *HomeAuthHandler) Register(c *gin.Context) {
	var req homeRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写昵称、邮箱、密码和验证码")
		return
	}
	if !h.verifyCaptcha(req.CaptchaKey, req.CaptchaCode) {
		response.Error(c, http.StatusBadRequest, 1, "图形验证码错误或已过期")
		return
	}
	if len(req.Password) < 6 {
		response.Error(c, http.StatusBadRequest, 1, "密码至少 6 位")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if _, err := mail.ParseAddress(req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "邮箱格式不正确")
		return
	}
	if !h.verifyEmailCode(req.Email, "register", req.Code) {
		response.Error(c, http.StatusBadRequest, 1, "邮箱验证码错误或已过期")
		return
	}

	var count int64
	h.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "邮箱已被注册")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	user := model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hash),
		Avatar:   "/images/config/avatar.jpg",
	}
	if err := h.db.Create(&user).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	token, err := service.MakeUserToken(user, h.cfg)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "注册成功", service.UserLoginResult{Token: token, User: user})
}

func (h *HomeAuthHandler) Login(c *gin.Context) {
	var req homeLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请输入账号、密码和验证码")
		return
	}
	if !h.verifyCaptcha(req.CaptchaKey, req.CaptchaCode) {
		response.Error(c, http.StatusBadRequest, 1, "图形验证码错误或已过期")
		return
	}
	result, err := service.LoginUser(h.db, h.cfg, req.Account, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 1, err.Error())
		return
	}
	response.OK(c, "登录成功", result)
}

func (h *HomeAuthHandler) Captcha(c *gin.Context) {
	a, err := secureRandInt(1, 9)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	b, err := secureRandInt(1, 9)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	key, err := randomToken(16)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	code := fmt.Sprintf("%d", a+b)
	h.saveCaptcha(key, code)
	image := buildCaptchaSVG(fmt.Sprintf("%d + %d = ?", a, b))
	response.OK(c, "获取成功", gin.H{
		"key":   key,
		"image": "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(image)),
	})
}

func (h *HomeAuthHandler) SendEmailCode(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写邮箱")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Purpose = strings.TrimSpace(strings.ToLower(req.Purpose))
	if req.Purpose != "register" && req.Purpose != "reset_password" && req.Purpose != "profile_email" {
		response.Error(c, http.StatusBadRequest, 1, "验证码用途不正确")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "邮箱格式不正确")
		return
	}

	var count int64
	h.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
	if req.Purpose == "register" && count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "邮箱已被注册")
		return
	}
	if req.Purpose == "reset_password" && count == 0 {
		response.Error(c, http.StatusBadRequest, 1, "邮箱还未注册")
		return
	}
	if req.Purpose == "profile_email" && count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "邮箱已被其他用户使用")
		return
	}

	code, err := randomNumericCode(6)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if !h.saveEmailCode(req.Email, req.Purpose, code) {
		response.Error(c, http.StatusTooManyRequests, 1, "验证码发送太频繁，请稍后再试")
		return
	}

	subject := h.siteName() + " 邮箱验证码"
	body := fmt.Sprintf("<p>您的验证码是：<strong style=\"font-size:20px;\">%s</strong></p><p>10 分钟内有效，请勿转发给他人。</p>", code)
	if err := service.SendMailTo(h.cfg.Mail, req.Email, subject, body); err != nil {
		h.deleteEmailCode(req.Email, req.Purpose)
		log.Printf("[mail] email code send failed: %v", err)
		response.Error(c, http.StatusInternalServerError, 1, "验证码邮件发送失败")
		return
	}
	data := gin.H{}
	if h.cfg.App.Debug {
		data["debug_code"] = code
	}
	response.OK(c, "验证码已发送", data)
}

func (h *HomeAuthHandler) siteName() string {
	var item model.SystemConfig
	if err := h.db.Where("`key` = ? AND status = ?", "site_name", 1).First(&item).Error; err == nil && strings.TrimSpace(item.Value) != "" {
		return strings.TrimSpace(item.Value)
	}
	return "wjfcm-go"
}

func (h *HomeAuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Password    string `json:"password" binding:"required"`
		CaptchaKey  string `json:"captcha_key" binding:"required"`
		CaptchaCode string `json:"captcha_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写邮箱、验证码、新密码和图形验证码")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !h.verifyCaptcha(req.CaptchaKey, req.CaptchaCode) {
		response.Error(c, http.StatusBadRequest, 1, "图形验证码错误或已过期")
		return
	}
	if len(req.Password) < 6 {
		response.Error(c, http.StatusBadRequest, 1, "密码至少 6 位")
		return
	}
	if !h.verifyEmailCode(req.Email, "reset_password", req.Code) {
		response.Error(c, http.StatusBadRequest, 1, "邮箱验证码错误或已过期")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	result := h.db.Model(&model.User{}).Where("email = ?", req.Email).Update("password", string(hash))
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, 1, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, 1, "用户不存在")
		return
	}
	response.OK(c, "密码已重置，请重新登录", nil)
}

func (h *HomeAuthHandler) Profile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "用户不存在")
		return
	}
	response.OK(c, "获取成功", user)
}

func (h *HomeAuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "用户不存在")
		return
	}
	var req struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		EmailCode string `json:"email_code"`
		Sex       int8   `json:"sex"`
		Tel       string `json:"tel"`
		City      string `json:"city"`
		Intro     string `json:"intro"`
		Avatar    string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写用户信息")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	updates := map[string]any{
		"name": req.Name, "sex": req.Sex, "tel": req.Tel,
		"city": req.City, "intro": req.Intro, "avatar": req.Avatar,
	}
	if req.Email != strings.ToLower(strings.TrimSpace(user.Email)) {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			response.Error(c, http.StatusBadRequest, 1, "邮箱格式不正确")
			return
		}
		if !h.verifyEmailCode(req.Email, "profile_email", req.EmailCode) {
			response.Error(c, http.StatusBadRequest, 1, "邮箱验证码错误或已过期")
			return
		}
		var count int64
		h.db.Model(&model.User{}).Where("email = ? AND id <> ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			response.Error(c, http.StatusBadRequest, 1, "邮箱已被其他用户使用")
			return
		}
		now := time.Now()
		updates["email"] = req.Email
		updates["email_verified_at"] = &now
	}
	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	h.db.First(&user, userID)
	response.OK(c, "操作成功", user)
}

func (h *HomeAuthHandler) OAuthRedirect(c *gin.Context) {
	provider := strings.ToLower(c.Param("provider"))
	if strings.HasSuffix(c.FullPath(), "/callback") {
		h.OAuthCallback(c)
		return
	}
	oauthCfg, ok := h.oauthProviderConfig(provider)
	if !ok {
		response.Error(c, http.StatusBadRequest, 1, "不支持的第三方授权")
		return
	}
	if oauthCfg.ClientID == "" || oauthCfg.ClientSecret == "" || oauthCfg.RedirectURI == "" {
		response.Error(c, http.StatusBadRequest, 1, provider+" 授权未配置 Client ID、Secret 或 Redirect")
		return
	}
	authURL := h.oauthAuthorizeURL(provider, oauthCfg)
	if authURL == "" {
		response.Error(c, http.StatusBadRequest, 1, "不支持的第三方授权")
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h *HomeAuthHandler) OAuthCallback(c *gin.Context) {
	provider := strings.ToLower(c.Param("provider"))
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		response.Error(c, http.StatusBadRequest, 1, "授权回调缺少 code")
		return
	}
	oauthCfg, ok := h.oauthProviderConfig(provider)
	if !ok || oauthCfg.ClientID == "" || oauthCfg.ClientSecret == "" || oauthCfg.RedirectURI == "" {
		response.Error(c, http.StatusBadRequest, 1, provider+" 授权未配置")
		return
	}
	profile, err := h.fetchOAuthProfile(provider, oauthCfg, code)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, err.Error())
		return
	}
	user, err := h.findOrCreateOAuthUser(profile)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	token, err := service.MakeUserToken(user, h.cfg)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	userJSON, _ := json.Marshal(user)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<!doctype html><meta charset="utf-8"><title>授权成功</title><script>
localStorage.setItem('home_token', %q);
localStorage.setItem('home_user', %q);
location.replace('/user');
</script>`, token, string(userJSON))))
}

func (h *HomeAuthHandler) oauthProviderConfig(provider string) (config.OAuthProviderConfig, bool) {
	switch provider {
	case "github":
		return h.cfg.OAuth.Github, true
	case "qq":
		return h.cfg.OAuth.QQ, true
	case "weibo":
		return h.cfg.OAuth.Weibo, true
	default:
		return config.OAuthProviderConfig{}, false
	}
}

func (h *HomeAuthHandler) oauthAuthorizeURL(provider string, cfg config.OAuthProviderConfig) string {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", cfg.RedirectURI)
	values.Set("response_type", "code")
	switch provider {
	case "github":
		values.Set("scope", "read:user user:email")
		return "https://github.com/login/oauth/authorize?" + values.Encode()
	case "qq":
		values.Set("scope", "get_user_info")
		return "https://graph.qq.com/oauth2.0/authorize?" + values.Encode()
	case "weibo":
		return "https://api.weibo.com/oauth2/authorize?" + values.Encode()
	default:
		return ""
	}
}

func (h *HomeAuthHandler) fetchOAuthProfile(provider string, cfg config.OAuthProviderConfig, code string) (oauthProfile, error) {
	switch provider {
	case "github":
		return h.fetchGithubProfile(cfg, code)
	case "qq":
		return h.fetchQQProfile(cfg, code)
	case "weibo":
		return h.fetchWeiboProfile(cfg, code)
	default:
		return oauthProfile{}, fmt.Errorf("不支持的第三方授权")
	}
}

func (h *HomeAuthHandler) fetchGithubProfile(cfg config.OAuthProviderConfig, code string) (oauthProfile, error) {
	token, err := postOAuthToken("https://github.com/login/oauth/access_token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURI},
		"code":          {code},
	})
	if err != nil {
		return oauthProfile{}, err
	}
	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getOAuthJSON("https://api.github.com/user", token, &user); err != nil {
		return oauthProfile{}, err
	}
	email := user.Email
	if email == "" {
		var emails []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}
		if err := getOAuthJSON("https://api.github.com/user/emails", token, &emails); err == nil {
			for _, item := range emails {
				if item.Primary {
					email = item.Email
					break
				}
			}
		}
	}
	return oauthProfile{ID: fmt.Sprintf("%d", user.ID), Name: firstNonEmpty(user.Name, user.Login), Email: email, Avatar: user.AvatarURL, Provider: "github"}, nil
}

func (h *HomeAuthHandler) fetchQQProfile(cfg config.OAuthProviderConfig, code string) (oauthProfile, error) {
	token, err := getOAuthToken("https://graph.qq.com/oauth2.0/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURI},
		"code":          {code},
	})
	if err != nil {
		return oauthProfile{}, err
	}
	body, err := httpGetString("https://graph.qq.com/oauth2.0/me?access_token=" + url.QueryEscape(token))
	if err != nil {
		return oauthProfile{}, err
	}
	body = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(body), ");"), "callback(")
	var me struct {
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal([]byte(body), &me); err != nil || me.OpenID == "" {
		return oauthProfile{}, fmt.Errorf("QQ openid 获取失败")
	}
	var info struct {
		Nickname    string `json:"nickname"`
		FigureURLQQ string `json:"figureurl_qq_2"`
	}
	infoURL := "https://graph.qq.com/user/get_user_info?access_token=" + url.QueryEscape(token) + "&oauth_consumer_key=" + url.QueryEscape(cfg.ClientID) + "&openid=" + url.QueryEscape(me.OpenID)
	if err := getJSON(infoURL, &info); err != nil {
		return oauthProfile{}, err
	}
	return oauthProfile{ID: me.OpenID, Name: info.Nickname, Avatar: info.FigureURLQQ, Provider: "qq"}, nil
}

func (h *HomeAuthHandler) fetchWeiboProfile(cfg config.OAuthProviderConfig, code string) (oauthProfile, error) {
	body, err := httpPostForm("https://api.weibo.com/oauth2/access_token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURI},
		"grant_type":    {"authorization_code"},
		"code":          {code},
	})
	if err != nil {
		return oauthProfile{}, err
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		UID         string `json:"uid"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.AccessToken == "" || payload.UID == "" {
		return oauthProfile{}, fmt.Errorf("微博授权令牌获取失败")
	}
	var info struct {
		ID              int64  `json:"id"`
		ScreenName      string `json:"screen_name"`
		AvatarLarge     string `json:"avatar_large"`
		ProfileImageURL string `json:"profile_image_url"`
	}
	infoURL := "https://api.weibo.com/2/users/show.json?access_token=" + url.QueryEscape(payload.AccessToken) + "&uid=" + url.QueryEscape(payload.UID)
	if err := getJSON(infoURL, &info); err != nil {
		return oauthProfile{}, err
	}
	return oauthProfile{ID: fmt.Sprintf("%d", info.ID), Name: info.ScreenName, Avatar: firstNonEmpty(info.AvatarLarge, info.ProfileImageURL), Provider: "weibo"}, nil
}

func (h *HomeAuthHandler) findOrCreateOAuthUser(profile oauthProfile) (model.User, error) {
	var user model.User
	query := h.db.Where("provider = ? AND provider_id = ?", profile.Provider, profile.ID)
	if profile.Email != "" {
		query = h.db.Where("email = ?", profile.Email).Or("provider = ? AND provider_id = ?", profile.Provider, profile.ID)
	}
	if err := query.First(&user).Error; err == nil {
		return user, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return user, err
	}
	password, err := bcrypt.GenerateFromPassword([]byte(profile.Provider+":"+profile.ID+":"+time.Now().String()), bcrypt.DefaultCost)
	if err != nil {
		return user, err
	}
	user = model.User{
		Name:       firstNonEmpty(profile.Name, profile.Provider+"用户"),
		Email:      profile.Email,
		Avatar:     firstNonEmpty(profile.Avatar, "/images/config/avatar.jpg"),
		Password:   string(password),
		Provider:   profile.Provider,
		ProviderID: profile.ID,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func (h *HomeAuthHandler) saveEmailCode(email string, purpose string, code string) bool {
	key := emailCodeKey(email, purpose)
	now := time.Now()
	h.codeMu.Lock()
	defer h.codeMu.Unlock()
	if old, ok := h.emailCodes[key]; ok && now.Sub(old.SentAt) < time.Minute {
		return false
	}
	h.emailCodes[key] = emailCode{
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: now.Add(10 * time.Minute),
		SentAt:    now,
	}
	return true
}

func (h *HomeAuthHandler) verifyEmailCode(email string, purpose string, code string) bool {
	key := emailCodeKey(email, purpose)
	h.codeMu.Lock()
	defer h.codeMu.Unlock()
	item, ok := h.emailCodes[key]
	if !ok || item.Purpose != purpose || time.Now().After(item.ExpiresAt) || item.Code != strings.TrimSpace(code) {
		return false
	}
	delete(h.emailCodes, key)
	return true
}

func (h *HomeAuthHandler) deleteEmailCode(email string, purpose string) {
	h.codeMu.Lock()
	defer h.codeMu.Unlock()
	delete(h.emailCodes, emailCodeKey(email, purpose))
}

func (h *HomeAuthHandler) saveCaptcha(key string, code string) {
	h.codeMu.Lock()
	defer h.codeMu.Unlock()
	now := time.Now()
	for itemKey, item := range h.captchas {
		if now.After(item.ExpiresAt) {
			delete(h.captchas, itemKey)
		}
	}
	h.captchas[key] = captchaCode{Code: code, ExpiresAt: now.Add(5 * time.Minute)}
}

func (h *HomeAuthHandler) verifyCaptcha(key string, code string) bool {
	h.codeMu.Lock()
	defer h.codeMu.Unlock()
	item, ok := h.captchas[strings.TrimSpace(key)]
	if !ok || time.Now().After(item.ExpiresAt) {
		return false
	}
	delete(h.captchas, strings.TrimSpace(key))
	return item.Code == strings.TrimSpace(code)
}

func emailCodeKey(email string, purpose string) string {
	return strings.ToLower(strings.TrimSpace(purpose)) + ":" + strings.ToLower(strings.TrimSpace(email))
}

func randomNumericCode(length int) (string, error) {
	var builder strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteString(n.String())
	}
	return builder.String(), nil
}

func secureRandInt(min int64, max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return 0, err
	}
	return min + n.Int64(), nil
}

func randomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func buildCaptchaSVG(text string) string {
	escaped := template.HTMLEscapeString(text)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="42" viewBox="0 0 128 42">
<rect width="128" height="42" rx="8" fill="#f8fafc"/>
<path d="M8 31 C24 10, 40 34, 58 16 S94 30, 120 12" fill="none" stroke="#99f6e4" stroke-width="3" opacity=".85"/>
<circle cx="22" cy="13" r="2" fill="#2563eb" opacity=".35"/>
<circle cx="103" cy="29" r="2" fill="#f59e0b" opacity=".45"/>
<text x="64" y="27" text-anchor="middle" font-family="Consolas, Menlo, monospace" font-size="20" font-weight="700" fill="#111827">%s</text>
</svg>`, escaped)
}

func postOAuthToken(endpoint string, values url.Values) (string, error) {
	body, err := httpPostForm(endpoint, values)
	if err != nil {
		return "", err
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.AccessToken != "" {
		return payload.AccessToken, nil
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("授权令牌响应异常")
	}
	token := parsed.Get("access_token")
	if token == "" {
		if payload.ErrorDesc != "" {
			return "", fmt.Errorf("%s", payload.ErrorDesc)
		}
		if payload.Error != "" {
			return "", fmt.Errorf("%s", payload.Error)
		}
		return "", fmt.Errorf("授权令牌获取失败")
	}
	return token, nil
}

func getOAuthToken(endpoint string, values url.Values) (string, error) {
	body, err := httpGetString(endpoint + "?" + values.Encode())
	if err != nil {
		return "", err
	}
	parsed, err := url.ParseQuery(body)
	if err != nil {
		return "", fmt.Errorf("授权令牌响应异常")
	}
	token := parsed.Get("access_token")
	if token == "" {
		return "", fmt.Errorf("授权令牌获取失败")
	}
	return token, nil
}

func httpPostForm(endpoint string, values url.Values) ([]byte, error) {
	resp, err := http.PostForm(endpoint, values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("OAuth 服务异常：%s", resp.Status)
	}
	return body, nil
}

func httpGetString(endpoint string) (string, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("OAuth 服务异常：%s", resp.Status)
	}
	return string(body), nil
}

func getJSON(endpoint string, target any) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("OAuth 服务异常：%s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func getOAuthJSON(endpoint string, token string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("OAuth 服务异常：%s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
