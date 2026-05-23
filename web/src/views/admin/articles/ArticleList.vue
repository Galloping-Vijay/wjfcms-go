<template>
  <form class="toolbar search-toolbar" @submit.prevent="search">
    <label class="search-field">
      <span>标题</span>
      <input v-model.trim="filters.title" placeholder="请输入" @keyup.enter="search" />
    </label>
    <label class="search-field">
      <span>作者</span>
      <input v-model.trim="filters.author" placeholder="请输入" @keyup.enter="search" />
    </label>
    <label class="search-field">
      <span>文章分类</span>
      <SearchableSelect v-model="filters.category_id" :options="categoryOptions" placeholder="全部分类" />
    </label>
    <label class="search-field">
      <span>状态</span>
      <SearchableSelect v-model="filters.status" :options="statusOptions" placeholder="全部" />
    </label>
    <label class="search-field">
      <span>是否删除</span>
      <SearchableSelect v-model="filters.delete" :options="deleteOptions" placeholder="正常" />
    </label>
    <button type="submit">搜索</button>
    <button type="button" class="secondary-button" @click="resetSearch">重置</button>
    <RouterLink v-if="canCreate" class="button-link" to="/admin/articles/create">新增文章</RouterLink>
    <button v-if="canReplaceArticles" type="button" class="secondary-button" @click="openReplaceDialog">批量替换</button>
  </form>

  <div v-if="canBatchAction && selectedIds.length" class="batch-toolbar">
    <span>已选 {{ selectedIds.length }} 条</span>
    <button v-if="selectedNormalArticles.length && canDelete" type="button" class="danger-button" @click="confirmBatchAction('delete')">批量删除</button>
    <button v-if="selectedDeletedArticles.length && canRestore" type="button" @click="confirmBatchAction('restore')">批量恢复</button>
    <button v-if="selectedDeletedArticles.length && canForceDelete" type="button" class="danger-button" @click="confirmBatchAction('force')">批量彻底删除</button>
    <button type="button" class="secondary-button" @click="clearSelection">取消选择</button>
  </div>

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th v-if="canBatchAction" class="selection-cell">
            <input type="checkbox" :checked="isPageSelected" :disabled="articles.length === 0" @change="togglePageSelection" />
          </th>
          <th>ID</th>
          <th>标题</th>
          <th>分类</th>
          <th>作者</th>
          <th>点击</th>
          <th>状态</th>
          <th>百家号</th>
          <th>创建时间</th>
          <th v-if="hasActions">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="article in articles" :key="article.id">
          <td v-if="canBatchAction" class="selection-cell">
            <input v-model="selectedIds" type="checkbox" :value="article.id" />
          </td>
          <td>{{ article.id }}</td>
          <td>{{ article.title }}</td>
          <td>{{ article.category?.name || '-' }}</td>
          <td>{{ article.author || '-' }}</td>
          <td>{{ article.click }}</td>
          <td>{{ article.status === 1 ? '已发布' : '待审核' }}</td>
          <td>{{ article.is_baijiahao ? '已推送' : '未推送' }}</td>
          <td>{{ article.created_at }}</td>
          <td v-if="hasActions" class="action-cell">
            <template v-if="isDeleted(article)">
              <button v-if="canRestore" type="button" class="small-button" @click="confirmAction(article, 'restore')">恢复</button>
              <button v-if="canForceDelete" type="button" class="danger-button small-button" @click="confirmAction(article, 'force')">彻底删除</button>
            </template>
            <template v-else>
              <button
                v-if="canPublishBaijiahao && article.status === 1 && !article.is_baijiahao"
                type="button"
                class="small-button"
                @click="confirmAction(article, 'baijiahao')"
              >
                推送百家号
              </button>
              <RouterLink v-if="canEdit" class="small-link" :to="`/admin/articles/${article.id}/edit`">编辑</RouterLink>
              <button v-if="canDelete" type="button" class="danger-button small-button" @click="confirmAction(article, 'delete')">删除</button>
            </template>
          </td>
        </tr>
        <tr v-if="!loading && articles.length === 0">
          <td :colspan="emptyColspan" class="empty-cell">暂无数据</td>
        </tr>
      </tbody>
    </table>
  </div>

  <p v-if="error" class="form-error">{{ error }}</p>
  <AdminPagination
    :page="filters.page"
    :page-size="filters.limit"
    :total="total"
    @change="changePage"
    @update:page-size="changePageSize"
  />

  <AdminConfirmDialog
    :open="confirmDialog.open"
    :title="actionTitle"
    :subtitle="actionSubtitle"
    :message="actionMessage"
    :confirm-text="actionConfirmText"
    :loading-text="actionLoadingText"
    :loading="actionLoading"
    @cancel="closeConfirm"
    @confirm="runConfirmedAction"
  />
  <div v-if="replaceDialog.open" class="modal-mask">
    <form class="modal-panel compact-modal" @submit.prevent="saveReplace">
      <header class="modal-header">
        <strong>批量替换文章内容</strong>
        <button type="button" class="text-button" @click="closeReplaceDialog">关闭</button>
      </header>
      <label>
        查找内容
        <input v-model.trim="replaceDialog.search" placeholder="请输入要查找的内容" />
      </label>
      <label>
        替换为
        <input v-model="replaceDialog.replace" placeholder="请输入替换后的内容，可留空" />
      </label>
      <div class="replace-field-group" aria-label="替换范围">
        <span>替换范围</span>
        <label v-for="field in replaceFields" :key="field.value" class="inline-check">
          <input v-model="replaceDialog.fields" type="checkbox" :value="field.value" />
          {{ field.label }}
        </label>
      </div>
      <p class="muted-text">默认用于替换文章 Markdown 和 HTML 正文，包含回收站文章；旧正文的 HTML 转义内容会自动兼容。</p>
      <p v-if="replaceError" class="form-error">{{ replaceError }}</p>
      <div class="form-actions">
        <button type="submit" :disabled="replaceSaving">{{ replaceSaving ? '替换中...' : '开始替换' }}</button>
        <button type="button" class="secondary-button" @click="closeReplaceDialog">取消</button>
      </div>
    </form>
  </div>
  <AdminNoticeDialog
    :open="notice.open"
    title="操作失败"
    subtitle="接口已返回错误信息。"
    :message="notice.message"
    @close="notice.open = false"
  />
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listResource } from '../../../api/adminResources'
import { deleteArticle, forceDeleteArticle, getArticles, publishArticleToBaijiahao, replaceArticles, restoreArticle } from '../../../api/articles'
import AdminConfirmDialog from '../../../components/admin/AdminConfirmDialog.vue'
import AdminNoticeDialog from '../../../components/admin/AdminNoticeDialog.vue'
import AdminPagination from '../../../components/admin/AdminPagination.vue'
import SearchableSelect from '../../../components/admin/SearchableSelect.vue'
import { useAuthStore } from '../../../stores/auth'
import { notifyAdminSuccess } from '../../../utils/adminToast'

