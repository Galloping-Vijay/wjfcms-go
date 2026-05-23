# Gin 后端

## 开发启动

```bash
cd wjfcm-go/server
copy .env.example .env
go mod tidy
go run ./cmd/api
```

默认监听：

```text
http://localhost:8080
```

前台 SEO 页面：

```text
GET /
GET /article/:id
GET /category/:id
GET /tag/:id
GET /search
GET /archive
GET /chat
GET /robots.txt
GET /sitemap.xml
```

健康检查：

```text
GET /api/health
```

图片上传：

```text
POST /api/admin/upload/image
```

支持字段：

- `file`
- `base64_img`
- `editormd-image-file`

文件默认保存到：

```text
../public/uploads/YYYYMMDD
```

SQL 日志支持全量、慢查询、异常查询分别控制：

```text
DB_LOG_SQL=false
DB_LOG_SLOW_SQL=true
DB_LOG_ERROR_SQL=true
DB_LOG_LEVEL=info
DB_SLOW_THRESHOLD_MS=200
```

其中 `DB_LOG_SQL` 控制每条 SQL 是否记录，`DB_LOG_SLOW_SQL` 控制超过阈值的慢查询是否记录，`DB_LOG_ERROR_SQL` 控制异常 SQL 是否记录。
日志会输出 SQL、耗时、影响行数，以及触发查询的业务文件和行号，例如：

```text
[gorm] [sql] [internal/handler/article.go:42] [3.21ms] [rows:10] SELECT * FROM `wjf_articles`
[gorm] [slow] [internal/handler/article.go:42] [250.34ms] [rows:10] SELECT * FROM `wjf_articles`
[gorm] [error] [internal/handler/article.go:42] [1.56ms] [rows:-] SELECT * FROM `wjf_articles` | sql: database is closed
```

后台 JWT 支持 Access Token 和 Refresh Token：

```text
JWT_SECRET=change-me
JWT_EXPIRES_MINUTES=120
JWT_REFRESH_EXPIRES_MINUTES=10080
```

Access Token 过期后，前端会调用 `POST /api/admin/auth/refresh` 自动刷新并重试原请求。

前台第三方登录支持 GitHub、QQ、微博 OAuth，开发环境回调地址示例：

```text
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT=http://localhost:8080/api/home/auth/github/callback

QQ_CLIENT_ID=
QQ_CLIENT_SECRET=
QQ_REDIRECT=http://localhost:8080/api/home/auth/qq/callback

WEIBO_CLIENT_ID=
WEIBO_CLIENT_SECRET=
WEIBO_REDIRECT=http://localhost:8080/api/home/auth/weibo/callback
```

上线时把 `*_REDIRECT` 改成正式域名，并在对应平台后台登记完全一致的回调 URL。

## 说明

- 默认读取当前目录 `.env`，也会尝试读取仓库根目录 `.env`。
- 数据库表前缀使用 `DB_PREFIX`，用于兼容现有 `wjf_` 数据表。
- 管理员登录接口兼容 bcrypt 密码。
- 角色权限绑定复用旧 Spatie 权限表：`roles`、`permissions`、`role_has_permissions`。
