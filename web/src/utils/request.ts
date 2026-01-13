import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

const TOKEN_KEY = 'cyp-docker-registry-token'

// Generate a unique request ID
function generateRequestID(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

const request: AxiosInstance = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest'
  }
})

// 请求拦截器
request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Add authorization header
    const token = localStorage.getItem(TOKEN_KEY)
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Add security headers
    if (config.headers) {
      config.headers['X-Request-ID'] = generateRequestID()
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse) => {
    return response
  },
  async (error) => {
    const { response } = error

    if (response?.status === 401) {
      // Clear auth data
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem('cyp-docker-registry-user')
      localStorage.removeItem('cyp-docker-registry-session')
      
      // Redirect to login (avoid circular import)
      if (window.location.pathname !== '/login') {
        window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`
      }
    }

    if (response?.status === 403) {
      const data = response.data
      if (data?.details === 'system_locked') {
        // Redirect to locked page
        if (window.location.pathname !== '/locked') {
          window.location.href = '/locked'
        }
      }
    }

    if (response?.status === 429) {
      console.warn('[RateLimit] Too many requests')
    }

    return Promise.reject(error)
  }
)

export default request
