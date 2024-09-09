package p2p

import (
	"context"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type BaseP2p struct {
	nodes     []*config.ReplicaInfo
	log       *logrus.Entry
	id        int64
	peerid    string
	output    chan<- []byte
	rpcServer *grpc.Server
	pb.UnimplementedP2PServer
}

func NewBaseP2p(log *logrus.Entry, id int64) (*BaseP2p, string, error) {
	cfg, err := config.NewConfig("config/configpot.yaml", id)
	utils.PanicOnError(err)
	info := cfg.GetNodeInfo(id)
	port := info.Address[strings.Index(info.Address, ":"):]
	listen, err := net.Listen("tcp", port)
	utils.PanicOnError(err)
	rpcServer := grpc.NewServer()
	utils.PanicOnError(err)
	bp := &BaseP2p{
		nodes:                  cfg.Nodes,
		log:                    log,
		id:                     id,
		output:                 nil,
		rpcServer:              rpcServer,
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
		peerid:                 cfg.Nodes[id].Address,
	}
	pb.RegisterP2PServer(rpcServer, bp)
	log.Infof("[UpgradeableConsensus] Server start at %s", listen.Addr().String())
	go rpcServer.Serve(listen)
	nid := cfg.Nodes[id].Address
	return bp, nid, nil
}

func (bp *BaseP2p) Subscribe(topic []byte) error {
	// do nothing
	return nil
}

func (bp *BaseP2p) UnSubscribe(topic []byte) error {
	// do nothing
	return nil
}

func (bp *BaseP2p) GetPeerID() string {
	// do nothing
	return bp.nodes[bp.id].Address
}

func (bp *BaseP2p) GetP2PType() string {
	// do nothing
	return "p2p"
}

// func (bp *BaseP2p) SetUnicastReceiver(receiver MsgReceiver) {
// 	bp.output = receiver.GetMsgByteEntrance()
// }

func (bp *BaseP2p) SetReceiver(ch chan<- []byte) {
	bp.output = ch
}

func (bp *BaseP2p) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	nodes := bp.nodes
	// bp.log.Debugf("broadcast packet to %d nodes", len(nodes)-1)
	for _, node := range nodes {
		if node.ID == bp.id {
			continue
		}
		err := bp.Unicast(node.Address, msgByte, consensusID, topic)
		if err != nil {
			bp.log.WithError(err).Warn("send msg failed")
			return err
		}
		// bp.log.Debugf("unicast to %s done", node.Address)
	}
	return nil
}

func (bp *BaseP2p) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	// conn, err := grpc.Dial(address, grpc.WithInsecure())
	// bp.log.Trace("[basep2p] unicast")
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

// BaseP2p implements P2PServer
func (bp *BaseP2p) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	bytePacket, err := proto.Marshal(in)
	// bp.log.Trace("[basep2p] received msg")
	if err != nil {
		bp.log.Warn("marshal packet failed")
		return nil, err
	}
	if bp.output != nil {
		bp.output <- bytePacket
	} else {
		bp.log.Warn("[BaseP2p] output nil")
	}
	return &pb.Empty{}, nil
}

func (bp *BaseP2p) Stop() {
	bp.rpcServer.Stop()
}

//func (bp *BaseP2p) Request(ctx context.Context, in *pb.HeaderRequest) (*pb.Header, error) {
//	id := in.Desid
//	address := bp.nodes[id].RpcAddress
//	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		bp.log.Warn("dial", address, "failed")
//		return nil, err
//	}
//	client := pb.NewP2PClient(conn)
//	header, err := client.Request(context.Background(), in)
//	if err != nil {
//		bp.log.Warn("[BaseP2P]\t sent request to ", address, "fail: ", err)
//		return nil, err
//	}
//	conn.Close()
//	return header, nil
//}
//
//func (bp *BaseP2p) PoTresRequest(ctx context.Context, request *pb.PoTRequest) (*pb.PoTResponse, error) {
//	id := request.Desid
//	address := bp.nodes[id].RpcAddress
//	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		bp.log.Warn("dial", address, "failed")
//		return nil, err
//	}
//	client := pb.NewP2PClient(conn)
//	potproof, err := client.PoTresRequest(context.Background(), request)
//	if err != nil {
//		bp.log.Warn("[BaseP2P]\t sent request to ", address, "fail: ", err)
//		return nil, err
//	}
//	conn.Close()
//	return potproof, nil
//}
