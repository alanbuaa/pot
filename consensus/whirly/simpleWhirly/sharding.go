package simpleWhirly

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
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
	SubConsensus        config.ConsensusConfig
}

type Sharding struct {
	Name                string
	ParentSharding      []string
	LeaderPublicAddress string
	Committee           []string
	CryptoElements      []byte
	SubConsensus        *config.ConsensusConfig

	controller *NodeController

	// WhrilyNodes map[string]*SimpleWhirlyImpl
	Nodes     map[string]model.Consensus // encodeAddress -> node
	nodesLock sync.Mutex
	epoch     int64
	active    int
}

func NewSharding(name string, nc *NodeController, s PoTSharding) *Sharding {
	ns := &Sharding{
		Name: name,
		// WhrilyNodes: make(map[string]*SimpleWhirlyImpl),
		Nodes: make(map[string]model.Consensus),
		// Nodes:               make(map[string]map[int64]model.Consensus),
		epoch:               nc.epoch,
		active:              0,
		controller:          nc,
		LeaderPublicAddress: s.LeaderPublicAddress,
		Committee:           s.Committee,
		SubConsensus:        &s.SubConsensus,
	}

	// The daemonNode is always sleep, it only replies to transactions to local client and forwards transactions to leader
	// 这里加上nc.PeerId, 是为了避免多个PoT节点在同一台机器上运行产生的冲突，因为，PoT的节点1和节点2都会创建 daemonNode
	// 创建其他的节点是不需要加nc.PeerId，因为其他节点的address本身就不一样
	// 注意修改停止 daemonNode 的逻辑
	address := EncodeAddress(name+nc.PeerId, ns.SubConsensus.ConsensusID, DaemonNodePublicAddress)
	ns.createConsensusNode(address)

	return ns
}

