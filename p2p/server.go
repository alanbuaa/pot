package p2p

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/utils"

	pad "p2padaptor"
)

// type MsgReceiver interface {
// 	GetMsgByteEntrance() chan<- []byte
// }

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
}

func BuildP2P(cfg *config.P2PConfig, log *logrus.Entry, id int64) (P2PAdaptor, string, error) {
	if cfg.Type == "p2p" {
		return NewBaseP2p(log, id)
	} else if cfg.Type == "p2p-adaptor" {
		return NewP2pAdaptor(log, id)
	}
	log.WithField("type", cfg.Type).Warn("p2p type error")
	return nil, "", nil
}

func NewP2pAdaptor(log *logrus.Entry, id int64) (*pad.NetworkAdaptor, string, error) {
	cfg, err := config.NewConfig("config/configpot.yaml", id)
	utils.PanicOnError(err)
	info := cfg.GetNodeInfo(id)
	port := info.Address[strings.Index(info.Address, ":")+1:]

	// New adaptor
	// port = "1" + port
	nada, err := pad.NewNetworkAdaptor(port)
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
		// fmt.Println("Start unicast service error: ", err.Error())
		return nil, "", err
	}

	fmt.Printf("id %d: Unicast service started\n", id)
	return nada, peerid, nil
}
