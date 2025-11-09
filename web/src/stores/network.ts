import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { NetworkTopology } from '@/types/api'

export const useNetworkStore = defineStore('network', () => {
  const topology = ref<NetworkTopology | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchTopology = async () => {
    loading.value = true
    error.value = null
    try {
      topology.value = await api.getNetworkTopology()
    } catch (err: any) {
      error.value = err.message || '获取网络拓扑失败'
      console.error('Failed to fetch network topology:', err)
    } finally {
      loading.value = false
    }
  }

  return {
    topology,
    loading,
    error,
    fetchTopology
  }
})
