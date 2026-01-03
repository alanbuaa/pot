# Phase 6: API 和状态持久化

## 概述

Phase 6 在 Phase 1-5 完整的共识升级功能基础上，提供外部交互接口和状态持久化能力，使升级系统可以：
- 通过 HTTP API 进行远程管理和监控
- 持久化升级状态，支持节点重启恢复
- 提供 CLI 工具方便运维操作

## 设计目标

### 核心目标
1. **API 接口**: 提供完整的 RESTful API 用于升级管理
2. **状态持久化**: 升级状态和配置可持久化存储
3. **CLI 工具**: 命令行工具简化常用操作
4. **监控能力**: 实时查询升级进度和指标

### 设计原则
- **RESTful 规范**: 遵循 REST 最佳实践
- **向后兼容**: API 版本控制，支持平滑升级
- **安全优先**: API 认证授权机制
- **幂等性**: 关键操作支持幂等

## 功能清单

### 1. HTTP API (api.go)

#### 升级管理 API
- `POST /api/v1/upgrade/propose` - 提交升级提案
- `POST /api/v1/upgrade/start` - 启动升级
- `POST /api/v1/upgrade/rollback` - 回退升级
- `GET /api/v1/upgrade/status` - 获取升级状态
- `GET /api/v1/upgrade/proposals` - 查询提案列表
- `GET /api/v1/upgrade/proposals/:id` - 获取提案详情

#### CDL 管理 API
- `POST /api/v1/cdl/validate` - 验证 CDL 描述符
- `POST /api/v1/cdl/compile` - 编译 CDL
- `GET /api/v1/cdl/templates` - 获取 CDL 模板

#### 监控 API
- `GET /api/v1/metrics/current` - 当前性能指标
- `GET /api/v1/metrics/history` - 历史指标
- `GET /api/v1/health` - 健康检查
- `GET /api/v1/phase` - 当前升级阶段

#### 治理 API
- `GET /api/v1/governance/committee` - 治理委员会信息
- `POST /api/v1/governance/sign` - 委员签名
- `GET /api/v1/governance/signatures/:proposalId` - 获取签名

### 2. 状态持久化 (persistence.go)

#### 存储接口
```go
type UpgradePersistence interface {
    // 保存升级状态
    SaveState(state *UpgradeState) error
    
    // 加载升级状态
    LoadState() (*UpgradeState, error)
    
    // 保存提案
    SaveProposal(proposal *UpgradeProposal) error
    
    // 加载提案
    LoadProposal(proposalID types.TxHash) (*UpgradeProposal, error)
    
    // 列出所有提案
    ListProposals() ([]*UpgradeProposal, error)
    
    // 保存切换/回退记录
    SaveEvent(event *UpgradeEvent) error
    
    // 查询事件历史
    QueryEvents(filter EventFilter) ([]*UpgradeEvent, error)
}
```

#### 存储实现
- **BoltDB 实现**: 嵌入式 KV 存储，轻量高效
- **存储目录**: `data/upgrade/`
  - `state.db` - 当前状态
  - `proposals.db` - 提案历史
  - `events.db` - 事件日志

#### 持久化内容
- 升级状态（Phase, Started, Completed 等）
- 提案配置（CDL, 高度参数等）
- 监控指标历史数据
- 切换/回退事件记录

### 3. CLI 工具 (cmd/upgrade-cli/)

#### 命令结构
```bash
upgrade-cli [global options] command [command options] [arguments...]

COMMANDS:
   propose      提交升级提案
   start        启动升级
   rollback     回退升级
   status       查询升级状态
   list         列出所有提案
   metrics      查看性能指标
   validate     验证 CDL 文件
   help, h      显示帮助信息

GLOBAL OPTIONS:
   --endpoint value  API 端点地址 (default: "http://localhost:8080")
   --config value    配置文件路径
   --verbose         详细输出
   --help, -h        显示帮助信息
   --version, -v     显示版本信息
```

