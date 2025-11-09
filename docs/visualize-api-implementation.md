# 可视化后端API实现方案

## 一、API架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        前端 (Vue3)                           │
│                     WebSocket + HTTP API                     │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    API Gateway (Gin)                         │
│  ┌──────────────┬──────────────┬──────────────────────────┐ │
│  │ REST API     │ WebSocket    │ Metrics Collector        │ │
│  │ Handlers     │ Hub          │ (定时采集)                │ │
│  └──────────────┴──────────────┴──────────────────────────┘ │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Service Layer (服务层)                      │
│  ┌──────────────┬──────────────┬──────────────────────────┐ │
│  │ POT Service  │ Committee    │ BCI Service              │ │
│  │              │ Service      │                          │ │
│  ├──────────────┼──────────────┼──────────────────────────┤ │
│  │ VDF Service  │ Network      │ Storage Service          │ │
│  │              │ Service      │                          │ │
│  └──────────────┴──────────────┴──────────────────────────┘ │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Data Access Layer                           │
│  ┌──────────────┬──────────────┬──────────────────────────┐ │
│  │ Worker       │ BlockStorage │ ChainReader              │ │
│  │ (POT)        │ (BoltDB)     │                          │ │
│  ├──────────────┼──────────────┼──────────────────────────┤ │
│  │ NodeController│MemPool      │ Executor                 │ │
│  │ (Committee)  │              │                          │ │
│  └──────────────┴──────────────┴──────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、API端点设计

### 2.1 整体架构API

#### GET `/api/system/overview`
**功能**: 获取系统整体概览数据

**返回示例**:
```json
{
  "uptime": 86400,
  "totalNodes": 4,
  "onlineNodes": 4,
  "consensusTypes": ["POT", "SimpleWhirly"],
  "networkStatus": "healthy",
  "networkUtilization": 65.5,
  "executorStatus": "normal",
  "storageUsage": {
    "total": 1073741824,
    "used": 536870912,
    "percentage": 50.0
  },
  "mempoolSize": 156,
  "currentTPS": 45.6,
  "currentHeight": 12345,
  "avgBlockTime": 12.5,
  "lastBlockTime": "2025-11-07T10:30:45Z"
}
```

**实现逻辑**:
```go
func (s *SystemService) GetOverview() (*SystemOverview, error) {
    return &SystemOverview{
        Uptime:        time.Since(s.startTime).Seconds(),
        TotalNodes:    len(s.config.Nodes),
        OnlineNodes:   s.getOnlineNodeCount(),
        ConsensusTypes: []string{"POT", "SimpleWhirly"},
        NetworkStatus: s.checkNetworkHealth(),
        NetworkUtilization: s.calculateNetworkUtilization(),
        ExecutorStatus: s.checkExecutorStatus(),
        StorageUsage: s.getStorageUsage(),
        MempoolSize: s.worker.mempool.GetSize(),
        CurrentTPS: s.calculateCurrentTPS(),
        CurrentHeight: s.worker.GetHeight(),
        AvgBlockTime: s.calculateAvgBlockTime(),
        LastBlockTime: s.getLastBlockTime(),
    }
}
```

---

#### GET `/api/network/topology`
**功能**: 获取P2P网络拓扑数据

**返回示例**:
```json
{
  "nodes": [
    {
      "id": 0,
      "peerId": "QmXxxx...",
      "address": "192.168.1.1:8080",
      "status": "online",
      "connections": 3,
      "latency": 15.5
    }
  ],
  "edges": [
    {
      "source": 0,
      "target": 1,
      "latency": 12.3,
      "messageCount": 1234
    }
  ],
  "p2pAdaptorType": "libp2p",
  "subscribedTopics": ["pot-consensus"],
  "messageQueueLength": 45,
  "networkBandwidth": 1048576
}
```

