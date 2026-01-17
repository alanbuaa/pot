package nodeController

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"google.golang.org/protobuf/proto"
)

const DaemonNodePublicAddress string = "daemonNode"
const BroadcastToAll string = "ALL"

// type ControllerMessage struct {
// 	Data         []byte
// 	Receiver     string
// 	ShardingName string
// }

type PoTSignal struct {
	Epoch             int64
	Proof             []byte
	ID                int64
	SelfPublicAddress []string
	Shardings         []PoTSharding
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

	// PoT
	PoTByteEntrance chan []byte // receive msg
	StopEntrance    chan string // receive the publicAddress of the node that should be stopped
	UpdateEntrance  chan string

	total int
	epoch int64 // The height of the POT chain

	// WhrilyNodes map[string]map[string]*SimpleWhirlyImpl // ShardingName -> NodeAddress -> Node
	Shardings map[string]*Sharding

	shardingsLock sync.Mutex
	Committee     []string
}

func NewNodeController(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *NodeController {
	log = logging.GetLogger().WithField("module", "WHIRLY.NODECTRL").WithField("c_id", cid)
	log.Info("Initializing NodeController")
	log.Debug("Starting NodeController with configuration")
	ctx, cancel := context.WithCancel(context.Background())
	nc := &NodeController{
		PeerId:         p2pAdaptor.GetPeerID(),
		ConsensusID:    cid,
		Config:         cfg,
		Executor:       exec,
		p2pAdaptor:     p2pAdaptor,
		Log:            log,
		cancel:         cancel,
		epoch:          0,
		StopEntrance:   make(chan string, 10),
		UpdateEntrance: make(chan string, 10),
		Shardings:      make(map[string]*Sharding),
		// WhrilyNodes: make(map[string]map[string]*SimpleWhirlyImpl),
	}

	nc.MsgByteEntrance = make(chan []byte, 10)
	nc.RequestByteEntrance = make(chan *pb.Request, 10)
	nc.PoTByteEntrance = make(chan []byte, 10)

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

//	func (nc *NodeController) DecodeMsgByte(msgByte []byte) (*ControllerMessage, error) {
//		msg := new(ControllerMessage)
//		err := json.Unmarshal(msgByte, msg)
//		if err != nil {
//			return nil, err
//		}
//		return msg, nil
//	}

func (nc *NodeController) Stop() {
	nc.Log.Info("Stopping NodeController")
	nc.Log.Debug("Canceling context")
	nc.cancel()
	nc.Log.Debug("Closing message channels")
	close(nc.MsgByteEntrance)
	close(nc.RequestByteEntrance)
	close(nc.PoTByteEntrance)
	close(nc.StopEntrance)
	close(nc.UpdateEntrance)
	nc.Log.Info("NodeController stopped successfully")
}

func (nc *NodeController) receiveMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-nc.RequestByteEntrance:
			if !ok {
				return // closed
			}

			if req.Sharding == nil {
				nc.Log.Warn("Received request with nil sharding field, dropping request")
				continue
			}
			nc.Log.WithField("sharding", string(req.Sharding)).Debug("Received request")
			shardingName := string(req.Sharding)
			nc.shardingsLock.Lock()
			sharding, ok2 := nc.Shardings[shardingName]
			if !ok2 {
				nc.Log.WithField("sharding", shardingName).Error("Failed to route request: sharding not found")
			} else {
				sharding.handleRequest(req)
			}
			nc.shardingsLock.Unlock()
		case msgByte, ok := <-nc.MsgByteEntrance:
			if !ok {
				nc.Log.Debug("MsgByteEntrance channel closed")
				return // closed
			}
			packet, err := DecodePacket(msgByte)
			if err != nil {
				nc.Log.WithError(err).Error("Failed to decode packet, dropping message")
				continue
			}
			go nc.handleMsg(packet)
		case address, ok := <-nc.StopEntrance:
			if !ok {
				return // closed
			}
			go nc.handleStop(address)
		case address, ok := <-nc.UpdateEntrance:
			if !ok {
				return // closed
			}
			go nc.handleUpdate(address)
		case potSignal, ok := <-nc.PoTByteEntrance:
			if !ok {
				return // closed
			}
			go nc.handlePotSignal(potSignal)
		}
	}
}

