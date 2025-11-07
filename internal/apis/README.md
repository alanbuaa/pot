# APIs Module - Refactored Architecture

## 概述

APIs 模块已重构为支持多种共识机制的灵活架构。新架构通过接口和适配器模式，允许轻松集成和扩展不同的共识算法。

## 目录结构

```
internal/apis/
├── apis.go              # API 服务器主文件，支持多共识
├── pot_adapter.go       # PoT Worker 适配器
├── USAGE.go            # 使用示例和文档
├── README.md           # 本文件
├── model/              # 数据模型和接口定义
│   ├── common.go       # 通用数据结构和转换函数
│   ├── pot_model.go    # PoT 共识服务接口
│   └── pow_model.go    # PoW 共识服务接口（预留）
└── handlers/           # HTTP 请求处理器
    ├── pot_handler.go  # PoT 共识请求处理
    └── pow_handler.go  # PoW 共识请求处理（预留）
```

## 核心组件

### 1. ApiServer (`apis.go`)

`ApiServer` 是 HTTP API 服务器的核心，负责：
- 管理多个共识处理器
- 配置路由和中间件
- 启动和停止 HTTP 服务

```go
type ApiServer struct {
    engine   *gin.Engine
    config   *Config
    handlers []ConsensusHandler
    log      *logrus.Entry
}
```

### 2. 数据模型 (`model/`)

#### `common.go`
定义通用的数据结构和转换函数：
- `HTTPTransaction` - HTTP API 格式的交易
- `HTTPTxInput` - 交易输入
- `HTTPTxOutput` - 交易输出
- `RequestData` - 通用请求包装
- `ResponseData` - 通用响应包装
- `HttpTx2Tx()` - HTTP 交易转换为内部格式

#### `pot_model.go`
定义 PoT 共识服务接口：
```go
type PotConsensusService interface {
    CheckLockTransaction(rawtx *types.RawTx) error
    CheckLockTransferTransaction(rawtx *types.RawTx) error
    CheckNonLockTransferTransaction(rawtx *types.RawTx) error
    CheckDevastateTransaction(rawtx *types.RawTx) error
    BroadcastClientTransaction(rawtx *types.RawTx, txType pb.TxType) error
    AddRawTxToMempool(rawtx *types.RawTx)
    GetCurrentHeight() uint64
}
```

#### `pow_model.go`
定义 PoW 共识服务接口（预留扩展）

### 3. 请求处理器 (`handlers/`)

#### `pot_handler.go`
处理 PoT 特定的 HTTP 请求：
- `HandleCreateLockTransaction` - 创建锁定交易
- `HandleLockTransferTransaction` - 锁定转账交易
- `HandleNonLockTransferTransaction` - 非锁定转账交易
- `HandleDevastateTransaction` - 销毁交易
- `HandleGetBlockHeight` - 获取区块高度
- `HandleHello` - 健康检查

#### `pow_handler.go`
处理 PoW 特定的 HTTP 请求（预留扩展）

### 4. 适配器 (`pot_adapter.go`)

`PotWorkerAdapter` 将 `pot.Worker` 适配为 `PotConsensusService` 接口：
- 提供对 Worker 私有方法的公开访问
- 统一接口调用方式
- 解耦 API 层和共识层

## 使用方法

### 基本使用

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/pot"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis"
)

func main() {
    logger := logrus.New()
    log := logger.WithField("module", "api")

    // 创建 API 服务器
    config := &apis.Config{Port: 18025}
    apiServer := apis.NewApiServer(config, log)

    // 创建并注册 PoT 服务
    potWorker := pot.NewWorker(...) // 您的 PoT Worker 实例
    potService := apis.NewPotWorkerAdapter(potWorker)
    apiServer.RegisterPotService(potService)

    // 启动服务器
    apiServer.Start()
}
```

### 支持多种共识

```go
// 注册 PoT
potService := apis.NewPotWorkerAdapter(potWorker)
apiServer.RegisterPotService(potService)

// 注册 PoW（未来）
// powService := apis.NewPowWorkerAdapter(powWorker)
// apiServer.RegisterPowService(powService)

