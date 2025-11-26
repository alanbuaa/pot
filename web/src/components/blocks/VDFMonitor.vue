<template>
  <div class="vdf-monitor card h-full">
     <div class="box">
			<div class="tit"><span>VDF 计算监控</span><p></p></div>
    </div>
    
    <a-spin :spinning="loading">
      <div class="space-y-3">
        <!-- 平均计算耗时 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">平均耗时</span>
          <span class="text-white">{{ vdf?.avgComputeTime?.toFixed(2) || '0.00' }}ms</span>
        </div>
        
        <!-- CPU 线程数 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">CPU 线程占用数</span>
          <a-tag color="blue">{{ vdf?.cpuCounter || 0 }}</a-tag>
        </div>
        
        <!-- VDF 进度展示 -->
        <div class="flex justify-around items-center">
          <!-- VDF0 进度 -->
          <div class="text-center">
            <CircleProgress
              :value="vdf?.vdf0?.progress || 0"
              :displayValue="formatNumber(vdf?.vdf0?.iterations || 0)"
              title="VDF0"
              :showPercentSign="false"
              color="rgba(255, 105, 180, 0.8)"
              backgroundColor="rgba(255, 105, 180, 0.1)"
              outerRingColor="rgba(255, 105, 180, 0.3)"
              width="120px"
              height="120px"
              :innerRadius="[40, 48]"
            />
            <div class="mt-2">
              <a-tag :color="getStatusColor(vdf?.vdf0?.status)" size="small">
                {{ vdf?.vdf0?.status || 'idle' }}
              </a-tag>
            </div>
          </div>

          <!-- VDF1 线程0 进度 -->
          <div class="text-center">
            <CircleProgress
              :value="vdf?.vdf1?.[0]?.progress || 0"
              :displayValue="formatNumber(vdf?.vdf1?.[0]?.iterations || 0)"
              title="VDF1"
              :showPercentSign="false"
              color="rgba(68, 136, 255, 0.8)"
              backgroundColor="rgba(68, 136, 255, 0.1)"
              outerRingColor="rgba(68, 136, 255, 0.3)"
              width="120px"
              height="120px"
              :innerRadius="[40, 48]"
            />
            <div class="mt-2">
              <a-tag :color="getStatusColor(vdf?.vdf1?.[0]?.status)" size="small">
                {{ vdf?.vdf1?.[0]?.status || 'idle' }}
              </a-tag>
            </div>
          </div>
        </div>
        
        <!-- 运行时长 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">运行时长</span>
          <span class="text-white font-semibold">
            {{ formatDuration(overview?.uptime || 0) }}
          </span>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { usePotStore } from '@/stores/pot'
import { useSystemStore } from '@/stores/system'
import { formatNumber, formatDuration } from '@/utils/format'
import CircleProgress from '@/components/CircleProgress.vue'

const potStore = usePotStore()
const { vdf, loading } = storeToRefs(potStore)
const systemStore = useSystemStore()
const { overview } = storeToRefs(systemStore)

function getStatusColor(status?: string) {
  const map: Record<string, string> = {
    'computing': 'processing',
    'done': 'success',
    'idle': 'orange'
  }
  return map[status || 'idle'] || 'default'
}
</script>

<style scoped>
.space-y-3 > * + * {
  margin-top: 0.75rem;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
