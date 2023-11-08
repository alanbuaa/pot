package consensus

import (
	"encoding/json"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

// UpgradeableConsensus implements Executor
func (uc *UpgradeableConsensus) VerifyTx(rtx types.RawTransaction) bool {
	return uc.executor.VerifyTx(rtx)
}

func (uc *UpgradeableConsensus) executeNormalTx(block types.Block, proof []byte, cid int64) {
	ntxs := [][]byte{}
	for _, rbtx := range block.GetTxs() {
		rtx := types.RawTransaction(rbtx)
		hash := rtx.Hash()
		tx, err := rtx.ToTx()
		if err != nil {
			uc.log.WithError(err).Warn("decode tx failed")
			continue
		}
		if tx.Type == pb.TransactionType_NORMAL {
			uc.commitLock.Lock()
			if _, ok := uc.commitedTxs[hash]; !ok {
				ntxs = append(ntxs, rbtx)
			} else {
				uc.log.WithField("hash", hash).Debug("tx already commited")
			}
			uc.inputLock.Lock()
			delete(uc.inputBuffer, hash)
			uc.inputLock.Unlock()
			uc.commitLock.Unlock()
		}
	}
	uc.executor.CommitBlock(&pb.ExecBlock{Txs: ntxs}, GenProof(block, proof, uc.cid), uc.cid)
}

func (uc *UpgradeableConsensus) CommitBlock(block types.Block, proof []byte, cid int64) {
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
				if !uc.executor.VerifyTx(rbtx) {
					uc.log.Warn("upgrade tx not verified")
					return
				}
				cc := new(config.ConsensusConfig)
				err := json.Unmarshal(tx.Payload, cc)
				if err != nil {
					uc.log.Warn("unmarshal upgrade tx failed")
					return
				}
				cc.Nodes = uc.config.Nodes
				cc.Keys = uc.config.Keys
				cc.F = uc.config.F
				uc.log.WithField("cc", cc).Info("get consensus config")
				uc.upgradeLock.Lock()
				txHash := rtx.Hash()
				uc.upgradeBuffer[txHash] = cc
				uc.upgradeLock.Unlock()

				if uc.config.Upgradeable.NetworkType == config.NetworkSync {
					go uc.sendTimeVote(txHash)
				} else {
					go uc.startNewConsensus(cc)
				}

			case pb.TransactionType_TIMEVOTE:
				if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
					uc.log.Warn("time vote not allowed in async network")
					continue
				}
				tv := new(TimeVote)
				if err := json.Unmarshal(tx.Payload, tv); err != nil {
					uc.log.Warn("unmarshal time vote failed")
					continue
				}
				if !tv.Verify(uc.config.Keys) {
					uc.log.Warn("time vote verify failed")
					continue
				}
				tvi := tv.TVI
				uc.log.WithField("time", tvi.Time).Debug("commit time vote")
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
				lock := new(Lock)
				if err := json.Unmarshal(tx.Payload, lock); err != nil {
					uc.log.WithError(err).Warn("parse lock failed")
					continue
				}
				uc.consensusLock.Lock()
				nc, ok := uc.candidates[lock.Cid]
				if !ok {
					uc.log.WithField("cid", lock.Cid).Warn("candidate not found")
					uc.consensusLock.Unlock()
					continue
				}
				if !nc.VerifyBlock(lock.Block, proof) {
					uc.log.WithField("cid", lock.Cid).Warn("block validate failed")
					uc.consensusLock.Unlock()
					continue
				}
				uc.upgradeConsensusTo(lock.Cid)
				uc.consensusLock.Unlock()
				// no further control tx should be executed
				// uc.log.WithField("cid", cid).Debug("commit block done 2")
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
			uc.outputBuffer[cid] = []types.Block{}
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

func (uc *UpgradeableConsensus) sendLockTx(block types.Block, proof []byte, cid int64) {
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
