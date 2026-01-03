# CDL (Consensus Description Language) 示例

本目录包含了各种共识算法的 CDL 描述示例。

## 文件列表

- `raft-consensus.yaml` - 类 Raft 共识算法
- `pow-consensus.yaml` - 简单的工作量证明 (PoW)
- `pbft-consensus.yaml` - 类 PBFT (Byzantine Fault Tolerance)

## CDL 结构

每个 CDL 文件包含以下主要部分:

### 1. 基本信息
```yaml
consensus:
  name: my-consensus        # 共识名称
  version: 1.0.0           # 版本号 (语义版本)
  type: custom             # 类型: custom, pow, pos, pbft 等
```

### 2. 组件配置
```yaml
components:
  crypto:
    hash: SHA256           # 哈希算法
    signature: ECDSA       # 签名算法
  network:
    topology: gossip       # 网络拓扑
    broadcast: reliable    # 广播方式
  storage:
    blockchain: merkle-chain  # 区块链结构
    state: simple-map         # 状态存储
```

### 3. 参数配置
```yaml
parameters:
  block_time: 2s           # 出块时间
  max_block_size: 2097152  # 区块大小
  # ... 其他自定义参数
```

### 4. 阶段定义
```yaml
phases:
  - name: propose          # 阶段名称
    entry: start           # 入口点
    exit: proposed         # 出口点
    actions:               # 动作列表
      - type: function
        name: create_proposal
```

### 5. 状态机
```yaml
state_machine:
  states:                  # 状态列表
    - idle
    - active
  transitions:             # 转换规则
    - from: idle
      to: active
      event: start
      condition: "ready"
```

### 6. 安全属性
```yaml
safety_properties:
  - name: agreement
    formula: "all_commit_same_value"
```

### 7. 性能要求
```yaml
performance_requirements:
  min_throughput: 100      # TPS
  max_latency: 10          # 秒
  fault_tolerance: 0.33    # 容错率
```

## 支持的组件选项

### 加密组件 (crypto)
- **hash**: `SHA256`, `SHA3`, `BLAKE2b`
- **signature**: `ECDSA`, `EdDSA`, `BLS`
- **vrf**: VRF 算法 (可选)
- **vdf**: VDF 算法 (可选)
- **threshold_sig**: 门限签名 (可选)

### 网络组件 (network)
- **topology**: `gossip`, `mesh`, `star`
- **broadcast**: `reliable`, `best-effort`

### 存储组件 (storage)
- **blockchain**: `merkle-chain`, `dag`
- **state**: `merkle-patricia`, `simple-map`

### 动作类型 (actions)
- **function**: 函数调用
- **event**: 事件处理
- **condition**: 条件判断

## 使用方法

### 1. 验证 CDL

```bash
# 使用 API
curl -X POST http://localhost:8080/api/cdl/validate \
  -H "Content-Type: application/json" \
  -d '{"cdl_yaml": "..."}'
```

### 2. 编译 CDL

```bash
# 使用 API
curl -X POST http://localhost:8080/api/cdl/compile \
  -H "Content-Type: application/json" \
  -d '{"cdl_yaml": "..."}'
```

### 3. 创建升级提案

```bash
# 读取 CDL 文件
CDL_CONTENT=$(cat examples/cdl/raft-consensus.yaml)

# 创建提案
curl -X POST http://localhost:8080/api/upgrade/propose \
  -H "Content-Type: application/json" \
  -d "{
    \"target_consensus\": \"custom\",
    \"cdl_yaml\": \"$CDL_CONTENT\",
    \"preexec_start_height\": 1000,
    \"switch_height\": 2000,
    \"description\": \"Upgrade to Raft consensus\"
  }"
```

### 4. 程序化使用

```go
package main

import (
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade/cdl"
    "github.com/sirupsen/logrus"
)

func main() {
    log := logrus.NewEntry(logrus.New())
    
    // 1. 解析 CDL
    parser := cdl.NewParser(log)
    descriptor, err := parser.ParseFile("examples/cdl/raft-consensus.yaml")
    if err != nil {
        panic(err)
    }
    
    // 2. 验证 CDL
    validator := cdl.NewValidator(log)
    if err := validator.Validate(descriptor); err != nil {
        panic(err)
    }
    
    // 3. 验证语义
    if err := validator.ValidateSemantics(descriptor); err != nil {
        panic(err)
    }
    
    // 4. 编译 CDL
    compiler := cdl.NewCompiler(log)
    runtime, err := compiler.Compile(descriptor, consensusID, cfg, p2pAdaptor)
    if err != nil {
        panic(err)
    }
    
    // 5. 运行共识
    runtime.Run()
    defer runtime.Stop()
}
```

## 最佳实践

### 1. 命名规范
- 使用小写字母和连字符: `my-consensus`
- 版本使用语义版本: `major.minor.patch`
- 状态名称清晰: `idle`, `proposing`, `voting`, `committed`

### 2. 状态机设计
- 保持状态数量合理 (3-10 个)
- 确保所有状态可达
- 提供回退路径 (错误处理)
- 避免循环依赖

### 3. 参数设置
- `block_time`: 根据网络延迟设置 (1-10s)
- `max_block_size`: 考虑网络带宽 (1-10MB)
- `fault_tolerance`: 通常 0.33 (BFT) 或 0.5 (CFT)

### 4. 安全属性
- 至少定义基本的安全属性
- 使用清晰的形式化描述
- 考虑拜占庭故障场景

### 5. 性能要求
- 设置现实的吞吐量目标
- 考虑网络条件和节点性能
- 容错率与共识类型匹配

## 调试技巧

### 验证 YAML 格式
```bash
# 使用 yq 验证
yq eval . raft-consensus.yaml

# 使用 Python
python3 -c "import yaml; yaml.safe_load(open('raft-consensus.yaml'))"
```

### 检查状态机
- 绘制状态转换图
- 验证所有状态可达
- 检查是否有死锁

### 常见错误

1. **名称格式错误**
   ```
   ✗ consensus name must contain only letters, numbers, underscores, and hyphens
   ```
   解决: 使用 `my-consensus` 而不是 `my consensus`

2. **版本格式错误**
   ```
   ✗ version must follow semantic versioning
   ```
   解决: 使用 `1.0.0` 而不是 `v1.0`

3. **哈希算法不支持**
   ```
   ✗ unsupported hash algorithm: MD5
   ```
   解决: 使用 `SHA256`, `SHA3`, 或 `BLAKE2b`

4. **状态不存在**
   ```
   ✗ unknown source state: unknow_state
   ```
   解决: 确保转换中的状态在 states 列表中定义

## 示例对比

| 特性 | Raft | PoW | PBFT |
|------|------|-----|------|
| 类型 | CFT | PoW | BFT |
| 容错率 | 50% | 49% | 33% |
| 吞吐量 | 中 | 低 | 高 |
| 延迟 | 低 | 高 | 中 |
| 能耗 | 低 | 高 | 低 |
| 适用场景 | 联盟链 | 公链 | 联盟链 |

## 参考资源

- [CDL 实现文档](../../CDL_IMPLEMENTATION.md)
- [共识升级协议](../../docs/upgrade_plan/upgrade-protocol-impl.md)
- [API 文档](../../web/API.md)

## 贡献

欢迎贡献新的 CDL 示例!请确保:
1. 通过 CDL 验证
2. 包含完整的文档注释
3. 遵循命名规范
4. 提供使用示例
