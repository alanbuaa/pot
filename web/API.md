# POT共识可视化系统 API文档

## 概述

本文档描述了POT（Proof of Time）共识可视化系统的RESTful API接口。系统提供实时的区块链状态监控和可视化数据。

**版本**: 1.0.0  
**基础URL**: `http://localhost:10000/api`（node0）

---

## 接口列表

### 1. 系统概览

#### GET `/system/overview`

获取系统运行状态的整体概览信息。

**响应示例**:
```json
{
  "uptime": 86400,
  "totalNodes": 12,
  "onlineNodes": 10,
  "consensusTypes": ["POT", "SimpleWhirly"],
  "networkStatus": "healthy",
  "networkUtilization": 55.5,
  "currentHeight": 12345,
  "avgBlockTime": 5.2,
  "lastBlockTime": "2024-01-01T12:00:00Z"
}
```

**字段说明**:
- `uptime` (integer): 系统运行时长（秒）
- `totalNodes` (integer): 总节点数
- `onlineNodes` (integer): 在线节点数
- `consensusTypes` (array): 支持的共识类型列表
- `networkStatus` (string): 网络状态 - `healthy` | `warning` | `error`
- `networkUtilization` (number): 网络利用率百分比 (0-100)
- `currentHeight` (integer): 当前区块高度
- `avgBlockTime` (number): 平均出块时间（秒）
- `lastBlockTime` (string): 最新区块时间（ISO 8601格式）

---

### 2. POT共识状态

#### GET `/pot/status`

获取POT共识机制的当前运行状态。

**响应示例**:
```json
{
  "consensusType": "POT",
  "epoch": 89,
  "difficulty": "0x1a2b3c",
  "currentHeight": 12345,
  "workFlag": true,
  "timestamp": "2024-01-01T12:00:00Z",
  "nonce": 123456,
  "uncleCount": 2,
  "avgMiningTime": 3.5,
  "miningSuccessRate": 85.5
}
```

**字段说明**:
- `consensusType` (string): 共识类型，固定为 "POT"
- `epoch` (integer): 当前epoch
- `difficulty` (string): 挖矿难度值（16进制字符串）
- `currentHeight` (integer): 当前区块高度
- `workFlag` (boolean): 挖矿工作状态
- `timestamp` (string): 当前时间戳（ISO 8601格式）
- `nonce` (integer): Nonce值
- `uncleCount` (integer): 叔块数量
- `avgMiningTime` (number): 平均挖矿耗时（秒）
- `miningSuccessRate` (number): 挖矿成功率 (0-100)

---

### 3. VDF计算状态

#### GET `/pot/vdf`

获取可验证延迟函数(VDF)的实时计算状态。

**响应示例**:
```json
{
  "vdf0": {
    "progress": 75.5,
    "iterations": 75000,
    "status": "computing",
    "channelBuffer": 5
  },
  "vdf1": [
    {
      "workerId": 0,
      "progress": 60.5,
      "iterations": 60000,
      "status": "computing"
    },
    {
      "workerId": 1,
      "progress": 80.0,
      "iterations": 80000,
      "status": "done"
    }
  ],
  "vdfHalf": {
    "progress": 50.0,
    "iterations": 50000,
    "status": "computing",
    "channelBuffer": 3
  },
  "vdfChecker": {
    "status": "active",
    "verifyFailCount": 0
  },
  "abortStatus": false,
  "avgComputeTime": 45.5,
  "totalIterations": 500000,
  "cpuCounter": 4
}
```

**VDF工作器字段说明**:
- `progress` (number): 进度百分比 (0-100)
- `iterations` (integer): 迭代次数
- `status` (string): 计算状态 - `computing` | `done` | `idle`
- `channelBuffer` (integer): 通道缓冲区大小

**VDF线程字段说明**:
- `workerId` (integer): 工作线程ID (0-3)
- `progress` (number): 进度百分比 (0-100)
- `iterations` (integer): 迭代次数（可选）
- `status` (string): 计算状态 - `computing` | `done` | `idle`

**整体状态字段**:
- `vdfChecker.status` (string): 验证器状态 - `active` | `inactive`
- `vdfChecker.verifyFailCount` (integer): 验证失败次数
- `abortStatus` (boolean): 中止状态
- `avgComputeTime` (number): 平均计算耗时（毫秒）
- `totalIterations` (integer): 总迭代次数
- `cpuCounter` (integer): 并行工作线程数

---

### 4. 委员会状态

#### GET `/committee/status`

获取委员会共识机制的当前状态。

