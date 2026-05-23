package requestlog

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wjfcm-go/internal/config"

	"github.com/gin-gonic/gin"
)

func TestWriteRotatesAndFindsRequestLog(t *testing.T) {
	store := NewStore(config.LogConfig{
		RequestEnabled:      true,
		RequestPath:         t.TempDir(),
		RequestOutput:       "file",
		RequestLevel:        "info",
		RequestMaxBodyBytes: 1024,
		RequestMaxFileBytes: 220,
	})

	first := &Entry{
		RequestID: "req-one",
		Level:     "info",
		Method:    "GET",
		Path:      "/first",
		Request:   Payload{Body: strings.Repeat("a", 180)},
		Response:  Payload{Body: "ok"},
		Status:    200,
	}
	second := &Entry{
		RequestID: "req-two",
		Level:     "info",
		Method:    "GET",
		Path:      "/second",
		Request:   Payload{Body: strings.Repeat("b", 180)},
		Response:  Payload{Body: "ok"},
		Status:    200,
	}

	if err := store.write(first); err != nil {
		t.Fatalf("write first request log: %v", err)
	}
	if err := store.write(second); err != nil {
		t.Fatalf("write second request log: %v", err)
	}

	if got, _, err := store.Find(first.RequestID); err != nil || got.Path != first.Path {
		t.Fatalf("find first request log = (%+v, %v), want path %s", got, err, first.Path)
	}
	if got, _, err := store.Find(second.RequestID); err != nil || got.Path != second.Path {
		t.Fatalf("find second request log = (%+v, %v), want path %s", got, err, second.Path)
	}

	files := 0
	err := filepath.WalkDir(store.cfg.RequestPath, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d != nil && !d.IsDir() {
			files++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk request log dir: %v", err)
	}
	if files < 2 {
		t.Fatalf("rotated request log files = %d, want at least 2", files)
	}
}

func TestMiddlewareOnlyLogsAPIWithoutHeadersAndUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	store := NewStore(config.LogConfig{
		RequestEnabled:      true,
		RequestPath:         dir,
		RequestOutput:       "file",
		RequestLevel:        "info",
		RequestOnlyAPI:      true,
		RequestMaxBodyBytes: 1024,
		RequestMaxRespBytes: 1024,
		RequestMaxFileBytes: 1024 * 1024,
	})

	router := gin.New()
	router.Use(Middleware(store))
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "home")
	})
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "pong"})
	})

	homeReq := httptest.NewRequest(http.MethodGet, "/", nil)
	homeReq.Header.Set(HeaderName, "home-request")
	router.ServeHTTP(httptest.NewRecorder(), homeReq)
	if _, _, err := store.Find("home-request"); !os.IsNotExist(err) {
		t.Fatalf("home request log error = %v, want not exist", err)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	apiReq.Header.Set(HeaderName, "api-request")
	apiReq.Header.Set("User-Agent", "test-agent")
	apiReq.Header.Set("Authorization", "Bearer secret")
	router.ServeHTTP(httptest.NewRecorder(), apiReq)

	entry, path, err := store.Find("api-request")
	if err != nil {
		t.Fatalf("find api request log: %v", err)
	}
	if entry.Path != "/api/ping" || !strings.Contains(entry.Response.Body, "pong") {
		t.Fatalf("api request log = %+v, want api path and response body", entry)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read api request log: %v", err)
	}
	if strings.Contains(string(raw), "user_agent") || strings.Contains(string(raw), "headers") || strings.Contains(string(raw), "Authorization") {
		t.Fatalf("api request log contains removed fields: %s", raw)
	}
}
