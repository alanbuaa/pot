# 外部客户端如何构造并提交交易给 PoT 共识

## 完整工作流程

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   Client     │ gRPC │   P2P Layer  │      │ Consensus    │      │   Executor   │
│ (external)   │─────▶│  (Proxy)     │─────▶│   Engine     │─────▶│  (Backend)   │
└──────────────┘      └──────────────┘      └──────────────┘      └──────────────┘
   构造交易              转发 Packet          处理 Request           执行交易
```

---

## 步骤 1: 客户端构造交易

### 位置
[cmd/client/main.go](cmd/client/main.go#L250-L255)

### 代码示例

```go
// 1. 构造交易 payload（业务数据）
innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))

// 2. 创建 Transaction 对象
tx := &pb.Transaction{
    Type:    pb.TransactionType_NORMAL,  // 交易类型
    Payload: []byte(innerTx),            // 交易数据
}
```

### Transaction 消息定义

根据 protobuf 定义（`pkg/proto/common.proto`）:

```protobuf
message Transaction {
    TransactionType type = 1;    // NORMAL, UPGRADE, etc.
    bytes payload = 2;            // 业务数据
}

enum TransactionType {
    NORMAL = 0;      // 普通交易
    UPGRADE = 1;     // 升级交易
}
```

---

## 步骤 2: 客户端发送交易

### 位置
[cmd/client/main.go](cmd/client/main.go#L183-L217)

### 完整流程代码

```go
func (client *Client) sendTx(tx *pb.Transaction) {
    // 1. 序列化 Transaction
    btx, err := proto.Marshal(tx)
    utils.PanicOnError(err)
    
    // 2. 包装为 Request（支持分片）
    request := &pb.Request{
        Tx:       btx,
        Sharding: []byte("default"),  // 分片标识
    }
    
    // 3. 序列化 Request
    rawRequest, err := proto.Marshal(request)
    utils.PanicOnError(err)
    
    // 4. 包装为 Packet（网络层协议）
    packet := &pb.Packet{
        Msg:         rawRequest,
        ConsensusID: -1,                          // 客户端交易标记为 -1
        Epoch:       -1,
        Type:        pb.PacketType_CLIENTPACKET,  // 关键：标记为客户端包
    }
    
    // 5. 通过 gRPC 发送到共识节点
    client.pendingTx.Add(1)
    cmd := command(tx.Payload)
    client.setSendTime(cmd, time.Now())
    
    _, err = client.p2pClient.Send(context.Background(), packet)
    utils.LogOnError(err, "send packet failed", client.log)
}
```

### gRPC 连接初始化

```go
func NewClient(log *logrus.Entry) *Client {
    cfg, err := config.NewConfig("config/configpot.yaml", 0)
    utils.PanicOnError(err)
    
    // 连接到共识节点的 RPC 地址
    conn, err := grpc.Dial(cfg.Nodes[0].RpcAddress, 
                           grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        panic(err)
    }
    
    p2pClient := pb.NewP2PClient(conn)  // 创建 P2P 客户端
    // ...
}
```

---

## 步骤 3: P2P 层接收并转发

### 位置
[p2p/p2p.go](p2p/p2p.go#L117-L128)

### 处理流程

```go
// BaseP2p 实现 P2PServer 接口
func (bp *BaseP2p) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
    // 1. 序列化 Packet
    bytePacket, err := proto.Marshal(in)
    if err != nil {
        bp.log.Warn("marshal packet failed")
        return nil, err
    }
    
    // 2. 转发到共识层
    if bp.output != nil {
        bp.output <- bytePacket  // 发送到共识引擎的消息通道
    } else {
        bp.log.Warn("[BaseP2p] output nil")
    }
    
    return &pb.Empty{}, nil
}
```

**关键点**: P2P 层通过 `output` channel 将字节流转发给共识引擎。

---

## 步骤 4: 共识引擎处理消息

### 4.1 消息接收循环

#### 位置
[consensus/pot/handle_msg.go](consensus/pot/handle_msg.go#L18-L35)

```go
func (e *PoTEngine) onReceiveMsg() {
    for {
        select {
        case msgByte, ok := <-e.MsgByteEntrance:
            if !ok {
                return
            }
            // 1. 解码 Packet
            packet, err := DecodePacket(msgByte)
            if err != nil {
                e.log.WithError(err).Warn("decode packet failed")
                continue
            }
            // 2. 处理 Packet
            e.handlePacket(packet)
        }
    }
}
```

### 4.2 Packet 分发

#### 位置
[consensus/pot/handle_msg.go](consensus/pot/handle_msg.go#L40-L90)

```go
func (e *PoTEngine) handlePacket(packet *pb.Packet) {
    if packet.Type == pb.PacketType_P2PPACKET {
        // 共识节点间的消息
        potMsg := new(pb.PoTMessage)
        if err := proto.Unmarshal(packet.Msg, potMsg); err != nil {
            e.log.WithError(err).Warn("decode pot message failed")
            return
        }
        err := e.handlePoTMsg(potMsg)
        // ...
        
    } else if packet.Type == pb.PacketType_CLIENTPACKET {
        // ⭐ 客户端交易消息
        request := new(pb.Request)
        if err := proto.Unmarshal(packet.Msg, request); err != nil {
            e.log.WithError(err).Warn("unmarshal msg failed")
            return
        }
        
        if request == nil {
            e.log.Warn("only request msg allowed in client packet")
            return
        }
        e.handleRequest(request)  // 处理客户端请求
    }
}
```

### 4.3 处理客户端请求

#### 位置
[consensus/pot/handle_msg.go](consensus/pot/handle_msg.go#L92-L113)

```go
func (e *PoTEngine) handleRequest(request *pb.Request) {
    // 1. 提取原始交易
    rtx := types.RawTransaction(request.Tx)
    
    // 2. 验证交易（调用 Executor）
    if !e.exec.VerifyTx(rtx) {
        e.log.Warn("executedBlock verify failed")
        return
    }
    
    // 3. 解码交易
    tx, err := rtx.ToTx()
    if err != nil {
        e.log.WithError(err).Warn("decode into transaction failed")
        return
    }
    
    // 4. 根据交易类型分发
    switch tx.Type {
    case pb.TransactionType_NORMAL:
        // 转发给上层共识（如果存在）
        if e.UpperConsensus != nil {
            e.UpperConsensus.GetRequestEntrance() <- request
        }
    default:
        e.log.Warn("transaction type unknown", tx.Type.String())
    }
}
```

**关键点**: 
- PoT 作为底层共识，接收交易后转发给上层共识（如 HotStuff）
- 上层共识负责打包交易到区块并达成共识

---

## 步骤 5: 交易进入 Mempool

### BCI 交易的特殊处理

对于 BCI 系统的交易（CreateLock、Devastate 等），工作流程有所不同：

#### 位置
[consensus/pot/http.go](consensus/pot/http.go#L242-L264)

```go
func handlerCreateLockTransaction(c *gin.Context, w *Worker) {
    // 1. HTTP 接收交易
    var transaction HttpTransaction
    if err := c.ShouldBindBodyWithJSON(&transaction); err != nil {
        // 错误处理
        return
    }
    
    // 2. 转换为内部格式
    tx, err := HttpTx2Tx(&transaction)
    if err != nil {
        // 错误处理
        return
    }
    
    // 3. 广播交易到所有节点
    err = w.broadcastClientTransaction(tx, pb.TxType_CreateLockTransaction)
    if err != nil {
        // 错误处理
        return
    }
    
    // 4. 添加到本地 Mempool
    w.mempool.AddRawTx(tx)
    
    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "msg":  "success",
    })
}
```

### 广播交易到所有节点

#### 位置
[consensus/pot/dci.go](consensus/pot/dci.go#L294-L313)

```go
func (w *Worker) broadcastClientTransaction(rawtx *types.RawTx, types pb.TxType) error {
    // 1. 包装为 ClientTransaction
    pbtx := pb.ClientTransaction{
        Tx:     rawtx.ToProto(),
        TxType: types,
    }
    b, err := proto.Marshal(&pbtx)
    if err != nil {
        return err
    }
    
    // 2. 包装为 PoTMessage
    message := &pb.PoTMessage{
        MsgType: pb.MessageType_Client_Transaction,
        MsgByte: b,
    }
    messagebyte, err := proto.Marshal(message)
    if err != nil {
        return err
    }
    
    // 3. 通过 P2P 广播
    err = w.Engine.Broadcast(messagebyte)
    if err != nil {
        return err
    }
    return nil
}
```

---

## 步骤 6: Executor 执行交易

### 交易验证

#### 位置
[cmd/executor/main.go](cmd/executor/main.go#L70-L79)

```go
func (p *PoTExecutor) VerifyTxs(ctx context.Context, request *pb.VerifyTxRequest) (*pb.VerifyTxResponse, error) {
    // 验证交易列表
    flag := make([]bool, len(request.GetTxs()))
    for i := 0; i < len(flag); i++ {
        flag[i] = true  // 简化实现：全部通过
    }
    reponse := &pb.VerifyTxResponse{
        Txs:  request.Txs,
        Flag: flag,
    }
    return reponse, nil
}
```

### 交易执行

#### 位置
[cmd/executor/main.go](cmd/executor/main.go#L81-L89)

```go
func (p *PoTExecutor) ExecuteTxs(ctx context.Context, request *pb.ExecuteTxRequest) (*pb.ExecuteTxResponse, error) {
    return &pb.ExecuteTxResponse{
        Tx:   request.Tx,
        Flag: true,      // 执行成功
        TxID: []byte{},  // 交易哈希
    }, nil
}
```

---

## 完整数据流图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           客户端 (External Client)                        │
└─────────────────────────────────────────────────────────────────────────┘
                                     │
                                     │ 1. 构造 pb.Transaction
                                     │    Type: NORMAL
                                     │    Payload: business data
                                     ▼
                            ┌────────────────┐
                            │  pb.Request    │
                            │  ┌──────────┐  │
                            │  │   Tx     │  │ 序列化的 Transaction
                            │  └──────────┘  │
                            │  ┌──────────┐  │
                            │  │ Sharding │  │ 分片信息
                            │  └──────────┘  │
                            └────────────────┘
                                     │
                                     │ 2. 包装为 pb.Packet
                                     │    Type: CLIENTPACKET
                                     │    ConsensusID: -1
                                     ▼
                            ┌────────────────┐
                            │  gRPC Send()   │ → RPC 地址 (localhost:7070)
                            └────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────┐
│                    P2P 层 (BaseP2p / P2PAdaptor)                        │
│                                    │                                     │
│                    3. Send(ctx, *pb.Packet)                             │
│                                    │                                     │
│                    4. output <- bytePacket  ────────────────────┐       │
│                                                                  │       │
└──────────────────────────────────────────────────────────────────┼───────┘
                                                                   │
┌──────────────────────────────────────────────────────────────────┼───────┐
│                        共识层 (PoTEngine)                         │       │
│                                                                   │       │
│  5. MsgByteEntrance <- msgByte ◀──────────────────────────────────┘       │
│            │                                                              │
│            ▼                                                              │
│  6. onReceiveMsg() 循环接收                                               │
│            │                                                              │
│            ▼                                                              │
│  7. DecodePacket(msgByte)                                                │
│            │                                                              │
│            ▼                                                              │
│  8. handlePacket(packet)                                                 │
│            │                                                              │
│            ├─── PacketType_P2PPACKET ──▶ handlePoTMsg()                  │
│            │                                                              │
│            └─── PacketType_CLIENTPACKET ──▶ handleRequest()              │
│                        │                                                 │
│                        ▼                                                 │
│            9. Unmarshal pb.Request                                       │
│                        │                                                 │
│                        ▼                                                 │
│            10. VerifyTx(rtx) ─────────────┐                              │
│                        │                   │                             │
│                        │                   │                             │
│                        ▼                   │                             │
│            11. rtx.ToTx()                  │                             │
│                        │                   │                             │
│                        ▼                   │                             │
│            12. switch tx.Type:             │                             │
│                  case NORMAL:              │                             │
│                    UpperConsensus          │                             │
│                      .GetRequestEntrance() │                             │
│                        <- request          │                             │
│                                            │                             │
└────────────────────────────────────────────┼─────────────────────────────┘
                                             │ gRPC 调用
                                             │
┌────────────────────────────────────────────┼─────────────────────────────┐
│                      Executor (gRPC Server)│                             │
│                                            │                             │
│            13. VerifyTxs() ◀───────────────┘                             │
│                        │                                                 │
│                        │ 返回验证结果 (flag: true/false)                   │
│                        ▼                                                 │
│            ┌──────────────────┐                                          │
│            │   交易验证逻辑    │                                          │
│            └──────────────────┘                                          │
│                        │                                                 │
│                        ▼                                                 │
│            14. ExecuteTxs()                                              │
│                        │                                                 │
│                        │ 返回执行结果 (TxID, Flag)                        │
│                        ▼                                                 │
│            ┌──────────────────┐                                          │
│            │  交易执行逻辑     │                                          │
│            │  状态更新         │                                          │
│            └──────────────────┘                                          │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

---

## 关键接口定义

### 1. P2P Service (protobuf)

```protobuf
service P2P {
    rpc Send(Packet) returns (Empty);
}

