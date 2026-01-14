<template>
  <div class="share-container">
    <div class="page-header">
      <div class="header-left">
        <h1>分享管理</h1>
        <p>管理镜像分享链接</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Share /></el-icon>
        创建分享
      </el-button>
    </div>

    <el-card class="share-list">
      <el-table :data="shareLinks" v-loading="loading" stripe>
        <el-table-column prop="image_ref" label="镜像" min-width="200">
          <template #default="{ row }">
            <div class="image-ref">
              <el-icon><Box /></el-icon>
              <span>{{ row.image_ref }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="分享码" width="180">
          <template #default="{ row }">
            <el-tooltip content="点击复制链接" placement="top">
              <el-button link @click="copyShareLink(row.code)">
                {{ row.code }}
              </el-button>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="密码保护" width="100">
          <template #default="{ row }">
            <el-tag :type="row.require_password ? 'success' : 'info'" size="small">
              {{ row.require_password ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="使用次数" width="120">
          <template #default="{ row }">
            {{ row.usage_count }} / {{ row.max_usage || '∞' }}
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="180">
          <template #default="{ row }">
            <span :class="{ 'expired': isExpired(row.expires_at) }">
              {{ formatDate(row.expires_at) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" @click="copyShareLink(row.code)">复制</el-button>
            <el-button size="small" type="danger" @click="revokeLink(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchShareLinks"
          @current-change="fetchShareLinks"
        />
      </div>
    </el-card>

    <!-- Create Share Dialog -->
    <el-dialog v-model="showCreateDialog" title="创建分享链接" width="500px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="镜像" prop="image_ref">
          <el-input v-model="createForm.image_ref" placeholder="如 myapp:latest" />
        </el-form-item>
        <el-form-item label="访问密码">
          <el-input 
            v-model="createForm.password" 
            type="password" 
            placeholder="可选，留空则无需密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="最大访问次数">
          <el-input-number 
            v-model="createForm.max_usage" 
            :min="0" 
            :max="1000"
            placeholder="0 表示不限制"
          />
          <span class="form-tip">0 表示不限制</span>
        </el-form-item>
        <el-form-item label="有效期">
          <el-select v-model="createForm.expires_in" style="width: 100%">
            <el-option label="1 小时" value="1h" />
            <el-option label="6 小时" value="6h" />
            <el-option label="24 小时" value="24h" />
            <el-option label="7 天" value="168h" />
            <el-option label="30 天" value="720h" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createShare">创建</el-button>
      </template>
    </el-dialog>

    <!-- Share Created Dialog -->
    <el-dialog v-model="showCreatedDialog" title="分享链接已创建" width="500px">
      <el-alert type="success" :closable="false" style="margin-bottom: 16px">
        分享链接创建成功！请复制以下链接分享给他人。
      </el-alert>
      <div class="share-url-box">
        <el-input v-model="createdShareUrl" readonly>
          <template #append>
            <el-button @click="copyCreatedUrl">复制</el-button>
          </template>
        </el-input>
      </div>
      <template #footer>
        <el-button type="primary" @click="showCreatedDialog = false">完成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Share, Box } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface ShareLink {
  id: number
  code: string
  image_ref: string
  require_password: boolean
  max_usage: number
  usage_count: number
  expires_at: string
  created_at: string
}

const loading = ref(false)
const shareLinks = ref<ShareLink[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const showCreateDialog = ref(false)
const creating = ref(false)
const createFormRef = ref()
const createForm = reactive({
  image_ref: '',
  password: '',
  max_usage: 0,
  expires_in: '24h'
})
const createRules = {
  image_ref: [
    { required: true, message: '请输入镜像名称', trigger: 'blur' }
  ]
}

const showCreatedDialog = ref(false)
const createdShareUrl = ref('')

onMounted(() => {
  fetchShareLinks()
})

async function fetchShareLinks() {
  loading.value = true
  try {
    const response = await request.get('/api/v1/share', {
      params: {
        page: pagination.page,
        page_size: pagination.pageSize
      }
    })
    shareLinks.value = response.data.links || []
    pagination.total = response.data.total || 0
  } catch (error) {
    console.error('获取分享链接失败:', error)
  } finally {
    loading.value = false
  }
}

async function createShare() {
  if (!createFormRef.value) return
  
  try {
    await createFormRef.value.validate()
  } catch {
    return
  }

  creating.value = true
  try {
    const response = await request.post('/api/v1/share', createForm)
    createdShareUrl.value = response.data.share_url
    showCreateDialog.value = false
    showCreatedDialog.value = true
    
    // Reset form
    createForm.image_ref = ''
    createForm.password = ''
    createForm.max_usage = 0
    createForm.expires_in = '24h'
    
    fetchShareLinks()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

async function revokeLink(link: ShareLink) {
  try {
    await ElMessageBox.confirm(
      `确定要撤销此分享链接吗？撤销后链接将立即失效。`,
      '撤销确认',
      { type: 'warning' }
    )
  } catch {
    return
  }

  try {
    await request.delete(`/api/v1/share/${link.code}`)
    ElMessage.success('分享链接已撤销')
    fetchShareLinks()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '撤销失败')
  }
}

function copyShareLink(code: string) {
  const url = `${window.location.origin}/s/${code}`
  navigator.clipboard.writeText(url)
  ElMessage.success('链接已复制到剪贴板')
}

function copyCreatedUrl() {
  navigator.clipboard.writeText(createdShareUrl.value)
  ElMessage.success('链接已复制到剪贴板')
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '永不过期'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function isExpired(dateStr: string): boolean {
  if (!dateStr) return false
  return new Date(dateStr) < new Date()
}
</script>

<style scoped>
.share-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  margin: 0 0 8px 0;
}

.header-left p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin: 0;
}

.share-list {
  background: var(--bg-secondary, #1a1f3a);
}

.image-ref {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-ref .el-icon {
  color: var(--primary, #00d4ff);
}

.expired {
  color: #ff3366;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.form-tip {
  margin-left: 8px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  font-size: 12px;
}

.share-url-box {
  margin: 16px 0;
}
</style>
