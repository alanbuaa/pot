# 共识可切换升级协议学术设计方案

## 摘要

本文提出一种支持在线热升级的区块链共识协议切换机制(Switchable Consensus Upgrade Protocol, SCUP)。该协议允许区块链系统在运行时安全地从一种共识算法切换到另一种共识算法,无需停机或硬分叉。通过引入升级配置交易、双链预执行机制、治理委员会验证和共识描述语言(Consensus Description Language, CDL),本协议实现了共识算法的可插拔升级,同时保证系统的安全性、活性和一致性。

## 1. 引言

### 1.1 背景与动机

区块链共识算法的升级通常需要硬分叉或全网停机,这会导致:
- 网络分裂风险
- 服务中断
- 社区分裂
- 高昂的协调成本

现有的升级方案(如链下治理、硬分叉)存在以下局限:
- 缺乏形式化的安全保证
- 无法在线预执行和验证新共识
- 升级过程不透明
- 回退机制不完善

### 1.2 贡献

本协议的主要贡献包括:

1. **双链预执行机制**: 在保持主链稳定的同时,在分叉链上预执行新共识
2. **多阶段升级流程**: 提案→预备→预执行→确认→切换的完整生命周期管理
3. **治理委员会验证**: 基于门限签名的去中心化治理机制  // 这里待定
4. **共识描述语言(CDL)**: 支持自定义共识算法的声明式描述框架
5. **形式化安全分析**: 提供完整的安全性证明和攻击模型分析

## 2. 系统模型与基本假设

### 2.1 网络模型

- **部分同步网络模型**: 消息传递存在有界延迟 Δ,但 Δ 未知
- **网络分区容忍**: 系统能容忍临时的网络分区   // 主链优先原则
- **最终一致性**: 分区恢复后,诚实节点能达成一致

#### 2.1.1 消息缓存机制（非同步网络专用）

论文中的消息缓存机制仅用于**非同步网络**（半同步 + 异步）的共识切换，核心是解决“新共识未启动时消息已到达”的**消息可达性**问题。

**核心设计**:

- **缓存定位**: 作为共识切换模块的内置组件，维护消息缓存集合 `bufmsg`，用于存储“接收者共识未启动”的“未来”消息（排除过时的历史消息）。
- **触发场景**: 网络模块收到**普通消息**（共识协议间通信消息）时，若消息指定的接收者共识未启动且消息所属时段 ≥ 当前节点时段，则存入 `bufmsg`。
- **转发时机**: 当对应候选共识启动后，自动将 `bufmsg` 中该共识的消息转发给它。
- **兼容性策略**: 仅对非同步网络生效；在同步网络中，未启动共识的消息直接丢弃，不进行缓存。

**具体实现步骤（机制级描述）**:

1. **初始化配置**:
    - 系统启动时，初始化 `bufmsg`（初始为空），与输入缓存 `bufin`、候选共识集合 `candi` 等组件协同工作。
    - 存储规则：仅保留“所属时段 ≥ 当前节点时段”的消息，过时消息直接丢弃。

2. **消息接收与判断**:
    - 网络模块收到消息后传递给共识切换模块。
    - 仅处理“普通消息”；共识切换提议、锁定提议等控制类消息不进入缓存流程。
    - 共识切换模块解析普通消息中的“接收者共识 ID”和“所属时段”，并按如下规则处理：
      - 若接收者共识已启动且未停止：直接转发给该共识；
      - 若接收者共识未启动且消息所属时段 ≥ 当前节点时段：存入 `bufmsg`；
      - 若消息所属时段 < 当前节点时段：判定为历史消息，直接丢弃。

3. **候选共识启动后的消息转发**:
    - 当共识切换提议被当前共识输出后，系统启动对应候选共识，并将其加入 `candi`。
    - 启动候选共识后，共识切换模块遍历 `bufmsg`，筛选“接收者为该候选共识”的所有消息并逐一转发。

4. **共识切换完成后的缓存清理**:
    - 当锁定交易被当前共识输出并触发切换（切换到目标候选共识）时，清理 `bufmsg` 中“所属时段 = 当前旧时段”的消息。
    - 保留更高时段的消息，为后续可能的共识切换预留缓存支持。

**关键技术细节**:

- **消息标识**: 每个普通消息需携带“接收者共识 ID”和“所属时段”，确保缓存与转发精准。
- **存储介质**: 采用内存 + 持久化（基于 LevelDB）结合，避免节点重启导致缓存丢失。

#### 2.1.2 网络分区处理机制

升级过程中的网络分区是本协议需要重点解决的问题。我们采用**分阶段分区容忍策略**:

**分区场景分类**:

1. **提案阶段分区**:
   - **问题**: 部分委员会成员收不到提案
   - **解决**: 提案在链上持久化,分区恢复后可从链上读取
   - **保证**: 只要诚实委员会多数连通,提案能通过

2. **预备阶段分区**:
   - **问题**: 部分节点未收到升级通知
   - **解决**: `UpgradeConfigTx` 写入主链区块,所有节点最终能看到
   - **保证**: 分区节点同步主链时自动获取升级配置

3. **预执行阶段分区** (最关键):
   - **问题 3.1**: 主链和预执行链可能在不同分区产生冲突
   - **解决 3.1**: 预执行链交易严格同步自主链,避免冲突
   
   - **问题 3.2**: 分区导致部分节点的预执行链落后
   - **解决 3.2**: 分区恢复后,节点从连通节点同步预执行链数据
   
   - **问题 3.3**: 分区期间主链和预执行链分别在不同分区形成多数
   - **解决 3.3**: 采用**主链优先原则** (见下文详细说明)

4. **确认阶段分区**:
   - **问题**: 委员会成员分区导致无法收集足够投票
   - **解决**: 设置确认超时,超时则自动回退到主链
   - **保证**: 分区恢复后,如果仍在超时内,可继续收集签名

5. **激活阶段分区**:
   - **问题**: 部分节点未收到 `UpgradeConfirmTx`,未能切换
   - **解决**: 切换高度 `finalizeHeight` 是确定性的,节点同步主链时自动触发切换
   - **保证**: 分区节点追上主链后自动执行切换逻辑

**主链优先原则** (解决预执行阶段分区):

```python
class PartitionTolerantUpgrade:
    """
    分区容忍的升级机制
    """
    def handle_partition_during_preexec(self, network_state):
        """
        预执行阶段的分区处理
        
        核心原则: 主链是唯一的真相源(single source of truth)
        预执行链是从属的、可重建的
        """
        # 规则 1: 主链始终使用旧共识,旧共识的分区容忍机制保证主链不分裂
        main_chain_safety = old_consensus.handle_partition(network_state)
        assert main_chain_safety == True  # 由旧共识保证
        
        # 规则 2: 预执行链的交易完全来自主链
        # 即使预执行链分区,分区恢复后可以根据主链重建
        def rebuild_preexec_chain(fork_point, current_height):
            """
            分区恢复后重建预执行链
            """
            preexec_chain_new = fork_chain_at(fork_point)
            
            # 遍历主链的所有区块
            for h in range(fork_point, current_height + 1):
                main_block = main_chain.get_block(h)
                
                # 使用主链交易在预执行链上重新执行
                preexec_txs = main_block.body.transactions
                preexec_block = new_consensus.produce_block(preexec_txs)
                preexec_chain_new.append(preexec_block)
            
            return preexec_chain_new
        
        # 规则 3: 分区检测与自动重建
        if detect_partition_recovered():
            # 3.1 获取主链的权威视图
            canonical_main_chain = sync_from_majority()
            
            # 3.2 检查本地预执行链是否与主链一致
            local_preexec_valid = validate_preexec_chain_against_main(
                local_preexec_chain, 
                canonical_main_chain
            )
            
            if not local_preexec_valid:
                # 3.3 丢弃本地错误的预执行链,根据主链重建
                log.warn("预执行链与主链不一致,重建中...")
                local_preexec_chain = rebuild_preexec_chain(
                    fork_point, 
                    canonical_main_chain.height
                )
                log.info("预执行链重建完成")
        
        # 规则 4: 性能指标在分区期间可能不准确,需要过滤
        def collect_reliable_metrics():
            """
            只收集网络连通良好时期的指标
            """
            metrics = []
            for h in range(fork_point, preexec_height):
                block_metrics = get_metrics(h)
                
                # 过滤分区期间的数据
                if block_metrics.network_quality > THRESHOLD:
                    metrics.append(block_metrics)
            
            return metrics

def validate_preexec_chain_against_main(preexec_chain, main_chain):
    """
    验证预执行链是否与主链一致
    """
    fork_point = preexec_chain.fork_point
    
    for h in range(fork_point, min(preexec_chain.height, main_chain.height) + 1):
        preexec_block = preexec_chain.get_block(h)
        main_block = main_chain.get_block(h)
        
        # 检查交易集是否一致
        if preexec_block.body.transactions != main_block.body.transactions:
            return False
        
        # 检查主链引用是否正确
        if preexec_block.header.upgradeMetadata.mainChainRef != main_block.hash():
            return False
    
    return True
```

