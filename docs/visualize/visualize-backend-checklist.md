# 可视化后端开发检查清单

本文档提供后端开发的详细检查清单和快速参考。

## 一、开发优先级

### 阶段1: 核心API实现（优先级：高）
- [ ] `GET /api/system/overview` - 系统概览
- [ ] `GET /api/pot/status` - POT状态
- [ ] `GET /api/pot/vdf` - VDF状态
- [ ] `GET /api/mempool/status` - 交易池状态

### 阶段2: 扩展API实现（优先级：中）
- [ ] `GET /api/committee/status` - 委员会状态
- [ ] `GET /api/bci/status` - BCI激励状态
- [ ] `GET /api/network/topology` - 网络拓扑
- [ ] `GET /api/storage/status` - 存储状态

### 阶段3: 高级功能（优先级：低）
- [ ] `GET /api/bci/balance/:address` - 地址余额查询
- [ ] `GET /api/metrics/history` - 历史数据
- [ ] WebSocket实时推送
- [ ] Metrics采集器

---

## 二、目录结构创建

```bash
mkdir -p internal/apis/visualization/{handlers,services,models,middleware,websocket,metrics}
```

需要创建的文件：
```
internal/apis/visualization/
├── handlers/
│   ├── system_handler.go
│   ├── pot_handler.go
│   ├── committee_handler.go
│   ├── bci_handler.go
│   ├── mempool_handler.go
│   ├── network_handler.go
│   ├── storage_handler.go
│   └── websocket_handler.go
├── services/
│   ├── system_service.go
│   ├── pot_service.go
│   ├── committee_service.go
│   ├── bci_service.go
│   ├── mempool_service.go
│   ├── network_service.go
│   ├── storage_service.go
│   └── metrics_service.go
├── models/
│   ├── response_models.go
│   └── request_models.go
├── middleware/
│   ├── auth.go
│   ├── ratelimit.go
│   └── cors.go
├── websocket/
│   ├── hub.go
│   └── client.go
├── metrics/
│   ├── collector.go
│   └── storage.go
└── server.go
```

---

## 三、核心代码模板

### 3.1 Response Models (`models/response_models.go`)

```go
package models

import "time"

// SystemOverview 系统概览
type SystemOverview struct {
    Uptime             float64       `json:"uptime"`
    TotalNodes         int           `json:"totalNodes"`
    OnlineNodes        int           `json:"onlineNodes"`
    ConsensusTypes     []string      `json:"consensusTypes"`
    NetworkStatus      string        `json:"networkStatus"`
    NetworkUtilization float64       `json:"networkUtilization"`
    ExecutorStatus     string        `json:"executorStatus"`
    StorageUsage       StorageUsage  `json:"storageUsage"`
    MempoolSize        int           `json:"mempoolSize"`
    CurrentTPS         float64       `json:"currentTPS"`
    CurrentHeight      int64         `json:"currentHeight"`
    AvgBlockTime       float64       `json:"avgBlockTime"`
    LastBlockTime      time.Time     `json:"lastBlockTime"`
}

type StorageUsage struct {
    Total      int64   `json:"total"`
    Used       int64   `json:"used"`
    Percentage float64 `json:"percentage"`
}

// POTStatus POT共识状态
type POTStatus struct {
    ConsensusType      string    `json:"consensusType"`
    Epoch              uint64    `json:"epoch"`
    Difficulty         string    `json:"difficulty"`
    CurrentHeight      int64     `json:"currentHeight"`
    WorkFlag           bool      `json:"workFlag"`
    Timestamp          time.Time `json:"timestamp"`
    Nonce              int64     `json:"nonce"`
    UncleCount         int       `json:"uncleCount"`
    AvgMiningTime      float64   `json:"avgMiningTime"`
    MiningSuccessRate  float64   `json:"miningSuccessRate"`
}

// VDFStatus VDF计算状态
type VDFStatus struct {
    VDF0            VDF0Status     `json:"vdf0"`
    VDF1            []VDF1Status   `json:"vdf1"`
    VDFHalf         VDFHalfStatus  `json:"vdfHalf"`
    VDFChecker      VDFCheckerStatus `json:"vdfChecker"`
    AbortStatus     bool           `json:"abortStatus"`
    AvgComputeTime  float64        `json:"avgComputeTime"`
    TotalIterations uint64         `json:"totalIterations"`
    CPUCounter      int            `json:"cpuCounter"`
}

type VDF0Status struct {
    Progress      float64 `json:"progress"`
    Iterations    uint64  `json:"iterations"`
    Status        string  `json:"status"`
    ChannelBuffer int     `json:"channelBuffer"`
}

type VDF1Status struct {
    WorkerID int     `json:"workerId"`
    Progress float64 `json:"progress"`
    Status   string  `json:"status"`
}

type VDFHalfStatus struct {
    Progress      float64 `json:"progress"`
    Status        string  `json:"status"`
    ChannelBuffer int     `json:"channelBuffer"`
}

type VDFCheckerStatus struct {
    Status          string `json:"status"`
    VerifyFailCount int    `json:"verifyFailCount"`
}

// CommitteeStatus 委员会状态
type CommitteeStatus struct {
    ConsensusType       string            `json:"consensusType"`
    CommitteeSize       uint64            `json:"committeeSize"`
    CommitteeCount      uint64            `json:"committeeCount"`
    ConfirmDelay        uint64            `json:"confirmDelay"`
    WorkHeight          uint64            `json:"workHeight"`
    BatchSize           int               `json:"batchSize"`
    InCommittee         bool              `json:"inCommittee"`
    Role                string            `json:"role"`
    Committee           []CommitteeMember `json:"committee"`
    CommitteePublicKey  string            `json:"committeePublicKey"`
    Shardings           []ShardingInfo    `json:"shardings"`
    WorkStage           string            `json:"workStage"`
    Timeout             int               `json:"timeout"`
    MessageQueueLength  int               `json:"messageQueueLength"`
    ElectionHeight      uint64            `json:"electionHeight"`
}

type CommitteeMember struct {
    Address   string `json:"address"`
    PublicKey string `json:"publicKey"`
    IsLeader  bool   `json:"isLeader"`
}

type ShardingInfo struct {
    Name             string `json:"name"`
    ID               int32  `json:"id"`
    LeaderAddress    string `json:"leaderAddress"`
    CommitteeMembers int    `json:"committeeMembers"`
    Status           string `json:"status"`
}

// 其他模型省略...
```

