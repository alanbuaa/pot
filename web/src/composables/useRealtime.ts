import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 轮询获取数据的组合式函数
 */
export function useRealtime<T>(
  fetchFn: () => Promise<T>,
  interval: number = 5000
) {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let timer: number | null = null

  const fetch = async () => {
    loading.value = true
    error.value = null
    try {
      data.value = await fetchFn()
    } catch (err: any) {
      error.value = err.message || '获取数据失败'
      console.error('Fetch error:', err)
    } finally {
      loading.value = false
    }
  }

  const startPolling = () => {
    fetch() // 立即执行一次
    timer = window.setInterval(fetch, interval)
  }

  const stopPolling = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(() => {
    startPolling()
  })

  onUnmounted(() => {
    stopPolling()
  })

  return {
    data,
    loading,
    error,
    fetch,
    startPolling,
    stopPolling
  }
}