func (s *Sharding) Stop() {
	for _, node := range s.Nodes {
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
	for encodeAddress, node := range s.Nodes {
		if encodeAddress == address {
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
		for _, node := range s.Nodes {
			go func(n model.Consensus) {
				n.GetMsgByteEntrance() <- packet.GetMsg()
			}(node)
		}
	} else {
		node, ok2 := s.Nodes[packet.ReceiverPublicAddress]
		if !ok2 {
			// s.controller.Log.Trace("Sharding handle message: Receiver public addresse does not exist")
		} else {
			node.GetMsgByteEntrance() <- packet.GetMsg()
		}
	}
	s.nodesLock.Unlock()
}

func (s *Sharding) handleRequest(req *pb.Request) {
	s.nodesLock.Lock()

	for _, node := range s.Nodes {
		go func(n model.Consensus) {
			n.GetRequestEntrance() <- req
		}(node)
	}

	s.nodesLock.Unlock()
}

func (s *Sharding) handleStop(address string) {
	s.nodesLock.Lock()
	node, ok2 := s.Nodes[address]
	if !ok2 {
		s.controller.Log.Warn("receive unkonwn publicAddress from StopEntrance")
	} else {
		node.Stop()
		delete(s.Nodes, address)
		s.active -= 1
	}
	s.nodesLock.Unlock()
}

func (s *Sharding) NodeManage(potSignal *PoTSignal) {
	s.controller.Log.Trace("NodeManage: ", s.Name)
	// Determine whether the leader belongs to oneself
	for _, address := range potSignal.SelfPublicAddress {
		// fmt.Println("---------", s.Name, "----", s.LeaderPublicAddress)
		for _, c := range s.Committee {
			if address == c && address != s.LeaderPublicAddress {
				_ = s.checkNodeStauts(address)
				// nc.Log.Info("create a new committee")
			} else {
				// fmt.Println("AAA = ", address, "\nDDD = ", c)
			}
		}

		if address == s.LeaderPublicAddress {
			// This new node or committee node attempts to become a leader
			node := s.checkNodeStauts(address)
			// Node response mechanism
			// s.controller.Log.WithFields(logrus.Fields{
			// 	"epoch":    potSignal.Epoch,
			// 	"proof":    potSignal.Proof,
			// 	"committe": s.Committee,
			// }).Info("print NewEpochConfirmation!!!")
			// s.controller.Log.WithFields(logrus.Fields{
			// 	"node": node,
			// }).Info("print NewEpochConfirmation222")

			// s.controller.Log.WithFields(logrus.Fields{
			// 	// "sender":       echoMsg.PublicAddress,
			// 	"epoch":    s.controller.epoch,
			// 	"sharding": s.Name,
			// }).Trace("begin create a new leader")

			node.NewEpochConfirmation(potSignal.Epoch, potSignal.Proof, s.Committee)
			s.controller.Log.WithFields(logrus.Fields{
				// "sender":       echoMsg.PublicAddress,
				"epoch":    s.controller.epoch,
				"sharding": s.Name,
			}).Info("A new leader is ready")
		} else {
			// fmt.Println("AAA = ", address, "\nBBB = ", s.LeaderPublicAddress)
		}
	}
}

// Address has not been encoded
func (s *Sharding) checkNodeStauts(publicAddress string) model.Consensus {
	s.nodesLock.Lock()
	address := EncodeAddress(s.Name, s.SubConsensus.ConsensusID, publicAddress)
	node, ok := s.Nodes[address]
	if !ok {
		// The address belongs to this controller, create a new node
		newNode := s.createConsensusNode(address)
		// update leader information in node
		leader := EncodeAddress(s.Name, s.SubConsensus.ConsensusID, s.LeaderPublicAddress)
		command1 := model.ExternalStatus{
			Command: "SetLeader",
			Epoch:   s.epoch,
			Leader:  leader,
		}
		newNode.UpdateExternalStatus(command1)

		command2 := model.ExternalStatus{
			Command: "SetEpoch",
			Epoch:   s.epoch,
		}
		newNode.UpdateExternalStatus(command2)

		s.controller.Log.WithFields(logrus.Fields{
			// "sender":       echoMsg.PublicAddress,
			"epoch":       s.controller.epoch,
			"nodeAddress": address,
			"sharding":    s.Name,
		}).Trace("create a new node")

		s.nodesLock.Unlock()
		return newNode
	} else {
		// update leader information in node
		leader := EncodeAddress(s.Name, s.SubConsensus.ConsensusID, s.LeaderPublicAddress)
		// don't update epoch, the update of epoch is triggered by the new leader
		command1 := model.ExternalStatus{
			Command: "SetLeader",
			Epoch:   s.epoch,
			Leader:  leader,
		}
		node.UpdateExternalStatus(command1)
	}
	s.nodesLock.Unlock()
	return node
}

// Address has already been encoded
func (s *Sharding) createConsensusNode(address string) model.Consensus {
	var c model.Consensus = nil
	switch s.SubConsensus.Type {
	case "whirly":
		switch s.SubConsensus.Whirly.Type {
		case "basic":
			// c = NewWhirly(int64(s.controller.total), s.controller.ConsensusID, s.epoch, s.controller.Config, s.controller.Executor, s, s.controller.Log, address, s.controller.StopEntrance)
			s.controller.Log.Warnf("Basic whirly is not supported yet")
		case "simple":
			s.active += 1
			s.controller.total += 1
			c = NewSimpleWhirly(
				int64(s.controller.total),
				s.SubConsensus.ConsensusID,
				s.epoch,
				s.controller.Config,
				s.controller.Executor,
				s,
				s.controller.Log,
				address,
				s.controller.StopEntrance,
			)
			s.Nodes[address] = c
		default:
			s.controller.Log.Warnf("whirly type not supported: %s", s.SubConsensus.Whirly.Type)
		}
	default:
		s.controller.Log.Warnf("CreateConsensusNode: consensus type not supported: %s", s.SubConsensus.Type)
	}
	return c
}

// func (s *Sharding) createNode2(publicAddress string) model.Consensus {
// 	s.nodesLock.Lock()

// 	// Encode address, find nodes through sharding name and address
// 	address := EncodeAddress(s.Name, publicAddress)
// 	consensusID := s.SubConsensus.ConsensusID
// 	nowConsensus, ok := s.Nodes[address]
// 	if !ok {
// 		// The address belongs to this controller, create a new node
// 		consensus := make(map[int64]model.Consensus)
// 		node := s.createConsensusNode(address)

// 		// update leader
// 		leader := EncodeAddress(s.Name, s.LeaderPublicAddress)
// 		command1 := model.ExternalStatus{
// 			Command: "SetLeader",
// 			Epoch:   s.epoch,
// 			Leader:  leader,
// 		}
// 		node.UpdateExternalStatus(command1)

// 		command2 := model.ExternalStatus{
// 			Command: "SetEpoch",
// 			Epoch:   s.epoch,
// 		}
// 		node.UpdateExternalStatus(command2)

// 		// update nodes in sharding
// 		consensus[consensusID] = node
// 		s.Nodes[address] = consensus

// 		s.controller.Log.WithFields(logrus.Fields{
// 			// "sender":       echoMsg.PublicAddress,
// 			"epoch":       s.controller.epoch,
// 			"nodeAddress": address,
// 		}).Info("create a new consensus and a node")

// 		s.nodesLock.Unlock()
// 		return node
// 	} else {
// 		nowNode, ok2 := nowConsensus[consensusID]
// 		if !ok2 {
// 			node := s.createConsensusNode(address)

// 			// update leader
// 			leader := EncodeAddress(s.Name, s.LeaderPublicAddress)
// 			command1 := model.ExternalStatus{
// 				Command: "SetLeader",
// 				Epoch:   s.epoch,
// 				Leader:  leader,
// 			}
// 			node.UpdateExternalStatus(command1)

// 			command2 := model.ExternalStatus{
// 				Command: "SetEpoch",
// 				Epoch:   s.epoch,
// 			}
// 			node.UpdateExternalStatus(command2)

// 			// update nodes in consensus
// 			nowConsensus[consensusID] = node

// 			s.controller.Log.WithFields(logrus.Fields{
// 				// "sender":       echoMsg.PublicAddress,
// 				"epoch":       s.controller.epoch,
// 				"nodeAddress": address,
// 			}).Info("create a new node in exist consensus")

// 			s.nodesLock.Unlock()
// 			return node
// 		} else {
// 			// update leader
// 			leader := EncodeAddress(s.Name, s.LeaderPublicAddress)
// 			// don't update epoch, the update of epoch is triggered by the new leader
// 			command1 := model.ExternalStatus{
// 				Command: "SetLeader",
// 				Epoch:   s.epoch,
// 				Leader:  leader,
// 			}
// 			nowNode.UpdateExternalStatus(command1)
// 		}
// 		s.nodesLock.Unlock()
// 		return nowNode
// 	}

// }
