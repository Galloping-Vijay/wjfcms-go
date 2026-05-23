<template>
  <div class="admin-shell" :class="{ collapsed: sidebarCollapsed }">
    <aside class="admin-sidebar">
      <div class="admin-brand">
        <span>{{ sidebarCollapsed ? 'G' : 'wjfcm-go' }}</span>
        <button type="button" class="sidebar-toggle" :title="sidebarCollapsed ? '展开菜单' : '收起菜单'" @click="toggleSidebar">
          {{ sidebarCollapsed ? '›' : '‹' }}
        </button>
      </div>
      <nav class="admin-menu">
        <template v-for="item in menus" :key="item.key">
          <div v-if="item.children?.length" class="admin-menu-group">
            <button type="button" class="admin-menu-title" :title="item.name" @click="toggleGroup(item.key)">
              <span>{{ item.name }}</span>
              <span class="menu-caret" :class="{ open: isGroupOpen(item.key) }">›</span>
            </button>
            <template v-for="child in item.children" :key="child.key">
              <div v-if="child.children?.length" v-show="isGroupOpen(item.key)" class="admin-menu-subgroup">
                <button type="button" class="admin-menu-title sub-title" :title="child.name" @click="toggleGroup(child.key)">
                  <span>{{ child.name }}</span>
                  <span class="menu-caret" :class="{ open: isGroupOpen(child.key) }">›</span>
                </button>
                <RouterLink
                  v-show="isGroupOpen(child.key)"
                  v-for="grandchild in child.children"
                  :key="grandchild.key"
                  class="admin-menu-item sub-item"
                  :class="{ 'menu-active': isMenuActive(grandchild.path) }"
                  :to="grandchild.path"
                  :title="grandchild.name"
                >
                  <span class="menu-text">{{ grandchild.name }}</span>
                  <span class="menu-short">{{ shortName(grandchild.name) }}</span>
                </RouterLink>
              </div>
              <RouterLink
                v-else
                v-show="isGroupOpen(item.key)"
                class="admin-menu-item"
                :class="{ 'menu-active': isMenuActive(child.path) }"
                :to="child.path"
                :title="child.name"
              >
                <span class="menu-text">{{ child.name }}</span>
                <span class="menu-short">{{ shortName(child.name) }}</span>
              </RouterLink>
            </template>
          </div>
          <RouterLink v-else class="admin-menu-item" :class="{ 'menu-active': isMenuActive(item.path) }" :to="item.path" :title="item.name">
            <span class="menu-text">{{ item.name }}</span>
            <span class="menu-short">{{ shortName(item.name) }}</span>
          </RouterLink>
        </template>
      </nav>
    </aside>

    <main class="admin-main">
      <header class="admin-topbar">
        <div class="admin-title-row">
          <button type="button" class="sidebar-toggle top-toggle" :title="sidebarCollapsed ? '展开菜单' : '收起菜单'" @click="toggleSidebar">☰</button>
          <strong>{{ pageTitle }}</strong>
        </div>
        <div class="admin-user">
          <span>{{ auth.admin?.username || auth.admin?.account || '管理员' }}</span>
          <button type="button" class="text-button" @click="openProfile">资料</button>
          <button type="button" class="text-button" @click="openPassword">密码</button>
          <button type="button" class="text-button" @click="logout">退出</button>
        </div>
      </header>
      <section class="admin-content">
        <RouterView />
      </section>
    </main>
    <AdminToast :items="toasts" />

    <div v-if="profileDialog.open" class="modal-mask">
      <form class="modal-panel admin-account-modal" @submit.prevent="saveProfile">
        <header class="modal-header">
          <strong>个人资料</strong>
          <button type="button" class="text-button" @click="closeProfile">关闭</button>
        </header>
        <label>
          账号
          <input :value="auth.admin?.account || ''" disabled />
        </label>
        <label>
          昵称
          <input v-model.trim="profileForm.username" required />
        </label>
        <label>
          手机号
          <input v-model.trim="profileForm.tel" />
        </label>
        <label>
          邮箱
          <input v-model.trim="profileForm.email" type="email" />
        </label>
        <label>
          性别
          <select v-model.number="profileForm.sex">
            <option :value="-1">保密</option>
            <option :value="0">男</option>
            <option :value="1">女</option>
          </select>
        </label>
        <p v-if="profileError" class="form-error">{{ profileError }}</p>
        <div class="form-actions">
          <button type="submit" :disabled="profileSaving">{{ profileSaving ? '保存中...' : '保存' }}</button>
          <button type="button" class="secondary-button" @click="closeProfile">取消</button>
        </div>
      </form>
    </div>

    <div v-if="passwordDialog.open" class="modal-mask">
      <form class="modal-panel admin-account-modal" @submit.prevent="savePassword">
        <header class="modal-header">
          <strong>修改密码</strong>
          <button type="button" class="text-button" @click="closePassword">关闭</button>
        </header>
        <label>
          原密码
          <input v-model="passwordForm.old_password" type="password" autocomplete="current-password" required />
        </label>
        <label>
          新密码
          <input v-model="passwordForm.password" type="password" autocomplete="new-password" required />
        </label>
        <label>
          确认新密码
          <input v-model="passwordForm.confirm_password" type="password" autocomplete="new-password" required />
        </label>
        <p v-if="passwordError" class="form-error">{{ passwordError }}</p>
        <div class="form-actions">
          <button type="submit" :disabled="passwordSaving">{{ passwordSaving ? '保存中...' : '确认修改' }}</button>
          <button type="button" class="secondary-button" @click="closePassword">取消</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getMenus } from '../../api/adminResources'
