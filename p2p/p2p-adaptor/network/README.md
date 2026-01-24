## Network 模块

基于 libp2p 的底层网络通信模块，提供 P2P 网络基础设施。

---

## 📦 核心组件

### 1. Network 结构体

封装 libp2p 核心功能：

```go
type Network struct {
    host host.Host              // libp2p Host
    dht  *kaddht.IpfsDHT        // Kademlia DHT（节点发现）
    ctx  context.Context         // 全局上下文
    ps   *pubsub.PubSub         // GossipSub（主题订阅）
}
```

**主要方法**：
- `NewNetwork(port, dhtPath, keyPath, bootstrapStrings, isWithScore)` - 创建网络实例
- `JoinTopic(topicName, isOnlyGossip)` - 加入 PubSub 主题
- `GetHost()` / `GetDHT()` / `GetPeerID()` - 获取底层对象

### 2. AppScore（节点评分系统）

实现 GossipSub 应用层评分机制：

```go
type AppScore struct {
    feedbackchan chan *ValidationFeedback  // 消息验证反馈
    // 评分衰减参数
    DecayMessageDeliveries        float64
    DecayInvalidMessageDeliveries float64
}
```

**功能**：
- 追踪消息传递质量
- 惩罚无效消息发送者
- 自动衰减评分（防止历史污染）

### 3. Evidence（传输证明）

**状态**：❌ **接口已定义，但未实际使用**

```go
type TransmissionEvidence interface {
    VerifyEvidence() (bool, error)
    GetShareList() ([][]byte, error)
}
```

**BLS 实现**（`evidence_bls.go`）：
- `BLSEvidence` - BLS 聚合签名的传输证明
- `BLSManagement` - 证明管理器（添加、验证）
- 用途：追踪消息转发路径，分配激励份额（0-1）

**未使用原因**：
- `Message.IsAddTransferEvidence` 仅为占位符
- 测试代码已全部注释（`evidence_bls_test.go`）
- 无模块调用 `AddEvidence` / `VerifyEvidence`

---

## 🔍 节点发现机制

### Kademlia DHT 实现

基于 Kademlia 算法的分布式哈希表（DHT），提供去中心化节点发现能力。

#### 1. Bootstrap 引导发现

**初始化流程**：

```go
func bootstrap(host, dht, bootstrappeers, ctx) error {
    // 1. 启动 DHT 服务
    dht.Bootstrap(ctx)
    
    // 2. 并发连接引导节点
    for _, peerAddr := range bootstrappeers {
        peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
        host.Connect(ctx, *peerinfo)  // 2 秒超时
    }
    
    // 3. 至少连接一个节点才能成功
    if !anybootstrappeer {
        return errors.New("can not find or connect to a bootstrap peer")
    }
}
```

**引导节点选择**：

```go
// 默认：IPFS 公网引导节点
if bootstrapStrings == nil {
    BootstrapPeers = kaddht.DefaultBootstrapPeers
} else {
    // 自定义：私有网络引导节点
    BootstrapPeers, err = BuildBootstrapPeers(bootstrapStrings)
}
```

**公网引导节点示例**：
- `/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN`
- `/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ`

#### 2. Topic-based 发现

**主题订阅时自动发现**：

```go
func (n *Network) findTopicPeers(topic string) {
    // 1. 广播兴趣：告诉 DHT 自己订阅了该主题
    routingDiscovery := drouting.NewRoutingDiscovery(n.dht)
    dutil.Advertise(n.ctx, routingDiscovery, topic)
    
    // 2. 查找订阅者：从 DHT 查询订阅相同主题的节点
    peerChan := routingDiscovery.FindPeers(n.ctx, topic)
    
    // 3. 建立连接
    for peer := range peerChan {
        if peer.ID != n.host.ID() {  // 跳过自己
            n.host.Connect(n.ctx, peer)
        }
    }
}
```

**特性**：
- **主题隔离**：只发现订阅相同主题的节点
- **持续发现**：定期刷新 DHT 广播（`Advertise` TTL 默认 2 小时）
- **容错设计**：连接失败不影响后续节点

