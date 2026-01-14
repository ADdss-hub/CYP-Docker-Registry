<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Delete, View, CopyDocument, Refresh } from '@element-plus/icons-vue'
import request from '@/utils/request'
import Pagination from '@/components/Pagination.vue'

interface Layer {
  digest: string
  size: number
  media_type: string
}

interface ImageInfo {
  name: string
  tag: string
  digest: string
  size: number
  created_at: string
  layers: Layer[]
}

const loading = ref(false)
const images = ref<ImageInfo[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const searchKeyword = ref('')
const detailDialogVisible = ref(false)
const selectedImage = ref<ImageInfo | null>(null)

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const fetchImages = async () => {
  loading.value = true
  try {
    const endpoint = searchKeyword.value ? '/images/search' : '/images'
    const params: Record<string, unknown> = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (searchKeyword.value) {
      params.q = searchKeyword.value
    }
    
    const res = await request.get(endpoint, { params })
    images.value = res.data?.images || []
    total.value = res.data?.total || 0
  } catch (error) {
    console.error('获取镜像列表失败:', error)
    ElMessage.error('获取镜像列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  fetchImages()
}

const handlePageChange = (page: number, size: number) => {
  currentPage.value = page
  pageSize.value = size
  fetchImages()
}

const showDetail = (image: ImageInfo) => {
  selectedImage.value = image
  detailDialogVisible.value = true
}

const copyPullCommand = (image: ImageInfo) => {
  const cmd = `docker pull localhost:8080/${image.name}:${image.tag}`
  navigator.clipboard.writeText(cmd).then(() => {
    ElMessage.success('拉取命令已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const deleteImage = async (image: ImageInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除镜像 ${image.name}:${image.tag} 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await request.delete(`/images/${image.name}/${image.tag}`)
    ElMessage.success('镜像删除成功')
    fetchImages()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除镜像失败:', error)
      ElMessage.error('删除镜像失败')
    }
  }
}

const truncateDigest = (digest: string): string => {
  if (!digest) return '-'
  if (digest.length > 20) {
    return digest.substring(0, 20) + '...'
  }
  return digest
}

watch(searchKeyword, (newVal, oldVal) => {
  if (newVal === '' && oldVal !== '') {
    handleSearch()
  }
})

onMounted(() => {
  fetchImages()
})
</script>

<template>
  <div class="images-page">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="search-box">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索镜像名称或标签..."
          clearable
          @keyup.enter="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="handleSearch">
          <el-icon><Search /></el-icon>
          搜索
        </el-button>
      </div>
      <el-button @click="fetchImages" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <!-- 镜像表格 -->
    <div class="table-container">
      <el-table
        :data="images"
        v-loading="loading"
        stripe
        style="width: 100%"
        empty-text="暂无镜像数据"
      >
        <el-table-column label="镜像名称" min-width="200">
          <template #default="{ row }">
            <div class="image-name-cell">
              <span class="name">{{ row.name }}</span>
              <span class="tag">:{{ row.tag }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="摘要" min-width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.digest" placement="top">
              <code class="digest">{{ truncateDigest(row.digest) }}</code>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            <span class="size">{{ formatBytes(row.size) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="层数" width="80" align="center">
          <template #default="{ row }">
            <span>{{ row.layers?.length || 0 }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span class="date">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <div class="actions">
              <el-button size="small" text type="primary" @click="showDetail(row)">
                <el-icon><View /></el-icon>
                详情
              </el-button>
              <el-button size="small" text type="primary" @click="copyPullCommand(row)">
                <el-icon><CopyDocument /></el-icon>
                复制
              </el-button>
              <el-button size="small" text type="danger" @click="deleteImage(row)">
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页 -->
    <Pagination
      v-model:currentPage="currentPage"
      v-model:pageSize="pageSize"
      :total="total"
      @change="handlePageChange"
    />

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`镜像详情 - ${selectedImage?.name}:${selectedImage?.tag}`"
      width="700px"
      class="detail-dialog"
    >
      <div class="detail-content" v-if="selectedImage">
        <div class="detail-section">
          <h4>基本信息</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="label">镜像名称</span>
              <span class="value">{{ selectedImage.name }}</span>
            </div>
            <div class="detail-item">
              <span class="label">标签</span>
              <span class="value tag-value">{{ selectedImage.tag }}</span>
            </div>
            <div class="detail-item">
              <span class="label">大小</span>
              <span class="value">{{ formatBytes(selectedImage.size) }}</span>
            </div>
            <div class="detail-item">
              <span class="label">创建时间</span>
              <span class="value">{{ formatDate(selectedImage.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4>摘要</h4>
          <code class="digest-full">{{ selectedImage.digest }}</code>
        </div>

        <div class="detail-section">
          <h4>拉取命令</h4>
          <div class="pull-command">
            <code>docker pull localhost:8080/{{ selectedImage.name }}:{{ selectedImage.tag }}</code>
            <el-button size="small" type="primary" @click="copyPullCommand(selectedImage)">
              <el-icon><CopyDocument /></el-icon>
              复制
            </el-button>
          </div>
        </div>

        <div class="detail-section" v-if="selectedImage.layers?.length">
          <h4>镜像层 ({{ selectedImage.layers.length }})</h4>
          <div class="layers-list">
            <div class="layer-item" v-for="(layer, index) in selectedImage.layers" :key="layer.digest">
              <span class="layer-index">#{{ index + 1 }}</span>
              <div class="layer-info">
                <el-tooltip :content="layer.digest" placement="top">
                  <code class="layer-digest">{{ truncateDigest(layer.digest) }}</code>
                </el-tooltip>
                <span class="layer-size">{{ formatBytes(layer.size) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.images-page {
  animation: fadeIn 0.3s ease-out;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  gap: 16px;
}

.search-box {
  display: flex;
  gap: 12px;
  flex: 1;
  max-width: 500px;
}

.search-input {
  flex: 1;
}

.table-container {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.image-name-cell {
  display: flex;
  align-items: baseline;
}

.image-name-cell .name {
  color: var(--text-color);
  font-weight: 500;
}

.image-name-cell .tag {
  color: var(--highlight-color);
  font-family: var(--font-mono);
  font-size: 13px;
}

.digest {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--muted-text);
  background-color: var(--bg-color);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.size {
  font-family: var(--font-mono);
  color: var(--highlight-color);
}

.date {
  color: var(--muted-text);
  font-size: 13px;
}

.actions {
  display: flex;
  gap: 4px;
}

/* 详情对话框样式 */
.detail-content {
  color: var(--text-color);
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section:last-child {
  margin-bottom: 0;
}

.detail-section h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--muted-text);
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 8px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-item .label {
  font-size: 12px;
  color: var(--muted-text);
}

.detail-item .value {
  font-size: 14px;
  color: var(--text-color);
}

.detail-item .tag-value {
  color: var(--highlight-color);
  font-family: var(--font-mono);
}

.digest-full {
  display: block;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--muted-text);
  background-color: var(--bg-color);
  padding: 12px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.pull-command {
  display: flex;
  align-items: center;
  gap: 12px;
  background-color: var(--bg-color);
  padding: 12px;
  border-radius: var(--radius-sm);
}

.pull-command code {
  flex: 1;
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--highlight-color);
}

.layers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.layer-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background-color: var(--bg-color);
  border-radius: var(--radius-sm);
}

.layer-index {
  font-size: 12px;
  color: var(--muted-text);
  min-width: 30px;
}

.layer-info {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.layer-digest {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--muted-text);
}

.layer-size {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--highlight-color);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* 详情对话框样式 */
.detail-dialog :deep(.el-dialog__title) {
  color: var(--text-color, #e6edf3) !important;
  font-weight: 600;
}

.detail-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid var(--border-color, #30363d);
  padding: 16px 20px;
}

.detail-dialog :deep(.el-dialog__body) {
  color: var(--text-color, #e6edf3);
  padding: 20px;
}
</style>
