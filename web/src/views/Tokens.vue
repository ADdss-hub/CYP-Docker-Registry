<template>
  <div class="tokens-container">
    <div class="page-header">
      <div class="header-left">
        <h1>访问令牌</h1>
        <p>管理个人访问令牌 (PAT)</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Key /></el-icon>
        创建令牌
      </el-button>
    </div>

    <el-alert type="info" :closable="false" style="margin-bottom: 20px">
      个人访问令牌可用于 API 认证和 Docker CLI 登录。请妥善保管您的令牌。
    </el-alert>

    <el-card class="tokens-list">
      <el-table :data="tokens" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" width="200">
          <template #default="{ row }">
            <div class="token-name">
              <el-icon><Key /></el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="scopes" label="权限范围" min-width="200">
          <template #default="{ row }">
            <el-tag 
              v-for="scope in row.scopes" 
              :key="scope" 
              size="small" 
              style="margin-right: 4px"
            >
              {{ scope }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_used_at" label="最后使用" width="180">
          <template #default="{ row }">
            {{ row.last_used_at ? formatDate(row.last_used_at) : '从未使用' }}
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="180">
          <template #default="{ row }">
            <span :class="{ 'expired': isExpired(row.expires_at) }">
              {{ row.expires_at ? formatDate(row.expires_at) : '永不过期' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="deleteToken(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create Token Dialog -->
    <el-dialog v-model="showCreateDialog" title="创建访问令牌" width="500px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="令牌名称" prop="name">
          <el-input v-model="createForm.name" placeholder="如 ci-deploy" />
        </el-form-item>
        <el-form-item label="权限范围" prop="scopes">
          <el-checkbox-group v-model="createForm.scopes">
            <el-checkbox label="registry:read">镜像读取</el-checkbox>
            <el-checkbox label="registry:write">镜像写入</el-checkbox>
            <el-checkbox label="registry:delete">镜像删除</el-checkbox>
            <el-checkbox label="admin:read">管理读取</el-checkbox>
            <el-checkbox label="admin:write">管理写入</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="有效期">
          <el-select v-model="createForm.expires_in" style="width: 100%">
            <el-option label="7 天" value="7d" />
            <el-option label="30 天" value="30d" />
            <el-option label="90 天" value="90d" />
            <el-option label="1 年" value="365d" />
            <el-option label="永不过期" value="" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createToken">创建</el-button>
      </template>
    </el-dialog>

    <!-- Token Created Dialog -->
    <el-dialog v-model="showCreatedDialog" title="令牌已创建" width="600px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" style="margin-bottom: 16px">
        <strong>请立即复制并保存此令牌！</strong><br>
        出于安全考虑，令牌只会显示一次，关闭此对话框后将无法再次查看。
      </el-alert>
      <div class="token-display">
        <el-input v-model="createdToken" readonly type="textarea" :rows="2" />
        <el-button type="primary" @click="copyToken" style="margin-top: 12px">
          复制令牌
        </el-button>
      </div>
      <div class="usage-example">
        <h4>使用示例</h4>
        <p>Docker CLI 登录：</p>
        <code>docker login -u &lt;username&gt; -p {{ createdToken }} {{ host }}</code>
        <p style="margin-top: 12px">API 请求：</p>
        <code>curl -H "Authorization: Token {{ createdToken }}" {{ host }}/api/images</code>
      </div>
      <template #footer>
        <el-button type="primary" @click="showCreatedDialog = false">我已保存令牌</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Key } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface Token {
  id: number
  name: string
  scopes: string[]
  last_used_at: string
  expires_at: string
  created_at: string
}

const loading = ref(false)
const tokens = ref<Token[]>([])

const showCreateDialog = ref(false)
const creating = ref(false)
const createFormRef = ref()
const createForm = reactive({
  name: '',
  scopes: ['registry:read'] as string[],
  expires_in: '30d'
})
const createRules = {
  name: [
    { required: true, message: '请输入令牌名称', trigger: 'blur' }
  ],
  scopes: [
    { required: true, message: '请选择至少一个权限', trigger: 'change', type: 'array', min: 1 }
  ]
}

const showCreatedDialog = ref(false)
const createdToken = ref('')

const host = computed(() => window.location.host)

onMounted(() => {
  fetchTokens()
})

async function fetchTokens() {
  loading.value = true
  try {
    const response = await request.get('/api/v1/tokens')
    tokens.value = response.data.tokens || []
  } catch (error) {
    console.error('Failed to fetch tokens:', error)
  } finally {
    loading.value = false
  }
}

async function createToken() {
  if (!createFormRef.value) return
  
  try {
    await createFormRef.value.validate()
  } catch {
    return
  }

  creating.value = true
  try {
    const response = await request.post('/api/v1/tokens', createForm)
    createdToken.value = response.data.plain_token
    showCreateDialog.value = false
    showCreatedDialog.value = true
    
    // Reset form
    createForm.name = ''
    createForm.scopes = ['registry:read']
    createForm.expires_in = '30d'
    
    fetchTokens()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

async function deleteToken(token: Token) {
  try {
    await ElMessageBox.confirm(
      `确定要删除令牌 "${token.name}" 吗？删除后使用此令牌的应用将无法访问。`,
      '删除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }

  try {
    await request.delete(`/api/v1/tokens/${token.id}`)
    ElMessage.success('令牌已删除')
    fetchTokens()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '删除失败')
  }
}

function copyToken() {
  navigator.clipboard.writeText(createdToken.value)
  ElMessage.success('令牌已复制到剪贴板')
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function isExpired(dateStr: string): boolean {
  if (!dateStr) return false
  return new Date(dateStr) < new Date()
}
</script>

<style scoped>
.tokens-container {
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

.tokens-list {
  background: var(--bg-secondary, #1a1f3a);
}

.token-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.token-name .el-icon {
  color: var(--primary, #00d4ff);
}

.expired {
  color: #ff3366;
}

.token-display {
  background: rgba(0, 0, 0, 0.3);
  padding: 16px;
  border-radius: 8px;
}

.usage-example {
  margin-top: 20px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
}

.usage-example h4 {
  color: var(--text-primary, #ffffff);
  margin: 0 0 12px 0;
}

.usage-example p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin: 8px 0 4px 0;
  font-size: 13px;
}

.usage-example code {
  display: block;
  background: rgba(0, 212, 255, 0.1);
  color: #00d4ff;
  padding: 8px 12px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}
</style>
