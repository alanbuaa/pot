# 可视化API端点总结

本文档总结了所有可视化大屏需要的API端点，方便前端开发时快速查阅。

## API端点清单

### 1. 系统概览 API

#### `GET /api/system/overview`
**用途**: 获取系统整体概览数据，用于顶部状态栏显示

**返回字段**:
```typescript
interface SystemOverview {
  uptime: number;              // 系统运行时长（秒）
  totalNodes: number;          // 总节点数
  onlineNodes: number;         // 在线节点数
  consensusTypes: string[];    // 支持的共识类型 ["POT", "SimpleWhirly"]
  networkStatus: string;       // 网络状态 "healthy" | "warning" | "error"
  networkUtilization: number;  // 网络利用率 (0-100)
  executorStatus: string;      // 执行器状态 "normal" | "error"
  storageUsage: {
    total: number;             // 总存储空间
    used: number;              // 已使用空间
    percentage: number;        // 使用百分比
  };
  mempoolSize: number;         // 交易池大小
  currentTPS: number;          // 当前TPS
  currentHeight: number;       // 当前区块高度
  avgBlockTime: number;        // 平均出块时间（秒）
  lastBlockTime: string;       // 最新区块时间 (ISO 8601)
}
```

**刷新频率**: 1秒
**使用位置**: 顶部状态栏

---

### 2. POT共识 API

#### `GET /api/pot/status`
**用途**: 获取POT共识核心状态

**返回字段**:
```typescript
interface POTStatus {
  consensusType: string;       // "POT"
  epoch: number;               // 当前epoch
  difficulty: string;          // 挖矿难度值
  currentHeight: number;       // 当前区块高度
  workFlag: boolean;           // 挖矿工作状态
  timestamp: string;           // 当前时间戳 (ISO 8601)
  nonce: number;               // Nonce值
  uncleCount: number;          // 叔块数量
  avgMiningTime: number;       // 平均挖矿耗时（秒）
  miningSuccessRate: number;   // 挖矿成功率 (0-100)
}
```

**刷新频率**: 5秒
**使用位置**: 区块1 - POT共识状态

---

#### `GET /api/pot/vdf`
**用途**: 获取VDF计算详细状态

**返回字段**:
```typescript
interface VDFStatus {
  vdf0: {
    progress: number;          // 进度百分比 (0-100)
    iterations: number;        // 迭代次数
    status: string;            // "computing" | "done" | "idle"
    channelBuffer: number;     // 通道缓冲区大小
  };
  vdf1: Array<{
    workerId: number;          // 工作线程ID (0-3)
    progress: number;          // 进度百分比
    status: string;            // 状态
  }>;
  vdfHalf: {
    progress: number;          // VDF半计算进度
    status: string;            // 状态
    channelBuffer: number;     // 缓冲区大小
  };
  vdfChecker: {
    status: string;            // 验证器状态
    verifyFailCount: number;   // 验证失败次数
  };
  abortStatus: boolean;        // 中止状态
  avgComputeTime: number;      // 平均计算耗时（毫秒）
  totalIterations: number;     // 总迭代次数
  cpuCounter: number;          // 并行工作线程数
}
```

**刷新频率**: 1秒
**使用位置**: 区块1 + 区块5 - VDF计算监控

---

### 3. 委员会共识 API

#### `GET /api/committee/status`
**用途**: 获取委员会共识状态

**返回字段**:
```typescript
interface CommitteeStatus {
  consensusType: string;       // "SimpleWhirly" | "CRWhirly" | "Whirly"
  committeeSize: number;       // 委员会大小 (4)
  committeeCount: number;      // 委员会数量 (4)
  confirmDelay: number;        // 确认延迟 (6)
  workHeight: number;          // 委员会工作高度
  batchSize: number;           // 批处理大小 (2-10)
  inCommittee: boolean;        // 是否在委员会中
  role: string;                // "leader" | "member" | "observer"
  committee: Array<{
    address: string;           // 地址
    publicKey: string;         // 公钥
    isLeader: boolean;         // 是否为Leader
  }>;
  committeePublicKey: string;  // 委员会聚合公钥
  shardings: Array<{
    name: string;              // 分片名称
    id: number;                // 分片ID
    leaderAddress: string;     // Leader地址
    committeeMembers: number;  // 委员会成员数
    status: string;            // "active" | "inactive"
  }>;
  workStage: string;           // "init" | "shuffle" | "draw" | "share" | "consensus"
  timeout: number;             // 超时配置（毫秒）
  messageQueueLength: number;  // 消息队列长度
  electionHeight: number;      // 选举区块高度
}
```

