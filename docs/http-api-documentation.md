# PoT Blockchain HTTP API 文档

## 概述

PoT (Proof of Time) 区块链 HTTP API 提供了与区块链网络交互的 RESTful 接口。该 API 运行在端口 `18025` 上，支持多种交易类型的创建和查询功能。

**基础信息：**
- 服务端口: `18025`
- 基础路径: `/api`
- 数据格式: `JSON`
- 响应格式: 统一的 JSON 格式

## 通用响应格式

所有 API 接口都遵循以下响应格式：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}  // 可选，部分接口包含额外数据
}
```

### 状态码说明

- `200`: 请求成功
- `500`: 服务器内部错误

## 数据类型定义

### HTTPTransaction 交易结构

```json
{
  "txid": "string",           // 交易ID (十六进制字符串，可为空，系统自动计算)
  "txInputs": [               // 交易输入数组 (基于UTXO模型)
    {
      "txid": "string",       // 引用的UTXO交易ID (十六进制)
      "voutput": "string",    // UTXO输出索引 (字符串数字)
      "scriptSig": "string",  // 签名数据 (十六进制，PQC签名)
      "value": "string",      // 输入金额 (字符串数字，必须与UTXO一致)
      "address": "string",    // 输入地址 (十六进制，PQC公钥)
      "bciType": "string"     // BCI类型 (字符串数字，0-4)
    }
  ],
  "txOutputs": [              // 交易输出数组
    {
      "address": "string",    // 接收地址 (十六进制，PQC公钥)
      "value": "string",      // 输出金额 (字符串数字)
      "interest": "string",   // 利息金额 (字符串数字)
      "proof": "string",      // 证明数据 (十六进制，可为空)
      "lockTime": "string",   // 锁定时间 (Unix时间戳字符串)
      "bciType": "string",    // BCI类型 (字符串数字，0-4)
      "data": "string",       // 附加数据 (十六进制，可包含智能合约数据)
      "burnLock": "string",   // 销毁锁定时间 (Unix时间戳字符串)
      "rate": "string"        // 利率 (字符串浮点数，如"0.05"表示5%)
    }
  ],
  "transactionFee": "string"  // 交易手续费 (字符串数字，单位：最小货币单位)
}
```

### 请求数据格式

```json
{
  "transaction": {
    // HTTPTransaction 结构
  },
  "type": "string"  // 交易类型标识 (用于日志和验证)
}
```

### 实际交易创建流程

**基于 PoT 区块链的实际实现，交易创建遵循以下流程：**

1. **获取 PQC 密钥对**
   - 通过 gRPC `GetPqcKey` 接口获取后量子密码学密钥
   - 公钥作为地址使用
   - 私钥用于签名交易

2. **查询可用 UTXO**
   - 通过 gRPC `GetBalance` 接口查询地址的UTXO
   - 选择合适的UTXO作为交易输入

3. **构建交易输入**
   - 使用UTXO的交易ID和输出索引
   - 用私钥对"txid:voutput"字符串进行PQC签名
   - 填入相关地址和金额信息

4. **构建交易输出**
   - 设置接收地址和金额
   - 根据交易类型设置相应的锁定时间和利率
   - 确保输入金额 = 输出金额 + 手续费

5. **提交交易**
   - 通过HTTP接口提交完整的交易数据
   - 系统会验证签名和余额
   - 广播到网络并加入内存池
```

## API 接口详情

### 1. 健康检查接口

检查服务状态的简单接口。

**接口信息:**
- **URL**: `/api/hello`
- **方法**: `POST`
- **Content-Type**: `application/json`

**请求参数:**
```json
{}  // 空对象
```

**响应示例:**
```json
{
  "code": 200,
  "msg": "success"
}
```

---

### 2. 创建锁定交易

创建一个新的锁定交易，用于将资产锁定在区块链上。锁定交易可以设置锁定时间和利率。

**接口信息:**
- **URL**: `/api/createlocktransaction`
- **方法**: `POST`
- **Content-Type**: `application/json`

**前置条件:**
1. 需要有可用的 UTXO (未花费交易输出)
2. 需要对应的 PQC 私钥进行签名
3. 确保输入金额大于等于输出金额加手续费

