<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

const version = ref('加载中...')
const currentYear = new Date().getFullYear()

onMounted(async () => {
  try {
    // 尝试多个版本接口
    const endpoints = ['/api/v1/version', '/version', '/api/v1/system/info']
    
    for (const endpoint of endpoints) {
      try {
        const response = await request.get(endpoint)
        const data = response.data
        if (data?.version && data.version.trim() !== '') {
          version.value = data.version
          return
        }
      } catch {
        // 继续尝试下一个接口
      }
    }
    
    // 如果所有接口都失败，使用默认版本
    version.value = '1.0.7'
  } catch {
    version.value = '1.0.7'
  }
})
</script>

<template>
  <el-footer class="footer">
    <div class="footer-content">
      <div class="copyright">
        版权所有 © {{ currentYear }} CYP
      </div>
      <div class="divider">|</div>
      <div class="version">
        版本 v{{ version }}
      </div>
    </div>
    <div class="contact">
      联系方式：nasDSSCYP@outlook.com
    </div>
  </el-footer>
</template>

<style scoped>
.footer {
  height: auto;
  min-height: 60px;
  background-color: #161b22;
  border-top: 1px solid #30363d;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 16px 24px;
  gap: 8px;
}

.footer-content {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #8b949e;
  font-size: 14px;
}

.divider {
  color: #30363d;
}

.copyright {
  color: #8b949e;
}

.version {
  color: var(--highlight-color);
  font-family: 'JetBrains Mono', 'Consolas', monospace;
}

.contact {
  color: #6e7681;
  font-size: 12px;
}
</style>
