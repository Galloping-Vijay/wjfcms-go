# laravel-wjfcms 迁移到 Gin + Vue 说明书

## 1.1 迁移目标

当前项目是 Laravel 6 + Blade + Layui 的 CMS 后台管理系统。迁移目标不是简单替换框架，而是将现有单体 MVC 应用改造成：

- 后端：Gin 提供 RESTful API。
- 前端：Vue 构建前台页面和后台管理端。
- 数据库：优先复用现有 MySQL 表结构，必要时做兼容调整。
- 认证：统一整理前台用户、后台管理员、API Token、第三方登录。
- 权限：保留当前 RBAC 能力，迁移为 Go 可维护的权限模型。

推荐迁移原则：

- 业务数据优先保留。
- 功能按模块逐步替换。
- 新旧系统可以短期并行。
- 后端先接口化，前端再组件化。
- 先迁核心 CMS 能力，再迁第三方集成。

## 1.2 站点地址
前台: http://blog.localhost/

后台: http://blog.localhost/admin
账号:13015920170
密码:qqwei12345

## 1.3 前提
在不影响数据的情况下开发,只修改前后端技术栈.

## 2. 当前项目关键事实

### 2.1 技术栈

- Laravel 6.x
- PHP `^7.1.3`
- MySQL，表前缀默认 `wjf_`
- Blade 模板
- Layui / layuiadmin 后台 UI
- Laravel Mix + Webpack
- Vue 2 依赖已存在，但当前核心页面仍以 Blade 为主
- Spatie Laravel Permission 做 RBAC
- JWT Auth / Passport / Socialite / Laravel WeChat 等扩展

### 2.2 当前入口

- Web 入口：`public/index.php`
- Web 路由：`routes/web.php`
- API 路由：`routes/api.php`
- CLI 入口：`artisan`
- 后台入口：`/admin/login`
- 前台首页：`/`

### 2.3 当前模块

- 前台内容模块：文章、分类、标签、归档、搜索、评论、友情链接。
- 用户模块：注册、登录、用户中心、头像上传、评论操作。
- 后台模块：管理员、用户、权限、角色、系统配置、文章、分类、标签、评论、导航、友情链接、聊天内容。
- 第三方模块：微信服务、百度提交、百家号发布、QQ/微博/微信登录。
- 基础设施：图片上传、水印、邮件验证码、验证码、队列、Redis 配置。

## 3. 目标架构

### 3.1 后端目录建议

```text
server/
  cmd/
    api/
      main.go
  internal/
    config/
    router/
    middleware/
    handler/
      admin/
      home/
      auth/
      api/
    service/
    repository/
    model/
    dto/
    response/
    validator/
    pkg/
      captcha/
      uploader/
      watermark/
      wechat/
      baidu/
      baijiahao/
  migrations/
  docs/
```

建议分层：

- `handler`：只处理 HTTP 请求、参数绑定、响应。
- `service`：业务逻辑。
- `repository`：数据库查询。
- `model`：数据库模型。
- `dto`：请求和响应结构。
- `middleware`：认证、权限、日志、限流、跨域。

### 3.2 前端目录建议

```text
web/
  src/
    api/
    router/
    stores/
    layouts/
      admin/
      home/
    views/
      admin/
      home/
      auth/
    components/
    utils/
    styles/
```

建议选型：

- Vue 3 + Vite
- Pinia 状态管理
- Vue Router
- Axios
- Element Plus / Naive UI / Ant Design Vue 三选一
- Markdown 编辑器继续选成熟组件

如果你想尽量少重写后台 UI，也可以短期保留 Layui 静态资源；但从长期维护看，后台建议改成 Vue 组件库。

### 3.3 部署形态

推荐两种方式：

方案 A：前后端分离部署。

```text
Nginx
  /               -> Vue 前台构建产物
  /admin          -> Vue 后台构建产物
  /api            -> Gin 服务
  /uploads        -> 静态资源目录或对象存储
```

方案 B：Gin 托管 Vue 静态产物。

