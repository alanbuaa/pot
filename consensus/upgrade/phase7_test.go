package upgrade

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// TestCheckpointPersistence 测试检查点持久化
func TestCheckpointPersistence(t *testing.T) {
	// 创建临时数据库
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	// 创建 RollbackManager
	candidateID := "test-candidate"
	rm := NewRollbackManagerWithStorage(candidateID, nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建检查点
	checkpoint := rm.CreateCheckpoint(1000, "hotstuff", []byte("state_hash_1000"))
	assert.NotNil(t, checkpoint)
	assert.Equal(t, uint64(1000), checkpoint.Height)

	// 从存储加载检查点
	loadedCP, err := rm.LoadCheckpoint(1000)
	require.NoError(t, err)
	assert.Equal(t, checkpoint.Height, loadedCP.Height)
	assert.Equal(t, checkpoint.ConsensusType, loadedCP.ConsensusType)
	assert.Equal(t, checkpoint.StateHash, loadedCP.StateHash)

	// 测试最新检查点
	checkpoint2 := rm.CreateCheckpoint(1100, "pot", []byte("state_hash_1100"))
	latestCP, err := rm.LoadLatestCheckpoint()
	require.NoError(t, err)
	assert.Equal(t, checkpoint2.Height, latestCP.Height)
}

// TestCheckpointList 测试检查点列表
func TestCheckpointList(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	candidateID := "test-candidate"
	rm := NewRollbackManagerWithStorage(candidateID, nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建多个检查点
	for i := uint64(100); i <= 500; i += 100 {
		rm.CreateCheckpoint(i, "hotstuff", []byte("state_hash"))
	}

	// 列出检查点
	checkpoints, err := rm.ListCheckpoints(10)
	require.NoError(t, err)
	assert.Equal(t, 5, len(checkpoints))

	// 检查顺序（应该从最新到最旧）
	assert.Equal(t, uint64(500), checkpoints[0].Height)
	assert.Equal(t, uint64(100), checkpoints[4].Height)
}

// TestCheckpointDeletion 测试检查点删除
func TestCheckpointDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	candidateID := "test-candidate"
	rm := NewRollbackManagerWithStorage(candidateID, nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建检查点
	for i := uint64(100); i <= 500; i += 100 {
		rm.CreateCheckpoint(i, "hotstuff", []byte("state_hash"))
	}

	// 删除300之后的检查点
	err = upgradeStorage.DeleteCheckpointsAfter(300)
	require.NoError(t, err)

	// 验证
	checkpoints, err := rm.ListCheckpoints(10)
	require.NoError(t, err)
	assert.Equal(t, 3, len(checkpoints))

	// 最新检查点应该是300
	latestCP, err := rm.LoadLatestCheckpoint()
	require.NoError(t, err)
	assert.Equal(t, uint64(300), latestCP.Height)
}

// TestVotePersistence 测试投票持久化
func TestVotePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	vm := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建提案
	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{1, 2, 3},
		TargetConsensus: "pow",
		Threshold:       7,
	}

	// 启动投票
	err = vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 投票
	err = vm.CastVote(proposal.ProposalID, 1, VoteYes, []byte("signature1"))
	require.NoError(t, err)

	err = vm.CastVote(proposal.ProposalID, 2, VoteYes, []byte("signature2"))
	require.NoError(t, err)

	err = vm.CastVote(proposal.ProposalID, 3, VoteNo, []byte("signature3"))
	require.NoError(t, err)

	// 验证投票已持久化
	voteRecord, err := upgradeStorage.GetVote(proposal.ProposalID, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), voteRecord.NodeID)
	assert.Equal(t, int(VoteYes), voteRecord.Option)

	// 获取所有投票
	votes, err := upgradeStorage.GetProposalVotes(proposal.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(votes))
}

// TestProposalStatusPersistence 测试提案状态持久化
func TestProposalStatusPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	vm := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建提案
	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{4, 5, 6},
		TargetConsensus: "hotstuff",
		Threshold:       7,
	}

	// 启动投票
	err = vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 投5票通过
	for i := int64(1); i <= 5; i++ {
		err = vm.CastVote(proposal.ProposalID, i, VoteYes, []byte("sig"))
		require.NoError(t, err)
	}

	// 等待状态更新
	time.Sleep(100 * time.Millisecond)

	// 验证提案状态已持久化
	status, err := upgradeStorage.GetProposalStatus(proposal.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, int(VoteStatusPassed), status.Status)
	assert.Equal(t, 5, status.YesCount)
	assert.True(t, status.QuorumReached)
}

// TestLoadProposalStatus 测试加载提案状态
func TestLoadProposalStatus(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	vm := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	// 创建提案并投票
	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{7, 8, 9},
		TargetConsensus: "pot",
		Threshold:       7,
	}

	err = vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	for i := int64(1); i <= 3; i++ {
		err = vm.CastVote(proposal.ProposalID, i, VoteYes, []byte("sig"))
		require.NoError(t, err)
	}

	// 创建新的 VotingManager 并加载状态
	vm2 := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))
	loadedVote, err := vm2.LoadProposalStatus(proposal.ProposalID)
	require.NoError(t, err)

	assert.Equal(t, 3, len(loadedVote.Votes))
	assert.Equal(t, 3, loadedVote.VoteCount[VoteYes])
	assert.Equal(t, VoteStatusPending, loadedVote.Status)
}

