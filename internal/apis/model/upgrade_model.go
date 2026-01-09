package model

import (
	"time"

	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
)

// UpgradeService defines the interface for upgrade consensus service
type UpgradeService interface {
	// GetUpgradeManager returns the upgrade manager
	GetUpgradeManager() *upgrade.UpgradeManager
}

// ProposeUpgradeRequest represents the request to propose an upgrade
type ProposeUpgradeRequest struct {
	TargetConsensus      string                 `json:"target_consensus" binding:"required"`
	CDLYaml              string                 `json:"cdl_yaml"`
	ForkHeight           uint64                 `json:"fork_height"`
	CandidateStartHeight uint64                 `json:"candidate_start_height" binding:"required"`
	SwitchHeight         uint64                 `json:"switch_height" binding:"required"`
	Incentive          uint64                 `json:"incentive"`
	Description        string                 `json:"description"`
	ConsensusParams    map[string]interface{} `json:"consensus_params"`
}

// StartUpgradeRequest represents the request to start an upgrade
type StartUpgradeRequest struct {
	ProposalID string `json:"proposal_id" binding:"required"`
}

// RollbackRequest represents the request to rollback an upgrade
type RollbackRequest struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

// ValidateCDLRequest represents the request to validate CDL
type ValidateCDLRequest struct {
	CDLYaml string `json:"cdl_yaml" binding:"required"`
}

// CompileCDLRequest represents the request to compile CDL
type CompileCDLRequest struct {
	CDLYaml string `json:"cdl_yaml" binding:"required"`
}

// QueryEventsRequest represents the request to query events
type QueryEventsRequest struct {
	ProposalID string `json:"proposal_id"`
	EventType  *int   `json:"event_type"`
	Limit      int    `json:"limit"`
}

// UpgradeStatusResponse represents the upgrade status response
type UpgradeStatusResponse struct {
	Phase           string              `json:"phase"`
	Started         bool                `json:"started"`
	Completed       bool                `json:"completed"`
	Failed          bool                `json:"failed"`
	StartTime       *int64              `json:"start_time,omitempty"`
	EndTime         *int64              `json:"end_time,omitempty"`
	FailureReason   string              `json:"failure_reason,omitempty"`
	CurrentProposal *ProposalSummary    `json:"current_proposal,omitempty"`
	Metrics         *PerformanceMetrics `json:"metrics,omitempty"`
}

// ProposalSummary represents a summary of an upgrade proposal
type ProposalSummary struct {
	ProposalID           string `json:"proposal_id"`
	TargetConsensus      string `json:"target_consensus"`
	CandidateStartHeight uint64 `json:"candidate_start_height"`
	SwitchHeight         uint64 `json:"switch_height"`
	Description          string `json:"description,omitempty"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	Throughput   float64 `json:"throughput"`
	AvgBlockTime float64 `json:"avg_block_time"`
	TxCount      uint64  `json:"tx_count"`
	BlockCount   uint64  `json:"block_count"`
	LastUpdated  int64   `json:"last_updated"`
}

// ProposalResponse represents a proposal creation response
type ProposalResponse struct {
	ProposalID string `json:"proposal_id"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"created_at"`
}

// CDLValidationResponse represents CDL validation response
type CDLValidationResponse struct {
	Valid   bool   `json:"valid"`
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// ConvertUpgradeStateToResponse converts UpgradeState to response format
func ConvertUpgradeStateToResponse(state *upgrade.UpgradeState, metrics *upgrade.PerformanceMetrics) *UpgradeStatusResponse {
	response := &UpgradeStatusResponse{
		Phase:     state.Phase.String(),
		Started:   state.Started,
		Completed: state.Completed,
		Failed:    state.Failed,
	}

	if state.Started {
		startTime := state.StartTime.Unix()
		response.StartTime = &startTime
	}

	if state.Completed || state.Failed {
		endTime := state.EndTime.Unix()
		response.EndTime = &endTime
	}

	if state.Failed {
		response.FailureReason = state.FailureReason
	}

	if state.CurrentProposal != nil {
		response.CurrentProposal = &ProposalSummary{
			ProposalID:           state.CurrentProposal.ProposalID.String(),
			TargetConsensus:      state.CurrentProposal.TargetConsensus,
			CandidateStartHeight: state.CurrentProposal.PreexecStartHeight,
			SwitchHeight:         state.CurrentProposal.SwitchHeight,
			Description:          state.CurrentProposal.Description,
		}
	}

	if metrics != nil {
		response.Metrics = &PerformanceMetrics{
			Throughput:   metrics.AvgThroughput,
			AvgBlockTime: metrics.AvgBlockTime,
			TxCount:      0, // Sum of TxCounts
			BlockCount:   metrics.TotalBlocks,
			LastUpdated:  time.Now().Unix(),
		}
		// Calculate total transactions
		for _, count := range metrics.TxCounts {
			response.Metrics.TxCount += uint64(count)
		}
	}

	return response
}
