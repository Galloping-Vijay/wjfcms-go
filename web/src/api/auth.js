import http from './http'

export function loginAdmin(payload) {
  return http.post('/admin/auth/login', payload)
}

export function getAdminProfile() {
  return http.get('/admin/profile')
}

export function updateAdminProfile(payload) {
  return http.put('/admin/profile', payload)
}

export function updateAdminPassword(payload) {
  return http.put('/admin/password', payload)
}

export function logoutAdmin() {
  return http.post('/admin/auth/logout')
}
