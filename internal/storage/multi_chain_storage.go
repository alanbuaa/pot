package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// MultiChainStorage 多链存储接口
type MultiChainStorage interface {
	// 主链操作
	StoreMainBlock(block *types.Block) error
	GetMainBlock(height uint64) (*types.Block, error)
	DeleteMainBlocksFrom(height uint64) error

	// 候选链操作
	StoreCandidateBlock(candidateID string, block *types.Block) error
	GetCandidateBlock(candidateID string, height uint64) (*types.Block, error)
	GetCandidateBlocks(candidateID string, from, to uint64) ([]*types.Block, error)
	DeleteCandidateChain(candidateID string) error
	DeleteCandidateBlocksFrom(candidateID string, forkPoint uint64) error
	ListCandidateChains() ([]string, error)

	// 提升操作
	PromoteToMainChain(candidateID string, block *types.Block) error

	// 关闭
	Close() error
}

// LevelDBMultiChainStorage LevelDB 实现
type LevelDBMultiChainStorage struct {
	db *leveldb.DB
}

// NewLevelDBMultiChainStorage 创建存储
func NewLevelDBMultiChainStorage(dbPath string) (*LevelDBMultiChainStorage, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return &LevelDBMultiChainStorage{db: db}, nil
}

// 键前缀
const (
	mainChainPrefix      = "main:"
	candidateChainPrefix = "candidate:" // candidate:<candidateID>:<height>
	candidateListPrefix  = "candidate_list:"
)

// StoreMainBlock 存储主链区块
func (s *LevelDBMultiChainStorage) StoreMainBlock(block *types.Block) error {
	key := makeKey(mainChainPrefix, block.Header.Height)
	value, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}
	return s.db.Put(key, value, nil)
}

// GetMainBlock 获取主链区块
func (s *LevelDBMultiChainStorage) GetMainBlock(height uint64) (*types.Block, error) {
	key := makeKey(mainChainPrefix, height)
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	return deserializeBlock(value)
}

// DeleteMainBlocksFrom 删除指定高度之后的主链区块
func (s *LevelDBMultiChainStorage) DeleteMainBlocksFrom(height uint64) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	prefix := []byte(mainChainPrefix)
	for iter.Seek(makeKey(mainChainPrefix, height)); iter.Valid(); iter.Next() {
		key := iter.Key()
		// 检查是否是主链键
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == mainChainPrefix {
			batch.Delete(key)
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return s.db.Write(batch, nil)
}

// StoreCandidateBlock 存储候选链区块
func (s *LevelDBMultiChainStorage) StoreCandidateBlock(candidateID string, block *types.Block) error {
	key := makeCandidateKey(candidateID, block.Header.Height)
	value, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}

	// 批量操作：存储区块 + 更新候选链列表
	batch := new(leveldb.Batch)
	batch.Put(key, value)

	// 记录候选链ID
	listKey := []byte(candidateListPrefix + candidateID)
	batch.Put(listKey, []byte("1"))

	return s.db.Write(batch, nil)
}

// GetCandidateBlock 获取候选链区块
func (s *LevelDBMultiChainStorage) GetCandidateBlock(candidateID string, height uint64) (*types.Block, error) {
	key := makeCandidateKey(candidateID, height)
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate block: %w", err)
	}
	return deserializeBlock(value)
}

// GetCandidateBlocks 获取候选链区块范围
func (s *LevelDBMultiChainStorage) GetCandidateBlocks(candidateID string, from, to uint64) ([]*types.Block, error) {
	blocks := make([]*types.Block, 0, to-from+1)
	for height := from; height <= to; height++ {
		block, err := s.GetCandidateBlock(candidateID, height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %w", height, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// DeleteCandidateChain 删除整个候选链
func (s *LevelDBMultiChainStorage) DeleteCandidateChain(candidateID string) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	prefix := []byte(candidateChainPrefix + candidateID + ":")
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == string(prefix) {
			batch.Delete(key)
		} else {
			break
		}
	}

	// 删除候选链列表记录
	listKey := []byte(candidateListPrefix + candidateID)
	batch.Delete(listKey)

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return s.db.Write(batch, nil)
}

// DeleteCandidateBlocksFrom 删除候选链指定高度之后的区块
func (s *LevelDBMultiChainStorage) DeleteCandidateBlocksFrom(candidateID string, forkPoint uint64) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	prefix := []byte(candidateChainPrefix + candidateID + ":")
	startKey := makeCandidateKey(candidateID, forkPoint)
	for iter.Seek(startKey); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == string(prefix) {
			batch.Delete(key)
		} else {
			break
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return s.db.Write(batch, nil)
}

// ListCandidateChains 列出所有候选链ID
func (s *LevelDBMultiChainStorage) ListCandidateChains() ([]string, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	candidates := make([]string, 0)
	prefix := []byte(candidateListPrefix)

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) > len(prefix) && string(key[:len(prefix)]) == candidateListPrefix {
			candidateID := string(key[len(prefix):])
			candidates = append(candidates, candidateID)
		} else {
			break
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return candidates, nil
}

// PromoteToMainChain 将候选链区块提升为主链区块
func (s *LevelDBMultiChainStorage) PromoteToMainChain(candidateID string, block *types.Block) error {
	batch := new(leveldb.Batch)

	// 从候选链删除
	candidateKey := makeCandidateKey(candidateID, block.Header.Height)
	batch.Delete(candidateKey)

	// 添加到主链
	mainKey := makeKey(mainChainPrefix, block.Header.Height)
	value, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}
	batch.Put(mainKey, value)

	return s.db.Write(batch, nil)
}

// Close 关闭数据库
func (s *LevelDBMultiChainStorage) Close() error {
	return s.db.Close()
}

// 辅助函数
func makeKey(prefix string, height uint64) []byte {
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], height)
	return key
}

func makeCandidateKey(candidateID string, height uint64) []byte {
	prefix := candidateChainPrefix + candidateID + ":"
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], height)
	return key
}

func serializeBlock(block *types.Block) ([]byte, error) {
	// 使用 JSON 序列化
	return json.Marshal(block)
}

func deserializeBlock(data []byte) (*types.Block, error) {
	block := &types.Block{}
	err := json.Unmarshal(data, block)
	return block, err
}
