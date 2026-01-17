package consensus

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

// UpgradeableConsensus implements Executor
func (uc *UpgradeableConsensus) VerifyTx(rtx types.RawTransaction) bool {
	return uc.executor.VerifyTx(rtx)
}

func (uc *UpgradeableConsensus) executeNormalTx(block types.ConsensusBlock, proof []byte, cid int64) {
	uc.log.WithFields(logrus.Fields{"cid": cid, "tx_count": len(block.GetTxs())}).Debug("Executing normal transactions")
	ntxs := [][]byte{}
	for _, rbtx := range block.GetTxs() {
		rtx := types.RawTransaction(rbtx)
		hash := rtx.Hash()
		tx, err := rtx.ToTx()
		if err != nil {
			uc.log.WithError(err).Warn("Failed to decode transaction")
			continue
		}
		if tx.Type == pb.TransactionType_NORMAL {
			uc.commitLock.Lock()
			if _, ok := uc.commitedTxs[hash]; !ok {
				ntxs = append(ntxs, rbtx)
			} else {
				uc.log.WithField("hash", hash).Trace("Transaction already committed, skipping")
			}
			uc.inputLock.Lock()
			delete(uc.inputBuffer, hash)
			uc.inputLock.Unlock()
			uc.commitLock.Unlock()
		}
	}
	uc.log.WithFields(logrus.Fields{"cid": cid, "new_tx_count": len(ntxs)}).Debug("Committing block to executor")
	uc.executor.CommitBlock(&pb.WhirlyBlock{Txs: ntxs}, GenProof(block, proof, uc.cid), uc.cid)
}