const route = useRoute()
const auth = useAuthStore()
const loading = ref(false)
const actionLoading = ref(false)
const error = ref('')
const articles = ref([])
const categories = ref([])
const total = ref(0)
const selectedIds = ref([])
const confirmDialog = reactive({
  open: false,
  article: null,
  ids: [],
  action: 'delete'
})
const notice = reactive({
  open: false,
  message: ''
})
const replaceDialog = reactive({
  open: false,
  search: '',
  replace: '',
  fields: ['content', 'markdown']
})
const replaceSaving = ref(false)
const replaceError = ref('')
const filters = reactive({
  title: '',
  author: '',
  category_id: route.query.category_id || '',
  status: '',
  delete: '',
  page: 1,
  limit: 15
})
const categoryOptions = computed(() => [
  { label: '全部分类', value: '' },
  ...categories.value.map((category) => ({ label: category.name, value: category.id }))
])
const statusOptions = [
  { label: '全部', value: '' },
  { label: '待审核', value: 0 },
  { label: '已发布', value: 1 }
]
const deleteOptions = [
  { label: '正常', value: '' },
  { label: '回收站', value: 1 },
  { label: '全部含删除', value: 2 }
]
const canCreate = computed(() => auth.can('/admin/article/store'))
const canEdit = computed(() => auth.can('/admin/article/update'))
const canDelete = computed(() => auth.can('/admin/article/destroy'))
const canRestore = computed(() => auth.can('/admin/article/update'))
const canForceDelete = computed(() => auth.can('/admin/article/destroy'))
const canPublishBaijiahao = computed(() => auth.can('/admin/article/update'))
const canReplaceArticles = computed(() => auth.can('/admin/article/replace'))
const hasActions = computed(() => canEdit.value || canDelete.value || canRestore.value || canForceDelete.value || canPublishBaijiahao.value)
const canBatchAction = computed(() => canDelete.value || canRestore.value || canForceDelete.value)
const emptyColspan = computed(() => 8 + (hasActions.value ? 1 : 0) + (canBatchAction.value ? 1 : 0))
const selectedArticles = computed(() => articles.value.filter((article) => selectedIds.value.includes(article.id)))
const selectedNormalArticles = computed(() => selectedArticles.value.filter((article) => !isDeleted(article)))
const selectedDeletedArticles = computed(() => selectedArticles.value.filter((article) => isDeleted(article)))
const isPageSelected = computed(() => articles.value.length > 0 && articles.value.every((article) => selectedIds.value.includes(article.id)))
const actionTitle = computed(() => actionText('title'))
const actionSubtitle = computed(() => {
  if (confirmDialog.action === 'baijiahao') return '推送成功后会标记为已推送，避免重复提交。'
  if (confirmDialog.action === 'delete') return '删除前请确认文章没有关联评论。'
  if (confirmDialog.action === 'force') return '彻底删除后数据不可恢复，请谨慎操作。'
  return '恢复后该数据会回到正常列表。'
})
const actionMessage = computed(() => {
  if (confirmDialog.ids.length) return `确认${actionText('verb')}选中的 ${confirmDialog.ids.length} 篇文章？`
  return `确认${actionText('verb')}文章「${confirmDialog.article?.title || ''}」？`
})
const actionConfirmText = computed(() => `确认${actionText('verb')}`)
const actionLoadingText = computed(() => `${actionText('verb')}中...`)
const replaceFields = [
  { label: 'HTML 正文', value: 'content' },
  { label: 'Markdown', value: 'markdown' },
  { label: '标题', value: 'title' },
  { label: '简介', value: 'description' }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = {}
    for (const [key, value] of Object.entries(filters)) {
      if (value !== '' && value !== undefined && value !== null) params[key] = value
    }
    const res = await getArticles(params)
    articles.value = res.data || []
    selectedIds.value = selectedIds.value.filter((id) => articles.value.some((article) => article.id === id))
    total.value = res.count || 0
    const maxPage = Math.max(1, Math.ceil(total.value / filters.limit))
    if (filters.page > maxPage) {
      filters.page = maxPage
      await load()
    }
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  const res = await listResource('categories', { page: 1, limit: 1000 })
  categories.value = res.data || []
}