**请求参数示例 (基于实际UTXO):**
```json
{
  "transaction": {
    "txid": "",  // 留空，系统自动计算
    "txInputs": [
      {
        "txid": "0xa1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
        "voutput": "0",
        "scriptSig": "0x308201a0300d06092a864886f70d01010105000382018d00308201880282018100...", // PQC签名 (实际长度约4KB)
        "value": "1000000",
        "address": "0x5f41dce11f9b9a19b92c6dad6d2b8b8f00000000000000007695cd781cde492f5254a88066024237129eb549b0864c8f", // PQC公钥
        "bciType": "1"  // 矿工类型
      }
    ],
    "txOutputs": [
      {
        "address": "0x742d35cc6e11111111111111111111111111111111111111111111111111", 
        "value": "999000",  // 锁定金额
        "interest": "0",    // 初始利息为0
        "proof": "0x",
        "lockTime": "1672531200",  // Unix时间戳：2023-01-01
        "bciType": "1",
        "data": "0x",
        "burnLock": "0",    // 非销毁交易
        "rate": "0.05"     // 5% 年化利率
      }
    ],
    "transactionFee": "1000"  // 手续费
  },
  "type": "CreateLockTransaction"
}
```

**签名生成示例 (基于实际代码):**
```go
// 1. 获取UTXO信息
utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(utxo.GetTxid()), utxo.GetVoutput())
// 例如: "0xa1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456:0"

// 2. 使用PQC私钥签名
pqckey := &crypto.PqcKey{
    Privkey: secretKey,  // 从GetPqcKey获取
    Pubkey:  publicKey,  // 从GetPqcKey获取
    Scheme:  crypto.PqcScheme,
}
signature, err := pqckey.Sign([]byte(utxokey))

// 3. 将签名转换为十六进制字符串用于HTTP请求
scriptSig := hexutil.Encode(signature)
```

**响应示例:**
```json
{
  "code": 200,
  "msg": "success"
}
```

**错误响应示例:**
```json
{
  "code": 500,
  "msg": "transaction validation failed: insufficient balance"
}
```

**常见错误:**
- `invalid UTXO`: 引用的UTXO不存在或已被使用
- `invalid signature`: PQC签名验证失败
- `insufficient balance`: 输入金额小于输出金额加手续费
- `invalid lock time`: 锁定时间格式错误或小于当前时间
```

---

### 3. 锁定转账交易

执行锁定资产之间的转账操作。

**接口信息:**
- **URL**: `/api/locktransfertransaction`
- **方法**: `POST`
- **Content-Type**: `application/json`

**请求参数:**
```json
{
  "transaction": {
    "txid": "0x2345678901bcdefab",
    "txInputs": [
      {
        "txid": "0x1234567890abcdef",
        "voutput": "0",
        "scriptSig": "0x4730440220...",
        "value": "999000",
        "address": "0x742d35cc6e11111111111111",
        "bciType": "1"
      }
    ],
    "txOutputs": [
      {
        "address": "0x742d35cc6e22222222222222",
        "value": "998000",
        "interest": "0",
        "proof": "0x",
        "lockTime": "1640995200",
        "bciType": "1",
        "data": "0x",
        "burnLock": "0",
        "rate": "0.0"
      }
    ],
    "transactionFee": "1000"
  },
  "type": "LockTransferTransaction"
}
```

**响应示例:**
```json
{
  "code": 200,
  "msg": "success"
}
```

---

### 4. 非锁定转账交易

执行普通的非锁定资产转账。

**接口信息:**
- **URL**: `/api/nonlocktransfertransaction`
- **方法**: `POST`
- **Content-Type**: `application/json`

**请求参数:**
```json
{
  "transaction": {
    "txid": "0x3456789012cdefabc",
    "txInputs": [
      {
        "txid": "0x2345678901bcdefab",
        "voutput": "0",
        "scriptSig": "0x4730440220...",
        "value": "500000",
        "address": "0x742d35cc6e33333333333333",
        "bciType": "0"
      }
    ],
    "txOutputs": [
      {
        "address": "0x742d35cc6e44444444444444",
        "value": "499000",
        "interest": "0",
        "proof": "0x",
        "lockTime": "0",
        "bciType": "0",
        "data": "0x",
        "burnLock": "0",
        "rate": "0.0"
      }
    ],
    "transactionFee": "1000"
  },
  "type": "NonLockTransferTransaction"
}
```

**响应示例:**
```json
{
  "code": 200,
  "msg": "success"
}
```

---

### 5. 销毁交易

执行资产销毁操作，永久性地从流通中移除资产。销毁交易可以包含以太坊跨链数据。

**接口信息:**
- **URL**: `/api/devastatetransaction`
- **方法**: `POST`
- **Content-Type**: `application/json`

**特殊说明:**
- 销毁交易的输出地址必须是零地址 (全0地址)
- 可以在 `data` 字段中包含跨链交易数据
- `burnLock` 字段设置销毁锁定时间

**请求参数示例 (基于实际实现):**
```json
{
  "transaction": {
    "txid": "",  // 留空，系统自动计算
    "txInputs": [
      {
        "txid": "0xa1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
        "voutput": "0",
        "scriptSig": "0x308201a0300d06092a864886f70d01010105000382018d00...", // PQC签名
        "value": "500000",
        "address": "0x5f41dce11f9b9a19b92c6dad6d2b8b8f00000000000000007695cd781cde492f5254a88066024237129eb549b0864c8f",
        "bciType": "0"  // 国库类型
      }
    ],
    "txOutputs": [
      {
        "address": "0x0000000000000000000000000000000000000000000000000000000000000000", // 销毁地址(零地址)
        "value": "0",       // 销毁后价值为0
        "interest": "0",
        "proof": "0x",
        "lockTime": "0",    // 销毁交易无锁定时间
        "bciType": "2",     // 销毁类型
        "data": "0x02f86c808504a817c800827530948ba1cf949857e...", // 跨链交易数据(以太坊交易的编码数据)
        "burnLock": "1672531200",  // 销毁锁定时间
        "rate": "0.0"
      }
    ],
    "transactionFee": "5000"  // 销毁交易通常手续费较高
  },
  "type": "DevastateTransaction"
}
```

**跨链数据构建示例 (基于实际代码):**
```go
// 1. 构建以太坊交易
amountByte := big.NewInt(amount)
data := append([]byte{0x0D, 0x02}, amountByte.Bytes()...)
ethTx := ethtype.NewTransaction(nonce, toAddress, big.NewInt(amount), gasLimit, gasPrice, data)

