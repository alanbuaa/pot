package p2p

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type BaseP2P struct {
	nodes             map[int64]*config.NodeInfo // 使用 Nodes map
	log               *logrus.Entry
	id                int64
	peerid            string
	output            chan<- []byte
	rpcServer         *grpc.Server
	discoveredNodes   map[int64]string      // 已发现的节点 nodeID -> address
	discoveryCallback NodeDiscoveryCallback // 节点发现回调
	discoveryMu       sync.RWMutex          // 保护 discoveredNodes
	pb.UnimplementedP2PServer
}

func NewBaseP2P(cfg *config.Config, log *logrus.Entry, id int64) (*BaseP2P, string, error) {
	info, err := cfg.GetNodeFromSet(id)
	if err != nil {
		return nil, "", err
	}
	port := info.P2PAddress[strings.Index(info.P2PAddress, ":"):]
	listen, err := net.Listen("tcp", port)
	utils.PanicOnError(err)
	rpcServer := grpc.NewServer()
	utils.PanicOnError(err)
	bp := &BaseP2P{
		nodes:                  cfg.GetAllNodes(), // 使用 GetAllNodes 获取 Nodes
		log:                    log,
		id:                     id,
		output:                 nil,
		rpcServer:              rpcServer,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
		peerid:                 cfg.Node.P2PAddress,
		discoveredNodes:        make(map[int64]string),
	}
	// 将自己添加到已发现节点列表
	bp.discoveredNodes[id] = cfg.Node.P2PAddress
	pb.RegisterP2PServer(rpcServer, bp)
	log.Infof("[UpgradeableConsensus] Server start at %s", listen.Addr().String())
	go rpcServer.Serve(listen)
	nid := cfg.Node.P2PAddress
	return bp, nid, nil
}

func (bp *BaseP2P) Subscribe(topic []byte) error {
	return nil
}

func (bp *BaseP2P) UnSubscribe(topic []byte) error {
	return nil
}

func (bp *BaseP2P) GetPeerID() string {
	return bp.nodes[bp.id].P2PAddress
}

func (bp *BaseP2P) GetP2PType() string {
	return "p2p"
}

// func (bp *BaseP2P) SetUnicastReceiver(receiver MsgReceiver) {
// 	bp.output = receiver.GetMsgByteEntrance()
// }

func (bp *BaseP2P) SetReceiver(ch chan<- []byte) {
	bp.output = ch
}

func (bp *BaseP2P) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	nodes := bp.nodes
	// bp.log.Debugf("broadcast packet to %d nodes", len(nodes)-1)
	for _, node := range nodes {
		if node.ID == bp.id {
			continue
		}
		err := bp.Unicast(node.P2PAddress, msgByte, consensusID, topic)
		if err != nil {
			bp.log.WithError(err).Warn("send msg failed")
			return err
		}
		// bp.log.Debugf("unicast to %s done", node.P2PAddress)
	}
	return nil
}

func (bp *BaseP2P) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	// conn, err := grpc.Dial(address, grpc.WithInsecure())
	// bp.log.Trace("[basep2p] unicast")
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		bp.log.Warn("dial", address, "failed")
		return err
	}
	client := pb.NewP2PClient(conn)
	packet := new(pb.Packet)
	err = proto.Unmarshal(msgByte, packet)
	utils.PanicOnError(err)
	// bp.log.Debug("unicast packet to", address)
	_, err = client.Send(context.Background(), packet)
	if err != nil {
		bp.log.WithError(err).Warn("send to ", address, "failed")
		return err
	}
	conn.Close()
	return nil
}

// BaseP2P implements P2PServer
func (bp *BaseP2P) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	bp.log.WithFields(logrus.Fields{
		"packet_type":  in.Type.String(),
		"consensus_id": in.ConsensusID,
		"epoch":        in.Epoch,
	}).Trace("[TRACE-1] P2P layer received packet")

	bytePacket, err := proto.Marshal(in)
	if err != nil {
		bp.log.Warn("marshal packet failed")
		return nil, err
	}
	if bp.output != nil {
		bp.log.Trace("[TRACE-1.1] Forwarding packet to consensus engine")
		bp.output <- bytePacket
	} else {
		bp.log.Warn("[BaseP2P] output nil")
	}
	return &pb.Empty{}, nil
}

