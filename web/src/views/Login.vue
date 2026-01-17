<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="logo">
          <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          </svg>
        </div>
        <h1>CYP-Docker Registry</h1>
        <p class="subtitle">容器镜像私有仓库管理系统</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="用户名"
            prefix-icon="User"
            size="large"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :disabled="loading"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-alert
          v-if="errorMessage"
          :title="errorMessage"
          type="error"
          :closable="false"
          show-icon
          style="margin-bottom: 16px"
        />

        <el-alert
          v-if="remainingAttempts !== null && remainingAttempts <= 2"
          :title="`警告：剩余 ${remainingAttempts} 次尝试机会，超过将锁定系统`"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 16px"
        />

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="login-button"
            @click="handleLogin"
          >
            {{ loading ? '登录中...' : '登录' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-links">
        <router-link to="/register">没有账户？立即注册</router-link>
      </div>

      <div class="login-footer">
        <p>CYP-Docker Registry v1.2.4</p>
        <p>版权所有 © 2026 CYP</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useLockStore } from '@/stores/lock'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const lockStore = useLockStore()

const formRef = ref()
const loading = ref(false)
const errorMessage = ref('')
const remainingAttempts = ref<number | null>(null)

const form = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
}

onMounted(async () => {
  // Check if already authenticated
  if (authStore.isAuthenticated) {
    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
    return
  }

  // Check lock status
  await lockStore.fetchLockStatus()
  if (lockStore.isLocked) {
    router.push({ name: 'locked' })
  }
})

async function handleLogin() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  errorMessage.value = ''
  remainingAttempts.value = null

  try {
    const response = await authStore.login({
      username: form.username,
      password: form.password
    })

    if (response.must_change_password) {
      ElMessage.warning('请修改默认密码')
    }

    if (response.lock_warning) {
      ElMessage.warning('检测到异常登录行为，请注意账户安全')
    }

    ElMessage.success('登录成功')

    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
  } catch (error: any) {
    const data = error.response?.data
    errorMessage.value = data?.error || '登录失败，请检查用户名和密码'
    
    if (data?.remaining_attempts !== undefined) {
      remainingAttempts.value = data.remaining_attempts
    }

    // Check if system is locked
    if (data?.details === 'system_locked') {
      lockStore.setLockStatus(true, data.lock_reason)
      router.push({ name: 'locked' })
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1f3a 100%);
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: var(--bg-secondary, #1a1f3a);
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  color: var(--primary, #00d4ff);
  margin-bottom: 16px;
}

.login-header h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  font-size: 14px;
  margin: 0;
}

.login-form {
  margin-bottom: 16px;
}

.login-links {
  text-align: center;
  margin-bottom: 24px;
}

.login-links a {
  color: var(--primary, #00d4ff);
  text-decoration: none;
  font-size: 14px;
}

.login-links a:hover {
  text-decoration: underline;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.login-footer {
  text-align: center;
  color: var(--text-secondary, rgba(255, 255, 255, 0.4));
  font-size: 12px;
}

.login-footer p {
  margin: 4px 0;
}

:deep(.el-input__wrapper) {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

:deep(.el-input__inner) {
  color: var(--text-primary, #ffffff);
}

:deep(.el-input__prefix) {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
}
</style>
