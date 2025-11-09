<template>
  <div class="blockchain-container">
    <div class="blockchain-header">
      <BlockOutlined />
      <span>区块链</span>
      <span class="block-count">{{ blocks.length }} 区块</span>
    </div>
    
    <div class="blockchain-scroll" ref="scrollContainer">
      <TransitionGroup name="block-slide" tag="div" class="blockchain-track">
        <div
          v-for="block in visibleBlocks"
          :key="block.hash"
          class="block-card"
          :class="{ 'block-new': block.isNew }"
          @click="selectBlock(block)"
        >
          <div class="block-header">
            <span class="block-height">#{{ block.height }}</span>
            <a-tag size="small" :color="getConsensusColor(block.consensusType)">
              {{ block.consensusType }}
            </a-tag>
          </div>
          
          <div class="block-body">
            <div class="block-info">
              <span class="label">哈希</span>
              <span class="value">{{ formatHash(block.hash) }}</span>
            </div>
            <div class="block-info">
              <span class="label">交易数</span>
              <span class="value">{{ block.txCount }}</span>
            </div>
            <div class="block-info">
              <span class="label">时间戳</span>
              <span class="value">{{ formatTimestamp(block.timestamp) }}</span>
            </div>
          </div>
          
          <div class="block-arrow">→</div>
        </div>
      </TransitionGroup>
    </div>
    
    <!-- 区块详情模态框 -->
    <a-modal
      v-model:open="showBlockDetail"
      title="区块详情"
      :footer="null"
      width="600px"
    >
      <a-descriptions v-if="selectedBlock" :column="1" bordered size="small">
        <a-descriptions-item label="区块高度">{{ selectedBlock.height }}</a-descriptions-item>
        <a-descriptions-item label="区块哈希">{{ selectedBlock.hash }}</a-descriptions-item>
        <a-descriptions-item label="父哈希">{{ selectedBlock.parentHash }}</a-descriptions-item>
        <a-descriptions-item label="共识类型">
          <a-tag :color="getConsensusColor(selectedBlock.consensusType)">
            {{ selectedBlock.consensusType }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="交易数量">{{ selectedBlock.txCount }}</a-descriptions-item>
        <a-descriptions-item label="时间戳">
          {{ new Date(selectedBlock.timestamp * 1000).toLocaleString() }}
        </a-descriptions-item>
        <a-descriptions-item label="区块大小">{{ selectedBlock.size }} bytes</a-descriptions-item>
        <a-descriptions-item label="生产者">{{ selectedBlock.producer }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { BlockOutlined } from '@ant-design/icons-vue'
import { formatHash, formatTimestamp } from '@/utils/format'
import { mockService } from '@/services/mock'

interface Block {
  height: number
  hash: string
  parentHash: string
  timestamp: number
  txCount: number
  consensusType: string
  size: number
  producer: string
  isNew?: boolean
}

const blocks = ref<Block[]>([])
const scrollContainer = ref<HTMLElement | null>(null)
const showBlockDetail = ref(false)
const selectedBlock = ref<Block | null>(null)
const maxVisibleBlocks = 15

// 只显示最近的区块
const visibleBlocks = computed(() => {
  return blocks.value.slice(-maxVisibleBlocks)
})

// 初始化区块数据
onMounted(() => {
  // 初始化一些区块
  for (let i = 0; i < 10; i++) {
    addBlock(false)
  }
  
  // 定期添加新区块 (模拟出块)
  const interval = setInterval(() => {
    addBlock(true)
    mockService.incrementBlock() // 同步 mock 服务的区块高度
  }, 3000) // 每3秒一个新区块
  
  // 组件卸载时清理定时器
  onUnmounted(() => {
    clearInterval(interval)
  })
})

// 添加新区块
function addBlock(isNew: boolean) {
  const lastBlock = blocks.value[blocks.value.length - 1]
  const height = lastBlock ? lastBlock.height + 1 : 1
  
  const consensusTypes = ['POT', 'POW', 'HotStuff', 'Whirly']
  const producers = ['Node-1', 'Node-2', 'Node-3', 'Node-4']
  
  const block: Block = {
    height,
    hash: generateHash(),
    parentHash: lastBlock ? lastBlock.hash : '0x0000000000000000',
    timestamp: Math.floor(Date.now() / 1000),
    txCount: Math.floor(Math.random() * 100) + 1,
    consensusType: consensusTypes[Math.floor(Math.random() * consensusTypes.length)],
    size: Math.floor(Math.random() * 50000) + 10000,
    producer: producers[Math.floor(Math.random() * producers.length)],
    isNew
  }
  
  blocks.value.push(block)
  
  // 标记为新区块后,1秒后移除标记
  if (isNew) {
    setTimeout(() => {
      block.isNew = false
    }, 1000)
    
    // 自动滚动到最新区块
    setTimeout(() => {
      if (scrollContainer.value) {
        scrollContainer.value.scrollLeft = scrollContainer.value.scrollWidth
      }
    }, 100)
  }
}

// 生成模拟哈希
function generateHash(): string {
  return '0x' + Array.from({ length: 64 }, () => 
    Math.floor(Math.random() * 16).toString(16)
  ).join('')
}

// 获取共识类型颜色
function getConsensusColor(type: string): string {
  const colors: Record<string, string> = {
    'POT': 'cyan',
    'POW': 'gold',
    'HotStuff': 'red',
    'Whirly': 'purple'
  }
  return colors[type] || 'blue'
}

// 选择区块查看详情
function selectBlock(block: Block) {
  selectedBlock.value = block
  showBlockDetail.value = true
}
</script>

<style scoped>
.blockchain-container {
  height: 100%;
  background: rgba(10, 14, 39, 0.6);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 8px;
  display: flex;
  flex-direction: column;
}

.blockchain-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
  padding: 0 4px;
}

.block-count {
  margin-left: auto;
  font-size: 12px;
  color: #888;
}

.blockchain-scroll {
  flex: 1;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-behavior: smooth;
}

.blockchain-scroll::-webkit-scrollbar {
  height: 4px;
}

.blockchain-scroll::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 2px;
}