---

### 3.2 Service示例 (`services/pot_service.go`)

```go
package services

import (
    "time"
    
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/zzz136454872/upgradeable-consensus/consensus/pot"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization/models"
)

type POTService struct {
    worker *pot.Worker
}

func NewPOTService(worker *pot.Worker) *POTService {
    return &POTService{worker: worker}
}

func (s *POTService) GetStatus() (*models.POTStatus, error) {
    latestBlock := s.worker.chainReader.GetLatest()
    if latestBlock == nil {
        return nil, fmt.Errorf("no latest block found")
    }
    
    header := latestBlock.GetHeader()
    
    return &models.POTStatus{
        ConsensusType:     "POT",
        Epoch:             s.worker.epoch,
        Difficulty:        header.Difficulty.String(),
        CurrentHeight:     s.worker.Height,
        WorkFlag:          s.worker.workFlag,
        Timestamp:         s.worker.timestamp,
        Nonce:             header.Nonce,
        UncleCount:        len(header.UncleHash),
        AvgMiningTime:     s.calculateAvgMiningTime(),
        MiningSuccessRate: s.calculateSuccessRate(),
    }, nil
}

func (s *POTService) GetVDFStatus() (*models.VDFStatus, error) {
    vdf0Status := models.VDF0Status{
        Progress:      s.calculateVDF0Progress(),
        Iterations:    s.worker.vdf0.Iterations,
        Status:        s.getVDFStatusString(s.worker.vdf0),
        ChannelBuffer: len(s.worker.vdf0Chan),
    }
    
    vdf1Statuses := make([]models.VDF1Status, len(s.worker.vdf1))
    for i, vdf := range s.worker.vdf1 {
        vdf1Statuses[i] = models.VDF1Status{
            WorkerID: i,
            Progress: s.calculateVDF1Progress(vdf),
            Status:   s.getVDFStatusString(vdf),
        }
    }
    
    vdfHalfStatus := models.VDFHalfStatus{
        Progress:      s.calculateVDFHalfProgress(),
        Status:        s.getVDFStatusString(s.worker.vdfhalf),
        ChannelBuffer: len(s.worker.vdfhalfchan),
    }
    
    vdfCheckerStatus := models.VDFCheckerStatus{
        Status:          "ready", // 根据实际情况
        VerifyFailCount: 0,       // 需要添加计数器
    }
    
    return &models.VDFStatus{
        VDF0:            vdf0Status,
        VDF1:            vdf1Statuses,
        VDFHalf:         vdfHalfStatus,
        VDFChecker:      vdfCheckerStatus,
        AbortStatus:     s.worker.abort != nil && s.worker.abort.IsAborted(),
        AvgComputeTime:  s.calculateAvgComputeTime(),
        TotalIterations: s.getTotalIterations(),
        CPUCounter:      len(s.worker.vdf1),
    }, nil
}

// 辅助方法
func (s *POTService) calculateVDF0Progress() float64 {
    // TODO: 实现实际的进度计算逻辑
    // 可能需要在Worker中添加进度跟踪
    return 0.0
}

func (s *POTService) calculateVDF1Progress(vdf *types.VDF) float64 {
    // TODO: 实现
    return 0.0
}

func (s *POTService) calculateVDFHalfProgress() float64 {
    // TODO: 实现
    return 0.0
}

func (s *POTService) getVDFStatusString(vdf *types.VDF) string {
    if vdf == nil {
        return "idle"
    }
    // TODO: 根据实际状态返回
    return "computing"
}

func (s *POTService) calculateAvgMiningTime() float64 {
    // TODO: 需要在Worker中添加统计
    return 0.0
}

func (s *POTService) calculateSuccessRate() float64 {
    // TODO: 需要在Worker中添加统计
    return 0.0
}

func (s *POTService) calculateAvgComputeTime() float64 {
    // TODO: 实现
    return 0.0
}

func (s *POTService) getTotalIterations() uint64 {
    // TODO: 实现
    return 0
}
```

