<template>
  <form class="article-form" @submit.prevent="submit">
    <section class="form-section">
      <label>
        标题
        <input v-model.trim="form.title" type="text" />
      </label>

      <div class="form-grid">
        <label>
          分类
          <select v-model.number="form.category_id">
            <option :value="0">请选择分类</option>
            <option v-for="category in categories" :key="category.id" :value="category.id">
              {{ category.name }}
            </option>
          </select>
        </label>

        <label>
          作者
          <input v-model.trim="form.author" type="text" />
        </label>

        <label>
          状态
          <select v-model.number="form.status">
            <option :value="0">待审核</option>
            <option :value="1">已发布</option>
          </select>
        </label>

        <label>
          置顶
          <select v-model="form.is_top">
            <option :value="false">否</option>
            <option :value="true">是</option>
          </select>
        </label>
        
        <label>
          点击数
          <input v-model="form.click" type="number" />
        </label>
      </div>

      <div class="tag-picker">
        <span class="field-label">标签</span>
        <label v-for="tag in tags" :key="tag.id" class="tag-check">
          <input v-model="selectedTags" type="checkbox" :value="tag.name" @change="syncKeywordsFromTags" />
          <span>{{ tag.name }}</span>
        </label>
      </div>

      <label>
        描述
        <textarea v-model.trim="form.description" rows="3"></textarea>
      </label>

      <label>
        封面图
        <div class="cover-row">
          <input v-model.trim="form.cover" type="text" />
          <input ref="coverInput" class="file-input" type="file" accept="image/*" @change="uploadCover" />
          <button type="button" @click="coverInput?.click()">上传</button>
        </div>
      </label>
      <img v-if="form.cover" class="cover-preview" :src="form.cover" alt="封面预览" />
    </section>

    <section class="form-section article-editor-section">
      <div class="editor-shell">
        <div class="editor-head">
          <div>
            <strong>文章内容</strong>
            <span>Markdown 实时预览</span>
          </div>
          <div class="editor-toolbar">
            <input ref="bodyImageInput" class="file-input" type="file" accept="image/*" @change="uploadBodyImage" />
            <button type="button" title="加粗" @click="wrapSelection('**', '**', '加粗文字')">B</button>
            <button type="button" title="行内代码" @click="wrapSelection('`', '`', 'code')">`</button>
            <button type="button" title="引用" @click="prefixSelection('> ')">引用</button>
            <button type="button" title="无序列表" @click="prefixSelection('- ')">列表</button>
            <button type="button" title="代码块" @click="insertCodeBlock">代码块</button>
            <button type="button" title="链接" @click="insertMarkdown('[链接文字](https://)')">链接</button>
            <button type="button" title="插入图片" @click="bodyImageInput?.click()">图片</button>
            <button type="button" class="secondary-button" title="同步 HTML" @click="syncMarkdownToHtml">同步</button>
          </div>
        </div>
        <div class="editor-panels">
          <div class="editor-pane">
            <div class="pane-title">Markdown</div>
            <textarea
              ref="markdownInput"
              v-model="form.markdown"
              class="markdown-editor"
              spellcheck="false"
              @input="syncMarkdownToHtml"
              @paste="handleMarkdownPaste"
            ></textarea>
          </div>
          <div class="editor-pane preview-pane">
            <div class="pane-title">预览</div>
            <div class="article-content" v-html="form.content"></div>
          </div>
        </div>
      </div>
      <details class="html-compat-panel">
        <summary>自动生成的 HTML（兼容旧数据）</summary>
        <label>
          HTML 内容
          <textarea v-model="form.content" class="html-editor" rows="8"></textarea>
        </label>
      </details>
    </section>

    <p v-if="error" class="form-error">{{ error }}</p>
    <div class="form-actions">
      <button type="submit" :disabled="loading">{{ loading ? '保存中...' : '保存' }}</button>
      <RouterLink class="button-link muted" to="/admin/articles">返回列表</RouterLink>
    </div>
  </form>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listResource } from '../../../api/adminResources'
import { createArticle, getAdminArticle, updateArticle, uploadArticleImage } from '../../../api/articles'
import { notifyAdminSuccess } from '../../../utils/adminToast'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => Boolean(route.params.id))
const loading = ref(false)
const error = ref('')
const categories = ref([])
const tags = ref([])
const selectedTags = ref([])
const coverInput = ref(null)
const bodyImageInput = ref(null)
const markdownInput = ref(null)

const form = reactive({
  category_id: 0,
  title: '',
  author: '',
  content: '',
  markdown: '',
  description: '',
  keywords: '',
  cover: '',
  is_top: false,
  click: 90,
  status: 0,
  is_baijiahao: false
})

async function loadOptions() {
  const [categoryRes, tagRes] = await Promise.all([
    listResource('categories'),
    listResource('tags')
  ])
  categories.value = categoryRes.data || []
  tags.value = tagRes.data || []
  syncSelectedTagsFromKeywords()
}

async function loadArticle() {
  if (!isEdit.value) return
  const res = await getAdminArticle(route.params.id)
  Object.assign(form, {
    category_id: res.data.category_id || 0,
    title: res.data.title || '',
    author: res.data.author || '',
    content: res.data.content || '',
    markdown: res.data.markdown || '',
    description: res.data.description || '',
    keywords: res.data.keywords || '',
    cover: res.data.cover || '',
    is_top: Boolean(res.data.is_top),
    click: res.data.click ?? 90,
    status: res.data.status || 0,
    is_baijiahao: Boolean(res.data.is_baijiahao)
  })
  syncSelectedTagsFromKeywords()
}

function syncKeywordsFromTags() {
  form.keywords = selectedTags.value.join(',')
}

function syncSelectedTagsFromKeywords() {
  selectedTags.value = String(form.keywords || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

async function uploadCover(event) {
  const file = event.target.files?.[0]
  if (!file) return
  error.value = ''
  try {
    const res = await uploadArticleImage(file)
    form.cover = res.data.src
    notifyAdminSuccess('封面上传成功')
  } catch (err) {
    error.value = err.message
  } finally {
    event.target.value = ''
  }
}

async function uploadBodyImage(event) {
  const file = event.target.files?.[0]
  if (!file) return
  try {
    await uploadBodyImageFile(file)
  } catch (err) {
    error.value = err.message
  } finally {
    event.target.value = ''
  }
}

async function uploadBodyImageFile(file) {
  error.value = ''
  const res = await uploadArticleImage(file)
  insertMarkdown(`![${file.name || 'image'}](${res.data.src})`)
  notifyAdminSuccess('图片上传成功')
}

async function handleMarkdownPaste(event) {
  const file = [...(event.clipboardData?.items || [])]
    .find((item) => item.kind === 'file' && item.type.startsWith('image/'))
    ?.getAsFile()
  if (!file) return
  event.preventDefault()
  try {
    await uploadBodyImageFile(file)
  } catch (err) {
    error.value = err.message
  }
}

function insertMarkdown(text) {
  const target = markdownInput.value
  if (!target) {
    form.markdown += `\n${text}\n`
    syncMarkdownToHtml()
    return
  }
  const start = target.selectionStart || 0
  const end = target.selectionEnd || 0
  form.markdown = `${form.markdown.slice(0, start)}${text}${form.markdown.slice(end)}`
  syncMarkdownToHtml()
  requestAnimationFrame(() => {
    target.focus()
    target.setSelectionRange(start + text.length, start + text.length)
  })
}

function wrapSelection(before, after, fallback) {
  const target = markdownInput.value
  if (!target) {
    insertMarkdown(`${before}${fallback}${after}`)
    return
  }
  const start = target.selectionStart || 0
  const end = target.selectionEnd || 0
  const selected = form.markdown.slice(start, end) || fallback
  form.markdown = `${form.markdown.slice(0, start)}${before}${selected}${after}${form.markdown.slice(end)}`
  syncMarkdownToHtml()
  requestAnimationFrame(() => {
    target.focus()
    target.setSelectionRange(start + before.length, start + before.length + selected.length)
  })
}

function insertCodeBlock() {
  wrapSelection('```go\n', '\n```', 'fmt.Println("hello")')
}

function prefixSelection(prefix) {
  const target = markdownInput.value
  if (!target) {
    insertMarkdown(`${prefix}内容`)
    return
  }
  const start = target.selectionStart || 0
  const end = target.selectionEnd || 0
  const selected = form.markdown.slice(start, end) || '内容'
  const next = selected.split('\n').map((line) => `${prefix}${line}`).join('\n')
  form.markdown = `${form.markdown.slice(0, start)}${next}${form.markdown.slice(end)}`
  syncMarkdownToHtml()
  requestAnimationFrame(() => {
    target.focus()
    target.setSelectionRange(start, start + next.length)
  })
}

function syncMarkdownToHtml() {
  if (form.markdown.trim()) {
    form.content = renderMarkdown(form.markdown)
  }
}

function renderMarkdown(markdown) {
  const output = []
  const paragraph = []
  const list = { type: '', items: [] }
  let codeLines = null

  const flushParagraph = () => {
    if (!paragraph.length) return
    output.push(`<p>${paragraph.map(inlineMarkdown).join('<br>')}</p>`)
    paragraph.length = 0
  }
  const flushList = () => {
    if (!list.items.length) return
    output.push(`<${list.type}>${list.items.map((item) => `<li>${inlineMarkdown(item)}</li>`).join('')}</${list.type}>`)
    list.type = ''
    list.items = []
  }

  String(markdown).replace(/\r\n/g, '\n').split('\n').forEach((line) => {
    if (codeLines) {
      if (/^```/.test(line.trim())) {
        output.push(`<pre><code>${codeLines.join('\n')}</code></pre>`)
        codeLines = null
      } else {
        codeLines.push(escapeHtml(line))
      }
      return
    }

    if (/^```/.test(line.trim())) {
      flushParagraph()
      flushList()
      codeLines = []
      return
    }

    if (!line.trim()) {
      flushParagraph()
      flushList()
      return
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      flushParagraph()
      flushList()
      const level = heading[1].length
      output.push(`<h${level}>${inlineMarkdown(heading[2])}</h${level}>`)
      return
    }

    const unordered = line.match(/^[-*]\s+(.+)$/)
    const ordered = line.match(/^\d+\.\s+(.+)$/)
    if (unordered || ordered) {
      flushParagraph()
      const type = unordered ? 'ul' : 'ol'
      if (list.type && list.type !== type) flushList()
      list.type = type
      list.items.push(unordered ? unordered[1] : ordered[1])
      return
    }

    const quote = line.match(/^>\s?(.+)$/)
    if (quote) {
      flushParagraph()
      flushList()
      output.push(`<blockquote>${inlineMarkdown(quote[1])}</blockquote>`)
      return
    }

    paragraph.push(line)
  })

  flushParagraph()
  flushList()
  if (codeLines) output.push(`<pre><code>${codeLines.join('\n')}</code></pre>`)
  return output.join('\n')
}

function inlineMarkdown(text) {
  return escapeHtml(text)
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1">')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
    .replace(/~~(.+?)~~/g, '<del>$1</del>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

async function submit() {
  error.value = ''
  if (!form.title) {
    error.value = '请填写文章标题'
    return
  }
  syncKeywordsFromTags()
  syncMarkdownToHtml()
  loading.value = true
  try {
    if (isEdit.value) {
      await updateArticle(route.params.id, form)
      notifyAdminSuccess('文章保存成功')
    } else {
      await createArticle(form)
      notifyAdminSuccess('文章新增成功')
    }
    router.push('/admin/articles')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadOptions()
  await loadArticle()
})
</script>
