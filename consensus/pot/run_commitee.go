package pot

import (
	"blockchain-crypto/blockchain_api"
	"blockchain-crypto/proof/schnorr_proof/bls12381"
	"blockchain-crypto/share/mrpvss/bls12381"
	"blockchain-crypto/shuffle"
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/utils"
	"blockchain-crypto/verifiable_draw"
	"container/list"
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"os"
	"strings"
)

var (
	BigN   = uint64(8)
	SmallN = uint64(4)
	// test
	g1Degree = uint32(1 << 7)
	// test
	g2Degree = uint32(1 << 7)
	mevLog   = logrus.New()
	mevLogs  = make(map[int64]*logrus.Logger)
	ids      = []int64{0, 1, 2, 3}
)

// 自定义Formatter
type MyFormatter struct {
	logrus.TextFormatter
}

// Format方法实现了logrus.Formatter接口
func (f *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 自定义输出格式
	timestamp := entry.Time.Format("15:04:05")
	return []byte(fmt.Sprintf("[%s] %s", timestamp, entry.Message)), nil
}
func init() {
	for _, id := range ids {
		// 为每个 ID 创建一个日志文件
		file, err := os.OpenFile(fmt.Sprintf("mev%v.log", id), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			mevLog.Fatalf("Failed to open log file: %v", err)
		}

		// 创建一个新的 Logrus 日志器实例，并将输出设置为上面创建的文件
		logger := &logrus.Logger{
			Out:       file,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.InfoLevel,
		}
		logger.SetFormatter(&MyFormatter{})
		mevLogs[id] = logger
	}
	// 启动 caulk+ gRPC
	err := utils.RunCaulkPlusGRPC()
	if err != nil {
		fmt.Printf("failed to start caulk-plus gRPC process: %v", err)
	}
}

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
	WorkHeight     uint64              // 委员会工作的PoT区块高度
	IsLeader       bool                // 自己是否为该委员会的领导者
	G              *bls12381.PointG1   // G^sk = pk, 用于 DPVSS
	CommitteePK    *bls12381.PointG1   // 委员会聚合公钥 y = g^s
	MemberPKList   []*bls12381.PointG1 // 份额持有者的公钥列表，用于PVSS分发份额时查找委员会成员
	ShareCommits   []*bls12381.PointG1 // 聚合的份额承诺列表
	SelfMemberList []*DPVSSMember      // 每个委员会成员的DPVSS相关信息，数量为自己出的块中作为委员会成员的数量
}

// DPVSSMember 记录委员会成员相关信息
type DPVSSMember struct {
	Index        uint32            // 下标
	SelfSK       *bls12381.Fr      // 自己用于份额分发的私钥, 用于解密自己的份额
	SelfPK       *bls12381.PointG1 // 自己用于份额分发的公钥, 用于查找自己的份额
	AggrEncShare *mrpvss.EncShare  // 聚合加密份额
	Share        *bls12381.Fr      // 解密份额（聚合的）
}

func (c *CommitteeMark) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("CommitteeMark {WorkHeight: %d, ", c.WorkHeight))
	builder.WriteString(fmt.Sprintf("IsLeader: %v, ", c.IsLeader))
	builder.WriteString(fmt.Sprintf("G: %v, ", c.G[0][0]))
	builder.WriteString(fmt.Sprintf("CommitteePK: %v, ", c.CommitteePK[0][0]))
	builder.WriteString("MemberPKList: [")
	for _, pk := range c.MemberPKList {
		builder.WriteString(fmt.Sprintf("%v, ", pk[0][0]))
	}
	builder.WriteString("], ")
	builder.WriteString("ShareCommits: [")
	for _, commit := range c.ShareCommits {
		builder.WriteString(fmt.Sprintf("%v, ", commit[0][0]))
	}
	builder.WriteString("], ")

	builder.WriteString("SelfMemberList: [")
	for _, member := range c.SelfMemberList {
		builder.WriteString(fmt.Sprintf("%v, ", member))
	}
	builder.WriteString("]}")

	return builder.String()
}

func (m *DPVSSMember) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("DPVSSMember{Index: %d, ", m.Index))
	builder.WriteString(fmt.Sprintf("SelfSK: %v, ", m.SelfSK[0]))
	builder.WriteString(fmt.Sprintf("SelfPK: %v, ", m.SelfPK[0][0]))
	builder.WriteString(fmt.Sprintf("AggrEncShare: %v, ", m.AggrEncShare))
	builder.WriteString(fmt.Sprintf("Share: %v}", m.Share[0]))
	return builder.String()
}

func (c *CommitteeMark) DeepCopy() *CommitteeMark {
	group1 := bls12381.NewG1()
	ret := &CommitteeMark{
		WorkHeight: c.WorkHeight,
		IsLeader:   c.IsLeader,
	}
	if c.G != nil {
		ret.G = group1.New().Set(c.G)
	}
	if c.CommitteePK != nil {
		ret.CommitteePK = group1.New().Set(c.CommitteePK)
	}
	if c.MemberPKList != nil {
		ret.MemberPKList = make([]*bls12381.PointG1, len(c.MemberPKList))
		for i := 0; i < len(c.MemberPKList); i++ {
			if c.MemberPKList[i] != nil {
				ret.MemberPKList[i] = group1.New().Set(c.MemberPKList[i])
			}
		}
	}
	if c.ShareCommits != nil {
		ret.ShareCommits = make([]*bls12381.PointG1, len(c.ShareCommits))
		for i := 0; i < len(c.ShareCommits); i++ {
			if c.ShareCommits[i] != nil {
				ret.ShareCommits[i] = group1.New().Set(c.ShareCommits[i])
			}
		}
	}
	if c.SelfMemberList != nil {
		ret.SelfMemberList = make([]*DPVSSMember, len(c.SelfMemberList))
		for i := 0; i < len(c.SelfMemberList); i++ {
			ret.SelfMemberList[i] = c.SelfMemberList[i].DeepCopy()
		}
	}
	return ret
}

func (d *DPVSSMember) DeepCopy() *DPVSSMember {
	group1 := bls12381.NewG1()
	ret := &DPVSSMember{
		Index: d.Index,
	}
	if d.SelfSK != nil {
		ret.SelfSK = bls12381.NewFr().Set(d.SelfSK)
	}
	if d.SelfPK != nil {
		ret.SelfPK = group1.New().Set(d.SelfPK)
	}
	if d.AggrEncShare != nil {
		ret.AggrEncShare = d.AggrEncShare.DeepCopy()
	}
	if d.Share != nil {
		ret.Share = bls12381.NewFr().Set(d.Share)
	}
	return ret
}

func BackUpCommitteeQueue(q *Queue) *Queue {
	backup := new(Queue)
	for e := q.Front(); e != nil; e = e.Next() {
		backup.PushBack(e.Value.(*CommitteeMark).DeepCopy())
	}
	return backup
}

func GetMarkByWorkHeight(q *Queue, workHeight uint64) *CommitteeMark {
	for e := q.Front(); e != nil; e = e.Next() {
		if e.Value.(*CommitteeMark).WorkHeight == workHeight {
			return e.Value.(*CommitteeMark)
		}
	}
	return nil
}

