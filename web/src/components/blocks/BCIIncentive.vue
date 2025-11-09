<template>
  <div class="bci-incentive card h-full">
    <h3 class="text-lg font-semibold mb-4 text-incentive flex items-center gap-2">
      <DollarOutlined />
      BCI 激励
    </h3>
    
    <a-spin :spinning="loading">
      <div class="space-y-4">
        <!-- 总奖励 -->
        <div>
          <div class="text-sm text-gray-400 mb-1">总奖励金额</div>
          <div class="text-2xl font-bold text-incentive">
            {{ status?.totalReward?.toLocaleString() || 0 }}
          </div>
        </div>
        
        <!-- 锁定激励 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">锁定激励</span>
          <span class="text-white font-semibold">
            {{ status?.lockedReward?.toLocaleString() || 0 }}
          </span>
        </div>
        
        <!-- 待分配奖励 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">待分配奖励</span>
          <a-tag color="gold">{{ status?.pendingRewards || 0 }}</a-tag>
        </div>
        
        <!-- 累计利息 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">累计利息</span>
          <span class="text-white">{{ status?.totalInterest?.toLocaleString() || 0 }}</span>
        </div>
        
        <!-- UTXO数量 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">UTXO数量</span>
          <span class="text-white">{{ status?.utxoCount || 0 }}</span>
        </div>
        
        <!-- 奖励分配比例 -->
        <div>
          <div class="text-sm text-gray-400 mb-2">奖励分配</div>
          <div class="space-y-1 text-xs">
            <div class="flex items-center justify-between">
              <span>矿工</span>
              <span>{{ status?.rewardRatio?.miner || 0 }}%</span>
            </div>
            <div class="flex items-center justify-between">
              <span>国库</span>
              <span>{{ status?.rewardRatio?.exchequer || 0 }}%</span>
            </div>
            <div class="flex items-center justify-between">
              <span>委员会Leader</span>
              <span>{{ status?.rewardRatio?.committeeLeader || 0 }}%</span>
            </div>
            <div class="flex items-center justify-between">
              <span>委员会成员</span>
              <span>{{ status?.rewardRatio?.committeeMember || 0 }}%</span>
            </div>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { DollarOutlined } from '@ant-design/icons-vue'
import { useBCIStore } from '@/stores/bci'

const bciStore = useBCIStore()
const { status, loading } = storeToRefs(bciStore)
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}

.space-y-1 > * + * {
  margin-top: 0.25rem;
}
</style>
