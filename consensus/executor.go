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
				uc.log.Info("=================================================")
				uc.log.Info("  Processing UPGRADE Transaction")
				uc.log.Info("=================================================")
				if !uc.executor.VerifyTx(rbtx) {
					uc.log.Warn("Upgrade transaction verification failed")
					return
				}

				// 尝试解析为 UpgradeConfigTransaction (新格式)
				upgradeConfig := new(pb.UpgradeConfigTransaction)
				if err := json.Unmarshal(tx.Payload, upgradeConfig); err == nil && upgradeConfig.TargetConsensus != "" {
					uc.log.WithFields(logrus.Fields{
						"target_consensus": upgradeConfig.TargetConsensus,
						"consensus_id":     upgradeConfig.ConsensusId,
						"fork_height":      upgradeConfig.ForkHeight,
						"switch_height":    upgradeConfig.SwitchHeight,
						"description":      upgradeConfig.Description,
					}).Info("Received UpgradeProposal (new format)")

					// 从提案构建共识配置
					cc := uc.buildConsensusConfigFromProposal(upgradeConfig)
					if cc == nil {
						uc.log.Warn("Failed to build consensus config from upgrade proposal")
						return
					}

					uc.log.WithFields(logrus.Fields{
						"target_type": cc.Type,
						"target_cid":  cc.ConsensusID,
					}).Info("Upgrade consensus config created from proposal")

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
				} else {
					// 尝试解析为 ConsensusConfig (旧格式)
					cc := new(config.ConsensusConfig)
					err := json.Unmarshal(tx.Payload, cc)
					if err != nil {
						uc.log.WithError(err).Warn("Failed to unmarshal upgrade transaction payload (both formats)")
						return
					}
					cc.Nodes = uc.config.Nodes
					cc.Keys = uc.config.Keys
					cc.Fault = uc.config.Fault
					uc.log.WithFields(logrus.Fields{
						"target_type": cc.Type,
						"target_cid":  cc.ConsensusID,
					}).Info("Received ConsensusConfig (legacy format)")
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
				uc.log.Info("Processing lock/confirm transaction")

				// 首先尝试解析为 UpgradeConfirmTransaction (来自 upgrade 模块)
				confirm := new(pb.UpgradeConfirmTransaction)
				if err := json.Unmarshal(tx.Payload, confirm); err == nil && len(confirm.ProposalId) > 0 {
					// 这是一个确认交易
					uc.handleConfirmTransaction(tx, rtx)
					continue
				}

				// 否则按原有的 Lock 交易处理
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

// buildConsensusConfigFromProposal 从升级提案构建共识配置
// 支持从 UpgradeConfigTransaction (来自 consensus/upgrade 模块) 构建 ConsensusConfig
func (uc *UpgradeableConsensus) buildConsensusConfigFromProposal(proposal *pb.UpgradeConfigTransaction) *config.ConsensusConfig {
	cc := &config.ConsensusConfig{
		Type:        proposal.TargetConsensus,
		ConsensusID: proposal.ConsensusId,
		Nodes:       uc.config.Nodes,
		Keys:        uc.config.Keys,
		Fault:       uc.config.Fault,
	}

	uc.log.WithFields(logrus.Fields{
		"target_consensus": proposal.TargetConsensus,
		"consensus_id":     proposal.ConsensusId,
	}).Info("Building consensus config from proposal")

	// 解析自定义共识参数
	var params map[string]interface{}
	if proposal.ConsensusParams != "" {
		if err := json.Unmarshal([]byte(proposal.ConsensusParams), &params); err != nil {
			uc.log.WithError(err).Debug("Failed to parse consensus params, using defaults")
		}
	}

	// 根据目标共识类型创建特定配置
	switch proposal.TargetConsensus {
	case "hotstuff":
		hsType := "basic"
		batchSize := 10
		batchTimeout := 1
		timeout := 2

		if params != nil {
			if t, ok := params["type"].(string); ok {
				hsType = t
			}
			if bs, ok := params["batch_size"].(float64); ok {
				batchSize = int(bs)
			}
			if bt, ok := params["batch_timeout"].(float64); ok {
				batchTimeout = int(bt)
			}
			if to, ok := params["timeout"].(float64); ok {
				timeout = int(to)
			}
		}

		cc.HotStuff = &config.HotStuffConfig{
			Type:         hsType,
			BatchSize:    batchSize,
			BatchTimeout: batchTimeout,
			Timeout:      timeout,
		}
		uc.log.WithFields(logrus.Fields{
			"hotstuff_type": hsType,
			"batch_size":    batchSize,
			"batch_timeout": batchTimeout,
			"timeout":       timeout,
		}).Info("Created HotStuff config")

	case "whirly":
		wType := "basic"
		batchSize := 10
		timeout := 2

		if params != nil {
			if t, ok := params["type"].(string); ok {
				wType = t
			}
			if bs, ok := params["batch_size"].(float64); ok {
				batchSize = int(bs)
			}
			if to, ok := params["timeout"].(float64); ok {
				timeout = int(to)
			}
		}

		cc.Whirly = &config.WhirlyConfig{
			Type:      wType,
			BatchSize: batchSize,
			Timeout:   timeout,
		}
		uc.log.WithFields(logrus.Fields{
			"whirly_type": wType,
			"batch_size":  batchSize,
			"timeout":     timeout,
		}).Info("Created Whirly config")

	case "pot":
		// PoT 共识配置
		snum := int64(2)
		sysPara := "123456789"
		vdf0Iter := 100000
		vdf1Iter := 80000

		cc.PoT = &config.PoTConfig{
			Snum:          snum,
			SysPara:       sysPara,
			Vdf0Iteration: vdf0Iter,
			Vdf1Iteration: vdf1Iter,
		}
		uc.log.WithFields(logrus.Fields{
			"snum":           snum,
			"vdf0_iteration": vdf0Iter,
		}).Info("Created PoT config")

	case "pow":
		// PoW 共识配置使用默认参数
		uc.log.Info("Created PoW config with defaults")

	default:
		uc.log.WithField("consensus_type", proposal.TargetConsensus).Warn("Unknown consensus type, using default hotstuff")
		cc.Type = "hotstuff"
		cc.HotStuff = &config.HotStuffConfig{
			Type:         "basic",
			BatchSize:    10,
			BatchTimeout: 1,
			Timeout:      2,
		}
	}

	return cc
}

// handleConfirmTransaction 处理确认交易（用于简化的投票流程）
func (uc *UpgradeableConsensus) handleConfirmTransaction(tx *pb.Transaction, rtx types.RawTransaction) {
	uc.log.Info("=================================================")
	uc.log.Info("  Processing CONFIRM Transaction")
	uc.log.Info("=================================================")

	// 解析确认交易
	confirm := new(pb.UpgradeConfirmTransaction)
	if err := json.Unmarshal(tx.Payload, confirm); err != nil {
		uc.log.WithError(err).Warn("Failed to parse confirm transaction")
		return
	}

	var proposalHash types.TxHash
	copy(proposalHash[:], confirm.ProposalId)

	uc.log.WithFields(logrus.Fields{
		"proposal_id":  proposalHash,
		"confirmer_id": confirm.ConfirmerId,
		"approved":     confirm.Approved,
	}).Info("Received upgrade confirmation")

	if !confirm.Approved {
		uc.log.Info("Upgrade rejected by confirmer")
		return
	}

	// 查找对应的升级配置
	uc.upgradeLock.Lock()
	cc, ok := uc.upgradeBuffer[proposalHash]
	uc.upgradeLock.Unlock()

	if !ok {
		uc.log.WithField("proposal_id", proposalHash).Warn("Upgrade proposal not found in buffer")
		return
	}

	uc.log.WithFields(logrus.Fields{
		"target_type": cc.Type,
		"target_cid":  cc.ConsensusID,
	}).Info("Upgrade confirmed, starting new consensus")

	// 启动新共识
	go uc.startNewConsensus(cc)
}
