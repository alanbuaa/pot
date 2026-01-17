package node

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	id int64
	// pot        *pot.PoTEngine
	consensus  model.Consensus
	rpcServer  *grpc.Server
	log        *logrus.Entry
	p2pAdaptor p2p.P2PAdaptor
	pb.UnimplementedP2PServer
}

func NewNode(id int64, cfgPath string) *Node {
	// Create dedicated logger for this node to avoid log conflicts in multi-node scenarios
	nodeLogger := logging.CreateNodeLogger(cfgPath, id)
	log := nodeLogger.WithField("module", "NODE").WithField("id", id)

	log.Info("Initializing node")

	// Load configuration
	cfg, err := config.NewConfig(cfgPath, id)
	utils.PanicOnError(err)
	log.WithField("config_path", cfgPath).Info("Configuration loaded successfully")

	info := cfg.GetNodeInfo(id)
	port := info.RpcAddress[strings.Index(info.RpcAddress, ":"):]

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
	apiPort, err := strconv.Atoi(strings.Split(cfg.RestServerAddress, ":")[1])
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
		consensus:              c,
		rpcServer:              rpcServer,
		p2pAdaptor:             p,
		log:                    log,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
	}

	// Register P2P server
	pb.RegisterP2PServer(rpcServer, node)
	log.Info("P2P server registered")

	// Start RPC listener
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.WithError(err).WithField("port", port).Fatal("Failed to listen on TCP port")
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

// func (node *Node) Request(ctx context.Context, in *pb.HeaderRequest) (*pb.Header, error) {
//	node.log.Infoln("[node]\t receive request for header")
//	hash := in.Hashes
//	st := node.pot.GetBlockStorage()
//	header, err := st.Get(hash)
//	if err != nil {
//		node.log.Error("get block err for ", err)
//		return nil, err
//	} else {
//		return header.ToProto(), nil
//	}
//
// }
//
// func (node *Node) PoTresRequest(ctx context.Context, request *pb.PoTRequest) (*pb.PoTResponse, error) {
//	node.log.Infoln("[node]\t receive request for header")
//	epoch := request.GetEpoch()
//	st := node.pot.GetBlockStorage()
//	res, err := st.GetVDFresbyEpoch(epoch)
//	if err != nil {
//		node.log.Error("[node]\t get PoTProof error ", err)
//		return nil, err
//	} else {
//		return &pb.PoTResponse{
//			Epoch:   epoch,
//			Proof:   res,
//			Address: node.id,
//		}, nil
//	}
// }

func RegisterApiServices(apiServer *apis.ApiServer, c model.Consensus, log *logrus.Entry) {
	consensusType := c.GetConsensusType()
	log.WithField("consensus_type", consensusType).Info("Registering API services")

	switch consensusType {
	case "pot":
		if potEngine, ok := c.(*pot.PoTEngine); ok {
			// Register transaction API service
			apiServer.RegisterPotService(apis.NewPotWorkerAdapter(potEngine.Worker))
			log.Info("Registered PoT transaction API service")

			// Register monitoring API service
			apiServer.RegisterMonitorService(apis.NewPotMonitorAdapter(potEngine, log))
			log.Info("Registered PoT monitoring API service")
		} else {
			log.Error("Failed to cast consensus to PoTEngine for API registration")
		}
	// case "pow":
	// 	apiServer.RegisterPowService(apis.NewPowWorkerAdapter(cons.worker))
	// 	log.Info("Registered PoW API service")
	default:
		log.WithField("consensus_type", consensusType).Info("No specific API service to register for this consensus type")
	}
}
