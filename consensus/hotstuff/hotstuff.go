package hotstuff

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/niclabs/tcrsa"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

// common hotstuff func defined in the paper
type HotStuff interface {
	Msg(msgType pb.MsgType, node *pb.WhirlyBlock, qc *pb.QuorumCert) *pb.Msg
	VoteMsg(msgType pb.MsgType, node *pb.WhirlyBlock, qc *pb.QuorumCert, justify []byte) *pb.Msg
	CreateLeaf(parentHash []byte, txs []types.RawTransaction, justify *pb.QuorumCert) *pb.WhirlyBlock
	QC(msgType pb.MsgType, sig tcrsa.Signature, blockHash []byte) *pb.QuorumCert
	MatchingMsg(msg *pb.Msg, msgType pb.MsgType) bool
	MatchingQC(qc *pb.QuorumCert, msgType pb.MsgType) bool
	SafeNode(node *pb.WhirlyBlock, qc *pb.QuorumCert) bool
	GetMsgByteEntrance() chan<- []byte
	GetRequestEntrance() chan<- *pb.Request
	GetSelfInfo() *config.ReplicaInfo
	Stop()
	GetConsensusID() int64
	GetWeight(nid int64) float64
	GetMaxAdversaryWeight() float64
}

type HotStuffImpl struct {
	ID              int64
	ConsensusID     int64
	BlockStorage    types.WhirlyBlockStorage
	View            *View
	Config          *config.ConsensusConfig
	TimeChan        *utils.Timer
	BatchTimeChan   *utils.Timer
	CurExec         *CurProposal
	MemPool         *types.MemPool
	HighQC          *pb.QuorumCert
	PrepareQC       *pb.QuorumCert // highQC
	PreCommitQC     *pb.QuorumCert // lockQC
	CommitQC        *pb.QuorumCert
	MsgByteEntrance chan []byte // receive msg
	RequestEntrance chan *pb.Request
	p2pAdaptor      p2p.P2PAdaptor
	Log             *logrus.Entry
	Executor        executor.Executor

	// control items
	Wg     *sync.WaitGroup
	Closed chan []byte
}

func (hs *HotStuffImpl) Init(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) {
	hs.ID = id
	hs.ConsensusID = cid
	cfg.F = (len(cfg.Nodes) - 1) / 3
	hs.Config = cfg
	hs.Executor = exec
	hs.p2pAdaptor = p2pAdaptor
	hs.Log = log.WithField("cid", cid)

	hs.MsgByteEntrance = make(chan []byte, 10)
	hs.RequestEntrance = make(chan *pb.Request, 10)

	if p2pAdaptor != nil {
		// p2pAdaptor.SetReceiver(hs)
		p2pAdaptor.SetReceiver(hs.GetMsgByteEntrance())
		p2pAdaptor.Subscribe([]byte(hs.Config.Topic))
	} else {
		hs.Log.Warn("p2p is nil, just for testing")
	}

	hs.MemPool = types.NewMemPool()
	hs.Log.Trace("[HOTSTUFF] Init block storage")
	hs.BlockStorage = types.NewBlockStorageImpl(strconv.Itoa(int(cid)) + "-" + strconv.Itoa(int(id)))
	hs.Wg = new(sync.WaitGroup)
	hs.Closed = make(chan []byte)
}

func (hs *HotStuffImpl) GetConsensusID() int64 {
	return hs.ConsensusID
}

