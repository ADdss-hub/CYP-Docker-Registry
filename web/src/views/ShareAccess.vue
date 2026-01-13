<template>
  <div class="share-container">
    <div class="share-card">
      <div class="share-header">
        <div class="logo">
          <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
            <path d="M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92-1.31-2.92-2.92-2.92z"/>
          </svg>
        </div>
        <h1>镜像分享</h1>
        <p class="subtitle">CYP-Registry 镜像分享链接</p>
      </div>

      <div v-if="loading" class="loading-state">
        <el-icon class="is-loading" :size="40"><Loading /></el-icon>
        <p>正在验证分享链接...</p>
      </div>

      <div v-else-if="error" class="error-state">
        <el-icon :size="60" color="#ff3366"><CircleClose /></el-icon>
        <h2>链接无效</h2>
        <p>{{ errorMessage }}</p>
        <el-button type="primary" @click="goHome">返回首页</el-button>
      </div>

      <div v-else-if="requirePassword && !authenticated" class="password-form">
        <p>此分享链接需要密码访问</p>
        <el-input
          v-model="password"
          type="password"
          placeholder="请输入访问密码"
          size="large"
          show-password
          @keyup.enter="verifyPassword"
        />
        <el-button
          type="primary"
          size="large"
          :loading="verifying"
          @click="verifyPassword"
          style="margin-top: 16px; width: 100%"
        >
          验证密码
        </el-button>
      </div>

      <div v-else class="share-content">
        <div class="image-info">
          <h2>{{ shareInfo?.image_ref }}</h2>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="镜像名称">
              {{ shareInfo?.image_ref }}
            </el-descriptions-item>
            <el-descriptions-item label="分享者">
              {{ shareInfo?.created_by_name || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="过期时间">
              {{ formatDate(shareInfo?.expires_at) }}
            </el-descriptions-item>
            <el-descriptions-item label="剩余访问次数">
              {{ shareInfo?.max_usage ? shareInfo.max_usage - shareInfo.usage_count : '无限制' }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="pull-command">
          <h3>拉取命令</h3>
          <div class="command-box">
            <code>{{ pullCommand }}</code>
            <el-button type="primary" size="small" @click="copyCommand">
              复制
            </el-button>
          </div>
        </div>
      </div>

      <div class="share-footer">
        <p>CYP-Registry | Copyright © 2026 CYP</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, CircleClose } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface ShareInfo {
  code: string
  image_ref: string
  created_by: number
  created_by_name?: string
  max_usage: number
  usage_count: number
  expires_at: string
  require_password: boolean
}

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref(false)
const errorMessage = ref('')
const shareInfo = ref<ShareInfo | null>(null)
const requirePassword = ref(false)
const authenticated = ref(false)
const password = ref('')
const verifying = ref(false)

const shareCode = computed(() => route.params.code as string)

const pullCommand = computed(() => {
  if (!shareInfo.value) return ''
  return `docker pull ${window.location.host}/${shareInfo.value.image_ref}`
})

onMounted(async () => {
  await fetchShareInfo()
})

async function fetchShareInfo() {
  loading.value = true
  error.value = false

  try {
    const response = await request.get(`/api/v1/share/${shareCode.value}`)
    shareInfo.value = response.data
    requirePassword.value = response.data.require_password
    
    if (!requirePassword.value) {
      authenticated.value = true
    }
  } catch (err: any) {
    error.value = true
    const status = err.response?.status
    
    if (status === 404) {
      errorMessage.value = '分享链接不存在或已过期'
    } else if (status === 410) {
      errorMessage.value = '分享链接已达到最大访问次数'
    } else {
      errorMessage.value = '无法访问分享链接'
    }
  } finally {
    loading.value = false
  }
}

async function verifyPassword() {
  if (!password.value) {
    ElMessage.warning('请输入密码')
    return
  }

  verifying.value = true
  try {
    await request.post(`/api/v1/share/${shareCode.value}/verify`, {
      password: password.value
    })
    authenticated.value = true
    ElMessage.success('验证成功')
  } catch {
    ElMessage.error('密码错误')
  } finally {
    verifying.value = false
  }
}

function copyCommand() {
  navigator.clipboard.writeText(pullCommand.value)
  ElMessage.success('已复制到剪贴板')
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return '无限制'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function goHome() {
  router.push('/')
}
</script>

<style scoped>
.share-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1f3a 100%);
  padding: 20px;
}

.share-card {
  width: 100%;
  max-width: 500px;
  background: var(--bg-secondary, #1a1f3a);
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.share-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  color: var(--primary, #00d4ff);
  margin-bottom: 16px;
}

.share-header h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  font-size: 14px;
  margin: 0;
}

.loading-state,
.error-state {
  text-align: center;
  padding: 40px 0;
}

.loading-state p,
.error-state p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin-top: 16px;
}

.error-state h2 {
  color: var(--text-primary, #ffffff);
  margin: 16px 0 8px 0;
}

.password-form {
  text-align: center;
}

.password-form p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin-bottom: 16px;
}

.share-content {
  margin-bottom: 24px;
}

.image-info h2 {
  color: var(--primary, #00d4ff);
  font-size: 18px;
  margin: 0 0 16px 0;
  word-break: break-all;
}

.pull-command {
  margin-top: 24px;
}

.pull-command h3 {
  color: var(--text-primary, #ffffff);
  font-size: 14px;
  margin: 0 0 12px 0;
}

.command-box {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(0, 0, 0, 0.3);
  padding: 12px;
  border-radius: 8px;
}

.command-box code {
  flex: 1;
  color: #00d4ff;
  font-family: monospace;
  font-size: 13px;
  word-break: break-all;
}

.share-footer {
  text-align: center;
  color: var(--text-secondary, rgba(255, 255, 255, 0.4));
  font-size: 12px;
  margin-top: 24px;
}
</style>
