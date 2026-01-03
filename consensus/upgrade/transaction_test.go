package upgrade

import (
	"testing"
	"time"

	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestCreateUpgradeProposal(t *testing.T) {
	tests := []struct {
		name            string
		targetConsensus string
		cdl             *CDLDescriptor
		currentHeight   uint64
		threshold       uint32
		incentive       uint64
		proposer        []byte
		description     string
		wantErr         bool
	}{
		{
			name:            "valid proposal for existing consensus",
			targetConsensus: "hotstuff",
			cdl:             nil,
			currentHeight:   100,
			threshold:       3,
			incentive:       1000000,
			proposer:        []byte("proposer1"),
			description:     "upgrade to hotstuff",
			wantErr:         false,
		},
		{
			name:            "valid proposal with CDL",
			targetConsensus: "custom",
			cdl: &CDLDescriptor{
				Name:    "CustomPoW",
				Version: "1.0",
			},
			currentHeight: 100,
			threshold:     2,
			incentive:     500000,
			proposer:      []byte("proposer2"),
			description:   "upgrade to custom PoW",
			wantErr:       false,
		},
		{
			name:            "custom consensus without CDL should fail validation",
			targetConsensus: "custom",
			cdl:             nil,
			currentHeight:   100,
			threshold:       2,
			incentive:       500000,
			proposer:        []byte("proposer3"),
			description:     "invalid custom",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, err := CreateUpgradeProposal(
				tt.targetConsensus,
				tt.cdl,
				tt.currentHeight,
				tt.threshold,
				tt.incentive,
				tt.proposer,
				tt.description,
			)

			if tt.wantErr {
				if err == nil {
					// 验证提案参数应该失败
					err = ValidateProposalParameters(proposal, tt.currentHeight)
					if err == nil {
						t.Error("expected error but got none")
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateUpgradeProposal() error = %v", err)
			}

			// 验证提案基本字段
			if proposal.TargetConsensus != tt.targetConsensus {
				t.Errorf("TargetConsensus = %v, want %v", proposal.TargetConsensus, tt.targetConsensus)
			}

			if proposal.Threshold != tt.threshold {
				t.Errorf("Threshold = %v, want %v", proposal.Threshold, tt.threshold)
			}

			if proposal.Incentive != tt.incentive {
				t.Errorf("Incentive = %v, want %v", proposal.Incentive, tt.incentive)
			}

			// 验证高度参数
			if proposal.ForkHeight <= tt.currentHeight {
				t.Error("ForkHeight should be greater than current height")
			}

			if proposal.PreexecStartHeight < proposal.ForkHeight {
				t.Error("PreexecStartHeight should be >= ForkHeight")
			}

			if proposal.SwitchHeight <= proposal.PreexecStartHeight {
				t.Error("SwitchHeight should be > PreexecStartHeight")
			}

			// 验证 CDL 哈希（如果是自定义共识）
			if tt.targetConsensus == "custom" && tt.cdl != nil {
				if proposal.MetadataHash == nil {
					t.Error("MetadataHash should not be nil for custom consensus")
				}
			}

			// 验证提案 ID 不为空
			if proposal.ProposalID == (types.TxHash{}) {
				t.Error("ProposalID should not be empty")
			}
		})
	}
}

func TestValidateProposalParameters(t *testing.T) {
	currentHeight := uint64(100)

	tests := []struct {
		name     string
		proposal *UpgradeProposal
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid proposal",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       1200,
				Threshold:          3,
			},
			wantErr: false,
		},
		{
			name: "fork height in the past",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         50,
				PreexecStartHeight: 150,
				SwitchHeight:       1150,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "fork height must be in the future",
		},
		{
			name: "preexec start before fork",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         200,
				PreexecStartHeight: 150,
				SwitchHeight:       1200,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "preexec start height must be >= fork height",
		},
		{
			name: "switch height not after preexec",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       200,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "switch height must be after preexec start height",
		},
		{
			name: "fork gap too small",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         105,
				PreexecStartHeight: 105,
				SwitchHeight:       1105,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "fork gap too small",
		},
		{
			name: "preexec gap too small",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       250,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "preexec gap too small",
		},
		{
			name: "invalid consensus type",
			proposal: &UpgradeProposal{
				TargetConsensus:    "invalid_consensus",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       1200,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "invalid consensus type",
		},
		{
			name: "custom consensus without CDL",
			proposal: &UpgradeProposal{
				TargetConsensus:    "custom",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       1200,
				Threshold:          3,
			},
			wantErr: true,
			errMsg:  "custom consensus requires CDL descriptor",
		},
		{
			name: "zero threshold",
			proposal: &UpgradeProposal{
				TargetConsensus:    "hotstuff",
				ForkHeight:         200,
				PreexecStartHeight: 200,
				SwitchHeight:       1200,
				Threshold:          0,
			},
			wantErr: true,
			errMsg:  "threshold must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProposalParameters(tt.proposal, currentHeight)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateProposalParameters() error = %v", err)
			}
		})
	}
}

