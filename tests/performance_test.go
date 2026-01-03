package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// BenchmarkConsensusFactoryCreation 基准测试：共识工厂创建实例
func BenchmarkConsensusFactoryCreation(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	factory := upgrade.NewConsensusFactory(1, nil, nil, log)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{1, 2, 3},
		TargetConsensus: "hotstuff",
		PrepareHeight:   1000,
		PreexecHeight:   2000,
		SwitchHeight:    3000,
	}

	baseConfig := &upgrade.ConsensusConfig{
		NodeID:       1,
		BlockTime:    time.Second,
		MaxBlockSize: 1024 * 1024,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateConsensus(proposal, baseConfig)
	}
}

// BenchmarkProposalValidation 基准测试：提案验证
func BenchmarkProposalValidation(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	factory := upgrade.NewConsensusFactory(1, nil, nil, log)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{4, 5, 6},
		TargetConsensus: "pow",
		PrepareHeight:   1000,
		PreexecHeight:   2000,
		SwitchHeight:    3000,
		Threshold:       7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.ValidateProposal(proposal)
	}
}

// BenchmarkVotingManagerStartVoting 基准测试：启动投票
func BenchmarkVotingManagerStartVoting(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proposal := &upgrade.UpgradeProposal{
			ProposalID:      types.TxHash{byte(i), byte(i >> 8), byte(i >> 16)},
			TargetConsensus: "hotstuff",
			Threshold:       7,
		}
		vm.StartVoting(proposal, 1*time.Hour)
	}
}

// BenchmarkVoteCastingSequential 基准测试：顺序投票
func BenchmarkVoteCastingSequential(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{7, 8, 9},
		TargetConsensus: "pot",
		Threshold:       int64(b.N),
	}

	vm.StartVoting(proposal, 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.CastVote(proposal.ProposalID, int64(i), upgrade.VoteYes, []byte("sig"))
	}
}

// BenchmarkVoteCastingWithPersistence 基准测试：带持久化的投票
func BenchmarkVoteCastingWithPersistence(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	vm := upgrade.NewVotingManagerWithStorage(upgradeStorage, logrus.NewEntry(logrus.New()))

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{10, 11, 12},
		TargetConsensus: "whirly",
		Threshold:       int64(b.N),
	}

	vm.StartVoting(proposal, 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.CastVote(proposal.ProposalID, int64(i), upgrade.VoteYes, []byte("sig"))
	}
}

// BenchmarkQuorumCheck 基准测试：法定人数检查
func BenchmarkQuorumCheck(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{13, 14, 15},
		TargetConsensus: "hotstuff",
		Threshold:       100,
	}

	vm.StartVoting(proposal, 1*time.Hour)

	// 预先投票
	for i := int64(0); i < 100; i++ {
		vm.CastVote(proposal.ProposalID, i, upgrade.VoteYes, []byte("sig"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.GetVotingResult(proposal.ProposalID)
	}
}

// BenchmarkCheckpointCreationMemory 基准测试：内存检查点创建
func BenchmarkCheckpointCreationMemory(b *testing.B) {
	rm := upgrade.NewRollbackManager(nil, nil, logrus.NewEntry(logrus.New()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.CreateCheckpoint(uint64(i), "hotstuff", []byte("state_hash"))
	}
}

// BenchmarkCheckpointCreationPersistent 基准测试：持久化检查点创建
func BenchmarkCheckpointCreationPersistent(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	rm := upgrade.NewRollbackManagerWithStorage(nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.CreateCheckpoint(uint64(i), "hotstuff", []byte("state_hash"))
	}
}

// BenchmarkCheckpointRetrieval 基准测试：检查点检索
func BenchmarkCheckpointRetrieval(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	rm := upgrade.NewRollbackManagerWithStorage(nil, nil, upgradeStorage, logrus.NewEntry(logrus.New()))

	// 预先创建检查点
	for i := uint64(0); i < 1000; i += 100 {
		rm.CreateCheckpoint(i, "hotstuff", []byte("state_hash"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		height := uint64((i % 10) * 100)
		rm.LoadCheckpoint(height)
	}
}

// BenchmarkMessageBuffering 基准测试：消息缓存（已在 bufmsg_test.go 中实现）

// BenchmarkStorageWrite 基准测试：存储写入
func BenchmarkStorageWrite(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkpoint := &storage.RollbackCheckpoint{
			Height:        uint64(i),
			ConsensusType: "hotstuff",
			Timestamp:     time.Now(),
			StateHash:     []byte("hash"),
			BlockHash:     []byte("block"),
			Description:   "test",
		}
		upgradeStorage.StoreCheckpoint(checkpoint)
	}
}

// BenchmarkStorageRead 基准测试：存储读取
func BenchmarkStorageRead(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	// 预先写入数据
	for i := 0; i < 1000; i++ {
		checkpoint := &storage.RollbackCheckpoint{
			Height:        uint64(i),
			ConsensusType: "hotstuff",
			Timestamp:     time.Now(),
			StateHash:     []byte("hash"),
			BlockHash:     []byte("block"),
			Description:   "test",
		}
		upgradeStorage.StoreCheckpoint(checkpoint)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		height := uint64(i % 1000)
		upgradeStorage.GetCheckpoint(height)
	}
}

// BenchmarkProposalStatusUpdate 基准测试：提案状态更新
func BenchmarkProposalStatusUpdate(b *testing.B) {
	tmpDir := b.TempDir()
	upgradeStorage, err := storage.NewLevelDBUpgradeStorage(tmpDir + "/upgrade.db")
	if err != nil {
		b.Fatal(err)
	}
	defer upgradeStorage.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proposalID := types.TxHash{byte(i), byte(i >> 8), byte(i >> 16)}
		status := &storage.ProposalVoteStatus{
			ProposalID:    proposalID,
			Status:        0,
			StartTime:     time.Now(),
			EndTime:       time.Now().Add(1 * time.Hour),
			VotingPeriod:  int64(time.Hour),
			YesCount:      5,
			NoCount:       2,
			AbstainCount:  0,
			QuorumReached: true,
		}
		upgradeStorage.StoreProposalStatus(proposalID, status)
	}
}

// BenchmarkConcurrentVoting 基准测试：并发投票
func BenchmarkConcurrentVoting(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:      types.TxHash{16, 17, 18},
		TargetConsensus: "pot",
		Threshold:       int64(b.N),
	}

	vm.StartVoting(proposal, 1*time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		nodeID := int64(0)
		for pb.Next() {
			vm.CastVote(proposal.ProposalID, nodeID, upgrade.VoteYes, []byte("sig"))
			nodeID++
		}
	})
}

