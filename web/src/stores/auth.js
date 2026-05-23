import { defineStore } from 'pinia'
import { getAdminProfile, loginAdmin, logoutAdmin, updateAdminPassword, updateAdminProfile } from '../api/auth'

const legacyTokenKey = 'wjfcms_admin_token'
const legacyRefreshTokenKey = 'wjfcms_admin_refresh_token'
const legacyAdminKey = 'wjfcms_admin'
const legacyPermissionKey = 'wjfcms_admin_permissions'

const tokenKey = 'wjfcm_go_admin_token'
const refreshTokenKey = 'wjfcm_go_admin_refresh_token'
const adminKey = 'wjfcm_go_admin'
const permissionKey = 'wjfcm_go_admin_permissions'

function takeStorageValue(key, legacyKey, fallback = '') {
  const current = localStorage.getItem(key)
  if (current !== null) return current
  const legacy = localStorage.getItem(legacyKey)
  if (legacy !== null) {
    localStorage.setItem(key, legacy)
    localStorage.removeItem(legacyKey)
    return legacy
  }
  return fallback
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: takeStorageValue(tokenKey, legacyTokenKey),
    refreshToken: takeStorageValue(refreshTokenKey, legacyRefreshTokenKey),
    admin: JSON.parse(takeStorageValue(adminKey, legacyAdminKey, 'null')),
    permissions: JSON.parse(takeStorageValue(permissionKey, legacyPermissionKey, '[]'))
  }),
  actions: {
    persistSession(token, refreshToken, admin, permissions = []) {
      this.token = token
      this.refreshToken = refreshToken
      localStorage.setItem(tokenKey, token)
      localStorage.setItem(refreshTokenKey, refreshToken)
      this.persistProfile(admin, permissions)
    },
    persistProfile(admin, permissions = []) {
      this.admin = admin
      this.permissions = permissions
      localStorage.setItem(adminKey, JSON.stringify(admin))
      localStorage.setItem(permissionKey, JSON.stringify(permissions))
    },
    async login(account, password) {
      const res = await loginAdmin({ account, password })
      this.persistSession(res.data.token, res.data.refresh_token || '', res.data.admin, res.data.permissions || [])
    },
    async fetchProfile() {
      const res = await getAdminProfile()
      if (res.data?.admin) {
        this.persistProfile(res.data.admin, res.data.permissions || [])
      } else {
        this.persistProfile(res.data, this.permissions)
      }
    },
    async updateProfile(payload) {
      const res = await updateAdminProfile(payload)
      this.persistProfile(res.data, this.permissions)
      return res.data
    },
    async updatePassword(payload) {
      return updateAdminPassword(payload)
    },
    can(...urls) {
      if (this.admin?.id === 1) return true
      if (!urls.length) return true
      const owned = new Set((this.permissions || []).map((url) => String(url).replace(/\/$/, '')))
      return urls.some((url) => owned.has(String(url).replace(/\/$/, '')))
    },
    async logout(remote = false) {
      if (remote && this.token) {
        try {
          await logoutAdmin()
        } catch {
          // 本地仍然清理登录态，避免退出按钮被网络问题卡住。
        }
      }
      this.token = ''
      this.refreshToken = ''
      this.admin = null
      this.permissions = []
      localStorage.removeItem(tokenKey)
      localStorage.removeItem(refreshTokenKey)
      localStorage.removeItem(adminKey)
      localStorage.removeItem(permissionKey)
      localStorage.removeItem(legacyTokenKey)
      localStorage.removeItem(legacyRefreshTokenKey)
      localStorage.removeItem(legacyAdminKey)
      localStorage.removeItem(legacyPermissionKey)
    }
  }
})
