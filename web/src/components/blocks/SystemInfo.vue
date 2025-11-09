<template>
  <div class="system-info card h-full">
    <h3 class="text-sm font-semibold mb-3 flex items-center gap-2">
      <InfoCircleOutlined />
      系统信息
    </h3>
    
    <div class="space-y-2 text-xs">
      <div class="flex items-center justify-between">
        <span class="text-gray-400">执行器状态</span>
        <a-badge 
          :status="overview?.executorStatus === 'normal' ? 'success' : 'error'" 
          :text="overview?.executorStatus || 'unknown'"
        />
      </div>
      
      <div class="flex items-center justify-between">
        <span class="text-gray-400">平均出块时间</span>
        <span class="text-white">{{ overview?.avgBlockTime?.toFixed(2) || '0.00' }}s</span>
      </div>
      
      <div class="flex items-center justify-between">
        <span class="text-gray-400">总存储</span>
        <span class="text-white font-semibold">{{ formatStorage(totalStorage) }}</span>
      </div>
      
      <div class="flex items-center justify-between">
        <span class="text-gray-400">已使用</span>
        <span class="text-white">{{ formatStorage(usedStorage) }}</span>
      </div>
      
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-gray-400">存储使用率</span>
          <span class="text-white">{{ storagePercentage.toFixed(1) }}%</span>
        </div>
        <a-progress
          :percent="storagePercentage"
          :stroke-color="getStorageColor(storagePercentage)"
          :show-info="false"
          size="small"
        />
      </div>
      
      <div class="text-center pt-2 text-gray-500 text-xs">
        Mock 模式运行中
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { InfoCircleOutlined } from '@ant-design/icons-vue'
import { useSystemStore } from '@/stores/system'
import { useStorageStore } from '@/stores/storage'

const systemStore = useSystemStore()
const storageStore = useStorageStore()
const { overview } = storeToRefs(systemStore)
const { status: storageInfo } = storeToRefs(storageStore)

// 计算总存储和使用率
const totalStorage = computed(() => {
  return (storageInfo.value?.totalSize || 0) / 1024 // 转为 GB
})

const usedStorage = computed(() => {
  const total = storageInfo.value?.totalSize || 0
  return total * 0.35 / 1024 // 假设使用35%，转为GB
})

const storagePercentage = computed(() => {
  return 35 // 假设35%使用率
})

function formatStorage(gb: number): string {
  if (gb >= 1024) {
    return (gb / 1024).toFixed(2) + ' TB'
  }
  return gb.toFixed(2) + ' GB'
}

function getStorageColor(percentage: number): string {
  if (percentage >= 90) return '#ff4d4f'
  if (percentage >= 70) return '#faad14'
  return '#9370db'
}
</script>

<style scoped>
.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
