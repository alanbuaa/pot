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

func NewNode(id int64) *Node {
	cfg, err := config.NewConfig("config/config.yaml", id)
	utils.PanicOnError(err)
	info := cfg.GetNodeInfo(id)
	log := logging.GetLogger().WithField("id", id)
	port := info.RpcAddress[strings.Index(info.RpcAddress, ":"):]

	p, nid, err := p2p.BuildP2P(cfg.P2P, log, id)
	if p == nil {
		panic("build p2p failed")
	}
	log.WithFields(logrus.Fields{
		"nid":  nid,
		"type": p.GetP2PType(),
	}).Info("get network id")
	utils.PanicOnError(err)
	log.WithField("nid", nid).Info("get network id")
	utils.PanicOnError(err)

	// startup api server
	apiPort, _ := strconv.Atoi(strings.Split(cfg.RestServerAddress, ":")[1])
	apiServer := apis.NewApiServer(&apis.Config{Port: apiPort}, log)

	e := executor.BuildExecutor(cfg.Executor, log)
	c := consensus.BuildConsensus(id, 10001, cfg.Consensus, e, p, log)

	RegisterApiServices(apiServer, c, log)

	if c == nil {
		panic("Initialize consensus failed")
	}

	// Start RPC server
	rpcServer := grpc.NewServer()
	utils.PanicOnError(err)

	node := &Node{
		id:                     id,
		consensus:              c,
		rpcServer:              rpcServer,
		p2pAdaptor:             p,
		log:                    log,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
	}

	pb.RegisterP2PServer(rpcServer, node)
	listen, err := net.Listen("tcp", port)
	utils.PanicOnError(err)
	log.Infof("[UpgradeableConsensus] Server start at %s", listen.Addr().String())
	go rpcServer.Serve(listen)

	apiServer.Start()

	return node
}

func (node *Node) Stop() {
	node.rpcServer.Stop()
	node.consensus.Stop()
	node.p2pAdaptor.Stop()
}

func (node *Node) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	node.log.Infoln("[node]\t receive request for header")
	if in.Type != pb.PacketType_CLIENTPACKET {
		pbMsg := new(pb.PoTMessage)
		_ = proto.Unmarshal(in.Msg, pbMsg)
		node.consensus.GetMsgByteEntrance() <- in.GetMsg()
		// protos := new(pb.HeaderRequest)
		// _ = proto.Unmarshal(pbMsg.MsgByte, protos)
		// node.log.Warn("[Node] packet type error ", hexutil.Encode(protos.Hashes))
		return &pb.Empty{}, nil
	}
	request := new(pb.Request)
	if err := proto.Unmarshal(in.Msg, request); err != nil {
		node.log.WithError(err).Warn("[Node] decode request error")
		return &pb.Empty{}, nil
	}
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
	switch c.GetConsensusType() {
	case "pot":
		if potEngine, ok := c.(*pot.PoTEngine); ok {
			// Register transaction API service
			apiServer.RegisterPotService(apis.NewPotWorkerAdapter(potEngine.Worker))
			log.Info("Registered PoT API service")

			// Register monitoring API service
			apiServer.RegisterMonitorService(apis.NewPotMonitorAdapter(potEngine, log))
			log.Info("Registered PoT Monitor API service")
		} else {
			log.Warn("Failed to cast Consensus to PoTEngine for API registration")
		}
	// case *pow.PoTEngine:
	// 	apiServer.RegisterPowService(apis.NewPowWorkerAdapter(cons.worker))
	// 	log.Info("Registered PoW API service")
	default:
		log.Info("No matching consensus API service to register")
	}
}