async function search() {
  filters.page = 1
  await load()
}

async function resetSearch() {
  filters.title = ''
  filters.author = ''
  filters.category_id = ''
  filters.status = ''
  filters.delete = ''
  filters.page = 1
  await load()
}

async function changePage(page) {
  filters.page = page
  await load()
}

async function changePageSize(pageSize) {
  filters.limit = pageSize
}

function confirmAction(article, action) {
  error.value = ''
  confirmDialog.article = article
  confirmDialog.ids = []
  confirmDialog.action = action
  confirmDialog.open = true
}

function confirmBatchAction(action) {
  const candidates = action === 'delete' ? selectedNormalArticles.value : selectedDeletedArticles.value
  const ids = candidates.map((article) => article.id)
  if (!ids.length) return
  error.value = ''
  confirmDialog.article = null
  confirmDialog.ids = ids
  confirmDialog.action = action
  confirmDialog.open = true
}

function closeConfirm() {
  if (actionLoading.value) return
  confirmDialog.open = false
  confirmDialog.article = null
  confirmDialog.ids = []
}

function openReplaceDialog() {
  replaceError.value = ''
  replaceDialog.search = ''
  replaceDialog.replace = ''
  replaceDialog.fields = ['content', 'markdown']
  replaceDialog.open = true
}