func (bp *BaseP2P) Stop() {
	bp.rpcServer.Stop()
}

// ========== 节点发现相关方法 ==========

// DiscoverPeers 启动静态节点发现（BaseP2P 使用配置文件中的节点列表）
func (bp *BaseP2P) DiscoverPeers() error {
	bp.log.Info("[BaseP2P] Starting static node discovery")

	// BaseP2P 使用预定义的静态节点列表
	// 通过 gRPC 健康检查验证节点可达性
	for nodeID, nodeInfo := range bp.nodes {
		if nodeID == bp.id {
			continue // 跳过自己
		}

		// 尝试连接验证节点是否在线
		go bp.probeNode(nodeID, nodeInfo.P2PAddress)
	}

	return nil
}

// probeNode 探测节点是否在线
func (bp *BaseP2P) probeNode(nodeID int64, address string) {
	bp.log.WithFields(logrus.Fields{
		"node_id": nodeID,
		"address": address,
	}).Debug("[BaseP2P] Probing node")

	// 尝试建立 gRPC 连接
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		bp.log.WithError(err).WithField("node_id", nodeID).Debug("[BaseP2P] Node probe failed")
		return
	}
	defer conn.Close()

	// 记录已发现的节点
	bp.discoveryMu.Lock()
	if _, exists := bp.discoveredNodes[nodeID]; !exists {
		bp.discoveredNodes[nodeID] = address
		bp.discoveryMu.Unlock()

		bp.log.WithFields(logrus.Fields{
			"node_id": nodeID,
			"address": address,
			"total":   len(bp.discoveredNodes),
		}).Info("[BaseP2P] Node discovered")

		// 通知回调
		if bp.discoveryCallback != nil {
			if err := bp.discoveryCallback.OnNodeDiscovered(nodeID, address); err != nil {
				bp.log.WithError(err).Warn("[BaseP2P] Discovery callback failed")
			}
		}
	} else {
		bp.discoveryMu.Unlock()
	}
}

// RegisterNodeDiscoveryCallback 注册节点发现回调
func (bp *BaseP2P) RegisterNodeDiscoveryCallback(callback NodeDiscoveryCallback) {
	bp.discoveryCallback = callback
	bp.log.Info("[BaseP2P] Node discovery callback registered")
}

// GetDiscoveredNodes 获取已发现的节点列表
func (bp *BaseP2P) GetDiscoveredNodes() map[int64]string {
	bp.discoveryMu.RLock()
	defer bp.discoveryMu.RUnlock()

	// 返回副本避免并发修改
	result := make(map[int64]string, len(bp.discoveredNodes))
	for id, addr := range bp.discoveredNodes {
		result[id] = addr
	}
	return result
}

// GetNodeCount 获取已发现的节点数量
func (bp *BaseP2P) GetNodeCount() int {
	bp.discoveryMu.RLock()
	defer bp.discoveryMu.RUnlock()
	return len(bp.discoveredNodes)
}

// WaitForNodes 等待指定数量的节点上线
func (bp *BaseP2P) WaitForNodes(count int, timeoutSec int) error {
	bp.log.WithFields(logrus.Fields{
		"required": count,
		"timeout":  timeoutSec,
	}).Info("[BaseP2P] Waiting for nodes to come online")

	timeout := time.Duration(timeoutSec) * time.Second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ticker.C:
			current := bp.GetNodeCount()
			bp.log.WithFields(logrus.Fields{
				"current":  current,
				"required": count,
				"elapsed":  time.Since(start).Seconds(),
			}).Debug("[BaseP2P] Waiting for nodes...")

			if current >= count {
				bp.log.WithField("count", current).Info("[BaseP2P] Required nodes are online")
				return nil
			}

			if time.Since(start) >= timeout {
				return fmt.Errorf("timeout waiting for nodes: got %d, required %d", current, count)
			}
		}
	}
}
