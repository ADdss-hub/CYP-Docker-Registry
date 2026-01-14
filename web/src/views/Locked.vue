<template>
  <div class="locked-container">
    <div class="locked-card">
      <div class="locked-header">
        <div class="logo lock-icon">
          <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
            <path d="M12 1C8.676 1 6 3.676 6 7v2H4v14h16V9h-2V7c0-3.324-2.676-6-6-6zm0 2c2.276 0 4 1.724 4 4v2H8V7c0-2.276 1.724-4 4-4zm0 10c1.1 0 2 .9 2 2s-.9 2-2 2-2-.9-2-2 .9-2 2-2z"/>
          </svg>
        </div>
        <h1>系统已锁定</h1>
        <p class="subtitle lock-reason">{{ lockReasonText }}</p>
      </div>

      <el-alert
        v-if="requireManual"
        type="error"
        :closable="false"
        class="security-alert"
      >
        <template #title>
          <strong>系统检测到严重的安全威胁！</strong>
        </template>
        <p class="alert-content">
          请联系系统管理员手动解锁。<br>
          解锁命令: <code>./scripts/unlock.sh</code>
        </p>
      </el-alert>

      <div v-if="!requireManual && unlockAt" class="auto-unlock-info">
        <el-tag type="info" size="large">
          自动解锁时间: {{ formatTime(unlockAt) }}
        </el-tag>
      </div>

      <el-form
        v-if="!requireManual"
        class="unlock-form"
        @submit.prevent="handleUnlock"
      >
        <el-form-item>
          <el-input
            v-model="password"
            type="password"
            placeholder="输入管理员密码"
            prefix-icon="Lock"
            size="large"
            show-password
            :disabled="loading"
            @keyup.enter="handleUnlock"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="unlock-button"
            @click="handleUnlock"
          >
            {{ loading ? '解锁中...' : '申请紧急解锁' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="lock-details">
        <h3>锁定详情</h3>
        <div class="details-grid">
          <div class="detail-item">
            <span class="detail-label">锁定时间</span>
            <span class="detail-value">{{ formatDate(lockedAt) }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">触发IP</span>
            <span class="detail-value">{{ lockedByIP || '未知' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">触发用户</span>
            <span class="detail-value">{{ lockedByUser || '匿名' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">锁定类型</span>
            <span class="detail-value">{{ lockTypeText }}</span>
          </div>
        </div>
      </div>

      <div class="locked-footer">
        <p>CYP-Docker Registry v{{ version }}</p>
        <p>版权所有 © 2026 CYP</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useLockStore } from '@/stores/lock'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const lockStore = useLockStore()
const appStore = useAppStore()

const password = ref('')
const loading = ref(false)
const version = ref(appStore.version || '1.0.8')

const lockReason = computed(() => lockStore.lockStatus?.lock_reason || '')
const requireManual = computed(() => lockStore.lockStatus?.require_manual ?? true)
const unlockAt = computed(() => lockStore.lockStatus?.unlock_at || '')
const lockedAt = computed(() => lockStore.lockStatus?.locked_at || '')
const lockedByIP = computed(() => lockStore.lockStatus?.locked_by_ip || '')
const lockedByUser = computed(() => lockStore.lockStatus?.locked_by_user || '')
const lockType = computed(() => lockStore.lockStatus?.lock_type || '')

const lockTypeText = computed(() => {
  switch (lockType.value) {
    case 'bypass_attempt':
      return '绕过尝试'
    case 'rule_triggered':
      return '规则触发'
    case 'manual':
      return '手动锁定'
    case 'too_many_failed_attempts':
      return '登录失败次数过多'
    default:
      return '未知'
  }
})

const lockReasonText = computed(() => {
  const reason = lockReason.value
  if (!reason) return '检测到安全威胁'
  
  const reasonMap: Record<string, string> = {
    'too_many_failed_attempts': '登录失败次数过多',
    'bypass_attempt': '检测到绕过尝试',
    'rule_triggered': '安全规则触发',
    'manual': '管理员手动锁定',
    'security_threat': '检测到安全威胁'
  }
  
  return reasonMap[reason] || reason
})

onMounted(async () => {
  await lockStore.fetchLockStatus()
  
  // If not locked, redirect to home
  if (!lockStore.isLocked) {
    router.push('/')
  }
})

function formatDate(dateStr: string): string {
  if (!dateStr) return '未知'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function formatTime(dateStr: string): string {
  if (!dateStr) return '未知'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

async function handleUnlock() {
  if (!password.value) {
    ElMessage.warning('请输入管理员密码')
    return
  }

  loading.value = true
  try {
    const success = await lockStore.requestUnlock(password.value)
    if (success) {
      ElMessage.success('系统已解锁')
      router.push('/login')
    } else {
      ElMessage.error('解锁失败，密码错误')
    }
  } catch {
    ElMessage.error('解锁请求失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.locked-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1f3a 100%);
  padding: 20px;
}

.locked-card {
  width: 100%;
  max-width: 480px;
  background: var(--bg-secondary, #1a1f3a);
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.locked-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  margin-bottom: 16px;
}

.lock-icon {
  color: #ff3366;
}

.locked-header h1 {
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

.lock-reason {
  color: #ff3366 !important;
}

.security-alert {
  margin-bottom: 24px;
  text-align: left;
}

.alert-content {
  margin: 8px 0 0 0;
}

.auto-unlock-info {
  text-align: center;
  margin-bottom: 24px;
}

.unlock-form {
  margin-bottom: 24px;
}

.unlock-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.lock-details {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
}

.lock-details h3 {
  color: var(--text-primary, #ffffff);
  font-size: 14px;
  font-weight: 500;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
}

.detail-value {
  font-size: 14px;
  color: var(--text-primary, #ffffff);
}

.locked-footer {
  text-align: center;
  color: var(--text-secondary, rgba(255, 255, 255, 0.4));
  font-size: 12px;
}

.locked-footer p {
  margin: 4px 0;
}

code {
  background: rgba(0, 212, 255, 0.1);
  color: #00d4ff;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
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

@media (max-width: 500px) {
  .details-grid {
    grid-template-columns: 1fr;
  }
}
</style>
