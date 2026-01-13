import request from '@/utils/request'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  user: {
    id: number
    username: string
    role: string
  }
  token: string
  must_change_password?: boolean
}

export interface User {
  id: number
  username: string
  email: string
  role: string
  is_active: boolean
  created_at: string
}

// Login
export function login(data: LoginRequest) {
  return request.post<LoginResponse>('/api/v1/auth/login', data)
}

// Logout
export function logout() {
  return request.post('/api/v1/auth/logout')
}

// Get current user
export function getCurrentUser() {
  return request.get<User>('/api/v1/auth/me')
}

// Change password
export function changePassword(data: { old_password: string; new_password: string }) {
  return request.post('/api/v1/auth/change-password', data)
}

// Verify token
export function verifyToken() {
  return request.post('/api/v1/auth/verify-token')
}
