package pot

import (
	"container/list"
	"crypto/rand"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/shuffle"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
	"github.com/zzz136454872/upgradeable-consensus/crypto/verifiable_draw"
)

var (
	group1 = bls12381.NewG1()
)

type queue = list.List

// CommitteeMark 记录委员会相关信息
type CommitteeMark struct {
	WorkHeight  uint32              // 委员会工作的PoT区块高度
	PKList      []*bls12381.PointG1 // 份额持有者的公钥列表，用于PVSS分发份额时查找委员会成员
	CommitteePK *bls12381.PointG1   // 委员会聚合公钥 y = g^s
}

// SelfCommitteeMark 记录自己所在的委员会相关信息
type SelfCommitteeMark struct {
	WorkHeight   uint32            // 委员会工作的PoT区块高度
	SelfPK       *bls12381.PointG1 // 自己的公钥,用于查找自己的份额
	AggrEncShare *mrpvss.EncShare  // 聚合加密份额
	Share        *bls12381.Fr      // 解密份额（聚合的）
}

type LocalStorage struct {
	// 通用
	BigN   uint32            // 候选公钥列表大小
	SmallN uint32            // 委员会大小
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

type Block struct {
	// 参数生成阶段
	Srs            *srs.SRS                    // 上一区块输出的SRS
	SrsUpdateProof *schnorr_proof.SchnorrProof // 上一区块输出的SRS更新证明
	// 置换阶段
	ShuffleProof *verifiable_draw.DrawProof // 公钥列表置换证明（上一区块的输出）
	// 抽签阶段
	DrawProof *verifiable_draw.DrawProof // 抽签证明（上一区块的输出）
	// DPVSS阶段
	CommitteeWorkHeightList []uint32              // 每个PVSS所对应委员会的工作高度
	HolderPKLists           [][]*bls12381.PointG1 // 多个PVSS参与者公钥列表，数量 1~SmallN-1
	ShareCommitLists        [][]*bls12381.PointG1 // 多个PVSS份额承诺列表，数量 1~SmallN-1
	CoeffCommitLists        [][]*bls12381.PointG1 // 多个PVSS系数承诺列表，数量 1~SmallN-1
	EncShareLists           [][]*mrpvss.EncShare  // 多个PVSS加密份额列表，数量 1~SmallN-1
	CommitteePKList         []*bls12381.PointG1   // 多个PVSS对应的委员会公钥（公钥y = g^s，s为秘密），数量 1~SmallN-1
}

var (
	localStorage  *LocalStorage
	receivedBlock *Block
)

func inInitStage(height uint32) bool {
	return height <= localStorage.BigN
}

func inShuffleStage(height uint32) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= localStorage.BigN {
		return false
	}
	relativeHeight := height % localStorage.BigN
	if relativeHeight > 0 && relativeHeight <= localStorage.SmallN {
		return true
	}
	return false
}

func isTheFirstShuffle(height uint32) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= localStorage.BigN {
		return false
	}
	if height%localStorage.BigN == 1 {
		return true
	}
	return false
}

func isTheLastShuffle(height uint32) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= localStorage.BigN {
		return false
	}
	if height%localStorage.BigN == localStorage.SmallN {
		return true
	}
	return false
}

func inDrawStage(height uint32) bool {
	return height > localStorage.BigN+localStorage.SmallN
}

func inDPVSSStage(height uint32) bool {
	return height > localStorage.BigN+localStorage.SmallN+1
}
func inWorkStage(height uint32) bool {
	return height > localStorage.BigN+2*localStorage.SmallN
}

