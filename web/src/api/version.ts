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
  const response = await request.get('/api/version')
  // 处理响应数据格式
  const data = response.data?.data || response.data || response
  return {
    version: data.version || '未知',
    build_time: data.build_time,
    git_commit: data.git_commit
  }
}

/**
 * 获取完整版本信息（包含构建信息）
 */
export async function getFullVersion(): Promise<VersionInfo> {
  const response = await request.get('/api/version/full')
  // 处理响应数据格式
  const data = response.data?.data || response.data || response
  return {
    version: data.version || '未知',
    build_time: data.build_time,
    git_commit: data.git_commit
  }
}