function closeReplaceDialog() {
  if (replaceSaving.value) return
  replaceDialog.open = false
  replaceDialog.search = ''
  replaceDialog.replace = ''
  replaceDialog.fields = ['content', 'markdown']
}

async function saveReplace() {
  replaceError.value = ''
  const search = replaceDialog.search.trim()
  if (!search) {
    replaceError.value = '请填写要查找的内容'
    return
  }
  if (!replaceDialog.fields.length) {
    replaceError.value = '请选择替换范围'
    return
  }
  replaceSaving.value = true
  try {
    const res = await replaceArticles({
      search,
      replace: replaceDialog.replace,
      fields: replaceDialog.fields
    })
    const changed = res.data?.changed ?? 0
    notifyAdminSuccess(`文章批量替换完成，更新 ${changed} 条`)
    closeReplaceDialog()
    await load()
  } catch (err) {
    replaceError.value = err.message || '批量替换失败'
  } finally {
    replaceSaving.value = false
  }
}

async function runConfirmedAction() {
  const ids = confirmDialog.ids.length ? [...confirmDialog.ids] : confirmDialog.article ? [confirmDialog.article.id] : []
  if (!ids.length) return
  actionLoading.value = true
  error.value = ''
  try {
    for (const id of ids) await runArticleAction(id, confirmDialog.action)
    const successText = actionText('verb')
    selectedIds.value = selectedIds.value.filter((id) => !ids.includes(id))
    actionLoading.value = false
    closeConfirm()
    notifyAdminSuccess(`文章${successText}成功`)
    await load()
  } catch (err) {
    actionLoading.value = false
    closeConfirm()
    notice.message = err.message || `${actionText('verb')}失败`
    notice.open = true
  }
}

async function runArticleAction(id, action) {
  if (action === 'baijiahao') {
    await publishArticleToBaijiahao(id, { original: 1 })
  } else if (action === 'restore') {
    await restoreArticle(id)
  } else if (action === 'force') {
    await forceDeleteArticle(id)
  } else {
    await deleteArticle(id)
  }
}

function togglePageSelection(event) {
  if (event.target.checked) {
    selectedIds.value = Array.from(new Set(selectedIds.value.concat(articles.value.map((article) => article.id))))
  } else {
    const pageIds = new Set(articles.value.map((article) => article.id))
    selectedIds.value = selectedIds.value.filter((id) => !pageIds.has(id))
  }
}

function clearSelection() {
  selectedIds.value = []
}

function isDeleted(row) {
  return Boolean(row.deleted_at && row.deleted_at.Valid !== false)
}

function actionText(type) {
  const map = {
    delete: { title: '删除文章', verb: '删除' },
    restore: { title: '恢复文章', verb: '恢复' },
    force: { title: '彻底删除文章', verb: '彻底删除' },
    baijiahao: { title: '推送百家号', verb: '推送' }
  }
  return map[confirmDialog.action]?.[type] || map.delete[type]
}

watch(
  () => route.query.category_id,
  async (categoryID) => {
    filters.category_id = categoryID || ''
    filters.page = 1
    await load()
  }
)

onMounted(async () => {
  await loadCategories()
  await load()
})
</script>
