export function notifyAdminSuccess(message) {
  notifyAdminToast(message, 'success')
}

export function notifyAdminToast(message, type = 'success') {
  if (!message || typeof window === 'undefined') return
  renderFallbackToast(message, type)
}

function renderFallbackToast(message, type) {
  const wrap = document.querySelector('.admin-toast-wrap') || document.createElement('div')
  if (!wrap.classList.contains('admin-toast-wrap')) {
    wrap.className = 'admin-toast-wrap'
    wrap.setAttribute('aria-live', 'polite')
    document.body.appendChild(wrap)
  }

  const toast = document.createElement('div')
  toast.className = `admin-toast ${type || 'success'}`
  toast.innerHTML = `<span class="admin-toast-icon">✓</span><span></span>`
  toast.lastElementChild.textContent = message
  wrap.appendChild(toast)
  window.setTimeout(() => {
    toast.remove()
    if (!wrap.childElementCount) wrap.remove()
  }, 2200)
}
