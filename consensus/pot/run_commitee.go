package pot

import (
	"container/list"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	mrpvss "github.com/zzz136454872/upgradeable-consensus/crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/shuffle"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/verifiable_draw"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

type Sharding struct {
	Name            string
	Id              int32
	LeaderAddress   string
	Committee       []string
	consensusconfig config.ConsensusConfig
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

func (w *Worker) CommitteeUpdate(height uint64) {

	if height >= CommiteeDelay+Commiteelen {
		committee := make([]string, Commiteelen)
		selfaddress := make([]string, 0)
		for i := uint64(0); i < Commiteelen; i++ {
			block, err := w.chainReader.GetByHeight(height - CommiteeDelay - i)
			if err != nil {
				return
			}
			if block != nil {
				header := block.GetHeader()
				committee[i] = hexutil.Encode(header.PublicKey)
				flag, _ := w.TryFindKey(crypto.Convert(header.Hash()))
				if flag {
					selfaddress = append(selfaddress, hexutil.Encode(header.PublicKey))
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
			F:           w.config.F,
		}

		sharding1 := simpleWhirly.PoTSharding{
			Name:                hexutil.EncodeUint64(1),
			ParentSharding:      nil,
			LeaderPublicAddress: committee[0],
			Committee:           committee,
			CryptoElements:      nil,
			SubConsensus:        consensus,
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

		shardings := []simpleWhirly.PoTSharding{sharding1}
		potsignal := &simpleWhirly.PoTSignal{
			Epoch:             int64(height),
			Proof:             make([]byte, 0),
			ID:                0,
			SelfPublicAddress: selfaddress,
			Shardings:         shardings,
		}
		b, err := json.Marshal(potsignal)
		if err != nil {
			w.log.WithError(err)
			return
		}
		if w.potSignalChan != nil {
			w.potSignalChan <- b
		}
	}
	//if epoch > 10 && w.ID == 1 {
	//	block, err := w.chainReader.GetByHeight(epoch - 1)
	//	if err != nil {
	//		return
	//	}
	//	header := block.GetHeader()
	//	fill, err := os.OpenFile("difficulty", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	_, err = fill.WriteString(fmt.Sprintf("%d\Commitees", header.Difficulty.Int64()))
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	fill.Close()
	//}
}

type queue = list.List

type CryptoSet struct {
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 通用
	sk *bls12381.Fr      // 该节点的私钥
	g  *bls12381.PointG1 // G1生成元
	h  *bls12381.PointG1 // G1生成元
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 更新阶段
	localSRS         *srs.SRS                    // 本地记录的最新有效SRS
	receivedSRSBytes []byte                      // 上一区块输出的SRS(压缩形式)
	srsUpdateProof   *schnorr_proof.SchnorrProof // 上一区块输出的SRS更新证明
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 置换阶段
	prevshuffledPKList    []*bls12381.PointG1        // 置换用的公钥列表（上一区块的输出，当前待出块区块的输入）
	shuffleProof          *verifiable_draw.DrawProof // 公钥列表置换证明（上一区块的输出）
	prevRCommitForShuffle *bls12381.PointG1          // 前一有效置换随机数承诺
	selfShuffledPK        *bls12381.PointG1          // 自己的置换后的公钥（上一区块的输出）
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 抽签阶段
	prevRCommitForDraw *bls12381.PointG1          // 前一有效抽签随机数承诺
	drawProof          *verifiable_draw.DrawProof // 抽签证明（上一区块的输出）
	//
	committeeQueue *queue
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// DPVSS阶段
	threshold uint32 // DPVSS恢复门限
	// selfPKList          []*bls12381.PointG1   // 自己公钥的列表
	receivedPVSSPKLists [][]*bls12381.PointG1 // 多个PVSS参与者公钥列表，数量 1~Commitees-1（上一区块的输出）
	shareCommitLists    [][]*bls12381.PointG1 // 多个PVSS份额承诺列表，数量 1~Commitees-1（上一区块的输出）
	coeffCommitLists    [][]*bls12381.PointG1 // 多个PVSS系数承诺列表，数量 1~Commitees-1（上一区块的输出）
	encShareLists       [][]*mrpvss.EncShare  // 多个PVSS加密份额列表，数量 1~Commitees-1（上一区块的输出）

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

// uponReceivedBlock 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func (w *Worker) uponReceivedBlock(
	height uint64, block *types.Block,
) bool {
	// 如果处于初始化阶段（参数生成阶段）
	receiveCryptoSet := block.GetHeader().CryptoSet
	if inInitStage(height) {
		// 解析压缩的srs
		receivedSRS := receiveCryptoSet.SRS
		// 检查srs的更新证明，如果验证失败，则丢弃
		if !srs.Verify(receivedSRS, w.Cryptoset.localSRS.G1Power(1), receiveCryptoSet.SrsUpdateProof) {
			// 记录最新srs
			return false
		}
		return true
	}
	// 判断阶段
	doShuffle := inShuffleStage(height)
	doDraw := inDrawStage(height)
	doDPVSS := inDPVSSStage(height)
	// 如果处于置换阶段，则验证置换(验证部分)
	if doShuffle {
		// 验证置换
		// 如果验证失败，丢弃
		if !shuffle.Verify(w.Cryptoset.localSRS, w.Cryptoset.prevshuffledPKList, w.Cryptoset.prevRCommitForShuffle, receiveCryptoSet.ShuffleProof) {
			return false
		}
		// 更新部分在后面
	}
	// 如果处于抽签阶段，则验证抽签(验证部分)
	if doDraw {
		// 验证抽签
		// 如果验证失败，丢弃
		if !verifiable_draw.Verify(w.Cryptoset.localSRS, CandidateKeyLen, w.Cryptoset.prevshuffledPKList, Commitees, w.Cryptoset.prevRCommitForDraw, receiveCryptoSet.DrawProof) {
			return false
		}
		// 更新部分在后面
	}
	// 如果处于DPVSS阶段
	if doDPVSS {
		// 验证PVSS加密份额
		pvssTimes := len(w.Cryptoset.receivedPVSSPKLists)
		for i := 0; i < pvssTimes; i++ {
			if !mrpvss.VerifyEncShares(Commitees, w.Cryptoset.threshold, w.Cryptoset.g, w.Cryptoset.h, receiveCryptoSet.PVSSPKLists[i], receiveCryptoSet.ShareCommitLists[i], receiveCryptoSet.CoeffCommitLists[i], receiveCryptoSet.EncShareLists[i]) {
				// 如果加密份额验证不通过，简单方法：丢弃
				return false
			}
		}
		// 更新部分在后面
	}
	return true
}

func (w *Worker) ImprovedCommitteeUpdate(height uint64, committeeNum int, block *types.Block) {
	//// 判断阶段
	//doShuffle := inShuffleStage(height)
	//doDraw := inDrawStage(height)
	//doDPVSS := inDPVSSStage(height)
	//// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//cryptoset := block.GetHeader().CryptoSet
	//// 全部验证通过后
	//// 置换阶段（更新部分）
	//if doShuffle {
	//	// 如果验证成功, 记录该置换为最新有效置换，获取自己置换后的公钥
	//	// 记录该置换公钥列表为最新置换公钥列表
	//	w.Cryptoset.prevshuffledPKList = cryptoset.ShuffleProof.SelectedPubKeys
	//	// 记录该置换随机数承诺为最新置换随机数承诺 prevRCommitForShuffle
	//	w.Cryptoset.prevRCommitForShuffle.Set(cryptoset.ShuffleProof.RCommit)
	//	// 如果处于置换阶段的最后一次置换（height%CandidateKeyLen==Commitees），更新 prevRCommitForDraw = shuffleProof.RCommit
	//	//  因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
	//	//  顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
	//	// 抽签阶段不改变 prevRCommitForDraw
	//	if isTheLastShuffle(height) {
	//		w.Cryptoset.prevRCommitForDraw.Set(w.Cryptoset.prevRCommitForShuffle)
	//	}
	//}
	//// 抽签阶段（更新部分）
	//if doDraw {
	//	// 如果验证成功, 获取抽签结果
	//	isSelected, pk := verifiable_draw.IsSelected(sk, drawProof.RCommit, drawProof.SelectedPubKeys)
	//	// 如果抽中，创建新条目记录
	//	if isSelected {
	//		committeeQueue.PushBack(CommitteeMark{
	//			WorkHeight:   height + Commitees,
	//			PK:           pk,
	//			AggrEncShare: mrpvss.NewEmptyEncShare(),
	//			Share:        bls12381.NewFr().Zero(),
	//		})
	//	}
	//}
	//// DPVSS阶段（更新部分）
	//if doDPVSS {
	//	// 如果加密份额验证通过, 如果自己是对应份额持有者（自己的公钥列表与PVSS公钥列表有交集），则获取对应加密份额，聚合加密份额
	//	for e := committeeQueue.Front(); e != nil; e = e.Next() {
	//		heightDiff := int64(e.Value.(*CommitteeMark).WorkHeight) - int64(height)
	//		// 如果自己是对应份额持有者（自己的公钥列表与PVSS公钥列表有交集），则获取对应加密份额，聚合加密份额
	//		if heightDiff > 0 && heightDiff < int64(Commitees) {
	//			// 查找公钥位置，根据位置获取对应加密份额，聚合加密份额
	//			pvssPKList := receivedPVSSPKLists[heightDiff-1]
	//			encShares := encShareLists[heightDiff-1]
	//			pvssPKNum := len(pvssPKList)
	//			for i := 0; i < pvssPKNum; i++ {
	//				if group1.Equal(e.Value.(*CommitteeMark).PK, pvssPKList[i]) {
	//					// 聚合加密份额
	//					mrpvss.AggregateEncShares(e.Value.(*CommitteeMark).AggrEncShare, encShares[i])
	//					e.Value.(*CommitteeMark).AggrEncShare = encShares[i]
	//				}
	//			}
	//		}
	//		// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
	//		if heightDiff == 1 {
	//			// 解密加密份额
	//			e.Value.(*CommitteeMark).Share = mrpvss.DecryptAggregateShare(g, sk, e.Value.(*CommitteeMark).AggrEncShare, Commitees-1)
	//		}
	//	}
	//}
	//// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//// 检查自己是否需要作为委员会成员工作
	//// 遍历队列查找，如果自己该作为委员会成员执行业务处理，则解密聚合加密份额，并发送信号
	//for e := committeeQueue.Front(); e != nil; e = e.Next() {
	//	if height == e.Value.(*CommitteeMark).WorkHeight {
	//		// 从队列中移除该委员会标记
	//		// TODO 移除前接收数据（如果需要）
	//		committeeQueue.Remove(e)
	//		// TODO 发送相关信号
	//	}
	//}
}

func (w *Worker) SetWhirly(impl *simpleWhirly.NodeController) {
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
				flag, _ := w.TryFindKey(crypto.Convert(header.Hash()))
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
