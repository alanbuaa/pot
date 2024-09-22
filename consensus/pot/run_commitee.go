package pot

import (
	"container/list"
	"crypto/rand"
	"fmt"

	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	mrpvss "github.com/zzz136454872/upgradeable-consensus/crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/shuffle"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/verifiable_draw"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

var (
	BigN   = uint64(8)
	SmallN = uint64(4)
)

var (
	// test
	g1Degree = uint32(1 << 7)
	// test
	g2Degree = uint32(1 << 7)
)

type Queue = list.List

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
	SelfPubKey   *bls12381.PointG1 // 自己的公钥,用于查找自己的份额
	SelfPrivKey  *bls12381.Fr      // 自己的私钥, 用于解密自己的份额
	AggrEncShare *mrpvss.EncShare  // 聚合加密份额
	Share        *bls12381.Fr      // 解密份额（聚合的）
}

func (c *CommitteeMark) DeepCopy() *CommitteeMark {
	group1 := bls12381.NewG1()
	newPKList := make([]*bls12381.PointG1, len(c.PKList))
	for i := 0; i < len(c.PKList); i++ {
		if c.PKList[i] != nil {
			newPKList[i] = group1.New().Set(c.PKList[i])
		}
	}
	return &CommitteeMark{
		WorkHeight:  c.WorkHeight,
		PKList:      newPKList,
		CommitteePK: group1.New().Set(c.CommitteePK),
	}
}

func (s *SelfCommitteeMark) DeepCopy() *SelfCommitteeMark {
	group1 := bls12381.NewG1()
	return &SelfCommitteeMark{
		WorkHeight:   s.WorkHeight,
		SelfPubKey:   group1.New().Set(s.SelfPubKey),
		SelfPrivKey:  bls12381.NewFr().Set(s.SelfPrivKey),
		AggrEncShare: s.AggrEncShare.DeepCopy(),
		Share:        bls12381.NewFr().Set(s.Share),
	}
}

func BackUpUnenabledCommitteeQueue(q *Queue) *Queue {
	backup := new(Queue)
	for e := q.Front(); e != nil; e = e.Next() {
		backup.PushBack(e.Value.(*CommitteeMark).DeepCopy())
	}
	return backup
}

func BackUpUnenabledSelfCommitteeQueue(q *Queue) *Queue {
	backup := new(Queue)
	for e := q.Front(); e != nil; e = e.Next() {
		backup.PushBack(e.Value.(*SelfCommitteeMark).DeepCopy())
	}
	return backup
}

// func (w *Worker) simpleLeaderUpdate(parent *types.Header) {
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
//			returnf
//		}
//		if w.potSignalChan != nil {
//			w.potSignalChan <- b
//		}
//	}
// }

// func (w *Worker) committeeCheck(id int64, header *types.Header) bool {
//	if _, exist := w.committee.Get(id); !exist {
//		w.committee.Set(id, header)
//		return false
//	}
//	return true
// }
//
// func (w *Worker) committeeSizeCheck() bool {
//	return w.committee.Len() == 4
// }

func (w *Worker) GetPeerQueue() chan *types.Block {
	return w.peerMsgQueue
}

