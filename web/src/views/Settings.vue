<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Edit, View, Hide, Warning } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface Credential {
  registry: string
  username: string
  password: string
  created_at: string
  updated_at: string
}

interface SyncRecord {
  id: string
  image_name: string
  image_tag: string
  target_registry: string
  status: string
  started_at: string
  completed_at: string
  error_message: string
}

const loading = ref(false)
const credentials = ref<Credential[]>([])
const syncHistory = ref<SyncRecord[]>([])

// 凭证表单
const credDialogVisible = ref(false)
const credForm = ref({
  registry: '',
  username: '',
  password: ''
})
const isEditMode = ref(false)
const showPassword = ref(false)

// 同步表单
const syncDialogVisible = ref(false)
const syncForm = ref({
  image_name: '',
  image_tag: '',
  target_registry: '',
  target_name: ''
})

const fetchCredentials = async () => {
  try {
    const res = await request.get('/credentials')
    credentials.value = res.data?.credentials || []
  } catch (error) {
    console.error('获取凭证列表失败:', error)
  }
}

const fetchSyncHistory = async () => {
  try {
    const res = await request.get('/sync/history', { params: { page: 1, page_size: 10 } })
    syncHistory.value = res.data?.records || []
  } catch (error) {
    console.error('获取同步历史失败:', error)
  }
}

const fetchAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchCredentials(), fetchSyncHistory()])
  } finally {
    loading.value = false
  }
}

const showAddCredential = () => {
  isEditMode.value = false
  credForm.value = { registry: '', username: '', password: '' }
  showPassword.value = false
  credDialogVisible.value = true
}

const showEditCredential = (cred: Credential) => {
  isEditMode.value = true
  credForm.value = {
    registry: cred.registry,
    username: cred.username,
    password: ''
  }
  showPassword.value = false
  credDialogVisible.value = true
}

const saveCredential = async () => {
  if (!credForm.value.registry || !credForm.value.username) {
    ElMessage.warning('请填写仓库地址和用户名')
    return
  }
  if (!isEditMode.value && !credForm.value.password) {
    ElMessage.warning('请填写密码')
    return
  }

  try {
    await request.post('/credentials', credForm.value)
    ElMessage.success(isEditMode.value ? '凭证更新成功' : '凭证添加成功')
    credDialogVisible.value = false
    fetchCredentials()
  } catch (error) {
    ElMessage.error('保存凭证失败')
  }
}

