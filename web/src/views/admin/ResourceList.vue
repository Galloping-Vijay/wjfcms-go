<template>
  <div v-if="isSystemConfig" class="config-tabs" role="tablist" aria-label="系统配置分组">
    <button
      v-for="group in systemConfigGroups"
      :key="group.value"
      type="button"
      :class="{ active: filters.group === group.value }"
      @click="setSystemConfigGroup(group.value)"
    >
      {{ group.label }}
    </button>
  </div>

  <form class="toolbar search-toolbar" @submit.prevent="search">
    <label v-for="field in searchFields" :key="field.key" class="search-field">
      <span>{{ field.label }}</span>
      <SearchableSelect
        v-if="field.type === 'select'"
        v-model="filters[field.key]"
        :options="fieldOptions(field)"
        placeholder="全部"
      />
      <input v-else v-model.trim="filters[field.key]" :placeholder="field.placeholder || '请输入'" @keyup.enter="search" />
    </label>
    <button type="submit">搜索</button>
    <button type="button" class="secondary-button" @click="resetSearch">重置</button>
    <button v-if="canCreate" type="button" @click="openCreate()">新增</button>
    <button v-if="canReplaceComments" type="button" class="secondary-button" @click="openReplaceDialog">批量替换</button>
  </form>

  <div v-if="canBatchAction && selectedIds.length" class="batch-toolbar">
    <span>已选 {{ selectedIds.length }} 条</span>
    <button v-if="selectedNormalRows.length && canDelete" type="button" class="danger-button" @click="confirmBatchAction('delete')">批量删除</button>
    <button v-if="selectedDeletedRows.length && canRestore" type="button" @click="confirmBatchAction('restore')">批量恢复</button>
    <button v-if="selectedDeletedRows.length && canForceDelete" type="button" class="danger-button" @click="confirmBatchAction('force')">批量彻底删除</button>
    <button type="button" class="secondary-button" @click="clearSelection">取消选择</button>
  </div>

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th v-if="canBatchAction" class="selection-cell">
            <input type="checkbox" :checked="isPageSelected" :disabled="displayRows.length === 0" @change="togglePageSelection" />
          </th>
          <th v-for="column in config.columns" :key="column.key">{{ column.label }}</th>
          <th v-if="hasActions">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in displayRows" :key="row.id">
          <td v-if="canBatchAction" class="selection-cell">
            <input v-model="selectedIds" type="checkbox" :value="row.id" />
          </td>
          <td v-for="column in config.columns" :key="column.key">
            <span v-if="column.tree" class="tree-cell" :style="{ paddingLeft: `${(row._depth || 0) * 22}px` }">
              <button
                v-if="row.children?.length"
                type="button"
                class="tree-toggle"
                :title="isTreeCollapsed(row.id) ? '展开' : '折叠'"
                @click="toggleTreeNode(row.id)"
              >
                {{ isTreeCollapsed(row.id) ? '▸' : '▾' }}
              </button>
              <span v-else class="tree-marker">·</span>
              {{ formatValue(row, column) }}
            </span>
            <button
              v-else-if="column.toggle"
              type="button"
              class="status-toggle"
              :class="{ active: Number(row[column.key]) === 1 }"
              :disabled="!canEdit || toggleLoadingKey === toggleKey(row, column)"
              @click="toggleRowValue(row, column)"
            >
              {{ toggleLabel(row, column) }}
            </button>
            <input
              v-else-if="column.inlineEdit === 'number'"
              class="inline-number-input"
              type="number"
              :value="row[column.key]"
              :disabled="!canEdit || isDeleted(row) || inlineLoadingKey === inlineKey(row, column)"
              @change="updateInlineValue(row, column, $event)"
            />
            <template v-else>{{ formatValue(row, column) }}</template>
          </td>
          <td v-if="hasActions" class="action-cell">
            <template v-if="isDeleted(row)">
              <button v-if="canRestore" type="button" class="small-button" @click="confirmAction(row, 'restore')">恢复</button>
              <button v-if="canForceDelete" type="button" class="danger-button small-button" @click="confirmAction(row, 'force')">彻底删除</button>
            </template>
            <template v-else>
              <RouterLink v-if="resource === 'categories'" class="small-link" :to="{ path: '/admin/articles', query: { category_id: row.id } }">文章</RouterLink>
              <button v-if="canCreateChild" type="button" class="small-button" @click="openCreate(row)">添加子级</button>
              <button v-if="canPassword" type="button" class="small-button" @click="openPasswordDialog(row)">密码</button>
              <button v-if="canEdit" type="button" class="small-button" @click="openEdit(row)">编辑</button>
              <button v-if="canDelete" type="button" class="danger-button small-button" @click="confirmAction(row, 'delete')">删除</button>
            </template>
          </td>
        </tr>
        <tr v-if="!loading && rows.length === 0">
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

  <div v-if="dialog.open" class="modal-mask">
    <form class="modal-panel" @submit.prevent="save">
      <header class="modal-header">
        <strong>{{ dialog.mode === 'create' ? '新增' : '编辑' }}{{ config.title }}</strong>
        <button type="button" class="text-button" @click="closeDialog">关闭</button>
      </header>

      <label v-for="field in config.fields" :key="field.key">
        {{ field.label }}
        <div v-if="isImageValueField(field)" class="config-image-field">
          <button type="button" class="config-image-preview" @click="imageInputs[field.key]?.click()">
            <img :src="form[field.key] || '/images/config/default-img.jpg'" :alt="field.label" />
          </button>
          <input
            :ref="(el) => setImageInput(field.key, el)"
            class="hidden-file"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp,image/bmp"
            @change="uploadConfigImage($event, field)"
          />
          <div class="config-image-tools">
            <input v-model.trim="form[field.key]" placeholder="图片地址" />
            <button type="button" :disabled="uploadingField === field.key" @click="imageInputs[field.key]?.click()">
              {{ uploadingField === field.key ? '上传中...' : '上传图片' }}
            </button>
          </div>
        </div>
        <textarea v-else-if="field.type === 'textarea'" v-model="form[field.key]" :rows="field.rows || 4"></textarea>
        <select v-else-if="field.type === 'select'" v-model="form[field.key]">
          <option v-for="option in fieldOptions(field)" :key="option.value" :value="option.value" :disabled="option.disabled">{{ option.label }}</option>
        </select>
        <select v-else-if="field.type === 'multi-select'" v-model="form[field.key]" multiple>
          <option v-for="option in fieldOptions(field)" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
        <div v-else-if="field.type === 'permission-tree'" class="permission-tree">
          <PermissionNode
            v-for="node in permissionTree"
            :key="node.id"
            :node="node"
            :selected="selectedPermissionIds"
            @toggle="togglePermission"
          />
        </div>
        <input v-else v-model="form[field.key]" :type="field.type || 'text'" :placeholder="field.placeholder || ''" />
      </label>

      <p v-if="dialogError" class="form-error">{{ dialogError }}</p>
      <div class="form-actions">
        <button type="submit" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
        <button type="button" class="secondary-button" @click="closeDialog">取消</button>
      </div>
    </form>
  </div>

  <div v-if="passwordDialog.open" class="modal-mask">
    <form class="modal-panel compact-modal" @submit.prevent="savePassword">
      <header class="modal-header">
        <strong>修改{{ config.title }}密码</strong>
        <button type="button" class="text-button" @click="closePasswordDialog">关闭</button>
      </header>
      <label>
        账号
        <input :value="rowDisplayName(passwordDialog.row)" disabled />
      </label>
      <label>
        新密码
        <input v-model="passwordDialog.password" type="password" placeholder="请输入新密码" autocomplete="new-password" />
      </label>
      <label>
        确认密码
        <input v-model="passwordDialog.confirmPassword" type="password" placeholder="请再次输入新密码" autocomplete="new-password" />
      </label>
      <p v-if="passwordError" class="form-error">{{ passwordError }}</p>
      <div class="form-actions">
        <button type="submit" :disabled="passwordSaving">{{ passwordSaving ? '保存中...' : '保存密码' }}</button>
        <button type="button" class="secondary-button" @click="closePasswordDialog">取消</button>
      </div>
    </form>
  </div>

  <div v-if="replaceDialog.open" class="modal-mask">
    <form class="modal-panel compact-modal" @submit.prevent="saveReplace">
      <header class="modal-header">
        <strong>批量替换评论内容</strong>
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
      <p class="muted-text">会替换所有评论中的匹配文本，包括回收站评论。</p>
      <p v-if="replaceError" class="form-error">{{ replaceError }}</p>
      <div class="form-actions">
        <button type="submit" :disabled="replaceSaving">{{ replaceSaving ? '替换中...' : '开始替换' }}</button>
        <button type="button" class="secondary-button" @click="closeReplaceDialog">取消</button>
      </div>
    </form>
  </div>

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
import {
  createResource,
  deleteResource,
  forceDeleteResource,
  getRole,
  getRolePermissionTree,
  listResource,
  restoreResource,
  updateResource,
  updateResourcePassword,
  uploadAdminImage,
  replaceComments
} from '../../api/adminResources'
import AdminConfirmDialog from '../../components/admin/AdminConfirmDialog.vue'
import AdminNoticeDialog from '../../components/admin/AdminNoticeDialog.vue'
import AdminPagination from '../../components/admin/AdminPagination.vue'
import PermissionNode from '../../components/admin/PermissionNode.vue'
import SearchableSelect from '../../components/admin/SearchableSelect.vue'
import { notifyAdminSuccess } from '../../utils/adminToast'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const auth = useAuthStore()
const rows = ref([])
const optionRows = reactive({
  roles: [],
  permissions: [],
  categories: [],
  navs: []
})
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const actionLoading = ref(false)
const toggleLoadingKey = ref('')
const inlineLoadingKey = ref('')
const passwordSaving = ref(false)
const replaceSaving = ref(false)
const error = ref('')
const dialogError = ref('')
const passwordError = ref('')
const replaceError = ref('')
const selectedIds = ref([])
const filters = reactive({
  keyword: '',
  page: 1,
  limit: 15
})
const dialog = reactive({
  open: false,
  mode: 'create',
  id: null
})
const confirmDialog = reactive({
  open: false,
  row: null,
  ids: [],
  action: 'delete'
})
const passwordDialog = reactive({
  open: false,
  row: null,
  password: '',
  confirmPassword: ''
})
const replaceDialog = reactive({
  open: false,
  search: '',
  replace: ''
})
const notice = reactive({
  open: false,
  message: ''
})
const form = reactive({})
const imageInputs = reactive({})
const permissionTree = ref([])
const selectedPermissionIds = ref([])
const uploadingField = ref('')
const collapsedTreeIds = ref(new Set())

