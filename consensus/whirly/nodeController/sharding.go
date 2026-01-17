package nodeController

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/crWhirly"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"

	bc_api "blockchain-crypto/blockchain_api"
)

type PoTSharding struct {
	Name                string
	ParentSharding      []string
	LeaderPublicAddress string
	Committee           []string
	// CryptoElements      []byte
	CryptoElements bc_api.CommitteeConfig
	SubConsensus   config.ConsensusConfig
}

type Sharding struct {
	Name                string
	ParentSharding      []string
	LeaderPublicAddress string
	Committee           []string
	CryptoElements      bc_api.CommitteeConfig
	SubConsensus        *config.ConsensusConfig

	controller *NodeController

	// WhrilyNodes map[string]*SimpleWhirlyImpl
	Nodes     map[string]model.Consensus // encodeAddress -> node
	nodesLock sync.Mutex
	epoch     int64
	active    int
}

func NewSharding(name string, nc *NodeController, s PoTSharding) *Sharding {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", name)
	logger.WithFields(logrus.Fields{
		"consensus_id":   s.SubConsensus.ConsensusID,
		"consensus_type": s.SubConsensus.Type,
		"leader":         s.LeaderPublicAddress,
		"committee_size": len(s.Committee),
		"epoch":          nc.epoch,
	}).Info("Initializing new sharding")

	ns := &Sharding{
		Name:                name,
		Nodes:               make(map[string]model.Consensus),
		epoch:               nc.epoch,
		active:              0,
		controller:          nc,
		LeaderPublicAddress: s.LeaderPublicAddress,
		Committee:           s.Committee,
		SubConsensus:        &s.SubConsensus,
		CryptoElements:      s.CryptoElements,
	}

	// The daemonNode is always sleep, it only replies to transactions to local client and forwards transactions to leader
	// 这里加上nc.PeerId, 是为了避免多个PoT节点在同一台机器上运行产生的冲突，因为，PoT的节点1和节点2都会创建 daemonNode
	// 创建其他的节点是不需要加nc.PeerId，因为其他节点的address本身就不一样
	// 注意修改停止 daemonNode 的逻辑
	address := EncodeAddress(name+nc.PeerId, ns.SubConsensus.ConsensusID, DaemonNodePublicAddress)
	logger.WithField("daemon_node", address).Debug("Creating daemon node for sharding")
	daemonNode := ns.createConsensusNode(address)

	// Initialize daemon node with leader and epoch to prevent leader lookup failures
	// For daemon nodes, set leader address to point to itself (with PeerId) so message routing works
	// This allows the daemon node to find itself in Sharding.Nodes map when handling requests
	leaderAddress := address // Daemon node is its own leader for request handling
	logger.WithFields(logrus.Fields{
		"daemon_node":  address,
		"leader":       leaderAddress,
		"epoch":        nc.epoch,
		"consensus_id": ns.SubConsensus.ConsensusID,
	}).Debug("Initializing daemon node with leader and epoch")

	// Set leader for all epochs from 0 to current epoch to handle epoch mismatches
	for epoch := int64(0); epoch <= nc.epoch; epoch++ {
		command := model.ExternalStatus{
			Command: "SetLeader",
			Epoch:   epoch,
			Leader:  leaderAddress,
		}
		daemonNode.UpdateExternalStatus(command)
	}

	// Set current epoch
	command := model.ExternalStatus{
		Command: "SetEpoch",
		Epoch:   nc.epoch,
	}
	daemonNode.UpdateExternalStatus(command)

	return ns
}

func (s *Sharding) Stop() {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	logger.WithField("node_count", len(s.Nodes)).Info("Stopping all nodes in sharding")
	for address, node := range s.Nodes {
		logger.WithField("node_address", address).Debug("Stopping node")
		node.Stop()
	}
	logger.Info("All nodes stopped successfully")
}

// Sharding implements P2PAdaptor
func (s *Sharding) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
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

	logger.WithFields(logrus.Fields{
		"consensus_id": consensusID,
		"msg_size":     len(msgByte),
	}).Debug("Broadcasting message to all nodes")
	return s.controller.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (s *Sharding) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	for encodeAddress, node := range s.Nodes {
		if encodeAddress == address {
			if node.GetMsgByteEntrance() != nil {
				logger.WithField("target", address).Debug("Unicast via local channel")
				node.GetMsgByteEntrance() <- msgByte
			} else {
				logger.WithField("target", address).Warn("Node message entrance is nil")
			}
			s.nodesLock.Unlock()
			return nil
		}
	}
	logger.WithField("target", address).Debug("Unicast via P2P adapter (remote node)")
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

