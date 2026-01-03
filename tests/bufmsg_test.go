package tests

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// TestBufferedMessageEarlyArrival 测试消息提前到达场景
// 场景：预执行链的消息在预执行开始前到达
func TestBufferedMessageEarlyArrival(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建存储
	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	dualChainStorage, err := storage.NewLevelDBDualChainStorage(tmpDir + "/dual.db")
	require.NoError(t, err)
	defer dualChainStorage.Close()

	log := logrus.NewEntry(logrus.New())

	// 创建消息缓存
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	// 模拟预执行开始前收到消息
	targetHeight := uint64(1000)
	proposalID := types.TxHash{1, 2, 3}

	// 创建早到的消息
	for i := uint64(1000); i < 1010; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: i},
			Txs:    []*types.Tx{},
		}

		err = msgCache.BufferMessage(proposalID, msg)
		require.NoError(t, err)
	}

	// 验证消息已缓存
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 10, len(buffered))

	// 模拟预执行启动
	msgCache.MarkPreexecStarted(proposalID, targetHeight)

	// 重新获取缓存消息（应该可用）
	buffered = msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 10, len(buffered))

	// 处理缓存消息
	for _, msg := range buffered {
		block := msg.(*types.Block)
		assert.GreaterOrEqual(t, block.Header.Height, targetHeight)
	}

	// 清理缓存
	msgCache.ClearProposalMessages(proposalID)
	buffered = msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 0, len(buffered))
}

// TestBufferedMessagePersistence 测试消息缓存持久化
// 场景：节点重启后能恢复缓存的消息
func TestBufferedMessagePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/msgcache.db"

	proposalID := types.TxHash{4, 5, 6}
	log := logrus.NewEntry(logrus.New())

	// 第一阶段：存储消息
	{
		msgStorage, err := storage.NewLevelDBMessageCacheStorage(dbPath)
		require.NoError(t, err)

		msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

		// 缓存消息
		for i := uint64(100); i < 110; i++ {
			msg := &types.Block{
				Header: &types.Header{Height: i},
				Txs:    []*types.Tx{},
			}
			err = msgCache.BufferMessage(proposalID, msg)
			require.NoError(t, err)
		}

		msgStorage.Close()
	}

	// 模拟重启
	time.Sleep(100 * time.Millisecond)

	// 第二阶段：恢复消息
	{
		msgStorage, err := storage.NewLevelDBMessageCacheStorage(dbPath)
		require.NoError(t, err)
		defer msgStorage.Close()

		msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

		// 从存储恢复消息
		err = msgCache.RecoverMessages(proposalID)
		require.NoError(t, err)

		// 验证消息已恢复
		buffered := msgCache.GetBufferedMessages(proposalID)
		assert.Equal(t, 10, len(buffered))
	}
}

// TestBufferedMessageClearOnSwitch 测试切换时清理消息
// 场景：切换完成后清理所有缓存消息
func TestBufferedMessageClearOnSwitch(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{7, 8, 9}

	// 缓存消息
	for i := uint64(200); i < 220; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: i},
			Txs:    []*types.Tx{},
		}
		err = msgCache.BufferMessage(proposalID, msg)
		require.NoError(t, err)
	}

	// 验证消息已缓存
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 20, len(buffered))

	// 模拟切换完成，清理消息
	msgCache.OnSwitchComplete(proposalID)

	// 验证消息已清理
	buffered = msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 0, len(buffered))

	// 验证持久化存储也已清理
	recovered, err := msgStorage.GetMessages(proposalID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(recovered))
}

// TestBufferedMessageExpiry 测试消息过期清理
// 场景：长时间未使用的消息应该被清理
func TestBufferedMessageExpiry(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{10, 11, 12}

	// 缓存消息
	msg := &types.Block{
		Header: &types.Header{Height: 300},
		Txs:    []*types.Tx{},
	}
	err = msgCache.BufferMessage(proposalID, msg)
	require.NoError(t, err)

	// 设置较短的过期时间（测试用）
	msgCache.SetExpiryDuration(100 * time.Millisecond)

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 清理过期消息
	msgCache.CleanupExpiredMessages()

	// 验证消息已清理
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 0, len(buffered))
}

// TestBufferedMessageOrdering 测试消息顺序处理
// 场景：缓存的消息应该按高度顺序处理
func TestBufferedMessageOrdering(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{13, 14, 15}

	// 乱序添加消息
	heights := []uint64{405, 401, 403, 402, 404}
	for _, h := range heights {
		msg := &types.Block{
			Header: &types.Header{Height: h},
			Txs:    []*types.Tx{},
		}
		err = msgCache.BufferMessage(proposalID, msg)
		require.NoError(t, err)
	}

	// 获取排序后的消息
	buffered := msgCache.GetOrderedMessages(proposalID)
	require.Equal(t, 5, len(buffered))

	// 验证顺序
	for i, msg := range buffered {
		block := msg.(*types.Block)
		expectedHeight := uint64(401 + i)
		assert.Equal(t, expectedHeight, block.Header.Height)
	}
}

