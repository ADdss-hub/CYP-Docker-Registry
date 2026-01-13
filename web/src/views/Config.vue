<template>
  <div class="config-container">
    <div class="page-header">
      <h1>系统配置</h1>
      <p>管理系统配置和安全设置</p>
    </div>

    <el-tabs v-model="activeTab" class="config-tabs">
      <!-- General Settings -->
      <el-tab-pane label="基本设置" name="general">
        <el-card>
          <el-form :model="generalConfig" label-width="150px">
            <el-form-item label="系统名称">
              <el-input v-model="generalConfig.app_name" />
            </el-form-item>
            <el-form-item label="监听端口">
              <el-input-number v-model="generalConfig.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="日志级别">
              <el-select v-model="generalConfig.log_level">
                <el-option label="Debug" value="debug" />
                <el-option label="Info" value="info" />
                <el-option label="Warn" value="warn" />
                <el-option label="Error" value="error" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveGeneralConfig">保存</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- Security Settings -->
      <el-tab-pane label="安全设置" name="security">
        <el-card>
          <el-form :model="securityConfig" label-width="180px">
            <h3>认证设置</h3>
            <el-form-item label="强制登录">
              <el-switch v-model="securityConfig.force_login" disabled />
              <span class="form-tip">系统默认开启，不可关闭</span>
            </el-form-item>
            <el-form-item label="会话超时">
              <el-select v-model="securityConfig.session_timeout">
                <el-option label="1 小时" value="1h" />
                <el-option label="4 小时" value="4h" />
                <el-option label="12 小时" value="12h" />
                <el-option label="24 小时" value="24h" />
              </el-select>
            </el-form-item>
            <el-form-item label="IP 绑定">
              <el-switch v-model="securityConfig.enforce_ip_binding" />
              <span class="form-tip">开启后，IP 变化需重新登录</span>
            </el-form-item>

            <el-divider />

            <h3>锁定策略</h3>
            <el-form-item label="最大登录失败次数">
              <el-input-number v-model="securityConfig.max_login_attempts" :min="1" :max="10" />
            </el-form-item>
            <el-form-item label="锁定时长">
              <el-select v-model="securityConfig.lock_duration">
                <el-option label="30 分钟" value="30m" />
                <el-option label="1 小时" value="1h" />
                <el-option label="2 小时" value="2h" />
                <el-option label="24 小时" value="24h" />
              </el-select>
            </el-form-item>
            <el-form-item label="渐进延迟">
              <el-switch v-model="securityConfig.progressive_delay" />
              <span class="form-tip">失败后逐渐增加等待时间</span>
            </el-form-item>

            <el-divider />

            <h3>入侵检测</h3>
            <el-form-item label="启用入侵检测">
              <el-switch v-model="securityConfig.intrusion_detection" />
            </el-form-item>
            <el-form-item label="绕过尝试锁定">
              <el-switch v-model="securityConfig.lock_on_bypass" />
              <span class="form-tip">检测到绕过尝试立即锁定</span>
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="saveSecurityConfig">保存</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- Storage Settings -->
      <el-tab-pane label="存储设置" name="storage">
        <el-card>
          <el-form :model="storageConfig" label-width="150px">
            <el-form-item label="存储路径">
              <el-input v-model="storageConfig.storage_path" />
            </el-form-item>
            <el-form-item label="缓存大小">
              <el-input v-model="storageConfig.cache_size" placeholder="如 10GB" />
            </el-form-item>
            <el-form-item label="自动清理">
              <el-switch v-model="storageConfig.auto_cleanup" />
            </el-form-item>
            <el-form-item label="保留天数" v-if="storageConfig.auto_cleanup">
              <el-input-number v-model="storageConfig.retention_days" :min="1" :max="365" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveStorageConfig">保存</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- Notification Settings -->
      <el-tab-pane label="通知设置" name="notification">
        <el-card>
          <el-form :model="notifyConfig" label-width="150px">
            <h3>Webhook 通知</h3>
            <el-form-item label="启用 Webhook">
              <el-switch v-model="notifyConfig.webhook_enabled" />
            </el-form-item>
            <el-form-item label="Webhook URL" v-if="notifyConfig.webhook_enabled">
              <el-input v-model="notifyConfig.webhook_url" placeholder="https://..." />
            </el-form-item>

            <el-divider />

            <h3>邮件通知</h3>
            <el-form-item label="启用邮件">
              <el-switch v-model="notifyConfig.email_enabled" />
            </el-form-item>
            <el-form-item label="SMTP 服务器" v-if="notifyConfig.email_enabled">
              <el-input v-model="notifyConfig.smtp_host" />
            </el-form-item>
            <el-form-item label="SMTP 端口" v-if="notifyConfig.email_enabled">
              <el-input-number v-model="notifyConfig.smtp_port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="发件人邮箱" v-if="notifyConfig.email_enabled">
              <el-input v-model="notifyConfig.smtp_user" />
            </el-form-item>
            <el-form-item label="收件人邮箱" v-if="notifyConfig.email_enabled">
              <el-input v-model="notifyConfig.notify_email" placeholder="多个邮箱用逗号分隔" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="saveNotifyConfig">保存</el-button>
              <el-button @click="testNotification">测试通知</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const activeTab = ref('general')

