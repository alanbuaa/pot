package simpleWhirly

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
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
	log.Debug("[Node Controller] starting")
	ctx, cancel := context.WithCancel(context.Background())
	nc := &NodeController{
		PeerId:       p2pAdaptor.GetPeerID(),
		ConsensusID:  cid,
		Config:       cfg,
		Executor:     exec,
		p2pAdaptor:   p2pAdaptor,
		Log:          log.WithField("consensus id", cid),
		cancel:       cancel,
		epoch:        0,
		StopEntrance: make(chan string, 10),
		Shardings:    make(map[string]*Sharding),
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
func DecodePacket(packet []byte) (*pb.Packet, error) {
	p := new(pb.Packet)
	err := proto.Unmarshal(packet, p)
	if err != nil {
		return nil, err
	}
	return p, nil
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
		case req, ok := <-nc.RequestByteEntrance:
			if !ok {
				return // closed
			}

			if req.Sharding == nil {
				nc.Log.Warn("Invaild Request")
				continue
			}
			shardingName := string(req.Sharding)
			nc.shardingsLock.Lock()
			sharding, ok2 := nc.Shardings[shardingName]
			if !ok2 {
				nc.Log.Warn("Receive request error: Invaild sharding name")
			} else {
				sharding.handleRequest(req)
			}
			nc.shardingsLock.Unlock()
		case msgByte, ok := <-nc.MsgByteEntrance:
			if !ok {
				return // closed
			}
			packet, err := DecodePacket(msgByte)
			if err != nil {
				nc.Log.WithError(err).Warn("decode packet failed")
				continue
			}
			go nc.handleMsg(packet)
		case address, ok := <-nc.StopEntrance:
			if !ok {
				return // closed
			}
			go nc.handleStop(address)
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
	if packet.ReceiverShardingName == "" {
		nc.Log.Warn("Receive message error: Invaild sharding name")
		return
	}
	nc.shardingsLock.Lock()
	sharding, ok := nc.Shardings[packet.ReceiverShardingName]
	if !ok {
		nc.Log.Warn("Receive message error: Sharding name does not exist")
	} else {
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
		nc.Log.WithField("error", err.Error()).Error("Unmarshal potSignal failed.")
		return
	}

	// Ignoring pot signals from old epochs
	if potSignal.Epoch <= nc.epoch {
		return
	} else {
		nc.epoch = potSignal.Epoch
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
			nc.Log.Info("create a new sharding: ", s.Name)
			ns := NewSharding(s.Name, nc, s)
			nc.Shardings[s.Name] = ns
		} else {
			// update sharding
			localSharding.LeaderPublicAddress = s.LeaderPublicAddress
			localSharding.Committee = s.Committee
			if localSharding.SubConsensus.ConsensusID != s.SubConsensus.ConsensusID {
				// 此时意味者发送了共识切换，需要为新共识创建一个 DaemonNode 节点
				// 请注意，创建节点的共识类型，是依据 localSharding.SubConsensus 指定的，因此在创建前需要先更新 localSharding.SubConsensus
				nc.Log.Info("create a new consensus DaemonNode in ", s.Name, " for consensusID: ", s.SubConsensus.ConsensusID)
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
			nc.Log.Info("delete a sharding")
			nc.Shardings[name].Stop()
			delete(nc.Shardings, name)
		}
	}
	nc.shardingsLock.Unlock()
}

func (nc *NodeController) NodeManage(potSignal *PoTSignal) {
	fmt.Println("========================", nc.epoch, "========================")
	fmt.Println("SelfPublicAddress: ", potSignal.SelfPublicAddress)
	// fmt.Printf("%+v\n", potSignal.Shardings)
	// println(potSignal.Shardings)

	for _, s := range nc.Shardings {
		go s.NodeManage(potSignal)
	}

}

// stop a committee node
func (nc *NodeController) handleStop(address string) {
	shardingName, _, _ := DecodeAddress(address)
	nc.shardingsLock.Lock()
	sharding, ok := nc.Shardings[shardingName]
	if !ok {
		// create a new sharding
		nc.Log.Warn("Stop Node Error: Sharding does not exist")
	} else {
		go sharding.handleStop(address)
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
	res := strings.Split(address, "-")
	if len(res) != 3 {
		println("DecodeAddress Error: Illegal parameter")
		return "", 0, ""
	}
	shardingName := res[0]
	rawAddress := res[2]

	consensusID, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		println("DecodeAddress Error: Illegal ConsensusID")
		return "", 0, ""
	}
	return shardingName, consensusID, rawAddress
}