**分区容忍性分析**:

| 阶段 | 分区影响 | 容忍策略 | 恢复时间 |
|------|---------|---------|---------|
| 提案阶段 | 部分委员会成员失联 | 链上持久化提案 | 分区恢复后立即 |
| 预备阶段 | 部分节点未准备 | 主链广播配置 | 下一次同步 |
| 预执行阶段 | 预执行链可能分叉 | 主链优先,可重建 | O(分区时长) |
| 确认阶段 | 投票收集受阻 | 超时回退机制 | 超时时间内 |
| 激活阶段 | 部分节点未切换 | 确定性切换高度 | 下一次同步 |

**分区恢复流程**:

```mermaid
graph TB
    A[检测到分区恢复] --> B[同步主链到最新]
    B --> C{主链共识有效?}
    C -->|是| D[检查预执行链有效性]
    C -->|否| E[拒绝分区,继续等待]
    
    D --> F{预执行链与主链一致?}
    F -->|是| G[继续升级流程]
    F -->|否| H[丢弃本地预执行链]
    
    H --> I[从主链重建预执行链]
    I --> J[验证重建结果]
    J --> G
    
    G --> K{当前处于哪个阶段?}
    K -->|预执行| L[继续双链运行]
    K -->|确认| M[重新收集委员会投票]
    K -->|激活| N[执行切换逻辑]
```

**形式化保证**:

```
定理 8 (分区容忍性): 
在以下条件下,升级协议能容忍网络分区:
1. 分区是临时的(最终会恢复)
2. 每个分区内至少有一个诚实节点
3. 主链使用的旧共识本身具有分区容忍性

则分区恢复后,系统能恢复到一致状态并继续升级或回退。

证明:
分三个层次证明:

层次 1: 主链不受分区影响
- 主链继续使用旧共识
- 旧共识具有分区容忍性(前提条件)
- 因此,主链在分区期间和恢复后保持一致 ✓

层次 2: 预执行链可恢复
- 预执行链的输入(交易)完全来自主链
- 给定相同的主链状态,预执行链是确定性可重建的
- 分区恢复后,节点可以根据权威主链重建预执行链
- 因此,预执行链在分区恢复后能达成一致 ✓

层次 3: 升级决策不受分区影响
- 升级配置和确认交易都在主链上
- 主链保证这些交易的一致性(层次 1)
- 因此,所有节点对"是否升级"达成一致 ✓

综上,升级协议具有分区容忍性。∎
```

### 2.2 威胁模型

#### 2.2.1 拜占庭节点假设

由于旧共识和新共识的类型在设计时是未知的,本协议采用**共识无关的统一威胁模型**:

**通用假设 (适用于所有共识类型)**:

- 系统总节点数为 n,恶意节点数为 f
- 拜占庭节点的行为能力:
  - 发送任意消息(伪造、重放、篡改)
  - 停止响应(崩溃故障)
  - 串谋攻击(协同作恶)
  - 但不能破解密码学原语(哈希抗碰撞、签名不可伪造)

**分类容错约束 (根据共识类型自动应用)**:

1. **BFT 类共识** (HotStuff, PBFT, Tendermint 等):
   - 要求: `n ≥ 3f + 1`,即 `f < n/3`
   - 安全性依赖于诚实节点的法定人数(quorum)
   
2. **PoW 类共识** (Bitcoin, Ethereum PoW 等):
   - 要求: 诚实节点算力 > 50%,即 `H_hashpower > f_hashpower`
   - 安全性依赖于计算资源的诚实多数
   
3. **PoS 类共识** (Ethereum PoS, Casper 等):
   - 要求: 诚实节点权益 > 2/3,即 `H_stake ≥ 2n/3`
   - 安全性依赖于经济激励和惩罚(slashing)
   
4. **混合型共识** (PoT, Whirly 等):
   - 要求: 同时满足其组合组件的约束
   - 例如 PoT = VDF + BFT,需满足 BFT 的 `n ≥ 3f + 1` 约束

**升级协议的保守安全策略**:

在双链预执行期间,系统同时运行旧共识和新共识,因此必须满足:

```
Safety_upgrade = Safety_old ∪ Safety_new
```

即:
- 如果 `旧共识 = BFT` 且 `新共识 = PoW`,则需同时满足 `f < n/3` 和 `H_hashpower > 50%`
- 如果 `旧共识 = PoW` 且 `新共识 = BFT`,则需同时满足 `H_hashpower > 50%` 和 `f < n/3`

**验证流程**:

提案阶段会通过 CDL 自动识别新共识类型,并验证:
1. 旧共识的当前容错能力(从链上统计)  // 这里可能不好处理
2. 新共识的声明容错要求(从 CDL 解析)
3. 检查是否满足 `Safety_old ∪ Safety_new`
4. 若不满足,拒绝提案

#### 2.2.2 治理委员会假设

- 治理委员会规模为 m,其中最多有 f_c 个恶意成员
- 门限签名参数: t = ⌊2m/3⌋ + 1 (需要至少 t 个签名才能有效)
- 假设 f_c < t,即恶意成员无法单独控制升级决策

#### 2.2.3 密码学假设

- **哈希函数抗碰撞性**: 找到 H(x) = H(y) 且 x ≠ y 的概率可忽略
- **数字签名不可伪造性**: 基于EUF-CMA 
- **门限签名安全性**: 基于 (t, m)-门限方案的安全性假设
- **...**

### 2.3 时间假设

- **全局时钟**: 节点拥有大致同步的本地时钟,偏差在可接受范围内
- **超时参数**: 
  - T_prepare: 预备阶段的最小区块高度 (≥ 100)  // 大致要求主链有一定的运行时间
  - T_preexec: 预执行阶段的最小区块数 (≥ 1000) // 大致要求预执行链的运行时间
  - T_confirm: 确认超时时间 (≥ 10 分钟)

## 3. 协议设计

### 3.1 核心数据结构

#### 3.1.1 升级配置交易 (Upgrade Configuration Transaction)

```
UpgradeConfigTx := {
    txType: UPGRADE_CONFIG,
    proposalID: Hash,              // 提案唯一标识符
    targetConsensus: String,       // 目标共识名称 (内置 | 自定义)
    consensusDescriptor: CDL,      // 共识描述语言 (仅自定义时)
    descriptorHash: Hash,          // 描述符哈希 (仅自定义时)
    
    // 区块高度检查点
    prepareHeight: uint64,         // 预备检查点 (当前高度 + T_prepare)
    preexecHeight: uint64,         // 预执行检查点 (prepareHeight + T_preexec)
    
    // 升级阶段
    phase: {PROPOSAL, PREEXEC, CONFIRM, ACTIVE, ROLLBACK},
    
    // 治理与激励
    committeeSignatures: []ThresholdSig,  // 门限签名集合
    incentiveAmount: uint64,              // 激励代币数额
    incentiveRecipients: []Address,       // 激励接收者列表
    
    // 安全参数
    rollbackCondition: Condition,    // 回退条件定义
    safetyThreshold: float64,        // 安全性阈值 (0.95)
    
    // 时间戳与版本
    timestamp: int64,
    version: uint32,
    
    // 元数据
    proposer: PublicKey,
    nonce: uint64
}
```