func (w *Worker) CommitteeUpdate(height uint64) {
	// if epoch > 10 && w.ID == 1 {
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
	// }
	// potsignal := &simpleWhirly.PoTSignal{
	// 	Epoch:               int64(epoch),
	// 	Proof:               nil,
	// 	ID:                  0,
	// 	LeaderPublicAddress: committee[0],
	// 	Committee:           committee,
	// 	SelfPublicAddress:   selfaddress,
	// 	CryptoElements:      nil,
	// }
	// if height >= CommiteeDelay+Commiteelen {
	//	committee := make([]string, Commiteelen)
	//	selfaddress := make([]string, 0)
	//	for i := uint64(0); i < Commiteelen; i++ {
	//		block, err := w.chainReader.GetByHeight(height - CommiteeDelay - i)
	//		if err != nil {
	//			return
	//		}
	//		if block != nil {
	//			header := block.GetHeader()
	//			committee[i] = hexutil.Encode(header.PublicKey)
	//			flag, _ := w.TryFindKey(crypto.Convert(header.Hash()))
	//			if flag {
	//				selfaddress = append(selfaddress, hexutil.Encode(header.PublicKey))
	//			}
	//		}
	//	}
	//
	//	whilyConsensus := &config.WhirlyConfig{
	//		Type:      "simple",
	//		BatchSize: 2,
	//		Timeout:   2,
	//	}
	//
	//	consensus := config.ConsensusConfig{
	//		Type:        "whirly",
	//		ConsensusID: 1201,
	//		Whirly:      whilyConsensus,
	//		Nodes:       w.config.Nodes,
	//		Topic:       w.config.Topic,
	//		F:           w.config.F,
	//	}
	//
	//	sharding1 := nodeController.PoTSharding{
	//		Name:                hexutil.EncodeUint64(1),
	//		ParentSharding:      nil,
	//		LeaderPublicAddress: committee[0],
	//		Committee:           committee,
	//		CryptoElements:      bc_api.CommitteeConfig{},
	//		SubConsensus:        consensus,
	//	}
	//
	//	//w.log.Error(len(committee))
	//
	//	//sharding2 := simpleWhirly.PoTSharding{
	//	//	Name:                "hello_world",
	//	//	ParentSharding:      nil,
	//	//	LeaderPublicAddress: committee[0],
	//	//	Committee:           committee,
	//	//	CryptoElements:      nil,
	//	//	SubConsensus:        consensus,
	//	//}
	//	//shardings := []simpleWhirly.PoTSharding{sharding1, sharding2}
	//
	//	shardings := []nodeController.PoTSharding{sharding1}
	//	potsignal := &nodeController.PoTSignal{
	//		Epoch:             int64(height),
	//		Proof:             make([]byte, 0),
	//		ID:                0,
	//		SelfPublicAddress: selfaddress,
	//		Shardings:         shardings,
	//	}
	//	b, err := json.Marshal(potsignal)
	//	if err != nil {
	//		w.log.WithError(err)
	//		return
	//	}
	//	if w.potSignalChan != nil {
	//		w.potSignalChan <- b
	//	}
	// }

}

type CryptoSet struct {
	// 通用
	BigN   uint64            // 候选公钥列表大小
	SmallN uint64            // 委员会大小
	G      *bls12381.PointG1 // G1生成元
	H      *bls12381.PointG1 // G1生成元
	// 参数生成阶段
	LocalSRS *srs.SRS // 本地记录的最新有效SRS
	// 置换阶段
	PrevShuffledPKList    []*bls12381.PointG1 // 前一有效置换后的公钥列表
	PrevRCommitForShuffle *bls12381.PointG1   // 前一有效置换随机数承诺
	// 抽签阶段
	PrevRCommitForDraw          *bls12381.PointG1 // 前一有效抽签随机数承诺
	UnenabledCommitteeQueue     *Queue            // 抽签产生的委员会队列（未启用的）
	UnenabledSelfCommitteeQueue *Queue            // 自己所在的委员会队列（未启用的）
	// DPVSS阶段
	Threshold uint32 // DPVSS恢复门限
}

func (c *CryptoSet) Backup(height uint64) *CryptoSet {
	group1 := bls12381.NewG1()
	backup := &CryptoSet{
		BigN:      c.BigN,
		SmallN:    c.SmallN,
		Threshold: c.Threshold,
	}
	if c.G != nil {
		backup.G = group1.New().Set(c.G)
	}
	if c.H != nil {
		backup.H = group1.New().Set(c.H)
	}
	if c.PrevRCommitForShuffle != nil {
		backup.PrevRCommitForShuffle = group1.New().Set(c.PrevRCommitForShuffle)
	}
	if c.PrevRCommitForDraw != nil {
		backup.PrevRCommitForDraw = group1.New().Set(c.PrevRCommitForDraw)
	}
	if inInitStage(height) && c.LocalSRS != nil {
		backup.LocalSRS = new(srs.SRS).Set(c.LocalSRS)
	}
	if c.PrevShuffledPKList != nil {
		backup.PrevShuffledPKList = make([]*bls12381.PointG1, len(c.PrevShuffledPKList))
		for i, p := range c.PrevShuffledPKList {
			if p != nil {
				backup.PrevShuffledPKList[i] = new(bls12381.PointG1).Set(p)
			}
		}
	}
	if c.UnenabledCommitteeQueue != nil {
		backup.UnenabledCommitteeQueue = BackUpUnenabledCommitteeQueue(c.UnenabledCommitteeQueue)
		backup.UnenabledSelfCommitteeQueue = BackUpUnenabledSelfCommitteeQueue(c.UnenabledSelfCommitteeQueue)
	}
	return backup
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
func isTheFirstShuffle(height uint64) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= BigN {
		return false
	}
	if height%BigN == 1 {
		return true
	}
	return false
}

