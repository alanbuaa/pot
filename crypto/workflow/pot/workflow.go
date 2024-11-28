package pot

import (
	"container/list"
	"crypto/rand"
	"fmt"
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
	// test
	g1Degree = uint32(1 << 8)
	// test
	g2Degree = uint32(1 << 8)
)

type queue = list.List

// CommitteeMark 记录委员会相关信息
type CommitteeMark struct {
	WorkHeight   uint32              // 委员会工作的PoT区块高度
	MemberPKList []*bls12381.PointG1 // 份额持有者的公钥列表，用于PVSS分发份额时查找委员会成员
	CommitteePK  *bls12381.PointG1   // 委员会聚合公钥 y = g^s
}

// SelfMemberMark 记录自己所在的委员会相关信息
type SelfMemberMark struct {
	WorkHeight   uint32            // 委员会工作的PoT区块高度
	SelfPubKey   *bls12381.PointG1 // 自己的公钥, 用于查找自己的份额
	SelfPrivKey  *bls12381.Fr      // 自己的私钥, 用于解密自己的份额
	AggrEncShare *mrpvss.EncShare  // 聚合加密份额
	Share        *bls12381.Fr      // 解密份额（聚合的）
}

// TODO 向业务层传递所需数据
// type CommitteeConfig struct {
// 	H           []byte      // 生成元h
// 	PK          []byte      // 委员会公钥
// 	DPVSSConfig DPVSSConfig // 委员会成员、领导者拥有
// }
//
// type DPVSSConfig struct {
// 	Index        uint32   // 自己的位置
// 	ShareCommits [][]byte // 份额承诺
// 	Share        []byte   // 份额
// 	IsLeader     bool
// }

type MEVLocalStorage struct {
	// 通用
	candidateBlockPrivateKey *bls12381.Fr      // 候选块的
	BigN                     uint32            // 候选公钥列表大小
	SmallN                   uint32            // 委员会大小
	G                        *bls12381.PointG1 // G1生成元
	H                        *bls12381.PointG1 // G1生成元
	// 参数生成阶段
	LocalSRS *srs.SRS // 本地记录的最新有效SRS
	// 置换阶段
	PrevShuffledPKList    []*bls12381.PointG1 // 前一有效置换后的公钥列表
	PrevRCommitForShuffle *bls12381.PointG1   // 前一有效置换随机数承诺
	// 抽签阶段
	PrevRCommitForDraw           *bls12381.PointG1 // 前一有效抽签随机数承诺
	UnenabledCommitteeQueue      *queue            // 抽签产生的委员会队列（未启用的）
	UnenabledSelfMemberMarkQueue *queue            // 自己所在的委员会队列（未启用的）
	// DPVSS阶段
	Threshold uint32 // DPVSS恢复门限
}

type CryptoSet struct {
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
	MemberPKList            []*bls12381.PointG1   // 多个PVSS对应的委员会公钥（公钥y = g^s，s为秘密），数量 1~SmallN-1
}

func inInitStage(height uint32, localStorage *MEVLocalStorage) bool {
	return height <= localStorage.BigN
}

func inShuffleStage(height uint32, localStorage *MEVLocalStorage) bool {
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

func isTheFirstShuffle(height uint32, localStorage *MEVLocalStorage) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= localStorage.BigN {
		return false
	}
	if height%localStorage.BigN == 1 {
		return true
	}
	return false
}

func isTheLastShuffle(height uint32, localStorage *MEVLocalStorage) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= localStorage.BigN {
		return false
	}
	if height%localStorage.BigN == localStorage.SmallN {
		return true
	}
	return false
}

func inDrawStage(height uint32, localStorage *MEVLocalStorage) bool {
	return height > localStorage.BigN+localStorage.SmallN
}

func inDPVSSStage(height uint32, localStorage *MEVLocalStorage) bool {
	return height > localStorage.BigN+localStorage.SmallN+1
}
func inWorkStage(height uint32, localStorage *MEVLocalStorage) bool {
	return height > localStorage.BigN+2*localStorage.SmallN
}