**响应示例**:
```json
{
  "consensusType": "SimpleWhirly",
  "committeeSize": 4,
  "committeeCount": 4,
  "confirmDelay": 6,
  "workHeight": 12339,
  "batchSize": 5,
  "inCommittee": true,
  "role": "member",
  "committee": [
    {
      "address": "0x1234567890abcdef1234567890abcdef12345678",
      "publicKey": "0xabcdef1234567890",
      "isLeader": true
    }
  ],
  "committeePublicKey": "0x1234567890abcdef",
  "shardings": [
    {
      "name": "Shard-0",
      "id": 0,
      "leaderAddress": "0x1234567890abcdef1234567890abcdef12345678",
      "committeeMembers": 4,
      "status": "active"
    }
  ],
  "workStage": "consensus",
  "timeout": 5000,
  "messageQueueLength": 10,
  "electionHeight": 12245
}
```

**字段说明**:
- `consensusType` (string): 共识类型 - `SimpleWhirly` | `CRWhirly` | `Whirly`
- `committeeSize` (integer): 委员会大小
- `committeeCount` (integer): 委员会数量
- `confirmDelay` (integer): 确认延迟（区块数）
- `workHeight` (integer): 委员会工作高度
- `batchSize` (integer): 批处理大小
- `inCommittee` (boolean): 是否在委员会中
- `role` (string): 节点角色 - `leader` | `member` | `observer`
- `workStage` (string): 工作阶段 - `init` | `shuffle` | `draw` | `share` | `consensus`
- `timeout` (integer): 超时配置（毫秒）
- `messageQueueLength` (integer): 消息队列长度
- `electionHeight` (integer): 选举区块高度

**委员会成员字段**:
- `address` (string): 成员地址（40位16进制）
- `publicKey` (string): 成员公钥
- `isLeader` (boolean): 是否为Leader

**分片字段**:
- `name` (string): 分片名称
- `id` (integer): 分片ID
- `leaderAddress` (string): Leader地址
- `committeeMembers` (integer): 委员会成员数
- `status` (string): 分片状态 - `active` | `inactive`

---

### 5. BCI激励状态

#### GET `/bci/status`

获取BCI (Blockchain Incentive) 激励系统的状态。

**响应示例**:
```json
{
  "totalReward": 65536,
  "lockedReward": 5000,
  "totalInterest": 2500,
  "rewardRatio": {
    "exchequer": 30,
    "miner": 50,
    "uncleBlockMiner": 2,
    "committeeLeader": 20,
    "committeeMember": 10
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
  "incentiveHeight": 12344,
  "pendingRewards": 50,
  "utxoCount": 1500,
  "totalDistributed": 50000
}
```

**字段说明**:
- `totalReward` (number): 总奖励金额
- `lockedReward` (number): 锁定的激励金额
- `totalInterest` (number): 累计利息总数
- `coinbaseLock` (integer): Coinbase锁定期（区块数）
- `executeHeight` (integer): 执行高度
- `incentiveHeight` (integer): 激励高度
- `pendingRewards` (number): 待分配奖励数量
- `utxoCount` (integer): UTXO总数
- `totalDistributed` (number): 历史累计分发总额

**奖励分配比例**:
- `rewardRatio.exchequer` (number): 国库占比 (0-100)
- `rewardRatio.miner` (number): 矿工占比 (0-100)
- `rewardRatio.uncleBlockMiner` (number): 叔块矿工占比 (0-100)
- `rewardRatio.committeeLeader` (number): 委员会Leader占比 (0-100)
- `rewardRatio.committeeMember` (number): 委员会成员占比 (0-100)

**锁定期利率**:
- `lockRates.saving` (number): 活期利率
- `lockRates.halfYear` (number): 半年期利率
- `lockRates.oneYear` (number): 一年期利率
- `lockRates.threeYears` (number): 三年期利率
- `lockRates.tenYears` (number): 十年期利率

---

### 6. 交易池状态

#### GET `/mempool/status`

获取交易池的实时状态信息。

**响应示例**:
```json
{
  "totalSize": 100,
  "markedTxs": 60,
  "unmarkedTxs": 40,
  "txTypes": {
    "normal": 70,
    "bci": 20,
    "devastate": 10
  },
  "avgConfirmTime": 8.5,
  "verifySuccessRate": 95.5,
  "memoryUsage": 10485760,
  "recentTxs": [
    {
      "hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
      "type": "normal",
      "timestamp": "2024-01-01T12:00:00Z",
      "status": "confirmed"
    }
  ]
}
```

**字段说明**:
- `totalSize` (integer): 交易池总大小
- `markedTxs` (integer): 已提议交易数
- `unmarkedTxs` (integer): 待提议交易数
- `avgConfirmTime` (number): 平均确认时间（秒）
- `verifySuccessRate` (number): 验证成功率 (0-100)
- `memoryUsage` (integer): 内存占用（字节）

