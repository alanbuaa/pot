# POT可视化监控API使用指南

## 概述

POT可视化监控系统提供了一套完整的RESTful API接口，用于实时监控POT共识系统的运行状态。本指南介绍如何在POT节点中集成和使用这些监控接口。

## 架构说明

监控系统采用分层架构：

```
handlers/monitor_handler.go  ← HTTP请求处理层
         ↓
model/monitor_model.go        ← 数据模型定义
         ↓
pot_monitor.go                ← 监控逻辑实现
         ↓
pot_monitor_adapter.go        ← 适配器层
         ↓
consensus/pot/worker.go       ← POT共识核心
consensus/pot/engine.go
```

## 集成步骤

### 1. 在PoT引擎启动时注册监控服务

在你的POT节点启动代码中（通常在 `cmd/server/server.go` 或类似文件中）：

```go
import (
    "github.com/zzz136454872/upgradeable-consensus/consensus/pot"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis"
)

// 创建POT引擎
potEngine := pot.NewPoTEngine(nodeID, consensusID, config, executor, adaptor, logger)

// 创建API服务器
apiConfig := &apis.Config{
    Port: 8080, // 监听端口
}
apiServer := apis.NewApiServer(apiConfig, logger)

// 注册POT共识服务（用于交易提交等）
potService := apis.NewPotWorkerAdapter(potEngine.Worker)
apiServer.RegisterPotService(potService)

// 注册监控服务（用于可视化）
monitorService := apis.NewPotMonitorAdapter(potEngine, logger)
apiServer.RegisterMonitorService(monitorService)

// 启动API服务器
if err := apiServer.Start(); err != nil {
    logger.Fatalf("Failed to start API server: %v", err)
}
```

### 2. 启动节点

启动节点后，API服务器将在配置的端口上监听HTTP请求。

```bash
make run_server
```

### 3. 测试API接口

使用curl或任何HTTP客户端测试接口：

```bash
# 获取系统概览
curl http://localhost:8080/api/system/overview

# 获取POT共识状态
curl http://localhost:8080/api/pot/status

# 获取VDF计算状态
curl http://localhost:8080/api/pot/vdf

# 获取委员会状态
curl http://localhost:8080/api/committee/status

# 获取BCI激励状态
curl http://localhost:8080/api/bci/status

# 获取交易池状态
curl http://localhost:8080/api/mempool/status

# 获取网络拓扑
curl http://localhost:8080/api/network/topology
```

## API端点说明

### 1. 系统概览
**GET** `/api/system/overview`

返回系统运行时长、节点数量、当前高度等整体信息。

### 2. POT共识状态
**GET** `/api/pot/status`

返回当前epoch、难度值、工作状态、nonce等POT共识相关信息。

### 3. VDF计算状态
**GET** `/api/pot/vdf`

返回VDF0、VDF1、VDFHalf的计算进度、迭代次数、状态等详细信息。

### 4. 委员会状态
**GET** `/api/committee/status`

返回委员会成员、分片信息、工作阶段等委员会共识信息。

### 5. BCI激励状态
**GET** `/api/bci/status`

返回奖励分配、锁定期利率、待处理奖励等BCI激励系统信息。

### 6. 交易池状态
**GET** `/api/mempool/status`

返回交易池大小、已提议/未提议交易数、最近交易等信息。

### 7. 网络拓扑
**GET** `/api/network/topology`

返回节点列表、连接关系、P2P网络信息等。

## 前端集成

前端可以使用这些API接口来实现可视化界面：

```typescript
// 示例：获取POT状态
async function getPOTStatus() {
  const response = await fetch('http://localhost:8080/api/pot/status');
  const data = await response.json();
  console.log('POT Status:', data);
  return data;
}

// 示例：轮询获取VDF状态
setInterval(async () => {
  const vdfStatus = await fetch('http://localhost:8080/api/pot/vdf')
    .then(res => res.json());
  updateVDFVisualization(vdfStatus);
}, 1000); // 每秒更新一次
```

## CORS配置

API服务器已经配置了CORS支持，允许跨域请求。前端可以直接从浏览器调用这些API。

## 扩展功能

### 添加新的监控指标

1. 在 `model/monitor_model.go` 中定义新的数据结构
2. 在 `MonitorService` 接口中添加新方法
3. 在 `pot_monitor.go` 中实现该方法
4. 在 `handlers/monitor_handler.go` 中添加HTTP处理器
5. 在 `RegisterRoutes` 中注册新路由

### 示例：添加区块统计接口

```go
// 1. 在 monitor_model.go 中添加
type BlockStats struct {
    TotalBlocks     uint64  `json:"totalBlocks"`
    AvgBlockSize    int64   `json:"avgBlockSize"`
    AvgTxsPerBlock  float64 `json:"avgTxsPerBlock"`
}

// 2. 在 MonitorService 接口中添加
type MonitorService interface {
    // ... 现有方法
    GetBlockStats() (*BlockStats, error)
}

// 3. 在 pot_monitor.go 中实现
func (m *PotMonitor) GetBlockStats() (*model.BlockStats, error) {
    // 实现逻辑
    return &model.BlockStats{
        TotalBlocks:    m.worker.GetChainReader().GetCurrentHeight(),
        AvgBlockSize:   1024,
        AvgTxsPerBlock: 10.5,
    }, nil
}

// 4. 在 monitor_handler.go 中添加
func (h *MonitorHandler) HandleBlockStats(c *gin.Context) {
    stats, err := h.service.GetBlockStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, stats)
}

// 5. 注册路由
func (h *MonitorHandler) RegisterRoutes(group *gin.RouterGroup) {
    // ... 现有路由
    group.GET("/block/stats", h.HandleBlockStats)
}
```

## 性能优化建议

1. **缓存频繁访问的数据**：对于变化不频繁的数据（如网络拓扑），可以在monitor中添加缓存机制
2. **批量查询**：前端应该合理控制轮询频率，避免过度请求
3. **使用WebSocket**：对于需要实时更新的数据，考虑实现WebSocket推送机制

## 故障排查

### API返回500错误
- 检查POT引擎是否正常运行
- 查看日志中的错误信息
- 确认Worker和Engine的getter方法返回有效数据

### 数据不准确
- VDF进度和状态是基于当前状态的快照
- 某些统计数据（如平均值）可能需要历史数据积累
- 检查Worker的锁机制是否正常工作

### CORS错误
- 确认API服务器的CORS中间件已正确配置
- 检查前端请求的URL是否正确
- 查看浏览器控制台的详细错误信息

## 相关文件

- `internal/apis/apis.go` - API服务器主入口
- `internal/apis/pot_monitor.go` - POT监控逻辑实现
- `internal/apis/pot_monitor_adapter.go` - 监控服务适配器
- `internal/apis/model/monitor_model.go` - 监控数据模型
- `internal/apis/handlers/monitor_handler.go` - HTTP请求处理器
- `consensus/pot/worker.go` - POT Worker（添加了getter方法）
- `consensus/pot/engine.go` - POT Engine（添加了getter方法）
- `consensus/pot/mempool.go` - 交易池（添加了统计方法）

## 更多信息

详细的API文档请参考：`web/API.md`