func (w *Worker) GetPeerQueue() chan *types.Block {
	return w.peerMsgQueue
}

func (w *Worker) CommitteeUpdate(height uint64) {
	//
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
	//	// potsignal := &simpleWhirly.PoTSignal{
	//	// 	Epoch:               int64(epoch),
	//	// 	Proof:               nil,
	//	// 	ID:                  0,
	//	// 	LeaderPublicAddress: committee[0],
	//	// 	Committee:           committee,
	//	// 	SelfPublicAddress:   selfaddress,
	//	// 	CryptoElements:      nil,
	//	// }
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
	// //if epoch > 10 && w.ID == 1 {
	// //	block, err := w.chainReader.GetByHeight(epoch - 1)
	// //	if err != nil {
	// //		return
	// //	}
	// //	header := block.GetHeader()
	// //	fill, err := os.OpenFile("difficulty", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	// //	if err != nil {
	// //		fmt.Println(err)
	// //	}
	// //	_, err = fill.WriteString(fmt.Sprintf("%d\Commitees", header.Difficulty.Int64()))
	// //	if err != nil {
	// //		fmt.Println(err)
	// //	}
	// //	fill.Close()
	// //}
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
	PrevRCommitForDraw *bls12381.PointG1 // 前一有效抽签随机数承诺
	CommitteeMarkQueue *Queue            // （未启用的）委员会队列
	// MemberMarkQueue    *Queue            // 自己（作为成员）所在的（未启用的）委员会队列
	// DPVSS阶段
	Threshold uint32 // DPVSS恢复门限
}

func (c *CryptoSet) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CryptoSet {H: %v, ", c.H[0][0]))
	sb.WriteString(fmt.Sprintf("SRS.g1[1]: %v, ", c.LocalSRS.G1PowerOf(1)[0][0]))
	sb.WriteString(fmt.Sprintf("PrevShuffledPKList: %v, ", PrintPointG1List(c.PrevShuffledPKList)))
	sb.WriteString(fmt.Sprintf("PrevRCommitForShuffle: %v, ", c.PrevRCommitForShuffle[0][0]))
	sb.WriteString(fmt.Sprintf("PrevRCommitForDraw: %v, ", c.PrevRCommitForDraw[0][0]))
	sb.WriteString("CommitteeMarkQueue: [")
	for e := c.CommitteeMarkQueue.Front(); e != nil; e = e.Next() {
		mark := e.Value.(*CommitteeMark)
		sb.WriteString(fmt.Sprintf("%v, ", mark))
	}
	return sb.String()
}

// Backup the CryptoSet at height `height`, `c` should be a new CryptoSet
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
	if inInitStage(height, c.BigN) && c.LocalSRS != nil {
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
	if c.CommitteeMarkQueue != nil {
		backup.CommitteeMarkQueue = BackUpCommitteeQueue(c.CommitteeMarkQueue)
	}
	return backup
}

// Restore the CryptoSet using `backup` at height `height`, `c` should be an existing CryptoSet, can not be nil or new
func (c *CryptoSet) Restore(height uint64, backup *CryptoSet) *CryptoSet {
	group1 := bls12381.NewG1()

	c.BigN = backup.BigN
	c.SmallN = backup.SmallN
	c.Threshold = backup.Threshold

	if backup.G != nil {
		c.G = group1.New().Set(backup.G)
	}
	if backup.H != nil {
		c.H = group1.New().Set(backup.H)
	}
	if backup.PrevRCommitForShuffle != nil {
		c.PrevRCommitForShuffle = group1.New().Set(backup.PrevRCommitForShuffle)
	}
	if backup.PrevRCommitForDraw != nil {
		c.PrevRCommitForDraw = group1.New().Set(backup.PrevRCommitForDraw)
	}
	if inInitStage(height, backup.BigN) && backup.LocalSRS != nil {
		c.LocalSRS = new(srs.SRS).Set(backup.LocalSRS)
	}
	if backup.PrevShuffledPKList != nil {
		c.PrevShuffledPKList = make([]*bls12381.PointG1, len(backup.PrevShuffledPKList))
		for i, p := range backup.PrevShuffledPKList {
			if p != nil {
				c.PrevShuffledPKList[i] = new(bls12381.PointG1).Set(p)
			}
		}
	}
	if backup.CommitteeMarkQueue != nil {
		c.CommitteeMarkQueue = BackUpCommitteeQueue(backup.CommitteeMarkQueue)
	}
	return backup
}

func inInitStage(height, N uint64) bool {
	return height <= N
}
func inShuffleStage(height, N, n uint64) bool {
	// 置换阶段（简单置换） [1+kN, Commitees+kN] k>0
	if height <= N {
		return false
	}
	relativeHeight := height % N
	return relativeHeight > 0 && relativeHeight <= n
}
func isTheLastShuffle(height, N, n uint64) bool {
	// 置换阶段（简单置换） [1+kN, Commitees+kN] k>0
	if height <= N {
		return false
	}
	return height%N == n
}

func inDrawStage(height, N, n uint64) bool {
	return height > N+n
}
func isTheFirstShuffle(height, N uint64) bool {
	// 置换阶段（简单置换） [1+kN, SmallN+kN] k>0
	if height <= N {
		return false
	}
	return height%N == 1
}

func inDPVSSStage(height uint64, N, n uint64) bool {
	return height > N+n+1
}
func inWorkStage(height, N, n uint64) bool {
	return height > N+2*n
}

// TODO isSelfBlock 判断高度为 height 的区块是否是自己出块的

func (w *Worker) isSelfBlock(block *types.Block) bool {
	ok, _ := w.TryFindKey(crypto.Convert(block.Hash()))
	return ok
}

func getStageList(height uint64, N uint64, n uint64) []string {
	var stages []string
	if inInitStage(height, N) {
		stages = append(stages, "Init")
	}
	if inShuffleStage(height, N, n) {
		stages = append(stages, "Shuffle")
	}
	if inDrawStage(height, N, n) {
		stages = append(stages, "Draw")
	}
	if inDPVSSStage(height, N, n) {
		stages = append(stages, "DPVSS")
	}
	if inWorkStage(height, N, n) {
		stages = append(stages, "Work")
	}
	return stages
}

func PrintPointG1List(list []*bls12381.PointG1) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, s := range list {
		// 对于第一个元素，不需要前置逗号
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%v", s[0][0]))
	}
	sb.WriteString("]")
	return sb.String()
}

func PrintEncShareList(list []*mrpvss.EncShare) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, s := range list {
		// 对于第一个元素，不需要前置逗号
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%v", s))
	}
	sb.WriteString("]")
	return sb.String()
}

// getBlockRange 返回h所在的区间, h ∈ [kN+n+1, (k+1)N+n] 则返回 ((k-1)N+1, kN)
func getBlockRange(h, N, n uint64) (uint64, uint64) {
	// 计算k的值
	k := (h - n - 1) / N
	if k < 1 {
		return 0, 0
	}
	start := (k-1)*N + 1
	end := k * N
	return start, end
}

