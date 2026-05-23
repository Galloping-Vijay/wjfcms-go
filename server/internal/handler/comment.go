package handler

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"wjfcm-go/internal/model"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CommentHandler struct {
	db *gorm.DB
}

type commentListItem struct {
	model.Comment
	Username        string `json:"username"`
	Title           string `json:"title"`
	ReplyToUsername string `json:"reply_to_username"`
	ReplyToContent  string `json:"reply_to_content"`
	OriginUsername  string `json:"origin_username"`
	OriginContent   string `json:"origin_content"`
}

type publicCommentItem struct {
	ID              uint                `json:"id"`
	Pid             uint                `json:"pid"`
	OriginID        uint                `json:"origin_id"`
	Content         string              `json:"content"`
	CreatedAt       any                 `json:"created_at"`
	UserID          uint                `json:"user_id"`
	UserName        string              `json:"user_name"`
	Avatar          string              `json:"avatar"`
	ReplyToUsername string              `json:"reply_to_username"`
	ReplyToContent  string              `json:"reply_to_content"`
	Zan             uint                `json:"zan"`
	Cai             uint                `json:"cai"`
	Child           []publicCommentItem `json:"child"`
}

func NewCommentHandler(db *gorm.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

func (h *CommentHandler) Index(c *gin.Context) {
	var comments []commentListItem
	commentTable := h.db.NamingStrategy.TableName("comments")
	userTable := h.db.NamingStrategy.TableName("users")
	articleTable := h.db.NamingStrategy.TableName("articles")
	parentCommentAlias := "parent_comments"
	parentUserAlias := "parent_users"
	originCommentAlias := "origin_comments"
	originUserAlias := "origin_users"
	query := h.db.Model(&model.Comment{}).
		Select(fmt.Sprintf(
			"%s.*, %s.name AS username, %s.title AS title, %s.name AS reply_to_username, %s.content AS reply_to_content, %s.name AS origin_username, %s.content AS origin_content",
			commentTable,
			userTable,
			articleTable,
			parentUserAlias,
			parentCommentAlias,
			originUserAlias,
			originCommentAlias,
		)).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.user_id", userTable, userTable, commentTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.article_id", articleTable, articleTable, commentTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s AS %s ON %s.id = %s.pid", commentTable, parentCommentAlias, parentCommentAlias, commentTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s AS %s ON %s.id = %s.user_id", userTable, parentUserAlias, parentUserAlias, parentCommentAlias)).
		Joins(fmt.Sprintf("LEFT JOIN %s AS %s ON %s.id = %s.origin_id", commentTable, originCommentAlias, originCommentAlias, commentTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s AS %s ON %s.id = %s.user_id", userTable, originUserAlias, originUserAlias, originCommentAlias))

	if username := c.Query("username"); username != "" {
		query = query.Where(userTable+".name LIKE ?", "%"+username+"%")
	}
	if title := c.Query("title"); title != "" {
		query = query.Where(articleTable+".title LIKE ?", "%"+title+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where(commentTable+".status = ?", status)
	}
	if c.Query("delete") == "1" {
		query = query.Unscoped().Where(commentTable + ".deleted_at IS NOT NULL")
	} else if c.Query("delete") == "2" {
		query = query.Unscoped()
	}

	page, pageSize := pageParams(c)
	var total int64
	query.Count(&total)

	if err := query.Order(commentTable + ".id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&comments).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	for i := range comments {
		comments[i].Content = html.UnescapeString(comments[i].Content)
		comments[i].ReplyToContent = html.UnescapeString(comments[i].ReplyToContent)
		comments[i].OriginContent = html.UnescapeString(comments[i].OriginContent)
	}
	response.Page(c, "获取成功", comments, total)
}

func (h *CommentHandler) PublicIndex(c *gin.Context) {
	articleID := c.Query("article_id")
	if articleID == "" {
		response.Error(c, http.StatusBadRequest, 1, "缺少文章ID")
		return
	}

	var article model.Article
	if err := h.db.Where("status = ?", 1).First(&article, articleID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}

	page, pageSize := pageParams(c)
	var roots []model.Comment
	query := h.db.Preload("User").
		Where("article_id = ? AND status = 1 AND type = 1 AND pid = 0", articleID)
	var total int64
	query.Model(&model.Comment{}).Count(&total)
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&roots).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	items := make([]publicCommentItem, 0, len(roots))
	for _, root := range roots {
		item := h.toPublicComment(root)
		item.Child = h.commentChildren(root.ID)
		items = append(items, item)
	}
	response.Page(c, "获取成功", items, total)
}

func (h *CommentHandler) PublicStore(c *gin.Context) {
	var req struct {
		ArticleID uint   `json:"article_id" binding:"required"`
		Pid       uint   `json:"pid"`
		OriginID  uint   `json:"origin_id"`
		Content   string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写评论内容")
		return
	}
	userID := uint(c.GetUint64("user_id"))
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, 401, "请先登录再评论")
		return
	}

	var article model.Article
	if err := h.db.Where("status = ?", 1).First(&article, req.ArticleID).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "文章不存在")
		return
	}

	comment := model.Comment{
		UserID:    userID,
		Type:      1,
		Pid:       req.Pid,
		OriginID:  req.OriginID,
		ArticleID: req.ArticleID,
		Content:   html.EscapeString(req.Content),
		Status:    1,
	}
	if err := h.db.Create(&comment).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	response.OK(c, "评论成功", comment)
}