import AdminToast from '../../components/admin/AdminToast.vue'
import { useAuthStore } from '../../stores/auth'
import { notifyAdminSuccess } from '../../utils/adminToast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menus = ref(fallbackMenus())
const sidebarCollapsed = ref(localStorage.getItem('admin.sidebar.collapsed') === '1')
const openGroups = ref(new Set())
const toasts = ref([])
const profileSaving = ref(false)
const passwordSaving = ref(false)
const profileError = ref('')
const passwordError = ref('')
const profileDialog = reactive({ open: false })
const passwordDialog = reactive({ open: false })
const profileForm = reactive({
  username: '',
  tel: '',
  email: '',
  sex: -1
})
const passwordForm = reactive({
  old_password: '',
  password: '',
  confirm_password: ''
})
let toastSeed = 0

const pageTitle = computed(() => {
  if (route.name === 'admin.articles') return '文章管理'
  if (route.name === 'admin.articles.create') return '新增文章'
  if (route.name === 'admin.articles.edit') return '编辑文章'
  if (route.name === 'admin.comments') return '评论管理'
  if (route.name === 'admin.categories') return '分类管理'
  if (route.name === 'admin.tags') return '标签管理'
  if (route.name === 'admin.users') return '用户列表'
  if (route.name === 'admin.admins') return '管理员列表'
  if (route.name === 'admin.roles') return '角色管理'
  if (route.name === 'admin.permissions') return '权限管理'
  if (route.name === 'admin.navs') return '导航管理'
  if (route.name === 'admin.friendLinks') return '友情链接'
  if (route.name === 'admin.chats') return '闲言碎语'
  if (route.name === 'admin.systemConfigs') return '系统配置'
  return '控制台'
})

async function logout() {
  await auth.logout(true)
  router.push('/admin/login')
}

function openProfile() {
  profileError.value = ''
  profileForm.username = auth.admin?.username || ''
  profileForm.tel = auth.admin?.tel || ''
  profileForm.email = auth.admin?.email || ''
  profileForm.sex = auth.admin?.sex ?? -1
  profileDialog.open = true
}

function closeProfile() {
  if (profileSaving.value) return
  profileDialog.open = false
}

async function saveProfile() {
  profileSaving.value = true
  profileError.value = ''
  try {
    await auth.updateProfile({
      username: profileForm.username,
      tel: profileForm.tel,
      email: profileForm.email,
      sex: Number(profileForm.sex)
    })
    profileDialog.open = false
    notifyAdminSuccess('个人资料已保存')
  } catch (err) {
    profileError.value = err.message
  } finally {
    profileSaving.value = false
  }
}

