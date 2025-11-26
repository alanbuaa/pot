import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { StorageStatus } from '@/types/api'

export const useStorageStore = defineStore('storage', () => {
  const status = ref<StorageStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      // TODO: 实现 getStorageStatus API
      // status.value = await api.getStorageStatus()
      // Mock data for now
      status.value = {
        totalSize: 1024 * 1024 * 1024 * 100, // 100GB
        usedSize: 1024 * 1024 * 1024 * 35, // 35GB
        availableSize: 1024 * 1024 * 1024 * 65, // 65GB
        utilizationRate: 35
      }
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
