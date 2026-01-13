<template>
  <div class="audit-container">
    <div class="page-header">
      <h1>审计日志</h1>
      <p>查看系统安全事件和访问记录</p>
    </div>

    <el-card class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="事件类型">
          <el-select v-model="filters.eventType" placeholder="全部" clearable>
            <el-option label="登录成功" value="login_success" />
            <el-option label="登录失败" value="auth_failure" />
            <el-option label="系统锁定" value="system_locked" />
            <el-option label="系统解锁" value="system_unlocked" />
            <el-option label="未授权访问" value="unauthorized_access" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filters.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchLogs">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
          <el-button type="success" @click="exportLogs">导出</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="logs-card">
      <el-table :data="logs" v-loading="loading" stripe>
        <el-table-column prop="timestamp" label="时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="getLevelType(row.level)" size="small">
              {{ row.level }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="event" label="事件" width="150" />
        <el-table-column prop="username" label="用户" width="120" />
        <el-table-column prop="ip_address" label="IP地址" width="140" />
        <el-table-column prop="action" label="操作" width="100" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源" min-width="200" show-overflow-tooltip />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchLogs"
          @current-change="fetchLogs"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

interface AuditLog {
  id: number
  timestamp: string
  level: string
  event: string
  user_id?: number
  username?: string
  ip_address: string
  resource: string
  action: string
  status: string
  details?: Record<string, any>
  blockchain_hash?: string
}

const loading = ref(false)
const logs = ref<AuditLog[]>([])

const filters = reactive({
  eventType: '',
  dateRange: null as [Date, Date] | null
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

onMounted(() => {
  fetchLogs()
})

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: pagination.page,
      page_size: pagination.pageSize
    }

    if (filters.eventType) {
      params.event_type = filters.eventType
    }

    if (filters.dateRange) {
      params.start_date = filters.dateRange[0].toISOString()
      params.end_date = filters.dateRange[1].toISOString()
    }

    const response = await request.get('/api/v1/audit/logs', { params })
    logs.value = response.data.logs || []
    pagination.total = response.data.total || 0
  } catch (error) {
    console.error('Failed to fetch audit logs:', error)
    // Use mock data for demo
    logs.value = generateMockLogs()
    pagination.total = logs.value.length
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  filters.eventType = ''
  filters.dateRange = null
  pagination.page = 1
  fetchLogs()
}

async function exportLogs() {
  try {
    const response = await request.get('/api/v1/audit/logs/export', {
      responseType: 'blob'
    })
    
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `audit-logs-${new Date().toISOString().split('T')[0]}.json`)
    document.body.appendChild(link)
    link.click()
    link.remove()
    
    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败')
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function getLevelType(level: string): string {
  switch (level) {
    case 'critical':
      return 'danger'
    case 'error':
      return 'danger'
    case 'warn':
      return 'warning'
    case 'info':
      return 'info'
    default:
      return 'info'
  }
}

function generateMockLogs(): AuditLog[] {
  return [
    {
      id: 1,
      timestamp: new Date().toISOString(),
      level: 'info',
      event: 'login_success',
      username: 'admin',
      ip_address: '192.168.1.100',
      resource: '/api/v1/auth/login',
      action: 'login',
      status: 'success'
    },
    {
      id: 2,
      timestamp: new Date(Date.now() - 3600000).toISOString(),
      level: 'warn',
      event: 'auth_failure',
      username: 'unknown',
      ip_address: '192.168.1.50',
      resource: '/api/v1/auth/login',
      action: 'login',
      status: 'failure'
    }
  ]
}
</script>

<style scoped>
.audit-container {
  padding: 20px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  margin: 0 0 8px 0;
}

.page-header p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin: 0;
}

.filter-card {
  margin-bottom: 20px;
}

.logs-card {
  background: var(--bg-secondary, #1a1f3a);
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
