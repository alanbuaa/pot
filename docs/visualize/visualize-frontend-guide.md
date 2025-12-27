# 可视化大屏前端开发指南

本文档提供前端开发的详细指导，确保可以直接根据方案开始开发。

## 一、项目初始化

### 1.1 创建Vue 3项目
```bash
# 使用Vite创建项目
npm create vite@latest pot-visualization -- --template vue-ts

cd pot-visualization
npm install
```

### 1.2 安装依赖
```bash
# UI组件库
npm install ant-design-vue@4.1.2
npm install @ant-design/icons-vue@^6.1.0

# 图表库
npm install echarts vue-echarts

# 网络图
npm install @antv/g6

# 工具库
npm install axios
npm install @vueuse/core
npm install dayjs

# 状态管理
npm install pinia@^2.0.23

# 样式
npm install tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

**package.json 依赖版本：**
```json
{
  "dependencies": {
    "vue": "^3.3.4",
    "ant-design-vue": "4.1.2",
    "@ant-design/icons-vue": "^6.1.0",
    "pinia": "^2.0.23",
    "echarts": "^5.4.0",
    "vue-echarts": "^6.6.0",
    "@antv/g6": "^4.8.0",
    "axios": "^1.4.0",
    "@vueuse/core": "^10.0.0",
    "dayjs": "^1.11.0",
    "tailwindcss": "^3.3.0"
  },
  "devDependencies": {
    "vite": "^2.9.16",
    "vue-tsc": "^0.35.0",
    "typescript": "^4.5.4",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0"
  }
}
```

### 1.3 配置TailwindCSS
```js
// tailwind.config.js
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'bg-primary': '#0a0e27',
        'bg-secondary': '#141d3a',
        'border': '#1e3a5f',
        'pot': '#00d4ff',
        'committee': '#00ff88',
        'transaction': '#ffa500',
        'incentive': '#ffd700',
        'storage': '#9370db',
        'vdf': '#ff69b4',
      }
    },
  },
  plugins: [],
}
```

---

## 二、项目结构

```
pot-visualization/
├── public/
├── src/
│   ├── assets/              # 静态资源
│   ├── components/          # 组件
│   │   ├── common/          # 通用组件
│   │   │   ├── StatusCard.vue
│   │   │   ├── ProgressCircle.vue
│   │   │   └── NumberFlip.vue
│   │   ├── blocks/          # 区块组件
│   │   │   ├── PotConsensus.vue       # 左上：POT共识状态
│   │   │   ├── VDFMonitor.vue         # 左中：VDF计算监控
│   │   │   ├── PerformanceMetrics.vue # 左下：性能指标
│   │   │   ├── NetworkTopology.vue    # 中心：网络拓扑（双层）
│   │   │   ├── MempoolMonitor.vue     # 右上：交易池监控
│   │   │   ├── BCIIncentive.vue       # 右中：BCI激励
│   │   │   ├── StorageStatus.vue      # 右下：存储状态
│   │   │   ├── CommitteeDetails.vue   # 底部左：委员会详情
│   │   │   └── SystemLogs.vue         # 底部右：系统日志
│   │   └── layout/          # 布局组件
│   │       └── TopBar.vue
│   ├── composables/         # 组合式函数
│   │   ├── useWebSocket.ts
│   │   ├── useRealtime.ts
│   │   ├── useMetrics.ts
│   │   └── useG6Graph.ts    # G6图表hooks
│   ├── stores/              # Pinia状态管理
│   │   ├── system.ts
│   │   ├── pot.ts
│   │   ├── committee.ts
│   │   ├── bci.ts
│   │   ├── mempool.ts
│   │   ├── network.ts
│   │   └── storage.ts
│   ├── services/            # API服务
│   │   ├── api.ts
│   │   └── websocket.ts
│   ├── types/               # TypeScript类型
│   │   └── api.ts
│   ├── utils/               # 工具函数
│   │   ├── format.ts
│   │   ├── chart.ts
│   │   └── g6-config.ts     # G6配置
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```
│   │   ├── bci.ts
│   │   ├── mempool.ts
│   │   ├── network.ts
│   │   └── storage.ts
│   ├── services/            # API服务
│   │   ├── api.ts
│   │   └── websocket.ts
│   ├── types/               # TypeScript类型
│   │   └── api.ts
│   ├── utils/               # 工具函数
│   │   ├── format.ts
│   │   └── chart.ts
│   ├── App.vue              # 根组件
│   └── main.ts              # 入口文件
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

---

### 2.1 主入口文件配置

#### `src/main.ts`
```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import App from './App.vue'
import './assets/main.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(Antd)
app.mount('#app')
```

#### `vite.config.ts`
```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:9090',
        changeOrigin: true
      }
    }
  }
})
```

---

## 三、核心代码实现

### 3.1 API服务层

#### `src/services/api.ts`
```typescript
import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  }
})