// VerifyCryptoSet 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func (w *Worker) VerifyCryptoSet(height uint64, block *types.Block) bool {
	if block.GetHeader().Height == 0 {
		return true
	}
	cryptoSet := w.CryptoSet
	N := cryptoSet.BigN
	n := cryptoSet.SmallN
	cryptoElement := block.GetHeader().CryptoElement
	mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify, stages: %v\n", height, w.ID, getStageList(height, N, n))
	mevLogs[w.ID].Printf("\t\tlocal crypto set: %v\n", cryptoSet)
	mevLogs[w.ID].Printf("\t\treceived block: %x, srs.g1[1]: %v\n", block.Header.Hashes[:4], cryptoElement.SRS.G1PowerOf(1)[0][0])

	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height, N) {
		err := srs.Verify(cryptoElement.SRS, cryptoSet.LocalSRS.G1PowerOf(1), cryptoElement.SrsUpdateProof)
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-Init | error: %v\n", height, w.ID, err)
			return false
		}
		return true
	}
	// 如果处于置换阶段，则验证置换
	if inShuffleStage(height, N, n) {
		// 验证置换
		// 第一次置换
		if isTheFirstShuffle(height, N) {
			// 获取前面BigN个区块的公钥
			cryptoSet.PrevShuffledPKList = w.getPrevNBlockPKList(height-N, height-1)
		}
		// 如果验证失败，丢弃
		err := shuffle.Verify(cryptoSet.LocalSRS, cryptoSet.PrevShuffledPKList, cryptoSet.PrevRCommitForShuffle, cryptoElement.ShuffleProof, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-Shuffle | error: %v\n", height, w.ID, err)
			fmt.Printf("[Block %2v|Node %v] Verify-Shuffle | error: %v\n", height, w.ID, err)
			return false
		}
	}
	// 如果处于抽签阶段，则验证抽签
	if inDrawStage(height, N, n) {
		// 验证抽签
		// 如果验证失败，丢弃
		err := verifiable_draw.Verify(cryptoSet.LocalSRS, uint32(N), cryptoSet.PrevShuffledPKList, uint32(n), cryptoSet.PrevRCommitForDraw, cryptoElement.DrawProof, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-Draw: Verify Draw error: %v\n", height, w.ID, err)
			fmt.Printf("[Block %2v|Node %v] Verify-Draw: Verify Draw error: %v\n", height, w.ID, err)
			return false
		}
	}
	// 如果处于DPVSS阶段
	if inDPVSSStage(height, N, n) {
		// 验证PVSS加密份额
		pvssTimes := len(cryptoElement.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 如果不存在对应委员会，丢弃
			mark := GetMarkByWorkHeight(cryptoSet.CommitteeMarkQueue, cryptoElement.CommitteeWorkHeightList[i])
			if mark == nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-DPVSS | error: no corresponding committee, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Verify-DPVSS | error: no corresponding committee, pvss index = %v\n", height, w.ID, i)
				return false
			}
			// 如果成员公钥列表对不上，丢弃
			group1 := bls12381.NewG1()
			for j := 0; j < len(mark.MemberPKList); j++ {
				if !group1.Equal(mark.MemberPKList[j], cryptoElement.HolderPKLists[i][j]) {
					mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-DPVSS | error: incorrect member pk list, pvss index = %v\n", height, w.ID, i)
					fmt.Printf("[Block %2v|Node %v] Verify-DPVSS | error: incorrect member pk list, pvss index = %v\n", height, w.ID, i)
					// return false
					break
				}
			}
			// 如果验证失败，丢弃
			if !mrpvss.VerifyEncShares(uint32(n), cryptoSet.Threshold, mark.G, cryptoSet.H, cryptoElement.HolderPKLists[i], cryptoElement.ShareCommitLists[i], cryptoElement.CoeffCommitLists[i], cryptoElement.EncShareLists[i]) {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify-DPVSS | error: failed to verify encShares, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Verify-DPVSS: failed to verify encShares, pvss index = %v\n", height, w.ID, i)
				return false
			}
		}
	}
	return true
}

