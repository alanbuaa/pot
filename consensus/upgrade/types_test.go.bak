package upgrade

import (
	"testing"
	"time"

	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestUpgradeProposal_ToProto(t *testing.T) {
	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3, 4},
		TargetConsensus:    "hotstuff",
		ForkHeight:         1000,
		PreexecStartHeight: 1100,
		SwitchHeight:       1200,
		RollbackCondition: &pb.RollbackCondition{
			MaxErrorRate:       0.05,
			MaxLatencyIncrease: 0.2,
			MinThroughputRatio: 0.8,
			TimeoutBlocks:      100,
		},
		Threshold:   3,
		Incentive:   1000000,
		Timestamp:   time.Now(),
		Proposer:    []byte("proposer1"),
		Description: "Test upgrade",
		ConsensusID: 2,
		Nonce:       1,
	}

	protoProposal := proposal.ToProto()

	if protoProposal.TargetConsensus != proposal.TargetConsensus {
		t.Errorf("TargetConsensus mismatch: got %s, want %s",
			protoProposal.TargetConsensus, proposal.TargetConsensus)
	}

	if protoProposal.ForkHeight != proposal.ForkHeight {
		t.Errorf("ForkHeight mismatch: got %d, want %d",
			protoProposal.ForkHeight, proposal.ForkHeight)
	}

	if protoProposal.SwitchHeight != proposal.SwitchHeight {
		t.Errorf("SwitchHeight mismatch: got %d, want %d",
			protoProposal.SwitchHeight, proposal.SwitchHeight)
	}

	if protoProposal.Threshold != proposal.Threshold {
		t.Errorf("Threshold mismatch: got %d, want %d",
			protoProposal.Threshold, proposal.Threshold)
	}
}

func TestUpgradeProposal_Hash(t *testing.T) {
	proposal1 := &UpgradeProposal{
		TargetConsensus:    "hotstuff",
		ForkHeight:         1000,
		PreexecStartHeight: 1100,
		SwitchHeight:       1200,
		Threshold:          3,
	}

	proposal2 := &UpgradeProposal{
		TargetConsensus:    "hotstuff",
		ForkHeight:         1000,
		PreexecStartHeight: 1100,
		SwitchHeight:       1200,
		Threshold:          3,
	}

	hash1 := proposal1.Hash()
	hash2 := proposal2.Hash()

	if hash1 != hash2 {
		t.Error("Hash of identical proposals should be the same")
	}

	// 改变一个字段
	proposal2.ForkHeight = 1001
	hash3 := proposal2.Hash()

	if hash1 == hash3 {
		t.Error("Hash of different proposals should be different")
	}
}

func TestProposalFromProto(t *testing.T) {
	protoProposal := &pb.UpgradeConfigTransaction{
		ProposalId:         []byte{1, 2, 3, 4},
		TargetConsensus:    "pow",
		ForkHeight:         2000,
		PreexecStartHeight: 2100,
		SwitchHeight:       2200,
		RollbackCondition: &pb.RollbackCondition{
			MaxErrorRate:       0.1,
			MaxLatencyIncrease: 0.3,
			MinThroughputRatio: 0.7,
			TimeoutBlocks:      200,
		},
		Threshold:   4,
		Incentive:   2000000,
		Timestamp:   time.Now().Unix(),
		Proposer:    []byte("proposer2"),
		Description: "Test upgrade 2",
		ConsensusId: 3,
		Nonce:       2,
	}

	proposal := ProposalFromProto(protoProposal)

	if proposal.TargetConsensus != protoProposal.TargetConsensus {
		t.Errorf("TargetConsensus mismatch: got %s, want %s",
			proposal.TargetConsensus, protoProposal.TargetConsensus)
	}

	if proposal.ForkHeight != protoProposal.ForkHeight {
		t.Errorf("ForkHeight mismatch: got %d, want %d",
			proposal.ForkHeight, protoProposal.ForkHeight)
	}

	if proposal.Threshold != protoProposal.Threshold {
		t.Errorf("Threshold mismatch: got %d, want %d",
			proposal.Threshold, protoProposal.Threshold)
	}
}

