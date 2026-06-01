package pot

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

type PoTEngine struct {
	id     int64
	peerId string

	// consensus config
	consensusID int64
	config      *config.ConsensusConfig

	// consensus executor
	exec executor.Executor
	log  *logrus.Entry

	// network
	Adaptor         p2p.P2PAdaptor
	isBaseP2P       bool
	MsgByteEntrance chan []byte
	RequestEntrance chan *pb.Request
	Topic           []byte

	// consensus work
	Height int64
	worker *Worker

	//headerStorage  *types.HeaderStorage
	blockStorage *types.BlockStorage
	//chainReader    *ChainReader
	UpperConsensus *nodeController.NodeController
}

func NewEngine(nid int64, cid int64, config *config.ConsensusConfig, exec executor.Executor, adaptor p2p.P2PAdaptor, log *logrus.Entry) *PoTEngine {
	ch := make(chan []byte, 1024)
	//st := types.NewHeaderStorage(nid)
	potconfig := config.PoT

	var bst *types.BlockStorage
	if potconfig.StorageMode == 0 {
		bst = types.NewBlockStorage(nid, cid)
	} else {
		bst = types.GetExistedBlockStorage(nid, cid)
	}
	e := &PoTEngine{
		id:              nid,
		peerId:          adaptor.GetPeerID(),
		consensusID:     cid,
		exec:            exec,
		log:             log,
		Adaptor:         adaptor,
		config:          config,
		MsgByteEntrance: ch,
		// Worker:          worker,
		//headerStorage: st,
		blockStorage: bst,
		Topic:        []byte(config.Topic),
	}

	worker := NewWorker(nid, config, log, bst, e)
	e.worker = worker

	// adaptor.SetReceiver(e)
	adaptor.SetReceiver(e.GetMsgByteEntrance())
	err := adaptor.Subscribe([]byte(config.Topic))
	if adaptor.GetP2PType() == "p2p" {
		e.peerId = config.Nodes[nid].Address
	} else {
		e.peerId = adaptor.GetPeerID()
	}
	if err != nil {
		return nil
	}

	if potconfig.StorageMode == 0 {
		e.start()
	} else {
		e.restart()
	}

	return e
}

func (e *PoTEngine) restart() {

	whirly := e.StartCommitee()
	e.worker.SetWhirly(whirly)
	epoch, vdfres, err := e.blockStorage.GetHighestVDFRes()
	curparent := types.DefaultGenesisBlock()
	e.worker.chainReader.SetHeight(0, curparent)

	for i := uint64(1); i <= epoch-1; i++ {
		blocks, err := e.blockStorage.GetbyHeight(i)
		if err != nil {
			panic(err)
		}
		res0, err := e.blockStorage.GetVDFresbyEpoch(i)
		if err != nil {
			panic(err)
		}
		parent, _ := e.worker.blockSelection(blocks, res0, i)

		e.worker.handleBlockExecutedHeader(parent)
		err = e.worker.handleBlockRawTx(parent)
		if err != nil {
			panic(err)
		}
		e.worker.chainReader.SetHeight(i, parent)
		curparent = parent
	}
	if err != nil {
		go e.worker.Work()
		e.log.Errorf("[PoT]\tRestart from storage error, startfrom genesis block")
		return
	}
	vdfres, err = e.blockStorage.GetVDFresbyEpoch(epoch - 1)

	res := &types.VDF0res{
		Epoch: epoch - 2,
		Res:   vdfres,
	}

	e.worker.vdf0Chan <- res
	go e.worker.OnGetVdf0Response()
	go e.onReceiveMsg()
	go e.worker.rpcserver.Serve(e.worker.listener)
	go e.worker.handleVdfhalf()
}

func (e *PoTEngine) start() {

	e.log.Infof("[PoT]\tPoT Consensus Engine starts working")
	whirly := e.StartCommitee()
	e.worker.SetWhirly(whirly)

	go e.worker.OnGetVdf0Response()
	go e.worker.Work()
	go e.worker.handleVdfhalf()
	go e.onReceiveMsg()
	go e.worker.rpcserver.Serve(e.worker.listener)
}
func (e *PoTEngine) GetRequestEntrance() chan<- *pb.Request {
	if e.UpperConsensus != nil && e.UpperConsensus.GetRequestEntrance() != nil {
		return e.UpperConsensus.GetRequestEntrance()
	}
	return nil
}

func (e *PoTEngine) GetMsgByteEntrance() chan<- []byte {
	return e.MsgByteEntrance
}

func (e *PoTEngine) Stop() {
	_ = os.RemoveAll("dbfile/node0-" + strconv.Itoa(int(e.id)))
	e.worker.stop()
	close(e.GetMsgByteEntrance())
}

func (e *PoTEngine) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (e *PoTEngine) GetWeight(nid int64) float64 {
	return 0
}

func (e *PoTEngine) GetMaxAdversaryWeight() float64 {
	return 0
}

func (e *PoTEngine) GetConsensusID() int64 {
	return e.consensusID
}

func (e *PoTEngine) UpdateExternalStatus(status model.ExternalStatus) {
	return
}

func (e *PoTEngine) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	return
}

func (e *PoTEngine) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	return
}

func (e *PoTEngine) Broadcast(msgByte []byte) error {
	if e.Adaptor == nil {
		return fmt.Errorf("can't find p2p adaptor")
	}
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: e.consensusID,
		Epoch:       e.Height,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	//e.log.Infof("[PoT]\t packet len %f KB", float64(len(bytePacket))/float64(1024))
	return e.Adaptor.Broadcast(bytePacket, e.consensusID, e.Topic)
}

func (e *PoTEngine) Unicast(address string, msgByte []byte) error {
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: e.consensusID,
		Epoch:       e.Height,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	//e.log.Infof("unicast byte:%s", hexutil.Encode(msgByte))
	return e.Adaptor.Unicast(address, bytePacket, e.consensusID, e.Topic)
}

func (e *PoTEngine) GetBlockStorage() *types.BlockStorage {
	return e.blockStorage
}

func (e *PoTEngine) SetWhirly(whirly2 *nodeController.NodeController) {
	e.UpperConsensus = whirly2
}

func (e *PoTEngine) StartCommitee() *nodeController.NodeController {
	whirlyconfig := &config.ConsensusConfig{
		Type:        "whirly",
		ConsensusID: 1009,
		Whirly: &config.WhirlyConfig{
			Type:      "simple",
			BatchSize: 10,
			Timeout:   2000,
		},
		Nodes: e.config.Nodes,
		Keys:  e.config.Keys,
		Topic: e.config.Topic,
		F:     e.config.F,
	}

	//s := simpleWhirly.NewSimpleWhirly(e.id, 1009, whirlyconfig, e.exec, e.Adaptor, e.log, "", nil)
	s := nodeController.NewNodeController(e.id, 1009, whirlyconfig, e.exec, e.Adaptor, e.log)
	e.UpperConsensus = s
	e.log.Infof("[PoT]\tCommitee consensus whirly get prepared")
	return s
}
func (e *PoTEngine) GetPeerID() string {
	return e.peerId
}
