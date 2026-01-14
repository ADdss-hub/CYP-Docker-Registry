<template>
  <div class="dns-container">
    <div class="page-header">
      <h1>DNS 解析服务</h1>
      <p class="subtitle">输入域名查询 DNS 记录</p>
    </div>

    <el-card class="dns-card">
      <div class="dns-input-section">
        <el-input
          v-model="domain"
          placeholder="请输入域名，例如：example.com"
          size="large"
          clearable
          @keyup.enter="handleResolve"
        >
          <template #prepend>
            <el-icon><Link /></el-icon>
          </template>
          <template #append>
            <el-button
              type="primary"
              :loading="loading"
              @click="handleResolve"
            >
              解析
            </el-button>
          </template>
        </el-input>
      </div>

      <el-alert
        v-if="errorMessage"
        :title="errorMessage"
        type="error"
        :closable="true"
        show-icon
        style="margin-top: 16px"
        @close="errorMessage = ''"
      />

      <div v-if="result" class="dns-result">
        <div class="result-header">
          <h3>解析结果</h3>
          <div class="result-meta">
            <el-tag type="info" size="small">
              域名: {{ result.domain }}
            </el-tag>
            <el-tag type="success" size="small">
              耗时: {{ result.duration_ms }}ms
            </el-tag>
            <el-tag size="small">
              记录数: {{ result.records.length }}
            </el-tag>
          </div>
        </div>

        <el-table :data="result.records" stripe style="width: 100%">
          <el-table-column prop="type" label="类型" width="100">
            <template #default="{ row }">
              <el-tag :type="getRecordTypeColor(row.type)" size="small">
                {{ row.type }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="值" min-width="300">
            <template #default="{ row }">
              <code class="record-value">{{ row.value }}</code>
              <el-button
                type="primary"
                link
                size="small"
                @click="copyValue(row.value)"
              >
                复制
              </el-button>
            </template>
          </el-table-column>
          <el-table-column prop="ttl" label="优先级/TTL" width="120">
            <template #default="{ row }">
              {{ row.ttl || '-' }}
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-if="!result && !loading" class="dns-tips">
        <h4>支持的记录类型</h4>
        <div class="record-types">
          <el-tag type="primary">A (IPv4)</el-tag>
          <el-tag type="success">AAAA (IPv6)</el-tag>
          <el-tag type="warning">CNAME</el-tag>
          <el-tag type="danger">MX</el-tag>
          <el-tag type="info">TXT</el-tag>
          <el-tag>NS</el-tag>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface DNSRecord {
  type: string
  value: string
  ttl?: number
}

interface DNSResult {
  domain: string
  records: DNSRecord[]
  resolve_at: string
  duration_ms: number
}

const domain = ref('')
const loading = ref(false)
const errorMessage = ref('')
const result = ref<DNSResult | null>(null)

async function handleResolve() {
  if (!domain.value.trim()) {
    ElMessage.warning('请输入域名')
    return
  }

  loading.value = true
  errorMessage.value = ''
  result.value = null

  try {
    const response = await request.post<DNSResult>('/api/v1/dns/resolve', {
      domain: domain.value.trim()
    })
    result.value = response.data
    ElMessage.success('解析完成')
  } catch (error: any) {
    const data = error.response?.data
    errorMessage.value = data?.error || '解析失败，请检查域名是否有效'
  } finally {
    loading.value = false
  }
}

function getRecordTypeColor(type: string): string {
  const colors: Record<string, string> = {
    'A': 'primary',
    'AAAA': 'success',
    'CNAME': 'warning',
    'MX': 'danger',
    'TXT': 'info',
    'NS': ''
  }
  return colors[type] || ''
}

function copyValue(value: string) {
  navigator.clipboard.writeText(value)
    .then(() => {
      ElMessage.success('已复制到剪贴板')
    })
    .catch(() => {
      ElMessage.error('复制失败')
    })
}
</script>

<style scoped>
.dns-container {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  font-size: 14px;
  margin: 0;
}

.dns-card {
  background: var(--bg-secondary, #1a1f3a);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

.dns-input-section {
  margin-bottom: 24px;
}

.dns-result {
  margin-top: 24px;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.result-header h3 {
  color: var(--text-primary, #ffffff);
  font-size: 16px;
  font-weight: 500;
  margin: 0;
}

.result-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.record-value {
  background: rgba(0, 212, 255, 0.1);
  color: #00d4ff;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  word-break: break-all;
}

.dns-tips {
  margin-top: 24px;
  padding: 16px;
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border-radius: 8px;
}

.dns-tips h4 {
  color: var(--text-primary, #ffffff);
  font-size: 14px;
  font-weight: 500;
  margin: 0 0 12px 0;
}

.record-types {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

:deep(.el-card__body) {
  padding: 24px;
}

:deep(.el-input__wrapper) {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border, rgba(255, 255, 255, 0.1));
}

:deep(.el-input__inner) {
  color: var(--text-primary, #ffffff);
}

:deep(.el-input-group__prepend) {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.05));
  border-color: var(--border, rgba(255, 255, 255, 0.1));
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
}

:deep(.el-input-group__append) {
  background: transparent;
  border-color: var(--border, rgba(255, 255, 255, 0.1));
}

:deep(.el-table) {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: rgba(255, 255, 255, 0.05);
  --el-table-row-hover-bg-color: rgba(255, 255, 255, 0.05);
  --el-table-border-color: rgba(255, 255, 255, 0.1);
  --el-table-text-color: var(--text-primary, #ffffff);
  --el-table-header-text-color: var(--text-secondary, rgba(255, 255, 255, 0.6));
}

:deep(.el-table__row) {
  background-color: var(--secondary-bg, #161b22) !important;
}

:deep(.el-table__row--striped) {
  background-color: var(--bg-color, #0d1117) !important;
}

:deep(.el-table__row td) {
  border-bottom-color: rgba(255, 255, 255, 0.1) !important;
}

:deep(.el-table__header th) {
  background-color: var(--bg-color, #0d1117) !important;
  color: var(--label-text, #c9d1d9) !important;
  font-weight: 500;
}
</style>
