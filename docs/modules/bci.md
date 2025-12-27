# BCI (Blockchain Incentive) 模块

## 概述

BCI（Blockchain Incentive）是 PoT 共识机制中的激励分配模块，用于管理区块奖励的产生、验证、记录和分配。该模块确保矿工、委员会成员及其他参与方能够获得相应的奖励。

## 模块依赖关系

BCI 模块作为 PoT 共识的核心组件之一，依赖以下 PoT 系统功能：

### 1. 内存池（Mempool）

**依赖位置**: [consensus/pot/mempool.go](../../consensus/pot/mempool.go)

**作用**：
- 存储和管理待处理的 BciReward
- 提供 BciReward 的增删查改操作
- 维护 BciReward 的提议状态（proposed 标志）
- 管理 UTXO 和交易缓存

**关键方法**：
```go
func (c *Mempool) AddBciReward(rewards ...*BciReward)
func (c *Mempool) GetAllBciRewards() []*BciReward
func (c *Mempool) MarkBciRewardProposed(coinbaseProofs []types.CoinbaseProof)
func (c *Mempool) HasBciRewardByCoinbaseProof(coinbaseproof *types.CoinbaseProof) bool
func (c *Mempool) RemoveBciRewardByTxHash(hash []byte, Address []byte)
```

### 2. 链状态读取器（ChainReader）

**依赖位置**: [consensus/pot/chainreader.go](../../consensus/pot/chainreader.go)

**作用**：
- 读取区块链当前状态（高度、区块数据）
- 访问 BoltDB 数据库进行 UTXO 查询
- 验证 BciReward 证明时查找对应区块
- 提供 UTXO 查找功能

**关键方法**：
```go
func (c *ChainReader) GetByHeight(height uint64) (*types.Block, error)
func (c *ChainReader) GetBoltDb() *bolt.DB
func (c *ChainReader) FindUTXO(address []byte) (map[string]*types.TxOutput, error)
func (c *ChainReader) GetCurrentHeight() uint64
```

**使用场景**：
- 验证 BciReward 时查找 proof 中指定高度的区块
- 查询用户余额（GetBalance 接口）
- 校验交易时读取 UTXO 状态
- 检查 coinbase proof 是否已被兑现

### 3. VDF（可验证延迟函数）

**依赖位置**: 集成在 Worker 结构中

**作用**：
- 提供随机性来源用于 BCI 奖励的抽样分配
- 确保奖励分配的不可预测性和公平性

**使用方式**：
```go
// 使用 VDF0 输出作为随机种子
rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
```

**应用场景**：
- 在 `GenerateCoinbaseTx` 中按权重随机抽样 BciReward
- 确保相同的 VDF 输出产生相同的抽样结果（可验证性）

### 4. 加密模块（crypto）

**依赖位置**: [crypto/](../../crypto/)

**作用**：
- 计算哈希（区块哈希、交易哈希、Merkle 根）
- 生成和验证密钥对
- 提供字节数组转换工具

**关键功能**：
```go
crypto.Hash(data []byte) [32]byte           // 哈希计算
crypto.Convert([]byte) [32]byte             // 字节数组转换
crypto.ComputeMerkleRoot(txs [][]byte)      // Merkle 树计算
```

**使用场景**：
- BciProof 中的 BlockHash 和 TxHash 验证
- Coinbase 交易的 Txid 计算
- VDF 输出哈希用于随机种子

### 5. 存储模块（Storage）

**依赖位置**: [internal/storage/pot](../../internal/storage/pot/)

**作用**：
- 持久化区块和交易数据
- 提供 BoltDB 数据库访问
- 管理 UTXO 存储桶

**关键常量**：
```go
storage.UTXOBucket  // UTXO 存储桶名称
```

**使用场景**：
- 交易校验时查询输入 UTXO 是否存在
- 检查 UTXO 是否已被使用
- 验证交易输入的解锁条件

### 6. P2P 网络引擎（Engine）

**依赖位置**: Worker.Engine (PoTEngine)

**作用**：
- 广播 BciReward 到 P2P 网络
- 广播客户端交易
- 接收其他节点发送的 BCI 相关消息

**关键方法**：
```go
func (e *Engine) Broadcast(data []byte) error
```

**使用场景**：
- `SendBci` 接口广播 BciReward
- 广播客户端交易（锁定、转账、销毁等）
- 同步 BCI 奖励信息到网络