**实现逻辑**:
```go
func (s *NetworkService) GetTopology() (*NetworkTopology, error) {
    nodes := make([]NodeInfo, 0)
    for id, nodeConfig := range s.config.Nodes {
        nodes = append(nodes, NodeInfo{
            ID:          id,
            PeerID:      s.getPeerID(id),
            Address:     nodeConfig.Address,
            Status:      s.getNodeStatus(id),
            Connections: s.getConnectionCount(id),
            Latency:     s.measureLatency(id),
        })
    }
    
    edges := s.buildEdges()
    
    return &NetworkTopology{
        Nodes:              nodes,
        Edges:              edges,
        P2PAdaptorType:     s.getAdaptorType(),
        SubscribedTopics:   s.getSubscribedTopics(),
        MessageQueueLength: len(s.worker.MsgByteEntrance),
        NetworkBandwidth:   s.calculateNetworkBandwidth(),
    }
}
```

---

### 2.2 POT共识API

#### GET `/api/pot/status`
**功能**: 获取POT共识状态

**返回示例**:
```json
{
  "consensusType": "POT",
  "epoch": 1234,
  "difficulty": "1234567890",
  "currentHeight": 12345,
  "workFlag": true,
  "timestamp": "2025-11-07T10:30:45Z",
  "nonce": 123456,
  "uncleCount": 2,
  "avgMiningTime": 12.5,
  "miningSuccessRate": 85.5
}
```

**实现逻辑**:
```go
func (s *POTService) GetStatus() (*POTStatus, error) {
    worker := s.worker
    latestBlock := worker.chainReader.GetLatest()
    header := latestBlock.GetHeader()
    
    return &POTStatus{
        ConsensusType:      "POT",
        Epoch:              worker.epoch,
        Difficulty:         header.Difficulty.String(),
        CurrentHeight:      worker.Height,
        WorkFlag:           worker.workFlag,
        Timestamp:          worker.timestamp,
        Nonce:              header.Nonce,
        UncleCount:         len(header.UncleHash),
        AvgMiningTime:      s.calculateAvgMiningTime(),
        MiningSuccessRate:  s.calculateSuccessRate(),
    }, nil
}
```

---

#### GET `/api/pot/vdf`
**功能**: 获取VDF计算状态

**返回示例**:
```json
{
  "vdf0": {
    "progress": 75.5,
    "iterations": 1000000,
    "status": "computing",
    "channelBuffer": 10
  },
  "vdf1": [
    {
      "workerId": 0,
      "progress": 85.2,
      "status": "computing"
    },
    {
      "workerId": 1,
      "progress": 82.1,
      "status": "computing"
    },
    {
      "workerId": 2,
      "progress": 80.5,
      "status": "computing"
    },
    {
      "workerId": 3,
      "progress": 78.9,
      "status": "computing"
    }
  ],
  "vdfHalf": {
    "progress": 50.0,
    "status": "computing",
    "channelBuffer": 5
  },
  "vdfChecker": {
    "status": "ready",
    "verifyFailCount": 3
  },
  "abortStatus": false,
  "avgComputeTime": 125.5,
  "totalIterations": 5000000,
  "cpuCounter": 4
}
```

**实现逻辑**:
```go
func (s *VDFService) GetStatus() (*VDFStatus, error) {
    worker := s.worker
    
    vdf0Status := VDF0Status{
        Progress:      s.calculateVDF0Progress(worker.vdf0),
        Iterations:    worker.vdf0.Iterations,
        Status:        s.getVDFStatusString(worker.vdf0),
        ChannelBuffer: len(worker.vdf0Chan),
    }
    
    // 获取所有VDF1工作线程状态（cpuCounter数量）
    vdf1Statuses := make([]VDF1Status, len(worker.vdf1))
    for i, vdf := range worker.vdf1 {
        vdf1Statuses[i] = VDF1Status{
            WorkerID: i,
            Progress: s.calculateVDF1Progress(vdf),
            Status:   s.getVDFStatusString(vdf),
        }
    }
    
    vdfHalfStatus := VDFHalfStatus{
        Progress:      s.calculateVDFHalfProgress(worker.vdfhalf),
        Status:        s.getVDFStatusString(worker.vdfhalf),
        ChannelBuffer: len(worker.vdfhalfchan),
    }
    
    return &VDFStatus{
        VDF0:             vdf0Status,
        VDF1:             vdf1Statuses,
        VDFHalf:          vdfHalfStatus,
        VDFChecker:       s.getVDFCheckerStatus(),
        AbortStatus:      worker.abort.IsAborted(),
        AvgComputeTime:   s.calculateAvgComputeTime(),
        TotalIterations:  s.getTotalIterations(),
        CPUCounter:       len(worker.vdf1), // cpuCounter
    }, nil
}
```

