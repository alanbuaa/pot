package consensus

import (
	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"google.golang.org/protobuf/proto"
)

// UpgradeableConsensus implements P2PAdaptor
func (uc *UpgradeableConsensus) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	uc.log.WithFields(logrus.Fields{
		"cid":      consensusID,
		"epoch":    uc.epoch,
		"msg_size": len(msgByte),
		"topic":    string(topic),
	}).Trace("Broadcasting message")
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: consensusID,
		Epoch:       uc.epoch,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	err = uc.p2pAdaptor.Broadcast(bytePacket, -1, topic)
	if err != nil {
		uc.log.WithError(err).Warn("Broadcast failed")
	}
	return err
}

func (uc *UpgradeableConsensus) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	uc.log.WithFields(logrus.Fields{
		"address":  address,
		"cid":      consensusID,
		"epoch":    uc.epoch,
		"msg_size": len(msgByte),
		"topic":    string(topic),
	}).Trace("Unicasting message")
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: consensusID,
		Epoch:       uc.epoch,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	err = uc.p2pAdaptor.Unicast(address, bytePacket, -1, topic)
	if err != nil {
		uc.log.WithError(err).WithField("address", address).Warn("Unicast failed")
	}
	return err
}

func (uc *UpgradeableConsensus) SetReceiver(ch chan<- []byte) {
	// do nothing
}

func (uc *UpgradeableConsensus) Subscribe(topic []byte) error {
	uc.log.WithField("topic", string(topic)).Debug("Subscribing to topic")
	uc.p2pAdaptor.Subscribe(topic)
	return nil
}

func (uc *UpgradeableConsensus) UnSubscribe(topic []byte) error {
	// do nothing
	return nil
}

func (uc *UpgradeableConsensus) GetPeerID() string {
	// do nothing
	return ""
}

func (uc *UpgradeableConsensus) GetP2PType() string {
	// do nothing
	return ""
}