// BenchmarkMultiProposalHandling 基准测试：多提案处理
func BenchmarkMultiProposalHandling(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	// 创建多个提案
	proposals := make([]*upgrade.UpgradeProposal, 100)
	for i := 0; i < 100; i++ {
		proposals[i] = &upgrade.UpgradeProposal{
			ProposalID:      types.TxHash{byte(i), byte(i >> 8), 0},
			TargetConsensus: "hotstuff",
			Threshold:       7,
		}
		vm.StartVoting(proposals[i], 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proposalIdx := i % 100
		nodeID := int64(i / 100)
		vm.CastVote(proposals[proposalIdx].ProposalID, nodeID, upgrade.VoteYes, []byte("sig"))
	}
}

// BenchmarkSignatureVerification 基准测试：签名验证
func BenchmarkSignatureVerification(b *testing.B) {
	log := logrus.NewEntry(logrus.New())
	vm := upgrade.NewVotingManager(log)

	proposalID := types.TxHash{19, 20, 21}
	signature := []byte("test_signature_data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.VerifyVoteSignature(proposalID, int64(i), signature)
	}
}

// 性能报告生成
func BenchmarkGeneratePerformanceReport(b *testing.B) {
	b.Skip("Manual run only - generates performance report")

	fmt.Println("=== Phase 7 Performance Report ===\n")

	// 运行各项基准测试并收集结果
	benchmarks := []struct {
		name string
		fn   func(*testing.B)
	}{
		{"ConsensusFactoryCreation", BenchmarkConsensusFactoryCreation},
		{"ProposalValidation", BenchmarkProposalValidation},
		{"VotingManagerStartVoting", BenchmarkVotingManagerStartVoting},
		{"VoteCastingSequential", BenchmarkVoteCastingSequential},
		{"QuorumCheck", BenchmarkQuorumCheck},
		{"CheckpointCreationMemory", BenchmarkCheckpointCreationMemory},
		{"CheckpointCreationPersistent", BenchmarkCheckpointCreationPersistent},
		{"StorageWrite", BenchmarkStorageWrite},
		{"StorageRead", BenchmarkStorageRead},
	}

	for _, bm := range benchmarks {
		result := testing.Benchmark(bm.fn)
		fmt.Printf("%s:\n", bm.name)
		fmt.Printf("  Operations: %d\n", result.N)
		fmt.Printf("  Time per op: %v\n", result.NsPerOp())
		fmt.Printf("  Memory per op: %d bytes\n", result.AllocedBytesPerOp())
		fmt.Println()
	}
}