// 2. 签名以太坊交易
signedTx, err := ethtype.SignTx(ethTx, ethtype.NewEIP155Signer(chainID), ethPrivateKey)

// 3. 编码为字节数组
txData, err := signedTx.MarshalBinary()

// 4. 转换为十六进制字符串用于HTTP请求
dataHex := hexutil.Encode(txData)
```

**销毁地址常量 (基于实际代码):**
```go
// pot.BurnoutAddress - 系统定义的销毁地址
BurnoutAddress = []byte{0x00, 0x00, 0x00, 0x00, ...} // 全零地址
```

**响应示例:**
```json
{
  "code": 200,
  "msg": "success"
}
```

**错误响应示例:**
```json
{
  "code": 500,
  "msg": "devastate transaction validation failed: invalid burn address"
}
```

**销毁交易验证规则:**
- 输出地址必须是预定义的销毁地址
- 输出价值必须为0
- 必须设置正确的 `bciType` 为销毁类型
- 可选的跨链数据必须符合对应链的格式
```

---

### 6. 获取区块高度

查询当前区块链的最新区块高度。

**接口信息:**
- **URL**: `/api/getblockheight`
- **方法**: `GET`

**请求参数:**
无需参数

**响应示例:**
```json
{
  "code": 200,
  "msg": "success",
  "height": 12345
}
```

---

## BCI 类型说明

BCI (Blockchain Credit Index) 类型标识不同的账户或交易类型：

- `0`: Exchequer (国库)
- `1`: Miner (矿工)
- `2`: UncleBlockMiner (叔块矿工)
- `3`: CommitteeLeader (委员会领导者)
- `4`: CommitteeMember (委员会成员)

## 错误处理

### 常见错误类型

1. **JSON 解析错误**
   ```json
   {
     "code": 500,
     "msg": "decode request body error"
   }
   ```

2. **交易验证失败**
   ```json
   {
     "code": 500,
     "msg": "transaction validation failed: 具体原因"
   }
   ```

3. **广播失败**
   ```json
   {
     "code": 500,
     "msg": "broadcast transaction failed: 具体原因"
   }
   ```

