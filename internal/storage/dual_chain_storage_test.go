package storage

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestNewLevelDBDualChainStorage(t *testing.T) {
	dbPath := "/tmp/test_dual_chain_storage"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	if storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestStoreAndGetMainBlock(t *testing.T) {
	dbPath := "/tmp/test_main_block"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建测试区块
	block := createTestBlock(100)

	// 存储区块
	if err := storage.StoreMainBlock(block); err != nil {
		t.Fatalf("Failed to store block: %v", err)
	}

	// 获取区块
	retrieved, err := storage.GetMainBlock(100)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}

	if retrieved.Header.Height != block.Header.Height {
		t.Errorf("Height mismatch: got %d, want %d",
			retrieved.Header.Height, block.Header.Height)
	}

	if retrieved.Header.Address != block.Header.Address {
		t.Errorf("Address mismatch: got %d, want %d",
			retrieved.Header.Address, block.Header.Address)
	}
}

func TestStoreAndGetPreexecBlock(t *testing.T) {
	dbPath := "/tmp/test_preexec_block"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建测试区块
	block := createTestBlock(200)

	// 存储预执行链区块
	if err := storage.StorePreexecBlock(block); err != nil {
		t.Fatalf("Failed to store preexec block: %v", err)
	}

	// 获取预执行链区块
	retrieved, err := storage.GetPreexecBlock(200)
	if err != nil {
		t.Fatalf("Failed to get preexec block: %v", err)
	}

	if retrieved.Header.Height != block.Header.Height {
		t.Errorf("Height mismatch: got %d, want %d",
			retrieved.Header.Height, block.Header.Height)
	}
}

func TestGetPreexecBlocks(t *testing.T) {
	dbPath := "/tmp/test_preexec_blocks"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 存储多个区块
	for i := uint64(100); i <= 105; i++ {
		block := createTestBlock(i)
		if err := storage.StorePreexecBlock(block); err != nil {
			t.Fatalf("Failed to store block %d: %v", i, err)
		}
	}

	// 获取区块范围
	blocks, err := storage.GetPreexecBlocks(100, 105)
	if err != nil {
		t.Fatalf("Failed to get blocks: %v", err)
	}

	if len(blocks) != 6 {
		t.Errorf("Expected 6 blocks, got %d", len(blocks))
	}

	for i, block := range blocks {
		expectedHeight := uint64(100 + i)
		if block.Header.Height != expectedHeight {
			t.Errorf("Block %d height mismatch: got %d, want %d",
				i, block.Header.Height, expectedHeight)
		}
	}
}

func TestDeleteMainBlocksFrom(t *testing.T) {
	dbPath := "/tmp/test_delete_main_blocks"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 存储多个区块
	for i := uint64(100); i <= 110; i++ {
		block := createTestBlock(i)
		if err := storage.StoreMainBlock(block); err != nil {
			t.Fatalf("Failed to store block %d: %v", i, err)
		}
	}

	// 删除从105开始的区块
	if err := storage.DeleteMainBlocksFrom(105); err != nil {
		t.Fatalf("Failed to delete blocks: %v", err)
	}

	// 验证100-104的区块仍然存在
	for i := uint64(100); i <= 104; i++ {
		_, err := storage.GetMainBlock(i)
		if err != nil {
			t.Errorf("Block %d should still exist, but got error: %v", i, err)
		}
	}

	// 验证105-110的区块已被删除
	for i := uint64(105); i <= 110; i++ {
		_, err := storage.GetMainBlock(i)
		if err == nil {
			t.Errorf("Block %d should be deleted, but still exists", i)
		}
	}
}