---

### 2.3 委员会共识API

#### GET `/api/committee/status`
**功能**: 获取委员会共识状态

**返回示例**:
```json
{
  "consensusType": "SimpleWhirly",
  "committeeSize": 4,
  "committeeCount": 4,
  "confirmDelay": 6,
  "workHeight": 12340,
  "batchSize": 10,
  "inCommittee": true,
  "role": "member",
  "committee": [
    {
      "address": "0x1234...",
      "publicKey": "0xabcd...",
      "isLeader": true
    },
    {
      "address": "0x5678...",
      "publicKey": "0xef01...",
      "isLeader": false
    }
  ],
  "committeePublicKey": "0x5678...",
  "shardings": [
    {
      "name": "shard-1",
      "id": 1,
      "leaderAddress": "0x1234...",
      "committeeMembers": 4,
      "status": "active"
    }
  ],
  "workStage": "consensus",
  "timeout": 2000,
  "messageQueueLength": 15,
  "electionHeight": 12334
}
```

**实现逻辑**:
```go
func (s *CommitteeService) GetStatus() (*CommitteeStatus, error) {
    nc := s.nodeController
    worker := s.worker
    
    // 获取委员会成员列表
    committee := make([]CommitteeMember, 0)
    latestBlock := worker.chainReader.GetLatest()
    height := latestBlock.GetHeader().Height
    
    for i := uint64(0); i < Commiteelen; i++ {
        block, _ := worker.chainReader.GetByHeight(height - CommiteeDelay - i)
        if block != nil {
            header := block.GetHeader()
            isLeader := i == 0
            committee = append(committee, CommitteeMember{
                Address:   hexutil.Encode(header.CommiteePubkey),
                PublicKey: hexutil.Encode(header.CommiteePubkey),
                IsLeader:  isLeader,
            })
        }
    }
    
    // 获取分片信息
    shardings := make([]ShardingInfo, 0)
    for name, shard := range nc.Shardings {
        shardings = append(shardings, ShardingInfo{
            Name:             name,
            ID:               shard.Id,
            LeaderAddress:    shard.LeaderAddress,
            CommitteeMembers: len(shard.Committee),
            Status:           s.getShardingStatus(shard),
        })
    }
    
    return &CommitteeStatus{
        ConsensusType:       s.getConsensusType(),
        CommitteeSize:       Commiteelen,
        CommitteeCount:      SmallN,
        ConfirmDelay:        ConfirmDelay,
        WorkHeight:          height,
        BatchSize:           s.getBatchSize(),
        InCommittee:         s.isInCommittee(),
        Role:                s.getRole(),
        Committee:           committee,
        CommitteePublicKey:  s.getCommitteePublicKey(),
        Shardings:           shardings,
        WorkStage:           s.getWorkStage(),
        Timeout:             s.getTimeout(),
        MessageQueueLength:  len(nc.MsgByteEntrance),
        ElectionHeight:      height - CommiteeDelay,
    }, nil
}
```

---

### 2.4 BCI激励API

#### GET `/api/bci/status`
**功能**: 获取BCI激励状态

