<template>
  <div class="committee-info card h-full">
    <h3 class="text-sm font-semibold mb-3 flex items-center gap-2">
      <TeamOutlined />
      委员会信息
    </h3>
    
    <a-spin :spinning="loading">
      <div class="space-y-2 text-xs">
        <div class="flex items-center justify-between">
          <span class="text-gray-400">共识类型</span>
          <a-tag color="blue" size="small">{{ status?.consensusType || '-' }}</a-tag>
        </div>
        
        <div class="flex items-center justify-between">
          <span class="text-gray-400">委员会大小</span>
          <span class="text-white">{{ status?.committeeSize || 0 }}</span>
        </div>
        
        <div class="flex items-center justify-between">
          <span class="text-gray-400">工作高度</span>
          <span class="text-white">{{ status?.workHeight || 0 }}</span>
        </div>
        
        <div class="flex items-center justify-between">
          <span class="text-gray-400">当前角色</span>
          <a-tag :color="status?.role === 'leader' ? 'gold' : 'green'" size="small">
            {{ status?.role || 'observer' }}
          </a-tag>
        </div>
        
        <div class="flex items-center justify-between">
          <span class="text-gray-400">工作阶段</span>
          <a-tag color="processing" size="small">{{ status?.workStage || '-' }}</a-tag>
        </div>
        
        <div class="flex items-center justify-between">
          <span class="text-gray-400">消息队列</span>
          <span class="text-white">{{ status?.messageQueueLength || 0 }}</span>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { TeamOutlined } from '@ant-design/icons-vue'
import { useCommitteeStore } from '@/stores/committee'

const committeeStore = useCommitteeStore()
const { status, loading } = storeToRefs(committeeStore)
</script>

<style scoped>
.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
