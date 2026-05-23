import http from './http'

export function listResource(resource, params = {}) {
  return http.get(`/admin/${resource}`, { params })
}

export function createResource(resource, payload) {
  return http.post(`/admin/${resource}`, payload)
}

export function updateResource(resource, id, payload) {
  return http.put(`/admin/${resource}/${id}`, payload)
}

export function updateResourcePassword(resource, id, payload) {
  return http.put(`/admin/${resource}/${id}/password`, payload)
}

export function deleteResource(resource, id) {
  return http.delete(`/admin/${resource}/${id}`)
}

export function restoreResource(resource, id) {
  return http.post(`/admin/${resource}/${id}/restore`)
}

export function forceDeleteResource(resource, id) {
  return http.delete(`/admin/${resource}/${id}/force`)
}

export function getMenus() {
  return http.get('/admin/menus')
}

export function getRole(id) {
  return http.get(`/admin/roles/${id}`)
}

export function getRolePermissionTree(params = {}) {
  return http.get('/admin/roles/permission-tree', { params })
}

export function uploadAdminImage(file) {
  const data = new FormData()
  data.append('file', file)
  return http.post('/admin/upload/image', data, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export function replaceComments(payload) {
  return http.post('/admin/comments/replace', payload)
}
