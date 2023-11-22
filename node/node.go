package node

import (
	"context"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
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
	pot        *pot.PoTEngine
	rpcServer  *grpc.Server
	log        *logrus.Entry
	p2pAdaptor p2p.P2PAdaptor
	pb.UnimplementedP2PServer
}

func NewNode(id int64) *Node {
	cfg, err := config.NewConfig("config/configpot.yaml", id)
	utils.PanicOnError(err)
	info := cfg.GetNodeInfo(id)
	log := logging.GetLogger().WithField("id", id)
	port := info.RpcAddress[strings.Index(info.RpcAddress, ":"):]
	listen, err := net.Listen("tcp", port)
	utils.PanicOnError(err)
	rpcServer := grpc.NewServer()
	utils.PanicOnError(err)

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
	e := executor.BuildExecutor(cfg.Executor, log)
	c := pot.NewEngine(id, 10001, cfg.Consensus, e, p, log)
	node := &Node{
		id:                     id,
		pot:                    c,
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
	node.pot.Stop()
	node.p2pAdaptor.Stop()
}

func (node *Node) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	//node.log.Infoln("[node]\t receive request for header")
	if in.Type != pb.PacketType_CLIENTPACKET {
		pbmsg := new(pb.PoTMessage)
		_ = proto.Unmarshal(in.Msg, pbmsg)
		node.pot.GetMsgByteEntrance() <- in.GetMsg()
		//protos := new(pb.HeaderRequest)
		//_ = proto.Unmarshal(pbmsg.MsgByte, protos)
		//node.log.Warn("[Node] packet type error ", hexutil.Encode(protos.Hashes))
		return &pb.Empty{}, nil
	}
	request := new(pb.Request)
	if err := proto.Unmarshal(in.Msg, request); err != nil {
		node.log.WithError(err).Warn("[Node] decode request error")
		return &pb.Empty{}, nil
	}
	node.pot.GetRequestEntrance() <- request
	return &pb.Empty{}, nil
}

//func (node *Node) Request(ctx context.Context, in *pb.HeaderRequest) (*pb.Header, error) {
//	node.log.Infoln("[node]\t receive request for header")
//	hash := in.Hashes
//	st := node.pot.GetStorage()
//	header, err := st.Get(hash)
//	if err != nil {
//		node.log.Error("get block err for ", err)
//		return nil, err
//	} else {
//		return header.ToProto(), nil
//	}
//
//}
//
//func (node *Node) PoTresRequest(ctx context.Context, request *pb.PoTRequest) (*pb.PoTResponse, error) {
//	node.log.Infoln("[node]\t receive request for header")
//	epoch := request.GetEpoch()
//	st := node.pot.GetStorage()
//	res, err := st.GetPoTbyEpoch(epoch)
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
//}
