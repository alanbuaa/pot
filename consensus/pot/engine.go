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
	storage "github.com/zzz136454872/upgradeable-consensus/internal/storage/pot"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
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
	Worker *Worker

	//headerStorage  *types.HeaderStorage
	blockStorage *storage.BlockStorage
	//chainReader    *ChainReader
	UpperConsensus *nodeController.NodeController
}

func NewPoTEngine(nid int64, cid int64, config *config.ConsensusConfig, exec executor.Executor, adaptor p2p.P2PAdaptor, log *logrus.Entry) *PoTEngine {
	log = log.WithField("module", "PoT").WithField("c_id", cid)
	log.Info("Initializing PoT consensus")
	ch := make(chan []byte, 1024)
	//st := types.NewHeaderStorage(nid)
	log.WithField("datadir", config.Nodes[nid].DataDir).Debug("Creating block storage")
	bst := storage.NewBlockStorage(nid, config.Nodes[nid].DataDir)
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
	log.Debug("Initializing genesis block")
	bst.Put(types.DefaultGenesisBlock())
	log.Debug("Creating PoT worker")
	worker := NewWorker(nid, config, log, bst, e)
	e.Worker = worker

	// adaptor.SetReceiver(e)
	adaptor.SetReceiver(e.GetMsgByteEntrance())
	log.WithField("topic", config.Topic).Debug("Subscribing to P2P topic")
	err := adaptor.Subscribe([]byte(config.Topic))
	if adaptor.GetP2PType() == "p2p" {
		e.peerId = config.Nodes[nid].P2PAddress
		log.WithField("peer_id", e.peerId).Debug("Using base P2P with address as peer ID")
	} else {
		e.peerId = adaptor.GetPeerID()
		log.WithField("peer_id", e.peerId).Debug("Using libp2p with generated peer ID")
	}
	if err != nil {
		log.WithError(err).Error("Failed to subscribe to P2P topic")
		return nil
	}
	log.Info("P2P configuration completed")

	e.start()
	return e
}
func (e *PoTEngine) start() {

	e.log.Info("Starting PoT Consensus Engine")
	e.log.Debug("Initializing committee consensus (Whirly)")
	whirly := e.StartCommitee()
	e.Worker.SetWhirly(whirly)
	e.log.Debug("Committee consensus initialized")

	// Send initial PoT signal to create genesis sharding before accepting client requests
	e.log.Debug("Sending initial PoT signal for genesis sharding")
	e.Worker.SendInitialPoTSignal()

	e.log.Debug("Starting PoT worker goroutines")
	go e.Worker.OnGetVdf0Response()
	go e.Worker.Work()
	go e.Worker.handleVdfhalf()
	go e.onReceiveMsg()
	go e.Worker.rpcserver.Serve(e.Worker.listener)
	e.log.Info("PoT Consensus Engine started successfully, all worker goroutines running")
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
	e.log.Info("Stopping PoT Consensus Engine")
	dbPath := "dbfile/node0-" + strconv.Itoa(int(e.id))
	e.log.WithField("path", dbPath).Debug("Cleaning up database files")
	_ = os.RemoveAll(dbPath)
	e.log.Debug("Stopping worker")
	e.Worker.Stop()
	e.log.Debug("Closing message entrance channel")
	close(e.GetMsgByteEntrance())
	e.log.Info("PoT Consensus Engine stopped successfully")
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
		e.log.Error("P2P adaptor not found")
		return fmt.Errorf("can't find p2p adaptor")
	}
	e.log.WithField("msg_size", len(msgByte)).Trace("Broadcasting message")
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

func (e *PoTEngine) GetBlockStorage() *storage.BlockStorage {
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
		Nodes:  e.config.Nodes,
		Keys:   e.config.Keys,
		Topic:  e.config.Topic,
		Fault:  e.config.Fault,
		NodeId: e.config.NodeId, // 传递父共识id
	}

	//s := simpleWhirly.NewSimpleWhirly(e.id, 1009, whirlyconfig, e.exec, e.Adaptor, e.log, "", nil)
	s := nodeController.NewNodeController(e.id, 1009, whirlyconfig, e.exec, e.Adaptor, e.log)
	e.UpperConsensus = s
	e.log.Debug("Committee consensus (Whirly) initialized")
	return s
}

func (e *PoTEngine) GetPeerID() string {
	return e.peerId
}

func (e *PoTEngine) GetConsensusType() string {
	return "pot"
}

// GetConfig returns the consensus configuration
func (e *PoTEngine) GetConfig() *config.ConsensusConfig {
	return e.config
}

// GetAdaptor returns the P2P adaptor
func (e *PoTEngine) GetAdaptor() p2p.P2PAdaptor {
	return e.Adaptor
}
