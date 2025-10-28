# PoT Blockchain HTTP API 文档

这是 PoT (Proof of Time) 区块链 HTTP API 的简化文档。

## 快速开始

- **完整文档**: [http-api-documentation.md](./http-api-documentation.md)
- **测试文件**: [rest-client-tests.http](./rest-client-tests.http)
- **服务端口**: `18025`
- **基础路径**: `/api`

## 主要接口

1. **健康检查**: `POST /api/hello`
2. **获取区块高度**: `GET /api/getblockheight`
3. **创建锁定交易**: `POST /api/createlocktransaction`
4. **锁定转账交易**: `POST /api/locktransfertransaction`
5. **非锁定转账交易**: `POST /api/nonlocktransfertransaction`
6. **销毁交易**: `POST /api/devastatetransaction`

请查看完整文档获取详细信息。