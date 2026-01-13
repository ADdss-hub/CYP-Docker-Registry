import request from '@/utils/request'

export interface Image {
  name: string
  tag: string
  digest: string
  size: number
  created_at: string
  pushed_at: string
  architecture: string
  os: string
  labels: Record<string, string>
}

export interface Repository {
  name: string
  description: string
  tags_count: number
  size: number
  created_at: string
  updated_at: string
  is_public: boolean
}

export interface Tag {
  name: string
  digest: string
  size: number
  created_at: string
}

// List repositories
export function listRepositories(params?: { page?: number; page_size?: number; search?: string }) {
  return request.get('/api/v1/repositories', { params })
}

// Get repository
export function getRepository(name: string) {
  return request.get(`/api/v1/repositories/${encodeURIComponent(name)}`)
}

// Delete repository
export function deleteRepository(name: string) {
  return request.delete(`/api/v1/repositories/${encodeURIComponent(name)}`)
}

// List tags
export function listTags(repo: string, params?: { page?: number; page_size?: number }) {
  return request.get(`/api/v1/repositories/${encodeURIComponent(repo)}/tags`, { params })
}

// Get tag
export function getTag(repo: string, tag: string) {
  return request.get(`/api/v1/repositories/${encodeURIComponent(repo)}/tags/${tag}`)
}

// Delete tag
export function deleteTag(repo: string, tag: string) {
  return request.delete(`/api/v1/repositories/${encodeURIComponent(repo)}/tags/${tag}`)
}

// Get image manifest
export function getManifest(repo: string, reference: string) {
  return request.get(`/v2/${encodeURIComponent(repo)}/manifests/${reference}`)
}
