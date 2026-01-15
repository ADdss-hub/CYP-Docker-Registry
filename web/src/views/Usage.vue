<template>
  <div class="usage-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h1>使用方法</h1>
      <p class="subtitle">容器镜像个人仓库使用指南</p>
    </div>

    <!-- 快速开始 -->
    <div class="section">
      <h3>快速开始</h3>
      <div class="step-list">
        <div class="step-item">
          <div class="step-number">1</div>
          <div class="step-content">
            <h4>登录仓库</h4>
            <p>使用 Docker CLI 登录到您的私有仓库：</p>
            <div class="code-block">
              <code>docker login {{ registryHost }}</code>
              <el-button size="small" text @click="copyCode(`docker login ${registryHost}`)">复制</el-button>
            </div>
            <p class="tip">输入您的用户名和密码（或访问令牌）进行认证</p>
          </div>
        </div>
        <div class="step-item">
          <div class="step-number">2</div>
          <div class="step-content">
            <h4>推送镜像</h4>
            <p>标记并推送本地镜像到仓库：</p>
            <div class="code-block">
              <code>docker tag myimage:latest {{ registryHost }}/myimage:latest</code>
              <el-button size="small" text @click="copyCode(`docker tag myimage:latest ${registryHost}/myimage:latest`)">复制</el-button>
            </div>
            <div class="code-block">
              <code>docker push {{ registryHost }}/myimage:latest</code>
              <el-button size="small" text @click="copyCode(`docker push ${registryHost}/myimage:latest`)">复制</el-button>
            </div>
          </div>
        </div>
        <div class="step-item">
          <div class="step-number">3</div>
          <div class="step-content">
            <h4>拉取镜像</h4>
            <p>从仓库拉取镜像：</p>
            <div class="code-block">
              <code>docker pull {{ registryHost }}/myimage:latest</code>
              <el-button size="small" text @click="copyCode(`docker pull ${registryHost}/myimage:latest`)">复制</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 访问令牌 -->
    <div class="section">
      <h3>访问令牌</h3>
      <div class="info-content">
        <p>访问令牌（Personal Access Token）可用于 API 认证和 Docker CLI 登录，比使用密码更安全。</p>
        <h4>创建令牌</h4>
        <ol>
          <li>进入 <router-link to="/tokens">访问令牌</router-link> 页面</li>
          <li>点击"创建令牌"按钮</li>
          <li>输入令牌名称并选择权限范围</li>
          <li>复制并安全保存生成的令牌</li>
        </ol>
        <h4>使用令牌登录</h4>
        <div class="code-block">
          <code>docker login -u &lt;用户名&gt; -p &lt;令牌&gt; {{ registryHost }}</code>
        </div>
        <h4>API 请求认证</h4>
        <div class="code-block">
          <code>curl -H "Authorization: Token &lt;令牌&gt;" {{ registryHost }}/api/v1/images</code>
        </div>
      </div>
    </div>

    <!-- 镜像加速 -->
    <div class="section">
      <h3>镜像加速</h3>
      <div class="info-content">
        <p>镜像加速功能可以缓存常用的公共镜像，加快拉取速度。</p>
        <h4>配置 Docker 使用加速</h4>
        <p>编辑 Docker 配置文件 <code>/etc/docker/daemon.json</code>：</p>
        <div class="code-block">
          <code>{{ daemonConfig }}</code>
          <el-button size="small" text @click="copyCode(daemonConfig)">复制</el-button>
        </div>
        <p>重启 Docker 服务：</p>
        <div class="code-block">
          <code>sudo systemctl restart docker</code>
          <el-button size="small" text @click="copyCode('sudo systemctl restart docker')">复制</el-button>
        </div>
      </div>
    </div>

    <!-- 组织管理 -->
    <div class="section">
      <h3>组织管理</h3>
      <div class="info-content">
        <p>组织功能允许您创建团队并管理成员权限。</p>
        <h4>创建组织</h4>
        <ol>
          <li>进入 <router-link to="/orgs">组织管理</router-link> 页面</li>
          <li>点击"创建组织"按钮</li>
          <li>输入组织名称（唯一标识）和显示名称</li>
        </ol>
        <h4>推送镜像到组织</h4>
        <div class="code-block">
          <code>docker push {{ registryHost }}/&lt;组织名&gt;/myimage:latest</code>
        </div>
      </div>
    </div>

    <!-- 镜像分享 -->
    <div class="section">
      <h3>镜像分享</h3>
      <div class="info-content">
        <p>分享功能允许您创建临时链接，让他人无需登录即可拉取指定镜像。</p>
        <h4>创建分享链接</h4>
        <ol>
          <li>进入 <router-link to="/share">分享管理</router-link> 页面</li>
          <li>点击"创建分享"按钮</li>
          <li>选择要分享的镜像</li>
          <li>设置密码保护（可选）和有效期</li>
          <li>复制生成的分享链接</li>
        </ol>
      </div>
    </div>

    <!-- P2P 分发 -->
    <div class="section">
      <h3>P2P 分发</h3>
      <div class="info-content">
        <p>P2P 分发功能可以在多个节点之间共享镜像层，减少带宽消耗。</p>
        <h4>启用 P2P</h4>
        <ol>
          <li>进入 <router-link to="/p2p">P2P 分发</router-link> 页面</li>
          <li>开启 P2P 开关</li>
          <li>等待节点发现和连接</li>
        </ol>
        <p class="tip">P2P 功能需要节点之间网络互通，NAT 环境下可能需要配置端口映射。</p>
      </div>
    </div>

    <!-- DNS 解析 -->
    <div class="section">
      <h3>DNS 解析</h3>
      <div class="info-content">
        <p>内置 DNS 解析工具，可以查询域名的各类 DNS 记录。</p>
        <h4>支持的记录类型</h4>
        <div class="tag-list">
          <el-tag type="primary">A (IPv4)</el-tag>
          <el-tag type="success">AAAA (IPv6)</el-tag>
          <el-tag type="warning">CNAME</el-tag>
          <el-tag type="danger">MX</el-tag>
          <el-tag type="info">TXT</el-tag>
          <el-tag>NS</el-tag>
        </div>
        <p>进入 <router-link to="/dns">DNS 解析</router-link> 页面使用此功能。</p>
      </div>
    </div>

    <!-- 常见问题 -->
    <div class="section">
      <h3>常见问题</h3>
      <el-collapse>
        <el-collapse-item title="无法登录仓库？" name="1">
          <p>请检查以下几点：</p>
          <ul>
            <li>确认用户名和密码正确</li>
            <li>如果使用访问令牌，确认令牌未过期且有正确的权限</li>
            <li>检查网络连接是否正常</li>
            <li>确认仓库地址正确</li>
          </ul>
        </el-collapse-item>
        <el-collapse-item title="推送镜像失败？" name="2">
          <p>可能的原因：</p>
          <ul>
            <li>未登录或登录已过期</li>
            <li>没有推送权限</li>
            <li>镜像标签格式不正确</li>
            <li>存储空间不足</li>
          </ul>
        </el-collapse-item>
        <el-collapse-item title="如何删除镜像？" name="3">
          <p>进入镜像管理页面，找到要删除的镜像，点击删除按钮。注意：删除操作不可恢复。</p>
        </el-collapse-item>
        <el-collapse-item title="系统被锁定怎么办？" name="4">
          <p>系统锁定后不允许手动解锁。请联系管理员或重新安装系统。</p>
        </el-collapse-item>
      </el-collapse>
    </div>

    <!-- 联系支持 -->
    <div class="section contact">
      <h3>联系支持</h3>
      <div class="contact-info">
        <p>如有问题或建议，请联系：</p>
        <p><strong>邮箱：</strong><a href="mailto:nasDSSCYP@outlook.com">nasDSSCYP@outlook.com</a></p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'