func (w *Worker) UpdateLocalCryptoSetByBlock(height uint64, block *types.Block) error {
	group1 := bls12381.NewG1()
	cryptoElement := block.GetHeader().CryptoElement
	cryptoSet := w.CryptoSet
	N := cryptoSet.BigN
	n := cryptoSet.SmallN
	mevLogs[w.ID].Printf("[Block %2v|Node %v] Update | bolck hash: %x parent hash: %x, stage: %v\n", height, w.ID, block.Header.Hashes[:4], block.Header.ParentHash[:4], getStageList(height, N, n))
	mevLogs[w.ID].Printf("\t\tlocal crypto set: %v\n", cryptoSet)
	mevLogs[w.ID].Printf("\t\treceived block: %x, srs.g1[1]: %v\n", block.Header.Hashes[:4], cryptoElement.SRS.G1PowerOf(1)[0][0])
	// 初始化阶段
	if inInitStage(height, N) {
		// 记录最新srs
		cryptoSet.LocalSRS = cryptoElement.SRS
		// 如果处于最后一次初始化阶段，保存SRS为文件（用于Caulk+）,并启动Caulk+进程
		if !inInitStage(height+1, N) {
			// 更新 H
			cryptoSet.H = group1.New().Set(cryptoSet.LocalSRS.G1PowerOf(123))
			// 保存SRS至srs.binary文件
			var err error
			err = cryptoSet.LocalSRS.ToBinaryFile(w.ID)
			if err != nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-Init: failed to save srs%v\n", height, w.ID, err)
				return fmt.Errorf("[Block %2v|Node %v] Update-Init: failed to save srs%v\n", height, w.ID, err)
			}
		}
	}
	// 置换阶段
	if inShuffleStage(height, N, n) {
		// 记录该置换公钥列表为最新置换公钥列表
		cryptoSet.PrevShuffledPKList = cryptoElement.ShuffleProof.SelectedPubKeys
		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
		cryptoSet.PrevRCommitForShuffle.Set(cryptoElement.ShuffleProof.RCommit)
		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
		if isTheLastShuffle(height, N, n) {
			cryptoSet.PrevRCommitForDraw.Set(cryptoSet.PrevRCommitForShuffle)
		}
	}
	// 抽签阶段
	if inDrawStage(height, N, n) {
		// 初始化 CommitteeMark
		mark := &CommitteeMark{
			WorkHeight:     height + n,
			IsLeader:       false,
			G:              group1.New().Set(cryptoElement.DrawProof.RCommit),
			CommitteePK:    group1.Zero(),
			MemberPKList:   cryptoElement.DrawProof.SelectedPubKeys,
			ShareCommits:   nil,
			SelfMemberList: []*DPVSSMember{},
		}

		// 如果自己是出块人
		if w.isSelfBlock(block) {
			mark.IsLeader = true
		}

		// 查找自己出的块是否被选为委员会成员
		// 对于每个自己区块的公私钥对获取抽签结果，是否被选中，如果被选中则获取自己的公钥
		// 获取自己的区块私钥(同一置换/抽签分组内的，即1~32，33~64...)
		startHeight, endHeight := getBlockRange(height, N, n)
		mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-Draw | need pklist from %v to %v \n", height, w.ID, startHeight, endHeight)
		selfPrivKeyList := w.getSelfPrivKeyList(startHeight, endHeight)
		for _, sk := range selfPrivKeyList {
			isSelected, index, pk := verifiable_draw.IsSelected(sk, cryptoElement.DrawProof.RCommit, cryptoElement.DrawProof.SelectedPubKeys)
			// 如果抽中，在自己所在的委员会队列创建新条目记录
			if isSelected {
				mark.SelfMemberList = append(mark.SelfMemberList, &DPVSSMember{
					Index:        index,
					SelfSK:       sk,
					SelfPK:       pk,
					AggrEncShare: mrpvss.NewEmptyEncShare(),
					Share:        bls12381.NewFr().Zero(),
				})
				// mevLogs[w.ID].Printf("pk= sk ^ rCommit: %v", group1.Equal(pk, group1.MulScalar(group1.New(), cryptoElement.DrawProof.RCommit, sk)))
			}
		}
		// 如果不只是用户，则记录份额承诺
		if mark.IsLeader || len(mark.SelfMemberList) > 0 {
			mark.ShareCommits = make([]*bls12381.PointG1, n)
			for i := uint64(0); i < n; i++ {
				mark.ShareCommits[i] = group1.New()
			}
		}
		// 记录抽出来的委员会
		cryptoSet.CommitteeMarkQueue.PushBack(mark)
		// mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-Draw | drawn mark: %v, \n", height, w.ID, mark)
		// fmt.Printf("[Block %2v|Node %v] Update-Draw | drawn mark: %v\n", height, w.ID, mark)
	}

	// DPVSS阶段
	if inDPVSSStage(height, N, n) {
		pvssTimes := len(cryptoElement.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 寻找对应委员会
			mark := GetMarkByWorkHeight(cryptoSet.CommitteeMarkQueue, cryptoElement.CommitteeWorkHeightList[i])
			// mevLogs[w.ID].Printf("                                  get mark which work height = %v, %v\n", cryptoElement.CommitteeWorkHeightList[i], mark)
			// mevLogs[w.ID].Printf("                                  get cryptoElement: CommitteePK = %v\n", cryptoElement.CommitteePKList[i][0][0])
			// mevLogs[w.ID].Printf("                                  HolderPKLists[%v] = %v\n", cryptoElement.CommitteeWorkHeightList[i], PrintPointG1List(cryptoElement.HolderPKLists[i]))
			// mevLogs[w.ID].Printf("                                  ShareCommitLists[%v] = %v\n", cryptoElement.CommitteeWorkHeightList[i], PrintPointG1List(cryptoElement.ShareCommitLists[i]))
			// mevLogs[w.ID].Printf("                                  CoeffCommitLists[%v] = %v\n", cryptoElement.CommitteeWorkHeightList[i], PrintPointG1List(cryptoElement.CoeffCommitLists[i]))
			// mevLogs[w.ID].Printf("                                  EncShareLists[%v] = %v\n", cryptoElement.CommitteeWorkHeightList[i], PrintEncShareList(cryptoElement.EncShareLists[i]))
			// fmt.Printf("                                  get mark which work height = %v, %v\n", cryptoElement.CommitteeWorkHeightList[i], mark)
			// should checked in verify function
			if mark == nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-DPVSS: cannot find committee to aggregate pvss, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Update-DPVSS: cannot find committee to aggregate pvss, pvss index = %v\n", height, w.ID, i)
			} else {
				// 聚合对应委员会的公钥
				mark.CommitteePK = mrpvss.AggregateCommitteePK(mark.CommitteePK, cryptoElement.CommitteePKList[i])
				// mevLogs[w.ID].Printf("                                  AggregateCommitteePK = %v\n", mark.CommitteePK[0][0])
				// fmt.Printf("                                  AggregateCommitteePK = %v\n", mark.CommitteePK[0][0])
				// 如果自己是委员会的领导者或成员，则聚合份额承诺
				if mark.IsLeader || len(mark.SelfMemberList) > 0 {
					shareCommitList, err := mrpvss.AggregateShareCommitList(mark.ShareCommits, cryptoElement.ShareCommitLists[i])
					if err != nil {
						mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-DPVSS: failed to AggregateShareCommitList: %v\n", height, w.ID, err)
						return fmt.Errorf("[Block %2v|Node %v] Update-DPVSS: failed to AggregateShareCommitList: %v\n", height, w.ID, err)
					}
					mark.ShareCommits = shareCommitList
				}
				// 对于每个是委员会成员的自己的节点，聚合份额承诺
				for j := 0; j < len(mark.SelfMemberList); j++ {
					holderPKList := cryptoElement.HolderPKLists[i]
					encShares := cryptoElement.EncShareLists[i]
					holderNum := len(holderPKList)
					member := mark.SelfMemberList[j]
					for k := 0; k < holderNum; k++ {
						if group1.Equal(member.SelfPK, holderPKList[k]) {
							// 聚合加密份额
							member.AggrEncShare = mrpvss.AggregateEncShares(member.AggrEncShare, encShares[k])

							tmp := mrpvss.DecryptAggregateShare(mark.G, member.SelfSK, encShares[k], uint32(n-1))
							if tmp == nil {
								mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-DPVSS: AggregateEncShares: cannot find decrypt encShare, selfPK = %v, selfSK = %v, encShare = %v\n", height, w.ID, member.SelfPK, member.SelfSK, encShares[k])
							}
							// tmp = mrpvss.DecryptAggregateShare(mark.G, member.SelfSK, member.AggrEncShare, uint32(n-1))
							// if tmp == nil {
							// 	mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-DPVSS: AggregateEncShares: cannot find decrypt AggrEncShare, selfSK = %v,aggrEncShare = %v\n", height, w.ID, member.SelfSK[0], member.AggrEncShare)
							// }
						}
					}
					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
					if mark.WorkHeight-height == 1 {
						// 解密聚合加密份额
						member.Share = mrpvss.DecryptAggregateShare(mark.G, member.SelfSK, member.AggrEncShare, uint32(n-1))
						if member.Share == nil {
							mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS: DecryptAggregateShare: cannot find decrypt encShare, selfPK = %v, selfSK = %v, encShare = %v\n", height, w.ID, member.SelfPK, member.SelfSK, member.AggrEncShare)
						}
					}
				}
			}

		}
	}
	// mevLogs[w.ID].Printf("[Block %2v|Node %v] Update | CommitteeMarkQueue: [\n", height, w.ID)
	// fmt.Printf("[Block %2v|Node %v] Update | CommitteeMarkQueue: \n", height, w.ID)
	// for e := cryptoSet.CommitteeMarkQueue.Front(); e != nil; e = e.Next() {
	// 	mark := e.Value.(*CommitteeMark)
	// 	mevLogs[w.ID].Printf("                                   %v\n", mark)
	// 	fmt.Printf("                                   %v\n", mark)
	// }
	// mevLogs[w.ID].Printf("]\n")
	// fmt.Printf("]\n")
	// 委员会更新阶段
	if inWorkStage(height, N, n) {
		// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
		// 获取该委员会
		e := cryptoSet.CommitteeMarkQueue.Front()
		if e == nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Update-Work | err: no committee in queue", height, w.ID)
			return fmt.Errorf("[Block %2v|Node %v] Update-Work | err: no committee in queue", height, w.ID)
		}
		committeeMark := e.Value.(*CommitteeMark)
		// 删除该委员会
		cryptoSet.CommitteeMarkQueue.Remove(cryptoSet.CommitteeMarkQueue.Front())

		var leaderInput *blockchain_api.CommitteeConfig
		var memberInputs []*blockchain_api.CommitteeConfig
		var userInput *blockchain_api.CommitteeConfig
		// 如果自己是领导者
		if committeeMark.IsLeader {
			leaderInput = &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
				DPVSSConfig: blockchain_api.DPVSSConfig{
					Index:        0,
					ShareCommits: make([][]byte, n),
					Share:        nil,
					IsLeader:     true,
				},
			}
			for j := uint64(0); j < n; j++ {
				leaderInput.DPVSSConfig.ShareCommits[j] = group1.ToCompressed(committeeMark.ShareCommits[j])
			}
		}
		// 如果有委员会成员节点
		for i := 0; i < len(committeeMark.SelfMemberList); i++ {
			member := committeeMark.SelfMemberList[i]
			memberInput := &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
				DPVSSConfig: blockchain_api.DPVSSConfig{
					Index:        member.Index,
					ShareCommits: make([][]byte, n),
					Share:        member.Share.ToBytes(),
					IsLeader:     false,
				},
			}
			for j := uint64(0); j < n; j++ {
				memberInput.DPVSSConfig.ShareCommits[j] = group1.ToCompressed(committeeMark.ShareCommits[j])
			}
			memberInputs = append(memberInputs, memberInput)
		}
		// 作为用户
		if leaderInput == nil && memberInputs == nil {
			userInput = &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
			}
		}
		sendConsensusNodeInputs(w.ID, height, leaderInput, memberInputs, userInput)
	}
	w.CryptoSetMap[crypto.Convert(block.GetHeader().Hash())] = w.CryptoSet.Backup(block.Header.Height)
	return nil
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

