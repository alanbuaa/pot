package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// CachedMessage 缓存的消息
type CachedMessage struct {
	MessageID   []byte       // 消息唯一标识
	MessageType string       // 消息类型
	Content     []byte       // 消息内容（序列化后）
	TargetEpoch uint64       // 目标纪元/高度
	ReceivedAt  time.Time    // 接收时间
	ProposalID  types.TxHash // 关联的升级提案ID
}

// MessageCacheStorage 消息缓存存储接口
type MessageCacheStorage interface {
	// 存储消息
	StoreMessage(msg *CachedMessage) error

	// 获取消息
	GetMessage(messageID []byte) (*CachedMessage, error)

	// 获取特定纪元的所有消息
	GetMessagesForEpoch(epoch uint64) ([]*CachedMessage, error)

	// 获取特定提案的所有消息
	GetMessagesForProposal(proposalID types.TxHash) ([]*CachedMessage, error)

	// 删除消息
	DeleteMessage(messageID []byte) error

	// 清除特定纪元之前的所有消息
	ClearMessagesBefore(epoch uint64) error

	// 清除特定提案的所有消息
	ClearMessagesForProposal(proposalID types.TxHash) error

	// 获取缓存统计
	GetCacheStats() (*CacheStats, error)

	// 关闭
	Close() error
}

// CacheStats 缓存统计信息
type CacheStats struct {
	TotalMessages  int64
	OldestEpoch    uint64
	NewestEpoch    uint64
	TotalSize      int64
	MessagesByType map[string]int64
}

// LevelDBMessageCacheStorage LevelDB 实现
type LevelDBMessageCacheStorage struct {
	db *leveldb.DB
}

// NewLevelDBMessageCacheStorage 创建消息缓存存储
func NewLevelDBMessageCacheStorage(dbPath string) (*LevelDBMessageCacheStorage, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb for message cache: %w", err)
	}
	return &LevelDBMessageCacheStorage{db: db}, nil
}

// 键前缀
const (
	messagePrefixByID       = "msg:id:"       // msg:id:<messageID>
	messagePrefixByEpoch    = "msg:epoch:"    // msg:epoch:<epoch>:<messageID>
	messagePrefixByProposal = "msg:proposal:" // msg:proposal:<proposalID>:<messageID>
	messageStatsKey         = "msg:stats"
)

// StoreMessage 存储消息
func (s *LevelDBMessageCacheStorage) StoreMessage(msg *CachedMessage) error {
	batch := new(leveldb.Batch)

	// 序列化消息
	data, err := serializeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// 主键：按 messageID 存储
	keyByID := makeMessageKeyByID(msg.MessageID)
	batch.Put(keyByID, data)

	// 索引1：按 epoch 存储
	keyByEpoch := makeMessageKeyByEpoch(msg.TargetEpoch, msg.MessageID)
	batch.Put(keyByEpoch, msg.MessageID)

	// 索引2：按 proposalID 存储
	keyByProposal := makeMessageKeyByProposal(msg.ProposalID, msg.MessageID)
	batch.Put(keyByProposal, msg.MessageID)

	// 写入批次
	if err := s.db.Write(batch, nil); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// GetMessage 获取消息
func (s *LevelDBMessageCacheStorage) GetMessage(messageID []byte) (*CachedMessage, error) {
	key := makeMessageKeyByID(messageID)
	data, err := s.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return deserializeMessage(data)
}

// GetMessagesForEpoch 获取特定纪元的所有消息
func (s *LevelDBMessageCacheStorage) GetMessagesForEpoch(epoch uint64) ([]*CachedMessage, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	prefix := makeEpochPrefix(epoch)
	messages := make([]*CachedMessage, 0)

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !hasPrefix(key, prefix) {
			break
		}

		// 提取 messageID
		messageID := iter.Value()
		msg, err := s.GetMessage(messageID)
		if err != nil {
			continue // 忽略错误，继续处理
		}
		messages = append(messages, msg)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return messages, nil
}

// GetMessagesForProposal 获取特定提案的所有消息
func (s *LevelDBMessageCacheStorage) GetMessagesForProposal(proposalID types.TxHash) ([]*CachedMessage, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	prefix := makeProposalPrefix(proposalID)
	messages := make([]*CachedMessage, 0)

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !hasPrefix(key, prefix) {
			break
		}

		// 提取 messageID
		messageID := iter.Value()
		msg, err := s.GetMessage(messageID)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return messages, nil
}

// DeleteMessage 删除消息
func (s *LevelDBMessageCacheStorage) DeleteMessage(messageID []byte) error {
	// 先获取消息以获得索引信息
	msg, err := s.GetMessage(messageID)
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)

	// 删除主键
	keyByID := makeMessageKeyByID(messageID)
	batch.Delete(keyByID)

	// 删除索引
	keyByEpoch := makeMessageKeyByEpoch(msg.TargetEpoch, messageID)
	batch.Delete(keyByEpoch)

	keyByProposal := makeMessageKeyByProposal(msg.ProposalID, messageID)
	batch.Delete(keyByProposal)

	return s.db.Write(batch, nil)
}

