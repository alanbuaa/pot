package upgrade

import (
	"time"

	"github.com/zzz136454872/upgradeable-consensus/types"
)

// UpgradeEvent 升级事件
type UpgradeEvent struct {
	ID          uint64                 `json:"id"`
	Type        EventType              `json:"type"`
	ProposalID  types.TxHash           `json:"proposal_id"`
	Phase       UpgradePhase           `json:"phase"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EventType 事件类型
type EventType int

const (
	EventProposalCreated EventType = iota
	EventUpgradeStarted
	EventPhaseChanged
	EventSwitchExecuted
	EventRollbackExecuted
	EventUpgradeCompleted
	EventUpgradeFailed
)

func (t EventType) String() string {
	switch t {
	case EventProposalCreated:
		return "ProposalCreated"
	case EventUpgradeStarted:
		return "UpgradeStarted"
	case EventPhaseChanged:
		return "PhaseChanged"
	case EventSwitchExecuted:
		return "SwitchExecuted"
	case EventRollbackExecuted:
		return "RollbackExecuted"
	case EventUpgradeCompleted:
		return "UpgradeCompleted"
	case EventUpgradeFailed:
		return "UpgradeFailed"
	default:
		return "Unknown"
	}
}

// EventFilter 事件查询过滤器
type EventFilter struct {
	ProposalID *types.TxHash `json:"proposal_id,omitempty"`
	EventType  *EventType    `json:"event_type,omitempty"`
	StartTime  *time.Time    `json:"start_time,omitempty"`
	EndTime    *time.Time    `json:"end_time,omitempty"`
	Limit      int           `json:"limit,omitempty"`
}

// UpgradePersistence 升级状态持久化接口
type UpgradePersistence interface {
	// SaveState 保存升级状态
	SaveState(state *UpgradeState) error

	// LoadState 加载升级状态
	LoadState() (*UpgradeState, error)

	// SaveProposal 保存升级提案
	SaveProposal(proposal *UpgradeProposal) error

	// LoadProposal 加载指定提案
	LoadProposal(proposalID types.TxHash) (*UpgradeProposal, error)

	// ListProposals 列出所有提案
	ListProposals() ([]*UpgradeProposal, error)

	// DeleteProposal 删除提案
	DeleteProposal(proposalID types.TxHash) error

	// SaveEvent 保存事件
	SaveEvent(event *UpgradeEvent) error

	// QueryEvents 查询事件
	QueryEvents(filter EventFilter) ([]*UpgradeEvent, error)

	// SaveMetrics 保存性能指标
	SaveMetrics(height uint64, metrics *AggregateMetrics) error

	// LoadMetrics 加载指定高度的指标
	LoadMetrics(height uint64) (*AggregateMetrics, error)

	// QueryMetrics 查询指标范围
	QueryMetrics(startHeight, endHeight uint64) (map[uint64]*AggregateMetrics, error)

	// Close 关闭持久化存储
	Close() error

	// Clear 清空所有数据（测试用）
	Clear() error
}

// AggregateMetrics 聚合性能指标（用于持久化）
type AggregateMetrics struct {
	Height        uint64    `json:"height"`
	AvgBlockTime  float64   `json:"avg_block_time"`
	AvgTxPerBlock float64   `json:"avg_tx_per_block"`
	TotalTx       uint64    `json:"total_tx"`
	BlockCount    uint64    `json:"block_count"`
	MaxBlockTime  float64   `json:"max_block_time"`
	MinBlockTime  float64   `json:"min_block_time"`
	Timestamp     time.Time `json:"timestamp"`
}
