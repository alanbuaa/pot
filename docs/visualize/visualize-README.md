# 可视化大屏开发文档总览

欢迎查看POT共识可视化大屏完整开发文档。本文档集提供了从需求分析到实际开发的完整指南。

## 📚 文档导航

### 1️⃣ 需求和规划文档

#### [`visualize-plan.md`](./visualize-plan.md) - 可视化指标规划
**用途**: 定义所有需要可视化的指标  
**包含内容**:
- 整体架构板块 (15个指标)
- POT共识状态 (14个指标)
- 委员会共识 (16个指标)
- BCI激励 (16个指标)
- 交易池和执行状态 (10个指标)
- 网络和P2P (3个指标)

**适合人群**: 产品经理、需求分析师、开发团队

---

### 2️⃣ 前端设计文档

#### [`visualize-layout-design.md`](./visualize-layout-design.md) - 页面布局设计
**用途**: UI/UX设计方案  
**包含内容**:
- 4×3网格布局方案
- 8个区块的详细设计
- 顶部状态栏和底部时间轴
- 技术栈建议（Vue 3 + ECharts + TailwindCSS）
- 颜色方案和交互设计
- 响应式布局策略

**适合人群**: UI/UX设计师、前端开发

---

#### [`visualize-frontend-guide.md`](./visualize-frontend-guide.md) - 前端开发指南
**用途**: 前端实现的完整教程  
**包含内容**:
- 项目初始化步骤
- 完整项目结构
- 核心代码实现（API服务、Store、组件）
- 开发流程（分5个阶段）
- 示例代码（StatusCard、ProgressCircle等）
- 环境配置和部署

**适合人群**: 前端开发工程师

---

### 3️⃣ 后端设计文档

#### [`visualize-api-implementation.md`](./visualize-api-implementation.md) - API实现方案
**用途**: 后端API设计和实现逻辑  
**包含内容**:
- API架构设计（三层架构）
- 8个核心API端点详细设计
- WebSocket实时推送机制
- 数据采集器设计
- 性能优化建议
- 安全性考虑（认证、限流、CORS）
- 部署建议

**适合人群**: 后端开发工程师、架构师

---

#### [`visualize-api-summary.md`](./visualize-api-summary.md) - API端点总结
**用途**: API快速参考手册  
**包含内容**:
- 所有API端点清单
- TypeScript类型定义
- 刷新频率建议
- WebSocket使用说明
- 前端状态管理建议
- 开发检查清单

**适合人群**: 前后端开发工程师（快速查阅）

---

#### [`visualize-backend-checklist.md`](./visualize-backend-checklist.md) - 后端开发检查清单
**用途**: 后端开发的详细步骤指南  
**包含内容**:
- 开发优先级（3个阶段）
- 目录结构和文件创建
- 核心代码模板（Models、Services、Handlers）
- Worker统计字段添加
- 系统集成步骤
- 测试检查清单
- 常见问题解答

**适合人群**: 后端开发工程师（实施阶段）

---

## 🚀 快速开始指南

### 如果你是项目经理
1. 阅读 [`visualize-plan.md`](./visualize-plan.md) 了解所有指标
2. 阅读 [`visualize-layout-design.md`](./visualize-layout-design.md) 查看UI设计
3. 制定开发计划和时间表

### 如果你是前端开发
1. 阅读 [`visualize-layout-design.md`](./visualize-layout-design.md) 了解设计
2. 阅读 [`visualize-api-summary.md`](./visualize-api-summary.md) 了解API
3. 按照 [`visualize-frontend-guide.md`](./visualize-frontend-guide.md) 开始开发

**推荐开发流程**:
```
第1-2天: 项目初始化 + API服务层
第3天:   状态管理 (Pinia Stores)
第4-5天: 通用组件开发
第6-10天: 8个区块组件开发
第11-12天: 布局组件 + 集成测试
```

### 如果你是后端开发
1. 阅读 [`visualize-api-implementation.md`](./visualize-api-implementation.md) 了解架构
2. 阅读 [`visualize-api-summary.md`](./visualize-api-summary.md) 确认API规范
3. 按照 [`visualize-backend-checklist.md`](./visualize-backend-checklist.md) 开始开发

**推荐开发流程**:
```
第1天:  创建目录结构 + Models定义
第2-3天: 核心API (system/overview, pot/status, pot/vdf)
第4-5天: 扩展API (committee, bci, mempool, network, storage)
第6天:   WebSocket实现
第7天:   Metrics采集器
第8天:   集成测试
```

---

## 📋 技术栈总览

### 前端技术栈
```
核心框架:    Vue 3 + TypeScript
UI组件库:    Element Plus
图表库:      ECharts 5.x
网络图:      @antv/g6
状态管理:    Pinia
HTTP客户端:  Axios
工具库:      @vueuse/core, dayjs
样式:        TailwindCSS
实时通信:    WebSocket
```

### 后端技术栈
```
语言:       Go 1.21+
Web框架:    Gin
WebSocket:  gorilla/websocket
日志:       logrus
数据库:     BoltDB (已有)
缓存:       go-cache (可选)
监控:       Prometheus (可选)
```

---

## 🎯 关键指标对照表

| 板块 | 主要指标数量 | 刷新频率 | API端点 |
|------|-------------|---------|---------|
| 系统概览 | 15个 | 1秒 | `/api/system/overview` |
| POT共识 | 14个 | 5秒 | `/api/pot/status`, `/api/pot/vdf` |
| 委员会 | 16个 | 5秒 | `/api/committee/status` |
| BCI激励 | 16个 | 10秒 | `/api/bci/status` |
| 交易池 | 10个 | 5秒 | `/api/mempool/status` |
| 网络拓扑 | 3个 | 10秒 | `/api/network/topology` |
| 存储状态 | 5个 | 30秒 | `/api/storage/status` |
| 历史数据 | 动态 | 30秒 | `/api/metrics/history` |