4. **十六进制解码错误**
   ```json
   {
     "code": 500,
     "msg": "invalid hex string"
   }
   ```

## 使用示例

### 完整的交易创建流程 (基于实际实现)

#### 1. 获取密钥和查询余额

**第一步：获取PQC密钥对**
```bash
# 通过gRPC客户端获取密钥 (需要指定区块高度)
curl -X POST http://localhost:9866/GetPqcKey \
  -H "Content-Type: application/json" \
  -d '{"height": 1}'
```

**第二步：查询地址余额和UTXO**
```bash
# 使用获取的公钥查询余额
curl -X POST http://localhost:9866/GetBalance \
  -H "Content-Type: application/json" \
  -d '{"address": "0x5f41dce11f9b9a19b92c6dad6d2b8b8f..."}'
```

#### 2. 创建锁定交易完整示例

```javascript
// JavaScript 示例代码
async function createLockTransaction() {
  // 1. 准备UTXO数据 (从GetBalance响应中获取)
  const utxo = {
    txid: "0xa1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
    voutput: 0,
    value: 1000000,
    address: "0x5f41dce11f9b9a19b92c6dad6d2b8b8f..."
  };
  
  // 2. 生成签名字符串
  const utxoKey = `${utxo.txid}:${utxo.voutput}`;
  
  // 3. 使用PQC私钥签名 (需要通过gRPC客户端完成)
  const signature = await signWithPQC(utxoKey, privateKey);
  
  // 4. 构建HTTP请求
  const transaction = {
    txid: "",  // 留空，系统计算
    txInputs: [{
      txid: utxo.txid,
      voutput: utxo.voutput.toString(),
      scriptSig: signature,
      value: utxo.value.toString(),
      address: utxo.address,
      bciType: "1"
    }],
    txOutputs: [{
      address: "0x742d35cc6e22222222222222222222222222222222222222222222222222",
      value: "995000",  // 原金额减去手续费
      interest: "0",
      proof: "0x",
      lockTime: "1672531200",  // 锁定到2023年
      bciType: "1",
      data: "0x",
      burnLock: "0",
      rate: "0.05"  // 5%年化收益
    }],
    transactionFee: "5000"
  };
  
  // 5. 发送HTTP请求
  const response = await fetch('http://localhost:18025/api/createlocktransaction', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ transaction, type: "CreateLockTransaction" })
  });
  
  return await response.json();
}
```

#### 3. 销毁交易与跨链操作

```javascript
async function createDevastateTransaction() {
  // 1. 构建以太坊跨链交易数据
  const ethTxData = await buildEthereumTransaction({
    to: "0x742d35cc6e33333333333333333333333333333333333333333333333333",
    value: "500000",
    gasLimit: 21000,
    gasPrice: "20000000000"
  });
  
  // 2. 构建销毁交易
  const transaction = {
    txid: "",
    txInputs: [{
      txid: "0xa1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
      voutput: "0",
      scriptSig: await signWithPQC(utxoKey, privateKey),
      value: "500000",
      address: "0x5f41dce11f9b9a19b92c6dad6d2b8b8f...",
      bciType: "0"
    }],
    txOutputs: [{
      address: "0x0000000000000000000000000000000000000000000000000000000000000000", // 销毁地址
      value: "0",
      interest: "0",
      proof: "0x",
      lockTime: "0",
      bciType: "2",  // 销毁类型
      data: ethTxData,  // 以太坊交易数据
      burnLock: Math.floor(Date.now() / 1000).toString(),
      rate: "0.0"
    }],
    transactionFee: "5000"
  };
  
  // 3. 提交销毁交易
  const response = await fetch('http://localhost:18025/api/devastatetransaction', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ transaction, type: "DevastateTransaction" })
  });
  
  return await response.json();
}
```

### 转账流程决策树

```
开始
 ↓
是否已锁定? 
 ├─ 是 → 使用 /api/locktransfertransaction
 └─ 否 → 使用 /api/nonlocktransfertransaction
      ↓
    需要销毁?
     ├─ 是 → 使用 /api/devastatetransaction
     └─ 否 → 继续普通转账
```

### 错误处理最佳实践