// checkReceivedBlock 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func checkReceivedBlock(height uint32, localStorage *MEVLocalStorage, receivedBlock *CryptoSet) bool {
	// 如果处于初始化阶段
	if inInitStage(height, localStorage) {
		// 检查srs的更新证明，如果验证失败，则丢弃
		prevSRSG1FirstElem := group1.One()
		if localStorage.LocalSRS != nil {
			localStorage.LocalSRS.G1PowerOf(1)
		}
		if !srs.Verify(receivedBlock.Srs, prevSRSG1FirstElem, receivedBlock.SrsUpdateProof) {
			fmt.Printf("[Height %d]:Verify SRS error\n", height)
			return false
		}
		return true
	}
	// 如果处于置换阶段，则验证置换
	if inShuffleStage(height, localStorage) {
		// 验证置换
		// 第一次置换
		if isTheFirstShuffle(height, localStorage) {
			// 获取前面BigN个区块的公钥
			localStorage.PrevShuffledPKList = getPrevNBlockPKList(height-localStorage.BigN, height-1)
		}
		// 如果验证失败，丢弃
		if !shuffle.Verify(localStorage.LocalSRS, localStorage.PrevShuffledPKList, localStorage.PrevRCommitForShuffle, receivedBlock.ShuffleProof) {
			return false
		}
	}
	// 如果处于抽签阶段，则验证抽签
	if inDrawStage(height, localStorage) {
		// 验证抽签
		// 如果验证失败，丢弃
		if !verifiable_draw.Verify(localStorage.LocalSRS, localStorage.BigN, localStorage.PrevShuffledPKList, localStorage.SmallN, localStorage.PrevRCommitForDraw, receivedBlock.DrawProof) {
			return false
		}
	}
	// 如果处于DPVSS阶段
	if inDPVSSStage(height, localStorage) {
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
func updateLocalRecords(height uint32, localStorage *MEVLocalStorage, receivedBlock *CryptoSet) {
	// 初始化阶段
	if inInitStage(height, localStorage) {
		// 记录最新srs
		localStorage.LocalSRS = receivedBlock.Srs
	}
	// 置换阶段
	if inShuffleStage(height, localStorage) {
		// 记录该置换公钥列表为最新置换公钥列表
		localStorage.PrevShuffledPKList = receivedBlock.ShuffleProof.SelectedPubKeys
		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
		localStorage.PrevRCommitForShuffle.Set(receivedBlock.ShuffleProof.RCommit)
		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
		if isTheLastShuffle(height, localStorage) {
			localStorage.PrevRCommitForDraw.Set(localStorage.PrevRCommitForShuffle)
		}
	}
	// 抽签阶段
	if inDrawStage(height, localStorage) {
		// 记录抽出来的委员会（公钥列表）
		localStorage.UnenabledCommitteeQueue.PushBack(&CommitteeMark{
			WorkHeight:   height + localStorage.SmallN,
			MemberPKList: receivedBlock.DrawProof.SelectedPubKeys,
			CommitteePK:  group1.Zero(),
		})
		// 对于每个自己区块的公私钥对获取抽签结果，是否被选中，如果被选中则获取自己的公钥
		// 获取自己的区块私钥(同一置换/抽签分组内的，即1~32，33~64...)
		maxHeight := ((height - localStorage.SmallN) >> 5) << 5
		minHeight := maxHeight - localStorage.SmallN + 1
		selfPrivKeyList := getSelfPrivKeyList(minHeight, maxHeight)
		for _, privKey := range selfPrivKeyList {
			isSelected, pk := verifiable_draw.IsSelected(privKey, receivedBlock.DrawProof.RCommit, receivedBlock.DrawProof.SelectedPubKeys)
			// 如果抽中，在自己所在的委员会队列创建新条目记录
			if isSelected {
				// test
				fmt.Printf("Mark committee you are in, current height: %d, committee work height: %d\n", height, height+localStorage.SmallN)
				localStorage.UnenabledSelfMemberMarkQueue.PushBack(&SelfMemberMark{
					WorkHeight:   height + localStorage.SmallN,
					SelfPubKey:   pk,
					SelfPrivKey:  privKey,
					AggrEncShare: mrpvss.NewEmptyEncShare(),
					Share:        bls12381.NewFr().Zero(),
				})
			}
		}
	}
	// DPVSS阶段
	if inDPVSSStage(height, localStorage) {
		pvssTimes := len(receivedBlock.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 聚合对应委员会的公钥
			for e := localStorage.UnenabledCommitteeQueue.Front(); e != nil; e = e.Next() {
				// 在未启用委员会队列中找到对应委员会（通过委员会的工作高度）
				mark := e.Value.(*CommitteeMark)
				if mark.WorkHeight == receivedBlock.CommitteeWorkHeightList[i] {
					// 聚合该委员会的公钥
					mark.CommitteePK = mrpvss.AggregateLeaderPK(mark.CommitteePK, receivedBlock.MemberPKList[i])
				}
			}
			// 如果自己是对应份额持有者(委员会工作高度相等)，则获取对应加密份额，聚合加密份额
			for e := localStorage.UnenabledSelfMemberMarkQueue.Front(); e != nil; e = e.Next() {
				mark := e.Value.(*SelfMemberMark)
				// 在未启用自己所在委员会的队列中找到对应委员会（通过委员会的工作高度）
				if mark.WorkHeight == receivedBlock.CommitteeWorkHeightList[i] {
					holderPKList := receivedBlock.HolderPKLists[i]
					encShares := receivedBlock.EncShareLists[i]
					holderNum := len(holderPKList)
					// 遍历查找自己对应的份额
					for j := 0; j < holderNum; j++ {
						// 在该PVSS中找到自己的对应的份额（通过公钥）
						if group1.Equal(mark.SelfPubKey, holderPKList[j]) {
							// 聚合加密份额
							mark.AggrEncShare = mrpvss.AggregateEncShares(mark.AggrEncShare, encShares[j])
						}
					}
					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
					if int64(mark.WorkHeight)-int64(height) == 1 {
						// 解密聚合加密份额
						mark.Share = mrpvss.DecryptAggregateShare(localStorage.G, mark.SelfPrivKey, mark.AggrEncShare, localStorage.SmallN-1)
					}
				}
			}
		}
	}
	// 委员会工作阶段
	if inWorkStage(height, localStorage) {
		// TODO 判断自己是否作为领导者进行工作
		if isSelfBlock(height) {
			// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
			if localStorage.UnenabledCommitteeQueue.Front() != nil {
				e := localStorage.UnenabledCommitteeQueue.Front()
				// 获取该委员会公钥
				committeeInfo := e.Value.(*CommitteeMark).CommitteePK
				// 删除该委员会
				localStorage.UnenabledCommitteeQueue.Remove(localStorage.UnenabledCommitteeQueue.Front())
				// TODO 传递该委员会信息给业务链（作为委员会领导者）
				handleCommitteeAsLeader(height, committeeInfo)
				return
			}
		}

		// 检查自己是否需要作为委员会成员工作
		// 遍历队列查找，如果自己该作为委员会成员执行业务处理，则解密聚合加密份额，并发送信号
		for e := localStorage.UnenabledSelfMemberMarkQueue.Front(); e != nil; e = e.Next() {
			if height == e.Value.(*SelfMemberMark).WorkHeight {
				// 从自己所属委员会队列中移除该委员会标记
				memberInfo := e.Value.(*SelfMemberMark)
				localStorage.UnenabledSelfMemberMarkQueue.Remove(e)
				// TODO 发送相关信号
				handleCommitteeAsMember(height, memberInfo)
				return
			}
		}
		// 作为用户，传递抗审查数据
		// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
		if localStorage.UnenabledCommitteeQueue.Front() != nil {
			e := localStorage.UnenabledCommitteeQueue.Front()
			// 获取该委员会公钥
			committeeInfo := e.Value.(*CommitteeMark).CommitteePK
			// 删除该委员会
			localStorage.UnenabledCommitteeQueue.Remove(localStorage.UnenabledCommitteeQueue.Front())
			// TODO 传递该委员会公钥给业务链(用于向该委员会发送交易)
			handleCommitteeAsUser(height, committeeInfo)
		}
	}
}

// height 要出块的区块高度
func uponMining(height uint32, localStorage *MEVLocalStorage) (*CryptoSet, error) {
	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height, localStorage) {
		// test
		fmt.Printf("in Init Stage, update SRS\n")
		// 更新 srs, 并生成证明
		var newSRS *srs.SRS
		var newSrsUpdateProof *schnorr_proof.SchnorrProof
		// 如果之前未生成SRS，新生成一个SRS
		if localStorage.LocalSRS == nil {
			newSRS, newSrsUpdateProof = srs.NewSRS(g1Degree, g2Degree)
		} else {
			r, _ := bls12381.NewFr().Rand(rand.Reader)
			newSRS, newSrsUpdateProof = localStorage.LocalSRS.Update(r)
		}
		// 更新后的SRS和更新证明写入区块
		return &CryptoSet{
			Srs:            newSRS,
			SrsUpdateProof: newSrsUpdateProof,
		}, nil
	}
	var err error
	var newShuffleProof *verifiable_draw.DrawProof
	var newDrawProof *verifiable_draw.DrawProof
	var committeeWorkHeightList []uint32
	var holderPKLists [][]*bls12381.PointG1
	var shareCommitsList [][]*bls12381.PointG1
	var coeffCommitsList [][]*bls12381.PointG1
	var encSharesList [][]*mrpvss.EncShare
	var committeePKList []*bls12381.PointG1
	// 如果处于置换阶段，则进行置换
	if inShuffleStage(height, localStorage) {
		// test
		fmt.Printf("in Shuffle Stage, shuffle\n")
		// 第一次置换
		if isTheFirstShuffle(height, localStorage) {
			// 获取前面BigN个区块的公钥
			localStorage.PrevShuffledPKList = getPrevNBlockPKList(height-localStorage.BigN, height-1)
		}
		newShuffleProof = shuffle.SimpleShuffle(localStorage.LocalSRS, localStorage.PrevShuffledPKList, localStorage.PrevRCommitForShuffle)
	}
	// 如果处于抽签阶段，则进行抽签
	if inDrawStage(height, localStorage) {
		// test
		fmt.Printf("in Draw Stage, draw\n")
		// 生成秘密向量用于抽签，向量元素范围为 [1, BigN]
		permutation, err := utils.GenRandomPermutation(localStorage.BigN)
		if err != nil {
			return nil, err
		}
		secretVector := permutation[:localStorage.SmallN]
		newDrawProof, err = verifiable_draw.Draw(localStorage.LocalSRS, localStorage.BigN, localStorage.PrevShuffledPKList, localStorage.SmallN, secretVector, localStorage.PrevRCommitForDraw)
		if err != nil {
			return nil, err
		}
	}
	// 如果处于PVSS阶段，则进行PVSS的分发份额
	if inDPVSSStage(height, localStorage) {
		// 计算向哪些委员会进行pvss(委员会index从1开始)
		pvssTimes := height - localStorage.BigN - localStorage.SmallN - 1
		if pvssTimes > localStorage.SmallN-1 {
			pvssTimes = localStorage.SmallN - 1
		}
		committeeWorkHeightList = make([]uint32, pvssTimes)       // 要PVSS的委员会的工作高度
		holderPKLists = make([][]*bls12381.PointG1, pvssTimes)    // 委员会成员（份额持有者）的公钥列表
		shareCommitsList = make([][]*bls12381.PointG1, pvssTimes) // 多个份额承诺列表
		coeffCommitsList = make([][]*bls12381.PointG1, pvssTimes) // 多个系数承诺列表
		encSharesList = make([][]*mrpvss.EncShare, pvssTimes)     // 多个加密份额列表
		committeePKList = make([]*bls12381.PointG1, pvssTimes)    // 委员会公钥列表
		i := uint32(0)
		// 遍历未启用的委员会，从第二个开始遍历，第一个已经完成PVSS（数量应该为pvssTimes）
		for e := localStorage.UnenabledCommitteeQueue.Front().Next(); e != nil; e = e.Next() {
			// 委员会的工作高度 height - pvssTimes + i + SmallN
			committeeWorkHeightList[i] = height - pvssTimes + i + localStorage.SmallN
			secret, _ := bls12381.NewFr().Rand(rand.Reader)
			holderPKLists[i] = e.Value.(*CommitteeMark).MemberPKList
			shareCommitsList[i], coeffCommitsList[i], encSharesList[i], committeePKList[i], err = mrpvss.EncShares(localStorage.G, localStorage.H, holderPKLists[i], secret, localStorage.Threshold)
			if err != nil {
				return nil, err
			}
			i++
		}
	}
	return &CryptoSet{
		Srs:                     nil,
		SrsUpdateProof:          nil,
		ShuffleProof:            newShuffleProof,
		DrawProof:               newDrawProof,
		CommitteeWorkHeightList: committeeWorkHeightList,
		HolderPKLists:           holderPKLists,
		ShareCommitLists:        shareCommitsList,
		CoeffCommitLists:        coeffCommitsList,
		EncShareLists:           encSharesList,
		MemberPKList:            committeePKList,
	}, nil
}

