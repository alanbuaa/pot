package upgrade

import (
	"crypto/rand"
	"testing"

	"github.com/niclabs/tcrsa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// TestCreateUpgradeProposal 测试创建升级提案
func TestCreateUpgradeProposal(t *testing.T) {
	committee := createTestCommittee(t)
	currentHeight := uint64(1000)

	proposal, err := CreateUpgradeProposal(
		"pow",
		nil,
		currentHeight,
		committee,
		1000000,
	)

	require.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.Equal(t, "pow", proposal.TargetConsensus)
	assert.Equal(t, currentHeight+100, proposal.ForkHeight)
	assert.Equal(t, currentHeight+150, proposal.PreexecStartHeight)
	assert.Equal(t, currentHeight+1150, proposal.SwitchHeight)
	assert.Equal(t, committee.GetThreshold(), proposal.Threshold)
	assert.Equal(t, uint64(1000000), proposal.Incentive)
}

// TestValidateProposalParameters 测试提案参数验证
func TestValidateProposalParameters(t *testing.T) {
	committee := createTestCommittee(t)
	currentHeight := uint64(1000)

	tests := []struct {
		name          string
		modifyProposal func(*UpgradeProposal)
		expectError   bool
	}{
		{
			name:          "valid proposal",
			modifyProposal: func(p *UpgradeProposal) {},
			expectError:   false,
		},
		{
			name: "fork height too low",
			modifyProposal: func(p *UpgradeProposal) {
				p.ForkHeight = currentHeight - 10
			},
			expectError: true,
		},
		{
			name: "preexec height before fork",
			modifyProposal: func(p *UpgradeProposal) {
				p.PreexecStartHeight = p.ForkHeight - 10
			},
			expectError: true,
		},
		{
			name: "switch height before preexec",
			modifyProposal: func(p *UpgradeProposal) {
				p.SwitchHeight = p.PreexecStartHeight - 10
			},
			expectError: true,
		},
		{
			name: "preexec phase too short",
			modifyProposal: func(p *UpgradeProposal) {
				p.SwitchHeight = p.PreexecStartHeight + 50
			},
			expectError: true,
		},
		{
			name: "empty target consensus",
			modifyProposal: func(p *UpgradeProposal) {
				p.TargetConsensus = ""
			},
			expectError: true,
		},
		{
			name: "zero threshold",
			modifyProposal: func(p *UpgradeProposal) {
				p.Threshold = 0
			},
			expectError: true,
		},
		{
			name: "insufficient committee members",
			modifyProposal: func(p *UpgradeProposal) {
				p.Threshold = uint32(len(p.CommitteePubkeys) + 1)
			},
			expectError: true,
		},
		{
			name: "nil rollback condition",
			modifyProposal: func(p *UpgradeProposal) {
				p.RollbackCondition = nil
			},
			expectError: true,
		},
		{
			name: "invalid error rate",
			modifyProposal: func(p *UpgradeProposal) {
				p.RollbackCondition.MaxErrorRate = 1.5
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, err := CreateUpgradeProposal("pow", nil, currentHeight, committee, 1000000)
			require.NoError(t, err)

			tt.modifyProposal(proposal)

			err = ValidateProposalParameters(proposal, currentHeight)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProposalSerialization 测试提案序列化
func TestProposalSerialization(t *testing.T) {
	committee := createTestCommittee(t)
	currentHeight := uint64(1000)

	original, err := CreateUpgradeProposal("pow", nil, currentHeight, committee, 1000000)
	require.NoError(t, err)

	// 转换为 protobuf
	pbProposal := original.ToProto()
	assert.NotNil(t, pbProposal)
	assert.Equal(t, original.TargetConsensus, pbProposal.TargetConsensus)
	assert.Equal(t, original.ForkHeight, pbProposal.ForkHeight)

	// 从 protobuf 转换回来
	restored := ProposalFromProto(pbProposal)
	assert.NotNil(t, restored)
	assert.Equal(t, original.ProposalID, restored.ProposalID)
	assert.Equal(t, original.TargetConsensus, restored.TargetConsensus)
	assert.Equal(t, original.ForkHeight, restored.ForkHeight)
	assert.Equal(t, original.PreexecStartHeight, restored.PreexecStartHeight)
	assert.Equal(t, original.SwitchHeight, restored.SwitchHeight)
}

// TestPackUnpackUpgradeTransaction 测试交易打包和解包
func TestPackUnpackUpgradeTransaction(t *testing.T) {
	committee := createTestCommittee(t)
	currentHeight := uint64(1000)

	proposal, err := CreateUpgradeProposal("pow", nil, currentHeight, committee, 1000000)
	require.NoError(t, err)

	// 打包为交易
	tx, err := PackUpgradeTransaction(proposal)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, pb.TransactionType_UPGRADE, tx.Type)

	// 从交易解包
	restored, err := UnpackUpgradeTransaction(tx)
	require.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, proposal.ProposalID, restored.ProposalID)
	assert.Equal(t, proposal.TargetConsensus, restored.TargetConsensus)
}

// TestUpgradeConfirmTransaction 测试升级确认交易
func TestUpgradeConfirmTransaction(t *testing.T) {
	committee := createTestCommittee(t)
	member := committee.GetMember(0)
	require.NotNil(t, member)

	proposalID := types.TxHash{}
	rand.Read(proposalID[:])

	// 创建一个私钥用于测试
	privKey := crypto.GenerateKey()

	// 创建确认交易
	tx, err := CreateUpgradeConfirmTransaction(
		proposalID,
		true,
		member.ID,
		privKey,
	)

	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, pb.TransactionType_UPGRADE, tx.Type) // 暂时使用 UPGRADE 类型

	// 注意：验证需要实际的签名实现，这里仅测试创建
}

// TestGovernanceCommittee 测试治理委员会
func TestGovernanceCommittee(t *testing.T) {
	members := createTestMembers(t, 7)
	threshold := uint32(5)

	committee, err := NewGovernanceCommittee(members, threshold)
	require.NoError(t, err)
	assert.NotNil(t, committee)
	assert.Equal(t, threshold, committee.GetThreshold())
	assert.Equal(t, 7, committee.GetMemberCount())

	// 测试获取成员
	member := committee.GetMember(0)
	assert.NotNil(t, member)
	assert.Equal(t, int64(0), member.ID)

	// 测试活跃成员
	activeCount := committee.CountActiveMembers()
	assert.Equal(t, 7, activeCount)

	// 设置成员为非活跃
	err = committee.SetMemberActive(0, false)
	require.NoError(t, err)
	activeCount = committee.CountActiveMembers()
	assert.Equal(t, 6, activeCount)
}

// TestCommitteeThresholdUpdate 测试更新阈值
func TestCommitteeThresholdUpdate(t *testing.T) {
	members := createTestMembers(t, 7)
	committee, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	// 有效的阈值更新
	err = committee.UpdateThreshold(4)
	assert.NoError(t, err)
	assert.Equal(t, uint32(4), committee.GetThreshold())

	// 无效的阈值（超过活跃成员数）
	err = committee.UpdateThreshold(10)
	assert.Error(t, err)

	// 无效的阈值（零）
	err = committee.UpdateThreshold(0)
	assert.Error(t, err)
}

// TestCommitteeMemberManagement 测试成员管理
func TestCommitteeMemberManagement(t *testing.T) {
	members := createTestMembers(t, 5)
	committee, err := NewGovernanceCommittee(members, 3)
	require.NoError(t, err)

	// 添加新成员
	newMember := &CommitteeMember{
		ID:        100,
		Name:      "New Member",
		PublicKey: []byte("new-pubkey"),
		Weight:    1,
		Active:    true,
	}

	err = committee.AddMember(newMember)
	require.NoError(t, err)
	assert.Equal(t, 6, committee.GetMemberCount())

	// 重复添加
	err = committee.AddMember(newMember)
	assert.Error(t, err)

	// 移除成员
	err = committee.RemoveMember(100)
	require.NoError(t, err)
	assert.Equal(t, 5, committee.GetMemberCount())

	// 移除不存在的成员
	err = committee.RemoveMember(999)
	assert.Error(t, err)
}

// TestCommitteeVoting 测试投票功能
func TestCommitteeVoting(t *testing.T) {
	members := createTestMembers(t, 7)
	committee, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	proposal, err := CreateUpgradeProposal("pow", nil, 1000, committee, 1000000)
	require.NoError(t, err)

	// 成员投票
	err = committee.VoteProposal(proposal, 0, true)
	assert.NoError(t, err)

	// 非活跃成员投票
	committee.SetMemberActive(1, false)
	err = committee.VoteProposal(proposal, 1, true)
	assert.Error(t, err)

	// 不存在的成员投票
	err = committee.VoteProposal(proposal, 999, true)
	assert.Error(t, err)
}

// TestCommitteeVoteCalculation 测试投票计算
func TestCommitteeVoteCalculation(t *testing.T) {
	members := createTestMembers(t, 7)
	// 设置不同的权重
	for i, member := range members {
		member.Weight = uint32(i + 1)
	}

	committee, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	// 计算投票结果
	approvals := []int64{0, 1, 2, 3, 4} // 权重: 1+2+3+4+5 = 15
	rejections := []int64{5, 6}         // 权重: 6+7 = 13

	approveWeight, rejectWeight, err := committee.CalculateVotes(approvals, rejections)
	require.NoError(t, err)
	assert.Equal(t, uint32(15), approveWeight)
	assert.Equal(t, uint32(13), rejectWeight)

	// 检查是否达到法定人数
	assert.True(t, committee.IsQuorumReached(approveWeight))
}

// TestCommitteeHash 测试委员会哈希
func TestCommitteeHash(t *testing.T) {
	members := createTestMembers(t, 7)
	committee1, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	committee2, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	// 相同配置的委员会应该有相同的哈希
	hash1 := committee1.Hash()
	hash2 := committee2.Hash()
	assert.Equal(t, hash1, hash2)

	// 不同阈值的委员会应该有不同的哈希
	committee3, err := NewGovernanceCommittee(members, 4)
	require.NoError(t, err)
	hash3 := committee3.Hash()
	assert.NotEqual(t, hash1, hash3)
}

// TestCommitteeClone 测试委员会克隆
func TestCommitteeClone(t *testing.T) {
	members := createTestMembers(t, 7)
	original, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)

	cloned := original.Clone()
	assert.NotNil(t, cloned)
	assert.Equal(t, original.GetThreshold(), cloned.GetThreshold())
	assert.Equal(t, original.GetMemberCount(), cloned.GetMemberCount())

	// 验证深拷贝
	cloned.UpdateThreshold(4)
	assert.NotEqual(t, original.GetThreshold(), cloned.GetThreshold())
}

// 辅助函数

func createTestCommittee(t *testing.T) *GovernanceCommittee {
	members := createTestMembers(t, 7)
	committee, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)
	return committee
}