message Packet {
    bytes msg = 1;
    int64 consensusID = 2;
    int64 epoch = 3;
    PacketType type = 4;
}

enum PacketType {
    P2PPACKET = 0;      // 共识节点间消息
    CLIENTPACKET = 1;   // 客户端交易消息
}
```

### 2. Request Message

```protobuf
message Request {
    bytes tx = 1;        // 序列化的 Transaction
    bytes sharding = 2;  // 分片标识
}
```

### 3. Transaction Message

```protobuf
message Transaction {
    TransactionType type = 1;
    bytes payload = 2;
}

enum TransactionType {
    NORMAL = 0;
    UPGRADE = 1;
}
```

---

## 两种客户端提交方式对比

### 方式 1: gRPC 直连（推荐用于测试）

**位置**: [cmd/client/main.go](cmd/client/main.go)

**优点**:
- 简单直接，易于测试
- 支持自动重连
- 延迟追踪

**使用场景**:
- 性能测试
- 负载生成
- 开发调试

**示例**:
```bash
# 启动客户端（自动发送交易）
make run_client

# 或手动运行
go run cmd/client/main.go
```

### 方式 2: HTTP API（推荐用于生产）

**位置**: [consensus/pot/http.go](consensus/pot/http.go)

**优点**:
- RESTful 风格，易于集成
- 支持多种交易类型（CreateLock, Devastate, Transfer 等）
- 适合 Web 应用

**使用场景**:
- 生产环境
- Web 前端集成
- 跨语言调用

**示例**:
```bash
# 创建锁定交易
curl -X POST http://localhost:8088/api/transaction/createlock \
  -H "Content-Type: application/json" \
  -d '{
    "txinput": [...],
    "txoutput": [...]
  }'