// Sharding implements Executor
func (s *Sharding) CommitBlock(block types.ConsensusBlock, proof []byte, cid int64) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	logger.Trace("[TRACE-5] Sharding.CommitBlock called")

	txs := block.GetTxs()
	newBlock := &pb.WhirlyBlock{
		Txs:          txs,
		ShardingName: []byte(s.Name),
		Incentive:    block.GetIncentive(),
	}
	logger.WithFields(logrus.Fields{
		"consensus_id": cid,
		"tx_count":     len(txs),
		"proof_size":   len(proof),
	}).Trace("[TRACE-5.1] Committing block to executor")

	if s.controller.Executor == nil {
		logger.Error("[TRACE-5-ERROR] Executor is nil!")
		return
	}

	s.controller.Executor.CommitBlock(newBlock, proof, cid)
	logger.Trace("[TRACE-5.2] Block committed to executor successfully")
}

func (s *Sharding) VerifyTx(tx types.RawTransaction) bool {
	return s.controller.Executor.VerifyTx(tx)
}

func (s *Sharding) handleMsg(packet *pb.Packet) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	if packet.ReceiverPublicAddress == "" {
		logger.Warn("Dropping message with empty receiver address")
		msgByte := packet.GetMsg()
		msg := new(pb.WhirlyMsg)
		err := proto.Unmarshal(msgByte, msg)
		if err != nil {
			logger.WithError(err).Error("Failed to unmarshal message for debugging")
			return
		}

		var msgType string
		switch msg.Payload.(type) {
		case *pb.WhirlyMsg_Request:
			msgType = "Request"
		case *pb.WhirlyMsg_WhirlyProposal:
			msgType = "Proposal"
		case *pb.WhirlyMsg_WhirlyVote:
			msgType = "Vote"
		case *pb.WhirlyMsg_NewLeaderNotify:
			msgType = "LeaderNotify"
		case *pb.WhirlyMsg_NewLeaderEcho:
			msgType = "LeaderEcho"
		case *pb.WhirlyMsg_WhirlyPing:
			msgType = "Ping"
		default:
			msgType = "Unknown"
		}
		logger.WithField("msg_type", msgType).Debug("Message details")
		return
	}

	if packet.ReceiverPublicAddress == BroadcastToAll {
		logger.WithField("node_count", len(s.Nodes)).Debug("Broadcasting message to all local nodes")
		for _, node := range s.Nodes {
			go func(n model.Consensus) {
				n.GetMsgByteEntrance() <- packet.GetMsg()
			}(node)
		}
	} else {
		node, ok2 := s.Nodes[packet.ReceiverPublicAddress]
		if !ok2 {
			logger.WithField("target", packet.ReceiverPublicAddress).Debug("Target node not found in local nodes")
		} else {
			logger.WithField("target", packet.ReceiverPublicAddress).Debug("Routing message to specific node")
			node.GetMsgByteEntrance() <- packet.GetMsg()
		}
	}
}

func (s *Sharding) handleRequest(req *pb.Request) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	logger.WithFields(logrus.Fields{
		"node_count": len(s.Nodes),
		"tx_count":   len(req.Tx),
	}).Debug("Distributing request to all nodes")

	for _, node := range s.Nodes {
		go func(n model.Consensus) {
			n.GetRequestEntrance() <- req
		}(node)
	}
}

func (s *Sharding) handleStop(address string) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node, ok2 := s.Nodes[address]
	if !ok2 {
		logger.WithField("address", address).Warn("Attempted to stop unknown node")
	} else {
		logger.WithFields(logrus.Fields{
			"address":       address,
			"active_before": s.active,
		}).Info("Stopping node")
		node.Stop()
		delete(s.Nodes, address)
		s.active -= 1
		logger.WithField("active_after", s.active).Debug("Node stopped successfully")
	}
}

func (s *Sharding) handleUpdate(address string) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node, ok2 := s.Nodes[address]
	if !ok2 {
		logger.WithField("address", address).Warn("Attempted to update unknown node")
	} else {
		logger.WithField("address", address).Debug("Updating node crypto elements")
		command := model.ExternalStatus{
			Command:        "SetCryptoElements",
			CryptoElements: s.CryptoElements,
		}
		node.UpdateExternalStatus(command)
		logger.WithField("address", address).Debug("Node updated successfully")
	}
}