func (w *Worker) GenerateCryptoSetFromLocal(height uint64) (*types.CryptoElement, error) {
	// 如果处于初始化阶段（参数生成阶段）
	cryptoSet := w.CryptoSet
	N := cryptoSet.BigN
	n := cryptoSet.SmallN
	mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining | stage: %v\n", height, w.ID, getStageList(height, N, n))
	fmt.Printf("[Block %2v|Node %v] Mining | stage: %v\n", height, w.ID, getStageList(height, N, n))
	if inInitStage(height, N) {
		// 更新 srs, 并生成证明
		var newSRS *srs.SRS
		var newSrsUpdateProof *schnorr_proof.SchnorrProof
		// // 如果之前未生成SRS，新生成一个SRS
		// if cryptoSet.LocalSRS == nil {
		// 	newSRS, newSrsUpdateProof = srs.NewSRS(g1Degree, g2Degree)
		// } else {
		r, _ := bls12381.NewFr().Rand(rand.Reader)
		newSRS, newSrsUpdateProof = cryptoSet.LocalSRS.Update(r)
		// }
		// 更新后的SRS和更新证明写入区块
		return &types.CryptoElement{
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
	if inShuffleStage(height, N, n) {
		if isTheFirstShuffle(height, N) {
			// 获取前面BigN个区块的公钥
			cryptoSet.PrevShuffledPKList = w.getPrevNBlockPKList(height-N, height-1)
			// mevLogs[w.ID].Printf("[Mining]: Shuffle | Node %v, Block %2v|len of \n", w.ID, height,len(cryptoSet.PrevShuffledPKList))
		}
		newShuffleProof, err = shuffle.SimpleShuffle(cryptoSet.LocalSRS, cryptoSet.PrevShuffledPKList, cryptoSet.PrevRCommitForShuffle, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining-Shuffle | failed to shuffle: %v\n", height, w.ID, err)
			return nil, fmt.Errorf("[Block %2v|Node %v] Mining-Shuffle | failed to shuffle: %v\n", height, w.ID, err)
		}
	}
	// 如果处于抽签阶段，则进行抽签
	if inDrawStage(height, N, n) {
		// 生成秘密向量用于抽签，向量元素范围为 [1, BigN]
		permutation, err := utils.GenRandomPermutation(uint32(N))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining-Draw | failed to generate permutation: %v\n", height, w.ID, err)
			return nil, fmt.Errorf("[Block %2v|Node %v] Mining-Draw | failed to generate permutation: %v\n", height, w.ID, err)
		}
		secretVector := permutation[:n]
		newDrawProof, err = verifiable_draw.Draw(cryptoSet.LocalSRS, uint32(N), cryptoSet.PrevShuffledPKList, uint32(n), secretVector, cryptoSet.PrevRCommitForDraw, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining-Draw | failed to draw: %v\n", height, w.ID, err)
			return nil, fmt.Errorf("[Block %2v|Node %v] Mining-Draw | failed to draw: %v\n", height, w.ID, err)
		}
	}
	// 如果处于PVSS阶段，则进行PVSS的分发份额
	if inDPVSSStage(height, N, n) {
		// 计算向哪些委员会进行pvss(委员会index从1开始)
		pvssTimes := n - 1
		if height < N+2*n {
			pvssTimes = height - N - n - 1
		}
		// 要PVSS的委员会的工作高度
		committeeWorkHeightList = make([]uint64, pvssTimes)
		for i := uint64(0); i < pvssTimes; i++ {
			// 委员会的工作高度 height - pvssTimes + i + n
			committeeWorkHeightList[i] = height - pvssTimes + i + n
		}
		holderPKLists = make([][]*bls12381.PointG1, pvssTimes)
		shareCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		coeffCommitsList = make([][]*bls12381.PointG1, pvssTimes)
		encSharesList = make([][]*mrpvss.EncShare, pvssTimes)
		committeePKList = make([]*bls12381.PointG1, pvssTimes)
		// TODO
		for i := uint64(0); i < pvssTimes; i++ {
			// 创建秘密
			secret := bls12381.NewFr()
			for {
				secret.Rand(rand.Reader)
				if !secret.IsZero() {
					break
				}
			}
			mark := GetMarkByWorkHeight(cryptoSet.CommitteeMarkQueue, committeeWorkHeightList[i])
			if mark == nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining-DPVSS | cannot find committee workHeight = %v\n", height, w.ID, committeeWorkHeightList[i])
				return nil, fmt.Errorf("[Block %2v|Node %v] Mining-DPVSS | cannot find committee workHeight = %v", height, w.ID, committeeWorkHeightList[i])
			}
			holderPKLists[i] = mark.MemberPKList
			shareCommitsList[i], coeffCommitsList[i], encSharesList[i], committeePKList[i], err = mrpvss.EncShares(mark.G, cryptoSet.H, holderPKLists[i], secret, cryptoSet.Threshold)
			if err != nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Mining-DPVSS | deal encShares failed: %v", height, w.ID, err)
				return nil, fmt.Errorf("[Block %2v|Node %v] Mining-DPVSS | deal encShares failed: %v", height, w.ID, err)
			}
		}
	}
	return &types.CryptoElement{
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

// TODO
func sendConsensusNodeInputs(
	id int64,
	height uint64,
	leaderInput *blockchain_api.CommitteeConfig,
	memberInputs []*blockchain_api.CommitteeConfig,
	userInput *blockchain_api.CommitteeConfig,
) {
	mevLogs[id].Printf("[Block %2v|Node %v] Send: Start send configs\n", height, id)
	if leaderInput != nil {
		mevLogs[id].Printf("[Block %2v|Node %v] Send: Contain a leader\n", height, id)
		// mevLogs[id].Printf("[Block %2v] Send: Contain a leader: %v\n", height, leaderInput)
	}
	mevLogs[id].Printf("[Block %2v|Node %v] Send: Contain %v members:\n", height, id, len(memberInputs))
	// for index, member := range memberInputs {
	// 	mevLogs[id].Printf("[Block %2v] Send: \tmember %v: %v\n", height, index, member)
	// }
	if leaderInput == nil && (memberInputs == nil || len(memberInputs) == 0) {
		// mevLogs[id].Printf("[Block %2v] Send: Only a user: %v\n", height, userInput)
		mevLogs[id].Printf("[Block %2v|Node %v] Send: Only a user\n", height, id)
	}
}

// getSelfPrivKeyList 获取高度位于 [minHeight, maxHeight] 内所有自己出块的区块私钥
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
			mevLogs[w.ID].Printf("\t\tget sk from branch at height %v, block hash: %v, sk: %v\n", i, block.Header.Hashes, fr)
		}
	}
	return ret
}