const deleteCredential = async (registry: string) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 "${registry}" 的凭证吗？`,
      '删除确认',
      { type: 'warning' }
    )
    await request.delete(`/credentials/${encodeURIComponent(registry)}`)
    ElMessage.success('凭证已删除')
    fetchCredentials()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除凭证失败')
    }
  }
}

const showSyncDialog = () => {
  syncForm.value = {
    image_name: '',
    image_tag: 'latest',
    target_registry: credentials.value[0]?.registry || '',
    target_name: ''
  }
  syncDialogVisible.value = true
}

const startSync = async () => {
  if (!syncForm.value.image_name || !syncForm.value.image_tag || !syncForm.value.target_registry) {
    ElMessage.warning('请填写完整信息')
    return
  }

  try {
    await request.post('/sync', syncForm.value)
    ElMessage.success('同步任务已启动')
    syncDialogVisible.value = false
    fetchSyncHistory()
  } catch (error) {
    ElMessage.error('启动同步失败')
  }
}

const retrySync = async (id: string) => {
  try {
    await request.post(`/sync/retry/${id}`)
    ElMessage.success('重试任务已启动')
    fetchSyncHistory()
  } catch (error) {
    ElMessage.error('重试失败')
  }
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const getStatusType = (status: string): string => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'danger'
    case 'running': return 'primary'
    default: return 'info'
  }
}

const getStatusText = (status: string): string => {
  switch (status) {
    case 'completed': return '已完成'
    case 'failed': return '失败'
    case 'running': return '进行中'
    case 'pending': return '等待中'
    default: return status
  }
}

onMounted(() => {
  fetchAll()
})
</script>

<template>
  <div class="settings-page" v-loading="loading">
    <!-- 公共仓库凭证 -->
    <div class="section">
      <div class="section-header">
        <h3>公共仓库凭证</h3>
        <el-button type="primary" @click="showAddCredential">
          <el-icon><Plus /></el-icon>
          添加凭证
        </el-button>
      </div>
      <div class="credentials-list">
        <div class="credential-card" v-for="cred in credentials" :key="cred.registry">
          <div class="cred-header">
            <div class="cred-registry">{{ cred.registry }}</div>
            <div class="cred-actions">
              <el-button size="small" text type="primary" @click="showEditCredential(cred)">
                <el-icon><Edit /></el-icon>编辑
              </el-button>
              <el-button size="small" text type="danger" @click="deleteCredential(cred.registry)">
                <el-icon><Delete /></el-icon>删除
              </el-button>
            </div>
          </div>
          <div class="cred-info">
            <div class="cred-item">
              <span class="label">用户名：</span>
              <span class="value">{{ cred.username }}</span>
            </div>
            <div class="cred-item">
              <span class="label">密码：</span>
              <span class="value password">********</span>
            </div>
            <div class="cred-item">
              <span class="label">创建时间：</span>
              <span class="value">{{ formatDate(cred.created_at) }}</span>
            </div>
          </div>
        </div>
        <div class="empty-state" v-if="credentials.length === 0">
          <span>暂无凭证配置，点击"添加凭证"开始配置</span>
        </div>
      </div>
    </div>

    <!-- 镜像同步 -->
    <div class="section">
      <div class="section-header">
        <h3>镜像同步</h3>
        <el-button type="primary" @click="showSyncDialog" :disabled="credentials.length === 0">
          <el-icon><Plus /></el-icon>
          新建同步
        </el-button>
      </div>
      <div class="sync-tip" v-if="credentials.length === 0">
        <el-icon><Warning /></el-icon>
        请先添加公共仓库凭证后再进行镜像同步
      </div>
      <div class="sync-history" v-else>
        <div class="history-title">同步历史</div>
        <el-table :data="syncHistory" stripe style="width: 100%" empty-text="暂无同步记录">
          <el-table-column label="镜像" min-width="180">
            <template #default="{ row }">
              <span class="image-name">{{ row.image_name }}:{{ row.image_tag }}</span>
            </template>
          </el-table-column>
          <el-table-column label="目标仓库" min-width="150">
            <template #default="{ row }">
              <span>{{ row.target_registry }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="开始时间" width="170">
            <template #default="{ row }">
              <span class="date">{{ formatDate(row.started_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button 
                v-if="row.status === 'failed'" 
                size="small" 
                text 
                type="primary" 
                @click="retrySync(row.id)"
              >
                重试
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 凭证编辑对话框 -->
    <el-dialog
      v-model="credDialogVisible"
      :title="isEditMode ? '编辑凭证' : '添加凭证'"
      width="500px"
    >
      <el-form :model="credForm" label-width="100px">
        <el-form-item label="仓库地址" required>
          <el-input 
            v-model="credForm.registry" 
            placeholder="例如：docker.io" 
            :disabled="isEditMode"
          />
        </el-form-item>
        <el-form-item label="用户名" required>
          <el-input v-model="credForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" :required="!isEditMode">
          <el-input 
            v-model="credForm.password" 
            :type="showPassword ? 'text' : 'password'"
            :placeholder="isEditMode ? '留空则不修改密码' : '请输入密码'"
          >
            <template #suffix>
              <el-icon class="password-toggle" @click="showPassword = !showPassword">
                <View v-if="showPassword" />
                <Hide v-else />
              </el-icon>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="credDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveCredential">保存</el-button>
      </template>
    </el-dialog>

    <!-- 同步对话框 -->
    <el-dialog v-model="syncDialogVisible" title="新建同步任务" width="500px">
      <el-form :model="syncForm" label-width="100px">
        <el-form-item label="镜像名称" required>
          <el-input v-model="syncForm.image_name" placeholder="例如：myapp" />
        </el-form-item>
        <el-form-item label="镜像标签" required>
          <el-input v-model="syncForm.image_tag" placeholder="例如：latest" />
        </el-form-item>
        <el-form-item label="目标仓库" required>
          <el-select v-model="syncForm.target_registry" placeholder="选择目标仓库" style="width: 100%">
            <el-option 
              v-for="cred in credentials" 
              :key="cred.registry" 
              :label="cred.registry" 
              :value="cred.registry" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="目标名称">
          <el-input v-model="syncForm.target_name" placeholder="留空则使用原名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="syncDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="startSync">开始同步</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.settings-page {
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

.credentials-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.credential-card {
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
  transition: border-color 0.2s;
}

.credential-card:hover {
  border-color: var(--highlight-color);
}

.cred-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.cred-registry {
  font-weight: 500;
  color: var(--text-color);
  font-family: var(--font-mono);
}

.cred-actions {
  display: flex;
  gap: 8px;
}

.cred-info {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.cred-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
}

.cred-item .label {
  color: var(--muted-text);
}

.cred-item .value {
  color: var(--text-color);
}

.cred-item .value.password {
  font-family: var(--font-mono);
  color: var(--muted-text);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--muted-text);
}

.sync-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background-color: rgba(210, 153, 34, 0.1);
  border-radius: var(--radius-sm);
  color: var(--warning-color);
}

.sync-history {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-title {
  font-size: 14px;
  color: var(--muted-text);
}

.image-name {
  font-family: var(--font-mono);
  color: var(--highlight-color);
}

.date {
  color: var(--muted-text);
  font-size: 13px;
}

.password-toggle {
  cursor: pointer;
  color: var(--muted-text);
}

.password-toggle:hover {
  color: var(--highlight-color);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
