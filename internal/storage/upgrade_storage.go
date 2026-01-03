package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// UpgradeStorage 升级相关数据的持久化存储
type UpgradeStorage interface {
	// 检查点操作
	StoreCheckpoint(checkpoint *RollbackCheckpoint) error
	GetCheckpoint(height uint64) (*RollbackCheckpoint, error)
	GetLatestCheckpoint() (*RollbackCheckpoint, error)
	ListCheckpoints(limit int) ([]*RollbackCheckpoint, error)
	DeleteCheckpointsAfter(height uint64) error

	// 投票记录操作
	StoreVote(proposalID types.TxHash, vote *VoteRecord) error
	GetVote(proposalID types.TxHash, nodeID int64) (*VoteRecord, error)
	GetProposalVotes(proposalID types.TxHash) ([]*VoteRecord, error)
	StoreProposalStatus(proposalID types.TxHash, status *ProposalVoteStatus) error
	GetProposalStatus(proposalID types.TxHash) (*ProposalVoteStatus, error)
	DeleteProposalData(proposalID types.TxHash) error

	// 关闭
	Close() error
}

// RollbackCheckpoint 回退检查点（持久化结构）
type RollbackCheckpoint struct {
	Height        uint64    `json:"height"`
	ConsensusType string    `json:"consensus_type"`
	Timestamp     time.Time `json:"timestamp"`
	StateHash     []byte    `json:"state_hash"`
	BlockHash     []byte    `json:"block_hash"`
	Description   string    `json:"description"`
}

// VoteRecord 投票记录（持久化结构）
type VoteRecord struct {
	ProposalID types.TxHash `json:"proposal_id"`
	NodeID     int64        `json:"node_id"`
	Option     int          `json:"option"` // 0=Abstain, 1=Yes, 2=No
	Timestamp  time.Time    `json:"timestamp"`
	Signature  []byte       `json:"signature"`
}

// ProposalVoteStatus 提案投票状态（持久化结构）
type ProposalVoteStatus struct {
	ProposalID    types.TxHash `json:"proposal_id"`
	Status        int          `json:"status"` // 0=Pending, 1=Passed, 2=Rejected, 3=Expired
	StartTime     time.Time    `json:"start_time"`
	EndTime       time.Time    `json:"end_time"`
	VotingPeriod  int64        `json:"voting_period"` // 纳秒
	YesCount      int          `json:"yes_count"`
	NoCount       int          `json:"no_count"`
	AbstainCount  int          `json:"abstain_count"`
	QuorumReached bool         `json:"quorum_reached"`
}

// LevelDBUpgradeStorage LevelDB 实现的升级存储
type LevelDBUpgradeStorage struct {
	db *leveldb.DB
}

// NewLevelDBUpgradeStorage 创建升级存储
func NewLevelDBUpgradeStorage(dbPath string) (*LevelDBUpgradeStorage, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return &LevelDBUpgradeStorage{db: db}, nil
}

// 键前缀
const (
	checkpointPrefix     = "checkpoint:"
	checkpointLatestKey  = "checkpoint:latest"
	votePrefix           = "vote:"
	proposalStatusPrefix = "proposal_status:"
	proposalVotesPrefix  = "proposal_votes:"
)

// StoreCheckpoint 存储检查点
func (s *LevelDBUpgradeStorage) StoreCheckpoint(checkpoint *RollbackCheckpoint) error {
	// 存储检查点数据
	key := makeCheckpointKey(checkpoint.Height)
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	batch := new(leveldb.Batch)
	batch.Put(key, value)

	// 更新最新检查点引用
	latestKey := []byte(checkpointLatestKey)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, checkpoint.Height)
	batch.Put(latestKey, heightBytes)

	return s.db.Write(batch, nil)
}

// GetCheckpoint 获取指定高度的检查点
func (s *LevelDBUpgradeStorage) GetCheckpoint(height uint64) (*RollbackCheckpoint, error) {
	key := makeCheckpointKey(height)
	value, err := s.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("checkpoint not found at height %d", height)
		}
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	var checkpoint RollbackCheckpoint
	if err := json.Unmarshal(value, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}
	return &checkpoint, nil
}

// GetLatestCheckpoint 获取最新检查点
func (s *LevelDBUpgradeStorage) GetLatestCheckpoint() (*RollbackCheckpoint, error) {
	// 读取最新检查点高度
	latestKey := []byte(checkpointLatestKey)
	heightBytes, err := s.db.Get(latestKey, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("no checkpoints found")
		}
		return nil, fmt.Errorf("failed to get latest checkpoint reference: %w", err)
	}

	height := binary.BigEndian.Uint64(heightBytes)
	return s.GetCheckpoint(height)
}