**返回示例**:
```json
{
  "totalReward": 65536,
  "lockedReward": 32768,
  "totalInterest": 1234.56,
  "rewardRatio": {
    "exchequer": 30.0,
    "miner": 50.0,
    "uncleBlockMiner": 2.0,
    "committeeLeader": 20.0,
    "committeeMember": 10.0
  },
  "lockRates": {
    "saving": 0.1,
    "halfYear": 0.5,
    "oneYear": 1.0,
    "threeYears": 2.0,
    "tenYears": 5.0
  },
  "coinbaseLock": 6,
  "executeHeight": 12345,
  "incentiveHeight": 12340,
  "pendingRewards": 15,
  "utxoCount": 5678,
  "totalDistributed": 1234567
}
```

**实现逻辑**:
```go
func (s *BCIService) GetStatus() (*BCIStatus, error) {
    worker := s.worker
    
    // 计算总奖励和锁定奖励
    totalReward := TotalReward
    lockedReward := s.calculateLockedReward()
    totalInterest := s.calculateTotalInterest()
    
    // 获取UTXO统计
    utxoCount := s.getUTXOCount()
    
    return &BCIStatus{
        TotalReward:   totalReward,
        LockedReward:  lockedReward,
        TotalInterest: totalInterest,
        RewardRatio: map[string]float64{
            "exchequer":       bcimap[exchequer] * 100,
            "miner":           bcimap[Miner] * 100,
            "uncleBlockMiner": bcimap[UncleBlockMiner] * 100,
            "committeeLeader": bcimap[CommitteeLeader] * 100,
            "committeeMember": bcimap[CommitteeMember] * 100,
        },
        LockRates: map[string]float64{
            "saving":     savingRate * 100,
            "halfYear":   HalfYearRate * 100,
            "oneYear":    OneYearRate * 100,
            "threeYears": ThreeYearRate * 100,
            "tenYears":   TenYearRate * 100,
        },
        CoinbaseLock:        CoinbaseLock,
        ExecuteHeight:       worker.executeheight,
        IncentiveHeight:     worker.incentiveheight,
        PendingRewards:      s.getPendingRewardsCount(),
        UTXOCount:           utxoCount,
        TotalDistributed:    s.calculateTotalDistributed(),
    }, nil
}
```

---

#### GET `/api/bci/balance/:address`
**功能**: 查询地址余额

**返回示例**:
```json
{
  "address": "0x1234...",
  "balance": 1234567,
  "utxos": [
    {
      "txid": "0xabcd...",
      "vout": 0,
      "value": 100,
      "lockHeight": 12345
    }
  ],
  "lockedBalance": 500,
  "availableBalance": 1234067
}
```

**实现逻辑**:
```go
func (s *BCIService) GetBalance(address []byte) (*BalanceInfo, error) {
    worker := s.worker
    
    // 查找UTXO
    utxos, err := worker.chainReader.FindUTXO(address)
    if err != nil {
        return nil, err
    }
    
    balance := int64(0)
    lockedBalance := int64(0)
    utxoList := make([]UTXOInfo, 0)
    
    for utxokey, utxo := range utxos {
        balance += utxo.Value
        
        // 检查是否锁定
        if utxo.LockHeight > worker.Height {
            lockedBalance += utxo.Value
        }
        
        parts := strings.Split(utxokey, ":")
        txid := parts[0]
        vout, _ := strconv.ParseInt(parts[1], 10, 64)
        
        utxoList = append(utxoList, UTXOInfo{
            TxID:       txid,
            Vout:       vout,
            Value:      utxo.Value,
            LockHeight: utxo.LockHeight,
        })
    }
    
    return &BalanceInfo{
        Address:          hexutil.Encode(address),
        Balance:          balance,
        UTXOs:            utxoList,
        LockedBalance:    lockedBalance,
        AvailableBalance: balance - lockedBalance,
    }, nil
}
```

---

### 2.5 交易池和执行API

#### GET `/api/mempool/status`
**功能**: 获取交易池状态

