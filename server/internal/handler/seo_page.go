package handler

import (
	"encoding/xml"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"wjfcm-go/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const seoPageSize = 10

type SEOPageHandler struct {
	db *gorm.DB
}

type seoArticleItem struct {
	ID              uint64
	Title           string
	TitleHTML       template.HTML
	Description     string
	DescriptionHTML template.HTML
	Keywords        string
	Author          string
	Cover           string
	Click           uint
	IsTop           bool
	CreatedAt       time.Time
}

type seoPageData struct {
	SiteName         string
	Title            string
	Keywords         string
	Description      string
	Canonical        string
	OGType           string
	OGImage          string
	JSONLD           template.JS
	Configs          map[string]string
	Categories       []model.Category
	Navs             []model.Nav
	Tags             []model.Tag
	HotArticles      []model.Article
	FriendLinks      []model.FriendLink
	Articles         []seoArticleItem
	TopArticles      []seoArticleItem
	Archive          []seoArchiveGroup
	Chats            []model.Chat
	Article          *model.Article
	PrevArticle      *model.Article
	NextArticle      *model.Article
	Category         *model.Category
	Tag              *model.Tag
	Query            string
	Page             int
	HasPrev          bool
	HasNext          bool
	PrevURL          string
	NextURL          string
	Pages            []seoPageLink
	CurrentPath      string
	ActiveCategoryID uint
	Year             int
	Message          string
}

type seoPageLink struct {
	Number int
	URL    string
	Active bool
}

type seoArchiveArticle struct {
	ID        uint64
	Title     string
	CreatedAt time.Time
	Click     uint
}

type seoArchiveGroup struct {
	ID       uint
	Name     string
	Articles []seoArchiveArticle
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func NewSEOPageHandler(db *gorm.DB) *SEOPageHandler {
	return &SEOPageHandler{db: db}
}

func (h *SEOPageHandler) Index(c *gin.Context) {
	page := positivePage(c.Query("page"))
	articles, total, err := h.listArticles(page, func(db *gorm.DB) *gorm.DB {
		return db.Where("is_top = ?", 0)
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	topArticles, err := h.topArticles()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	data := h.baseData(c)
	data.Title = h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = h.configValue(data.Configs, "site_seo_description", h.configValue(data.Configs, "site_description", ""))
	data.Keywords = h.configValue(data.Configs, "site_seo_keywords", "")
	data.Canonical = absoluteRequestURL(c, "/")
	data.Articles = articles
	if page == 1 {
		data.TopArticles = topArticles
	}
	data.Page = page
	data.HasPrev = page > 1
	data.HasNext = page*seoPageSize < int(total)
	data.PrevURL = pageURL("/", page-1)
	data.NextURL = pageURL("/", page+1)
	data.Pages = buildPageLinks("/", page, total)
	data.JSONLD = template.JS(h.websiteJSONLD(data, "/"))
	c.HTML(http.StatusOK, "seo_index.tmpl", data)
}

func (h *SEOPageHandler) Category(c *gin.Context) {
	var category model.Category
	if err := h.db.First(&category, c.Param("id")).Error; err != nil {
		c.String(http.StatusNotFound, "分类不存在")
		return
	}
	page := positivePage(c.Query("page"))
	articles, total, err := h.listArticles(page, func(db *gorm.DB) *gorm.DB {
		return db.Where("category_id = ?", category.ID)
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	path := "/category/" + c.Param("id")
	data := h.baseData(c)
	data.Title = category.Name + " | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = category.Description
	data.Keywords = firstNonEmpty(category.Keywords, category.Name)
	data.Canonical = absoluteRequestURL(c, path)
	data.Category = &category
	data.ActiveCategoryID = category.ID
	data.Articles = articles
	data.Page = page
	data.HasPrev = page > 1
	data.HasNext = page*seoPageSize < int(total)
	data.PrevURL = pageURL(path, page-1)
	data.NextURL = pageURL(path, page+1)
	data.Pages = buildPageLinks(path, page, total)
	data.JSONLD = template.JS(h.collectionJSONLD(data, category.Name, path))
	c.HTML(http.StatusOK, "seo_list.tmpl", data)
}

func (h *SEOPageHandler) Tag(c *gin.Context) {
	var tag model.Tag
	if err := h.db.First(&tag, c.Param("id")).Error; err != nil {
		c.String(http.StatusNotFound, "标签不存在")
		return
	}
	page := positivePage(c.Query("page"))
	articles, total, err := h.listArticles(page, func(db *gorm.DB) *gorm.DB {
		return db.Where("keywords LIKE ?", "%"+tag.Name+"%")
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	path := "/tag/" + c.Param("id")
	data := h.baseData(c)
	data.Title = tag.Name + " | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "和 " + tag.Name + " 相关的文章"
	data.Keywords = tag.Name
	data.Canonical = absoluteRequestURL(c, path)
	data.Tag = &tag
	data.Articles = articles
	data.Page = page
	data.HasPrev = page > 1
	data.HasNext = page*seoPageSize < int(total)
	data.PrevURL = pageURL(path, page-1)
	data.NextURL = pageURL(path, page+1)
	data.Pages = buildPageLinks(path, page, total)
	data.JSONLD = template.JS(h.collectionJSONLD(data, tag.Name, path))
	c.HTML(http.StatusOK, "seo_list.tmpl", data)
}

func (h *SEOPageHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	page := positivePage(c.Query("page"))
	articles, total, err := h.listArticles(page, func(db *gorm.DB) *gorm.DB {
		if query == "" {
			return db
		}
		like := "%" + query + "%"
		return db.Where("title LIKE ? OR description LIKE ? OR keywords LIKE ?", like, like, like)
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	path := "/search"
	data := h.baseData(c)
	data.Title = "搜索：" + query + " | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "站内搜索：" + query
	data.Keywords = query
	data.Canonical = absoluteRequestURL(c, path)
	data.Query = query
	data.Articles = highlightArticles(articles, query)
	data.Page = page
	data.HasPrev = page > 1
	data.HasNext = page*seoPageSize < int(total)
	data.PrevURL = pageURL(path+"?q="+url.QueryEscape(query), page-1)
	data.NextURL = pageURL(path+"?q="+url.QueryEscape(query), page+1)
	data.Pages = buildPageLinks(path+"?q="+url.QueryEscape(query), page, total)
	data.JSONLD = template.JS(h.collectionJSONLD(data, "搜索："+query, path))
	c.HTML(http.StatusOK, "seo_list.tmpl", data)
}

func (h *SEOPageHandler) Archive(c *gin.Context) {
	data := h.baseData(c)
	data.Title = "文章归档 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "按分类整理的文章归档"
	data.Keywords = "文章归档," + h.configValue(data.Configs, "site_seo_keywords", "")
	data.Canonical = absoluteRequestURL(c, "/archive")
	data.Archive = h.archiveGroups()
	data.JSONLD = template.JS(h.collectionJSONLD(data, "文章归档", "/archive"))
	c.HTML(http.StatusOK, "seo_archive.tmpl", data)
}

func (h *SEOPageHandler) Chat(c *gin.Context) {
	var chats []model.Chat
	page := positivePage(c.Query("page"))
	query := h.db.Model(&model.Chat{})
	var total int64
	query.Count(&total)
	if err := query.Order("id DESC").Offset((page - 1) * seoPageSize).Limit(seoPageSize).Find(&chats).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	data := h.baseData(c)
	data.Title = "有些话 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "一些碎片想法和站点动态"
	data.Keywords = "有些话,碎片想法," + h.configValue(data.Configs, "site_seo_keywords", "")
	data.Canonical = absoluteRequestURL(c, "/chat")
	data.Chats = chats
	data.Page = page
	data.HasPrev = page > 1
	data.HasNext = page*seoPageSize < int(total)
	data.PrevURL = pageURL("/chat", page-1)
	data.NextURL = pageURL("/chat", page+1)
	data.Pages = buildPageLinks("/chat", page, total)
	data.JSONLD = template.JS(h.collectionJSONLD(data, "有些话", "/chat"))
	c.HTML(http.StatusOK, "seo_chat.tmpl", data)
}

func (h *SEOPageHandler) Login(c *gin.Context) {
	data := h.baseData(c)
	data.Title = "登录 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "用户登录"
	data.Keywords = "登录," + data.SiteName
	data.Canonical = absoluteRequestURL(c, "/login")
	data.JSONLD = template.JS(h.collectionJSONLD(data, "登录", "/login"))
	c.HTML(http.StatusOK, "seo_login.tmpl", data)
}

func (h *SEOPageHandler) Register(c *gin.Context) {
	data := h.baseData(c)
	data.Title = "注册 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "用户注册"
	data.Keywords = "注册," + data.SiteName
	data.Canonical = absoluteRequestURL(c, "/register")
	data.JSONLD = template.JS(h.collectionJSONLD(data, "注册", "/register"))
	c.HTML(http.StatusOK, "seo_register.tmpl", data)
}

func (h *SEOPageHandler) ForgotPassword(c *gin.Context) {
	data := h.baseData(c)
	data.Title = "找回密码 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "找回密码"
	data.Keywords = "找回密码," + data.SiteName
	data.Canonical = absoluteRequestURL(c, "/forgot-password")
	data.JSONLD = template.JS(h.collectionJSONLD(data, "找回密码", "/forgot-password"))
	c.HTML(http.StatusOK, "seo_forgot_password.tmpl", data)
}

func (h *SEOPageHandler) User(c *gin.Context) {
	data := h.baseData(c)
	data.Title = "用户中心 | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = "用户中心"
	data.Keywords = "用户中心," + data.SiteName
	data.Canonical = absoluteRequestURL(c, "/user")
	data.JSONLD = template.JS(h.collectionJSONLD(data, "用户中心", "/user"))
	c.HTML(http.StatusOK, "seo_user.tmpl", data)
}

func (h *SEOPageHandler) Blank(c *gin.Context) {
	message := strings.TrimSpace(c.Query("message"))
	if message == "" {
		message = "页面错误"
	}
	data := h.baseData(c)
	data.Title = message + " | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = message
	data.Keywords = data.SiteName
	data.Canonical = absoluteRequestURL(c, "/blank")
	data.Message = message
	data.JSONLD = template.JS(h.collectionJSONLD(data, message, "/blank"))
	c.HTML(http.StatusOK, "seo_blank.tmpl", data)
}

func (h *SEOPageHandler) Article(c *gin.Context) {
	var article model.Article
	if err := h.db.Preload("Category").Where("status = ?", 1).First(&article, c.Param("id")).Error; err != nil {
		c.String(http.StatusNotFound, "文章不存在")
		return
	}
	path := "/article/" + c.Param("id")
	data := h.baseData(c)
	data.Title = article.Title + " | " + h.configValue(data.Configs, "seo_title", data.SiteName)
	data.Description = article.Description
	data.Keywords = article.Keywords
	data.Canonical = absoluteRequestURL(c, path)
	data.OGType = "article"
	data.OGImage = absoluteAssetURL(c, firstNonEmpty(article.Cover, "/images/config/avatar.jpg"))
	data.Article = &article
	data.PrevArticle = h.neighborArticle(article.ID, "prev")
	data.NextArticle = h.neighborArticle(article.ID, "next")
	data.ActiveCategoryID = article.CategoryID
	data.JSONLD = template.JS(h.articleJSONLD(data, article, path))
	c.HTML(http.StatusOK, "seo_article.tmpl", data)
}

func (h *SEOPageHandler) neighborArticle(id uint64, direction string) *model.Article {
	var item model.Article
	query := h.db.Where("status = ?", 1)
	if direction == "prev" {
		query = query.Where("id < ?", id).Order("id DESC")
	} else {
		query = query.Where("id > ?", id).Order("id ASC")
	}
	if err := query.First(&item).Error; err != nil {
		return nil
	}
	return &item
}

func (h *SEOPageHandler) Robots(c *gin.Context) {
	base := requestBaseURL(c)
	c.String(http.StatusOK, "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api\nSitemap: %s/sitemap.xml\n", base)
}

func (h *SEOPageHandler) Sitemap(c *gin.Context) {
	base := requestBaseURL(c)
	urls := []sitemapURL{
		{Loc: base + "/", ChangeFreq: "daily", Priority: "1.0"},
		{Loc: base + "/archive", ChangeFreq: "weekly", Priority: "0.6"},
		{Loc: base + "/chat", ChangeFreq: "weekly", Priority: "0.5"},
	}
	var categories []model.Category
	h.db.Order("sort DESC, id DESC").Find(&categories)
	for _, category := range categories {
		urls = append(urls, sitemapURL{Loc: fmt.Sprintf("%s/category/%d", base, category.ID), LastMod: category.UpdatedAt.Format("2006-01-02"), ChangeFreq: "weekly", Priority: "0.7"})
	}
	var tags []model.Tag
	h.db.Order("id DESC").Find(&tags)
	for _, tag := range tags {
		urls = append(urls, sitemapURL{Loc: fmt.Sprintf("%s/tag/%d", base, tag.ID), LastMod: tag.UpdatedAt.Format("2006-01-02"), ChangeFreq: "weekly", Priority: "0.6"})
	}
	var articles []model.Article
	h.db.Where("status = ?", 1).Order("id DESC").Limit(5000).Find(&articles)
	for _, article := range articles {
		urls = append(urls, sitemapURL{Loc: fmt.Sprintf("%s/article/%d", base, article.ID), LastMod: article.UpdatedAt.Format("2006-01-02"), ChangeFreq: "monthly", Priority: "0.8"})
	}
	c.XML(http.StatusOK, sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
}

func (h *SEOPageHandler) baseData(c *gin.Context) seoPageData {
	configs := h.configMap()
	var categories []model.Category
	h.db.Order("sort DESC, id DESC").Find(&categories)
	var navs []model.Nav
	h.db.Order("sort DESC, id DESC").Find(&navs)
	var tags []model.Tag
	h.db.Order("id DESC").Limit(30).Find(&tags)
	var hot []model.Article
	h.db.Where("status = ?", 1).Order("click DESC").Limit(8).Find(&hot)
	var links []model.FriendLink
	h.db.Where("status = ?", 1).Order("sort DESC, id DESC").Limit(30).Find(&links)
	siteName := h.configValue(configs, "site_name", "臭大佬")
	return seoPageData{
		SiteName:    siteName,
		OGType:      "website",
		OGImage:     absoluteAssetURL(c, h.configValue(configs, "site_logo", "/images/config/avatar.jpg")),
		Configs:     configs,
		Categories:  categories,
		Navs:        navs,
		Tags:        tags,
		HotArticles: hot,
		FriendLinks: links,
		CurrentPath: c.Request.URL.Path,
		Year:        time.Now().Year(),
	}
}

func (h *SEOPageHandler) topArticles() ([]seoArticleItem, error) {
	var articles []model.Article
	if err := h.db.Where("status = ? AND is_top = ?", 1, 1).Order("id DESC").Limit(6).Find(&articles).Error; err != nil {
		return nil, err
	}
	return h.articleItems(articles), nil
}

func (h *SEOPageHandler) listArticles(page int, scope func(*gorm.DB) *gorm.DB) ([]seoArticleItem, int64, error) {
	query := h.db.Model(&model.Article{}).Where("status = ?", 1)
	if scope != nil {
		query = scope(query)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var articles []model.Article
	if err := query.Order("is_top DESC, id DESC").Offset((page - 1) * seoPageSize).Limit(seoPageSize).Find(&articles).Error; err != nil {
		return nil, 0, err
	}
	return h.articleItems(articles), total, nil
}

func (h *SEOPageHandler) articleItems(articles []model.Article) []seoArticleItem {
	items := make([]seoArticleItem, 0, len(articles))
	for _, article := range articles {
		items = append(items, seoArticleItem{
			ID: article.ID, Title: article.Title, TitleHTML: template.HTML(template.HTMLEscapeString(article.Title)),
			Description: article.Description, DescriptionHTML: template.HTML(template.HTMLEscapeString(article.Description)), Keywords: article.Keywords,
			Author: firstNonEmpty(article.Author, "臭大佬"), Cover: firstNonEmpty(article.Cover, "/images/config/avatar.jpg"),
			Click: article.Click, IsTop: article.IsTop, CreatedAt: article.CreatedAt,
		})
	}
	return items
}

func (h *SEOPageHandler) archiveGroups() []seoArchiveGroup {
	var categories []model.Category
	if err := h.db.Order("sort ASC, id ASC").Find(&categories).Error; err != nil {
		return []seoArchiveGroup{}
	}
	groups := make([]seoArchiveGroup, 0, len(categories))
	groupIndex := make(map[uint]int, len(categories))
	for _, category := range categories {
		groupIndex[category.ID] = len(groups)
		groups = append(groups, seoArchiveGroup{ID: category.ID, Name: category.Name, Articles: []seoArchiveArticle{}})
	}
	var articles []model.Article
	if err := h.db.Where("status = ?", 1).Select("id", "title", "category_id", "created_at", "click").Order("id ASC").Find(&articles).Error; err != nil {
		return groups
	}
	for _, article := range articles {
		index, ok := groupIndex[article.CategoryID]
		if !ok {
			continue
		}
		groups[index].Articles = append(groups[index].Articles, seoArchiveArticle{
			ID: article.ID, Title: article.Title, CreatedAt: article.CreatedAt, Click: article.Click,
		})
	}
	return groups
}

func (h *SEOPageHandler) configMap() map[string]string {
	var rows []model.SystemConfig
	h.db.Where("status = ?", 1).Find(&rows)
	configs := make(map[string]string, len(rows))
	for _, row := range rows {
		configs[row.Key] = row.Value
	}
	return configs
}

func (h *SEOPageHandler) configValue(configs map[string]string, key string, fallback string) string {
	if value := strings.TrimSpace(configs[key]); value != "" {
		return value
	}
	return fallback
}

func (h *SEOPageHandler) websiteJSONLD(data seoPageData, _ string) string {
	return fmt.Sprintf(`{"@context":"https://schema.org","@type":"WebSite","name":%q,"url":%q,"description":%q,"potentialAction":{"@type":"SearchAction","target":%q,"query-input":"required name=search_term_string"}}`,
		data.SiteName, requestOriginFromCanonical(data.Canonical), data.Description, requestOriginFromCanonical(data.Canonical)+"/search?q={search_term_string}")
}

func (h *SEOPageHandler) collectionJSONLD(data seoPageData, name string, _ string) string {
	return fmt.Sprintf(`{"@context":"https://schema.org","@type":"CollectionPage","name":%q,"url":%q,"description":%q}`, name, data.Canonical, data.Description)
}

func (h *SEOPageHandler) articleJSONLD(data seoPageData, article model.Article, _ string) string {
	image := absoluteAssetURLFromCanonical(data.Canonical, firstNonEmpty(article.Cover, "/images/config/avatar.jpg"))
	return fmt.Sprintf(`{"@context":"https://schema.org","@type":"Article","headline":%q,"description":%q,"image":%q,"url":%q,"datePublished":%q,"dateModified":%q,"author":{"@type":"Person","name":%q},"publisher":{"@type":"Organization","name":%q,"logo":{"@type":"ImageObject","url":%q}}}`,
		article.Title, article.Description, image, data.Canonical, article.CreatedAt.Format(time.RFC3339), article.UpdatedAt.Format(time.RFC3339), firstNonEmpty(article.Author, data.SiteName), data.SiteName, absoluteAssetURLFromCanonical(data.Canonical, "/images/config/avatar.jpg"))
}

func positivePage(value string) int {
	page, _ := strconv.Atoi(value)
	if page < 1 {
		return 1
	}
	return page
}

func pageURL(path string, page int) string {
	if page <= 1 {
		return strings.Split(path, "?")[0]
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%spage=%d", path, sep, page)
}

func buildPageLinks(path string, current int, total int64) []seoPageLink {
	totalPages := int((total + int64(seoPageSize) - 1) / int64(seoPageSize))
	if totalPages <= 1 {
		return nil
	}
	start := current - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > totalPages {
		end = totalPages
	}
	if end-start < 4 {
		start = end - 4
		if start < 1 {
			start = 1
		}
	}
	links := make([]seoPageLink, 0, end-start+1)
	for i := start; i <= end; i++ {
		links = append(links, seoPageLink{Number: i, URL: pageURL(path, i), Active: i == current})
	}
	return links
}

func highlightArticles(items []seoArticleItem, keyword string) []seoArticleItem {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return items
	}
	for i := range items {
		items[i].TitleHTML = highlightText(items[i].Title, keyword)
		items[i].DescriptionHTML = highlightText(items[i].Description, keyword)
	}
	return items
}

func highlightText(value string, keyword string) template.HTML {
	escapedValue := template.HTMLEscapeString(value)
	escapedKeyword := template.HTMLEscapeString(keyword)
	if escapedKeyword == "" {
		return template.HTML(escapedValue)
	}
	lowerValue := strings.ToLower(escapedValue)
	lowerKeyword := strings.ToLower(escapedKeyword)
	var builder strings.Builder
	for {
		index := strings.Index(lowerValue, lowerKeyword)
		if index < 0 {
			builder.WriteString(escapedValue)
			break
		}
		builder.WriteString(escapedValue[:index])
		match := escapedValue[index : index+len(escapedKeyword)]
		builder.WriteString(`<mark class="search-mark">` + match + `</mark>`)
		escapedValue = escapedValue[index+len(escapedKeyword):]
		lowerValue = lowerValue[index+len(escapedKeyword):]
	}
	return template.HTML(builder.String())
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func requestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func absoluteRequestURL(c *gin.Context, path string) string {
	return requestBaseURL(c) + path
}

func absoluteAssetURL(c *gin.Context, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return requestBaseURL(c) + "/" + strings.TrimPrefix(path, "/")
}

func absoluteAssetURLFromCanonical(canonical string, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	origin := requestOriginFromCanonical(canonical)
	return origin + "/" + strings.TrimPrefix(path, "/")
}

func requestOriginFromCanonical(canonical string) string {
	parts := strings.SplitN(canonical, "/", 4)
	if len(parts) >= 3 {
		return parts[0] + "//" + parts[2]
	}
	return strings.TrimRight(canonical, "/")
}

func SafeHTML(value string) template.HTML {
	return template.HTML(html.UnescapeString(value))
}
