package pot

import (
	mrpvss "blockchain-crypto/share/mrpvss/bls12381"
	"blockchain-crypto/shuffle"
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/verifiable_draw"
	"container/list"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

var (
	BigN   = uint64(64)
	SmallN = uint64(4)
)
var (
	group1 = bls12381.NewG1()
	// test
	g1Degree = uint32(1 << 8)
	// test
	g2Degree = uint32(1 << 8)
)

type Sharding struct {
	Name            string
	Id              int32
	LeaderAddress   string
	Committee       []string
	consensusconfig config.ConsensusConfig
}

// CommitteeMark 记录委员会相关信息
type CommitteeMark struct {
	WorkHeight  uint64              // 委员会工作的PoT区块高度
	PKList      []*bls12381.PointG1 // 份额持有者的公钥列表，用于PVSS分发份额时查找委员会成员
	CommitteePK *bls12381.PointG1   // 委员会聚合公钥 y = g^s
}

// SelfCommitteeMark 记录自己所在的委员会相关信息
type SelfCommitteeMark struct {
	WorkHeight   uint64            // 委员会工作的PoT区块高度
	SelfPK       *bls12381.PointG1 // 自己的公钥,用于查找自己的份额
	AggrEncShare *mrpvss.EncShare  // 聚合加密份额
	Share        *bls12381.Fr      // 解密份额（聚合的）
}

//func (w *Worker) simpleLeaderUpdate(parent *types.Header) {
//	if parent != nil {
//		// address := parent.Address
//		address := parent.Address
//		if !w.committeeCheck(address, parent) {
//			return
//		}
//		if w.committeeSizeCheck() && w.whirly == nil {
//
//			whirlyConfig := &config.ConsensusConfig{
//				Type:        "whirly",
//				ConsensusID: 1009,
//				Whirly: &config.WhirlyConfig{
//					Type:      "simple",
//					BatchSize: 10,
//					Timeout:   2000,
//				},
//				Nodes: w.config.Nodes,
//				Keys:  w.config.Keys,
//				F:     w.config.F,
//			}
//			s := simpleWhirly.NewSimpleWhirly(w.ID, 1009, whirlyConfig, w.Engine.exec, w.Engine.Adaptor, w.log, "", nil)
//			w.whirly = s
//			//w.Engine.SetWhirly(s)
//			// w.potSignalChan = w.whirly.GetPoTByteEntrance()
//			w.log.Errorf("[PoT]\t Start committee consensus at epoch %d", parent.ExecHeight+1)
//			return
//		}
//		potSignal := &simpleWhirly.PoTSignal{
//			Epoch:               int64(parent.ExecHeight),
//			Proof:               parent.PoTProof[0],
//			ID:                  parent.Address,
//			LeaderPublicAddress: parent.PeerId,
//		}
//		b, err := json.Marshal(potSignal)
//		if err != nil {
//			w.log.WithError(err)
//			return
//		}
//		if w.potSignalChan != nil {
//			w.potSignalChan <- b
//		}
//	}
//}

//func (w *Worker) committeeCheck(id int64, header *types.Header) bool {
//	if _, exist := w.committee.Get(id); !exist {
//		w.committee.Set(id, header)
//		return false
//	}
//	return true
//}
//
//func (w *Worker) committeeSizeCheck() bool {
//	return w.committee.Len() == 4
//}

func (w *Worker) GetPeerQueue() chan *types.Block {
	return w.peerMsgQueue
}

// SendInitialPoTSignal sends an initial PoT signal to create the genesis sharding
// This ensures the NodeController has a sharding available before client requests arrive
func (w *Worker) SendInitialPoTSignal() {
	w.log.Info("Generating initial PoT signal for genesis sharding")

	// Create initial committee from config nodes
	// Use first Commiteelen nodes as the genesis committee
	committee := make([]string, 0, Commiteelen)
	for nodeID := range w.config.Nodes {
		if len(committee) >= int(Commiteelen) {
			break
		}
		// Use node ID as placeholder public key for genesis committee
		// In real deployment, this should be the actual node public key
		committee = append(committee, hexutil.EncodeUint64(uint64(nodeID)))
	}

	w.log.WithFields(logrus.Fields{
		"committee_size": len(committee),
		"leader":         committee[0],
	}).Debug("Genesis committee created")

	// Create consensus config for the genesis sharding
	whilyConsensus := &config.WhirlyConfig{
		Type:      "simple",
		BatchSize: 2,
		Timeout:   2,
	}

	consensus := config.ConsensusConfig{
		Type:        "whirly",
		ConsensusID: 1201,
		Whirly:      whilyConsensus,
		Nodes:       w.config.Nodes,
		Topic:       w.config.Topic,
		Fault:       w.config.Fault,
	}

	// Create genesis sharding with name matching client requests
	sharding := nodeController.PoTSharding{
		Name:                hexutil.EncodeUint64(1),
		ParentSharding:      nil,
		LeaderPublicAddress: committee[0],
		Committee:           committee,
		SubConsensus:        consensus,
	}

	shardings := []nodeController.PoTSharding{sharding}

	// For genesis, include all committee members as self addresses
	// This allows all nodes to participate in consensus from the start
	selfPublicAddress := make([]string, len(committee))
	copy(selfPublicAddress, committee)

	// Create PoT signal for epoch 1 (SimpleWhirly starts from epoch 1, not 0)
	potsignal := &nodeController.PoTSignal{
		Epoch:             1,
		Proof:             make([]byte, 0),
		ID:                0,
		SelfPublicAddress: selfPublicAddress, // Include all committee members for genesis
		Shardings:         shardings,
	}

	b, err := json.Marshal(potsignal)
	if err != nil {
		w.log.WithError(err).Error("Failed to marshal initial PoT signal")
		return
	}

	if w.potSignalChan != nil {
		w.log.WithFields(logrus.Fields{
			"epoch":          1,
			"sharding_name":  sharding.Name,
			"committee_size": len(committee),
		}).Info("Sending initial PoT signal for genesis sharding")
		w.potSignalChan <- b
	} else {
		w.log.Warn("PoT signal channel not initialized, skipping initial signal")
	}
}

func (w *Worker) CommitteeUpdate(height uint64) {

	if height >= CommiteeDelay+Commiteelen {
		w.log.WithFields(logrus.Fields{
			"height":        height,
			"delay":         CommiteeDelay,
			"committee_len": Commiteelen,
		}).Debug("Updating committee membership")
		committee := make([]string, Commiteelen)
		selfaddress := make([]string, 0)
		for i := uint64(0); i < Commiteelen; i++ {
			block, err := w.chainReader.GetByHeight(height - CommiteeDelay - i)
			if err != nil {
				w.log.WithFields(logrus.Fields{
					"height":        height,
					"target_height": height - CommiteeDelay - i,
				}).WithError(err).Warn("Failed to retrieve block for committee update")
				return
			}
			if block != nil {
				header := block.GetHeader()
				committee[i] = hexutil.Encode(header.CommiteePubkey)
				flag, _ := w.TryFindCommiteeKey(crypto.Convert(header.Hash()))
				if flag {
					w.log.WithFields(logrus.Fields{
						"height":     height,
						"position":   i,
						"public_key": utils.EncodeShortPrint(header.CommiteePubkey),
					}).Debug("Found own committee key in block")
					selfaddress = append(selfaddress, hexutil.Encode(header.CommiteePubkey))
				}
			}
		}
		// potsignal := &simpleWhirly.PoTSignal{
		// 	Epoch:               int64(epoch),
		// 	Proof:               nil,
		// 	ID:                  0,
		// 	LeaderPublicAddress: committee[0],
		// 	Committee:           committee,
		// 	SelfPublicAddress:   selfaddress,
		// 	CryptoElements:      nil,
		// }
		whilyConsensus := &config.WhirlyConfig{
			Type:      "simple",
			BatchSize: 2,
			Timeout:   2,
		}

		consensus := config.ConsensusConfig{
			Type:        "whirly",
			ConsensusID: 1201,
			Whirly:      whilyConsensus,
			Nodes:       w.config.Nodes,
			Topic:       w.config.Topic,
			Fault:       w.config.Fault,
		}

		sharding1 := nodeController.PoTSharding{
			Name:                hexutil.EncodeUint64(1),
			ParentSharding:      nil,
			LeaderPublicAddress: committee[0],
			Committee:           committee,
			// CryptoElements:      blockchain_api.CommitteeConfig{},
			SubConsensus: consensus,
		}

		//w.log.Error(len(committee))

		//sharding2 := simpleWhirly.PoTSharding{
		//	Name:                "hello_world",
		//	ParentSharding:      nil,
		//	LeaderPublicAddress: committee[0],
		//	Committee:           committee,
		//	CryptoElements:      nil,
		//	SubConsensus:        consensus,
		//}
		//shardings := []simpleWhirly.PoTSharding{sharding1, sharding2}

		shardings := []nodeController.PoTSharding{sharding1}
		potsignal := &nodeController.PoTSignal{
			Epoch:             int64(height),
			Proof:             make([]byte, 0),
			ID:                0,
			SelfPublicAddress: selfaddress,
			Shardings:         shardings,
		}
		b, err := json.Marshal(potsignal)
		if err != nil {
			w.log.WithFields(logrus.Fields{
				"height":    height,
				"epoch":     potsignal.Epoch,
				"shardings": len(shardings),
			}).WithError(err).Error("Failed to marshal PoT signal for committee update")
			return
		}
		if w.potSignalChan != nil {
			w.log.WithFields(logrus.Fields{
				"height":             height,
				"epoch":              int64(height),
				"committee_size":     len(committee),
				"self_address_count": len(selfaddress),
				"shardings":          len(shardings),
			}).Debug("Sending PoT signal for committee update")
			w.potSignalChan <- b
		}
	}
	epoch := height
	if epoch > 1 && w.ID == 0 {
		w.log.WithFields(logrus.Fields{
			"epoch":     epoch,
			"worker_id": w.ID,
		}).Trace("Processing epoch block for BCI recording")
		block, err := w.chainReader.GetByHeight(epoch - 1)
		if err != nil {
			w.log.WithFields(logrus.Fields{
				"epoch":        epoch,
				"block_height": epoch - 1,
			}).WithError(err).Warn("Failed to retrieve block for BCI recording")
			return
		}
		header := block.GetHeader()

		txs := block.GetRawTx()
		coinbasetx := txs[0]
		if coinbasetx.IsCoinBase() {
			lines := []string{
				fmt.Sprintf("[%d]%s", epoch, hexutil.Encode(header.Hash())),
				fmt.Sprintf("[%d]%s", epoch, hexutil.Encode(coinbasetx.Txid[:])),
				fmt.Sprintf("[%d]%s", epoch, hexutil.Encode(coinbasetx.TxOutput[0].Address)),
				fmt.Sprintf("[%d]%s", epoch, hexutil.EncodeBig(big.NewInt(coinbasetx.TxOutput[0].Value))),
				"",
			}
			if err := utils.AppendLinesToFile("bci", lines); err != nil {
				w.log.WithFields(logrus.Fields{
					"epoch":      epoch,
					"block_hash": utils.EncodeShortPrint(header.Hash()),
					"tx_id":      utils.EncodeShortPrint(coinbasetx.Txid[:]),
					"file":       "bci",
				}).WithError(err).Error("Failed to write coinbase transaction to BCI file")
			} else {
				w.log.WithFields(logrus.Fields{
					"epoch":      epoch,
					"block_hash": utils.EncodeShortPrint(header.Hash()),
					"value":      coinbasetx.TxOutput[0].Value,
				}).Debug("Successfully recorded coinbase transaction to BCI file")
			}
		}
	}
}

type queue = list.List

type CryptoSet struct {
	// 通用
	BigN   uint32            // 候选公钥列表大小
	SmallN uint64            // 委员会大小
	SK     *bls12381.Fr      // 该节点的私钥
	G      *bls12381.PointG1 // G1生成元
	H      *bls12381.PointG1 // G1生成元
	// 参数生成阶段
	LocalSRS *srs.SRS // 本地记录的最新有效SRS
	// 置换阶段
	PrevShuffledPKList    []*bls12381.PointG1 // 前一有效置换后的公钥列表
	PrevRCommitForShuffle *bls12381.PointG1   // 前一有效置换随机数承诺
	// 抽签阶段
	PrevRCommitForDraw          *bls12381.PointG1 // 前一有效抽签随机数承诺
	UnenabledCommitteeQueue     *queue            // 抽签产生的委员会队列（未启用的）
	UnenabledSelfCommitteeQueue *queue            // 自己所在的委员会队列（未启用的）
	// DPVSS阶段
	Threshold uint32 // DPVSS恢复门限
}

func inInitStage(height uint64) bool {
	return height <= CandidateKeyLen
}
func inShuffleStage(height uint64) bool {
	// 置换阶段（简单置换） [1+kN, Commitees+kN] k>0
	if height <= CandidateKeyLen {
		return false
	}
	relativeHeight := height % CandidateKeyLen
	if relativeHeight > 0 && relativeHeight <= Commitees {
		return true
	}
	return false
}
func isTheLastShuffle(height uint64) bool {
	// 置换阶段（简单置换） [1+kN, Commitees+kN] k>0
	if height <= CandidateKeyLen {
		return false
	}
	if height%CandidateKeyLen == Commitees {
		return true
	}
	return false
}

func inDrawStage(height uint64) bool {
	return height > CandidateKeyLen+Commitees
}
func inDPVSSStage(height uint64) bool {
	return height > CandidateKeyLen+Commitees+1
}
func inWorkStage(height uint64) bool {
	return height > BigN+2*SmallN
}

// // uponReceivedBlock 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
// func (w *Worker) uponReceivedBlock(
// 	height uint64, block *types.Block,
// ) bool {
// 	// 如果处于初始化阶段（参数生成阶段）
// 	receivedBlock := block.GetHeader().CryptoElement
// 	if inInitStage(height) {
// 		// 检查srs的更新证明，如果验证失败，则丢弃
// 		if !srs.Verify(receivedBlock.SRS, w.Cryptoset.LocalSRS.G1PowerOf(1), receivedBlock.SrsUpdateProof) {
// 			return false
// 		}
// 		return true
// 	}
// 	// 如果处于置换阶段，则验证置换
// 	if inShuffleStage(height) {
// 		// 验证置换
// 		// 如果验证失败，丢弃
// 		if !shuffle.Verify(w.Cryptoset.LocalSRS, w.Cryptoset.PrevShuffledPKList, w.Cryptoset.PrevRCommitForShuffle, receivedBlock.ShuffleProof) {
// 			return false
// 		}
// 	}
// 	// 如果处于抽签阶段，则验证抽签
// 	if inDrawStage(height) {
// 		// 验证抽签
// 		// 如果验证失败，丢弃
// 		if !verifiable_draw.Verify(w.Cryptoset.LocalSRS, uint32(BigN), w.Cryptoset.PrevShuffledPKList, uint32(w.Cryptoset.SmallN), w.Cryptoset.PrevRCommitForDraw, receivedBlock.DrawProof) {
// 			return false
// 		}
// 	}
// 	// 如果处于DPVSS阶段
// 	if inDPVSSStage(height) {
// 		// 验证PVSS加密份额
// 		pvssTimes := len(receivedBlock.CommitteeWorkHeightList)
// 		for i := 0; i < pvssTimes; i++ {
// 			// 如果验证失败，丢弃
// 			if !mrpvss.VerifyEncShares(uint32(SmallN), w.Cryptoset.Threshold, w.Cryptoset.G, w.Cryptoset.H, receivedBlock.HolderPKLists[i], receivedBlock.ShareCommitLists[i], receivedBlock.CoeffCommitLists[i], receivedBlock.EncShareLists[i]) {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// func (w *Worker) UpdateLocalCryptoSetByBlock(height uint64, receivedBlock *types.Block) {
// 	if inInitStage(height) {
// 		// 记录最新srs
// 		w.Cryptoset.LocalSRS = receivedBlock.GetHeader().CryptoElement.SRS
// 	}
// 	// 置换阶段
// 	if inShuffleStage(height) {
// 		// 记录该置换公钥列表为最新置换公钥列表
// 		w.Cryptoset.PrevShuffledPKList = receivedBlock.GetHeader().CryptoElement.ShuffleProof.SelectedPubKeys
// 		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
// 		w.Cryptoset.PrevRCommitForShuffle.Set(receivedBlock.GetHeader().CryptoElement.ShuffleProof.RCommit)
// 		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
// 		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
// 		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
// 		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
// 		if isTheLastShuffle(height) {
// 			w.Cryptoset.PrevRCommitForDraw.Set(w.Cryptoset.PrevRCommitForShuffle)
// 		}
// 	}
// 	// 抽签阶段
// 	if inDrawStage(height) {
// 		// 记录抽出来的委员会（公钥列表）
// 		w.Cryptoset.UnenabledCommitteeQueue.PushBack(CommitteeMark{
// 			WorkHeight:  height + SmallN,
// 			PKList:      receivedBlock.GetHeader().CryptoElement.DrawProof.SelectedPubKeys,
// 			CommitteePK: group1.Zero(),
// 		})
// 		// 获取抽签结果
// 		isSelected, pk := verifiable_draw.IsSelected(w.Cryptoset.SK, receivedBlock.GetHeader().CryptoElement.DrawProof.RCommit, receivedBlock.GetHeader().CryptoElement.DrawProof.SelectedPubKeys)
// 		// 如果抽中，在自己所在的委员会队列创建新条目记录
// 		if isSelected {
// 			w.Cryptoset.UnenabledSelfCommitteeQueue.PushBack(SelfCommitteeMark{
// 				WorkHeight:   height + SmallN,
// 				SelfPK:       pk,
// 				AggrEncShare: mrpvss.NewEmptyEncShare(),
// 				Share:        bls12381.NewFr().Zero(),
// 			})
// 		}
// 	}
// 	// DPVSS阶段
// 	if inDPVSSStage(height) {
// 		pvssTimes := len(receivedBlock.GetHeader().CryptoElement.CommitteeWorkHeightList)
// 		for i := 0; i < pvssTimes; i++ {
// 			// 聚合对应委员会的公钥
// 			for e := w.Cryptoset.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
// 				if e.Value.(*CommitteeMark).WorkHeight == receivedBlock.GetHeader().CryptoElement.CommitteeWorkHeightList[i] {
// 					e.Value.(*CommitteeMark).CommitteePK = mrpvss.AggregateLeaderPK(e.Value.(*CommitteeMark).CommitteePK, receivedBlock.GetHeader().CryptoElement.CommitteePKList[i])
// 				}
// 			}
// 			for e := w.Cryptoset.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
// 				// 如果自己是对应份额持有者(委员会工作高度相等)，则获取对应加密份额，聚合加密份额
// 				if e.Value.(SelfCommitteeMark).WorkHeight == receivedBlock.GetHeader().CryptoElement.CommitteeWorkHeightList[i] {
// 					holderPKList := receivedBlock.GetHeader().CryptoElement.HolderPKLists[i]
// 					encShares := receivedBlock.GetHeader().CryptoElement.EncShareLists[i]
// 					holderNum := len(holderPKList)
// 					for j := 0; j < holderNum; j++ {
// 						if group1.Equal(e.Value.(*SelfCommitteeMark).SelfPK, holderPKList[j]) {
// 							// 聚合加密份额
// 							mrpvss.AggregateEncShares(e.Value.(*SelfCommitteeMark).AggrEncShare, encShares[j])
// 							e.Value.(*SelfCommitteeMark).AggrEncShare = encShares[j]
// 						}
// 					}
// 					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
// 					if int64(e.Value.(*SelfCommitteeMark).WorkHeight)-int64(height) == 1 {
// 						// 解密聚合加密份额
// 						e.Value.(*SelfCommitteeMark).Share = mrpvss.DecryptAggregateShare(w.Cryptoset.G, w.Cryptoset.SK, e.Value.(*SelfCommitteeMark).AggrEncShare, uint32(SmallN-1))
// 					}
// 				}
// 			}

// 		}
// 	}
// 	// 委员会更新阶段
// 	if inWorkStage(height) {
// 		// 更新未工作的委员会队列，删除第一个元素
// 		if w.Cryptoset.UnenabledCommitteeQueue.Front() != nil {
// 			w.Cryptoset.UnenabledCommitteeQueue.Remove(w.Cryptoset.UnenabledCommitteeQueue.Front())
// 		}
// 		// 检查自己是否需要作为委员会成员工作
// 		// 遍历队列查找，如果自己该作为委员会成员执行业务处理，则解密聚合加密份额，并发送信号
// 		for e := w.Cryptoset.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
// 			if height == e.Value.(*SelfCommitteeMark).WorkHeight {
// 				// 从自己所属委员会队列中移除该委员会标记
// 				// TODO 移除前接收数据（如果需要）
// 				w.Cryptoset.UnenabledSelfCommitteeQueue.Remove(e)
// 				// TODO 发送相关信号
// 			}
// 		}
// 	}
// }

func (w *Worker) SetWhirly(impl *nodeController.NodeController) {
	w.whirly = impl
	w.potSignalChan = impl.GetPoTByteEntrance()
}

//func (w *Worker) CommiteeLenCheck() bool {
//	if len(w.Commitee) != Commiteelen {
//		return false
//	}
//	return true
//}

//func (w *Worker) GetCommiteeLeader() string {
//	return w.Commitee[Commiteelen-1]
//}

//func (w *Worker) UpdateCommitee(commiteeid int, peerid string) []string {
//	if w.CommiteeLenCheck() {
//		w.Commitee[commiteeid] = w.Commitee[commiteeid][1:Commiteelen]
//		w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
//	}
//	if w.CommiteeLenCheck() {
//		return w.Commitee[commiteeid]
//	}
//	return nil
//}
//
//func (w *Worker) AppendCommitee(commiteeid int, peerid string) {
//	w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
//}

func (w *Worker) GetBackupCommitee(epoch uint64) ([]string, []string) {
	commitee := make([]string, Commiteelen)
	selfAddr := make([]string, 0)
	if epoch > BackupCommiteeSize && epoch%BackupCommiteeSize == 0 {
		for i := uint64(1); i <= BackupCommiteeSize; i++ {
			getepoch := epoch - 64 + i
			block, err := w.chainReader.GetByHeight(getepoch)
			if err != nil {
				return nil, nil
			}
			if block != nil {
				header := block.GetHeader()
				commiteekey := header.PublicKey
				commitee = append(commitee, hexutil.Encode(commiteekey))
				flag, _ := w.TryFindCommiteeKey(crypto.Convert(header.Hash()))
				if flag {
					selfAddr = append(selfAddr, hexutil.Encode(header.PublicKey))
				}
			}
		}
		w.BackupCommitee = commitee
		w.SelfAddress = selfAddr
	} else {
		commitee = w.BackupCommitee
		selfAddr = w.SelfAddress
	}
	return commitee, selfAddr
}

// TODO: Shuffle function need to complete
func (w *Worker) ShuffleCommitee(epoch uint64, backupcommitee []string, comitteenum int) [][]string {
	commitees := make([][]string, comitteenum)
	for i := 0; i < comitteenum; i++ {
		if len(backupcommitee) > 4 {
			commitees[i] = backupcommitee[:4]
		} else {
			commitees[i] = backupcommitee
		}
	}
	return commitees
}

func IsContain(parent []string, son string) bool {
	for _, s := range parent {
		if s == son {
			return true
		}
	}
	return false
}

func (w *Worker) GenerateCryptoSetFromLocal(height uint64) (crypto.CryptoElement, error) {
	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height) {
		// 更新 srs, 并生成证明
		r, _ := bls12381.NewFr().Rand(rand.Reader)
		newSRS, newSrsUpdateProof := w.Cryptoset.LocalSRS.Update(r)
		// 更新后的SRS和更新证明写入区块
		return crypto.CryptoElement{
			SRS:            newSRS,
			SrsUpdateProof: newSrsUpdateProof,
		}, nil
	}
	var err error
	var newShuffleProof *verifiable_draw.DrawProof = nil
	var newDrawProof *verifiable_draw.DrawProof = nil
	var committeeWorkHeightList []uint64 = nil
	var holderPKLists [][]*bls12381.PointG1 = nil
	var shareCommitsList [][]*bls12381.PointG1 = nil
	var coeffCommitsList [][]*bls12381.PointG1 = nil
	var encSharesList [][]*mrpvss.EncShare = nil
	var committeePKList []*bls12381.PointG1 = nil
	// 如果处于置换阶段，则进行置换
	if inShuffleStage(height) {
		newShuffleProof, _ = shuffle.SimpleShuffle(w.Cryptoset.LocalSRS, w.Cryptoset.PrevShuffledPKList, w.Cryptoset.PrevRCommitForShuffle)
	}
	// 如果处于抽签阶段，则进行抽签
	if inDrawStage(height) {
		// TODO 生成秘密向量用于抽签，向量元素范围为 [1,BigN]
		secretVector := []uint32{9, 12, 5, 29}
		newDrawProof, err = verifiable_draw.Draw(w.Cryptoset.LocalSRS, w.Cryptoset.BigN, w.Cryptoset.PrevShuffledPKList, uint32(SmallN), secretVector, w.Cryptoset.PrevRCommitForDraw)
		// TODO handle error，能否直接return？还是循环调用？
		if err != nil {
			return crypto.CryptoElement{}, err
		}
	}
	// 如果处于PVSS阶段，则进行PVSS的分发份额
	if inDPVSSStage(height) {
		// 计算向哪些委员会进行pvss(委员会index从1开始)
		pvssTimes := height - BigN - SmallN - 1
		if pvssTimes > w.Cryptoset.SmallN-1 {
			pvssTimes = w.Cryptoset.SmallN - 1
		}
		// 要PVSS的委员会的工作高度
		committeeWorkHeightList = make([]uint64, pvssTimes)
		holderPKLists = make([][]*bls12381.PointG1, pvssTimes)
		shareCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		coeffCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		encSharesList = make([][]*mrpvss.EncShare, pvssTimes)
		committeePKList = make([]*bls12381.PointG1, pvssTimes)
		i := uint64(0)
		for e := w.Cryptoset.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
			// 委员会的工作高度 height - pvssTimes + i + SmallN
			committeeWorkHeightList[i] = height - pvssTimes + i + SmallN
			secret, _ := bls12381.NewFr().Rand(rand.Reader)
			holderPKLists[i] = e.Value.(*CommitteeMark).PKList
			shareCommitsList[i], coeffCommitsList[i], encSharesList[i], committeePKList[i], err = mrpvss.EncShares(w.Cryptoset.G, w.Cryptoset.H, holderPKLists[i], secret, uint32(uint64(w.Cryptoset.Threshold)))
			// TODO handle error，能否直接return？还是循环调用？
			if err != nil {
				return crypto.CryptoElement{}, err
			}
			i++
		}
	}
	return crypto.CryptoElement{
		SRS:                     nil,
		SrsUpdateProof:          nil,
		ShuffleProof:            newShuffleProof,
		DrawProof:               newDrawProof,
		CommitteeWorkHeightList: committeeWorkHeightList,
		HolderPKLists:           holderPKLists,
		ShareCommitLists:        shareCommitsList,
		CoeffCommitLists:        coeffCommitsList,
		EncShareLists:           encSharesList,
		CommitteePKList:         committeePKList,
	}, nil
}