**返回示例**:
```json
{
  "totalSize": 156,
  "markedTxs": 20,
  "unmarkedTxs": 136,
  "txTypes": {
    "normal": 100,
    "bci": 30,
    "devastate": 26
  },
  "avgConfirmTime": 15.5,
  "verifySuccessRate": 98.5,
  "memoryUsage": 1048576,
  "recentTxs": [
    {
      "hash": "0x1234...",
      "type": "normal",
      "timestamp": "2025-11-07T10:30:45Z",
      "status": "pending"
    }
  ]
}
```

**实现逻辑**:
```go
func (s *MempoolService) GetStatus() (*MempoolStatus, error) {
    mempool := s.worker.mempool
    
    // 统计交易类型
    txTypes := s.countTxTypes()
    
    // 获取最近交易
    recentTxs := s.getRecentTxs(10)
    
    marked, unmarked := mempool.CountMarked()
    
    return &MempoolStatus{
        TotalSize:          mempool.GetSize(),
        MarkedTxs:          marked,
        UnmarkedTxs:        unmarked,
        TxTypes:            txTypes,
        AvgConfirmTime:     s.calculateAvgConfirmTime(),
        VerifySuccessRate:  s.calculateVerifySuccessRate(),
        MemoryUsage:        s.calculateMemoryUsage(),
        RecentTxs:          recentTxs,
    }, nil
}

func (mp *MemPool) CountMarked() (int, int) {
    mp.lock.Lock()
    defer mp.lock.Unlock()
    
    marked := 0
    unmarked := 0
    p := mp.order.Front()
    
    for p != nil {
        if p.Value.(*WrappedRawTx).proposed {
            marked++
        } else {
            unmarked++
        }
        p = p.Next()
    }
    
    return marked, unmarked
}
```

---

### 2.6 存储API

#### GET `/api/storage/status`
**功能**: 获取存储状态

**返回示例**:
```json
{
  "totalSize": 1073741824,
  "buckets": {
    "blocks": {
      "size": 536870912,
      "count": 12345
    },
    "executed": {
      "size": 268435456,
      "count": 12340
    },
    "utxo": {
      "size": 134217728,
      "count": 5678
    },
    "client": {
      "size": 67108864,
      "count": 1234
    }
  },
  "compressionRatio": 0.65,
  "vdfHeight": 12345,
  "blockCount": 12345
}
```

**实现逻辑**:
```go
func (s *StorageService) GetStatus() (*StorageStatus, error) {
    blockStorage := s.worker.blockStorage
    boltdb := blockStorage.GetBoltdb()
    
    buckets := make(map[string]BucketInfo)
    
    // 统计各个bucket的大小和数量
    boltdb.View(func(tx *bolt.Tx) error {
        buckets["blocks"] = s.getBucketInfo(tx, BlocksBucket)
        buckets["executed"] = s.getBucketInfo(tx, ExecutedBucket)
        buckets["utxo"] = s.getBucketInfo(tx, UTXOBucket)
        buckets["client"] = s.getBucketInfo(tx, ClientBucket)
        return nil
    })
    
    totalSize := s.calculateTotalSize(buckets)
    
    return &StorageStatus{
        TotalSize:        totalSize,
        Buckets:          buckets,
        CompressionRatio: s.calculateCompressionRatio(),
        VDFHeight:        blockStorage.vdfheight,
        BlockCount:       buckets["blocks"].Count,
    }, nil
}
```

---

### 2.7 历史数据API