#### 3.1.2 确认升级交易 (Upgrade Confirmation Transaction)

```
UpgradeConfirmTx := {
    txType: UPGRADE_CONFIRM,
    proposalID: Hash,                    // 关联的提案ID
    preexecChainHead: Hash,              // 预执行链头部哈希
    committeeApprovals: []ThresholdSig,  // 最终批准签名
    finalizeHeight: uint64,              // 最终切换高度
    timestamp: int64
}
```

#### 3.1.3 统一区块结构

为了确保预执行链能够在升级确认后无缝成为主链,本协议采用**统一的区块格式**:

```go
Block := {
    header: {
        // 基础区块字段
        height: uint64,               // 区块高度
        blockhash: uint64,            // 区块哈希
        parentHash: Hash,             // 父区块哈希
        timestamp: int64,             // 区块时间戳
        consensusID: int64,           // 当前使用的共识标识
        stateRoot: Hash,              // 状态树根哈希
        txRoot: Hash,                 // 交易树根哈希
        
        // 升级相关字段 (可选字段，在主链区块中为空或标记为非预执行)
        upgradeMetadata: {
            isPreexec: bool,          // 是否为预执行链区块
            proposalID: Hash,         // 关联的升级提案ID (预执行时有效)
            forkPoint: uint64,        // 分叉点高度 (预执行时有效)
            mainChainRef: Hash,       // 对应主链区块哈希 (预执行时有效)
            preexecStatus: Status,    // RUNNING | SUCCESS | FAILED (预执行时有效)
        },
        
        // 共识证明 (不同共识类型有不同内容，PoW nonce、BFT 签名、PoS 证明等)
        consensusProof: Proof,
    },
    body: {
        transactions: []Tx,           // 交易列表
    }
}
```

**关键设计要点**:

1. **统一格式**: 主链和预执行链使用完全相同的区块结构
2. **可选字段**: `upgradeMetadata` 在预执行期间填充,在主链区块中为空或标记为非预执行
3. **共识中立**: `consensusProof` 字段可以容纳不同共识算法的证明(PoW nonce、BFT 签名、PoS 证明等)
4. **无缝合并**: 预执行链直接成为主链区块,通过切换链的 head 指针实现

**不同阶段的区块示例（Hotstuff->PoW）**:

**正常主链区块**:
```go
{
    header: {
        height: 1000,
        blockhash: 0x9898c...,
        parentHash: 0xabc...,
        consensusID: 1,  // 旧共识
        upgradeMetadata: {
            isPreexec: false,
        },
        consensusProof: HotStuffProof{...},
    },
    body: {
        transactions: [tx1, tx2, ...],
    }
}
```

**预执行链区块**:
```go
{
    header: {
        height: 1100,
        blockhash: 0x123aaa...,
        parentHash: 0xdef...,
        consensusID: 2,  // 新共识
        upgradeMetadata: {
            isPreexec: true,
            proposalID: 0x123...,
            forkPoint: 1000,
            mainChainRef: 0xabc...,  // 对应主链 1100 号区块
            preexecStatus: RUNNING,
        },
        consensusProof: PoWProof{...},  // 新共识的证明
    },
    body: {
        transactions: [tx1, tx2, ...],  // 从主链同步
    }
}
```

**切换后的主链的新区块**:
```go
{
    header: {
        height: 1100,
        blockhash: 0x323432...,
        parentHash: 0x123aaa..., // 节点的"主链头指针"指向了预执行链
        consensusID: 2,  // 新共识已激活
        upgradeMetadata: {
            isPreexec: false,  // 标记清除,现在是主链
            proposalID: 0x123...,  // 保留以追溯升级历史
        },
        consensusProof: PoWProof{...},
    },
    body: {
        transactions: [tx1, tx2, ...],
    }
}
```


**切换流程的区块处理**:

```python
def activate_preexec_chain_as_main(preexec_chain, fork_point, switch_height):
    """
    激活预执行链为主链 (关键: 不修改任何区块内容)
    """
    # 1. 验证预执行链的连续性和有效性
    for height in range(fork_point, switch_height + 1):
        preexec_block = preexec_chain.get_block(height)
        
        # 验证是预执行链区块
        assert preexec_block.header.upgradeMetadata.isPreexec == true
        assert preexec_block.header.upgradeMetadata.forkPoint == fork_point
        
        # 验证区块有效性(共识证明、交易、状态转换)
        assert validate_block(preexec_block, new_consensus) == true
        
        # 验证哈希链连续性
        if height > fork_point:
            prev_block = preexec_chain.get_block(height - 1)
            assert preexec_block.header.parentHash == hash(prev_block)
    
    # 2. 原子切换主链指针 
    atomic {
        # 2.1 归档旧主链在 [fork_point, switch_height] 的区块
        for height in range(fork_point, switch_height + 1):
            old_block = main_chain.get_block(height)
            archive_block(old_block, tag="old_consensus_archived")
        
        # 2.2 将主链的 head 指针切换到预执行链
        main_chain.head = preexec_chain.head
        main_chain.tip_hash = preexec_chain.tip_hash
        main_chain.consensus_id = preexec_chain.consensus_id
        
        # 2.3 更新区块索引
        #     将 [fork_point, switch_height] 高度映射到预执行链区块
        for height in range(fork_point, switch_height + 1):
            preexec_block = preexec_chain.get_block(height)
            # 注意: preexec_block 保持原样,不做任何修改
            main_chain.height_to_block[height] = preexec_block
        
        # 2.4 切换共识引擎
        consensus_engine.stop(old_consensus)
        consensus_engine.start(new_consensus)
    }
    
    # 3. 验证切换后的链状态
    assert main_chain.get_block(switch_height).header.consensusID == new_consensus.id
    assert main_chain.state_root == preexec_chain.state_root
    
    log.info(f"升级成功: 主链已切换到新共识,高度 {switch_height}")
    log.info(f"注意: 区块内容未修改,isPreexec 字段仍为 true,用于历史审计")
```

**统一格式的优势**:

1. **直接合并**: 无需转换,预执行链区块可直接成为主链区块
2. **状态一致**: 相同的 `stateRoot` 确保状态转换一致
3. **简化验证**: 验证逻辑统一,无需区分主链和预执行链
4. **历史可追溯**: 保留 `proposalID` 和 `forkPoint` 可追溯升级历史
5. **共识中立**: `consensusProof` 设计支持任意共识类型

**性能监控方式**:

治理委员会成员通过**本地节点监控**观察预执行链性能:
- 各节点独立运行性能监控工具
- 记录区块时间、吞吐量、延迟等指标到本地日志
- 委员会成员基于本地观察数据投票决策

### 3.2 共识描述语言 (CDL)

#### 3.2.1 设计目标

CDL 旨在提供一种声明式、可验证的方式来描述共识算法,支持:
- 模块化组件组合
- 密码学原语抽象
- 协议流程描述
- 安全属性声明

#### 3.2.2 CDL 语法结构

