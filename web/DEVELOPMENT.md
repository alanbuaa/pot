# POT 可视化前端开发指南

## 项目概述

本项目是一个前后端分离的可视化大屏系统，用于实时监控和展示 POT 共识系统的运行状态。

## 目录结构说明

```
web/
├── src/
│   ├── assets/              # 静态资源
│   │   └── main.css         # 全局样式
│   ├── components/          # 组件目录
│   │   ├── blocks/          # 功能区块组件
│   │   │   ├── PotConsensus.vue       # POT共识状态
│   │   │   ├── VDFMonitor.vue         # VDF计算监控
│   │   │   ├── PerformanceMetrics.vue # 性能指标
│   │   │   ├── NetworkTopology.vue    # 网络拓扑
│   │   │   ├── MempoolMonitor.vue     # 交易池监控
│   │   │   ├── BCIIncentive.vue       # BCI激励
│   │   │   └── StorageStatus.vue      # 存储状态
│   │   └── layout/          # 布局组件
│   │       └── TopBar.vue             # 顶部状态栏
│   ├── composables/         # 组合式函数（Vue Composition API）
│   │   └── useRealtime.ts             # 实时数据轮询hooks
│   ├── services/            # API服务层
│   │   ├── api.ts                     # HTTP API客户端
│   │   └── websocket.ts               # WebSocket服务
│   ├── stores/              # Pinia状态管理
│   │   ├── system.ts                  # 系统状态
│   │   ├── pot.ts                     # POT共识状态
│   │   ├── committee.ts               # 委员会状态
│   │   ├── bci.ts                     # BCI激励状态
│   │   ├── mempool.ts                 # 交易池状态
│   │   ├── network.ts                 # 网络拓扑状态
│   │   └── storage.ts                 # 存储状态
│   ├── types/               # TypeScript类型定义
│   │   └── api.ts                     # API接口类型
│   ├── utils/               # 工具函数
│   │   └── format.ts                  # 格式化函数
│   ├── App.vue              # 根组件
│   ├── main.ts              # 应用入口
│   ├── shims-vue.d.ts       # Vue类型声明
│   └── vite-env.d.ts        # Vite环境变量类型
```

## 核心技术栈

### 1. Vue 3 + Composition API
使用 `<script setup>` 语法，更简洁的组件编写方式。

### 2. TypeScript
所有代码都使用 TypeScript，确保类型安全。

### 3. Pinia 状态管理
每个功能模块有独立的 store，职责清晰。

### 4. Ant Design Vue
使用 Ant Design Vue 4.x 组件库，提供丰富的 UI 组件。

### 5. TailwindCSS
原子化 CSS 框架，快速构建样式。

## 开发流程

### 第一步：安装依赖

```bash
cd web
npm install
```

### 第二步：启动开发服务器

```bash
npm run dev
```

访问 `http://localhost:3000` 查看效果。

### 第三步：连接后端 API

确保后端服务运行在 `http://localhost:8080`，并且实现了以下 API 端点：

- `/api/system/overview`
- `/api/pot/status`
- `/api/pot/vdf`
- `/api/committee/status`
- `/api/mempool/status`
- `/api/network/topology`
- `/api/bci/status`
- `/api/storage/status`
- WebSocket: `/api/ws`

### 第四步：调试

使用浏览器开发者工具查看：
- Network 面板：检查 API 请求
- Console 面板：查看日志输出
- Vue DevTools：调试组件状态

## API 集成说明

### HTTP API 调用

所有 HTTP API 调用都通过 `src/services/api.ts` 统一管理：

```typescript
import { api } from '@/services/api'

// 获取 POT 状态
const status = await api.getPotStatus()
```

### WebSocket 连接

WebSocket 连接在 `App.vue` 中初始化：

```typescript
import { WebSocketService } from '@/services/websocket'

const wsService = new WebSocketService('ws://localhost:8080/api/ws')
wsService.connect()
wsService.subscribe(['pot', 'vdf', 'system'])

// 监听消息
wsService.on('pot', (data) => {
  // 处理 POT 数据更新
})
```

## 状态管理

