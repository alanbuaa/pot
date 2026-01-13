# 消息缓存器使用指南

## 概述

消息缓存器用于处理**非同步网络**场景下的消息可达性问题，特别是当节点收到的消息epoch比当前处理的epoch更高时。

## 核心功能

### 1. 消息缓存（基于Epoch）

当节点收到消息时，会根据以下规则处理：

```go
// 场景1: 消息epoch < 当前epoch => 直接丢弃（过时消息）
// 场景2: 消息epoch == 当前epoch && 共识已运行 => 直接转发
// 场景3: 消息epoch >= 当前epoch && 共识未运行 => 缓存起来
```

### 2. 消息结构

每个缓存的消息包含：

```go
type CachedMessage struct {
    Message              interface{} // 实际消息内容
    ConsensusID          int64       // 发送者共识ID
    ReceiverConsensusID  int64       // 接收者共识ID（关键）
    Epoch                uint64      // 消息所属epoch（关键）
    ReceivedTime         time.Time
    ProcessedAfterSwitch bool
}
```

## 使用示例

### 示例1: 网络层集成

在P2P网络模块收到消息时调用：

```go
// 在 p2p/server.go 或类似文件中
func (s *P2PServer) OnReceiveMessage(rawMsg []byte) {
    // 解析消息
    msg := ParseMessage(rawMsg)
    
    // 提取关键字段
    senderConsensusID := msg.GetSenderConsensusID()
    receiverConsensusID := msg.GetReceiverConsensusID()
    epoch := msg.GetEpoch()
    
    // 交给升级管理器处理
    if err := s.upgradeManager.OnNetworkMessage(
        msg,
        senderConsensusID,
        receiverConsensusID,
        epoch,
    ); err != nil {
        log.WithError(err).Warn("Failed to process network message")
    }
}
```

### 示例2: Epoch更新触发消息处理

当共识推进到新的epoch时：

```go
// 在共识模块中
func (c *Consensus) OnEpochChange(newEpoch uint64) {
    // 通知升级管理器更新epoch
    c.upgradeManager.SetCurrentEpoch(newEpoch)
    
    // SetCurrentEpoch内部会自动处理缓存的该epoch消息：
    // 1. 从缓存中取出epoch == newEpoch的所有消息
    // 2. 逐个转发给对应的共识实例
}
```

### 示例3: 候选共识启动时冲刷缓存

当候选共识启动时，自动转发之前缓存的消息：

```go
// 在 UpgradeManager 中
func (um *UpgradeManager) StartUpgrade(proposal *UpgradeProposal, newConsensus model.Consensus) error {
    // ... 启动候选链 ...
    
    err := um.multiChainManager.StartCandidateChain(candidateID, forkHeight, newConsensus)
    if err != nil {
        return err
    }
    
    // 候选共识启动后，自动冲刷缓存
    // 这个调用已经集成在代码中，无需手动调用
    um.OnCandidateConsensusStarted(newConsensus.GetConsensusID())
    
    // 内部会：
    // 1. 取出所有 receiverConsensusID == newConsensusID 的消息
    // 2. 逐个转发给 newConsensus
    // 3. 从缓存和持久化存储中删除
}
```

### 示例4: 共识切换后清理旧消息

当共识切换完成时，清理旧epoch的消息：

```go
// 在切换逻辑中
func (um *UpgradeManager) ExecuteSwitch() error {
    oldEpoch := um.currentEpoch
    
    // ... 执行切换逻辑 ...
    
    // 切换完成后清理旧epoch消息
    um.OnConsensusSwitched(oldEpoch)
    
    // 内部会：
    // 1. 清理内存中 epoch == oldEpoch 的消息
    // 2. 清理持久化存储中的对应消息
    // 3. 保留更高epoch的消息（为后续升级预留）
}
```

## 工作流程图

```
节点收到消息
    ↓
判断消息类型
    ↓
├─ 控制消息（升级相关） → handleControlMessage()
└─ 普通消息（共识通信）
       ↓
   判断epoch
       ↓
   ├─ epoch < 当前epoch → 丢弃
   ├─ epoch == 当前epoch
   │      ↓
   │  共识已运行? 
   │      ↓
   │  ├─ 是 → 直接转发
   │  └─ 否 → 缓存
   └─ epoch > 当前epoch → 缓存
              ↓
          等待epoch到达
              ↓
      SetCurrentEpoch(newEpoch)
              ↓
      自动取出并转发该epoch消息
```

## 配置说明

### 缓存大小

