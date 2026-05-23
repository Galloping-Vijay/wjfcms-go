# wjfcm-go

wjfcm-go 是一个基于 Gin + Vue 的 CMS 项目。前台公开页面由 Gin 服务端渲染，后台管理端由 Vue 提供。

## 目录规划

```text
wjfcm-go/
  server/       Gin 后端服务
  web/          Vue 前端项目
  public/       公开静态资源、上传文件、根目录验证文件
  docs/         项目文档、接口说明、进度记录
  deploy/       部署配置，如 Nginx、systemd、Docker 等
```

## 当前能力

- `server/` 已完成 Gin 基础骨架、数据库连接、JWT 登录、文章/分类/标签基础接口。
- 新安装环境支持访问 `/install` 初始化：填写数据库、站点信息和超级管理员后自动建表、写入初始数据并生成 `.env`。
- 前台公开页面已改为 Gin 服务端渲染：首页、分类、标签、搜索、归档、有些话、文章详情、`robots.txt`、`sitemap.xml` 会直接输出完整 HTML。
- `web/` 现在只保留 Vue 后台，不再承载前台页面。
- `public/` 放站点公开资源和域名根目录验证文件，例如 `favicon.ico`、`ads.txt`、`bdunion.txt`、`google*.html`。
- 详细记录见 [docs/progress.md](docs/progress.md)。

## 启动方式

### 前台 SEO 页面 + Gin API

前台公开页面和 API 都由 Gin 服务提供。开发时启动这个服务后，直接访问 `http://localhost:8080/` 查看前台 SEO 页面。

```bash
cd wjfcm-go/server
copy .env.example .env
go mod tidy
go run ./cmd/api
```

也可以指定配置文件启动：

```bash
go run ./cmd/api -f .env.local
```

常用地址：

```text
前台首页: http://localhost:8080/
文章详情: http://localhost:8080/article/1
分类页面: http://localhost:8080/category/1
标签页面: http://localhost:8080/tag/1
文章归档: http://localhost:8080/archive
有些话: http://localhost:8080/chat
Sitemap: http://localhost:8080/sitemap.xml
Robots: http://localhost:8080/robots.txt
API 健康检查: http://localhost:8080/api/health
根验证文件: http://localhost:8080/ads.txt
安装向导: http://localhost:8080/install
```

### Vue 后台

后台管理由 Vue + Vite 提供。开发时单独启动 Vue，只访问后台页面。

```bash
cd wjfcm-go/web
copy .env.example .env
npm install
npm run dev
```

开发地址：

```text
后台登录: http://localhost:5173/admin/login
```

`web/.env` 需要指向 Gin API：

```env
VITE_API_BASE_URL=http://localhost:8080/api
```

## 部署方式

下面以 Linux 服务器、Nginx、systemd、MySQL 为例。前端和后端可以部署在同一台机器，也可以分开部署。

如果不想把源码上传到服务器，只部署二进制、Vue 构建产物、模板、静态资源和生产配置，请看 [无源码部署说明](docs/no-source-deploy.md)。

### 1. 后端部署

如果是新站点，可以只准备 `.env.example` 或一个最小 `.env`，启动后访问 `/install` 走安装向导。安装向导会连接数据库、创建数据表、写入基础配置和超级管理员，并生成正式 `.env`。

如果是已有站点或手工部署，先准备生产环境配置：

```bash
cd wjfcm-go/server
cp .env.example .env
```

重点修改 `.env`：

```env
APP_ENV=production
APP_DEBUG=false
APP_PORT=8080
APP_URL=https://api.example.com
JWT_SECRET=请换成足够长的随机字符串
JWT_EXPIRES_MINUTES=120
JWT_REFRESH_EXPIRES_MINUTES=10080

LOG_CHANNEL=daily
LOG_PATH=storage/logs
LOG_MAX_SIZE_MB=50
REQUEST_LOG_ENABLED=false
REQUEST_LOG_PATH=storage/request-logs
REQUEST_LOG_OUTPUT=file
REQUEST_LOG_LEVEL=info
REQUEST_LOG_ONLY_API=true
REQUEST_LOG_MAX_BODY_KB=256
REQUEST_LOG_MAX_RESPONSE_KB=64
REQUEST_LOG_MAX_FILE_MB=20

DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=你的数据库
DB_USERNAME=你的账号
DB_PASSWORD=你的密码
DB_PREFIX=wjf_

CORS_ALLOW_ORIGINS=https://www.example.com,https://example.com
PUBLIC_DIR=/www/wwwroot/wjfcm-go/public
UPLOAD_BASE_PATH=uploads

GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT=https://www.example.com/api/home/auth/github/callback
QQ_CLIENT_ID=
QQ_CLIENT_SECRET=
QQ_REDIRECT=https://www.example.com/api/home/auth/qq/callback
WEIBO_CLIENT_ID=
WEIBO_CLIENT_SECRET=
WEIBO_REDIRECT=https://www.example.com/api/home/auth/weibo/callback
```

编译 Gin 服务：

```bash
cd wjfcm-go/server
go mod tidy
go build -o wjfcm-go-api ./cmd/api
```

推荐用 systemd 托管后端进程：