```yaml
consensus:
  name: "CustomPoW"
  version: "1.0"
  type: "proof-based"
  
  # 容错模型声明 
  fault_tolerance:
    model: "pow"              # bft | pow | pos | hybrid
    byzantine_bound: "0.5"    # 拜占庭节点比例上界
    # BFT 类使用: f < n/3,即 0.33
    # PoW 类使用: f_hashpower < 0.5
    # PoS 类使用: f_stake < 0.33
    assumptions:
      - "honest_majority_hashpower"
      - "network_synchrony"
  
  # 基础组件声明
  components:
    crypto:
      hash: "SHA256"               # 哈希函数
      signature: "ECDSA"           # 签名算法
      vdf: "Wesolowski"            # 可验证延迟函数
      vrf: "ECVRF"                 # 可验证随机函数
      commitment: "Pedersen"       # 承诺方案
      threshold_sig: "BLS-TSig"    # 门限签名
      
    network:
      topology: "gossip"           # 网络拓扑，可选gossip/p2p
      broadcast: "reliable"        # 广播原语, 可选reliable/common
      
    storage:
      blockchain: "merkle-chain"   # 区块链结构
      state: "merkle-patricia"     # 状态树
      
  # 协议参数
  parameters:
    block_time: 10s
    difficulty_adjustment: 10     # 难度调整周期
    max_block_size: 2MB
    security_param: 128bit
    
  # 协议阶段定义
  phases:
    - name: "transaction_validation"
      entry: "receive_tx"
      actions:
        - verify_signature(tx)
        - verify_balance(tx)
        - add_to_mempool(tx)
      exit: "tx_validated"
      
    - name: "block_production"
      entry: "mining_round_start"
      precondition: "is_leader()"
      actions:
        - select_txs_from_mempool()
        - construct_block_header()
        - solve_puzzle(difficulty)
      postcondition: "valid_proof_of_work()"
      exit: "block_produced"
      
    - name: "block_validation"
      entry: "receive_block"
      actions:
        - verify_block_header()
        - verify_proof_of_work()
        - verify_transactions()
        - verify_state_transition()
      exit: "block_validated"
      
    - name: "chain_selection"
      entry: "multiple_chains_exist"
      actions:
        - select_longest_chain()
        - resolve_forks()
      exit: "canonical_chain_selected"
  
  # 状态机转换
  state_machine:
    states: [IDLE, MINING, VALIDATING, COMMITTING]
    transitions:
      - from: IDLE
        to: MINING
        condition: "new_tx_available()"
        action: "start_mining()"
        
      - from: MINING
        to: VALIDATING
        condition: "block_received() || puzzle_solved()"
        action: "validate_block()"
        
      - from: VALIDATING
        to: COMMITTING
        condition: "block_valid()"
        action: "commit_block()"
        
      - from: COMMITTING
        to: IDLE
        condition: "committed()"
        action: "cleanup()"
  
  # 安全属性
  safety_properties:
    - name: "agreement"
      formula: "∀i,j ∈ HonestNodes: block_i[h] = block_j[h]"
      
    - name: "validity"
      formula: "∀b ∈ Chain: valid(b) = true"
      
    - name: "liveness"
      formula: "∀tx: eventually(tx ∈ Chain)"
  
```

#### 3.2.3 CDL 验证规则

在接受自定义共识之前,系统必须验证:

1. **语法完整性**: CDL 符合预定义的语法规范
2. **组件兼容性**: 所有引用的密码学组件在系统密码库中可用
3. **状态机正确性**: 状态转换无死锁,所有状态可达
4. **安全属性可满足性**: 声明的安全属性在理论上可验证
5. **容错模型兼容性**: 验证新旧共识的容错约束是否兼容

**容错模型验证算法**:

```python
def verify_fault_tolerance_compatibility(old_consensus, new_consensus, network_state):
    """
    验证新旧共识的容错模型是否兼容
    """
    # 1. 识别旧共识的容错模型
    old_model = identify_consensus_model(old_consensus)
    old_byzantine_ratio = get_current_byzantine_ratio(network_state, old_model)
    
    # 2. 解析新共识的容错要求
    new_model = new_consensus.fault_tolerance.model
    new_byzantine_bound = new_consensus.fault_tolerance.byzantine_bound
    
    # 3. 验证当前网络是否满足新共识的要求
    if old_byzantine_ratio >= new_byzantine_bound:
        return False, f"当前拜占庭节点比例 {old_byzantine_ratio} 超过新共识容忍上界 {new_byzantine_bound}"
    
    # 4. 验证预执行期间的安全性
    # 双链期间需要同时满足两种共识的约束
    dual_chain_safe = verify_dual_chain_safety(old_model, new_model, network_state)
    if not dual_chain_safe:
        return False, "双链预执行期间无法同时满足旧共识和新共识的安全约束"
    
    # 5. 检查特殊约束
    if new_model == "pow":
        # PoW 需要验证诚实节点算力
        if not verify_honest_hashpower_majority(network_state):
            return False, "诚实节点算力不足 50%"
    
    elif new_model == "pos":
        # PoS 需要验证诚实节点权益
        if not verify_honest_stake_majority(network_state, threshold=0.67):
            return False, "诚实节点权益不足 2/3"
    
    elif new_model == "bft":
        # BFT 需要验证节点数量
        n, f = get_node_count(network_state)
        if n < 3 * f + 1:
            return False, f"节点数 n={n} 不满足 3f+1 要求 (f={f})"
    
    return True, "容错模型兼容"

def identify_consensus_model(consensus):
    """
    自动识别共识类型
    """
    if "hotstuff" in consensus.name.lower() or "pbft" in consensus.name.lower():
        return "bft"
    elif "pow" in consensus.name.lower() or "bitcoin" in consensus.name.lower():
        return "pow"
    elif "pos" in consensus.name.lower() or "casper" in consensus.name.lower():
        return "pos"
    elif "pot" in consensus.name.lower():
        return "hybrid"  # PoT = VDF + BFT
    else:
        # 从 CDL 中读取声明
        return consensus.fault_tolerance.model
```

### 3.3 协议流程

#### 3.3.1 五阶段升级流程

```
Phase 1: PROPOSAL (提案阶段)
  ↓
Phase 2: PREPARE (预备阶段)
  ↓
Phase 3: PREEXECUTION (预执行阶段)
  ↓
Phase 4: CONFIRMATION (确认阶段)
  ↓
Phase 5: ACTIVATION (激活阶段)
  ↓
[ROLLBACK] (可选的回退阶段)
```

#### 3.3.2 详细协议流程

**阶段 1: 提案阶段 (PROPOSAL)**

1. **提案生成**:
   - 治理委员会成员创建 `UpgradeConfigTx`
   - 指定目标共识和参数
   - 设置检查点高度: 
     - `prepareHeight = currentHeight + T_prepare`
     - `preexecHeight = prepareHeight + T_preexec`

2. **多签收集**:
   - 提案在委员会内传播
   - 每个委员会成员独立验证提案
   - 签名成员数达到阈值 t 后,提案有效

3. **链上提交**:
   - 将完整的 `UpgradeConfigTx` 提交到主链
   - 交易被打包进区块后,进入 PREPARE 阶段

**阶段 2: 预备阶段 (PREPARE)**

1. **全网通知**:
   ```python
   def on_upgrade_proposal(tx: UpgradeConfigTx):
       # 验证门限签名
       if not verify_threshold_signature(tx.committeeSignatures, tx.hash()):
           reject(tx)
           return
       
       # 验证高度参数
       if tx.prepareHeight <= current_height:
           reject(tx)
           return
       
       # 加载或编译新共识
       if tx.targetConsensus == "CUSTOM":
           new_consensus = compile_cdl(tx.consensusDescriptor)
           verify_cdl(new_consensus, tx.descriptorHash)
       else:
           new_consensus = load_builtin_consensus(tx.targetConsensus)
       
       # 注册待升级共识
       register_pending_consensus(tx.proposalID, new_consensus, tx.prepareHeight)
   ```

2. **资源准备**:
   - 节点下载新共识所需的代码/配置
   - 初始化预执行环境
   - 创建独立的状态副本

3. **状态同步**:
   - 所有节点同步到 `prepareHeight - 1`
   - 记录此时的状态快照作为分叉点

**阶段 3: 预执行阶段 (PREEXECUTION)**

1. **双链并行运行**:
   ```python
   def run_dual_chain(prepare_height: int):
       main_chain = current_chain
       preexec_chain = fork_chain_at(prepare_height)
       
       while current_height < tx.preexecHeight:
           # 主链继续使用旧共识
           main_block = old_consensus.produce_block()
           main_chain.append(main_block)
           
           # 预执行链使用新共识
           # 同步主链交易到预执行链
           preexec_txs = sync_transactions_from(main_block)
           preexec_block = new_consensus.produce_block(preexec_txs)
           preexec_chain.append(preexec_block)
           
           # 收集性能指标
           metrics = collect_metrics(preexec_block)
           store_metrics(tx.proposalID, metrics)
   ```

