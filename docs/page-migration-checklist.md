# 页面迁移对照清单

本清单按旧版 页面逐页核对新版 Gin + Vue。状态说明：

- `done`: 页面主流程已搬运，可继续细化样式和边角行为。
- `partial`: 已有基础页面，但旧功能未完整对齐。
- `todo`: 尚未迁移。

## 前台公共布局

| 旧页面/组件 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| `resources/views/layouts/home.blade.php` | `server/templates/seo_*.tmpl` | done | 前台已改 Gin 服务端渲染，SEO meta、站点验证、统计代码、谷歌广告、页脚版权/备案/举报邮箱已补 |
| `page-agent` 集成 | `server/templates/seo_parts.tmpl` | done | Gin HTML 直出页面已接入，后续按需要改为可配置开关 |
| `layouts/home/toolbar.blade.php` | `server/templates/seo_parts.tmpl` | done | HTML+JS 右侧工具条、微信、打赏、返回顶部已完成 |
| 侧栏组件合集 | `server/templates/seo_parts.tmpl` | done | 公众号、广告位、申请友链表单已完成；历史上的今天已移除 |

## 前台页面

| 旧页面 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| 首页 `home/index/index.blade.php` | `server/templates/seo_index.tmpl` | done | Gin HTML 直出，旧 Vue 前台页已删除 |
| 文章详情 `home/index/article.blade.php` | `server/templates/seo_article.tmpl` | done | Gin HTML 直出正文、JSON-LD、转载提示、上一篇/下一篇、图片放大预览、评论 HTML+JS |
| 分类页 `home/index/category.blade.php` | `server/templates/seo_list.tmpl` | done | Gin HTML 直出，分页、阅读原文、关键词、空状态已补 |
| 标签页 `home/index/tag.blade.php` | `server/templates/seo_list.tmpl` | done | Gin HTML 直出，旧 Vue 前台页已删除 |
| 归档页 `home/index/archive.blade.php` | `server/templates/seo_archive.tmpl` | done | Gin HTML 直出分类归档 |
| 有些话 `home/index/chat.blade.php` | `server/templates/seo_chat.tmpl` | done | Gin HTML 直出列表和分页 |
| 搜索页/搜索动作 | `server/templates/seo_list.tmpl`、`/search?q=` | done | Gin HTML 直出搜索结果 |
| 错误页 `blank` | `server/templates/seo_blank.tmpl`、`/blank` | done | 兼容旧错误页入口，支持通过 `message` 查询参数展示提示 |

## 前台用户

| 旧页面/接口 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| 登录 `auth/login.blade.php` | `server/templates/seo_login.tmpl`、`POST /api/home/auth/login` | done | HTML+JS、图形验证码、忘记密码入口已完成 |
| 注册 `auth/register.blade.php` | `server/templates/seo_register.tmpl`、`POST /api/home/auth/register` | done | HTML+JS、邮箱验证码、图形验证码、确认密码校验已完成 |
| 第三方授权 `/auth/{social}` | `GET /api/home/auth/:provider`、`GET /api/home/auth/:provider/callback` | done | GitHub/QQ/微博 OAuth 跳转、回调、自动创建/绑定前台用户已补；上线需填写真实 Client 配置 |
| 用户中心 `home/user/index.blade.php` | `server/templates/seo_user.tmpl` | done | HTML+JS 读取资料、退出登录已完成 |
| 用户资料修改 `home/user/modify.blade.php` | `server/templates/seo_user.tmpl`、`PUT /api/home/profile` | done | HTML+JS 修改资料、头像上传、修改邮箱验证码已完成 |
| 头像上传 `/user/uploadImage` | `POST /api/home/upload/image` | done | 后续可加裁剪 |

## 评论

| 旧页面/接口 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| 评论列表 `layouts/home/msgboard.blade.php` | `server/templates/seo_article.tmpl`、`GET /api/home/comments` | done | HTML+JS 评论列表、回复、表情、点赞/踩已完成 |
| 评论提交 `/user/comment` | `POST /api/home/comments` | done | 当前已要求登录后评论 |
| 评论动作 `/user/commentAction` | `POST /api/home/comments/:id/action` | done | 登录用户可点赞、踩 |

## 后台

| 模块 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| 文章列表/新增/编辑 | `ArticleList.vue`、`ArticleForm.vue` | done | 百家号推送、批量删除、软删除恢复按钮已完成 |
| 分类管理 | `ResourceList.vue` | done | 树折叠、添加子分类、分类文章快捷入口、排序行内编辑已完成 |
| 标签管理 | `ResourceList.vue` | done | 批量删除、恢复/彻底删除已完成 |
| 评论管理 | `ResourceList.vue` | done | 批量删除、恢复/彻底删除、审核开关、批量替换已完成 |
| 角色管理 | `ResourceList.vue` | done | 状态开关、恢复/彻底删除已完成 |
| 权限管理 | `ResourceList.vue` | done | 树折叠、添加子菜单、排序行内编辑、上级下拉、自动层级、绑定校验已完成 |
| 管理员列表 | `ResourceList.vue` | done | 账号密码弹窗、批量删除、恢复/彻底删除已完成 |
| 用户列表 | `ResourceList.vue` | done | 新增用户、账号密码弹窗、批量删除、恢复/彻底删除已完成 |
| 导航/友链/闲言碎语/系统配置 | `ResourceList.vue` | done | 导航树、导航旧字段、友链申请/审核、闲言碎语、系统配置图片上传、基础/SEO/微信/小程序分组筛选已完成 |
| 微信关键词回复 | `ResourceList.vue` | done | 关键词搜索、新增、编辑、审核开关、排序行内编辑已完成；旧表无 `deleted_at`，按普通删除处理 |
| 微信服务端入口 `/wechat` | `WechatHandler` | done | 已支持服务器校验、关注事件回复、文本关键词回复、QQ AI / 图灵文本兜底，以及图片/语音的图灵兜底 |

## 工具/第三方

| 旧页面/接口 | 新版位置 | 状态 | 待补功能 |
| --- | --- | --- | --- |
| 百度主动推送 `/tools/linkSubmit` | `GET /tools/linkSubmit`、`POST /api/admin/tools/baidu-submit` | done | 已按旧逻辑提交首页、登录注册、有些话、文章、分类、标签 URL |
| 调试接口 `/tools/tuling`、后台 `/admin/test/*` | - | done | 旧版调试入口，不迁移到生产新版 |
| 百家号回调 `/baidu/serve` | `ANY /baidu/serve` | done | 已按旧逻辑校验签名并解密 `encrypt`，上线需用真实回调参数和 AES Key 联调 |
| 百家号文章发布 `/admin/baijiahao/article/publish` | `POST /api/admin/articles/:id/baijiahao` | done | 已支持文章列表一键发布 |
| 文章批量替换 `/admin/article/replace` | `POST /api/admin/articles/replace`、文章列表“批量替换” | done | 支持替换标题、简介、Markdown、HTML 正文，包含回收站文章，并兼容旧正文 HTML 转义内容 |
| 评论批量替换 `/admin/comment/replace` | `POST /api/admin/comments/replace`、评论列表“批量替换” | done | 支持查找/替换评论内容，包含回收站评论 |
