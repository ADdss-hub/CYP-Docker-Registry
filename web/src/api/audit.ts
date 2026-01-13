import request from '@/utils/request'

export interface AuditLog {
  id: number
  timestamp: string
  level: string
  event: string
  user_id?: number
  username?: string
  ip_address: string
  user_agent?: string
  action: string
  resource?: string
  status: string
  details?: Record<string, any>
  blockchain_hash?: string
}

export interface AuditLogQuery {
  page?: number
  page_size?: number
  start_date?: string
  end_date?: string
  level?: string
  event?: string
  user_id?: number
  ip_address?: string
  status?: string
}

// List audit logs
export function listAuditLogs(params?: AuditLogQuery) {
  return request.get('/api/v1/audit/logs', { params })
}

// Get audit log by ID
export function getAuditLog(id: number) {
  return request.get(`/api/v1/audit/logs/${id}`)
}

// Export audit logs
export function exportAuditLogs(params?: { start_date?: string; end_date?: string }) {
  return request.get('/api/v1/audit/logs/export', { params, responseType: 'blob' })
}

// Verify audit log integrity
export function verifyAuditLogs() {
  return request.post('/api/v1/audit/verify')
}

// Get audit statistics
export function getAuditStats() {
  return request.get('/api/v1/audit/stats')
}