func TestPerformanceMetrics_ToProto(t *testing.T) {
	metrics := &PerformanceMetrics{
		StartHeight:   1000,
		EndHeight:     1010,
		TotalBlocks:   10,
		FailedBlocks:  1,
		BlockTimes:    []float64{5.0, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9},
		TxCounts:      []uint32{100, 101, 102, 103, 104, 105, 106, 107, 108, 109},
		AvgBlockTime:  5.45,
		AvgThroughput: 105.0,
		ErrorRate:     0.1,
	}

	protoMetrics := metrics.ToProto()

	if protoMetrics.AvgBlockTime != metrics.AvgBlockTime {
		t.Errorf("AvgBlockTime mismatch: got %f, want %f",
			protoMetrics.AvgBlockTime, metrics.AvgBlockTime)
	}

	if protoMetrics.ErrorRate != metrics.ErrorRate {
		t.Errorf("ErrorRate mismatch: got %f, want %f",
			protoMetrics.ErrorRate, metrics.ErrorRate)
	}

	if len(protoMetrics.BlockMetrics) != len(metrics.BlockTimes) {
		t.Errorf("BlockMetrics count mismatch: got %d, want %d",
			len(protoMetrics.BlockMetrics), len(metrics.BlockTimes))
	}
}

func TestPerformanceMetrics_Evaluate(t *testing.T) {
	metrics := &PerformanceMetrics{
		ErrorRate: 0.03,
	}

	condition := &pb.RollbackCondition{
		MaxErrorRate: 0.05,
	}

	if !metrics.Evaluate(condition) {
		t.Error("Metrics should pass evaluation")
	}

	// 错误率过高
	metrics.ErrorRate = 0.06
	if metrics.Evaluate(condition) {
		t.Error("Metrics should fail evaluation due to high error rate")
	}
}

func TestCDLDescriptor_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cdl       *CDLDescriptor
		shouldErr bool
	}{
		{
			name: "valid CDL",
			cdl: &CDLDescriptor{
				Name:    "TestConsensus",
				Version: "1.0.0",
				Type:    "test",
			},
			shouldErr: false,
		},
		{
			name: "missing name",
			cdl: &CDLDescriptor{
				Version: "1.0.0",
				Type:    "test",
			},
			shouldErr: true,
		},
		{
			name: "missing version",
			cdl: &CDLDescriptor{
				Name: "TestConsensus",
				Type: "test",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cdl.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("Validate() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestCDLDescriptor_Hash(t *testing.T) {
	cdl1 := &CDLDescriptor{
		Name:    "TestConsensus",
		Version: "1.0.0",
		Type:    "test",
	}

	cdl2 := &CDLDescriptor{
		Name:    "TestConsensus",
		Version: "1.0.0",
		Type:    "test",
	}

	hash1 := cdl1.Hash()
	hash2 := cdl2.Hash()

	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}

	if string(hash1) != string(hash2) {
		t.Error("Hash of identical CDLs should be the same")
	}

	// 改变一个字段
	cdl2.Version = "2.0.0"
	hash3 := cdl2.Hash()

	if string(hash1) == string(hash3) {
		t.Error("Hash of different CDLs should be different")
	}
}

func TestUpgradeState(t *testing.T) {
	state := &UpgradeState{
		Phase:          pb.UpgradePhase_PHASE_PROPOSED,
		ForkHeight:     1000,
		PreexecStarted: false,
		SwitchReady:    false,
		Switched:       false,
	}

	if state.Phase != pb.UpgradePhase_PHASE_PROPOSED {
		t.Errorf("Initial phase should be PROPOSED, got %v", state.Phase)
	}

	// 模拟状态转换
	state.Phase = pb.UpgradePhase_PHASE_PREEXEC
	state.PreexecStarted = true

	if !state.PreexecStarted {
		t.Error("PreexecStarted should be true")
	}

	state.Phase = pb.UpgradePhase_PHASE_SWITCHED
	state.Switched = true

	if !state.Switched {
		t.Error("Switched should be true")
	}
}

func TestChainState(t *testing.T) {
	chainState := &ChainState{
		ConsensusID:     1,
		CurrentHeight:   100,
		LatestBlockHash: types.TxHash{1, 2, 3},
		Consensus:       nil,
	}

	if chainState.ConsensusID != 1 {
		t.Errorf("ConsensusID should be 1, got %d", chainState.ConsensusID)
	}

	if chainState.CurrentHeight != 100 {
		t.Errorf("CurrentHeight should be 100, got %d", chainState.CurrentHeight)
	}
}
