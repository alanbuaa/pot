package upgradeable_consensus

import (
	"context"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/logging"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	id         int64
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
	listen, err := net.Listen("tcp", port)
	utils.PanicOnError(err)
	rpcServer := grpc.NewServer()
	utils.PanicOnError(err)

	// p, nid, err := p2p.NewBaseP2p(log, id)
	p, nid, err := p2p.BuildP2P(cfg.P2P, log, id)
	if p == nil {
		panic("build p2p failed")
	}
	log.WithFields(logrus.Fields{
		"nid":  nid,
		"type": p.GetP2PType(),
	}).Info("get network id")
	utils.PanicOnError(err)
	e := executor.BuildExecutor(cfg.Executor, log)
	c := consensus.BuildConsensus(id, -1, cfg.Consensus, e, p, log)
	if c == nil {
		panic("Initialize consensus failed")
	}
	node := &Node{
		id:                     id,
		consensus:              c,
		rpcServer:              rpcServer,
		p2pAdaptor:             p,
		log:                    log,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
	}
	pb.RegisterP2PServer(rpcServer, node)
	log.Infof("[UpgradeableConsensus] Server start at %s", listen.Addr().String())
	go rpcServer.Serve(listen)
	return node
}

func (node *Node) Stop() {
	node.rpcServer.Stop()
	node.consensus.Stop()
	node.p2pAdaptor.Stop()
}

// Node implements P2PServer
func (node *Node) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	if in.Type != pb.PacketType_CLIENTPACKET {
		node.log.Warn("[Node] packet type error")
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
