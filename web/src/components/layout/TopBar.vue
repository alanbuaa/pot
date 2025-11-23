<template>
  <div class="top-bar bg-bg-secondary border-b border-border px-6 py-5">
    <div class="flex items-center gap-8">
      <!-- LOGO -->
      <div class="logo flex-shrink-0">
        <div class="logo-box">
          <span class="logo-text">POT</span>
        </div>
      </div>
      
      <!-- 指标区域 - 铺满剩余空间 -->
      <div class="flex items-center gap-8 flex-1 justify-around">
        <!-- 运行时长 -->
        <div class="stat-item-large">
          <div class="stat-label">运行时长</div>
          <div class="stat-value">{{ formatDuration(overview?.uptime || 0) }}</div>
        </div>
        
        <!-- 区块高度 -->
        <div class="stat-item-large">
          <div class="stat-label">区块高度</div>
          <div class="stat-value text-committee-leader">
            {{ formatNumber(overview?.currentHeight || 0) }}
          </div>
        </div>
        
        <!-- 当前TPS -->
        <div class="stat-item-large">
          <div class="stat-label">实时TPS</div>
          <div class="stat-value text-pot-mining">
            {{ overview?.currentTPS?.toFixed(2) || '0.00' }}
          </div>
        </div>
        
        <!-- 节点状态 -->
        <div class="stat-item-large">
          <div class="stat-label">节点状态</div>
          <div class="stat-value flex items-center gap-2">
            <span :class="['status-dot', getNetworkStatusClass()]"></span>
            {{ overview?.onlineNodes || 0 }}/{{ overview?.totalNodes || 0 }}
          </div>
        </div>
        
        <!-- 网络健康度 - 信号格 -->
        <div class="stat-item-large">
          <div class="stat-label">网络健康</div>
          <div class="stat-value flex items-center justify-center">
            <NetworkSignal :level="getSignalLevel()" :status="getNetworkStatus()" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useSystemStore } from '@/stores/system'
import { formatNumber, formatDuration } from '@/utils/format'
import NetworkSignal from '@/components/NetworkSignal.vue'

const systemStore = useSystemStore()
const { overview } = storeToRefs(systemStore)

function getNetworkStatusClass() {
  const status = overview.value?.networkStatus
  if (status === 'healthy') return 'online'
  if (status === 'warning') return 'warning'
  return 'error'
}

function getSignalLevel(): number {
  const utilization = overview.value?.networkUtilization || 0
  if (utilization >= 90) return 4
  if (utilization >= 70) return 3
  if (utilization >= 50) return 2
  if (utilization >= 30) return 1
  return 0
}

function getNetworkStatus(): 'healthy' | 'warning' | 'error' {
  const status = overview.value?.networkStatus
  if (status === 'healthy') return 'healthy'
  if (status === 'warning') return 'warning'
  return 'error'
}
</script>

<style scoped>
.logo-box {
  width: 60px;
  height: 60px;
  background: linear-gradient(135deg, #00d4ff, #0099cc);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 212, 255, 0.3);
}

.logo-text {
  font-size: 24px;
  font-weight: 900;
  color: #fff;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.stat-item-large {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-label {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);
  white-space: nowrap;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: #ffffff;
  display: flex;
  align-items: center;
  gap: 6px;
}

@media (max-width: 1280px) {
  .stat-item-compact {
    min-width: 70px;
  }
  
  .stat-label-compact {
    font-size: 10px;
  }
  
  .stat-value-compact {
    font-size: 13px;
  }
}
</style>
