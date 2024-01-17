package whirlyUtilities

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/niclabs/tcrsa"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

// common whirlyUtilities func defined in the paper
type WhirlyUtilities interface {
	ProposalMsg(block *pb.WhirlyBlock, qc *pb.QuorumCert, proof *pb.SimpleWhirlyProof, epoch int64) *pb.WhirlyMsg
	VoteMsg(blockView uint64, blockHash []byte, flag bool, qc *pb.QuorumCert, partSig []byte, proof *pb.SimpleWhirlyProof, epoch int64) *pb.WhirlyMsg
	NewViewMsg(qc *pb.QuorumCert, viewNum uint64) *pb.WhirlyMsg
	NewLeaderNotifyMsg(epoch int64, proof []byte) *pb.WhirlyMsg
	NewLeaderEchoMsg(leader int64, block *pb.WhirlyBlock, proof *pb.SimpleWhirlyProof, epoch int64, vHeghit uint64) *pb.WhirlyMsg
	PingMsg() *pb.WhirlyMsg
	CreateLeaf(parentHash []byte, viewNum uint64, txs []types.RawTransaction, justify *pb.QuorumCert) *pb.WhirlyBlock
	QC(view uint64, sig tcrsa.Signature, blockHash []byte) *pb.QuorumCert
	GetMsgByteEntrance() chan<- []byte
	GetRequestEntrance() chan<- *pb.Request
	GetSelfInfo() *config.ReplicaInfo
	GetConsensusID() int64
	GetP2pAdaptorType() string
}

type WhirlyUtilitiesImpl struct {
	ID              int64
	PeerId          string
	ConsensusID     int64
	BlockStorage    types.WhirlyBlockStorage
	View            *View
	Config          *config.ConsensusConfig
	TimeChan        *utils.Timer
	MemPool         *types.MemPool
	MsgByteEntrance chan []byte // receive msg
	RequestEntrance chan *pb.Request
	p2pAdaptor      p2p.P2PAdaptor
	Log             *logrus.Entry
	Executor        executor.Executor
	Topic           string
	Weight          int64
}

func (wu *WhirlyUtilitiesImpl) Init(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) {
	wu.ID = id
	wu.PeerId = p2pAdaptor.GetPeerID()
	wu.ConsensusID = cid
	wu.Config = cfg
	wu.Executor = exec
	wu.p2pAdaptor = p2pAdaptor
	wu.Log = log.WithField("consensus id", cid)
	wu.MsgByteEntrance = make(chan []byte, 10)
	wu.RequestEntrance = make(chan *pb.Request, 10)
	wu.MemPool = types.NewMemPool()
	// wu.Log.Debugf("[HOTSTUFF] Init block storage")
	newPeerId1 := strings.Replace(wu.PeerId, ".", "", -1)
	newPeerId2 := strings.Replace(newPeerId1, ":", "", -1)
	wu.BlockStorage = types.NewBlockStorageImpl(strconv.Itoa(int(cid)) + "-" + newPeerId2)

	// Set receiver
	//p2pAdaptor.SetReceiver(wu.GetMsgByteEntrance())
	wu.Topic = cfg.Topic
	// Subscribe topic
	//err := p2pAdaptor.Subscribe([]byte(wu.Topic))
	//if err != nil {
	//	wu.Log.Error("Subscribe error: ", err.Error())
	//	return
	//}
	wu.Log.Info("Joined to topic: ", wu.Topic)
}

func (wu *WhirlyUtilitiesImpl) InitForLocalTest(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) {
	wu.ID = id
	wu.PeerId = p2pAdaptor.GetPeerID()
	wu.ConsensusID = cid
	wu.Config = cfg
	wu.Executor = exec
	wu.p2pAdaptor = p2pAdaptor
	wu.Log = log.WithField("consensus id", cid)
	wu.MsgByteEntrance = make(chan []byte, 10)
	wu.RequestEntrance = make(chan *pb.Request, 10)
	wu.MemPool = types.NewMemPool()
	// wu.Log.Debugf("[HOTSTUFF] Init block storage")
	newPeerId1 := strings.Replace(wu.PeerId, ".", "", -1)
	newPeerId2 := strings.Replace(newPeerId1, ":", "", -1)
	wu.BlockStorage = types.NewBlockStorageImpl(strconv.Itoa(int(cid)) + "-" + newPeerId2)

	// Set receiver
	p2pAdaptor.SetReceiver(wu.GetMsgByteEntrance())
	wu.Topic = cfg.Topic
	// Subscribe topic
	err := p2pAdaptor.Subscribe([]byte(wu.Topic))
	if err != nil {
		wu.Log.Error("Subscribe error: ", err.Error())
		return
	}
	wu.Log.Info("Joined to topic: ", wu.Topic)
}

