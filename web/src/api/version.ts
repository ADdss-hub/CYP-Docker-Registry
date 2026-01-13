import request from '@/utils/request'

export interface VersionInfo {
  version: string
  build_time?: string
  git_commit?: string
}

/**
 * 获取系统版本信息
 */
export async function getVersion(): Promise<VersionInfo> {
  return request.get('/version')
}

/**
 * 获取完整版本信息（包含构建信息）
 */
export async function getFullVersion(): Promise<VersionInfo> {
  return request.get('/version/full')
}
