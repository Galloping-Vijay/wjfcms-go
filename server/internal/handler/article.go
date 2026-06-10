package handler

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"wjfcms-go/internal/config"
	"wjfcms-go/internal/model"
	"wjfcms-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ArticleHandler struct {
	db  *gorm.DB
	cfg config.Config
}

type articleRequest struct {
	CategoryID  uint   `json:"category_id"`
	Title       string `json:"title" binding:"required"`
	Author      string `json:"author"`
	Content     string `json:"content"`
	Markdown    string `json:"markdown"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Cover       string `json:"cover"`
	IsTop       bool   `json:"is_top"`
	Status      int8   `json:"status"`
	Click       uint   `json:"click"`
	IsBaijiahao bool   `json:"is_baijiahao"`
}

func NewArticleHandler(cfg config.Config, db *gorm.DB) *ArticleHandler {
	return &ArticleHandler{cfg: cfg, db: db}
}

func (h *ArticleHandler) Index(c *gin.Context) {
	var articles []model.Article
	query := h.db.Model(&model.Article{}).Preload("Category")

	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR keywords LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if title := c.Query("title"); title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}
	if author := c.Query("author"); author != "" {
		query = query.Where("author LIKE ?", "%"+author+"%")
	}
	if categoryID := c.Query("category_id"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if isTop := c.Query("is_top"); isTop != "" {
		query = query.Where("is_top = ?", isTop)
	}
	query = applyDeleteFilter(query, c.Query("delete"))

	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)

	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles).Error
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	response.Page(c, "获取成功", articles, total)
}

func (h *ArticleHandler) PublicIndex(c *gin.Context) {
	var articles []model.Article
	query := h.db.Model(&model.Article{}).Preload("Category").Where("status = ?", 1)

	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR keywords LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if categoryID := c.Query("category_id"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if tag := c.Query("tag"); tag != "" {
		query = query.Where("keywords LIKE ?", "%"+tag+"%")
	}
	if isTop := c.Query("is_top"); isTop != "" {
		query = query.Where("is_top = ?", isTop)
	}

	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)

	err := query.Order("is_top DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles).Error
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	response.Page(c, "获取成功", articles, total)
}

func (h *ArticleHandler) PublicArchive(c *gin.Context) {
	var categories []model.Category
	if err := h.db.Order("sort ASC, id ASC").Find(&categories).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	var articles []model.Article
	if err := h.db.
		Preload("Category").
		Where("status = ?", 1).
		Select("id", "title", "category_id", "created_at", "click").
		Order("id ASC").
		Find(&articles).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	type archiveArticle struct {
		ID           uint64 `json:"id"`
		Title        string `json:"title"`
		CategoryID   uint   `json:"category_id"`
		CategoryName string `json:"category_name"`
		CreatedAt    any    `json:"created_at"`
		Click        uint   `json:"click"`
	}
	type archiveGroup struct {
		ID       uint             `json:"id"`
		Name     string           `json:"name"`
		Articles []archiveArticle `json:"articles"`
	}

	groups := make([]archiveGroup, 0, len(categories))
	groupIndex := make(map[uint]int, len(categories))
	for _, category := range categories {
		groupIndex[category.ID] = len(groups)
		groups = append(groups, archiveGroup{ID: category.ID, Name: category.Name, Articles: []archiveArticle{}})
	}

	for _, article := range articles {
		index, ok := groupIndex[article.CategoryID]
		if !ok {
			continue
		}
		categoryName := ""
		if article.Category != nil {
			categoryName = article.Category.Name
		}
		groups[index].Articles = append(groups[index].Articles, archiveArticle{
			ID: article.ID, Title: article.Title, CategoryID: article.CategoryID,
			CategoryName: categoryName, CreatedAt: article.CreatedAt, Click: article.Click,
		})
	}

	response.OK(c, "获取成功", groups)
}

func (h *ArticleHandler) Show(c *gin.Context) {
	var article model.Article
	if err := h.db.Preload("Category").First(&article, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}
	article.Content = html.UnescapeString(article.Content)
	response.OK(c, "获取成功", article)
}

func (h *ArticleHandler) PublicShow(c *gin.Context) {
	var article model.Article
	if err := h.db.Preload("Category").Where("status = ?", 1).First(&article, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}
	h.db.Model(&article).UpdateColumn("click", gorm.Expr("click + ?", 1))
	article.Click++
	article.Content = html.UnescapeString(article.Content)

	var previous, next model.Article
	previousOK := h.db.Where("status = ? AND id < ?", 1, article.ID).
		Select("id", "title").
		Order("id DESC").
		First(&previous).Error == nil
	nextOK := h.db.Where("status = ? AND id > ?", 1, article.ID).
		Select("id", "title").
		Order("id ASC").
		First(&next).Error == nil

	data := gin.H{"article": article}
	if previousOK {
		data["previous"] = previous
	}
	if nextOK {
		data["next"] = next
	}
	response.OK(c, "获取成功", data)
}

func (h *ArticleHandler) Store(c *gin.Context) {
	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写文章标题")
		return
	}

	article := model.Article{
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Author:      req.Author,
		Content:     req.Content,
		Markdown:    req.Markdown,
		Description: req.Description,
		Keywords:    req.Keywords,
		Cover:       req.Cover,
		IsTop:       req.IsTop,
		Status:      req.Status,
		Click:       req.Click,
		IsBaijiahao: req.IsBaijiahao,
	}

	if err := h.db.Create(&article).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", article)
}

func (h *ArticleHandler) Update(c *gin.Context) {
	var article model.Article
	if err := h.db.First(&article, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}

	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写文章标题")
		return
	}

	updates := map[string]any{
		"category_id":  req.CategoryID,
		"title":        req.Title,
		"author":       req.Author,
		"content":      req.Content,
		"markdown":     req.Markdown,
		"description":  req.Description,
		"keywords":     req.Keywords,
		"cover":        req.Cover,
		"is_top":       req.IsTop,
		"status":       req.Status,
		"click":        req.Click,
		"is_baijiahao": req.IsBaijiahao,
	}
	if err := h.db.Model(&article).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", article)
}

func (h *ArticleHandler) Replace(c *gin.Context) {
	var req struct {
		Search  string   `json:"search"`
		Replace string   `json:"replace"`
		Fields  []string `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写替换内容")
		return
	}
	search := strings.TrimSpace(req.Search)
	if search == "" {
		response.Error(c, http.StatusBadRequest, 1, "请填写要查找的内容")
		return
	}

	fields := articleReplaceFields(req.Fields)
	if len(fields) == 0 {
		response.Error(c, http.StatusBadRequest, 1, "请选择替换范围")
		return
	}

	escapedSearch := html.EscapeString(search)
	query := h.db.Model(&model.Article{}).Unscoped()
	for index, field := range fields {
		condition := field + " LIKE ?"
		if index == 0 {
			query = query.Where(condition, "%"+search+"%")
		} else {
			query = query.Or(condition, "%"+search+"%")
		}
		if field == "content" && escapedSearch != search {
			query = query.Or(condition, "%"+escapedSearch+"%")
		}
	}

	var articles []model.Article
	selectFields := append([]string{"id"}, fields...)
	if err := query.Select(selectFields).Find(&articles).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	changed := 0
	for _, article := range articles {
		updates := make(map[string]any)
		for _, field := range fields {
			raw := articleReplaceFieldValue(article, field)
			next := replaceArticleText(raw, search, req.Replace, field == "content")
			if next != raw {
				updates[field] = next
			}
		}
		if len(updates) == 0 {
			continue
		}
		if err := h.db.Unscoped().Model(&model.Article{}).Where("id = ?", article.ID).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			return
		}
		changed++
	}

	response.OK(c, "操作成功", gin.H{"matched": len(articles), "changed": changed})
}