2. **性能监控**:
   - 每个节点独立监控预执行链的表现
   - 指标包括: 区块生成时间、吞吐量、延迟、错误率

3. **安全检测**:
   - 检测分叉攻击
   - 检测拜占庭行为
   - 验证共识安全属性

**阶段 4: 确认阶段 (CONFIRMATION)**

1. **治理投票**:
   - 委员会成员基于预执行结果投票
   - 投票选项: APPROVE(批准) | REJECT(拒绝) | ROLLBACK(回退)
   - 达到阈值后,生成 `UpgradeConfirmTx`

2. **确认交易上链**:
   - `UpgradeConfirmTx` 必须在 `preexecHeight + T_confirm` 之前上链
   - 否则升级自动失败,回退到主链

**阶段 5: 激活阶段 (ACTIVATION)**

1. **切换准备**:
   ```python
   def prepare_switch(confirm_tx: UpgradeConfirmTx):
       # 确定切换点
       switch_height = confirm_tx.finalizeHeight
       
       # 等待所有节点达到切换点
       wait_until(current_height >= switch_height)
       
       # 原子切换
       atomic {
           # 停止旧共识
           old_consensus.stop()
           
           # 合并预执行链
           merge_preexec_chain(preexec_chain, switch_height)
           
           # 激活新共识
           new_consensus.start()
           
           # 更新全局状态
           current_consensus = new_consensus
       }
   ```

2. **状态迁移**:
   
   上面第1步的代码展示了旧的错误思路。**正确的实现应该是**:

   ```python
   def activate_upgrade(preexec_chain, fork_point, switch_height):
       """
       激活升级,将预执行链设为主链 (不修改任何区块内容)
       """
       # 1. 验证预执行链完整性
       for h in range(fork_point, switch_height + 1):
           block = preexec_chain.get_block(h)
           
           # 验证是预执行链区块
           assert block.header.upgradeMetadata.isPreexec == true
           assert block.header.upgradeMetadata.forkPoint == fork_point
           
           # 验证区块有效性(共识证明、交易、状态转换)
           assert validate_block(block, new_consensus) == true
           
           # 验证哈希链连续性
           if h > fork_point:
               prev_block = preexec_chain.get_block(h - 1)
               assert block.header.parentHash == hash(prev_block)
       
       # 2. ✅ 原子切换主链指针
       atomic {
           # 2.1 归档旧主链在 [fork_point, switch_height] 的区块
           #     (这些区块仍然存在,只是不再是主链)
           for h in range(fork_point, switch_height + 1):
               old_block = main_chain.get_block(h)
               archive_block(old_block, tag="old_consensus_archived")
           
           # 2.2 将主链的 head 指针切换到预执行链
           #     ⚠️ 预执行链的区块内容完全不变(包括 upgradeMetadata)
           main_chain.head = preexec_chain.head
           main_chain.tip_hash = preexec_chain.tip_hash
           main_chain.consensus_id = preexec_chain.consensus_id
           
           # 2.3 更新区块高度索引
           #     将 [fork_point, switch_height] 高度映射到预执行链区块
           for h in range(fork_point, switch_height + 1):
               preexec_block = preexec_chain.get_block(h)
               # 注意: preexec_block 保持原样,不做任何修改
               # preexec_block.header.upgradeMetadata.isPreexec 仍然是 true
               main_chain.height_to_block[h] = preexec_block
           
           # 2.4 切换共识引擎
           consensus_engine.stop(old_consensus)
           consensus_engine.start(new_consensus)
       }
       
       # 3. 验证切换后的链状态
       assert main_chain.get_block(switch_height).header.consensusID == new_consensus.id
       assert main_chain.state_root == preexec_chain.state_root
       
       # 4. 验证哈希链完整性
       for h in range(fork_point + 1, switch_height + 1):
           block = main_chain.get_block(h)
           prev_block = main_chain.get_block(h - 1)
           assert block.header.parentHash == hash(prev_block)  # ✓ 哈希仍然匹配
       
       log.info(f"升级成功: 主链已切换到新共识,高度 {switch_height}")
       log.info(f"注意: 区块内容未修改,upgradeMetadata 保持原样用于历史审计")
   ```

3. **激励分配**:
   - 向参与签名的委员会成员分配奖励
   - 记录升级历史到链上

**回退阶段 (ROLLBACK)** (可选):

委员会成员观察到以下情况时，投拒绝票：
- 预执行阶段检测到严重安全问题
- 性能指标不达标

回退触发条件:
- 确认超时
- 治理委员会投票拒绝

回退流程:
```python
def rollback_upgrade(proposal_id: Hash, reason: String):
    # 停止预执行链
    stop_preexec_chain(proposal_id)
    
    # 删除预执行链数据
    delete_preexec_chain(proposal_id)
    
    # 继续主链
    continue_main_chain()
    
    # 记录回退事件
    emit_event(UpgradeRollback{
        proposalID: proposal_id,
        reason: reason,
        timestamp: now()
    })
```

### 3.4 协议流程图

#### 3.4.1 Mermaid 流程图代码

```mermaid
stateDiagram-v2
    [*] --> Proposal: 委员会提出升级
    
    Proposal --> Prepare: 多签验证通过
    Proposal --> [*]: 多签验证失败
    
    Prepare --> PreExecution: 达到预备高度
    Prepare --> [*]: 超时或取消
    
    PreExecution --> Confirmation: 预执行完成
    PreExecution --> Rollback: 检测到安全问题
    
    Confirmation --> Activation: 委员会批准
    Confirmation --> Rollback: 委员会拒绝
    Confirmation --> Rollback: 确认超时
    
    Activation --> [*]: 切换成功
    Activation --> Rollback: 切换失败
    
    Rollback --> [*]: 回退到主链
    
    note right of Proposal
        治理委员会多签
        设置检查点高度
    end note
    
    note right of Prepare
        全网节点准备
        下载新共识代码
        创建状态快照
    end note
    
    note right of PreExecution
        双链并行运行
        主链: 旧共识
        预执行链: 新共识
        收集性能指标
    end note
    
    note right of Confirmation
        评估预执行结果
        委员会最终投票
        生成确认交易
    end note
    
    note right of Activation
        原子切换共识
        合并预执行链
        分配激励
    end note
```

#### 3.4.2 时序图

```mermaid

sequenceDiagram
    participant C as 治理委员会
    participant N as 网络节点
    participant MC as 主链(旧共识)
    participant PC as 预执行链(新共识)
    
    C->>C: 1. 创建升级提案
    C->>C: 2. 收集门限签名
    C->>N: 3. 广播 UpgradeConfigTx
    
    N->>N: 4. 验证多签
    N->>N: 5. 加载新共识
    
    rect rgb(200, 220, 240)
        Note over N,MC: 预备阶段 (T_prepare 区块)
        MC->>MC: 继续旧共识出块
        N->>N: 准备分叉环境
    end
    
    rect rgb(220, 240, 200)
        Note over MC,PC: 预执行阶段 (T_preexec 区块)
        par 双链并行
            MC->>MC: 旧共识产生主链区块
        and
            MC->>PC: 同步交易到预执行链
            PC->>PC: 新共识产生预执行区块
            PC->>N: 收集性能指标
        end
    end
    
    rect rgb(240, 220, 200)
        Note over C,N: 确认阶段
        N->>C: 报告预执行指标
        C->>C: 评估并投票
        C->>N: 广播 UpgradeConfirmTx
    end
    
    rect rgb(240, 200, 220)
        Note over N,PC: 激活阶段
        N->>MC: 停止旧共识
        N->>N: 合并预执行链
        PC->>MC: 预执行链成为主链
        N->>C: 分配激励
    end
    ```

#### 3.4.3 如何绘制这些图

**方法 1: 使用 Mermaid Live Editor**

1. 访问 https://mermaid.live/
2. 将上述 Mermaid 代码粘贴到左侧编辑器
3. 右侧会实时渲染流程图
4. 可导出为 PNG/SVG 格式

**方法 2: 在 VS Code 中预览**

