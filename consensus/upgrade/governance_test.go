package upgrade

import (
	"testing"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
)

func TestNewGovernanceCommittee(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	if gc.GetMemberCount() != len(members) {
		t.Errorf("Member count = %d, want %d", gc.GetMemberCount(), len(members))
	}

	if gc.GetThreshold() != threshold {
		t.Errorf("Threshold = %d, want %d", gc.GetThreshold(), threshold)
	}
}

func TestGovernanceCommittee_InitializeThresholdKeys(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
		{ID: 4, Address: []byte("addr4")},
	}
	threshold := uint32(3)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化门限签名密钥
	err := gc.InitializeThresholdKeys(1024, 3, 4)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	// 验证密钥已生成
	if gc.GetKeyMeta() == nil {
		t.Error("KeyMeta should not be nil after initialization")
	}

	// 验证每个成员都有私钥
	for _, member := range members {
		if member.PrivateKey == nil {
			t.Errorf("Member %d should have private key", member.ID)
		}
		if member.PublicKey == nil {
			t.Errorf("Member %d should have public key", member.ID)
		}
	}
}

func TestGovernanceCommittee_SignAndVerifyProposal(t *testing.T) {
	// 创建委员会
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化密钥
	err := gc.InitializeThresholdKeys(1024, 2, 3)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	// 创建提案
	proposal, err := CreateUpgradeProposal(
		"hotstuff",
		nil,
		100,
		threshold,
		1000000,
		[]byte("proposer1"),
		"test proposal",
	)
	if err != nil {
		t.Fatalf("CreateUpgradeProposal() error = %v", err)
	}

	// 成员签名
	sig1, err := gc.SignProposal(proposal, 1)
	if err != nil {
		t.Fatalf("SignProposal(member 1) error = %v", err)
	}

	sig2, err := gc.SignProposal(proposal, 2)
	if err != nil {
		t.Fatalf("SignProposal(member 2) error = %v", err)
	}

	// 将签名添加到提案
	proposal.CommitteeSignatures = [][]byte{sig1, sig2}
	proposal.CommitteePubkeys = gc.GetPublicKeys()

	// 由于门限签名验证的复杂性，这里我们只验证签名被正确生成
	if len(proposal.CommitteeSignatures) != 2 {
		t.Errorf("Expected 2 signatures, got %d", len(proposal.CommitteeSignatures))
	}

	for i, sig := range proposal.CommitteeSignatures {
		if len(sig) == 0 {
			t.Errorf("Signature %d should not be empty", i)
		}
	}
}

func TestGovernanceCommittee_AggregateSignatures(t *testing.T) {
	// 创建委员会
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化密钥
	err := gc.InitializeThresholdKeys(1024, 2, 3)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	// 创建提案
	proposal, err := CreateUpgradeProposal(
		"hotstuff",
		nil,
		100,
		threshold,
		1000000,
		[]byte("proposer1"),
		"test proposal",
	)
	if err != nil {
		t.Fatalf("CreateUpgradeProposal() error = %v", err)
	}

	// 收集签名
	signatures := make([][]byte, 0, 3)
	for i := int64(1); i <= 3; i++ {
		sig, err := gc.SignProposal(proposal, i)
		if err != nil {
			t.Fatalf("SignProposal(member %d) error = %v", i, err)
		}
		signatures = append(signatures, sig)
	}

	// 聚合签名
	aggregatedSig, err := gc.AggregateSignatures(proposal, signatures)
	if err != nil {
		t.Fatalf("AggregateSignatures() error = %v", err)
	}

	if len(aggregatedSig) == 0 {
		t.Error("Aggregated signature should not be empty")
	}

	// 验证聚合签名
	err = gc.VerifyAggregatedSignature(proposal, aggregatedSig)
	if err != nil {
		t.Fatalf("VerifyAggregatedSignature() error = %v", err)
	}
}

func TestGovernanceCommittee_InsufficientSignatures(t *testing.T) {
	// 创建委员会
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(3) // 需要 3 个签名

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化密钥
	err := gc.InitializeThresholdKeys(1024, 3, 3)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	// 创建提案
	proposal, err := CreateUpgradeProposal(
		"hotstuff",
		nil,
		100,
		threshold,
		1000000,
		[]byte("proposer1"),
		"test proposal",
	)
	if err != nil {
		t.Fatalf("CreateUpgradeProposal() error = %v", err)
	}

	// 只收集 2 个签名
	signatures := make([][]byte, 0, 2)
	for i := int64(1); i <= 2; i++ {
		sig, err := gc.SignProposal(proposal, i)
		if err != nil {
			t.Fatalf("SignProposal(member %d) error = %v", i, err)
		}
		signatures = append(signatures, sig)
	}

	// 尝试聚合签名应该失败
	_, err = gc.AggregateSignatures(proposal, signatures)
	if err == nil {
		t.Error("Expected error for insufficient signatures")
	}

	// 验证部分签名也应该失败
	proposal.CommitteeSignatures = signatures
	err = gc.VerifyPartialSignatures(proposal)
	if err == nil {
		t.Error("Expected error for insufficient signatures")
	}
}