func (h *ArticleHandler) Destroy(c *gin.Context) {
	var count int64
	h.db.Model(&model.Comment{}).Where("article_id = ?", c.Param("id")).Count(&count)
	if count > 0 {
		response.Error(c, http.StatusBadRequest, 1, "该文章存在评论，不能删除")
		return
	}
	if err := h.db.Delete(&model.Article{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ArticleHandler) Restore(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.db.Unscoped().Model(&model.Article{}).Where("id = ?", id).Update("deleted_at", nil).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ArticleHandler) ForceDelete(c *gin.Context) {
	if err := h.db.Unscoped().Delete(&model.Article{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *ArticleHandler) PublishBaijiahao(c *gin.Context) {
	if h.cfg.Baijiahao.AppID == "" || h.cfg.Baijiahao.AppToken == "" {
		response.Error(c, http.StatusBadRequest, 1, "请先配置 BAIJIAHAO_APP_ID 和 BAIJIAHAO_APP_TOKEN")
		return
	}

	var req struct {
		Original int `json:"original"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Original == 0 {
		req.Original = 1
	}

	var article model.Article
	if err := h.db.First(&article, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}
	if article.Status == 0 {
		response.Error(c, http.StatusBadRequest, 1, "文章未审核通过")
		return
	}
	if article.IsBaijiahao {
		response.Error(c, http.StatusBadRequest, 1, "文章已推送过了")
		return
	}

	coverImages, _ := json.Marshal([]map[string]string{{"src": article.Cover}})
	form := url.Values{}
	form.Set("app_id", h.cfg.Baijiahao.AppID)
	form.Set("app_token", h.cfg.Baijiahao.AppToken)
	form.Set("title", article.Title)
	form.Set("content", html.UnescapeString(article.Content))
	form.Set("origin_url", strings.TrimRight(h.cfg.App.URL, "/")+"/article/"+strconv.FormatUint(article.ID, 10))
	form.Set("cover_images", string(coverImages))
	form.Set("is_original", strconv.Itoa(req.Original))

	httpClient := http.Client{Timeout: 15 * time.Second}
	httpResp, err := httpClient.Post(
		"http://baijiahao.baidu.com/builderinner/open/resource/article/publish",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, "百家号推送失败："+err.Error())
		return
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		response.Error(c, http.StatusBadGateway, 1, "读取百家号响应失败")
		return
	}
	var result struct {
		Errno  int    `json:"errno"`
		Errmsg string `json:"errmsg"`
		Data   any    `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Errmsg == "" {
		response.Error(c, http.StatusBadGateway, 1, "百家号响应异常")
		return
	}
	if result.Errno != 0 {
		response.Error(c, http.StatusBadGateway, 1, result.Errmsg)
		return
	}

	if err := h.db.Model(&article).Update("is_baijiahao", true).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, result.Errmsg, gin.H{"id": article.ID, "is_baijiahao": true, "result": result.Data})
}

func pageParams(c *gin.Context) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("limit", c.DefaultQuery("page_size", "10")))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func articleReplaceFields(fields []string) []string {
	if len(fields) == 0 {
		fields = []string{"content", "markdown"}
	}
	allowed := map[string]bool{
		"title":       true,
		"description": true,
		"content":     true,
		"markdown":    true,
	}
	result := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if !allowed[field] || seen[field] {
			continue
		}
		seen[field] = true
		result = append(result, field)
	}
	return result
}

func articleReplaceFieldValue(article model.Article, field string) string {
	switch field {
	case "title":
		return article.Title
	case "description":
		return article.Description
	case "content":
		return article.Content
	case "markdown":
		return article.Markdown
	default:
		return ""
	}
}

func replaceArticleText(raw string, search string, replacement string, escaped bool) string {
	next := strings.ReplaceAll(raw, search, replacement)
	if !escaped {
		return next
	}

	escapedSearch := html.EscapeString(search)
	if escapedSearch != search {
		next = strings.ReplaceAll(next, escapedSearch, html.EscapeString(replacement))
	}
	if next == raw {
		visible := html.UnescapeString(raw)
		replaced := strings.ReplaceAll(visible, search, replacement)
		if replaced != visible {
			next = html.EscapeString(replaced)
		}
	}
	return next
}
