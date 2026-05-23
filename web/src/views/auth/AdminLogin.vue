<template>
  <main class="login-page">
    <form class="login-panel" @submit.prevent="submit">
      <h1>后台登录</h1>
      <label>
        账号
        <input v-model.trim="form.account" type="text" autocomplete="username" />
      </label>
      <label>
        密码
        <input v-model="form.password" type="password" autocomplete="current-password" />
      </label>
      <p v-if="error" class="form-error">{{ error }}</p>
      <button type="submit" :disabled="loading">
        {{ loading ? '登录中...' : '登录' }}
      </button>
    </form>
  </main>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const error = ref('')

const form = reactive({
  account: '',
  password: ''
})

async function submit() {
  error.value = ''
  if (!form.account || !form.password) {
    error.value = '请输入账号和密码'
    return
  }

  loading.value = true
  try {
    await auth.login(form.account, form.password)
    router.push(route.query.redirect || '/admin')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}
</script>
