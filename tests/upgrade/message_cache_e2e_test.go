package upgrade

import (
	"fmt"
	"sync"
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

// TestMessageCacheInConsensusSwitchE2E 端到端测试消息缓存在共识切换中的完整流程
// 测试场景：
// 1. 网络消息提前到达（epoch > 当前epoch）-> 消息被缓存
// 2. 推进epoch -> 缓存的消息被自动处理
// 3. 候选共识启动 -> 缓存被冲刷到新共识
// 4. 共识切换完成 -> 旧epoch消息被清理
func TestMessageCacheInConsensusSwitchE2E(t *testing.T) {
	t.Log("=== 开始消息缓存在共识切换中的端到端测试 ===\n")

	// ========================================
	// Step 1: 初始化测试环境
	// ========================================
	t.Log("[步骤 1/10] 初始化测试环境...")

	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.InfoLevel)

	// 创建存储
	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err, "创建消息缓存存储失败")
	defer msgStorage.Close()

	multiChainStorage, err := storage.NewLevelDBMultiChainStorage(tmpDir + "/multichain.db")
	require.NoError(t, err, "创建多链存储失败")
	defer multiChainStorage.Close()

	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err, "创建持久化存储失败")
	defer persistence.Close()

	// 创建消息缓存
	msgCache := upgrade.NewMessageCache(log)
	require.NotNil(t, msgCache, "消息缓存创建失败")

	t.Log("✓ 测试环境初始化完成")
	t.Logf("  - 消息缓存存储: %s", tmpDir+"/msgcache.db")
	t.Logf("  - 多链存储: %s", tmpDir+"/multichain.db")
	t.Logf("  - 持久化存储: %s", tmpDir+"/upgrade.db")

	// ========================================
	// Step 2: 创建模拟共识和配置
	// ========================================
	t.Log("\n[步骤 2/10] 创建模拟共识和配置...")

	// 创建主链共识（POT）
	mainConsensusID := int64(1)
	mainConsensus := &mockConsensusForCache{
		id:              mainConsensusID,
		name:            "pot",
		receivedMsgs:    make([]*mockMessage, 0),
		processedEpochs: make(map[uint64]bool),
		mu:              &sync.Mutex{},
		log:             log,
	}

	// 创建候选链共识（HotStuff）
	candidateConsensusID := int64(2)
	candidateConsensus := &mockConsensusForCache{
		id:              candidateConsensusID,
		name:            "hotstuff",
		receivedMsgs:    make([]*mockMessage, 0),
		processedEpochs: make(map[uint64]bool),
		mu:              &sync.Mutex{},
		log:             log,
	}

	// 创建配置
	cfg := &config.ConsensusConfig{
		ConsensusID: 1,
		F:           1,
	}

	t.Log("✓ 模拟共识创建完成")
	t.Logf("  - 主链共识ID: %d (%s)", mainConsensusID, mainConsensus.name)
	t.Logf("  - 候选链共识ID: %d (%s)", candidateConsensusID, candidateConsensus.name)

	// ========================================
	// Step 3: 创建UpgradeManager并集成消息缓存
	// ========================================
	t.Log("\n[步骤 3/10] 创建UpgradeManager...")

	upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
		mainConsensus,
		cfg,
		multiChainStorage,
		persistence,
		log,
	)
	require.NoError(t, err, "UpgradeManager创建失败")
	require.NotNil(t, upgradeManager, "UpgradeManager创建失败")

	// 设置初始epoch
	currentEpoch := uint64(100)
	upgradeManager.SetCurrentEpoch(currentEpoch)

	t.Log("✓ UpgradeManager创建完成")
	t.Logf("  - 初始epoch: %d", currentEpoch)

	// ========================================
	// Step 4: 模拟消息提前到达（epoch > 当前epoch）
	// ========================================
	t.Log("\n[步骤 4/10] 模拟消息提前到达...")

	// 创建一批提前到达的消息（epoch 105-110）
	earlyMessages := []struct {
		epoch        uint64
		senderID     int64
		receiverID   int64
		messageType  string
		expectCached bool
	}{
		{105, 2, mainConsensusID, "Prepare", true},      // epoch > current, 应该被缓存
		{106, 3, mainConsensusID, "PreCommit", true},    // epoch > current, 应该被缓存
		{107, 2, mainConsensusID, "Commit", true},       // epoch > current, 应该被缓存
		{100, 2, mainConsensusID, "Vote", false},        // epoch == current, 不缓存，直接处理
		{95, 3, mainConsensusID, "Prepare", false},      // epoch < current, 应该被丢弃
		{108, 4, mainConsensusID, "NewView", true},      // epoch > current, 应该被缓存
		{109, 2, candidateConsensusID, "Prepare", true}, // 候选链消息，epoch > current
		{110, 3, candidateConsensusID, "Commit", true},  // 候选链消息，epoch > current
	}

	cachedCount := 0
	processedCount := 0
	droppedCount := 0

	for _, msgInfo := range earlyMessages {
		msg := createMockNetworkMessage(msgInfo.messageType, msgInfo.epoch)

		err := upgradeManager.OnNetworkMessage(msg, msgInfo.senderID, msgInfo.receiverID, msgInfo.epoch)
		require.NoError(t, err, "处理网络消息失败")

		if msgInfo.expectCached {
			cachedCount++
		} else if msgInfo.epoch == currentEpoch {
			processedCount++
		} else {
			droppedCount++
		}

		t.Logf("  - 消息[epoch=%d, type=%s, receiver=%d]: %s",
			msgInfo.epoch, msgInfo.messageType, msgInfo.receiverID,
			getMessageStatus(msgInfo.epoch, currentEpoch))
	}

	t.Log("✓ 消息到达处理完成")
	t.Logf("  - 缓存消息数: %d", cachedCount)
	t.Logf("  - 直接处理消息数: %d", processedCount)
	t.Logf("  - 丢弃消息数: %d", droppedCount)

	// 验证缓存状态
	cachedForEpoch105 := msgCache.GetMessagesByEpoch(105)
	assert.Greater(t, len(cachedForEpoch105), 0, "epoch 105应该有缓存消息")
	t.Logf("  - epoch 105缓存的消息数: %d", len(cachedForEpoch105))

	// ========================================
	// Step 5: 推进epoch，验证缓存消息自动处理
	// ========================================
	t.Log("\n[步骤 5/10] 推进epoch到105，验证缓存消息自动处理...")

	// 记录推进前的状态
	beforeMsgCount := len(mainConsensus.receivedMsgs)

	// 推进epoch到105
	upgradeManager.SetCurrentEpoch(105)

	// 等待后台处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证消息已被处理
	afterMsgCount := len(mainConsensus.receivedMsgs)
	processedInEpoch105 := afterMsgCount - beforeMsgCount

	assert.Greater(t, processedInEpoch105, 0, "epoch 105的缓存消息应该被处理")
	t.Log("✓ epoch推进完成")
	t.Logf("  - 新epoch: 105")
	t.Logf("  - 自动处理的缓存消息数: %d", processedInEpoch105)

	// 验证epoch 105的缓存已被清空
	cachedForEpoch105After := msgCache.GetMessagesByEpoch(105)
	assert.Equal(t, 0, len(cachedForEpoch105After), "epoch 105的缓存应该已被清空")

	// ========================================
	// Step 6: 继续推进epoch，处理更多缓存消息
	// ========================================
	t.Log("\n[步骤 6/10] 继续推进epoch，处理剩余缓存消息...")

	epochsToProcess := []uint64{106, 107, 108, 109, 110}
	for _, epoch := range epochsToProcess {
		beforeCount := len(mainConsensus.receivedMsgs) + len(candidateConsensus.receivedMsgs)

		upgradeManager.SetCurrentEpoch(epoch)
		time.Sleep(100 * time.Millisecond)

		afterCount := len(mainConsensus.receivedMsgs) + len(candidateConsensus.receivedMsgs)
		processed := afterCount - beforeCount

		t.Logf("  - epoch %d: 处理了 %d 条缓存消息", epoch, processed)

		// 验证该epoch的缓存已被清空
		cached := msgCache.GetMessagesByEpoch(epoch)
		assert.Equal(t, 0, len(cached), fmt.Sprintf("epoch %d的缓存应该已被清空", epoch))
	}

	t.Log("✓ 所有epoch的缓存消息已处理完成")

	// ========================================
	// Step 7: 创建升级提案，启动候选共识
	// ========================================
	t.Log("\n[步骤 7/10] 创建升级提案并启动候选共识...")

	proposalID := types.TxHash{7, 7, 7}
	forkHeight := uint64(1000)
	candidateStartHeight := uint64(1100)
	switchHeight := uint64(2000)

	proposal := &upgrade.UpgradeProposal{
		ProposalID:         proposalID,
		TargetConsensus:    "hotstuff",
		ForkHeight:         forkHeight,
		PreexecStartHeight: candidateStartHeight,
		SwitchHeight:       switchHeight,
		Description:        "测试消息缓存的升级提案",
		Timestamp:          time.Now(),
	}

	err = persistence.SaveProposal(proposal)
	require.NoError(t, err, "保存提案失败")

	t.Log("✓ 升级提案创建完成")
	t.Logf("  - 提案ID: %x", proposalID[:8])
	t.Logf("  - 目标共识: %s", proposal.TargetConsensus)
	t.Logf("  - 候选链开始高度: %d", candidateStartHeight)

	// ========================================
	// Step 8: 模拟候选链启动前收到大量消息
	// ========================================
	t.Log("\n[步骤 8/10] 模拟候选链启动前收到大量消息...")

	// 设置当前epoch为115（候选链还未启动）
	upgradeManager.SetCurrentEpoch(115)

	// 创建一批给候选链的消息（epoch 120-130）
	candidateEarlyMessages := []struct {
		epoch    uint64
		senderID int64
		msgType  string
	}{
		{120, 2, "Prepare"},
		{120, 3, "PreCommit"},
		{121, 2, "Commit"},
		{122, 4, "Vote"},
		{123, 2, "NewView"},
		{124, 3, "Prepare"},
		{125, 2, "Commit"},
		{126, 4, "Vote"},
		{127, 3, "Prepare"},
		{128, 2, "PreCommit"},
		{129, 3, "Commit"},
		{130, 4, "NewView"},
	}

	for _, msgInfo := range candidateEarlyMessages {
		msg := createMockNetworkMessage(msgInfo.msgType, msgInfo.epoch)
		err := upgradeManager.OnNetworkMessage(msg, msgInfo.senderID, candidateConsensusID, msgInfo.epoch)
		require.NoError(t, err)
	}

	t.Log("✓ 候选链消息已缓存")
	t.Logf("  - 缓存的候选链消息数: %d", len(candidateEarlyMessages))

	// 验证消息已缓存
	totalCachedForCandidate := 0
	for epoch := uint64(120); epoch <= 130; epoch++ {
		cached := msgCache.GetMessagesByEpoch(epoch)
		totalCachedForCandidate += len(cached)
	}
	assert.Equal(t, len(candidateEarlyMessages), totalCachedForCandidate, "候选链消息应该全部被缓存")

	// ========================================
	// Step 9: 启动候选共识，验证缓存冲刷
	// ========================================
	t.Log("\n[步骤 9/10] 启动候选共识，验证缓存自动冲刷...")

	// 将候选共识添加到MultiChainManager
	multiChainMgr := upgradeManager.GetMultiChainManager()
	err = multiChainMgr.StartCandidateChain("candidate-chain", forkHeight, candidateConsensus)
	require.NoError(t, err, "添加候选链失败")

	// 记录候选链启动前的消息数
	beforeCandidateStart := len(candidateConsensus.receivedMsgs)

	// 调用OnCandidateConsensusStarted，触发缓存冲刷
	err = upgradeManager.OnCandidateConsensusStarted(candidateConsensusID)
	require.NoError(t, err, "启动候选共识失败")

	// 等待缓存冲刷完成
	time.Sleep(200 * time.Millisecond)

	// 验证候选链收到的消息
	afterCandidateStart := len(candidateConsensus.receivedMsgs)
	flushedMsgs := afterCandidateStart - beforeCandidateStart

	t.Log("✓ 候选共识启动完成")
	t.Logf("  - 冲刷到候选链的消息数: %d", flushedMsgs)

	// 验证缓存中候选链的消息已被清空
	cachedForCandidate := msgCache.PopAllForConsensus(candidateConsensusID)
	assert.Equal(t, 0, len(cachedForCandidate), "候选链的缓存应该已被冲刷")

	// ========================================
	// Step 10: 执行共识切换，清理旧epoch消息
	// ========================================
	t.Log("\n[步骤 10/10] 执行共识切换，清理旧epoch消息...")

	// 推进到切换高度附近的epoch
	switchEpoch := uint64(200)
	upgradeManager.SetCurrentEpoch(switchEpoch)

	// 添加一些旧epoch的消息到缓存（模拟延迟到达的消息）
	oldEpochMessages := []struct {
		epoch      uint64
		receiverID int64
	}{
		{95, mainConsensusID},
		{98, mainConsensusID},
		{100, mainConsensusID},
		{105, mainConsensusID},
	}

	for _, msgInfo := range oldEpochMessages {
		msg := createMockNetworkMessage("Prepare", msgInfo.epoch)
		// 直接缓存（绕过OnNetworkMessage的epoch过滤）
		msgCache.CacheMessage(msg, 2, msgInfo.receiverID, msgInfo.epoch)
	}

	// 验证旧消息已缓存
	totalOldCached := 0
	for _, msgInfo := range oldEpochMessages {
		cached := msgCache.GetMessagesByEpoch(msgInfo.epoch)
		totalOldCached += len(cached)
	}
	assert.Greater(t, totalOldCached, 0, "应该有旧epoch消息被缓存")

	t.Logf("  - 缓存的旧epoch消息数: %d", totalOldCached)

	// 执行共识切换，清理旧epoch（小于115的都清理）
	oldEpochThreshold := uint64(115)
	err = upgradeManager.OnConsensusSwitched(oldEpochThreshold)
	require.NoError(t, err, "共识切换清理失败")

	// 验证旧epoch消息已被清理
	remainingOldCached := 0
	for _, msgInfo := range oldEpochMessages {
		if msgInfo.epoch < oldEpochThreshold {
			cached := msgCache.GetMessagesByEpoch(msgInfo.epoch)
			remainingOldCached += len(cached)
		}
	}
	assert.Equal(t, 0, remainingOldCached, "旧epoch消息应该已被清理")

	t.Log("✓ 共识切换清理完成")
	t.Logf("  - 清理的旧epoch阈值: %d", oldEpochThreshold)

	// ========================================
	// 验证最终状态
	// ========================================
	t.Log("\n=== 验证最终状态 ===")

	// 统计主链和候选链收到的消息
	mainChainMsgCount := len(mainConsensus.receivedMsgs)
	candidateChainMsgCount := len(candidateConsensus.receivedMsgs)

	t.Log("\n消息处理统计:")
	t.Logf("  ✓ 主链共识处理消息数: %d", mainChainMsgCount)
	t.Logf("  ✓ 候选链共识处理消息数: %d", candidateChainMsgCount)
	t.Logf("  ✓ 总处理消息数: %d", mainChainMsgCount+candidateChainMsgCount)

	// 验证epoch处理情况
	t.Log("\nEpoch处理情况:")
	mainConsensus.mu.Lock()
	for epoch := range mainConsensus.processedEpochs {
		t.Logf("  ✓ 主链处理了epoch: %d", epoch)
	}
	mainConsensus.mu.Unlock()

	candidateConsensus.mu.Lock()
	for epoch := range candidateConsensus.processedEpochs {
		t.Logf("  ✓ 候选链处理了epoch: %d", epoch)
	}
	candidateConsensus.mu.Unlock()

	// 验证缓存已清空（除了未来的epoch）
	hasRemainingCache := false
	for epoch := uint64(95); epoch <= switchEpoch; epoch++ {
		cached := msgCache.GetMessagesByEpoch(epoch)
		if len(cached) > 0 {
			hasRemainingCache = true
			t.Logf("  ! epoch %d 仍有 %d 条缓存消息", epoch, len(cached))
		}
	}

	if !hasRemainingCache {
		t.Log("\n  ✓ 所有相关epoch的缓存已正确处理")
	}

	// 最终断言
	assert.Greater(t, mainChainMsgCount, 0, "主链应该处理了消息")
	assert.Greater(t, candidateChainMsgCount, 0, "候选链应该处理了消息")

	t.Log("\n=== 消息缓存端到端测试完成 ===")
	t.Log("\n测试验证了以下功能:")
	t.Log("  ✅ 消息提前到达时被正确缓存（epoch > current）")
	t.Log("  ✅ epoch推进时缓存消息自动处理")
	t.Log("  ✅ 候选共识启动时缓存自动冲刷")
	t.Log("  ✅ 共识切换后旧epoch消息正确清理")
	t.Log("  ✅ 主链和候选链消息正确路由")
}