// TestBufferedMessageDuplicateHandling 测试重复消息处理
// 场景：相同高度的消息应该被去重
func TestBufferedMessageDuplicateHandling(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{16, 17, 18}

	// 添加相同高度的消息多次
	for i := 0; i < 3; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: 500},
			Txs:    []*types.Tx{},
		}
		err = msgCache.BufferMessage(proposalID, msg)
		// 第一次应该成功，后续应该被忽略
		if i == 0 {
			require.NoError(t, err)
		}
	}

	// 验证只有一条消息
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.Equal(t, 1, len(buffered))
}

// TestBufferedMessageBoundedSize 测试消息缓存大小限制
// 场景：缓存大小应该有上限，防止内存溢出
func TestBufferedMessageBoundedSize(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{19, 20, 21}

	// 设置最大缓存数量
	maxSize := 100
	msgCache.SetMaxBufferSize(maxSize)

	// 尝试添加超过限制的消息
	for i := uint64(0); i < 150; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: i},
			Txs:    []*types.Tx{},
		}
		msgCache.BufferMessage(proposalID, msg)
	}

	// 验证缓存大小不超过限制
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.LessOrEqual(t, len(buffered), maxSize)
}

// TestBufferedMessageMultiProposal 测试多提案消息隔离
// 场景：不同提案的消息应该互不干扰
func TestBufferedMessageMultiProposal(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposal1 := types.TxHash{22, 23, 24}
	proposal2 := types.TxHash{25, 26, 27}

	// 为不同提案添加消息
	for i := uint64(0); i < 5; i++ {
		msg1 := &types.Block{
			Header: &types.Header{Height: 600 + i},
			Txs:    []*types.Tx{},
		}
		msgCache.BufferMessage(proposal1, msg1)

		msg2 := &types.Block{
			Header: &types.Header{Height: 700 + i},
			Txs:    []*types.Tx{},
		}
		msgCache.BufferMessage(proposal2, msg2)
	}

	// 验证消息隔离
	buffered1 := msgCache.GetBufferedMessages(proposal1)
	buffered2 := msgCache.GetBufferedMessages(proposal2)

	assert.Equal(t, 5, len(buffered1))
	assert.Equal(t, 5, len(buffered2))

	// 验证高度不同
	block1 := buffered1[0].(*types.Block)
	block2 := buffered2[0].(*types.Block)
	assert.Equal(t, uint64(600), block1.Header.Height)
	assert.Equal(t, uint64(700), block2.Header.Height)

	// 清理一个提案不应影响另一个
	msgCache.ClearProposalMessages(proposal1)
	buffered1 = msgCache.GetBufferedMessages(proposal1)
	buffered2 = msgCache.GetBufferedMessages(proposal2)
	assert.Equal(t, 0, len(buffered1))
	assert.Equal(t, 5, len(buffered2))
}

// TestBufferedMessageConcurrency 测试并发安全
// 场景：多个goroutine同时操作消息缓存
func TestBufferedMessageConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	require.NoError(t, err)
	defer msgStorage.Close()

	log := logrus.NewEntry(logrus.New())
	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, log)

	proposalID := types.TxHash{28, 29, 30}

	// 并发写入
	done := make(chan bool)
	for g := 0; g < 10; g++ {
		go func(base uint64) {
			for i := uint64(0); i < 10; i++ {
				msg := &types.Block{
					Header: &types.Header{Height: base + i},
					Txs:    []*types.Tx{},
				}
				msgCache.BufferMessage(proposalID, msg)
			}
			done <- true
		}(uint64(g * 100))
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证消息已全部写入
	buffered := msgCache.GetBufferedMessages(proposalID)
	assert.GreaterOrEqual(t, len(buffered), 90) // 允许一些去重
}

// BenchmarkMessageBuffering 基准测试：消息缓存性能
func BenchmarkMessageBuffering(b *testing.B) {
	tmpDir := b.TempDir()
	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	if err != nil {
		b.Fatal(err)
	}
	defer msgStorage.Close()

	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, logrus.NewEntry(logrus.New()))
	proposalID := types.TxHash{31, 32, 33}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: uint64(i)},
			Txs:    []*types.Tx{},
		}
		msgCache.BufferMessage(proposalID, msg)
	}
}

// BenchmarkMessageRetrieval 基准测试：消息检索性能
func BenchmarkMessageRetrieval(b *testing.B) {
	tmpDir := b.TempDir()
	msgStorage, err := storage.NewLevelDBMessageCacheStorage(tmpDir + "/msgcache.db")
	if err != nil {
		b.Fatal(err)
	}
	defer msgStorage.Close()

	msgCache := upgrade.NewMessageCacheWithStorage(msgStorage, logrus.NewEntry(logrus.New()))
	proposalID := types.TxHash{34, 35, 36}

	// 预先填充消息
	for i := 0; i < 1000; i++ {
		msg := &types.Block{
			Header: &types.Header{Height: uint64(i)},
			Txs:    []*types.Tx{},
		}
		msgCache.BufferMessage(proposalID, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgCache.GetBufferedMessages(proposalID)
	}
}
