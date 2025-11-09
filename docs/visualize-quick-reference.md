# 可视化开发快速参考卡

## 🚀 5分钟快速开始

### 前端开发者
```bash
# 1. 创建项目
npm create vite@latest pot-visualization -- --template vue-ts
cd pot-visualization

# 2. 安装依赖
npm install element-plus echarts vue-echarts @antv/g6 axios pinia @vueuse/core dayjs

# 3. 配置TailwindCSS
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p

# 4. 启动开发
npm run dev
```

### 后端开发者
```bash
# 1. 创建可视化API目录
cd internal/apis
mkdir -p visualization/{handlers,services,models,middleware,websocket,metrics}

# 2. 在Worker中添加统计字段 (参考后端检查清单)

# 3. 实现核心API (按优先级)
#    - GET /api/system/overview
#    - GET /api/pot/status
#    - GET /api/pot/vdf

# 4. 启动可视化服务 (端口9090)
```

---

## 📊 8个区块速查表

| # | 位置 | 名称 | 组件 | API | 刷新 |
|---|------|------|------|-----|------|
| 1 | 左上 | POT共识 | `PotConsensus.vue` | `/api/pot/status`, `/api/pot/vdf` | 5秒 |
| 2 | 中上 | 委员会 | `CommitteeConsensus.vue` | `/api/committee/status` | 5秒 |
| 3 | 右上中 | 交易池 | `MempoolMonitor.vue` | `/api/mempool/status` | 5秒 |
| 4 | 右上 | 网络 | `NetworkTopology.vue` | `/api/network/topology` | 10秒 |
| 5 | 左下 | VDF | `VDFMonitor.vue` | `/api/pot/vdf` | 1秒 |
| 6 | 中下 | BCI | `BCIIncentive.vue` | `/api/bci/status` | 10秒 |
| 7 | 右下中 | 存储 | `StorageStatus.vue` | `/api/storage/status` | 30秒 |
| 8 | 右下 | 性能 | `PerformanceMetrics.vue` | `/api/system/overview` | 1秒 |

---

## 🔌 API端点速查

```typescript
// 系统概览
GET /api/system/overview
→ { uptime, totalNodes, currentTPS, currentHeight, ... }

// POT状态
GET /api/pot/status
→ { consensusType, epoch, difficulty, workFlag, ... }

// VDF状态
GET /api/pot/vdf
→ { vdf0: {...}, vdf1: [...], vdfHalf: {...}, cpuCounter }

// 委员会状态
GET /api/committee/status
→ { committeeSize, committee: [...], shardings: [...], ... }

// BCI激励
GET /api/bci/status
→ { totalReward, rewardRatio: {...}, lockRates: {...}, ... }

// 交易池
GET /api/mempool/status
→ { totalSize, markedTxs, unmarkedTxs, txTypes: {...}, ... }

// 网络拓扑
GET /api/network/topology
→ { nodes: [...], edges: [...], subscribedTopics, ... }

// 存储状态
GET /api/storage/status
→ { totalSize, buckets: {...}, compressionRatio, ... }

// 历史数据
GET /api/metrics/history?metric=tps&timeRange=24h&interval=5m
→ { data: [...], avg, max, min }

// WebSocket
ws://localhost:8080/api/ws
→ send: { action: "subscribe", types: ["pot", "vdf"] }
→ recv: { type: "pot", timestamp, data: {...} }
```

---

## 🎨 颜色代码

```css
/* 背景 */
bg-primary:   #0a0e27  /* 深蓝黑 */
bg-secondary: #141d3a  /* 深蓝 */
border:       #1e3a5f  /* 蓝灰 */

/* 功能色 */
pot:       #00d4ff  /* 青蓝 - POT共识 */
committee: #00ff88  /* 翠绿 - 委员会 */
transaction: #ffa500  /* 橙色 - 交易 */
incentive: #ffd700  /* 金色 - 激励 */
storage:   #9370db  /* 紫色 - 存储 */
vdf:       #ff69b4  /* 粉红 - VDF */

/* 状态色 */
normal:  #00ff00  /* 绿 */
warning: #ffff00  /* 黄 */
error:   #ff0000  /* 红 */
idle:    #808080  /* 灰 */
```

---

## 💻 核心组件模板

### StatusCard 组件
```vue
<StatusCard
  label="区块高度"
  :value="12345"
  status="normal"
  subtitle="最新区块"
/>
```

### ProgressCircle 组件
```vue
<ProgressCircle
  :progress="75.5"
  :size="120"
  color="#00d4ff"
  label="Epoch"
/>
```

### Pinia Store 使用
```typescript
// 在组件中
const potStore = usePotStore()
const status = computed(() => potStore.status)

// 获取数据
await potStore.fetchStatus()

// WebSocket更新
potStore.updateFromWebSocket(data)
```