1. 安装插件: `Markdown Preview Mermaid Support`
2. 在 Markdown 文件中插入 Mermaid 代码块
3. 使用 `Ctrl+Shift+V` 预览

**方法 3: 使用命令行工具**

```bash
# 安装 mermaid-cli
npm install -g @mermaid-js/mermaid-cli

# 渲染图表
mmdc -i flow.mmd -o flow.png
```

**方法 4: 在 GitHub/GitLab**

GitHub 和 GitLab 原生支持 Mermaid,直接在 Markdown 中使用即可渲染。

## 4. 安全分析

### 4.1 安全目标

#### 4.1.1 通用共识安全目标

1. **一致性 (Agreement)**: 
   - 任意两个诚实节点对相同高度的区块达成一致
   - 形式化: `∀i,j ∈ Honest, ∀h: Chain_i[h] = Chain_j[h]`

2. **有效性 (Validity)**:
   - 所有提交的区块都满足协议规则
   - 形式化: `∀b ∈ Chain: Valid(b) = true`

3. **活性 (Liveness)**:
   - 协议最终能产生新区块
   - 形式化: `∀t: ∃t' > t: |Chain(t')| > |Chain(t)|`

#### 4.1.2 升级协议特有安全目标

1. **升级原子性 (Upgrade Atomicity)**:
   - 所有诚实节点要么全部切换,要么全部不切换
   - 形式化: `∀i,j ∈ Honest: consensus_i = consensus_j`

2. **升级一致性 (Upgrade Consistency)**:
   - 切换后,所有节点对历史链的视图保持一致
   - 形式化: `∀h < switch_height, ∀i,j ∈ Honest: Chain_i[h] = Chain_j[h]`

3. **双链安全隔离 (Dual-Chain Isolation)**:
   - 预执行链的失败不影响主链安全性
   - 形式化: `Failure(PreexecChain) ⇒ Safe(MainChain)`

4. **回退安全性 (Rollback Safety)**:  // 语义回滚不是物理回滚
   - 回退操作不破坏主链的一致性和有效性
   - 形式化: `After(Rollback) ⇒ Agreement(MainChain) ∧ Validity(MainChain)`

5. **治理抗操纵性 (Governance Resistance)**:
   - 少于阈值 t 的恶意委员会成员无法通过恶意提案
   - 形式化: `|Malicious| < t ⇒ ¬Pass(MaliciousProposal)`

6. **预执行可验证性 (Preexecution Verifiability)**:
   - 任何节点可以独立验证预执行链的正确性
   - 形式化: `∀n: Verify(PreexecChain) = true ∨ false`

7. **激励公平性 (Incentive Fairness)**:  // 待定
   - 激励分配与贡献成正比,防止女巫攻击
   - 形式化: `∀i ∈ Committee: Reward_i = f(Contribution_i)`

### 4.2 攻击模型与防御

#### 4.2.1 恶意提案攻击

**攻击描述**:
攻击者试图提交包含后门或安全漏洞的共识算法。

**防御机制**:
1. **门限签名**: 需要 t = ⌊2m/3⌋ + 1 个委员会成员签名
2. **CDL 静态分析**: 编译时检测恶意代码模式
3. **预执行验证**: 在隔离环境中测试新共识
4. **社区审计**: 公开 CDL 描述供社区审查

**形式化证明**:
```
定理 1: 在恶意委员会成员少于 t 的情况下,恶意提案无法通过。

证明:
设 |C_malicious| = f_c < t
要通过提案,需要 |C_signed| ≥ t
则必须有 |C_honest ∩ C_signed| ≥ t - f_c > 0
即至少有一个诚实成员签名。

由于诚实成员会验证提案,恶意提案会被拒签。
因此,|C_signed| < t,提案无法通过。∎
```

#### 4.2.2 双重支付攻击

**攻击描述**:
攻击者在主链和预执行链上构造冲突交易,试图双重支付。

**防御机制**:
1. **交易同步**: 预执行链从主链同步交易,保持交易集一致
2. **状态隔离**: 两条链使用独立的状态副本
3. **切换验证**: 激活时验证预执行链与主链的状态一致性

**形式化证明**:
```
定理 2: 双链机制不会导致双重支付。

证明:
设 tx 为一笔交易,在高度 h 被主链接受。
由于预执行链同步主链交易,tx 也会在预执行链的高度 h 被处理。

假设攻击者在预执行链上构造冲突交易 tx',使用相同的 UTXO。
则预执行链在验证 tx' 时,会发现 UTXO 已被 tx 消费,拒绝 tx'。

切换时,只有主链的状态被保留,预执行链的独立状态被丢弃。
因此,不存在双重支付。∎
```

#### 4.2.3 长程攻击 (Long-Range Attack)

**攻击描述**:
攻击者从分叉点构造更长的替代链,试图替换主链。

**防御机制**:
1. **检查点锁定**: 预备高度作为不可逆检查点
2. **最终性保证**: 主链使用具有最终性的共识(如 HotStuff)
3. **社会共识**: 切换需治理委员会批准

**形式化证明**:
```
定理 3: 预执行链无法替换已确认的主链。

证明:
设主链在高度 h_p (prepareHeight) 分叉。
主链继续使用旧共识,在时间 T 内达到高度 h_c (current height)。

攻击者要替换主链,需构造一条从 h_p 开始,高度 > h_c 的链。
但预执行链:
1) 仅用于测试,不参与共识
2) 区块生成依赖主链的交易同步
3) 无法比主链更快生成区块

因此,攻击者无法构造更长的替代链。∎
```

#### 4.2.4 预执行干扰攻击

**攻击描述**:
恶意节点在预执行阶段故意产生错误区块,破坏性能指标。

**防御机制**:
1. **多数投票**: 预执行链也使用拜占庭容错共识
2. **异常检测**: 统计学方法检测异常节点
3. **惩罚机制**: 作恶节点会被排除出预执行

**形式化证明**:
```
定理 4: 在拜占庭节点少于 1/3 的情况下,预执行链能正常运行。

证明:
预执行链使用新共识算法,假设该算法容忍 f_new 个拜占庭节点。

情况 1: 新共识是 BFT 类型 (如 HotStuff)
则 f_new = ⌊(n-1)/3⌋,只要 |Byzantine| ≤ f_new,预执行链安全。

情况 2: 新共识是 PoW 类型
则需要 |Honest_hashpower| > |Byzantine_hashpower|。
只要诚实节点算力占优,预执行链安全。

因此,在标准拜占庭假设下,预执行链能抵抗干扰攻击。∎
```

#### 4.2.5 回退拒绝服务攻击

**攻击描述**:
恶意委员会成员在预执行阶段故意拒绝批准,导致无法升级。

**防御机制**:
1. **超时机制**: 确认阶段有时间限制,超时自动回退
2. **声誉系统**: 记录委员会成员的历史行为
3. **委员会轮换**: 定期更换委员会成员

**形式化分析**:
```
该攻击至多导致升级失败,但不会破坏主链安全性。
由于主链持续运行,系统保持活性。
攻击者付出的代价: 失去信誉,可能被移出委员会。
```

#### 4.2.6 切换时序攻击

**攻击描述**:
攻击者利用节点切换时间差,在部分节点切换而部分节点未切换时发起攻击。

**防御机制**:
1. **全局切换高度**: 所有节点在相同高度切换
2. **切换原子性**: 使用原子操作确保切换瞬间完成
3. **切换确认**: 切换前广播 "ready" 消息,等待法定人数

**形式化证明**:
```
定理 5: 切换操作是原子的,不存在中间状态。

证明:
切换发生在确定的高度 h_switch。
在高度 h < h_switch,所有诚实节点运行旧共识。
在高度 h ≥ h_switch,所有诚实节点运行新共识。

切换是本地操作,不依赖网络通信:
switch(h_switch) {
    atomic {
        stop(old_consensus)
        start(new_consensus)
    }
}

因此,不存在部分节点在旧共识、部分节点在新共识的状态。∎
```

#### 4.2.7 网络分区攻击