**刷新频率**: 5秒
**使用位置**: 区块2 - 委员会共识

---

### 4. BCI激励 API

#### `GET /api/bci/status`
**用途**: 获取BCI激励系统状态

**返回字段**:
```typescript
interface BCIStatus {
  totalReward: number;         // 总奖励金额 (65536)
  lockedReward: number;        // 锁定的激励金额
  totalInterest: number;       // 累计利息总数
  rewardRatio: {
    exchequer: number;         // 国库 30%
    miner: number;             // 矿工 50%
    uncleBlockMiner: number;   // 叔块矿工 2%
    committeeLeader: number;   // 委员会Leader 20%
    committeeMember: number;   // 委员会成员 10%
  };
  lockRates: {
    saving: number;            // 活期 0.1%
    halfYear: number;          // 半年 0.5%
    oneYear: number;           // 一年 1%
    threeYears: number;        // 三年 2%
    tenYears: number;          // 十年 5%
  };
  coinbaseLock: number;        // Coinbase锁定期 (6块)
  executeHeight: number;       // 执行高度
  incentiveHeight: number;     // 激励高度
  pendingRewards: number;      // 待分配奖励数量
  utxoCount: number;           // UTXO总数
  totalDistributed: number;    // 历史累计分发总额
}
```

**刷新频率**: 10秒
**使用位置**: 区块6 - BCI激励

---

#### `GET /api/bci/balance/:address`
**用途**: 查询指定地址余额（可选功能）

**参数**:
- `address`: 地址（hex编码）

**返回字段**:
```typescript
interface BalanceInfo {
  address: string;             // 地址
  balance: number;             // 总余额
  utxos: Array<{
    txid: string;              // 交易ID
    vout: number;              // 输出索引
    value: number;             // 金额
    lockHeight: number;        // 锁定高度
  }>;
  lockedBalance: number;       // 锁定余额
  availableBalance: number;    // 可用余额
}
```

**刷新频率**: 按需
**使用位置**: 可选 - 地址余额查询功能

---

### 5. 交易池 API

#### `GET /api/mempool/status`
**用途**: 获取交易池状态

**返回字段**:
```typescript
interface MempoolStatus {
  totalSize: number;           // 交易池总大小
  markedTxs: number;           // 已提议交易数
  unmarkedTxs: number;         // 待提议交易数
  txTypes: {
    normal: number;            // 普通交易数
    bci: number;               // BCI交易数
    devastate: number;         // Devastate交易数
  };
  avgConfirmTime: number;      // 平均确认时间（秒）
  verifySuccessRate: number;   // 验证成功率 (0-100)
  memoryUsage: number;         // 内存占用（字节）
  recentTxs: Array<{
    hash: string;              // 交易哈希
    type: string;              // 交易类型
    timestamp: string;         // 时间戳 (ISO 8601)
    status: string;            // "pending" | "confirmed"
  }>;
}
```

**刷新频率**: 5秒
**使用位置**: 区块3 - 交易池监控

---

### 6. 网络拓扑 API

#### `GET /api/network/topology`
**用途**: 获取P2P网络拓扑数据

**返回字段**:
```typescript
interface NetworkTopology {
  nodes: Array<{
    id: number;                // 节点ID
    peerId: string;            // PeerID
    address: string;           // 地址
    status: string;            // "online" | "offline"
    connections: number;       // 连接数
    latency: number;           // 延迟（毫秒）
  }>;
  edges: Array<{
    source: number;            // 源节点ID
    target: number;            // 目标节点ID
    latency: number;           // 延迟（毫秒）
    messageCount: number;      // 消息数量
  }>;
  p2pAdaptorType: string;      // "p2p" | "libp2p"
  subscribedTopics: string[];  // 订阅主题列表
  messageQueueLength: number;  // 消息队列长度
  networkBandwidth: number;    // 网络带宽（字节/秒）
}
```

**刷新频率**: 10秒
**使用位置**: 区块4 - 网络拓扑