const registryHost = computed(() => window.location.host)

const daemonConfig = computed(() => {
  return JSON.stringify({
    "registry-mirrors": [`http://${registryHost.value}`]
  }, null, 2)
})

function copyCode(code: string) {
  navigator.clipboard.writeText(code)
    .then(() => {
      ElMessage.success('已复制到剪贴板')
    })
    .catch(() => {
      ElMessage.error('复制失败')
    })
}
</script>

<style scoped>
.usage-page {
  animation: fadeIn 0.3s ease-out;
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}

.page-header {
  margin-bottom: 32px;
}

.page-header h1 {
  color: var(--text-color, #e6edf3);
  font-size: 28px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--muted-text, #8b949e);
  font-size: 16px;
  margin: 0;
}

.section {
  background-color: var(--secondary-bg, #161b22);
  border: 1px solid var(--border-color, #30363d);
  border-radius: var(--radius-md, 8px);
  padding: 24px;
  margin-bottom: 24px;
}

.section h3 {
  margin: 0 0 20px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color, #e6edf3);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color, #30363d);
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.step-item {
  display: flex;
  gap: 16px;
}

.step-number {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #1890ff, #096dd9);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  color: white;
  flex-shrink: 0;
}

.step-content {
  flex: 1;
}

.step-content h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color, #e6edf3);
}

.step-content p {
  margin: 0 0 12px 0;
  color: var(--muted-text, #8b949e);
  font-size: 14px;
  line-height: 1.6;
}

.code-block {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: var(--bg-color, #0d1117);
  border: 1px solid var(--border-color, #30363d);
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 12px;
}

.code-block code {
  font-family: var(--font-mono, 'Consolas', monospace);
  font-size: 13px;
  color: var(--highlight-color, #58a6ff);
  word-break: break-all;
}

.tip {
  font-size: 13px;
  color: var(--muted-text, #8b949e);
  font-style: italic;
}

.info-content {
  color: var(--muted-text, #8b949e);
  font-size: 14px;
  line-height: 1.8;
}

.info-content h4 {
  margin: 20px 0 12px 0;
  font-size: 15px;
  font-weight: 500;
  color: var(--text-color, #e6edf3);
}

.info-content h4:first-child {
  margin-top: 0;
}

.info-content p {
  margin: 0 0 12px 0;
}

.info-content ol,
.info-content ul {
  margin: 0 0 16px 0;
  padding-left: 24px;
}

.info-content li {
  margin-bottom: 8px;
}

.info-content a {
  color: var(--highlight-color, #58a6ff);
  text-decoration: none;
}

.info-content a:hover {
  text-decoration: underline;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 12px 0;
}

.contact-info {
  color: var(--muted-text, #8b949e);
  font-size: 14px;
}

.contact-info p {
  margin: 0 0 8px 0;
}

.contact-info a {
  color: var(--highlight-color, #58a6ff);
}

/* Element Plus 组件样式覆盖 */
:deep(.el-collapse) {
  border: none;
  --el-collapse-header-bg-color: var(--bg-color, #0d1117);
  --el-collapse-content-bg-color: var(--secondary-bg, #161b22);
}

:deep(.el-collapse-item__header) {
  background-color: var(--bg-color, #0d1117);
  color: var(--text-color, #e6edf3);
  border-bottom-color: var(--border-color, #30363d);
  font-weight: 500;
}

:deep(.el-collapse-item__wrap) {
  background-color: var(--secondary-bg, #161b22);
  border-bottom-color: var(--border-color, #30363d);
}

:deep(.el-collapse-item__content) {
  color: var(--muted-text, #8b949e);
  padding: 16px 20px;
}

:deep(.el-collapse-item__content p) {
  margin: 0 0 12px 0;
}

:deep(.el-collapse-item__content ul) {
  margin: 0;
  padding-left: 20px;
}

:deep(.el-collapse-item__content li) {
  margin-bottom: 6px;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 768px) {
  .usage-page {
    padding: 16px;
  }
  
  .step-item {
    flex-direction: column;
    gap: 12px;
  }
  
  .code-block {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