// ListCheckpoints 列出最近的检查点
func (s *LevelDBUpgradeStorage) ListCheckpoints(limit int) ([]*RollbackCheckpoint, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	checkpoints := make([]*RollbackCheckpoint, 0, limit)
	prefix := []byte(checkpointPrefix)

	// 从后向前遍历
	iter.Last()
	for iter.Valid() && len(checkpoints) < limit {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == checkpointPrefix {
			// 跳过 latest 键
			if string(key) != checkpointLatestKey {
				var checkpoint RollbackCheckpoint
				if err := json.Unmarshal(iter.Value(), &checkpoint); err == nil {
					checkpoints = append(checkpoints, &checkpoint)
				}
			}
		}
		iter.Prev()
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return checkpoints, nil
}

// DeleteCheckpointsAfter 删除指定高度之后的检查点
func (s *LevelDBUpgradeStorage) DeleteCheckpointsAfter(height uint64) error {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)
	prefix := []byte(checkpointPrefix)

	for iter.Seek(makeCheckpointKey(height + 1)); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == checkpointPrefix {
			// 不删除 latest 键，稍后更新
			if string(key) != checkpointLatestKey {
				batch.Delete(key)
			}
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	// 更新 latest 引用到指定高度
	if height > 0 {
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, height)
		batch.Put([]byte(checkpointLatestKey), heightBytes)
	} else {
		batch.Delete([]byte(checkpointLatestKey))
	}

	return s.db.Write(batch, nil)
}

// StoreVote 存储投票记录
func (s *LevelDBUpgradeStorage) StoreVote(proposalID types.TxHash, vote *VoteRecord) error {
	key := makeVoteKey(proposalID, vote.NodeID)
	value, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %w", err)
	}
	return s.db.Put(key, value, nil)
}

// GetVote 获取投票记录
func (s *LevelDBUpgradeStorage) GetVote(proposalID types.TxHash, nodeID int64) (*VoteRecord, error) {
	key := makeVoteKey(proposalID, nodeID)
	value, err := s.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("vote not found")
		}
		return nil, fmt.Errorf("failed to get vote: %w", err)
	}

	var vote VoteRecord
	if err := json.Unmarshal(value, &vote); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vote: %w", err)
	}
	return &vote, nil
}

// GetProposalVotes 获取提案的所有投票
func (s *LevelDBUpgradeStorage) GetProposalVotes(proposalID types.TxHash) ([]*VoteRecord, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	votes := make([]*VoteRecord, 0)
	prefix := makeProposalVotesPrefix(proposalID)

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(prefix) && string(key[:len(prefix)]) == string(prefix) {
			var vote VoteRecord
			if err := json.Unmarshal(iter.Value(), &vote); err == nil {
				votes = append(votes, &vote)
			}
		} else {
			break
		}
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return votes, nil
}

// StoreProposalStatus 存储提案投票状态
func (s *LevelDBUpgradeStorage) StoreProposalStatus(proposalID types.TxHash, status *ProposalVoteStatus) error {
	key := makeProposalStatusKey(proposalID)
	value, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal proposal status: %w", err)
	}
	return s.db.Put(key, value, nil)
}

// GetProposalStatus 获取提案投票状态
func (s *LevelDBUpgradeStorage) GetProposalStatus(proposalID types.TxHash) (*ProposalVoteStatus, error) {
	key := makeProposalStatusKey(proposalID)
	value, err := s.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("proposal status not found")
		}
		return nil, fmt.Errorf("failed to get proposal status: %w", err)
	}

	var status ProposalVoteStatus
	if err := json.Unmarshal(value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal status: %w", err)
	}
	return &status, nil
}

// DeleteProposalData 删除提案相关的所有数据
func (s *LevelDBUpgradeStorage) DeleteProposalData(proposalID types.TxHash) error {
	batch := new(leveldb.Batch)

	// 删除投票记录
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	votesPrefix := makeProposalVotesPrefix(proposalID)
	for iter.Seek(votesPrefix); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= len(votesPrefix) && string(key[:len(votesPrefix)]) == string(votesPrefix) {
			batch.Delete(key)
		} else {
			break
		}
	}

	// 删除提案状态
	statusKey := makeProposalStatusKey(proposalID)
	batch.Delete(statusKey)

	return s.db.Write(batch, nil)
}

// Close 关闭存储
func (s *LevelDBUpgradeStorage) Close() error {
	return s.db.Close()
}

// 辅助函数
func makeCheckpointKey(height uint64) []byte {
	key := make([]byte, len(checkpointPrefix)+8)
	copy(key, checkpointPrefix)
	binary.BigEndian.PutUint64(key[len(checkpointPrefix):], height)
	return key
}

func makeVoteKey(proposalID types.TxHash, nodeID int64) []byte {
	prefix := makeProposalVotesPrefix(proposalID)
	key := make([]byte, len(prefix)+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix):], uint64(nodeID))
	return key
}

func makeProposalVotesPrefix(proposalID types.TxHash) []byte {
	prefix := votePrefix + proposalID.String() + ":"
	return []byte(prefix)
}

func makeProposalStatusKey(proposalID types.TxHash) []byte {
	key := proposalStatusPrefix + proposalID.String()
	return []byte(key)
}
