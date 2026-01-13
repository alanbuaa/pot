# POT 共识可视化大屏

[![Vue](https://img.shields.io/badge/Vue-3.3.4-brightgreen.svg)](https://vuejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.3-blue.svg)](https://www.typescriptlang.org/)
[![Vite](https://img.shields.io/badge/Vite-5.0-646CFF.svg)](https://vitejs.dev/)

基于 Vue 3 + TypeScript + Ant Design Vue 构建的 POT 共识系统可视化监控大屏，提供实时数据监控、网络拓扑展示和 3D 视觉效果。

## ✨ 核心特性

- 🎨 **3D 粒子背景** - Three.js 驱动的 2000+ 粒子动画
- 📊 **实时监控** - POT 共识、VDF 计算、性能指标
- 🌐 **网络拓扑** - 双层网络结构可视化（委员会层 + POT 节点层）
- 💰 **BCI 激励** - 奖励分配、锁定状态、利息统计
- 🔗 **区块链动画** - 底部横向滚动展示区块增长
- 🔄 **双重数据源** - HTTP 轮询 + WebSocket 实时推送
- 🎭 **Mock 模式** - 支持前端独立开发和演示

## 🚀 快速开始

### 环境要求

- Node.js >= 18.0.0
- npm >= 9.0.0

### 安装与运行

```bash
# 1. 进入项目目录
cd web

# 2. 安装依赖
npm install

# 3. 启动开发服务器（Mock 模式）
npm run dev

# 4. 访问应用
# http://localhost:5173
```

### Mock 模式 vs 真实后端

**Mock 模式**（默认，无需后端）：
```env
# .env.development
VITE_USE_MOCK=true
```

**真实后端模式**：
```env
# .env.development
VITE_USE_MOCK=false
VITE_API_BASE_URL=http://localhost:8080
```

## 🎯 功能模块

| 模块 | 功能 | 位置 |
|------|------|------|
| 顶部状态栏 | 运行时长、区块高度、TPS、节点状态 | 全屏顶部 |
| POT 共识 | Epoch、难度、挖矿状态 | 左侧上 |
| VDF 监控 | VDF0/VDF1/VDFHalf 进度 | 左侧中 |
| 性能指标 | TPS、出块时间 | 左侧下 |
| 网络拓扑 | 委员会 + POT 节点双层结构 | 中心区域 |
| 交易池监控 | 交易数量、类型分布 | 右侧上 |
| BCI 激励 | 奖励金额、分配比例 | 右侧中 |
| 系统信息 | 执行器状态、存储使用率 | 右侧下 |
| 区块链动画 | 区块横向滚动展示 | 底部 |

## 🛠️ 技术栈

- **Vue 3.3.4** - 组合式 API
- **TypeScript 5.3** - 类型安全
- **Vite 5.0** - 快速构建
- **Ant Design Vue 4.1.2** - UI 组件
- **Pinia 2.0.23** - 状态管理
- **@antv/g6 4.8** - 网络拓扑
- **ECharts 5.4** - 数据可视化
- **Three.js 0.160** - 3D 效果
- **TailwindCSS 3.3** - 样式框架

## 📁 项目结构

```
web/
├── src/
│   ├── components/       # 组件
│   │   ├── blocks/       # 功能模块（POT、VDF、交易池等）
│   │   ├── layout/       # 布局组件（顶部栏等）
│   │   ├── BlockChain.vue      # 区块链动画
│   │   └── ThreeBackground.vue # 3D 背景
│   ├── services/         # API 服务
│   │   ├── api.ts        # HTTP 客户端
│   │   ├── mock.ts       # Mock 数据
│   │   └── websocket.ts  # WebSocket
│   ├── stores/           # 状态管理（Pinia）
│   ├── types/            # TypeScript 类型
│   └── utils/            # 工具函数
├── docs/                 # 详细文档
├── package.json
└── vite.config.ts
```

## 📚 文档

| 文档 | 说明 |
|------|------|
| [QUICKSTART.md](QUICKSTART.md) | 快速上手指南 |
| [DEVELOPMENT.md](DEVELOPMENT.md) | 开发指南 |
| [INSTALL.md](INSTALL.md) | 详细安装说明 |
| [API.md](API.md) | API 接口文档 |
| [CHANGES.md](CHANGES.md) | 更新日志 |

## 🔧 常用命令

```bash
# 开发
npm run dev          # 启动开发服务器
npm run build        # 构建生产版本
npm run preview      # 预览构建结果

# 代码质量
npm run type-check   # TypeScript 类型检查
npm run lint         # 代码检查
```

## 🌐 API 接口

系统需要后端提供以下 API（详见 [API.md](API.md)）：

```
GET /api/system/overview      # 系统概览
GET /api/pot/status          # POT 状态
GET /api/pot/vdf             # VDF 状态
GET /api/committee/status    # 委员会状态
GET /api/mempool/status      # 交易池状态
GET /api/network/topology    # 网络拓扑
GET /api/bci/status          # BCI 激励
WS  /api/ws                  # WebSocket 实时推送
```

## 🐛 常见问题

**Q: 页面显示空白？**
- 检查浏览器控制台是否有错误
- 确认是否已执行 `npm install`
- 检查 Mock 模式配置

**Q: API 请求跨域？**
- 开发模式已配置代理（`vite.config.ts`）
- 生产环境需后端配置 CORS

**Q: 端口被占用？**
- Vite 会自动选择下一个可用端口
- 或手动指定：`npm run dev -- --port 3000`

**Q: 如何切换到真实后端？**
```bash
# 1. 修改 .env.development
VITE_USE_MOCK=false

# 2. 确保后端服务运行
cd /home/ldc/workspace/pot
make run_server

# 3. 重启前端
npm run dev
```

## 📊 数据刷新策略

| 数据类型 | 刷新频率 | Mock 支持 |
|---------|---------|-----------|
| 系统概览 | 1秒 | ✅ |
| VDF 状态 | 1秒 | ✅ |
| POT 状态 | 5秒 | ✅ |
| 委员会、交易池 | 5秒 | ✅ |
| 网络拓扑、BCI | 10秒 | ✅ |

支持 WebSocket 实时推送，连接失败时自动降级到 HTTP 轮询。

## 🎨 视觉特性

- ✨ **3D 粒子背景** - 2000+ 粒子，POT 主题配色
- 💫 **区块脉冲动画** - 新区块出现动画效果
- 🌈 **网络拓扑** - G6 自定义节点渲染
- 📈 **平滑过渡** - 所有数据变化平滑动画

## 🚢 部署

### 构建

```bash
npm run build
```

### Nginx 配置示例

```nginx
server {
    listen 80;
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

## 📄 许可证

本项目采用 Apache 2.0 许可证，详见项目根目录 LICENSE 文件。

## 🙏 致谢

感谢所有贡献者以及开源项目的支持。

---

**开始构建你的可视化大屏！** 🚀