func (hs *HotStuffImpl) DecodeMsgByte(msgByte []byte) (*pb.Msg, error) {
	msg := new(pb.Msg)
	err := proto.Unmarshal(msgByte, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type CurProposal struct {
	Node          *pb.WhirlyBlock
	DocumentHash  []byte
	PrepareVote   []*tcrsa.SigShare
	PreCommitVote []*tcrsa.SigShare
	CommitVote    []*tcrsa.SigShare
	HighQC        []*pb.QuorumCert
}

func NewCurProposal() *CurProposal {
	return &CurProposal{
		Node:          nil,
		DocumentHash:  nil,
		PrepareVote:   make([]*tcrsa.SigShare, 0),
		PreCommitVote: make([]*tcrsa.SigShare, 0),
		CommitVote:    make([]*tcrsa.SigShare, 0),
		HighQC:        make([]*pb.QuorumCert, 0),
	}
}

type View struct {
	ViewNum      uint64 // view number
	Primary      int64  // the leader's id
	ViewChanging bool
}

func NewView(viewNum uint64, primary int64) *View {
	return &View{
		ViewNum:      viewNum,
		Primary:      primary,
		ViewChanging: false,
	}
}

func (h *HotStuffImpl) Msg(msgType pb.MsgType, node *pb.WhirlyBlock, qc *pb.QuorumCert) *pb.Msg {
	msg := &pb.Msg{}
	switch msgType {
	case pb.MsgType_PREPARE:
		msg.Payload = &pb.Msg_Prepare{Prepare: &pb.Prepare{
			CurProposal: node,
			HighQC:      qc,
			ViewNum:     h.View.ViewNum,
		}}
	case pb.MsgType_PRECOMMIT:
		msg.Payload = &pb.Msg_PreCommit{PreCommit: &pb.PreCommit{PrepareQC: qc, ViewNum: h.View.ViewNum}}
	case pb.MsgType_COMMIT:
		msg.Payload = &pb.Msg_Commit{Commit: &pb.Commit{PreCommitQC: qc, ViewNum: h.View.ViewNum}}
	case pb.MsgType_NEWVIEW:
		msg.Payload = &pb.Msg_NewView{NewView: &pb.NewView{PrepareQC: qc, ViewNum: h.View.ViewNum}}
	case pb.MsgType_DECIDE:
		msg.Payload = &pb.Msg_Decide{Decide: &pb.Decide{CommitQC: qc, ViewNum: h.View.ViewNum}}
	}
	return msg
}

func (h *HotStuffImpl) VoteMsg(msgType pb.MsgType, node *pb.WhirlyBlock, qc *pb.QuorumCert, justify []byte) *pb.Msg {
	msg := &pb.Msg{}
	switch msgType {
	case pb.MsgType_PREPARE_VOTE:
		msg.Payload = &pb.Msg_PrepareVote{PrepareVote: &pb.PrepareVote{
			BlockHash:  node.Hash,
			Qc:         qc,
			PartialSig: justify,
			ViewNum:    h.View.ViewNum,
		}}
	case pb.MsgType_PRECOMMIT_VOTE:
		msg.Payload = &pb.Msg_PreCommitVote{PreCommitVote: &pb.PreCommitVote{
			BlockHash:  node.Hash,
			Qc:         qc,
			PartialSig: justify,
			ViewNum:    h.View.ViewNum,
		}}
	case pb.MsgType_COMMIT_VOTE:
		msg.Payload = &pb.Msg_CommitVote{CommitVote: &pb.CommitVote{
			BlockHash:  node.Hash,
			Qc:         qc,
			PartialSig: justify,
			ViewNum:    h.View.ViewNum,
		}}
	}
	return msg
}

func (h *HotStuffImpl) CreateLeaf(parentHash []byte, txs []types.RawTransaction, justify *pb.QuorumCert) *pb.WhirlyBlock {
	b := &pb.WhirlyBlock{
		ParentHash: parentHash,
		Hash:       nil,
		Height:     h.View.ViewNum,
		Txs:        types.RawTxArrayToBytes(txs),
		Justify:    justify,
	}

	b.Hash = types.Hash(b)
	return b
}

func (h *HotStuffImpl) QC(msgType pb.MsgType, sig tcrsa.Signature, blockHash []byte) *pb.QuorumCert {
	marshal, _ := json.Marshal(sig)
	return &pb.QuorumCert{
		BlockHash: blockHash,
		Type:      msgType,
		ViewNum:   h.View.ViewNum,
		Signature: marshal,
	}
}

func (h *HotStuffImpl) MatchingMsg(msg *pb.Msg, msgType pb.MsgType) bool {
	switch msgType {
	case pb.MsgType_PREPARE:
		return msg.GetPrepare() != nil && msg.GetPrepare().ViewNum == h.View.ViewNum
	case pb.MsgType_PREPARE_VOTE:
		return msg.GetPrepareVote() != nil && msg.GetPrepareVote().ViewNum == h.View.ViewNum
	case pb.MsgType_PRECOMMIT:
		return msg.GetPreCommit() != nil && msg.GetPreCommit().ViewNum == h.View.ViewNum
	case pb.MsgType_PRECOMMIT_VOTE:
		return msg.GetPreCommitVote() != nil && msg.GetPreCommitVote().ViewNum == h.View.ViewNum
	case pb.MsgType_COMMIT:
		return msg.GetCommit() != nil && msg.GetCommit().ViewNum == h.View.ViewNum
	case pb.MsgType_COMMIT_VOTE:
		return msg.GetCommitVote() != nil && msg.GetCommitVote().ViewNum == h.View.ViewNum
	case pb.MsgType_NEWVIEW:
		return msg.GetNewView() != nil && msg.GetNewView().ViewNum == h.View.ViewNum
	}
	return false
}

func (h *HotStuffImpl) MatchingQC(qc *pb.QuorumCert, msgType pb.MsgType) bool {
	return qc.Type == msgType && qc.ViewNum == h.View.ViewNum
}

func (h *HotStuffImpl) SafeNode(node *pb.WhirlyBlock, qc *pb.QuorumCert) bool {
	return bytes.Equal(node.ParentHash, h.PreCommitQC.BlockHash) || //safety rule
		qc.ViewNum > h.PreCommitQC.ViewNum // liveness rule
}

func (h *HotStuffImpl) GetMsgByteEntrance() chan<- []byte {
	return h.MsgByteEntrance
}

func (h *HotStuffImpl) GetRequestEntrance() chan<- *pb.Request {
	return h.RequestEntrance
}

func (h *HotStuffImpl) Stop() {
	h.Log.Info("stopping consensus")
	close(h.Closed)
	h.Wg.Wait()
	h.BlockStorage.Close()
	// _ = os.RemoveAll("dbfile/node" + strconv.Itoa(int(h.ID)))
}

// GetLeader get the leader replica in view
func (h *HotStuffImpl) GetLeader() int64 {
	id := int64(h.View.ViewNum) % int64(len(h.Config.Nodes))
	return id
}

func (h *HotStuffImpl) GetSelfInfo() *config.ReplicaInfo {
	self := &config.ReplicaInfo{}
	for _, info := range h.Config.Nodes {
		if info.ID == h.ID {
			self = info
			break
		}
	}
	return self
}

func (h *HotStuffImpl) GetNetworkInfo() map[int64]string {
	networkInfo := make(map[int64]string)
	for _, info := range h.Config.Nodes {
		if info.ID == h.ID {
			continue
		}
		networkInfo[info.ID] = info.Address
	}
	return networkInfo
}

func (h *HotStuffImpl) Broadcast(msg *pb.Msg) error {
	if h.p2pAdaptor == nil {
		h.Log.Warn("p2pAdaptor nil")
		return nil
	}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)

	for _, node := range h.Config.Nodes {
		if node.ID == h.ID {
			continue
		}
		err := h.p2pAdaptor.Unicast(node.Address, msgByte, h.ConsensusID, []byte("consensus"))
		if err != nil {
			h.Log.WithError(err).Warn("send msg failed")
			return err
		}
	}

	// return h.p2pAdaptor.Broadcast(msgByte, h.ConsensusID, []byte("consensus"))
	return nil
}

func (h *HotStuffImpl) Unicast(address string, msg *pb.Msg) error {
	if h.p2pAdaptor == nil {
		h.Log.Warn("p2pAdaptor nil")
		return nil
	}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)
	return h.p2pAdaptor.Unicast(address, msgByte, h.ConsensusID, []byte("consensus"))
}