func (s *StorageService) getBucketInfo(tx *bolt.Tx, bucketName string) BucketInfo {
    bucket := tx.Bucket([]byte(bucketName))
    if bucket == nil {
        return BucketInfo{}
    }
    
    size := 0
    count := 0
    
    bucket.ForEach(func(k, v []byte) error {
        size += len(k) + len(v)
        count++
        return nil
    })
    
    return BucketInfo{
        Size:  size,
        Count: count,
    }
}
```

---

### 2.7 历史数据API

#### GET `/api/metrics/history`
**功能**: 获取历史指标数据

**参数**:
- `metric`: 指标类型 (tps/difficulty/blocktime)
- `timeRange`: 时间范围 (1h/6h/24h/7d)
- `interval`: 采样间隔 (1m/5m/1h)

**返回示例**:
```json
{
  "metric": "tps",
  "timeRange": "24h",
  "interval": "5m",
  "data": [
    {
      "timestamp": "2025-11-07T10:00:00Z",
      "value": 45.6
    },
    {
      "timestamp": "2025-11-07T10:05:00Z",
      "value": 48.2
    }
  ],
  "avg": 46.5,
  "max": 85.3,
  "min": 12.1
}
```

**实现逻辑**:
```go
func (s *MetricsService) GetHistory(metric, timeRange, interval string) (*MetricsHistory, error) {
    duration := s.parseTimeRange(timeRange)
    intervalDuration := s.parseInterval(interval)
    
    startTime := time.Now().Add(-duration)
    endTime := time.Now()
    
    dataPoints := make([]DataPoint, 0)
    
    // 从时序数据库或缓存中查询
    switch metric {
    case "tps":
        dataPoints = s.queryTPS(startTime, endTime, intervalDuration)
    case "difficulty":
        dataPoints = s.queryDifficulty(startTime, endTime, intervalDuration)
    case "blocktime":
        dataPoints = s.queryBlockTime(startTime, endTime, intervalDuration)
    }
    
    // 计算统计数据
    avg, max, min := s.calculateStats(dataPoints)
    
    return &MetricsHistory{
        Metric:    metric,
        TimeRange: timeRange,
        Interval:  interval,
        Data:      dataPoints,
        Avg:       avg,
        Max:       max,
        Min:       min,
    }, nil
}
```

---

## 三、WebSocket实时推送

### 3.1 WebSocket Hub设计

```go
type WebSocketHub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

type Client struct {
    hub      *WebSocketHub
    conn     *websocket.Conn
    send     chan []byte
    filter   map[string]bool // 订阅的数据类型过滤器
}

func (h *WebSocketHub) Run() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
            
        case <-ticker.C:
            // 定时推送实时数据
            h.pushRealtimeData()
        }
    }
}
```

### 3.2 实时数据推送

```go
func (h *WebSocketHub) pushRealtimeData() {
    // 收集实时数据
    data := h.collectRealtimeData()
    
    // 序列化为JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return
    }
    
    // 广播给所有客户端
    h.broadcast <- jsonData
}

func (h *WebSocketHub) collectRealtimeData() *RealtimeData {
    return &RealtimeData{
        Timestamp:     time.Now(),
        CurrentHeight: h.worker.Height,
        CurrentTPS:    h.calculateCurrentTPS(),
        WorkFlag:      h.worker.workFlag,
        MempoolSize:   h.worker.mempool.GetSize(),
        VDF0Progress:  h.calculateVDF0Progress(),
        NetworkStatus: h.getNetworkStatus(),
    }
}
```

### 3.3 客户端订阅机制

**WebSocket连接**: `ws://localhost:8080/api/ws`

**订阅消息格式**:
```json
{
  "action": "subscribe",
  "types": ["pot", "committee", "mempool", "vdf"]
}
```

**实时推送消息格式**:
```json
{
  "type": "pot",
  "timestamp": "2025-11-07T10:30:45Z",
  "data": {
    "height": 12345,
    "tps": 45.6,
    "workFlag": true
  }
}
```

---

## 四、数据采集器设计

### 4.1 Metrics Collector

```go
type MetricsCollector struct {
    worker         *Worker
    nodeController *nodeController.NodeController
    storage        *MetricsStorage
    interval       time.Duration
}

func (mc *MetricsCollector) Start() {
    ticker := time.NewTicker(mc.interval)
    defer ticker.Stop()
    
    for range ticker.C {
        mc.collectMetrics()
    }
}

func (mc *MetricsCollector) collectMetrics() {
    metrics := &Metrics{
        Timestamp:     time.Now(),
        Height:        mc.worker.Height,
        Epoch:         mc.worker.epoch,
        Difficulty:    mc.worker.chainReader.GetLatest().GetHeader().Difficulty,
        TPS:           mc.calculateTPS(),
        MempoolSize:   mc.worker.mempool.GetSize(),
        VDF0Progress:  mc.calculateVDF0Progress(),
        NetworkStatus: mc.getNetworkStatus(),
    }
    
    // 存储到时序数据库或内存缓存
    mc.storage.Store(metrics)
}
```