```

---

## 实践建议

### 1. 开发测试流程

```bash
# 步骤 1: 生成密钥
make run_genkey

# 步骤 2: 启动 Executor
make run_executor

# 步骤 3: 启动共识节点
make run_server

# 步骤 4: 发送测试交易
make run_client
```

### 2. 自定义客户端开发

参考 [cmd/client/main.go](cmd/client/main.go#L183-L217) 实现：

```go
// 1. 连接到共识节点
conn, err := grpc.Dial("localhost:7070", 
    grpc.WithTransportCredentials(insecure.NewCredentials()))
client := pb.NewP2PClient(conn)

// 2. 构造交易
tx := &pb.Transaction{
    Type:    pb.TransactionType_NORMAL,
    Payload: []byte("your_business_data"),
}

// 3. 包装为 Request
btx, _ := proto.Marshal(tx)
request := &pb.Request{
    Tx:       btx,
    Sharding: []byte("default"),
}

// 4. 包装为 Packet
rawRequest, _ := proto.Marshal(request)
packet := &pb.Packet{
    Msg:         rawRequest,
    ConsensusID: -1,
    Epoch:       -1,
    Type:        pb.PacketType_CLIENTPACKET,
}

// 5. 发送
client.Send(context.Background(), packet)
```

### 3. 配置文件

确保 [config/config.yaml](config/config.yaml) 正确配置：

```yaml
nodes:
  - id: 0
    address: "127.0.0.1:6060"     # P2P 地址
    rpc_address: "127.0.0.1:7070"  # RPC 地址（客户端连接这里）
    address_ip: "127.0.0.1"
    pubkey: "..."