func (s *Sharding) NodeManage(potSignal *PoTSignal) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	logger.WithFields(logrus.Fields{
		"committee_size": len(s.Committee),
		"self_addresses": len(potSignal.SelfPublicAddress),
		"leader":         s.LeaderPublicAddress,
	}).Info("Managing nodes for sharding")

	// Update all existing nodes with new epoch and leader information
	// This ensures daemon nodes and other nodes stay synchronized with PoT epoch changes
	s.nodesLock.Lock()
	for nodeAddr, node := range s.Nodes {
		// For daemon nodes (containing "daemonNode"), leader address should be the node itself
		// For other nodes, use the standard leader encoding
		var leaderAddress string
		if strings.Contains(nodeAddr, DaemonNodePublicAddress) {
			leaderAddress = nodeAddr // Daemon node is its own leader
		} else {
			leaderAddress = EncodeAddress(s.Name, s.SubConsensus.ConsensusID, s.LeaderPublicAddress)
		}

		logger.WithFields(logrus.Fields{
			"node":   nodeAddr,
			"epoch":  potSignal.Epoch,
			"leader": leaderAddress,
		}).Trace("Updating existing node with new epoch and leader")

		// Set leader for all epochs from 0 to current epoch
		for epoch := int64(0); epoch <= potSignal.Epoch; epoch++ {
			command := model.ExternalStatus{
				Command: "SetLeader",
				Epoch:   epoch,
				Leader:  leaderAddress,
			}
			node.UpdateExternalStatus(command)
		}

		// Update epoch
		command := model.ExternalStatus{
			Command: "SetEpoch",
			Epoch:   potSignal.Epoch,
		}
		node.UpdateExternalStatus(command)
	}
	s.nodesLock.Unlock()

	// Determine whether the leader belongs to oneself
	for _, address := range potSignal.SelfPublicAddress {
		for _, c := range s.Committee {
			if address == c && address != s.LeaderPublicAddress {
				logger.WithField("address", address).Debug("Checking/creating committee node")
				_ = s.checkNodeStauts(address)
			}
		}

		if address == s.LeaderPublicAddress {
			// This new node or committee node attempts to become a leader
			logger.WithFields(logrus.Fields{
				"address": address,
				"epoch":   potSignal.Epoch,
			}).Debug("Preparing new leader node")
			node := s.checkNodeStauts(address)

			node.NewEpochConfirmation(potSignal.Epoch, potSignal.Proof, s.Committee)
			logger.WithFields(logrus.Fields{
				"epoch":   s.controller.epoch,
				"address": address,
			}).Info("New leader is ready")
		}
	}
}

// Address has not been encoded
func (s *Sharding) checkNodeStauts(publicAddress string) model.Consensus {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	address := EncodeAddress(s.Name, s.SubConsensus.ConsensusID, publicAddress)
	node, ok := s.Nodes[address]
	if !ok {
		// The address belongs to this controller, create a new node
		logger.WithFields(logrus.Fields{
			"address":        address,
			"public_address": publicAddress,
			"epoch":          s.epoch,
		}).Info("Creating new consensus node")

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

		command3 := model.ExternalStatus{
			Command:        "SetCryptoElements",
			CryptoElements: s.CryptoElements,
		}
		newNode.UpdateExternalStatus(command3)

		logger.WithFields(logrus.Fields{
			"address": address,
			"leader":  leader,
		}).Debug("Node initialized with leader and crypto elements")

		return newNode
	} else {
		// update leader information in node
		logger.WithField("address", address).Debug("Updating existing node leader info")
		leader := EncodeAddress(s.Name, s.SubConsensus.ConsensusID, s.LeaderPublicAddress)
		// don't update epoch, the update of epoch is triggered by the new leader
		command1 := model.ExternalStatus{
			Command: "SetLeader",
			Epoch:   s.epoch,
			Leader:  leader,
		}
		node.UpdateExternalStatus(command1)
	}
	return node
}

// Address has already been encoded
func (s *Sharding) createConsensusNode(address string) model.Consensus {
	logger := logging.GetLogger().WithField("module", "WHIRLY.SHARD").WithField("sharding", s.Name)
	var c model.Consensus = nil

	switch s.SubConsensus.Type {
	case "whirly":
		switch s.SubConsensus.Whirly.Type {
		case "basic":
			logger.Warn("Basic whirly is not supported yet")
		case "simple":
			s.active += 1
			s.controller.total += 1
			logger.WithFields(logrus.Fields{
				"address":      address,
				"node_id":      s.controller.total,
				"consensus_id": s.SubConsensus.ConsensusID,
				"active_count": s.active,
			}).Info("Creating SimpleWhirly node")
			c = simpleWhirly.NewSimpleWhirly(
				int64(s.controller.total),
				s.SubConsensus.ConsensusID,
				s.epoch,
				s.controller.Config,
				s,
				s,
				s.controller.Log,
				address,
				s.controller.StopEntrance,
				s.controller.UpdateEntrance,
			)
			s.Nodes[address] = c
		case "censorship-resistance":
			s.active += 1
			s.controller.total += 1
			logger.WithFields(logrus.Fields{
				"address":      address,
				"node_id":      s.controller.total,
				"consensus_id": s.SubConsensus.ConsensusID,
				"active_count": s.active,
			}).Info("Creating CensorshipResistant Whirly node")
			c = crWhirly.NewCrWhirly(
				int64(s.controller.total),
				s.SubConsensus.ConsensusID,
				s.epoch,
				s.controller.Config,
				s,
				s,
				s.controller.Log,
				address,
				s.controller.StopEntrance,
				s.controller.UpdateEntrance,
			)
			s.Nodes[address] = c
		default:
			logger.WithField("whirly_type", s.SubConsensus.Whirly.Type).Error("Unsupported whirly type")
		}
	default:
		logger.WithField("consensus_type", s.SubConsensus.Type).Error("Unsupported consensus type")
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
