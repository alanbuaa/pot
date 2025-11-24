<template>
  <div class="pot-consensus card h-full">
    <div class="box">
			<div class="tit"><span>POT 基本信息</span><p></p></div>
    </div>

    <a-spin :spinning="loading">
      <div class="progress-grid">
        <!-- 第一行 -->
        <CircleProgress 
          :value="status?.epoch || 0" 
          title="Epoch"
          :showPercentSign="false"
          color="rgba(243, 83, 49, 0.8)"
          backgroundColor="rgba(243, 83, 49, 0.1)"
          outerRingColor="rgba(243, 83, 49, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
        
        <CircleProgress 
          :value="normalizeValue(status?.difficulty || '', 0, 1000000000)" 
          title="挖矿难度"
          :showPercentSign="false"
          color="rgba(0, 153, 255, 0.8)"
          backgroundColor="rgba(0, 153, 255, 0.1)"
          outerRingColor="rgba(0, 153, 255, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
        
        <CircleProgress 
          :value="normalizeValue(status?.currentHeight || 0, 0, 100000)" 
          :displayValue="Math.floor(status?.currentHeight || 0).toString()"
          title="区块高度"
          :showPercentSign="false"
          color="rgba(255, 215, 0, 0.8)"
          backgroundColor="rgba(255, 215, 0, 0.1)"
          outerRingColor="rgba(255, 215, 0, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
        
        <!-- 第二行 -->
        <CircleProgress 
          :value="status?.workFlag ? 100 : 0" 
          title="挖矿状态"
          :showPercentSign="false"
          color="rgba(0, 255, 136, 0.8)"
          backgroundColor="rgba(0, 255, 136, 0.1)"
          outerRingColor="rgba(0, 255, 136, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
        
        <CircleProgress 
          :value="status?.uncleCount || 0" 
          title="叔块数量"
          :showPercentSign="false"
          color="rgba(255, 165, 0, 0.8)"
          backgroundColor="rgba(255, 165, 0, 0.1)"
          outerRingColor="rgba(255, 165, 0, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
        
        <CircleProgress 
          :value="normalizeValue(status?.avgMiningTime || 0, 0, 60)" 
          :displayValue="(status?.avgMiningTime || 0).toFixed(2)"
          title="平均出块时间"
          :showPercentSign="false"
          color="rgba(138, 43, 226, 0.8)"
          backgroundColor="rgba(138, 43, 226, 0.1)"
          outerRingColor="rgba(138, 43, 226, 0.3)"
          width="100%"
          height="100px"
          :innerRadius="[35, 42]"
        />
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { usePotStore } from '@/stores/pot'
import CircleProgress from '@/components/CircleProgress.vue'

const potStore = usePotStore()
const { status, loading } = storeToRefs(potStore)

// 将值标准化到0-100范围用于显示
function normalizeValue(value: number | string, min: number, max: number): number {
  const numValue = typeof value === 'string' ? parseInt(value) : value
  if (isNaN(numValue)) return 0
  return Math.min(100, Math.max(0, ((numValue - min) / (max - min)) * 100))
}
</script>

<style scoped>
.progress-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  padding: 0.25rem;
}
</style>