function openPassword() {
  passwordError.value = ''
  passwordForm.old_password = ''
  passwordForm.password = ''
  passwordForm.confirm_password = ''
  passwordDialog.open = true
}

function closePassword() {
  if (passwordSaving.value) return
  passwordDialog.open = false
}

async function savePassword() {
  passwordError.value = ''
  if (passwordForm.password !== passwordForm.confirm_password) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  passwordSaving.value = true
  try {
    await auth.updatePassword({
      old_password: passwordForm.old_password,
      password: passwordForm.password
    })
    passwordDialog.open = false
    notifyAdminSuccess('密码已修改，请重新登录')
    window.setTimeout(async () => {
      await auth.logout(false)
      router.push('/admin/login')
    }, 900)
  } catch (err) {
    passwordError.value = err.message
  } finally {
    passwordSaving.value = false
  }
}

async function loadMenus() {
  try {
    const res = await getMenus()
    menus.value = normalizeMenus(res.data || [])
  } catch {
    menus.value = fallbackMenus()
  }
  ensureOpenGroups()
}

function ensureOpenGroups() {
  const next = new Set(openGroups.value)
  collectGroupKeys(menus.value).forEach((key) => next.add(key))
  openGroups.value = next
}

function collectGroupKeys(items = []) {
  return items.flatMap((item) => {
    if (!item.children?.length) return []
    return [item.key, ...collectGroupKeys(item.children)]
  })
}

function isGroupOpen(key) {
  return openGroups.value.has(key)
}

function toggleGroup(key) {
  const next = new Set(openGroups.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  openGroups.value = next
}

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem('admin.sidebar.collapsed', sidebarCollapsed.value ? '1' : '0')
}

function showToast(event) {
  event.preventDefault()
  const detail = event.detail || {}
  const id = ++toastSeed
  toasts.value = toasts.value.concat({
    id,
    message: detail.message || '操作成功',
    type: detail.type || 'success'
  })
  window.setTimeout(() => {
    toasts.value = toasts.value.filter((toast) => toast.id !== id)
  }, 2200)
}

function isMenuActive(path) {
  const current = normalizeActivePath(route.path)
  const target = normalizeActivePath(path)
  if (!target) return false
  if (target === '/admin') return current === '/admin'
  return current === target || current.startsWith(`${target}/`)
}

function normalizeActivePath(path = '') {
  const normalized = String(path).split('?')[0].replace(/\/$/, '')
  return normalized || '/'
}

function shortName(name) {
  return String(name || '').trim().slice(0, 1)
}

function normalizeMenus(items) {
  const result = []
  const usedPaths = new Set()
  for (const item of items) {
    let children = normalizeMenus(item.children || item.child || [])
    const path = mapLegacyAdminPath(item.url)
    if (isWechatMenuGroup(item)) {
      if (!children.some((child) => child.path === '/admin/wx-keywords')) {
        children = [{ key: `${item.id}-wx-keywords`, name: '关键词回复', path: '/admin/wx-keywords' }].concat(children)
      }
      result.push({ key: item.id, name: item.name, path: '', children })
      continue
    }
    if (path === '/admin/admins' && String(item.name || '').includes('用户管理')) {
      usedPaths.add('/admin/admins')
      usedPaths.add('/admin/users')
      result.push({ key: `${item.id}-admins`, name: '管理员列表', path: '/admin/admins' })
      result.push({ key: `${item.id}-users`, name: '用户列表', path: '/admin/users' })
      continue
    }
    if (children.length > 0) {
      result.push({ key: item.id, name: item.name, path, children })
    } else if (path && !usedPaths.has(path)) {
      usedPaths.add(path)
      result.push({ key: item.id, name: item.name, path })
    }
  }
  return result
}

