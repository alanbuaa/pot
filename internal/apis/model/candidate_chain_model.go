package model

// Candidate chain management models

// StartCandidateChainRequest represents request to start a candidate chain
type StartCandidateChainRequest struct {
	ProposalID  string `json:"proposal_id" binding:"required"`
	CandidateID string `json:"candidate_id" binding:"required"`
}

// GetCandidateStateRequest represents request to get candidate chain state
type GetCandidateStateRequest struct {
	CandidateID string `json:"candidate_id" binding:"required"`
}

// MergeCandidateChainRequest represents request to merge candidate chain to main chain
type MergeCandidateChainRequest struct {
	CandidateID string `json:"candidate_id" binding:"required"`
}

// RollbackCandidateChainRequest represents request to rollback a candidate chain
type RollbackCandidateChainRequest struct {
	CandidateID string `json:"candidate_id" binding:"required"`
	Reason      string `json:"reason"`
}

// CandidateChainResponse represents candidate chain state response
type CandidateChainResponse struct {
	CandidateID   string `json:"candidate_id"`
	ProposalID    string `json:"proposal_id"`
	ForkHeight    uint64 `json:"fork_height"`
	CurrentHeight uint64 `json:"current_height"`
	Status        string `json:"status"` // running, merged, rolled_back
	BlockCount    uint64 `json:"block_count"`
}

// ListCandidateChainsResponse represents list of all candidate chains
type ListCandidateChainsResponse struct {
	Chains []CandidateChainResponse `json:"chains"`
	Count  int                      `json:"count"`
}