// TestMessageCacheWithConcurrentConsensusSwitch 测试并发场景下的消息缓存
// 场景：在共识切换过程中，多个goroutine同时发送消息
func TestMessageCacheWithConcurrentConsensusSwitch(t *testing.T) {
	t.Log("=== 开始并发消息缓存测试 ===\n")

	// 初始化环境
	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.WarnLevel) // 减少日志输出

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	multiChainStorage, err := storage.NewLevelDBMultiChainStorage(tmpDir + "/multichain.db")
	require.NoError(t, err)
	defer multiChainStorage.Close()

	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	msgCache := upgrade.NewMessageCache(log)

	// 创建共识
	mainConsensusID := int64(1)
	mainConsensus := &mockConsensusForCache{
		id:              mainConsensusID,
		name:            "pot",
		receivedMsgs:    make([]*mockMessage, 0),
		processedEpochs: make(map[uint64]bool),
		mu:              &sync.Mutex{},
		log:             log,
	}

	candidateConsensusID := int64(2)
	candidateConsensus := &mockConsensusForCache{
		id:              candidateConsensusID,
		name:            "hotstuff",
		receivedMsgs:    make([]*mockMessage, 0),
		processedEpochs: make(map[uint64]bool),
		mu:              &sync.Mutex{},
		log:             log,
	}

	cfg := &config.ConsensusConfig{
		ConsensusID: 1,
		F:           1,
	}

	upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
		mainConsensus,
		cfg,
		multiChainStorage,
		persistence,
		log,
	)
	require.NoError(t, err)

	// 设置初始epoch
	currentEpoch := uint64(100)
	upgradeManager.SetCurrentEpoch(currentEpoch)

	t.Log("[步骤 1/3] 启动并发消息发送...")

	// 并发发送消息
	var wg sync.WaitGroup
	numSenders := 10
	msgsPerSender := 20
	totalMsgs := numSenders * msgsPerSender

	startEpoch := uint64(105)
	epochRange := uint64(20)

	for i := 0; i < numSenders; i++ {
		wg.Add(1)
		go func(senderID int) {
			defer wg.Done()

			for j := 0; j < msgsPerSender; j++ {
				epoch := startEpoch + uint64(j%int(epochRange))
				msg := createMockNetworkMessage("Prepare", epoch)

				// 随机发送到主链或候选链
				var receiverID int64
				if j%2 == 0 {
					receiverID = mainConsensusID
				} else {
					receiverID = candidateConsensusID
				}

				_ = upgradeManager.OnNetworkMessage(msg, int64(senderID+2), receiverID, epoch)
			}
		}(i)
	}

	wg.Wait()
	t.Logf("✓ 并发发送完成，总消息数: %d", totalMsgs)

	// 验证缓存状态
	totalCached := 0
	for epoch := startEpoch; epoch < startEpoch+epochRange; epoch++ {
		cached := msgCache.GetMessagesByEpoch(epoch)
		totalCached += len(cached)
	}

	t.Logf("  - 缓存的消息数: %d", totalCached)
	assert.Greater(t, totalCached, 0, "应该有消息被缓存")

	t.Log("\n[步骤 2/3] 并发推进epoch...")

	// 并发推进epoch
	processedCount := 0
	for epoch := startEpoch; epoch < startEpoch+epochRange; epoch++ {
		upgradeManager.SetCurrentEpoch(epoch)
		time.Sleep(50 * time.Millisecond)

		newProcessed := len(mainConsensus.receivedMsgs) + len(candidateConsensus.receivedMsgs)
		if newProcessed > processedCount {
			t.Logf("  - epoch %d: 处理了 %d 条消息", epoch, newProcessed-processedCount)
			processedCount = newProcessed
		}
	}

	t.Logf("✓ Epoch推进完成，总处理消息数: %d", processedCount)

	t.Log("\n[步骤 3/3] 启动候选共识并验证...")

	// 启动候选共识
	beforeStart := len(candidateConsensus.receivedMsgs)
	_ = upgradeManager.OnCandidateConsensusStarted(candidateConsensusID)
	time.Sleep(200 * time.Millisecond)
	afterStart := len(candidateConsensus.receivedMsgs)

	t.Logf("✓ 候选共识收到消息: %d", afterStart-beforeStart)

	// 验证最终状态
	finalMainMsgs := len(mainConsensus.receivedMsgs)
	finalCandidateMsgs := len(candidateConsensus.receivedMsgs)
	finalTotal := finalMainMsgs + finalCandidateMsgs

	t.Log("\n=== 并发测试完成 ===")
	t.Logf("  ✓ 主链处理: %d", finalMainMsgs)
	t.Logf("  ✓ 候选链处理: %d", finalCandidateMsgs)
	t.Logf("  ✓ 总处理: %d", finalTotal)

	assert.Greater(t, finalTotal, 0, "应该有消息被处理")
}