### 4.2 数据存储方案

**选项1: 内存环形缓存**
```go
type RingBuffer struct {
    data  []Metrics
    size  int
    index int
    mu    sync.RWMutex
}

func (rb *RingBuffer) Push(m Metrics) {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    rb.data[rb.index] = m
    rb.index = (rb.index + 1) % rb.size
}
```

**选项2: 时序数据库 (InfluxDB/Prometheus)**
- 更适合长期存储和复杂查询
- 支持聚合和降采样

---

## 五、性能优化建议

### 5.1 缓存策略
```go
type CacheManager struct {
    systemCache    *cache.Cache // 系统概览缓存
    networkCache   *cache.Cache // 网络拓扑缓存
    storageCache   *cache.Cache // 存储状态缓存
}

// 设置不同的过期时间
func NewCacheManager() *CacheManager {
    return &CacheManager{
        systemCache:  cache.New(5*time.Second, 10*time.Second),
        networkCache: cache.New(10*time.Second, 20*time.Second),
        storageCache: cache.New(30*time.Second, 60*time.Second),
    }
}
```

### 5.2 并发控制
- 使用sync.RWMutex保护共享数据
- Worker Pool处理耗时计算
- 异步采集数据，避免阻塞API响应

### 5.3 数据压缩
- WebSocket消息使用gzip压缩
- 历史数据降采样（1分钟→5分钟→1小时）

---

## 六、安全性考虑

### 6.1 认证和授权
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !validateToken(token) {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 6.2 限流
```go
func RateLimitMiddleware(rps int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), rps*2)
    
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Too Many Requests"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 6.3 CORS配置
```go
func CORSMiddleware() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}
```

---

## 七、部署建议

### 7.1 API服务启动
```go
func StartVisualizationAPI(worker *Worker, nc *nodeController.NodeController, port int) {
    // 创建服务
    services := &Services{
        System:    NewSystemService(worker, nc),
        POT:       NewPOTService(worker),
        Committee: NewCommitteeService(worker, nc),
        BCI:       NewBCIService(worker),
        Mempool:   NewMempoolService(worker),
        Storage:   NewStorageService(worker),
        Network:   NewNetworkService(worker, nc),
        VDF:       NewVDFService(worker),
    }
    
    // 创建API服务器
    config := &ApiConfig{Port: port}
    log := logrus.WithField("service", "visualization-api")
    apiServer := NewVisualizationApiServer(config, services, log)
    
    // 启动WebSocket Hub
    hub := NewWebSocketHub(services)
    go hub.Run()
    
    // 启动Metrics采集器
    collector := NewMetricsCollector(worker, nc, 5*time.Second)
    go collector.Start()
    
    // 启动HTTP服务器
    apiServer.Start()
}
```

### 7.2 监控和日志
- 使用Prometheus监控API性能
- 结构化日志记录关键操作
- 错误追踪和报警

---

## 八、示例代码结构

```
internal/
  apis/
    visualization/
      handlers/
        system_handler.go
        pot_handler.go
        committee_handler.go
        bci_handler.go
        mempool_handler.go
        storage_handler.go
        websocket_handler.go
      services/
        system_service.go
        pot_service.go
        committee_service.go
        bci_service.go
        mempool_service.go
        storage_service.go
        metrics_service.go
      models/
        response_models.go
        request_models.go
      middleware/
        auth.go
        ratelimit.go
        cors.go
      websocket/
        hub.go
        client.go
      metrics/
        collector.go
        storage.go
      server.go
```

这个API实现方案提供了完整的后端支持，可以为前端可视化大屏提供实时和历史数据。