```

---

## 常见问题

### Q1: 交易发送后没有响应？

**检查清单**:
1. Executor 是否启动？(`make run_executor`)
2. 共识节点是否启动？(`make run_server`)
3. RPC 地址是否正确？(默认 `localhost:7070`)
4. 防火墙是否阻止连接？

### Q2: PoT 引擎为什么转发给 UpperConsensus？

**原因**: PoT 是底层共识（负责 VDF 计算、委员会选举），上层共识（如 HotStuff）负责交易打包和区块排序。

**架构**:
```
HotStuff (上层) ← 负责交易打包、区块共识
    ↑
PoT (底层)     ← 负责委员会选举、VDF 计算
```

### Q3: 如何追踪交易状态？

客户端代码已实现延迟追踪：

```go
// 记录发送时间
client.setSendTime(cmd, time.Now())

// 接收回复时计算延迟
duration := time.Since(client.getSendTime(cmd))
client.log.Infof("the message %s latency is %s", string(replyMsg.Tx), duration)
```

---

## 相关文件索引

| 组件 | 文件路径 | 说明 |
|------|----------|------|
| **客户端** | [cmd/client/main.go](cmd/client/main.go) | gRPC 客户端实现 |
| **P2P 层** | [p2p/p2p.go](p2p/p2p.go) | BaseP2p 实现 |
| **共识引擎** | [consensus/pot/engine.go](consensus/pot/engine.go) | PoTEngine 主逻辑 |
| **消息处理** | [consensus/pot/handle_msg.go](consensus/pot/handle_msg.go) | 消息分发与处理 |
| **Mempool** | [consensus/pot/mempool.go](consensus/pot/mempool.go) | 交易池管理 |
| **Executor** | [cmd/executor/main.go](cmd/executor/main.go) | 交易执行服务 |
| **HTTP API** | [consensus/pot/http.go](consensus/pot/http.go) | RESTful API |
| **BCI 逻辑** | [consensus/pot/dci.go](consensus/pot/dci.go) | BCI 交易处理 |
| **交易类型** | [types/transaction.go](types/transaction.go) | Transaction 定义 |
| **Protobuf** | `pkg/proto/common.proto` | 消息定义 |

---

## 上层共识 (Whirly) 与底层共识 (POT) 交互架构

### 双层共识架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                         应用层 (Client)                              │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    上层共识 - Whirly (NodeController)                │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │  职责:                                                      │     │
│  │  • 交易打包成区块                                           │     │
│  │  • BFT 共识协议 (Simple/CR Whirly)                          │     │
│  │  • 快速区块确认 (毫秒级)                                    │     │
│  │  • 管理多个 Sharding                                        │     │
│  └────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
                          ▲                    │
                          │ PoTSignal         │ 区块数据
                          │ (委员会更新)       │
                          │                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      底层共识 - POT (PoTEngine)                      │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │  职责:                                                      │     │
│  │  • VDF 计算 (可验证延迟函数)                                │     │
│  │  • 委员会选举                                               │     │
│  │  • 长期区块链维护                                           │     │
│  │  • 安全性保障                                               │     │
│  └────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
                          Executor (交易执行)
```