```go
// 默认最大缓存10000条消息
messageCache.SetMaxCacheSize(10000)
```

### 消息超时

```go
// 默认5分钟超时
messageCache.SetMessageTimeout(5 * time.Minute)

// 启动自动清理任务（每分钟清理一次过期消息）
stopChan := messageCache.StartCleanupTask(1 * time.Minute)

// 停止清理任务
close(stopChan)
```

## 性能考虑

### 内存占用

- 每条缓存消息约占用 100-500 字节（取决于消息大小）
- 10000条消息约占用 1-5 MB 内存
- 建议根据节点配置调整 `maxCacheSize`

### 持久化策略

```go
// 写入策略：收到消息时同步写入LevelDB
messageCache.CacheMessage(msg, senderID, receiverID, epoch)
// ↓ 内部调用
bufmsgStorage.Store(msg)  // LevelDB写入

// 恢复策略：节点重启时自动加载
// 在 NewUpgradeManagerWithPersistence 中自动恢复
```

### 查询性能

- `GetMessagesByEpoch(epoch)` - O(N) 时间复杂度
- `PopAllForConsensus(consensusID)` - O(N) 时间复杂度
- `DropByEpoch(epoch)` - O(N) 时间复杂度

其中 N 为缓存消息总数，通常 < 10000。

## 监控和调试

### 获取缓存统计

```go
stats := messageCache.GetCacheStats()

fmt.Printf("新共识消息数: %d\n", stats.NewConsensusMessageCount)
fmt.Printf("旧共识消息数: %d\n", stats.OldConsensusMessageCount)
fmt.Printf("最大缓存大小: %d\n", stats.MaxCacheSize)
fmt.Printf("是否已切换: %v\n", stats.Switched)
fmt.Printf("切换高度: %d\n", stats.SwitchHeight)
```

### 日志跟踪

启用DEBUG级别日志可查看详细的缓存操作：

```go
log.SetLevel(logrus.DebugLevel)

// 日志示例：
// [DEBUG] Cached new consensus message (pre-switch) key=1_123_1234567890
// [INFO] Popped cached messages for consensus consensus_id=2 count=15
// [INFO] Processing cached messages for current epoch epoch=100 count=5
```

## 注意事项

1. **消息必须携带必要字段**
   - `receiverConsensusID`: 接收者共识ID
   - `epoch`: 消息所属epoch/时段

2. **控制消息不缓存**
   - 升级配置交易、升级确认交易等控制消息直接处理
   - 仅普通共识通信消息进入缓存

3. **同步网络不使用缓存**
   - 如果网络是同步的，未启动共识的消息直接丢弃
   - 缓存机制仅用于非同步网络（半同步+异步）

4. **Epoch必须单调递增**
   - 系统假设epoch值随时间单调递增
   - 不支持epoch回退

## 测试

查看测试用例了解详细用法：

```bash
# 运行消息缓存测试
go test -v ./consensus/upgrade/ -run TestMessageCache

# 运行持久化存储测试
go test -v ./internal/storage/ -run TestMessageCacheStorage
```

## 故障排查

### 问题1: 消息丢失

**症状**: 候选共识启动后没有收到消息

**排查步骤**:
1. 检查消息是否正确设置了 `receiverConsensusID`
2. 检查缓存统计: `GetCacheStats()`
3. 查看日志中是否有 "Cached new consensus message" 记录
4. 确认 `OnCandidateConsensusStarted()` 是否被调用

### 问题2: 内存占用过高

**症状**: 节点内存持续增长

**排查步骤**:
1. 检查缓存大小: `GetCacheStats().NewConsensusMessageCount`
2. 确认清理任务是否运行: `StartCleanupTask()`
3. 检查是否有消息超时未清理
4. 考虑降低 `maxCacheSize` 或 `messageTimeout`

### 问题3: 消息重复处理

**症状**: 同一消息被处理多次

**排查步骤**:
1. 确认 `PopAllForConsensus()` 会删除消息
2. 检查是否多次调用 `OnCandidateConsensusStarted()`
3. 查看日志中 "Popped cached messages" 的调用次数

## 进一步阅读

- [upgrade-protocol-impl.md](../../docs/upgrade_plan/upgrade-protocol-impl.md#53-消息缓存机制bufmsg非同步网络专用) - 设计文档
- [upgrade-protocol-academic.md](../../docs/upgrade_plan/upgrade-protocol-academic.md) - 学术论文
- [message_cache.go](message_cache.go) - 源代码
- [message_cache_storage.go](../../internal/storage/message_cache_storage.go) - 持久化实现