```text
Gin
  /api/*          -> API
  /assets/*       -> 静态资源
  /*              -> Vue history fallback
```

开发阶段推荐方案 A，生产阶段两种都可以。

## 4. 数据库迁移策略

### 4.1 优先复用的表

这些表建议直接保留，并用 Go 模型适配：

- `wjf_users`
- `wjf_admins`
- `wjf_articles`
- `wjf_categories`
- `wjf_tags`
- `wjf_comments`
- `wjf_article_tags`
- `wjf_navs`
- `wjf_friend_links`
- `wjf_chats`
- `wjf_system_configs`
- `wjf_oauth_users`
- `wjf_wx_keywords`

### 4.2 需要重点核对的字段

- `created_at` / `updated_at` / `deleted_at`
- Laravel 软删除字段 `deleted_at`
- 文章 `content` 与 `markdown`
- 文章状态、置顶、点击量字段
- 分类层级字段
- 权限表中的菜单字段，如 `parent_id`、`level`、`display_menu`、`icon`、`url`
- 图片路径是否为 Laravel public 相对路径

### 4.3 ORM 选择

推荐 GORM，原因：

- 对 MySQL、软删除、分页比较友好。
- 与 Laravel 常见字段习惯适配成本低。
- 社区成熟，项目交接更容易。

也可以使用 `sqlc` 或 `ent`，但迁移初期 GORM 成本最低。

## 5. 认证与权限迁移

### 5.1 当前认证问题

当前系统有多套认证：

- 前台用户：Laravel `web` guard。
- 后台管理员：Laravel `admin` guard。
- API：JWT guard。
- 第三方登录：Socialite。
- Passport OAuth 相关能力。

迁移时不能简单照搬，需要先确定目标认证策略。

### 5.2 推荐目标方案

后台管理端：

- 使用 Access Token + Refresh Token。
- Access Token 短有效期。
- Refresh Token 存数据库或 Redis。
- 管理员密码继续使用 bcrypt 或兼容 Laravel hash。

前台用户端：

- 可以使用同一套 Token 机制。
- 如果前台更偏传统网站，也可用 HttpOnly Cookie Session。

API：

- 统一从 `Authorization: Bearer <token>` 读取。

### 5.3 Laravel 密码兼容

Laravel 默认使用 bcrypt。Go 可以用 `golang.org/x/crypto/bcrypt` 验证旧密码。

迁移时不需要强制用户重置密码，只要保持 bcrypt 校验兼容即可。

### 5.4 RBAC 迁移

当前使用 Spatie Laravel Permission。迁移有两种方案：

方案 A：自研 RBAC。

- 保留现有角色、权限、菜单表结构。
- 在 Gin 中实现权限中间件。
- 前端根据用户权限渲染菜单和按钮。

方案 B：使用 Casbin。

- 更标准，适合复杂权限。
- 但需要将现有 Spatie 数据迁移到 Casbin policy。

推荐先用方案 A，理由是当前系统权限更像“后台菜单 + 操作权限”，自研迁移更直接。

## 6. API 设计建议

### 6.1 响应格式

建议统一响应：

```json
{
  "code": 0,
  "message": "操作成功",
  "data": {},
  "meta": {}
}
```

分页格式：

```json
{
  "code": 0,
  "message": "OK",
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100
  }
}
```

这样可以兼容当前 Laravel 中 `resJson()` 的思路。

### 6.2 后台 API 示例

```text
POST   /api/admin/auth/login
POST   /api/admin/auth/logout
GET    /api/admin/profile

GET    /api/admin/articles
POST   /api/admin/articles
GET    /api/admin/articles/:id
PUT    /api/admin/articles/:id
DELETE /api/admin/articles/:id
POST   /api/admin/articles/:id/restore
DELETE /api/admin/articles/:id/force

GET    /api/admin/categories
POST   /api/admin/categories
PUT    /api/admin/categories/:id
DELETE /api/admin/categories/:id

GET    /api/admin/tags
GET    /api/admin/comments
GET    /api/admin/users
GET    /api/admin/admins
GET    /api/admin/roles
GET    /api/admin/permissions
```

