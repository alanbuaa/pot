package p2p

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"

	pad "p2padaptor"
)

// type MsgReceiver interface {
// 	GetMsgByteEntrance() chan<- []byte
// }

// NodeDiscoveryCallback 节点发现回调接口
// 使用 p2padaptor 的接口定义以保持兼容性
type NodeDiscoveryCallback = pad.NodeDiscoveryCallback

type P2PAdaptor interface {
	// NewXXX(log *logrus.Entry, id int64) (P2PAdaptor, string, error)
	Broadcast(msgByte []byte, consensusID int64, topic []byte) error
	Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error
	// SetUnicastReceiver(receiver MsgReceiver)
	SetReceiver(ch chan<- []byte)
	Subscribe(topic []byte) error
	UnSubscribe(topic []byte) error
	GetPeerID() string
	GetP2PType() string
	// should return synchronously
	Stop()

	// 节点发现相关方法
	// DiscoverPeers 启动节点发现进程
	DiscoverPeers() error
	// RegisterNodeDiscoveryCallback 注册节点发现回调
	RegisterNodeDiscoveryCallback(callback NodeDiscoveryCallback)
	// GetDiscoveredNodes 获取已发现的节点列表
	GetDiscoveredNodes() map[int64]string
	// GetNodeCount 获取已发现的节点数量
	GetNodeCount() int
	// WaitForNodes 等待指定数量的节点上线
	WaitForNodes(count int, timeout int) error
}

func BuildP2P(cfg *config.Config, log *logrus.Entry, id int64) (P2PAdaptor, string, error) {
	switch cfg.Network.P2PType {
	case "p2p-base":
		return NewBaseP2P(cfg, log, id)
	case "p2p-adaptor":
		return NewP2PAdaptor(cfg, log, id)
	}
	log.WithField("type", cfg.Network.P2PType).Warn("p2p type error")
	return nil, "", nil
}

func NewP2PAdaptor(cfg *config.Config, log *logrus.Entry, id int64) (*pad.NetworkAdaptor, string, error) {
	info, err := cfg.GetNodeFromSet(id)
	if err != nil {
		return nil, "", err
	}
	port := strings.Split(info.P2PAddress, ":")[1]

	// New adaptor
	dataDir := info.DataDir
	if dataDir == "" {
		dataDir = cfg.Node.DataDir
	}
	nada, err := pad.NewNetworkAdaptor(port, cfg.Network.Topic, dataDir)
	if err != nil {
		log.WithField("error", err).Error("NewNetworkAdaptor error in port: ", port)
		return nil, "", err
	}

	peerid := nada.GetPeerID()
	fmt.Printf("id %d:PeerID:%s\n", id, peerid)

	// Start unicast
	err = nada.StartUnicast()
	if err != nil {
		log.WithField("error", err).Error("Start unicast service error")
		return nil, "", err
	}

	log.Debugf("id %d: Unicast service started\n", id)

	return nada, peerid, nil
}
