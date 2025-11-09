<template>
  <div class="mempool-monitor card h-full">
    <h3 class="text-lg font-semibold mb-4 text-transaction flex items-center gap-2">
      <DatabaseOutlined />
      交易池监控
    </h3>
    
    <a-spin :spinning="loading">
      <div class="space-y-4">
        <!-- 交易池总大小 -->
        <div>
          <div class="text-sm text-gray-400 mb-1">交易池总大小</div>
          <div class="text-2xl font-bold text-white">
            {{ status?.totalSize || 0 }}
          </div>
        </div>
        
        <!-- 已提议/待提议 -->
        <div class="grid grid-cols-2 gap-2">
          <div class="text-center p-2 bg-green-500 bg-opacity-10 rounded">
            <div class="text-xs text-gray-400">已提议</div>
            <div class="text-lg font-semibold text-green-400">
              {{ status?.markedTxs || 0 }}
            </div>
          </div>
          <div class="text-center p-2 bg-yellow-500 bg-opacity-10 rounded">
            <div class="text-xs text-gray-400">待提议</div>
            <div class="text-lg font-semibold text-yellow-400">
              {{ status?.unmarkedTxs || 0 }}
            </div>
          </div>
        </div>
        
        <!-- 交易类型分布 -->
        <div>
          <div class="text-sm text-gray-400 mb-2">交易类型分布</div>
          <div class="space-y-2">
            <div class="flex items-center justify-between text-sm">
              <span>普通交易</span>
              <a-tag>{{ status?.txTypes?.normal || 0 }}</a-tag>
            </div>
            <div class="flex items-center justify-between text-sm">
              <span>BCI交易</span>
              <a-tag color="gold">{{ status?.txTypes?.bci || 0 }}</a-tag>
            </div>
            <div class="flex items-center justify-between text-sm">
              <span>Devastate</span>
              <a-tag color="red">{{ status?.txTypes?.devastate || 0 }}</a-tag>
            </div>
          </div>
        </div>
        
        <!-- 平均确认时间 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">平均确认时间</span>
          <span class="text-white">{{ status?.avgConfirmTime?.toFixed(2) || '0.00' }}s</span>
        </div>
        
        <!-- 验证成功率 -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-gray-400">验证成功率</span>
            <span class="text-white">{{ status?.verifySuccessRate?.toFixed(1) || '0.0' }}%</span>
          </div>
          <a-progress
            :percent="status?.verifySuccessRate || 0"
            :stroke-color="{ '0%': '#ffa500', '100%': '#00ff88' }"
            :show-info="false"
          />
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { DatabaseOutlined } from '@ant-design/icons-vue'
import { useMempoolStore } from '@/stores/mempool'

const mempoolStore = useMempoolStore()
const { status, loading } = storeToRefs(mempoolStore)
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
