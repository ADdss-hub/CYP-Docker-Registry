import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import request from '@/utils/request'

export interface LockStatus {
  is_locked: boolean
  lock_reason: string
  lock_type: string
  locked_at: string
  locked_by_ip: string
  locked_by_user?: string
  unlock_at?: string
  require_manual: boolean
}

export const useLockStore = defineStore('lock', () => {
  const lockStatus = ref<LockStatus | null>(null)
  const loading = ref(false)

  const isLocked = computed(() => lockStatus.value?.is_locked ?? false)
  const lockReason = computed(() => lockStatus.value?.lock_reason ?? '')
  const requireManual = computed(() => lockStatus.value?.require_manual ?? true)

  async function fetchLockStatus(): Promise<LockStatus | null> {
    loading.value = true
    try {
      const response = await request.get<LockStatus>('/api/v1/system/lock/status')
      lockStatus.value = response.data
      return lockStatus.value
    } catch {
      return null
    } finally {
      loading.value = false
    }
  }

  function setLockStatus(locked: boolean, reason: string = '') {
    if (!lockStatus.value) {
      lockStatus.value = {
        is_locked: locked,
        lock_reason: reason,
        lock_type: 'rule_triggered',
        locked_at: new Date().toISOString(),
        locked_by_ip: '',
        require_manual: true
      }
    } else {
      lockStatus.value.is_locked = locked
      lockStatus.value.lock_reason = reason
    }
  }

  async function requestUnlock(password: string): Promise<boolean> {
    loading.value = true
    try {
      await request.post('/api/v1/system/lock/unlock', { password })
      lockStatus.value = null
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  function clearLockStatus() {
    lockStatus.value = null
  }

  return {
    lockStatus,
    loading,
    isLocked,
    lockReason,
    requireManual,
    fetchLockStatus,
    setLockStatus,
    requestUnlock,
    clearLockStatus
  }
})
