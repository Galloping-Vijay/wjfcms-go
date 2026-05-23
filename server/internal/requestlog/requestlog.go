package requestlog

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"wjfcm-go/internal/config"

	"github.com/gin-gonic/gin"
)

const HeaderName = "X-Request-ID"

type SQLRecord struct {
	Type     string  `json:"type"`
	Source   string  `json:"source"`
	Elapsed  float64 `json:"elapsed_ms"`
	Rows     string  `json:"rows"`
	SQL      string  `json:"sql"`
	Error    string  `json:"error,omitempty"`
	CreateAt string  `json:"create_time"`
}

type Entry struct {
	RequestID string      `json:"request_id"`
	Level     string      `json:"level"`
	Method    string      `json:"method"`
	Path      string      `json:"path"`
	RawQuery  string      `json:"raw_query,omitempty"`
	ClientIP  string      `json:"client_ip"`
	Request   Payload     `json:"request"`
	Response  Payload     `json:"response"`
	SQL       []SQLRecord `json:"sql"`
	Status    int         `json:"status"`
	Latency   float64     `json:"latency_ms"`
	StartTime string      `json:"start_time"`
	EndTime   string      `json:"end_time"`
}

type Payload struct {
	Query     map[string][]string `json:"query,omitempty"`
	Form      map[string][]string `json:"form,omitempty"`
	Body      string              `json:"body,omitempty"`
	Truncated bool                `json:"truncated,omitempty"`
}

type Store struct {
	cfg     config.LogConfig
	enabled bool
	mu      sync.RWMutex
	writeMu sync.Mutex
	active  map[uint64]*Entry
}

var defaultStore = NewStore(config.LogConfig{})

func NewStore(cfg config.LogConfig) *Store {
	maxBody := cfg.RequestMaxBodyBytes
	if maxBody <= 0 {
		maxBody = 256 * 1024
	}
	maxResp := cfg.RequestMaxRespBytes
	if maxResp <= 0 {
		maxResp = 64 * 1024
	}
	return &Store{
		cfg: config.LogConfig{
			RequestEnabled:      cfg.RequestEnabled,
			RequestType:         firstNonEmpty(cfg.RequestType, "json"),
			RequestPath:         firstNonEmpty(cfg.RequestPath, "storage/request-logs"),
			RequestOutput:       firstNonEmpty(cfg.RequestOutput, "file"),
			RequestLevel:        firstNonEmpty(cfg.RequestLevel, "info"),
			RequestOnlyAPI:      cfg.RequestOnlyAPI,
			RequestMaxBodyBytes: maxBody,
			RequestMaxRespBytes: maxResp,
			RequestMaxFileBytes: cfg.RequestMaxFileBytes,
			RequestKeepDays:     cfg.RequestKeepDays,
		},
		enabled: cfg.RequestEnabled,
		active:  map[uint64]*Entry{},
	}
}

func SetDefault(store *Store) {
	if store != nil {
		defaultStore = store
	}
}

func Middleware(store *Store) gin.HandlerFunc {
	if store == nil {
		store = defaultStore
	}
	return store.Middleware()
}

func (s *Store) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(HeaderName))
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set(HeaderName, requestID)

		if !s.enabled || !s.shouldLogPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		requestPayload := s.captureRequest(c)
		writer := &bodyWriter{ResponseWriter: c.Writer, max: s.cfg.RequestMaxRespBytes}
		c.Writer = writer

		entry := &Entry{
			RequestID: requestID,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			RawQuery:  c.Request.URL.RawQuery,
			ClientIP:  c.ClientIP(),
			Request:   requestPayload,
			StartTime: start.Format(time.RFC3339Nano),
		}
		gid := currentGID()
		s.start(gid, entry)
		defer s.finish(gid)

		c.Next()

		end := time.Now()
		entry.Status = c.Writer.Status()
		entry.Latency = float64(end.Sub(start).Microseconds()) / 1000
		entry.EndTime = end.Format(time.RFC3339Nano)
		entry.Level = levelForStatus(entry.Status)
		entry.Response = Payload{Body: writer.body.String(), Truncated: writer.truncated}
		if s.shouldWrite(entry.Level) {
			if err := s.write(entry); err != nil {
				log.Printf("[request-log] write failed: %v", err)
			}
		}
	}
}

func AddSQL(record SQLRecord) {
	defaultStore.AddSQL(record)
}

func Find(requestID string) (Entry, string, error) {
	return defaultStore.Find(requestID)
}

func (s *Store) AddSQL(record SQLRecord) {
	if s == nil || !s.enabled {
		return
	}
	gid := currentGID()
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry := s.active[gid]; entry != nil {
		entry.SQL = append(entry.SQL, record)
	}
}