// =============================================
// NodeController function 1: message forwarding
// =============================================
func (nc *NodeController) handleMsg(packet *pb.Packet) {

	// msgByte := packet.GetMsg()
	// msg := new(pb.WhirlyMsg)
	// err := proto.Unmarshal(msgByte, msg)
	// if err != nil {
	// 	nc.Log.Error("Unmarshal error: ", err)
	// }
	// fmt.Printf("Payload: %+v\n", msg.Payload)
	// switch msg.Payload.(type) {
	// case *pb.WhirlyMsg_Request:
	// 	fmt.Printf("msg: %+v\n", msg.GetRequest())
	// case *pb.WhirlyMsg_WhirlyProposal:
	// 	fmt.Printf("msg: %+v\n", msg.GetCrWhirlyProposal())
	// case *pb.WhirlyMsg_WhirlyVote:
	// 	fmt.Printf("msg: %+v\n", msg.GetCrWhirlyVote())
	// case *pb.WhirlyMsg_NewLeaderNotify:
	// 	fmt.Printf("msg: %+v\n", msg.GetNewLeaderNotify())
	// case *pb.WhirlyMsg_NewLeaderEcho:
	// 	fmt.Printf("msg: %+v\n", msg.GetNewLeaderEcho())
	// case *pb.WhirlyMsg_WhirlyPing:
	// 	fmt.Printf("msg: %+v\n", msg.GetWhirlyPing())
	// default:
	// 	nc.Log.Warn("Receive unsupported msg")
	// 	return
	// }

	// nc.Log.WithFields(logrus.Fields{
	// 	"ReceiverShardingName":  packet.ReceiverShardingName,
	// 	"Epoch":                 packet.Epoch,
	// 	"ReceiverPublicAddress": packet.ReceiverPublicAddress,
	// 	"Msg":                   hex.EncodeToString(packet.Msg)[0:10],
	// }).Warn("handleMsg in NodeController")

	if packet.ReceiverShardingName == "" {
		nc.Log.Warn("Dropping message with empty receiver sharding name")
		return
	}
	nc.shardingsLock.Lock()
	sharding, ok := nc.Shardings[packet.ReceiverShardingName]
	if !ok {
		nc.Log.WithFields(logrus.Fields{
			"sharding": packet.ReceiverShardingName,
			"epoch":    packet.Epoch,
		}).Debug("Dropping message for non-existent sharding")
	} else {
		nc.Log.WithFields(logrus.Fields{
			"sharding": packet.ReceiverShardingName,
			"epoch":    packet.Epoch,
		}).Debug("Routing message to sharding")
		go sharding.handleMsg(packet)
	}
	nc.shardingsLock.Unlock()
}

// =============================================
// NodeController function 2: node management
// =============================================
func (nc *NodeController) handlePotSignal(potSignalBytes []byte) {
	potSignal := &PoTSignal{}
	err := json.Unmarshal(potSignalBytes, potSignal)
	if err != nil {
		nc.Log.WithError(err).Error("Failed to unmarshal PoT signal")
		return
	}

	// Ignoring pot signals from old epochs (but always accept epoch 0 for genesis)
	if potSignal.Epoch < nc.epoch {
		nc.Log.WithFields(logrus.Fields{
			"signal_epoch":  potSignal.Epoch,
			"current_epoch": nc.epoch,
		}).Debug("Ignoring PoT signal from old epoch")
		return
	} else if potSignal.Epoch > nc.epoch {
		nc.Log.WithFields(logrus.Fields{
			"old_epoch": nc.epoch,
			"new_epoch": potSignal.Epoch,
		}).Info("Processing PoT signal for new epoch")
		nc.epoch = potSignal.Epoch
	} else {
		// potSignal.Epoch == nc.epoch
		// For epoch 0 (genesis), we should still process it to create initial sharding
		nc.Log.WithFields(logrus.Fields{
			"epoch": nc.epoch,
		}).Debug("Processing PoT signal for current epoch (genesis or update)")
	}

	nc.ShardManage(potSignal)
	nc.NodeManage(potSignal)
}

func (nc *NodeController) ShardManage(potSignal *PoTSignal) {
	for _, s := range potSignal.Shardings {
		nc.shardingsLock.Lock()
		localSharding, ok := nc.Shardings[s.Name]
		if !ok {
			// create a new sharding
			nc.Log.WithFields(logrus.Fields{
				"sharding":     s.Name,
				"consensus_id": s.SubConsensus.ConsensusID,
				"leader":       s.LeaderPublicAddress,
			}).Info("Creating new sharding")
			ns := NewSharding(s.Name, nc, s)
			nc.Shardings[s.Name] = ns
		} else {
			// update sharding
			localSharding.LeaderPublicAddress = s.LeaderPublicAddress
			localSharding.Committee = s.Committee
			if localSharding.SubConsensus.ConsensusID != s.SubConsensus.ConsensusID {
				// 此时意味者发送了共识切换，需要为新共识创建一个 DaemonNode 节点
				// 请注意，创建节点的共识类型，是依据 localSharding.SubConsensus 指定的，因此在创建前需要先更新 localSharding.SubConsensus
				nc.Log.WithFields(logrus.Fields{
					"sharding":         s.Name,
					"old_consensus_id": localSharding.SubConsensus.ConsensusID,
					"new_consensus_id": s.SubConsensus.ConsensusID,
				}).Info("Creating DaemonNode for consensus switch")
				localSharding.SubConsensus = &s.SubConsensus
				address := EncodeAddress(localSharding.Name+nc.PeerId, s.SubConsensus.ConsensusID, DaemonNodePublicAddress)
				localSharding.createConsensusNode(address)
			} else {
				localSharding.SubConsensus = &s.SubConsensus
			}
		}
		nc.shardingsLock.Unlock()
	}

	nc.shardingsLock.Lock()
	for name := range nc.Shardings {
		flag := 0 // 标识此分区是否应该删除
		for _, s := range potSignal.Shardings {
			if name == s.Name {
				flag = 1
				break
			}
		}
		if flag == 0 {
			// delete a sharding
			nc.Log.WithField("sharding", name).Info("Deleting obsolete sharding")
			nc.Shardings[name].Stop()
			delete(nc.Shardings, name)
		}
	}
	nc.shardingsLock.Unlock()
}

