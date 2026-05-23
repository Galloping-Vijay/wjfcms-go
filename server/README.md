# Gin 后端

## 开发启动

```bash
cd wjfcm-go/server
copy .env.example .env
go mod tidy
go run ./cmd/api
```

指定配置文件启动：

```bash
go run ./cmd/api -f .env.local
go run ./cmd/api -f /www/wwwroot/wjfcm-go/server/.env.prod
```

默认监听：

```text
http://localhost:8080
```

## 基础配置说明

```env
APP_ENV=local
APP_KEY=
APP_DEBUG=true
APP_CONSOLE_COLOR=true
LOG_CHANNEL=stack
LOG_PATH=storage/logs
LOG_MAX_SIZE_MB=50
```

这些配置的作用：

- `APP_ENV`：运行环境。未显式设置 `APP_DEBUG` 时，`production` 或 `prod` 会默认关闭 debug，并让 Gin 使用 release 模式。
- `APP_KEY`：应用密钥。安装向导会自动生成；当 `JWT_SECRET` 未配置或仍为 `change-me` 时，会作为 JWT 签名密钥兜底。
- `APP_DEBUG`：调试开关。控制 Gin 模式和部分调试输出，例如开发环境邮件验证码接口会返回 `debug_code`。
- `APP_CONSOLE_COLOR`：控制 Gin 控制台日志是否使用彩色输出。
- `LOG_CHANNEL`：服务日志输出位置，影响 Gin 请求日志、标准 `log`、GORM SQL 日志。
- `LOG_PATH`：服务运行日志目录，仅在 `LOG_CHANNEL=single/file/daily` 时使用。
- `LOG_MAX_SIZE_MB`：服务运行日志单文件最大大小，超过后自动分割。

`LOG_CHANNEL` 支持：

```text
stack/stdout/console  输出到控制台
stderr                输出到标准错误
single/file           写入 storage/logs/wjfcm-go.log
daily                 写入 storage/logs/wjfcm-go-YYYY-MM-DD.log
null/discard/none     丢弃日志
```

请求链路日志：

```env
REQUEST_LOG_ENABLED=false
REQUEST_LOG_TYPE=json
REQUEST_LOG_PATH=storage/request-logs
REQUEST_LOG_OUTPUT=file
REQUEST_LOG_LEVEL=info
REQUEST_LOG_ONLY_API=true
REQUEST_LOG_MAX_BODY_KB=256
REQUEST_LOG_MAX_RESPONSE_KB=64
REQUEST_LOG_MAX_FILE_MB=20
REQUEST_LOG_KEEP_DAYS=14
```

开启后每次请求都会返回 `X-Request-ID` 响应头，JSON API 响应体也会带 `request_id`。默认只记录 `/api/` 接口请求，不记录前台 HTML 页面；日志不再记录 `headers` 和 `user_agent`。请求参数、接口返回结果和该请求触发的 SQL 会写入 `REQUEST_LOG_PATH/YYYY-MM-DD/requests-YYYY-MM-DD.log`，超过 `REQUEST_LOG_MAX_FILE_MB` 后自动分割。需要查询某次请求时使用：

```text
GET /api/admin/request-logs/:request_id
```

按 `request_id` 查询日志要求 `REQUEST_LOG_OUTPUT=file` 或 `both`。

新站点初始化安装：

```text
GET /install
POST /install
```

安装向导会创建数据库表、写入初始管理员和基础配置，并生成 `.env` 与 `.install.lock`。安装成功后请重启服务。

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
GITHUB_REDIRECT=http://localhost:8080/api/auth/github/callback

QQ_CLIENT_ID=
QQ_CLIENT_SECRET=
QQ_REDIRECT=http://localhost:8080/api/auth/qq/callback

WEIBO_CLIENT_ID=
WEIBO_CLIENT_SECRET=
WEIBO_REDIRECT=http://localhost:8080/api/auth/weibo/callback
```

上线时把 `*_REDIRECT` 改成正式域名，并在对应平台后台登记完全一致的回调 URL。

## 说明

- 默认读取当前目录 `.env`，也会尝试读取上级目录 `.env`。传入 `-f` 后只读取指定配置文件。
- 数据库表前缀使用 `DB_PREFIX`，用于兼容现有 `wjf_` 数据表。
- 管理员登录接口兼容 bcrypt 密码。
- 角色权限绑定复用旧 Spatie 权限表：`roles`、`permissions`、`role_has_permissions`。