// checkReceivedBlock 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func checkReceivedBlock(height uint32) bool {
	// 如果处于初始化阶段
	if inInitStage(height) {
		// 检查srs的更新证明，如果验证失败，则丢弃
		if !srs.Verify(receivedBlock.Srs, localStorage.LocalSRS.G1PowerOf(1), receivedBlock.SrsUpdateProof) {
			return false
		}
		return true
	}
	// 如果处于置换阶段，则验证置换
	if inShuffleStage(height) {
		// 验证置换
		// 如果验证失败，丢弃
		if !shuffle.Verify(localStorage.LocalSRS, localStorage.PrevShuffledPKList, localStorage.PrevRCommitForShuffle, receivedBlock.ShuffleProof) {
			return false
		}
	}
	// 如果处于抽签阶段，则验证抽签
	if inDrawStage(height) {
		// 验证抽签
		// 如果验证失败，丢弃
		if !verifiable_draw.Verify(localStorage.LocalSRS, localStorage.BigN, localStorage.PrevShuffledPKList, localStorage.SmallN, localStorage.PrevRCommitForDraw, receivedBlock.DrawProof) {
			return false
		}
	}
	// 如果处于DPVSS阶段
	if inDPVSSStage(height) {
		// 验证PVSS加密份额
		pvssTimes := len(receivedBlock.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 如果验证失败，丢弃
			if !mrpvss.VerifyEncShares(localStorage.SmallN, localStorage.Threshold, localStorage.G, localStorage.H, receivedBlock.HolderPKLists[i], receivedBlock.ShareCommitLists[i], receivedBlock.CoeffCommitLists[i], receivedBlock.EncShareLists[i]) {
				return false
			}
		}
	}
	return true
}

// 全部验证通过后
func updateLocalRecords(height uint32) {
	if inInitStage(height) {
		// 记录最新srs
		localStorage.LocalSRS = receivedBlock.Srs
	}
	// 置换阶段
	if inShuffleStage(height) {
		// 记录该置换公钥列表为最新置换公钥列表
		localStorage.PrevShuffledPKList = receivedBlock.ShuffleProof.SelectedPubKeys
		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
		localStorage.PrevRCommitForShuffle.Set(receivedBlock.ShuffleProof.RCommit)
		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
		if isTheLastShuffle(height) {
			localStorage.PrevRCommitForDraw.Set(localStorage.PrevRCommitForShuffle)
		}
	}
	// 抽签阶段
	if inDrawStage(height) {
		// 记录抽出来的委员会（公钥列表）
		localStorage.UnenabledCommitteeQueue.PushBack(CommitteeMark{
			WorkHeight:  height + localStorage.SmallN,
			PKList:      receivedBlock.DrawProof.SelectedPubKeys,
			CommitteePK: group1.Zero(),
		})
		// 获取抽签结果
		isSelected, pk := verifiable_draw.IsSelected(localStorage.SK, receivedBlock.DrawProof.RCommit, receivedBlock.DrawProof.SelectedPubKeys)
		// 如果抽中，在自己所在的委员会队列创建新条目记录
		if isSelected {
			localStorage.UnenabledSelfCommitteeQueue.PushBack(SelfCommitteeMark{
				WorkHeight:   height + localStorage.SmallN,
				SelfPK:       pk,
				AggrEncShare: mrpvss.NewEmptyEncShare(),
				Share:        bls12381.NewFr().Zero(),
			})
		}
	}
	// DPVSS阶段
	if inDPVSSStage(height) {
		pvssTimes := len(receivedBlock.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 聚合对应委员会的公钥
			for e := localStorage.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
				if e.Value.(*CommitteeMark).WorkHeight == receivedBlock.CommitteeWorkHeightList[i] {
					e.Value.(*CommitteeMark).CommitteePK = mrpvss.AggregateLeaderPK(e.Value.(*CommitteeMark).CommitteePK, receivedBlock.CommitteePKList[i])
				}
			}
			for e := localStorage.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
				// 如果自己是对应份额持有者(委员会工作高度相等)，则获取对应加密份额，聚合加密份额
				if e.Value.(SelfCommitteeMark).WorkHeight == receivedBlock.CommitteeWorkHeightList[i] {
					holderPKList := receivedBlock.HolderPKLists[i]
					encShares := receivedBlock.EncShareLists[i]
					holderNum := len(holderPKList)
					for j := 0; j < holderNum; j++ {
						if group1.Equal(e.Value.(*SelfCommitteeMark).SelfPK, holderPKList[j]) {
							// 聚合加密份额
							mrpvss.AggregateEncShares(e.Value.(*SelfCommitteeMark).AggrEncShare, encShares[j])
							e.Value.(*SelfCommitteeMark).AggrEncShare = encShares[j]
						}
					}
					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
					if int64(e.Value.(*SelfCommitteeMark).WorkHeight)-int64(height) == 1 {
						// 解密聚合加密份额
						e.Value.(*SelfCommitteeMark).Share = mrpvss.DecryptAggregateShare(localStorage.G, localStorage.SK, e.Value.(*SelfCommitteeMark).AggrEncShare, localStorage.SmallN-1)
					}
				}
			}

		}
	}
	// 委员会更新阶段
	if inWorkStage(height) {
		// 更新未工作的委员会队列，删除第一个元素
		if localStorage.UnenabledCommitteeQueue.Front() != nil {
			localStorage.UnenabledCommitteeQueue.Remove(localStorage.UnenabledCommitteeQueue.Front())
		}
		// 检查自己是否需要作为委员会成员工作
		// 遍历队列查找，如果自己该作为委员会成员执行业务处理，则解密聚合加密份额，并发送信号
		for e := localStorage.UnenabledSelfCommitteeQueue.Front(); e != nil; e = e.Next() {
			if height == e.Value.(*SelfCommitteeMark).WorkHeight {
				// 从自己所属委员会队列中移除该委员会标记
				// TODO 移除前接收数据（如果需要）
				localStorage.UnenabledSelfCommitteeQueue.Remove(e)
				// TODO 发送相关信号
			}
		}
	}
}

