package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// DualChainStorage 双链存储接口
type DualChainStorage interface {
	// 主链操作
	StoreMainBlock(block *types.Block) error
	GetMainBlock(height uint64) (*types.Block, error)
	DeleteMainBlocksFrom(height uint64) error

	// 预执行链操作
	StorePreexecBlock(block *types.Block) error
	GetPreexecBlock(height uint64) (*types.Block, error)
	GetPreexecBlocks(from, to uint64) ([]*types.Block, error)
	DeletePreexecBlocks(forkPoint uint64) error

	// 提升操作
	PromoteToMainChain(block *types.Block) error

	// 关闭
	Close() error
}

// LevelDBDualChainStorage LevelDB 实现
type LevelDBDualChainStorage struct {
	db *leveldb.DB
}

// NewLevelDBDualChainStorage 创建存储
func NewLevelDBDualChainStorage(dbPath string) (*LevelDBDualChainStorage, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return &LevelDBDualChainStorage{db: db}, nil
}

// 键前缀
const (
	mainChainPrefix    = "main:"
	preexecChainPrefix = "preexec:"
)

// StoreMainBlock 存储主链区块
func (s *LevelDBDualChainStorage) StoreMainBlock(block *types.Block) error {
	key := makeKey(mainChainPrefix, block.Header.Height)
	value, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}
	return s.db.Put(key, value, nil)
}

// GetMainBlock 获取主链区块
func (s *LevelDBDualChainStorage) GetMainBlock(height uint64) (*types.Block, error) {
	key := makeKey(mainChainPrefix, height)
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	return deserializeBlock(value)
}

// DeleteMainBlocksFrom 删除指定高度之后的主链区块
func (s *LevelDBDualChainStorage) DeleteMainBlocksFrom(height uint64) error {
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

// StorePreexecBlock 存储预执行链区块
func (s *LevelDBDualChainStorage) StorePreexecBlock(block *types.Block) error {
	key := makeKey(preexecChainPrefix, block.Header.Height)
	value, err := serializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}
	return s.db.Put(key, value, nil)
}

// GetPreexecBlock 获取预执行链区块
func (s *LevelDBDualChainStorage) GetPreexecBlock(height uint64) (*types.Block, error) {
	key := makeKey(preexecChainPrefix, height)
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get preexec block: %w", err)
	}
	return deserializeBlock(value)
}

// GetPreexecBlocks 获取预执行链区块范围
func (s *LevelDBDualChainStorage) GetPreexecBlocks(from, to uint64) ([]*types.Block, error) {
	blocks := make([]*types.Block, 0, to-from+1)
	for height := from; height <= to; height++ {
		block, err := s.GetPreexecBlock(height)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %w", height, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// DeletePreexecBlocks 删除预执行链
func (s *LevelDBDualChainStorage) DeletePreexecBlocks(forkPoint uint64) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	prefix := []byte(preexecChainPrefix)
	for iter.Seek(makeKey(preexecChainPrefix, forkPoint)); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == preexecChainPrefix {
			batch.Delete(key)
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return s.db.Write(batch, nil)
}

// PromoteToMainChain 将预执行链区块提升为主链区块
func (s *LevelDBDualChainStorage) PromoteToMainChain(block *types.Block) error {
	batch := new(leveldb.Batch)

	// 从预执行链删除
	preexecKey := makeKey(preexecChainPrefix, block.Header.Height)
	batch.Delete(preexecKey)

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
func (s *LevelDBDualChainStorage) Close() error {
	return s.db.Close()
}

// 辅助函数
func makeKey(prefix string, height uint64) []byte {
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