---

## 🔧 常用工具函数

```typescript
// 数字格式化
formatNumber(12345) → "12,345"
formatNumber(1234567890) → "12345..."

// 百分比
formatPercent(0.855) → "85.5%"

// 时间格式化
formatTime(timestamp) → "2025-11-08 10:30:45"

// 时长格式化
formatDuration(86400) → "1天0小时0分"

// 文件大小
formatBytes(1073741824) → "1.00 GB"
```

---

## 🐛 调试技巧

### 前端调试
```javascript
// 1. 在浏览器Console中
console.log(potStore.status)

// 2. 使用Vue DevTools
// 安装: Vue.js devtools 浏览器插件

// 3. 检查API调用
// Network面板 → 查看XHR/Fetch

// 4. 模拟数据
const mockData = { ... }
potStore.status = mockData
```

### 后端调试
```go
// 1. 添加日志
log.Infof("VDF Status: %+v", vdfStatus)

// 2. 测试API
curl http://localhost:9090/api/pot/status | jq

// 3. 检查Worker状态
if worker == nil {
    return nil, fmt.Errorf("worker not initialized")
}

// 4. 性能分析
import _ "net/http/pprof"
go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
```

---

## ⚡ 性能优化清单

### 前端
- [ ] 使用`v-memo`缓存组件
- [ ] 图表使用`notMerge: false`
- [ ] 大列表使用虚拟滚动
- [ ] 防抖/节流频繁更新
- [ ] WebSocket优先于轮询
- [ ] 路由懒加载

### 后端
- [ ] 添加缓存层（5-30秒）
- [ ] 使用读写锁
- [ ] 限制并发连接数
- [ ] 异步处理耗时操作
- [ ] 预聚合历史数据
- [ ] 启用Gzip压缩

---

## 📝 Git提交规范

```bash
# 功能开发
git commit -m "feat(viz): 添加POT共识状态组件"

# Bug修复
git commit -m "fix(viz): 修复VDF进度计算错误"

# 文档更新
git commit -m "docs(viz): 更新API文档"

# 样式调整
git commit -m "style(viz): 优化卡片间距"

# 重构
git commit -m "refactor(viz): 重构API服务层"

# 性能优化
git commit -m "perf(viz): 优化图表渲染性能"
```

---

## 🧪 测试命令

```bash
# 前端
npm run test:unit       # 单元测试
npm run test:e2e        # E2E测试
npm run lint            # 代码检查
npm run build           # 生产构建

# 后端
go test ./internal/apis/visualization/...  # 单元测试
go test -race ./...                        # 竞态检测
go test -bench=.                           # 基准测试
go vet ./...                               # 静态检查

# API测试
curl http://localhost:9090/api/pot/status
ab -n 1000 -c 100 http://localhost:9090/api/system/overview
```

---

## 📦 构建部署

```bash
# 前端构建
npm run build
# 输出: dist/

# 后端构建
go build -o pot-node .

# Docker构建
docker build -t pot-node:latest .
docker run -p 8080:8080 -p 9090:9090 pot-node:latest

# Nginx配置
upstream api_backend {
    server 127.0.0.1:9090;
}

server {
    listen 80;
    
    # 前端静态文件
    location / {
        root /var/www/pot-visualization/dist;
        try_files $uri $uri/ /index.html;
    }
    
    # API代理
    location /api/ {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## 📞 快速联系

| 问题类型 | 查阅文档 | 关键词 |
|---------|---------|--------|
| 不知道有什么指标 | `visualize-plan.md` | 指标、板块 |
| 页面怎么布局 | `visualize-layout-design.md` | 布局、区块、颜色 |
| 前端怎么开发 | `visualize-frontend-guide.md` | Vue、组件、Store |
| API怎么设计 | `visualize-api-implementation.md` | 架构、Service |
| API端点是什么 | `visualize-api-summary.md` | 端点、TypeScript |
| 后端怎么实现 | `visualize-backend-checklist.md` | 检查清单、模板 |
| 总体怎么开始 | `visualize-README.md` | 总览、流程 |

---

## ⏱️ 预估开发时间

| 角色 | 阶段 | 时间 |
|------|------|------|
| 前端 | 项目初始化 | 1-2天 |
| 前端 | 通用组件 | 2天 |
| 前端 | 区块组件 | 4-5天 |
| 前端 | 集成测试 | 2天 |
| **前端总计** | | **9-11天** |
| 后端 | 目录和Models | 1天 |
| 后端 | 核心API | 2-3天 |
| 后端 | 扩展API | 2-3天 |
| 后端 | WebSocket | 1天 |
| 后端 | 测试集成 | 1天 |
| **后端总计** | | **7-9天** |

**总计**: 16-20天（前后端并行）

---

**打印此卡片，贴在显示器旁边！📌**
