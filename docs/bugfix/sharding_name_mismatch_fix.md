# Sharding名称不匹配问题修复

## 问题现象

交易始终无法路由到Whirly共识层，错误日志：
```log
16:17:53.28229 [DEBUG] [WHIRLY.NODECTRL] Received request | sharding=0x1
16:17:53.28232 [ERROR] Failed to route request: sharding not found | c_id=1009 sharding=0x1
```

## 根本原因

### 客户端发送的Sharding名称
**位置**: [cmd/executor/main.go:183](cmd/executor/main.go#L183)
```go
request := &pb.Request{Tx: btx, Sharding: []byte("0x1")}
```
→ Sharding名称: `"0x1"` (3字符)

### PoT生成的Sharding名称
**位置**: [consensus/pot/run_commitee.go:176](consensus/pot/run_commitee.go#L176)
```go
sharding1 := nodeController.PoTSharding{
    Name: hexutil.EncodeUint64(1),  // ← 问题所在
    ...
}
```
→ `hexutil.EncodeUint64(1)` 生成: `"0x0000000000000001"` (18字符，8字节完整编码)

### Sharding查找失败
**位置**: [nodeController.go:148](consensus/whirly/nodeController/nodeController.go#L148)
```go
shardingName := string(req.Sharding)  // "0x1"
sharding, ok := nc.Shardings[shardingName]  // 查找 "0x1"
if !ok {
    nc.Log.Error("Failed to route request: sharding not found")  // ← 失败
}
```

**nc.Shardings映射内容**:
```go
{
    "0x0000000000000001": <Sharding对象>  // ← 键不匹配!
}
```

**结果**: `"0x1"` ≠ `"0x0000000000000001"` → 查找失败 → 交易丢弃

## 完整的失败链路

```
客户端发送交易
  ↓
Request.Sharding = "0x1"
  ↓
PoT接收并转发给Whirly NodeController
  ↓
NodeController查找 nc.Shardings["0x1"]
  ↓
映射中只有 "0x0000000000000001" 键
  ↓
查找失败 → ERROR: sharding not found
  ↓
交易被丢弃 ❌
```

## 修复方案

### 修改内容
**文件**: [consensus/pot/run_commitee.go](consensus/pot/run_commitee.go#L176)

**修改前**:
```go
sharding1 := nodeController.PoTSharding{
    Name: hexutil.EncodeUint64(1),  // 生成 "0x0000000000000001"
    ...
}
```

**修改后**:
```go
sharding1 := nodeController.PoTSharding{
    Name: "0x1",  // 与客户端请求保持一致
    ...
}
```

### 修复原理

1. **统一命名格式**: 使用简洁的十六进制表示 `"0x1"` 而非完整的8字节编码
2. **客户端请求兼容**: 确保Sharding名称与`pb.Request.Sharding`字段值完全匹配
3. **简化映射查找**: 减少字符串长度，提高查找效率

## 验证步骤

重新运行测试，应该看到：

**成功日志**:
```log
[DEBUG] [WHIRLY.NODECTRL] Received request | sharding=0x1
[INFO] [WHIRLY.SHARD] Initializing new sharding | sharding=0x1
[DEBUG] [WHIRLY.SHARD] Creating daemon node for sharding | daemon_node=...
[DEBUG] Initializing daemon node with leader and epoch | epoch=X
[DEBUG] [WHIRLY.SHARD] Distributing request to all nodes | tx_count=X
```

**不应再出现**:
```log
[ERROR] Failed to route request: sharding not found  ❌
```

## 附加说明

### 为什么之前的Leader修复没有生效？

之前的修复（Daemon节点初始化leader、PoT信号更新节点、GetLeader降级策略）都是**正确的**，但：
- 这些修复解决的是**leader映射为空**的问题
- 但如果**sharding本身就不存在**，根本不会走到GetLeader逻辑
- **Sharding查找失败** → 请求被直接丢弃 → leader逻辑完全没有执行

### 问题优先级

1. **最高优先级**: Sharding名称匹配（本次修复）← 如果这个不通，后续都无法执行
2. **高优先级**: Sharding初始化（NewSharding创建daemon节点并设置leader）
3. **中优先级**: PoT信号同步（更新所有节点的epoch和leader）
4. **防御性措施**: GetLeader降级策略（容错机制）

### 其他潜在问题

如果修复后仍有问题，检查以下几点：

1. **PoT信号是否发送**:
   ```log
   [DEBUG] Sending PoT signal for committee update | shardings=1
   ```

2. **Sharding是否创建成功**:
   ```log
   [INFO] Creating new sharding | sharding=0x1
   ```

3. **Daemon节点是否初始化**:
   ```log
   [DEBUG] Initializing daemon node with leader and epoch
   ```

4. **Committee是否为空**:
   ```log
   [DEBUG] Sending PoT signal | self_address_count=X  ← 应该 > 0
   ```

## 修复提交信息

```
fix: 修复sharding名称不匹配导致交易路由失败

问题：
- 客户端请求使用 "0x1" 作为sharding名称
- PoT使用 hexutil.EncodeUint64(1) 生成 "0x0000000000000001"
- 字符串不匹配导致sharding查找失败，交易被丢弃

修复：
- 将sharding名称改为简洁格式 "0x1"
- 与客户端请求保持一致
- 确保NodeController.Shardings映射可以正确查找

影响：
- 解决交易无法路由到Whirly共识层的问题
- 交易现在可以正常转发到daemon节点处理
```

## 相关文件

- 客户端请求: [cmd/executor/main.go:183](cmd/executor/main.go#L183)
- PoT Sharding生成: [consensus/pot/run_commitee.go:176](consensus/pot/run_commitee.go#L176)
- Sharding路由: [nodeController.go:148](consensus/whirly/nodeController/nodeController.go#L148)
- 协议定义: [pkg/proto/common.proto:40](pkg/proto/common.proto#L40)
