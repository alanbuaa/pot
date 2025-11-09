import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { StorageStatus } from '@/types/api'

export const useStorageStore = defineStore('storage', () => {
  const status = ref<StorageStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      status.value = await api.getStorageStatus()
    } catch (err: any) {
      error.value = err.message || '获取存储状态失败'
      console.error('Failed to fetch storage status:', err)
    } finally {
      loading.value = false
    }
  }

  return {
    status,
    loading,
    error,
    fetchStatus
  }
})
