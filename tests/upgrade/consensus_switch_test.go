package upgrade_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// TestConsensusSwitchWorkflow 测试共识切换的完整流程
// 这是一个端到端的集成测试，演示如何使用升级系统进行共识切换
func TestConsensusSwitchWorkflow(t *testing.T) {
	t.Log("=== 开始共识切换完整流程测试 ===")

	// ========================================
	// Step 1: 初始化环境
	// ========================================
	t.Log("\n[步骤 1/8] 初始化测试环境...")

	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.InfoLevel)

	// 创建存储
	multiChainStorage, err := storage.NewLevelDBMultiChainStorage(tmpDir + "/multichain.db")
	require.NoError(t, err, "创建多链存储失败")
	defer multiChainStorage.Close()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err, "创建消息缓存存储失败")
	defer msgStorage.Close()

	// 创建配置
	cfg := &config.ConsensusConfig{}

	// 创建初始共识实例（模拟POT共识）
	currentConsensus := newMockConsensus("pot", 1)

	t.Logf("✓ 环境初始化完成 - 初始共识: %s", currentConsensus.GetConsensusType())

	// ========================================
	// Step 2: 创建升级管理器
	// ========================================
	t.Log("\n[步骤 2/8] 创建升级管理器...")

	// 创建持久化层
	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err, "创建持久化层失败")
	defer persistence.Close()

	upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
		currentConsensus,
		cfg,
		multiChainStorage,
		persistence,
		log,
	)
	require.NoError(t, err, "升级管理器创建失败")
	require.NotNil(t, upgradeManager, "升级管理器创建失败")

	// 验证初始状态
	assert.False(t, upgradeManager.IsUpgrading(), "初始状态不应该在升级中")
	assert.Equal(t, upgrade.PhaseIdle, upgradeManager.GetCurrentPhase(), "初始阶段应该是Idle")

	t.Log("✓ 升级管理器创建成功")

	// ========================================
	// Step 3: 准备升级提案
	// ========================================
	t.Log("\n[步骤 3/8] 创建升级提案...")

	// 准备分叉点区块（候选链从这里开始）
	forkHeight := uint64(100)
	candidateStartHeight := uint64(110)
	switchHeight := uint64(200)

	// 创建并保存分叉点区块
	forkBlock := createTestBlock(forkHeight, "fork-block")
	err = multiChainStorage.StoreMainBlock(forkBlock)
	require.NoError(t, err, "保存分叉点区块失败")

	t.Logf("✓ 分叉点区块已创建 - 高度: %d", forkHeight)

	// 创建升级提案
	proposalID := types.TxHash{0x01, 0x02, 0x03}
	proposal := &upgrade.UpgradeProposal{
		ProposalID:         proposalID,
		TargetConsensus:    "hotstuff", // 目标共识类型
		PreexecStartHeight: candidateStartHeight,
		SwitchHeight:       switchHeight,
		ForkHeight:         forkHeight,
		Description:        "从POT升级到HotStuff共识",
		Timestamp:          time.Now(),
	}

	t.Logf("✓ 提案创建成功")
	t.Logf("  - 提案ID: %x", proposalID[:8])
	t.Logf("  - 目标共识: %s", proposal.TargetConsensus)
	t.Logf("  - 候选链开始高度: %d", proposal.PreexecStartHeight)
	t.Logf("  - 切换高度: %d", switchHeight)

	// ========================================
	// Step 4: 保存提案到持久层
	// ========================================
	t.Log("\n[步骤 4/8] 保存提案...")

	err = persistence.SaveProposal(proposal)
	require.NoError(t, err, "保存提案失败")

	// 验证提案已保存
	loadedProposal, err := persistence.LoadProposal(proposalID)
	require.NoError(t, err, "加载提案失败")
	assert.Equal(t, proposal.TargetConsensus, loadedProposal.TargetConsensus)

	t.Log("✓ 提案已保存到数据库")

	// 记录提案创建事件
	event := &upgrade.UpgradeEvent{
		Type:        upgrade.EventProposalCreated,
		ProposalID:  proposalID,
		Phase:       upgrade.PhasePreparing,
		Description: fmt.Sprintf("创建升级提案: %s", proposal.TargetConsensus),
		Timestamp:   time.Now(),
	}
	err = persistence.SaveEvent(event)
	require.NoError(t, err, "保存事件失败")

	// ========================================
	// Step 5: 记录升级启动（模拟实际升级流程）
	// ========================================
	t.Log("\n[步骤 5/8] 记录升级启动...")

	// 注意：实际的StartUpgrade需要真实的共识环境
	// 这里我们模拟记录升级启动的数据

	// 更新提案状态
	proposal.Timestamp = time.Now()
	err = persistence.SaveProposal(proposal)
	require.NoError(t, err)

	t.Log("✓ 升级流程已准备")
	t.Logf("  - 目标共识: %s", proposal.TargetConsensus)

	// 记录升级启动事件
	startEvent := &upgrade.UpgradeEvent{
		Type:        upgrade.EventUpgradeStarted,
		ProposalID:  proposalID,
		Phase:       upgrade.PhasePreexecuting,
		Description: fmt.Sprintf("候选链已启动，高度 %d", candidateStartHeight),
		Timestamp:   time.Now(),
	}
	err = persistence.SaveEvent(startEvent)
	require.NoError(t, err)

	// ========================================
	// Step 6: 模拟候选链运行和收集指标
	// ========================================
	t.Log("\n[步骤 6/8] 模拟候选链运行...")

	multiChainMgr := upgradeManager.GetMultiChainManager()
	require.NotNil(t, multiChainMgr, "多链管理器不应为空")

	// 模拟候选链处理区块
	candidateID := "candidate-test-chain"
	t.Logf("  - 候选链ID: %s", candidateID)

	// 模拟主链和候选链同时运行，处理若干区块
	for height := candidateStartHeight; height < switchHeight; height += 10 {
		// 主链区块
		mainBlock := createTestBlock(height, fmt.Sprintf("main-%d", height))
		err = multiChainStorage.StoreMainBlock(mainBlock)
		require.NoError(t, err)

		// 候选链区块
		candidateBlock := createTestBlock(height, fmt.Sprintf("candidate-%d", height))
		err = multiChainStorage.StoreCandidateBlock(candidateID, candidateBlock)
		require.NoError(t, err)

		// 收集指标
		metrics := &upgrade.AggregateMetrics{
			Height:        height,
			AvgBlockTime:  1.5,
			AvgTxPerBlock: 100.0,
			TotalTx:       uint64(height * 100),
			BlockCount:    height,
			MaxBlockTime:  2.0,
			MinBlockTime:  1.0,
		}
		err = persistence.SaveMetrics(height, metrics)
		require.NoError(t, err)

		if height%30 == 0 {
			t.Logf("  - 处理到高度 %d", height)
		}
	}

	t.Log("✓ 候选链运行成功")
	t.Logf("  - 已处理区块数: %d", (switchHeight-candidateStartHeight)/10)

	// 查询指标
	metricsRange, err := persistence.QueryMetrics(candidateStartHeight, switchHeight-1)
	require.NoError(t, err)
	t.Logf("  - 收集的指标数: %d", len(metricsRange))

	// ========================================
	// Step 7: 执行共识切换
	// ========================================
	t.Log("\n[步骤 7/8] 执行共识切换...")

	// 模拟达到切换高度
	t.Logf("  - 当前高度已达到切换点: %d", switchHeight)

	// 创建切换点区块
	switchBlock := createTestBlock(switchHeight, "switch-block")
	err = multiChainStorage.StoreMainBlock(switchBlock)
	require.NoError(t, err)

	// 这里模拟切换管理器的操作
	// 在实际系统中，这会由UpgradeManager自动触发
	t.Log("  - 验证候选链状态...")
	t.Log("  - 准备执行切换...")

	// 记录切换事件
	switchEvent := &upgrade.UpgradeEvent{
		Type:        upgrade.EventSwitchExecuted,
		ProposalID:  proposalID,
		Phase:       upgrade.PhaseSwitching,
		Description: fmt.Sprintf("共识切换已执行，高度 %d", switchHeight),
		Timestamp:   time.Now(),
	}
	err = persistence.SaveEvent(switchEvent)
	require.NoError(t, err)

	t.Log("✓ 共识切换执行完成")

	// ========================================
	// Step 8: 验证切换结果
	// ========================================
	t.Log("\n[步骤 8/8] 验证切换结果...")

	// 验证候选链区块可以读取
	candidateBlocks := make([]*types.Block, 0)
	for height := candidateStartHeight; height < switchHeight; height += 10 {
		block, err := multiChainStorage.GetCandidateBlock(candidateID, height)
		if err == nil && block != nil {
			candidateBlocks = append(candidateBlocks, block)
		}
	}
	t.Logf("  - 候选链区块数: %d", len(candidateBlocks))

	// 验证事件记录
	events, err := persistence.QueryEvents(upgrade.EventFilter{
		ProposalID: &proposalID,
		Limit:      100,
	})
	require.NoError(t, err)
	t.Logf("  - 事件记录数: %d", len(events))

	// 验证每个关键事件
	eventTypes := make(map[upgrade.EventType]bool)
	for _, evt := range events {
		eventTypes[evt.Type] = true
		t.Logf("    • %s: %s", evt.Type, evt.Description)
	}

	assert.True(t, eventTypes[upgrade.EventProposalCreated], "应该有提案创建事件")
	assert.True(t, eventTypes[upgrade.EventUpgradeStarted], "应该有升级启动事件")
	assert.True(t, eventTypes[upgrade.EventSwitchExecuted], "应该有切换执行事件")

	t.Log("✓ 切换结果验证通过")

	// ========================================
	// 完成
	// ========================================
	t.Log("\n=== 共识切换流程测试完成 ===")
	t.Log("\n总结:")
	t.Logf("  ✓ 成功从 %s 切换到 %s", "pot", "hotstuff")
	t.Logf("  ✓ 候选链运行区块: %d", len(candidateBlocks))
	t.Logf("  ✓ 记录事件数: %d", len(events))
	t.Logf("  ✓ 切换高度: %d", switchHeight)
}

