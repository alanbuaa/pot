<template>
  <div class="storage-status card h-full">
    <h3 class="text-lg font-semibold mb-4 text-storage flex items-center gap-2">
      <HddOutlined />
      存储状态
    </h3>
    
    <a-spin :spinning="loading">
      <div class="space-y-4">
        <!-- 总存储 -->
        <div>
          <div class="text-sm text-gray-400 mb-1">总存储</div>
          <div class="text-xl font-bold text-white">
            {{ formatBytes(status?.totalSize || 0) }}
          </div>
        </div>
        
        <!-- 区块数量 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">区块数量</span>
          <a-tag color="purple">{{ status?.blockCount || 0 }}</a-tag>
        </div>
        
        <!-- VDF高度 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">VDF高度</span>
          <span class="text-white">{{ status?.vdfHeight || 0 }}</span>
        </div>
        
        <!-- 压缩率 -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-gray-400">压缩率</span>
            <span class="text-white">{{ ((status?.compressionRatio || 0) * 100).toFixed(1) }}%</span>
          </div>
          <a-progress
            :percent="(status?.compressionRatio || 0) * 100"
            :stroke-color="{ '0%': '#9370db', '100%': '#8a2be2' }"
            :show-info="false"
          />
        </div>
        
        <!-- Buckets -->
        <div>
          <div class="text-sm text-gray-400 mb-2">存储桶</div>
          <div class="space-y-2 text-xs">
            <div class="flex items-center justify-between">
              <span>Blocks</span>
              <span>{{ formatBytes(status?.buckets?.blocks?.size || 0) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>UTXO</span>
              <span>{{ formatBytes(status?.buckets?.utxo?.size || 0) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>Executed</span>
              <span>{{ formatBytes(status?.buckets?.executed?.size || 0) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>Client</span>
              <span>{{ formatBytes(status?.buckets?.client?.size || 0) }}</span>
            </div>
          </div>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { HddOutlined } from '@ant-design/icons-vue'
import { useStorageStore } from '@/stores/storage'
import { formatBytes } from '@/utils/format'

const storageStore = useStorageStore()
const { status, loading } = storeToRefs(storageStore)
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
