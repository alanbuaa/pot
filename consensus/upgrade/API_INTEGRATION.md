# Phase 6 HTTP API 集成

## 概览

Phase 6 的 HTTP API 已成功集成到现有的 `internal/apis` 结构中，遵循项目现有的 API 架构模式。

## 架构设计

### 文件结构

```
internal/apis/
├── apis.go                          # API 服务器主文件
├── handlers/
│   ├── pot_handler.go              # PoT 共识处理器（现有）
│   ├── pow_handler.go              # PoW 共识处理器（现有）
│   ├── monitor_handler.go          # 监控处理器（现有）
│   └── upgrade_handler.go          # 升级共识处理器（新增）
└── model/
    ├── common.go                    # 通用模型（现有）
    ├── pot_model.go                # PoT 模型（现有）
    ├── pow_model.go                # PoW 模型（现有）
    ├── monitor_model.go            # 监控模型（现有）
    └── upgrade_model.go            # 升级模型（新增）

consensus/upgrade/
└── service_adapter.go              # 服务适配器（新增）
```

### 设计模式

遵循现有的三层架构：

1. **服务接口层** (`model/upgrade_model.go`)
   - `UpgradeService` 接口定义
   - 请求/响应数据结构
   - 数据转换函数

2. **处理器层** (`handlers/upgrade_handler.go`)
   - `UpgradeHandler` 实现 `ConsensusHandler` 接口
   - 依赖注入 `UpgradeService` 接口
   - 实现 14 个 HTTP 端点

3. **服务适配器层** (`consensus/upgrade/service_adapter.go`)
   - `UpgradeServiceAdapter` 将 `UpgradeManager` 适配为 `UpgradeService`
   - 解耦 handler 和具体实现

## API 端点

所有端点前缀：`/api`

### 提案管理
- `POST /upgrade/propose` - 创建升级提案
- `GET /upgrade/proposals` - 列出所有提案
- `GET /upgrade/proposals/:id` - 获取特定提案

### 升级控制
- `POST /upgrade/start` - 启动升级流程（待完善）
- `POST /upgrade/rollback` - 回滚升级
- `GET /upgrade/status` - 获取升级状态
- `GET /upgrade/phase` - 获取当前阶段

### CDL 操作
- `POST /cdl/validate` - 验证 CDL 配置
- `POST /cdl/compile` - 编译 CDL

### 指标和监控
- `GET /metrics/current` - 获取当前性能指标
- `GET /metrics/history` - 获取历史指标
- `GET /events` - 查询升级事件
- `GET /health` - 健康检查

## 集成步骤

### 1. 定义服务接口 (internal/apis/model/upgrade_model.go)

```go
type UpgradeService interface {
    GetUpgradeManager() *upgrade.UpgradeManager
}
```

### 2. 创建处理器 (internal/apis/handlers/upgrade_handler.go)

```go
type UpgradeHandler struct {
    service model.UpgradeService
    log     *logrus.Entry
}

func (h *UpgradeHandler) RegisterRoutes(group *gin.RouterGroup) {
    // 注册所有路由
}
```

### 3. 创建服务适配器 (consensus/upgrade/service_adapter.go)

```go
type UpgradeServiceAdapter struct {
    manager *UpgradeManager
}

func (a *UpgradeServiceAdapter) GetUpgradeManager() *UpgradeManager {
    return a.manager
}
```

### 4. 注册服务 (internal/apis/apis.go)

```go
func (s *ApiServer) RegisterUpgradeService(service model.UpgradeService) {
    handler := handlers.NewUpgradeHandler(service, s.log)
    s.handlers = append(s.handlers, handler)
}
```

### 5. 添加 GetPersistence 方法 (consensus/upgrade/manager.go)

```go
func (m *UpgradeManager) GetPersistence() UpgradePersistence {
    return m.persistence
}
```

## 使用示例

### 启动服务器时注册升级服务