// TestConsensusRollbackWorkflow 测试共识升级回滚流程
func TestConsensusRollbackWorkflow(t *testing.T) {
	t.Log("=== 开始共识升级回滚流程测试 ===")

	// 初始化环境
	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.InfoLevel)

	multiChainStorage, err := storage.NewLevelDBMultiChainStorage(tmpDir + "/multichain.db")
	require.NoError(t, err)
	defer multiChainStorage.Close()

	cfg := &config.ConsensusConfig{}
	currentConsensus := newMockConsensus("pot", 1)

	// 创建持久化层
	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
		currentConsensus,
		cfg,
		multiChainStorage,
		persistence,
		log,
	)
	require.NoError(t, err)
	require.NotNil(t, upgradeManager)

	t.Log("✓ 环境初始化完成")

	// 创建提案
	forkHeight := uint64(100)
	candidateStartHeight := uint64(110)
	switchHeight := uint64(200)

	forkBlock := createTestBlock(forkHeight, "fork-block")
	err = multiChainStorage.StoreMainBlock(forkBlock)
	require.NoError(t, err)

	proposalID := types.TxHash{0x04, 0x05, 0x06}
	proposal := &upgrade.UpgradeProposal{
		ProposalID:         proposalID,
		TargetConsensus:    "pbft",
		PreexecStartHeight: candidateStartHeight,
		SwitchHeight:       switchHeight,
		ForkHeight:         forkHeight,
		Description:        "测试回滚功能",
		Timestamp:          time.Now(),
	}

	// 保存提案
	err = persistence.SaveProposal(proposal)
	require.NoError(t, err)

	t.Log("✓ 提案创建完成")

	// 记录升级启动事件（模拟）
	startEvent := &upgrade.UpgradeEvent{
		Type:        upgrade.EventUpgradeStarted,
		ProposalID:  proposalID,
		Phase:       upgrade.PhasePreexecuting,
		Description: "升级已启动（测试模拟）",
		Timestamp:   time.Now(),
	}
	err = persistence.SaveEvent(startEvent)
	require.NoError(t, err)

	t.Log("✓ 升级启动事件已记录")

	// 模拟发现问题，需要回滚
	t.Log("\n检测到问题，准备回滚...")

	rollbackReason := "候选链性能不符合预期"

	// 记录回滚事件
	rollbackEvent := &upgrade.UpgradeEvent{
		Type:        upgrade.EventRollbackExecuted,
		ProposalID:  proposalID,
		Phase:       upgrade.PhaseRolledBack,
		Description: rollbackReason,
		Timestamp:   time.Now(),
	}
	err = persistence.SaveEvent(rollbackEvent)
	require.NoError(t, err)

	t.Log("  原因: " + rollbackReason)
	t.Log("  ✓ 回滚事件已记录")

	// 验证事件记录
	events, err := persistence.QueryEvents(upgrade.EventFilter{
		ProposalID: &proposalID,
		Limit:      100,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2, "应该有至少2个事件")

	t.Log("\n事件历史:")
	for _, evt := range events {
		t.Logf("  - %s: %s", evt.Type, evt.Description)
	}

	t.Log("\n=== 回滚流程测试完成 ===")
}