// TODO isSelfBlock 判断高度为 height 的区块是否是自己出块的
func isSelfBlock(height uint32) bool {
	return false
}

// TODO getSelfPrivKeyList 获取高度位于 [minHeight, maxHeight] 内所有自己出块的区块私钥
func getSelfPrivKeyList(minHeight, maxHeight uint32) []*bls12381.Fr {
	var ret []*bls12381.Fr
	return ret
}

// TODO getPrevNBlockPKList 获取高度位于 [minHeight, maxHeight] 内所有区块的公钥
func getPrevNBlockPKList(minHeight, maxHeight uint32) []*bls12381.PointG1 {
	ret := make([]*bls12381.PointG1, maxHeight-minHeight+1)
	return ret
}

// TODO handleCommitteeAsUser 作为用户传递该委员会信息给业务链(用于向该委员会发送交易)
func handleCommitteeAsUser(height uint32, pk *bls12381.PointG1) {

}

// TODO handleCommitteeAsMember 作为委员会成员传递该委员会信息给业务链(用于处理业务)
func handleCommitteeAsMember(height uint32, info *SelfMemberMark) {

}

// TODO handleCommitteeAsLeader 作为委员会领导者传递该委员会信息给业务链(用于处理业务)
func handleCommitteeAsLeader(height uint32, info *bls12381.PointG1) {

}