func createTestCommitteeWithKeys(t *testing.T) *GovernanceCommittee {
	members := createTestMembers(t, 7)
	committee, err := NewGovernanceCommittee(members, 5)
	require.NoError(t, err)
	return committee
}

func createTestMembers(t *testing.T, count int) []*CommitteeMember {
	members := make([]*CommitteeMember, count)
	for i := 0; i < count; i++ {
		pubkey := make([]byte, 32)
		rand.Read(pubkey)

		members[i] = &CommitteeMember{
			ID:        int64(i),
			Name:      "Member " + string(rune('A'+i)),
			PublicKey: pubkey,
			Weight:    1,
			Active:    true,
		}
	}
	return members
}

func createTestMembersWithKeys(t *testing.T, count int) []*CommitteeMember {
	// 简化版本：直接返回基本成员
	return createTestMembers(t, count)
}

// TestThresholdSignature 测试门限签名（需要实际的门限签名设置）
func TestThresholdSignature(t *testing.T) {
	t.Skip("Threshold signature requires proper key generation setup")

	// 这个测试需要实际的门限签名密钥生成
	// 留待完整的门限签名集成后实现

	keySize := 2048
	threshold := uint16(3)
	totalShares := uint16(5)

	// 生成门限签名密钥
	keyMeta, keyShares, err := tcrsa.NewKey(keySize, threshold, totalShares, nil)
	require.NoError(t, err)

	// 创建带门限签名的委员会
	members := make([]*CommitteeMember, totalShares)
	for i := uint16(0); i < totalShares; i++ {
		members[i] = &CommitteeMember{
			ID:        int64(i),
			Name:      "Member " + string(rune('A'+int(i))),
			PublicKey: []byte("pubkey-" + string(rune('A'+int(i)))),
			Weight:    1,
			Active:    true,
			KeyShare:  keyShares[i],
		}
	}

	committee, err := NewGovernanceCommitteeWithThresholdSig(members, uint32(threshold), keyMeta)
	require.NoError(t, err)

	// 创建提案
	proposal, err := CreateUpgradeProposal("pow", nil, 1000, committee, 1000000)
	require.NoError(t, err)

	// 收集部分签名
	partialSigs := make([][]byte, threshold)
	for i := uint16(0); i < threshold; i++ {
		sig, err := committee.SignProposal(proposal, int64(i))
		require.NoError(t, err)
		partialSigs[i] = sig

		// 验证部分签名
		err = committee.VerifyPartialSignature(proposal, sig, int64(i))
		require.NoError(t, err)
	}

	// 组合签名
	fullSig, err := committee.CombineSignatures(proposal, partialSigs)
	require.NoError(t, err)
	assert.NotNil(t, fullSig)

	// 验证完整签名
	err = committee.VerifyFullSignature(proposal, fullSig)
	require.NoError(t, err)
}
