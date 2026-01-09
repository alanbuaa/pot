package storage

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestNewLevelDBMultiChainStorage(t *testing.T) {
	dbPath := "/tmp/test_multi_chain_storage"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
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

	storage, err := NewLevelDBMultiChainStorage(dbPath)
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

func TestStoreAndGetCandidateBlock(t *testing.T) {
	dbPath := "/tmp/test_candidate_block"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建测试区块
	block := createTestBlock(200)
	candidateID := "pow-upgrade-1"

	// 存储候选链区块
	if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
		t.Fatalf("Failed to store candidate block: %v", err)
	}

	// 获取候选链区块
	retrieved, err := storage.GetCandidateBlock(candidateID, 200)
	if err != nil {
		t.Fatalf("Failed to get candidate block: %v", err)
	}

	if retrieved.Header.Height != block.Header.Height {
		t.Errorf("Height mismatch: got %d, want %d",
			retrieved.Header.Height, block.Header.Height)
	}
}

func TestGetCandidateBlocks(t *testing.T) {
	dbPath := "/tmp/test_candidate_blocks"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	candidateID := "hotstuff-upgrade-1"

	// 存储多个区块
	for i := uint64(100); i <= 105; i++ {
		block := createTestBlock(i)
		if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
			t.Fatalf("Failed to store block %d: %v", i, err)
		}
	}

	// 获取区块范围
	blocks, err := storage.GetCandidateBlocks(candidateID, 100, 105)
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

	storage, err := NewLevelDBMultiChainStorage(dbPath)
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

func TestDeleteCandidateBlocksFrom(t *testing.T) {
	dbPath := "/tmp/test_delete_candidate_blocks"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	candidateID := "pow-upgrade-2"

	// 存储多个候选链区块
	for i := uint64(200); i <= 210; i++ {
		block := createTestBlock(i)
		if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
			t.Fatalf("Failed to store candidate block %d: %v", i, err)
		}
	}

	// 删除从205开始的候选链区块
	if err := storage.DeleteCandidateBlocksFrom(candidateID, 205); err != nil {
		t.Fatalf("Failed to delete candidate blocks: %v", err)
	}

	// 验证200-204的区块仍然存在
	for i := uint64(200); i <= 204; i++ {
		_, err := storage.GetCandidateBlock(candidateID, i)
		if err != nil {
			t.Errorf("Candidate block %d should still exist, but got error: %v", i, err)
		}
	}

	// 验证205-210的区块已被删除
	for i := uint64(205); i <= 210; i++ {
		_, err := storage.GetCandidateBlock(candidateID, i)
		if err == nil {
			t.Errorf("Candidate block %d should be deleted, but still exists", i)
		}
	}
}

func TestDeleteCandidateChain(t *testing.T) {
	dbPath := "/tmp/test_delete_candidate_chain"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	candidateID := "test-candidate-1"

	// 存储多个候选链区块
	for i := uint64(100); i <= 110; i++ {
		block := createTestBlock(i)
		if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
			t.Fatalf("Failed to store candidate block %d: %v", i, err)
		}
	}

	// 删除整个候选链
	if err := storage.DeleteCandidateChain(candidateID); err != nil {
		t.Fatalf("Failed to delete candidate chain: %v", err)
	}

	// 验证所有区块都已被删除
	for i := uint64(100); i <= 110; i++ {
		_, err := storage.GetCandidateBlock(candidateID, i)
		if err == nil {
			t.Errorf("Candidate block %d should be deleted", i)
		}
	}

	// 验证候选链不再在列表中
	chains, err := storage.ListCandidateChains()
	if err != nil {
		t.Fatalf("Failed to list candidate chains: %v", err)
	}

	for _, chain := range chains {
		if chain == candidateID {
			t.Errorf("Deleted candidate chain %s should not be in list", candidateID)
		}
	}
}

