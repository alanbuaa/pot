package apis

import (
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// PotWorkerAdapter adapts pot.Worker to implement model.PotConsensusService interface
type PotWorkerAdapter struct {
	worker *pot.Worker
}

// NewPotWorkerAdapter creates a new adapter for pot.Worker
func NewPotWorkerAdapter(worker *pot.Worker) model.PotConsensusService {
	return &PotWorkerAdapter{
		worker: worker,
	}
}

// CheckLockTransaction validates a lock transaction
func (a *PotWorkerAdapter) CheckLockTransaction(rawtx *types.RawTx) error {
	return a.worker.CheckLockTransaction(rawtx)
}

// CheckLockTransferTransaction validates a lock transfer transaction
func (a *PotWorkerAdapter) CheckLockTransferTransaction(rawtx *types.RawTx) error {
	return a.worker.CheckLockTransferTransaction(rawtx)
}

// CheckNonLockTransferTransaction validates a non-lock transfer transaction
func (a *PotWorkerAdapter) CheckNonLockTransferTransaction(rawtx *types.RawTx) error {
	return a.worker.CheckNonLockTransferTransaction(rawtx)
}

// CheckDevastateTransaction validates a devastate transaction
func (a *PotWorkerAdapter) CheckDevastateTransaction(rawtx *types.RawTx) error {
	return a.worker.CheckDevastateTransaction(rawtx)
}

// BroadcastClientTransaction broadcasts a transaction to the network
func (a *PotWorkerAdapter) BroadcastClientTransaction(rawtx *types.RawTx, txType pb.TxType) error {
	return a.worker.BroadcastClientTransaction(rawtx, txType)
}

// AddRawTxToMempool adds a transaction to the mempool
func (a *PotWorkerAdapter) AddRawTxToMempool(rawtx *types.RawTx) {
	a.worker.GetMempool().AddRawTx(rawtx)
}

// GetCurrentHeight returns the current blockchain height
func (a *PotWorkerAdapter) GetCurrentHeight() uint64 {
	return a.worker.GetChainReader().GetCurrentHeight()
}