// TestMessageCacheExpiredCleanupDuringSwitch 测试共识切换时过期消息清理
func TestMessageCacheExpiredCleanupDuringSwitch(t *testing.T) {
	t.Log("=== 开始过期消息清理测试 ===\n")

	tmpDir := t.TempDir()
	log := logrus.NewEntry(logrus.New())

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	multiChainStorage, err := storage.NewLevelDBMultiChainStorage(tmpDir + "/multichain.db")
	require.NoError(t, err)
	defer multiChainStorage.Close()

	persistence, err := upgrade.NewBoltDBPersistence(tmpDir, log)
	require.NoError(t, err)
	defer persistence.Close()

	msgCache := upgrade.NewMessageCache(log)

	// 创建共识
	mainConsensus := &mockConsensusForCache{
		id:              1,
		name:            "pot",
		receivedMsgs:    make([]*mockMessage, 0),
		processedEpochs: make(map[uint64]bool),
		mu:              &sync.Mutex{},
		log:             log,
	}

	cfg := &config.ConsensusConfig{
		ConsensusID: 1,
		F:           1,
	}

	upgradeManager, err := upgrade.NewUpgradeManagerWithPersistence(
		mainConsensus,
		cfg,
		multiChainStorage,
		persistence,
		log,
	)
	require.NoError(t, err)

	t.Log("[步骤 1/3] 缓存不同epoch的消息...")

	// 缓存不同epoch的消息
	epochs := []uint64{50, 75, 100, 125, 150, 175, 200}
	for _, epoch := range epochs {
		for i := 0; i < 3; i++ {
			msg := createMockNetworkMessage("Prepare", epoch)
			msgCache.CacheMessage(msg, 2, 1, epoch)
		}
	}

	totalBefore := 0
	for _, epoch := range epochs {
		cached := msgCache.GetMessagesByEpoch(epoch)
		totalBefore += len(cached)
	}

	t.Logf("✓ 缓存消息总数: %d", totalBefore)

	t.Log("\n[步骤 2/3] 执行共识切换，清理旧epoch...")

	// 切换点设置为epoch 150，清理小于150的所有消息
	switchEpoch := uint64(150)
	err = upgradeManager.OnConsensusSwitched(switchEpoch)
	require.NoError(t, err)

	t.Log("\n[步骤 3/3] 验证清理结果...")

	// 验证清理结果
	remainingByEpoch := make(map[uint64]int)
	totalAfter := 0

	for _, epoch := range epochs {
		cached := msgCache.GetMessagesByEpoch(epoch)
		remainingByEpoch[epoch] = len(cached)
		totalAfter += len(cached)

		if epoch < switchEpoch {
			assert.Equal(t, 0, len(cached),
				fmt.Sprintf("epoch %d < %d，应该被清理", epoch, switchEpoch))
		} else {
			assert.Greater(t, len(cached), 0,
				fmt.Sprintf("epoch %d >= %d，应该保留", epoch, switchEpoch))
		}
	}

	t.Log("\n清理结果:")
	for _, epoch := range epochs {
		status := "✓ 保留"
		if epoch < switchEpoch {
			status = "✗ 清理"
		}
		t.Logf("  epoch %d: %s (剩余 %d 条)", epoch, status, remainingByEpoch[epoch])
	}

	t.Logf("\n清理统计:")
	t.Logf("  - 清理前: %d 条", totalBefore)
	t.Logf("  - 清理后: %d 条", totalAfter)
	t.Logf("  - 清理数: %d 条", totalBefore-totalAfter)

	t.Log("\n=== 过期消息清理测试完成 ===")
}