.blockchain-track {
  display: flex;
  gap: 12px;
  padding: 4px;
  min-width: min-content;
}

.block-card {
  position: relative;
  min-width: 180px;
  background: linear-gradient(135deg, rgba(20, 30, 70, 0.8), rgba(30, 40, 90, 0.8));
  border: 1px solid rgba(0, 212, 255, 0.3);
  border-radius: 6px;
  padding: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  flex-shrink: 0;
}

.block-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 212, 255, 0.3);
  border-color: rgba(0, 212, 255, 0.6);
}

.block-new {
  animation: blockPulse 1s ease-out;
  border-color: rgba(0, 212, 255, 0.8);
  box-shadow: 0 0 20px rgba(0, 212, 255, 0.5);
}

@keyframes blockPulse {
  0% {
    transform: scale(0.8);
    opacity: 0;
  }
  50% {
    transform: scale(1.05);
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.block-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  padding-bottom: 6px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.block-height {
  font-size: 13px;
  font-weight: 600;
  color: #00d4ff;
}

.block-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 11px;
}

.block-info {
  display: flex;
  justify-content: space-between;
}

.block-info .label {
  color: #888;
}

.block-info .value {
  color: #fff;
  font-weight: 500;
}

.block-arrow {
  position: absolute;
  right: -14px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 20px;
  color: rgba(0, 212, 255, 0.4);
  font-weight: bold;
}

.block-card:last-child .block-arrow {
  display: none;
}

/* Transition 动画 */
.block-slide-enter-active {
  transition: all 0.5s ease;
}

.block-slide-enter-from {
  opacity: 0;
  transform: translateX(50px);
}
</style>