// getPrevNBlockPKList 获取高度位于 [minHeight, maxHeight] 内所有区块的公钥
func (w *Worker) getPrevNBlockPKList(minHeight, maxHeight uint64) []*bls12381.PointG1 {
	ret := make([]*bls12381.PointG1, 0)
	group1 := bls12381.NewG1()
	for i := minHeight; i <= maxHeight; i++ {
		block, err := w.chainReader.GetByHeight(i)
		if err != nil {
			return nil
		}
		pub := block.GetHeader().PublicKey
		point, err := group1.FromBytes(pub)
		if err != nil {
			return nil
		}
		ret = append(ret, point)
	}
	return ret
}

func (w *Worker) getPrevNBlockPKListByBranch(minHeight, maxHeight uint64, branch []*types.Block) []*bls12381.PointG1 {
	fmt.Printf("need pklist from %d to %d \n", minHeight, maxHeight)
	group1 := bls12381.NewG1()
	n := len(branch)
	k := 1
	if n < 1 {
		return nil
	}
	branchstartheight := branch[n-k].GetHeader().Height
	branchendheight := branch[0].GetHeader().Height

	ret := make([]*bls12381.PointG1, 0)
	for i := minHeight; i <= maxHeight; i++ {
		if i >= branchstartheight && i < branchendheight {
			block := branch[branchendheight-i]
			fmt.Printf("get height %d pk from branch at branch block height %d\n", i, block.GetHeader().Height)
			pub := block.GetHeader().PublicKey
			point, err := group1.FromBytes(pub)
			if err != nil {
				return nil
			}
			ret = append(ret, point)
			k++
		} else {
			block, err := w.chainReader.GetByHeight(i)
			if err != nil {
				return nil
			}
			pub := block.GetHeader().PublicKey
			point, err := group1.FromBytes(pub)
			if err != nil {
				return nil
			}
			ret = append(ret, point)
		}
	}
	return ret
}

// getSelfPrivKeyListByBranch 获取高度位于 [minHeight, maxHeight] 内所有自己出块的区块私钥
func (w *Worker) getSelfPrivKeyListByBranch(minHeight, maxHeight uint64, branch []*types.Block) []*bls12381.Fr {
	fmt.Printf("need pklist from %d to %d \n", minHeight, maxHeight)
	var ret []*bls12381.Fr
	n := len(branch)
	k := 1
	branchheight := branch[n-k].GetHeader().Height
	branchendheight := branch[0].GetHeader().Height

	for i := minHeight; i <= maxHeight; i++ {
		if i >= branchheight && i <= branchendheight {
			block := branch[branchendheight-i]
			fmt.Printf("get height %d pk from branch at branch block height %d\n", i, block.GetHeader().Height)
			flag, priv := w.TryFindKey(crypto.Convert(block.Hash()))
			if flag {
				fr := bls12381.NewFr().FromBytes(priv)
				ret = append(ret, fr)
				mevLogs[w.ID].Printf("\t\tget sk from branch at height %v, block height at branch: %v, block hash: %v, sk: %v\n", i, block.GetHeader().Height, block.Header.Hashes, fr)
			}
			k++
		} else {
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
	}
	return ret
}

// VerifyCryptoSetByBranch 当收到区块，height 为该区块的高度, 返回该区块是否验证通过
func (w *Worker) VerifyCryptoSetByBranch(height uint64, block *types.Block, branch []*types.Block) bool {
	if block.GetHeader().Height == 0 {
		return true
	}
	cryptoSet := w.CryptoSet
	N := cryptoSet.BigN
	n := cryptoSet.SmallN
	cryptoElement := block.GetHeader().CryptoElement

	mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR] | stage: %v\n", height, w.ID, getStageList(height, N, n))
	mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR] | stages: %v\n", height, w.ID, getStageList(height, N, n))
	mevLogs[w.ID].Printf("\t\tlocal crypto set: %v\n", cryptoSet)
	mevLogs[w.ID].Printf("\t\treceived block: %x, srs.g1[1]: %v\n", block.Header.Hashes[:4], cryptoElement.SRS.G1PowerOf(1)[0][0])

	// 如果处于初始化阶段（参数生成阶段）
	if inInitStage(height, N) {
		// 检查srs的更新证明，如果验证失败，则丢弃
		err := srs.Verify(cryptoElement.SRS, cryptoSet.LocalSRS.G1PowerOf(1), cryptoElement.SrsUpdateProof)
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-Init | error: %v\n", height, w.ID, err)
			fmt.Printf("[Block %2v|Node %v] Verify[CR]-Init | error: %v\n", height, w.ID, err)
			return false
		}
		return true
	}
	// 如果处于置换阶段，则验证置换
	if inShuffleStage(height, N, n) {
		// 验证置换
		// 第一次置换
		if isTheFirstShuffle(height, N) {
			// 获取前面BigN个区块的公钥

			cryptoSet.PrevShuffledPKList = w.getPrevNBlockPKListByBranch(height-N, height-1, branch)
		}
		// 如果验证失败，丢弃
		err := shuffle.Verify(cryptoSet.LocalSRS, cryptoSet.PrevShuffledPKList, cryptoSet.PrevRCommitForShuffle, cryptoElement.ShuffleProof, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-Shuffle | error: %v\n", height, w.ID, err)
			fmt.Printf("[Block %2v|Node %v] Verify[CR]-Shuffle | Verify Shuffle error: %v\n", height, w.ID, err)
			return false
		}
	}
	// 如果处于抽签阶段，则验证抽签
	if inDrawStage(height, N, n) {
		// 验证抽签
		// 如果验证失败，丢弃
		err := verifiable_draw.Verify(cryptoSet.LocalSRS, uint32(N), cryptoSet.PrevShuffledPKList, uint32(n), cryptoSet.PrevRCommitForDraw, cryptoElement.DrawProof, uint64(w.ID))
		if err != nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-Draw: Verify Draw error: %v\n", height, w.ID, err)
			fmt.Printf("[Block %2v|Node %v] Verify[CR]-Draw: Verify Draw error: %v\n", height, w.ID, err)
			return false
		}
	}
	// 如果处于DPVSS阶段
	if inDPVSSStage(height, N, n) {
		// 验证PVSS加密份额
		pvssTimes := len(cryptoElement.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 如果不存在对应委员会，丢弃
			mark := GetMarkByWorkHeight(cryptoSet.CommitteeMarkQueue, cryptoElement.CommitteeWorkHeightList[i])
			if mark == nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: no corresponding committee, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: no corresponding committee, pvss index = %v\n", height, w.ID, i)
				return false
			}
			// 如果成员公钥列表对不上，丢弃
			group1 := bls12381.NewG1()
			for j := 0; j < len(mark.MemberPKList); j++ {
				if !group1.Equal(mark.MemberPKList[j], cryptoElement.HolderPKLists[i][j]) {
					mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: incorrect member pk list, pvss index = %v\n", height, w.ID, i)
					fmt.Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: incorrect member pk list, pvss index = %v\n", height, w.ID, i)
					// return false
					break
				}
			}
			// 如果验证失败，丢弃
			if !mrpvss.VerifyEncShares(uint32(n), cryptoSet.Threshold, mark.G, cryptoSet.H, cryptoElement.HolderPKLists[i], cryptoElement.ShareCommitLists[i], cryptoElement.CoeffCommitLists[i], cryptoElement.EncShareLists[i]) {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: failed to verify encShares, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Verify[CR]-DPVSS | error: failed to verify encShares, pvss index = %v\n", height, w.ID, i)
				return false
			}
		}
	}
	return true
}

