package model

import "github.com/zzz136454872/upgradeable-consensus/pb"

type Consensus interface {
	// Consensus should has a initialize function like this
	// NewConsensus(nid int64, cid int64, cfg *config.ConsensusConfig, exec executor.Executor, p2pAdaptor p2p.P2PAdaptor, log *logrus.Entry) Consensus

	// Consensus Implements MsgReceiver
	GetRequestEntrance() chan<- *pb.Request

	GetMsgByteEntrance() chan<- []byte
	// The Stop function should return synchronously
	Stop()
	GetConsensusID() int64
	VerifyBlock(block []byte, proof []byte) bool

	UpdateExternalStatus(status ExternalStatus)
	NewEpochConfirmation(epoch int64, proof []byte, committee []string)
	RequestLatestBlock(epoch int64, proof []byte, committee []string)

	// the following functions are for synchronous consensus upgrade only
	GetWeight(nid int64) float64
	GetMaxAdversaryWeight() float64
}

type ExternalStatus struct {
	Command string
	Epoch   int64
	Leader  string
}