```ini
[Unit]
Description=wjfcm-go API
After=network.target

[Service]
Type=simple
WorkingDirectory=/www/wwwroot/wjfcm-go/server
ExecStart=/www/wwwroot/wjfcm-go/server/wjfcm-go-api -f /www/wwwroot/wjfcm-go/server/.env
Restart=always
RestartSec=3
User=www
Group=www
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

保存为 `/etc/systemd/system/wjfcm-go-api.service` 后启动：

```bash
systemctl daemon-reload
systemctl enable wjfcm-go-api
systemctl start wjfcm-go-api
systemctl status wjfcm-go-api
```

### 2. Vue 后台部署

Vue 生产环境现在只服务后台，不再负责任何前台页面。

生产环境需要把接口地址指向线上 Gin API：

```bash
cd wjfcm-go/web
cp .env.example .env.production
```

`.env.production` 示例：

```env
VITE_API_BASE_URL=https://api.example.com/api
```

构建 Vue 静态文件：

```bash
cd wjfcm-go/web
npm install
npm run build
```

构建产物在：

```text
wjfcm-go/web/dist/
```

当前推荐部署方式是“Gin 承接前台 SEO 页面 + Vue 承接后台页面”：

- 首页 `/`、文章 `/article/:id`、分类 `/category/:id`、标签 `/tag/:id`、搜索 `/search`、归档 `/archive`、有些话 `/chat` 走 Gin 服务端渲染。
- `/api/` 走 Gin API。
- `/admin` 交给 Vue `dist/`。
- `/login`、`/register`、`/forgot-password`、`/user` 也走 Gin HTML+JS。

也就是说，生产环境不要再把全部路径都直接交给 Vue `index.html`，否则 SEO 页面又会退回 SPA 空壳。

### 3. Nginx 示例

推荐单域名部署，这样前台 SEO 页、后台、API 都在同一个域名下，搜索引擎看到的 canonical 也最稳定：

```nginx
server {
    listen 80;
    server_name www.example.com;
    root /www/wwwroot/wjfcm-go/web/dist;
    index index.html;

    location ~ ^/(api|article|category|tag|search|archive|chat|login|register|forgot-password|user|install|blank|robots\.txt|sitemap\.xml|tools|wechat|baidu)(/|$) {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location ^~ /uploads/ {
        alias /www/wwwroot/wjfcm-go/public/uploads/;
        expires 30d;
        access_log off;
    }

    location ^~ /images/ {
        alias /www/wwwroot/wjfcm-go/public/images/;
        expires 30d;
        access_log off;
    }

    location ~ ^/(favicon\.ico|ads\.txt|bdunion\.txt|google.*\.html)$ {
        root /www/wwwroot/wjfcm-go/public;
        access_log off;
    }

    location /admin/ {
        try_files $uri $uri/ /index.html;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

如果必须拆成两个域名，可以让 `www.example.com` 仍然反代公开 SEO 页面到 Gin，让 `api.example.com` 只承接 `/api/`。不要让 `www.example.com/article/1` 直接落到 Vue 空壳。

```nginx
server {
    listen 80;
    server_name api.example.com;

    location /api/ {
        proxy_pass http://127.0.0.1:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

`www.example.com` 仍按上面的单域名配置处理，只是前端 `.env.production` 改成：

```env
VITE_API_BASE_URL=https://api.example.com/api
```

单域名时，前端 `.env.production` 写：

```env
VITE_API_BASE_URL=/api
```

### 4. 路由归属

| 路径 | 服务 | 用途 |
| --- | --- | --- |
| `/` | Gin | 前台首页 SEO HTML |
| `/article/:id` | Gin | 文章详情 SEO HTML |
| `/category/:id` | Gin | 分类页 SEO HTML |
| `/tag/:id` | Gin | 标签页 SEO HTML |
| `/search` | Gin | 搜索页 SEO HTML |
| `/archive` | Gin | 归档页 SEO HTML |
| `/chat` | Gin | 有些话 SEO HTML |
| `/robots.txt` | Gin | 搜索引擎规则 |
| `/sitemap.xml` | Gin | 站点地图 |
| `/install` | Gin | 新站点初始化安装 |
| `/api/*` | Gin | API |
| `/admin/*` | Vue | 后台管理 |
| `/login`、`/register`、`/forgot-password`、`/user` | Gin | 用户交互页 HTML+JS |

### 5. 部署检查

部署后建议检查：

- `systemctl status wjfcm-go-api` 后端是否运行。
- `curl http://127.0.0.1:8080/api/home/articles` 后端接口是否正常。
- `curl https://www.example.com/article/1` 是否能直接看到文章标题、正文、`description`、`canonical`、JSON-LD，而不是只有 Vue 空壳。
- `curl https://www.example.com/sitemap.xml` 和 `curl https://www.example.com/robots.txt` 是否正常。
- Nginx 站点是否能打开 Vue 后台：`https://www.example.com/admin/login`。
- 文章图片、LOGO、头像等 `/uploads/` 静态资源是否能访问。
- 后台登录、文章新增编辑、图片上传、评论、友链申请是否正常。

## 开发原则

- 数据库优先兼容现有 `wjf_` 表。
- 前台 SEO 页面由 Gin 输出完整 HTML。
- Vue 只负责后台管理端。
- 前后端配置、部署脚本和文档统一使用 `wjfcm-go` 命名。
