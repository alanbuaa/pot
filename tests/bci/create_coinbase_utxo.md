# BCI 系统中如何添加 UTXO 交易给某个地址

## 问题分析

当前错误：
```
❌ No address with available UTXOs found in heights 1-100
💡 Possible reasons:
   1. The node hasn't mined enough blocks yet
   2. Coinbase transactions haven't reached confirmation height (ConfirmDelay)
   3. Try running: make run_server (and wait for mining)
```

**根本原因**：系统中还没有可用的 UTXO（未花费交易输出）。

## 解决方案

### ✅ 方法 1: 运行节点挖矿产生 Coinbase 交易（推荐）

这是最正规的方式，通过 POT 共识挖矿自然产生 UTXO。

#### 步骤：

1. **启动共识节点**
   ```bash
   # 终端 1: 启动 executor
   make run_executor
   
   # 终端 2: 启动 server（共识节点）
   make run_server
   ```

2. **等待挖矿**
   - 节点会自动挖矿，每个区块包含一个 **Coinbase 交易**
   - Coinbase 交易自动给矿工地址创建 UTXO
   - 需要等待至少 **ConfirmDelay** 个区块（通常 6-10 个）后，UTXO 才可用

3. **运行测试代码**
   ```bash
   go run tests/bci/main.go
   ```
   - 修改后的代码会自动搜索高度 1-100，找到第一个有余额的地址
   - 找到后会显示地址、余额、UTXO 数量

#### Coinbase 交易的特点

根据代码分析（`types/txdata.go:438`）：
```go
func (r *RawTx) IsCoinBase() bool {
    return len(r.TxInput) == 1 && r.TxInput[0].Voutput == -1
}
```

Coinbase 交易：
- ✅ 只有 1 个 TxInput
- ✅ TxInput 的 `Voutput = -1`（标记为 coinbase）
- ✅ TxOutput[0] 包含矿工地址和奖励金额
- ✅ 无需输入签名（因为是新创建的币）

#### Coinbase 创建位置

在 `consensus/pot/bci.go:954`：
```go
func (w *Worker) GenerateCoinbaseTxWithoutMinerKey(...) *types.RawTx {
    // Coinbase 输入（无前置交易）
    txin := types.TxInput{
        Txid:      [32]byte{},    // 空
        Voutput:   -1,             // 标记为 coinbase
        Scriptsig: nil,
        Value:     0,
        Address:   []byte{},
    }
    
    // 矿工奖励输出
    minerout := types.TxOutput{
        Address:  pubkeybyte,      // 矿工的 PQC 公钥
        Value:    totalreward,     // 奖励金额
        LockTime: CoinbaseLock,    // 锁定期
    }
    
    tx := &types.RawTx{
        Txid:           [32]byte{},
        TxInput:        []types.TxInput{txin},
        TxOutput:       []types.TxOutput{minerout},
        CoinbaseProofs: coinbaseproof,
    }
    return tx
}
```

### 方法 2: 手动创建测试 Coinbase（开发调试）

如果你不想等待挖矿，可以手动创建测试 UTXO。

#### 创建辅助函数

在 `tests/bci/` 目录创建 `create_test_utxo.go`：

```go
package main

import (
    "context"
    "fmt"
    "github.com/zzz136454872/upgradeable-consensus/crypto"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
    "github.com/zzz136454872/upgradeable-consensus/types"
)

// 创建测试 Coinbase 交易
func createTestCoinbase(targetAddr []byte, amount int64) *types.RawTx {
    // Coinbase 输入
    txin := types.TxInput{
        Txid:      [32]byte{},
        Voutput:   -1,  // 关键：标记为 coinbase
        Scriptsig: nil,
        Value:     0,
        Address:   []byte{},
        BciType:   0,
    }
    
    // 目标地址输出
    txout := types.TxOutput{
        Address:   targetAddr,
        Value:     amount,
        ScriptPk:  targetAddr,
        LockTime:  7,
        BciType:   1,
        BurnLock:  0,
        Interest:  0,
        Rate:      0,
    }
    
    tx := &types.RawTx{
        Txid:           [32]byte{},
        TxInput:        []types.TxInput{txin},
        TxOutput:       []types.TxOutput{txout},
        CoinbaseProofs: []types.CoinbaseProof{},
        TransactionFee: 0,
    }
    
    // 计算并设置交易哈希
    tx.Txid = tx.Hash()
    return tx
}

// 发送测试 Coinbase（需要有对应的 gRPC 接口）
func sendTestCoinbase(client pb.BciExectorClient, targetAddr []byte, amount int64) error {
    tx := createTestCoinbase(targetAddr, amount)
    
    // 编码交易
    txProto := tx.ToProto()
    
    // 这里需要对应的 gRPC 接口来接收 coinbase 交易
    // 具体实现取决于你的系统设计
    fmt.Printf("Created test coinbase:\n")
    fmt.Printf("  TxID: %x\n", tx.Txid)
    fmt.Printf("  To: %x\n", targetAddr)
    fmt.Printf("  Amount: %d\n", amount)
    
    return nil
}
```

### 方法 3: 修改创世块预分配（永久方案）

修改创世块配置，在系统启动时预分配一些 UTXO。

查找 `consensus/pot/chainreader.go` 中的创世块定义：
```bash
grep -n "DefaultGenesisBlock\|GenesisBlock" consensus/pot/*.go
```

然后在创世块中添加初始 UTXO。

## 当前代码改进

已修改 `tests/bci/main.go`：

✅ **智能搜索**：自动遍历高度 1-100，查找第一个有余额的地址  
✅ **友好提示**：清晰说明为什么找不到 UTXO  
✅ **详细日志**：显示签名时间、验证时间、签名长度等  
✅ **错误处理**：优雅处理各种异常情况  

## 实际操作步骤

### 立即可行的方案：

**步骤 1**: 启动服务
```bash
# 终端 1
cd /home/ldc/workspace/pot
make run_executor

# 终端 2
make run_server
```

**步骤 2**: 等待挖矿
- 观察日志，等待至少 15+ 个区块
- 确保超过 `ConfirmDelay`（通常是 6-10 个区块）

**步骤 3**: 运行测试
```bash
go run tests/bci/main.go
```

### 预期输出

成功时：
```
[Init] VDF instance using 20 CPUs
🔍 Searching for an address with available UTXOs...
✅ Found address with balance at height 23
   Address: 0x3fb3f86611ac7cdf6e737df100bdf75e61636b0a66653030fbac70c79267d7d33d9568efe3194900101c9ee9a0a5d69d
   Balance: 1000000
   UTXOs: 1

🔑 UTXO key: 0xabcd...1234:0
💰 Amount: 1000000

🔐 Signing UTXO...
✅ Signature time: 45.23 ms
✅ Verification time: 12.34 ms
✅ Signature valid: true
📏 Signature length: 2420 bytes
```

## 关键参数说明

- **ConfirmDelay**: 交易确认延迟（6-10 个区块）
- **CoinbaseLock**: Coinbase 锁定期
- **TotalReward**: 矿工总奖励
- **bcimap[Miner]**: 矿工奖励比例

## 下一步

如果问题仍然存在：
1. 检查节点是否正常运行（查看日志）
2. 确认已挖出足够的区块（>15 个）
3. 检查 `config/config.yaml` 中的 `ConfirmDelay` 配置
4. 查看数据库中是否有区块数据：`ls -la data/node-0/`

需要我帮你进一步调试吗？

