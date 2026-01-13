<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  House,
  Picture,
  Lightning,
  Monitor,
  Setting,
  InfoFilled,
  OfficeBuilding,
  Share,
  Key,
  Document,
  User,
  SwitchButton,
  Edit,
  List,
  Connection,
  Lock
} from '@element-plus/icons-vue'
import Footer from './Footer.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const menuItems = computed(() => {
  const items = [
    { path: '/', name: '仪表盘', icon: House },
    { path: '/images', name: '镜像管理', icon: Picture },
    { path: '/accelerator', name: '镜像加速', icon: Lightning },
    { path: '/p2p', name: 'P2P 分发', icon: Connection },
    { path: '/signatures', name: '镜像签名', icon: Edit },
    { path: '/sbom', name: 'SBOM 管理', icon: List },
    { path: '/orgs', name: '组织管理', icon: OfficeBuilding },
    { path: '/share', name: '分享管理', icon: Share },
    { path: '/tokens', name: '访问令牌', icon: Key },
    { path: '/system', name: '系统信息', icon: Monitor },
    { path: '/settings', name: '系统设置', icon: Setting },
  ]

  // Admin-only items
  if (authStore.isAdmin) {
    items.push({ path: '/audit', name: '审计日志', icon: Document })
    items.push({ path: '/tuf', name: 'TUF 管理', icon: Lock })
    items.push({ path: '/config', name: '系统配置', icon: Setting })
  }

  items.push({ path: '/about', name: '关于', icon: InfoFilled })

  return items
})

const isCollapse = ref(false)

const handleSelect = (path: string) => {
  router.push(path)
}

const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <el-container class="layout">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '200px'" class="aside">
      <div class="logo">
        <span v-if="!isCollapse">容器镜像仓库</span>
        <span v-else>CR</span>
      </div>
      <el-menu
        :default-active="route.path"
        class="menu"
        :collapse="isCollapse"
        background-color="#161b22"
        text-color="#8b949e"
        active-text-color="#58a6ff"
        @select="handleSelect"
      >
        <el-menu-item
          v-for="item in menuItems"
          :key="item.path"
          :index="item.path"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.name }}</template>
        </el-menu-item>
      </el-menu>
      <div class="collapse-btn" @click="isCollapse = !isCollapse">
        <el-icon>
          <component :is="isCollapse ? 'ArrowRight' : 'ArrowLeft'" />
        </el-icon>
      </div>
    </el-aside>

    <!-- 主内容区 -->
    <el-container class="main-container">
      <el-header class="header">
        <div class="header-title">
          {{ menuItems.find(item => item.path === route.path)?.name || '容器镜像仓库' }}
        </div>
        <div class="header-info">
          <el-dropdown trigger="click">
            <span class="user-dropdown">
              <el-icon><User /></el-icon>
              <span>{{ authStore.user?.username || 'User' }}</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="router.push('/tokens')">
                  <el-icon><Key /></el-icon>
                  访问令牌
                </el-dropdown-item>
                <el-dropdown-item @click="router.push('/settings')">
                  <el-icon><Setting /></el-icon>
                  设置
                </el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main">
        <slot />
      </el-main>

      <Footer />
    </el-container>
  </el-container>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  background-color: var(--bg-color);
}

.aside {
  background-color: #161b22;
  border-right: 1px solid #30363d;
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: bold;
  color: var(--highlight-color);
  border-bottom: 1px solid #30363d;
  white-space: nowrap;
  overflow: hidden;
}

.menu {
  flex: 1;
  border-right: none;
}

.menu :deep(.el-menu-item) {
  border-radius: 8px;
  margin: 4px 8px;
}

.menu :deep(.el-menu-item:hover) {
  background-color: #21262d !important;
}

.menu :deep(.el-menu-item.is-active) {
  background-color: rgba(88, 166, 255, 0.1) !important;
}

.collapse-btn {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #8b949e;
  border-top: 1px solid #30363d;
}

.collapse-btn:hover {
  color: var(--highlight-color);
}

.main-container {
  background-color: var(--bg-color);
}

.header {
  background-color: #161b22;
  border-bottom: 1px solid #30363d;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-title {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color);
}

.header-info {
  color: #8b949e;
}

.user-dropdown {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: #8b949e;
  padding: 8px 12px;
  border-radius: 6px;
  transition: all 0.2s;
}

.user-dropdown:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--highlight-color);
}

.author {
  font-size: 14px;
}

.main {
  padding: 24px;
  background-color: var(--bg-color);
}
</style>