---

## 📊 区块布局对照表

| 区块位置 | 区块名称 | 主要内容 | 数据来源 | 组件文件 |
|---------|---------|---------|---------|---------|
| 左上 | POT共识状态 | Epoch进度、难度、VDF0、挖矿状态 | POT API | `PotConsensus.vue` |
| 中上 | 委员会共识 | 拓扑图、工作阶段、分片状态 | Committee API | `CommitteeConsensus.vue` |
| 右上(中) | 交易池监控 | 饼图、类型分布、实时交易流 | Mempool API | `MempoolMonitor.vue` |
| 右上 | 网络拓扑 | P2P节点图、网络指标 | Network API | `NetworkTopology.vue` |
| 左下 | VDF计算 | VDF0/1/half进度、性能指标 | VDF API | `VDFMonitor.vue` |
| 中下 | BCI激励 | 奖励分配、利率、UTXO | BCI API | `BCIIncentive.vue` |
| 右下(中) | 存储状态 | 数据库容量、存储趋势 | Storage API | `StorageStatus.vue` |
| 右下 | 性能指标 | TPS、队列、系统健康度 | System API | `PerformanceMetrics.vue` |

---

## 🔄 开发协作流程

### 1. 需求确认阶段
- [ ] 产品经理确认所有指标需求
- [ ] UI设计师完成视觉设计稿
- [ ] 前后端确认API规范

### 2. 并行开发阶段
```
后端开发                     前端开发
   ↓                           ↓
创建API框架                  项目初始化
   ↓                           ↓
实现核心API                  开发通用组件
   ↓                           ↓
实现扩展API         ←→      开发区块组件（使用Mock数据）
   ↓                           ↓
WebSocket实现               集成真实API
```

### 3. 联调测试阶段
- [ ] API接口联调
- [ ] WebSocket实时数据测试
- [ ] 数据准确性验证
- [ ] 性能测试

### 4. 部署上线阶段
- [ ] 后端API部署
- [ ] 前端打包部署
- [ ] 监控配置
- [ ] 文档归档

---

## 🧪 测试策略

### 后端测试
```bash
# 单元测试
go test ./internal/apis/visualization/...

# API测试
curl http://localhost:9090/api/pot/status

# 压力测试
ab -n 1000 -c 100 http://localhost:9090/api/system/overview
```

### 前端测试
```bash
# 单元测试
npm run test:unit

# E2E测试
npm run test:e2e

# 开发环境
npm run dev
```

---

## 📦 部署架构

```
┌─────────────────────────────────────────────────┐
│               Nginx (反向代理)                    │
│  - 静态文件服务 (前端)                             │
│  - API代理转发                                    │
└──────────────┬──────────────────────────────────┘
               │
       ┌───────┴───────┐
       │               │
   ┌───▼───┐      ┌───▼───┐
   │ 前端   │      │ 后端   │
   │ Vue3  │      │ Go API │
   │ Dist  │      │ :9090  │
   └───────┘      └───┬───┘
                      │
                  ┌───▼───┐
                  │ POT   │
                  │Worker │
                  └───────┘
```

---

## 🎨 可视化效果预览

### 顶部状态栏
```
运行时长: 2天3小时  |  区块高度: 12,345  |  TPS: 45.6  |  节点: 4/4  |  网络: 健康
```

### 区块1: POT共识
```
┌─────────────────────────┐
│  Epoch: 1234  ◉ 75%    │
│  ┌─────────┐            │
│  │ 难度仪表 │            │
│  │  12345  │            │
│  └─────────┘            │
│  VDF0: ████████░░ 80%  │
│  VDF1: [█][█][█][█]     │
│  挖矿: ● 工作中          │
└─────────────────────────┘
```

---

## 📝 开发注意事项

### 性能优化
1. **前端**:
   - 使用虚拟滚动处理大量交易
   - 图表按需加载
   - WebSocket优先，HTTP轮询备用

2. **后端**:
   - 实现多级缓存
   - 限流保护
   - 避免在API Handler中执行耗时操作

### 数据准确性
1. 所有数值保留2位小数
2. 大数字使用千分位分隔
3. 时间统一使用ISO 8601格式
4. 百分比范围0-100

### 错误处理
1. API超时时间: 10秒
2. 失败重试: 最多3次
3. WebSocket断线自动重连
4. 降级方案: 显示历史数据

---

## 🆘 问题排查

### 常见问题
1. **API返回空数据?**
   - 检查Worker是否正确初始化
   - 检查是否有权限访问数据

2. **WebSocket无法连接?**
   - 检查端口是否开放
   - 检查CORS配置
   - 查看浏览器Console错误

3. **前端显示异常?**
   - 检查API返回数据格式
   - 查看浏览器Network面板
   - 检查TypeScript类型定义

---

## 📞 支持和反馈

如有问题，请查阅相关文档或联系开发团队。

---

## ✅ 总检查清单

### 开始开发前
- [ ] 阅读所有相关文档
- [ ] 理解技术架构
- [ ] 确认开发环境
- [ ] 准备开发工具

### 开发过程中
- [ ] 按照API规范开发
- [ ] 保持代码质量
- [ ] 及时提交代码
- [ ] 编写测试用例

### 开发完成后
- [ ] 完成所有功能
- [ ] 通过所有测试
- [ ] 更新文档
- [ ] 代码Review

---

**祝开发顺利！🎉**

最后更新: 2025-11-08
