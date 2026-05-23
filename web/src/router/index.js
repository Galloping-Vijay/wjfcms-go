import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AdminLayout from '../layouts/admin/AdminLayout.vue'
import AdminLogin from '../views/auth/AdminLogin.vue'
import AdminDashboard from '../views/admin/Dashboard.vue'
import AdminArticles from '../views/admin/articles/ArticleList.vue'
import ArticleForm from '../views/admin/articles/ArticleForm.vue'
import AdminResourceList from '../views/admin/ResourceList.vue'

const routes = [
  {
    path: '/admin/login',
    name: 'admin.login',
    component: AdminLogin,
    meta: { guest: true }
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'admin.dashboard', component: AdminDashboard },
      { path: 'articles', name: 'admin.articles', component: AdminArticles },
      { path: 'articles/create', name: 'admin.articles.create', component: ArticleForm },
      { path: 'articles/:id/edit', name: 'admin.articles.edit', component: ArticleForm },
      { path: 'comments', name: 'admin.comments', component: AdminResourceList, meta: { resource: 'comments' } },
      { path: 'categories', name: 'admin.categories', component: AdminResourceList, meta: { resource: 'categories' } },
      { path: 'tags', name: 'admin.tags', component: AdminResourceList, meta: { resource: 'tags' } },
      { path: 'users', name: 'admin.users', component: AdminResourceList, meta: { resource: 'users' } },
      { path: 'admins', name: 'admin.admins', component: AdminResourceList, meta: { resource: 'admins' } },
      { path: 'roles', name: 'admin.roles', component: AdminResourceList, meta: { resource: 'roles' } },
      { path: 'permissions', name: 'admin.permissions', component: AdminResourceList, meta: { resource: 'permissions' } },
      { path: 'navs', name: 'admin.navs', component: AdminResourceList, meta: { resource: 'navs' } },
      { path: 'friend-links', name: 'admin.friendLinks', component: AdminResourceList, meta: { resource: 'friend-links' } },
      { path: 'chats', name: 'admin.chats', component: AdminResourceList, meta: { resource: 'chats' } },
      { path: 'wx-keywords', name: 'admin.wxKeywords', component: AdminResourceList, meta: { resource: 'wx-keywords' } },
      { path: 'system-configs', name: 'admin.systemConfigs', component: AdminResourceList, meta: { resource: 'system-configs' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) {
    return { name: 'admin.login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guest && auth.token) {
    return { name: 'admin.dashboard' }
  }
  return true
})

export default router
