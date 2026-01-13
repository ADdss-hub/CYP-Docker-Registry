<template>
  <div class="tuf-container">
    <el-card class="status-card">
      <template #header>
        <div class="card-header">
          <span>TUF 仓库状态</span>
          <el-button
            v-if="!status.initialized"
            type="primary"
            size="small"
            @click="initializeRepo"
            :loading="initializing"
          >
            初始化仓库
          </el-button>
          <el-button
            v-else
            type="primary"
            size="small"
            @click="refreshTimestamp"
            :loading="refreshing"
          >
            刷新 Timestamp
          </el-button>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="初始化状态">
          <el-tag :type="status.initialized ? 'success' : 'warning'">
            {{ status.initialized ? '已初始化' : '未初始化' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="密钥数量">
          {{ status.key_count }}
        </el-descriptions-item>
        <el-descriptions-item label="Root 版本">
          v{{ status.root_version }}
        </el-descriptions-item>
        <el-descriptions-item label="Root 过期时间">
          <el-tag :type="status.root_expired ? 'danger' : 'success'">
            {{ formatDate(status.root_expires) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Targets 版本">
          v{{ status.targets_version }}
        </el-descriptions-item>
        <el-descriptions-item label="目标数量">
          {{ status.target_count }}
        </el-descriptions-item>
        <el-descriptions-item label="Snapshot 版本">
          v{{ status.snapshot_version }}
        </el-descriptions-item>
        <el-descriptions-item label="Snapshot 过期时间">
          {{ formatDate(status.snapshot_expires) }}
        </el-descriptions-item>
        <el-descriptions-item label="Timestamp 版本">
          v{{ status.timestamp_version }}
        </el-descriptions-item>
        <el-descriptions-item label="Timestamp 过期时间">
          <el-tag :type="status.timestamp_expired ? 'danger' : 'success'">
            {{ formatDate(status.timestamp_expires) }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="warnings.length" class="warnings-section">
        <el-alert
          v-for="(warning, index) in warnings"
          :key="index"
          :title="warning"
          type="warning"
          show-icon
          :closable="false"
          style="margin-top: 10px"
        />
      </div>
    </el-card>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card class="keys-card">
          <template #header>
            <div class="card-header">
              <span>密钥管理</span>
            </div>
          </template>

          <el-table :data="status.keys || []" stripe>
            <el-table-column label="密钥ID" min-width="150">
              <template #default="{ row }">
                <span class="key-id">{{ row.id }}</span>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="100">
              <template #default="{ row }">
                {{ row.type }}
              </template>
            </el-table-column>
            <el-table-column label="角色" width="120">
              <template #default="{ row }">
                <el-tag
                  v-for="role in row.roles"
                  :key="role"
                  size="small"
                  style="margin: 2px"
                >
                  {{ role }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button
                  type="warning"
                  link
                  size="small"
                  @click="rotateKey(row.roles[0])"
                >
                  轮换
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-button
            type="primary"
            style="margin-top: 15px"
            @click="exportKeys"
          >
            导出公钥
          </el-button>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="delegations-card">
          <template #header>
            <div class="card-header">
              <span>委托管理</span>
              <el-button type="primary" size="small" @click="showDelegationDialog = true">
                添加委托
              </el-button>
            </div>
          </template>

          <el-table :data="delegations" stripe>
            <el-table-column label="名称" prop="name" width="120" />
            <el-table-column label="路径" min-width="150">
              <template #default="{ row }">
                <el-tag
                  v-for="path in row.paths"
                  :key="path"
                  size="small"
                  type="info"
                  style="margin: 2px"
                >
                  {{ path }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="阈值" prop="threshold" width="80" />
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button
                  type="danger"
                  link
                  size="small"
                  @click="removeDelegation(row.name)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-if="!delegations.length" description="暂无委托" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="targets-card">
      <template #header>
        <div class="card-header">
          <span>目标文件 ({{ targets.length }})</span>
          <el-upload
            :show-file-list="false"
            :before-upload="handleUpload"
            action=""
          >
            <el-button type="primary" size="small">添加目标</el-button>
          </el-upload>
        </div>
      </template>

      <el-table :data="targets" stripe>
        <el-table-column label="名称" prop="name" min-width="200" />
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ formatBytes(row.length) }}
          </template>
        </el-table-column>
        <el-table-column label="SHA256" min-width="300">
          <template #default="{ row }">
            <el-tooltip :content="row.hashes?.sha256" placement="top">
              <span class="hash-value">{{ truncateHash(row.hashes?.sha256) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="verifyTarget(row.name)">
              验证
            </el-button>
            <el-button type="danger" link size="small" @click="removeTarget(row.name)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!targets.length" description="暂无目标文件" />
    </el-card>

    <!-- 添加委托对话框 -->
    <el-dialog v-model="showDelegationDialog" title="添加委托" width="500px">
      <el-form :model="delegationForm" label-width="100px">
        <el-form-item label="委托名称">
          <el-input v-model="delegationForm.name" placeholder="例如: releases" />
        </el-form-item>
        <el-form-item label="路径模式">
          <el-select
            v-model="delegationForm.paths"
            multiple
            filterable
            allow-create
            placeholder="输入路径模式，如 releases/*"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="签名阈值">
          <el-input-number v-model="delegationForm.threshold" :min="1" :max="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDelegationDialog = false">取消</el-button>
        <el-button type="primary" @click="addDelegation" :loading="addingDelegation">
          添加
        </el-button>
      </template>
    </el-dialog>

    <!-- 添加目标对话框 -->
    <el-dialog v-model="showTargetDialog" title="添加目标" width="500px">
      <el-form :model="targetForm" label-width="100px">
        <el-form-item label="目标名称">
          <el-input v-model="targetForm.name" placeholder="例如: image:v1.0.0" />
        </el-form-item>
        <el-form-item label="文件">
          <span>{{ targetForm.file?.name || '未选择文件' }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTargetDialog = false">取消</el-button>
        <el-button type="primary" @click="submitTarget" :loading="addingTarget">
          添加
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UploadRawFile } from 'element-plus'
import request from '@/api'

interface TUFStatus {
  initialized: boolean
  key_count: number
  root_version: number
  root_expires: string
  root_expired: boolean
  targets_version: number
  targets_expires: string
  target_count: number
  snapshot_version: number
  snapshot_expires: string
  timestamp_version: number
  timestamp_expires: string
  timestamp_expired: boolean
  keys: Array<{
    id: string
    type: string
    roles: string[]
  }>
}

interface TUFTarget {
  name: string
  length: number
  hashes: { sha256: string }
  custom?: Record<string, unknown>
}

interface TUFDelegation {
  name: string
  paths: string[]
  threshold: number
  terminating: boolean
}

const status = ref<TUFStatus>({
  initialized: false,
  key_count: 0,
  root_version: 0,
  root_expires: '',
  root_expired: false,
  targets_version: 0,
  targets_expires: '',
  target_count: 0,
  snapshot_version: 0,
  snapshot_expires: '',
  timestamp_version: 0,
  timestamp_expires: '',
  timestamp_expired: false,
  keys: []
})

const targets = ref<TUFTarget[]>([])
const delegations = ref<TUFDelegation[]>([])
const warnings = ref<string[]>([])

const initializing = ref(false)
const refreshing = ref(false)
const addingDelegation = ref(false)
const addingTarget = ref(false)

const showDelegationDialog = ref(false)
const showTargetDialog = ref(false)

const delegationForm = ref({
  name: '',
  paths: [] as string[],
  threshold: 1
})

const targetForm = ref({
  name: '',
  file: null as File | null
})

const fetchStatus = async () => {
  try {
    const res = await request.get('/api/v1/tuf/status')
    if (res.data.code === 0) {
      status.value = res.data.data
    }
  } catch (error) {
    console.error('获取TUF状态失败', error)
  }
}

const fetchTargets = async () => {
  try {
    const res = await request.get('/api/v1/tuf/targets')
    if (res.data.code === 0) {
      targets.value = res.data.data || []
    }
  } catch (error) {
    console.error('获取目标列表失败', error)
  }
}

const fetchDelegations = async () => {
  try {
    const res = await request.get('/api/v1/tuf/delegations')
    if (res.data.code === 0) {
      delegations.value = res.data.data || []
    }
  } catch (error) {
    console.error('获取委托列表失败', error)
  }
}

const checkExpiry = async () => {
  try {
    const res = await request.get('/api/v1/tuf/expiry')
    if (res.data.code === 0) {
      warnings.value = res.data.warnings || []
    }
  } catch (error) {
    console.error('检查过期状态失败', error)
  }
}

const initializeRepo = async () => {
  initializing.value = true
  try {
    const res = await request.post('/api/v1/tuf/initialize')
    if (res.data.code === 0) {
      ElMessage.success('TUF仓库初始化成功')
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('初始化失败')
  } finally {
    initializing.value = false
  }
}

const refreshTimestamp = async () => {
  refreshing.value = true
  try {
    const res = await request.post('/api/v1/tuf/refresh')
    if (res.data.code === 0) {
      ElMessage.success('Timestamp已刷新')
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('刷新失败')
  } finally {
    refreshing.value = false
  }
}

const rotateKey = async (role: string) => {
  try {
    await ElMessageBox.confirm(
      `确定要轮换 ${role} 角色的密钥吗？这将生成新的密钥对。`,
      '确认轮换',
      { type: 'warning' }
    )

    const res = await request.post(`/api/v1/tuf/keys/rotate/${role}`)
    if (res.data.code === 0) {
      ElMessage.success('密钥轮换成功')
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('密钥轮换失败')
    }
  }
}

const exportKeys = async () => {
  try {
    const res = await request.get('/api/v1/tuf/keys/export')
    if (res.data.code === 0) {
      const blob = new Blob([JSON.stringify(res.data.data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'tuf-public-keys.json'
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success('公钥已导出')
    }
  } catch (error) {
    ElMessage.error('导出失败')
  }
}

const addDelegation = async () => {
  if (!delegationForm.value.name || !delegationForm.value.paths.length) {
    ElMessage.warning('请填写完整信息')
    return
  }

  addingDelegation.value = true
  try {
    const res = await request.post('/api/v1/tuf/delegations', delegationForm.value)
    if (res.data.code === 0) {
      ElMessage.success('委托添加成功')
      showDelegationDialog.value = false
      delegationForm.value = { name: '', paths: [], threshold: 1 }
      await fetchDelegations()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('添加失败')
  } finally {
    addingDelegation.value = false
  }
}

const removeDelegation = async (name: string) => {
  try {
    await ElMessageBox.confirm(`确定要删除委托 "${name}" 吗？`, '确认删除', { type: 'warning' })
    const res = await request.delete(`/api/v1/tuf/delegations/${name}`)
    if (res.data.code === 0) {
      ElMessage.success('委托已删除')
      await fetchDelegations()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleUpload = (file: UploadRawFile) => {
  targetForm.value.file = file
  targetForm.value.name = file.name
  showTargetDialog.value = true
  return false
}

const submitTarget = async () => {
  if (!targetForm.value.name || !targetForm.value.file) {
    ElMessage.warning('请填写完整信息')
    return
  }

  addingTarget.value = true
  try {
    const formData = new FormData()
    formData.append('file', targetForm.value.file)

    const res = await request.post(`/api/v1/tuf/targets/${targetForm.value.name}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    if (res.data.code === 0) {
      ElMessage.success('目标添加成功')
      showTargetDialog.value = false
      targetForm.value = { name: '', file: null }
      await fetchTargets()
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('添加失败')
  } finally {
    addingTarget.value = false
  }
}

const verifyTarget = async (name: string) => {
  ElMessage.info('请上传要验证的文件')
  // 实际实现需要文件上传
}

const removeTarget = async (name: string) => {
  try {
    await ElMessageBox.confirm(`确定要删除目标 "${name}" 吗？`, '确认删除', { type: 'warning' })
    const res = await request.delete(`/api/v1/tuf/targets/${name}`)
    if (res.data.code === 0) {
      ElMessage.success('目标已删除')
      await fetchTargets()
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const formatDate = (date: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024
    i++
  }
  return `${bytes.toFixed(2)} ${units[i]}`
}

const truncateHash = (hash: string) => {
  if (!hash) return '-'
  if (hash.length <= 20) return hash
  return `${hash.slice(0, 10)}...${hash.slice(-10)}`
}

onMounted(() => {
  fetchStatus()
  fetchTargets()
  fetchDelegations()
  checkExpiry()
})
</script>

<style scoped>
.tuf-container {
  padding: 20px;
}

.status-card,
.keys-card,
.delegations-card,
.targets-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.key-id {
  font-family: monospace;
  font-size: 12px;
}

.hash-value {
  font-family: monospace;
  font-size: 12px;
}

.warnings-section {
  margin-top: 15px;
}
</style>
