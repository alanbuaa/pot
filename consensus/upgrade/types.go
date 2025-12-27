package upgrade

import (
	"encoding/json"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// UpgradeProposal 升级提案
type UpgradeProposal struct {
	ProposalID          types.TxHash
	TargetConsensus     string
	CDLDescriptor       *CDLDescriptor
	ForkHeight          uint64
	PreexecStartHeight  uint64
	SwitchHeight        uint64
	RollbackCondition   *pb.RollbackCondition
	CommitteeSignatures [][]byte
	CommitteePubkeys    [][]byte
	Threshold           uint32
	Incentive           uint64
	Timestamp           time.Time
	Proposer            []byte
	Description         string
	ConsensusParams     map[string]interface{}
	ConsensusID         int64
	MetadataHash        []byte
	Nonce               uint64
}

// ToProto 转换为 protobuf
func (up *UpgradeProposal) ToProto() *pb.UpgradeConfigTransaction {
	var cdlStr string
	if up.CDLDescriptor != nil {
		cdlStr = up.CDLDescriptor.Serialize()
	}

	paramsBytes, _ := json.Marshal(up.ConsensusParams)

	return &pb.UpgradeConfigTransaction{
		ProposalId:          up.ProposalID[:],
		TargetConsensus:     up.TargetConsensus,
		CdlYaml:             cdlStr,
		ForkHeight:          up.ForkHeight,
		PreexecStartHeight:  up.PreexecStartHeight,
		SwitchHeight:        up.SwitchHeight,
		RollbackCondition:   up.RollbackCondition,
		CommitteeSignatures: up.CommitteeSignatures,
		CommitteePubkeys:    up.CommitteePubkeys,
		Threshold:           up.Threshold,
		Incentive:           up.Incentive,
		Timestamp:           up.Timestamp.Unix(),
		Proposer:            up.Proposer,
		Description:         up.Description,
		ConsensusParams:     string(paramsBytes),
		ConsensusId:         up.ConsensusID,
		MetadataHash:        up.MetadataHash,
		Nonce:               up.Nonce,
	}
}

// Hash 计算提案哈希
func (up *UpgradeProposal) Hash() types.TxHash {
	protoTx := up.ToProto()
	data, _ := json.Marshal(protoTx)
	hashBytes := crypto.Hash(data)
	var hash types.TxHash
	copy(hash[:], hashBytes)
	return hash
}

// UpgradeState 升级状态
type UpgradeState struct {
	CurrentProposal *UpgradeProposal
	Phase           pb.UpgradePhase
	ForkHeight      uint64
	PreexecStarted  bool
	PreexecHeight   uint64
	SwitchHeight    uint64
	SwitchReady     bool
	Metrics         *PerformanceMetrics
	RollbackReason  string
	ActivatedAt     time.Time
	LastUpdated     time.Time
	Switched        bool
}

// ChainState 链状态
type ChainState struct {
	ConsensusID     int64
	CurrentHeight   uint64
	LatestBlockHash types.TxHash
	Consensus       model.Consensus
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	StartHeight   uint64
	EndHeight     uint64
	TotalBlocks   uint64
	FailedBlocks  uint64
	BlockTimes    []float64
	TxCounts      []uint32
	AvgBlockTime  float64
	AvgThroughput float64
	ErrorRate     float64
}

// ToProto 转换为 protobuf
func (pm *PerformanceMetrics) ToProto() *pb.ExecutionMetrics {
	blockMetrics := make([]*pb.BlockMetric, len(pm.BlockTimes))
	for i, blockTime := range pm.BlockTimes {
		var txCount uint32
		if i < len(pm.TxCounts) {
			txCount = pm.TxCounts[i]
		}
		blockMetrics[i] = &pb.BlockMetric{
			Height:    pm.StartHeight + uint64(i),
			BlockTime: blockTime,
			TxCount:   txCount,
			Success:   true,
		}
	}

	return &pb.ExecutionMetrics{
		AvgBlockTime:  pm.AvgBlockTime,
		AvgThroughput: pm.AvgThroughput,
		ErrorRate:     pm.ErrorRate,
		TotalBlocks:   pm.TotalBlocks,
		FailedBlocks:  pm.FailedBlocks,
		Timestamp:     time.Now().Unix(),
		BlockMetrics:  blockMetrics,
	}
}

// Evaluate 评估指标是否满足要求
func (pm *PerformanceMetrics) Evaluate(condition *pb.RollbackCondition) bool {
	if pm.ErrorRate > condition.MaxErrorRate {
		return false
	}
	return true
}

// CDLDescriptor CDL 共识描述符 (简化版本)
type CDLDescriptor struct {
	Name                    string
	Version                 string
	Type                    string
	Components              interface{}
	Parameters              interface{}
	Phases                  interface{}
	StateMachine            interface{}
	SafetyProperties        interface{}
	PerformanceRequirements interface{}
}

// Serialize 序列化为 YAML 字符串
func (cdl *CDLDescriptor) Serialize() string {
	// 简化实现：返回 JSON
	data, _ := json.Marshal(cdl)
	return string(data)
}

// Hash 计算 CDL 哈希
func (cdl *CDLDescriptor) Hash() []byte {
	data := cdl.Serialize()
	hash := crypto.Hash([]byte(data))
	return hash[:]
}

// Validate 验证 CDL 完整性
func (cdl *CDLDescriptor) Validate() error {
	if cdl.Name == "" {
		return ErrInvalidCDL
	}
	if cdl.Version == "" {
		return ErrInvalidCDL
	}
	return nil
}

// ProposalFromProto 从 protobuf 转换
func ProposalFromProto(pb *pb.UpgradeConfigTransaction) *UpgradeProposal {
	var cdl *CDLDescriptor
	if pb.CdlYaml != "" {
		cdl = &CDLDescriptor{}
		json.Unmarshal([]byte(pb.CdlYaml), cdl)
	}

	var params map[string]interface{}
	if pb.ConsensusParams != "" {
		json.Unmarshal([]byte(pb.ConsensusParams), &params)
	}

	var proposalID types.TxHash
	copy(proposalID[:], pb.ProposalId)

	return &UpgradeProposal{
		ProposalID:          proposalID,
		TargetConsensus:     pb.TargetConsensus,
		CDLDescriptor:       cdl,
		ForkHeight:          pb.ForkHeight,
		PreexecStartHeight:  pb.PreexecStartHeight,
		SwitchHeight:        pb.SwitchHeight,
		RollbackCondition:   pb.RollbackCondition,
		CommitteeSignatures: pb.CommitteeSignatures,
		CommitteePubkeys:    pb.CommitteePubkeys,
		Threshold:           pb.Threshold,
		Incentive:           pb.Incentive,
		Timestamp:           time.Unix(pb.Timestamp, 0),
		Proposer:            pb.Proposer,
		Description:         pb.Description,
		ConsensusParams:     params,
		ConsensusID:         pb.ConsensusId,
		MetadataHash:        pb.MetadataHash,
		Nonce:               pb.Nonce,
	}
}