### Store 结构

每个 store 都遵循相同的结构：

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'

export const useXxxStore = defineStore('xxx', () => {
  const data = ref<DataType | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchData = async () => {
    loading.value = true
    error.value = null
    try {
      data.value = await api.getXxx()
    } catch (err: any) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, fetchData }
})
```

### 在组件中使用 Store

```vue
<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useXxxStore } from '@/stores/xxx'

const xxxStore = useXxxStore()
const { data, loading } = storeToRefs(xxxStore)

// 获取数据
xxxStore.fetchData()
</script>
```

## 组件开发规范

### 组件结构

```vue
<template>
  <div class="component-name card">
    <h3 class="title">标题</h3>
    <a-spin :spinning="loading">
      <div class="content">
        <!-- 内容 -->
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useXxxStore } from '@/stores/xxx'

const xxxStore = useXxxStore()
const { data, loading } = storeToRefs(xxxStore)
</script>

<style scoped>
/* 组件样式 */
</style>
```

### 样式规范

1. 使用 TailwindCSS 原子类优先
2. 自定义样式使用 `scoped` 避免污染
3. 颜色使用 TailwindCSS 配置的主题色

## 数据刷新策略

### 轮询刷新

在 `App.vue` 中设置了多个轮询定时器：

```typescript
// 1秒轮询：高频数据
setInterval(() => {
  systemStore.fetchOverview()
  potStore.fetchVDF()
}, 1000)

// 5秒轮询：中频数据
setInterval(() => {
  potStore.fetchStatus()
  committeeStore.fetchStatus()
  mempoolStore.fetchStatus()
}, 5000)

// 10秒轮询：低频数据
setInterval(() => {
  networkStore.fetchTopology()
  bciStore.fetchStatus()
}, 10000)
```

### WebSocket 实时更新

WebSocket 连接成功后，会接收服务器推送的实时数据，自动更新 store 中的状态。

## 常见问题

### 1. API 请求跨域问题

在 `vite.config.ts` 中配置了代理：

```typescript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

### 2. TypeScript 类型错误

确保所有 API 响应类型都在 `src/types/api.ts` 中定义。

### 3. 组件样式不生效

检查是否正确引入了 TailwindCSS 和 Ant Design Vue 的样式文件。

### 4. WebSocket 连接失败

检查后端 WebSocket 服务是否正常运行，端点是否正确。

## 性能优化建议

1. **图表优化**
   - 限制历史数据点数量（最多 200 个）
   - 使用 ECharts 的 `notMerge: false` 增量更新

2. **网络拓扑优化**
   - 节点数量 > 50 时降低渲染频率
   - 使用节点聚合功能

3. **数据更新优化**
   - WebSocket 批量更新
   - 使用防抖/节流控制更新频率

4. **组件渲染优化**
   - 使用 `v-show` 而非 `v-if` 频繁切换
   - 大列表使用虚拟滚动

## 部署说明

### 构建生产版本

```bash
npm run build
```

### 部署到 Nginx

将 `dist/` 目录中的文件部署到 Nginx：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    root /path/to/dist;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 下一步开发计划

当前已完成的功能模块：
- ✅ 项目基础结构
- ✅ API 服务层
- ✅ 状态管理（Pinia Stores）
- ✅ 基础组件（左侧、右侧面板）
- ✅ 顶部状态栏
- ✅ 网络拓扑框架（需要后续完善 G6 集成）

待完善的功能：
- [ ] 网络拓扑 G6 图表完整实现
- [ ] 委员会详情面板
- [ ] 系统日志/事件流
- [ ] 历史数据趋势图
- [ ] 响应式布局优化
- [ ] 主题切换功能
- [ ] 数据导出功能

## 参考资料

- [Vue 3 文档](https://vuejs.org/)
- [Ant Design Vue 文档](https://antdv.com/)
- [Pinia 文档](https://pinia.vuejs.org/)
- [TailwindCSS 文档](https://tailwindcss.com/)
- [@antv/g6 文档](https://g6.antv.antgroup.com/)
- [ECharts 文档](https://echarts.apache.org/)