#### 使用示例
```bash
# 验证 CDL
upgrade-cli validate --file consensus.yaml

# 提交提案
upgrade-cli propose \
  --cdl consensus.yaml \
  --fork-height 1000 \
  --switch-height 2000 \
  --preexec-start 1500

# 启动升级
upgrade-cli start --proposal-id 0x123abc...

# 查看状态
upgrade-cli status

# 回退
upgrade-cli rollback --reason "performance issue"

# 查看指标
upgrade-cli metrics --interval 5s
```

## 技术实现

### API Server 架构

```
┌─────────────────────────────────────────┐
│         HTTP API Server                 │
│  (基于 gin-gonic/gin 或 net/http)      │
└────────────┬────────────────────────────┘
             │
        ┌────┴────┐
        │ Router  │
        └────┬────┘
             │
    ┌────────┴────────┐
    │   Middleware    │
    │  - Auth         │
    │  - Logging      │
    │  - CORS         │
    │  - RateLimit    │
    └────────┬────────┘
             │
    ┌────────┴────────────────────────┐
    │      API Handlers               │
    │  - UpgradeHandler               │
    │  - CDLHandler                   │
    │  - MetricsHandler               │
    │  - GovernanceHandler            │
    └────────┬────────────────────────┘
             │
    ┌────────┴────────────────────────┐
    │    UpgradeManager               │
    │  + Persistence Layer            │
    └─────────────────────────────────┘
```

### 持久化层设计

```
┌──────────────────────────────────────┐
│      UpgradeManager                  │
└────────────┬─────────────────────────┘
             │
    ┌────────┴──────────┐
    │  Persistence      │
    │  Interface        │
    └────────┬──────────┘
             │
    ┌────────┴──────────┐
    │  BoltDBPersistence│
    └────────┬──────────┘
             │
    ┌────────┴──────────────────────┐
    │  BoltDB Database Files        │
    │  - state.db    (升级状态)     │
    │  - proposals.db (提案)        │
    │  - events.db   (事件日志)     │
    └───────────────────────────────┘
```

### 目录结构

```
consensus/upgrade/
├── api.go                 # HTTP API 实现
├── api_test.go            # API 测试
├── persistence.go         # 持久化接口定义
├── boltdb_persistence.go  # BoltDB 实现
├── persistence_test.go    # 持久化测试
├── PHASE6_README.md       # 本文档

cmd/
├── upgrade-cli/           # CLI 工具
│   ├── main.go           # 入口
│   ├── commands.go       # 命令实现
│   ├── client.go         # API 客户端
│   └── README.md         # CLI 文档

internal/apis/
└── upgrade/              # API 路由和处理器
    ├── router.go
    ├── handlers.go
    └── middleware.go

data/upgrade/             # 持久化数据目录
├── state.db
├── proposals.db
└── events.db
```

## API 规范

### 请求/响应格式

#### 通用响应格式
```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "timestamp": 1704268800
}
```

#### 错误响应
```json
{
  "code": 400,
  "message": "invalid parameter: switch_height must be greater than fork_height",
  "error": "INVALID_PARAMETER",
  "timestamp": 1704268800
}
```

### API 详细定义

#### 1. 提交升级提案
```
POST /api/v1/upgrade/propose
Content-Type: application/json

Request:
{
  "target_consensus": "HotStuff",
  "cdl_yaml": "...",
  "fork_height": 1000,
  "preexec_start_height": 1500,
  "switch_height": 2000,
  "rollback_condition": {
    "metric": "throughput",
    "operator": "<",
    "threshold": 0.8,
    "window_size": 100
  },
  "incentive": 100000,
  "description": "Upgrade to HotStuff for better performance"
}

Response:
{
  "code": 0,
  "message": "proposal created",
  "data": {
    "proposal_id": "0x123abc...",
    "status": "pending_signatures",
    "created_at": 1704268800
  }
}
```

#### 2. 启动升级
```
POST /api/v1/upgrade/start
Content-Type: application/json

Request:
{
  "proposal_id": "0x123abc..."
}

Response:
{
  "code": 0,
  "message": "upgrade started",
  "data": {
    "phase": "Preparing",
    "started_at": 1704268800
  }
}
```

