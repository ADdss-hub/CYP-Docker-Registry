<template>
  <div class="register-container">
    <div class="register-card">
      <div class="register-header">
        <div class="logo">
          <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          </svg>
        </div>
        <h1>CYP-Docker Registry</h1>
        <p class="subtitle">创建新账户</p>
      </div>

      <!-- 注册成功后显示令牌信息 -->
      <div v-if="registerSuccess" class="token-result">
        <el-alert
          title="注册成功！"
          type="success"
          :closable="false"
          show-icon
          style="margin-bottom: 16px"
        />
        
        <div class="token-info">
          <h3>您的个人访问令牌</h3>
          <p class="token-warning">请妥善保存此令牌，它只会显示一次！</p>
          <div class="token-display">
            <code>{{ generatedToken }}</code>
            <el-button type="primary" size="small" @click="copyToken">
              复制
            </el-button>
          </div>
        </div>

        <el-button
          type="primary"
          size="large"
          class="register-button"
          @click="goToLogin"
        >
          前往登录
        </el-button>
      </div>

      <!-- 注册表单 -->
      <el-form
        v-else
        ref="formRef"
        :model="form"
        :rules="rules"
        class="register-form"
        @submit.prevent="handleRegister"
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
          />
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :disabled="loading"
            @keyup.enter="handleRegister"
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

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="register-button"
            @click="handleRegister"
          >
            {{ loading ? '注册中...' : '注册' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div v-if="!registerSuccess" class="register-links">
        <router-link to="/login">已有账户？立即登录</router-link>
      </div>

      <div class="register-footer">
        <p>CYP-Docker Registry v{{ version }}</p>
        <p>版权所有 © 2026 CYP</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const appStore = useAppStore()

const formRef = ref()
const loading = ref(false)
const errorMessage = ref('')
const version = ref(appStore.version || '1.0.5')
const registerSuccess = ref(false)
const generatedToken = ref('')

const form = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度为3-20个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 32, message: '密码长度为6-32个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

async function handleRegister() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  errorMessage.value = ''

  try {
    const response = await request.post('/api/v1/auth/register', {
      username: form.username,
      password: form.password
    })

    // 注册成功，显示生成的令牌
    if (response.data?.token) {
      generatedToken.value = response.data.token
      registerSuccess.value = true
      ElMessage.success('注册成功！请保存您的个人令牌')
    } else {
      ElMessage.success('注册成功，请登录')
      router.push('/login')
    }
  } catch (error: any) {
    const data = error.response?.data
    errorMessage.value = data?.error || '注册失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

function copyToken() {
  navigator.clipboard.writeText(generatedToken.value)
    .then(() => {
      ElMessage.success('令牌已复制到剪贴板')
    })
    .catch(() => {
      ElMessage.error('复制失败，请手动复制')
    })
}

function goToLogin() {
  router.push('/login')
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1f3a 100%);
  padding: 20px;
}

.register-card {
  width: 100%;
  max-width: 400px;
  background: var(--bg-secondary, #1a1f3a);
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.register-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  color: var(--primary, #00d4ff);
  margin-bottom: 16px;
}

.register-header h1 {
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

.register-form {
  margin-bottom: 24px;
}

.register-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.register-links {
  text-align: center;
  margin-bottom: 24px;
}

.register-links a {
  color: var(--primary, #00d4ff);
  text-decoration: none;
  font-size: 14px;
}

.register-links a:hover {
  text-decoration: underline;
}

.register-footer {
  text-align: center;
  color: var(--text-secondary, rgba(255, 255, 255, 0.4));
  font-size: 12px;
}

.register-footer p {
  margin: 4px 0;
}

/* 令牌显示样式 */
.token-result {
  margin-bottom: 24px;
}

.token-info {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
}

.token-info h3 {
  color: var(--text-primary, #ffffff);
  font-size: 16px;
  font-weight: 500;
  margin: 0 0 8px 0;
}

.token-warning {
  color: #ff9800;
  font-size: 12px;
  margin: 0 0 12px 0;
}

.token-display {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 4px;
  padding: 12px;
}

.token-display code {
  flex: 1;
  color: var(--primary, #00d4ff);
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
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
