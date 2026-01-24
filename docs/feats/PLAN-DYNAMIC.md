# Plan: 非动态共识成员发现与维护方案

针对POT可升级共识框架，设计一个支持固定节点集的非动态共识成员管理系统，区别于动态共识（如POT委员会轮换），确保配置驱动的节点发现、连接管理和共识切换时的成员一致性。

## Steps

1. **在[consensus/upgrade/manager.go](consensus/upgrade/manager.go)中扩展`UpgradeProposal`结构** - 添加`MembershipConfig`字段，包含目标共识的完整节点列表（ID、地址、公钥）、是否动态标志、拓扑模式（全连接/部分连接）

2. **创建[consensus/membership/](consensus/membership/)模块** - 实现`StaticMembershipManager`，负责节点列表加载、验证、持久化；提供`GetMembers()`, `ValidateMember()`, `SyncMembership()`接口；支持从配置文件和`UpgradeProposal`初始化

3. **在[p2p/p2p.go](p2p/p2p.go)和[p2p/p2p-adaptor/](p2p/p2p-adaptor/)中添加连接预热机制** - 扩展`P2PAdaptor`接口增加`PrepareConnections(peers []PeerInfo, consensusID int64)方法`；在共识切换前（PreexecPhase）建立目标节点连接；复用已有连接池避免重复握手

4. **在[consensus/upgrade/manager.go](consensus/upgrade/manager.go)的切换流程中集成成员验证** - 在`SwitchPhase`前验证所有节点是否拥有相同的`MembershipConfig`（通过哈希比对）；在`buildNewConsensusConfig()`中注入`MembershipConfig`到新共识实例；失败时触发回滚

5. **修改[consensus/consensusFactory.go](consensus/consensusFactory.go)的`BuildConsensus`** - 根据`MembershipConfig.IsDynamic`标志选择初始化路径；非动态共识强制使用静态节点列表；动态共识保留现有DHT发现逻辑；为BaseP2p和libp2p分别适配

6. **在[config/config.go](config/config.go)中添加成员管理配置** - 扩展`ConsensusConfig`添加`Membership`字段（拓扑类型、连接超时、心跳间隔）；支持三种配置模式：单节点开发模式、多节点静态模式、升级提案模式

## Further Considerations

1. **节点列表更新策略** - 非动态共识是否允许运行时添加观察者节点？是否支持节点临时下线后重新加入？建议：核心验证节点固定，支持只读观察者动态加入

2. **P2P层多共识复用** - 当前`consensusID`参数已存在于`Unicast`，是否为每个共识实例创建独立Topic？建议：主链和候选链共享BaseP2p连接，通过consensusID路由消息

3. **成员配置冲突处理** - 如果节点配置的成员列表与`UpgradeProposal`不一致？建议：优先级 UpgradeProposal > 本地配置，治理投票阶段检测冲突并拒绝提案

## 上下文信息

### 1. P2P网络的核心接口和方法

**P2PAdaptor接口**:
- `Broadcast(msgByte []byte, consensusID int64, topic []byte) error` - 广播消息
- `Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error` - 单播消息
- `SetReceiver(ch chan<- []byte)` - 设置消息接收通道
- `Subscribe(topic []byte) error` - 订阅主题
- `UnSubscribe(topic []byte) error` - 取消订阅
- `GetNodeID() int64` - 获取节点ID
- `GetP2PType() string` - 获取P2P类型
- `Stop()` - 停止网络服务

**两种P2P实现**:
1. **BaseP2p**: 基于gRPC的简单P2P，直接使用节点地址通信，维护节点列表
2. **P2P-Adaptor**: 基于libp2p的高级实现
   - 使用Kademlia DHT进行节点发现
   - 支持GossipSub协议
   - Bootstrap节点机制
   - 通过`findTopicPeers()`发现订阅同一主题的节点

### 2. 节点配置和管理的当前实现方式

