# Upgrade CLI Tool

命令行工具,用于管理和监控共识升级系统。

## 构建

```bash
make build  # 构建所有二进制文件,包括 upgrade-cli
# 或
go build -o bin/upgrade-cli ./cmd/upgrade-cli
```

## 使用

### 基本命令

#### 1. 检查 API 健康状态

```bash
./bin/upgrade-cli health
```

输出示例:
```json
{
  "status": "healthy",
  "message": "Upgrade API is running"
}
```

#### 2. 获取升级状态

```bash
./bin/upgrade-cli status
```

输出示例:
```json
{
  "currentPhase": 1,
  "status": "running",
  "metadata": {
    "lastUpdate": "2024-01-01T00:00:00Z"
  }
}
```

### 选项

- `-e, --endpoint <url>`: API 端点地址(默认: `http://localhost:8080`)
- `-o, --output <format>`: 输出格式(当前仅支持 json)
- `-h, --help`: 显示帮助信息

### 使用示例

```bash
# 使用自定义 API 端点
./bin/upgrade-cli -e http://192.168.1.100:8080 status

# 检查远程服务器健康状态
./bin/upgrade-cli --endpoint http://production-server:8080 health
```

## 功能说明

### 当前支持的命令

1. **health** - 检查升级 API 服务健康状态
   - 返回服务运行状态
   - 用于监控和健康检查

2. **status** - 获取当前升级状态
   - 当前阶段编号
   - 升级状态(running, completed, failed 等)
   - 元数据信息

### 未来扩展

计划支持的命令:
- `propose` - 提交升级提案
- `list-proposals` - 列出所有提案
- `validate-cdl` - 验证 CDL 文件
- `rollback` - 触发回滚操作

## 错误处理

CLI 工具会返回适当的退出码:
- `0` - 成功
- `1` - 失败(带错误信息)

错误信息输出到 stderr:
```bash
./bin/upgrade-cli status
Error: failed to get status: connection refused
```

## 集成示例

### 在脚本中使用

```bash
#!/bin/bash

# 检查服务健康状态
if ./bin/upgrade-cli health >/dev/null 2>&1; then
    echo "Service is healthy"
else
    echo "Service is down!"
    exit 1
fi

# 获取当前阶段
PHASE=$(./bin/upgrade-cli status | jq -r '.currentPhase')
echo "Current phase: $PHASE"
```

### 监控集成

```bash
# 每 10 秒检查一次状态
watch -n 10 './bin/upgrade-cli status'
```

## 依赖

- Go 1.22+
- github.com/spf13/cobra v1.10.2
- 运行中的升级 API 服务器

## 架构

CLI 工具直接调用 HTTP API 端点:
- `/api/upgrade/health` - 健康检查
- `/api/upgrade/status` - 状态查询

所有响应均为 JSON 格式,便于解析和处理。
