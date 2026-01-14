<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Edit, Refresh, Check, Close } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface CacheStats {
  total_size: number
  entry_count: number
  hit_count: number
  miss_count: number
  hit_rate: number
}

interface CacheEntry {
  digest: string
  size: number
  last_access: string
  access_count: number
}

interface UpstreamSource {
  name: string
  url: string
  priority: number
  enabled: boolean
}

const loading = ref(false)
const cacheStats = ref<CacheStats | null>(null)
const cacheEntries = ref<CacheEntry[]>([])
const upstreams = ref<UpstreamSource[]>([])

const upstreamDialogVisible = ref(false)
const upstreamForm = ref<UpstreamSource>({
  name: '',
  url: '',
  priority: 1,
  enabled: true
})
const isEditMode = ref(false)
const editingName = ref('')

const formatBytes = (bytes: number | null | undefined): string => {
  if (bytes === null || bytes === undefined || isNaN(bytes) || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i < 0 || i >= sizes.length) return '0 B'
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const hitRatePercent = computed(() => {
  if (!cacheStats.value) return 0
  return (cacheStats.value.hit_rate * 100).toFixed(1)
})

const fetchCacheStats = async () => {
  try {
    const res = await request.get('/accel/cache/stats')
    cacheStats.value = res.data?.data || res.data
  } catch (error) {
    console.error('获取缓存统计失败:', error)
  }
}

const fetchCacheEntries = async () => {
  try {
    const res = await request.get('/accel/cache/entries')
    const data = res.data?.data || res.data
    cacheEntries.value = data?.entries || []
  } catch (error) {
    console.error('获取缓存条目失败:', error)
  }
}

const fetchUpstreams = async () => {
  try {
    const res = await request.get('/accel/upstreams')
    const data = res.data?.data || res.data
    upstreams.value = data?.upstreams || []
  } catch (error) {
    console.error('获取上游源失败:', error)
  }
}

const fetchAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchCacheStats(), fetchCacheEntries(), fetchUpstreams()])
  } finally {
    loading.value = false
  }
}

const clearCache = async () => {
  try {
    await ElMessageBox.confirm('确定要清空所有缓存吗？', '清空缓存', { type: 'warning' })
    await request.delete('/accel/cache')
    ElMessage.success('缓存已清空')
    fetchAll()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error('清空缓存失败')
  }
}

const deleteCacheEntry = async (digest: string) => {
  try {
    await request.delete(`/accel/cache/${encodeURIComponent(digest)}`)
    ElMessage.success('缓存条目已删除')
    fetchCacheEntries()
    fetchCacheStats()
  } catch (error) {
    ElMessage.error('删除缓存条目失败')
  }
}

const showAddUpstream = () => {
  isEditMode.value = false
  upstreamForm.value = { name: '', url: '', priority: upstreams.value.length + 1, enabled: true }
  upstreamDialogVisible.value = true
}

const showEditUpstream = (upstream: UpstreamSource) => {
  isEditMode.value = true
  editingName.value = upstream.name
  upstreamForm.value = { ...upstream }
  upstreamDialogVisible.value = true
}

const saveUpstream = async () => {
  if (!upstreamForm.value.name || !upstreamForm.value.url) {
    ElMessage.warning('请填写完整信息')
    return
  }
  try {
    if (isEditMode.value) {
      await request.put(`/accel/upstreams/${encodeURIComponent(editingName.value)}`, upstreamForm.value)
      ElMessage.success('上游源更新成功')
    } else {
      await request.post('/accel/upstreams', upstreamForm.value)
      ElMessage.success('上游源添加成功')
    }
    upstreamDialogVisible.value = false
    fetchUpstreams()
  } catch (error) {
    ElMessage.error(isEditMode.value ? '更新上游源失败' : '添加上游源失败')
  }
}

const deleteUpstream = async (name: string) => {
  try {
    await ElMessageBox.confirm(`确定要删除上游源 "${name}" 吗？`, '删除确认', { type: 'warning' })
    await request.delete(`/accel/upstreams/${encodeURIComponent(name)}`)
    ElMessage.success('上游源已删除')
    fetchUpstreams()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error('删除上游源失败')
  }
}

