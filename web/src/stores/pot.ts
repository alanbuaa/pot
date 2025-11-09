import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { POTStatus, VDFStatus } from '@/types/api'

export const usePotStore = defineStore('pot', () => {
  const status = ref<POTStatus | null>(null)
  const vdf = ref<VDFStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      status.value = await api.getPotStatus()
    } catch (err: any) {
      error.value = err.message || '获取POT状态失败'
      console.error('Failed to fetch POT status:', err)
    } finally {
      loading.value = false
    }
  }

  const fetchVDF = async () => {
    try {
      vdf.value = await api.getVDFStatus()
    } catch (err: any) {
      console.error('Failed to fetch VDF status:', err)
    }
  }

  return {
    status,
    vdf,
    loading,
    error,
    fetchStatus,
    fetchVDF
  }
})