---

### 7. 存储 API

#### `GET /api/storage/status`
**用途**: 获取存储状态

**返回字段**:
```typescript
interface StorageStatus {
  totalSize: number;           // 总大小（字节）
  buckets: {
    blocks: {
      size: number;            // BlocksBucket大小
      count: number;           // 记录数
    };
    executed: {
      size: number;            // ExecutedBucket大小
      count: number;           // 记录数
    };
    utxo: {
      size: number;            // UTXOBucket大小
      count: number;           // 记录数
    };
    client: {
      size: number;            // ClientBucket大小
      count: number;           // 记录数
    };
  };
  compressionRatio: number;    // 压缩率 (0-1)
  vdfHeight: number;           // VDF高度记录
  blockCount: number;          // 区块总数
}
```

**刷新频率**: 30秒
**使用位置**: 区块7 - 存储状态

---

### 8. 历史指标 API

#### `GET /api/metrics/history`
**用途**: 获取历史指标数据（用于趋势图）

**参数**:
- `metric`: 指标类型 `"tps" | "difficulty" | "blocktime"`
- `timeRange`: 时间范围 `"1h" | "6h" | "24h" | "7d"`
- `interval`: 采样间隔 `"1m" | "5m" | "1h"`

**返回字段**:
```typescript
interface MetricsHistory {
  metric: string;              // 指标类型
  timeRange: string;           // 时间范围
  interval: string;            // 采样间隔
  data: Array<{
    timestamp: string;         // 时间戳 (ISO 8601)
    value: number;             // 指标值
  }>;
  avg: number;                 // 平均值
  max: number;                 // 最大值
  min: number;                 // 最小值
}
```

**刷新频率**: 30秒
**使用位置**: 底部时间轴

---

## WebSocket 实时推送

### WebSocket 连接
**端点**: `ws://localhost:8080/api/ws`

### 订阅消息格式
```json
{
  "action": "subscribe",
  "types": ["pot", "committee", "mempool", "vdf", "network"]
}
```

### 推送消息格式
```json
{
  "type": "pot",
  "timestamp": "2025-11-08T10:30:45Z",
  "data": {
    "height": 12345,
    "tps": 45.6,
    "workFlag": true
  }
}
```

### 推送频率
- 关键指标（POT状态、VDF进度）: 1秒
- 一般指标（交易池、委员会）: 5秒
- 网络拓扑: 10秒

---

## API使用建议

### 1. 初始加载顺序
1. `GET /api/system/overview` - 系统概览
2. `GET /api/pot/status` - POT状态
3. `GET /api/pot/vdf` - VDF状态
4. `GET /api/committee/status` - 委员会状态
5. `GET /api/mempool/status` - 交易池状态
6. `GET /api/network/topology` - 网络拓扑
7. `GET /api/bci/status` - BCI激励
8. `GET /api/storage/status` - 存储状态

### 2. 轮询策略
- **1秒轮询**: `system/overview`, `pot/vdf`
- **5秒轮询**: `pot/status`, `committee/status`, `mempool/status`
- **10秒轮询**: `network/topology`, `bci/status`
- **30秒轮询**: `storage/status`, `metrics/history`

### 3. WebSocket策略
建议优先使用WebSocket接收实时数据，HTTP轮询作为备用方案。

### 4. 错误处理
- 超时时间：10秒
- 失败重试：最多3次，指数退避
- 降级策略：WebSocket失败时回退到HTTP轮询

### 5. 缓存策略
- 系统概览：5秒缓存
- 网络拓扑：10秒缓存
- 存储状态：30秒缓存

---

## 前端状态管理建议

### Pinia Store结构
```typescript
// stores/pot.ts
export const usePotStore = defineStore('pot', {
  state: () => ({
    status: null as POTStatus | null,
    vdf: null as VDFStatus | null,
  }),
  actions: {
    async fetchStatus() { ... },
    async fetchVDF() { ... },
  }
})

// stores/committee.ts
export const useCommitteeStore = defineStore('committee', { ... })

// stores/bci.ts
export const useBCIStore = defineStore('bci', { ... })

// stores/mempool.ts
export const useMempoolStore = defineStore('mempool', { ... })

// stores/network.ts
export const useNetworkStore = defineStore('network', { ... })

// stores/storage.ts
export const useStorageStore = defineStore('storage', { ... })

// stores/system.ts
export const useSystemStore = defineStore('system', { ... })
```