const statusOptions = [
  { label: '禁用', value: 0 },
  { label: '启用', value: 1 }
]
const yesNoOptions = [
  { label: '否', value: 0 },
  { label: '是', value: 1 }
]
const allYesNoOptions = [
  { label: '全部', value: '' },
  { label: '否', value: 0 },
  { label: '是', value: 1 }
]
const guardOptions = [
  { label: 'admin', value: 'admin' },
  { label: 'home', value: 'home' }
]
const allGuardOptions = [
  { label: '全部', value: '' },
  ...guardOptions
]
const allStatusOptions = [
  { label: '全部', value: '' },
  ...statusOptions
]
const commentStatusOptions = [
  { label: '全部', value: '' },
  { label: '待审核', value: 0 },
  { label: '已发布', value: 1 }
]
const sexSearchOptions = [
  { label: '全部', value: '' },
  { label: '保密', value: -1 },
  { label: '男', value: 0 },
  { label: '女', value: 1 }
]
const deleteOptions = [
  { label: '正常', value: '' },
  { label: '回收站', value: 1 },
  { label: '全部含删除', value: 2 }
]

const configs = {
  comments: {
    title: '评论',
    creatable: false,
    recyclable: true,
    searchFields: [
      { key: 'username', label: '评论者' },
      { key: 'title', label: '文章标题' },
      { key: 'status', label: '状态', type: 'select', options: commentStatusOptions },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'username', label: '用户' },
      { key: 'title', label: '文章' },
      { key: 'reply_relation', label: '回复关系', format: formatCommentReplyRelation },
      { key: 'content', label: '内容', max: 42 },
      { key: 'status', label: '状态', map: { 0: '待审核', 1: '已发布' }, toggle: true },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [
      { key: 'content', label: '内容', type: 'textarea', rows: 5 },
      { key: 'status', label: '状态', type: 'select', options: [{ label: '待审核', value: 0 }, { label: '已发布', value: 1 }], normalize: Number }
    ]
  },
  categories: {
    title: '分类',
    recyclable: true,
    childable: true,
    treeParentKey: 'pid',
    searchFields: [
      { key: 'name', label: '名称' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '分类名称', tree: true },
      { key: 'keywords', label: '关键词' },
      { key: 'sort', label: '排序', inlineEdit: 'number' },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [
      { key: 'name', label: '分类名称', required: true },
      { key: 'keywords', label: '关键词' },
      { key: 'description', label: '描述', type: 'textarea' },
      { key: 'sort', label: '排序', type: 'number', normalize: Number },
      { key: 'pid', label: '上级分类', type: 'select', optionsFrom: 'categories', normalize: Number, default: 0, emptyLabel: '顶级分类' }
    ]
  },
  tags: {
    title: '标签',
    recyclable: true,
    searchFields: [
      { key: 'name', label: '标签名' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '标签名' },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [{ key: 'name', label: '标签名', required: true }]
  },
  users: {
    title: '用户',
    passwordable: true,
    recyclable: true,
    searchFields: [
      { key: 'name', label: '昵称' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '昵称' },
      { key: 'email', label: '邮箱' },
      { key: 'tel', label: '手机号' },
      { key: 'city', label: '城市' },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [
      { key: 'name', label: '昵称', required: true },
      { key: 'email', label: '邮箱' },
      { key: 'password', label: '密码', type: 'password', createRequired: true, placeholder: '编辑时请使用行内密码按钮修改' },
      { key: 'tel', label: '手机号' },
      { key: 'sex', label: '性别', type: 'select', options: [{ label: '保密', value: 0 }, { label: '男', value: 1 }, { label: '女', value: 2 }], normalize: Number },
      { key: 'city', label: '城市' },
      { key: 'intro', label: '介绍', type: 'textarea' },
      { key: 'avatar', label: '头像' }
    ]
  },
  admins: {
    title: '管理员',
    passwordable: true,
    recyclable: true,
    searchFields: [
      { key: 'account', label: '账号' },
      { key: 'username', label: '昵称' },
      { key: 'sex', label: '性别', type: 'select', options: sexSearchOptions },
      { key: 'status', label: '状态', type: 'select', options: allStatusOptions },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'account', label: '账号' },
      { key: 'username', label: '昵称' },
      { key: 'role_names', label: '角色' },
      { key: 'status', label: '状态', map: { 0: '禁用', 1: '正常' }, toggle: true },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [
      { key: 'account', label: '账号', required: true },
      { key: 'username', label: '昵称', required: true },
      { key: 'password', label: '密码', type: 'password', createRequired: true, placeholder: '编辑时留空则不修改' },
      { key: 'role_ids', label: '角色', type: 'multi-select', optionsFrom: 'roles', normalize: normalizeIDArray },
      { key: 'tel', label: '手机号' },
      { key: 'email', label: '邮箱' },
      { key: 'sex', label: '性别', type: 'select', options: [{ label: '保密', value: -1 }, { label: '男', value: 0 }, { label: '女', value: 1 }], normalize: Number },
      { key: 'status', label: '状态', type: 'select', options: statusOptions, normalize: Number }
    ]
  },
  roles: {
    title: '角色',
    recyclable: true,
    searchFields: [
      { key: 'name', label: '角色名称' },
      { key: 'description', label: '角色描述' },
      { key: 'guard_name', label: '权限组', type: 'select', options: allGuardOptions },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '角色名称' },
      { key: 'description', label: '描述' },
      { key: 'guard_name', label: 'Guard' },
      { key: 'status', label: '状态', map: { 0: '禁用', 1: '正常' }, toggle: true }
    ],
    fields: [
      { key: 'name', label: '角色名称', required: true },
      { key: 'description', label: '描述' },
      { key: 'guard_name', label: 'Guard', type: 'select', options: guardOptions, default: 'admin' },
      { key: 'status', label: '状态', type: 'select', options: statusOptions, normalize: Number, default: 1 },
      { key: 'permission_ids', label: '权限', type: 'permission-tree' }
    ]
  },
  permissions: {
    title: '权限',
    recyclable: true,
    childable: true,
    treeParentKey: 'parent_id',
    searchFields: [
      { key: 'name', label: '名称' },
      { key: 'guard_name', label: '权限组', type: 'select', options: allGuardOptions },
      { key: 'display_menu', label: '菜单是否显示', type: 'select', options: allYesNoOptions },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '权限名称', tree: true },
      { key: 'url', label: '地址' },
      { key: 'guard_name', label: 'Guard' },
      { key: 'icon', label: '图标' },
      { key: 'sort_order', label: '排序', inlineEdit: 'number' },
      { key: 'display_menu', label: '菜单', map: { 0: '否', 1: '是' }, toggle: true }
    ],
    fields: [
      { key: 'name', label: '权限名称', required: true },
      { key: 'guard_name', label: 'Guard', type: 'select', options: guardOptions, default: 'admin' },
      { key: 'url', label: '地址' },
      { key: 'icon', label: '图标' },
      { key: 'parent_id', label: '上级权限', type: 'select', optionsFrom: 'permissions', normalize: Number, default: 0 },
      { key: 'sort_order', label: '排序', type: 'number', normalize: Number },
      { key: 'display_menu', label: '显示菜单', type: 'select', options: yesNoOptions, normalize: Number, default: 1 }
    ]
  },
  navs: {
    title: '导航',
    recyclable: true,
    childable: true,
    treeParentKey: 'pid',
    searchFields: [
      { key: 'name', label: '名称' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '菜单名', tree: true },
      { key: 'url', label: '链接' },
      { key: 'target', label: '打开方式' },
      { key: 'icon', label: '图标' },
      { key: 'pid', label: '上级' },
      { key: 'sort', label: '排序', inlineEdit: 'number' }
    ],
    fields: [
      { key: 'name', label: '菜单名', required: true },
      { key: 'url', label: '链接' },
      { key: 'target', label: '打开方式', type: 'select', options: [
        { label: '当前窗口', value: '_self' },
        { label: '新窗口', value: '_blank' },
        { label: '父窗口', value: '_parent' },
        { label: '顶层窗口', value: '_top' },
        { label: '命名窗口', value: 'framename' }
      ], default: '_self' },
      { key: 'icon', label: '图标' },
      { key: 'pid', label: '上级导航', type: 'select', optionsFrom: 'navs', normalize: Number, default: 0, emptyLabel: '顶级导航' },
      { key: 'sort', label: '排序', type: 'number', normalize: Number }
    ]
  },
  'friend-links': {
    title: '友情链接',
    recyclable: true,
    searchFields: [
      { key: 'name', label: '链接名' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'name', label: '链接名' },
      { key: 'url', label: '地址' },
      { key: 'email', label: '邮箱' },
      { key: 'status', label: '状态', map: { 0: '禁用', 1: '启用' }, toggle: true },
      { key: 'sort', label: '排序' }
    ],
    fields: [
      { key: 'name', label: '链接名', required: true },
      { key: 'url', label: '地址', required: true },
      { key: 'email', label: '邮箱' },
      { key: 'status', label: '状态', type: 'select', options: statusOptions, normalize: Number },
      { key: 'sort', label: '排序', type: 'number', normalize: Number }
    ]
  },
  chats: {
    title: '闲言碎语',
    recyclable: true,
    searchFields: [
      { key: 'content', label: '内容' },
      { key: 'delete', label: '是否删除', type: 'select', options: deleteOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'content', label: '内容', max: 60 },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [{ key: 'content', label: '内容', type: 'textarea', required: true, rows: 5 }]
  },
  'wx-keywords': {
    title: '微信关键词',
    recyclable: false,
    searchFields: [
      { key: 'name', label: '关键词' },
      { key: 'key_value', label: '回复内容' },
      { key: 'status', label: '状态', type: 'select', options: allStatusOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'sort', label: '排序', inlineEdit: 'number' },
      { key: 'key_name', label: '关键词' },
      { key: 'key_value', label: '回复内容', max: 80 },
      { key: 'status', label: '状态', map: { 0: '待审核', 1: '已审核' }, toggle: true },
      { key: 'created_at', label: '创建时间' }
    ],
    fields: [
      { key: 'key_name', label: '关键词', required: true },
      { key: 'key_value', label: '回复内容', type: 'textarea', required: true, rows: 5 },
      { key: 'sort', label: '排序', type: 'number', normalize: Number },
      { key: 'status', label: '状态', type: 'select', options: statusOptions, normalize: Number, default: 1 }
    ]
  },
  'system-configs': {
    title: '系统配置',
    creatable: false,
    deletable: false,
    searchFields: [
      { key: 'title', label: '标题' },
      { key: 'key', label: '字段' },
      { key: 'config_type', label: '配置类型' },
      { key: 'status', label: '状态', type: 'select', options: allStatusOptions }
    ],
    columns: [
      { key: 'id', label: 'ID' },
      { key: 'title', label: '标题' },
      { key: 'key', label: '字段' },
      { key: 'value', label: '值', max: 42 },
      { key: 'type', label: '类型' },
      { key: 'status', label: '状态', map: { 0: '关闭', 1: '开启' }, toggle: true }
    ],
    fields: [
      { key: 'title', label: '标题', required: true },
      { key: 'key', label: '字段', required: true },
      { key: 'value', label: '值', type: 'textarea', rows: 3 },
      { key: 'type', label: '类型' },
      { key: 'config_type', label: '配置类型', type: 'number', normalize: Number },
      { key: 'status', label: '状态', type: 'select', options: statusOptions, normalize: Number }
    ]
  }
}

const resource = computed(() => route.meta.resource)
const config = computed(() => configs[resource.value] || configs.comments)
const permissionConfig = computed(() => resourcePermissions[resource.value] || {})
const canCreate = computed(() => config.value.creatable !== false && canDo(permissionConfig.value.create))
const canEdit = computed(() => config.value.editable !== false && canDo(permissionConfig.value.edit))
const canDelete = computed(() => config.value.deletable !== false && canDo(permissionConfig.value.delete))
const canRestore = computed(() => config.value.recyclable === true && canDo(permissionConfig.value.restore))
const canForceDelete = computed(() => config.value.recyclable === true && canDo(permissionConfig.value.forceDelete))
const canCreateChild = computed(() => config.value.childable === true && canCreate.value)
const canPassword = computed(() => config.value.passwordable === true && canEdit.value)
const canReplaceComments = computed(() => resource.value === 'comments' && canDo('/admin/comment/replace'))
const hasActions = computed(() => canEdit.value || canDelete.value || canRestore.value || canForceDelete.value || canCreateChild.value || canPassword.value)
const canBatchAction = computed(() => canDelete.value || canRestore.value || canForceDelete.value)
const emptyColspan = computed(() => config.value.columns.length + (hasActions.value ? 1 : 0) + (canBatchAction.value ? 1 : 0))
const searchFields = computed(() => config.value.searchFields || [{ key: 'keyword', label: '关键词' }])
const isSystemConfig = computed(() => resource.value === 'system-configs')
const displayRows = computed(() => {
  if (!config.value.treeParentKey) return rows.value
  return flattenVisibleTree(buildTree(rows.value, config.value.treeParentKey), 0)
})
const selectedRows = computed(() => displayRows.value.filter((row) => selectedIds.value.includes(row.id)))
const selectedNormalRows = computed(() => selectedRows.value.filter((row) => !isDeleted(row)))
const selectedDeletedRows = computed(() => selectedRows.value.filter((row) => isDeleted(row)))
const isPageSelected = computed(() => displayRows.value.length > 0 && displayRows.value.every((row) => selectedIds.value.includes(row.id)))
const actionTitle = computed(() => `${actionText('verb')}${config.value.title}`)
const actionSubtitle = computed(() => {
  if (confirmDialog.action === 'force') return '彻底删除后数据不可恢复，请谨慎操作。'
  if (confirmDialog.action === 'restore') return '恢复后该数据会回到正常列表。'
  return '删除后会按当前模块规则处理，请确认后再继续。'
})
const actionMessage = computed(() => {
  if (confirmDialog.ids.length) return `确认${actionText('verb')}选中的 ${confirmDialog.ids.length} 条${config.value.title}？`
  const row = confirmDialog.row
  if (!row) return ''
  return `确认${actionText('verb')}${config.value.title}「${rowDisplayName(row)}」？`
})
const actionConfirmText = computed(() => `确认${actionText('verb')}`)
const actionLoadingText = computed(() => `${actionText('verb')}中...`)

const systemConfigGroups = [
  { label: '全部', value: '' },
  { label: '基本信息', value: 'basic' },
  { label: '联系方式', value: 'contact' },
  { label: 'SEO 设置', value: 'seo' },
  { label: '公众号 / 微信配置', value: 'wechat' },
  { label: '小程序配置', value: 'mini' }
]

const resourcePermissions = {
  comments: {
    edit: ['/admin/comment/update', '/admin/comment/replace'],
    delete: '/admin/comment/destroy',
    restore: ['/admin/comment/update', '/admin/comment/replace'],
    forceDelete: '/admin/comment/destroy'
  },
  categories: {
    create: '/admin/category/store',
    edit: '/admin/category/update',
    delete: '/admin/category/destroy',
    restore: '/admin/category/update',
    forceDelete: '/admin/category/destroy'
  },
  tags: {
    create: '/admin/tag/store',
    edit: ['/admin/tag/update', '/admin/tag/upda'],
    delete: '/admin/tag/destroy',
    restore: ['/admin/tag/update', '/admin/tag/upda'],
    forceDelete: '/admin/tag/destroy'
  },
  users: {
    create: '/admin/user/store',
    edit: '/admin/user/update',
    delete: '/admin/user/destroy',
    restore: '/admin/user/update',
    forceDelete: '/admin/user/destroy'
  },
  admins: {
    create: '/admin/admin/store',
    edit: '/admin/admin/update',
    delete: '/admin/admin/destroy',
    restore: '/admin/admin/update',
    forceDelete: '/admin/admin/destroy'
  },
  roles: {
    create: '/admin/role/store',
    edit: '/admin/role/update',
    delete: '/admin/role/destroy',
    restore: '/admin/role/update',
    forceDelete: '/admin/role/destroy'
  },
  permissions: {
    create: '/admin/permission/store',
    edit: '/admin/permission/update',
    delete: '/admin/permission/destroy',
    restore: '/admin/permission/update',
    forceDelete: '/admin/permission/destroy'
  },
  navs: {
    create: '/admin/nav/store',
    edit: '/admin/nav/update',
    delete: '/admin/nav/destroy',
    restore: '/admin/nav/update',
    forceDelete: '/admin/nav/destroy'
  },
  'friend-links': {
    create: '/admin/friendLinks/store',
    edit: '/admin/friendLinks/update',
    delete: '/admin/friendLinks/destroy',
    restore: '/admin/friendLinks/update',
    forceDelete: '/admin/friendLinks/destroy'
  },
  chats: {
    create: '/admin/chat/store',
    edit: '/admin/chat/update',
    delete: '/admin/chat/destroy',
    restore: '/admin/chat/update',
    forceDelete: '/admin/chat/destroy'
  },
  'wx-keywords': {
    create: '/admin/weChat/keyword/store',
    edit: '/admin/weChat/keyword/update',
    delete: '/admin/weChat/keyword/destroy',
    restore: '/admin/weChat/keyword/update',
    forceDelete: '/admin/weChat/keyword/destroy'
  },
  'system-configs': { edit: '/admin/systemConfig/update' }
}

function canDo(permission) {
  if (!permission) return true
  return Array.isArray(permission) ? auth.can(...permission) : auth.can(permission)
}

function formatValue(row, column) {
  if (column.format) return column.format(row, column)
  const raw = row[column.key]
  if (column.map) return column.map[raw] ?? raw
  if (column.max && typeof raw === 'string' && raw.length > column.max) {
    return `${raw.slice(0, column.max)}...`
  }
  return raw ?? '-'
}

function toggleKey(row, column) {
  return `${resource.value}:${row.id}:${column.key}`
}

function toggleLabel(row, column) {
  const value = row[column.key]
  if (column.map) return column.map[value] ?? value
  return Number(value) === 1 ? '开启' : '关闭'
}

async function toggleRowValue(row, column) {
  if (!canEdit.value || isDeleted(row)) return
  const key = toggleKey(row, column)
  toggleLoadingKey.value = key
  try {
    const nextValue = Number(row[column.key]) === 1 ? 0 : 1
    const payload = await buildRowPayload(row, { [column.key]: nextValue })
    await updateResource(resource.value, row.id, payload)
    row[column.key] = nextValue
    notifyAdminSuccess(`${config.value.title}${toggleLabel(row, column)}成功`)
  } catch (err) {
    notice.message = err.message || '状态更新失败'
    notice.open = true
  } finally {
    toggleLoadingKey.value = ''
  }
}

function inlineKey(row, column) {
  return `${resource.value}:${row.id}:${column.key}`
}

async function updateInlineValue(row, column, event) {
  if (!canEdit.value || isDeleted(row)) return
  const nextValue = Number(event.target.value)
  if (!Number.isFinite(nextValue)) {
    event.target.value = row[column.key] ?? 0
    return
  }
  const key = inlineKey(row, column)
  inlineLoadingKey.value = key
  try {
    const payload = await buildRowPayload(row, { [column.key]: nextValue })
    await updateResource(resource.value, row.id, payload)
    row[column.key] = nextValue
    notifyAdminSuccess(`${config.value.title}排序保存成功`)
    invalidateCurrentOptions()
    await load()
  } catch (err) {
    event.target.value = row[column.key] ?? 0
    notice.message = err.message || '排序保存失败'
    notice.open = true
  } finally {
    inlineLoadingKey.value = ''
  }
}

async function buildRowPayload(row, overrides = {}) {
  const payload = { ...row, ...overrides }
  delete payload.children
  delete payload._depth
  delete payload.created_at
  delete payload.updated_at
  delete payload.deleted_at

  if (resource.value === 'roles') {
    const res = await getRole(row.id)
    payload.permission_ids = res.data?.permission_ids || []
  }
  if (resource.value === 'admins' && !Array.isArray(payload.role_ids)) {
    payload.role_ids = []
  }
  return payload
}

function formatCommentReplyRelation(row) {
  if (!row.pid) return '主评论'

  const targetName = row.reply_to_username || `评论 #${row.pid}`
  const targetContent = previewText(row.reply_to_content, 18)
  const rootText = row.origin_id && row.origin_id !== row.pid ? ` / 所属楼 #${row.origin_id}` : ''
  return `回复 @${targetName}${targetContent ? `：${targetContent}` : ''}${rootText}`
}

function previewText(value, max = 20) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  if (!text) return ''
  return text.length > max ? `${text.slice(0, max)}...` : text
}

function isDeleted(row) {
  return Boolean(row.deleted_at && row.deleted_at.Valid !== false)
}

function actionText(type) {
  const map = {
    delete: { verb: '删除' },
    restore: { verb: '恢复' },
    force: { verb: '彻底删除' }
  }
  return map[confirmDialog.action]?.[type] || map.delete[type]
}

function resetForm(row = null, overrides = {}) {
  for (const key of Object.keys(form)) delete form[key]
  for (const field of config.value.fields || []) {
    if (field.type === 'permission-tree') continue
    if (field.type === 'multi-select') {
      form[field.key] = getMultiSelectValue(field, row)
    } else {
      form[field.key] = row?.[field.key] ?? field.default ?? ''
    }
  }
  Object.assign(form, overrides)
}

async function openCreate(parent = null) {
  dialog.mode = 'create'
  dialog.id = null
  dialogError.value = ''
  await loadOptionRows()
  resetForm(null, childDefaults(parent))
  await loadPermissionTree()
  dialog.open = true
}

async function openEdit(row) {
  dialog.mode = 'edit'
  dialog.id = row.id
  dialogError.value = ''
  await loadOptionRows()
  resetForm(row)
  if (resource.value === 'roles') {
    const res = await getRole(row.id)
    selectedPermissionIds.value = res.data.permission_ids || []
    await loadPermissionTree(row.id, form.guard_name || 'admin')
  }
  dialog.open = true
}

function closeDialog() {
  dialog.open = false
}

function openPasswordDialog(row) {
  passwordError.value = ''
  passwordDialog.row = row
  passwordDialog.password = ''
  passwordDialog.confirmPassword = ''
  passwordDialog.open = true
}

function closePasswordDialog() {
  if (passwordSaving.value) return
  passwordDialog.open = false
  passwordDialog.row = null
  passwordDialog.password = ''
  passwordDialog.confirmPassword = ''
}

function openReplaceDialog() {
  replaceError.value = ''
  replaceDialog.search = ''
  replaceDialog.replace = ''
  replaceDialog.open = true
}

function closeReplaceDialog() {
  if (replaceSaving.value) return
  replaceDialog.open = false
  replaceDialog.search = ''
  replaceDialog.replace = ''
}

async function saveReplace() {
  replaceError.value = ''
  const search = replaceDialog.search.trim()
  if (!search) {
    replaceError.value = '请填写要查找的内容'
    return
  }
  replaceSaving.value = true
  try {
    const res = await replaceComments({ search, replace: replaceDialog.replace })
    const changed = res.data?.changed ?? 0
    notifyAdminSuccess(`评论批量替换完成，更新 ${changed} 条`)
    closeReplaceDialog()
    await load()
  } catch (err) {
    replaceError.value = err.message || '批量替换失败'
  } finally {
    replaceSaving.value = false
  }
}

async function savePassword() {
  const row = passwordDialog.row
  if (!row) return
  passwordError.value = ''
  if (!passwordDialog.password) {
    passwordError.value = '请填写新密码'
    return
  }
  if (passwordDialog.password.length < 6) {
    passwordError.value = '密码至少 6 位'
    return
  }
  if (passwordDialog.password !== passwordDialog.confirmPassword) {
    passwordError.value = '两次密码不一致'
    return
  }
  passwordSaving.value = true
  try {
    await updateResourcePassword(resource.value, row.id, { password: passwordDialog.password })
    notifyAdminSuccess(`${config.value.title}密码修改成功`)
    closePasswordDialog()
  } catch (err) {
    passwordError.value = err.message
  } finally {
    passwordSaving.value = false
  }
}

function childDefaults(parent) {
  if (!parent || !config.value.treeParentKey || !Number.isFinite(Number(parent.id))) return {}
  return { [config.value.treeParentKey]: parent.id }
}

function buildPayload() {
  const payload = {}
  for (const field of config.value.fields || []) {
    if (field.type === 'permission-tree') {
      payload[field.key] = selectedPermissionIds.value
      continue
    }
    if ((field.required || (dialog.mode === 'create' && field.createRequired)) && !form[field.key]) {
      throw new Error(`请填写${field.label}`)
    }
    if (dialog.mode === 'edit' && field.type === 'password' && !form[field.key]) continue
    payload[field.key] = field.normalize ? field.normalize(form[field.key]) : form[field.key]
  }
  return payload
}

function isImageValueField(field) {
  if (resource.value !== 'system-configs' || field.key !== 'value') return false
  const key = String(form.key || '').toLowerCase()
  const type = String(form.type || '').toLowerCase()
  return type.includes('image') || type.includes('img') || ['site_logo', 'site_avatar', 'site_qrcode'].includes(key) || /(_logo|_avatar|_image|_img|_pic|_qrcode|_pay_qrcode)$/.test(key)
}

function setImageInput(key, el) {
  if (el) imageInputs[key] = el
}

async function uploadConfigImage(event, field) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  if (!/^image\/(jpeg|png|gif|webp|bmp)$/.test(file.type)) {
    dialogError.value = '只能上传 jpg、png、gif、webp、bmp 图片'
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    dialogError.value = '图片不能超过 5MB'
    return
  }
  uploadingField.value = field.key
  dialogError.value = ''
  try {
    const res = await uploadAdminImage(file)
    form[field.key] = res.data?.src || ''
    notifyAdminSuccess('图片上传成功')
  } catch (err) {
    dialogError.value = err.message
  } finally {
    uploadingField.value = ''
  }
}