func TestListCandidateChains(t *testing.T) {
	dbPath := "/tmp/test_list_candidates"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建多个候选链
	candidateIDs := []string{"pow-upgrade", "hotstuff-upgrade", "custom-upgrade"}

	for _, candidateID := range candidateIDs {
		block := createTestBlock(100)
		if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
			t.Fatalf("Failed to store candidate block for %s: %v", candidateID, err)
		}
	}

	// 列出所有候选链
	chains, err := storage.ListCandidateChains()
	if err != nil {
		t.Fatalf("Failed to list candidate chains: %v", err)
	}

	if len(chains) != len(candidateIDs) {
		t.Errorf("Expected %d candidate chains, got %d", len(candidateIDs), len(chains))
	}

	// 验证所有候选链都在列表中
	chainMap := make(map[string]bool)
	for _, chain := range chains {
		chainMap[chain] = true
	}

	for _, expectedID := range candidateIDs {
		if !chainMap[expectedID] {
			t.Errorf("Expected candidate chain %s not found in list", expectedID)
		}
	}
}

func TestPromoteToMainChain(t *testing.T) {
	dbPath := "/tmp/test_promote_to_main"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	candidateID := "pow-upgrade-final"

	// 创建并存储候选链区块
	block := createTestBlock(300)
	if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
		t.Fatalf("Failed to store candidate block: %v", err)
	}

	// 提升到主链
	if err := storage.PromoteToMainChain(candidateID, block); err != nil {
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

	// 验证区块不再在候选链中
	_, err = storage.GetCandidateBlock(candidateID, 300)
	if err == nil {
		t.Error("Block should not exist in candidate chain after promotion")
	}
}

func TestMultipleCandidateChains(t *testing.T) {
	dbPath := "/tmp/test_multiple_candidates"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 创建多个候选链，每个都有多个区块
	candidates := map[string][]uint64{
		"pow-1":      {100, 101, 102},
		"hotstuff-1": {100, 101, 102, 103},
		"custom-1":   {100, 101},
	}

	for candidateID, heights := range candidates {
		for _, height := range heights {
			block := createTestBlock(height)
			if err := storage.StoreCandidateBlock(candidateID, block); err != nil {
				t.Fatalf("Failed to store block for %s at height %d: %v",
					candidateID, height, err)
			}
		}
	}

	// 验证每个候选链的区块
	for candidateID, heights := range candidates {
		for _, height := range heights {
			block, err := storage.GetCandidateBlock(candidateID, height)
			if err != nil {
				t.Errorf("Failed to get block for %s at height %d: %v",
					candidateID, height, err)
			}
			if block.Header.Height != height {
				t.Errorf("Height mismatch for %s: got %d, want %d",
					candidateID, block.Header.Height, height)
			}
		}
	}

	// 验证候选链列表
	chains, err := storage.ListCandidateChains()
	if err != nil {
		t.Fatalf("Failed to list candidate chains: %v", err)
	}

	if len(chains) != len(candidates) {
		t.Errorf("Expected %d candidate chains, got %d", len(candidates), len(chains))
	}
}

func TestMakeCandidateKey(t *testing.T) {
	key1 := makeCandidateKey("pow-1", 100)
	key2 := makeCandidateKey("pow-1", 100)
	key3 := makeCandidateKey("pow-1", 101)
	key4 := makeCandidateKey("pow-2", 100)

	if string(key1) != string(key2) {
		t.Error("Keys for same candidate and height should be identical")
	}

	if string(key1) == string(key3) {
		t.Error("Keys for different heights should be different")
	}

	if string(key1) == string(key4) {
		t.Error("Keys for different candidates should be different")
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

	storage, err := NewLevelDBMultiChainStorage(dbPath)
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

	storage, err := NewLevelDBMultiChainStorage(dbPath)
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

func BenchmarkStoreCandidateBlock(b *testing.B) {
	dbPath := "/tmp/bench_store_candidate"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	block := createTestBlock(100)
	candidateID := "bench-candidate"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Header.Height = uint64(i)
		storage.StoreCandidateBlock(candidateID, block)
	}
}

func BenchmarkGetCandidateBlock(b *testing.B) {
	dbPath := "/tmp/bench_get_candidate"
	defer os.RemoveAll(dbPath)

	storage, err := NewLevelDBMultiChainStorage(dbPath)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	candidateID := "bench-candidate"

	// 预先存储一些区块
	for i := 0; i < 1000; i++ {
		block := createTestBlock(uint64(i))
		storage.StoreCandidateBlock(candidateID, block)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.GetCandidateBlock(candidateID, uint64(i%1000))
	}
}
