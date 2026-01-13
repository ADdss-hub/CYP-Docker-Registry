import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized } from 'vue-router'

// Whitelist routes that don't require authentication
const WHITELIST = ['login', 'locked', 'share-access']

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/locked',
      name: 'locked',
      component: () => import('@/views/Locked.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/views/Dashboard.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/images',
      name: 'images',
      component: () => import('@/views/Images.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/accelerator',
      name: 'accelerator',
      component: () => import('@/views/Accelerator.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/system',
      name: 'system',
      component: () => import('@/views/System.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/Settings.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/about',
      name: 'about',
      component: () => import('@/views/About.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/audit',
      name: 'audit',
      component: () => import('@/views/Audit.vue'),
      meta: { requiresAuth: true, roles: ['admin'] }
    },
    {
      path: '/orgs',
      name: 'orgs',
      component: () => import('@/views/Org.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/share',
      name: 'share',
      component: () => import('@/views/Share.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tokens',
      name: 'tokens',
      component: () => import('@/views/Tokens.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/config',
      name: 'config',
      component: () => import('@/views/Config.vue'),
      meta: { requiresAuth: true, roles: ['admin'] }
    },
    {
      path: '/signatures',
      name: 'signatures',
      component: () => import('@/views/Signature.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/sbom',
      name: 'sbom',
      component: () => import('@/views/Sbom.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/p2p',
      name: 'p2p',
      component: () => import('@/views/P2P.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tuf',
      name: 'tuf',
      component: () => import('@/views/TUF.vue'),
      meta: { requiresAuth: true, roles: ['admin'] }
    },
    {
      path: '/s/:code',
      name: 'share-access',
      component: () => import('@/views/ShareAccess.vue'),
      meta: { requiresAuth: false }
    }
  ]
})

// Navigation guard for authentication
router.beforeEach(async (to: RouteLocationNormalized, _from: RouteLocationNormalized, next) => {
  // Dynamic import to avoid circular dependency
  const { useAuthStore } = await import('@/stores/auth')
  const { useLockStore } = await import('@/stores/lock')
  
  const authStore = useAuthStore()
  const lockStore = useLockStore()

  // Check if system is locked
  if (lockStore.isLocked && to.name !== 'locked') {
    next({ name: 'locked' })
    return
  }

  // Whitelist routes
  if (WHITELIST.includes(to.name as string)) {
    next()
    return
  }

  // Restore session if needed
  if (authStore.token && !authStore.user) {
    await authStore.restoreSession()
  }

  // Check authentication
  if (to.meta.requiresAuth !== false && !authStore.isAuthenticated) {
    console.warn('[Security] Unauthorized access blocked:', to.path)
    
    // Log access attempt
    logAccessAttempt(to.path, 'unauthorized_access')
    
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  // Check role-based access
  if (to.meta.roles && Array.isArray(to.meta.roles)) {
    const userRole = authStore.user?.role
    if (!userRole || !to.meta.roles.includes(userRole)) {
      console.warn('[Security] Insufficient permissions:', userRole, 'required:', to.meta.roles)
      next({ name: 'dashboard' })
      return
    }
  }

  next()
})

// Log access attempts (for security auditing)
async function logAccessAttempt(path: string, attemptType: string) {
  try {
    // This would be sent to the backend for logging
    console.log('[Audit] Access attempt:', { path, attemptType, timestamp: new Date().toISOString() })
  } catch {
    // Ignore logging errors
  }
}

export default router
