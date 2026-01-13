<template>
  <div class="locked-container">
    <div class="lock-screen">
      <div class="lock-icon">
        <svg viewBox="0 0 24 24" width="80" height="80" fill="currentColor">
          <path d="M12 1C8.676 1 6 3.676 6 7v2H4v14h16V9h-2V7c0-3.324-2.676-6-6-6zm0 2c2.276 0 4 1.724 4 4v2H8V7c0-2.276 1.724-4 4-4zm0 10c1.1 0 2 .9 2 2s-.9 2-2 2-2-.9-2-2 .9-2 2-2z"/>
        </svg>
      </div>
      
      <h1>系统已锁定</h1>
      <p class="lock-reason">{{ lockReason || '检测到安全威胁' }}</p>

      <el-alert
        v-if="requireManual"
        type="error"
        :closable="false"
        style="margin: 20px 0; text-align: left"
      >
        <template #title>
          <strong>系统检测到严重的安全威胁！</strong>
        </template>
        <p style="margin: 8px 0 0 0">
          请联系系统管理员手动解锁。<br>
          解锁命令: <code>./scripts/unlock.sh</code>
        </p>
      </el-alert>

      <div v-if="!requireManual && unlockAt" class="auto-unlock-info">
        <p>自动解锁时间: {{ formatTime(unlockAt) }}</p>
      </div>

      <div class="unlock-form" v-if="!requireManual">
        <el-input
          v-model="password"
          type="password"
          placeholder="输入管理员密码"
          size="large"
          show-password
          :disabled="loading"
        />
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          @click="handleUnlock"
          style="margin-top: 16px; width: 100%"
        >
          申请紧急解锁
        </el-button>
      </div>

      <div class="lock-info">
        <h3>锁定详情</h3>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="锁定时间">
            {{ formatDate(lockedAt) }}
          </el-descriptions-item>
          <el-descriptions-item label="触发IP">
            {{ lockedByIP || '未知' }}
          </el-descriptions-item>
          <el-descriptions-item label="触发用户">
            {{ lockedByUser || '匿名' }}
          </el-descriptions-item>
          <el-descriptions-item label="锁定类型">
            {{ lockTypeText }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="copyright">
        CYP-Registry v{{ version }} | Copyright © 2026 CYP
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
const version = ref(appStore.version || '1.0.0')

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
    default:
      return '未知'
  }
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

.lock-screen {
  text-align: center;
  max-width: 600px;
  padding: 40px;
  background: var(--bg-secondary, #1a1f3a);
  border-radius: 12px;
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.lock-icon {
  color: #ff3366;
  margin-bottom: 20px;
}

.lock-screen h1 {
  color: var(--text-primary, #ffffff);
  font-size: 28px;
  margin: 0 0 16px 0;
}

.lock-reason {
  color: #ff3366;
  font-size: 16px;
  margin: 0 0 20px 0;
}

.auto-unlock-info {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin: 20px 0;
}

.unlock-form {
  margin: 24px 0;
  max-width: 300px;
  margin-left: auto;
  margin-right: auto;
}

.lock-info {
  margin-top: 32px;
  text-align: left;
}

.lock-info h3 {
  color: var(--text-primary, #ffffff);
  font-size: 16px;
  margin: 0 0 16px 0;
}

.copyright {
  margin-top: 40px;
  font-size: 12px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.4));
}

:deep(.el-descriptions) {
  --el-descriptions-item-bordered-label-background: rgba(255, 255, 255, 0.05);
}

:deep(.el-descriptions__label) {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
}

:deep(.el-descriptions__content) {
  color: var(--text-primary, #ffffff);
}

code {
  background: rgba(0, 212, 255, 0.1);
  color: #00d4ff;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}
</style>