#### 3. 获取升级状态
```
GET /api/v1/upgrade/status

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "phase": "Preexecuting",
    "started": true,
    "completed": false,
    "failed": false,
    "start_time": 1704268800,
    "current_proposal": {
      "proposal_id": "0x123abc...",
      "target_consensus": "HotStuff",
      "switch_height": 2000
    },
    "current_height": 1750,
    "metrics": {
      "throughput": 1200,
      "latency": 0.5,
      "error_rate": 0.001
    }
  }
}
```

## 安全设计

### 认证机制
- **API Key**: 简单的 API Key 认证
- **JWT**: 支持 JWT token 认证
- **TLS**: 生产环境强制 HTTPS

### 授权控制
- **角色**: admin, operator, viewer
- **权限**:
  - admin: 所有操作
  - operator: 启动/回退升级
  - viewer: 只读查询

### 限流保护
- 基于 IP 的请求限流
- 基于用户的操作限流
- 关键操作（propose, start, rollback）额外限流

## 监控和日志

### 日志规范
- **结构化日志**: 使用 logrus 或 zap
- **日志级别**: DEBUG, INFO, WARN, ERROR
- **关键操作审计**: 提案提交、升级启动、回退等

### 指标暴露
- Prometheus 格式 `/metrics` 端点
- 关键指标:
  - `upgrade_phase` - 当前阶段
  - `upgrade_proposals_total` - 提案总数
  - `upgrade_switches_total` - 切换总数
  - `upgrade_rollbacks_total` - 回退总数
  - `api_requests_total` - API 请求数
  - `api_request_duration_seconds` - API 延迟

## 测试计划

### 单元测试
- API handlers 测试
- 持久化层测试
- CLI 命令测试

### 集成测试
- 端到端 API 调用测试
- 持久化恢复测试
- 并发访问测试

### 性能测试
- API 并发压测
- 持久化性能测试
- 大量提案查询性能

## 部署指南

### 启动 API Server
```bash
# 构建
make build-api-server

# 运行
./bin/upgrade-api-server \
  --port 8080 \
  --db-path ./data/upgrade \
  --auth-enabled \
  --api-key your-secret-key
```

### 配置文件
```yaml
# config/upgrade-api.yaml
server:
  port: 8080
  tls:
    enabled: true
    cert: /path/to/cert.pem
    key: /path/to/key.pem

database:
  path: ./data/upgrade
  max_size_mb: 1024

auth:
  enabled: true
  method: jwt  # jwt or api_key
  jwt_secret: your-jwt-secret
  api_keys:
    - admin-key-1
    - operator-key-2

rate_limit:
  requests_per_minute: 60
  burst: 10

logging:
  level: info
  format: json
  output: stdout
```

## 开发计划

### 第一步：核心接口定义（今天）
- [ ] 定义 API 接口规范
- [ ] 定义 Persistence 接口
- [ ] 创建基础文件结构

### 第二步：持久化实现
- [ ] 实现 BoltDB persistence
- [ ] 编写持久化测试
- [ ] 集成到 UpgradeManager

### 第三步：HTTP API 实现
- [ ] 实现 API handlers
- [ ] 添加中间件（auth, logging, cors）
- [ ] 编写 API 测试

### 第四步：CLI 工具
- [ ] 实现 CLI 命令
- [ ] 实现 API 客户端
- [ ] 编写使用文档

### 第五步：集成测试
- [ ] 端到端测试
- [ ] 性能测试
- [ ] 文档完善

## 预期成果

完成 Phase 6 后，升级系统将具备：

✅ **完整的 API 接口** - 支持远程管理和监控
✅ **状态持久化** - 节点重启不丢失升级状态
✅ **CLI 工具** - 方便运维操作
✅ **生产就绪** - 安全、可靠、可观测

这将使共识升级系统从原型走向生产可用！