func (h *CommentHandler) PublicAction(c *gin.Context) {
	var req struct {
		ActionType string `json:"action_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请选择操作类型")
		return
	}
	field := ""
	switch req.ActionType {
	case "zan":
		field = "zan"
	case "cai":
		field = "cai"
	default:
		response.Error(c, http.StatusBadRequest, 1, "操作类型错误")
		return
	}

	var comment model.Comment
	if err := h.db.Where("status = ?", 1).First(&comment, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "评论不存在")
		return
	}
	if err := h.db.Model(&comment).UpdateColumn(field, gorm.Expr(field+" + ?", 1)).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	if err := h.db.First(&comment, comment.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", gin.H{
		"id":  comment.ID,
		"zan": comment.Zan,
		"cai": comment.Cai,
	})
}

func (h *CommentHandler) commentChildren(originID uint) []publicCommentItem {
	var comments []model.Comment
	if err := h.db.Preload("User").
		Where("status = 1 AND type = 1 AND origin_id = ?", originID).
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		return nil
	}

	items := make([]publicCommentItem, 0, len(comments))
	byID := map[uint]model.Comment{}
	userByID := map[uint]string{}
	var origin model.Comment
	if err := h.db.Preload("User").First(&origin, originID).Error; err == nil {
		byID[origin.ID] = origin
		if origin.User != nil {
			userByID[origin.ID] = origin.User.Name
		}
	}
	for _, comment := range comments {
		byID[comment.ID] = comment
		if comment.User != nil {
			userByID[comment.ID] = comment.User.Name
		}
	}
	for _, comment := range comments {
		item := h.toPublicComment(comment)
		if parent, ok := byID[comment.Pid]; ok {
			item.ReplyToContent = html.UnescapeString(parent.Content)
			item.ReplyToUsername = userByID[parent.ID]
		}
		items = append(items, item)
	}
	return items
}

func (h *CommentHandler) toPublicComment(comment model.Comment) publicCommentItem {
	userName := "游客"
	avatar := "/images/config/avatar.jpg"
	if comment.User != nil {
		if comment.User.Name != "" {
			userName = comment.User.Name
		}
		if comment.User.Avatar != "" {
			avatar = comment.User.Avatar
		}
	}
	return publicCommentItem{
		ID:        comment.ID,
		Pid:       comment.Pid,
		OriginID:  comment.OriginID,
		Content:   html.UnescapeString(comment.Content),
		CreatedAt: comment.CreatedAt,
		UserID:    comment.UserID,
		UserName:  userName,
		Avatar:    avatar,
		Zan:       comment.Zan,
		Cai:       comment.Cai,
		Child:     []publicCommentItem{},
	}
}

func (h *CommentHandler) Update(c *gin.Context) {
	var comment model.Comment
	if err := h.db.First(&comment, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, 1, "评论不存在")
		return
	}
	var req struct {
		Content string `json:"content"`
		Status  int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 1, "请填写评论信息")
		return
	}
	if err := h.db.Model(&comment).Updates(map[string]any{
		"content": req.Content,
		"status":  req.Status,
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", comment)
}

func (h *CommentHandler) Replace(c *gin.Context) {
	var req struct {
		Search  string `json:"search"`
		Replace string `json:"replace"`
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

	escapedSearch := html.EscapeString(search)
	query := h.db.Model(&model.Comment{}).Unscoped().
		Where("content LIKE ?", "%"+search+"%")
	if escapedSearch != search {
		query = query.Or("content LIKE ?", "%"+escapedSearch+"%")
	}

	var comments []model.Comment
	if err := query.Select("id", "content").Find(&comments).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	changed := 0
	for _, comment := range comments {
		raw := comment.Content
		next := strings.ReplaceAll(raw, search, req.Replace)
		if escapedSearch != search {
			next = strings.ReplaceAll(next, escapedSearch, html.EscapeString(req.Replace))
		}
		if next == raw {
			visible := html.UnescapeString(raw)
			replaced := strings.ReplaceAll(visible, search, req.Replace)
			if replaced != visible {
				next = html.EscapeString(replaced)
			}
		}
		if next == raw {
			continue
		}
		if err := h.db.Unscoped().Model(&model.Comment{}).Where("id = ?", comment.ID).Update("content", next).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, 1, err.Error())
			return
		}
		changed++
	}

	response.OK(c, "操作成功", gin.H{"matched": len(comments), "changed": changed})
}

func (h *CommentHandler) Destroy(c *gin.Context) {
	if err := h.db.Delete(&model.Comment{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *CommentHandler) Restore(c *gin.Context) {
	if err := h.db.Unscoped().Model(&model.Comment{}).Where("id = ?", c.Param("id")).Update("deleted_at", nil).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}

func (h *CommentHandler) ForceDelete(c *gin.Context) {
	if err := h.db.Unscoped().Delete(&model.Comment{}, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}
	response.OK(c, "操作成功", nil)
}
