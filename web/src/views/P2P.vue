<template>
  <div class="p2p-container">
    <el-card class="status-card">
      <template #header>
        <div class="card-header">
          <span>P2P 分发状态</span>
          <el-switch
            v-model="status.enabled"
            :loading="loading"
            @change="toggleP2P"
            active-text="启用"
            inactive-text="禁用"
          />
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="运行状态">
          <el-tag :type="status.running ? 'success' : 'info'">
            {{ status.running ? '运行中' : '已停止' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="节点ID">
          <el-tooltip :content="status.peer_id" placement="top">
            <span class="peer-id">{{ truncateId(status.peer_id) }}</span>
          </el-tooltip>
          <el-button
            v-if="status.peer_id"
            type="primary"
            link
            size="small"
            @click="copyToClipboard(status.peer_id)"
          >
            复制
          </el-button>
        </el-descriptions-item>
        <el-descriptions-item label="连接节点数">
          {{ status.connected_peers }} / {{ status.peer_count }}
        </el-descriptions-item>
        <el-descriptions-item label="分享模式">
          <el-tag>{{ shareModeText(status.share_mode) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="已发送">
          {{ formatBytes(status.bytes_sent) }}
        </el-descriptions-item>
        <el-descriptions-item label="已接收">
          {{ formatBytes(status.bytes_received) }}
        </el-descriptions-item>
        <el-descriptions-item label="分享Blob数">
          {{ status.blobs_shared }}
        </el-descriptions-item>
        <el-descriptions-item label="接收Blob数">
          {{ status.blobs_received }}
        </el-descriptions-item>
        <el-descriptions-item label="运行时间">
          {{ status.uptime || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="NAT状态">
          <el-tag :type="natStatusType(status.nat_status?.type)">
            {{ natStatusText(status.nat_status?.type) }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="status.addresses?.length" class="addresses-section">
        <h4>监听地址</h4>
        <el-tag
          v-for="addr in status.addresses"
          :key="addr"
          class="address-tag"
          type="info"
        >
          {{ addr }}
        </el-tag>
      </div>
    </el-card>

    <el-card class="peers-card">
      <template #header>
        <div class="card-header">
          <span>对等节点 ({{ peers.length }})</span>
          <el-button type="primary" size="small" @click="showConnectDialog = true">
            连接节点
          </el-button>
        </div>
      </template>

      <el-table :data="peers" stripe style="width: 100%">
        <el-table-column label="节点ID" min-width="200">
          <template #default="{ row }">
            <el-tooltip :content="row.id" placement="top">
              <span class="peer-id">{{ truncateId(row.id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="地址" min-width="250">
          <template #default="{ row }">
            <span v-if="row.addresses?.length">{{ row.addresses[0] }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="已发送" width="120">
          <template #default="{ row }">
            {{ formatBytes(row.bytes_sent) }}
          </template>
        </el-table-column>
        <el-table-column label="已接收" width="120">
          <template #default="{ row }">
            {{ formatBytes(row.bytes_received) }}
          </template>
        </el-table-column>
        <el-table-column label="延迟" width="100">
          <template #default="{ row }">
            {{ row.latency || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="最后活跃" width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_seen) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              type="danger"
              link
              size="small"
              @click="disconnectPeer(row.id)"
            >
              断开
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!peers.length" description="暂无连接的节点" />
    </el-card>

    <el-card class="blobs-card">
      <template #header>
        <div class="card-header">
          <span>本地 Blob 缓存</span>
          <el-button size="small" @click="refreshBlobs">刷新</el-button>
        </div>
      </template>

      <el-table :data="blobs" stripe style="width: 100%" max-height="300">
        <el-table-column label="摘要" min-width="400">
          <template #default="{ row }">
            <el-tooltip :content="row" placement="top">
              <span class="blob-digest">{{ row }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="announceBlob(row)">
              宣布
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!blobs.length" description="暂无本地缓存" />
    </el-card>

    <!-- 连接节点对话框 -->
    <el-dialog v-model="showConnectDialog" title="连接节点" width="500px">
      <el-form :model="connectForm" label-width="100px">
        <el-form-item label="节点地址">
          <el-input
            v-model="connectForm.address"
            placeholder="/ip4/x.x.x.x/tcp/4001/p2p/QmXXX..."
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showConnectDialog = false">取消</el-button>
        <el-button type="primary" @click="connectPeer" :loading="connecting">
          连接
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/api'

interface P2PStatus {
  enabled: boolean
  running: boolean
  peer_id: string
  addresses: string[]
  peer_count: number
  connected_peers: number
  bytes_sent: number
  bytes_received: number
  blobs_shared: number
  blobs_received: number
  uptime: string
  share_mode: string
  nat_status?: {
    type: string
    public_ip: string
    reachable: boolean
  }
}

interface PeerInfo {
  id: string
  addresses: string[]
  connected_at: string
  last_seen: string
  bytes_sent: number
  bytes_received: number
  latency: string
}

const status = ref<P2PStatus>({
  enabled: false,
  running: false,
  peer_id: '',
  addresses: [],
  peer_count: 0,
  connected_peers: 0,
  bytes_sent: 0,
  bytes_received: 0,
  blobs_shared: 0,
  blobs_received: 0,
  uptime: '',
  share_mode: 'selective'
})

const peers = ref<PeerInfo[]>([])
const blobs = ref<string[]>([])
const loading = ref(false)
const connecting = ref(false)
const showConnectDialog = ref(false)
const connectForm = ref({ address: '' })

let refreshTimer: number | null = null

const fetchStatus = async () => {
  try {
    const res = await request.get('/api/v1/p2p/status')
    if (res.data.code === 0) {
      status.value = res.data.data
    }
  } catch (error) {
    console.error('获取P2P状态失败', error)
  }
}

const fetchPeers = async () => {
  try {
    const res = await request.get('/api/v1/p2p/peers')
    if (res.data.code === 0) {
      peers.value = res.data.data || []
    }
  } catch (error) {
    console.error('获取节点列表失败', error)
  }
}

const refreshBlobs = async () => {
  try {
    const res = await request.get('/api/v1/p2p/blobs')
    if (res.data.code === 0) {
      blobs.value = res.data.data || []
    }
  } catch (error) {
    console.error('获取Blob列表失败', error)
  }
}

const toggleP2P = async (enabled: boolean) => {
  loading.value = true
  try {
    const endpoint = enabled ? '/api/v1/p2p/enable' : '/api/v1/p2p/disable'
    const res = await request.post(endpoint)
    if (res.data.code === 0) {
      ElMessage.success(enabled ? 'P2P已启用' : 'P2P已禁用')
      await fetchStatus()
    } else {
      ElMessage.error(res.data.message)
      status.value.enabled = !enabled
    }
  } catch (error) {
    ElMessage.error('操作失败')
    status.value.enabled = !enabled
  } finally {
    loading.value = false
  }
}

const connectPeer = async () => {
  if (!connectForm.value.address) {
    ElMessage.warning('请输入节点地址')
    return
  }

  connecting.value = true
  try {
    const res = await request.post('/api/v1/p2p/peers/connect', {
      address: connectForm.value.address
    })
    if (res.data.code === 0) {
      ElMessage.success('连接成功')
      showConnectDialog.value = false
      connectForm.value.address = ''
      await fetchPeers()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('连接失败')
  } finally {
    connecting.value = false
  }
}

const disconnectPeer = async (peerId: string) => {
  try {
    const res = await request.delete(`/api/v1/p2p/peers/${peerId}`)
    if (res.data.code === 0) {
      ElMessage.success('已断开连接')
      await fetchPeers()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('断开失败')
  }
}

const announceBlob = async (digest: string) => {
  try {
    const res = await request.post(`/api/v1/p2p/blobs/${digest}/announce`)
    if (res.data.code === 0) {
      ElMessage.success('已宣布到P2P网络')
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (error) {
    ElMessage.error('宣布失败')
  }
}

const truncateId = (id?: string) => {
  if (!id) return '-'
  if (id.length <= 16) return id
  return `${id.slice(0, 8)}...${id.slice(-8)}`
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024
    i++
  }
  return `${bytes.toFixed(2)} ${units[i]}`
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

const shareModeText = (mode: string) => {
  const modes: Record<string, string> = {
    all: '全部分享',
    selective: '选择性分享',
    none: '不分享'
  }
  return modes[mode] || mode
}

const natStatusType = (type: string) => {
  const types: Record<string, string> = {
    none: 'success',
    full_cone: 'success',
    restricted_cone: 'warning',
    port_restricted: 'warning',
    symmetric: 'danger',
    unknown: 'info'
  }
  return types[type] || 'info'
}

const natStatusText = (type: string) => {
  const texts: Record<string, string> = {
    none: '公网IP',
    full_cone: '完全锥形NAT',
    restricted_cone: '受限锥形NAT',
    port_restricted: '端口受限NAT',
    symmetric: '对称NAT',
    unknown: '未知'
  }
  return texts[type] || type || '未知'
}

const copyToClipboard = (text?: string) => {
  if (!text) return
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
}

onMounted(() => {
  fetchStatus()
  fetchPeers()
  refreshBlobs()

  // 定时刷新
  refreshTimer = window.setInterval(() => {
    fetchStatus()
    fetchPeers()
  }, 10000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.p2p-container {
  padding: 20px;
}

.status-card,
.peers-card,
.blobs-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.peer-id {
  font-family: monospace;
  font-size: 13px;
}

.blob-digest {
  font-family: monospace;
  font-size: 12px;
}

.addresses-section {
  margin-top: 20px;
}

.addresses-section h4 {
  margin-bottom: 10px;
  color: #606266;
}

.address-tag {
  margin: 4px;
  font-family: monospace;
  font-size: 12px;
}
</style>