const toggleUpstream = async (upstream: UpstreamSource) => {
  try {
    const endpoint = upstream.enabled ? 'disable' : 'enable'
    await request.post(`/accel/upstreams/${encodeURIComponent(upstream.name)}/${endpoint}`)
    ElMessage.success(upstream.enabled ? '上游源已禁用' : '上游源已启用')
    fetchUpstreams()
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

const truncateDigest = (digest: string): string => {
  if (!digest) return '-'
  return digest.length > 24 ? digest.substring(0, 24) + '...' : digest
}

onMounted(() => {
  fetchAll()
})
</script>

<template>
  <div class="accelerator-page" v-loading="loading">
    <!-- 缓存统计 -->
    <div class="section">
      <div class="section-header">
        <h3>缓存统计</h3>
        <div class="section-actions">
          <el-button @click="fetchAll" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button type="danger" @click="clearCache">
            <el-icon><Delete /></el-icon>
            清空缓存
          </el-button>
        </div>
      </div>
      <div class="stats-grid" v-if="cacheStats">
        <div class="stat-card">
          <div class="stat-label">缓存条目</div>
          <div class="stat-value">{{ cacheStats.entry_count }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">缓存大小</div>
          <div class="stat-value">{{ formatBytes(cacheStats.total_size) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">命中次数</div>
          <div class="stat-value">{{ cacheStats.hit_count }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">未命中次数</div>
          <div class="stat-value">{{ cacheStats.miss_count }}</div>
        </div>
        <div class="stat-card highlight">
          <div class="stat-label">命中率</div>
          <div class="stat-value">{{ hitRatePercent }}%</div>
        </div>
      </div>
    </div>

    <!-- 上游源配置 -->
    <div class="section">
      <div class="section-header">
        <h3>上游源配置</h3>
        <el-button type="primary" @click="showAddUpstream">
          <el-icon><Plus /></el-icon>
          添加上游源
        </el-button>
      </div>
      <div class="upstreams-list">
        <div class="upstream-card" v-for="upstream in upstreams" :key="upstream.name" :class="{ disabled: !upstream.enabled }">
          <div class="upstream-header">
            <div class="upstream-name">
              <span class="priority-badge">{{ upstream.priority }}</span>
              {{ upstream.name }}
            </div>
            <el-tag :type="upstream.enabled ? 'success' : 'info'" size="small">
              {{ upstream.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </div>
          <div class="upstream-url"><code>{{ upstream.url }}</code></div>
          <div class="upstream-actions">
            <el-button size="small" text type="primary" @click="showEditUpstream(upstream)">
              <el-icon><Edit /></el-icon>编辑
            </el-button>
            <el-button size="small" text :type="upstream.enabled ? 'warning' : 'success'" @click="toggleUpstream(upstream)">
              <el-icon><component :is="upstream.enabled ? Close : Check" /></el-icon>
              {{ upstream.enabled ? '禁用' : '启用' }}
            </el-button>
            <el-button size="small" text type="danger" @click="deleteUpstream(upstream.name)">
              <el-icon><Delete /></el-icon>删除
            </el-button>
          </div>
        </div>
        <div class="empty-state" v-if="upstreams.length === 0"><span>暂无上游源配置</span></div>
      </div>
    </div>

    <!-- 缓存条目列表 -->
    <div class="section">
      <div class="section-header"><h3>缓存条目 ({{ cacheEntries.length }})</h3></div>
      <div class="table-container">
        <el-table :data="cacheEntries" stripe style="width: 100%" empty-text="暂无缓存数据" max-height="400">
          <el-table-column label="摘要" min-width="250">
            <template #default="{ row }">
              <el-tooltip :content="row.digest" placement="top">
                <code class="digest">{{ truncateDigest(row.digest) }}</code>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="120">
            <template #default="{ row }"><span class="size">{{ formatBytes(row.size) }}</span></template>
          </el-table-column>
          <el-table-column label="访问次数" width="100" align="center">
            <template #default="{ row }"><span>{{ row.access_count }}</span></template>
          </el-table-column>
          <el-table-column label="最后访问" width="180">
            <template #default="{ row }"><span class="date">{{ formatDate(row.last_access) }}</span></template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button size="small" text type="danger" @click="deleteCacheEntry(row.digest)">
                <el-icon><Delete /></el-icon>删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 上游源编辑对话框 -->
    <el-dialog v-model="upstreamDialogVisible" :title="isEditMode ? '编辑上游源' : '添加上游源'" width="500px" class="upstream-dialog">
      <el-form :model="upstreamForm" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="upstreamForm.name" placeholder="例如：Docker Hub" :disabled="isEditMode" />
        </el-form-item>
        <el-form-item label="URL" required>
          <el-input v-model="upstreamForm.url" placeholder="例如：https://registry-1.docker.io" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="upstreamForm.priority" :min="1" :max="100" />
          <span class="form-tip">数字越小优先级越高</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="upstreamForm.enabled" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="upstreamDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUpstream">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.accelerator-page {
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
.stats-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
}
@media (max-width: 1000px) {
  .stats-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 600px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}
.stat-card {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
  text-align: center;
}
.stat-card.highlight {
  border-color: var(--highlight-color);
  background-color: rgba(88, 166, 255, 0.1);
}
.stat-label {
  font-size: 12px;
  color: var(--muted-text);
  margin-bottom: 8px;
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
  font-family: var(--font-mono);
}
.stat-card.highlight .stat-value {
  color: var(--highlight-color);
}
.upstreams-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.upstream-card {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
  transition: border-color 0.2s;
}
.upstream-card:hover {
  border-color: var(--highlight-color);
}
.upstream-card.disabled {
  opacity: 0.6;
}
.upstream-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.upstream-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: var(--text-color);
}
.priority-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background-color: var(--primary-color);
  color: white;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
}
.upstream-url {
  margin-bottom: 12px;
}
.upstream-url code {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--muted-text);
}
.upstream-actions {
  display: flex;
  gap: 8px;
}
.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--muted-text);
}
.table-container {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.digest {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--muted-text);
}
.size {
  font-family: var(--font-mono);
  color: var(--highlight-color);
}
.date {
  color: var(--muted-text);
  font-size: 13px;
}
.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: var(--muted-text);
}
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* 对话框样式 */
.upstream-dialog :deep(.el-dialog__title) {
  color: #ffffff !important;
  font-weight: 600 !important;
  font-size: 18px !important;
  opacity: 1 !important;
}

.upstream-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid var(--border-color, #30363d);
  padding-bottom: 16px;
  background-color: var(--secondary-bg, #161b22);
}

.upstream-dialog :deep(.el-dialog__body) {
  color: var(--text-color, #e6edf3);
}

.upstream-dialog :deep(.el-form-item__label) {
  color: var(--label-text, #c9d1d9) !important;
}
</style>