// 请求拦截器
apiClient.interceptors.request.use(
  config => {
    // 可以添加token等
    return config
  },
  error => Promise.reject(error)
)

// 响应拦截器
apiClient.interceptors.response.use(
  response => response.data,
  error => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export const api = {
  // 系统API
  getSystemOverview: () => apiClient.get('/system/overview'),
  
  // POT API
  getPotStatus: () => apiClient.get('/pot/status'),
  getVDFStatus: () => apiClient.get('/pot/vdf'),
  
  // 委员会API
  getCommitteeStatus: () => apiClient.get('/committee/status'),
  
  // BCI API
  getBCIStatus: () => apiClient.get('/bci/status'),
  getBalance: (address: string) => apiClient.get(`/bci/balance/${address}`),
  
  // 交易池API
  getMempoolStatus: () => apiClient.get('/mempool/status'),
  
  // 网络API
  getNetworkTopology: () => apiClient.get('/network/topology'),
  
  // 存储API
  getStorageStatus: () => apiClient.get('/storage/status'),
  
  // 历史数据API
  getMetricsHistory: (metric: string, timeRange: string, interval: string) => 
    apiClient.get(`/metrics/history`, { params: { metric, timeRange, interval } }),
}
```

---

#### `src/services/websocket.ts`
```typescript
import { ref } from 'vue'

export class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectInterval = 3000
  
  public connected = ref(false)
  public error = ref<string | null>(null)
  
  constructor(private url: string) {}
  
  connect() {
    try {
      this.ws = new WebSocket(this.url)
      
      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.connected.value = true
        this.reconnectAttempts = 0
        this.error.value = null
      }
      
      this.ws.onmessage = (event) => {
        const data = JSON.parse(event.data)
        this.handleMessage(data)
      }
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        this.error.value = 'WebSocket连接错误'
      }
      
      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.connected.value = false
        this.reconnect()
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      this.error.value = 'WebSocket创建失败'
    }
  }
  
  private reconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      console.log(`尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
      setTimeout(() => this.connect(), this.reconnectInterval)
    } else {
      this.error.value = 'WebSocket重连失败，已达到最大尝试次数'
    }
  }
  
  subscribe(types: string[]) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        action: 'subscribe',
        types
      }))
    }
  }
  
  private handleMessage(data: any) {
    // 根据type分发到不同的store
    // 在stores中监听
  }
  
  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }
}
```

---

### 3.2 Pinia Store

#### `src/stores/pot.ts`
```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import type { POTStatus, VDFStatus } from '@/types/api'

export const usePotStore = defineStore('pot', () => {
  const status = ref<POTStatus | null>(null)
  const vdf = ref<VDFStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  async function fetchStatus() {
    try {
      loading.value = true
      status.value = await api.getPotStatus()
    } catch (err) {
      error.value = '获取POT状态失败'
      console.error(err)
    } finally {
      loading.value = false
    }
  }
  
  async function fetchVDF() {
    try {
      vdf.value = await api.getVDFStatus()
    } catch (err) {
      error.value = '获取VDF状态失败'
      console.error(err)
    }
  }
  
  function updateFromWebSocket(data: any) {
    if (data.type === 'pot') {
      status.value = { ...status.value, ...data.data }
    } else if (data.type === 'vdf') {
      vdf.value = { ...vdf.value, ...data.data }
    }
  }
  
  return {
    status,
    vdf,
    loading,
    error,
    fetchStatus,
    fetchVDF,
    updateFromWebSocket,
  }
})
```

---

### 3.3 组合式函数

#### `src/composables/useRealtime.ts`
```typescript
import { ref, onMounted, onUnmounted } from 'vue'

export function useRealtime<T>(
  fetchFn: () => Promise<T>,
  interval: number = 5000
) {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let timer: number | null = null
  
  async function fetch() {
    try {
      loading.value = true
      error.value = null
      data.value = await fetchFn()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '请求失败'
      console.error(err)
    } finally {
      loading.value = false
    }
  }
  
  function startPolling() {
    fetch() // 立即执行一次
    timer = window.setInterval(fetch, interval)
  }
  
  function stopPolling() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }
  
  onMounted(startPolling)
  onUnmounted(stopPolling)
  
  return {
    data,
    loading,
    error,
    fetch,
    startPolling,
    stopPolling,
  }
}
```

---

### 3.4 通用组件

#### `src/components/common/StatusCard.vue`
```vue
<template>
  <div class="status-card bg-bg-secondary border border-border rounded-lg p-4">
    <div class="flex items-center justify-between mb-2">
      <span class="text-gray-400 text-sm">{{ label }}</span>
      <div 
        class="status-indicator w-2 h-2 rounded-full"
        :class="statusClass"
      ></div>
    </div>
    <div class="text-2xl font-bold" :class="valueColor">{{ value }}</div>
    <div v-if="subtitle" class="text-gray-500 text-xs mt-1">{{ subtitle }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  label: string
  value: string | number
  status?: 'normal' | 'warning' | 'error' | 'idle'
  subtitle?: string
  color?: string
}

const props = withDefaults(defineProps<Props>(), {
  status: 'normal',
  color: 'text-white'
})

const statusClass = computed(() => {
  const classes = {
    normal: 'bg-green-500',
    warning: 'bg-yellow-500',
    error: 'bg-red-500',
    idle: 'bg-gray-500',
  }
  return classes[props.status]
})

const valueColor = computed(() => props.color)
</script>

<style scoped>
.status-card {
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
}

.status-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 212, 255, 0.2);
}

.status-indicator {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
```

---

#### `src/components/common/ProgressCircle.vue`
```vue
<template>
  <div class="progress-circle" :style="{ width: size + 'px', height: size + 'px' }">
    <svg :width="size" :height="size">
      <!-- 背景圆 -->
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        :stroke="bgColor"
        :stroke-width="strokeWidth"
      />
      <!-- 进度圆 -->
      <circle
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        fill="none"
        :stroke="color"
        :stroke-width="strokeWidth"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
        stroke-linecap="round"
        transform-origin="center"
        :transform="`rotate(-90 ${size / 2} ${size / 2})`"
        class="progress-stroke"
      />
    </svg>
    <div class="progress-text">
      <div class="text-2xl font-bold">{{ Math.round(progress) }}%</div>
      <div v-if="label" class="text-xs text-gray-400 mt-1">{{ label }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  progress: number  // 0-100
  size?: number
  strokeWidth?: number
  color?: string
  bgColor?: string
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 120,
  strokeWidth: 8,
  color: '#00d4ff',
  bgColor: '#1e3a5f',
})

const radius = computed(() => (props.size - props.strokeWidth) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const dashOffset = computed(() => 
  circumference.value * (1 - Math.min(100, Math.max(0, props.progress)) / 100)
)
</script>

<style scoped>
.progress-circle {
  position: relative;
  display: inline-block;
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
}

.progress-stroke {
  transition: stroke-dashoffset 0.5s ease;
}
</style>
```

---

### 3.5 区块组件示例

#### `src/components/blocks/PotConsensus.vue`
```vue
<template>
  <div class="pot-consensus bg-bg-secondary border border-border rounded-lg p-4">
    <h3 class="text-lg font-bold mb-4 text-pot">POT共识状态</h3>
    
    <!-- Epoch进度 -->
    <div class="mb-4 flex justify-center">
      <ProgressCircle 
        :progress="epochProgress" 
        :size="100"
        color="#00d4ff"
        label="Epoch"
      />
    </div>
    
    <!-- 关键指标 -->
    <div class="grid grid-cols-2 gap-3">
      <StatusCard
        label="区块高度"
        :value="status?.currentHeight || 0"
        status="normal"
      />
      <StatusCard
        label="挖矿难度"
        :value="formatDifficulty(status?.difficulty)"
        status="normal"
      />
      <StatusCard
        label="Nonce"
        :value="status?.nonce || 0"
        status="normal"
      />
      <StatusCard
        label="叔块数量"
        :value="status?.uncleCount || 0"
        status="normal"
      />
    </div>
    
    <!-- 挖矿状态 -->
    <div class="mt-4 p-3 rounded bg-bg-primary">
      <div class="flex items-center justify-between">
        <span class="text-gray-400">挖矿状态</span>
        <div class="flex items-center gap-2">
          <div 
            class="w-3 h-3 rounded-full"
            :class="status?.workFlag ? 'bg-green-500 animate-pulse' : 'bg-gray-500'"
          ></div>
          <span class="text-sm">{{ status?.workFlag ? '工作中' : '空闲' }}</span>
        </div>
      </div>
      <div class="mt-2 text-xs text-gray-500">
        平均挖矿耗时: {{ status?.avgMiningTime?.toFixed(2) || 0 }}s
      </div>
      <div class="text-xs text-gray-500">
        挖矿成功率: {{ status?.miningSuccessRate?.toFixed(1) || 0 }}%
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePotStore } from '@/stores/pot'
import StatusCard from '@/components/common/StatusCard.vue'
import ProgressCircle from '@/components/common/ProgressCircle.vue'

const potStore = usePotStore()
const status = computed(() => potStore.status)

const epochProgress = computed(() => {
  // 这里可以根据实际情况计算epoch进度
  return (status.value?.epoch || 0) % 100
})

function formatDifficulty(difficulty?: string) {
  if (!difficulty) return '0'
  const num = BigInt(difficulty)
  return num.toString().slice(0, 8) + '...'
}
</script>
```

---

#### `src/components/blocks/NetworkTopology.vue` - **核心组件**
```vue
<template>
  <div class="network-topology bg-bg-secondary border border-border rounded-lg p-4 h-full">
    <h3 class="text-lg font-bold mb-4 text-center text-committee">
      网络拓扑图
    </h3>
    
    <!-- G6容器 -->
    <div ref="container" class="topology-container w-full h-[calc(100%-3rem)]"></div>
    
    <!-- 工具栏 -->
    <div class="absolute top-16 right-8 flex flex-col gap-2 bg-bg-primary p-2 rounded">
      <a-tooltip title="放大">
        <a-button size="small" @click="zoomIn">
          <ZoomInOutlined />
        </a-button>
      </a-tooltip>
      <a-tooltip title="缩小">
        <a-button size="small" @click="zoomOut">
          <ZoomOutOutlined />
        </a-button>
      </a-tooltip>
      <a-tooltip title="适应画布">
        <a-button size="small" @click="fitView">
          <FullscreenOutlined />
        </a-button>
      </a-tooltip>
    </div>
    
    <!-- 节点详情面板 -->
    <a-drawer
      v-model:open="detailVisible"
      title="节点详情"
      placement="right"
      :width="400"
    >
      <a-descriptions :column="1" bordered size="small" v-if="selectedNode">
        <a-descriptions-item label="节点ID">
          {{ selectedNode.id }}
        </a-descriptions-item>
        <a-descriptions-item label="类型">
          <a-tag :color="getNodeTypeColor(selectedNode.type)">
            {{ getNodeTypeName(selectedNode.type) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="地址">
          {{ selectedNode.address }}
        </a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-badge :status="getStatusType(selectedNode.status)" :text="selectedNode.status" />
        </a-descriptions-item>
        <a-descriptions-item label="连接数">
          {{ selectedNode.connections || 0 }}
        </a-descriptions-item>
      </a-descriptions>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { ZoomInOutlined, ZoomOutOutlined, FullscreenOutlined } from '@ant-design/icons-vue'
import G6, { Graph } from '@antv/g6'
import { useNetworkStore } from '@/stores/network'
import { storeToRefs } from 'pinia'

const container = ref<HTMLDivElement>()
const networkStore = useNetworkStore()
const { topology } = storeToRefs(networkStore)

let graph: Graph | null = null
const detailVisible = ref(false)
const selectedNode = ref<any>(null)

// 注册自定义节点 - Leader节点
G6.registerNode('leader-node', {
  draw(cfg: any, group: any) {
    // 外圈光晕
    const halo = group.addShape('circle', {
      attrs: {
        x: 0,
        y: 0,
        r: 35,
        fill: 'l(0) 0:#ffd700 1:#ff8800',
        opacity: 0.3
      }
    })
    
    // 主圆形
    const circle = group.addShape('circle', {
      attrs: {
        x: 0,
        y: 0,
        r: 25,
        fill: '#ffd700',
        stroke: '#ff8800',
        lineWidth: 3,
        shadowColor: '#ffd700',
        shadowBlur: 15
      },
      name: 'main-circle'
    })
    
    // 皇冠图标
    group.addShape('text', {
      attrs: {
        x: 0,
        y: 5,
        text: '👑',
        fontSize: 20,
        textAlign: 'center',
        textBaseline: 'middle'
      }
    })
    
    // 标签
    if (cfg.label) {
      group.addShape('text', {
        attrs: {
          x: 0,
          y: 40,
          text: cfg.label,
          fontSize: 12,
          fill: '#fff',
          textAlign: 'center'
        }
      })
    }
    
    // 添加脉冲动画
    halo.animate(
      {
        r: 40,
        opacity: 0
      },
      {
        duration: 2000,
        repeat: true,
        easing: 'easeCubic'
      }
    )
    
    return circle
  }
})

// 注册委员会成员节点
G6.registerNode('committee-node', {
  draw(cfg: any, group: any) {
    const circle = group.addShape('circle', {
      attrs: {
        x: 0,
        y: 0,
        r: 20,
        fill: '#00ff88',
        stroke: '#00cc66',
        lineWidth: 2
      },
      name: 'main-circle'
    })
    
    if (cfg.label) {
      group.addShape('text', {
        attrs: {
          x: 0,
          y: 32,
          text: cfg.label,
          fontSize: 11,
          fill: '#fff',
          textAlign: 'center'
        }
      })
    }
    
    return circle
  }
})

// 注册POT挖矿节点
G6.registerNode('pot-mining-node', {
  draw(cfg: any, group: any) {
    const circle = group.addShape('circle', {
      attrs: {
        x: 0,
        y: 0,
        r: 18,
        fill: '#00d4ff',
        stroke: '#0099cc',
        lineWidth: 2
      },
      name: 'main-circle'
    })
    
    // 闪电图标
    group.addShape('text', {
      attrs: {
        x: 0,
        y: 3,
        text: '⚡',
        fontSize: 14,
        textAlign: 'center',
        textBaseline: 'middle'
      }
    })
    
    return circle
  }
})

// 注册POT普通节点
G6.registerNode('pot-normal-node', {
  draw(cfg: any, group: any) {
    const circle = group.addShape('circle', {
      attrs: {
        x: 0,
        y: 0,
        r: 15,
        fill: '#4488ff',
        stroke: '#3366cc',
        lineWidth: 1
      },
      name: 'main-circle'
    })
    
    return circle
  }
})

// 自定义双层布局
G6.registerLayout('dual-layer', {
  getDefaultCfg() {
    return {
      center: [0, 0],
      committeeRadius: 200,
      potCenter: [0, 300],
      nodeSpacing: 80
    }
  },
  
  execute() {
    const self = this as any
    const nodes = self.nodes
    const width = self.width || 800
    const height = self.height || 600
    
    const committeeNodes = nodes.filter((n: any) => n.nodeType === 'committee')
    const potNodes = nodes.filter((n: any) => n.nodeType === 'pot')
    
    // 上层：委员会圆形布局
    const committeeRadius = 180
    const committeeCenter = [width / 2, height * 0.25]
    committeeNodes.forEach((node: any, i: number) => {
      const angle = (2 * Math.PI * i) / committeeNodes.length - Math.PI / 2
      node.x = committeeCenter[0] + committeeRadius * Math.cos(angle)
      node.y = committeeCenter[1] + committeeRadius * Math.sin(angle)
    })
    
    // 下层：POT节点力导向布局
    const potCenter = [width / 2, height * 0.7]
    const potRadius = 250
    potNodes.forEach((node: any, i: number) => {
      // 初始化位置（圆形分布）
      const angle = (2 * Math.PI * i) / potNodes.length
      node.x = potCenter[0] + (potRadius * Math.random() * 0.8) * Math.cos(angle)
      node.y = potCenter[1] + (potRadius * Math.random() * 0.8) * Math.sin(angle)
    })
  }
})

// 初始化图表
function initGraph() {
  if (!container.value) return
  
  const width = container.value.offsetWidth
  const height = container.value.offsetHeight
  
  graph = new G6.Graph({
    container: container.value,
    width,
    height,
    layout: {
      type: 'dual-layer'
    },
    modes: {
      default: [
        'drag-canvas',
        'zoom-canvas',
        'drag-node',
        {
          type: 'tooltip',
          formatText(model: any) {
            return `<div>
              <p>ID: ${model.id}</p>
              <p>类型: ${model.nodeType}</p>
              <p>状态: ${model.status}</p>
            </div>`
          }
        }
      ]
    },
    defaultNode: {
      size: 30,
      style: {
        fill: '#4488ff',
        stroke: '#3366cc',
        lineWidth: 2
      },
      labelCfg: {
        position: 'bottom',
        style: {
          fill: '#fff',
          fontSize: 12
        }
      }
    },
    defaultEdge: {
      type: 'line',
      style: {
        stroke: '#888',
        lineWidth: 1,
        opacity: 0.6,
        endArrow: false
      }
    },
    nodeStateStyles: {
      active: {
        stroke: '#00ff88',
        lineWidth: 3,
        shadowColor: '#00ff88',
        shadowBlur: 20
      },
      inactive: {
        opacity: 0.3
      }
    },
    edgeStateStyles: {
      active: {
        stroke: '#00d4ff',
        lineWidth: 2,
        opacity: 1
      }
    }
  })
  
  // 节点点击事件
  graph.on('node:click', (evt: any) => {
    const { item } = evt
    const model = item.getModel()
    selectedNode.value = model
    detailVisible.value = true
    
    // 高亮选中节点及其边
    graph?.setItemState(item, 'active', true)
    const edges = item.getEdges()
    edges.forEach((edge: any) => {
      graph?.setItemState(edge, 'active', true)
    })
  })
  
  // 画布点击事件（取消选中）
  graph.on('canvas:click', () => {
    graph?.getNodes().forEach((node) => {
      graph?.clearItemStates(node)
    })
    graph?.getEdges().forEach((edge) => {
      graph?.clearItemStates(edge)
    })
  })
  
  // 渲染数据
  updateGraph()
}

// 更新图表数据
function updateGraph() {
  if (!graph || !topology.value) return
  
  const { nodes, edges } = topology.value
  
  // 转换节点数据
  const graphNodes = nodes.map((node: any) => ({
    id: node.id,
    label: node.label || node.id.slice(0, 8),
    nodeType: node.type, // 'committee' or 'pot'
    type: getNodeShapeType(node),
    address: node.address,
    status: node.status,
    connections: node.connections,
    isLeader: node.isLeader
  }))
  
  // 转换边数据
  const graphEdges = edges.map((edge: any) => ({
    source: edge.source,
    target: edge.target,
    style: {
      stroke: getEdgeColor(edge.latency),
      lineWidth: edge.bandwidth ? Math.min(edge.bandwidth / 1000, 5) : 1
    },
    label: edge.latency ? `${edge.latency}ms` : ''
  }))
  
  graph.data({
    nodes: graphNodes,
    edges: graphEdges
  })
  
  graph.render()
}

// 获取节点形状类型
function getNodeShapeType(node: any) {
  if (node.type === 'committee') {
    return node.isLeader ? 'leader-node' : 'committee-node'
  } else {
    return node.isMining ? 'pot-mining-node' : 'pot-normal-node'
  }
}

// 根据延迟获取边颜色
function getEdgeColor(latency?: number) {
  if (!latency) return '#888888'
  if (latency < 50) return '#00ff88'
  if (latency < 200) return '#ffa500'
  return '#ff4444'
}

// 工具栏方法
function zoomIn() {
  graph?.zoom(1.2)
}

function zoomOut() {
  graph?.zoom(0.8)
}

function fitView() {
  graph?.fitView(20)
}

// 辅助方法
function getNodeTypeColor(type: string) {
  const colors: Record<string, string> = {
    'committee': 'green',
    'pot': 'blue'
  }
  return colors[type] || 'default'
}

function getNodeTypeName(type: string) {
  const names: Record<string, string> = {
    'committee': '委员会节点',
    'pot': 'POT节点'
  }
  return names[type] || type
}

function getStatusType(status: string) {
  const map: Record<string, any> = {
    'active': 'processing',
    'idle': 'default',
    'error': 'error'
  }
  return map[status] || 'default'
}

// 监听拓扑数据变化
watch(() => topology.value, () => {
  updateGraph()
}, { deep: true })

onMounted(() => {
  initGraph()
  networkStore.fetchTopology()
  
  // 定期更新
  const timer = setInterval(() => {
    networkStore.fetchTopology()
  }, 5000)
  
  onUnmounted(() => {
    clearInterval(timer)
    graph?.destroy()
  })
})
</script>

<style scoped>
.network-topology {
  position: relative;
}

.topology-container {
  background: linear-gradient(135deg, #0a0e27 0%, #141d3a 100%);
  border-radius: 8px;
}
</style>
```

---

#### `src/components/blocks/VDFMonitor.vue`
```vue
<template>
  <div class="vdf-monitor bg-bg-secondary border border-border rounded-lg p-4">
    <h3 class="text-lg font-bold mb-4 text-vdf">VDF计算监控</h3>
    
    <!-- VDF0进度 -->
    <div class="mb-4">
      <div class="flex justify-between items-center mb-2">
        <span class="text-sm text-gray-400">VDF0计算</span>
        <span class="text-xs text-gray-500">
          迭代: {{ vdf?.vdf0?.iterations?.toLocaleString() || 0 }}
        </span>
      </div>
      <div class="w-full bg-bg-primary rounded-full h-3">
        <div 
          class="h-3 rounded-full bg-gradient-to-r from-vdf to-pink-400 transition-all duration-500"
          :style="{ width: (vdf?.vdf0?.progress || 0) + '%' }"
        ></div>
      </div>
      <div class="text-right text-xs text-gray-500 mt-1">
        {{ vdf?.vdf0?.progress?.toFixed(1) || 0 }}%
      </div>
    </div>
    
    <!-- VDFhalf进度 -->
    <div class="mb-4">
      <div class="flex justify-between items-center mb-2">
        <span class="text-sm text-gray-400">VDFhalf计算</span>
        <span class="text-xs text-gray-500">
          缓冲: {{ vdf?.vdfHalf?.channelBuffer || 0 }}
        </span>
      </div>
      <div class="w-full bg-bg-primary rounded-full h-3">
        <div 
          class="h-3 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 transition-all duration-500"
          :style="{ width: (vdf?.vdfHalf?.progress || 0) + '%' }"
        ></div>
      </div>
    </div>
    
    <!-- VDF1并行线程 -->
    <div class="mb-4">
      <div class="text-sm text-gray-400 mb-2">
        VDF1并行线程 ({{ vdf?.cpuCounter || 0 }}个)
      </div>
      <div class="grid grid-cols-2 gap-2">
        <div 
          v-for="worker in vdf?.vdf1 || []"
          :key="worker.workerId"
          class="p-2 bg-bg-primary rounded"
        >
          <div class="text-xs text-gray-500 mb-1">Worker #{{ worker.workerId }}</div>
          <div class="w-full bg-gray-700 rounded-full h-2">
            <div 
              class="h-2 rounded-full bg-vdf transition-all"
              :style="{ width: worker.progress + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- VDF统计 -->
    <div class="grid grid-cols-2 gap-2">
      <div class="p-2 bg-bg-primary rounded">
        <div class="text-xs text-gray-500">平均耗时</div>
        <div class="text-lg font-bold text-vdf">
          {{ vdf?.avgComputeTime?.toFixed(1) || 0 }}ms
        </div>
      </div>
      <div class="p-2 bg-bg-primary rounded">
        <div class="text-xs text-gray-500">验证失败</div>
        <div class="text-lg font-bold text-red-500">
          {{ vdf?.vdfChecker?.verifyFailCount || 0 }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePotStore } from '@/stores/pot'

const potStore = usePotStore()
const vdf = computed(() => potStore.vdf)
</script>
```

---

### 3.6 主应用组件

#### `src/App.vue`
```vue
<template>
  <div class="app bg-bg-primary min-h-screen text-white">
    <!-- 顶部状态栏 -->
    <TopBar class="h-16" />
    
    <!-- 主体布局 -->
    <div class="main-layout flex h-[calc(100vh-8rem)] p-4 gap-4">
      <!-- 左侧面板 (20%) -->
      <div class="left-panel w-1/5 flex flex-col gap-4">
        <PotConsensus class="flex-1" />
        <VDFMonitor class="flex-1" />
        <PerformanceMetrics class="flex-1" />
      </div>
      
      <!-- 中心网络拓扑 (60%) -->
      <div class="center-panel w-3/5">
        <NetworkTopology class="h-full" />
      </div>
      
      <!-- 右侧面板 (20%) -->
      <div class="right-panel w-1/5 flex flex-col gap-4">
        <MempoolMonitor class="flex-1" />
        <BCIIncentive class="flex-1" />
        <StorageStatus class="flex-1" />
      </div>
    </div>
    
    <!-- 底部面板 -->
    <div class="bottom-panel h-48 flex gap-4 p-4">
      <CommitteeDetails class="w-1/2" />
      <SystemLogs class="w-1/2" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import TopBar from '@/components/layout/TopBar.vue'
import PotConsensus from '@/components/blocks/PotConsensus.vue'
import VDFMonitor from '@/components/blocks/VDFMonitor.vue'
import PerformanceMetrics from '@/components/blocks/PerformanceMetrics.vue'
import NetworkTopology from '@/components/blocks/NetworkTopology.vue'
import MempoolMonitor from '@/components/blocks/MempoolMonitor.vue'
import BCIIncentive from '@/components/blocks/BCIIncentive.vue'
import StorageStatus from '@/components/blocks/StorageStatus.vue'
import CommitteeDetails from '@/components/blocks/CommitteeDetails.vue'
import SystemLogs from '@/components/blocks/SystemLogs.vue'

import { usePotStore } from '@/stores/pot'
import { useCommitteeStore } from '@/stores/committee'
import { useNetworkStore } from '@/stores/network'
import { WebSocketService } from '@/services/websocket'

const potStore = usePotStore()
const committeeStore = useCommitteeStore()
const networkStore = useNetworkStore()

let wsService: WebSocketService | null = null

onMounted(() => {
  // 初始数据加载
  potStore.fetchStatus()
  potStore.fetchVDF()
  committeeStore.fetchStatus()
  networkStore.fetchTopology()
  
  // 建立WebSocket连接
  const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/ws'
  wsService = new WebSocketService(wsUrl)
  wsService.connect()
  
  // 订阅实时数据更新
  wsService.subscribe(['pot', 'vdf', 'committee', 'network', 'mempool'])
})
  wsService = new WebSocketService(wsUrl)
  wsService.connect()
  wsService.subscribe(['pot', 'vdf', 'committee', 'mempool', 'network'])
})

onUnmounted(() => {
  wsService?.disconnect()
})
</script>

<style>
@import './assets/main.css';

.app {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
}
</style>
```

---

## 四、开发流程

### 第一阶段：基础搭建（1-2天）
1. ✅ 项目初始化
2. ✅ 依赖安装
3. ✅ 配置TailwindCSS
4. ✅ 创建基础目录结构
5. ✅ 实现API服务层
6. ✅ 实现WebSocket服务

### 第二阶段：状态管理（1天）
1. ✅ 创建所有Pinia stores
2. ✅ 实现数据获取逻辑
3. ✅ 测试API调用

### 第三阶段：通用组件（2天）
1. ✅ StatusCard
2. ✅ ProgressCircle
3. ✅ NumberFlip
4. ✅ 其他通用组件

### 第四阶段：区块组件（4-5天）
按优先级开发：
1. ✅ PotConsensus（区块1）
2. ✅ VDFMonitor（区块5）
3. ✅ MempoolMonitor（区块3）
4. ✅ CommitteeConsensus（区块2）
5. ✅ BCIIncentive（区块6）
6. ✅ NetworkTopology（区块4）
7. ✅ StorageStatus（区块7）
8. ✅ PerformanceMetrics（区块8）

### 第五阶段：布局组件（1-2天）
1. ✅ TopBar
2. ✅ BottomTimeline

### 第六阶段：集成测试（2天）
1. ✅ 组件集成
2. ✅ 实时数据测试
3. ✅ 性能优化
4. ✅ 响应式调整

---

## 五、关键技术点

### 5.1 ECharts使用
```typescript
// 基础配置
const option = {
  backgroundColor: 'transparent',
  textStyle: {
    color: '#fff',
    fontFamily: 'Inter'
  },
  tooltip: {
    backgroundColor: 'rgba(20, 29, 58, 0.9)',
    borderColor: '#1e3a5f',
    textStyle: { color: '#fff' }
  }
}
```

### 5.2 数字动画
```typescript
import { CountUp } from 'countup.js'

const animateNumber = (el: HTMLElement, endVal: number) => {
  const countUp = new CountUp(el, endVal, {
    duration: 1,
    separator: ',',
  })
  countUp.start()
}
```

### 5.3 响应式适配
```vue
<script setup lang="ts">
import { useWindowSize } from '@vueuse/core'

const { width } = useWindowSize()
const isLargeScreen = computed(() => width.value > 1920)
const isMediumScreen = computed(() => width.value > 1280 && width.value <= 1920)
</script>
```

---

## 六、环境配置

### `.env.development`
```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/api/ws
```

### `.env.production`
```env
VITE_API_BASE_URL=https://api.example.com/api
VITE_WS_URL=wss://api.example.com/api/ws
```

---

## 七、测试和部署

### 开发环境运行
```bash
npm run dev
```

### 生产构建
```bash
npm run build
```

### 预览构建结果
```bash
npm run preview
```

---

## 八、开发注意事项

1. **性能优化**
   - 使用虚拟滚动处理大量数据
   - 图表按需加载
   - 防抖节流处理频繁更新

2. **错误处理**
   - 所有API调用都要try-catch
   - WebSocket断线重连
   - 降级方案（WebSocket→轮询）

3. **类型安全**
   - 所有API响应定义TypeScript类型
   - Props使用接口定义
   - 避免使用any

4. **代码规范**
   - 使用ESLint + Prettier
   - 组件命名采用PascalCase
   - 函数命名采用camelCase

---

这份开发指南提供了完整的前端实现方案，可以直接按照步骤开始开发！