func (w *Worker) UpdateLocalCryptoSetByBranch(height uint64, receivedBlock *types.Block, branch []*types.Block) error {
	group1 := bls12381.NewG1()
	cryptoElement := receivedBlock.GetHeader().CryptoElement
	cryptoSet := w.CryptoSet
	N := cryptoSet.BigN
	n := cryptoSet.SmallN
	mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR] | bolck hash: %x parent hash: %x, stage: %v\n", height, w.ID, receivedBlock.Header.Hashes[:4], receivedBlock.Header.ParentHash[:4], getStageList(height, N, n))
	mevLogs[w.ID].Printf("\t\tlocal crypto set: %v\n", cryptoSet)
	mevLogs[w.ID].Printf("\t\treceived block: %x, srs.g1[1]: %v\n", receivedBlock.Header.Hashes[:4], cryptoElement.SRS.G1PowerOf(1)[0][0])
	// 初始化阶段
	if inInitStage(height, N) {
		// 记录最新srs
		if height != 0 {
			cryptoSet.LocalSRS = cryptoElement.SRS
		}
		// 如果处于最后一次初始化阶段，保存SRS为文件（用于Caulk+）,并启动Caulk+进程
		if !inInitStage(height+1, N) {
			// 更新 H
			cryptoSet.H = group1.New().Set(cryptoSet.LocalSRS.G1PowerOf(1))
			// 保存SRS至srs.binary文件
			var err error
			err = cryptoSet.LocalSRS.ToBinaryFile(w.ID)
			if err != nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-Init | failed to save srs%v\n", height, w.ID, err)
				return fmt.Errorf("[Block %2v|Node %v] Update[CR]-Init | failed to save srs%v\n", height, w.ID, err)
			}
		}
	}
	// 置换阶段
	if inShuffleStage(height, N, n) {
		// 记录该置换公钥列表为最新置换公钥列表
		cryptoSet.PrevShuffledPKList = cryptoElement.ShuffleProof.SelectedPubKeys
		// 记录该置换随机数承诺为最新置换随机数承诺 localStorage.PrevRCommitForShuffle
		cryptoSet.PrevRCommitForShuffle.Set(cryptoElement.ShuffleProof.RCommit)
		// 如果处于置换阶段的最后一次置换（height%localStorage.BigN==SmallN），更新抽签随机数承诺
		//     因为先进行置换验证，再进行抽签验证，抽签使用的是同周期最后一次置换输出的
		//     顺序： 验证最后置换（收到区块）->进行抽签（出新区块）->验证抽签（收到区块）
		//     抽签阶段不改变 localStorage.PrevRCommitForDraw
		if isTheLastShuffle(height, N, n) {
			cryptoSet.PrevRCommitForDraw.Set(cryptoSet.PrevRCommitForShuffle)
		}
	}
	// 抽签阶段
	if inDrawStage(height, N, n) {
		// 初始化 CommitteeMark
		mark := &CommitteeMark{
			WorkHeight:     height + n,
			IsLeader:       false,
			G:              group1.New().Set(cryptoElement.DrawProof.RCommit),
			CommitteePK:    group1.Zero(),
			MemberPKList:   cryptoElement.DrawProof.SelectedPubKeys,
			ShareCommits:   nil,
			SelfMemberList: []*DPVSSMember{},
		}

		// 如果自己是出块人
		if w.isSelfBlock(receivedBlock) {
			mark.IsLeader = true
		}

		// 查找自己出的块是否被选为委员会成员
		// 对于每个自己区块的公私钥对获取抽签结果，是否被选中，如果被选中则获取自己的公钥
		// 获取自己的区块私钥(同一置换/抽签分组内的，即1~32，33~64...)
		startHeight, endHeight := getBlockRange(height, N, n)
		mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-Draw | need pklist from %v to %v \n", height, w.ID, startHeight, endHeight)
		selfPrivKeyList := w.getSelfPrivKeyListByBranch(startHeight, endHeight, branch)
		for _, sk := range selfPrivKeyList {
			isSelected, index, pk := verifiable_draw.IsSelected(sk, cryptoElement.DrawProof.RCommit, cryptoElement.DrawProof.SelectedPubKeys)
			// 如果抽中，在自己所在的委员会队列创建新条目记录
			if isSelected {
				mark.SelfMemberList = append(mark.SelfMemberList, &DPVSSMember{
					Index:        index,
					SelfPK:       pk,
					SelfSK:       sk,
					AggrEncShare: mrpvss.NewEmptyEncShare(),
					Share:        bls12381.NewFr().Zero(),
				})
			}
		}
		// 如果不只是用户，则记录份额承诺
		if mark.IsLeader || len(mark.SelfMemberList) > 0 {
			mark.ShareCommits = make([]*bls12381.PointG1, n)
			for i := uint64(0); i < n; i++ {
				mark.ShareCommits[i] = group1.New()
			}
		}
		// 记录抽出来的委员会
		cryptoSet.CommitteeMarkQueue.PushBack(mark)
		// mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-Draw | drawn mark: %v, \n", height, w.ID, mark)
		// fmt.Printf("[Block %2v|Node %v] Update[CR]-Draw | drawn mark: %v\n", height, w.ID, mark)
	}

	// DPVSS阶段
	if inDPVSSStage(height, N, n) {
		pvssTimes := len(cryptoElement.CommitteeWorkHeightList)
		for i := 0; i < pvssTimes; i++ {
			// 寻找对应委员会
			mark := GetMarkByWorkHeight(cryptoSet.CommitteeMarkQueue, cryptoElement.CommitteeWorkHeightList[i])
			// should checked in verify function
			if mark == nil {
				mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS: cannot find committee to aggregate pvss, pvss index = %v\n", height, w.ID, i)
				fmt.Printf("[Block %2v|Node %v] Update[CR]-DPVSS: cannot find committee to aggregate pvss, pvss index = %v\n", height, w.ID, i)
			} else {
				// 聚合对应委员会的公钥
				mark.CommitteePK = mrpvss.AggregateCommitteePK(mark.CommitteePK, cryptoElement.CommitteePKList[i])
				// 如果自己是委员会的领导者或成员，则聚合份额承诺
				if mark.IsLeader || len(mark.SelfMemberList) > 0 {
					shareCommitList, err := mrpvss.AggregateShareCommitList(mark.ShareCommits, cryptoElement.ShareCommitLists[i])
					if err != nil {
						mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS: failed to AggregateShareCommitList: %v\n", height, w.ID, err)
						return fmt.Errorf("[Block %2v|Node %v] Update[CR]-DPVSS: failed to AggregateShareCommitList: %v\n", height, w.ID, err)
					}
					mark.ShareCommits = shareCommitList
				}
				// 对于每个是委员会成员的自己的节点，聚合份额承诺
				for j := 0; j < len(mark.SelfMemberList); j++ {
					holderPKList := cryptoElement.HolderPKLists[i]
					encShares := cryptoElement.EncShareLists[i]
					holderNum := len(holderPKList)
					member := mark.SelfMemberList[j]
					for k := 0; k < holderNum; k++ {
						if group1.Equal(member.SelfPK, holderPKList[k]) {
							// 聚合加密份额
							member.AggrEncShare = mrpvss.AggregateEncShares(member.AggrEncShare, encShares[k])

							tmp := mrpvss.DecryptAggregateShare(mark.G, member.SelfSK, encShares[k], uint32(n-1))
							if tmp == nil {
								mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS: AggregateEncShares: cannot find decrypt encShare, selfPK = %v, selfSK = %v, encShare = %v\n", height, w.ID, member.SelfPK, member.SelfSK, encShares[k])
							}
						}
					}
					// 如果当前是该委员会的最后一次PVSS，后续没有新的PVSS份额，则解密聚合份额
					if mark.WorkHeight-height == 1 {
						// 解密聚合加密份额
						member.Share = mrpvss.DecryptAggregateShare(mark.G, member.SelfSK, member.AggrEncShare, uint32(n-1))
						if member.Share == nil {
							mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS: DecryptAggregateShare: cannot find decrypt encShare, selfPK = %v, selfSK = %v, encShare = %v\n", height, w.ID, member.SelfPK, member.SelfSK, member.AggrEncShare)
						}
					}
				}
			}
		}
	}
	// mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-DPVSS | CommitteeMarkQueue: [\n", height, w.ID)
	// fmt.Printf("[Block %2v|Node %v] Update[CR]-DPVSS | CommitteeMarkQueue: [\n", height, w.ID)
	// for e := cryptoSet.CommitteeMarkQueue.Front(); e != nil; e = e.Next() {
	// 	mark := e.Value.(*CommitteeMark)
	// 	mevLogs[w.ID].Printf("                                   %v\n", mark)
	// 	fmt.Printf("                                   %v\n", mark)
	// }
	// mevLogs[w.ID].Printf("]\n")
	// fmt.Printf("]\n")

	// 委员会更新阶段
	// TODO
	if inWorkStage(height, N, n) {
		// 更新未工作的委员会队列，删除第一个元素(该元素对应委员会在这个epoch工作)
		// 获取该委员会
		e := cryptoSet.CommitteeMarkQueue.Front()
		if e == nil {
			mevLogs[w.ID].Printf("[Block %2v|Node %v] Update[CR]-Work| err: no committee in queue", height, w.ID)
			return fmt.Errorf("[Block %2v|Node %v] Update[CR]-Work| err: no committee in queue", height, w.ID)
		}
		committeeMark := e.Value.(*CommitteeMark)
		// 删除该委员会
		cryptoSet.CommitteeMarkQueue.Remove(cryptoSet.CommitteeMarkQueue.Front())

		var leaderInput *blockchain_api.CommitteeConfig
		var memberInputs []*blockchain_api.CommitteeConfig
		var userInput *blockchain_api.CommitteeConfig
		// 如果自己是领导者
		if committeeMark.IsLeader {
			leaderInput = &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
				DPVSSConfig: blockchain_api.DPVSSConfig{
					Index:        0,
					ShareCommits: make([][]byte, n),
					Share:        nil,
					IsLeader:     true,
				},
			}
			for j := uint64(0); j < n; j++ {
				leaderInput.DPVSSConfig.ShareCommits[j] = group1.ToCompressed(committeeMark.ShareCommits[j])
			}
		}
		// 如果有委员会成员节点
		for i := 0; i < len(committeeMark.SelfMemberList); i++ {
			member := committeeMark.SelfMemberList[i]
			memberInput := &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
				DPVSSConfig: blockchain_api.DPVSSConfig{
					Index:        member.Index,
					ShareCommits: make([][]byte, n),
					Share:        member.Share.ToBytes(),
					IsLeader:     false,
				},
			}
			for j := uint64(0); j < n; j++ {
				memberInput.DPVSSConfig.ShareCommits[j] = group1.ToCompressed(committeeMark.ShareCommits[j])
			}
			memberInputs = append(memberInputs, memberInput)
		}
		// 作为用户
		if leaderInput == nil && memberInputs == nil {
			userInput = &blockchain_api.CommitteeConfig{
				H:           group1.ToCompressed(cryptoSet.H),
				CommitteePK: group1.ToCompressed(committeeMark.CommitteePK),
			}
		}
		sendConsensusNodeInputs(w.ID, height, leaderInput, memberInputs, userInput)
	}
	w.CryptoSetMap[crypto.Convert(receivedBlock.GetHeader().Hash())] = w.CryptoSet.Backup(receivedBlock.Header.Height)
	return nil
}