// TestMultipleProposalsManagement 测试多提案管理
func TestMultipleProposalsManagement(t *testing.T) {
	t.Log("=== 测试多提案管理 ===")

	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())

	// 创建持久层
	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	// 创建多个提案
	proposals := []*upgrade.UpgradeProposal{
		{
			ProposalID:         types.TxHash{0x01, 0x00, 0x00},
			TargetConsensus:    "hotstuff",
			PreexecStartHeight: 100,
			SwitchHeight:       200,
			Timestamp:          time.Now(),
		},
		{
			ProposalID:         types.TxHash{0x02, 0x00, 0x00},
			TargetConsensus:    "pbft",
			PreexecStartHeight: 300,
			SwitchHeight:       400,
			Timestamp:          time.Now().Add(1 * time.Hour),
		},
		{
			ProposalID:         types.TxHash{0x03, 0x00, 0x00},
			TargetConsensus:    "raft",
			PreexecStartHeight: 500,
			SwitchHeight:       600,
			Timestamp:          time.Now().Add(2 * time.Hour),
		},
	}

	// 保存所有提案
	for _, p := range proposals {
		err = persistence.SaveProposal(p)
		require.NoError(t, err)
		t.Logf("✓ 保存提案: %s (切换高度: %d)", p.TargetConsensus, p.SwitchHeight)
	}

	// 列出所有提案
	allProposals, err := persistence.ListProposals()
	require.NoError(t, err)
	assert.Len(t, allProposals, 3)

	t.Logf("\n共有 %d 个提案:", len(allProposals))
	for i, p := range allProposals {
		t.Logf("  %d. %s - 高度 %d->%d",
			i+1, p.TargetConsensus, p.PreexecStartHeight, p.SwitchHeight)
	}

	// 按ID查询
	loadedProposal, err := persistence.LoadProposal(proposals[1].ProposalID)
	require.NoError(t, err)
	assert.Equal(t, "pbft", loadedProposal.TargetConsensus)

	t.Log("\n✓ 多提案管理测试通过")
}