```javascript
async function handleTransactionError(error) {
  switch(error.msg) {
    case "get block error for block is not found":
      // 区块高度不存在，尝试使用更高的高度
      return await retryWithHigherHeight();
      
    case "transaction validation failed: insufficient balance":
      // 余额不足，检查UTXO状态
      return await checkUTXOStatus();
      
    case "invalid signature":
      // 签名无效，重新生成签名
      return await regenerateSignature();
      
    default:
      console.error("Unknown error:", error.msg);
      throw error;
  }
}
```

## 注意事项

### 重要技术细节

1. **UTXO 模型**: PoT区块链使用UTXO (未花费交易输出) 模型，类似比特币
   - 每笔交易必须引用具体的UTXO作为输入
   - 输入金额必须与引用的UTXO金额完全一致
   - 未使用的金额需要作为"找零"输出返回给自己

2. **后量子密码学 (PQC)**: 使用抗量子攻击的签名算法
   - 公钥长度约96字节，私钥更长
   - 签名数据通常4KB左右，比传统ECDSA大很多
   - 地址直接使用PQC公钥的十六进制表示

3. **数据格式要求**:
   - 所有数值以字符串形式传输，避免JSON精度丢失
   - 十六进制数据必须包含 `0x` 前缀
   - Unix时间戳使用秒级精度的字符串

4. **BCI (Blockchain Credit Index) 系统**:
   - `0`: Exchequer (国库) - 系统奖励和治理
   - `1`: Miner (矿工) - 挖矿奖励
   - `2`: UncleBlockMiner (叔块矿工) - 叔块奖励  
   - `3`: CommitteeLeader (委员会领导者) - 共识参与
   - `4`: CommitteeMember (委员会成员) - 共识参与

### 交易验证规则

1. **输入验证**:
   - 引用的UTXO必须存在且未被使用
   - PQC签名必须正确验证
   - 输入地址必须与UTXO的输出地址匹配

2. **输出验证**:
   - 输出金额必须为正数
   - 锁定时间不能早于当前时间 (对于锁定交易)
   - 销毁交易的输出地址必须为零地址

3. **余额验证**:
   ```
   总输入金额 = 总输出金额 + 交易手续费
   ```

### 错误处理指南

1. **连接错误**:
   - HTTP服务 (端口18025): 检查HTTP服务器状态
   - gRPC服务 (端口9866): 检查BCI服务器状态

2. **数据格式错误**:
   ```json
   {"code": 500, "msg": "decode request body error"}
   ```
   - 检查JSON格式是否正确
   - 确认所有必需字段都已提供

3. **签名验证错误**:
   ```json
   {"code": 500, "msg": "invalid signature"}
   ```
   - 确认使用正确的私钥
   - 检查签名的数据格式 (应为 "txid:voutput")
   - 验证PQC签名算法参数

4. **UTXO相关错误**:
   ```json
   {"code": 500, "msg": "UTXO not found"}
   ```
   - UTXO已被其他交易使用
   - 引用的交易ID或输出索引错误
   - 需要重新查询可用UTXO

### 性能考虑

1. **签名性能**: PQC签名比传统签名慢，典型签名时间：
   - 签名: ~50-100ms
   - 验证: ~10-20ms

2. **交易大小**: 由于PQC签名较大，单笔交易大小约5-10KB

3. **网络延迟**: 
   - HTTP接口响应时间: 10-100ms
   - gRPC接口响应时间: 5-50ms

### 开发调试技巧

1. **使用测试地址**:
   ```
   测试地址: 0x5f41dce11f9b9a19b92c6dad6d2b8b8f00000000000000007695cd781cde492f5254a88066024237129eb549b0864c8f
   ```

2. **查看详细日志**: 服务器端会输出详细的验证错误信息

3. **分步调试**:
   - 先调用 `/api/hello` 确认服务可用
   - 再调用 `/api/getblockheight` 检查区块链状态
   - 最后提交实际交易

4. **异步处理**: 交易提交成功不等于立即确认
   - 交易首先进入内存池
   - 等待矿工打包到区块中
   - 可通过区块高度变化确认交易状态

## 测试环境

- **测试网络**: 本地测试网络
- **服务地址**: `http://localhost:18025`
- **测试工具**: 支持 REST Client、Postman、curl 等

## 版本信息

- **API 版本**: v1.0
- **最后更新**: 2025-10-28
- **兼容性**: Go 1.22+