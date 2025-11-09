# POT 共识可视化项目 - 安装与运行指南

## 📦 安装依赖

### 前置要求
- Node.js >= 18.0.0
- npm >= 9.0.0 或 yarn >= 1.22.0

### 安装步骤

1. **进入 web 项目目录**
```bash
cd /home/ldc/workspace/pot/web
```

2. **安装所有依赖**
```bash
npm install
```

或使用 yarn:
```bash
yarn install
```

### 依赖说明

项目使用的主要依赖包括:

#### 核心框架
- `vue@^3.3.4` - Vue 3 框架
- `typescript@~5.3.0` - TypeScript 类型支持
- `vite@^5.0.0` - 快速构建工具

#### UI 库
- `ant-design-vue@^4.1.2` - Ant Design Vue 组件库
- `@ant-design/icons-vue@^6.1.0` - Ant Design 图标
- `tailwindcss@^3.3.0` - 实用优先的 CSS 框架

#### 状态管理与工具
- `pinia@^2.0.23` - Vue 3 状态管理
- `axios@^1.6.0` - HTTP 客户端
- `@vueuse/core@^10.7.0` - Vue 组合式 API 工具集

#### 可视化库
- `@antv/g6@^4.8.0` - 图可视化引擎 (网络拓扑)
- `echarts@^5.4.3` - Apache ECharts 图表库
- `vue-echarts@^6.6.0` - ECharts Vue 组件
- `three@^0.160.0` - Three.js 3D 图形库 ⭐ **新增**

#### 其他工具
- `dayjs@^1.11.10` - 日期时间处理
- `countup.js@^2.8.0` - 数字动画

## 🚀 运行项目

### 开发模式 (Mock 数据)

项目默认配置为 Mock 模式,无需后端即可运行:

```bash
npm run dev
```

浏览器会自动打开 `http://localhost:5173`

### 连接真实后端

如需连接真实后端 API:

1. 修改 `.env.development` 文件:
```env
# 关闭 Mock 模式
VITE_USE_MOCK=false

# 配置后端 API 地址 (如果不是 localhost:8080)
VITE_API_BASE_URL=http://your-backend-host:8080
```

2. 确保后端服务已启动:
```bash
# 在项目根目录
cd /home/ldc/workspace/pot
make run_server
```

3. 启动前端:
```bash
cd web
npm run dev
```

## 🏗️ 构建生产版本

```bash
npm run build
```

构建产物会输出到 `dist/` 目录。

## 🎨 主要功能

### 1. 自适应布局
- 无边缘留白的全屏布局
- 响应式设计适配不同屏幕尺寸
- 优化的顶部状态栏

### 2. 三维可视化效果
- Three.js 粒子背景动画
- 2000+ 动态粒子效果
- POT 主题色 (蓝、金、紫)

### 3. 网络拓扑图
- 使用 @antv/g6 渲染网络拓扑
- 双层布局 (委员会层 + 普通节点层)
- 交互式节点和边

### 4. 区块链动画
- 底部横向滚动区块链
- 新区块动画效果
- 点击查看区块详情

### 5. 实时监控
- POT 共识状态
- VDF 证明进度
- 内存池监控
- BCI 激励机制
- 存储状态
- 性能指标
- 委员会信息
- 系统信息

### 6. Mock 数据模式
- 完整的模拟数据生成器
- 支持前端独立开发和演示
- 自动递增的区块高度
- 模拟真实数据波动

## 🐛 常见问题

### 1. 安装依赖失败

如果 npm install 失败,可能是网络问题,建议使用国内镜像:

```bash
# 使用淘宝镜像
npm config set registry https://registry.npmmirror.com

# 重新安装
npm install
```

### 2. TypeScript 类型错误

安装依赖后,TypeScript 错误会自动消失。如果仍有问题:

```bash
# 重新安装类型定义
npm install --save-dev @types/three
```

### 3. 端口被占用

如果 5173 端口被占用,Vite 会自动选择下一个可用端口。你也可以手动指定:

```bash
npm run dev -- --port 3000
```

### 4. Mock 模式无法切换

确保 `.env.development` 文件中的环境变量正确:
```env
VITE_USE_MOCK=true  # Mock 模式
VITE_USE_MOCK=false # 真实后端
```

修改后需要重启开发服务器。

## 📁 项目结构

```
web/
├── public/              # 静态资源
├── src/
│   ├── assets/          # 资产文件
│   ├── components/      # Vue 组件
│   │   ├── blocks/      # 功能块组件
│   │   │   ├── BlockChain.vue      # 区块链动画 ⭐
│   │   │   ├── CommitteeInfo.vue   # 委员会信息 ⭐
│   │   │   ├── SystemInfo.vue      # 系统信息 ⭐
│   │   │   └── ...
│   │   ├── layout/      # 布局组件
│   │   └── ThreeBackground.vue    # 3D 背景 ⭐
│   ├── services/        # API 服务
│   │   ├── api.ts       # HTTP API
│   │   ├── mock.ts      # Mock 数据 ⭐
│   │   └── websocket.ts # WebSocket
│   ├── stores/          # Pinia 状态管理
│   ├── types/           # TypeScript 类型
│   ├── utils/           # 工具函数
│   ├── App.vue          # 根组件
│   └── main.ts          # 入口文件
├── .env.development     # 开发环境配置
├── package.json         # 依赖配置
├── vite.config.ts       # Vite 配置
└── tsconfig.json        # TypeScript 配置
```

⭐ 标记为最新添加/修改的文件

## 🎯 下一步

1. 运行 `npm install` 安装依赖
2. 运行 `npm run dev` 启动开发服务器
3. 在浏览器中查看效果
4. 根据需要调整配置

## 📝 更新日志

### 最新更新 (2024)

- ✅ 添加 Three.js 3D 粒子背景
- ✅ 创建 BlockChain.vue 区块链动画组件
- ✅ 创建 CommitteeInfo.vue 委员会信息组件
- ✅ 创建 SystemInfo.vue 系统信息组件
- ✅ 实现完整的 Mock 数据服务
- ✅ 优化顶部状态栏布局
- ✅ 重构主布局以支持底部区块链展示
- ✅ 支持 Mock/真实后端无缝切换

## 📞 技术支持

如有问题,请查看:
- [项目文档](./docs/)
- [快速开始指南](./QUICKSTART.md)
- [开发指南](./DEVELOPMENT.md)
- [项目概要](./PROJECT_SUMMARY.md)
