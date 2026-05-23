package handler

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WechatHandler struct {
	cfg config.Config
	db  *gorm.DB
}

type wechatMessage struct {
	XMLName     xml.Name `xml:"xml"`
	ToUser      string   `xml:"ToUserName"`
	FromUser    string   `xml:"FromUserName"`
	CreateTime  int64    `xml:"CreateTime"`
	MsgType     string   `xml:"MsgType"`
	Content     string   `xml:"Content"`
	PicURL      string   `xml:"PicUrl"`
	Recognition string   `xml:"Recognition"`
	Event       string   `xml:"Event"`
}

func NewWechatHandler(cfg config.Config, db *gorm.DB) *WechatHandler {
	return &WechatHandler{cfg: cfg, db: db}
}

func (h *WechatHandler) Verify(c *gin.Context) {
	if !h.validSignature(c) {
		c.String(http.StatusForbidden, "signature invalid")
		return
	}
	c.String(http.StatusOK, c.Query("echostr"))
}

func (h *WechatHandler) Serve(c *gin.Context) {
	if !h.validSignature(c) {
		c.String(http.StatusForbidden, "signature invalid")
		return
	}

	var message wechatMessage
	if err := c.ShouldBindXML(&message); err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	reply := h.replyText(message)
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(buildWechatTextXML(message.FromUser, message.ToUser, reply)))
}

func (h *WechatHandler) validSignature(c *gin.Context) bool {
	token := strings.TrimSpace(h.setting("wechat_official_account_token", h.cfg.Wechat.Token))
	if token == "" {
		return true
	}
	items := []string{token, c.Query("timestamp"), c.Query("nonce")}
	sort.Strings(items)
	sum := sha1.Sum([]byte(strings.Join(items, "")))
	return strings.EqualFold(hex.EncodeToString(sum[:]), c.Query("signature"))
}

func (h *WechatHandler) replyText(message wechatMessage) string {
	switch strings.ToLower(message.MsgType) {
	case "event":
		if strings.EqualFold(message.Event, "subscribe") {
			if value := h.keywordReply("关注"); value != "" {
				return value
			}
			return "我是一个有爆料、有态度、有内涵、有技术的公众号。感谢关注！回复关键词可解锁技能哦。"
		}
	case "text":
		if value := h.keywordReply(strings.TrimSpace(message.Content)); value != "" {
			return value
		}
		if value := h.aiTextReply(message.Content); value != "" {
			return value
		}
		return "暂时没有匹配到关键词，换个词试试吧。"
	case "image":
		if value := h.tulingReply(message.PicURL, 1); value != "" {
			return value
		}
		return "收到图片消息"
	case "voice":
		if strings.TrimSpace(message.Recognition) != "" {
			if value := h.keywordReply(message.Recognition); value != "" {
				return value
			}
			if value := h.aiTextReply(message.Recognition); value != "" {
				return value
			}
		}
		return "收到语音消息"
	case "video":
		return "收到视频消息"
	case "location":
		return "收到坐标消息"
	case "link":
		return "收到链接消息"
	}
	return "收到消息"
}

func (h *WechatHandler) keywordReply(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}
	var item model.WxKeyword
	if err := h.db.Where("status = ? AND key_name LIKE ?", 1, "%"+keyword+"%").
		Order("sort DESC, id DESC").
		First(&item).Error; err != nil {
		return ""
	}
	return item.KeyValue
}

func (h *WechatHandler) aiTextReply(question string) string {
	if value := h.qqAIReply(question); value != "" {
		return value
	}
	return h.tulingReply(question, 0)
}

func (h *WechatHandler) qqAIReply(question string) string {
	question = strings.TrimSpace(question)
	appID := h.setting("qq_ai_appid", h.cfg.QQAI.AppID)
	appKey := h.setting("qq_ai_appkey", h.cfg.QQAI.AppKey)
	apiURL := h.setting("qq_ai_url", h.cfg.QQAI.URL)
	if question == "" || appID == "" || appKey == "" || apiURL == "" {
		return ""
	}
	params := url.Values{}
	params.Set("app_id", appID)
	params.Set("time_stamp", strconv.FormatInt(time.Now().Unix(), 10))
	params.Set("nonce_str", strconv.FormatInt(time.Now().UnixNano(), 10))
	params.Set("question", question)
	params.Set("session", question)
	params.Set("sign", qqAISign(params, appKey))

	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.PostForm(apiURL, params)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}
	var payload struct {
		Msg  string `json:"msg"`
		Data struct {
			Answer string `json:"answer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ""
	}
	if payload.Msg == "ok" && payload.Data.Answer != "" {
		return trimWechatReply(payload.Data.Answer)
	}
	return ""
}

func (h *WechatHandler) tulingReply(input string, reqType int) string {
	input = strings.TrimSpace(input)
	apiKey := h.setting("tuling_api_key", h.cfg.Tuling.APIKey)
	apiURL := h.setting("tuling_api_url", h.cfg.Tuling.APIURL)
	if input == "" || apiKey == "" || apiURL == "" {
		return ""
	}
	perception := map[string]any{}
	switch reqType {
	case 1:
		perception["inputImage"] = map[string]string{"url": input}
	case 2:
		perception["inputMedia"] = map[string]string{"url": input}
	default:
		perception["inputText"] = map[string]string{"text": input}
	}
	payload := map[string]any{
		"reqType":    reqType,
		"perception": perception,
		"userInfo": map[string]string{
			"apiKey": apiKey,
			"userId": "wjfcm-go-wechat",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var result struct {
		Results []struct {
			ResultType string            `json:"resultType"`
			Values     map[string]string `json:"values"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Results) == 0 {
		return ""
	}
	first := result.Results[0]
	switch first.ResultType {
	case "text":
		return trimWechatReply(first.Values["text"])
	case "image":
		return first.Values["image"]
	case "url":
		return first.Values["url"]
	default:
		return ""
	}
}

func (h *WechatHandler) setting(key string, fallback string) string {
	var item model.SystemConfig
	if err := h.db.Where("`key` = ? AND status = ?", key, 1).First(&item).Error; err == nil && strings.TrimSpace(item.Value) != "" {
		return strings.TrimSpace(item.Value)
	}
	return strings.TrimSpace(fallback)
}

func qqAISign(params url.Values, appKey string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "sign" && params.Get(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		parts = append(parts, key+"="+url.QueryEscape(params.Get(key)))
	}
	parts = append(parts, "app_key="+appKey)
	sum := md5.Sum([]byte(strings.Join(parts, "&")))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func trimWechatReply(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 500 {
		return string(runes[:500]) + "......"
	}
	return value
}

func buildWechatTextXML(toUser string, fromUser string, content string) string {
	return `<xml>
<ToUserName><![CDATA[` + toUser + `]]></ToUserName>
<FromUserName><![CDATA[` + fromUser + `]]></FromUserName>
<CreateTime>` + strconv.FormatInt(time.Now().Unix(), 10) + `</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[` + strings.ReplaceAll(content, "]]>", "]]]]><![CDATA[>") + `]]></Content>
</xml>`
}