**配置结构**:
```go
type NodeInfo struct {
    ID               int64
    Address          string         // P2P地址 (e.g., 127.0.0.1:8081)
    RpcAddress       string         // RPC地址
    BciRpcAddress    string         // BCI RPC地址
    PrivateKeyPath   string         // 私钥路径
    PublicKeyPath    string         // 公钥路径
    ApiServerAddress string         // API服务器地址
    DataDir          string         // 数据目录
}

type ConsensusConfig struct {
    Node  *NodeInfo           // 单节点配置
    Nodes []*config.NodeInfo  // 多节点配置（仅部分共识使用）
    Total int                 // 总节点数
    Fault int                 // 容错节点数
    // ...
}
```

**关键发现**:
- **当前架构**: 单节点配置模式 - 每个节点只知道自己的信息
- **静态节点列表**: `Nodes`存在但**仅在部分共识中使用** (如POT、HotStuff)
- **动态vs非动态标志**: config.yaml中POT配置有`dynamic: true`字段（目前未完全实现）

### 3. 共识升级时节点列表如何传递和使用

**共识创建流程**:
```go
BuildConsensus(nid, cid, cfg, exec, p2pAdaptor, log)
  ├─ cfg.Nodes 传递给具体共识实现
  ├─ HotStuff: 遍历 cfg.Nodes 广播消息
  ├─ POT: 使用 cfg.Nodes[nid] 获取节点信息
  └─ 配置继承: cfg.Upgradeable.InitConsensus.Nodes = cfg.Nodes
```

**升级管理器**:
- `ConsensusFactory.CreateConsensus()` 基于`UpgradeProposal`创建新共识实例
- 新共识配置通过配置文件构建
- **关键问题**: 当前没有明确的节点列表更新机制

### 4. 现有的节点发现和连接机制

**libp2p方式**:
```go
findTopicPeers(topic):
  1. 通过DHT广播自己对topic的兴趣 (Advertise)
  2. 查找订阅同一topic的其他节点 (FindPeers)
  3. 连接发现的节点 (host.Connect)
  4. 加入GossipSub主题
```

**Bootstrap流程**:
- 支持配置Bootstrap节点列表或使用IPFS默认Bootstrap节点
- 启动时连接Bootstrap节点建立DHT
- 通过DHT发现其他节点

**BaseP2p方式**:
- 静态节点列表
- 直接通过地址建立gRPC连接
- 广播时遍历所有节点列表

### 5. 动态vs非动态共识需要考虑的差异点

#### **非动态共识特点**:
- **固定成员集**: 节点集合在共识运行期间不变（如HotStuff、POW）
- **配置驱动**: 节点列表通过配置文件预定义
- **确定性**: 所有节点知道完整的成员列表
- **投票权重**: 基于节点数量计算（1/3、1/2容错模型）

#### **动态共识特点**:
- **变化成员集**: 节点可以加入/离开（如POT的委员会轮换）
- **运行时更新**: 成员列表在共识运行中更新
- **不确定性**: 节点可能不知道完整的网络拓扑
- **epoch/sharding**: POT通过epoch选举委员会

#### **升级场景差异**:

| 维度 | 非动态共识 | 动态共识 |
|------|-----------|---------|
| **节点发现** | 通过配置文件静态配置 | 需要运行时发现机制（DHT/Gossip） |
| **成员验证** | 预共享公钥列表 | 需要动态验证机制（门限签名/BLS聚合） |
| **切换同步** | 所有节点必须在同一高度切换 | 可能需要处理部分节点滞后 |
| **消息路由** | 直接广播到所有已知节点 | 基于主题订阅路由 |
| **状态一致性** | 强一致性要求 | 可能需要最终一致性 |

#### **关键挑战**:
1. **升级提案传播**: 非动态共识如何确保所有节点接收升级提案？
2. **配置同步**: 新共识的节点列表如何分发和同步？
3. **连接建立**: 切换后如何快速建立新共识的节点连接？
4. **旧消息处理**: MessageCache已实现缓存，但如何处理节点列表变化的消息？
5. **P2P层复用**: 同一P2P连接能否支持多个共识实例？