func (uc *UpgradeableConsensus) CommitBlock(block types.ConsensusBlock, proof []byte, cid int64) {
	// uc.log.WithField("cid", cid).Debug("commit block")
	if cid == uc.working.GetConsensusID() {
		go uc.executeNormalTx(block, proof, cid)

		for _, rbtx := range block.GetTxs() {
			rtx := types.RawTransaction(rbtx)
			tx, err := rtx.ToTx()
			if err != nil {
				uc.log.WithError(err).Warn("decode tx failed")
				continue
			}
			switch tx.Type {
			case pb.TransactionType_NORMAL:
				continue
			case pb.TransactionType_UPGRADE:
				uc.log.Info("Processing upgrade transaction")
				if !uc.executor.VerifyTx(rbtx) {
					uc.log.Warn("Upgrade transaction verification failed")
					return
				}
				cc := new(config.ConsensusConfig)
				err := json.Unmarshal(tx.Payload, cc)
				if err != nil {
					uc.log.WithError(err).Warn("Failed to unmarshal upgrade transaction payload")
					return
				}
				cc.Nodes = uc.config.Nodes
				cc.Keys = uc.config.Keys
				cc.F = uc.config.F
				uc.log.WithFields(logrus.Fields{
					"target_type": cc.Type,
					"target_cid":  cc.ConsensusID,
				}).Info("Upgrade consensus config received")
				uc.upgradeLock.Lock()
				txHash := rtx.Hash()
				uc.upgradeBuffer[txHash] = cc
				uc.upgradeLock.Unlock()
				uc.log.WithField("hash", txHash).Debug("Upgrade config buffered")

				if uc.config.Upgradeable.NetworkType == config.NetworkSync {
					uc.log.Debug("Sync mode: sending time vote")
					go uc.sendTimeVote(txHash)
				} else {
					uc.log.Debug("Async mode: starting new consensus immediately")
					go uc.startNewConsensus(cc)
				}

			case pb.TransactionType_TIMEVOTE:
				if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
					uc.log.Warn("Time vote not allowed in async network mode")
					continue
				}
				tv := new(TimeVote)
				if err := json.Unmarshal(tx.Payload, tv); err != nil {
					uc.log.WithError(err).Warn("Failed to unmarshal time vote")
					continue
				}
				if !tv.Verify(uc.config.Keys) {
					uc.log.Warn("Time vote verification failed")
					continue
				}
				tvi := tv.TVI
				uc.log.WithFields(logrus.Fields{"node_id": tvi.ID, "time": tvi.Time}).Debug("Processing time vote")
				uc.timeWeightLock.Lock()
				if _, ok := uc.timeWeight[tvi.Hash]; !ok {
					uc.timeWeight[tvi.Hash] = NewTimeWeightRecord()
				}
				r := uc.timeWeight[tvi.Hash]
				if _, ok := r.voted[tvi.ID]; ok {
					uc.log.WithField("node", tvi.ID).Warn("time vote multiple times")
					uc.timeWeightLock.Unlock()
					continue
				}
				r.voted[tvi.ID] = true
				r.weight[tvi.Time] += uc.GetWeight(tvi.ID)
				if r.weight[tvi.Time] <= uc.GetMaxAdversaryWeight() {
					uc.timeWeightLock.Unlock()
					continue
				}
				if r.starting {
					uc.timeWeightLock.Unlock()
					continue
				}
				r.starting = true
				uc.timeWeightLock.Unlock()
				uc.upgradeLock.Lock()
				cc, ok := uc.upgradeBuffer[tvi.Hash]
				uc.upgradeLock.Unlock()
				if !ok {
					uc.log.Warn("upgrade tx not found")
					continue
				}
				// uc.consensusLock.Lock()
				// if _, ok := uc.candidates[cc.ConsensusID]; ok { // already started
				// 	uc.consensusLock.Unlock()
				// 	continue
				// }
				// uc.consensusLock.Unlock()
				delay := tvi.Time + int64(uc.config.Upgradeable.CommitTime) - time.Now().Unix()
				uc.log.WithField("delay", delay).Debug("delay start consensus")
				if delay < 0 {
					uc.log.Warn("delay error")
					continue
				}
				go func() {
					time.Sleep(time.Duration(delay) * time.Second)
					uc.startNewConsensus(cc)
				}()

			case pb.TransactionType_LOCK:
				uc.log.Info("Processing lock transaction for consensus upgrade")
				lock := new(Lock)
				if err := json.Unmarshal(tx.Payload, lock); err != nil {
					uc.log.WithError(err).Warn("Failed to parse lock transaction")
					continue
				}
				uc.log.WithField("cid", lock.Cid).Debug("Lock transaction targets consensus")
				uc.consensusLock.Lock()
				nc, ok := uc.candidates[lock.Cid]
				if !ok {
					uc.log.WithField("cid", lock.Cid).Warn("Candidate consensus not found for lock")
					uc.consensusLock.Unlock()
					continue
				}
				uc.log.WithField("cid", lock.Cid).Debug("Verifying lock block")
				if !nc.VerifyBlock(lock.Block, proof) {
					uc.log.WithField("cid", lock.Cid).Warn("Lock block verification failed")
					uc.consensusLock.Unlock()
					continue
				}
				uc.log.WithField("cid", lock.Cid).Info("Lock verified, upgrading to candidate consensus")
				uc.upgradeConsensusTo(lock.Cid)
				uc.consensusLock.Unlock()
				// no further control tx should be executed
				return
			default:
				uc.log.WithField("type", tx.Type).Warn("transaction type unknown")
			}
		}
	} else {
		uc.log.WithField("cid", cid).Info("candidate consensus commit tx")
		uc.outputLock.Lock()
		if _, ok := uc.outputBuffer[cid]; !ok {
			//first commit
			uc.outputBuffer[cid] = []types.ConsensusBlock{}
			go uc.sendLockTx(block, proof, cid)
		}
		uc.outputBuffer[cid] = append(uc.outputBuffer[cid], block)
		uc.outputLock.Unlock()
	}
	// uc.log.WithField("cid", cid).Debug("commit block done")
}

func (uc *UpgradeableConsensus) sendTimeVote(h types.TxHash) {
	tvi := &TimeVoteInner{
		Hash: h,
		Time: time.Now().Unix(),
		ID:   uc.nid,
	}
	tv := tvi.Sign(uc.config.Keys)
	btv, err := json.Marshal(tv)
	utils.PanicOnError(err)
	brtx := types.BuildByteTx(pb.TransactionType_TIMEVOTE, btv)
	request := &pb.Request{
		Tx: brtx,
	}
	uc.consensusLock.RLock()
	defer uc.consensusLock.RUnlock()
	uc.working.GetRequestEntrance() <- request
}

func (uc *UpgradeableConsensus) sendLockTx(block types.ConsensusBlock, proof []byte, cid int64) {
	bblock, err := proto.Marshal(block)
	utils.PanicOnError(err)
	lock := &Lock{
		Block: bblock,
		Proof: proof,
		Cid:   cid,
	}
	byteLock, err := json.Marshal(lock)
	utils.PanicOnError(err)
	rbtx := types.BuildByteTx(pb.TransactionType_LOCK, byteLock)
	request := &pb.Request{Tx: rbtx}
	uc.consensusLock.RLock()
	uc.working.GetRequestEntrance() <- request
	uc.consensusLock.RUnlock()
}
