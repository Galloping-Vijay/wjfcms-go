# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WjfCMS-Go 是一个 CMS 内容管理系统，后端使用 Go (Gin)，前端管理面板使用 Vue 3 + Vite。公开页面由 Gin 通过 Go 模板进行服务端渲染（SSR），仅管理后台是 Vue SPA。

## Development Commands

### Backend (server/)

```bash
cd server
go run ./cmd/api                     # 启动开发服务器（默认端口 8080）
go run ./cmd/api -f .env.local       # 使用自定义 env 文件
go build -o wjfcms-go-api ./cmd/api  # 生产构建
go test ./...                        # 运行测试
```

### Frontend (web/)

```bash
cd web
npm install      # 安装依赖
npm run dev      # 开发服务器（端口 5173，代理 /api 到 :8080）
npm run build    # 生产构建（输出到 web/dist/）
npm run preview  # 预览生产构建
```

## Architecture

### 目录结构

```
server/
  cmd/api/main.go           # 应用入口
  internal/
    config/config.go        # 配置加载（.env 文件）
    database/               # GORM MySQL 连接
    handler/                # HTTP 请求处理器（22 个文件）
    middleware/auth.go      # JWT 认证中间件
    model/                  # GORM 模型定义（14 个文件）
    requestlog/             # 请求日志中间件
    response/response.go    # 统一 JSON 响应
    router/router.go        # 路由定义
    service/                # 业务逻辑服务
  templates/                # SSR Go 模板（12 个 .tmpl 文件）

web/
  src/
    api/                    # API 客户端层（Axios）
    components/admin/       # 共享管理 UI 组件
    layouts/admin/          # 管理后台布局
    router/index.js         # Vue Router
    stores/auth.js          # Pinia 认证状态
    views/admin/            # 管理页面视图
```

### 分层架构

配置 → 数据库 → 路由 → 中间件 → Handler → Model/Service

### 路由设计

- `/`、`/article/:id`、`/category/:id` 等 — SEO 页面（Gin SSR）
- `/api/auth/*` — 公开认证接口
- `/api/home/*` — 公开内容 API
- `/api/admin/*` — 管理 API（JWT 保护 + 权限检查）
- `/admin/*` — Vue SPA 管理后台

## Key Technical Decisions

- 数据库表使用 `wjf_` 前缀，兼容旧 Laravel CMS 表结构
- 公开 SEO 页面由 Gin 服务端渲染，Vue 仅用于管理后台
- 基于角色的权限系统复用旧 Spatie 权限表（`roles`、`permissions`、`role_has_permissions`）
- Admin ID 1 为超级管理员，拥有全部权限
- 全局使用软删除，支持恢复和强制删除
- 配置通过 `.env` 文件加载（godotenv），支持 `-f` 参数指定自定义文件
- `/install` 安装向导可自动建表、填充数据、生成 `.env`
