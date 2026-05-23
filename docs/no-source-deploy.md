# 无源码部署说明

本文说明如何把 `wjfcm-go` 部署到服务器，但不把 Go / Vue 源码上传到服务器。服务器只保留运行必需的产物、模板、静态资源和配置。

## 部署产物包含什么

当前项目的 Gin 服务运行时会读取 `server/templates/*.tmpl`，Vue 后台由 Nginx 读取 `web/dist`。因此“无源码部署包”建议包含：

```text
wjfcm-go-release/
  server/
    wjfcm-go-api        # Go 编译后的二进制文件
    .env                # 生产配置，服务器上单独维护，不提交 Git；新站点也可以通过 /install 生成
    .env.example        # 可选，安装前作为配置参考
    .install.lock       # 安装成功后生成，用于防止重复安装
    templates/          # Gin 前台 SEO 模板，运行时需要
  web/
    dist/               # Vue 后台构建产物
  public/
    favicon.ico
    ads.txt
    bdunion.txt
    google*.html
    images/
    uploads/            # 上传目录，生产环境需要可写
```

不需要上传：

```text
server/cmd/
server/internal/
server/go.mod
server/go.sum
web/src/
web/package.json
web/node_modules/
docs/
.git/
```

## 本地或 CI 打包

推荐在本地或 CI 机器完成编译和构建，然后把产物上传到服务器。

### 1. 编译 Go 后端

如果服务器是 Linux amd64：

```bash
cd server
go mod tidy
# go build -o wjfcm-go-api ./cmd/api
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../release/server/wjfcm-go-api ./cmd/api
```

如果在 Windows PowerShell 上交叉编译 Linux amd64：

```powershell
cd server
go mod tidy
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -trimpath -ldflags="-s -w" -o ..\release\server\wjfcm-go-api ./cmd/api
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
```

如果服务器就是 Windows，则去掉 `GOOS/GOARCH`，输出 `wjfcm-go-api.exe`。

### 2. 构建 Vue 后台

单域名部署建议：

```env
VITE_API_BASE_URL=/api
```

独立 API 域名部署示例：

```env
VITE_API_BASE_URL=https://api.example.com/api
```

构建：

```bash
cd web
npm install
npm run build
```

### 3. 组装 release 目录

Linux/macOS：

```bash
rm -rf release
mkdir -p release/server release/web release/public

cp server/wjfcm-go-api release/server/
cp -r server/templates release/server/
cp -r web/dist release/web/
cp -r public/* release/public/
cp server/.env.example release/server/.env.example
```

Windows PowerShell：

```powershell
Remove-Item -Recurse -Force release -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force release\server, release\web, release\public | Out-Null

Copy-Item server\templates release\server\templates -Recurse
Copy-Item web\dist release\web\dist -Recurse
Copy-Item public\* release\public -Recurse
Copy-Item server\.env.example release\server\.env.example
```

生产 `.env` 建议在服务器上手工创建、通过 CI Secret 注入，或第一次启动后访问 `/install` 生成。不建议从开发机直接打包真实密码。

## 首次安装向导

新服务器第一次部署时，可以先上传二进制、`templates/`、`web/dist/`、`public/` 和 `server/.env.example`，然后启动服务：

```bash
systemctl start wjfcm-go-api
```

打开：

```text
https://www.example.com/install
```

安装向导会完成：

- 检查安装状态。
- 填写站点名称、站点地址、服务端口。
- 填写数据库地址、端口、库名、账号、密码、表前缀。
- 填写超级管理员账号和密码。
- 自动创建数据库和数据表。
- 写入超级管理员、角色、权限菜单、基础系统配置、默认分类和前台导航。
- 生成 `server/.env`。
- 生成 `server/.install.lock`，防止重复安装。

安装成功后需要重启服务，让新 `.env` 生效：

```bash
systemctl restart wjfcm-go-api
```

然后访问：

```text
https://www.example.com/admin/login
```

如果要重新安装，必须先备份数据库和上传文件，再删除服务器上的 `server/.install.lock`。生产环境不要随意删除安装锁。

## 上传到服务器

示例目录：

```text
/www/wwwroot/wjfcm-go/
  server/
  web/
  public/
```

上传方式可以用 `scp`、`rsync`、宝塔文件管理、CI/CD Artifact 等。上传后确认二进制可执行：

```bash
chmod +x /www/wwwroot/wjfcm-go/server/wjfcm-go-api
```

## 服务器配置

`/www/wwwroot/wjfcm-go/server/.env` 示例：

```env
APP_ENV=production
APP_DEBUG=false
APP_PORT=8080
APP_URL=https://www.example.com
JWT_SECRET=请换成足够长的随机字符串

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
DB_DATABASE=wjfcm_go
DB_USERNAME=wjfcm_go
DB_PASSWORD=请填写生产密码
DB_PREFIX=wjf_

CORS_ALLOW_ORIGINS=https://www.example.com
PUBLIC_DIR=/www/wwwroot/wjfcm-go/public
UPLOAD_BASE_PATH=uploads
```

注意：当前 Gin 会从工作目录读取 `templates/*.tmpl`，所以 systemd 的 `WorkingDirectory` 必须指向 `server/`。

如果开启 `REQUEST_LOG_ENABLED=true`，默认只记录 `/api/` 接口请求，日志会写到 `REQUEST_LOG_PATH/YYYY-MM-DD/requests-YYYY-MM-DD.log`，并按 `REQUEST_LOG_MAX_FILE_MB` 自动分割。生产环境建议保证 `server/storage/` 可写，并定期清理或备份。

## systemd 示例

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

启动：

```bash
systemctl daemon-reload
systemctl enable wjfcm-go-api
systemctl start wjfcm-go-api
systemctl status wjfcm-go-api
```

## Nginx 示例

Vue 后台静态产物由 Nginx 读取，前台 SEO 页面和 API 反代给 Gin：

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

## 发布更新流程

建议每次发版按这个顺序：

1. 本地或 CI 编译新的 `wjfcm-go-api`。
2. 本地或 CI 构建新的 `web/dist`。
3. 备份服务器当前二进制和 `web/dist`。
4. 上传新的二进制、`templates/`、`web/dist/`。
5. 不覆盖服务器生产 `.env` 和 `public/uploads/`。
6. 执行 `systemctl restart wjfcm-go-api`。
7. 检查首页、文章详情、后台登录、API 健康检查和上传目录。

## 检查命令

```bash
curl http://127.0.0.1:8080/api/health
curl https://www.example.com/
curl https://www.example.com/article/1
curl https://www.example.com/sitemap.xml
curl https://www.example.com/admin/login
```

文章页 `curl` 应该能直接看到标题、正文、meta、JSON-LD 等 HTML 内容。如果只看到 Vue 空壳，说明 Nginx 把 SEO 页面错误地交给了 Vue。

## 后续可优化

如果你希望服务器连 `templates/` 都不放，只保留一个二进制，可以把 Gin 模板改成 `go:embed` 编译进程序。当前版本还没有做模板嵌入，所以部署包仍需要携带 `server/templates/`。