func TestDeletePreexecBlocks(t *testing.T) {
	dbPath := "/tmp/test_delete_preexec_blocks"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 存储多个预执行链区块
	for i := uint64(200); i <= 210; i++ {
		block := createTestBlock(i)
		if err := storage.StorePreexecBlock(block); err != nil {
			t.Fatalf("Failed to store preexec block %d: %v", i, err)
		}
	}

	// 删除从205开始的预执行链区块
	if err := storage.DeletePreexecBlocks(205); err != nil {
		t.Fatalf("Failed to delete preexec blocks: %v", err)
	}

	// 验证200-204的区块仍然存在
	for i := uint64(200); i <= 204; i++ {
		_, err := storage.GetPreexecBlock(i)
		if err != nil {
			t.Errorf("Preexec block %d should still exist, but got error: %v", i, err)
		}
	}

	// 验证205-210的区块已被删除
	for i := uint64(205); i <= 210; i++ {
		_, err := storage.GetPreexecBlock(i)
		if err == nil {
			t.Errorf("Preexec block %d should be deleted, but still exists", i)
		}
	}
}

func TestPromoteToMainChain(t *testing.T) {
	dbPath := "/tmp/test_promote_to_main"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建并存储预执行链区块
	block := createTestBlock(300)
	if err := storage.StorePreexecBlock(block); err != nil {
		t.Fatalf("Failed to store preexec block: %v", err)
	}

	// 提升到主链
	if err := storage.PromoteToMainChain(block); err != nil {
		t.Fatalf("Failed to promote block: %v", err)
	}

	// 验证区块现在在主链中
	mainBlock, err := storage.GetMainBlock(300)
	if err != nil {
		t.Fatalf("Failed to get main block after promotion: %v", err)
	}

	if mainBlock.Header.Height != block.Header.Height {
		t.Errorf("Promoted block height mismatch: got %d, want %d",
			mainBlock.Header.Height, block.Header.Height)
	}

	// 验证区块不再在预执行链中
	_, err = storage.GetPreexecBlock(300)
	if err == nil {
		t.Error("Block should not exist in preexec chain after promotion")
	}
}

func TestMakeKey(t *testing.T) {
	key1 := makeKey("test:", 100)
	key2 := makeKey("test:", 100)
	key3 := makeKey("test:", 101)

	if string(key1) != string(key2) {
		t.Error("Keys for same height should be identical")
	}

	if string(key1) == string(key3) {
		t.Error("Keys for different heights should be different")
	}

	// 验证键的格式
	if len(key1) != len("test:")+8 {
		t.Errorf("Key length should be %d, got %d", len("test:")+8, len(key1))
	}
}

func TestSerializeDeserializeBlock(t *testing.T) {
	block := createTestBlock(100)

	// 序列化
	data, err := serializeBlock(block)
	if err != nil {
		t.Fatalf("Failed to serialize block: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data should not be empty")
	}

	// 反序列化
	deserialized, err := deserializeBlock(data)
	if err != nil {
		t.Fatalf("Failed to deserialize block: %v", err)
	}

	if deserialized.Header.Height != block.Header.Height {
		t.Errorf("Height mismatch after serialization: got %d, want %d",
			deserialized.Header.Height, block.Header.Height)
	}

	if deserialized.Header.Address != block.Header.Address {
		t.Errorf("Address mismatch after serialization: got %d, want %d",
			deserialized.Header.Address, block.Header.Address)
	}
}

// 辅助函数：创建测试区块
func createTestBlock(height uint64) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Height:     height,
			ParentHash: []byte{1, 2, 3, 4},
			Timestamp:  time.Now(),
			Address:    int64(height),
			Difficulty: big.NewInt(1000),
			Nonce:      int64(height),
		},
		Txs: []*types.Tx{
			{
				Data: []byte{byte(height), 1, 2, 3},
			},
		},
	}
}

func BenchmarkStoreMainBlock(b *testing.B) {
	dbPath := "/tmp/bench_store_main"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	block := createTestBlock(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Header.Height = uint64(i)
		storage.StoreMainBlock(block)
	}
}

func BenchmarkGetMainBlock(b *testing.B) {
	dbPath := "/tmp/bench_get_main"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBDualChainStorage(dbPath)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 预先存储一些区块
	for i := 0; i < 1000; i++ {
		block := createTestBlock(uint64(i))
		storage.StoreMainBlock(block)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.GetMainBlock(uint64(i % 1000))
	}
}
