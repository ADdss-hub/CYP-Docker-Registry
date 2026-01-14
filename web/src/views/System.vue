<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Download, Warning, CircleCheck } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface SystemInfo {
  os: string
  os_version: string
  arch: string
  hostname: string
  docker_version: string
  containerd_version: string
  cpu_cores: number
  memory_total: number
  disk_total: number
  disk_free: number
}

interface CompatibilityReport {
  compatible: boolean
  warnings: string[]
  errors: string[]
}

interface VersionInfo {
  current: string
  latest: string
  has_update: boolean
  release_at: string
  changelog: string
}

const loading = ref(false)
const systemInfo = ref<SystemInfo | null>(null)
const compatibility = ref<CompatibilityReport | null>(null)
const versionInfo = ref<VersionInfo | null>(null)
const updateStatus = ref('')
const checkingUpdate = ref(false)

const formatBytes = (bytes: number | null | undefined): string => {
  if (bytes === null || bytes === undefined || isNaN(bytes) || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i < 0 || i >= sizes.length) return '0 B'
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const diskUsagePercent = computed(() => {
  if (!systemInfo.value || !systemInfo.value.disk_total || systemInfo.value.disk_total === 0) return 0
  const used = (systemInfo.value.disk_total || 0) - (systemInfo.value.disk_free || 0)
  const percent = Math.round((used / systemInfo.value.disk_total) * 100)
  return isNaN(percent) ? 0 : percent
})

const diskUsedFormatted = computed(() => {
  if (!systemInfo.value || !systemInfo.value.disk_total) return '0 B'
  return formatBytes((systemInfo.value.disk_total || 0) - (systemInfo.value.disk_free || 0))
})

const fetchSystemInfo = async () => {
  try {
    const res = await request.get('/system/info')
    systemInfo.value = res.data
  } catch (error) {
    console.error('获取系统信息失败:', error)
  }
}

const fetchCompatibility = async () => {
  try {
    const res = await request.get('/system/compatibility')
    compatibility.value = res.data
  } catch (error) {
    console.error('获取兼容性信息失败:', error)
  }
}

const fetchUpdateStatus = async () => {
  try {
    const res = await request.get('/update/status')
    updateStatus.value = res.data?.status || ''
    if (res.data?.version_info) {
      versionInfo.value = res.data.version_info
    }
  } catch (error) {
    console.error('获取更新状态失败:', error)
  }
}

const fetchAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchSystemInfo(), fetchCompatibility(), fetchUpdateStatus()])
  } finally {
    loading.value = false
  }
}

const refreshSystemInfo = async () => {
  loading.value = true
  try {
    const res = await request.get('/system/refresh')
    systemInfo.value = res.data?.info
    ElMessage.success('系统信息已刷新')
  } catch (error) {
    ElMessage.error('刷新系统信息失败')
  } finally {
    loading.value = false
  }
}

const checkUpdate = async () => {
  checkingUpdate.value = true
  try {
    const res = await request.get('/update/check')
    versionInfo.value = res.data
    if (res.data?.has_update) {
      ElMessage.success(`发现新版本: ${res.data.latest}`)
    } else {
      ElMessage.info('当前已是最新版本')
    }
  } catch (error) {
    ElMessage.error('检查更新失败')
  } finally {
    checkingUpdate.value = false
  }
}

const downloadUpdate = async () => {
  if (!versionInfo.value?.latest) {
    ElMessage.warning('请先检查更新')
    return
  }
  try {
    await request.post('/update/download', { version: versionInfo.value.latest })
    ElMessage.success('开始下载更新')
    fetchUpdateStatus()
  } catch (error) {
    ElMessage.error('下载更新失败')
  }
}

onMounted(() => {
  fetchAll()
})
</script>

