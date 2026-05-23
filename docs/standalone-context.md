# wjfcm-go 独立项目上下文

## 项目定位

`wjfcm-go` 是从旧 `laravel-wjfcms` 迁移出来的独立项目。后端使用 Gin，后台使用 Vue，前台公开页面使用 Gin 模板服务端渲染，以保证 SEO。

## 旧项目来源

- 旧项目名称：`laravel-wjfcms`
- 旧项目路径：`D:\wwwroot\php\laravel-wjfcms`
- 旧技术栈：Laravel 6、Blade、Layui、MySQL、Spatie Permission、Laravel WeChat、Socialite
- 迁移说明：见 `docs/migration.md`
- 迁移进度：见 `docs/progress.md`
- 页面清单：见 `docs/page-migration-checklist.md`

如果以后只打开 `wjfcm-go` 独立目录，Codex 能继续识别新版项目；如果还需要继续对照旧 Laravel 的源码细节，需要同时提供旧项目路径或旧项目只读副本。

## 当前目录约定

```text
wjfcm-go/
  server/       Gin 后端、前台 SEO 模板、API
  web/          Vue 后台管理端
  public/       站点公开静态资源、上传资源、验证文件
  docs/         迁移说明、进度、页面清单
  deploy/       部署配置
```

## 静态资源约定

`server/.env` 中 `PUBLIC_DIR=../public`，表示从 `wjfcm-go/server` 启动时，公开资源目录指向 `wjfcm-go/public`。

这些根文件会直接通过域名访问：

```text
/favicon.ico
/ads.txt
/bdunion.txt
/google6131677a16626804.html
/root.txt
```

`/images/*` 和 `/uploads/*` 继续作为静态目录访问。`/robots.txt` 和 `/sitemap.xml` 由 Gin 动态输出，便于根据站点配置生成。

`public/` 已清理旧 Laravel/Layui 构建产物，仅保留新版仍需要的内容：

- 根目录验证文件：`favicon.ico`、`ads.txt`、`bdunion.txt`、`google*.html`、`root.txt`、`robots.txt`。
- `images/`：站点 LOGO、头像、微信/打赏二维码、默认图等历史配置图片。
- `uploads/`：后台和前台上传资源。

旧版 `public/css`、`public/js`、`public/fonts`、`public/static`、`index.php`、`mix-manifest.json`、`web.config` 属于 Laravel/Layui 运行或构建文件，新版不再依赖，已删除。

## 当前已完成重点

- Gin API、GORM、JWT、权限中间件、SQL 日志、上传。
- Vue 后台：文章、分类、标签、评论、用户、管理员、角色、权限、导航、友链、闲言碎语、系统配置、微信关键词回复。
- 前台：Gin 模板直出首页、列表、文章详情、归档、有些话、登录注册、找回密码、用户中心。
- SEO：动态 meta、JSON-LD、`robots.txt`、`sitemap.xml`、根目录验证文件访问。
- 评论：表情、盖楼回复、点赞/踩、后台回复关系展示、批量替换。
- 文章：Markdown/HTML 编辑、百家号推送、软删除恢复、批量替换。
- 微信：公众号验证、关键词回复、QQ AI / 图灵兜底、后台微信配置。
- 第三方：OAuth 代码、百家号发布、百度主动推送。

## 仍需重点确认

- 真实 OAuth 平台回调配置。
- 真实 SMTP 发信。
- 微信公众号服务器 URL 联调。
- 百家号/百度站长线上接口联调。
