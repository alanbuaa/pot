package model

import (
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// PotConsensusService defines the interface for PoT consensus operations
// This interface abstracts the consensus-specific operations needed by the API layer
type PotConsensusService interface {
	// Transaction validation methods
	CheckLockTransaction(rawtx *types.RawTx) error
	CheckLockTransferTransaction(rawtx *types.RawTx) error
	CheckNonLockTransferTransaction(rawtx *types.RawTx) error
	CheckDevastateTransaction(rawtx *types.RawTx) error

	// Transaction broadcast methods
	BroadcastClientTransaction(rawtx *types.RawTx, txType pb.TxType) error

	// Mempool operations
	AddRawTxToMempool(rawtx *types.RawTx)

	// Chain query methods
	GetCurrentHeight() uint64
}
