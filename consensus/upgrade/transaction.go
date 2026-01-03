package upgrade

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/niclabs/tcrsa"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// CreateUpgradeProposal 创建升级提案
func CreateUpgradeProposal(
	targetConsensus string,
	cdl *CDLDescriptor,
	currentHeight uint64,
	threshold uint32,
	incentive uint64,
	proposer []byte,
	description string,
) (*UpgradeProposal, error) {

	// 计算区块高度
	forkHeight := currentHeight + 100 // 分叉点在当前高度之后 100 个区块
	preexecStartHeight := forkHeight  // 预执行从分叉点开始
	switchHeight := forkHeight + 1000 // 切换点在分叉点之后 1000 个区块

	proposal := &UpgradeProposal{
		TargetConsensus:    targetConsensus,
		CDLDescriptor:      cdl,
		ForkHeight:         forkHeight,
		PreexecStartHeight: preexecStartHeight,
		SwitchHeight:       switchHeight,
		Threshold:          threshold,
		Incentive:          incentive,
		Timestamp:          time.Now(),
		Proposer:           proposer,
		Description:        description,
		ConsensusID:        0, // 将在后续设置
		Nonce:              uint64(time.Now().UnixNano()),
		RollbackCondition: &pb.RollbackCondition{
			MaxErrorRate:       0.05,
			MaxLatencyIncrease: 0.2,
			MinThroughputRatio: 0.8,
			TimeoutBlocks:      100,
		},
	}

	// 如果是自定义共识，必须有 CDL
	if targetConsensus == "custom" {
		if cdl == nil {
			return nil, errors.New("custom consensus requires CDL descriptor")
		}
		// 计算 CDL 哈希并存储到 MetadataHash
		proposal.MetadataHash = cdl.Hash()
	}

	// 计算提案 ID
	proposal.ProposalID = proposal.Hash()

	return proposal, nil
}

// SignProposal 委员会成员签名提案
func SignProposal(
	proposal *UpgradeProposal,
	privateKey *tcrsa.KeyShare,
	publicKey *tcrsa.KeyMeta,
) ([]byte, error) {
	hash := proposal.Hash()

	// 创建文档哈希
	documentHash, err := crypto.CreateDocumentHash(hash[:], publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create document hash: %w", err)
	}

	// 使用门限签名
	sigShare, err := crypto.TSign(documentHash, privateKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// 序列化签名
	sigBytes, err := json.Marshal(sigShare)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signature: %w", err)
	}

	return sigBytes, nil
}

// VerifyProposalSignatures 验证提案签名
func VerifyProposalSignatures(
	proposal *UpgradeProposal,
	publicKey *tcrsa.KeyMeta,
) error {
	if len(proposal.CommitteeSignatures) < int(proposal.Threshold) {
		return fmt.Errorf("insufficient signatures: got %d, need %d",
			len(proposal.CommitteeSignatures), proposal.Threshold)
	}

	hash := proposal.Hash()
	documentHash, err := crypto.CreateDocumentHash(hash[:], publicKey)
	if err != nil {
		return fmt.Errorf("failed to create document hash: %w", err)
	}

	validCount := 0

	for _, sigBytes := range proposal.CommitteeSignatures {
		// 反序列化签名
		sig := &tcrsa.SigShare{}
		if err := json.Unmarshal(sigBytes, sig); err != nil {
			continue // 跳过无效签名
		}

		// 验证部分签名
		if err := crypto.VerifyPartSig(sig, documentHash, publicKey); err == nil {
			validCount++
		}
	}

	if validCount < int(proposal.Threshold) {
		return fmt.Errorf("invalid signatures: only %d valid out of %d",
			validCount, len(proposal.CommitteeSignatures))
	}

	return nil
}

// AggregateSignatures 聚合签名
func AggregateSignatures(
	proposal *UpgradeProposal,
	publicKey *tcrsa.KeyMeta,
) ([]byte, error) {
	if len(proposal.CommitteeSignatures) < int(proposal.Threshold) {
		return nil, fmt.Errorf("insufficient signatures: got %d, need %d",
			len(proposal.CommitteeSignatures), proposal.Threshold)
	}

	hash := proposal.Hash()
	documentHash, err := crypto.CreateDocumentHash(hash[:], publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create document hash: %w", err)
	}

	// 反序列化签名
	sigShares := make(tcrsa.SigShareList, 0, len(proposal.CommitteeSignatures))
	for _, sigBytes := range proposal.CommitteeSignatures {
		sig := &tcrsa.SigShare{}
		if err := json.Unmarshal(sigBytes, sig); err != nil {
			continue // 跳过无效签名
		}
		sigShares = append(sigShares, sig)
	}

	if len(sigShares) < int(proposal.Threshold) {
		return nil, fmt.Errorf("too many invalid signatures")
	}

	// 聚合签名
	fullSig, err := crypto.CreateFullSignature(documentHash, sigShares, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate signatures: %w", err)
	}

	return fullSig, nil
}

