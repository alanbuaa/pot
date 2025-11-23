<template>
  <div class="vdf-monitor card h-full">
     <div class="box">
			<div class="tit"><span>VDF 计算监控</span><p></p></div>
    </div>
    
    <a-spin :spinning="loading">
      <div class="space-y-6">
        <!-- VDF0 进度 -->
        <div class="text-center">
          <div class="text-sm text-gray-400 mb-2">VDF0</div>
          <a-progress
            type="circle"
            :percent="vdf?.vdf0?.progress || 0"
            :width="100"
            :stroke-color="{ '0%': '#ff69b4', '100%': '#ff1493' }"
          >
            <template #format="percent">
              <div class="text-center">
                <div class="text-xl font-bold">{{ percent }}%</div>
                <div class="text-xs text-gray-400">
                  {{ formatNumber(vdf?.vdf0?.iterations || 0) }}
                </div>
              </div>
            </template>
          </a-progress>
          <div class="mt-2">
            <a-tag :color="getStatusColor(vdf?.vdf0?.status)">
              {{ vdf?.vdf0?.status || 'idle' }}
            </a-tag>
          </div>
        </div>
        
        <!-- VDF1 并行线程 -->
        <div>
          <div class="text-sm text-gray-400 mb-2">VDF1 并行线程</div>
          <div class="space-y-2">
            <div v-for="worker in vdf?.vdf1 || []" :key="worker.workerId">
              <div class="flex items-center justify-between mb-1 text-xs">
                <span>线程 {{ worker.workerId }}</span>
                <span>{{ worker.progress }}%</span>
              </div>
              <a-progress
                :percent="worker.progress"
                :stroke-color="{ '0%': '#4488ff', '100%': '#00d4ff' }"
                :show-info="false"
                size="small"
              />
            </div>
          </div>
        </div>
        
        <!-- VDFHalf 进度 -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-gray-400">VDFHalf</span>
            <span class="text-white text-sm">{{ vdf?.vdfHalf?.progress || 0 }}%</span>
          </div>
          <a-progress
            :percent="vdf?.vdfHalf?.progress || 0"
            :stroke-color="{ '0%': '#9370db', '100%': '#8a2be2' }"
          />
        </div>
        
        <!-- VDF Checker -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">验证失败次数</span>
          <a-tag :color="vdf?.vdfChecker?.verifyFailCount ? 'red' : 'green'">
            {{ vdf?.vdfChecker?.verifyFailCount || 0 }}
          </a-tag>
        </div>
        
        <!-- 平均计算耗时 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">平均耗时</span>
          <span class="text-white">{{ vdf?.avgComputeTime?.toFixed(2) || '0.00' }}ms</span>
        </div>
        
        <!-- CPU 线程数 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">CPU 线程数</span>
          <a-tag color="blue">{{ vdf?.cpuCounter || 0 }}</a-tag>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { ClockCircleOutlined } from '@ant-design/icons-vue'
import { usePotStore } from '@/stores/pot'
import { formatNumber } from '@/utils/format'

const potStore = usePotStore()
const { vdf, loading } = storeToRefs(potStore)

function getStatusColor(status?: string) {
  const map: Record<string, string> = {
    'computing': 'processing',
    'done': 'success',
    'idle': 'default'
  }
  return map[status || 'idle'] || 'default'
}
</script>

<style scoped>
.space-y-6 > * + * {
  margin-top: 1.5rem;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