func (h *HotStuffImpl) ProcessProposal(b *pb.WhirlyBlock, p []byte) {
	h.Executor.CommitBlock(b, p, h.ConsensusID)
	// for _, tx := range txs {
	// 	h.Executor.CommitTx(tx, h.ConsensusID)
	// 	// msg := &pb.Msg{Payload: &pb.Msg_Reply{Reply: &pb.Reply{Tx: tx, Receipt: []byte(result)}}}
	// 	// err := h.Unicast("localhost:9999", msg)
	// 	// if err != nil {
	// 	// 	fmt.Println(err.Error())
	// 	// }
	// }
	h.MemPool.Remove(types.RawTxArrayFromBytes(b.Txs))
}

// GenerateGenesisBlock returns genesis block
func GenerateGenesisBlock() *pb.WhirlyBlock {
	genesisBlock := &pb.WhirlyBlock{
		ParentHash: nil,
		Hash:       nil,
		Height:     0,
		Txs:        nil,
		Justify:    nil,
	}
	hash := types.Hash(genesisBlock)
	genesisBlock.Hash = hash
	genesisBlock.Committed = true
	return genesisBlock
}

func (hs *HotStuffImpl) GetWeight(nid int64) float64 {
	if nid < int64(len(hs.Config.Nodes)) {
		return 1.0 / float64(len(hs.Config.Nodes))
	}
	return 0.0
}

func (hs *HotStuffImpl) GetMaxAdversaryWeight() float64 {
	return 1.0 / 3.0
}

func (hs *HotStuffImpl) UpdateExternalStatus(status model.ExternalStatus) {
	return
}

func (hs *HotStuffImpl) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	return
}

func (hs *HotStuffImpl) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	return
}
