<template>
  <div class="sbom-page">
    <div class="page-header">
      <h1>SBOM 管理</h1>
      <p class="subtitle">软件物料清单 (Software Bill of Materials) - 管理镜像依赖和漏洞扫描</p>
    </div>

    <div class="actions-bar">
      <el-button type="primary" @click="showGenerateDialog = true">
        <el-icon><Document /></el-icon>
        生成 SBOM
      </el-button>
      <el-button type="warning" @click="showScanDialog = true">
        <el-icon><Warning /></el-icon>
        漏洞扫描
      </el-button>
      <el-button @click="loadSBOMs">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-card class="sbom-card">
      <template #header>
        <div class="card-header">
          <span>SBOM 列表</span>
          <el-tag type="info">共 {{ total }} 个</el-tag>
        </div>
      </template>

      <el-table :data="sboms" v-loading="loading" stripe>
        <el-table-column prop="image_ref" label="镜像" min-width="200">
          <template #default="{ row }">
            <div class="image-ref">
              <el-icon><Box /></el-icon>
              <span>{{ row.image_ref }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="format" label="格式" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.format }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="generator" label="生成器" width="100" />
        <el-table-column prop="generated_at" label="生成时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.generated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="包数量" width="100">
          <template #default="{ row }">
            {{ row.packages?.length || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="漏洞" width="150">
          <template #default="{ row }">
            <div class="vuln-summary" v-if="row.vulnerabilities?.length">
              <el-tag type="danger" size="small">{{ getVulnCount(row, 'CRITICAL') }} 严重</el-tag>
              <el-tag type="warning" size="small">{{ getVulnCount(row, 'HIGH') }} 高危</el-tag>
            </div>
            <el-tag v-else type="success" size="small">无漏洞</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewSBOM(row)">查看</el-button>
            <el-button size="small" @click="exportSBOM(row.image_ref)">导出</el-button>
            <el-button size="small" type="danger" @click="deleteSBOM(row.image_ref)">删除</el-button>
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
          @size-change="loadSBOMs"
          @current-change="loadSBOMs"
        />
      </div>
    </el-card>

    <!-- Generate Dialog -->
    <el-dialog v-model="showGenerateDialog" title="生成 SBOM" width="500px">
      <el-form :model="generateForm" label-width="100px">
        <el-form-item label="镜像引用" required>
          <el-input v-model="generateForm.image_ref" placeholder="例如: myregistry/myimage:v1.0" />
        </el-form-item>
        <el-form-item label="输出格式">
          <el-select v-model="generateForm.format" style="width: 100%">
            <el-option label="SPDX JSON" value="spdx-json" />
            <el-option label="CycloneDX JSON" value="cyclonedx-json" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGenerateDialog = false">取消</el-button>
        <el-button type="primary" @click="generateSBOM" :loading="generating">生成</el-button>
      </template>
    </el-dialog>

    <!-- Scan Dialog -->
    <el-dialog v-model="showScanDialog" title="漏洞扫描" width="600px">
      <el-form :model="scanForm" label-width="100px">
        <el-form-item label="镜像引用" required>
          <el-input v-model="scanForm.image_ref" placeholder="例如: myregistry/myimage:v1.0" />
        </el-form-item>
      </el-form>
      <div v-if="scanResult" class="scan-result">
        <el-divider>扫描结果</el-divider>
        <div class="summary-cards">
          <el-card shadow="never" class="summary-card critical">
            <div class="count">{{ scanResult.summary.critical }}</div>
            <div class="label">严重</div>
          </el-card>
          <el-card shadow="never" class="summary-card high">
            <div class="count">{{ scanResult.summary.high }}</div>
            <div class="label">高危</div>
          </el-card>
          <el-card shadow="never" class="summary-card medium">
            <div class="count">{{ scanResult.summary.medium }}</div>
            <div class="label">中危</div>
          </el-card>
          <el-card shadow="never" class="summary-card low">
            <div class="count">{{ scanResult.summary.low }}</div>
            <div class="label">低危</div>
          </el-card>
        </div>
        <el-table :data="scanResult.vulnerabilities" max-height="300" v-if="scanResult.vulnerabilities?.length">
          <el-table-column prop="id" label="CVE ID" width="150" />
          <el-table-column prop="package" label="包名" width="150" />
          <el-table-column prop="severity" label="严重性" width="100">
            <template #default="{ row }">
              <el-tag :type="getSeverityType(row.severity)" size="small">{{ row.severity }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="描述" />
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showScanDialog = false; scanResult = null">关闭</el-button>
        <el-button type="primary" @click="scanVulnerabilities" :loading="scanning">扫描</el-button>
      </template>
    </el-dialog>

    <!-- View Dialog -->
    <el-dialog v-model="showViewDialog" title="SBOM 详情" width="800px">
      <div v-if="currentSBOM">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="镜像">{{ currentSBOM.image_ref }}</el-descriptions-item>
          <el-descriptions-item label="格式">{{ currentSBOM.format }}</el-descriptions-item>
          <el-descriptions-item label="生成器">{{ currentSBOM.generator }}</el-descriptions-item>
          <el-descriptions-item label="生成时间">{{ formatDate(currentSBOM.generated_at) }}</el-descriptions-item>
        </el-descriptions>
        <el-divider>依赖包列表 ({{ currentSBOM.packages?.length || 0 }})</el-divider>
        <el-table :data="currentSBOM.packages" max-height="400">
          <el-table-column prop="name" label="包名" />
          <el-table-column prop="version" label="版本" width="120" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="license" label="许可证" width="150" />
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showViewDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Warning, Refresh, Box } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface SBOMPackage {
  name: string
  version: string
  type: string
  license: string
}

interface Vulnerability {
  id: string
  package: string
  version: string
  severity: string
  title: string
}

interface SBOM {
  id: number
  image_ref: string
  format: string
  generator: string
  generated_at: string
  packages: SBOMPackage[]
  vulnerabilities: Vulnerability[]
}

interface ScanResult {
  image_ref: string
  scanned_at: string
  scanner: string
  vulnerabilities: Vulnerability[]
  summary: { critical: number; high: number; medium: number; low: number; total: number }
}

const loading = ref(false)
const generating = ref(false)
const scanning = ref(false)
const sboms = ref<SBOM[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const showGenerateDialog = ref(false)
const showScanDialog = ref(false)
const showViewDialog = ref(false)
const generateForm = ref({ image_ref: '', format: 'spdx-json' })
const scanForm = ref({ image_ref: '' })
const scanResult = ref<ScanResult | null>(null)
const currentSBOM = ref<SBOM | null>(null)

const formatDate = (date: string) => new Date(date).toLocaleString('zh-CN')

const getSeverityType = (severity: string) => {
  const map: Record<string, string> = { CRITICAL: 'danger', HIGH: 'warning', MEDIUM: 'info', LOW: 'success' }
  return map[severity] || 'info'
}

const getVulnCount = (sbom: SBOM, severity: string) => {
  return sbom.vulnerabilities?.filter(v => v.severity === severity).length || 0
}

const loadSBOMs = async () => {
  loading.value = true
  try {
    const res = await request.get('/api/v1/sbom', { params: { page: page.value, page_size: pageSize.value } })
    sboms.value = res.data.sboms || []
    total.value = res.data.total || 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载 SBOM 列表失败')
  } finally {
    loading.value = false
  }
}

const generateSBOM = async () => {
  if (!generateForm.value.image_ref) {
    ElMessage.warning('请输入镜像引用')
    return
  }
  generating.value = true
  try {
    await request.post('/api/v1/sbom/generate', generateForm.value)
    ElMessage.success('SBOM 生成成功')
    showGenerateDialog.value = false
    generateForm.value = { image_ref: '', format: 'spdx-json' }
    loadSBOMs()
  } catch (error: any) {
    ElMessage.error(error.message || '生成失败')
  } finally {
    generating.value = false
  }
}

const scanVulnerabilities = async () => {
  if (!scanForm.value.image_ref) {
    ElMessage.warning('请输入镜像引用')
    return
  }
  scanning.value = true
  try {
    const res = await request.post('/api/v1/sbom/scan', scanForm.value)
    scanResult.value = res.data
  } catch (error: any) {
    ElMessage.error(error.message || '扫描失败')
  } finally {
    scanning.value = false
  }
}

const viewSBOM = (sbom: SBOM) => {
  currentSBOM.value = sbom
  showViewDialog.value = true
}

const exportSBOM = async (imageRef: string) => {
  try {
    const res = await request.get(`/api/v1/sbom/${encodeURIComponent(imageRef)}/export`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.download = `sbom-${imageRef.replace(/[/:]/g, '_')}.json`
    link.click()
    window.URL.revokeObjectURL(url)
  } catch (error: any) {
    ElMessage.error(error.message || '导出失败')
  }
}

const deleteSBOM = async (imageRef: string) => {
  try {
    await ElMessageBox.confirm('确定要删除此 SBOM 吗？', '确认删除', { type: 'warning' })
    await request.delete(`/api/v1/sbom/${encodeURIComponent(imageRef)}`)
    ElMessage.success('SBOM 已删除')
    loadSBOMs()
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error.message || '删除失败')
  }
}

onMounted(() => { loadSBOMs() })
</script>

<style scoped>
.sbom-page { padding: 20px; }
.page-header { margin-bottom: 20px; }
.page-header h1 { margin: 0 0 8px 0; font-size: 24px; }
.subtitle { color: #666; margin: 0; }
.actions-bar { margin-bottom: 20px; display: flex; gap: 10px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.image-ref { display: flex; align-items: center; gap: 8px; }
.pagination { margin-top: 20px; display: flex; justify-content: flex-end; }
.vuln-summary { display: flex; gap: 5px; flex-wrap: wrap; }
.scan-result { margin-top: 20px; }
.summary-cards { display: flex; gap: 15px; margin-bottom: 20px; }
.summary-card { flex: 1; text-align: center; }
.summary-card .count { font-size: 28px; font-weight: bold; }
.summary-card .label { font-size: 12px; color: #666; }
.summary-card.critical .count { color: #f56c6c; }
.summary-card.high .count { color: #e6a23c; }
.summary-card.medium .count { color: #409eff; }
.summary-card.low .count { color: #67c23a; }
</style>
