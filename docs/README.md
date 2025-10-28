# PoT 区块链文档

欢迎来到 PoT (Proof of Time) 区块链项目文档中心。

## 📚 文档导航

### API 文档
- **[HTTP API 完整文档](./http-api-documentation.md)** - 详细的 API 接口文档
- **[API 简化文档](./api-documentation.md)** - 快速参考指南
- **[API 测试指南](./api-testing-guide.md)** - 测试使用指南

### 测试文件
- **[Rest Client 测试文件](./rest-client-tests.http)** - VS Code Rest Client 测试用例
- **[Thunder Client 配置](./thunder-client-collection.json)** - Thunder Client 集合文件
- **[快速测试示例](./quick-test-example.md)** - 简单测试示例

## 🚀 快速开始

1. **启动服务器**
   ```bash
   make run_server
   ```

2. **运行测试**
   - 在 VS Code 中安装 Rest Client 扩展
   - 打开 `rest-client-tests.http` 文件
   - 点击 "Send Request" 执行测试

3. **查看 API 文档**
   - 阅读 [HTTP API 完整文档](./http-api-documentation.md)
   - 参考 [API 测试指南](./api-testing-guide.md)

## 📋 API 接口概览

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 健康检查 | POST | `/api/hello` | 检查服务状态 |
| 获取区块高度 | GET | `/api/getblockheight` | 查询当前区块高度 |
| 创建锁定交易 | POST | `/api/createlocktransaction` | 创建锁定交易 |
| 锁定转账交易 | POST | `/api/locktransfertransaction` | 锁定资产转账 |
| 非锁定转账交易 | POST | `/api/nonlocktransfertransaction` | 普通转账 |
| 销毁交易 | POST | `/api/devastatetransaction` | 销毁资产 |

## 🔧 开发工具

### VS Code 扩展
- **Rest Client** - HTTP 请求测试
- **Thunder Client** - API 测试工具

### 测试工具
- **curl** - 命令行 HTTP 客户端
- **Postman** - 图形化 API 测试工具

## 📖 文档说明

每个文档都有其特定用途：

- **完整 API 文档**: 包含所有接口的详细说明、参数格式、响应示例
- **测试文件**: 包含各种测试用例，可直接在 VS Code 中执行
- **测试指南**: 详细的测试流程和最佳实践

## 🤝 贡献

如果您发现文档有误或需要改进，请：

1. 创建 Issue 报告问题
2. 提交 Pull Request 改进文档
3. 参与讨论和反馈

---

*文档版本: v1.0*  
*最后更新: 2025-10-28*