function mapLegacyAdminPath(url = '') {
  const normalized = String(url).replace(/\/$/, '')
  if (isLegacyActionPath(normalized)) return ''
  const exact = {
    '/admin': '/admin',
    '/admin/index': '/admin',
    '/admin/index/main': '/admin',
    '/admin/article': '/admin/articles',
    '/admin/article/index': '/admin/articles',
    '/admin/category/index': '/admin/categories',
    '/admin/tag/index': '/admin/tags',
    '/admin/comment/index': '/admin/comments',
    '/admin/chat/index': '/admin/chats',
    '/admin/nav/index': '/admin/navs',
    '/admin/friendLinks/index': '/admin/friend-links',
    '/admin/wechat': '',
    '/admin/weChat': '',
    '/admin/weChat/keyword/index': '/admin/wx-keywords',
    '/admin/admin/index': '/admin/admins',
    '/admin/user/index': '/admin/users',
    '/admin/role/index': '/admin/roles',
    '/admin/permission/index': '/admin/permissions',
    '/admin/systemConfig/basal': '/admin/system-configs'
  }
  if (exact[normalized]) return exact[normalized]
  if (normalized.includes('/article')) return '/admin/articles'
  if (normalized.includes('/category')) return '/admin/categories'
  if (normalized.includes('/tag')) return '/admin/tags'
  if (normalized.includes('/comment')) return '/admin/comments'
  if (normalized.toLowerCase().includes('/wechat') && !normalized.includes('/weChat/keyword')) return ''
  if (normalized.includes('/chat')) return '/admin/chats'
  if (normalized.includes('/nav')) return '/admin/navs'
  if (normalized.includes('/friendLinks')) return '/admin/friend-links'
  if (normalized.includes('/weChat/keyword')) return '/admin/wx-keywords'
  if (normalized.includes('/admin/admin')) return '/admin/admins'
  if (normalized.includes('/admin/user')) return '/admin/users'
  if (normalized.includes('/role')) return '/admin/roles'
  if (normalized.includes('/permission')) return '/admin/permissions'
  if (normalized.includes('/systemConfig')) return '/admin/system-configs'
  return ''
}

function isWechatMenuGroup(item) {
  const name = String(item.name || '').trim()
  const url = String(item.url || '').replace(/\/$/, '').toLowerCase()
  return name === '微信配置' || url === '/admin/wechat'
}

function isLegacyActionPath(path) {
  return /\/(show|create|store|edit|update|destroy|uploadImage|replace)$/.test(path)
}

function fallbackMenus() {
  return [
    { key: 'dashboard', name: '控制台', path: '/admin' },
    {
      key: 'content',
      name: '内容管理',
      children: [
        { key: 'articles', name: '文章管理', path: '/admin/articles' },
        { key: 'comments', name: '评论管理', path: '/admin/comments' },
        { key: 'categories', name: '分类管理', path: '/admin/categories' },
        { key: 'tags', name: '标签管理', path: '/admin/tags' },
        { key: 'navs', name: '导航管理', path: '/admin/navs' },
        { key: 'friend-links', name: '友情链接', path: '/admin/friend-links' },
        { key: 'chats', name: '闲言碎语', path: '/admin/chats' },
        {
          key: 'wechat',
          name: '微信配置',
          children: [{ key: 'wx-keywords', name: '关键词回复', path: '/admin/wx-keywords' }]
        }
      ]
    },
    {
      key: 'system',
      name: '系统管理',
      children: [
        { key: 'admins', name: '管理员列表', path: '/admin/admins' },
        { key: 'users', name: '用户列表', path: '/admin/users' },
        { key: 'roles', name: '角色管理', path: '/admin/roles' },
        { key: 'permissions', name: '权限管理', path: '/admin/permissions' },
        { key: 'system-configs', name: '系统配置', path: '/admin/system-configs' }
      ]
    }
  ]
}

onMounted(() => {
  window.addEventListener('admin:toast', showToast)
  ensureOpenGroups()
  auth.fetchProfile().catch(() => {})
  loadMenus()
})

onBeforeUnmount(() => {
  window.removeEventListener('admin:toast', showToast)
})
</script>