<template>
  <div class="system-page" v-loading="loading">
    <!-- 系统信息 -->
    <div class="section">
      <div class="section-header">
        <h3>宿主机信息</h3>
        <el-button @click="refreshSystemInfo" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <div class="info-grid" v-if="systemInfo">
        <div class="info-card">
          <div class="info-label">主机名</div>
          <div class="info-value">{{ systemInfo.hostname || '-' }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">操作系统</div>
          <div class="info-value">{{ systemInfo.os }} {{ systemInfo.os_version }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">系统架构</div>
          <div class="info-value">{{ systemInfo.arch }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">CPU核心</div>
          <div class="info-value">{{ systemInfo.cpu_cores }} 核</div>
        </div>
        <div class="info-card">
          <div class="info-label">内存总量</div>
          <div class="info-value">{{ formatBytes(systemInfo.memory_total) }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">Docker版本</div>
          <div class="info-value">{{ systemInfo.docker_version || '未检测到' }}</div>
        </div>
        <div class="info-card">
          <div class="info-label">Containerd版本</div>
          <div class="info-value">{{ systemInfo.containerd_version || '未检测到' }}</div>
        </div>
        <div class="info-card wide">
          <div class="info-label">磁盘使用</div>
          <div class="disk-info">
            <el-progress 
              :percentage="diskUsagePercent" 
              :stroke-width="12"
              :color="diskUsagePercent > 80 ? '#f85149' : diskUsagePercent > 60 ? '#d29922' : '#3fb950'"
            />
            <div class="disk-text">
              已用 {{ diskUsedFormatted }} / 总计 {{ formatBytes(systemInfo.disk_total) }}
              <span class="disk-free">(剩余 {{ formatBytes(systemInfo.disk_free) }})</span>
            </div>
          </div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <span>暂无系统信息</span>
      </div>
    </div>

    <!-- 兼容性检查 -->
    <div class="section">
      <div class="section-header">
        <h3>兼容性检查</h3>
      </div>
      <div class="compatibility-content" v-if="compatibility">
        <div class="compat-status" :class="{ success: compatibility.compatible, warning: !compatibility.compatible }">
          <el-icon :size="24">
            <CircleCheck v-if="compatibility.compatible" />
            <Warning v-else />
          </el-icon>
          <span>{{ compatibility.compatible ? '系统兼容性良好' : '存在兼容性问题' }}</span>
        </div>
        <div class="compat-warnings" v-if="compatibility.warnings?.length">
          <div class="compat-item warning" v-for="(warn, index) in compatibility.warnings" :key="'w' + index">
            <el-icon><Warning /></el-icon>
            <span>{{ typeof warn === 'string' ? warn : warn.message || warn.component }}</span>
          </div>
        </div>
        <div class="compat-errors" v-if="compatibility.errors?.length">
          <div class="compat-item error" v-for="(err, index) in compatibility.errors" :key="'e' + index">
            <el-icon><Warning /></el-icon>
            <span>{{ typeof err === 'string' ? err : err.message || err.component }}</span>
          </div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <span>暂无兼容性信息</span>
      </div>
    </div>

    <!-- 版本更新 -->
    <div class="section">
      <div class="section-header">
        <h3>版本更新</h3>
        <div class="section-actions">
          <el-button @click="checkUpdate" :loading="checkingUpdate">
            <el-icon><Refresh /></el-icon>
            检查更新
          </el-button>
          <el-button 
            type="primary" 
            @click="downloadUpdate" 
            :disabled="!versionInfo?.has_update"
          >
            <el-icon><Download /></el-icon>
            下载更新
          </el-button>
        </div>
      </div>
      <div class="version-content" v-if="versionInfo">
        <div class="version-grid">
          <div class="version-card">
            <div class="version-label">当前版本</div>
            <div class="version-value current">v{{ versionInfo.current || '1.0.7' }}</div>
          </div>
          <div class="version-card">
            <div class="version-label">最新版本</div>
            <div class="version-value" :class="{ latest: versionInfo.has_update }">
              v{{ versionInfo.latest || versionInfo.current || '1.0.7' }}
              <el-tag v-if="versionInfo.has_update" type="success" size="small">有更新</el-tag>
            </div>
          </div>
        </div>
        <div class="update-status" v-if="updateStatus">
          <span class="status-label">更新状态：</span>
          <span class="status-value">{{ updateStatus }}</span>
        </div>
        <div class="changelog" v-if="versionInfo.changelog">
          <div class="changelog-title">更新日志</div>
          <div class="changelog-content">{{ versionInfo.changelog }}</div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <span>点击"检查更新"获取版本信息</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.system-page {
  animation: fadeIn 0.3s ease-out;
}

.section {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 20px;
  margin-bottom: 24px;
}

.section:last-child {
  margin-bottom: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.section-actions {
  display: flex;
  gap: 12px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

@media (max-width: 1000px) {
  .info-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 600px) {
  .info-grid { grid-template-columns: 1fr; }
}

.info-card {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
}

.info-card.wide {
  grid-column: span 2;
}

@media (max-width: 600px) {
  .info-card.wide { grid-column: span 1; }
}

.info-label {
  font-size: 12px;
  color: var(--muted-text);
  margin-bottom: 8px;
}

.info-value {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
  font-family: var(--font-mono);
}

.disk-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.disk-text {
  font-size: 13px;
  color: var(--muted-text);
  font-family: var(--font-mono);
}

.disk-free {
  color: var(--highlight-color);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--muted-text);
}

.compatibility-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.compat-status {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.compat-status.success {
  background-color: rgba(63, 185, 80, 0.15);
  color: var(--success-color);
}

.compat-status.warning {
  background-color: rgba(210, 153, 34, 0.15);
  color: var(--warning-color);
}

.compat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  border-radius: var(--radius-sm);
  font-size: 14px;
}

.compat-item.warning {
  background-color: rgba(210, 153, 34, 0.1);
  color: var(--warning-color);
}

.compat-item.error {
  background-color: rgba(248, 81, 73, 0.1);
  color: var(--error-color);
}

.version-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.version-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

@media (max-width: 600px) {
  .version-grid { grid-template-columns: 1fr; }
}

.version-card {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 20px;
  text-align: center;
}

.version-label {
  font-size: 12px;
  color: var(--muted-text);
  margin-bottom: 8px;
}

.version-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
  font-family: var(--font-mono);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.version-value.current {
  color: var(--muted-text);
}

.version-value.latest {
  color: var(--success-color);
}

.update-status {
  padding: 12px 16px;
  background-color: var(--bg-color);
  border-radius: var(--radius-sm);
}

.status-label {
  color: var(--muted-text);
}

.status-value {
  color: var(--highlight-color);
  font-family: var(--font-mono);
}

.changelog {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
}

.changelog-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 12px;
}

.changelog-content {
  font-size: 13px;
  color: var(--muted-text);
  white-space: pre-wrap;
  line-height: 1.6;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
