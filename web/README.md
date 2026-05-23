# Vue 后台

## 开发启动

```bash
cd wjfcm-go/web
copy .env.example .env
npm install
npm run dev
```

默认地址：

```text
http://localhost:5173
```

后台入口：

```text
http://localhost:5173/admin/login
```

前台页面不再由 Vue 承载，开发时请访问 Gin：

```text
http://localhost:8080/
http://localhost:8080/article/1
http://localhost:8080/login
```

## 当前范围

- 后台登录页
- 后台基础布局
- 文章列表
- 文章新增/编辑
- 文章封面上传
- 文章 Markdown 编辑、HTML 预览、正文图片插入
- 通用后台列表页，支持按模块新增、编辑、删除

## 路由归属

- 公开前台 SEO 页面：Gin 模板，位于 `../server/templates/seo_*.tmpl`
- 后台管理页面：Vue，位于 `src/views/admin`
- 前台用户交互页面：Gin 模板，位于 `../server/templates/seo_login.tmpl`、`seo_register.tmpl`、`seo_forgot_password.tmpl`、`seo_user.tmpl`