// TestVoteSignatureVerification 测试投票签名验证
func TestVoteSignatureVerification(t *testing.T) {
	vm := NewVotingManager(logrus.NewEntry(logrus.New()))

	proposalID := types.TxHash{10, 11, 12}

	// 测试空签名
	err := vm.VerifyVoteSignature(proposalID, 1, []byte{})
	assert.Error(t, err)

	// 测试非空签名（当前实现总是接受）
	err = vm.VerifyVoteSignature(proposalID, 1, []byte("signature"))
	assert.NoError(t, err)
}

// TestCheckpointStrategy 测试检查点创建策略
func TestCheckpointStrategy(t *testing.T) {
	candidateID := "test-candidate"
	rm := NewRollbackManager(candidateID, nil, nil, logrus.NewEntry(logrus.New()))

	proposalHeight := uint64(1000)

	// 预执行开始前应创建检查点
	assert.True(t, rm.ShouldCreateCheckpoint(999, proposalHeight))

	// 每100个区块应创建检查点
	assert.True(t, rm.ShouldCreateCheckpoint(100, proposalHeight))
	assert.True(t, rm.ShouldCreateCheckpoint(200, proposalHeight))
	assert.True(t, rm.ShouldCreateCheckpoint(300, proposalHeight))

	// 其他高度不应创建
	assert.False(t, rm.ShouldCreateCheckpoint(101, proposalHeight))
	assert.False(t, rm.ShouldCreateCheckpoint(998, proposalHeight))
}

// TestVotingQuorum 测试投票法定人数
func TestVotingQuorum(t *testing.T) {
	vm := NewVotingManager(logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{13, 14, 15},
		TargetConsensus: "whirly",
		Threshold:       7,
	}

	err := vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 投5票赞成（超过2/3 * 7 = 4.67）
	for i := int64(1); i <= 5; i++ {
		err = vm.CastVote(proposal.ProposalID, i, VoteYes, []byte("sig"))
		require.NoError(t, err)
	}

	result, err := vm.GetVotingResult(proposal.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, VoteStatusPassed, result.Status)
	assert.True(t, result.QuorumReached)
}

// TestVotingRejection 测试提案被拒绝
func TestVotingRejection(t *testing.T) {
	vm := NewVotingManager(logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{16, 17, 18},
		TargetConsensus: "pow",
		Threshold:       7,
	}

	err := vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 投4票反对（达到拒绝阈值）
	for i := int64(1); i <= 4; i++ {
		err = vm.CastVote(proposal.ProposalID, i, VoteNo, []byte("sig"))
		require.NoError(t, err)
	}

	result, err := vm.GetVotingResult(proposal.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, VoteStatusRejected, result.Status)
}

// TestDoubleVoting 测试防止重复投票
func TestDoubleVoting(t *testing.T) {
	vm := NewVotingManager(logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{19, 20, 21},
		TargetConsensus: "hotstuff",
		Threshold:       7,
	}

	err := vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 第一次投票
	err = vm.CastVote(proposal.ProposalID, 1, VoteYes, []byte("sig1"))
	require.NoError(t, err)

	// 第二次投票应该失败
	err = vm.CastVote(proposal.ProposalID, 1, VoteNo, []byte("sig2"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already voted")
}

// TestVotingExpiry 测试投票过期
func TestVotingExpiry(t *testing.T) {
	vm := NewVotingManager(logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{22, 23, 24},
		TargetConsensus: "pot",
		Threshold:       7,
	}

	// 启动短期投票
	err := vm.StartVoting(proposal, 100*time.Millisecond)
	require.NoError(t, err)

	// 等待投票过期
	time.Sleep(200 * time.Millisecond)

	// 过期后投票应该失败
	err = vm.CastVote(proposal.ProposalID, 1, VoteYes, []byte("sig"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestProposalDataDeletion 测试提案数据删除
func TestProposalDataDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	require.NoError(t, err)
	defer upgradeStorage.Close()

	vm := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{25, 26, 27},
		TargetConsensus: "whirly",
		Threshold:       7,
	}

	err = vm.StartVoting(proposal, 1*time.Hour)
	require.NoError(t, err)

	// 投票
	err = vm.CastVote(proposal.ProposalID, 1, VoteYes, []byte("sig"))
	require.NoError(t, err)

	// 删除提案数据
	err = upgradeStorage.DeleteProposalData(proposal.ProposalID)
	require.NoError(t, err)

	// 验证数据已删除
	_, err = upgradeStorage.GetProposalStatus(proposal.ProposalID)
	assert.Error(t, err)

	votes, err := upgradeStorage.GetProposalVotes(proposal.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(votes))
}

// BenchmarkCheckpointCreation 基准测试：检查点创建
func BenchmarkCheckpointCreation(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	candidateID := "test-candidate"
	rm := NewRollbackManagerWithStorage(candidateID, nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.CreateCheckpoint(uint64(i), "hotstuff", []byte("state_hash"))
	}
}

// BenchmarkVoteCasting 基准测试：投票
func BenchmarkVoteCasting(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	vm := NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	proposal := &UpgradeProposal{
		ProposalID:      types.TxHash{28, 29, 30},
		TargetConsensus: "pow",
		Threshold:       uint32(b.N),
	}

	err = vm.StartVoting(proposal, 1*time.Hour)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.CastVote(proposal.ProposalID, int64(i), VoteYes, []byte("sig"))
	}
}

// TestMain 测试入口
func TestMain(m *testing.M) {
	// 设置日志级别
	logrus.SetLevel(logrus.WarnLevel)

	// 运行测试
	code := m.Run()

	os.Exit(code)
}
