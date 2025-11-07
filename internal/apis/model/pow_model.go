package model

// PowConsensusService defines the interface for PoW consensus operations
// This interface abstracts the consensus-specific operations needed by the API layer
type PowConsensusService interface {
	// TODO: Define PoW specific methods
	// For example:
	// SubmitWork(nonce uint64, hash []byte) error
	// GetDifficulty() uint64
	GetCurrentHeight() uint64
}
