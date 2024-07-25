package simpleWhirly

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

type PoTSharding struct {
	Name                string
	ParentSharding      []string
	LeaderPublicAddress string
	Committee           []string
	CryptoElements      []byte
}

type Sharding struct {
	Name                string
	ParentSharding      []string
	LeaderPublicAddress string
	Committee           []string
	CryptoElements      []byte

	controller *NodeController

	WhrilyNodes map[string]*SimpleWhirlyImpl
	nodesLock   sync.Mutex
	epoch       int64
	active      int
}

func NewSharding(name string, nc *NodeController, s PoTSharding) *Sharding {
	ns := &Sharding{
		Name:                name,
		WhrilyNodes:         make(map[string]*SimpleWhirlyImpl),
		epoch:               nc.epoch,
		active:              0,
		controller:          nc,
		LeaderPublicAddress: s.LeaderPublicAddress,
		Committee:           s.Committee,
	}

	// The daemonNode is always sleep, it only reply tx to local client
	ns.active += 1
	nc.total += 1
	address := EncodeAddress(name+nc.PeerId, DaemonNodePublicAddress)
	simpleWhirly := NewSimpleWhirly(int64(nc.total), nc.ConsensusID, nc.epoch, nc.Config, nc.Executor, ns, nc.Log, address, nil)
	ns.WhrilyNodes[address] = simpleWhirly

	// ns.createNode(DaemonNodePublicAddress)
	return ns
}

func (s *Sharding) Stop() {
	for _, node := range s.WhrilyNodes {
		node.Stop()
	}
}

// Sharding implements P2PAdaptor
func (s *Sharding) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	packet := &pb.Packet{
		Msg:                   msgByte,
		ConsensusID:           consensusID,
		Epoch:                 0,
		Type:                  pb.PacketType_P2PPACKET,
		ReceiverPublicAddress: BroadcastToAll,
		ReceiverShardingName:  s.Name,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)

	return s.controller.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (s *Sharding) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	s.nodesLock.Lock()
	for publicAddress, node := range s.WhrilyNodes {
		if publicAddress == address {
			if node.GetMsgByteEntrance() != nil {
				// nc.Log.Info("Unicast by channel")
				node.GetMsgByteEntrance() <- msgByte
			} else {
				s.controller.Log.Warn("the MsgByteEntrance of node is nil")
			}
			s.nodesLock.Unlock()
			return nil
		}
	}
	// nc.Log.Info("Unicast by p2padptor")
	s.nodesLock.Unlock()

	packet := &pb.Packet{
		Msg:                   msgByte,
		ConsensusID:           consensusID,
		Epoch:                 0,
		Type:                  pb.PacketType_P2PPACKET,
		ReceiverPublicAddress: address,
		ReceiverShardingName:  s.Name,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	return s.controller.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (s *Sharding) SetReceiver(ch chan<- []byte) {
	// do nothing
}

func (s *Sharding) Subscribe(topic []byte) error {
	// do nothing
	return nil
}

func (s *Sharding) UnSubscribe(topic []byte) error {
	// do nothing
	return nil
}

func (s *Sharding) GetPeerID() string {
	return s.controller.p2pAdaptor.GetPeerID()
}

func (s *Sharding) GetP2PType() string {
	return s.controller.p2pAdaptor.GetP2PType()
}

func (s *Sharding) handleMsg(packet *pb.Packet) {
	s.nodesLock.Lock()
	if packet.ReceiverPublicAddress == "" {
		s.controller.Log.Warn("Sharding handle message error: Invaild receiver public address")
		fmt.Printf("packet: %+v\n", packet)

		msgByte := packet.GetMsg()
		msg := new(pb.WhirlyMsg)
		err := proto.Unmarshal(msgByte, msg)
		if err != nil {
			s.controller.Log.Error("Unmarshal error: ", err)
		}
		fmt.Printf("Payload: %+v\n", msg.Payload)
		switch msg.Payload.(type) {
		case *pb.WhirlyMsg_Request:
			fmt.Printf("msg: %+v\n", msg.GetRequest())
		case *pb.WhirlyMsg_WhirlyProposal:
			fmt.Printf("msg: %+v\n", msg.GetWhirlyProposal())
		case *pb.WhirlyMsg_WhirlyVote:
			fmt.Printf("msg: %+v\n", msg.GetWhirlyVote())
		case *pb.WhirlyMsg_NewLeaderNotify:
			fmt.Printf("msg: %+v\n", msg.GetNewLeaderNotify())
		case *pb.WhirlyMsg_NewLeaderEcho:
			fmt.Printf("msg: %+v\n", msg.GetNewLeaderEcho())
		case *pb.WhirlyMsg_WhirlyPing:
			fmt.Printf("msg: %+v\n", msg.GetWhirlyPing())
		default:
			s.controller.Log.Warn("Receive unsupported msg")
			return
		}
	}

	if packet.ReceiverPublicAddress == BroadcastToAll {
		for _, node := range s.WhrilyNodes {
			go func(n *SimpleWhirlyImpl) {
				n.GetMsgByteEntrance() <- packet.GetMsg()
			}(node)
		}
	} else {
		node, ok2 := s.WhrilyNodes[packet.ReceiverPublicAddress]
		if !ok2 {
			// s.controller.Log.Trace("Sharding handle message: Receiver public addresse does not exist")
		} else {
			node.GetMsgByteEntrance() <- packet.GetMsg()
		}
	}
	s.nodesLock.Unlock()
}

func (s *Sharding) handleStop(address string) {
	s.nodesLock.Lock()
	node, ok2 := s.WhrilyNodes[address]
	if !ok2 {
		s.controller.Log.Warn("receive unkonwn publicAddress from StopEntrance")
	} else {
		node.Stop()
		delete(s.WhrilyNodes, address)
		s.active -= 1
	}
	s.nodesLock.Unlock()
}

func (s *Sharding) createNode(publicAddress string) *SimpleWhirlyImpl {
	s.nodesLock.Lock()
	address := EncodeAddress(s.Name, publicAddress)
	node, ok := s.WhrilyNodes[address]
	if !ok {
		// The address belongs to this controller, create a new simpleWhirly node
		s.active += 1
		s.controller.total += 1

		simpleWhirly := NewSimpleWhirly(int64(s.controller.total), s.controller.ConsensusID, s.epoch, s.controller.Config, s.controller.Executor, s, s.controller.Log, address, s.controller.StopEntrance)
		// update leader
		leader := EncodeAddress(s.Name, s.LeaderPublicAddress)
		simpleWhirly.SetLeader(s.epoch, leader)
		simpleWhirly.SetEpoch(s.epoch)

		s.WhrilyNodes[address] = simpleWhirly

		s.controller.Log.WithFields(logrus.Fields{
			// "sender":       echoMsg.PublicAddress,
			"epoch":       s.controller.epoch,
			"nodeAddress": address,
		}).Info("create a new node")

		s.nodesLock.Unlock()
		return simpleWhirly
	} else {
		// update leader
		leader := EncodeAddress(s.Name, s.LeaderPublicAddress)
		// don't update epoch
		node.SetLeader(s.epoch, leader)
	}
	s.nodesLock.Unlock()
	return node
}
