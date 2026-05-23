package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/response"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	cfg config.Config
}

func NewUploadHandler(cfg config.Config) *UploadHandler {
	return &UploadHandler{cfg: cfg}
}

func (h *UploadHandler) Image(c *gin.Context) {
	if base64Image := c.PostForm("base64_img"); base64Image != "" {
		h.saveBase64Image(c, base64Image)
		return
	}

	field := "file"
	file, err := c.FormFile(field)
	if err != nil {
		field = "editormd-image-file"
		file, err = c.FormFile(field)
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, 1, "没有要上传的文件")
		return
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if !allowedImageExt(ext) {
		response.Error(c, http.StatusBadRequest, 1, "图片类型不被允许")
		return
	}

	relativeDir, absoluteDir, err := h.uploadDirs()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	name := fmt.Sprintf("%d.%s", time.Now().UnixNano(), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(absoluteDir, name)); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, "保存文件失败")
		return
	}

	url := h.publicURL(relativeDir, name)
	if field == "editormd-image-file" {
		c.JSON(http.StatusOK, gin.H{"success": 1, "message": "上传成功", "url": url})
		return
	}
	response.OK(c, "上传成功", gin.H{"src": url, "title": file.Filename})
}

func (h *UploadHandler) saveBase64Image(c *gin.Context, input string) {
	re := regexp.MustCompile(`^data:\s*image/(\w+);base64,`)
	match := re.FindStringSubmatch(input)
	if len(match) < 2 {
		response.Error(c, http.StatusBadRequest, 1, "不是base64格式")
		return
	}

	ext := strings.ToLower(match[1])
	if ext == "jpeg" {
		ext = "jpg"
	}
	if !allowedImageExt(ext) {
		response.Error(c, http.StatusBadRequest, 1, "图片类型不被允许")
		return
	}

	content, err := base64.StdEncoding.DecodeString(re.ReplaceAllString(input, ""))
	if err != nil {
		response.Error(c, http.StatusBadRequest, 1, "base64解析失败")
		return
	}

	relativeDir, absoluteDir, err := h.uploadDirs()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 1, err.Error())
		return
	}

	name := fmt.Sprintf("%d.%s", time.Now().UnixNano(), ext)
	if err := os.WriteFile(filepath.Join(absoluteDir, name), content, 0644); err != nil {
		response.Error(c, http.StatusInternalServerError, 1, "保存文件失败")
		return
	}

	response.OK(c, "上传成功", gin.H{"src": h.publicURL(relativeDir, name), "title": "文章图片"})
}

func (h *UploadHandler) uploadDirs() (string, string, error) {
	day := time.Now().Format("20060102")
	relativeDir := strings.Trim(h.cfg.Upload.BasePath, "/") + "/" + day
	absoluteDir := filepath.Join(h.cfg.Upload.PublicDir, filepath.FromSlash(relativeDir))
	if err := os.MkdirAll(absoluteDir, 0755); err != nil {
		return "", "", err
	}
	return relativeDir, absoluteDir, nil
}

func (h *UploadHandler) publicURL(relativeDir string, name string) string {
	return strings.TrimRight(h.cfg.App.URL, "/") + "/" + strings.Trim(relativeDir, "/") + "/" + name
}

func allowedImageExt(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "gif", "png", "bmp", "webp":
		return true
	default:
		return false
	}
}