const generalConfig = reactive({
  app_name: 'CYP-Registry',
  port: 8080,
  log_level: 'info'
})

const securityConfig = reactive({
  force_login: true,
  session_timeout: '24h',
  enforce_ip_binding: true,
  max_login_attempts: 3,
  lock_duration: '1h',
  progressive_delay: true,
  intrusion_detection: true,
  lock_on_bypass: true
})

const storageConfig = reactive({
  storage_path: '/app/data',
  cache_size: '10GB',
  auto_cleanup: true,
  retention_days: 30
})

const notifyConfig = reactive({
  webhook_enabled: false,
  webhook_url: '',
  email_enabled: false,
  smtp_host: '',
  smtp_port: 587,
  smtp_user: '',
  notify_email: ''
})

onMounted(() => {
  fetchConfig()
})

async function fetchConfig() {
  try {
    const response = await request.get('/api/v1/config')
    // Merge with defaults
    Object.assign(generalConfig, response.data.general || {})
    Object.assign(securityConfig, response.data.security || {})
    Object.assign(storageConfig, response.data.storage || {})
    Object.assign(notifyConfig, response.data.notify || {})
  } catch (error) {
    console.error('Failed to fetch config:', error)
  }
}

async function saveGeneralConfig() {
  try {
    await request.put('/api/v1/config/general', generalConfig)
    ElMessage.success('基本设置已保存')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  }
}

async function saveSecurityConfig() {
  try {
    await request.put('/api/v1/config/security', securityConfig)
    ElMessage.success('安全设置已保存')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  }
}

async function saveStorageConfig() {
  try {
    await request.put('/api/v1/config/storage', storageConfig)
    ElMessage.success('存储设置已保存')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  }
}

async function saveNotifyConfig() {
  try {
    await request.put('/api/v1/config/notify', notifyConfig)
    ElMessage.success('通知设置已保存')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  }
}

async function testNotification() {
  try {
    await request.post('/api/v1/config/notify/test')
    ElMessage.success('测试通知已发送')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '发送失败')
  }
}
</script>

<style scoped>
.config-container {
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

.config-tabs :deep(.el-tabs__content) {
  padding: 20px 0;
}

.config-tabs :deep(.el-card) {
  background: var(--bg-secondary, #1a1f3a);
}

h3 {
  color: var(--text-primary, #ffffff);
  font-size: 16px;
  margin: 0 0 20px 0;
}

.form-tip {
  margin-left: 12px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  font-size: 12px;
}
</style>