func (wu *WhirlyUtilitiesImpl) GetConsensusID() int64 {
	return wu.ConsensusID
}

func (wu *WhirlyUtilitiesImpl) GetP2pAdaptorType() string {
	return wu.p2pAdaptor.GetP2PType()
}

func (wu *WhirlyUtilitiesImpl) DecodeMsgByte(msgByte []byte) (*pb.WhirlyMsg, error) {
	msg := new(pb.WhirlyMsg)
	err := proto.Unmarshal(msgByte, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
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

func (wu *WhirlyUtilitiesImpl) ProposalMsg(block *pb.WhirlyBlock, qc *pb.QuorumCert, proof *pb.SimpleWhirlyProof, epoch int64) *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}

	wMsg.Payload = &pb.WhirlyMsg_WhirlyProposal{WhirlyProposal: &pb.WhirlyProposal{
		SenderId: uint64(wu.ID),
		Block:    block,
		HighQC:   qc,
		SwProof:  proof,
		Epoch:    uint64(epoch),
		PeerId:   wu.p2pAdaptor.GetPeerID(),
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) VoteMsg(blockView uint64, blockHash []byte, flag bool, qc *pb.QuorumCert, partSig []byte, proof *pb.SimpleWhirlyProof, epoch int64) *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}

	wMsg.Payload = &pb.WhirlyMsg_WhirlyVote{WhirlyVote: &pb.WhirlyVote{
		SenderId:   uint64(wu.ID),
		BlockView:  blockView,
		BlockHash:  blockHash,
		Flag:       flag,
		Qc:         qc,
		PartialSig: partSig,
		SwProof:    proof,
		Epoch:      uint64(epoch),
		PeerId:     wu.p2pAdaptor.GetPeerID(),
		Weight:     uint64(wu.Weight),
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) NewViewMsg(qc *pb.QuorumCert, viewNum uint64) *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}

	wMsg.Payload = &pb.WhirlyMsg_WhirlyNewView{WhirlyNewView: &pb.WhirlyNewView{
		LockQC:  qc,
		ViewNum: viewNum,
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) NewLeaderNotifyMsg(epoch int64, proof []byte, committee []string) *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}

	wMsg.Payload = &pb.WhirlyMsg_NewLeaderNotify{NewLeaderNotify: &pb.NewLeaderNotify{
		Leader:    uint64(wu.ID),
		Epoch:     uint64(epoch),
		Proof:     proof,
		PeerId:    wu.p2pAdaptor.GetPeerID(),
		Committee: committee,
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) NewLeaderEchoMsg(leader int64, block *pb.WhirlyBlock, proof *pb.SimpleWhirlyProof, epoch int64, vHeghit uint64) *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}

	wMsg.Payload = &pb.WhirlyMsg_NewLeaderEcho{NewLeaderEcho: &pb.NewLeaderEcho{
		Leader:   uint64(leader),
		SenderId: uint64(wu.ID),
		Epoch:    uint64(epoch),
		Block:    block,
		SwProof:  proof,
		VHeight:  vHeghit,
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) PingMsg() *pb.WhirlyMsg {
	wMsg := &pb.WhirlyMsg{}
	wMsg.Payload = &pb.WhirlyMsg_WhirlyPing{WhirlyPing: &pb.WhirlyPing{
		Id:     uint64(wu.ID),
		PeerId: wu.p2pAdaptor.GetPeerID(),
	}}
	return wMsg
}

func (wu *WhirlyUtilitiesImpl) CreateLeaf(parentHash []byte, viewNum uint64, txs []types.RawTransaction, justify *pb.QuorumCert) *pb.WhirlyBlock {
	b := &pb.WhirlyBlock{
		ParentHash: parentHash,
		Hash:       nil,
		Height:     viewNum,
		Txs:        types.RawTxArrayToBytes(txs),
		Justify:    justify,
	}

	b.Hash = types.Hash(b)
	return b
}

func (wu *WhirlyUtilitiesImpl) QC(view uint64, sig tcrsa.Signature, blockHash []byte) *pb.QuorumCert {
	marshal, _ := json.Marshal(sig)
	return &pb.QuorumCert{
		BlockHash: blockHash,
		ViewNum:   view,
		Signature: marshal,
	}
}

func (wu *WhirlyUtilitiesImpl) Proof(view uint64, proof []*pb.WhirlyVote, blockHash []byte) *pb.SimpleWhirlyProof {
	return &pb.SimpleWhirlyProof{
		BlockHash: blockHash,
		ViewNum:   view,
		Proof:     proof,
	}
}

func (wu *WhirlyUtilitiesImpl) GetMsgByteEntrance() chan<- []byte {
	return wu.MsgByteEntrance
}

func (wu *WhirlyUtilitiesImpl) GetRequestEntrance() chan<- *pb.Request {
	return wu.RequestEntrance
}

// GetLeader get the leader replica in view
func (wu *WhirlyUtilitiesImpl) GetLeader(viewNum int64) int64 {
	id := viewNum % int64(len(wu.Config.Nodes))
	return id
}

func (wu *WhirlyUtilitiesImpl) GetSelfInfo() *config.ReplicaInfo {
	self := &config.ReplicaInfo{}
	for _, info := range wu.Config.Nodes {
		if info.ID == wu.ID {
			self = info
			break
		}
	}
	return self
}

func (wu *WhirlyUtilitiesImpl) GetNetworkInfo() map[int64]string {
	networkInfo := make(map[int64]string)
	for _, info := range wu.Config.Nodes {
		if info.ID == wu.ID {
			continue
		}
		networkInfo[info.ID] = info.Address
	}
	return networkInfo
}

func (wu *WhirlyUtilitiesImpl) Broadcast(msg *pb.WhirlyMsg) error {
	if wu.p2pAdaptor == nil {
		wu.Log.Warn("p2pAdaptor nil")
		return nil
	}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)

	// for _, node := range wu.Config.Nodes {
	// 	if node.ID == wu.ID {
	// 		continue
	// 	}
	// 	err := wu.p2pAdaptor.Unicast(node.Address, msgByte, wu.ConsensusID, []byte("consensus"))
	// 	if err != nil {
	// 		wu.Log.WithError(err).Warn("send msg failed")
	// 		return err
	// 	}
	// }
	// return nil
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: wu.ConsensusID,
		Epoch:       0,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	// return wu.p2pAdaptor.Broadcast(bytePacket, wu.ConsensusID, []byte("this-is-consensus-topic"))
	return wu.p2pAdaptor.Broadcast(bytePacket, wu.ConsensusID, []byte(wu.Topic))
}

func (wu *WhirlyUtilitiesImpl) Unicast(address string, msg *pb.WhirlyMsg) error {
	if wu.p2pAdaptor == nil {
		wu.Log.Warn("p2pAdaptor nil")
		return nil
	}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: wu.ConsensusID,
		Epoch:       0,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	// utils.PanicOnError(err)
	// return wu.p2pAdaptor.Unicast(address, bytePacket, wu.ConsensusID, []byte("this-is-consensus-topic"))
	return wu.p2pAdaptor.Unicast(address, bytePacket, wu.ConsensusID, []byte(wu.Topic))
}

func (wu *WhirlyUtilitiesImpl) ProcessProposal(b *pb.WhirlyBlock, p []byte) {
	// wu.Log.Debugf("[whu] Process proposal")
	wu.Executor.CommitBlock(b, p, wu.ConsensusID)
	// wu.Log.Debugf("[whu] after Process proposal")
	// for _, tx := range txs {
	// 	wu.Executor.CommitTx(tx, wu.ConsensusID)
	// 	// msg := &pb.Msg{Payload: &pb.Msg_Reply{Reply: &pb.Reply{Tx: tx, Receipt: []byte(result)}}}
	// 	// err := wu.Unicast("localhost:9999", msg)
	// 	// if err != nil {
	// 	// 	fmt.Println(err.Error())
	// 	// }
	// }
	wu.MemPool.Remove(types.RawTxArrayFromBytes(b.Txs))
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

func (wu *WhirlyUtilitiesImpl) GetWeight(nid int64) float64 {
	if nid < int64(len(wu.Config.Nodes)) {
		return 1.0 / float64(len(wu.Config.Nodes))
	}
	return 0.0
}

func (wu *WhirlyUtilitiesImpl) GetMaxAdversaryWeight() float64 {
	return 1.0 / 3.0
}