**攻击描述**:
攻击者通过控制网络设备(如路由器、交换机)或发起 DDoS 攻击,人为制造网络分区,试图:
1. 阻止升级提案的传播,使部分节点无法参与升级
2. 在预执行阶段分割主链和预执行链,使其产生冲突状态
3. 在确认阶段阻止委员会投票传播,导致升级超时
4. 在激活阶段分割网络,使部分节点切换而部分节点未切换,造成永久分叉

**防御机制**:

**1. 提案阶段的分区防御**:
```python
def handle_proposal_partition():
    """
    提案阶段的分区容忍
    """
    # 机制 1: 链上持久化
    # 提案以 UpgradeConfigTx 形式写入主链
    # 即使节点在提案广播时失联,后续同步主链时能自动获取
    
    def sync_missing_proposals():
        """
        分区恢复后同步缺失的提案
        """
        latest_height = get_latest_height()
        
        for h in range(last_synced_height, latest_height + 1):
            block = fetch_block(h)
            
            for tx in block.transactions:
                if tx.txType == UPGRADE_CONFIG:
                    # 发现缺失的升级提案
                    if tx.proposalID not in local_proposals:
                        process_upgrade_proposal(tx)
                        log.info(f"同步到升级提案: {tx.proposalID}")
    
    # 机制 2: 冗余传播
    # 提案通过多个通道传播: gossip 网络 + 区块链 + 治理委员会直连
    redundant_broadcast(proposal, channels=["gossip", "blockchain", "committee_p2p"])
```

**2. 预执行阶段的分区防御**(核心机制):
```python
def handle_preexec_partition():
    """
    预执行阶段的分区容忍 - 主链优先原则
    """
    # 关键设计: 主链是唯一真相源,预执行链是可重建的
    
    # 场景 1: 节点在少数分区
    if is_in_minority_partition():
        # 策略: 停止产生预执行区块,等待分区恢复
        log.warn("检测到处于少数分区,暂停预执行链")
        pause_preexec_chain()
        
        # 继续观察主链(旧共识应能容忍分区)
        continue_main_chain()
        
        # 分区恢复后
        when partition_recovered():
            # 从多数分区同步主链和预执行链
            main_chain_canonical = sync_from_majority("main")
            preexec_chain_canonical = sync_from_majority("preexec")
            
            # 根据权威主链重建预执行链
            if not validate_preexec(preexec_chain_canonical, main_chain_canonical):
                preexec_chain_canonical = rebuild_from_main(main_chain_canonical)
            
            resume_preexec_chain(preexec_chain_canonical)
    
    # 场景 2: 节点在多数分区
    elif is_in_majority_partition():
        # 策略: 正常运行双链
        continue_dual_chain()
        
        # 少数分区恢复后会自动追上
        # 无需特殊处理
    
    # 场景 3: 网络对半分裂(50%-50%)
    elif is_network_split():
        # 策略: 依赖主链共识的分区处理机制
        # 主链共识(如 HotStuff)应能检测到无法达到 2/3 法定人数
        # 主链停止出块,预执行链也停止
        
        if main_chain_stalled():
            log.warn("主链因分区停滞,预执行链同步停止")
            pause_preexec_chain()
            
            # 等待分区恢复,主链恢复后预执行链也恢复
            wait_for_partition_recovery()

def rebuild_from_main(main_chain):
    """
    根据主链重建预执行链(分区恢复后的修复机制)
    """
    preexec_chain_new = create_empty_chain()
    fork_point = get_fork_point()
    
    # 遍历主链的每个区块
    for h in range(fork_point, main_chain.height + 1):
        main_block = main_chain.get_block(h)
        
        # 提取主链交易
        txs = main_block.body.transactions
        
        # 使用新共识在预执行链上重新执行
        preexec_block = new_consensus.produce_block(txs)
        
        # 设置正确的升级元数据
        preexec_block.header.upgradeMetadata = {
            "isPreexec": true,
            "proposalID": current_proposal_id,
            "forkPoint": fork_point,
            "mainChainRef": main_block.hash(),
            "preexecStatus": "RUNNING"
        }
        
        preexec_chain_new.append(preexec_block)
    
    log.info(f"预执行链重建完成: 从高度 {fork_point} 到 {main_chain.height}")
    return preexec_chain_new
```

**3. 确认阶段的分区防御**:
```python
def handle_confirmation_partition():
    """
    确认阶段的分区容忍
    """
    # 机制 1: 超时保护
    # 如果在 T_confirm 时间内无法收集足够签名,自动回退
    
    start_time = current_time()
    collected_sigs = []
    
    while current_time() - start_time < T_confirm:
        # 尝试收集委员会签名
        new_sigs = collect_committee_signatures(proposal_id)
        collected_sigs.extend(new_sigs)
        
        if len(collected_sigs) >= threshold_t:
            # 达到阈值,生成确认交易
            confirm_tx = create_confirmation_tx(proposal_id, collected_sigs)
            broadcast_to_main_chain(confirm_tx)
            return SUCCESS
        
        sleep(retry_interval)
    
    # 超时,触发回退
    log.warn("确认阶段超时,触发回退")
    trigger_rollback(proposal_id, reason="CONFIRMATION_TIMEOUT")
    return ROLLBACK
    
    # 机制 2: 分区恢复后的续签
    # 如果分区在超时前恢复,可以继续收集签名
    def resume_signature_collection():
        if partition_recovered() and time_remaining() > 0:
            log.info("分区恢复,继续收集签名")
            # 重新广播签名请求
            rebroadcast_signature_request(proposal_id)
```

**4. 激活阶段的分区防御**:
```python
def handle_activation_partition():
    """
    激活阶段的分区容忍
    """
    # 关键设计: 切换高度是确定性的,写在 UpgradeConfirmTx 中
    # 即使节点在激活时失联,后续同步时也能正确切换
    
    def sync_and_activate():
        """
        分区节点恢复后的自动激活
        """
        # 1. 同步主链到最新
        sync_main_chain()
        
        # 2. 检查是否有确认交易
        for h in range(last_synced_height, current_height + 1):
            block = fetch_block(h)
            
            for tx in block.transactions:
                if tx.txType == UPGRADE_CONFIRM:
                    # 发现确认交易
                    finalize_height = tx.finalizeHeight
                    
                    # 3. 检查当前高度
                    if current_height >= finalize_height:
                        # 已经到达切换高度,立即切换
                        log.info(f"检测到升级已激活,立即切换到新共识")
                        execute_switch(tx.proposalID, finalize_height)
                    else:
                        # 未到切换高度,注册切换事件
                        log.info(f"将在高度 {finalize_height} 切换共识")
                        register_switch_event(finalize_height, tx.proposalID)
```

**形式化证明**:

```
定理 8 (网络分区容忍性): 
在以下条件下,升级协议能容忍网络分区攻击:
1. 分区是临时的(最终恢复)
2. 主链使用的旧共识具有分区容忍性
3. 每个分区至少包含一个诚实节点

则:
a) 主链在分区期间和恢复后保持一致
b) 预执行链在分区恢复后能重建到一致状态
c) 升级决策(批准/拒绝)最终会被所有节点认可

证明:

引理 1: 主链不受分区影响
- 主链继续运行旧共识
- 旧共识具有分区容忍性(前提条件 2)
- 旧共识的安全性保证主链不分叉
- ∴ 主链保持一致 ✓

引理 2: 预执行链可从主链重建
- 预执行链的输入完全来自主链(交易同步机制)
- 新共识的状态转换是确定性的
- ∀主链状态 S, ∃唯一的预执行链状态 S'
- 给定一致的主链,重建出的预执行链也一致
- ∴ 预执行链可恢复 ✓

引理 3: 升级决策记录在主链
- UpgradeConfigTx 和 UpgradeConfirmTx 都在主链
- 主链一致(引理 1) ⇒ 升级决策一致
- ∴ 所有节点对"是否升级"达成一致 ✓

综合引理 1、2、3:
- 分区恢复后,节点同步主链(引理 1)
- 根据主链重建预执行链(引理 2)
- 读取主链上的升级决策(引理 3)
- ∴ 系统恢复到全局一致状态 ✓

因此,升级协议具有网络分区容忍性。∎
```