### 组合式函数 (Composables)
```typescript
// composables/useRealtime.ts
export function useRealtime(apiEndpoint: string, interval: number) {
  const data = ref(null)
  const error = ref(null)
  const loading = ref(false)
  
  const fetch = async () => { ... }
  const startPolling = () => { ... }
  const stopPolling = () => { ... }
  
  return { data, error, loading, fetch, startPolling, stopPolling }
}

// composables/useWebSocket.ts
export function useWebSocket(url: string) { ... }
```

---

## 开发检查清单

- [ ] 所有API端点已实现
- [ ] WebSocket服务已启动
- [ ] CORS配置正确
- [ ] 错误处理完善
- [ ] 数据格式验证
- [ ] 性能监控就绪
- [ ] 日志记录完整
- [ ] 文档更新完成

---

## 后端实现建议

### Go 后端 API 实现框架

基于项目现有的 `internal/apis` 目录，建议的实现结构：

```
internal/apis/
├── handlers/
│   ├── system_handler.go      # 系统概览处理器
│   ├── pot_handler.go          # POT共识处理器
│   ├── committee_handler.go    # 委员会处理器
│   ├── bci_handler.go          # BCI激励处理器
│   ├── mempool_handler.go      # 交易池处理器
│   ├── network_handler.go      # 网络拓扑处理器
│   ├── storage_handler.go      # 存储处理器
│   └── websocket_handler.go    # WebSocket处理器
├── middleware/
│   ├── cors.go                 # CORS中间件
│   └── logger.go               # 日志中间件
└── router.go                   # 路由配置
```

### 示例代码

#### 1. 路由配置 (`router.go`)

```go
package apis

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func SetupRouter(node *node.Node) *gin.Engine {
	r := gin.Default()
	
	// CORS中间件
	r.Use(CORSMiddleware())
	
	// API路由组
	api := r.Group("/api")
	{
		// 系统API
		api.GET("/system/overview", handlers.GetSystemOverview(node))
		
		// POT API
		api.GET("/pot/status", handlers.GetPotStatus(node))
		api.GET("/pot/vdf", handlers.GetVDFStatus(node))
		
		// 委员会API
		api.GET("/committee/status", handlers.GetCommitteeStatus(node))
		
		// BCI API
		api.GET("/bci/status", handlers.GetBCIStatus(node))
		api.GET("/bci/balance/:address", handlers.GetBalance(node))
		
		// 交易池API
		api.GET("/mempool/status", handlers.GetMempoolStatus(node))
		
		// 网络API
		api.GET("/network/topology", handlers.GetNetworkTopology(node))
		
		// 存储API
		api.GET("/storage/status", handlers.GetStorageStatus(node))
		
		// 历史数据API
		api.GET("/metrics/history", handlers.GetMetricsHistory(node))
		
		// WebSocket
		api.GET("/ws", handlers.WebSocketHandler(node))
	}
	
	return r
}

// CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}
```

#### 2. 系统概览处理器 (`handlers/system_handler.go`)