### 6.3 前台 API 示例

```text
GET  /api/home
GET  /api/articles
GET  /api/articles/:id
GET  /api/categories/:id/articles
GET  /api/tags/:id/articles
GET  /api/archive
POST /api/search
GET  /api/friend-links
POST /api/comments
GET  /api/comments
```

## 7. 页面迁移策略

### 7.1 后台优先

后台是管理系统，页面结构明确，最适合先迁移：

1. 登录页。
2. 后台布局、菜单、标签页或面包屑。
3. 文章管理。
4. 分类管理。
5. 标签管理。
6. 评论管理。
7. 用户和管理员管理。
8. 角色权限管理。
9. 系统配置。
10. 友情链接、导航、聊天内容。

### 7.2 前台后迁

前台涉及 SEO 和页面细节，建议等 API 稳定后迁移：

1. 首页。
2. 文章详情。
3. 分类页。
4. 标签页。
5. 归档页。
6. 搜索页。
7. 用户中心。

如果非常重视 SEO，可以考虑 Nuxt，而不是纯 Vue SPA。

## 8. 第三方功能迁移

### 8.1 图片上传与水印

当前 Laravel 使用 Intervention Image 和全局 `waterMarkImage()`。Go 侧需要重写：

- 上传校验。
- 图片压缩。
- 封面裁剪。
- 水印。
- 默认图跳过水印。
- 本地 public 存储或七牛云存储。

### 8.2 社交登录

QQ、微博、微信网页版需要重新实现 OAuth 流程：

- redirect URL。
- callback 换 token。
- 拉取用户信息。
- 绑定或创建本地用户。
- 写入 `oauth_users`。

### 8.3 微信服务

微信公众号服务需要迁移：

- token 验证。
- 消息接收。
- 关键词回复。
- 图文或文本回复。

### 8.4 百度与百家号

这部分建议后置迁移，因为它们不影响 CMS 核心闭环。

## 9. 分阶段计划

### 阶段一：基础骨架

目标：让 Gin + Vue 项目跑起来。

交付：

- Gin 项目结构。
- Vue 项目结构。
- 数据库连接。
- 配置加载。
- 日志。
- CORS。
- 统一响应。
- 健康检查接口。

预计：1-2 天。

### 阶段二：认证与后台基础

目标：可以登录后台并进入管理界面。

交付：

- 管理员登录。
- Token 刷新。
- 当前管理员信息。
- 后台布局。
- 动态菜单。
- 权限中间件初版。

预计：3-5 天。

### 阶段三：CMS 核心模块

目标：完成内容管理闭环。

交付：

- 文章 CRUD。
- 分类 CRUD。
- 标签 CRUD。
- 评论管理。
- 上传图片。
- Markdown 编辑。
- 软删除、恢复、彻底删除。

预计：7-12 天。

### 阶段四：系统管理模块

目标：后台可完整维护基础数据和权限。

交付：

- 用户管理。
- 管理员管理。
- 角色管理。
- 权限管理。
- 系统配置。
- 导航管理。
- 友情链接管理。
- 聊天内容管理。

预计：7-12 天。

### 阶段五：前台页面

目标：替换当前 Blade 前台。

交付：

- 首页。
- 文章详情。
- 分类、标签、归档。
- 搜索。
- 评论。
- 用户中心。

预计：5-10 天。

### 阶段六：第三方集成

目标：恢复扩展能力。

交付：

- 邮件验证码。
- 验证码。
- 第三方登录。
- 微信服务。
- 百度提交。
- 百家号发布。
- 七牛云上传。

预计：7-15 天。

### 阶段七：联调、测试、上线

目标：生产可用。

交付：

- 数据迁移校验。
- 接口测试。
- 权限边界测试。
- 构建脚本。
- Nginx 配置。
- 日志和异常处理。
- 旧系统切换方案。

