<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getVersion } from '@/api/version'

const version = ref('加载中...')
const currentYear = new Date().getFullYear()

onMounted(async () => {
  try {
    const versionInfo = await getVersion()
    version.value = versionInfo.version || '未知'
  } catch {
    version.value = '未知'
  }
})
</script>

<template>
  <el-footer class="footer">
    <div class="footer-content">
      <div class="copyright">
        版权所有 © {{ currentYear }} CYP. All rights reserved.
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
