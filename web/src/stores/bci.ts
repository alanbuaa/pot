import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { BCIStatus } from '@/types/api'

export const useBCIStore = defineStore('bci', () => {
  const status = ref<BCIStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      status.value = await api.getBCIStatus()
    } catch (err: any) {
      error.value = err.message || '获取BCI状态失败'
      console.error('Failed to fetch BCI status:', err)
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