// ClearMessagesBefore 清除特定纪元之前的所有消息
func (s *LevelDBMessageCacheStorage) ClearMessagesBefore(epoch uint64) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)

	// 遍历所有按 epoch 索引的消息
	prefix := []byte(messagePrefixByEpoch)
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !hasPrefix(key, prefix) {
			break
		}

		// 提取 epoch
		msgEpoch := extractEpochFromKey(key, len(prefix))
		if msgEpoch >= epoch {
			continue
		}

		// 获取 messageID 并删除
		messageID := iter.Value()
		msg, err := s.GetMessage(messageID)
		if err != nil {
			continue
		}

		// 删除所有相关键
		batch.Delete(makeMessageKeyByID(messageID))
		batch.Delete(makeMessageKeyByEpoch(msg.TargetEpoch, messageID))
		batch.Delete(makeMessageKeyByProposal(msg.ProposalID, messageID))
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return s.db.Write(batch, nil)
}

// ClearMessagesForProposal 清除特定提案的所有消息
func (s *LevelDBMessageCacheStorage) ClearMessagesForProposal(proposalID types.TxHash) error {
	messages, err := s.GetMessagesForProposal(proposalID)
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)
	for _, msg := range messages {
		batch.Delete(makeMessageKeyByID(msg.MessageID))
		batch.Delete(makeMessageKeyByEpoch(msg.TargetEpoch, msg.MessageID))
		batch.Delete(makeMessageKeyByProposal(msg.ProposalID, msg.MessageID))
	}

	return s.db.Write(batch, nil)
}

// GetCacheStats 获取缓存统计
func (s *LevelDBMessageCacheStorage) GetCacheStats() (*CacheStats, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	stats := &CacheStats{
		MessagesByType: make(map[string]int64),
		OldestEpoch:    ^uint64(0), // 最大值
		NewestEpoch:    0,
	}

	prefix := []byte(messagePrefixByID)
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !hasPrefix(key, prefix) {
			break
		}

		msg, err := deserializeMessage(iter.Value())
		if err != nil {
			continue
		}

		stats.TotalMessages++
		stats.TotalSize += int64(len(iter.Value()))
		stats.MessagesByType[msg.MessageType]++

		if msg.TargetEpoch < stats.OldestEpoch {
			stats.OldestEpoch = msg.TargetEpoch
		}
		if msg.TargetEpoch > stats.NewestEpoch {
			stats.NewestEpoch = msg.TargetEpoch
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return stats, nil
}

// Close 关闭数据库
func (s *LevelDBMessageCacheStorage) Close() error {
	return s.db.Close()
}

// 辅助函数

func makeMessageKeyByID(messageID []byte) []byte {
	key := make([]byte, len(messagePrefixByID)+len(messageID))
	copy(key, messagePrefixByID)
	copy(key[len(messagePrefixByID):], messageID)
	return key
}

func makeMessageKeyByEpoch(epoch uint64, messageID []byte) []byte {
	prefix := messagePrefixByEpoch
	key := make([]byte, len(prefix)+8+len(messageID))
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], epoch)
	copy(key[len(prefix)+8:], messageID)
	return key
}

func makeMessageKeyByProposal(proposalID types.TxHash, messageID []byte) []byte {
	prefix := messagePrefixByProposal
	key := make([]byte, len(prefix)+len(proposalID)+len(messageID))
	copy(key, prefix)
	copy(key[len(prefix):], proposalID[:])
	copy(key[len(prefix)+len(proposalID):], messageID)
	return key
}

func makeEpochPrefix(epoch uint64) []byte {
	prefix := messagePrefixByEpoch
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], epoch)
	return key
}

func makeProposalPrefix(proposalID types.TxHash) []byte {
	prefix := messagePrefixByProposal
	key := make([]byte, len(prefix)+len(proposalID))
	copy(key, prefix)
	copy(key[len(prefix):], proposalID[:])
	return key
}

func hasPrefix(data, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

func extractEpochFromKey(key []byte, prefixLen int) uint64 {
	if len(key) < prefixLen+8 {
		return 0
	}
	return binary.BigEndian.Uint64(key[prefixLen : prefixLen+8])
}

func serializeMessage(msg *CachedMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func deserializeMessage(data []byte) (*CachedMessage, error) {
	msg := &CachedMessage{}
	err := json.Unmarshal(data, msg)
	return msg, err
}