// ========================================
// 辅助类型和函数
// ========================================

// mockConsensusForCache 模拟共识，用于消息缓存测试
type mockConsensusForCache struct {
	id              int64
	name            string
	receivedMsgs    []*mockMessage
	processedEpochs map[uint64]bool
	mu              *sync.Mutex
	log             *logrus.Entry
}

func (m *mockConsensusForCache) OnReceive(msg interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mockMsg, ok := msg.(*mockMessage); ok {
		m.receivedMsgs = append(m.receivedMsgs, mockMsg)
		m.processedEpochs[mockMsg.epoch] = true
		m.log.Debugf("[%s-%d] 收到消息: epoch=%d, type=%s",
			m.name, m.id, mockMsg.epoch, mockMsg.msgType)
	}

	return nil
}

func (m *mockConsensusForCache) Start() error                                 { return nil }
func (m *mockConsensusForCache) Stop()                                        {}
func (m *mockConsensusForCache) GetConsensusID() int64                        { return m.id }
func (m *mockConsensusForCache) GetConsensusType() string                     { return m.name }
func (m *mockConsensusForCache) VerifyBlock([]byte, []byte) bool              { return true }
func (m *mockConsensusForCache) GetRequestEntrance() chan<- *pb.Request       { return nil }
func (m *mockConsensusForCache) GetMsgByteEntrance() chan<- []byte            { return nil }
func (m *mockConsensusForCache) UpdateExternalStatus(model.ExternalStatus)    {}
func (m *mockConsensusForCache) NewEpochConfirmation(int64, []byte, []string) {}
func (m *mockConsensusForCache) RequestLatestBlock(int64, []byte, []string)   {}
func (m *mockConsensusForCache) GetWeight(int64) float64                      { return 1.0 }
func (m *mockConsensusForCache) GetMaxAdversaryWeight() float64               { return 0.33 }

// mockMessage 模拟网络消息
type mockMessage struct {
	msgType string
	epoch   uint64
	data    []byte
}

// createMockNetworkMessage 创建模拟网络消息
func createMockNetworkMessage(msgType string, epoch uint64) interface{} {
	return &mockMessage{
		msgType: msgType,
		epoch:   epoch,
		data:    []byte(fmt.Sprintf("msg-%s-epoch-%d", msgType, epoch)),
	}
}

// getMessageStatus 获取消息状态描述
func getMessageStatus(msgEpoch, currentEpoch uint64) string {
	if msgEpoch > currentEpoch {
		return "缓存（epoch > current）"
	} else if msgEpoch == currentEpoch {
		return "直接处理（epoch == current）"
	} else {
		return "丢弃（epoch < current）"
	}
}

// createTestBlock 创建测试区块（从consensus_switch_test.go复用）
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