// height 要出块的区块高度
func uponMining(height uint32) (*Block, error) {
	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height) {
		// 更新 srs, 并生成证明
		r, _ := bls12381.NewFr().Rand(rand.Reader)
		newSRS, newSrsUpdateProof := localStorage.LocalSRS.Update(r)
		// 更新后的SRS和更新证明写入区块
		return &Block{
			Srs:            newSRS,
			SrsUpdateProof: newSrsUpdateProof,
		}, nil
	}
	var err error
	var newShuffleProof *verifiable_draw.DrawProof = nil
	var newDrawProof *verifiable_draw.DrawProof = nil
	var committeeWorkHeightList []uint32 = nil
	var holderPKLists [][]*bls12381.PointG1 = nil
	var shareCommitsList [][]*bls12381.PointG1 = nil
	var coeffCommitsList [][]*bls12381.PointG1 = nil
	var encSharesList [][]*mrpvss.EncShare = nil
	var committeePKList []*bls12381.PointG1 = nil
	// 如果处于置换阶段，则进行置换
	if inShuffleStage(height) {
		newShuffleProof = shuffle.SimpleShuffle(localStorage.LocalSRS, localStorage.PrevShuffledPKList, localStorage.PrevRCommitForShuffle)
	}
	// 如果处于抽签阶段，则进行抽签
	if inDrawStage(height) {
		// 生成秘密向量用于抽签，向量元素范围为 [1,BigN]
		permutation, err := utils.GenRandomPermutation(localStorage.BigN)
		if err != nil {
			return nil, err
		}
		secretVector := permutation[:localStorage.SmallN]
		newDrawProof, err = verifiable_draw.Draw(localStorage.LocalSRS, localStorage.BigN, localStorage.PrevShuffledPKList, localStorage.SmallN, secretVector, localStorage.PrevRCommitForDraw)
		// TODO handle error，能否直接return？还是循环调用？
		if err != nil {
			return nil, err
		}
	}
	// 如果处于PVSS阶段，则进行PVSS的分发份额
	if inDPVSSStage(height) {
		// 计算向哪些委员会进行pvss(委员会index从1开始)
		pvssTimes := height - localStorage.BigN - localStorage.SmallN - 1
		if pvssTimes > localStorage.SmallN-1 {
			pvssTimes = localStorage.SmallN - 1
		}
		// 要PVSS的委员会的工作高度
		committeeWorkHeightList = make([]uint32, pvssTimes)
		holderPKLists = make([][]*bls12381.PointG1, pvssTimes)
		shareCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		coeffCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		encSharesList = make([][]*mrpvss.EncShare, pvssTimes)
		committeePKList = make([]*bls12381.PointG1, pvssTimes)
		i := uint32(0)
		for e := localStorage.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
			// 委员会的工作高度 height - pvssTimes + i + SmallN
			committeeWorkHeightList[i] = height - pvssTimes + i + localStorage.SmallN
			secret, _ := bls12381.NewFr().Rand(rand.Reader)
			holderPKLists[i] = e.Value.(*CommitteeMark).PKList
			shareCommitsList[i], coeffCommitsList[i], encSharesList[i], committeePKList[i], err = mrpvss.EncShares(localStorage.G, localStorage.H, holderPKLists[i], secret, localStorage.Threshold)
			// TODO handle error，能否直接return？还是循环调用？
			if err != nil {
				return nil, err
			}
			i++
		}
	}
	return &Block{
		Srs:                     nil,
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