---

### 3.3 Handler示例 (`handlers/pot_handler.go`)

```go
package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization/services"
)

type POTHandler struct {
    service *services.POTService
    log     *logrus.Entry
}

func NewPOTHandler(service *services.POTService, log *logrus.Entry) *POTHandler {
    return &POTHandler{
        service: service,
        log:     log,
    }
}

func (h *POTHandler) RegisterRoutes(r *gin.RouterGroup) {
    pot := r.Group("/pot")
    {
        pot.GET("/status", h.GetStatus)
        pot.GET("/vdf", h.GetVDFStatus)
    }
}

func (h *POTHandler) GetStatus(c *gin.Context) {
    status, err := h.service.GetStatus()
    if err != nil {
        h.log.WithError(err).Error("Failed to get POT status")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to get POT status",
        })
        return
    }
    
    c.JSON(http.StatusOK, status)
}

func (h *POTHandler) GetVDFStatus(c *gin.Context) {
    vdfStatus, err := h.service.GetVDFStatus()
    if err != nil {
        h.log.WithError(err).Error("Failed to get VDF status")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to get VDF status",
        })
        return
    }
    
    c.JSON(http.StatusOK, vdfStatus)
}
```

---

### 3.4 Server启动 (`server.go`)

```go
package visualization

import (
    "fmt"
    
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/pot"
    "github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization/handlers"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization/middleware"
    "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization/services"
)

type VisualizationServer struct {
    engine *gin.Engine
    port   int
    log    *logrus.Entry
}

func NewVisualizationServer(
    worker *pot.Worker,
    nc *nodeController.NodeController,
    port int,
    log *logrus.Entry,
) *VisualizationServer {
    gin.SetMode(gin.ReleaseMode)
    engine := gin.New()
    
    // 中间件
    engine.Use(gin.Recovery())
    engine.Use(middleware.CORSMiddleware())
    engine.Use(middleware.RateLimitMiddleware(100)) // 100 req/s
    
    // 创建服务
    potService := services.NewPOTService(worker)
    // committeeService := services.NewCommitteeService(worker, nc)
    // ... 其他服务
    
    // 创建处理器
    potHandler := handlers.NewPOTHandler(potService, log)
    // ... 其他处理器
    
    // 注册路由
    api := engine.Group("/api")
    {
        potHandler.RegisterRoutes(api)
        // ... 其他处理器注册
    }
    
    return &VisualizationServer{
        engine: engine,
        port:   port,
        log:    log,
    }
}

func (s *VisualizationServer) Start() error {
    addr := fmt.Sprintf("0.0.0.0:%d", s.port)
    s.log.Infof("Starting Visualization API server on %s", addr)
    
    go func() {
        if err := s.engine.Run(addr); err != nil {
            s.log.WithError(err).Fatal("Failed to start server")
        }
    }()
    
    return nil
}
```

---

## 四、需要在Worker中添加的统计字段

在 `consensus/pot/worker.go` 中添加：