```go
// 创建 API 服务器
apiServer := apis.NewApiServer(&apis.Config{Port: 8080}, log)

// 创建升级管理器（带持久化）
dbPath := "data/upgrade.db"
persistence, err := upgrade.NewBoltDBPersistence(dbPath)
if err != nil {
    log.Fatal(err)
}

upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
    currentConsensus,
    config,
    storage,
    persistence,
    log,
)
if err != nil {
    log.Fatal(err)
}

// 创建适配器并注册
adapter := upgrade.NewUpgradeServiceAdapter(upgradeManager)
apiServer.RegisterUpgradeService(adapter)

// 启动服务器
apiServer.Start()
```

### 创建升级提案

```bash
curl -X POST http://localhost:8080/api/upgrade/propose \
  -H "Content-Type: application/json" \
  -d '{
    "target_consensus": "hotstuff",
    "preexec_start_height": 100,
    "switch_height": 200,
    "description": "Upgrade to HotStuff consensus",
    "cdl_yaml": "name: hotstuff\ntype: consensus\nversion: 1.0.0"
  }'
```

### 查询升级状态

```bash
curl http://localhost:8080/api/upgrade/status
```

### 查询事件

```bash
curl http://localhost:8080/api/events?limit=10
```

## 数据模型

### ProposeUpgradeRequest

```go
type ProposeUpgradeRequest struct {
    TargetConsensus    string                 `json:"target_consensus" binding:"required"`
    CDLYaml            string                 `json:"cdl_yaml"`
    ForkHeight         uint64                 `json:"fork_height"`
    PreexecStartHeight uint64                 `json:"preexec_start_height" binding:"required"`
    SwitchHeight       uint64                 `json:"switch_height" binding:"required"`
    Incentive          uint64                 `json:"incentive"`
    Description        string                 `json:"description"`
    ConsensusParams    map[string]interface{} `json:"consensus_params"`
}
```

### UpgradeStatusResponse

```go
type UpgradeStatusResponse struct {
    Phase           string                 `json:"phase"`
    Started         bool                   `json:"started"`
    Completed       bool                   `json:"completed"`
    Failed          bool                   `json:"failed"`
    StartTime       *int64                 `json:"start_time,omitempty"`
    EndTime         *int64                 `json:"end_time,omitempty"`
    FailureReason   string                 `json:"failure_reason,omitempty"`
    CurrentProposal *ProposalSummary       `json:"current_proposal,omitempty"`
    Metrics         *PerformanceMetrics    `json:"metrics,omitempty"`
}
```

## 注意事项

### 待完善功能

1. **StartUpgrade 端点**：目前返回 501 Not Implemented
   - 原因：需要共识工厂来根据 `target_consensus` 创建新共识实例
   - 计划：实现 ConsensusFactory 后完善

2. **Rollback 实现**：目前仅调用 `Reset()`
   - 原因：RollbackManager 尚未实现
   - 计划：Phase 7 实现完整回滚逻辑

### 类型转换

- ProposalID 支持两种格式：
  - UUID 格式（用户友好）
  - 32字节十六进制（内部存储格式）
  
- 时间戳统一使用 Unix 时间戳（秒）

### 错误处理

所有端点遵循统一的响应格式：

```json
{
  "code": 200,
  "msg": "success",
  "data": { ... }
}
```

错误响应：

```json
{
  "code": 400,
  "msg": "Invalid request: ...",
  "data": null
}
```

## 测试建议

1. 单元测试（待实现）
   - 为每个 handler 方法编写测试
   - Mock UpgradeService 接口
   - 测试各种错误情况

2. 集成测试（待实现）
   - 启动完整的 API 服务器
   - 使用真实的 UpgradeManager
   - 测试端到端流程

3. 手动测试
   - 使用 curl 或 Postman 测试各个端点
   - 验证持久化是否正常工作
   - 检查日志输出

## 与现有 API 的一致性

✅ 遵循 ConsensusHandler 接口模式  
✅ 使用标准 ResponseData 结构  
✅ 依赖注入服务接口而非具体实现  
✅ 使用 logrus.Entry 进行日志记录  
✅ CORS 支持（继承自 ApiServer）  
✅ 路由分组（/api 前缀）  

## 下一步

1. 实现 ConsensusFactory 以支持动态创建共识实例
2. 完善 RollbackManager 实现
3. 编写 API 测试
4. 添加 API 文档（Swagger/OpenAPI）
5. 实现 CLI 工具以便命令行操作