```go
package handlers

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
)

type SystemOverview struct {
	Uptime             int64    `json:"uptime"`
	TotalNodes         int      `json:"totalNodes"`
	OnlineNodes        int      `json:"onlineNodes"`
	ConsensusTypes     []string `json:"consensusTypes"`
	NetworkStatus      string   `json:"networkStatus"`
	NetworkUtilization float64  `json:"networkUtilization"`
	ExecutorStatus     string   `json:"executorStatus"`
	StorageUsage       struct {
		Total      int64   `json:"total"`
		Used       int64   `json:"used"`
		Percentage float64 `json:"percentage"`
	} `json:"storageUsage"`
	MempoolSize   int     `json:"mempoolSize"`
	CurrentTPS    float64 `json:"currentTPS"`
	CurrentHeight int64   `json:"currentHeight"`
	AvgBlockTime  float64 `json:"avgBlockTime"`
	LastBlockTime string  `json:"lastBlockTime"`
}

var startTime = time.Now()

func GetSystemOverview(node *node.Node) gin.HandlerFunc {
	return func(c *gin.Context) {
		overview := SystemOverview{
			Uptime:         int64(time.Since(startTime).Seconds()),
			TotalNodes:     10, // 从节点管理器获取
			OnlineNodes:    8,
			ConsensusTypes: []string{"POT", "SimpleWhirly"},
			NetworkStatus:  "healthy",
			NetworkUtilization: 45.5,
			ExecutorStatus: "normal",
			MempoolSize:    node.Mempool.Size(),
			CurrentTPS:     calculateTPS(node),
			CurrentHeight:  node.Storage.GetCurrentHeight(),
			AvgBlockTime:   calculateAvgBlockTime(node),
			LastBlockTime:  time.Now().Format(time.RFC3339),
		}
		
		// 获取存储使用情况
		storageSize := node.Storage.GetTotalSize()
		overview.StorageUsage.Total = 1024 * 1024 * 1024 * 10 // 10GB
		overview.StorageUsage.Used = storageSize
		overview.StorageUsage.Percentage = float64(storageSize) / float64(overview.StorageUsage.Total) * 100
		
		c.JSON(http.StatusOK, overview)
	}
}

func calculateTPS(node *node.Node) float64 {
	// 计算最近1分钟的TPS
	// 实现逻辑...
	return 45.6
}

func calculateAvgBlockTime(node *node.Node) float64 {
	// 计算平均出块时间
	// 实现逻辑...
	return 5.2
}
```

#### 3. POT状态处理器 (`handlers/pot_handler.go`)

```go
package handlers

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
)

type POTStatus struct {
	ConsensusType     string  `json:"consensusType"`
	Epoch             int64   `json:"epoch"`
	Difficulty        string  `json:"difficulty"`
	CurrentHeight     int64   `json:"currentHeight"`
	WorkFlag          bool    `json:"workFlag"`
	Timestamp         string  `json:"timestamp"`
	Nonce             int64   `json:"nonce"`
	UncleCount        int     `json:"uncleCount"`
	AvgMiningTime     float64 `json:"avgMiningTime"`
	MiningSuccessRate float64 `json:"miningSuccessRate"`
}

func GetPotStatus(node *node.Node) gin.HandlerFunc {
	return func(c *gin.Context) {
		potConsensus := node.Consensus.(*consensus.POT)
		
		status := POTStatus{
			ConsensusType:     "POT",
			Epoch:             potConsensus.GetEpoch(),
			Difficulty:        potConsensus.GetDifficulty().Text(16),
			CurrentHeight:     potConsensus.GetHeight(),
			WorkFlag:          potConsensus.IsWorking(),
			Timestamp:         time.Now().Format(time.RFC3339),
			Nonce:             potConsensus.GetNonce(),
			UncleCount:        potConsensus.GetUncleCount(),
			AvgMiningTime:     calculateAvgMiningTime(potConsensus),
			MiningSuccessRate: calculateMiningSuccessRate(potConsensus),
		}
		
		c.JSON(http.StatusOK, status)
	}
}

type VDFStatus struct {
	VDF0 struct {
		Progress      float64 `json:"progress"`
		Iterations    int64   `json:"iterations"`
		Status        string  `json:"status"`
		ChannelBuffer int     `json:"channelBuffer"`
	} `json:"vdf0"`
	VDF1 []struct {
		WorkerID int     `json:"workerId"`
		Progress float64 `json:"progress"`
		Status   string  `json:"status"`
	} `json:"vdf1"`
	VDFHalf struct {
		Progress      float64 `json:"progress"`
		Status        string  `json:"status"`
		ChannelBuffer int     `json:"channelBuffer"`
	} `json:"vdfHalf"`
	VDFChecker struct {
		Status          string `json:"status"`
		VerifyFailCount int    `json:"verifyFailCount"`
	} `json:"vdfChecker"`
	AbortStatus     bool    `json:"abortStatus"`
	AvgComputeTime  float64 `json:"avgComputeTime"`
	TotalIterations int64   `json:"totalIterations"`
	CPUCounter      int     `json:"cpuCounter"`
}

func GetVDFStatus(node *node.Node) gin.HandlerFunc {
	return func(c *gin.Context) {
		potConsensus := node.Consensus.(*consensus.POT)
		vdfEngine := potConsensus.GetVDFEngine()
		
		status := VDFStatus{
			CPUCounter:      vdfEngine.GetCPUCounter(),
			AbortStatus:     vdfEngine.IsAborted(),
			AvgComputeTime:  vdfEngine.GetAvgComputeTime(),
			TotalIterations: vdfEngine.GetTotalIterations(),
		}
		
		// VDF0状态
		status.VDF0.Progress = vdfEngine.GetVDF0Progress()
		status.VDF0.Iterations = vdfEngine.GetVDF0Iterations()
		status.VDF0.Status = vdfEngine.GetVDF0Status()
		status.VDF0.ChannelBuffer = vdfEngine.GetVDF0ChannelBuffer()
		
		// VDF1并行线程状态
		for i := 0; i < 4; i++ {
			worker := struct {
				WorkerID int     `json:"workerId"`
				Progress float64 `json:"progress"`
				Status   string  `json:"status"`
			}{
				WorkerID: i,
				Progress: vdfEngine.GetVDF1Progress(i),
				Status:   vdfEngine.GetVDF1Status(i),
			}
			status.VDF1 = append(status.VDF1, worker)
		}
		
		// VDFHalf状态
		status.VDFHalf.Progress = vdfEngine.GetVDFHalfProgress()
		status.VDFHalf.Status = vdfEngine.GetVDFHalfStatus()
		status.VDFHalf.ChannelBuffer = vdfEngine.GetVDFHalfChannelBuffer()
		
		// VDFChecker状态
		status.VDFChecker.Status = vdfEngine.GetVDFCheckerStatus()
		status.VDFChecker.VerifyFailCount = vdfEngine.GetVerifyFailCount()
		
		c.JSON(http.StatusOK, status)
	}
}
```