function fieldOptions(field) {
  if (field.optionsFrom === 'categories') {
    return [{ label: field.emptyLabel || '全部分类', value: field.default ?? '' }].concat(
      flattenTree(buildTree(optionRows.categories, 'pid'), 0).map((category) => ({
        label: `${'— '.repeat(category._depth || 0)}${category.name}`,
        value: category.id,
        disabled: dialog.mode === 'edit' && category.id === dialog.id
      }))
    )
  }
  if (field.optionsFrom === 'navs') {
    return [{ label: field.emptyLabel || '顶级导航', value: field.default ?? 0 }].concat(
      flattenTree(buildTree(optionRows.navs, 'pid'), 0).map((nav) => ({
        label: `${'— '.repeat(nav._depth || 0)}${nav.name}`,
        value: nav.id,
        disabled: dialog.mode === 'edit' && nav.id === dialog.id
      }))
    )
  }
  if (field.optionsFrom === 'roles') {
    return optionRows.roles.map((role) => ({
      label: role.description ? `${role.name} - ${role.description}` : role.name,
      value: role.id
    }))
  }
  if (field.optionsFrom === 'permissions') {
    const options = [{ label: '顶级权限', value: 0 }]
    for (const item of flattenTree(buildTree(optionRows.permissions), 0)) {
      const disabled = dialog.mode === 'edit' && item.id === dialog.id
      options.push({
        label: `${'— '.repeat(item._depth || 0)}${item.name}${disabled ? '（当前）' : ''}`,
        value: item.id,
        disabled
      })
    }
    return options
  }
  return field.options || []
}