---

## 完整交易流程：从客户端到区块提交

### 阶段 1: 交易提交 (Client → POT)

```
Client
  │
  │ 1. 构造 Transaction + Request + Packet
  ▼
P2P Layer (gRPC)
  │
  │ 2. 转发到 PoTEngine.MsgByteEntrance
  ▼
PoTEngine.handlePacket()
  │
  │ 3. Type == CLIENTPACKET?
  ▼
PoTEngine.handleRequest()
  │
  │ 4. VerifyTx() - 调用 Executor 验证
  │ 5. 解析 Transaction.Type
  ▼
  switch tx.Type:
    case NORMAL:
      UpperConsensus.GetRequestEntrance() <- request  ← 转发给 Whirly
```

**关键代码**: [consensus/pot/handle_msg.go#L92-L107](consensus/pot/handle_msg.go#L92-L107)

```go
func (e *PoTEngine) handleRequest(request *pb.Request) {
    rtx := types.RawTransaction(request.Tx)
    
    // 验证交易
    if !e.exec.VerifyTx(rtx) {
        return
    }
    
    // 转发给上层共识 (Whirly)
    if e.UpperConsensus != nil {
        e.UpperConsensus.GetRequestEntrance() <- request
    }
}
```

---

### 阶段 2: 交易处理 (POT → Whirly → 区块打包)

```
NodeController (Whirly 上层)
  │
  │ 6. RequestByteEntrance <- request
  ▼
NodeController.handleRequest()
  │
  │ 7. 检查 Sharding (req.Sharding == "0x1")
  ▼
Sharding.handleRequest()
  │
  │ 8. 分发到所有节点
  ▼
for each Node in Sharding:
  Node.GetRequestEntrance() <- req
  │
  ▼
SimpleWhirly.OnReceiveRequest()
  │
  │ 9. 添加到 Mempool
  │ 10. Leader 定时打包区块
  ▼
SimpleWhirly.Propose()
  │
  │ 11. 创建 ProposalMsg
  │ 12. Broadcast(ProposalMsg)
  ▼
所有副本节点
  │
  │ 13. OnReceiveProposal()
  │ 14. 投票 (VoteMsg)
  ▼
Leader 收集投票
  │
  │ 15. 达到 2f+1 票
  │ 16. CommitBlock()
  ▼
Executor.CommitBlock()  ← 执行交易并返回回复
```

**Whirly 请求入口**: [consensus/whirly/nodeController/nodeController.go#L102-L104](consensus/whirly/nodeController/nodeController.go#L102-L104)

```go
func (nc *NodeController) GetRequestEntrance() chan<- *pb.Request {
    return nc.RequestByteEntrance
}
```

**Sharding 分发**: [consensus/whirly/nodeController/sharding.go#L213-L222](consensus/whirly/nodeController/sharding.go#L213-L222)

```go
func (s *Sharding) handleRequest(req *pb.Request) {
    s.nodesLock.Lock()
    
    // 广播到该 Sharding 的所有共识节点
    for _, node := range s.Nodes {
        go func(n model.Consensus) {
            n.GetRequestEntrance() <- req
        }(node)
    }
    
    s.nodesLock.Unlock()
}
```

---

### 阶段 3: POT 委员会更新 (POT → Whirly)

POT 每产生一个区块，会更新 Whirly 的委员会配置：

```
PoTEngine (每个区块)
  │
  ▼
Worker.CommitteeUpdate(height)
  │
  │ 1. 从最近 N 个区块提取委员会公钥
  │ 2. 构造 PoTSignal
  ▼
PoTSignal 数据结构:
  {
    Epoch: int64               // 当前高度
    SelfPublicAddress: []string // 本节点在委员会中的地址
    Shardings: []PoTSharding {
      Name: "0x1"              // Sharding 名称
      Committee: []string      // 委员会成员列表
      LeaderPublicAddress: string
      SubConsensus: {          // 子共识配置
        Type: "whirly"
        ConsensusID: 1201
        Whirly: { Type: "simple", BatchSize: 2, Timeout: 2 }
      }
    }
  }
  │
  │ 3. 序列化并发送
  ▼
potSignalChan <- marshalledPoTSignal
  │
  ▼
NodeController.PoTByteEntrance <- signal
  │
  │ 4. 解析 PoTSignal
  ▼
NodeController.handlePotSignal()
  │
  │ 5. 更新 Epoch
  ▼
NodeController.ShardManage()
  │
  │ 6. 创建或更新 Sharding
  ▼
for each Sharding in signal:
  if not exist:
    NewSharding()           // 创建新分片
    创建 DaemonNode
  else:
    更新 Committee
    更新 LeaderPublicAddress
    if ConsensusID 变化:
      createConsensusNode() // 创建新共识节点
```

**POT 更新委员会**: [consensus/pot/run_commitee.go#L118-L199](consensus/pot/run_commitee.go#L118-L199)

```go
func (w *Worker) CommitteeUpdate(height uint64) {
    // 从最近的区块提取委员会成员
    committee := make([]string, Commiteelen)
    selfaddress := make([]string, 0)
    
    for i := uint64(0); i < Commiteelen; i++ {
        block, _ := w.chainReader.GetByHeight(height - CommiteeDelay - i)
        header := block.GetHeader()
        committee[i] = hexutil.Encode(header.CommiteePubkey)
        
        // 检查自己是否在委员会中
        if flag, _ := w.TryFindCommiteeKey(crypto.Convert(header.Hash())); flag {
            selfaddress = append(selfaddress, hexutil.Encode(header.CommiteePubkey))
        }
    }
    
    // 构造 Sharding 配置
    sharding1 := nodeController.PoTSharding{
        Name:                hexutil.EncodeUint64(1), // "0x1"
        LeaderPublicAddress: committee[0],
        Committee:           committee,
        SubConsensus:        consensus,
    }
    
    // 发送 PoTSignal
    potsignal := &nodeController.PoTSignal{
        Epoch:             int64(height),
        SelfPublicAddress: selfaddress,
        Shardings:         []nodeController.PoTSharding{sharding1},
    }
    
    b, _ := json.Marshal(potsignal)
    w.potSignalChan <- b  // 发送给 NodeController
}
```

**Whirly 处理委员会更新**: [consensus/whirly/nodeController/nodeController.go#L258-L281](consensus/whirly/nodeController/nodeController.go#L258-L281)

```go
func (nc *NodeController) ShardManage(potSignal *PoTSignal) {
    for _, s := range potSignal.Shardings {
        nc.shardingsLock.Lock()
        localSharding, ok := nc.Shardings[s.Name]
        
        if !ok {
            // 创建新 Sharding
            ns := NewSharding(s.Name, nc, s)
            nc.Shardings[s.Name] = ns
        } else {
            // 更新现有 Sharding
            localSharding.LeaderPublicAddress = s.LeaderPublicAddress
            localSharding.Committee = s.Committee
            
            if localSharding.SubConsensus.ConsensusID != s.SubConsensus.ConsensusID {
                // 共识切换：创建新共识节点
                localSharding.SubConsensus = &s.SubConsensus
                address := EncodeAddress(localSharding.Name+nc.PeerId, 
                                        s.SubConsensus.ConsensusID, 
                                        DaemonNodePublicAddress)
                localSharding.createConsensusNode(address)
            }
        }
        nc.shardingsLock.Unlock()
    }
}
```

---

### 阶段 4: 区块执行与回复 (Executor → Client)

```
SimpleWhirly.CommitBlock()
  │
  │ 达成共识后调用
  ▼
Executor.CommitBlock(block, proof, consensusID)
  │
  │ (LocalExecutor 情况)
  ▼
for each tx in block:
  │
  │ 1. 解析交易
  │ 2. 计算结果 (arg1 + arg2)
  │ 3. 构造 Reply
  ▼
  pb.Msg{
    Payload: &pb.Msg_Reply{
      Reply: &pb.Reply{
        Tx: rawTx,
        Receipt: result  // "counter--sum"
      }
    }
  }
  │
  │ 4. 连接到 localhost:9999
  ▼
  gRPC Dial(localhost:9999)
    │
    ▼
  Client.Send(Packet)  ← 调用客户端的 gRPC 接口
    │
    ▼
  Client.replyChan <- msg
    │
    ▼
  Client.receiveReply()  ← 后台 goroutine 处理
    │
    │ 5. 解析 Reply
    │ 6. 更新交易状态
    │ 7. 计算延迟
    ▼
    pending → success
```

**LocalExecutor 发送回复**: [executor/localExeutor.go#L30-L59](executor/localExeutor.go#L30-L59)

```go
func (e *LocalExecutor) CommitBlock(block types.ConsensusBlock, proof []byte, cid int64) {
    for _, rtx := range block.GetTxs() {
        tx, _ := types.RawTransaction(rtx).ToTx()
        if tx.Type != pb.TransactionType_NORMAL {
            continue
        }
        
        // 执行交易 (简单加法示例)
        split := strings.Split(string(tx.Payload), ",")
        arg1, _ := strconv.Atoi(split[0])
        arg2, _ := strconv.Atoi(split[1])
        e.counter++
        
        // 构造回复
        rawReceipt := []byte(strconv.Itoa(e.counter) + "--" + strconv.Itoa(arg1+arg2))
        msg := &pb.Msg{Payload: &pb.Msg_Reply{
            Reply: &pb.Reply{Tx: rtx, Receipt: rawReceipt}
        }}
        
        // 发送到客户端
        conn, _ := grpc.Dial("localhost:9999", ...)
        client := pb.NewP2PClient(conn)
        packet := &pb.Packet{
            Msg:         msgByte,
            ConsensusID: -1,
            Type:        pb.PacketType_CLIENTPACKET,
        }
        client.Send(context.Background(), packet)
        conn.Close()
    }
}
```

---

## 关键数据流图：POT ↔ Whirly 交互

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          客户端交易流 (从下往上)                          │
└─────────────────────────────────────────────────────────────────────────┘

  Client                 POT Engine           NodeController        Whirly Node
    │                        │                      │                    │
    │ 1. Request             │                      │                    │
    ├───────────────────────▶│                      │                    │
    │                        │ 2. VerifyTx()        │                    │
    │                        │                      │                    │
    │                        │ 3. Forward Request   │                    │
    │                        ├─────────────────────▶│                    │
    │                        │                      │ 4. Route by        │
    │                        │                      │    Sharding        │
    │                        │                      ├───────────────────▶│
    │                        │                      │                    │
    │                        │                      │ 5. AddToMempool    │
    │                        │                      │                    │
    │                        │                      │ 6. Propose (Leader)│
    │                        │                      │                    │
    │                        │                      │ 7. Vote (Replicas) │
    │                        │                      │                    │
    │                        │                      │ 8. CommitBlock     │
    │                        │                      │    (2f+1 votes)    │
    │                        │                      │                    │
    │ 9. Reply (via Executor)│                      │                    │
    │◀───────────────────────┴──────────────────────┴────────────────────┘


┌─────────────────────────────────────────────────────────────────────────┐
│                       委员会更新流 (从上往下)                             │
└─────────────────────────────────────────────────────────────────────────┘

  POT Worker          PoTSignal Chan       NodeController      Sharding
    │                        │                      │              │
    │ 1. 产生新区块          │                      │              │
    │    (VDF 挖矿)          │                      │              │
    │                        │                      │              │
    │ 2. 提取委员会          │                      │              │
    │    (最近 N 个区块)     │                      │              │
    │                        │                      │              │
    │ 3. 构造 PoTSignal      │                      │              │
    ├───────────────────────▶│                      │              │
    │                        │ 4. Forward           │              │
    │                        ├─────────────────────▶│              │
    │                        │                      │              │
    │                        │ 5. ShardManage()     │              │
    │                        │                      │              │
    │                        │ 6. 创建/更新 Sharding │              │
    │                        │                      ├─────────────▶│
    │                        │                      │              │
    │                        │ 7. createConsensusNode()            │
    │                        │    (如果委员会变化)  │              │
    │                        │                      │              │
    │                        │ 8. 新共识节点就绪    │              │
```

---

## POT 与 Whirly 的职责划分

| 维度 | POT (底层) | Whirly (上层) |
|------|-----------|--------------|
| **主要职责** | VDF 计算、委员会选举 | 交易打包、BFT 共识 |
| **时间尺度** | 长期 (秒级) | 短期 (毫秒级) |
| **区块生成** | 通过 VDF 挖矿 | 通过 BFT 投票 |
| **安全保障** | 不可预测性、防操纵 | 拜占庭容错 |
| **处理对象** | 委员会成员、Epoch | 用户交易、快速确认 |
| **持久化** | 主链存储 | 临时区块 (依赖 POT) |
| **与客户端** | 间接 (转发) | 直接 (接收请求) |

---

## 初始化流程

```go
// 1. POT 启动时创建 Whirly (NodeController)
func (e *PoTEngine) StartCommitee() *nodeController.NodeController {
    whirlyconfig := &config.ConsensusConfig{
        Type:        "whirly",
        ConsensusID: 1009,
        Whirly: &config.WhirlyConfig{
            Type:      "simple",
            BatchSize: 10,
            Timeout:   2000,
        },
        // 复用 POT 的配置
        Nodes: e.config.Nodes,
        Keys:  e.config.Keys,
        Topic: e.config.Topic,
        F:     e.config.F,
    }
    
    // 创建 NodeController (Whirly 管理器)
    s := nodeController.NewNodeController(e.id, 1009, whirlyconfig, e.exec, e.Adaptor, e.log)
    e.UpperConsensus = s  // 设置上层共识
    
    return s
}

// 2. 连接 PoTSignal 通道
func (w *Worker) SetWhirly(impl *nodeController.NodeController) {
    w.potSignalChan = impl.GetPoTByteEntrance()  // 获取 Whirly 的信号入口
}
```

---

## 配置示例

### POT 配置 (config/config.yaml)

```yaml
consensus:
  type: "pot"
  consensus_id: 10001
  pot:
    snum: 2
    vdf0Iteration: 100000
    vdf1Iteration: 80000
    batchsize: 100
    timeout: 5
    commiteeSize: 4       # 委员会大小
    confirmDelay: 6       # 确认延迟
    executorAddress: 127.0.0.1:9877

# 上层 Whirly 配置由 POT 动态创建，参数硬编码在:
# - consensus/pot/engine.go (StartCommitee)
# - consensus/pot/run_commitee.go (CommitteeUpdate)
```

### Whirly 动态配置

```go
// 在 POT 代码中硬编码
whirlyconfig := &config.ConsensusConfig{
    Type:        "whirly",
    ConsensusID: 1009,  // 固定 ID
    Whirly: &config.WhirlyConfig{
        Type:      "simple",  // 使用 Simple Whirly
        BatchSize: 10,        // 每个区块 10 笔交易
        Timeout:   2000,      // 2 秒超时
    },
}
```

---

## 总结

外部客户端提交交易的核心步骤：

1. ✅ **构造 `pb.Transaction`** (Type + Payload)
2. ✅ **包装为 `pb.Request`** (支持分片)
3. ✅ **包装为 `pb.Packet`** (Type = CLIENTPACKET)
4. ✅ **通过 gRPC 发送** (`p2pClient.Send()`)
5. ✅ **P2P 层转发** (`output <- bytePacket`)
6. ✅ **共识引擎处理** (`handleRequest()`)
7. ✅ **Executor 验证执行** (`VerifyTxs` → `ExecuteTxs`)
8. ✅ **返回结果** (通过 gRPC 回调)

### POT 与 Whirly 交互要点

- **POT 职责**: 委员会选举、VDF 计算、长期安全
- **Whirly 职责**: 交易打包、BFT 共识、快速确认
- **交互方向**:
  - **上行** (Client → POT → Whirly): 交易流
  - **下行** (POT → Whirly): 委员会更新 (PoTSignal)
  - **回流** (Whirly → Executor → Client): 交易回复
- **关键通道**:
  - `UpperConsensus.GetRequestEntrance()`: POT → Whirly 交易转发
  - `potSignalChan`: POT → Whirly 委员会更新
  - `RequestByteEntrance`: Whirly 接收交易入口

**关键点**: 
- 使用 `pb.PacketType_CLIENTPACKET` 标识客户端交易
- 交易经过验证后转发给上层共识打包
- 支持两种提交方式：gRPC 直连、HTTP API
- POT 和 Whirly 通过通道 (channel) 进行解耦通信
