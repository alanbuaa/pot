package consensus

import (
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

// UpgradeableConsensus implements P2PAdaptor
func (uc *UpgradeableConsensus) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: consensusID,
		Epoch:       uc.epoch,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	return uc.p2pAdaptor.Broadcast(bytePacket, -1, topic)
}

func (uc *UpgradeableConsensus) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	packet := &pb.Packet{
		Msg:         msgByte,
		ConsensusID: consensusID,
		Epoch:       uc.epoch,
		Type:        pb.PacketType_P2PPACKET,
	}
	bytePacket, err := proto.Marshal(packet)
	utils.PanicOnError(err)
	return uc.p2pAdaptor.Unicast(address, bytePacket, -1, topic)
}

func (uc *UpgradeableConsensus) SetReceiver(ch chan<- []byte) {
	// do nothing
}

func (uc *UpgradeableConsensus) Subscribe(topic []byte) error {
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
