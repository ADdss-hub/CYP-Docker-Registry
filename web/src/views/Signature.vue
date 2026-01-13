<template>
  <div class="signature-page">
    <div class="page-header">
      <h1>镜像签名管理</h1>
      <p class="subtitle">管理容器镜像的数字签名，确保镜像完整性和来源可信</p>
    </div>

    <div class="actions-bar">
      <el-button type="primary" @click="showSignDialog = true">
        <el-icon><Edit /></el-icon>
        签名镜像
      </el-button>
      <el-button @click="showVerifyDialog = true">
        <el-icon><Check /></el-icon>
        验证签名
      </el-button>
      <el-button @click="loadSignatures">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-card class="signatures-card">
      <template #header>
        <div class="card-header">
          <span>签名列表</span>
          <el-tag type="info">共 {{ total }} 个签名</el-tag>
        </div>
      </template>

      <el-table :data="signatures" v-loading="loading" stripe>
        <el-table-column prop="image_ref" label="镜像" min-width="200">
          <template #default="{ row }">
            <div class="image-ref">
              <el-icon><Box /></el-icon>
              <span>{{ row.image_ref }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="signed_by" label="签名者" width="120" />
        <el-table-column prop="signed_at" label="签名时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.signed_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="verified" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.verified ? 'success' : 'danger'">
              {{ row.verified ? '已验证' : '未验证' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="key_id" label="密钥ID" width="150">
          <template #default="{ row }">
            <code>{{ row.key_id || 'default' }}</code>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="verifySignature(row.image_ref)">验证</el-button>
            <el-button size="small" type="danger" @click="deleteSignature(row.image_ref)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadSignatures"
          @current-change="loadSignatures"
        />
      </div>
    </el-card>

    <!-- Sign Dialog -->
    <el-dialog v-model="showSignDialog" title="签名镜像" width="500px">
      <el-form :model="signForm" label-width="100px">
        <el-form-item label="镜像引用" required>
          <el-input v-model="signForm.image_ref" placeholder="例如: myregistry/myimage:v1.0" />
        </el-form-item>
        <el-form-item label="密钥ID">
          <el-input v-model="signForm.key_id" placeholder="留空使用默认密钥" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSignDialog = false">取消</el-button>
        <el-button type="primary" @click="signImage" :loading="signing">签名</el-button>
      </template>
    </el-dialog>

    <!-- Verify Dialog -->
    <el-dialog v-model="showVerifyDialog" title="验证签名" width="500px">
      <el-form :model="verifyForm" label-width="100px">
        <el-form-item label="镜像引用" required>
          <el-input v-model="verifyForm.image_ref" placeholder="例如: myregistry/myimage:v1.0" />
        </el-form-item>
      </el-form>
      <div v-if="verifyResult" class="verify-result">
        <el-result
          :icon="verifyResult.verified ? 'success' : 'error'"
          :title="verifyResult.verified ? '签名验证通过' : '签名验证失败'"
          :sub-title="verifyResult.error || ''"
        >
          <template #extra v-if="verifyResult.signature">
            <div class="signature-details">
              <p><strong>签名者:</strong> {{ verifyResult.signature.signed_by }}</p>
              <p><strong>签名时间:</strong> {{ formatDate(verifyResult.signature.signed_at) }}</p>
            </div>
          </template>
        </el-result>
      </div>
      <template #footer>
        <el-button @click="showVerifyDialog = false; verifyResult = null">关闭</el-button>
        <el-button type="primary" @click="doVerify" :loading="verifying">验证</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Check, Refresh, Box } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface Signature {
  image_ref: string
  digest: string
  signature: string
  signed_by: string
  signed_at: string
  key_id: string
  verified: boolean
}

interface VerifyResult {
  image_ref: string
  verified: boolean
  signature?: Signature
  error?: string
}

const loading = ref(false)
const signing = ref(false)
const verifying = ref(false)
const signatures = ref<Signature[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const showSignDialog = ref(false)
const showVerifyDialog = ref(false)
const signForm = ref({ image_ref: '', key_id: '' })
const verifyForm = ref({ image_ref: '' })
const verifyResult = ref<VerifyResult | null>(null)

const formatDate = (date: string) => {
  return new Date(date).toLocaleString('zh-CN')
}

const loadSignatures = async () => {
  loading.value = true
  try {
    const res = await request.get('/api/v1/signatures', {
      params: { page: page.value, page_size: pageSize.value }
    })
    signatures.value = res.data.signatures || []
    total.value = res.data.total || 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载签名列表失败')
  } finally {
    loading.value = false
  }
}

const signImage = async () => {
  if (!signForm.value.image_ref) {
    ElMessage.warning('请输入镜像引用')
    return
  }
  signing.value = true
  try {
    await request.post('/api/v1/signatures', signForm.value)
    ElMessage.success('镜像签名成功')
    showSignDialog.value = false
    signForm.value = { image_ref: '', key_id: '' }
    loadSignatures()
  } catch (error: any) {
    ElMessage.error(error.message || '签名失败')
  } finally {
    signing.value = false
  }
}

const verifySignature = async (imageRef: string) => {
  verifyForm.value.image_ref = imageRef
  showVerifyDialog.value = true
  await doVerify()
}

const doVerify = async () => {
  if (!verifyForm.value.image_ref) {
    ElMessage.warning('请输入镜像引用')
    return
  }
  verifying.value = true
  try {
    const res = await request.post('/api/v1/signatures/verify', verifyForm.value)
    verifyResult.value = res.data
  } catch (error: any) {
    verifyResult.value = { image_ref: verifyForm.value.image_ref, verified: false, error: error.message }
  } finally {
    verifying.value = false
  }
}

const deleteSignature = async (imageRef: string) => {
  try {
    await ElMessageBox.confirm('确定要删除此签名吗？', '确认删除', { type: 'warning' })
    await request.delete(`/api/v1/signatures/${encodeURIComponent(imageRef)}`)
    ElMessage.success('签名已删除')
    loadSignatures()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

onMounted(() => {
  loadSignatures()
})
</script>

<style scoped>
.signature-page { padding: 20px; }
.page-header { margin-bottom: 20px; }
.page-header h1 { margin: 0 0 8px 0; font-size: 24px; }
.subtitle { color: #666; margin: 0; }
.actions-bar { margin-bottom: 20px; display: flex; gap: 10px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.image-ref { display: flex; align-items: center; gap: 8px; }
.pagination { margin-top: 20px; display: flex; justify-content: flex-end; }
.verify-result { margin-top: 20px; }
.signature-details { text-align: left; }
.signature-details p { margin: 5px 0; }
</style>