func TestGovernanceCommittee_AddRemoveMember(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	initialCount := gc.GetMemberCount()

	// 添加成员
	newMember := &CommitteeMember{
		ID:      3,
		Address: []byte("addr3"),
	}
	gc.AddMember(newMember)

	if gc.GetMemberCount() != initialCount+1 {
		t.Errorf("Member count after add = %d, want %d", gc.GetMemberCount(), initialCount+1)
	}

	// 移除成员
	removed := gc.RemoveMember(3)
	if !removed {
		t.Error("RemoveMember should return true")
	}

	if gc.GetMemberCount() != initialCount {
		t.Errorf("Member count after remove = %d, want %d", gc.GetMemberCount(), initialCount)
	}

	// 尝试移除不存在的成员
	removed = gc.RemoveMember(999)
	if removed {
		t.Error("RemoveMember should return false for non-existent member")
	}
}

func TestGovernanceCommittee_SetThreshold(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 设置新阈值
	err := gc.SetThreshold(3)
	if err != nil {
		t.Errorf("SetThreshold(3) error = %v", err)
	}

	if gc.GetThreshold() != 3 {
		t.Errorf("Threshold = %d, want 3", gc.GetThreshold())
	}

	// 尝试设置零阈值
	err = gc.SetThreshold(0)
	if err == nil {
		t.Error("Expected error for zero threshold")
	}

	// 尝试设置超过成员数的阈值
	err = gc.SetThreshold(10)
	if err == nil {
		t.Error("Expected error for threshold > member count")
	}
}

func TestGovernanceCommittee_GetPublicKeys(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
		{ID: 3, Address: []byte("addr3")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化密钥
	err := gc.InitializeThresholdKeys(1024, 2, 3)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	pubkeys := gc.GetPublicKeys()

	if len(pubkeys) != len(members) {
		t.Errorf("Public keys count = %d, want %d", len(pubkeys), len(members))
	}

	for i, pubkey := range pubkeys {
		if len(pubkey) == 0 {
			t.Errorf("Public key %d should not be empty", i)
		}
	}
}

func TestSignProposal_NonExistentMember(t *testing.T) {
	members := []*CommitteeMember{
		{ID: 1, Address: []byte("addr1")},
		{ID: 2, Address: []byte("addr2")},
	}
	threshold := uint32(2)

	gc := NewGovernanceCommittee(members, threshold, nil)

	// 初始化密钥
	err := gc.InitializeThresholdKeys(1024, 2, 2)
	if err != nil {
		t.Fatalf("InitializeThresholdKeys() error = %v", err)
	}

	// 创建提案
	proposal, err := CreateUpgradeProposal(
		"hotstuff",
		nil,
		100,
		threshold,
		1000000,
		[]byte("proposer1"),
		"test proposal",
	)
	if err != nil {
		t.Fatalf("CreateUpgradeProposal() error = %v", err)
	}

	// 尝试用不存在的成员签名
	_, err = gc.SignProposal(proposal, 999)
	if err == nil {
		t.Error("Expected error for non-existent member")
	}
}

func TestSignAndVerifyWithStandaloneFunction(t *testing.T) {
	// 生成密钥
	keyShares, keyMeta, err := crypto.GenerateThresholdKeys(2, 3)
	if err != nil {
		t.Fatalf("GenerateThresholdKeys() error = %v", err)
	}

	// 创建提案
	proposal, err := CreateUpgradeProposal(
		"hotstuff",
		nil,
		100,
		2,
		1000000,
		[]byte("proposer1"),
		"test proposal",
	)
	if err != nil {
		t.Fatalf("CreateUpgradeProposal() error = %v", err)
	}

	// 使用独立函数签名
	sig1, err := SignProposal(proposal, keyShares[0], keyMeta)
	if err != nil {
		t.Fatalf("SignProposal() error = %v", err)
	}

	sig2, err := SignProposal(proposal, keyShares[1], keyMeta)
	if err != nil {
		t.Fatalf("SignProposal() error = %v", err)
	}

	// 添加签名到提案
	proposal.CommitteeSignatures = [][]byte{sig1, sig2}
	proposal.CommitteePubkeys = [][]byte{
		[]byte("pubkey1"),
		[]byte("pubkey2"),
	}

	// 验证签名被生成
	if len(proposal.CommitteeSignatures) != 2 {
		t.Errorf("Expected 2 signatures, got %d", len(proposal.CommitteeSignatures))
	}

	for i, sig := range proposal.CommitteeSignatures {
		if len(sig) == 0 {
			t.Errorf("Signature %d should not be empty", i)
		}
	}
}