func inDPVSSStage(height uint64) bool {
	return height > CandidateKeyLen+Commitees+1
}
func inWorkStage(height uint64) bool {
	return height > BigN+2*SmallN
}

// TODO isSelfBlock 判断高度为 height 的区块是否是自己出块的
func isSelfBlock(height uint64) bool {
	return false
}

// uponReceivedBlock 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func (w *Worker) uponReceivedBlock(
	height uint64, block *types.Block,
) bool {
	group1 := bls12381.NewG1()
	// 如果处于初始化阶段（参数生成阶段）
	receivedBlock := block.GetHeader().CryptoElement
	if inInitStage(height) {
		// TODO
		prevSRSG1FirstElem := group1.One()
		if w.Cryptoset.LocalSRS != nil {
			prevSRSG1FirstElem = w.Cryptoset.LocalSRS.G1PowerOf(1)
		}

		// 检查srs的更新证明，如果验证失败，则丢弃
		if !srs.Verify(receivedBlock.SRS, prevSRSG1FirstElem, receivedBlock.SrsUpdateProof) {
			fmt.Printf("[Height %d]:Verify SRS error\n", height)
			return false
		}
		return true
	}
	// 如果处于置换阶段，则验证置换
	if inShuffleStage(height) {
		// 验证置换
		// 第一次置换
		if isTheFirstShuffle(height) {
			// 获取前面BigN个区块的公钥
			w.Cryptoset.PrevShuffledPKList = w.getPrevNBlockPKList(height-BigN, height-1)
		}
		// 如果验证失败，丢弃
		if !shuffle.Verify(w.Cryptoset.LocalSRS, w.Cryptoset.PrevShuffledPKList, w.Cryptoset.PrevRCommitForShuffle, receivedBlock.ShuffleProof) {
			return false
		}
	}
	// 如果处于抽签阶段，则验证抽签
	if inDrawStage(height) {
		// 验证抽签
		// 如果验证失败，丢弃
		if !verifiable_draw.Verify(w.Cryptoset.LocalSRS, uint32(BigN), w.Cryptoset.PrevShuffledPKList, uint32(w.Cryptoset.SmallN), w.Cryptoset.PrevRCommitForDraw, receivedBlock.DrawProof) {
			return false
		}
	}
	// 如果处于DPVSS阶段
	if inDPVSSStage(height) {
		// 验证PVSS加密份额
		pvssTimes := len(receivedBlock.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 如果验证失败，丢弃
			if !mrpvss.VerifyEncShares(uint32(SmallN), w.Cryptoset.Threshold, w.Cryptoset.G, w.Cryptoset.H, receivedBlock.HolderPKLists[i], receivedBlock.ShareCommitLists[i], receivedBlock.CoeffCommitLists[i], receivedBlock.EncShareLists[i]) {
				return false
			}
		}
	}
	return true
}

