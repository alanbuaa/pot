# 共识可切换升级协议实现文档

## 目录

- [1. 概述](#1-概述)
- [2. 项目架构改动](#2-项目架构改动)
- [3. 核心数据结构实现](#3-核心数据结构实现)
- [4. 升级配置交易实现](#4-升级配置交易实现)
- [5. 双链管理机制](#5-双链管理机制)
- [6. 治理委员会与多签验证](#6-治理委员会与多签验证)
- [7. CDL 引擎实现](#7-cdl-引擎实现)
- [8. 预执行与性能监控](#8-预执行与性能监控)
- [9. 切换与回退机制](#9-切换与回退机制)
- [10. 自定义 PoW 共识示例](#10-自定义-pow-共识示例)
- [11. API 接口设计](#11-api-接口设计)
- [12. 测试方案](#12-测试方案)

---

## 1. 概述

本文档基于现有项目代码结构,详细说明如何实现共识可切换升级协议。实现分为以下几个模块:

```
consensus/
├── upgrade/                    # 新增: 升级协议核心模块
│   ├── types.go               # 升级相关类型定义
│   ├── transaction.go         # 升级配置交易
│   ├── governance.go          # 治理委员会管理
│   ├── dual_chain.go          # 双链管理
│   ├── cdl/                   # CDL 引擎
│   │   ├── parser.go         # CDL 解析器
│   │   ├── validator.go      # CDL 验证器
│   │   ├── compiler.go       # CDL 编译器
│   │   └── runtime.go        # CDL 运行时
│   ├── metrics.go            # 性能指标收集
│   ├── switch.go             # 切换逻辑
│   └── rollback.go           # 回退逻辑

types/
├── upgrade_tx.go              # 新增: 升级交易类型
└── block.go                   # 修改: 支持双链区块

pkg/proto/
└── upgrade.proto              # 新增: 升级相关 protobuf 定义

internal/storage/
├── dual_chain_storage.go      # 新增: 双链存储
└── metrics_storage.go         # 新增: 指标存储
```

## 2. 项目架构改动

### 2.1 现有代码改动点

#### 2.1.1 `types/transaction.go` 扩展

**位置**: `/root/ldc/workspace/pot/types/transaction.go`

**改动内容**: 添加新的交易类型

```go
// 在现有 TransactionType 中新增
const (
    TxTypeNormal           = 0
    TxTypeUpgradeConfig    = 1  // 新增: 升级配置交易
    TxTypeUpgradeConfirm   = 2  // 新增: 升级确认交易
    TxTypeTimeVote         = 3
    TxTypeLock             = 4
)
```

#### 2.1.2 `types/block.go` 扩展

**位置**: `/root/ldc/workspace/pot/types/block.go`

**改动内容**: 在 `Header` 中添加升级相关字段

```go
type Header struct {
    Height         uint64
    ParentHash     []byte
    UncleHash      [][]byte
    Mixdigest      []byte
    Difficulty     *big.Int
    Nonce          int64
    Timestamp      time.Time
    PoTProof       [][]byte
    Address        int64
    PeerId         string
    TxHash         []byte
    ExeHash        []byte
    Hashes         []byte
    PublicKey      []byte
    CryptoElement  crypto.CryptoElement
    CommiteePubkey []byte
    
    // 新增: 升级相关字段
    ConsensusID         int64   // 当前使用的共识 ID
    PreexecChainRef     []byte  // 预执行链引用 (如果存在)
    UpgradePhase        uint8   // 升级阶段标识
}
```

#### 2.1.3 `consensus/consensusFactory.go` 扩展

**位置**: `/root/ldc/workspace/pot/consensus/consensusFactory.go`

**改动内容**: 添加动态共识加载支持

```go
// 在 BuildConsensus 中添加对自定义共识的支持
func BuildConsensus(
    nid int64,
    cid int64,
    cfg *config.ConsensusConfig,
    exec executor.Executor,
    p2pAdaptor p2p.P2PAdaptor,
    log *logrus.Entry,
) model.Consensus {
    var c model.Consensus = nil
    
    // ... 现有代码 ...
    
    // 新增: 自定义共识加载
    case "custom":
        // 从 CDL 描述符加载自定义共识
        cdlEngine := upgrade.NewCDLEngine(log)
        c = cdlEngine.LoadConsensus(cfg.CustomCDL, nid, cid, cfg, exec, p2pAdaptor)
    
    default:
        log.Warnf("init consensus type not supported: %s", cfg.Type)
    }
    
    return c
}

// 新增: 根据配置动态构建共识
func BuildConsensusFromConfig(
    nid int64,
    upgradeCfg *upgrade.UpgradeConfig,
    exec executor.Executor,
    p2pAdaptor p2p.P2PAdaptor,
    log *logrus.Entry,
) model.Consensus {
    cfg := upgradeCfg.ToConsensusConfig()
    return BuildConsensus(nid, upgradeCfg.ConsensusID, cfg, exec, p2pAdaptor, log)
}
```

#### 2.1.4 `consensus/upgradeableConsensus.go` 扩展

**位置**: `/root/ldc/workspace/pot/consensus/upgradeableConsensus.go`

**改动内容**: 集成升级协议管理器

```go
type UpgradeableConsensus struct {
    // ... 现有字段 ...
    
    // 新增: 升级协议管理器
    upgradeManager *upgrade.UpgradeManager
}

func NewUpgradeableConsensus(...) *UpgradeableConsensus {
    uc := &UpgradeableConsensus{
        // ... 现有初始化 ...
    }
    
    // 新增: 初始化升级管理器
    uc.upgradeManager = upgrade.NewUpgradeManager(
        uc,
        uc.config,
        uc.log,
    )
    
    return uc
}

// 新增: 处理升级交易
func (uc *UpgradeableConsensus) handleUpgradeTx(tx *pb.Transaction) {
    uc.upgradeManager.ProcessUpgradeTransaction(tx)
}
```

### 2.2 新增文件清单

| 文件路径 | 作用 | 依赖 |
|----------|------|------|
| `consensus/upgrade/manager.go` | 升级协议总控制器 | 所有升级模块 |
| `consensus/upgrade/types.go` | 升级类型定义 | protobuf |
| `consensus/upgrade/transaction.go` | 升级交易处理 | crypto, types |
| `consensus/upgrade/governance.go` | 治理委员会 | crypto (门限签名) |
| `consensus/upgrade/dual_chain.go` | 双链管理 | storage |
| `consensus/upgrade/cdl/*` | CDL 引擎 | yaml, parser |
| `consensus/upgrade/metrics.go` | 性能监控 | time, statistics |
| `consensus/upgrade/switch.go` | 切换逻辑 | consensus |
| `types/upgrade_tx.go` | 升级交易类型 | proto |
| `pkg/proto/upgrade.proto` | protobuf 定义 | - |
| `internal/storage/dual_chain_storage.go` | 双链存储 | leveldb |

## 3. 核心数据结构实现

### 3.1 Protobuf 定义

**文件**: `pkg/proto/upgrade.proto`

```protobuf
syntax = "proto3";
package pb;

option go_package = "github.com/zzz136454872/upgradeable-consensus/pkg/proto";

// 升级阶段枚举
enum UpgradePhase {
    PHASE_NONE = 0;
    PHASE_PROPOSAL = 1;
    PHASE_PREPARE = 2;
    PHASE_PREEXECUTION = 3;
    PHASE_CONFIRMATION = 4;
    PHASE_ACTIVATION = 5;
    PHASE_ROLLBACK = 6;
}

// 升级配置交易
message UpgradeConfigTransaction {
    // 提案标识
    bytes proposal_id = 1;
    
    // 目标共识
    string target_consensus = 2;           // "pot" | "pow" | "hotstuff" | "custom"
    bytes consensus_descriptor_hash = 3;   // 自定义共识的 CDL 哈希
    string consensus_descriptor_cdl = 4;   // CDL YAML 内容
    
    // 高度检查点
    uint64 prepare_height = 5;
    uint64 preexec_height = 6;
    
    // 升级阶段
    UpgradePhase phase = 7;
    
    // 治理
    repeated bytes committee_signatures = 8;  // 门限签名
    repeated bytes committee_pubkeys = 9;     // 委员会公钥列表
    uint32 threshold = 10;                     // 签名阈值
    
    // 激励
    uint64 incentive_amount = 11;
    repeated bytes incentive_recipients = 12;
    
    // 安全参数
    RollbackCondition rollback_condition = 13;
    double safety_threshold = 14;
    
    // 元数据
    int64 timestamp = 15;
    uint32 version = 16;
    bytes proposer = 17;
    uint64 nonce = 18;
}

// 回退条件
message RollbackCondition {
    double max_error_rate = 1;          // 最大错误率 (0.05 = 5%)
    double max_block_time_ratio = 2;    // 最大区块时间比例 (1.5 = 150%)
    double min_throughput_ratio = 3;    // 最小吞吐量比例 (0.8 = 80%)
    uint64 timeout_blocks = 4;          // 超时区块数
}

// 升级确认交易
message UpgradeConfirmTransaction {
    bytes proposal_id = 1;
    bytes preexec_chain_head = 2;
    ExecutionMetrics preexec_metrics = 3;
    repeated bytes committee_approvals = 4;
    uint64 finalize_height = 5;
    int64 timestamp = 6;
    bool approved = 7;  // true=批准, false=拒绝
}

// 执行指标
message ExecutionMetrics {
    double avg_block_time = 1;          // 平均区块时间 (秒)
    double avg_throughput = 2;          // 平均吞吐量 (tx/s)
    double avg_latency = 3;             // 平均延迟 (秒)
    double error_rate = 4;              // 错误率
    uint64 total_blocks = 5;            // 总区块数
    uint64 total_txs = 6;               // 总交易数
    repeated BlockMetric block_metrics = 7;  // 每个区块的指标
}

// 单个区块指标
message BlockMetric {
    uint64 height = 1;
    double block_time = 2;
    uint32 tx_count = 3;
    bool has_error = 4;
    string error_msg = 5;
}

// 预执行链区块头
message PreexecHeader {
    uint64 height = 1;
    bytes parent_hash = 2;
    uint64 fork_point = 3;           // 分叉点高度
    bytes main_chain_ref = 4;        // 对应主链区块
    int64 timestamp = 5;
    int64 consensus_id = 6;
    bytes state_root = 7;
    bytes tx_root = 8;
    ExecutionStatus status = 9;
}

enum ExecutionStatus {
    STATUS_RUNNING = 0;
    STATUS_SUCCESS = 1;
    STATUS_FAILED = 2;
}

// CDL 共识描述
message ConsensusDescriptorCDL {
    string name = 1;
    string version = 2;
    string type = 3;
    bytes components = 4;      // YAML 序列化
    bytes parameters = 5;      // YAML 序列化
    bytes phases = 6;          // YAML 序列化
    bytes state_machine = 7;   // YAML 序列化
    bytes safety_properties = 8;
    bytes performance_requirements = 9;
}
```

### 3.2 Go 类型定义

**文件**: `consensus/upgrade/types.go`

```go
package upgrade

import (
    "time"
    
    "github.com/zzz136454872/upgradeable-consensus/crypto"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
    "github.com/zzz136454872/upgradeable-consensus/types"
)

// UpgradeProposal 升级提案
type UpgradeProposal struct {
    ProposalID          types.TxHash
    TargetConsensus     string
    DescriptorHash      []byte
    DescriptorCDL       *CDLDescriptor
    
    PrepareHeight       uint64
    PreexecHeight       uint64
    
    Phase               pb.UpgradePhase
    
    CommitteeSignatures [][]byte
    CommitteePubkeys    [][]byte
    Threshold           uint32
    
    IncentiveAmount     uint64
    IncentiveRecipients [][]byte
    
    RollbackCondition   *pb.RollbackCondition
    SafetyThreshold     float64
    
    Timestamp           time.Time
    Version             uint32
    Proposer            []byte
    Nonce               uint64
}

// ToProto 转换为 protobuf
func (up *UpgradeProposal) ToProto() *pb.UpgradeConfigTransaction {
    var cdlStr string
    if up.DescriptorCDL != nil {
        cdlStr = up.DescriptorCDL.Serialize()
    }
    
    return &pb.UpgradeConfigTransaction{
        ProposalId:              up.ProposalID[:],
        TargetConsensus:         up.TargetConsensus,
        ConsensusDescriptorHash: up.DescriptorHash,
        ConsensusDescriptorCdl:  cdlStr,
        PrepareHeight:           up.PrepareHeight,
        PreexecHeight:           up.PreexecHeight,
        Phase:                   up.Phase,
        CommitteeSignatures:     up.CommitteeSignatures,
        CommitteePubkeys:        up.CommitteePubkeys,
        Threshold:               up.Threshold,
        IncentiveAmount:         up.IncentiveAmount,
        IncentiveRecipients:     up.IncentiveRecipients,
        RollbackCondition:       up.RollbackCondition,
        SafetyThreshold:         up.SafetyThreshold,
        Timestamp:               up.Timestamp.Unix(),
        Version:                 up.Version,
        Proposer:                up.Proposer,
        Nonce:                   up.Nonce,
    }
}

// Hash 计算提案哈希
func (up *UpgradeProposal) Hash() types.TxHash {
    protoTx := up.ToProto()
    data, _ := proto.Marshal(protoTx)
    return crypto.Hash(data)
}

// UpgradeState 升级状态
type UpgradeState struct {
    CurrentProposal  *UpgradeProposal
    Phase            pb.UpgradePhase
    
    // 主链状态
    MainChain        *ChainState
    
    // 预执行链状态
    PreexecChain     *ChainState
    
    // 性能指标
    Metrics          *PerformanceMetrics
    
    // 切换状态
    SwitchHeight     uint64
    Switched         bool
}

// ChainState 链状态
type ChainState struct {
    ConsensusID      int64
    CurrentHeight    uint64
    Head             *types.Block
    Consensus        model.Consensus
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
    StartHeight      uint64
    EndHeight        uint64
    
    BlockTimes       []time.Duration
    Throughputs      []float64
    Latencies        []time.Duration
    Errors           []error
    
    AvgBlockTime     time.Duration
    AvgThroughput    float64
    AvgLatency       time.Duration
    ErrorRate        float64
}

// ToProto 转换为 protobuf
func (pm *PerformanceMetrics) ToProto() *pb.ExecutionMetrics {
    blockMetrics := make([]*pb.BlockMetric, len(pm.BlockTimes))
    for i := range pm.BlockTimes {
        blockMetrics[i] = &pb.BlockMetric{
            Height:     pm.StartHeight + uint64(i),
            BlockTime:  pm.BlockTimes[i].Seconds(),
            TxCount:    uint32(pm.Throughputs[i]),
            HasError:   pm.Errors[i] != nil,
            ErrorMsg:   func() string {
                if pm.Errors[i] != nil {
                    return pm.Errors[i].Error()
                }
                return ""
            }(),
        }
    }
    
    return &pb.ExecutionMetrics{
        AvgBlockTime:  pm.AvgBlockTime.Seconds(),
        AvgThroughput: pm.AvgThroughput,
        AvgLatency:    pm.AvgLatency.Seconds(),
        ErrorRate:     pm.ErrorRate,
        TotalBlocks:   uint64(len(pm.BlockTimes)),
        TotalTxs:      uint64(pm.AvgThroughput * float64(len(pm.BlockTimes))),
        BlockMetrics:  blockMetrics,
    }
}

// Evaluate 评估指标是否满足要求
func (pm *PerformanceMetrics) Evaluate(condition *pb.RollbackCondition) bool {
    if pm.ErrorRate > condition.MaxErrorRate {
        return false
    }
    // 其他评估逻辑...
    return true
}
```

## 4. 升级配置交易实现

### 4.1 交易创建与验证

**文件**: `consensus/upgrade/transaction.go`

```go
package upgrade

import (
    "errors"
    "fmt"
    
    "github.com/zzz136454872/upgradeable-consensus/crypto"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
    "github.com/zzz136454872/upgradeable-consensus/types"
    "google.golang.org/protobuf/proto"
)

// CreateUpgradeProposal 创建升级提案
func CreateUpgradeProposal(
    targetConsensus string,
    cdl *CDLDescriptor,
    currentHeight uint64,
    committee *GovernanceCommittee,
    incentive uint64,
) (*UpgradeProposal, error) {
    
    proposal := &UpgradeProposal{
        TargetConsensus:     targetConsensus,
        PrepareHeight:       currentHeight + 100,  // T_prepare
        PreexecHeight:       currentHeight + 1100, // T_prepare + T_preexec
        Phase:               pb.UpgradePhase_PHASE_PROPOSAL,
        IncentiveAmount:     incentive,
        IncentiveRecipients: committee.GetPublicKeys(),
        Threshold:           committee.Threshold,
        RollbackCondition: &pb.RollbackCondition{
            MaxErrorRate:        0.05,
            MaxBlockTimeRatio:   1.5,
            MinThroughputRatio:  0.8,
            TimeoutBlocks:       100,
        },
        SafetyThreshold:     0.95,
        Timestamp:           time.Now(),
        Version:             1,
        Nonce:               rand.Uint64(),
    }
    
    // 如果是自定义共识,设置 CDL
    if targetConsensus == "custom" {
        if cdl == nil {
            return nil, errors.New("custom consensus requires CDL descriptor")
        }
        proposal.DescriptorCDL = cdl
        proposal.DescriptorHash = cdl.Hash()
    }
    
    // 计算提案 ID
    proposal.ProposalID = proposal.Hash()
    
    return proposal, nil
}

// SignProposal 委员会成员签名提案
func SignProposal(
    proposal *UpgradeProposal,
    privateKey crypto.PrivateKey,
    publicKey crypto.PublicKey,
) ([]byte, error) {
    hash := proposal.Hash()
    
    // 使用门限签名
    sigShare, err := crypto.TSign(hash[:], privateKey, publicKey)
    if err != nil {
        return nil, err
    }
    
    sigBytes, err := sigShare.MarshalBinary()
    if err != nil {
        return nil, err
    }
    
    return sigBytes, nil
}

// VerifyProposalSignatures 验证提案签名
func VerifyProposalSignatures(
    proposal *UpgradeProposal,
    committee *GovernanceCommittee,
) error {
    if len(proposal.CommitteeSignatures) < int(proposal.Threshold) {
        return fmt.Errorf("insufficient signatures: got %d, need %d",
            len(proposal.CommitteeSignatures), proposal.Threshold)
    }
    
    hash := proposal.Hash()
    validCount := 0
    
    for i, sigBytes := range proposal.CommitteeSignatures {
        if i >= len(proposal.CommitteePubkeys) {
            continue
        }
        
        // 反序列化签名
        sig := &tcrsa.SigShare{}
        if err := sig.UnmarshalBinary(sigBytes); err != nil {
            continue
        }
        
        // 验证部分签名
        pubkey := committee.GetMemberByPublicKey(proposal.CommitteePubkeys[i])
        if pubkey == nil {
            continue
        }
        
        if crypto.VerifyPartSig(sig, hash[:], pubkey.PublicKey) == nil {
            validCount++
        }
    }
    
    if validCount < int(proposal.Threshold) {
        return fmt.Errorf("invalid signatures: only %d valid out of %d",
            validCount, len(proposal.CommitteeSignatures))
    }
    
    return nil
}

// PackUpgradeTransaction 将提案打包成交易
func PackUpgradeTransaction(proposal *UpgradeProposal) (*pb.Transaction, error) {
    protoProposal := proposal.ToProto()
    payload, err := proto.Marshal(protoProposal)
    if err != nil {
        return nil, err
    }
    
    return &pb.Transaction{
        Type:    pb.TransactionType_UPGRADE,
        Payload: payload,
    }, nil
}

// UnpackUpgradeTransaction 从交易中解析提案
func UnpackUpgradeTransaction(tx *pb.Transaction) (*UpgradeProposal, error) {
    if tx.Type != pb.TransactionType_UPGRADE {
        return nil, errors.New("not an upgrade transaction")
    }
    
    protoProposal := &pb.UpgradeConfigTransaction{}
    if err := proto.Unmarshal(tx.Payload, protoProposal); err != nil {
        return nil, err
    }
    
    return ProposalFromProto(protoProposal), nil
}

// ProposalFromProto 从 protobuf 转换
func ProposalFromProto(pb *pb.UpgradeConfigTransaction) *UpgradeProposal {
    var cdl *CDLDescriptor
    if pb.ConsensusDescriptorCdl != "" {
        cdl = ParseCDL(pb.ConsensusDescriptorCdl)
    }
    
    var proposalID types.TxHash
    copy(proposalID[:], pb.ProposalId)
    
    return &UpgradeProposal{
        ProposalID:          proposalID,
        TargetConsensus:     pb.TargetConsensus,
        DescriptorHash:      pb.ConsensusDescriptorHash,
        DescriptorCDL:       cdl,
        PrepareHeight:       pb.PrepareHeight,
        PreexecHeight:       pb.PreexecHeight,
        Phase:               pb.Phase,
        CommitteeSignatures: pb.CommitteeSignatures,
        CommitteePubkeys:    pb.CommitteePubkeys,
        Threshold:           pb.Threshold,
        IncentiveAmount:     pb.IncentiveAmount,
        IncentiveRecipients: pb.IncentiveRecipients,
        RollbackCondition:   pb.RollbackCondition,
        SafetyThreshold:     pb.SafetyThreshold,
        Timestamp:           time.Unix(pb.Timestamp, 0),
        Version:             pb.Version,
        Proposer:            pb.Proposer,
        Nonce:               pb.Nonce,
    }
}

// ValidateProposalParameters 验证提案参数
func ValidateProposalParameters(
    proposal *UpgradeProposal,
    currentHeight uint64,
) error {
    // 检查高度参数
    if proposal.PrepareHeight <= currentHeight {
        return errors.New("prepare height must be in the future")
    }
    
    if proposal.PreexecHeight <= proposal.PrepareHeight {
        return errors.New("preexec height must be after prepare height")
    }
    
    minPrepareGap := uint64(100)
    if proposal.PrepareHeight - currentHeight < minPrepareGap {
        return fmt.Errorf("prepare gap too small: need at least %d blocks", minPrepareGap)
    }
    
    minPreexecGap := uint64(1000)
    if proposal.PreexecHeight - proposal.PrepareHeight < minPreexecGap {
        return fmt.Errorf("preexec gap too small: need at least %d blocks", minPreexecGap)
    }
    
    // 检查共识类型
    validConsensus := map[string]bool{
        "pot": true, "pow": true, "hotstuff": true,
        "whirly": true, "custom": true,
    }
    if !validConsensus[proposal.TargetConsensus] {
        return fmt.Errorf("invalid consensus type: %s", proposal.TargetConsensus)
    }
    
    // 自定义共识必须有 CDL
    if proposal.TargetConsensus == "custom" && proposal.DescriptorCDL == nil {
        return errors.New("custom consensus requires CDL descriptor")
    }
    
    // 验证 CDL 哈希
    if proposal.DescriptorCDL != nil {
        computedHash := proposal.DescriptorCDL.Hash()
        if !bytes.Equal(computedHash, proposal.DescriptorHash) {
            return errors.New("CDL descriptor hash mismatch")
        }
    }
    
    return nil
}
```

## 5. 双链管理机制

### 5.1 双链管理器

**文件**: `consensus/upgrade/dual_chain.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/model"
    "github.com/zzz136454872/upgradeable-consensus/types"
)

// DualChainManager 双链管理器
type DualChainManager struct {
    mainChain    *ChainState
    preexecChain *ChainState
    
    forkPoint    uint64
    active       bool
    
    storage      DualChainStorage
    log          *logrus.Entry
    mu           sync.RWMutex
}

// NewDualChainManager 创建双链管理器
func NewDualChainManager(
    mainConsensus model.Consensus,
    storage DualChainStorage,
    log *logrus.Entry,
) *DualChainManager {
    return &DualChainManager{
        mainChain: &ChainState{
            ConsensusID:   mainConsensus.GetConsensusID(),
            CurrentHeight: 0,
            Consensus:     mainConsensus,
        },
        storage: storage,
        log:     log,
        active:  false,
    }
}

// StartPreexecution 启动预执行链
func (dcm *DualChainManager) StartPreexecution(
    forkHeight uint64,
    newConsensus model.Consensus,
) error {
    dcm.mu.Lock()
    defer dcm.mu.Unlock()
    
    if dcm.active {
        return fmt.Errorf("preexecution already active")
    }
    
    // 获取分叉点状态
    forkBlock, err := dcm.storage.GetBlock(forkHeight)
    if err != nil {
        return fmt.Errorf("failed to get fork block: %w", err)
    }
    
    // 创建预执行链状态
    dcm.preexecChain = &ChainState{
        ConsensusID:   newConsensus.GetConsensusID(),
        CurrentHeight: forkHeight,
        Head:          forkBlock,
        Consensus:     newConsensus,
    }
    
    dcm.forkPoint = forkHeight
    dcm.active = true
    
    dcm.log.WithFields(logrus.Fields{
        "fork_height": forkHeight,
        "new_consensus": newConsensus.GetConsensusType(),
    }).Info("Started preexecution chain")
    
    return nil
}

// ProcessMainChainBlock 处理主链区块
func (dcm *DualChainManager) ProcessMainChainBlock(block *types.Block) error {
    dcm.mu.Lock()
    defer dcm.mu.Unlock()
    
    // 验证区块
    if err := dcm.validateBlock(dcm.mainChain, block); err != nil {
        return fmt.Errorf("invalid main chain block: %w", err)
    }
    
    // 存储区块
    if err := dcm.storage.StoreMainBlock(block); err != nil {
        return fmt.Errorf("failed to store main block: %w", err)
    }
    
    // 更新状态
    dcm.mainChain.CurrentHeight = block.Header.Height
    dcm.mainChain.Head = block
    
    // 如果预执行链活跃,同步交易
    if dcm.active {
        if err := dcm.syncTransactionsToPreexec(block); err != nil {
            dcm.log.WithError(err).Warn("Failed to sync transactions to preexec chain")
        }
    }
    
    return nil
}

// syncTransactionsToPreexec 同步交易到预执行链
func (dcm *DualChainManager) syncTransactionsToPreexec(mainBlock *types.Block) error {
    if !dcm.active || dcm.preexecChain == nil {
        return nil
    }
    
    // 过滤掉升级相关交易 (不在预执行链上执行)
    txs := dcm.filterNormalTransactions(mainBlock.Txs)
    
    // 在预执行链上处理这些交易
    // 注意: 这里需要调用新共识的出块逻辑
    // 具体实现取决于新共识的接口
    
    dcm.log.WithFields(logrus.Fields{
        "main_height": mainBlock.Header.Height,
        "tx_count":    len(txs),
    }).Debug("Synced transactions to preexec chain")
    
    return nil
}

// ProcessPreexecBlock 处理预执行链区块
func (dcm *DualChainManager) ProcessPreexecBlock(block *types.Block) error {
    dcm.mu.Lock()
    defer dcm.mu.Unlock()
    
    if !dcm.active {
        return fmt.Errorf("preexecution not active")
    }
    
    // 验证区块
    if err := dcm.validateBlock(dcm.preexecChain, block); err != nil {
        return fmt.Errorf("invalid preexec block: %w", err)
    }
    
    // 存储预执行区块
    if err := dcm.storage.StorePreexecBlock(block); err != nil {
        return fmt.Errorf("failed to store preexec block: %w", err)
    }
    
    // 更新状态
    dcm.preexecChain.CurrentHeight = block.Header.Height
    dcm.preexecChain.Head = block
    
    return nil
}

// validateBlock 验证区块
func (dcm *DualChainManager) validateBlock(chain *ChainState, block *types.Block) error {
    // 高度检查
    if block.Header.Height != chain.CurrentHeight + 1 {
        return fmt.Errorf("invalid block height: expected %d, got %d",
            chain.CurrentHeight+1, block.Header.Height)
    }
    
    // 父哈希检查
    if chain.Head != nil {
        if !bytes.Equal(block.Header.ParentHash, chain.Head.Hash()) {
            return fmt.Errorf("invalid parent hash")
        }
    }
    
    // 使用对应共识验证区块
    // (这里简化处理,实际需要调用共识的验证接口)
    
    return nil
}

// MergePreexecChain 合并预执行链到主链
func (dcm *DualChainManager) MergePreexecChain(switchHeight uint64) error {
    dcm.mu.Lock()
    defer dcm.mu.Unlock()
    
    if !dcm.active {
        return fmt.Errorf("no active preexecution")
    }
    
    dcm.log.WithFields(logrus.Fields{
        "fork_point":    dcm.forkPoint,
        "switch_height": switchHeight,
    }).Info("Merging preexec chain to main chain")
    
    // 获取预执行链从分叉点到切换点的所有区块
    preexecBlocks, err := dcm.storage.GetPreexecBlocks(dcm.forkPoint, switchHeight)
    if err != nil {
        return fmt.Errorf("failed to get preexec blocks: %w", err)
    }
    
    // 删除主链上从分叉点之后的区块 (它们将被预执行链替换)
    if err := dcm.storage.DeleteMainBlocksFrom(dcm.forkPoint + 1); err != nil {
        return fmt.Errorf("failed to delete old main blocks: %w", err)
    }
    
    // 将预执行链区块标记为主链区块
    for _, block := range preexecBlocks {
        if err := dcm.storage.PromoteToMainChain(block); err != nil {
            return fmt.Errorf("failed to promote block %d: %w", block.Header.Height, err)
        }
    }
    
    // 更新主链状态
    dcm.mainChain.ConsensusID = dcm.preexecChain.ConsensusID
    dcm.mainChain.CurrentHeight = switchHeight
    dcm.mainChain.Head = preexecBlocks[len(preexecBlocks)-1]
    dcm.mainChain.Consensus = dcm.preexecChain.Consensus
    
    // 清理预执行链
    dcm.preexecChain = nil
    dcm.active = false
    
    dcm.log.Info("Preexec chain merged successfully")
    
    return nil
}

// RollbackPreexecution 回退预执行
func (dcm *DualChainManager) RollbackPreexecution() error {
    dcm.mu.Lock()
    defer dcm.mu.Unlock()
    
    if !dcm.active {
        return fmt.Errorf("no active preexecution to rollback")
    }
    
    dcm.log.Info("Rolling back preexecution")
    
    // 停止预执行链共识
    if dcm.preexecChain != nil && dcm.preexecChain.Consensus != nil {
        dcm.preexecChain.Consensus.Stop()
    }
    
    // 删除预执行链数据
    if err := dcm.storage.DeletePreexecBlocks(dcm.forkPoint); err != nil {
        dcm.log.WithError(err).Warn("Failed to delete preexec blocks")
    }
    
    // 清理状态
    dcm.preexecChain = nil
    dcm.active = false
    dcm.forkPoint = 0
    
    dcm.log.Info("Preexecution rolled back successfully")
    
    return nil
}

// GetMainChainHeight 获取主链高度
func (dcm *DualChainManager) GetMainChainHeight() uint64 {
    dcm.mu.RLock()
    defer dcm.mu.RUnlock()
    return dcm.mainChain.CurrentHeight
}

// GetPreexecChainHeight 获取预执行链高度
func (dcm *DualChainManager) GetPreexecChainHeight() uint64 {
    dcm.mu.RLock()
    defer dcm.mu.RUnlock()
    if dcm.preexecChain == nil {
        return 0
    }
    return dcm.preexecChain.CurrentHeight
}

// IsPreexecActive 预执行是否活跃
func (dcm *DualChainManager) IsPreexecActive() bool {
    dcm.mu.RLock()
    defer dcm.mu.RUnlock()
    return dcm.active
}

// filterNormalTransactions 过滤普通交易 (排除升级交易)
func (dcm *DualChainManager) filterNormalTransactions(txs []*types.Tx) []*types.Tx {
    filtered := make([]*types.Tx, 0, len(txs))
    for _, tx := range txs {
        // 解析交易类型
        pbTx, err := types.RawTransaction(tx.Data).ToTx()
        if err != nil {
            continue
        }
        
        // 只保留普通交易
        if pbTx.Type == pb.TransactionType_NORMAL {
            filtered = append(filtered, tx)
        }
    }
    return filtered
}
```

### 5.2 双链存储接口

**文件**: `internal/storage/dual_chain_storage.go`

```go
package storage

import (
    "encoding/binary"
    "fmt"
    
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/zzz136454872/upgradeable-consensus/types"
)

// DualChainStorage 双链存储接口
type DualChainStorage interface {
    // 主链操作
    StoreMainBlock(block *types.Block) error
    GetBlock(height uint64) (*types.Block, error)
    DeleteMainBlocksFrom(height uint64) error
    
    // 预执行链操作
    StorePreexecBlock(block *types.Block) error
    GetPreexecBlock(height uint64) (*types.Block, error)
    GetPreexecBlocks(from, to uint64) ([]*types.Block, error)
    DeletePreexecBlocks(forkPoint uint64) error
    
    // 链切换操作
    PromoteToMainChain(block *types.Block) error
}

// LevelDBDualChainStorage LevelDB 实现
type LevelDBDualChainStorage struct {
    db *leveldb.DB
}

// NewLevelDBDualChainStorage 创建存储
func NewLevelDBDualChainStorage(dbPath string) (*LevelDBDualChainStorage, error) {
    db, err := leveldb.OpenFile(dbPath, nil)
    if err != nil {
        return nil, err
    }
    return &LevelDBDualChainStorage{db: db}, nil
}

// 键前缀
const (
    mainChainPrefix   = "main:"
    preexecChainPrefix = "preexec:"
)

// StoreMainBlock 存储主链区块
func (s *LevelDBDualChainStorage) StoreMainBlock(block *types.Block) error {
    key := makeKey(mainChainPrefix, block.Header.Height)
    value, err := serializeBlock(block)
    if err != nil {
        return err
    }
    return s.db.Put(key, value, nil)
}

// GetBlock 获取主链区块
func (s *LevelDBDualChainStorage) GetBlock(height uint64) (*types.Block, error) {
    key := makeKey(mainChainPrefix, height)
    value, err := s.db.Get(key, nil)
    if err != nil {
        return nil, err
    }
    return deserializeBlock(value)
}

// DeleteMainBlocksFrom 删除指定高度之后的主链区块
func (s *LevelDBDualChainStorage) DeleteMainBlocksFrom(height uint64) error {
    iter := s.db.NewIterator(nil, nil)
    defer iter.Release()
    
    prefix := []byte(mainChainPrefix)
    batch := new(leveldb.Batch)
    
    for iter.Next() {
        key := iter.Key()
        if len(key) < len(prefix) {
            continue
        }
        
        if string(key[:len(prefix)]) != mainChainPrefix {
            continue
        }
        
        h := binary.BigEndian.Uint64(key[len(prefix):])
        if h >= height {
            batch.Delete(key)
        }
    }
    
    return s.db.Write(batch, nil)
}

// StorePreexecBlock 存储预执行链区块
func (s *LevelDBDualChainStorage) StorePreexecBlock(block *types.Block) error {
    key := makeKey(preexecChainPrefix, block.Header.Height)
    value, err := serializeBlock(block)
    if err != nil {
        return err
    }
    return s.db.Put(key, value, nil)
}

// GetPreexecBlock 获取预执行链区块
func (s *LevelDBDualChainStorage) GetPreexecBlock(height uint64) (*types.Block, error) {
    key := makeKey(preexecChainPrefix, height)
    value, err := s.db.Get(key, nil)
    if err != nil {
        return nil, err
    }
    return deserializeBlock(value)
}

// GetPreexecBlocks 获取预执行链区块范围
func (s *LevelDBDualChainStorage) GetPreexecBlocks(from, to uint64) ([]*types.Block, error) {
    blocks := make([]*types.Block, 0, to-from+1)
    for h := from; h <= to; h++ {
        block, err := s.GetPreexecBlock(h)
        if err != nil {
            return nil, fmt.Errorf("failed to get block at height %d: %w", h, err)
        }
        blocks = append(blocks, block)
    }
    return blocks, nil
}

// DeletePreexecBlocks 删除预执行链
func (s *LevelDBDualChainStorage) DeletePreexecBlocks(forkPoint uint64) error {
    iter := s.db.NewIterator(nil, nil)
    defer iter.Release()
    
    prefix := []byte(preexecChainPrefix)
    batch := new(leveldb.Batch)
    
    for iter.Next() {
        key := iter.Key()
        if len(key) < len(prefix) {
            continue
        }
        
        if string(key[:len(prefix)]) == preexecChainPrefix {
            batch.Delete(key)
        }
    }
    
    return s.db.Write(batch, nil)
}

// PromoteToMainChain 将预执行链区块提升为主链区块
func (s *LevelDBDualChainStorage) PromoteToMainChain(block *types.Block) error {
    // 从预执行链删除
    preexecKey := makeKey(preexecChainPrefix, block.Header.Height)
    
    // 添加到主链
    mainKey := makeKey(mainChainPrefix, block.Header.Height)
    value, err := serializeBlock(block)
    if err != nil {
        return err
    }
    
    batch := new(leveldb.Batch)
    batch.Delete(preexecKey)
    batch.Put(mainKey, value)
    
    return s.db.Write(batch, nil)
}

// 辅助函数
func makeKey(prefix string, height uint64) []byte {
    key := make([]byte, len(prefix)+8)
    copy(key, prefix)
    binary.BigEndian.PutUint64(key[len(prefix):], height)
    return key
}

func serializeBlock(block *types.Block) ([]byte, error) {
    // 使用 protobuf 或 JSON 序列化
    // 这里简化处理
    return json.Marshal(block)
}

func deserializeBlock(data []byte) (*types.Block, error) {
    block := &types.Block{}
    err := json.Unmarshal(data, block)
    return block, err
}

// Close 关闭数据库
func (s *LevelDBDualChainStorage) Close() error {
    return s.db.Close()
}
```

## 6. 治理委员会与多签验证

**文件**: `consensus/upgrade/governance.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    
    "github.com/niclabs/tcrsa"
    "github.com/zzz136454872/upgradeable-consensus/config"
    "github.com/zzz136454872/upgradeable-consensus/crypto"
)

// GovernanceCommittee 治理委员会
type GovernanceCommittee struct {
    members   []*CommitteeMember
    threshold uint32  // 门限签名阈值
    
    // 门限签名相关
    keyMeta   *tcrsa.KeyMeta
    
    mu        sync.RWMutex
}

// CommitteeMember 委员会成员
type CommitteeMember struct {
    ID         int64
    PublicKey  *tcrsa.KeyShare
    PrivateKey *tcrsa.KeyShare  // 只有本节点成员才有
    Address    []byte
}

// NewGovernanceCommittee 创建治理委员会
func NewGovernanceCommittee(members []*CommitteeMember, threshold uint32) *GovernanceCommittee {
    return &GovernanceCommittee{
        members:   members,
        threshold: threshold,
    }
}

// InitializeThresholdKeys 初始化门限签名密钥
func (gc *GovernanceCommittee) InitializeThresholdKeys(
    keySize int,
    threshold uint16,
    numShares uint16,
) error {
    // 生成门限签名密钥
    keyShares, keyMeta, err := crypto.GenerateThresholdKeys(keySize, threshold, numShares)
    if err != nil {
        return fmt.Errorf("failed to generate threshold keys: %w", err)
    }
    
    gc.keyMeta = keyMeta
    
    // 分发密钥给成员
    for i, member := range gc.members {
        if i < len(keyShares) {
            member.PublicKey = keyMeta.PublicKey
            member.PrivateKey = keyShares[i]
        }
    }
    
    return nil
}

// SignProposal 委员会成员签名提案
func (gc *GovernanceCommittee) SignProposal(
    proposal *UpgradeProposal,
    memberID int64,
) ([]byte, error) {
    gc.mu.RLock()
    defer gc.mu.RUnlock()
    
    // 查找成员
    member := gc.getMemberByID(memberID)
    if member == nil {
        return nil, fmt.Errorf("member %d not found", memberID)
    }
    
    if member.PrivateKey == nil {
        return nil, fmt.Errorf("member %d has no private key", memberID)
    }
    
    // 计算提案哈希
    hash := proposal.Hash()
    
    // 生成部分签名
    sigShare, err := crypto.TSign(hash[:], member.PrivateKey, gc.keyMeta.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to sign: %w", err)
    }
    
    // 序列化签名
    sigBytes, err := sigShare.MarshalBinary()
    if err != nil {
        return nil, fmt.Errorf("failed to marshal signature: %w", err)
    }
    
    return sigBytes, nil
}

// AggregateSignatures 聚合门限签名
func (gc *GovernanceCommittee) AggregateSignatures(
    proposal *UpgradeProposal,
    signatures [][]byte,
) ([]byte, error) {
    if len(signatures) < int(gc.threshold) {
        return nil, fmt.Errorf("insufficient signatures: got %d, need %d",
            len(signatures), gc.threshold)
    }
    
    // 反序列化签名
    sigShares := make([]*tcrsa.SigShare, 0, len(signatures))
    for _, sigBytes := range signatures {
        sig := &tcrsa.SigShare{}
        if err := sig.UnmarshalBinary(sigBytes); err != nil {
            continue  // 跳过无效签名
        }
        sigShares = append(sigShares, sig)
    }
    
    if len(sigShares) < int(gc.threshold) {
        return nil, fmt.Errorf("too many invalid signatures")
    }
    
    // 聚合签名
    hash := proposal.Hash()
    fullSig, err := crypto.CombineIntactTSig(sigShares, hash[:], gc.keyMeta.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to aggregate signatures: %w", err)
    }
    
    return fullSig, nil
}

// VerifyAggregatedSignature 验证聚合签名
func (gc *GovernanceCommittee) VerifyAggregatedSignature(
    proposal *UpgradeProposal,
    aggregatedSig []byte,
) error {
    hash := proposal.Hash()
    
    // 验证聚合签名
    if err := crypto.VerifySign(aggregatedSig, hash[:], gc.keyMeta.PublicKey); err != nil {
        return fmt.Errorf("invalid aggregated signature: %w", err)
    }
    
    return nil
}

// VerifyPartialSignatures 验证部分签名
func (gc *GovernanceCommittee) VerifyPartialSignatures(
    proposal *UpgradeProposal,
) error {
    if len(proposal.CommitteeSignatures) < int(gc.threshold) {
        return fmt.Errorf("insufficient signatures: got %d, need %d",
            len(proposal.CommitteeSignatures), gc.threshold)
    }
    
    hash := proposal.Hash()
    validCount := 0
    
    for i, sigBytes := range proposal.CommitteeSignatures {
        // 反序列化签名
        sig := &tcrsa.SigShare{}
        if err := sig.UnmarshalBinary(sigBytes); err != nil {
            continue
        }
        
        // 验证签名
        if err := crypto.VerifyPartSig(sig, hash[:], gc.keyMeta.PublicKey); err == nil {
            validCount++
        }
    }
    
    if validCount < int(gc.threshold) {
        return fmt.Errorf("insufficient valid signatures: got %d, need %d",
            validCount, gc.threshold)
    }
    
    return nil
}

// GetMemberByID 根据 ID 获取成员
func (gc *GovernanceCommittee) getMemberByID(id int64) *CommitteeMember {
    for _, member := range gc.members {
        if member.ID == id {
            return member
        }
    }
    return nil
}

// GetMemberByPublicKey 根据公钥获取成员
func (gc *GovernanceCommittee) GetMemberByPublicKey(pubkey []byte) *CommitteeMember {
    // 简化实现,实际需要比较公钥
    for _, member := range gc.members {
        // TODO: 正确比较公钥
        return member
    }
    return nil
}

// GetPublicKeys 获取所有成员公钥
func (gc *GovernanceCommittee) GetPublicKeys() [][]byte {
    gc.mu.RLock()
    defer gc.mu.RUnlock()
    
    keys := make([][]byte, len(gc.members))
    for i, member := range gc.members {
        // 序列化公钥
        keys[i] = []byte(fmt.Sprintf("member-%d-pubkey", member.ID))
    }
    return keys
}

// GetThreshold 获取阈值
func (gc *GovernanceCommittee) GetThreshold() uint32 {
    return gc.threshold
}

// GetMemberCount 获取成员数量
func (gc *GovernanceCommittee) GetMemberCount() int {
    gc.mu.RLock()
    defer gc.mu.RUnlock()
    return len(gc.members)
}
```

## 7. CDL 引擎实现

### 7.1 CDL 描述符定义

**文件**: `consensus/upgrade/cdl/types.go`

```go
package cdl

import (
    "crypto/sha256"
    "encoding/json"
    
    "gopkg.in/yaml.v3"
)

// CDLDescriptor CDL 共识描述符
type CDLDescriptor struct {
    Name       string                 `yaml:"name"`
    Version    string                 `yaml:"version"`
    Type       string                 `yaml:"type"`
    
    Components Components             `yaml:"components"`
    Parameters map[string]interface{} `yaml:"parameters"`
    Phases     []Phase                `yaml:"phases"`
    StateMachine StateMachine         `yaml:"state_machine"`
    
    SafetyProperties        []SafetyProperty        `yaml:"safety_properties"`
    PerformanceRequirements PerformanceRequirements `yaml:"performance_requirements"`
}

// Components 密码学和网络组件
type Components struct {
    Crypto  CryptoComponents  `yaml:"crypto"`
    Network NetworkComponents `yaml:"network"`
    Storage StorageComponents `yaml:"storage"`
}

// CryptoComponents 密码学组件
type CryptoComponents struct {
    Hash         string `yaml:"hash"`           // SHA256, SHA3, Blake2b
    Signature    string `yaml:"signature"`      // ECDSA, EdDSA, BLS
    VDF          string `yaml:"vdf"`            // Wesolowski, Pietrzak
    VRF          string `yaml:"vrf"`            // ECVRF
    Commitment   string `yaml:"commitment"`     // Pedersen, KZG
    ThresholdSig string `yaml:"threshold_sig"`  // BLS-TSig, ECDSA-TSig
}

// NetworkComponents 网络组件
type NetworkComponents struct {
    Topology  string `yaml:"topology"`   // gossip, structured
    Broadcast string `yaml:"broadcast"`  // reliable, best-effort
}

// StorageComponents 存储组件
type StorageComponents struct {
    Blockchain string `yaml:"blockchain"`  // merkle-chain, dag
    State      string `yaml:"state"`       // merkle-patricia, sparse-merkle
}

// Phase 协议阶段
type Phase struct {
    Name         string        `yaml:"name"`
    Entry        string        `yaml:"entry"`
    Precondition string        `yaml:"precondition,omitempty"`
    Actions      []Action      `yaml:"actions"`
    Postcondition string       `yaml:"postcondition,omitempty"`
    Exit         string        `yaml:"exit"`
}

// Action 阶段动作
type Action struct {
    Type       string                 `yaml:"type"`  // function, broadcast, verify
    Function   string                 `yaml:"function,omitempty"`
    Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

// StateMachine 状态机
type StateMachine struct {
    States      []string     `yaml:"states"`
    Transitions []Transition `yaml:"transitions"`
}

// Transition 状态转换
type Transition struct {
    From      string `yaml:"from"`
    To        string `yaml:"to"`
    Condition string `yaml:"condition"`
    Action    string `yaml:"action"`
}

// SafetyProperty 安全属性
type SafetyProperty struct {
    Name    string `yaml:"name"`
    Formula string `yaml:"formula"`
}

// PerformanceRequirements 性能要求
type PerformanceRequirements struct {
    MinThroughput  float64 `yaml:"min_throughput"`   // tx/s
    MaxLatency     float64 `yaml:"max_latency"`      // seconds
    FaultTolerance float64 `yaml:"fault_tolerance"`  // ratio
}

// Serialize 序列化为 YAML
func (cdl *CDLDescriptor) Serialize() string {
    data, _ := yaml.Marshal(cdl)
    return string(data)
}

// Hash 计算 CDL 哈希
func (cdl *CDLDescriptor) Hash() []byte {
    data := cdl.Serialize()
    hash := sha256.Sum256([]byte(data))
    return hash[:]
}

// Validate 验证 CDL 完整性
func (cdl *CDLDescriptor) Validate() error {
    // 检查必需字段
    if cdl.Name == "" {
        return fmt.Errorf("consensus name is required")
    }
    if cdl.Version == "" {
        return fmt.Errorf("version is required")
    }
    if cdl.Type == "" {
        return fmt.Errorf("consensus type is required")
    }
    
    // 验证组件
    if err := cdl.validateComponents(); err != nil {
        return fmt.Errorf("invalid components: %w", err)
    }
    
    // 验证阶段
    if len(cdl.Phases) == 0 {
        return fmt.Errorf("at least one phase is required")
    }
    
    // 验证状态机
    if err := cdl.validateStateMachine(); err != nil {
        return fmt.Errorf("invalid state machine: %w", err)
    }
    
    return nil
}

func (cdl *CDLDescriptor) validateComponents() error {
    // 验证哈希算法
    validHash := map[string]bool{
        "SHA256": true, "SHA3": true, "Blake2b": true,
    }
    if !validHash[cdl.Components.Crypto.Hash] {
        return fmt.Errorf("unsupported hash algorithm: %s", cdl.Components.Crypto.Hash)
    }
    
    // 验证签名算法
    validSig := map[string]bool{
        "ECDSA": true, "EdDSA": true, "BLS": true,
    }
    if !validSig[cdl.Components.Crypto.Signature] {
        return fmt.Errorf("unsupported signature algorithm: %s", cdl.Components.Crypto.Signature)
    }
    
    return nil
}

func (cdl *CDLDescriptor) validateStateMachine() error {
    if len(cdl.StateMachine.States) == 0 {
        return fmt.Errorf("state machine must have at least one state")
    }
    
    // 验证转换引用的状态都存在
    stateSet := make(map[string]bool)
    for _, state := range cdl.StateMachine.States {
        stateSet[state] = true
    }
    
    for _, trans := range cdl.StateMachine.Transitions {
        if !stateSet[trans.From] {
            return fmt.Errorf("unknown state in transition: %s", trans.From)
        }
        if !stateSet[trans.To] {
            return fmt.Errorf("unknown state in transition: %s", trans.To)
        }
    }
    
    return nil
}
```

### 7.2 CDL 解析器

**文件**: `consensus/upgrade/cdl/parser.go`

```go
package cdl

import (
    "fmt"
    "os"
    
    "gopkg.in/yaml.v3"
)

// Parser CDL 解析器
type Parser struct{}

// NewParser 创建解析器
func NewParser() *Parser {
    return &Parser{}
}

// Parse 解析 YAML 格式的 CDL
func (p *Parser) Parse(yamlContent string) (*CDLDescriptor, error) {
    cdl := &CDLDescriptor{}
    
    if err := yaml.Unmarshal([]byte(yamlContent), cdl); err != nil {
        return nil, fmt.Errorf("failed to parse CDL: %w", err)
    }
    
    return cdl, nil
}

// ParseFile 从文件解析 CDL
func (p *Parser) ParseFile(filename string) (*CDLDescriptor, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    return p.Parse(string(data))
}

// ParseCDL 快捷函数:解析 CDL
func ParseCDL(yamlContent string) *CDLDescriptor {
    parser := NewParser()
    cdl, err := parser.Parse(yamlContent)
    if err != nil {
        panic(fmt.Sprintf("failed to parse CDL: %v", err))
    }
    return cdl
}
```

### 7.3 CDL 验证器

**文件**: `consensus/upgrade/cdl/validator.go`

```go
package cdl

import (
    "fmt"
)

// Validator CDL 验证器
type Validator struct {
    supportedComponents map[string][]string
}

// NewValidator 创建验证器
func NewValidator() *Validator {
    return &Validator{
        supportedComponents: map[string][]string{
            "hash":          {"SHA256", "SHA3", "Blake2b"},
            "signature":     {"ECDSA", "EdDSA", "BLS"},
            "vdf":           {"Wesolowski", "Pietrzak"},
            "vrf":           {"ECVRF"},
            "commitment":    {"Pedersen", "KZG"},
            "threshold_sig": {"BLS-TSig", "ECDSA-TSig"},
            "topology":      {"gossip", "structured"},
            "broadcast":     {"reliable", "best-effort"},
            "blockchain":    {"merkle-chain", "dag"},
            "state":         {"merkle-patricia", "sparse-merkle"},
        },
    }
}

// Validate 完整验证 CDL
func (v *Validator) Validate(cdl *CDLDescriptor) error {
    // 基本验证
    if err := cdl.Validate(); err != nil {
        return err
    }
    
    // 组件兼容性验证
    if err := v.validateComponentCompatibility(cdl); err != nil {
        return fmt.Errorf("component compatibility check failed: %w", err)
    }
    
    // 状态机正确性验证
    if err := v.validateStateMachineCorrectness(cdl); err != nil {
        return fmt.Errorf("state machine correctness check failed: %w", err)
    }
    
    // 安全属性可满足性验证
    if err := v.validateSafetyProperties(cdl); err != nil {
        return fmt.Errorf("safety properties check failed: %w", err)
    }
    
    // 资源约束验证
    if err := v.validateResourceConstraints(cdl); err != nil {
        return fmt.Errorf("resource constraints check failed: %w", err)
    }
    
    return nil
}

// validateComponentCompatibility 验证组件兼容性
func (v *Validator) validateComponentCompatibility(cdl *CDLDescriptor) error {
    // 验证哈希算法
    if !v.isSupported("hash", cdl.Components.Crypto.Hash) {
        return fmt.Errorf("unsupported hash algorithm: %s", cdl.Components.Crypto.Hash)
    }
    
    // 验证签名算法
    if !v.isSupported("signature", cdl.Components.Crypto.Signature) {
        return fmt.Errorf("unsupported signature algorithm: %s", cdl.Components.Crypto.Signature)
    }
    
    // 验证 VDF (如果使用)
    if cdl.Components.Crypto.VDF != "" && !v.isSupported("vdf", cdl.Components.Crypto.VDF) {
        return fmt.Errorf("unsupported VDF: %s", cdl.Components.Crypto.VDF)
    }
    
    // 验证 VRF (如果使用)
    if cdl.Components.Crypto.VRF != "" && !v.isSupported("vrf", cdl.Components.Crypto.VRF) {
        return fmt.Errorf("unsupported VRF: %s", cdl.Components.Crypto.VRF)
    }
    
    return nil
}

// validateStateMachineCorrectness 验证状态机正确性
func (v *Validator) validateStateMachineCorrectness(cdl *CDLDescriptor) error {
    sm := cdl.StateMachine
    
    // 检查是否有初始状态
    hasInitialState := false
    for _, state := range sm.States {
        if state == "IDLE" || state == "INIT" {
            hasInitialState = true
            break
        }
    }
    if !hasInitialState {
        return fmt.Errorf("state machine must have an initial state (IDLE or INIT)")
    }
    
    // 检查死锁: 每个状态至少有一个出边
    stateOutgoing := make(map[string]int)
    for _, state := range sm.States {
        stateOutgoing[state] = 0
    }
    
    for _, trans := range sm.Transitions {
        stateOutgoing[trans.From]++
    }
    
    for state, count := range stateOutgoing {
        if count == 0 && state != "IDLE" {
            return fmt.Errorf("state %s has no outgoing transitions (potential deadlock)", state)
        }
    }
    
    // 检查可达性: 所有状态从初始状态可达
    reachable := v.computeReachableStates(sm)
    for _, state := range sm.States {
        if !reachable[state] && state != "IDLE" {
            return fmt.Errorf("state %s is not reachable from initial state", state)
        }
    }
    
    return nil
}

// computeReachableStates 计算从初始状态可达的所有状态
func (v *Validator) computeReachableStates(sm StateMachine) map[string]bool {
    reachable := make(map[string]bool)
    reachable["IDLE"] = true
    
    changed := true
    for changed {
        changed = false
        for _, trans := range sm.Transitions {
            if reachable[trans.From] && !reachable[trans.To] {
                reachable[trans.To] = true
                changed = true
            }
        }
    }
    
    return reachable
}

// validateSafetyProperties 验证安全属性
func (v *Validator) validateSafetyProperties(cdl *CDLDescriptor) error {
    // 至少应该声明基本的安全属性
    requiredProperties := []string{"agreement", "validity"}
    
    declaredProps := make(map[string]bool)
    for _, prop := range cdl.SafetyProperties {
        declaredProps[prop.Name] = true
    }
    
    for _, required := range requiredProperties {
        if !declaredProps[required] {
            return fmt.Errorf("missing required safety property: %s", required)
        }
    }
    
    return nil
}

// validateResourceConstraints 验证资源约束
func (v *Validator) validateResourceConstraints(cdl *CDLDescriptor) error {
    // 检查阶段数量 (防止过于复杂)
    if len(cdl.Phases) > 20 {
        return fmt.Errorf("too many phases: %d (max 20)", len(cdl.Phases))
    }
    
    // 检查状态数量
    if len(cdl.StateMachine.States) > 50 {
        return fmt.Errorf("too many states: %d (max 50)", len(cdl.StateMachine.States))
    }
    
    // 检查性能要求是否合理
    perf := cdl.PerformanceRequirements
    if perf.MinThroughput > 1000000 {
        return fmt.Errorf("unrealistic throughput requirement: %.0f tx/s", perf.MinThroughput)
    }
    if perf.MaxLatency < 0.001 {
        return fmt.Errorf("unrealistic latency requirement: %.3f s", perf.MaxLatency)
    }
    if perf.FaultTolerance > 0.5 {
        return fmt.Errorf("unrealistic fault tolerance: %.2f (max 0.5)", perf.FaultTolerance)
    }
    
    return nil
}

// isSupported 检查组件是否支持
func (v *Validator) isSupported(componentType, value string) bool {
    supported, ok := v.supportedComponents[componentType]
    if !ok {
        return false
    }
    
    for _, s := range supported {
        if s == value {
            return true
        }
    }
    return false
}
```

### 7.4 CDL 编译器

**文件**: `consensus/upgrade/cdl/compiler.go`

```go
package cdl

import (
    "fmt"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/config"
    "github.com/zzz136454872/upgradeable-consensus/consensus/model"
    "github.com/zzz136454872/upgradeable-consensus/executor"
    "github.com/zzz136454872/upgradeable-consensus/p2p"
)

// Compiler CDL 编译器
type Compiler struct {
    log *logrus.Entry
}

// NewCompiler 创建编译器
func NewCompiler(log *logrus.Entry) *Compiler {
    return &Compiler{log: log}
}

// Compile 编译 CDL 为可执行的共识实例
func (c *Compiler) Compile(
    cdl *CDLDescriptor,
    nid int64,
    cid int64,
    cfg *config.ConsensusConfig,
    exec executor.Executor,
    p2pAdaptor p2p.P2PAdaptor,
) (model.Consensus, error) {
    
    c.log.WithFields(logrus.Fields{
        "consensus": cdl.Name,
        "version":   cdl.Version,
        "type":      cdl.Type,
    }).Info("Compiling CDL consensus")
    
    // 创建运行时环境
    runtime := NewCDLRuntime(cdl, nid, cid, cfg, exec, p2pAdaptor, c.log)
    
    // 初始化密码学组件
    if err := runtime.InitializeCryptoComponents(); err != nil {
        return nil, fmt.Errorf("failed to initialize crypto components: %w", err)
    }
    
    // 初始化网络组件
    if err := runtime.InitializeNetworkComponents(); err != nil {
        return nil, fmt.Errorf("failed to initialize network components: %w", err)
    }
    
    // 编译阶段和状态机
    if err := runtime.CompilePhases(); err != nil {
        return nil, fmt.Errorf("failed to compile phases: %w", err)
    }
    
    if err := runtime.CompileStateMachine(); err != nil {
        return nil, fmt.Errorf("failed to compile state machine: %w", err)
    }
    
    c.log.Info("CDL consensus compiled successfully")
    
    return runtime, nil
}
```

### 7.5 CDL 运行时

**文件**: `consensus/upgrade/cdl/runtime.go`

```go
package cdl

import (
    "fmt"
    "sync"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/config"
    "github.com/zzz136454872/upgradeable-consensus/consensus/model"
    "github.com/zzz136454872/upgradeable-consensus/crypto"
    "github.com/zzz136454872/upgradeable-consensus/executor"
    "github.com/zzz136454872/upgradeable-consensus/p2p"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// CDLRuntime CDL 运行时环境
type CDLRuntime struct {
    cdl        *CDLDescriptor
    nid        int64
    cid        int64
    cfg        *config.ConsensusConfig
    exec       executor.Executor
    p2pAdaptor p2p.P2PAdaptor
    log        *logrus.Entry
    
    // 运行时状态
    currentState string
    stateMu      sync.RWMutex
    
    // 密码学组件
    hashFunc      crypto.HashFunc
    signFunc      crypto.SignFunc
    verifyFunc    crypto.VerifyFunc
    
    // 消息通道
    msgEntrance chan []byte
    reqEntrance chan *pb.Request
    
    // 控制
    stopChan chan struct{}
    wg       sync.WaitGroup
}

// NewCDLRuntime 创建 CDL 运行时
func NewCDLRuntime(
    cdl *CDLDescriptor,
    nid int64,
    cid int64,
    cfg *config.ConsensusConfig,
    exec executor.Executor,
    p2pAdaptor p2p.P2PAdaptor,
    log *logrus.Entry,
) *CDLRuntime {
    return &CDLRuntime{
        cdl:          cdl,
        nid:          nid,
        cid:          cid,
        cfg:          cfg,
        exec:         exec,
        p2pAdaptor:   p2pAdaptor,
        log:          log,
        currentState: "IDLE",
        msgEntrance:  make(chan []byte, 100),
        reqEntrance:  make(chan *pb.Request, 100),
        stopChan:     make(chan struct{}),
    }
}

// InitializeCryptoComponents 初始化密码学组件
func (rt *CDLRuntime) InitializeCryptoComponents() error {
    // 根据 CDL 指定的算法初始化密码学函数
    switch rt.cdl.Components.Crypto.Hash {
    case "SHA256":
        rt.hashFunc = crypto.SHA256Hash
    case "SHA3":
        rt.hashFunc = crypto.SHA3Hash
    case "Blake2b":
        rt.hashFunc = crypto.Blake2bHash
    default:
        return fmt.Errorf("unsupported hash algorithm: %s", rt.cdl.Components.Crypto.Hash)
    }
    
    switch rt.cdl.Components.Crypto.Signature {
    case "ECDSA":
        rt.signFunc = crypto.ECDSASign
        rt.verifyFunc = crypto.ECDSAVerify
    case "EdDSA":
        rt.signFunc = crypto.EdDSASign
        rt.verifyFunc = crypto.EdDSAVerify
    case "BLS":
        rt.signFunc = crypto.BLSSign
        rt.verifyFunc = crypto.BLSVerify
    default:
        return fmt.Errorf("unsupported signature algorithm: %s", rt.cdl.Components.Crypto.Signature)
    }
    
    rt.log.WithFields(logrus.Fields{
        "hash": rt.cdl.Components.Crypto.Hash,
        "sig":  rt.cdl.Components.Crypto.Signature,
    }).Info("Crypto components initialized")
    
    return nil
}

// InitializeNetworkComponents 初始化网络组件
func (rt *CDLRuntime) InitializeNetworkComponents() error {
    // 设置 P2P 接收器
    rt.p2pAdaptor.SetReceiver(rt.msgEntrance)
    rt.p2pAdaptor.Subscribe([]byte("consensus"))
    
    rt.log.WithField("topology", rt.cdl.Components.Network.Topology).Info("Network components initialized")
    
    return nil
}

// CompilePhases 编译协议阶段
func (rt *CDLRuntime) CompilePhases() error {
    rt.log.WithField("phase_count", len(rt.cdl.Phases)).Info("Compiling phases")
    
    // 验证每个阶段的动作
    for _, phase := range rt.cdl.Phases {
        for _, action := range phase.Actions {
            if err := rt.validateAction(action); err != nil {
                return fmt.Errorf("invalid action in phase %s: %w", phase.Name, err)
            }
        }
    }
    
    return nil
}

// CompileStateMachine 编译状态机
func (rt *CDLRuntime) CompileStateMachine() error {
    rt.log.WithField("state_count", len(rt.cdl.StateMachine.States)).Info("Compiling state machine")
    
    // 状态机已在 CDL 中定义,这里只需验证
    return nil
}

// validateAction 验证动作
func (rt *CDLRuntime) validateAction(action Action) error {
    validTypes := map[string]bool{
        "function": true, "broadcast": true, "verify": true,
        "select_txs": true, "construct_block": true, "solve_puzzle": true,
    }
    
    if !validTypes[action.Type] {
        return fmt.Errorf("unknown action type: %s", action.Type)
    }
    
    return nil
}

// 实现 model.Consensus 接口
func (rt *CDLRuntime) GetConsensusID() int64 {
    return rt.cid
}

func (rt *CDLRuntime) GetConsensusType() string {
    return rt.cdl.Name
}

func (rt *CDLRuntime) GetRequestEntrance() chan<- *pb.Request {
    return rt.reqEntrance
}

func (rt *CDLRuntime) GetMsgByteEntrance() chan<- []byte {
    return rt.msgEntrance
}

func (rt *CDLRuntime) Stop() {
    close(rt.stopChan)
    rt.wg.Wait()
}

func (rt *CDLRuntime) VerifyBlock(block []byte, proof []byte) bool {
    // 根据 CDL 定义验证区块
    // 简化实现
    return true
}

func (rt *CDLRuntime) UpdateExternalStatus(status model.ExternalStatus) {}

func (rt *CDLRuntime) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {}

func (rt *CDLRuntime) RequestLatestBlock(epoch int64, proof []byte, committee []string) {}

func (rt *CDLRuntime) GetWeight(nid int64) float64 {
    // 简化实现: 所有节点权重相等
    return 1.0 / float64(len(rt.cfg.Nodes))
}

func (rt *CDLRuntime) GetMaxAdversaryWeight() float64 {
    return rt.cdl.PerformanceRequirements.FaultTolerance
}

// Run 运行 CDL 共识
func (rt *CDLRuntime) Run() {
    rt.wg.Add(1)
    defer rt.wg.Done()
    
    rt.log.Info("CDL runtime started")
    
    for {
        select {
        case <-rt.stopChan:
            rt.log.Info("CDL runtime stopped")
            return
            
        case msg := <-rt.msgEntrance:
            rt.handleMessage(msg)
            
        case req := <-rt.reqEntrance:
            rt.handleRequest(req)
        }
    }
}

// handleMessage 处理消息
func (rt *CDLRuntime) handleMessage(msg []byte) {
    rt.log.WithField("size", len(msg)).Debug("Received message")
    // 根据 CDL 定义处理消息
}

// handleRequest 处理请求
func (rt *CDLRuntime) handleRequest(req *pb.Request) {
    rt.log.WithField("type", req.Type).Debug("Received request")
    // 根据 CDL 定义处理请求
}

// transitionState 状态转换
func (rt *CDLRuntime) transitionState(event string) error {
    rt.stateMu.Lock()
    defer rt.stateMu.Unlock()
    
    currentState := rt.currentState
    
    // 查找匹配的转换
    for _, trans := range rt.cdl.StateMachine.Transitions {
        if trans.From == currentState {
            // 简化: 假设事件匹配条件
            rt.currentState = trans.To
            rt.log.WithFields(logrus.Fields{
                "from": currentState,
                "to":   trans.To,
            }).Info("State transition")
            return nil
        }
    }
    
    return fmt.Errorf("no valid transition from state %s", currentState)
}
```

## 8. 预执行与性能监控

### 8.1 性能指标收集器

**文件**: `consensus/upgrade/metrics.go`

```go
package upgrade

import (
    "sync"
    "time"
    
    "github.com/sirupsen/logrus"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// MetricsCollector 性能指标收集器
type MetricsCollector struct {
    proposalID    types.TxHash
    startHeight   uint64
    currentHeight uint64
    
    blockMetrics  []*BlockMetrics
    
    mu            sync.RWMutex
    log           *logrus.Entry
}

// BlockMetrics 单个区块的指标
type BlockMetrics struct {
    Height        uint64
    Timestamp     time.Time
    BlockTime     time.Duration
    TxCount       int
    ThroughPut    float64
    Latency       time.Duration
    Error         error
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(
    proposalID types.TxHash,
    startHeight uint64,
    log *logrus.Entry,
) *MetricsCollector {
    return &MetricsCollector{
        proposalID:   proposalID,
        startHeight:  startHeight,
        currentHeight: startHeight,
        blockMetrics: make([]*BlockMetrics, 0),
        log:          log,
    }
}

// RecordBlock 记录区块指标
func (mc *MetricsCollector) RecordBlock(
    height uint64,
    blockTime time.Duration,
    txCount int,
    err error,
) {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    
    metrics := &BlockMetrics{
        Height:     height,
        Timestamp:  time.Now(),
        BlockTime:  blockTime,
        TxCount:    txCount,
        ThroughPut: float64(txCount) / blockTime.Seconds(),
        Latency:    blockTime,
        Error:      err,
    }
    
    mc.blockMetrics = append(mc.blockMetrics, metrics)
    mc.currentHeight = height
    
    mc.log.WithFields(logrus.Fields{
        "height":     height,
        "block_time": blockTime,
        "tx_count":   txCount,
        "throughput": metrics.ThroughPut,
    }).Debug("Block metrics recorded")
}

// ComputeAggregateMetrics 计算聚合指标
func (mc *MetricsCollector) ComputeAggregateMetrics() *PerformanceMetrics {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    
    if len(mc.blockMetrics) == 0 {
        return &PerformanceMetrics{}
    }
    
    metrics := &PerformanceMetrics{
        StartHeight: mc.startHeight,
        EndHeight:   mc.currentHeight,
        BlockTimes:  make([]time.Duration, len(mc.blockMetrics)),
        Throughputs: make([]float64, len(mc.blockMetrics)),
        Latencies:   make([]time.Duration, len(mc.blockMetrics)),
        Errors:      make([]error, len(mc.blockMetrics)),
    }
    
    var totalBlockTime time.Duration
    var totalThroughput float64
    var totalLatency time.Duration
    errorCount := 0
    
    for i, bm := range mc.blockMetrics {
        metrics.BlockTimes[i] = bm.BlockTime
        metrics.Throughputs[i] = bm.ThroughPut
        metrics.Latencies[i] = bm.Latency
        metrics.Errors[i] = bm.Error
        
        totalBlockTime += bm.BlockTime
        totalThroughput += bm.ThroughPut
        totalLatency += bm.Latency
        
        if bm.Error != nil {
            errorCount++
        }
    }
    
    count := len(mc.blockMetrics)
    metrics.AvgBlockTime = totalBlockTime / time.Duration(count)
    metrics.AvgThroughput = totalThroughput / float64(count)
    metrics.AvgLatency = totalLatency / time.Duration(count)
    metrics.ErrorRate = float64(errorCount) / float64(count)
    
    return metrics
}

// GetMetricsProto 获取 protobuf 格式的指标
func (mc *MetricsCollector) GetMetricsProto() *pb.ExecutionMetrics {
    aggregate := mc.ComputeAggregateMetrics()
    return aggregate.ToProto()
}

// EvaluateAgainstCondition 根据回退条件评估
func (mc *MetricsCollector) EvaluateAgainstCondition(
    condition *pb.RollbackCondition,
    baselineBlockTime time.Duration,
    baselineThroughput float64,
) (bool, string) {
    
    metrics := mc.ComputeAggregateMetrics()
    
    // 检查错误率
    if metrics.ErrorRate > condition.MaxErrorRate {
        return false, fmt.Sprintf("Error rate %.2f%% exceeds limit %.2f%%",
            metrics.ErrorRate*100, condition.MaxErrorRate*100)
    }
    
    // 检查区块时间
    blockTimeRatio := float64(metrics.AvgBlockTime) / float64(baselineBlockTime)
    if blockTimeRatio > condition.MaxBlockTimeRatio {
        return false, fmt.Sprintf("Block time ratio %.2f exceeds limit %.2f",
            blockTimeRatio, condition.MaxBlockTimeRatio)
    }
    
    // 检查吞吐量
    throughputRatio := metrics.AvgThroughput / baselineThroughput
    if throughputRatio < condition.MinThroughputRatio {
        return false, fmt.Sprintf("Throughput ratio %.2f below limit %.2f",
            throughputRatio, condition.MinThroughputRatio)
    }
    
    // 检查超时
    if mc.currentHeight - mc.startHeight > condition.TimeoutBlocks {
        return false, fmt.Sprintf("Execution timeout: %d blocks processed",
            mc.currentHeight - mc.startHeight)
    }
    
    return true, "All metrics within acceptable range"
}

// Reset 重置收集器
func (mc *MetricsCollector) Reset() {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    
    mc.blockMetrics = make([]*BlockMetrics, 0)
    mc.currentHeight = mc.startHeight
}

// GetCurrentHeight 获取当前高度
func (mc *MetricsCollector) GetCurrentHeight() uint64 {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    return mc.currentHeight
}

// GetBlockCount 获取已记录的区块数
func (mc *MetricsCollector) GetBlockCount() int {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    return len(mc.blockMetrics)
}
```

### 8.2 预执行监控器

**文件**: `consensus/upgrade/preexec_monitor.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/sirupsen/logrus"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// PreexecMonitor 预执行监控器
type PreexecMonitor struct {
    proposal       *UpgradeProposal
    dualChain      *DualChainManager
    collector      *MetricsCollector
    
    // 基准指标 (来自主链)
    baselineBlockTime  time.Duration
    baselineThroughput float64
    
    // 监控状态
    running        bool
    checkInterval  time.Duration
    
    mu             sync.RWMutex
    log            *logrus.Entry
    
    // 异常回调
    onAnomaly      func(reason string)
}

// NewPreexecMonitor 创建预执行监控器
func NewPreexecMonitor(
    proposal *UpgradeProposal,
    dualChain *DualChainManager,
    log *logrus.Entry,
) *PreexecMonitor {
    return &PreexecMonitor{
        proposal:      proposal,
        dualChain:     dualChain,
        collector:     NewMetricsCollector(proposal.ProposalID, proposal.PrepareHeight, log),
        checkInterval: 10 * time.Second,
        log:           log,
    }
}

// SetBaseline 设置基准指标
func (pm *PreexecMonitor) SetBaseline(blockTime time.Duration, throughput float64) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    pm.baselineBlockTime = blockTime
    pm.baselineThroughput = throughput
    
    pm.log.WithFields(logrus.Fields{
        "baseline_block_time": blockTime,
        "baseline_throughput": throughput,
    }).Info("Baseline metrics set")
}

// SetAnomalyCallback 设置异常回调
func (pm *PreexecMonitor) SetAnomalyCallback(callback func(reason string)) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    pm.onAnomaly = callback
}

// Start 启动监控
func (pm *PreexecMonitor) Start() {
    pm.mu.Lock()
    if pm.running {
        pm.mu.Unlock()
        return
    }
    pm.running = true
    pm.mu.Unlock()
    
    go pm.monitorLoop()
    
    pm.log.Info("Preexecution monitor started")
}

// Stop 停止监控
func (pm *PreexecMonitor) Stop() {
    pm.mu.Lock()
    pm.running = false
    pm.mu.Unlock()
    
    pm.log.Info("Preexecution monitor stopped")
}

// monitorLoop 监控循环
func (pm *PreexecMonitor) monitorLoop() {
    ticker := time.NewTicker(pm.checkInterval)
    defer ticker.Stop()
    
    for {
        pm.mu.RLock()
        running := pm.running
        pm.mu.RUnlock()
        
        if !running {
            break
        }
        
        <-ticker.C
        pm.checkMetrics()
    }
}

// checkMetrics 检查指标
func (pm *PreexecMonitor) checkMetrics() {
    // 评估当前指标
    pass, reason := pm.collector.EvaluateAgainstCondition(
        pm.proposal.RollbackCondition,
        pm.baselineBlockTime,
        pm.baselineThroughput,
    )
    
    if !pass {
        pm.log.WithField("reason", reason).Warn("Preexecution metrics anomaly detected")
        
        // 触发异常回调
        pm.mu.RLock()
        callback := pm.onAnomaly
        pm.mu.RUnlock()
        
        if callback != nil {
            callback(reason)
        }
    }
    
    // 记录定期报告
    metrics := pm.collector.ComputeAggregateMetrics()
    pm.log.WithFields(logrus.Fields{
        "avg_block_time": metrics.AvgBlockTime,
        "avg_throughput": metrics.AvgThroughput,
        "error_rate":     metrics.ErrorRate,
        "blocks":         pm.collector.GetBlockCount(),
    }).Debug("Preexecution metrics report")
}

// RecordPreexecBlock 记录预执行链区块
func (pm *PreexecMonitor) RecordPreexecBlock(
    height uint64,
    blockTime time.Duration,
    txCount int,
    err error,
) {
    pm.collector.RecordBlock(height, blockTime, txCount, err)
}

// GetMetrics 获取当前指标
func (pm *PreexecMonitor) GetMetrics() *pb.ExecutionMetrics {
    return pm.collector.GetMetricsProto()
}

// IsHealthy 预执行链是否健康
func (pm *PreexecMonitor) IsHealthy() bool {
    pass, _ := pm.collector.EvaluateAgainstCondition(
        pm.proposal.RollbackCondition,
        pm.baselineBlockTime,
        pm.baselineThroughput,
    )
    return pass
}
```

## 9. 切换与回退机制

### 9.1 切换管理器

**文件**: `consensus/upgrade/switch.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/model"
)

// SwitchManager 切换管理器
type SwitchManager struct {
    dualChain     *DualChainManager
    monitor       *PreexecMonitor
    
    switchHeight  uint64
    switched      bool
    
    mu            sync.RWMutex
    log           *logrus.Entry
}

// NewSwitchManager 创建切换管理器
func NewSwitchManager(
    dualChain *DualChainManager,
    monitor *PreexecMonitor,
    log *logrus.Entry,
) *SwitchManager {
    return &SwitchManager{
        dualChain: dualChain,
        monitor:   monitor,
        log:       log,
    }
}

// PrepareSwitc 准备切换
func (sm *SwitchManager) PrepareSwitch(switchHeight uint64) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    if sm.switched {
        return fmt.Errorf("already switched")
    }
    
    // 验证切换高度
    mainHeight := sm.dualChain.GetMainChainHeight()
    preexecHeight := sm.dualChain.GetPreexecChainHeight()
    
    if switchHeight <= mainHeight {
        return fmt.Errorf("switch height must be in the future")
    }
    
    if switchHeight > preexecHeight {
        return fmt.Errorf("preexec chain hasn't reached switch height")
    }
    
    // 检查预执行链健康状态
    if !sm.monitor.IsHealthy() {
        return fmt.Errorf("preexec chain is not healthy")
    }
    
    sm.switchHeight = switchHeight
    
    sm.log.WithField("switch_height", switchHeight).Info("Switch prepared")
    
    return nil
}

// ExecuteSwitch 执行切换
func (sm *SwitchManager) ExecuteSwitch() error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    if sm.switched {
        return fmt.Errorf("already switched")
    }
    
    sm.log.WithField("switch_height", sm.switchHeight).Info("Executing consensus switch")
    
    startTime := time.Now()
    
    // 步骤 1: 等待达到切换高度
    for {
        mainHeight := sm.dualChain.GetMainChainHeight()
        if mainHeight >= sm.switchHeight {
            break
        }
        time.Sleep(100 * time.Millisecond)
    }
    
    sm.log.Info("Reached switch height, starting atomic switch")
    
    // 步骤 2: 停止旧共识
    // (这部分由 DualChainManager 内部处理)
    
    // 步骤 3: 合并预执行链
    if err := sm.dualChain.MergePreexecChain(sm.switchHeight); err != nil {
        return fmt.Errorf("failed to merge preexec chain: %w", err)
    }
    
    // 步骤 4: 标记切换完成
    sm.switched = true
    
    elapsed := time.Since(startTime)
    sm.log.WithFields(logrus.Fields{
        "switch_height": sm.switchHeight,
        "elapsed":       elapsed,
    }).Info("Consensus switch completed successfully")
    
    return nil
}

// IsSwitched 是否已切换
func (sm *SwitchManager) IsSwitched() bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.switched
}

// GetSwitchHeight 获取切换高度
func (sm *SwitchManager) GetSwitchHeight() uint64 {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.switchHeight
}

// WaitForSwitch 等待切换完成
func (sm *SwitchManager) WaitForSwitch(timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for {
        if sm.IsSwitched() {
            return nil
        }
        
        if time.Now().After(deadline) {
            return fmt.Errorf("switch timeout after %v", timeout)
        }
        
        time.Sleep(100 * time.Millisecond)
    }
}
```

### 9.2 回退管理器

**文件**: `consensus/upgrade/rollback.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/sirupsen/logrus"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// RollbackManager 回退管理器
type RollbackManager struct {
    dualChain    *DualChainManager
    monitor      *PreexecMonitor
    
    rollbackExecuted bool
    rollbackReason   string
    
    mu           sync.RWMutex
    log          *logrus.Entry
}

// NewRollbackManager 创建回退管理器
func NewRollbackManager(
    dualChain *DualChainManager,
    monitor *PreexecMonitor,
    log *logrus.Entry,
) *RollbackManager {
    return &RollbackManager{
        dualChain: dualChain,
        monitor:   monitor,
        log:       log,
    }
}

// CheckRollbackConditions 检查是否需要回退
func (rm *RollbackManager) CheckRollbackConditions() (bool, string) {
    // 检查预执行链健康状态
    if !rm.monitor.IsHealthy() {
        metrics := rm.monitor.GetMetrics()
        return true, fmt.Sprintf("Preexec chain unhealthy: error_rate=%.2f%%",
            metrics.ErrorRate*100)
    }
    
    // 检查是否存在严重错误
    // (可以从 monitor 获取更详细的错误信息)
    
    return false, ""
}

// ExecuteRollback 执行回退
func (rm *RollbackManager) ExecuteRollback(reason string) error {
    rm.mu.Lock()
    defer rm.mu.Unlock()
    
    if rm.rollbackExecuted {
        return fmt.Errorf("rollback already executed")
    }
    
    rm.log.WithField("reason", reason).Warn("Executing rollback")
    
    startTime := time.Now()
    
    // 步骤 1: 停止监控
    rm.monitor.Stop()
    
    // 步骤 2: 回退预执行链
    if err := rm.dualChain.RollbackPreexecution(); err != nil {
        return fmt.Errorf("failed to rollback preexecution: %w", err)
    }
    
    // 步骤 3: 记录回退事件
    rm.rollbackExecuted = true
    rm.rollbackReason = reason
    
    elapsed := time.Since(startTime)
    rm.log.WithFields(logrus.Fields{
        "reason":  reason,
        "elapsed": elapsed,
    }).Info("Rollback completed")
    
    return nil
}

// IsRolledBack 是否已回退
func (rm *RollbackManager) IsRolledBack() bool {
    rm.mu.RLock()
    defer rm.mu.RUnlock()
    return rm.rollbackExecuted
}

// GetRollbackReason 获取回退原因
func (rm *RollbackManager) GetRollbackReason() string {
    rm.mu.RLock()
    defer rm.mu.RUnlock()
    return rm.rollbackReason
}

// AutoRollbackOnAnomaly 检测到异常时自动回退
func (rm *RollbackManager) AutoRollbackOnAnomaly() {
    // 设置监控器的异常回调
    rm.monitor.SetAnomalyCallback(func(reason string) {
        rm.log.WithField("reason", reason).Warn("Anomaly detected, triggering auto-rollback")
        
        if err := rm.ExecuteRollback(reason); err != nil {
            rm.log.WithError(err).Error("Auto-rollback failed")
        }
    })
}

// CreateRollbackReport 创建回退报告
func (rm *RollbackManager) CreateRollbackReport() *RollbackReport {
    rm.mu.RLock()
    defer rm.mu.RUnlock()
    
    return &RollbackReport{
        Executed:  rm.rollbackExecuted,
        Reason:    rm.rollbackReason,
        Timestamp: time.Now(),
        Metrics:   rm.monitor.GetMetrics(),
    }
}

// RollbackReport 回退报告
type RollbackReport struct {
    Executed  bool
    Reason    string
    Timestamp time.Time
    Metrics   *pb.ExecutionMetrics
}
```

### 9.3 升级管理器 (总控制器)

**文件**: `consensus/upgrade/manager.go`

```go
package upgrade

import (
    "fmt"
    "sync"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/config"
    "github.com/zzz136454872/upgradeable-consensus/consensus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/model"
    pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
    "github.com/zzz136454872/upgradeable-consensus/types"
)

// UpgradeManager 升级协议总管理器
type UpgradeManager struct {
    // 基础组件
    uc         *consensus.UpgradeableConsensus
    cfg        *config.ConsensusConfig
    log        *logrus.Entry
    
    // 治理
    committee  *GovernanceCommittee
    
    // 当前升级状态
    currentProposal *UpgradeProposal
    upgradeState    *UpgradeState
    
    // 子管理器
    dualChain       *DualChainManager
    monitor         *PreexecMonitor
    switchMgr       *SwitchManager
    rollbackMgr     *RollbackManager
    
    mu              sync.RWMutex
}

// NewUpgradeManager 创建升级管理器
func NewUpgradeManager(
    uc *consensus.UpgradeableConsensus,
    cfg *config.ConsensusConfig,
    log *logrus.Entry,
) *UpgradeManager {
    // 初始化治理委员会
    // (这里简化,实际应从配置读取)
    committee := initializeCommittee(cfg)
    
    // 初始化双链管理器
    storage := initializeStorage(cfg)
    dualChain := NewDualChainManager(uc.GetWorkingConsensus(), storage, log)
    
    return &UpgradeManager{
        uc:        uc,
        cfg:       cfg,
        log:       log,
        committee: committee,
        dualChain: dualChain,
    }
}

// ProcessUpgradeTransaction 处理升级交易
func (um *UpgradeManager) ProcessUpgradeTransaction(tx *pb.Transaction) error {
    um.mu.Lock()
    defer um.mu.Unlock()
    
    // 解析交易
    proposal, err := UnpackUpgradeTransaction(tx)
    if err != nil {
        return fmt.Errorf("failed to unpack upgrade transaction: %w", err)
    }
    
    // 验证提案
    if err := um.validateProposal(proposal); err != nil {
        return fmt.Errorf("invalid proposal: %w", err)
    }
    
    // 根据阶段处理
    switch proposal.Phase {
    case pb.UpgradePhase_PHASE_PROPOSAL:
        return um.handleProposalPhase(proposal)
        
    case pb.UpgradePhase_PHASE_PREPARE:
        return um.handlePreparePhase(proposal)
        
    case pb.UpgradePhase_PHASE_PREEXECUTION:
        return um.handlePreexecutionPhase(proposal)
        
    case pb.UpgradePhase_PHASE_CONFIRMATION:
        return um.handleConfirmationPhase(proposal)
        
    case pb.UpgradePhase_PHASE_ACTIVATION:
        return um.handleActivationPhase(proposal)
        
    default:
        return fmt.Errorf("unknown upgrade phase: %v", proposal.Phase)
    }
}

// validateProposal 验证提案
func (um *UpgradeManager) validateProposal(proposal *UpgradeProposal) error {
    // 验证签名
    if err := um.committee.VerifyPartialSignatures(proposal); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }
    
    // 验证参数
    currentHeight := um.dualChain.GetMainChainHeight()
    if err := ValidateProposalParameters(proposal, currentHeight); err != nil {
        return fmt.Errorf("parameter validation failed: %w", err)
    }
    
    // 如果是自定义共识,验证 CDL
    if proposal.TargetConsensus == "custom" && proposal.DescriptorCDL != nil {
        validator := cdl.NewValidator()
        if err := validator.Validate(proposal.DescriptorCDL); err != nil {
            return fmt.Errorf("CDL validation failed: %w", err)
        }
    }
    
    return nil
}

// handleProposalPhase 处理提案阶段
func (um *UpgradeManager) handleProposalPhase(proposal *UpgradeProposal) error {
    um.log.WithField("proposal_id", proposal.ProposalID).Info("Processing upgrade proposal")
    
    // 保存提案
    um.currentProposal = proposal
    
    // 初始化升级状态
    um.upgradeState = &UpgradeState{
        CurrentProposal: proposal,
        Phase:           pb.UpgradePhase_PHASE_PROPOSAL,
    }
    
    um.log.Info("Upgrade proposal accepted")
    
    return nil
}

// handlePreparePhase 处理预备阶段
func (um *UpgradeManager) handlePreparePhase(proposal *UpgradeProposal) error {
    um.log.Info("Entering prepare phase")
    
    currentHeight := um.dualChain.GetMainChainHeight()
    
    // 检查是否达到预备高度
    if currentHeight < proposal.PrepareHeight {
        return fmt.Errorf("not yet at prepare height: current=%d, prepare=%d",
            currentHeight, proposal.PrepareHeight)
    }
    
    // 加载或编译新共识
    var newConsensus model.Consensus
    var err error
    
    if proposal.TargetConsensus == "custom" {
        // 使用 CDL 编译自定义共识
        compiler := cdl.NewCompiler(um.log)
        newConsensus, err = compiler.Compile(
            proposal.DescriptorCDL,
            um.uc.GetNodeID(),
            proposal.ProposalID.Int64(),  // 使用提案 ID 作为新共识 ID
            um.cfg,
            um.uc.GetExecutor(),
            um.uc.GetP2PAdaptor(),
        )
    } else {
        // 加载内置共识
        newConsensus = consensus.BuildConsensus(
            um.uc.GetNodeID(),
            proposal.ProposalID.Int64(),
            um.cfg,
            um.uc.GetExecutor(),
            um.uc.GetP2PAdaptor(),
            um.log,
        )
    }
    
    if err != nil {
        return fmt.Errorf("failed to load new consensus: %w", err)
    }
    
    // 启动预执行链
    if err := um.dualChain.StartPreexecution(proposal.PrepareHeight, newConsensus); err != nil {
        return fmt.Errorf("failed to start preexecution: %w", err)
    }
    
    // 初始化监控器
    um.monitor = NewPreexecMonitor(proposal, um.dualChain, um.log)
    um.monitor.SetBaseline(10*time.Second, 100.0)  // 示例基准值
    um.monitor.Start()
    
    // 初始化回退管理器并启用自动回退
    um.rollbackMgr = NewRollbackManager(um.dualChain, um.monitor, um.log)
    um.rollbackMgr.AutoRollbackOnAnomaly()
    
    um.upgradeState.Phase = pb.UpgradePhase_PHASE_PREEXECUTION
    
    um.log.Info("Preexecution started")
    
    return nil
}

// handlePreexecutionPhase 处理预执行阶段
func (um *UpgradeManager) handlePreexecutionPhase(proposal *UpgradeProposal) error {
    // 预执行阶段主要是被动监控,由 monitor 和 rollbackMgr 自动处理
    um.log.Debug("Preexecution phase in progress")
    return nil
}

// handleConfirmationPhase 处理确认阶段
func (um *UpgradeManager) handleConfirmationPhase(proposal *UpgradeProposal) error {
    um.log.Info("Processing upgrade confirmation")
    
    // 检查预执行是否完成
    currentHeight := um.dualChain.GetPreexecChainHeight()
    if currentHeight < proposal.PreexecHeight {
        return fmt.Errorf("preexecution not complete: current=%d, required=%d",
            currentHeight, proposal.PreexecHeight)
    }
    
    // 获取预执行指标
    metrics := um.monitor.GetMetrics()
    
    // 评估指标
    if !um.monitor.IsHealthy() {
        um.log.Warn("Preexecution metrics not healthy, upgrade rejected")
        return um.rollbackMgr.ExecuteRollback("Metrics evaluation failed")
    }
    
    // 准备切换
    um.switchMgr = NewSwitchManager(um.dualChain, um.monitor, um.log)
    switchHeight := proposal.PreexecHeight + 10  // 给一些缓冲区块
    
    if err := um.switchMgr.PrepareSwitch(switchHeight); err != nil {
        return fmt.Errorf("failed to prepare switch: %w", err)
    }
    
    um.upgradeState.Phase = pb.UpgradePhase_PHASE_ACTIVATION
    um.upgradeState.SwitchHeight = switchHeight
    
    um.log.WithField("switch_height", switchHeight).Info("Upgrade confirmed, switch prepared")
    
    return nil
}

// handleActivationPhase 处理激活阶段
func (um *UpgradeManager) handleActivationPhase(proposal *UpgradeProposal) error {
    um.log.Info("Activating new consensus")
    
    // 执行切换
    if err := um.switchMgr.ExecuteSwitch(); err != nil {
        um.log.WithError(err).Error("Switch failed, executing rollback")
        return um.rollbackMgr.ExecuteRollback(fmt.Sprintf("Switch failed: %v", err))
    }
    
    // 分配激励
    um.distributeIncentives(proposal)
    
    // 更新状态
    um.upgradeState.Switched = true
    
    um.log.Info("Consensus upgrade completed successfully")
    
    return nil
}

// distributeIncentives 分配激励
func (um *UpgradeManager) distributeIncentives(proposal *UpgradeProposal) {
    um.log.WithField("amount", proposal.IncentiveAmount).Info("Distributing incentives")
    
    // 平分给所有签名的委员会成员
    // (实际实现需要与代币系统集成)
    perMember := proposal.IncentiveAmount / uint64(len(proposal.IncentiveRecipients))
    
    for _, recipient := range proposal.IncentiveRecipients {
        um.log.WithFields(logrus.Fields{
            "recipient": fmt.Sprintf("%x", recipient[:8]),
            "amount":    perMember,
        }).Debug("Incentive distributed")
    }
}

// GetUpgradeState 获取升级状态
func (um *UpgradeManager) GetUpgradeState() *UpgradeState {
    um.mu.RLock()
    defer um.mu.RUnlock()
    return um.upgradeState
}

// 辅助函数 (简化实现)
func initializeCommittee(cfg *config.ConsensusConfig) *GovernanceCommittee {
    // 实际应从配置读取委员会信息
    return NewGovernanceCommittee(nil, 5)
}

func initializeStorage(cfg *config.ConsensusConfig) DualChainStorage {
    // 实际应创建真实的存储
    storage, _ := storage.NewLevelDBDualChainStorage("./data/dual_chain")
    return storage
}
```

## 10. 自定义 PoW 共识示例

### 10.1 自定义 PoW 的 CDL 定义

**文件**: `examples/custom_pow.yaml`

```yaml
consensus:
  name: "CustomPoW"
  version: "1.0"
  type: "proof-based"
  
  # 密码学和网络组件
  components:
    crypto:
      hash: "SHA256"
      signature: "ECDSA"
      vdf: ""
      vrf: ""
      commitment: ""
      threshold_sig: ""
    
    network:
      topology: "gossip"
      broadcast: "best-effort"
    
    storage:
      blockchain: "merkle-chain"
      state: "merkle-patricia"
  
  # 协议参数
  parameters:
    block_time: "10s"
    initial_difficulty: 20
    difficulty_adjustment_interval: 2016
    max_block_size: 2097152  # 2MB
    security_param: 128
    reward_per_block: 50
  
  # 协议阶段
  phases:
    # 阶段 1: 交易验证
    - name: "transaction_validation"
      entry: "receive_tx"
      actions:
        - type: "verify"
          function: "verify_signature"
          parameters:
            input: "tx"
        - type: "verify"
          function: "verify_balance"
          parameters:
            input: "tx"
        - type: "function"
          function: "add_to_mempool"
          parameters:
            tx: "validated_tx"
      exit: "tx_validated"
    
    # 阶段 2: 区块生产 (挖矿)
    - name: "block_production"
      entry: "mining_round_start"
      actions:
        - type: "select_txs"
          function: "select_transactions_from_mempool"
          parameters:
            max_count: 1000
        - type: "function"
          function: "construct_block_header"
          parameters:
            txs: "selected_txs"
        - type: "function"
          function: "solve_pow_puzzle"
          parameters:
            header: "block_header"
            difficulty: "current_difficulty"
      postcondition: "valid_proof_of_work()"
      exit: "block_produced"
    
    # 阶段 3: 区块验证
    - name: "block_validation"
      entry: "receive_block"
      actions:
        - type: "verify"
          function: "verify_block_header"
          parameters:
            block: "received_block"
        - type: "verify"
          function: "verify_proof_of_work"
          parameters:
            block: "received_block"
            difficulty: "current_difficulty"
        - type: "verify"
          function: "verify_transactions"
          parameters:
            block: "received_block"
        - type: "verify"
          function: "verify_state_transition"
          parameters:
            block: "received_block"
      postcondition: "all_verifications_passed()"
      exit: "block_validated"
    
    # 阶段 4: 链选择
    - name: "chain_selection"
      entry: "multiple_chains_exist"
      actions:
        - type: "function"
          function: "select_longest_chain"
        - type: "function"
          function: "resolve_forks"
      exit: "canonical_chain_selected"
  
  # 状态机
  state_machine:
    states:
      - IDLE
      - MINING
      - VALIDATING
      - COMMITTING
      - SYNCING
    
    transitions:
      # IDLE -> MINING: 有新交易时开始挖矿
      - from: IDLE
        to: MINING
        condition: "new_tx_available() && is_miner()"
        action: "start_mining()"
      
      # MINING -> VALIDATING: 收到区块或找到解
      - from: MINING
        to: VALIDATING
        condition: "block_received() || puzzle_solved()"
        action: "validate_block()"
      
      # VALIDATING -> COMMITTING: 区块有效
      - from: VALIDATING
        to: COMMITTING
        condition: "block_valid()"
        action: "commit_block()"
      
      # VALIDATING -> IDLE: 区块无效
      - from: VALIDATING
        to: IDLE
        condition: "!block_valid()"
        action: "reject_block()"
      
      # COMMITTING -> IDLE: 提交完成
      - from: COMMITTING
        to: IDLE
        condition: "committed()"
        action: "cleanup()"
      
      # IDLE -> SYNCING: 检测到落后
      - from: IDLE
        to: SYNCING
        condition: "behind_network()"
        action: "start_sync()"
      
      # SYNCING -> IDLE: 同步完成
      - from: SYNCING
        to: IDLE
        condition: "synced()"
        action: "finish_sync()"
  
  # 安全属性
  safety_properties:
    - name: "agreement"
      formula: "∀i,j ∈ HonestNodes: block_i[h] = block_j[h]"
    
    - name: "validity"
      formula: "∀b ∈ Chain: valid_pow(b) ∧ valid_txs(b)"
    
    - name: "liveness"
      formula: "∀tx: eventually(tx ∈ Chain)"
    
    - name: "chain_quality"
      formula: "honest_blocks_ratio(k) ≥ (1-f) for any k consecutive blocks"
  
  # 性能要求
  performance_requirements:
    min_throughput: 10      # tx/s
    max_latency: 600        # seconds (10 minutes)
    fault_tolerance: 0.49   # < 50% hashpower
```

### 10.2 使用自定义 PoW 的示例代码

**文件**: `examples/upgrade_to_custom_pow.go`

```go
package main

import (
    "fmt"
    "io/ioutil"
    "time"
    
    "github.com/sirupsen/logrus"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade/cdl"
)

func main() {
    log := logrus.NewEntry(logrus.New())
    
    // 1. 解析自定义 PoW 的 CDL
    cdlContent, err := ioutil.ReadFile("examples/custom_pow.yaml")
    if err != nil {
        panic(fmt.Sprintf("Failed to read CDL file: %v", err))
    }
    
    parser := cdl.NewParser()
    powCDL, err := parser.Parse(string(cdlContent))
    if err != nil {
        panic(fmt.Sprintf("Failed to parse CDL: %v", err))
    }
    
    log.WithField("consensus", powCDL.Name).Info("CDL parsed successfully")
    
    // 2. 验证 CDL
    validator := cdl.NewValidator()
    if err := validator.Validate(powCDL); err != nil {
        panic(fmt.Sprintf("CDL validation failed: %v", err))
    }
    
    log.Info("CDL validation passed")
    
    // 3. 创建升级提案
    committee := createMockCommittee()
    currentHeight := uint64(1000)  // 假设当前区块高度
    
    proposal, err := upgrade.CreateUpgradeProposal(
        "custom",           // 自定义共识
        powCDL,             // CDL 描述符
        currentHeight,      // 当前高度
        committee,          // 治理委员会
        1000000,            // 激励金额
    )
    if err != nil {
        panic(fmt.Sprintf("Failed to create proposal: %v", err))
    }
    
    log.WithField("proposal_id", proposal.ProposalID).Info("Upgrade proposal created")
    
    // 4. 委员会成员签名
    signatures := collectCommitteeSignatures(proposal, committee)
    proposal.CommitteeSignatures = signatures
    proposal.CommitteePubkeys = committee.GetPublicKeys()
    
    log.WithField("sig_count", len(signatures)).Info("Collected committee signatures")
    
    // 5. 验证提案
    if err := upgrade.VerifyProposalSignatures(proposal, committee); err != nil {
        panic(fmt.Sprintf("Signature verification failed: %v", err))
    }
    
    log.Info("Proposal signatures verified")
    
    // 6. 打包成交易
    tx, err := upgrade.PackUpgradeTransaction(proposal)
    if err != nil {
        panic(fmt.Sprintf("Failed to pack transaction: %v", err))
    }
    
    log.WithField("tx_size", len(tx.Payload)).Info("Upgrade transaction packed")
    
    // 7. 提交交易到区块链
    // (这里简化,实际需要通过 P2P 网络广播)
    submitTransaction(tx)
    
    log.Info("Upgrade transaction submitted")
    
    // 8. 等待升级流程
    // 实际运行中,升级管理器会自动处理各个阶段
    log.Info("Upgrade process initiated. Stages:")
    log.Info("  1. PROPOSAL  - Transaction submitted")
    log.Info("  2. PREPARE   - Will start at height", proposal.PrepareHeight)
    log.Info("  3. PREEXEC   - Will run for", proposal.PreexecHeight - proposal.PrepareHeight, "blocks")
    log.Info("  4. CONFIRM   - Committee will evaluate metrics")
    log.Info("  5. ACTIVATE  - Switch to new consensus if approved")
    
    // 监控升级状态
    monitorUpgradeStatus(proposal)
}

// createMockCommittee 创建模拟委员会 (示例)
func createMockCommittee() *upgrade.GovernanceCommittee {
    members := make([]*upgrade.CommitteeMember, 7)
    for i := 0; i < 7; i++ {
        members[i] = &upgrade.CommitteeMember{
            ID: int64(i),
            // 实际需要真实的密钥
        }
    }
    
    committee := upgrade.NewGovernanceCommittee(members, 5)  // 7个成员,需要5个签名
    
    // 初始化门限签名密钥
    committee.InitializeThresholdKeys(2048, 5, 7)
    
    return committee
}

// collectCommitteeSignatures 收集委员会签名
func collectCommitteeSignatures(
    proposal *upgrade.UpgradeProposal,
    committee *upgrade.GovernanceCommittee,
) [][]byte {
    signatures := make([][]byte, 0)
    
    // 模拟收集足够的签名
    for i := int64(0); i < 5; i++ {
        sig, err := committee.SignProposal(proposal, i)
        if err != nil {
            continue
        }
        signatures = append(signatures, sig)
    }
    
    return signatures
}

// submitTransaction 提交交易
func submitTransaction(tx *pb.Transaction) {
    // 实际实现需要通过 P2P 网络广播
    fmt.Println("Transaction submitted to network")
}

// monitorUpgradeStatus 监控升级状态
func monitorUpgradeStatus(proposal *upgrade.UpgradeProposal) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        <-ticker.C
        
        // 查询升级状态
        // 实际实现需要从升级管理器获取状态
        fmt.Println("Monitoring upgrade status...")
        
        // 示例输出
        fmt.Printf("  Current height: %d\n", getCurrentHeight())
        fmt.Printf("  Prepare height: %d\n", proposal.PrepareHeight)
        fmt.Printf("  Preexec height: %d\n", proposal.PreexecHeight)
        
        // 如果升级完成,退出
        if isUpgradeComplete() {
            fmt.Println("Upgrade completed successfully!")
            break
        }
    }
}

func getCurrentHeight() uint64 {
    // 实际需要从区块链获取
    return 1050
}

func isUpgradeComplete() bool {
    // 实际需要检查升级管理器的状态
    return false
}
```

### 10.3 测试自定义 PoW

**文件**: `examples/test_custom_pow.go`

```go
package main

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade/cdl"
)

func TestCustomPoWCDL(t *testing.T) {
    // 解析 CDL
    parser := cdl.NewParser()
    powCDL, err := parser.ParseFile("examples/custom_pow.yaml")
    assert.NoError(t, err)
    assert.NotNil(t, powCDL)
    
    // 验证基本字段
    assert.Equal(t, "CustomPoW", powCDL.Name)
    assert.Equal(t, "1.0", powCDL.Version)
    assert.Equal(t, "proof-based", powCDL.Type)
    
    // 验证组件
    assert.Equal(t, "SHA256", powCDL.Components.Crypto.Hash)
    assert.Equal(t, "ECDSA", powCDL.Components.Crypto.Signature)
    assert.Equal(t, "gossip", powCDL.Components.Network.Topology)
    
    // 验证阶段
    assert.Len(t, powCDL.Phases, 4)
    assert.Equal(t, "transaction_validation", powCDL.Phases[0].Name)
    assert.Equal(t, "block_production", powCDL.Phases[1].Name)
    
    // 验证状态机
    assert.Len(t, powCDL.StateMachine.States, 5)
    assert.Contains(t, powCDL.StateMachine.States, "IDLE")
    assert.Contains(t, powCDL.StateMachine.States, "MINING")
    
    // 验证安全属性
    assert.Len(t, powCDL.SafetyProperties, 4)
    
    // CDL 验证
    validator := cdl.NewValidator()
    err = validator.Validate(powCDL)
    assert.NoError(t, err)
}

func TestCustomPoWCompilation(t *testing.T) {
    // 解析 CDL
    parser := cdl.NewParser()
    powCDL, err := parser.ParseFile("examples/custom_pow.yaml")
    assert.NoError(t, err)
    
    // 编译 CDL
    compiler := cdl.NewCompiler(log)
    runtime, err := compiler.Compile(
        powCDL,
        1,      // node id
        100,    // consensus id
        cfg,    // config
        exec,   // executor
        p2p,    // p2p adaptor
    )
    assert.NoError(t, err)
    assert.NotNil(t, runtime)
    
    // 验证运行时
    assert.Equal(t, int64(100), runtime.GetConsensusID())
    assert.Equal(t, "CustomPoW", runtime.GetConsensusType())
}

func TestCustomPoWExecution(t *testing.T) {
    // 创建运行时
    runtime := createCustomPoWRuntime(t)
    
    // 启动运行时
    go runtime.Run()
    defer runtime.Stop()
    
    // 提交交易
    tx := createTestTransaction()
    runtime.GetRequestEntrance() <- tx
    
    // 等待一段时间
    time.Sleep(2 * time.Second)
    
    // 验证状态
    // (实际测试需要检查区块是否生成)
}
```

## 11. API 接口设计

### 11.1 HTTP API 接口

**文件**: `internal/apis/upgrade_api.go`

```go
package apis

import (
    "encoding/json"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
)

// UpgradeAPI 升级相关 API
type UpgradeAPI struct {
    manager *upgrade.UpgradeManager
}

// NewUpgradeAPI 创建升级 API
func NewUpgradeAPI(manager *upgrade.UpgradeManager) *UpgradeAPI {
    return &UpgradeAPI{manager: manager}
}

// RegisterRoutes 注册路由
func (api *UpgradeAPI) RegisterRoutes(router *gin.Engine) {
    upgrade := router.Group("/api/v1/upgrade")
    {
        // 提案相关
        upgrade.POST("/propose", api.ProposeUpgrade)
        upgrade.GET("/proposal/:id", api.GetProposal)
        upgrade.GET("/proposals", api.ListProposals)
        
        // 签名相关
        upgrade.POST("/sign/:id", api.SignProposal)
        upgrade.GET("/signatures/:id", api.GetSignatures)
        
        // 状态查询
        upgrade.GET("/status", api.GetUpgradeStatus)
        upgrade.GET("/metrics", api.GetPreexecMetrics)
        
        // 控制操作
        upgrade.POST("/confirm/:id", api.ConfirmUpgrade)
        upgrade.POST("/rollback/:id", api.RollbackUpgrade)
    }
}

// ProposeUpgrade 提交升级提案
// POST /api/v1/upgrade/propose
// Body: { "target_consensus": "pow", "cdl": "...", "incentive": 1000000 }
func (api *UpgradeAPI) ProposeUpgrade(c *gin.Context) {
    var req struct {
        TargetConsensus string `json:"target_consensus"`
        CDL             string `json:"cdl,omitempty"`
        Incentive       uint64 `json:"incentive"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // 解析 CDL (如果有)
    var cdl *cdl.CDLDescriptor
    if req.CDL != "" {
        cdl = cdl.ParseCDL(req.CDL)
    }
    
    // 创建提案
    proposal, err := upgrade.CreateUpgradeProposal(
        req.TargetConsensus,
        cdl,
        api.getCurrentHeight(),
        api.getCommittee(),
        req.Incentive,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "proposal_id": proposal.ProposalID.Hex(),
        "prepare_height": proposal.PrepareHeight,
        "preexec_height": proposal.PreexecHeight,
    })
}

// GetProposal 获取提案详情
// GET /api/v1/upgrade/proposal/:id
func (api *UpgradeAPI) GetProposal(c *gin.Context) {
    proposalID := c.Param("id")
    
    // 查询提案
    proposal := api.manager.GetProposal(proposalID)
    if proposal == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Proposal not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "proposal_id": proposal.ProposalID.Hex(),
        "target_consensus": proposal.TargetConsensus,
        "prepare_height": proposal.PrepareHeight,
        "preexec_height": proposal.PreexecHeight,
        "phase": proposal.Phase.String(),
        "signatures": len(proposal.CommitteeSignatures),
        "threshold": proposal.Threshold,
    })
}

// ListProposals 列出所有提案
// GET /api/v1/upgrade/proposals
func (api *UpgradeAPI) ListProposals(c *gin.Context) {
    proposals := api.manager.ListProposals()
    
    result := make([]gin.H, len(proposals))
    for i, p := range proposals {
        result[i] = gin.H{
            "proposal_id": p.ProposalID.Hex(),
            "target_consensus": p.TargetConsensus,
            "phase": p.Phase.String(),
            "timestamp": p.Timestamp.Unix(),
        }
    }
    
    c.JSON(http.StatusOK, gin.H{"proposals": result})
}

// SignProposal 签名提案
// POST /api/v1/upgrade/sign/:id
// Body: { "member_id": 1 }
func (api *UpgradeAPI) SignProposal(c *gin.Context) {
    proposalID := c.Param("id")
    
    var req struct {
        MemberID int64 `json:"member_id"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // 签名提案
    signature, err := api.manager.SignProposal(proposalID, req.MemberID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "signature": fmt.Sprintf("%x", signature),
    })
}

// GetUpgradeStatus 获取升级状态
// GET /api/v1/upgrade/status
func (api *UpgradeAPI) GetUpgradeStatus(c *gin.Context) {
    state := api.manager.GetUpgradeState()
    
    if state == nil {
        c.JSON(http.StatusOK, gin.H{"status": "no_upgrade"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "proposal_id": state.CurrentProposal.ProposalID.Hex(),
        "phase": state.Phase.String(),
        "main_chain_height": state.MainChain.CurrentHeight,
        "preexec_chain_height": func() uint64 {
            if state.PreexecChain != nil {
                return state.PreexecChain.CurrentHeight
            }
            return 0
        }(),
        "switched": state.Switched,
    })
}

// GetPreexecMetrics 获取预执行指标
// GET /api/v1/upgrade/metrics
func (api *UpgradeAPI) GetPreexecMetrics(c *gin.Context) {
    metrics := api.manager.GetPreexecMetrics()
    
    if metrics == nil {
        c.JSON(http.StatusOK, gin.H{"status": "no_preexec"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "avg_block_time": metrics.AvgBlockTime,
        "avg_throughput": metrics.AvgThroughput,
        "avg_latency": metrics.AvgLatency,
        "error_rate": metrics.ErrorRate,
        "total_blocks": metrics.TotalBlocks,
        "total_txs": metrics.TotalTxs,
    })
}

// ConfirmUpgrade 确认升级
// POST /api/v1/upgrade/confirm/:id
func (api *UpgradeAPI) ConfirmUpgrade(c *gin.Context) {
    proposalID := c.Param("id")
    
    if err := api.manager.ConfirmUpgrade(proposalID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}

// RollbackUpgrade 回退升级
// POST /api/v1/upgrade/rollback/:id
// Body: { "reason": "Performance issue" }
func (api *UpgradeAPI) RollbackUpgrade(c *gin.Context) {
    proposalID := c.Param("id")
    
    var req struct {
        Reason string `json:"reason"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := api.manager.RollbackUpgrade(proposalID, req.Reason); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"status": "rolled_back"})
}

// 辅助方法
func (api *UpgradeAPI) getCurrentHeight() uint64 {
    // 从管理器获取当前高度
    return api.manager.GetCurrentHeight()
}

func (api *UpgradeAPI) getCommittee() *upgrade.GovernanceCommittee {
    return api.manager.GetCommittee()
}
```

### 11.2 CLI 命令

**文件**: `cmd/upgradecli/main.go`

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "upgradecli",
        Short: "CLI tool for consensus upgrade management",
    }
    
    // 提案命令
    proposeCmd := &cobra.Command{
        Use:   "propose [target_consensus]",
        Short: "Create a new upgrade proposal",
        Args:  cobra.ExactArgs(1),
        Run:   runPropose,
    }
    proposeCmd.Flags().String("cdl", "", "Path to CDL file for custom consensus")
    proposeCmd.Flags().Uint64("incentive", 1000000, "Incentive amount")
    
    // 签名命令
    signCmd := &cobra.Command{
        Use:   "sign [proposal_id]",
        Short: "Sign an upgrade proposal",
        Args:  cobra.ExactArgs(1),
        Run:   runSign,
    }
    signCmd.Flags().Int64("member-id", 0, "Committee member ID")
    
    // 状态命令
    statusCmd := &cobra.Command{
        Use:   "status",
        Short: "Get upgrade status",
        Run:   runStatus,
    }
    
    // 指标命令
    metricsCmd := &cobra.Command{
        Use:   "metrics",
        Short: "Get preexecution metrics",
        Run:   runMetrics,
    }
    
    // 确认命令
    confirmCmd := &cobra.Command{
        Use:   "confirm [proposal_id]",
        Short: "Confirm upgrade",
        Args:  cobra.ExactArgs(1),
        Run:   runConfirm,
    }
    
    // 回退命令
    rollbackCmd := &cobra.Command{
        Use:   "rollback [proposal_id]",
        Short: "Rollback upgrade",
        Args:  cobra.ExactArgs(1),
        Run:   runRollback,
    }
    rollbackCmd.Flags().String("reason", "", "Rollback reason")
    
    rootCmd.AddCommand(
        proposeCmd,
        signCmd,
        statusCmd,
        metricsCmd,
        confirmCmd,
        rollbackCmd,
    )
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func runPropose(cmd *cobra.Command, args []string) {
    targetConsensus := args[0]
    cdlPath, _ := cmd.Flags().GetString("cdl")
    incentive, _ := cmd.Flags().GetUint64("incentive")
    
    fmt.Printf("Creating upgrade proposal:\n")
    fmt.Printf("  Target consensus: %s\n", targetConsensus)
    fmt.Printf("  CDL file: %s\n", cdlPath)
    fmt.Printf("  Incentive: %d\n", incentive)
    
    // 实际实现需要调用 API
}

func runSign(cmd *cobra.Command, args []string) {
    proposalID := args[0]
    memberID, _ := cmd.Flags().GetInt64("member-id")
    
    fmt.Printf("Signing proposal %s as member %d\n", proposalID, memberID)
    
    // 实际实现需要调用 API
}

func runStatus(cmd *cobra.Command, args []string) {
    fmt.Println("Querying upgrade status...")
    
    // 实际实现需要调用 API
}

func runMetrics(cmd *cobra.Command, args []string) {
    fmt.Println("Fetching preexecution metrics...")
    
    // 实际实现需要调用 API
}

func runConfirm(cmd *cobra.Command, args []string) {
    proposalID := args[0]
    
    fmt.Printf("Confirming upgrade %s\n", proposalID)
    
    // 实际实现需要调用 API
}

func runRollback(cmd *cobra.Command, args []string) {
    proposalID := args[0]
    reason, _ := cmd.Flags().GetString("reason")
    
    fmt.Printf("Rolling back upgrade %s\n", proposalID)
    fmt.Printf("Reason: %s\n", reason)
    
    // 实际实现需要调用 API
}
```

## 12. 测试方案

### 12.1 单元测试

**文件**: `consensus/upgrade/upgrade_test.go`

```go
package upgrade_test

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
)

// TestUpgradeProposalCreation 测试提案创建
func TestUpgradeProposalCreation(t *testing.T) {
    committee := createTestCommittee(t)
    currentHeight := uint64(1000)
    
    proposal, err := upgrade.CreateUpgradeProposal(
        "pow",
        nil,
        currentHeight,
        committee,
        1000000,
    )
    
    require.NoError(t, err)
    assert.NotNil(t, proposal)
    assert.Equal(t, "pow", proposal.TargetConsensus)
    assert.Equal(t, currentHeight+100, proposal.PrepareHeight)
    assert.Equal(t, currentHeight+1100, proposal.PreexecHeight)
}

// TestProposalValidation 测试提案验证
func TestProposalValidation(t *testing.T) {
    committee := createTestCommittee(t)
    currentHeight := uint64(1000)
    
    proposal, _ := upgrade.CreateUpgradeProposal(
        "pow", nil, currentHeight, committee, 1000000,
    )
    
    // 测试参数验证
    err := upgrade.ValidateProposalParameters(proposal, currentHeight)
    assert.NoError(t, err)
    
    // 测试无效的高度
    invalidProposal := *proposal
    invalidProposal.PrepareHeight = currentHeight - 10
    err = upgrade.ValidateProposalParameters(&invalidProposal, currentHeight)
    assert.Error(t, err)
}

// TestCommitteeSignature 测试委员会签名
func TestCommitteeSignature(t *testing.T) {
    committee := createTestCommittee(t)
    proposal, _ := upgrade.CreateUpgradeProposal(
        "pow", nil, 1000, committee, 1000000,
    )
    
    // 收集签名
    signatures := make([][]byte, 0)
    for i := int64(0); i < 5; i++ {
        sig, err := committee.SignProposal(proposal, i)
        require.NoError(t, err)
        signatures = append(signatures, sig)
    }
    
    proposal.CommitteeSignatures = signatures
    proposal.CommitteePubkeys = committee.GetPublicKeys()
    
    // 验证签名
    err := upgrade.VerifyProposalSignatures(proposal, committee)
    assert.NoError(t, err)
}

// TestDualChainManagement 测试双链管理
func TestDualChainManagement(t *testing.T) {
    storage := createTestStorage(t)
    mainConsensus := createTestConsensus(t)
    newConsensus := createTestConsensus(t)
    
    dcm := upgrade.NewDualChainManager(mainConsensus, storage, testLog)
    
    // 启动预执行
    err := dcm.StartPreexecution(100, newConsensus)
    require.NoError(t, err)
    assert.True(t, dcm.IsPreexecActive())
    
    // 模拟主链区块
    mainBlock := createTestBlock(101)
    err = dcm.ProcessMainChainBlock(mainBlock)
    assert.NoError(t, err)
    
    // 模拟预执行链区块
    preexecBlock := createTestBlock(101)
    err = dcm.ProcessPreexecBlock(preexecBlock)
    assert.NoError(t, err)
    
    // 回退预执行
    err = dcm.RollbackPreexecution()
    assert.NoError(t, err)
    assert.False(t, dcm.IsPreexecActive())
}

// TestMetricsCollection 测试指标收集
func TestMetricsCollection(t *testing.T) {
    collector := upgrade.NewMetricsCollector(
        testProposalID,
        100,
        testLog,
    )
    
    // 记录区块指标
    for i := 0; i < 10; i++ {
        collector.RecordBlock(
            uint64(100+i),
            10*time.Second,
            100,
            nil,
        )
    }
    
    // 计算聚合指标
    metrics := collector.ComputeAggregateMetrics()
    assert.Equal(t, 10*time.Second, metrics.AvgBlockTime)
    assert.Equal(t, 10.0, metrics.AvgThroughput)
    assert.Equal(t, 0.0, metrics.ErrorRate)
}

// TestCDLParsing 测试 CDL 解析
func TestCDLParsing(t *testing.T) {
    cdlYAML := `
consensus:
  name: "TestConsensus"
  version: "1.0"
  type: "test"
  components:
    crypto:
      hash: "SHA256"
      signature: "ECDSA"
    network:
      topology: "gossip"
      broadcast: "reliable"
    storage:
      blockchain: "merkle-chain"
      state: "merkle-patricia"
  parameters:
    block_time: "10s"
  phases:
    - name: "test_phase"
      entry: "start"
      actions:
        - type: "function"
          function: "test_func"
      exit: "end"
  state_machine:
    states: ["IDLE", "RUNNING"]
    transitions:
      - from: "IDLE"
        to: "RUNNING"
        condition: "start()"
        action: "run()"
  safety_properties:
    - name: "agreement"
      formula: "test"
    - name: "validity"
      formula: "test"
  performance_requirements:
    min_throughput: 100
    max_latency: 10
    fault_tolerance: 0.33
`
    
    parser := cdl.NewParser()
    descriptor, err := parser.Parse(cdlYAML)
    
    require.NoError(t, err)
    assert.Equal(t, "TestConsensus", descriptor.Name)
    assert.Equal(t, "SHA256", descriptor.Components.Crypto.Hash)
}

// TestCDLValidation 测试 CDL 验证
func TestCDLValidation(t *testing.T) {
    descriptor := createTestCDL()
    
    validator := cdl.NewValidator()
    err := validator.Validate(descriptor)
    assert.NoError(t, err)
}

// TestSwitchProcess 测试切换流程
func TestSwitchProcess(t *testing.T) {
    dcm := createTestDualChainManager(t)
    monitor := createTestMonitor(t)
    
    switchMgr := upgrade.NewSwitchManager(dcm, monitor, testLog)
    
    // 准备切换
    err := switchMgr.PrepareSwitch(200)
    require.NoError(t, err)
    
    // 执行切换
    go func() {
        err := switchMgr.ExecuteSwitch()
        assert.NoError(t, err)
    }()
    
    // 等待切换
    err = switchMgr.WaitForSwitch(10 * time.Second)
    assert.NoError(t, err)
    assert.True(t, switchMgr.IsSwitched())
}

// TestRollbackProcess 测试回退流程
func TestRollbackProcess(t *testing.T) {
    dcm := createTestDualChainManager(t)
    monitor := createTestMonitor(t)
    
    rollbackMgr := upgrade.NewRollbackManager(dcm, monitor, testLog)
    
    // 执行回退
    err := rollbackMgr.ExecuteRollback("Test rollback")
    require.NoError(t, err)
    assert.True(t, rollbackMgr.IsRolledBack())
    assert.Equal(t, "Test rollback", rollbackMgr.GetRollbackReason())
}

// 辅助函数
func createTestCommittee(t *testing.T) *upgrade.GovernanceCommittee {
    members := make([]*upgrade.CommitteeMember, 7)
    for i := 0; i < 7; i++ {
        members[i] = &upgrade.CommitteeMember{ID: int64(i)}
    }
    committee := upgrade.NewGovernanceCommittee(members, 5)
    committee.InitializeThresholdKeys(2048, 5, 7)
    return committee
}

func createTestStorage(t *testing.T) upgrade.DualChainStorage {
    storage, err := storage.NewLevelDBDualChainStorage(t.TempDir())
    require.NoError(t, err)
    return storage
}

func createTestConsensus(t *testing.T) model.Consensus {
    // 返回模拟的共识实例
    return &mockConsensus{}
}

func createTestBlock(height uint64) *types.Block {
    return &types.Block{
        Header: &types.Header{Height: height},
        Txs:    []*types.Tx{},
    }
}
```

### 12.2 集成测试

**文件**: `tests/upgrade_integration_test.go`

```go
package tests

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
)

// TestFullUpgradeFlow 测试完整升级流程
func TestFullUpgradeFlow(t *testing.T) {
    // 1. 设置测试环境
    env := setupTestEnvironment(t)
    defer env.Cleanup()
    
    // 2. 创建升级提案
    proposal := env.CreateProposal("pow")
    assert.NotNil(t, proposal)
    
    // 3. 收集委员会签名
    env.CollectSignatures(proposal)
    assert.Len(t, proposal.CommitteeSignatures, 5)
    
    // 4. 提交升级交易
    tx, err := upgrade.PackUpgradeTransaction(proposal)
    assert.NoError(t, err)
    env.SubmitTransaction(tx)
    
    // 5. 等待达到预备高度
    env.WaitForHeight(proposal.PrepareHeight)
    
    // 6. 验证预执行链启动
    assert.True(t, env.IsPreexecActive())
    
    // 7. 运行预执行阶段
    env.RunPreexecution(proposal.PreexecHeight - proposal.PrepareHeight)
    
    // 8. 检查预执行指标
    metrics := env.GetPreexecMetrics()
    assert.NotNil(t, metrics)
    assert.Less(t, metrics.ErrorRate, 0.05)
    
    // 9. 确认升级
    err = env.ConfirmUpgrade(proposal.ProposalID)
    assert.NoError(t, err)
    
    // 10. 等待切换完成
    env.WaitForSwitch()
    
    // 11. 验证新共识激活
    assert.True(t, env.IsSwitched())
    assert.Equal(t, "pow", env.GetCurrentConsensusType())
}

// TestUpgradeRollback 测试升级回退
func TestUpgradeRollback(t *testing.T) {
    env := setupTestEnvironment(t)
    defer env.Cleanup()
    
    // 创建提案并启动预执行
    proposal := env.CreateProposal("pow")
    env.CollectSignatures(proposal)
    env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal))
    env.WaitForHeight(proposal.PrepareHeight)
    
    // 模拟预执行失败
    env.InjectPreexecError()
    
    // 验证自动回退
    time.Sleep(2 * time.Second)
    assert.True(t, env.IsRolledBack())
    assert.False(t, env.IsPreexecActive())
}

// TestCustomConsensusUpgrade 测试自定义共识升级
func TestCustomConsensusUpgrade(t *testing.T) {
    env := setupTestEnvironment(t)
    defer env.Cleanup()
    
    // 加载自定义 CDL
    cdl := env.LoadCDL("testdata/custom_consensus.yaml")
    assert.NotNil(t, cdl)
    
    // 创建自定义共识提案
    proposal := env.CreateCustomProposal(cdl)
    env.CollectSignatures(proposal)
    env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal))
    
    // 执行完整流程
    env.WaitForHeight(proposal.PrepareHeight)
    env.RunPreexecution(1000)
    env.ConfirmUpgrade(proposal.ProposalID)
    env.WaitForSwitch()
    
    // 验证自定义共识激活
    assert.True(t, env.IsSwitched())
    assert.Equal(t, cdl.Name, env.GetCurrentConsensusType())
}

// TestConcurrentUpgrades 测试并发升级处理
func TestConcurrentUpgrades(t *testing.T) {
    env := setupTestEnvironment(t)
    defer env.Cleanup()
    
    // 提交两个升级提案
    proposal1 := env.CreateProposal("pow")
    proposal2 := env.CreateProposal("hotstuff")
    
    env.CollectSignatures(proposal1)
    env.CollectSignatures(proposal2)
    
    env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal1))
    env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal2))
    
    // 验证只有一个提案被接受
    activeProposals := env.GetActiveProposals()
    assert.Len(t, activeProposals, 1)
}

// TestUpgradeWithNetworkPartition 测试网络分区下的升级
func TestUpgradeWithNetworkPartition(t *testing.T) {
    env := setupTestEnvironment(t)
    defer env.Cleanup()
    
    proposal := env.CreateProposal("pow")
    env.CollectSignatures(proposal)
    env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal))
    env.WaitForHeight(proposal.PrepareHeight)
    
    // 模拟网络分区
    env.SimulateNetworkPartition()
    
    // 运行预执行
    env.RunPreexecution(500)
    
    // 恢复网络
    env.HealNetworkPartition()
    
    // 验证节点能够同步并完成升级
    env.RunPreexecution(500)
    env.ConfirmUpgrade(proposal.ProposalID)
    env.WaitForSwitch()
    
    assert.True(t, env.AllNodesSwitched())
}

// TestEnvironment 测试环境
type TestEnvironment struct {
    nodes       []*TestNode
    committee   *upgrade.GovernanceCommittee
    t           *testing.T
}

func setupTestEnvironment(t *testing.T) *TestEnvironment {
    // 创建测试节点
    nodes := make([]*TestNode, 7)
    for i := 0; i < 7; i++ {
        nodes[i] = NewTestNode(i)
    }
    
    // 创建委员会
    committee := createTestCommittee()
    
    return &TestEnvironment{
        nodes:     nodes,
        committee: committee,
        t:         t,
    }
}

func (env *TestEnvironment) CreateProposal(consensusType string) *upgrade.UpgradeProposal {
    proposal, _ := upgrade.CreateUpgradeProposal(
        consensusType,
        nil,
        env.GetCurrentHeight(),
        env.committee,
        1000000,
    )
    return proposal
}

func (env *TestEnvironment) CollectSignatures(proposal *upgrade.UpgradeProposal) {
    signatures := make([][]byte, 0)
    for i := int64(0); i < 5; i++ {
        sig, _ := env.committee.SignProposal(proposal, i)
        signatures = append(signatures, sig)
    }
    proposal.CommitteeSignatures = signatures
    proposal.CommitteePubkeys = env.committee.GetPublicKeys()
}

func (env *TestEnvironment) Cleanup() {
    for _, node := range env.nodes {
        node.Stop()
    }
}
```

### 12.3 性能测试

**文件**: `tests/upgrade_benchmark_test.go`

```go
package tests

import (
    "testing"
    "time"
)

// BenchmarkProposalCreation 基准测试:提案创建
func BenchmarkProposalCreation(b *testing.B) {
    committee := createTestCommittee()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        upgrade.CreateUpgradeProposal(
            "pow", nil, 1000, committee, 1000000,
        )
    }
}

// BenchmarkSignatureVerification 基准测试:签名验证
func BenchmarkSignatureVerification(b *testing.B) {
    committee := createTestCommittee()
    proposal, _ := upgrade.CreateUpgradeProposal(
        "pow", nil, 1000, committee, 1000000,
    )
    
    // 收集签名
    signatures := make([][]byte, 5)
    for i := int64(0); i < 5; i++ {
        signatures[i], _ = committee.SignProposal(proposal, i)
    }
    proposal.CommitteeSignatures = signatures
    proposal.CommitteePubkeys = committee.GetPublicKeys()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        upgrade.VerifyProposalSignatures(proposal, committee)
    }
}

// BenchmarkCDLParsing 基准测试:CDL 解析
func BenchmarkCDLParsing(b *testing.B) {
    cdlYAML := loadTestCDL()
    parser := cdl.NewParser()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(cdlYAML)
    }
}

// BenchmarkBlockProcessing 基准测试:双链区块处理
func BenchmarkBlockProcessing(b *testing.B) {
    dcm := createTestDualChainManager()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        block := createTestBlock(uint64(i))
        dcm.ProcessMainChainBlock(block)
    }
}

// BenchmarkMetricsCollection 基准测试:指标收集
func BenchmarkMetricsCollection(b *testing.B) {
    collector := upgrade.NewMetricsCollector(testProposalID, 0, testLog)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        collector.RecordBlock(
            uint64(i),
            10*time.Second,
            100,
            nil,
        )
    }
}

// BenchmarkFullUpgradeFlow 基准测试:完整升级流程
func BenchmarkFullUpgradeFlow(b *testing.B) {
    for i := 0; i < b.N; i++ {
        env := setupBenchmarkEnvironment()
        
        proposal := env.CreateProposal("pow")
        env.CollectSignatures(proposal)
        env.SubmitTransaction(upgrade.PackUpgradeTransaction(proposal))
        env.WaitForSwitch()
        
        env.Cleanup()
    }
}
```

### 12.4 测试数据

**文件**: `testdata/custom_consensus.yaml`

```yaml
consensus:
  name: "TestConsensus"
  version: "1.0.0"
  type: "test"
  
  components:
    crypto:
      hash: "SHA256"
      signature: "ECDSA"
      vdf: ""
      vrf: ""
      commitment: ""
      threshold_sig: ""
    network:
      topology: "gossip"
      broadcast: "reliable"
    storage:
      blockchain: "merkle-chain"
      state: "merkle-patricia"
  
  parameters:
    block_time: "5s"
    max_block_size: 1048576
  
  phases:
    - name: "init"
      entry: "start"
      actions:
        - type: "function"
          function: "initialize"
      exit: "initialized"
  
  state_machine:
    states: ["IDLE", "RUNNING", "STOPPED"]
    transitions:
      - from: "IDLE"
        to: "RUNNING"
        condition: "start()"
        action: "run()"
      - from: "RUNNING"
        to: "STOPPED"
        condition: "stop()"
        action: "cleanup()"
  
  safety_properties:
    - name: "agreement"
      formula: "test"
    - name: "validity"
      formula: "test"
  
  performance_requirements:
    min_throughput: 50
    max_latency: 20
    fault_tolerance: 0.33
```

### 12.5 运行测试

**测试命令**:

```bash
# 运行所有单元测试
make test

# 运行特定包的测试
go test -v ./consensus/upgrade/...

# 运行集成测试
go test -v ./tests/ -tags=integration

# 运行基准测试
go test -bench=. ./tests/

# 生成测试覆盖率报告
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 运行竞态检测
go test -race ./consensus/upgrade/...

# 运行长时间压力测试
go test -v ./tests/ -timeout 30m -tags=stress
```

---

## 13. 实现步骤总结

### 13.1 第一阶段:核心基础设施 (1-2周)

1. 实现 protobuf 定义 (`pkg/proto/upgrade.proto`)
2. 实现核心类型 (`consensus/upgrade/types.go`)
3. 实现双链存储 (`internal/storage/dual_chain_storage.go`)
4. 编写单元测试

### 13.2 第二阶段:交易与治理 (1-2周)

1. 实现升级交易处理 (`consensus/upgrade/transaction.go`)
2. 实现治理委员会 (`consensus/upgrade/governance.go`)
3. 集成门限签名
4. 编写单元测试

### 13.3 第三阶段:双链管理 (2-3周)

1. 实现双链管理器 (`consensus/upgrade/dual_chain.go`)
2. 实现预执行监控 (`consensus/upgrade/preexec_monitor.go`)
3. 实现指标收集 (`consensus/upgrade/metrics.go`)
4. 编写集成测试

### 13.4 第四阶段:CDL 引擎 (2-3周)

1. 实现 CDL 解析器 (`consensus/upgrade/cdl/parser.go`)
2. 实现 CDL 验证器 (`consensus/upgrade/cdl/validator.go`)
3. 实现 CDL 编译器 (`consensus/upgrade/cdl/compiler.go`)
4. 实现 CDL 运行时 (`consensus/upgrade/cdl/runtime.go`)
5. 编写自定义共识示例

### 13.5 第五阶段:切换与回退 (1-2周)

1. 实现切换管理器 (`consensus/upgrade/switch.go`)
2. 实现回退管理器 (`consensus/upgrade/rollback.go`)
3. 实现升级管理器 (`consensus/upgrade/manager.go`)
4. 编写端到端测试

### 13.6 第六阶段:API 与工具 (1周)

1. 实现 HTTP API (`internal/apis/upgrade_api.go`)
2. 实现 CLI 工具 (`cmd/upgradecli/main.go`)
3. 编写 API 文档
4. 编写使用示例

### 13.7 第七阶段:测试与优化 (2周)

1. 完善单元测试和集成测试
2. 性能基准测试和优化
3. 安全审计
4. 文档完善

**总计: 约 10-14 周**

---

## 14. 注意事项

### 14.1 安全考虑

1. **门限签名密钥管理**: 确保密钥分发和存储的安全性
2. **CDL 验证**: 严格验证自定义 CDL,防止恶意代码注入
3. **双链隔离**: 确保预执行链故障不影响主链
4. **回退安全**: 确保回退操作的原子性和一致性

### 14.2 性能优化

1. **并行处理**: 主链和预执行链的区块处理可以并行
2. **缓存策略**: 缓存频繁访问的数据(如区块、状态)
3. **存储优化**: 使用高效的存储格式,定期清理过期数据
4. **网络优化**: 批量传输交易,压缩消息

### 14.3 兼容性

1. **向后兼容**: 新版本需兼容旧版本的数据格式
2. **版本控制**: 为 CDL 和协议添加版本号
3. **迁移工具**: 提供数据迁移工具

### 14.4 监控与日志

1. **详细日志**: 记录关键操作和状态变化
2. **性能指标**: 实时监控各阶段的性能指标
3. **告警机制**: 异常情况及时告警
4. **可视化**: 提供升级状态的可视化界面

---

## 15. 参考资料

- [项目 copilot-instructions.md](/root/ldc/workspace/pot/.github/copilot-instructions.md)
- [学术设计方案](./consensus-upgrade-protocol-academic.md)
- [HotStuff 论文](https://arxiv.org/abs/1803.05069)
- [Tezos 自修正白皮书](https://tezos.com/whitepaper.pdf)
- [门限签名标准 RFC](https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-threshold-bls-signature)

---