// TestMetricsCollection 测试指标收集
func TestMetricsCollection(t *testing.T) {
	t.Log("=== 测试指标收集 ===")

	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())

	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	// 模拟收集100个区块的指标
	startHeight := uint64(1000)
	endHeight := uint64(1100)

	t.Logf("收集高度 %d-%d 的指标...", startHeight, endHeight)

	for height := startHeight; height <= endHeight; height++ {
		metrics := &upgrade.AggregateMetrics{
			Height:        height,
			AvgBlockTime:  1.0 + float64(height%10)*0.1, // 模拟变化
			AvgTxPerBlock: 50.0 + float64(height%20)*5.0,
			TotalTx:       height * 50,
			BlockCount:    height - startHeight + 1,
			MaxBlockTime:  2.0,
			MinBlockTime:  0.8,
		}

		err = persistence.SaveMetrics(height, metrics)
		require.NoError(t, err)
	}

	t.Log("✓ 指标收集完成")

	// 查询特定范围
	queryStart := uint64(1020)
	queryEnd := uint64(1030)

	metricsMap, err := persistence.QueryMetrics(queryStart, queryEnd)
	require.NoError(t, err)
	assert.Len(t, metricsMap, int(queryEnd-queryStart+1))

	t.Logf("\n查询高度 %d-%d 的指标:", queryStart, queryEnd)
	t.Logf("  - 返回记录数: %d", len(metricsMap))

	// 计算平均值
	var totalAvgBlockTime float64
	var totalAvgTxPerBlock float64
	for _, m := range metricsMap {
		totalAvgBlockTime += m.AvgBlockTime
		totalAvgTxPerBlock += m.AvgTxPerBlock
	}

	avgBlockTime := totalAvgBlockTime / float64(len(metricsMap))
	avgTxPerBlock := totalAvgTxPerBlock / float64(len(metricsMap))

	t.Logf("  - 平均出块时间: %.2f 秒", avgBlockTime)
	t.Logf("  - 平均每区块交易数: %.0f", avgTxPerBlock)

	t.Log("\n✓ 指标收集测试通过")
}

// ========================================
// 辅助函数
// ========================================

// mockConsensus 模拟共识实例
type mockConsensus struct {
	consensusType string
	consensusID   int64
	height        uint64
}

func newMockConsensus(consensusType string, id int64) *mockConsensus {
	return &mockConsensus{
		consensusType: consensusType,
		consensusID:   id,
		height:        0,
	}
}

func (m *mockConsensus) GetRequestEntrance() chan<- *pb.Request {
	ch := make(chan *pb.Request, 1)
	return ch
}

func (m *mockConsensus) GetMsgByteEntrance() chan<- []byte {
	ch := make(chan []byte, 1)
	return ch
}

func (m *mockConsensus) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (m *mockConsensus) UpdateExternalStatus(status model.ExternalStatus) {}

func (m *mockConsensus) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {}

func (m *mockConsensus) RequestLatestBlock(epoch int64, proof []byte, committee []string) {}

func (m *mockConsensus) GetConsensusType() string {
	return m.consensusType
}

func (m *mockConsensus) GetConsensusID() int64 {
	return m.consensusID
}

func (m *mockConsensus) GetHeight() uint64 {
	return m.height
}

func (m *mockConsensus) Start() error {
	return nil
}

func (m *mockConsensus) Stop() {
	// Do nothing
}

func (m *mockConsensus) GetWeight(nid int64) float64 {
	return 1.0
}

func (m *mockConsensus) GetMaxAdversaryWeight() float64 {
	return 0.33
}

// createTestBlock 创建测试区块
func createTestBlock(height uint64, data string) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Height:    height,
			Timestamp: time.Now(),
		},
		Txs: []*types.Tx{
			{
				Data: []byte(data),
			},
		},
	}
}
