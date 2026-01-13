import request from '@/utils/request'

export interface SystemInfo {
  version: string
  build_time: string
  go_version: string
  os: string
  arch: string
  num_cpu: number
  hostname: string
  uptime: string
  environment: string
  features: Record<string, boolean>
}

export interface SystemStats {
  memory_usage: {
    alloc: number
    total_alloc: number
    sys: number
    num_gc: number
  }
  goroutine_count: number
  cpu_usage: number
  disk_usage: {
    total: number
    used: number
    free: number
    used_pct: number
  }
  uptime: number
}

export interface HealthStatus {
  status: string
  checks: Array<{
    name: string
    status: string
    message?: string
  }>
  timestamp: string
}

export interface LockStatus {
  is_locked: boolean
  lock_reason?: string
  locked_at?: string
  locked_by_ip?: string
}

// Get system info
export function getSystemInfo() {
  return request.get<SystemInfo>('/api/v1/system/info')
}

// Get system stats
export function getSystemStats() {
  return request.get<SystemStats>('/api/v1/system/stats')
}

// Get health status
export function getHealthStatus() {
  return request.get<HealthStatus>('/health')
}

// Get lock status
export function getLockStatus() {
  return request.get<LockStatus>('/api/v1/system/lock/status')
}

// Unlock system
export function unlockSystem(password: string) {
  return request.post('/api/v1/system/lock/unlock', { password })
}

// Trigger GC
export function triggerGC() {
  return request.post('/api/v1/system/gc')
}