func (w *Worker) UpdateLocalCryptoSetByBlock(height uint64, receivedBlock *types.Block) {
	group1 := bls12381.NewG1()
	cm := receivedBlock.GetHeader().CryptoElement
	// 初始化阶段
	if inInitStage(height) {
		fmt.Printf("[Update]: Init | Node %v, Block %v\n", w.ID, height)
		// 记录最新srs
		w.Cryptoset.LocalSRS = cm.SRS
		// 如果处于最后一次初始化阶段，保存SRS为文件（用于Caulk+）,并启动Caulk+进程
		if !inInitStage(height + 1) {
			// 更新 H
			w.Cryptoset.H = bls12381.NewG1().New().Set(w.Cryptoset.LocalSRS.G1PowerOf(1))
			// 保存SRS至srs.binary文件
			w.Cryptoset.LocalSRS.ToBinaryFile()
			// err := utils.RunCaulkPlusGRPC()
		}
	}
	// 置换阶段
	if inShuffleStage(height) {
		fmt.Printf("[Update]: Shuffle | Node %v, Block %v\n", w.ID, height)
		// 记录该置换公钥列表为最新置换公钥列表
		w.Cryptoset.PrevShuffledPKList = cm.ShuffleProof.SelectedPubKeys
		// fmt.Println(cm.ShuffleProof.SelectedPubKeys)
		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
		if cm.ShuffleProof.RCommit == nil || w.Cryptoset.PrevShuffledPKList == nil {
			panic("shuffle proof Rcommit is nil")
		}
		w.Cryptoset.PrevRCommitForShuffle.Set(cm.ShuffleProof.RCommit)
		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
		if isTheLastShuffle(height) {
			w.Cryptoset.PrevRCommitForDraw.Set(w.Cryptoset.PrevRCommitForShuffle)
		}
	}
	// 抽签阶段
	if inDrawStage(height) {
		fmt.Printf("[Update]: Draw | Node %v, Block %v\n", w.ID, height)
		// 记录抽出来的委员会（公钥列表）
		w.Cryptoset.UnenabledCommitteeQueue.PushBack(&CommitteeMark{
			WorkHeight:  height + SmallN,
			PKList:      cm.DrawProof.SelectedPubKeys,
			CommitteePK: group1.Zero(),
		})
		// 对于每个自己区块的公私钥对获取抽签结果，是否被选中，如果被选中则获取自己的公钥
		// 获取自己的区块私钥(同一置换/抽签分组内的，即1~32，33~64...)
		maxHeight := ((height - w.Cryptoset.SmallN) >> 5) << 5
		minHeight := maxHeight - w.Cryptoset.SmallN + 1
		selfPrivKeyList := w.getSelfPrivKeyList(minHeight, maxHeight)
		for _, privKey := range selfPrivKeyList {
			isSelected, pk := verifiable_draw.IsSelected(privKey, cm.DrawProof.RCommit, cm.DrawProof.SelectedPubKeys)
			// 如果抽中，在自己所在的委员会队列创建新条目记录
			if isSelected {
				// test
				fmt.Printf("Mark committee you are in, current height: %d, committee work height: %d\n", height, height+w.Cryptoset.SmallN)
				w.Cryptoset.UnenabledSelfCommitteeQueue.PushBack(&SelfCommitteeMark{
					WorkHeight:   height + w.Cryptoset.SmallN,
					SelfPubKey:   pk,
					SelfPrivKey:  privKey,
					AggrEncShare: mrpvss.NewEmptyEncShare(),
					Share:        bls12381.NewFr().Zero(),
				})
			}
		}
	}
	// DPVSS阶段
	if inDPVSSStage(height) {
		fmt.Printf("[Update]: DPVSS | Node %v, Block %v\n", w.ID, height)
		pvssTimes := len(cm.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 聚合对应委员会的公钥
			for e := w.Cryptoset.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
				if e.Value.(*CommitteeMark).WorkHeight == cm.CommitteeWorkHeightList[i] {
					e.Value.(*CommitteeMark).CommitteePK = mrpvss.AggregateLeaderPK(e.Value.(*CommitteeMark).CommitteePK, cm.CommitteePKList[i])
				}
			}
			for e := w.Cryptoset.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
				mark := e.Value.(*SelfCommitteeMark)
				// 如果自己是对应份额持有者(委员会工作高度相等)，则获取对应加密份额，聚合加密份额
				if mark.WorkHeight == cm.CommitteeWorkHeightList[i] {
					holderPKList := cm.HolderPKLists[i]
					encShares := cm.EncShareLists[i]
					holderNum := len(holderPKList)
					for j := 0; j < holderNum; j++ {
						if group1.Equal(mark.SelfPubKey, holderPKList[j]) {
							// 聚合加密份额
							mark.AggrEncShare = mrpvss.AggregateEncShares(mark.AggrEncShare, encShares[j])
						}
					}
					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
					if int64(mark.WorkHeight)-int64(height) == 1 {
						// 解密聚合加密份额
						mark.Share = mrpvss.DecryptAggregateShare(w.Cryptoset.G, mark.SelfPrivKey, mark.AggrEncShare, uint32(SmallN-1))
					}
				}
			}

		}
	}
	// 委员会更新阶段
	if inWorkStage(height) {
		fmt.Printf("[Update]: Update Committee | Node %v, Block %v\n", w.ID, height)
		// TODO 判断自己是否作为领导者进行工作
		if isSelfBlock(height) {
			// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
			if w.Cryptoset.UnenabledCommitteeQueue.Front() != nil {
				e := w.Cryptoset.UnenabledCommitteeQueue.Front()
				// 获取该委员会公钥
				committeeInfo := e.Value.(*CommitteeMark).CommitteePK
				// 删除该委员会
				w.Cryptoset.UnenabledCommitteeQueue.Remove(w.Cryptoset.UnenabledCommitteeQueue.Front())
				// TODO 传递该委员会信息给业务链（作为委员会领导者）
				handleCommitteeAsLeader(height, committeeInfo)
				return
			}
		}

		// 检查自己是否需要作为委员会成员工作
		// 遍历队列查找，如果自己该作为委员会成员执行业务处理，则解密聚合加密份额，并发送信号
		for e := w.Cryptoset.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
			if height == e.Value.(*SelfCommitteeMark).WorkHeight {
				// 从自己所属委员会队列中移除该委员会标记
				memberInfo := e.Value.(*SelfCommitteeMark)
				w.Cryptoset.UnenabledSelfCommitteeQueue.Remove(e)
				// TODO 发送相关信号
				handleCommitteeAsMember(height, memberInfo)
				return
			}
		}
		// 作为用户，传递抗审查数据
		// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
		if w.Cryptoset.UnenabledCommitteeQueue.Front() != nil {
			e := w.Cryptoset.UnenabledCommitteeQueue.Front()
			// 获取该委员会公钥
			committeeInfo := e.Value.(*CommitteeMark).CommitteePK
			// 删除该委员会
			w.Cryptoset.UnenabledCommitteeQueue.Remove(w.Cryptoset.UnenabledCommitteeQueue.Front())
			// TODO 传递该委员会公钥给业务链(用于向该委员会发送交易)
			handleCommitteeAsUser(height, committeeInfo)
		}
	}
}

