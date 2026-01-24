

## P2P 网络模块

基于 libp2p 的点对点网络通信层，提供广播、单播和节点发现能力，支持 POT 共识系统的网络通信需求。

---

## 📁 模块结构

```
p2p-adaptor/
├── network/          # 底层网络基础设施（libp2p 封装）
│   ├── network.go    # Network 核心：DHT、PubSub、引导节点
│   ├── score.go      # 节点评分系统（Peer Score）
│   ├── evidence.go   # 传输证明接口定义（未启用）
│   └── evidence_bls.go  # BLS 签名实现（未启用）
└── p2padaptor/       # 上层适配器（业务抽象）
    ├── networkadaptor.go  # NetworkAdaptor：主题订阅、节点发现
    └── unicastadaptor.go  # UnicastAdapter：点对点通信
```

### 模块依赖关系

```
┌─────────────────────────────┐
│  consensus/upgrade/...      │  共识系统
└──────────┬──────────────────┘
           │ 使用
┌──────────▼──────────────────┐
│  p2padaptor/                │  适配器层（组合 Network）
│  - NetworkAdaptor           │
│  - UnicastAdapter           │
└──────────┬──────────────────┘
           │ 依赖
┌──────────▼──────────────────┐
│  network/                   │  基础设施层
│  - Network (libp2p 封装)    │
│  - DHT + PubSub             │
└──────────┬──────────────────┘
           │ 依赖
┌──────────▼──────────────────┐
│  libp2p, pubsub, DHT        │  第三方库
└─────────────────────────────┘
```

**设计原则**：
- `network/` 提供可复用的 libp2p 基础能力（独立模块）
- `p2padaptor/` 封装业务抽象，简化共识层调用
- 单向依赖：`p2padaptor → network`，`network` 不依赖 `p2padaptor`

---

## ⚠️ 网络连接说明

**默认行为**：系统会连接到 **IPFS 公网引导节点**（`kaddht.DefaultBootstrapPeers`）

如需使用私有网络，请修改 `p2padaptor/networkadaptor.go` 中的 `NewNetworkAdaptor` 函数：

```go
// 传入自定义引导节点（替换 nil）
privateBootstrap := []string{
    "/ip4/192.168.1.100/tcp/6060/p2p/QmYourPeerID",
}
network, _, err := net.NewNetwork(netPort, dhtPath, keyPath, privateBootstrap, false)
```

---

## 🧩 核心接口

### 1. P2PAdaptor 接口

```go
type P2PAdaptor interface {
	// ------ For broadcast ------ //

	// Subscribe to the topic you are interested in
	Subscribe(topic []byte) error

	// UnSubscribe to topic
	UnSubscribe(topic []byte) error

	// Send a message to a subscribed topic, and the message
	// will be broadcast within the topic
	Broadcast(msgByte []byte, consensusID int64, topic []byte) error

	// ------- For Unicast ------- //

	// Start the unicast service and receive unicast
	// messages from other peers
	StartUnicast() error

	// Set up a channel to receive unicast messages from other peers
	SetUnicastReceiver(receiver MsgReceiver)

	// Send a unicast message to a given node, attempting
	// to find it and establish a connection with it
	Unicast(address string, msgByte []byte, consensusID int64) error

	// ------- For Network ------- //

	// Get the PeerID of the local node
	GetPeerID() string

	// Stop network
	Stop()
}
```

**主要功能**：
- **广播**：通过 PubSub Topic 向所有订阅者发送消息
- **单播**：点对点直接发送消息到指定 Peer
- **节点发现**：基于 DHT 的自动节点发现

---

## 🚀 使用方式

### 1. Go Module 配置

在项目 `go.mod` 中添加本地模块引用：

```go
replace p2padaptor => ./p2p/p2p-adaptor/p2padaptor
replace network => ./p2p/p2p-adaptor/network
```

### 2. 基本使用示例

```go
import "p2padaptor"

// 初始化网络适配器
adaptor, err := p2padaptor.NewNetworkAdaptor("6060", "consensus-topic", "./data")
if err != nil {
    log.Fatal(err)
}

// 订阅主题
err = adaptor.Subscribe([]byte("consensus-topic"))

// 启动单播服务
err = adaptor.StartUnicast()

// 设置消息接收通道
msgCh := make(chan []byte, 100)
adaptor.SetReceiver(msgCh)

// 广播消息
adaptor.Broadcast([]byte("hello"), 1, []byte("consensus-topic"))

// 单播消息
adaptor.Unicast("QmPeerID...", []byte("direct message"), 1, []byte("topic"))
```

---

## � 节点发现机制

### 双层发现策略

系统采用 **DHT + Topic-based** 双层节点发现机制：

```
1. DHT 引导发现（Bootstrap）
   ↓
2. Topic 订阅发现（Advertise）
   ↓
3. 回调通知业务层
```

### 1. DHT 引导发现

**启动阶段**：连接引导节点建立 DHT 网络

```go
// network/network.go
func bootstrap(host, dht, bootstrappeers, ctx) {
    dht.Bootstrap(ctx)  // 启动 DHT
    
    // 并发连接所有引导节点
    for _, peerAddr := range bootstrappeers {
        host.Connect(ctx, peerInfo)
    }
}
```

**特点**：
- 快速构建分布式路由表
- 支持跨子网节点发现
- 依赖引导节点列表（公网或私有）

