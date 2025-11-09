import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { MempoolStatus } from '@/types/api'

export const useMempoolStore = defineStore('mempool', () => {
  const status = ref<MempoolStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      status.value = await api.getMempoolStatus()
    } catch (err: any) {
      error.value = err.message || '获取交易池状态失败'
      console.error('Failed to fetch mempool status:', err)
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