**交易类型统计**:
- `txTypes.normal` (integer): 普通交易数
- `txTypes.bci` (integer): BCI交易数
- `txTypes.devastate` (integer): Devastate交易数

**最近交易字段**:
- `hash` (string): 交易哈希（64位16进制字符串）
- `type` (string): 交易类型 - `normal` | `bci` | `devastate`
- `timestamp` (string): 时间戳（ISO 8601格式）
- `status` (string): 交易状态 - `pending` | `confirmed`

---

### 7. 网络拓扑

#### GET `/network/topology`

获取P2P网络的拓扑结构信息。

**响应示例**:
```json
{
  "nodes": [
    {
      "id": 0,
      "peerId": "peer-committee-0",
      "address": "0x1234567890abcdef1234567890abcdef12345678",
      "status": "online",
      "connections": 8,
      "latency": 20,
      "type": "committee",
      "isLeader": true
    },
    {
      "id": 4,
      "peerId": "peer-pot-0",
      "address": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
      "status": "online",
      "connections": 3,
      "latency": 50,
      "type": "pot",
      "isLeader": false
    }
  ],
  "edges": [
    {
      "source": 0,
      "target": 1,
      "latency": 15,
      "messageCount": 500
    }
  ],
  "p2pAdaptorType": "libp2p",
  "subscribedTopics": ["blocks", "transactions", "consensus"],
  "messageQueueLength": 25,
  "networkBandwidth": 1048576
}
```

**节点字段说明**:
- `id` (integer): 节点ID（唯一标识符）
- `peerId` (string): PeerID
- `address` (string): 节点地址（40位16进制）
- `status` (string): 节点状态 - `online` | `offline`
- `connections` (integer): 连接数
- `latency` (integer): 延迟（毫秒）
- `type` (string): 节点类型 - `committee` | `pot`
- `isLeader` (boolean): 是否为Leader（仅committee类型）

**连接边字段说明**:
- `source` (integer): 源节点ID
- `target` (integer): 目标节点ID
- `latency` (integer): 连接延迟（毫秒）
- `messageCount` (integer): 消息数量

**网络信息字段**:
- `p2pAdaptorType` (string): P2P适配器类型 - `p2p` | `libp2p`
- `subscribedTopics` (array): 订阅的主题列表
- `messageQueueLength` (integer): 消息队列长度
- `networkBandwidth` (integer): 网络带宽（字节/秒）

---

## WebSocket实时推送

系统提供WebSocket接口用于实时数据推送：

**连接URL**: `ws://localhost:10000/api/ws`

**订阅主题**:
```json
{
  "action": "subscribe",
  "topics": ["pot", "vdf", "committee", "mempool", "network", "system", "bci"]
}
```

**消息格式**:
```json
{
  "type": "pot",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": { /* POTStatus数据 */ }
}
```

**支持的消息类型**:
- `pot`: POT共识状态更新
- `vdf`: VDF计算状态更新
- `committee`: 委员会状态更新
- `mempool`: 交易池状态更新
- `network`: 网络拓扑更新
- `system`: 系统概览更新
- `bci`: BCI激励状态更新

---

## 错误响应

所有接口在发生错误时返回统一的错误格式：

```json
{
  "code": 500,
  "message": "Internal Server Error",
  "details": "Database connection failed"
}
```

**常见HTTP状态码**:
- `200`: 成功
- `400`: 请求参数错误
- `404`: 资源不存在
- `500`: 服务器内部错误

---

## 开发建议

### Mock模式

系统支持Mock模式，适用于开发和测试：

在 `.env` 文件中设置：
```
VITE_USE_MOCK=true
```

### 轮询策略

推荐的数据轮询频率：
- **1秒**: 系统概览、VDF状态
- **5秒**: POT状态、委员会状态、交易池状态
- **10秒**: 网络拓扑、BCI状态

### 最佳实践

1. **优先使用WebSocket**: 对于需要实时更新的数据，建议使用WebSocket订阅而非轮询
2. **错误处理**: 所有API调用都应该有适当的错误处理机制
3. **数据缓存**: 对于变化不频繁的数据（如网络拓扑），建议在客户端进行适当缓存
4. **带宽优化**: 根据实际需求选择订阅的主题，避免接收不必要的数据

---

## 版本历史

- **v1.0.0** (2024-11-24): 初始版本，包含核心可视化数据接口

---

## 联系方式

如有问题或建议，请联系：
- Email: contact@pot.io
- GitHub: https://github.com/alanbuaa/pot

---

*本文档最后更新时间: 2024-11-24*