function getMultiSelectValue(field, row) {
  const value = row?.[field.key] || field.default || []
  if (Array.isArray(value) && value.length > 0) return [...value]
  if (field.optionsFrom !== 'roles' || !row?.role_names) return Array.isArray(value) ? [...value] : []

  const roleNames = String(row.role_names)
    .split(/[,，\s]+/)
    .map((item) => item.trim())
    .filter(Boolean)
  if (roleNames.length === 0) return []

  const roleNameSet = new Set(roleNames)
  return optionRows.roles
    .filter((role) => roleNameSet.has(role.name) || roleNameSet.has(role.description))
    .map((role) => role.id)
}

async function loadOptionRows() {
  if (needsOptionsFrom('categories') && optionRows.categories.length === 0) {
    const res = await listResource('categories', { page: 1, limit: 1000 })
    optionRows.categories = res.data || []
  }
  if (needsOptionsFrom('navs') && optionRows.navs.length === 0) {
    const res = await listResource('navs', { page: 1, limit: 1000 })
    optionRows.navs = res.data || []
  }
  if (resource.value === 'admins' && optionRows.roles.length === 0) {
    const res = await listResource('roles', { page: 1, limit: 1000 })
    optionRows.roles = res.data || []
  }
  if (resource.value === 'permissions') {
    const res = await listResource('permissions', { page: 1, limit: 1000 })
    optionRows.permissions = res.data || []
  }
}

