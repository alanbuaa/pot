package upgrade

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestSwitchManagerCreation 测试切换管理器创建
func TestSwitchManagerCreation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	sm := NewSwitchManager(dualChain, monitor, log)

	assert.NotNil(t, sm)
	assert.False(t, sm.IsSwitched())
	assert.False(t, sm.IsSwitching())
	assert.Equal(t, uint64(0), sm.switchHeight)
}

// TestSwitchManagerPrepare 测试准备切换
func TestSwitchManagerPrepare(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	sm := NewSwitchManager(dualChain, monitor, log)

	switchHeight := uint64(100)
	err := sm.PrepareSwitch(switchHeight)

	assert.NoError(t, err)
	assert.Equal(t, switchHeight, sm.switchHeight)
	assert.False(t, sm.IsSwitched())
	assert.False(t, sm.IsSwitching())
}

// TestSwitchManagerValidate 测试验证切换条件
func TestSwitchManagerValidate(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := NewPreexecMonitor(&UpgradeProposal{}, dualChain, log)

	sm := NewSwitchManager(dualChain, monitor, log)
	sm.PrepareSwitch(100)

	err := sm.ValidateSwitch()

	// 由于 monitor 未就绪，应该失败
	assert.Error(t, err)
}

// TestSwitchManagerAlreadySwitched 测试重复切换
func TestSwitchManagerAlreadySwitched(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	sm := NewSwitchManager(dualChain, monitor, log)
	sm.PrepareSwitch(100)

	// 手动设置为已切换状态
	sm.mu.Lock()
	sm.switched = true
	sm.mu.Unlock()

	err := sm.ExecuteSwitch()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already switched")
}

// TestSwitchManagerStatus 测试获取切换状态
func TestSwitchManagerStatus(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	sm := NewSwitchManager(dualChain, monitor, log)
	sm.PrepareSwitch(100)

	status := sm.GetSwitchStatus()

	assert.NotNil(t, status)
	assert.Equal(t, uint64(100), status.SwitchHeight)
	assert.False(t, status.Switched)
	assert.False(t, status.Switching)
}

// TestSwitchManagerReset 测试重置
func TestSwitchManagerReset(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	sm := NewSwitchManager(dualChain, monitor, log)
	sm.PrepareSwitch(100)

	// 手动设置为已切换状态
	sm.mu.Lock()
	sm.switched = true
	sm.switching = true
	sm.mu.Unlock()

	sm.Reset()

	assert.False(t, sm.IsSwitched())
	assert.False(t, sm.IsSwitching())
	assert.Equal(t, uint64(0), sm.switchHeight)
}

// TestRollbackManagerCreation 测试回退管理器创建
func TestRollbackManagerCreation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	mockStorage := &mockDualChainStorage{}

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)
	monitor := &PreexecMonitor{}

	rm := NewRollbackManager(dualChain, monitor, log)

	assert.NotNil(t, rm)
	assert.False(t, rm.IsRolledBack())
	// CanRollback在未初始化状态下可能返回false，这是合理的
}

// TestRollbackManagerExecute 测试执行回退
func TestRollbackManagerExecute(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	newConsensus := newMockConsensus()
	mockStorage := newMockDualChainStorage()

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)

	// 先启动预执行，这样才能回退
	forkBlock := newTestBlock(100)
	mockStorage.StoreMainBlock(forkBlock)
	dualChain.StartPreexecution(100, newConsensus)

	proposal := &UpgradeProposal{PreexecStartHeight: 100, SwitchHeight: 200}
	monitor := NewPreexecMonitor(proposal, dualChain, log)

	rm := NewRollbackManager(dualChain, monitor, log)

	err := rm.ExecuteRollback("test rollback")

	assert.NoError(t, err)
	assert.True(t, rm.IsRolledBack())
	assert.False(t, rm.CanRollback())
	assert.Equal(t, "test rollback", rm.GetRollbackReason())
}

// TestRollbackManagerAlreadyRolledBack 测试重复回退
func TestRollbackManagerAlreadyRolledBack(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	newConsensus := newMockConsensus()
	mockStorage := newMockDualChainStorage()

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)

	// 启动预执行
	forkBlock := newTestBlock(100)
	mockStorage.StoreMainBlock(forkBlock)
	dualChain.StartPreexecution(100, newConsensus)

	proposal := &UpgradeProposal{PreexecStartHeight: 100, SwitchHeight: 200}
	monitor := NewPreexecMonitor(proposal, dualChain, log)

	rm := NewRollbackManager(dualChain, monitor, log)

	// 第一次回退
	err := rm.ExecuteRollback("first rollback")
	assert.NoError(t, err)

	// 第二次回退应该失败
	err = rm.ExecuteRollback("second rollback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already rolled back")
}

