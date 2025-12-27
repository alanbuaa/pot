package storage

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestMessageCacheStorage(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "message_cache_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建存储
	storage, err := NewLevelDBMessageCacheStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	t.Run("StoreAndGetMessage", func(t *testing.T) {
		msg := &CachedMessage{
			MessageID:   []byte("msg1"),
			MessageType: "block",
			Content:     []byte("test content"),
			TargetEpoch: 100,
			ReceivedAt:  time.Now(),
			ProposalID:  types.TxHash{1, 2, 3},
		}

		err := storage.StoreMessage(msg)
		require.NoError(t, err)

		retrieved, err := storage.GetMessage([]byte("msg1"))
		require.NoError(t, err)
		assert.Equal(t, msg.MessageID, retrieved.MessageID)
		assert.Equal(t, msg.MessageType, retrieved.MessageType)
		assert.Equal(t, msg.TargetEpoch, retrieved.TargetEpoch)
	})

	t.Run("GetMessagesForEpoch", func(t *testing.T) {
		// 存储多个消息
		for i := 0; i < 5; i++ {
			msg := &CachedMessage{
				MessageID:   []byte{byte(i)},
				MessageType: "block",
				Content:     []byte("content"),
				TargetEpoch: 200,
				ReceivedAt:  time.Now(),
				ProposalID:  types.TxHash{1, 2, 3},
			}
			err := storage.StoreMessage(msg)
			require.NoError(t, err)
		}

		messages, err := storage.GetMessagesForEpoch(200)
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("GetMessagesForProposal", func(t *testing.T) {
		proposalID := types.TxHash{4, 5, 6}
		for i := 0; i < 3; i++ {
			msg := &CachedMessage{
				MessageID:   []byte{10, byte(i)},
				MessageType: "vote",
				Content:     []byte("vote content"),
				TargetEpoch: 300,
				ReceivedAt:  time.Now(),
				ProposalID:  proposalID,
			}
			err := storage.StoreMessage(msg)
			require.NoError(t, err)
		}

		messages, err := storage.GetMessagesForProposal(proposalID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(messages), 3)
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		msg := &CachedMessage{
			MessageID:   []byte("to_delete"),
			MessageType: "test",
			Content:     []byte("content"),
			TargetEpoch: 400,
			ReceivedAt:  time.Now(),
			ProposalID:  types.TxHash{7, 8, 9},
		}
		err := storage.StoreMessage(msg)
		require.NoError(t, err)

		err = storage.DeleteMessage([]byte("to_delete"))
		require.NoError(t, err)

		_, err = storage.GetMessage([]byte("to_delete"))
		assert.Error(t, err)
	})

	t.Run("ClearMessagesBefore", func(t *testing.T) {
		// 存储不同 epoch 的消息
		for epoch := uint64(500); epoch < 510; epoch++ {
			msg := &CachedMessage{
				MessageID:   []byte{byte(epoch)},
				MessageType: "block",
				Content:     []byte("content"),
				TargetEpoch: epoch,
				ReceivedAt:  time.Now(),
				ProposalID:  types.TxHash{10, 11, 12},
			}
			err := storage.StoreMessage(msg)
			require.NoError(t, err)
		}

		err := storage.ClearMessagesBefore(505)
		require.NoError(t, err)

		// 检查 epoch < 505 的消息已被删除
		for epoch := uint64(500); epoch < 505; epoch++ {
			_, err := storage.GetMessage([]byte{byte(epoch)})
			assert.Error(t, err)
		}

		// 检查 epoch >= 505 的消息仍然存在
		for epoch := uint64(505); epoch < 510; epoch++ {
			_, err := storage.GetMessage([]byte{byte(epoch)})
			assert.NoError(t, err)
		}
	})

	t.Run("ClearMessagesForProposal", func(t *testing.T) {
		proposalID := types.TxHash{13, 14, 15}
		for i := 0; i < 5; i++ {
			msg := &CachedMessage{
				MessageID:   []byte{20, byte(i)},
				MessageType: "test",
				Content:     []byte("content"),
				TargetEpoch: 600,
				ReceivedAt:  time.Now(),
				ProposalID:  proposalID,
			}
			err := storage.StoreMessage(msg)
			require.NoError(t, err)
		}

		err := storage.ClearMessagesForProposal(proposalID)
		require.NoError(t, err)

		messages, err := storage.GetMessagesForProposal(proposalID)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})

	t.Run("GetCacheStats", func(t *testing.T) {
		stats, err := storage.GetCacheStats()
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Greater(t, stats.TotalMessages, int64(0))
	})
}

func TestDualChainStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dual_chain_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	storage, err := NewLevelDBDualChainStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	t.Run("StoreAndGetMainBlock", func(t *testing.T) {
		block := &types.Block{
			Header: &types.Header{
				Height: 1,
			},
			Txs: []*types.Tx{},
		}

		err := storage.StoreMainBlock(block)
		require.NoError(t, err)

		retrieved, err := storage.GetMainBlock(1)
		require.NoError(t, err)
		assert.Equal(t, block.Header.Height, retrieved.Header.Height)
	})

	t.Run("StoreAndGetPreexecBlock", func(t *testing.T) {
		block := &types.Block{
			Header: &types.Header{
				Height: 100,
			},
			Txs: []*types.Tx{},
		}

		err := storage.StorePreexecBlock(block)
		require.NoError(t, err)

		retrieved, err := storage.GetPreexecBlock(100)
		require.NoError(t, err)
		assert.Equal(t, block.Header.Height, retrieved.Header.Height)
	})

	t.Run("GetPreexecBlocks", func(t *testing.T) {
		for i := uint64(200); i < 205; i++ {
			block := &types.Block{
				Header: &types.Header{
					Height: i,
				},
				Txs: []*types.Tx{},
			}
			err := storage.StorePreexecBlock(block)
			require.NoError(t, err)
		}

		blocks, err := storage.GetPreexecBlocks(200, 204)
		require.NoError(t, err)
		assert.Len(t, blocks, 5)
		assert.Equal(t, uint64(200), blocks[0].Header.Height)
		assert.Equal(t, uint64(204), blocks[4].Header.Height)
	})

	t.Run("DeletePreexecBlocks", func(t *testing.T) {
		for i := uint64(300); i < 310; i++ {
			block := &types.Block{
				Header: &types.Header{
					Height: i,
				},
				Txs: []*types.Tx{},
			}
			err := storage.StorePreexecBlock(block)
			require.NoError(t, err)
		}

		err := storage.DeletePreexecBlocks(305)
		require.NoError(t, err)

		// 305 及以后的区块应该被删除
		for i := uint64(305); i < 310; i++ {
			_, err := storage.GetPreexecBlock(i)
			assert.Error(t, err)
		}

		// 305 之前的区块应该还在
		for i := uint64(300); i < 305; i++ {
			_, err := storage.GetPreexecBlock(i)
			assert.NoError(t, err)
		}
	})

	t.Run("PromoteToMainChain", func(t *testing.T) {
		block := &types.Block{
			Header: &types.Header{
				Height: 400,
			},
			Txs: []*types.Tx{},
		}

		// 先存储为预执行链区块
		err := storage.StorePreexecBlock(block)
		require.NoError(t, err)

		// 提升到主链
		err = storage.PromoteToMainChain(block)
		require.NoError(t, err)

		// 应该能从主链获取
		_, err = storage.GetMainBlock(400)
		assert.NoError(t, err)

		// 应该从预执行链删除
		_, err = storage.GetPreexecBlock(400)
		assert.Error(t, err)
	})

	t.Run("DeleteMainBlocksFrom", func(t *testing.T) {
		for i := uint64(500); i < 510; i++ {
			block := &types.Block{
				Header: &types.Header{
					Height: i,
				},
				Txs: []*types.Tx{},
			}
			err := storage.StoreMainBlock(block)
			require.NoError(t, err)
		}

		err := storage.DeleteMainBlocksFrom(505)
		require.NoError(t, err)

		// 505 及以后的区块应该被删除
		for i := uint64(505); i < 510; i++ {
			_, err := storage.GetMainBlock(i)
			assert.Error(t, err)
		}

		// 505 之前的区块应该还在
		for i := uint64(500); i < 505; i++ {
			_, err := storage.GetMainBlock(i)
			assert.NoError(t, err)
		}
	})
}
