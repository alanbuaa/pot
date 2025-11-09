# POT 共识可视化大屏

这是一个基于 Vue 3 + TypeScript + Ant Design Vue 构建的 POT 共识系统可视化大屏项目。

## 技术栈

- **Vue 3.3.4** - 渐进式 JavaScript 框架
- **TypeScript 5.3** - JavaScript 的超集
- **Vite 5.0** - 下一代前端构建工具
- **Ant Design Vue 4.1.2** - 企业级 UI 组件库
- **Pinia 2.0.23** - Vue 状态管理
- **@antv/g6 4.8** - 图可视化引擎
- **ECharts 5.4** - 数据可视化图表库
- **TailwindCSS 3.3** - 原子化 CSS 框架

## 项目结构

```
web/
├── src/
│   ├── assets/           # 静态资源
│   ├── components/       # Vue 组件
│   │   ├── blocks/       # 功能区块组件
│   │   └── layout/       # 布局组件
│   ├── composables/      # 组合式函数
│   ├── services/         # API 服务
│   ├── stores/           # Pinia 状态管理
│   ├── types/            # TypeScript 类型定义
│   ├── utils/            # 工具函数
│   ├── App.vue           # 根组件
│   └── main.ts           # 入口文件
├── public/               # 公共静态资源
├── index.html            # HTML 模板
├── package.json          # 项目依赖
├── vite.config.ts        # Vite 配置
├── tsconfig.json         # TypeScript 配置
└── tailwind.config.js    # TailwindCSS 配置
```

## 功能模块

### 顶部状态栏
- 系统运行时长
- 区块高度
- 实时 TPS
- 节点状态
- 网络健康度

### 左侧面板
1. **POT 共识状态** - Epoch、难度、挖矿状态
2. **VDF 计算监控** - VDF0/VDF1/VDFHalf 进度
3. **性能指标** - TPS、出块时间、网络利用率

### 中心区域
- **网络拓扑图** - 双层网络结构可视化（委员会层 + POT节点层）

### 右侧面板
1. **交易池监控** - 交易数量、类型分布
2. **BCI 激励** - 奖励金额、分配比例
3. **存储状态** - 存储容量、区块数量

## 快速开始

### 安装依赖

```bash
cd web
npm install
```

### 开发环境运行

```bash
npm run dev
```

项目将运行在 `http://localhost:3000`

### 生产构建

```bash
npm run build
```

构建产物将输出到 `dist/` 目录

### 预览构建结果

```bash
npm run preview
```

## 环境变量

### 开发环境 (`.env.development`)
```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/api/ws
```

### 生产环境 (`.env.production`)
```env
VITE_API_BASE_URL=https://api.example.com/api
VITE_WS_URL=wss://api.example.com/api/ws
```

## API 接口

项目依赖后端提供以下 API 接口：

- `GET /api/system/overview` - 系统概览
- `GET /api/pot/status` - POT 共识状态
- `GET /api/pot/vdf` - VDF 计算状态
- `GET /api/committee/status` - 委员会状态
- `GET /api/mempool/status` - 交易池状态
- `GET /api/network/topology` - 网络拓扑
- `GET /api/bci/status` - BCI 激励状态
- `GET /api/storage/status` - 存储状态
- `WS /api/ws` - WebSocket 实时推送

详细 API 文档请参考 `docs/visualize-api-summary.md`

## 数据更新策略

- **1秒轮询**: 系统概览、VDF 状态
- **5秒轮询**: POT 状态、委员会状态、交易池状态
- **10秒轮询**: 网络拓扑、BCI 状态
- **30秒轮询**: 存储状态

同时支持 WebSocket 实时推送，当 WebSocket 连接失败时自动降级到 HTTP 轮询。

## 开发注意事项

1. **样式**: 使用 TailwindCSS 原子类 + 自定义 CSS
2. **状态管理**: 使用 Pinia，每个模块一个 store
3. **类型安全**: 所有 API 响应都有 TypeScript 类型定义
4. **组件拆分**: 按功能模块拆分组件，保持单一职责
5. **性能优化**: 
   - 使用 `v-show` 而非 `v-if` 来频繁切换显示
   - 大数据列表使用虚拟滚动
   - 图表数据限制最大数据点数量

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## 许可证

[项目许可证信息]