#### 3. Content ID (CID) 查询

**通过内容哈希查找提供者**：

```go
func (n *Network) QueryPeersWithCID(id cid.Cid) ([]peer.AddrInfo, error) {
    limit := 100
    peerChan := n.dht.FindProvidersAsync(n.ctx, id, limit)
    
    var peers []peer.AddrInfo
    for peerAddr := range peerChan {
        peers = append(peers, peerAddr)
    }
    return peers, nil
}
```

**用途**：
- 发布内容（`ProvideContent`）
- 查询提供者（`QueryPeersWithCID`）
- 支持内容寻址网络

### 发现参数配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| DHT Mode | `kaddht.ModeAuto` | 自动选择 Client/Server 模式 |
| 连接超时 | 2 秒 | `libp2p.WithDialTimeout(2*time.Second)` |
| 发现重试 | 单次 | `count > 0` 即停止（`findTopicPeers`） |
| Advertise TTL | 2 小时 | `dutil.Advertise` 默认值 |

### 单节点启动优化

**快速启动策略**：

```go
// 减少重试次数（原 count > 2）
if anyConnected || count > 0 {
    break  // 连接到任意节点即可继续
}
```

**场景**：
- ✅ 开发测试（单节点调试）
- ✅ 首个启动节点（无其他 peer）
- ⚠️ 生产环境建议等待多个节点

---

## ⚙️ 配置说明

### 引导节点

**默认行为**：连接到 **IPFS 公网引导节点**

```go
if bootstrapStrings == nil {
    BootstrapPeers = kaddht.DefaultBootstrapPeers  // ← IPFS 公网
} else {
    BootstrapPeers, err = BuildBootstrapPeers(bootstrapStrings)
}
```

**私有网络配置**：

```go
// 传入自定义引导节点列表
privateBootstrap := []string{
    "/ip4/192.168.1.100/tcp/6060/p2p/QmPeerID1",
    "/ip4/192.168.1.101/tcp/6060/p2p/QmPeerID2",
}
network, _, err := NewNetwork("6060", dhtPath, keyPath, privateBootstrap, false)
```

### Topic 评分参数

```go
scoreParam := &pubsub.TopicScoreParams{
    TopicWeight:                  1,
    TimeInMeshWeight:             0.0002777,  // P1: 网格时间
    FirstMessageDeliveriesWeight: 1,          // P2: 首次传递
    MeshMessageDeliveriesWeight:  -1,         // P3: 网格传递
    InvalidMessageDeliveriesWeight: -1,       // P4: 无效消息惩罚
}
```

---

## 🔌 外部依赖

- `github.com/libp2p/go-libp2p` - P2P 网络核心
- `github.com/libp2p/go-libp2p-kad-dht` - Kademlia DHT
- `github.com/libp2p/go-libp2p-pubsub` - GossipSub 协议
- `github.com/ipfs/go-ds-leveldb` - LevelDB 持久化（DHT）
- `github.com/chuwt/chia-bls-go` - BLS 签名（Evidence 模块）
- `github.com/ethereum/go-ethereum/crypto` - ECDSA 签名（Evidence 模块）

---

## 📚 使用指南

**本模块是底层基础设施**，通常不直接使用，而是通过 `p2padaptor` 适配器层调用。

如需直接使用：

```go
import "network"

// 创建网络
net, feedbackCh, err := network.NewNetwork("6060", "./dht", "./key", nil, false)

// 加入主题
topic, sub, err := net.JoinTopic("my-topic", false)

// 监听消息
go func() {
    for {
        msg, err := sub.Next(context.Background())
        // 处理消息
    }
}()

// 发布消息
topic.Publish(context.Background(), []byte("hello"))
```

---

## 🔍 模块定位

- **独立复用**：可作为通用 libp2p 封装库
- **无业务耦合**：不依赖 `p2padaptor` 或共识层
- **扩展性**：预留 Evidence 接口供未来激励系统使用