func (nc *NodeController) NodeManage(potSignal *PoTSignal) {
	nc.Log.WithFields(logrus.Fields{
		"epoch":              nc.epoch,
		"self_address_count": len(potSignal.SelfPublicAddress),
		"sharding_count":     len(potSignal.Shardings),
	}).Info("Managing nodes for new epoch")
	nc.Log.WithField("addresses", len(potSignal.SelfPublicAddress)).Debug("Self public addresses count")
	// fmt.Printf("%+v\n", potSignal.Shardings)
	// println(potSignal.Shardings)

	for _, s := range nc.Shardings {
		go s.NodeManage(potSignal)
	}

}

// stop a committee node
func (nc *NodeController) handleStop(address string) {
	shardingName, _, rawAddress := DecodeAddress(address)

	// For DaemonNode, the shardingName might contain PeerId (e.g., "0x1QmbF5...")
	// We need to find the actual sharding by removing the PeerId suffix
	nc.shardingsLock.Lock()
	sharding, ok := nc.Shardings[shardingName]

	// If not found and this is a DaemonNode, try to find by removing PeerId
	if !ok && rawAddress == DaemonNodePublicAddress && len(nc.PeerId) > 0 {
		// Try to remove PeerId suffix from shardingName
		if strings.HasSuffix(shardingName, nc.PeerId) {
			originalShardingName := strings.TrimSuffix(shardingName, nc.PeerId)
			sharding, ok = nc.Shardings[originalShardingName]
		}
	}

	defer nc.shardingsLock.Unlock()
	if !ok {
		nc.Log.WithFields(logrus.Fields{
			"sharding": shardingName,
			"address":  address,
		}).Error("Failed to stop node: sharding not found")
		return
	}
	nc.Log.WithFields(logrus.Fields{
		"sharding": shardingName,
		"address":  rawAddress,
	}).Debug("Stopping node")
	go sharding.handleStop(address)
}

// update a committee node
func (nc *NodeController) handleUpdate(address string) {
	shardingName, _, rawAddress := DecodeAddress(address)

	// For DaemonNode, the shardingName might contain PeerId (e.g., "0x1QmbF5...")
	// We need to find the actual sharding by removing the PeerId suffix
	nc.shardingsLock.Lock()
	sharding, ok := nc.Shardings[shardingName]

	// If not found and this is a DaemonNode, try to find by removing PeerId
	if !ok && rawAddress == DaemonNodePublicAddress && len(nc.PeerId) > 0 {
		// Try to remove PeerId suffix from shardingName
		if strings.HasSuffix(shardingName, nc.PeerId) {
			originalShardingName := strings.TrimSuffix(shardingName, nc.PeerId)
			sharding, ok = nc.Shardings[originalShardingName]
		}
	}

	if !ok {
		nc.Log.WithFields(logrus.Fields{
			"sharding": shardingName,
			"address":  address,
		}).Error("Failed to update node: sharding not found")
	} else {
		nc.Log.WithFields(logrus.Fields{
			"sharding": shardingName,
			"address":  rawAddress,
		}).Debug("Updating node")
		go sharding.handleUpdate(address)
	}
	nc.shardingsLock.Unlock()
}

func EncodeAddress(shardingName string, consensusID int64, address string) string {
	consensusIDStr := strconv.FormatInt(consensusID, 10)
	return shardingName + "-" + consensusIDStr + "-" + address
}

// Input: address that is encoded
// Output: shardingName, consensusID, rawAddress
func DecodeAddress(address string) (string, int64, string) {
	logger := logging.GetLogger().WithField("module", "WHIRLY.NODECTRL")
	res := strings.Split(address, "-")
	if len(res) < 3 {
		logger.WithField("address", address).Error("Failed to decode address: illegal parameter format")
		return "", 0, ""
	}

	// Handle the case where shardingName might contain additional parts (like PeerId)
	// Format: shardingName-consensusID-rawAddress or shardingNameWithPeerId-consensusID-rawAddress
	// We need to extract consensusID from the second-to-last segment
	rawAddress := res[len(res)-1]
	consensusIDStr := res[len(res)-2]

	consensusID, err := strconv.ParseInt(consensusIDStr, 10, 64)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"address":      address,
			"consensus_id": consensusIDStr,
		}).Error("Failed to decode address: illegal consensus ID")
		return "", 0, ""
	}

	// Everything before consensusID is the shardingName (may include PeerId)
	shardingName := strings.Join(res[:len(res)-2], "-")

	return shardingName, consensusID, rawAddress
}

func DecodePacket(packet []byte) (*pb.Packet, error) {
	p := new(pb.Packet)
	err := proto.Unmarshal(packet, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