func (w *Worker) SetWhirly(impl *nodeController.NodeController) {
	w.whirly = impl
	w.potSignalChan = impl.GetPoTByteEntrance()
}

// func (w *Worker) CommiteeLenCheck() bool {
//	if len(w.Commitee) != Commiteelen {
//		return false
//	}
//	return true
// }

// func (w *Worker) GetCommiteeLeader() string {
//	return w.Commitee[Commiteelen-1]
// }

// func (w *Worker) UpdateCommitee(commiteeid int, peerid string) []string {
//	if w.CommiteeLenCheck() {
//		w.Commitee[commiteeid] = w.Commitee[commiteeid][1:Commiteelen]
//		w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
//	}
//	if w.CommiteeLenCheck() {
//		return w.Commitee[commiteeid]
//	}
//	return nil
// }
//
// func (w *Worker) AppendCommitee(commiteeid int, peerid string) {
//	w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
// }

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

func (w *Worker) GenerateCryptoSetFromLocal(height uint64) (types.CryptoElement, error) {
	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height) {
		fmt.Printf("[Mining]: Init | Node %v, Block %v\n", w.ID, height)
		// 更新 srs, 并生成证明
		var newSRS *srs.SRS
		var newSrsUpdateProof *schnorr_proof.SchnorrProof
		// 如果之前未生成SRS，新生成一个SRS
		if w.Cryptoset.LocalSRS == nil {
			newSRS, newSrsUpdateProof = srs.NewSRS(g1Degree, g2Degree)
		} else {
			r, _ := bls12381.NewFr().Rand(rand.Reader)
			newSRS, newSrsUpdateProof = w.Cryptoset.LocalSRS.Update(r)
		}
		// 更新后的SRS和更新证明写入区块
		return types.CryptoElement{
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
		fmt.Printf("[Mining]: Shuffle | Node %v, Block %v\n", w.ID, height)
		if isTheFirstShuffle(height) {
			// 获取前面BigN个区块的公钥
			w.Cryptoset.PrevShuffledPKList = w.getPrevNBlockPKList(height-BigN, height-1)
			fmt.Println(len(w.Cryptoset.PrevShuffledPKList))
		}
		newShuffleProof = shuffle.SimpleShuffle(w.Cryptoset.LocalSRS, w.Cryptoset.PrevShuffledPKList, w.Cryptoset.PrevRCommitForShuffle)

		// shuffleproofbyte, _ := newShuffleProof.ToBytes()
		// var shuffleproof *verifiable_draw.DrawProof = nil
		// shuffleproof, _ = shuffleproof.FromBytes(shuffleproofbyte)

	}
	// 如果处于抽签阶段，则进行抽签
	if inDrawStage(height) {
		// test
		fmt.Printf("[Mining]: Draw | Node %v, Block %v\n", w.ID, height)
		// 生成秘密向量用于抽签，向量元素范围为 [1, BigN]
		permutation, err := utils.GenRandomPermutation(uint32(w.Cryptoset.BigN))
		if err != nil {
			return types.CryptoElement{}, err
		}
		secretVector := permutation[:w.Cryptoset.SmallN]
		newDrawProof, err = verifiable_draw.Draw(w.Cryptoset.LocalSRS, uint32(w.Cryptoset.BigN), w.Cryptoset.PrevShuffledPKList, uint32(w.Cryptoset.SmallN), secretVector, w.Cryptoset.PrevRCommitForDraw)
		if err != nil {
			return types.CryptoElement{}, err
		}
	}
	// 如果处于PVSS阶段，则进行PVSS的分发份额
	if inDPVSSStage(height) {
		// 计算向哪些委员会进行pvss(委员会index从1开始)
		pvssTimes := height - w.Cryptoset.BigN - w.Cryptoset.SmallN - 1
		fmt.Printf("[Mining]: DPVSS | Node %v, Block %v | PVSS Times: %v\n", w.ID, height, pvssTimes)
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
			if i >= pvssTimes {
				break
			}
			// 委员会的工作高度 height - pvssTimes + i + SmallN
			committeeWorkHeightList[i] = height - pvssTimes + i + SmallN
			secret, _ := bls12381.NewFr().Rand(rand.Reader)
			holderPKLists[i] = e.Value.(*CommitteeMark).PKList
			shareCommitsList[i], coeffCommitsList[i], encSharesList[i], committeePKList[i], err = mrpvss.EncShares(w.Cryptoset.G, w.Cryptoset.H, holderPKLists[i], secret, w.Cryptoset.Threshold)
			if err != nil {
				return types.CryptoElement{}, err
			}
			i++
		}
	}
	return types.CryptoElement{
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

// TODO handleCommitteeAsUser 作为用户传递该委员会信息给业务链(用于向该委员会发送交易)
func handleCommitteeAsUser(height uint64, pk *bls12381.PointG1) {

}

// TODO handleCommitteeAsMember 作为委员会成员传递该委员会信息给业务链(用于处理业务)
func handleCommitteeAsMember(height uint64, info *SelfCommitteeMark) {

}

// TODO handleCommitteeAsLeader 作为委员会领导者传递该委员会信息给业务链(用于处理业务)
func handleCommitteeAsLeader(height uint64, info *bls12381.PointG1) {

}

// TODO getSelfPrivKeyList 获取高度位于 [minHeight, maxHeight] 内所有自己出块的区块私钥
func (w *Worker) getSelfPrivKeyList(minHeight, maxHeight uint64) []*bls12381.Fr {
	var ret []*bls12381.Fr

	for i := minHeight; i <= maxHeight; i++ {
		block, err := w.chainReader.GetByHeight(i)
		if err != nil {
			return nil
		}

		flag, priv := w.TryFindKey(crypto.Convert(block.Hash()))
		if flag {
			fr := bls12381.NewFr().FromBytes(priv)
			ret = append(ret, fr)
		}
	}
	return ret
}

// TODO getPrevNBlockPKList 获取高度位于 [minHeight, maxHeight] 内所有区块的公钥
func (w *Worker) getPrevNBlockPKList(minHeight, maxHeight uint64) []*bls12381.PointG1 {
	ret := make([]*bls12381.PointG1, 0)
	for i := minHeight; i <= maxHeight; i++ {
		block, err := w.chainReader.GetByHeight(i)
		if err != nil {
			return nil
		}
		pub := block.GetHeader().PublicKey
		point, err := bls12381.NewG1().FromBytes(pub)
		if err != nil {
			return nil
		}
		ret = append(ret, point)
	}
	return ret
}