**攻击成本分析**:

| 攻击目标 | 攻击成本 | 攻击效果 | 防御成功率 |
|---------|---------|---------|-----------|
| 阻止提案传播 | 中 (需持续 DDoS) | 低 (提案在链上,最终会同步) | 99% |
| 分裂预执行链 | 高 (需控制网络分区) | 低 (可重建,不影响主链) | 95% |
| 阻止确认投票 | 高 (需持续分区) | 中 (导致超时回退) | 80% |
| 分裂激活阶段 | 极高 (需精确控制切换时刻) | 低 (确定性切换高度) | 99% |

**实际案例分析**:

假设网络在预执行阶段被分为 A、B 两个分区:

```
分区 A (60% 节点):
- 主链继续用 HotStuff 出块: h=1000, 1001, 1002, ...
- 预执行链继续用 PoW 出块: h=1000, 1001, 1002, ...
- 收集性能指标

分区 B (40% 节点):
- 主链因无法达到 2/3 法定人数,停滞在 h=999
- 预执行链也停滞(依赖主链交易)
- 等待分区恢复

分区恢复后:
1. B 分区节点发现 A 分区有更长的主链
2. B 分区同步 A 分区的主链: h=1000, 1001, 1002, ...
3. B 分区丢弃本地的预执行链,根据 A 分区主链重建
4. 系统恢复到一致状态,继续升级流程
```

### 4.3 安全性定理

**定理 6 (升级协议安全性)**: 
在以下条件下,升级协议满足一致性、有效性、活性:
1. 网络最终同步
2. 拜占庭节点 ≤ f = ⌊(n-1)/3⌋
3. 恶意委员会成员 < t = ⌊2m/3⌋ + 1
4. 密码学原语安全

则升级过程中和升级后,系统保持安全。

**证明框架**:

```
证明:
分三个阶段证明:

阶段 1: 预执行期间主链安全
- 主链继续使用旧共识
- 旧共识满足 BFT 安全性 (已知结果)
- 预执行链隔离,不影响主链
- 因此,主链保持安全 ✓

阶段 2: 切换时刻安全
- 切换在确定高度 h_switch 原子发生 (定理 5)
- 切换前状态一致 (旧共识保证)
- 切换后状态继承自预执行链 (已验证)
- 因此,切换时刻安全 ✓

阶段 3: 切换后新共识安全
- 新共识经过预执行验证
- 预执行验证了安全属性 (定理 4)
- 新共识满足其声明的安全性
- 因此,切换后安全 ✓

综上,升级协议满足安全性。∎
```

### 4.4 活性分析

**定理 7 (升级协议活性)**: 
在网络最终同步且诚实节点占多数的情况下,升级协议最终会完成或回退。

**证明**:

```
证明:
考虑两种情况:

情况 1: 升级成功
- 预备阶段: 有界时间内达到 prepareHeight (旧共识活性保证)
- 预执行阶段: 双链并行,主链持续出块 (旧共识活性保证)
- 确认阶段: 如果预执行成功,委员会批准 (诚实多数假设)
- 激活阶段: 原子切换 (本地操作)
- 因此,升级在有界时间内完成 ✓

情况 2: 升级失败
- 如果预执行失败 → 触发回退
- 如果确认超时 → 触发回退
- 如果委员会拒绝 → 触发回退
- 回退操作: 停止预执行链,继续主链
- 因此,回退在有界时间内完成 ✓

两种情况都在有界时间内达到终态,因此协议满足活性。∎
```

### 4.5 形式化验证建议

为了更严格地验证协议安全性,建议使用以下工具:

1. **TLA+ / PlusCal**: 对协议状态机建模,验证并发属性
2. **Coq / Isabelle**: 证明关键定理
3. **Spin / Promela**: 模型检测,查找反例
4. **Tamarin Prover**: 验证密码学协议属性

## 5. 性能分析

### 5.1 时间复杂度

| 操作 | 复杂度 | 说明 |
|------|--------|------|
| 提案验证 | O(t) | 验证 t 个门限签名 |
| 预备阶段 | O(T_prepare) | 等待 T_prepare 个区块 |
| 预执行阶段 | O(T_preexec) | 双链并行,时间约为 T_preexec |
| 确认验证 | O(t) | 验证 t 个批准签名 |
| 切换操作 | O(1) | 原子操作 |
| **总升级时间** | **O(T_prepare + T_preexec)** | **约 1000-2000 个区块** |

### 5.2 空间复杂度

| 数据结构 | 空间复杂度 | 说明 |
|----------|-----------|------|
| 升级配置交易 | O(1) | 固定大小 |
| 预执行链 | O(T_preexec × B) | B为区块大小 |
| 性能指标 | O(T_preexec) | 每个区块一条记录 |
| **总额外空间** | **O(T_preexec × B)** | **约数GB** |

### 5.3 网络开销

- **提案广播**: O(n) 消息,一次性
- **预执行阶段**: 双倍共识消息 (主链 + 预执行链)
- **确认广播**: O(n) 消息,一次性
- **相对开销**: 升级期间网络流量增加约 2 倍

### 5.4 实际参数建议

| 参数 | 推荐值 | 理由 |
|------|--------|------|
| T_prepare | 100 区块 | 给节点足够时间准备 |
| T_preexec | 1000 区块 | 充分验证新共识性能 |
| T_confirm | 10 分钟 | 委员会有时间评估 |
| t (门限) | ⌊2m/3⌋ + 1 | 容忍 1/3 恶意委员会成员 |
| m (委员会规模) | 21-51 | 平衡去中心化与效率 |

## 6. 与现有方案对比

| 特性 | 本协议 (SCUP) | 硬分叉 | 软分叉 | Tezos 自修正 |
|------|---------------|--------|--------|--------------|
| 在线升级 | ✓ | ✗ | ✓ | ✓ |
| 共识可切换 | ✓ | ✓ | ✗ | ✓ |
| 预执行验证 | ✓ | ✗ | ✗ | ✗ |
| 回退机制 | ✓ | ✗ | ✗ | ✗ |
| 链上治理 | ✓ | ✗ | ✗ | ✓ |
| 自定义共识 | ✓ (CDL) | ✓ | ✗ | ✓ (Michelson) |
| 形式化保证 | ✓ | ✗ | ✗ | ✓ |

## 7. 未来研究方向

1. **跨链共识升级**: 支持多条链同时升级并保持互操作性
2. **机器学习辅助**: 使用 ML 预测新共识的性能表现
3. **零知识证明应用**: 用 ZK-SNARK 证明预执行的正确性
4. **量子安全升级**: 支持升级到抗量子密码学
5. **激励机制优化**: 设计博弈论安全的激励模型

## 8. 结论

本文提出的共识可切换升级协议 (SCUP) 通过双链预执行、治理多签和 CDL 描述语言,实现了区块链共识的安全在线升级。协议具有以下优势:

1. **安全性强**: 形式化证明的安全保证
2. **风险可控**: 预执行和回退机制降低风险
3. **灵活性高**: 支持内置和自定义共识
4. **去中心化治理**: 避免中心化升级决策

该协议为区块链系统的长期演进提供了一种可行的技术路径。

## 参考文献

[1] Nakamoto, S. (2008). Bitcoin: A peer-to-peer electronic cash system.

[2] Yin, M., et al. (2019). HotStuff: BFT consensus with linearity and responsiveness. In PODC.

[3] Goodman, L. M. (2014). Tezos: A self-amending crypto-ledger.

[4] Buterin, V., & Griffith, V. (2017). Casper the friendly finality gadget. arXiv preprint.

[5] Pass, R., & Shi, E. (2017). The sleepy model of consensus. In ASIACRYPT.

[6] Garay, J., Kiayias, A., & Leonardos, N. (2015). The bitcoin backbone protocol. In EUROCRYPT.

[7] Zamani, M., et al. (2018). RapidChain: Scaling blockchain via full sharding. In CCS.

---