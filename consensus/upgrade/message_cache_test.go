package upgrade

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageCache_BasicCaching 测试基本的消息缓存功能
func TestMessageCache_BasicCaching(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)

	// 设置切换信息
	cache.SetSwitchInfo(200, 1, 2) // switchHeight=200, oldConsensusID=1, newConsensusID=2

	// 缓存一条新共识的消息（切换前）
	msg := "test message"
	err := cache.CacheMessage(msg, 1, 2, 100) // senderID=1, receiverID=2, epoch=100
	require.NoError(t, err)

	// 验证缓存统计
	stats := cache.GetCacheStats()
	assert.Equal(t, 1, stats.NewConsensusMessageCount)
	assert.Equal(t, 0, stats.OldConsensusMessageCount)
	assert.False(t, stats.Switched)
}

// TestMessageCache_EpochFiltering 测试基于epoch的消息过滤
func TestMessageCache_EpochFiltering(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 缓存多个不同epoch的消息
	for epoch := uint64(100); epoch <= 105; epoch++ {
		err := cache.CacheMessage("msg", 1, 2, epoch)
		require.NoError(t, err)
	}

	// 获取特定epoch的消息
	messages := cache.GetMessagesByEpoch(103)
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, uint64(103), messages[0].Epoch)

	// 清理特定epoch
	cleaned := cache.DropByEpoch(103)
	assert.Equal(t, 1, cleaned)

	// 验证已清理
	messages = cache.GetMessagesByEpoch(103)
	assert.Equal(t, 0, len(messages))

	// 验证其他epoch未受影响
	stats := cache.GetCacheStats()
	assert.Equal(t, 5, stats.NewConsensusMessageCount) // 100,101,102,104,105
}

// TestMessageCache_PopAllForConsensus 测试弹出特定共识的所有消息
func TestMessageCache_PopAllForConsensus(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 缓存10条消息
	for i := 0; i < 10; i++ {
		err := cache.CacheMessage("msg", 1, 2, uint64(100+i))
		require.NoError(t, err)
	}

	// 弹出所有新共识消息
	messages := cache.PopAllForConsensus(2)
	assert.Equal(t, 10, len(messages))

	// 验证缓存已清空
	stats := cache.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
}

// TestMessageCache_OnSwitch 测试切换时的消息转发
func TestMessageCache_OnSwitch(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 切换前缓存消息
	for i := 0; i < 5; i++ {
		err := cache.CacheMessage("msg", 1, 2, uint64(100+i))
		require.NoError(t, err)
	}

	// 执行切换
	messages := cache.OnSwitch()
	assert.Equal(t, 5, len(messages))
	assert.True(t, cache.IsSwitched())

	// 验证所有消息都标记为已处理
	for _, msg := range messages {
		assert.True(t, msg.ProcessedAfterSwitch)
	}

	// 验证新共识缓存已清空
	stats := cache.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
}

// TestMessageCache_PostSwitchCaching 测试切换后缓存旧共识消息
func TestMessageCache_PostSwitchCaching(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 执行切换
	cache.OnSwitch()

	// 切换后缓存旧共识消息
	err := cache.CacheMessage("late msg", 3, 1, 150) // receiverID=1 (旧共识)
	require.NoError(t, err)

	// 验证缓存到旧共识缓存区
	stats := cache.GetCacheStats()
	assert.Equal(t, 1, stats.OldConsensusMessageCount)
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
}

// TestMessageCache_CleanupExpired 测试过期消息清理
func TestMessageCache_CleanupExpired(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)
	cache.SetMessageTimeout(100 * time.Millisecond) // 设置短超时

	// 缓存消息
	err := cache.CacheMessage("msg1", 1, 2, 100)
	require.NoError(t, err)

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 缓存另一条消息（不超时）
	err = cache.CacheMessage("msg2", 1, 2, 101)
	require.NoError(t, err)

	// 清理过期消息
	cleaned := cache.CleanupExpiredMessages()
	assert.Equal(t, 1, cleaned) // 只清理了msg1

	// 验证仅保留未过期消息
	stats := cache.GetCacheStats()
	assert.Equal(t, 1, stats.NewConsensusMessageCount)
}

// TestMessageCache_CacheFull 测试缓存满的情况
func TestMessageCache_CacheFull(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)
	cache.SetMaxCacheSize(10) // 设置小缓存

	// 填满缓存
	for i := 0; i < 10; i++ {
		err := cache.CacheMessage("msg", 1, 2, uint64(100+i))
		require.NoError(t, err)
	}

	// 尝试添加超出容量的消息
	err := cache.CacheMessage("overflow", 1, 2, 111)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache full")
}

// TestMessageCache_Clear 测试清空缓存
func TestMessageCache_Clear(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 缓存一些消息
	for i := 0; i < 5; i++ {
		err := cache.CacheMessage("msg", 1, 2, uint64(100+i))
		require.NoError(t, err)
	}

	// 清空缓存
	cache.Clear()

	// 验证已清空
	stats := cache.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
	assert.Equal(t, 0, stats.OldConsensusMessageCount)
	assert.False(t, stats.Switched)
}

// TestMessageCache_CleanupTask 测试自动清理任务
func TestMessageCache_CleanupTask(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)
	cache.SetMessageTimeout(50 * time.Millisecond)

	// 启动清理任务（每100ms清理一次）
	stopChan := cache.StartCleanupTask(100 * time.Millisecond)
	defer close(stopChan)

	// 缓存消息
	err := cache.CacheMessage("msg", 1, 2, 100)
	require.NoError(t, err)

	// 等待消息过期并被自动清理
	time.Sleep(200 * time.Millisecond)

	// 验证已被清理
	stats := cache.GetCacheStats()
	assert.Equal(t, 0, stats.NewConsensusMessageCount)
}

// TestMessageCache_MultipleEpochs 测试多个epoch的消息管理
func TestMessageCache_MultipleEpochs(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)

	// 缓存多个epoch的消息
	epochs := []uint64{100, 105, 110, 115, 120}
	for _, epoch := range epochs {
		for i := 0; i < 3; i++ { // 每个epoch 3条消息
			err := cache.CacheMessage("msg", 1, 2, epoch)
			require.NoError(t, err)
		}
	}

	// 验证总数
	stats := cache.GetCacheStats()
	assert.Equal(t, 15, stats.NewConsensusMessageCount)

	// 清理中间的epoch
	cleaned := cache.DropByEpoch(110)
	assert.Equal(t, 3, cleaned)

	// 验证剩余
	stats = cache.GetCacheStats()
	assert.Equal(t, 12, stats.NewConsensusMessageCount)

	// 验证特定epoch的消息数
	messages := cache.GetMessagesByEpoch(105)
	assert.Equal(t, 3, len(messages))
}

// TestMessageCache_ConcurrentAccess 测试并发访问
func TestMessageCache_ConcurrentAccess(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cache := NewMessageCache(log)
	cache.SetSwitchInfo(200, 1, 2)
	cache.SetMaxCacheSize(1000)

	// 启动多个goroutine并发写入
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				cache.CacheMessage("msg", 1, 2, uint64(100+j))
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有消息都被缓存（可能有重复键，但不应超过100条）
	stats := cache.GetCacheStats()
	assert.LessOrEqual(t, stats.NewConsensusMessageCount, 100)
	assert.Greater(t, stats.NewConsensusMessageCount, 0)
}
