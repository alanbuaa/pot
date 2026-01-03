package upgrade

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/types"
	bolt "go.etcd.io/bbolt"
)

var (
	// Bucket names
	bucketState     = []byte("state")
	bucketProposals = []byte("proposals")
	bucketEvents    = []byte("events")
	bucketMetrics   = []byte("metrics")
)

// BoltDBPersistence BoltDB 持久化实现
type BoltDBPersistence struct {
	db  *bolt.DB
	log *logrus.Entry
}

// NewBoltDBPersistence 创建 BoltDB 持久化实例
func NewBoltDBPersistence(dbPath string, log *logrus.Entry) (*BoltDBPersistence, error) {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	// 确保目录存在
	dbFile := filepath.Join(dbPath, "upgrade.db")

	// 打开数据库
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 创建 buckets
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{bucketState, bucketProposals, bucketEvents, bucketMetrics}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("create bucket %s: %w", string(bucket), err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	log.WithField("db_file", dbFile).Info("BoltDB persistence initialized")

	return &BoltDBPersistence{
		db:  db,
		log: log,
	}, nil
}

// SaveState 保存升级状态
func (p *BoltDBPersistence) SaveState(state *UpgradeState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketState)
		return bucket.Put([]byte("current"), data)
	})
}

// LoadState 加载升级状态
func (p *BoltDBPersistence) LoadState() (*UpgradeState, error) {
	var state UpgradeState

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketState)
		data := bucket.Get([]byte("current"))
		if data == nil {
			return fmt.Errorf("no state found")
		}
		return json.Unmarshal(data, &state)
	})

	if err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveProposal 保存升级提案
func (p *BoltDBPersistence) SaveProposal(proposal *UpgradeProposal) error {
	data, err := json.Marshal(proposal)
	if err != nil {
		return fmt.Errorf("failed to marshal proposal: %w", err)
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketProposals)
		return bucket.Put(proposal.ProposalID[:], data)
	})
}

// LoadProposal 加载指定提案
func (p *BoltDBPersistence) LoadProposal(proposalID types.TxHash) (*UpgradeProposal, error) {
	var proposal UpgradeProposal

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketProposals)
		data := bucket.Get(proposalID[:])
		if data == nil {
			return fmt.Errorf("proposal not found: %s", proposalID.String())
		}
		return json.Unmarshal(data, &proposal)
	})

	if err != nil {
		return nil, err
	}

	return &proposal, nil
}

// ListProposals 列出所有提案
func (p *BoltDBPersistence) ListProposals() ([]*UpgradeProposal, error) {
	var proposals []*UpgradeProposal

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketProposals)
		return bucket.ForEach(func(k, v []byte) error {
			var proposal UpgradeProposal
			if err := json.Unmarshal(v, &proposal); err != nil {
				p.log.WithError(err).Warn("Failed to unmarshal proposal")
				return nil // 跳过损坏的数据
			}
			proposals = append(proposals, &proposal)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return proposals, nil
}

// DeleteProposal 删除提案
func (p *BoltDBPersistence) DeleteProposal(proposalID types.TxHash) error {
	return p.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketProposals)
		return bucket.Delete(proposalID[:])
	})
}

// SaveEvent 保存事件
func (p *BoltDBPersistence) SaveEvent(event *UpgradeEvent) error {
	// 生成事件 ID（使用时间戳）
	if event.ID == 0 {
		event.ID = uint64(time.Now().UnixNano())
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketEvents)
		key := uint64ToBytes(event.ID)
		return bucket.Put(key, data)
	})
}

// QueryEvents 查询事件
func (p *BoltDBPersistence) QueryEvents(filter EventFilter) ([]*UpgradeEvent, error) {
	var events []*UpgradeEvent
	limit := filter.Limit
	if limit == 0 {
		limit = 100 // 默认限制
	}

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketEvents)
		cursor := bucket.Cursor()

		// 倒序遍历（最新的在前）
		for k, v := cursor.Last(); k != nil && len(events) < limit; k, v = cursor.Prev() {
			var event UpgradeEvent
			if err := json.Unmarshal(v, &event); err != nil {
				p.log.WithError(err).Warn("Failed to unmarshal event")
				continue
			}

			// 应用过滤器
			if !matchEventFilter(&event, filter) {
				continue
			}

			events = append(events, &event)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return events, nil
}

// SaveMetrics 保存性能指标
func (p *BoltDBPersistence) SaveMetrics(height uint64, metrics *AggregateMetrics) error {
	metrics.Height = height
	metrics.Timestamp = time.Now()

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return p.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketMetrics)
		key := uint64ToBytes(height)
		return bucket.Put(key, data)
	})
}

// LoadMetrics 加载指定高度的指标
func (p *BoltDBPersistence) LoadMetrics(height uint64) (*AggregateMetrics, error) {
	var metrics AggregateMetrics

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketMetrics)
		key := uint64ToBytes(height)
		data := bucket.Get(key)
		if data == nil {
			return fmt.Errorf("metrics not found for height %d", height)
		}
		return json.Unmarshal(data, &metrics)
	})

	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

// QueryMetrics 查询指标范围
func (p *BoltDBPersistence) QueryMetrics(startHeight, endHeight uint64) (map[uint64]*AggregateMetrics, error) {
	result := make(map[uint64]*AggregateMetrics)

	err := p.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketMetrics)
		cursor := bucket.Cursor()

		min := uint64ToBytes(startHeight)

		for k, v := cursor.Seek(min); k != nil && bytesToUint64(k) <= endHeight; k, v = cursor.Next() {
			height := bytesToUint64(k)
			if height > endHeight {
				break
			}

			var metrics AggregateMetrics
			if err := json.Unmarshal(v, &metrics); err != nil {
				p.log.WithError(err).WithField("height", height).Warn("Failed to unmarshal metrics")
				continue
			}

			result[height] = &metrics
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Close 关闭持久化存储
func (p *BoltDBPersistence) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Clear 清空所有数据（测试用）
func (p *BoltDBPersistence) Clear() error {
	return p.db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{bucketState, bucketProposals, bucketEvents, bucketMetrics}
		for _, bucket := range buckets {
			if err := tx.DeleteBucket(bucket); err != nil {
				return err
			}
			if _, err := tx.CreateBucket(bucket); err != nil {
				return err
			}
		}
		return nil
	})
}

// 辅助函数

func matchEventFilter(event *UpgradeEvent, filter EventFilter) bool {
	if filter.ProposalID != nil && event.ProposalID != *filter.ProposalID {
		return false
	}
	if filter.EventType != nil && event.Type != *filter.EventType {
		return false
	}
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}

func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}

func bytesToUint64(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}
