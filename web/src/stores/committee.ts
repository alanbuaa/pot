import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { CommitteeStatus } from '@/types/api'

export const useCommitteeStore = defineStore('committee', () => {
  const status = ref<CommitteeStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      status.value = await api.getCommitteeStatus()
    } catch (err: any) {
      error.value = err.message || '获取委员会状态失败'
      console.error('Failed to fetch committee status:', err)
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
