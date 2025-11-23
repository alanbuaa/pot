<template>
  <div class="pot-consensus card h-full">
    <div class="box">
			<div class="tit"><span>POT 共识状态</span><p></p></div>
    </div>

    <a-spin :spinning="loading">
      <div class="space-y-4">
        <!-- Epoch 数字显示 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">Epoch</span>
          <span class="text-white font-semibold text-lg">
            {{ status?.epoch || 0 }}
          </span>
        </div>
        
        <!-- 挖矿难度 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">挖矿难度</span>
          <span class="text-white font-mono text-sm">
            {{ formatDifficulty(status?.difficulty || '') }}
          </span>
        </div>
        
        <!-- 区块高度 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">区块高度</span>
          <span class="text-white font-semibold">
            {{ formatNumber(status?.currentHeight || 0) }}
          </span>
        </div>
        
        <!-- 挖矿状态 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">挖矿状态</span>
          <a-badge
            :status="status?.workFlag ? 'processing' : 'default'"
            :text="status?.workFlag ? '工作中' : '空闲'"
          />
        </div>
        
        <!-- Nonce 值 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">Nonce</span>
          <span class="text-white font-mono text-sm">
            {{ status?.nonce || 0 }}
          </span>
        </div>
        
        <!-- 叔块数量 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">叔块数量</span>
          <a-tag color="orange">{{ status?.uncleCount || 0 }}</a-tag>
        </div>
        
        <!-- 平均挖矿时间 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">平均挖矿时间</span>
          <span class="text-white">{{ status?.avgMiningTime?.toFixed(2) || '0.00' }}s</span>
        </div>
        
        <!-- 挖矿成功率 -->
        <!-- <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-gray-400">成功率</span>
            <span class="text-white">{{ status?.miningSuccessRate?.toFixed(1) || '0.0' }}%</span>
          </div>
          <a-progress
            :percent="status?.miningSuccessRate || 0"
            :stroke-color="{ '0%': '#00d4ff', '100%': '#00ff88' }"
            :show-info="false"
          />
        </div> -->
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { ThunderboltOutlined } from '@ant-design/icons-vue'
import { usePotStore } from '@/stores/pot'
import { formatNumber, formatDifficulty } from '@/utils/format'

const potStore = usePotStore()
const { status, loading } = storeToRefs(potStore)
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}
</style>