### 7. gRPC 通信

**依赖位置**: [pkg/proto/](../../pkg/proto/)

**作用**：
- 定义 BCI 相关的 gRPC 接口
- 与执行器（Executor）通信获取奖励
- 提供客户端 API 服务

**关键接口**：
```go
service PotService {
    rpc SendBci(SendBciRequest) returns (SendBciResponse)
    rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse)
    rpc CreateLockTransaction(...) returns (...)
    rpc CreateLockTransferTransaction(...) returns (...)
    rpc CreateNonLockTransferTransaction(...) returns (...)
    rpc CreateDevastateTransaction(...) returns (...)
    rpc CreateBciToVsiTransaction(...) returns (...)
}
```

### 8. 执行器接口（Executor）

**作用**：
- BCI 奖励的实际产生方
- 执行交易并生成激励信息
- 通过 gRPC 向 Worker 返回 ExecutedBlock 和 BciReward

**交互流程**：
```
Worker → gRPC → Executor.GetIncentiveTx()
    ← ExecutedBlock + BciReward[]
```

**关键方法**：
```go
func (w *Worker) GetIncentiveTxFromExecutor(epoch uint64) ([]*types.ExecutedBlock, error)
```

### 9. 交易类型系统（types）

**依赖位置**: [types/](../../types/)

**作用**：
- 定义交易数据结构（RawTx, TxInput, TxOutput）
- 定义区块结构（Block, Header）
- 提供序列化/反序列化方法

**关键结构**：
```go
type RawTx struct {
    Txid           [32]byte
    TxInput        []TxInput
    TxOutput       []TxOutput
    CoinbaseProofs []CoinbaseProof
}

type CoinbaseProof struct {
    TxHash  []byte
    Address []byte
    Amount  int64
    Type    int32
}
```

### 10. 日志系统（logging）

**依赖位置**: Worker.log (*logrus.Entry)

**作用**：
- 记录 BCI 操作日志
- 调试和错误追踪

**使用示例**：
```go
w.log.Infof("[PoT]\tAdd %d Bci reward to %s", amount, address)
w.log.Errorf("get balance of %s", hexutil.Encode(addr))
```

## 依赖关系图

```
┌─────────────────────────────────────────────┐
│           BCI 模块核心功能                   │
├─────────────────────────────────────────────┤
│  - 奖励验证 (VerifyBciReward)               │
│  - Coinbase 生成 (GenerateCoinbaseTx)       │
│  - 交易校验 (Check*Transaction)             │
│  - 网络通信 (SendBci, broadcastXxx)         │
└─────────────────────────────────────────────┘
           ▲          ▲          ▲
           │          │          │
    ┌──────┴───┬──────┴────┬─────┴─────┐
    │          │           │           │
┌───▼────┐ ┌───▼────┐ ┌────▼─────┐ ┌───▼─────┐
│Mempool │ │ChainRdr│ │  VDF     │ │ Engine  │
│        │ │        │ │ (随机源) │ │ (P2P)   │
└───┬────┘ └───┬────┘ └──────────┘ └─────────┘
    │          │
    │     ┌────▼────┐
    │     │Storage  │
    │     │(BoltDB) │
    │     └─────────┘
    │
┌───▼──────────────────┐
│  crypto + types      │
│  (哈希、序列化)      │
└──────────────────────┘
```

## BCI 数据结构

### BciReward

