<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getVersion, getFullVersion } from '@/api/version'

interface VersionInfo {
  version: string
  build_time?: string
  git_commit?: string
}

const versionInfo = ref<VersionInfo | null>(null)
const currentYear = new Date().getFullYear()

onMounted(async () => {
  try {
    const info = await getFullVersion()
    versionInfo.value = info
  } catch {
    try {
      const info = await getVersion()
      versionInfo.value = info
    } catch {
      versionInfo.value = { version: '未知' }
    }
  }
})
</script>

<template>
  <div class="about-page">
    <!-- 项目信息 -->
    <div class="section hero">
      <div class="logo-container">
        <div class="logo">CR</div>
        <h1>容器镜像个人仓库</h1>
        <p class="subtitle">Container Registry</p>
      </div>
      <div class="version-info" v-if="versionInfo">
        <span class="version">v{{ versionInfo.version }}</span>
        <span class="build-info" v-if="versionInfo.build_time">
          构建时间: {{ versionInfo.build_time }}
        </span>
      </div>
    </div>

    <!-- 作者信息 -->
    <div class="section">
      <h3>作者信息</h3>
      <div class="author-card">
        <div class="author-avatar">CYP</div>
        <div class="author-info">
          <div class="author-name">CYP</div>
          <div class="author-contact">
            <span class="label">联系方式：</span>
            <a href="mailto:nasDSSCYP@outlook.com">nasDSSCYP@outlook.com</a>
          </div>
        </div>
      </div>
    </div>

    <!-- 版权声明 -->
    <div class="section">
      <h3>版权声明</h3>
      <div class="legal-content">
        <p>Copyright © {{ currentYear }} CYP. All rights reserved.</p>
        <p>本软件及其相关文档的版权归作者所有。未经作者书面许可，不得以任何形式复制、修改、分发或使用本软件的任何部分。</p>
      </div>
    </div>

    <!-- 免责声明 -->
    <div class="section">
      <h3>免责声明</h3>
      <div class="legal-content disclaimer">
        <p>本软件按"原样"提供，不提供任何明示或暗示的保证，包括但不限于对适销性、特定用途适用性和非侵权性的保证。</p>
        <p>在任何情况下，作者或版权持有人均不对因使用本软件或与本软件相关的任何索赔、损害或其他责任负责，无论是在合同诉讼、侵权行为或其他方面。</p>
        <p>用户应自行承担使用本软件的风险，并对使用本软件所产生的任何后果负责。</p>
      </div>
    </div>

    <!-- 使用条款 -->
    <div class="section">
      <h3>使用条款</h3>
      <div class="legal-content terms">
        <ol>
          <li>
            <strong>使用范围</strong>
            <p>本软件仅供个人学习和非商业用途使用。任何商业用途需获得作者的书面授权。</p>
          </li>
          <li>
            <strong>禁止行为</strong>
            <p>禁止将本软件用于任何违法活动，包括但不限于：存储或分发非法内容、侵犯他人知识产权、进行网络攻击等。</p>
          </li>
          <li>
            <strong>合规要求</strong>
            <p>用户应遵守相关法律法规和镜像仓库的使用条款。使用本软件访问第三方服务时，用户应遵守该服务的使用条款。</p>
          </li>
          <li>
            <strong>数据责任</strong>
            <p>用户对存储在本软件中的所有数据负责。作者不对用户数据的丢失、泄露或损坏承担任何责任。</p>
          </li>
          <li>
            <strong>条款修改</strong>
            <p>作者保留随时修改本条款的权利。继续使用本软件即表示接受修改后的条款。</p>
          </li>
        </ol>
      </div>
    </div>

    <!-- 技术栈 -->
    <div class="section">
      <h3>技术栈</h3>
      <div class="tech-stack">
        <div class="tech-group">
          <h4>后端</h4>
          <div class="tech-tags">
            <span class="tech-tag">Go</span>
            <span class="tech-tag">Gin</span>
            <span class="tech-tag">Viper</span>
            <span class="tech-tag">Zap</span>
          </div>
        </div>
        <div class="tech-group">
          <h4>前端</h4>
          <div class="tech-tags">
            <span class="tech-tag">Vue 3</span>
            <span class="tech-tag">TypeScript</span>
            <span class="tech-tag">Element Plus</span>
            <span class="tech-tag">Pinia</span>
            <span class="tech-tag">Vite</span>
          </div>
        </div>
        <div class="tech-group">
          <h4>协议</h4>
          <div class="tech-tags">
            <span class="tech-tag">Docker Registry V2</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 页脚 -->
    <div class="about-footer">
      <p>感谢使用容器镜像个人仓库</p>
      <p class="footer-copyright">© {{ currentYear }} CYP | nasDSSCYP@outlook.com</p>
    </div>
  </div>
</template>

<style scoped>
.about-page {
  animation: fadeIn 0.3s ease-out;
  max-width: 800px;
  margin: 0 auto;
}

.section {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 24px;
  margin-bottom: 24px;
}

.section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.hero {
  text-align: center;
  padding: 40px 24px;
}

.logo-container {
  margin-bottom: 24px;
}

.logo {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #1890ff, #096dd9);
  border-radius: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: bold;
  color: white;
  margin-bottom: 16px;
  box-shadow: 0 8px 24px rgba(24, 144, 255, 0.3);
}

.hero h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 600;
  color: var(--text-color);
}

.subtitle {
  margin: 0;
  font-size: 16px;
  color: var(--muted-text);
  font-family: var(--font-mono);
}

.version-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.version {
  font-size: 18px;
  font-weight: 500;
  color: var(--highlight-color);
  font-family: var(--font-mono);
}

.build-info {
  font-size: 12px;
  color: var(--muted-text);
}

.author-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 20px;
  background-color: var(--bg-color);
  border-radius: var(--radius-sm);
}

.author-avatar {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, #722ed1, #531dab);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
  color: white;
}

.author-info {
  flex: 1;
}

.author-name {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
}

.author-contact {
  font-size: 14px;
}

.author-contact .label {
  color: var(--muted-text);
}

.author-contact a {
  color: var(--highlight-color);
}

.author-contact a:hover {
  text-decoration: underline;
}

.legal-content {
  color: var(--muted-text);
  font-size: 14px;
  line-height: 1.8;
}

.legal-content p {
  margin: 0 0 12px 0;
}

.legal-content p:last-child {
  margin-bottom: 0;
}

.legal-content.disclaimer {
  padding: 16px;
  background-color: rgba(248, 81, 73, 0.1);
  border-radius: var(--radius-sm);
  border-left: 3px solid var(--error-color);
}

.legal-content.terms ol {
  margin: 0;
  padding-left: 20px;
}

.legal-content.terms li {
  margin-bottom: 16px;
}

.legal-content.terms li:last-child {
  margin-bottom: 0;
}

.legal-content.terms strong {
  color: var(--text-color);
  display: block;
  margin-bottom: 8px;
}

.legal-content.terms p {
  margin: 0;
}

.tech-stack {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tech-group h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--muted-text);
}

.tech-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tech-tag {
  padding: 6px 14px;
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  font-size: 13px;
  color: var(--text-color);
  font-family: var(--font-mono);
  transition: border-color 0.2s, color 0.2s;
}

.tech-tag:hover {
  border-color: var(--highlight-color);
  color: var(--highlight-color);
}

.about-footer {
  text-align: center;
  padding: 32px 0;
  color: var(--muted-text);
}

.about-footer p {
  margin: 0 0 8px 0;
}

.footer-copyright {
  font-size: 12px;
  color: var(--subtle-text);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