function needsOptionsFrom(source) {
  return [...(config.value.searchFields || []), ...(config.value.fields || [])].some((field) => field.optionsFrom === source)
}

function normalizeIDArray(value) {
  return (value || []).map(Number).filter((item) => Number.isFinite(item) && item > 0)
}

function buildTree(items, parentKey = 'parent_id') {
  const cloned = items.map((item) => ({ ...item, children: [] }))
  const byID = new Map(cloned.map((item) => [item.id, item]))
  const roots = []
  for (const item of cloned) {
    const parent = byID.get(item[parentKey])
    if (parent && parent.id !== item.id) {
      parent.children.push(item)
    } else {
      roots.push(item)
    }
  }
  const sortItems = (list) => {
    list.sort((a, b) => (b.sort_order || b.sort || 0) - (a.sort_order || a.sort || 0) || a.id - b.id)
    list.forEach((item) => sortItems(item.children || []))
  }
  sortItems(roots)
  return roots
}

function flattenTree(items, depth) {
  return items.flatMap((item) => [{ ...item, _depth: depth }, ...flattenTree(item.children || [], depth + 1)])
}

function flattenVisibleTree(items, depth) {
  return items.flatMap((item) => {
    const row = { ...item, _depth: depth }
    if (isTreeCollapsed(item.id)) return [row]
    return [row, ...flattenVisibleTree(item.children || [], depth + 1)]
  })
}

