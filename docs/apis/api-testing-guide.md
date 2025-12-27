# PoT API 测试使用指南

本指南将帮助您快速开始使用 PoT 区块链的 HTTP API。

## 📋 前置要求

1. **VS Code** 编辑器
2. **Rest Client** 扩展 (humao.rest-client)
3. **运行中的 PoT 服务器** (端口 18025)

## 🚀 快速开始

### 1. 安装 Rest Client 扩展

在 VS Code 中：
- 按 `Ctrl+Shift+X` 打开扩展面板
- 搜索 "Rest Client"
- 安装 "REST Client" by Huachao Mao

### 2. 启动 PoT 服务器

```bash
# 在项目根目录下
make run_server
```

### 3. 打开测试文件

在 VS Code 中打开 `docs/rest-client-tests.http` 文件。

### 4. 执行测试

点击每个请求上方的 `Send Request` 按钮执行测试。

## 🔧 基础测试流程

### 1. 健康检查

首先验证服务器是否正常运行：

```http
POST http://localhost:18025/api/hello
Content-Type: application/json

{}
```

**期望响应:**
```json
{
  "code": 200,
  "msg": "success"
}
```

### 2. 获取区块高度

检查当前区块链状态：

```http
GET http://localhost:18025/api/getblockheight
```

**期望响应:**
```json
{
  "code": 200,
  "msg": "success",
  "height": 12345
}
```

### 3. 创建锁定交易

创建一个新的锁定交易：

```http
POST http://localhost:18025/api/createlocktransaction
Content-Type: application/json

{
  "transaction": {
    "txid": "0x1234567890abcdef...",
    "txInputs": [...],
    "txOutputs": [...],
    "transactionFee": "1000"
  },
  "type": "CreateLockTransaction"
}
```

## 📊 测试数据说明

### 十六进制字段

以下字段需要使用十六进制格式：
- `txid`: 交易ID
- `address`: 地址
- `scriptSig`: 脚本签名
- `proof`: 证明数据
- `data`: 附加数据

### 数值字段

以下字段使用字符串格式的数字：
- `value`: 金额
- `transactionFee`: 交易费
- `interest`: 利息
- `lockTime`: 锁定时间
- `burnLock`: 销毁锁定时间
- `bciType`: BCI 类型
- `voutput`: 输出索引

### BCI 类型

- `0`: Exchequer (国库)
- `1`: Miner (矿工)  
- `2`: UncleBlockMiner (叔块矿工)
- `3`: CommitteeLeader (委员会领导者)
- `4`: CommitteeMember (委员会成员)

## 🧪 测试用例分类

### 基础功能测试

1. **健康检查** - 验证服务可用性
2. **区块高度查询** - 检查链状态
3. **单输入单输出交易** - 基础交易测试

### 复杂场景测试

1. **多输入多输出交易** - 复杂交易结构
2. **带利息的锁定交易** - 利息计算测试
3. **销毁交易** - 资产销毁功能

### 错误处理测试

1. **无效十六进制格式** - 数据格式验证
2. **JSON 解析错误** - 请求格式验证
3. **缺少必要字段** - 参数完整性验证
4. **数值格式错误** - 类型转换验证

### 性能和边界测试

1. **大交易测试** - 多输入输出性能
2. **最小交易费** - 边界条件测试
3. **零值交易** - 特殊场景测试

## 🔍 调试技巧

### 1. 查看响应详情

Rest Client 会在右侧面板显示完整的响应信息，包括：
- HTTP 状态码
- 响应头
- 响应体
- 响应时间

### 2. 保存响应数据

点击响应面板右上角的保存按钮，可以将响应保存为文件。

### 3. 生成代码片段

在响应面板中选择 "Generate Code Snippet" 可以生成对应的 curl、JavaScript 等代码。

### 4. 环境变量设置

在测试文件顶部设置变量：

```http
@baseUrl = http://localhost:18025
@contentType = application/json
```

## ⚠️ 常见问题

### 1. 连接被拒绝

**问题**: `ECONNREFUSED 127.0.0.1:18025`

**解决**: 确保 PoT 服务器正在运行

```bash
make run_server
```

### 2. JSON 解析错误

**问题**: `decode request body error`

**解决**: 检查 JSON 格式是否正确，移除注释

### 3. 十六进制解码失败

**问题**: `invalid hex string`

**解决**: 确保十六进制字符串格式正确，包含 `0x` 前缀

### 4. 交易验证失败

**问题**: `transaction validation failed`

**解决**: 检查交易数据的逻辑正确性，如输入输出金额平衡

## 📈 测试报告

### 成功测试记录

记录每次测试的结果：

```
✅ 健康检查 - 响应时间: 5ms
✅ 区块高度查询 - 当前高度: 1234
✅ 创建锁定交易 - 交易已提交
❌ 无效数据测试 - 预期错误已触发
```

### 性能基准

| 测试用例 | 平均响应时间 | 成功率 |
|---------|------------|--------|
| 健康检查 | 5ms | 100% |
| 区块高度查询 | 8ms | 100% |
| 创建锁定交易 | 50ms | 95% |
| 复杂交易 | 120ms | 90% |

## 🔗 相关资源

- [完整 API 文档](./http-api-documentation.md)
- [Rest Client 测试文件](./rest-client-tests.http)
- [项目 README](../README.md)
- [Quick Test Example](./quick-test-example.md)

## 💡 最佳实践

1. **测试顺序**: 先执行健康检查，再进行功能测试
2. **数据准备**: 使用不同的交易ID避免重复
3. **错误测试**: 验证错误处理机制是否正常
4. **性能监控**: 记录响应时间，监控性能变化
5. **文档同步**: 及时更新测试用例和文档

---

*最后更新: 2025-10-28*