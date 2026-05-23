import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'
const http = axios.create({
  baseURL,
  timeout: 15000
})

const rawHttp = axios.create({
  baseURL,
  timeout: 15000
})

let adminRefreshPromise = null

http.interceptors.request.use((config) => {
  const auth = useAuthStore()
  const url = String(config.url || '')
  if (url.startsWith('/admin') && auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const data = response.data
    if (data && typeof data === 'object' && 'code' in data && data.code !== 0) {
      return Promise.reject(new Error(data.msg || '请求失败'))
    }
    return data
  },
  async (error) => {
    const status = error.response?.status
    const url = String(error.config?.url || '')
    const isAdminRequest = url.startsWith('/admin')
    const isRefreshRequest = url === '/admin/auth/refresh'
    if (status === 401 && isAdminRequest && !isRefreshRequest && !error.config?._retry) {
      const auth = useAuthStore()
      if (auth.refreshToken) {
        try {
          error.config._retry = true
          const result = await refreshAdminToken(auth)
          error.config.headers = error.config.headers || {}
          error.config.headers.Authorization = `Bearer ${result.token}`
          return http(error.config)
        } catch {
          auth.logout()
        }
      } else {
        auth.logout()
      }
      const current = `${window.location.pathname}${window.location.search}`
      if (!window.location.pathname.startsWith('/admin/login')) {
        const redirect = encodeURIComponent(current)
        window.location.replace(`/admin/login?redirect=${redirect}`)
      }
    } else if (status === 401 && isAdminRequest) {
      const auth = useAuthStore()
      auth.logout()
      const current = `${window.location.pathname}${window.location.search}`
      if (!window.location.pathname.startsWith('/admin/login')) {
        const redirect = encodeURIComponent(current)
        window.location.replace(`/admin/login?redirect=${redirect}`)
      }
    }
    const message = error.response?.data?.msg || error.message || '请求失败'
    return Promise.reject(new Error(message))
  }
)

async function refreshAdminToken(auth) {
  if (!adminRefreshPromise) {
    adminRefreshPromise = rawHttp
      .post('/admin/auth/refresh', { refresh_token: auth.refreshToken })
      .then((response) => {
        const body = response.data
        if (body?.code !== 0) throw new Error(body?.msg || '登录状态已失效')
        const data = body.data || {}
        auth.persistSession(data.token, data.refresh_token || '', data.admin, data.permissions || [])
        return data
      })
      .finally(() => {
        adminRefreshPromise = null
      })
  }
  return adminRefreshPromise
}

export default http
