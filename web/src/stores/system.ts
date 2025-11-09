import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { SystemOverview } from '@/types/api'

export const useSystemStore = defineStore('system', () => {
  const overview = ref<SystemOverview | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchOverview = async () => {
    loading.value = true
    error.value = null
    try {
      overview.value = await api.getSystemOverview()
    } catch (err: any) {
      error.value = err.message || '获取系统概览失败'
      console.error('Failed to fetch system overview:', err)
    } finally {
      loading.value = false
    }
  }

  return {
    overview,
    loading,
    error,
    fetchOverview
  }
})