### 2. Topic 订阅发现

**订阅主题时**：通过 DHT 查找订阅相同主题的节点

```go
// network/network.go
func findTopicPeers(topic string) {
    routingDiscovery := drouting.NewRoutingDiscovery(dht)
    
    // 1. 广播自己对该主题感兴趣
    dutil.Advertise(ctx, routingDiscovery, topic)
    
    // 2. 查找其他订阅者
    peerChan := routingDiscovery.FindPeers(ctx, topic)
    
    // 3. 尝试连接
    for peer := range peerChan {
        host.Connect(ctx, peer)
    }
}
```

**特点**：
- 主题隔离（只发现同主题节点）
- 自动重试机制
- 容忍单节点启动（count > 0 即通过）

### 3. 高级发现 API（NetworkAdaptor）

**主动发现节点**：

```go
// p2padaptor/networkadaptor.go

// 启动节点发现进程
err := adaptor.DiscoverPeers()

// 注册发现回调
adaptor.RegisterNodeDiscoveryCallback(&MyCallback{})

// 等待指定数量节点上线
err := adaptor.WaitForNodes(3, 30)  // 等待 3 个节点，超时 30 秒
```

**回调接口**：

```go
type NodeDiscoveryCallback interface {
    OnNodeDiscovered(nodeID int64, address string) error
    OnNodeLost(nodeID int64) error
}
```

**实现示例**：

```go
type MyCallback struct{}

func (c *MyCallback) OnNodeDiscovered(nodeID int64, address string) error {
    fmt.Printf("New node joined: %d (%s)\n", nodeID, address)
    return nil
}

func (c *MyCallback) OnNodeLost(nodeID int64) error {
    fmt.Printf("Node lost: %d\n", nodeID)
    return nil
}
```

### 发现流程图

```
┌──────────────┐
│ 启动节点     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 连接引导节点 │ ← bootstrap(DefaultBootstrapPeers)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 加入 Topic   │ ← JoinTopic("consensus-topic")
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ DHT 查找节点 │ ← FindPeers(topic)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 触发回调     │ ← OnNodeDiscovered()
└──────────────┘
```

### 关键配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 发现间隔 | 10秒 | `DiscoverPeers` 重复查询间隔 |
| 连接超时 | 2秒 | 单个节点连接超时 |
| 重试次数 | 无限 | DHT 查询持续进行 |
| 单节点模式 | 支持 | `count > 0` 即可启动 |

### 私有网络节点发现

**场景**：不依赖 IPFS 公网，仅在私有网络发现节点

```go
// 1. 配置私有引导节点
privateBootstrap := []string{
    "/ip4/192.168.1.100/tcp/6060/p2p/QmNode1",
    "/ip4/192.168.1.101/tcp/6060/p2p/QmNode2",
}

// 2. 创建网络（关闭公网连接）
network, _, err := net.NewNetwork(port, dhtPath, keyPath, privateBootstrap, false)

// 3. 节点会自动通过私有 DHT 相互发现
```

**注意**：至少需要一个在线的引导节点，否则 DHT 无法初始化。

---

## �📌 传输证明（Evidence）

### 状态：❌ 未启用

`network/evidence.go` 和 `network/evidence_bls.go` 定义了传输证明接口和 BLS 签名实现，用于追踪消息转发路径并分配激励份额。

**设计目标**：
- 记录消息在 P2P 网络中的传播轨迹
- 通过 BLS 聚合签名验证中继节点
- 为网络层激励提供技术支撑

**当前状态**：
- ✅ 接口设计完整（`TransmissionEvidence`, `EvidenceManagement`）
- ✅ BLS 实现健全（签名聚合、份额分配）
- ❌ **未集成到消息传递流程**
- ❌ 测试代码已被注释

**相关文件**：
- `network/evidence.go` - 接口定义
- `network/evidence_bls.go` - BLS 实现
- `network/network.go:40` - `Message.IsAddTransferEvidence`（占位符）

如需启用此功能，需要修改 `NetworkAdaptor` 和 `Network` 的消息处理逻辑，在广播/单播时附加和验证 Evidence。

---

## 🔧 技术细节

### 依赖的第三方库

- **libp2p** - 点对点网络框架
- **go-libp2p-pubsub** - 主题订阅（GossipSub）
- **go-libp2p-kad-dht** - Kademlia DHT（节点发现）
- **chia-bls-go** - BLS 签名（Evidence 模块）

### 关键配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 监听端口 | 6060 | P2P 监听端口 |
| DHT 路径 | `{datadir}/network/dht/` | DHT 数据存储 |
| 密钥路径 | `{datadir}/network/key/` | libp2p 节点密钥 |
| 连接超时 | 2秒 | Dial 超时时间 |
| 引导节点 | `kaddht.DefaultBootstrapPeers` | IPFS 公网节点 |

---

## ⚡ 性能特性

- **消息验证**：GossipSub 内置评分系统（`score.go`）
- **节点发现**：DHT 自动发现，支持自定义引导节点
- **并发处理**：Topic 订阅使用独立 goroutine 监听
- **容错机制**：连接失败时自动降级（允许单节点运行）

---

## 📚 参考

- **POT 共识系统**：`consensus/pot/`
- **Upgrade 系统集成**：`consensus/upgrade/`
- **配置示例**：`config/config.yaml`