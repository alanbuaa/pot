package upgrade

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/niclabs/tcrsa"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
)

// GovernanceCommittee 治理委员会
type GovernanceCommittee struct {
	members   []*CommitteeMember
	threshold uint32 // 门限签名阈值

	// 门限签名相关
	keyMeta *tcrsa.KeyMeta

	mu sync.RWMutex
}

// CommitteeMember 委员会成员
type CommitteeMember struct {
	ID         int64
	PublicKey  []byte // 序列化后的公钥
	PrivateKey *tcrsa.KeyShare
	Address    []byte
}

// NewGovernanceCommittee 创建治理委员会
func NewGovernanceCommittee(members []*CommitteeMember, threshold uint32, keyMeta *tcrsa.KeyMeta) *GovernanceCommittee {
	return &GovernanceCommittee{
		members:   members,
		threshold: threshold,
		keyMeta:   keyMeta,
	}
}

// InitializeThresholdKeys 初始化门限签名密钥
func (gc *GovernanceCommittee) InitializeThresholdKeys(
	keySize int,
	threshold int,
	numShares int,
) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// 生成门限签名密钥
	keyShares, keyMeta, err := crypto.GenerateThresholdKeys(threshold, numShares)
	if err != nil {
		return fmt.Errorf("failed to generate threshold keys: %w", err)
	}

	gc.keyMeta = keyMeta

	// 分发密钥给成员
	for i, member := range gc.members {
		if i < len(keyShares) {
			member.PrivateKey = keyShares[i]
			// 序列化公钥
			pubkeyBytes, err := json.Marshal(keyMeta.PublicKey)
			if err != nil {
				return fmt.Errorf("failed to marshal public key: %w", err)
			}
			member.PublicKey = pubkeyBytes
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

	// 创建文档哈希
	documentHash, err := crypto.CreateDocumentHash(hash[:], gc.keyMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to create document hash: %w", err)
	}

	// 生成部分签名
	sigShare, err := crypto.TSign(documentHash, member.PrivateKey, gc.keyMeta)
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

// AggregateSignatures 聚合门限签名
func (gc *GovernanceCommittee) AggregateSignatures(
	proposal *UpgradeProposal,
	signatures [][]byte,
) ([]byte, error) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	if len(signatures) < int(gc.threshold) {
		return nil, fmt.Errorf("insufficient signatures: got %d, need %d",
			len(signatures), gc.threshold)
	}

	// 反序列化签名
	sigShares := make(tcrsa.SigShareList, 0, len(signatures))
	for _, sigBytes := range signatures {
		sig := &tcrsa.SigShare{}
		if err := json.Unmarshal(sigBytes, sig); err != nil {
			continue // 跳过无效签名
		}
		sigShares = append(sigShares, sig)
	}

	if len(sigShares) < int(gc.threshold) {
		return nil, fmt.Errorf("too many invalid signatures")
	}

	// 聚合签名
	hash := proposal.Hash()
	documentHash, err := crypto.CreateDocumentHash(hash[:], gc.keyMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to create document hash: %w", err)
	}

	fullSig, err := crypto.CreateFullSignature(documentHash, sigShares, gc.keyMeta)
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
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	hash := proposal.Hash()

	// 验证聚合签名
	ok, err := crypto.TVerify(gc.keyMeta, aggregatedSig, hash[:])
	if err != nil || !ok {
		return fmt.Errorf("invalid aggregated signature: %w", err)
	}

	return nil
}

// VerifyPartialSignatures 验证部分签名
func (gc *GovernanceCommittee) VerifyPartialSignatures(
	proposal *UpgradeProposal,
) error {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	if len(proposal.CommitteeSignatures) < int(gc.threshold) {
		return fmt.Errorf("insufficient signatures: got %d, need %d",
			len(proposal.CommitteeSignatures), gc.threshold)
	}

	hash := proposal.Hash()
	documentHash, err := crypto.CreateDocumentHash(hash[:], gc.keyMeta)
	if err != nil {
		return fmt.Errorf("failed to create document hash: %w", err)
	}

	validCount := 0

	for _, sigBytes := range proposal.CommitteeSignatures {
		// 反序列化签名
		sig := &tcrsa.SigShare{}
		if err := json.Unmarshal(sigBytes, sig); err != nil {
			continue
		}

		// 验证签名
		if err := crypto.VerifyPartSig(sig, documentHash, gc.keyMeta); err == nil {
			validCount++
		}
	}

	if validCount < int(gc.threshold) {
		return fmt.Errorf("insufficient valid signatures: got %d, need %d",
			validCount, gc.threshold)
	}

	return nil
}

// AddMember 添加成员
func (gc *GovernanceCommittee) AddMember(member *CommitteeMember) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.members = append(gc.members, member)
}

// RemoveMember 移除成员
func (gc *GovernanceCommittee) RemoveMember(memberID int64) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	for i, member := range gc.members {
		if member.ID == memberID {
			gc.members = append(gc.members[:i], gc.members[i+1:]...)
			return true
		}
	}
	return false
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
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	for _, member := range gc.members {
		// 简单比较序列化后的公钥
		if string(member.PublicKey) == string(pubkey) {
			return member
		}
	}
	return nil
}

// GetPublicKeys 获取所有成员公钥
func (gc *GovernanceCommittee) GetPublicKeys() [][]byte {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	keys := make([][]byte, len(gc.members))
	for i, member := range gc.members {
		keys[i] = member.PublicKey
	}
	return keys
}

// GetThreshold 获取阈值
func (gc *GovernanceCommittee) GetThreshold() uint32 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.threshold
}

// GetMemberCount 获取成员数量
func (gc *GovernanceCommittee) GetMemberCount() int {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return len(gc.members)
}

// GetKeyMeta 获取密钥元数据
func (gc *GovernanceCommittee) GetKeyMeta() *tcrsa.KeyMeta {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.keyMeta
}

// SetThreshold 设置新的阈值
func (gc *GovernanceCommittee) SetThreshold(threshold uint32) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if threshold == 0 {
		return fmt.Errorf("threshold must be greater than 0")
	}

	if int(threshold) > len(gc.members) {
		return fmt.Errorf("threshold (%d) cannot exceed member count (%d)", threshold, len(gc.members))
	}

	gc.threshold = threshold
	return nil
}
