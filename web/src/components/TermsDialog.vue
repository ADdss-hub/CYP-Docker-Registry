<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElDialog, ElButton, ElCheckbox, ElScrollbar } from 'element-plus'

const TERMS_ACCEPTED_KEY = 'cyp-docker-registry-terms-accepted'

const visible = ref(false)
const agreed = ref(false)
const currentYear = new Date().getFullYear()

const emit = defineEmits<{
  (e: 'accepted'): void
}>()

onMounted(() => {
  const accepted = localStorage.getItem(TERMS_ACCEPTED_KEY)
  if (accepted !== 'true') {
    visible.value = true
  } else {
    emit('accepted')
  }
})

const handleAccept = () => {
  if (agreed.value) {
    localStorage.setItem(TERMS_ACCEPTED_KEY, 'true')
    visible.value = false
    emit('accepted')
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="使用条款"
    width="600px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    class="terms-dialog"
    align-center
  >
    <div class="terms-content">
      <el-scrollbar height="400px">
        <div class="terms-section">
          <h3>欢迎使用容器镜像个人仓库</h3>
          <p class="welcome-text">
            在开始使用本软件之前，请仔细阅读以下条款。继续使用即表示您同意遵守这些条款。
          </p>
        </div>

        <div class="terms-section">
          <h4>版权声明</h4>
          <p>Copyright © {{ currentYear }} CYP. All rights reserved.</p>
          <p>本软件及其相关文档的版权归作者所有。未经作者书面许可，不得以任何形式复制、修改、分发或使用本软件的任何部分。</p>
        </div>

        <div class="terms-section disclaimer">
          <h4>免责声明</h4>
          <p>本软件按"原样"提供，不提供任何明示或暗示的保证，包括但不限于对适销性、特定用途适用性和非侵权性的保证。</p>
          <p>在任何情况下，作者或版权持有人均不对因使用本软件或与本软件相关的任何索赔、损害或其他责任负责，无论是在合同诉讼、侵权行为或其他方面。</p>
          <p>用户应自行承担使用本软件的风险，并对使用本软件所产生的任何后果负责。</p>
        </div>

        <div class="terms-section">
          <h4>使用条款</h4>
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

        <div class="terms-section contact">
          <h4>联系方式</h4>
          <p>作者：CYP</p>
          <p>邮箱：<a href="mailto:nasDSSCYP@outlook.com">nasDSSCYP@outlook.com</a></p>
        </div>
      </el-scrollbar>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-checkbox v-model="agreed" class="agree-checkbox">
          我已阅读并同意上述使用条款
        </el-checkbox>
        <el-button
          type="primary"
          :disabled="!agreed"
          @click="handleAccept"
        >
          开始使用
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.terms-dialog :deep(.el-dialog) {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
}

.terms-dialog :deep(.el-dialog__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
}

.terms-dialog :deep(.el-dialog__title) {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.terms-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.terms-dialog :deep(.el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
}

.terms-content {
  padding: 24px;
}

.terms-section {
  margin-bottom: 24px;
}

.terms-section:last-child {
  margin-bottom: 0;
}

.terms-section h3 {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 12px;
}

.terms-section h4 {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.welcome-text {
  color: var(--muted-text);
  font-size: 14px;
  line-height: 1.8;
}

.terms-section p {
  color: var(--muted-text);
  font-size: 14px;
  line-height: 1.8;
  margin-bottom: 8px;
}

.terms-section p:last-child {
  margin-bottom: 0;
}

.terms-section.disclaimer {
  background-color: rgba(248, 81, 73, 0.1);
  border-left: 3px solid var(--error-color);
  padding: 16px;
  border-radius: var(--radius-sm);
}

.terms-section.disclaimer h4 {
  border-bottom: none;
  padding-bottom: 0;
  color: var(--error-color);
}

.terms-section ol {
  padding-left: 20px;
  margin: 0;
}

.terms-section li {
  margin-bottom: 16px;
  color: var(--muted-text);
}

.terms-section li:last-child {
  margin-bottom: 0;
}

.terms-section li strong {
  color: var(--text-color);
  display: block;
  margin-bottom: 4px;
}

.terms-section li p {
  margin: 0;
}

.terms-section.contact {
  background-color: var(--bg-color);
  padding: 16px;
  border-radius: var(--radius-sm);
}

.terms-section.contact h4 {
  border-bottom: none;
  padding-bottom: 0;
}

.terms-section.contact a {
  color: var(--highlight-color);
}

.terms-section.contact a:hover {
  text-decoration: underline;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.agree-checkbox {
  --el-checkbox-text-color: var(--muted-text);
}

.agree-checkbox :deep(.el-checkbox__label) {
  color: var(--muted-text);
  font-size: 14px;
}

.agree-checkbox :deep(.el-checkbox__input.is-checked + .el-checkbox__label) {
  color: var(--text-color);
}
</style>