// 两种共识的 API 都可用
apiServer.Start()
```

## API 端点

### PoT 共识端点

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/createlocktransaction` | 创建锁定交易 |
| POST | `/api/locktransfertransaction` | 创建锁定转账交易 |
| POST | `/api/nonlocktransfertransaction` | 创建非锁定转账交易 |
| POST | `/api/devastatetransaction` | 创建销毁交易 |
| GET | `/api/getblockheight` | 获取当前区块高度 |
| POST | `/api/hello` | 健康检查 |

### 请求格式示例

```json
{
    "transaction": {
        "txid": "0x...",
        "txInputs": [{
            "txid": "0x...",
            "voutput": "0",
            "scriptSig": "0x...",
            "value": "1000",
            "address": "0x...",
            "bciType": "0"
        }],
        "txOutputs": [{
            "address": "0x...",
            "value": "900",
            "interest": "0",
            "proof": "0x...",
            "lockTime": "0",
            "bciType": "0",
            "data": "",
            "burnLock": "0",
            "rate": "0.0"
        }],
        "transactionFee": "100"
    },
    "type": "createlock"
}
```

### 响应格式

```json
{
    "code": 200,
    "msg": "success"
}
```

## 架构优势

### 1. 关注点分离
- **Model**: 定义数据结构和接口
- **Handlers**: 处理 HTTP 请求逻辑
- **Adapters**: 适配不同的共识实现
- **ApiServer**: 协调路由和服务

### 2. 可扩展性
- 通过实现接口轻松添加新的共识机制
- 新增共识类型无需修改现有代码
- 支持多个共识同时运行

### 3. 可测试性
- Handler 可以用 Mock 服务独立测试
- 接口使单元测试更简单
- 适配器模式便于集成测试

### 4. 类型安全
- Go 接口提供强类型检查
- 编译时验证接口实现
- 减少运行时错误

## 添加新的共识机制

### 步骤 1: 定义服务接口

在 `model/your_consensus_model.go` 中：

```go
type YourConsensusService interface {
    ValidateTransaction(tx *types.Tx) error
    BroadcastTransaction(tx *types.Tx) error
    GetCurrentHeight() uint64
}
```

### 步骤 2: 实现处理器

在 `handlers/your_consensus_handler.go` 中：

```go
type YourConsensusHandler struct {
    service model.YourConsensusService
    log     *logrus.Entry
}

func (h *YourConsensusHandler) RegisterRoutes(group *gin.RouterGroup) {
    group.POST("/yourEndpoint", h.HandleYourEndpoint)
}
```

### 步骤 3: 添加注册方法

在 `apis.go` 中添加：

```go
func (s *ApiServer) RegisterYourConsensus(service model.YourConsensusService) {
    handler := handlers.NewYourConsensusHandler(service, s.log)
    s.handlers = append(s.handlers, handler)
}
```

### 步骤 4: 创建适配器（如需要）

```go
type YourWorkerAdapter struct {
    worker *yourpkg.Worker
}

func (a *YourWorkerAdapter) ValidateTransaction(tx *types.Tx) error {
    return a.worker.Validate(tx)
}
```

## 迁移指南

### 从旧架构迁移

**旧代码：**
```go
worker := pot.NewWorker(...)
// 直接使用 worker 内部的 HTTP 服务
```

**新代码：**
```go
// 创建 Worker
worker := pot.NewWorker(...)

// 创建 API 服务器
apiServer := apis.NewApiServer(config, log)
potService := apis.NewPotWorkerAdapter(worker)
apiServer.RegisterPotService(potService)
apiServer.Start()
```

## 维护和最佳实践

1. **保持接口小而专注** - 每个接口应只包含必要的方法
2. **使用适配器解耦** - 不要让 API 层直接依赖共识实现
3. **统一错误处理** - 在 handler 层统一处理和记录错误
4. **添加日志** - 为调试和监控添加详细的日志
5. **编写测试** - 为每个 handler 和 adapter 编写单元测试

## 未来改进

- [ ] 添加 WebSocket 支持实时通知
- [ ] 实现 API 请求限流和认证
- [ ] 添加 OpenAPI/Swagger 文档
- [ ] 实现优雅关闭
- [ ] 添加 Prometheus 指标导出
- [ ] 支持更多共识机制（PoW, PBFT, 等）

## 相关文件

- 完整使用示例：`USAGE.go`
- PoT Worker 实现：`consensus/pot/worker.go`
- HTTP 协议文档：`docs/http-api-documentation.md`
- API 测试：`docs/rest-client-tests.http`