function isTreeCollapsed(id) {
  return collapsedTreeIds.value.has(Number(id))
}

function toggleTreeNode(id) {
  const next = new Set(collapsedTreeIds.value)
  const key = Number(id)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  collapsedTreeIds.value = next
}

async function loadPermissionTree(roleID = 0, guardName = 'admin') {
  if (resource.value !== 'roles') return
  const res = await getRolePermissionTree({ role_id: roleID, guard_name: guardName })
  permissionTree.value = res.data || []
  if (!roleID) {
    selectedPermissionIds.value = []
  }
}

function togglePermission(node, checked) {
  const ids = collectPermissionIds(node)
  if (checked) {
    selectedPermissionIds.value = Array.from(new Set(selectedPermissionIds.value.concat(ids)))
  } else {
    const remove = new Set(ids)
    selectedPermissionIds.value = selectedPermissionIds.value.filter((id) => !remove.has(id))
  }
}

function collectPermissionIds(node) {
  return [node.id].concat((node.children || []).flatMap(collectPermissionIds))
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = { page: filters.page, limit: filters.limit }
    for (const field of searchFields.value) {
      const value = filters[field.key]
      if (value !== '' && value !== undefined && value !== null) {
        params[field.key] = field.normalize ? field.normalize(value) : value
      }
    }
    const res = await listResource(resource.value, params)
    rows.value = res.data || []
    selectedIds.value = selectedIds.value.filter((id) => displayRows.value.some((row) => row.id === id))
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

async function search() {
  filters.page = 1
  await load()
}

async function resetSearch() {
  resetSearchFilters()
  await load()
}

async function changePage(page) {
  filters.page = page
  await load()
}

function changePageSize(pageSize) {
  filters.limit = pageSize
}

function resetSearchFilters() {
  for (const key of Object.keys(filters)) {
    if (key !== 'page' && key !== 'limit') delete filters[key]
  }
  filters.page = 1
  filters.limit = 15
  for (const field of searchFields.value) {
    filters[field.key] = field.default ?? ''
  }
  if (isSystemConfig.value) filters.group = typeof route.query.group === 'string' ? route.query.group : ''
}

async function setSystemConfigGroup(group) {
  filters.group = group
  filters.config_type = ''
  filters.page = 1
  await load()
}

async function save() {
  dialogError.value = ''
  saving.value = true
  try {
    const payload = buildPayload()
    if (dialog.mode === 'create') {
      await createResource(resource.value, payload)
      notifyAdminSuccess(`${config.value.title}新增成功`)
    } else {
      await updateResource(resource.value, dialog.id, payload)
      notifyAdminSuccess(`${config.value.title}保存成功`)
    }
    closeDialog()
    invalidateCurrentOptions()
    await load()
  } catch (err) {
    dialogError.value = err.message
  } finally {
    saving.value = false
  }
}

function invalidateCurrentOptions() {
  if (resource.value === 'categories') optionRows.categories = []
  if (resource.value === 'navs') optionRows.navs = []
  if (resource.value === 'permissions') optionRows.permissions = []
  if (resource.value === 'roles') optionRows.roles = []
}

function confirmAction(row, action) {
  error.value = ''
  confirmDialog.row = row
  confirmDialog.ids = []
  confirmDialog.action = action
  confirmDialog.open = true
}

function confirmBatchAction(action) {
  const candidates = action === 'delete' ? selectedNormalRows.value : selectedDeletedRows.value
  const ids = candidates.map((row) => row.id)
  if (!ids.length) return
  error.value = ''
  confirmDialog.row = null
  confirmDialog.ids = ids
  confirmDialog.action = action
  confirmDialog.open = true
}

function closeConfirm() {
  if (actionLoading.value) return
  confirmDialog.open = false
  confirmDialog.row = null
  confirmDialog.ids = []
}

async function runConfirmedAction() {
  const ids = confirmDialog.ids.length ? [...confirmDialog.ids] : confirmDialog.row ? [confirmDialog.row.id] : []
  if (!ids.length) return
  actionLoading.value = true
  try {
    for (const id of ids) await runResourceAction(id, confirmDialog.action)
    const successText = actionText('verb')
    selectedIds.value = selectedIds.value.filter((id) => !ids.includes(id))
    actionLoading.value = false
    closeConfirm()
    notifyAdminSuccess(`${config.value.title}${successText}成功`)
    invalidateCurrentOptions()
    await load()
  } catch (err) {
    actionLoading.value = false
    closeConfirm()
    notice.message = err.message || `${actionText('verb')}失败`
    notice.open = true
  }
}

async function runResourceAction(id, action) {
  if (action === 'restore') {
    await restoreResource(resource.value, id)
  } else if (action === 'force') {
    await forceDeleteResource(resource.value, id)
  } else {
    await deleteResource(resource.value, id)
  }
}

function togglePageSelection(event) {
  if (event.target.checked) {
    selectedIds.value = Array.from(new Set(selectedIds.value.concat(displayRows.value.map((row) => row.id))))
  } else {
    const pageIds = new Set(displayRows.value.map((row) => row.id))
    selectedIds.value = selectedIds.value.filter((id) => !pageIds.has(id))
  }
}

function clearSelection() {
  selectedIds.value = []
}

function rowDisplayName(row) {
  const keys = ['title', 'name', 'username', 'account', 'content', 'email', 'key', 'id']
  for (const key of keys) {
    const value = row?.[key]
    if (value !== undefined && value !== null && value !== '') {
      const text = String(value)
      return text.length > 28 ? `${text.slice(0, 28)}...` : text
    }
  }
  return `ID ${row?.id || ''}`
}

watch(resource, () => {
  resetSearchFilters()
  closeDialog()
  closePasswordDialog()
  closeReplaceDialog()
  closeConfirm()
  clearSelection()
  notice.open = false
  load()
})

watch(
  () => route.query.group,
  async () => {
    if (!isSystemConfig.value) return
    resetSearchFilters()
    await load()
  }
)

onMounted(async () => {
  resetSearchFilters()
  await loadOptionRows()
  await load()
})
</script>
