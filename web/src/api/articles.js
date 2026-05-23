import http from './http'

export function getArticles(params = {}) {
  return http.get('/admin/articles', { params })
}

export function getAdminArticle(id) {
  return http.get(`/admin/articles/${id}`)
}

export function createArticle(payload) {
  return http.post('/admin/articles', payload)
}

export function updateArticle(id, payload) {
  return http.put(`/admin/articles/${id}`, payload)
}

export function deleteArticle(id) {
  return http.delete(`/admin/articles/${id}`)
}

export function restoreArticle(id) {
  return http.post(`/admin/articles/${id}/restore`)
}

export function forceDeleteArticle(id) {
  return http.delete(`/admin/articles/${id}/force`)
}

export function publishArticleToBaijiahao(id, payload = {}) {
  return http.post(`/admin/articles/${id}/baijiahao`, payload)
}

export function replaceArticles(payload) {
  return http.post('/admin/articles/replace', payload)
}

export function uploadArticleImage(file) {
  const data = new FormData()
  data.append('file', file)
  return http.post('/admin/upload/image', data, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}