```go
type Worker struct {
    // ... 现有字段
    
    // 统计字段（新增）
    miningAttempts   uint64        // 挖矿尝试次数
    miningSuccesses  uint64        // 挖矿成功次数
    miningTimes      []float64     // 挖矿耗时记录（最近100次）
    vdfComputeTimes  []float64     // VDF计算耗时记录
    vdfFailCount     int           // VDF验证失败次数
    statsLock        sync.RWMutex  // 统计数据锁
}

// 添加统计方法
func (w *Worker) RecordMiningAttempt(success bool, duration float64) {
    w.statsLock.Lock()
    defer w.statsLock.Unlock()
    
    w.miningAttempts++
    if success {
        w.miningSuccesses++
    }
    
    // 保留最近100次记录
    if len(w.miningTimes) >= 100 {
        w.miningTimes = w.miningTimes[1:]
    }
    w.miningTimes = append(w.miningTimes, duration)
}

func (w *Worker) GetMiningStats() (float64, float64) {
    w.statsLock.RLock()
    defer w.statsLock.RUnlock()
    
    // 计算平均挖矿时间
    avgTime := 0.0
    if len(w.miningTimes) > 0 {
        sum := 0.0
        for _, t := range w.miningTimes {
            sum += t
        }
        avgTime = sum / float64(len(w.miningTimes))
    }
    
    // 计算成功率
    successRate := 0.0
    if w.miningAttempts > 0 {
        successRate = float64(w.miningSuccesses) / float64(w.miningAttempts) * 100
    }
    
    return avgTime, successRate
}
```

---

## 五、集成到现有系统

### 在POT Engine中启动可视化服务器

修改 `consensus/pot/engine.go`:

```go
import (
    visualization "github.com/zzz136454872/upgradeable-consensus/internal/apis/visualization"
)

func (e *PoTEngine) start() {
    e.log.Infof("[PoT]\tPoT Consensus Engine starts working")
    
    // ... 现有代码
    
    // 启动可视化API服务器
    if e.config.EnableVisualization {
        visPort := e.config.VisualizationPort
        if visPort == 0 {
            visPort = 9090 // 默认端口
        }
        
        visServer := visualization.NewVisualizationServer(
            e.Worker,
            e.UpperConsensus,
            visPort,
            e.log.WithField("component", "visualization"),
        )
        
        if err := visServer.Start(); err != nil {
            e.log.WithError(err).Error("Failed to start visualization server")
        } else {
            e.log.Infof("Visualization API server started on port %d", visPort)
        }
    }
}
```

### 在配置中添加可视化选项

修改 `config/config.go`:

```go
type ConsensusConfig struct {
    // ... 现有字段
    
    EnableVisualization bool `yaml:"enable_visualization"`
    VisualizationPort   int  `yaml:"visualization_port"`
}
```

在 `config/config.yaml` 中添加:

```yaml
enable_visualization: true
visualization_port: 9090
```

---

## 六、测试检查清单

### API测试
- [ ] 测试所有GET端点返回正确的JSON
- [ ] 测试错误处理（Worker为nil等情况）
- [ ] 测试CORS配置
- [ ] 测试限流功能
- [ ] 性能测试（并发请求）

### WebSocket测试
- [ ] 连接建立
- [ ] 订阅机制
- [ ] 数据推送
- [ ] 断线重连
- [ ] 多客户端支持

### 集成测试
- [ ] 与前端集成测试
- [ ] 数据准确性验证
- [ ] 实时性测试
- [ ] 长时间运行稳定性

---

## 七、性能优化建议

1. **缓存策略**
   - 使用 `github.com/patrickmn/go-cache`
   - 系统概览：5秒缓存
   - 网络拓扑：10秒缓存
   - 存储状态：30秒缓存

2. **并发控制**
   - 使用读写锁保护共享数据
   - 避免在Handler中执行耗时操作

3. **资源限制**
   - 限制WebSocket连接数
   - 限制历史数据查询范围
   - 使用连接池

---

## 八、部署建议

### Docker化
```dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go build -o pot-node .

EXPOSE 8080 9090

CMD ["./pot-node"]
```

### 监控
- 添加Prometheus metrics端点
- 记录API响应时间
- 监控WebSocket连接数

---

## 九、常见问题

### Q: VDF进度如何计算？
A: 需要在VDF计算时记录当前迭代次数和总迭代次数，计算百分比。

### Q: 如何处理Worker为nil的情况？
A: 在Service层添加nil检查，返回默认值或错误。

### Q: WebSocket断开如何处理？
A: 实现自动重连机制，前端在收到close事件后重新连接。

### Q: 如何测试API？
A: 使用Postman或curl测试，或使用`docs/rest-client-tests.http`文件。

---

按照这个检查清单，可以系统性地完成后端开发！