预计：5-10 天。

## 10. 时间评估

按我来开发的估算：

- MVP：2-3 周。
- 可替换当前主流程版本：5-8 周。
- 生产级完整迁移：8-12 周。

影响工期最大的因素：

- 是否保留 Layui。
- 前台是否要求 SEO。
- 微信、百度、百家号是否必须首版迁移。
- 权限是否要做到按钮级。
- 是否允许复用现有数据库结构。
- 是否要兼容旧密码和旧图片路径。

## 11. 主要风险

### 11.1 Blade 到 Vue 不是自动迁移

Blade 模板里可能混有 PHP 变量、路由、权限判断和静态资源路径。迁 Vue 时必须拆为：

- API 数据。
- Vue 组件。
- 路由状态。
- 权限状态。
- 样式和资源引用。

### 11.2 权限容易漏

后台菜单、接口权限、按钮权限是三层问题。只做菜单隐藏不够，后端接口必须校验权限。

### 11.3 软删除语义要保持

当前后台很多模块有：

- 普通列表。
- 回收站。
- 恢复。
- 彻底删除。

Go API 需要保留这些行为，否则后台功能会退化。

### 11.4 图片路径兼容

旧内容里的图片路径可能写死为 Laravel public 路径。迁移后要保证：

- 老图片能访问。
- 新上传路径稳定。
- 文章内容里的图片不失效。

### 11.5 SEO 风险

如果前台改为纯 Vue SPA，搜索引擎收录可能受影响。若博客/CMS 前台重视 SEO，建议使用：

- Nuxt SSR/SSG。
- 或 Gin 输出前台 HTML，Vue 只做后台。
- 或前台继续服务端渲染，后台使用 Vue。

## 12. 推荐实施路线

我建议按下面路线做：

1. 新建 `server/` 和 `web/`，不直接破坏现有 Laravel 项目。
2. 先用 Gin 连现有 MySQL，读取文章、分类、标签。
3. 做后台登录和管理员信息接口。
4. Vue 后台先跑通文章管理。
5. 再接分类、标签、评论。
6. 再迁移权限和菜单。
7. 核心后台稳定后迁移前台页面。
8. 最后迁微信、百度、百家号、第三方登录。

这个顺序的好处是：每一步都能验收，旧系统还能继续使用，迁移风险可控。

## 13. 第一版 MVP 范围建议

第一版不建议一次全做。MVP 可以只包含：

- Gin API 基础框架。
- Vue 后台基础框架。
- 管理员登录。
- 文章管理。
- 分类管理。
- 标签管理。
- 评论管理。
- 图片上传。
- 前台首页、文章详情、分类文章列表。

暂缓：

- 第三方登录。
- 微信服务。
- 百度提交。
- 百家号发布。
- Passport。
- 复杂权限配置页面。

## 14. 验收标准

### 后端

- 可以连接现有数据库。
- 可以兼容 Laravel bcrypt 密码。
- 所有接口返回统一 JSON。
- 管理接口需要认证。
- 权限接口需要后端校验。
- 文章、分类、标签、评论支持分页、搜索、软删除。
- 上传图片后旧内容和新内容都能正常访问。

### 前端

- 后台登录成功后进入首页。
- 菜单根据权限显示。
- 列表页支持搜索、分页、编辑、删除。
- 表单校验完整。
- Token 过期时可以刷新或重新登录。
- 生产构建可部署。

### 数据

- 老文章可读取。
- 老用户和管理员可登录。
- 老图片可访问。
- 老权限数据可转换或兼容。
- 表前缀 `wjf_` 不影响新 ORM 查询。

## 15. 结论

这个项目可以迁到 Gin + Vue，但更适合按“重构迁移”而不是“代码翻译”来做。

最稳妥的方案是：

- 数据库先保留。
- 后台先迁。
- 核心 CMS 先迁。
- 前台和第三方集成后迁。
- Laravel 与 Gin + Vue 短期并行。

这样既能控制风险，也能让每个阶段都有可验证成果。