#### 4. WebSocket处理器 (`handlers/websocket_handler.go`)

```go
package handlers

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

type WSClient struct {
	conn          *websocket.Conn
	subscriptions []string
	send          chan []byte
}

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
}

var hub = &WSHub{
	clients:    make(map[*WSClient]bool),
	broadcast:  make(chan []byte),
	register:   make(chan *WSClient),
	unregister: make(chan *WSClient),
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func WebSocketHandler(node *node.Node) gin.HandlerFunc {
	// 启动Hub
	go hub.Run()
	
	// 启动数据推送协程
	go pushDataPeriodically(node)
	
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		client := &WSClient{
			conn: conn,
			send: make(chan []byte, 256),
		}
		
		hub.register <- client
		
		// 读取客户端消息
		go func() {
			defer func() {
				hub.unregister <- client
				conn.Close()
			}()
			
			for {
				var msg map[string]interface{}
				err := conn.ReadJSON(&msg)
				if err != nil {
					break
				}
				
				// 处理订阅请求
				if action, ok := msg["action"].(string); ok && action == "subscribe" {
					if types, ok := msg["types"].([]interface{}); ok {
						client.subscriptions = make([]string, len(types))
						for i, t := range types {
							client.subscriptions[i] = t.(string)
						}
					}
				}
			}
		}()
		
		// 发送消息
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			
			for {
				select {
				case message, ok := <-client.send:
					if !ok {
						return
					}
					conn.WriteMessage(websocket.TextMessage, message)
				case <-ticker.C:
					// 心跳检测
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()
	}
}

func pushDataPeriodically(node *node.Node) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// 推送POT状态
		potStatus := getPOTStatusData(node)
		hub.broadcast <- potStatus
		
		// 推送VDF状态
		vdfStatus := getVDFStatusData(node)
		hub.broadcast <- vdfStatus
	}
}
```

### 注意事项

1. **错误处理**: 所有handler都应该有适当的错误处理
2. **性能**: 高频接口（1秒刷新）要注意性能优化
3. **并发安全**: 访问共享数据时要加锁
4. **日志**: 记录所有API调用和错误
5. **监控**: 添加Prometheus指标监控API性能

### 启动API服务器

在 `cmd/server/server.go` 中添加：

```go
func main() {
	// ... 初始化node
	
	// 启动API服务器
	router := apis.SetupRouter(node)
	go func() {
		if err := router.Run(":8080"); err != nil {
			log.Fatal("Failed to start API server:", err)
		}
	}()
	
	// ... 其他启动逻辑
}
```