// VerifyAggregatedSignature 验证聚合签名
func VerifyAggregatedSignature(
	proposal *UpgradeProposal,
	aggregatedSig []byte,
	publicKey *tcrsa.KeyMeta,
) error {
	hash := proposal.Hash()

	// 验证聚合签名
	ok, err := crypto.TVerify(publicKey, aggregatedSig, hash[:])
	if err != nil || !ok {
		return fmt.Errorf("invalid aggregated signature: %w", err)
	}

	return nil
}

// PackUpgradeTransaction 将提案打包成交易
func PackUpgradeTransaction(proposal *UpgradeProposal) (*pb.Transaction, error) {
	protoProposal := proposal.ToProto()
	payload, err := json.Marshal(protoProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proposal: %w", err)
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
	if err := json.Unmarshal(tx.Payload, protoProposal); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal: %w", err)
	}

	return ProposalFromProto(protoProposal), nil
}

// ValidateProposalParameters 验证提案参数
func ValidateProposalParameters(
	proposal *UpgradeProposal,
	currentHeight uint64,
) error {
	// 检查高度参数
	if proposal.ForkHeight <= currentHeight {
		return errors.New("fork height must be in the future")
	}

	if proposal.PreexecStartHeight < proposal.ForkHeight {
		return errors.New("preexec start height must be >= fork height")
	}

	if proposal.SwitchHeight <= proposal.PreexecStartHeight {
		return errors.New("switch height must be after preexec start height")
	}

	minPrepareGap := uint64(10)
	if proposal.ForkHeight-currentHeight < minPrepareGap {
		return fmt.Errorf("fork gap too small: need at least %d blocks", minPrepareGap)
	}

	minPreexecGap := uint64(100)
	if proposal.SwitchHeight-proposal.PreexecStartHeight < minPreexecGap {
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
	if proposal.TargetConsensus == "custom" && proposal.CDLDescriptor == nil {
		return errors.New("custom consensus requires CDL descriptor")
	}

	// 验证 CDL 哈希
	if proposal.CDLDescriptor != nil {
		computedHash := proposal.CDLDescriptor.Hash()
		if !bytes.Equal(computedHash, proposal.MetadataHash) {
			return errors.New("CDL descriptor hash mismatch")
		}
	}

	// 验证阈值
	if proposal.Threshold == 0 {
		return errors.New("threshold must be greater than 0")
	}

	return nil
}

// CreateConfirmTransaction 创建确认交易
func CreateConfirmTransaction(
	proposalID types.TxHash,
	approved bool,
	signature []byte,
	confirmerID int64,
) (*pb.UpgradeConfirmTransaction, error) {
	return &pb.UpgradeConfirmTransaction{
		ProposalId:  proposalID[:],
		Signature:   signature,
		ConfirmerId: confirmerID,
		Timestamp:   time.Now().Unix(),
		Approved:    approved,
	}, nil
}

// PackConfirmTransaction 打包确认交易
func PackConfirmTransaction(confirm *pb.UpgradeConfirmTransaction) (*pb.Transaction, error) {
	payload, err := json.Marshal(confirm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal confirm transaction: %w", err)
	}

	return &pb.Transaction{
		Type:    pb.TransactionType_LOCK,
		Payload: payload,
	}, nil
}

// UnpackConfirmTransaction 解析确认交易
func UnpackConfirmTransaction(tx *pb.Transaction) (*pb.UpgradeConfirmTransaction, error) {
	if tx.Type != pb.TransactionType_LOCK {
		return nil, errors.New("not an upgrade confirm transaction")
	}

	confirm := &pb.UpgradeConfirmTransaction{}
	if err := json.Unmarshal(tx.Payload, confirm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal confirm transaction: %w", err)
	}

	return confirm, nil
}