BciReward 是奖励的核心数据结构，定义在 [consensus/pot/mempool.go](../../consensus/pot/mempool.go#L28-L36):

```go
type BciReward struct {
    Address []byte    // 奖励接收地址
    Amount  int64     // 奖励数量
    Proof   BciProof  // 奖励证明
    BciType int32     // 奖励类型（链ID）
    weight  float64   // 权重（用于概率分配）
    DoDraw  bool      // 是否参与抽样
}
```

### BciProof

BciProof 包含奖励的来源证明，定义在 [consensus/pot/mempool.go](../../consensus/pot/mempool.go#L38-L42):

```go
type BciProof struct {
    Height    uint64   // 区块高度
    BlockHash []byte   // 区块哈希
    TxHash    []byte   // 交易哈希
}
```

## BCI 的产生

### 1. 产生来源

BCI 奖励由**执行器（Executor）**产生。在 PoT 架构中，执行器负责执行交易并根据执行结果生成 BCI 奖励。

关键流程在 [consensus/pot/worker.go](../../consensus/pot/worker.go#L850-L950) 的 `GetIncentiveTxFromExecutor` 函数中：

```go
func (w *Worker) GetIncentiveTxFromExecutor(epoch uint64) ([]*types.ExecutedBlock, error) {
    // 从执行器获取已执行的交易块和 BCI 奖励
    // 验证每个 BciReward 并加入内存池
}
```

### 2. 产生机制

- 执行器在执行交易后生成 `BciReward`
- Worker 通过 gRPC 调用从执行器获取这些奖励
- 每个奖励包含：
  - 接收地址（从交易数据中提取）
  - 奖励金额
  - 证明信息（高度、区块哈希、交易哈希）

### 3. 奖励验证

在加入内存池前，每个 BciReward 都会经过验证（见 [consensus/pot/bci.go](../../consensus/pot/bci.go#L60-L92) `VerifyBciReward`）：

```go
func (w *Worker) VerifyBciReward(reward *BciReward) (bool, *types.ExecutedTx, error) {
    // 1. 检查证明高度不超过当前执行高度
    // 2. 根据区块哈希在内存池中查找对应区块
    // 3. 在区块中查找对应的交易
    // 4. 验证交易哈希匹配
}
```

## BCI 的记录

### 1. 内存池管理

验证通过的 BCI 奖励存储在 Worker 的内存池中（[consensus/pot/mempool.go](../../consensus/pot/mempool.go#L372-L390)）：

```go
func (c *Mempool) AddBciReward(rewards ...*BciReward) {
    // 使用 "TxHash-Address" 作为唯一键
    // 存储为 WrappedBciReward，包含 proposed 标志
}
```

内存池使用 map 结构存储：
- **键**: `hexutil.Encode(TxHash) + "-" + hexutil.Encode(Address)`
- **值**: `WrappedBciReward`（包含奖励和提议状态标志）

### 2. 状态管理

内存池提供以下管理方法：
- `AddBciReward`: 添加新奖励
- `GetAllBciRewards`: 获取所有未提议的奖励
- `MarkBciRewardProposed`: 标记奖励已被打包进区块
- `RemoveBciRewardByTxHash`: 移除指定奖励
- `HasBciReward`: 检查奖励是否存在
- `HasBciRewardByCoinbaseProof`: 检查 coinbase proof 对应的奖励是否存在

### 3. Coinbase 交易记录

BCI 奖励最终记录在区块的 coinbase 交易中。每个 BciReward 会转换为 `CoinbaseProof`：

```go
type CoinbaseProof struct {
    TxHash  []byte   // 来源交易哈希
    Address []byte   // 接收地址
    Amount  int64    // 奖励金额
    Type    int32    // BCI 类型
    DoDraw  bool     // 是否参与抽样
}
```

## BCI 数量的决定

### 1. 总奖励计算

区块总奖励由 `CalcTotalReward` 函数计算（[consensus/pot/bci.go](../../consensus/pot/bci.go#L1460-L1469)）：

```go
func CalcTotalReward(height uint64) int64 {
    year := float64(height) / float64(OneYear*2)  // 每两年一个周期
    if year < 1 {
        return TotalReward  // 初始奖励 65536
    }
    halfTimes := math.Floor(math.Log2(year))
    return int64(math.Floor(float64(TotalReward) * math.Pow(0.5, halfTimes)))
}
```

**奖励衰减规则**：
- 初始每区块奖励：65536
- 每两年（2 * 365 * 144 个区块）衰减一半
- 采用对数衰减模型

### 2. 奖励分配比例

总奖励按预定义比例（`bcimap`）分配给不同角色（[consensus/pot/bci.go](../../consensus/pot/bci.go#L26-L36)）：

```go
var bcimap = map[int32]float64{
    exchequer:       0.3,   // 国库：30%
    Miner:           0.5,   // 矿工：50%
    UncleBlockMiner: 0.02,  // 叔块矿工：2%
    CommitteeLeader: 0.2,   // 委员会领导：20%
    CommitteeMember: 0.1,   // 委员会成员：10%
}
```

**注意**: 这些比例之和超过 100%（112%），表明可能存在部分角色重叠或特定分配逻辑。

### 3. Coinbase 交易生成

区块打包时，矿工调用 `GenerateCoinbaseTx` 生成 coinbase 交易（[consensus/pot/bci.go](../../consensus/pot/bci.go#L842-L943)）：

#### 矿工奖励

矿工固定获得总奖励的 50%：

```go
minerout := types.TxOutput{
    Address: pubkeybyte,
    Value:   int64(math.Floor(float64(totalreward) * bcimap[Miner])),  // 50%
    LockTime: CoinbaseLock,  // 锁定 6 个区块
}
```

#### 其他奖励的抽样分配

对于内存池中的 BCI 奖励（如叔块矿工、委员会成员等），采用**权重抽样**机制：

1. **按类型分组**：将 BciRewards 按 BciType（链 ID）分组
   ```go
   groupsdata := groupByChainID(Bcirewards)
   ```

2. **计算权重**：每个奖励的权重 = 其金额 / 同类型总金额
   ```go
   for _, reward := range rewards {
       reward.weight = float64(reward.Amount) / float64(total)
   }
   ```

3. **VDF 随机抽样**：使用 VDF0 输出作为随机种子，进行 `Selectn` 次抽样（当前 Selectn=1）
   ```go
   rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
   for i := 0; i < Selectn; i++ {
       r := rand.Float64()  // 生成 [0,1) 随机数
       acnum := 0.0
       for _, reward := range rewards {
           acnum += reward.weight
           if r <= acnum {
               selectreward[chainID] = append(selectreward[chainID], reward)
               break  // 找到一个即停止
           }
       }
   }
   ```

4. **生成输出**：为选中的奖励创建 TxOutput
   ```go
   case UncleBlockMiner:
       for _, reward := range rewards {
           txout := types.TxOutput{
               Address: reward.Address,
               Value:   int64(math.Floor(float64(totalreward) * bcimap[UncleBlockMiner] / float64(lenreward))),
               LockTime: CoinbaseLock,
           }
       }
   ```

### 4. 锁定期

所有 coinbase 输出都有 `CoinbaseLock = 6` 个区块的锁定期，即奖励在 6 个区块后才能使用。

## 完整流程总结

```
执行器 (Executor)
    ↓ 执行交易，生成 BciReward
    ↓
Worker.GetIncentiveTxFromExecutor
    ↓ 验证 (VerifyBciReward)
    ↓
内存池 (Mempool.AddBciReward)
    ↓ 存储为 WrappedBciReward
    ↓
矿工挖矿成功
    ↓
GenerateCoinbaseTx
    ↓ 1. 矿工获得固定 50%
    ↓ 2. 其他奖励按权重抽样分配
    ↓
Coinbase 交易打包进区块
    ↓
区块确认后（ConfirmDelay=6）
    ↓
奖励可用（UTXO）
```

## 关键常量

定义在 [consensus/pot/worker.go](../../consensus/pot/worker.go#L50-L65):

- `TotalReward = 65536`: 初始每区块总奖励
- `Selectn = 1`: 每类奖励抽样次数
- `CoinbaseLock = 6`: Coinbase 输出锁定期（区块数）
- `ConfirmDelay = 6`: 区块确认延迟
- `OneYear = 365 * 144`: 一年的区块数（假设每 10 分钟一个区块）

## 网络通信

### 发送 BCI 奖励

客户端/节点可通过 gRPC 接口 `SendBci` 提交 BCI 奖励（[consensus/pot/bci.go](../../consensus/pot/bci.go#L95-L104)）：

```go
func (w *Worker) SendBci(ctx context.Context, request *pb.SendBciRequest) (*pb.SendBciResponse, error) {
    // 1. 广播到网络
    err := w.broadcastSendBciRequest(request)
    // 2. 本地处理
    return w.handleSendBciRequest(request)
}
```

奖励会被封装为 `PoTMessage` 类型为 `SendBci_Request` 的消息广播到 P2P 网络。

## 安全性保证

1. **双重验证**：BCI 奖励在加入内存池和区块验证时都会被检查
2. **区块高度限制**：奖励证明的高度不能超过当前执行高度
3. **交易存在性验证**：必须在对应区块中找到证明中的交易
4. **防重复**：内存池使用 TxHash-Address 作为唯一键防止重复
5. **VDF 随机性**：使用 VDF 输出作为随机源，确保分配的不可预测性和公平性