// TestRollbackManagerForce 测试强制回退
func TestRollbackManagerForce(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	newConsensus := newMockConsensus()
	mockStorage := newMockDualChainStorage()

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)

	// 启动预执行
	forkBlock := newTestBlock(100)
	mockStorage.StoreMainBlock(forkBlock)
	dualChain.StartPreexecution(100, newConsensus)

	proposal := &UpgradeProposal{PreexecStartHeight: 100, SwitchHeight: 200}
	monitor := NewPreexecMonitor(proposal, dualChain, log)

	rm := NewRollbackManager(dualChain, monitor, log)

	err := rm.ForceRollback("emergency rollback")

	assert.NoError(t, err)
	assert.True(t, rm.IsRolledBack())
	assert.Equal(t, "emergency rollback", rm.GetRollbackReason())
}

// TestRollbackManagerStatus 测试获取回退状态
func TestRollbackManagerStatus(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	newConsensus := newMockConsensus()
	mockStorage := newMockDualChainStorage()

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)

	// 启动预执行
	forkBlock := newTestBlock(100)
	mockStorage.StoreMainBlock(forkBlock)
	dualChain.StartPreexecution(100, newConsensus)

	proposal := &UpgradeProposal{PreexecStartHeight: 100, SwitchHeight: 200}
	monitor := NewPreexecMonitor(proposal, dualChain, log)

	rm := NewRollbackManager(dualChain, monitor, log)

	status := rm.GetRollbackStatus()

	assert.NotNil(t, status)
	assert.False(t, status.RolledBack)
	assert.True(t, rm.CanRollback())
	assert.Equal(t, "", status.RollbackReason)

	// 执行回退后再次检查
	rm.ExecuteRollback("test reason")
	status = rm.GetRollbackStatus()

	assert.True(t, status.RolledBack)
	assert.False(t, rm.CanRollback())
	assert.Equal(t, "test reason", status.RollbackReason)
}

// TestRollbackManagerReset 测试重置
func TestRollbackManagerReset(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mockConsensus := newMockConsensus()
	newConsensus := newMockConsensus()
	mockStorage := newMockDualChainStorage()

	dualChain := NewDualChainManager(mockConsensus, mockStorage, log)

	// 启动预执行
	forkBlock := newTestBlock(100)
	mockStorage.StoreMainBlock(forkBlock)
	dualChain.StartPreexecution(100, newConsensus)

	proposal := &UpgradeProposal{PreexecStartHeight: 100, SwitchHeight: 200}
	monitor := NewPreexecMonitor(proposal, dualChain, log)

	rm := NewRollbackManager(dualChain, monitor, log)

	// 执行回退
	rm.ExecuteRollback("test rollback")
	assert.True(t, rm.IsRolledBack())

	// 重置
	rm.Reset()

	assert.False(t, rm.IsRolledBack())
	// 注意：回退后预执行链已停止，所以 CanRollback 为 false
	assert.False(t, rm.CanRollback())
	assert.Equal(t, "", rm.GetRollbackReason())
}

// TestMessageCacheCreation 测试消息缓存创建
func TestMessageCacheCreation(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := NewMessageCache(log)

	assert.NotNil(t, mc)
	assert.False(t, mc.IsSwitched())

	stats := mc.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
	assert.Equal(t, 0, stats.OldConsensusMessageCount)
}

// TestMessageCacheSetSwitchInfo 测试设置切换信息
func TestMessageCacheSetSwitchInfo(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := NewMessageCache(log)

	mc.SetSwitchInfo(100, 1, 2)

	assert.Equal(t, uint64(100), mc.switchHeight)
	assert.Equal(t, int64(1), mc.oldConsensusID)
	assert.Equal(t, int64(2), mc.newConsensusID)
}

// TestMessageCacheClearExpired 测试清理过期消息
func TestMessageCacheClearExpired(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := NewMessageCache(log)
	mc.SetMessageTimeout(100 * time.Millisecond)

	// 不会清理任何消息（没有缓存）
	cleaned := mc.CleanupExpiredMessages()
	assert.Equal(t, 0, cleaned)
}

// TestMessageCacheClear 测试清空缓存
func TestMessageCacheClear(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mc := NewMessageCache(log)
	mc.SetSwitchInfo(100, 1, 2)

	mc.Clear()

	assert.False(t, mc.IsSwitched())
	stats := mc.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
}
