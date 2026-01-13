import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '@/utils/request'

const TOKEN_KEY = 'cyp-registry-token'
const USER_KEY = 'cyp-registry-user'
const SESSION_KEY = 'cyp-registry-session'

export interface User {
  id: number
  username: string
  email?: string
  role: string
  is_active: boolean
}

export interface Session {
  id: string
  user_id: number
  ip: string
  expires_at: string
}

export interface LoginRequest {
  username: string
  password: string
  captcha?: string
}

export interface LoginResponse {
  user: User
  token: string
  session: Session
  must_change_password: boolean
  lock_warning: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<User | null>(null)
  const session = ref<Session | null>(null)
  const loading = ref(false)

  // Initialize user from localStorage
  const storedUser = localStorage.getItem(USER_KEY)
  if (storedUser) {
    try {
      user.value = JSON.parse(storedUser)
    } catch {
      localStorage.removeItem(USER_KEY)
    }
  }

  const storedSession = localStorage.getItem(SESSION_KEY)
  if (storedSession) {
    try {
      session.value = JSON.parse(storedSession)
    } catch {
      localStorage.removeItem(SESSION_KEY)
    }
  }

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(credentials: LoginRequest): Promise<LoginResponse> {
    loading.value = true
    try {
      const response = await request.post<LoginResponse>('/api/v1/auth/login', credentials)
      const data = response.data

      // Store token and user
      token.value = data.token
      user.value = data.user
      session.value = data.session

      localStorage.setItem(TOKEN_KEY, data.token)
      localStorage.setItem(USER_KEY, JSON.stringify(data.user))
      localStorage.setItem(SESSION_KEY, JSON.stringify(data.session))

      return data
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await request.post('/api/v1/auth/logout')
    } catch {
      // Ignore errors during logout
    } finally {
      clearAuth()
    }
  }

  function clearAuth() {
    token.value = null
    user.value = null
    session.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(SESSION_KEY)
  }

  async function verifyToken(): Promise<boolean> {
    if (!token.value) {
      return false
    }

    try {
      const response = await request.post('/api/v1/auth/verify-token', {
        token: token.value
      })
      return response.data.valid
    } catch {
      clearAuth()
      return false
    }
  }

  async function restoreSession(): Promise<boolean> {
    if (!token.value) {
      return false
    }

    const valid = await verifyToken()
    if (!valid) {
      clearAuth()
    }
    return valid
  }

  async function getCurrentUser(): Promise<User | null> {
    if (!token.value) {
      return null
    }

    try {
      const response = await request.get<{ user: User }>('/api/v1/auth/me')
      user.value = response.data.user
      localStorage.setItem(USER_KEY, JSON.stringify(user.value))
      return user.value
    } catch {
      return null
    }
  }

  return {
    token,
    user,
    session,
    loading,
    isAuthenticated,
    isAdmin,
    login,
    logout,
    clearAuth,
    verifyToken,
    restoreSession,
    getCurrentUser
  }
})
