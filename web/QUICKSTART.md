# POT 可视化项目快速上手指南

## 📦 项目已创建完成

恭喜！POT 共识可视化前端项目已在 `web/` 目录下创建完成。

## 🚀 快速启动（3步）

### 1️⃣ 安装依赖

```bash
cd /home/ldc/workspace/pot/web
npm install
```

或者使用提供的安装脚本：

```bash
cd /home/ldc/workspace/pot/web
./install.sh
```

### 2️⃣ 启动开发服务器

```bash
npm run dev
```

或者使用快速启动脚本：

```bash
./start.sh
```

### 3️⃣ 访问应用

打开浏览器访问：**http://localhost:3000**

## 📋 前提条件

### Node.js 环境

- Node.js >= 16.x
- npm >= 7.x

检查版本：

```bash
node -v
npm -v
```

### 后端服务

前端依赖后端 API 服务，请确保：

1. 后端服务运行在 `http://localhost:8080`
2. 实现了以下 API 端点（参考 `docs/visualize-api-summary.md`）：
   - `/api/system/overview`
   - `/api/pot/status`
   - `/api/pot/vdf`
   - `/api/committee/status`
   - `/api/mempool/status`
   - `/api/network/topology`
   - `/api/bci/status`
   - `/api/storage/status`
   - WebSocket: `/api/ws`

## 📁 项目结构

```
web/
├── src/
│   ├── components/      # Vue 组件
│   │   ├── blocks/      # 功能区块
│   │   └── layout/      # 布局组件
│   ├── services/        # API 服务
│   ├── stores/          # 状态管理
│   ├── types/           # 类型定义
│   ├── utils/           # 工具函数
│   └── App.vue          # 根组件
├── README.md            # 项目说明
├── DEVELOPMENT.md       # 开发指南
├── PROJECT_SUMMARY.md   # 项目总结
├── install.sh           # 安装脚本
└── start.sh             # 启动脚本
```

## 🔧 常用命令

```bash
# 安装依赖
npm install

# 启动开发服务器 (http://localhost:3000)
npm run dev

# 构建生产版本
npm run build

# 预览构建结果
npm run preview

# 类型检查
npm run type-check
```

## 🎨 功能模块

### 顶部状态栏
- 系统运行时长
- 区块高度
- 实时 TPS
- 节点状态
- 网络健康度

### 左侧面板
1. **POT 共识状态** - Epoch、难度、挖矿状态
2. **VDF 计算监控** - VDF0/VDF1/VDFHalf 进度
3. **性能指标** - TPS、出块时间

### 中心区域
- **网络拓扑图** - 双层网络结构可视化

### 右侧面板
1. **交易池监控** - 交易数量、类型分布
2. **BCI 激励** - 奖励金额、分配比例
3. **存储状态** - 存储容量、区块数量

## 🔌 API 接口配置

### 开发环境

编辑 `.env.development`：

```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/api/ws
```

### 生产环境

编辑 `.env.production`：

```env
VITE_API_BASE_URL=https://your-domain.com/api
VITE_WS_URL=wss://your-domain.com/api/ws
```

## 📚 文档

- **README.md** - 项目基本信息和快速开始
- **DEVELOPMENT.md** - 详细的开发指南
- **PROJECT_SUMMARY.md** - 项目完成情况总结
- **docs/visualize-api-summary.md** - API 接口文档（含后端实现建议）
- **docs/visualize-layout-design.md** - 布局设计方案
- **docs/visualize-frontend-guide.md** - 前端开发详细指南

## ⚠️ 注意事项

### 1. CORS 问题

如果遇到跨域问题，确保后端已配置 CORS：

```go
// Go 后端示例
router.Use(func(c *gin.Context) {
    c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
    c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    c.Next()
})
```

或使用 Vite 代理（已配置）。

### 2. WebSocket 连接

如果 WebSocket 连接失败，前端会自动降级到 HTTP 轮询模式。

### 3. 端口占用

如果 3000 端口被占用，可以修改 `vite.config.ts` 中的端口：

```typescript
server: {
  port: 3001  // 修改为其他端口
}
```

## 🐛 常见问题

### Q: 安装依赖失败？

A: 尝试清除缓存后重新安装：

```bash
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

### Q: 页面显示空白？

A: 检查：
1. 浏览器控制台是否有错误
2. 后端 API 是否正常运行
3. Network 面板查看 API 请求状态

### Q: 数据不更新？

A: 检查：
1. WebSocket 连接状态
2. 后端是否实现了推送
3. 浏览器控制台是否有错误

## 🎯 下一步工作

### 必须完成

1. **实现后端 API** - 根据 `docs/visualize-api-summary.md`
2. **完善网络拓扑** - 集成 @antv/g6 图表

### 可选增强

1. 委员会详情面板
2. 系统日志/事件流
3. 历史数据趋势图
4. 响应式布局优化
5. 主题切换功能

## 📞 技术支持

如遇到问题，请查看：

1. 项目文档（README.md, DEVELOPMENT.md）
2. API 文档（docs/visualize-api-summary.md）
3. 浏览器开发者工具控制台
4. Network 面板查看 API 请求

## ✨ 技术栈

- **Vue 3.3.4** - 前端框架
- **TypeScript 5.3** - 类型系统
- **Vite 5.0** - 构建工具
- **Ant Design Vue 4.1.2** - UI 组件库
- **Pinia 2.0.23** - 状态管理
- **@antv/g6 4.8** - 图可视化
- **ECharts 5.4** - 数据可视化
- **TailwindCSS 3.3** - CSS 框架

## 📄 许可证

[项目许可证信息]

---

**开始构建你的可视化大屏吧！** 🚀
