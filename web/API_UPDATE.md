# API接口更新说明

本次更新对POT共识可视化系统的数据接口进行了全面优化和清理。

## 更新内容

### 1. 类型定义优化 (`src/types/api.ts`)

**保留的核心类型**:
- ✅ `SystemOverview` - 系统概览
- ✅ `POTStatus` - POT共识状态
- ✅ `VDFStatus` - VDF计算状态
- ✅ `CommitteeStatus` - 委员会状态
- ✅ `BCIStatus` - BCI激励状态
- ✅ `MempoolStatus` - 交易池状态
- ✅ `NetworkTopology` - 网络拓扑
- ✅ `WSMessage` - WebSocket消息

**移除的类型**:
- ❌ `StorageStatus` - 当前可视化未使用
- ❌ `MetricsHistory` - 当前可视化未使用

**改进**:
- 添加了详细的JSDoc注释
- 所有枚举类型使用TypeScript字面量类型（如 `'healthy' | 'warning' | 'error'`）
- 统一字段描述和单位说明
- 优化了数据结构的一致性

### 2. API服务优化 (`src/services/api.ts`)

**保留的API方法**:
- ✅ `getSystemOverview()` - 获取系统概览
- ✅ `getPotStatus()` - 获取POT状态
- ✅ `getVDFStatus()` - 获取VDF状态
- ✅ `getCommitteeStatus()` - 获取委员会状态
- ✅ `getBCIStatus()` - 获取BCI激励状态
- ✅ `getMempoolStatus()` - 获取交易池状态
- ✅ `getNetworkTopology()` - 获取网络拓扑

**移除的API方法**:
- ❌ `getStorageStatus()` - 未在可视化中使用
- ❌ `getMetricsHistory()` - 未在可视化中使用
- ❌ `getBalance()` - 未在可视化中使用

**改进**:
- 添加了完整的JSDoc注释和返回类型
- 统一的错误处理机制
- 更清晰的代码组织结构
- 明确的泛型类型定义

### 3. Mock数据优化 (`src/services/mock.ts`)

**改进**:
- 所有方法添加了JSDoc注释
- 修复了类型安全问题（使用类型断言和const断言）
- 优化了随机数据生成逻辑
- 确保Mock数据与TypeScript类型定义完全一致

### 4. OpenAPI规范 (`openapi.json`)

生成了完整的OpenAPI 3.0.3规范文件，包含：
- 详细的接口描述和参数说明
- 完整的Schema定义
- 请求和响应示例
- 错误处理规范
- 服务器配置

### 5. API文档 (`API.md`)

生成了人类可读的Markdown文档，包含：
- 清晰的接口分类
- 详细的字段说明
- 完整的请求/响应示例
- WebSocket使用指南
- 开发建议和最佳实践

## 使用说明

### 查看API文档

1. **OpenAPI规范**: 可以使用Swagger UI或Redoc查看 `openapi.json`
   ```bash
   # 使用npx快速启动Swagger UI
   npx swagger-ui-watcher openapi.json
   ```

2. **Markdown文档**: 直接查看 `API.md` 文件

### Mock模式

在开发环境中使用Mock数据：

```bash
# .env 文件
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080/api
```

### 生产模式

连接真实API服务：

```bash
# .env 文件
VITE_USE_MOCK=false
VITE_API_BASE_URL=https://api.pot.io
```

## 数据刷新策略

根据可视化需求，建议的数据轮询频率：

| 数据类型 | 刷新频率 | 接口 |
|---------|---------|------|
| 系统概览 | 1秒 | `GET /system/overview` |
| VDF状态 | 1秒 | `GET /pot/vdf` |
| POT状态 | 5秒 | `GET /pot/status` |
| 委员会状态 | 5秒 | `GET /committee/status` |
| 交易池状态 | 5秒 | `GET /mempool/status` |
| 网络拓扑 | 10秒 | `GET /network/topology` |
| BCI激励 | 10秒 | `GET /bci/status` |

## 文件清单

```
web/
├── src/
│   ├── types/
│   │   └── api.ts          # ✨ 优化的类型定义
│   ├── services/
│   │   ├── api.ts          # ✨ 优化的API客户端
│   │   └── mock.ts         # ✨ 优化的Mock数据服务
│   └── ...
├── openapi.json            # 🆕 OpenAPI 3.0规范
├── API.md                  # 🆕 API文档
└── API_UPDATE.md          # 📄 本文档
```

## 类型安全

所有接口都有完整的TypeScript类型支持：

```typescript
import { api } from '@/services/api'
import type { SystemOverview, POTStatus } from '@/types/api'

// 类型安全的API调用
const overview: SystemOverview = await api.getSystemOverview()
const potStatus: POTStatus = await api.getPotStatus()
```

## WebSocket支持

系统支持WebSocket实时推送：

```typescript
import { WebSocketService } from '@/services/websocket'

const wsService = new WebSocketService('ws://localhost:8080/api/ws')
wsService.connect()

// 订阅感兴趣的主题
wsService.subscribe(['pot', 'vdf', 'system'])

// 监听数据更新
wsService.on('pot', (data: POTStatus) => {
  console.log('POT状态更新:', data)
})
```

## 后端实现建议

后端开发者可以参考 `openapi.json` 规范实现对应的API接口：

1. 使用OpenAPI Generator生成服务端代码骨架
2. 实现各个接口的业务逻辑
3. 确保返回的数据格式与类型定义一致
4. 实现WebSocket推送服务

## 测试

所有类型定义和API服务都已通过TypeScript类型检查：

```bash
# 类型检查
npm run type-check

# 构建测试
npm run build
```

## 注意事项

1. **向后兼容**: 移除的接口（StorageStatus、MetricsHistory）如需恢复，可参考git历史
2. **类型严格**: 所有枚举值都使用字面量类型，确保类型安全
3. **Mock数据**: Mock服务提供的数据会随机变化，用于演示动态效果
4. **WebSocket**: WebSocket连接失败时会自动降级到轮询模式

## 更新日期

2024-11-24

---

如有问题或建议，请联系开发团队。
