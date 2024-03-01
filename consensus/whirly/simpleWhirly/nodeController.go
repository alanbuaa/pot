package simpleWhirly

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

const DaemonNodePublicAddress string = "daemonNode"
const BroadcastToAll string = "ALL"

type ControllerMessage struct {
	Data     []byte
	Receiver string
}

type PoTSignal struct {
	Epoch               int64
	Proof               []byte
	ID                  int64
	LeaderPublicAddress string
	Committee           []string
	SelfPublicAddress   []string
	CryptoElements      []byte
}

type NodeController struct {
	PeerId              string
	ConsensusID         int64
	MsgByteEntrance     chan []byte // receive msg
	RequestByteEntrance chan *pb.Request
	p2pAdaptor          p2p.P2PAdaptor
	Log                 *logrus.Entry
	Executor            executor.Executor
	Config              *config.ConsensusConfig
	cancel              context.CancelFunc

	epoch  int64
	total  int
	active int

	// PoT
	PoTByteEntrance chan []byte // receive msg
	StopEntrance    chan string // receive the publicAddress of the node that should be stopped

	WhrilyNodes map[string]*SimpleWhirlyImpl
	nodesLock   sync.Mutex
	Committee   []string
}

func NewNodeController(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *NodeController {
	log.Debug("[Node Controller] starting")
	ctx, cancel := context.WithCancel(context.Background())
	nc := &NodeController{
		PeerId:      p2pAdaptor.GetPeerID(),
		ConsensusID: cid,
		Config:      cfg,
		Executor:    exec,
		p2pAdaptor:  p2pAdaptor,
		Log:         log.WithField("consensus id", cid),
		cancel:      cancel,
		epoch:       1,
		total:       1,
		active:      0,
		WhrilyNodes: make(map[string]*SimpleWhirlyImpl),
	}

	// The daemonNode is always sleep, it only forwards requests to the leader
	nc.nodesLock.Lock()
	simpleWhirly := NewSimpleWhirly(id, cid, nc.epoch, cfg, exec, nc, log, DaemonNodePublicAddress, nil)
	nc.WhrilyNodes[DaemonNodePublicAddress] = simpleWhirly
	nc.nodesLock.Unlock()

	nc.MsgByteEntrance = make(chan []byte, 10)
	nc.RequestByteEntrance = make(chan *pb.Request, 10)
	nc.PoTByteEntrance = make(chan []byte, 10)
	nc.StopEntrance = make(chan string, 10)

	go nc.receiveMsg(ctx)

	return nc
}

func (nc *NodeController) GetPoTByteEntrance() chan<- []byte {
	return nc.PoTByteEntrance
}

func (nc *NodeController) GetMsgByteEntrance() chan<- []byte {
	return nc.MsgByteEntrance
}
func (nc *NodeController) GetRequestEntrance() chan<- *pb.Request {
	return nc.RequestByteEntrance
}

func (nc *NodeController) DecodeMsgByte(msgByte []byte) (*ControllerMessage, error) {
	msg := new(ControllerMessage)
	err := json.Unmarshal(msgByte, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (nc *NodeController) Stop() {
	nc.cancel()
	close(nc.MsgByteEntrance)
	close(nc.RequestByteEntrance)
	close(nc.PoTByteEntrance)
	close(nc.StopEntrance)
}

func (nc *NodeController) receiveMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgByte, ok := <-nc.RequestByteEntrance:
			if !ok {
				return // closed
			}
			nc.nodesLock.Lock()
			for _, node := range nc.WhrilyNodes {
				go func(n *SimpleWhirlyImpl) {
					n.GetRequestEntrance() <- msgByte
				}(node)
			}
			nc.nodesLock.Unlock()
		case msgByte, ok := <-nc.MsgByteEntrance:
			if !ok {
				return // closed
			}
			msg, err := nc.DecodeMsgByte(msgByte)
			if err != nil {
				nc.Log.WithError(err).Warn("decode message to ControllerMessage failed")
				continue
			}
			go nc.handleMsg(msg)
		case address, ok := <-nc.StopEntrance:
			if !ok {
				return // closed
			}
			nc.nodesLock.Lock()
			node, ok2 := nc.WhrilyNodes[address]
			if !ok2 {
				nc.Log.Warn("receive unkonwn publicAddress from StopEntrance")
			} else {
				node.Stop()
				delete(nc.WhrilyNodes, address)
				nc.active -= 1
			}
			nc.nodesLock.Unlock()
		case potSignal, ok := <-nc.PoTByteEntrance:
			if !ok {
				return // closed
			}
			go nc.handlePotSignal(potSignal)
		}
	}
}

