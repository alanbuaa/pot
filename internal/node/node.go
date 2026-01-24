package node

import (
	"context"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	id     int64
	config *config.Config
	// pot        *pot.PoTEngine
	consensus  model.Consensus
	rpcServer  *grpc.Server
	log        *logrus.Entry
	p2pAdaptor p2p.P2PAdaptor
	pb.UnimplementedP2PServer
}

func NewNode(id int64, cfgPath string, log *logrus.Entry) *Node {
	// Create dedicated logger for this node to avoid log conflicts in multi-node scenarios
	log = log.WithField("module", "NODE").WithField("id", id)

	log.Info("Initializing node")

	// Load configuration
	cfg, err := config.NewConfig(cfgPath)
	utils.PanicOnError(err)
	log.WithField("config_path", cfgPath).Info("Configuration loaded successfully")

	info, err := cfg.GetNodeFromSet(id)
	utils.PanicOnError(err)

	// Initialize P2P network
	log.Info("Building P2P network")
	p, nid, err := p2p.BuildP2P(cfg, log, id)
	if err != nil {
		log.WithError(err).Fatal("Failed to build P2P network")
	}
	if p == nil {
		log.Fatal("P2P network is nil")
	}
	log.WithFields(logrus.Fields{
		"network_id": nid,
		"p2p_type":   p.GetP2PType(),
	}).Info("P2P network initialized successfully")

	// Initialize API server
	_, apiPortStr := utils.GetAddressPort(info.ApiServerAddress)
	apiPort, err := strconv.Atoi(apiPortStr)
	if err != nil {
		log.WithError(err).Fatal("Failed to parse API port")
	}
	log.WithField("port", apiPort).Info("Initializing API server")
	apiServer := apis.NewApiServer(&apis.Config{Port: apiPort}, log)

	// Initialize executor
	log.Info("Building executor")
	e := executor.BuildExecutor(cfg.Executor, log)
	if e == nil {
		log.Fatal("Failed to build executor")
	}
	log.Info("Executor initialized successfully")

	// Initialize consensus
	log.Info("Building consensus")
	c, err := consensus.BuildConsensus(id, 10001, cfg.Consensus, e, p, log)
	utils.PanicOnLogError(err, log)
	log.WithField("consensus_type", c.GetConsensusType()).Info("Consensus initialized successfully")

	// Register API services
	RegisterApiServices(apiServer, c, log)

	// Create RPC server
	log.Info("Creating gRPC server")
	rpcServer := grpc.NewServer()

	node := &Node{
		id:                     id,
		config:                 cfg,
		consensus:              c,
		rpcServer:              rpcServer,
		p2pAdaptor:             p,
		log:                    log,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
	}

	// 注册节点发现回调
	node.registerNodeDiscoveryCallback()

	// 启动节点发现
	log.Info("Starting node discovery")
	if err := p.DiscoverPeers(); err != nil {
		log.WithError(err).Warn("Failed to start node discovery")
	}

	// 非动态共识需要等待足够节点上线
	if !config.IsDynamicConsensus(cfg) {
		log.WithFields(logrus.Fields{
			"consensus_type": c.GetConsensusType(),
			"required_nodes": cfg.Total,
		}).Info("Static consensus detected, waiting for nodes to be online")

		// 等待节点上线（超时30秒）
		if err := p.WaitForNodes(cfg.Total, 30); err != nil {
			log.WithError(err).Warn("Not all nodes are online, proceeding anyway")
		} else {
			log.WithField("node_count", p.GetNodeCount()).Info("All required nodes are online")
		}
	}

	// Register P2P server
	pb.RegisterP2PServer(rpcServer, node)
	log.Info("P2P server registered")

	// Start RPC listener
	listen, err := net.Listen("tcp", info.RpcAddress)
	if err != nil {
		log.WithError(err).WithField("address", info.RpcAddress).Fatal("Failed to listen on TCP address")
	}
	log.WithField("address", listen.Addr().String()).Info("RPC server listening")
	go func() {
		if err := rpcServer.Serve(listen); err != nil {
			log.WithError(err).Error("RPC server stopped with error")
		}
	}()

	// Start API server
	log.Info("Starting API server")
	apiServer.Start()

	log.Info("Node initialization completed successfully")
	return node
}

func (node *Node) Stop() {
	node.log.Info("Stopping node")

	node.log.Info("Stopping RPC server")
	node.rpcServer.Stop()

	node.log.Info("Stopping consensus")
	node.consensus.Stop()

	node.log.Info("Stopping P2P adaptor")
	node.p2pAdaptor.Stop()

	node.log.Info("Node stopped successfully")
}

// Send 实现 P2PServer 接口 - 处理接收到的 P2P 消息。 这种命名真的很抽象！
func (node *Node) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	node.log.WithField("packet_type", in.Type).Trace("Received packet")

	if in.Type != pb.PacketType_CLIENTPACKET {
		// Handle consensus message
		pbMsg := new(pb.PoTMessage)
		if err := proto.Unmarshal(in.Msg, pbMsg); err != nil {
			node.log.WithError(err).Warn("Failed to unmarshal PoT message")
			return &pb.Empty{}, nil
		}
		node.log.Trace("Forwarding consensus message to consensus layer")
		node.consensus.GetMsgByteEntrance() <- in.GetMsg()
		return &pb.Empty{}, nil
	}

	// Handle client request
	request := new(pb.Request)
	if err := proto.Unmarshal(in.Msg, request); err != nil {
		node.log.WithError(err).Warn("Failed to decode client request")
		return &pb.Empty{}, nil
	}
	node.log.WithField("chain_id", request.ChainID).Trace("Forwarding client request to consensus")
	node.consensus.GetRequestEntrance() <- request
	return &pb.Empty{}, nil
}