func (s *Store) shouldLogPath(path string) bool {
	return !s.cfg.RequestOnlyAPI || strings.HasPrefix(path, "/api/")
}

func (s *Store) Find(requestID string) (Entry, string, error) {
	requestID = safeRequestID(requestID)
	if requestID == "" {
		return Entry{}, "", os.ErrNotExist
	}
	root := firstNonEmpty(s.cfg.RequestPath, "storage/request-logs")
	var found Entry
	var foundPath string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) != requestID+".json" {
			if !isRequestLogFile(path) {
				return nil
			}
			entry, ok, findErr := findEntryInJSONLines(path, requestID)
			if findErr != nil {
				return findErr
			}
			if !ok {
				return nil
			}
			found = entry
			foundPath = path
			return filepath.SkipAll
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := json.Unmarshal(body, &found); err != nil {
			return err
		}
		foundPath = path
		return filepath.SkipAll
	})
	if err != nil {
		return Entry{}, "", err
	}
	if foundPath == "" {
		return Entry{}, "", os.ErrNotExist
	}
	return found, foundPath, nil
}

func (s *Store) start(gid uint64, entry *Entry) {
	if gid == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[gid] = entry
}

func (s *Store) finish(gid uint64) {
	if gid == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, gid)
}

func (s *Store) captureRequest(c *gin.Context) Payload {
	payload := Payload{Query: c.Request.URL.Query()}
	if c.Request.Body == nil {
		return payload
	}
	contentType := c.ContentType()
	if strings.Contains(contentType, "multipart/form-data") {
		payload.Body = "[multipart form omitted]"
		return payload
	}
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	logBody := body
	if len(logBody) > s.cfg.RequestMaxBodyBytes {
		payload.Truncated = true
		logBody = logBody[:s.cfg.RequestMaxBodyBytes]
	}
	payload.Body = string(logBody)
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		_ = c.Request.ParseForm()
		payload.Form = c.Request.PostForm
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
	}
	return payload
}

func (s *Store) shouldWrite(level string) bool {
	want := levelRank(s.cfg.RequestLevel)
	got := levelRank(level)
	return got >= want
}

func (s *Store) write(entry *Entry) error {
	output := strings.ToLower(strings.TrimSpace(s.cfg.RequestOutput))
	if output == "" {
		output = "file"
	}
	body, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if output == "console" || output == "stdout" || output == "both" {
		pretty, _ := json.MarshalIndent(entry, "", "  ")
		log.Printf("[request-log] %s", pretty)
	}
	if output == "file" || output == "both" {
		path := filepath.Join(s.cfg.RequestPath, time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		file := filepath.Join(path, "requests-"+time.Now().Format("2006-01-02")+".log")
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return appendRotating(file, append(body, '\n'), s.cfg.RequestMaxFileBytes)
	}
	return nil
}

func isRequestLogFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".log" || ext == ".jsonl"
}

func findEntryInJSONLines(path string, requestID string) (Entry, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return Entry{}, false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			var entry Entry
			if err := json.Unmarshal(line, &entry); err == nil && entry.RequestID == requestID {
				return entry, true, nil
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return Entry{}, false, readErr
		}
	}
	return Entry{}, false, nil
}

func appendRotating(path string, data []byte, maxSize int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if maxSize > 0 {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 && info.Size()+int64(len(data)) > maxSize {
			if err := os.Rename(path, nextRotatedPath(path)); err != nil {
				return err
			}
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func nextRotatedPath(path string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	stamp := time.Now().Format("20060102150405")
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%s.%03d%s", base, stamp, i, ext)
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	max       int
	truncated bool
}

func (w *bodyWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *bodyWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *bodyWriter) capture(data []byte) {
	if w.max <= 0 || w.body.Len() >= w.max {
		w.truncated = true
		return
	}
	remain := w.max - w.body.Len()
	if len(data) > remain {
		w.body.Write(data[:remain])
		w.truncated = true
		return
	}
	w.body.Write(data)
}

func currentGID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 {
		return 0
	}
	id, _ := strconv.ParseUint(fields[1], 10, 64)
	return id
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return time.Now().Format("20060102150405") + "-" + hex.EncodeToString(bytes[:])
}

func safeRequestID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, value)
	return value
}

func levelForStatus(status int) string {
	switch {
	case status >= 500:
		return "error"
	case status >= 400:
		return "warn"
	default:
		return "info"
	}
}

func levelRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return 0
	case "info", "":
		return 1
	case "warn", "warning":
		return 2
	case "error":
		return 3
	default:
		return 1
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