func (nc *NodeController) handleMsg(msg *ControllerMessage) {
	if msg.Receiver == BroadcastToAll {
		nc.nodesLock.Lock()
		for _, node := range nc.WhrilyNodes {
			go func(n *SimpleWhirlyImpl) {
				n.GetMsgByteEntrance() <- msg.Data
			}(node)
		}
		nc.nodesLock.Unlock()
	} else {
		nc.nodesLock.Lock()
		node, ok := nc.WhrilyNodes[msg.Receiver]
		if !ok {
			nc.Log.Trace("ignore message")
		} else {
			node.GetMsgByteEntrance() <- msg.Data
		}
		nc.nodesLock.Unlock()
	}
}

// NodeController implements P2PAdaptor
func (nc *NodeController) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	packet := &pb.Packet{
		Msg:                   msgByte,
		ConsensusID:           nc.ConsensusID,
		Epoch:                 0,
		Type:                  pb.PacketType_P2PPACKET,
		ReceiverPublicAddress: BroadcastToAll,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)

	//nc.nodesLock.Lock()
	//for _, node := range nc.WhrilyNodes {
	//	go func(n *SimpleWhirlyImpl) {
	//		n.GetMsgByteEntrance() <- msgByte
	//	}(node)
	//}
	//nc.nodesLock.Unlock()

	return nc.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (nc *NodeController) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	nc.nodesLock.Lock()
	for publicAddress, node := range nc.WhrilyNodes {
		if publicAddress == address {
			if node.GetMsgByteEntrance() != nil {
				// nc.Log.Info("Unicast by channel")
				node.GetMsgByteEntrance() <- msgByte
			} else {
				nc.Log.Warn("the MsgByteEntrance of node is nil")
			}
			nc.nodesLock.Unlock()
			return nil
		}
	}
	// nc.Log.Info("Unicast by p2padptor")
	nc.nodesLock.Unlock()

	packet := &pb.Packet{
		Msg:                   msgByte,
		ConsensusID:           nc.ConsensusID,
		Epoch:                 0,
		Type:                  pb.PacketType_P2PPACKET,
		ReceiverPublicAddress: address,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	return nc.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (nc *NodeController) SetReceiver(ch chan<- []byte) {
	// do nothing
}

func (nc *NodeController) Subscribe(topic []byte) error {
	// do nothing
	return nil
}

func (nc *NodeController) UnSubscribe(topic []byte) error {
	// do nothing
	return nil
}

func (nc *NodeController) GetPeerID() string {
	return nc.p2pAdaptor.GetPeerID()
}

func (nc *NodeController) GetP2PType() string {
	return nc.p2pAdaptor.GetP2PType()
}

func (nc *NodeController) handlePotSignal(potSignalBytes []byte) {
	potSignal := &PoTSignal{}
	err := json.Unmarshal(potSignalBytes, potSignal)
	if err != nil {
		nc.Log.WithField("error", err.Error()).Error("Unmarshal potSignal failed.")
		return
	}

	// Ignoring pot signals from old epochs
	if potSignal.Epoch <= nc.epoch {
		return
	} else {
		nc.epoch = potSignal.Epoch
	}

	// Determine whether the leader belongs to oneself
	for _, address := range potSignal.SelfPublicAddress {
		if address == potSignal.LeaderPublicAddress {
			// The leader belongs to this controller, create a new simpleWhirly node
			nc.nodesLock.Lock()
			nc.active += 1
			nc.total += 1
			simpleWhirly := NewSimpleWhirly(int64(nc.total), nc.ConsensusID, nc.epoch, nc.Config, nil, nc, nc.Log, address, nc.StopEntrance)
			nc.WhrilyNodes[address] = simpleWhirly
			nc.nodesLock.Unlock()

			nc.Log.WithFields(logrus.Fields{
				// "sender":       echoMsg.PublicAddress,
				"epoch":       potSignal.Epoch,
				"nodeAddress": address,
			}).Info("create a new committee node")

			// This new simpleWhirly node attempts to become a leader
			simpleWhirly.NewLeader(potSignal)
			return
		}
	}
}