func TestPackAndUnpackUpgradeTransaction(t *testing.T) {
	proposal := &UpgradeProposal{
		ProposalID:         types.TxHash{1, 2, 3, 4},
		TargetConsensus:    "hotstuff",
		ForkHeight:         1000,
		PreexecStartHeight: 1100,
		SwitchHeight:       2100,
		Threshold:          3,
		Incentive:          1000000,
		Timestamp:          time.Now(),
		Proposer:           []byte("proposer1"),
		Description:        "Test upgrade",
		ConsensusID:        2,
		Nonce:              1,
		RollbackCondition: &pb.RollbackCondition{
			MaxErrorRate:       0.05,
			MaxLatencyIncrease: 0.2,
			MinThroughputRatio: 0.8,
			TimeoutBlocks:      100,
		},
	}

	// 打包
	tx, err := PackUpgradeTransaction(proposal)
	if err != nil {
		t.Fatalf("PackUpgradeTransaction() error = %v", err)
	}

	// 验证交易类型
	if tx.Type != pb.TransactionType_UPGRADE {
		t.Errorf("Transaction type = %v, want %v", tx.Type, pb.TransactionType_UPGRADE)
	}

	// 解包
	unpacked, err := UnpackUpgradeTransaction(tx)
	if err != nil {
		t.Fatalf("UnpackUpgradeTransaction() error = %v", err)
	}

	// 验证字段
	if unpacked.TargetConsensus != proposal.TargetConsensus {
		t.Errorf("TargetConsensus = %v, want %v", unpacked.TargetConsensus, proposal.TargetConsensus)
	}

	if unpacked.ForkHeight != proposal.ForkHeight {
		t.Errorf("ForkHeight = %v, want %v", unpacked.ForkHeight, proposal.ForkHeight)
	}

	if unpacked.SwitchHeight != proposal.SwitchHeight {
		t.Errorf("SwitchHeight = %v, want %v", unpacked.SwitchHeight, proposal.SwitchHeight)
	}

	if unpacked.Threshold != proposal.Threshold {
		t.Errorf("Threshold = %v, want %v", unpacked.Threshold, proposal.Threshold)
	}
}

func TestCreateConfirmTransaction(t *testing.T) {
	proposalID := types.TxHash{1, 2, 3, 4, 5, 6, 7, 8}
	signature := []byte("aggregated_signature")
	confirmerID := int64(1)

	// 创建批准的确认交易
	confirm, err := CreateConfirmTransaction(proposalID, true, signature, confirmerID)
	if err != nil {
		t.Fatalf("CreateConfirmTransaction() error = %v", err)
	}

	if !confirm.Approved {
		t.Error("Approved should be true")
	}

	if len(confirm.Signature) != len(signature) {
		t.Errorf("Signature length = %v, want %v", len(confirm.Signature), len(signature))
	}

	// 创建拒绝的确认交易
	confirmReject, err := CreateConfirmTransaction(proposalID, false, signature, confirmerID)
	if err != nil {
		t.Fatalf("CreateConfirmTransaction() error = %v", err)
	}

	if confirmReject.Approved {
		t.Error("Approved should be false")
	}
}

func TestPackAndUnpackConfirmTransaction(t *testing.T) {
	proposalID := types.TxHash{1, 2, 3, 4}
	confirm := &pb.UpgradeConfirmTransaction{
		ProposalId:  proposalID[:],
		Signature:   []byte("signature"),
		ConfirmerId: 1,
		Timestamp:   time.Now().Unix(),
		Approved:    true,
	}

	// 打包
	tx, err := PackConfirmTransaction(confirm)
	if err != nil {
		t.Fatalf("PackConfirmTransaction() error = %v", err)
	}

	// 验证交易类型
	if tx.Type != pb.TransactionType_LOCK {
		t.Errorf("Transaction type = %v, want %v", tx.Type, pb.TransactionType_LOCK)
	}

	// 解包
	unpacked, err := UnpackConfirmTransaction(tx)
	if err != nil {
		t.Fatalf("UnpackConfirmTransaction() error = %v", err)
	}

	// 验证字段
	if unpacked.Approved != confirm.Approved {
		t.Errorf("Approved = %v, want %v", unpacked.Approved, confirm.Approved)
	}

	if unpacked.ConfirmerId != confirm.ConfirmerId {
		t.Errorf("ConfirmerId = %v, want %v", unpacked.ConfirmerId, confirm.ConfirmerId)
	}
}

func TestUnpackUpgradeTransaction_InvalidType(t *testing.T) {
	// 创建一个非升级交易
	tx := &pb.Transaction{
		Type:    pb.TransactionType_NORMAL,
		Payload: []byte("test"),
	}

	_, err := UnpackUpgradeTransaction(tx)
	if err == nil {
		t.Error("expected error for non-upgrade transaction")
	}
}

func TestUnpackConfirmTransaction_InvalidType(t *testing.T) {
	// 创建一个非确认交易
	tx := &pb.Transaction{
		Type:    pb.TransactionType_NORMAL,
		Payload: []byte("test"),
	}

	_, err := UnpackConfirmTransaction(tx)
	if err == nil {
		t.Error("expected error for non-confirm transaction")
	}
}
