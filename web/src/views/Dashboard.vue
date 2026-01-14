<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import request from '@/utils/request'
import { Picture, Connection, Monitor, Refresh } from '@element-plus/icons-vue'

interface SystemInfo {
  os: string
  os_version: string
  arch: string
  hostname: string
  docker_version: string
  cpu_cores: number
  memory_total: number
  disk_total: number
  disk_free: number
}

interface CacheStats {
  total_size: number
  entry_count: number
  hit_count: number
  miss_count: number
  hit_rate: number
}

interface ImageInfo {
  name: string
  tag: string
  size: number
  created_at: string
}

const loading = ref(false)
const systemInfo = ref<SystemInfo | null>(null)
const cacheStats = ref<CacheStats | null>(null)
const recentImages = ref<ImageInfo[]>([])
const imageCount = ref(0)

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const diskUsagePercent = computed(() => {
  if (!systemInfo.value) return 0
  const used = systemInfo.value.disk_total - systemInfo.value.disk_free
  return Math.round((used / systemInfo.value.disk_total) * 100)
})

const memoryFormatted = computed(() => {
  if (!systemInfo.value) return '0 GB'
  return formatBytes(systemInfo.value.memory_total)
})

const fetchData = async () => {
  loading.value = true
  try {
    const [sysRes, cacheRes, imagesRes] = await Promise.allSettled([
      request.get('/system/info'),
      request.get('/accel/cache/stats'),
      request.get('/images', { params: { page: 1, page_size: 5 } })
    ])

    if (sysRes.status === 'fulfilled') {
      systemInfo.value = sysRes.value.data
    }
    if (cacheRes.status === 'fulfilled') {
      cacheStats.value = cacheRes.value.data
    }
    if (imagesRes.status === 'fulfilled') {
      recentImages.value = imagesRes.value.data?.images || []
      imageCount.value = imagesRes.value.data?.total || 0
    }
  } catch (error) {
    console.error('获取仪表盘数据失败:', error)
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="dashboard" v-loading="loading">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon images">
          <el-icon :size="28"><Picture /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ imageCount }}</div>
          <div class="stat-label">镜像总数</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon cache">
          <el-icon :size="28"><Connection /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ cacheStats?.entry_count || 0 }}</div>
          <div class="stat-label">缓存条目</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon storage">
          <el-icon :size="28"><Monitor /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ formatBytes(cacheStats?.total_size || 0) }}</div>
          <div class="stat-label">缓存大小</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon hit-rate">
          <el-icon :size="28"><Refresh /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ ((cacheStats?.hit_rate || 0) * 100).toFixed(1) }}%</div>
          <div class="stat-label">缓存命中率</div>
        </div>
      </div>
    </div>

    <div class="content-grid">
      <!-- 系统信息 -->
      <div class="tech-card system-info">
        <div class="card-header">
          <h3>系统概览</h3>
        </div>
        <div class="card-body" v-if="systemInfo">
          <div class="info-row">
            <span class="info-label">主机名</span>
            <span class="info-value">{{ systemInfo.hostname || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">操作系统</span>
            <span class="info-value">{{ systemInfo.os }} {{ systemInfo.os_version }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">架构</span>
            <span class="info-value">{{ systemInfo.arch }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Docker版本</span>
            <span class="info-value">{{ systemInfo.docker_version || '-' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">CPU核心</span>
            <span class="info-value">{{ systemInfo.cpu_cores }} 核</span>
          </div>
          <div class="info-row">
            <span class="info-label">内存</span>
            <span class="info-value">{{ memoryFormatted }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">磁盘使用</span>
            <div class="disk-usage">
              <el-progress 
                :percentage="diskUsagePercent" 
                :stroke-width="8"
                :color="diskUsagePercent > 80 ? '#f85149' : '#58a6ff'"
              />
              <span class="disk-text">
                {{ formatBytes(systemInfo.disk_total - systemInfo.disk_free) }} / {{ formatBytes(systemInfo.disk_total) }}
              </span>
            </div>
          </div>
        </div>
        <div class="card-body empty" v-else>
          <span>暂无系统信息</span>
        </div>
      </div>

      <!-- 最近镜像 -->
      <div class="tech-card recent-images">
        <div class="card-header">
          <h3>最近镜像</h3>
          <router-link to="/images" class="view-all">查看全部</router-link>
        </div>
        <div class="card-body" v-if="recentImages.length > 0">
          <div class="image-list">
            <div class="image-item" v-for="image in recentImages" :key="`${image.name}:${image.tag}`">
              <div class="image-info">
                <span class="image-name">{{ image.name }}</span>
                <span class="image-tag">:{{ image.tag }}</span>
              </div>
              <div class="image-meta">
                <span class="image-size">{{ formatBytes(image.size) }}</span>
                <span class="image-date">{{ formatDate(image.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="card-body empty" v-else>
          <span>暂无镜像</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  animation: fadeIn 0.3s ease-out;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: border-color 0.2s, transform 0.2s;
}

.stat-card:hover {
  border-color: var(--highlight-color);
  transform: translateY(-2px);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.stat-icon.images {
  background: linear-gradient(135deg, #1890ff, #096dd9);
}

.stat-icon.cache {
  background: linear-gradient(135deg, #52c41a, #389e0d);
}

.stat-icon.storage {
  background: linear-gradient(135deg, #722ed1, #531dab);
}

.stat-icon.hit-rate {
  background: linear-gradient(135deg, #fa8c16, #d46b08);
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--text-color);
  font-family: var(--font-mono);
}

.stat-label {
  font-size: 14px;
  color: var(--muted-text);
  margin-top: 4px;
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

@media (max-width: 900px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}

.tech-card {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.view-all {
  font-size: 13px;
  color: var(--highlight-color);
}

.card-body {
  padding: 16px 20px;
}

.card-body.empty {
  padding: 40px 20px;
  text-align: center;
  color: var(--muted-text);
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-color);
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  color: var(--muted-text);
  font-size: 14px;
}

.info-value {
  color: var(--text-color);
  font-family: var(--font-mono);
  font-size: 14px;
}

.disk-usage {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  min-width: 200px;
}

.disk-text {
  font-size: 12px;
  color: var(--muted-text);
  font-family: var(--font-mono);
}

.image-list {
  display: flex;
  flex-direction: column;
}

.image-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
}

.image-item:last-child {
  border-bottom: none;
}

.image-info {
  display: flex;
  align-items: baseline;
}

.image-name {
  color: var(--text-color);
  font-weight: 500;
}

.image-tag {
  color: var(--highlight-color);
  font-family: var(--font-mono);
  font-size: 13px;
}

.image-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--muted-text);
}

.image-size {
  font-family: var(--font-mono);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
