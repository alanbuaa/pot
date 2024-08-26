package simpleWhirly

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

const DaemonNodePublicAddress string = "daemonNode"
const BroadcastToAll string = "ALL"

func EncodeAddress(shardingName string, consensusID int64, address string) string {
	consensusIDStr := strconv.FormatInt(consensusID, 10)
	return shardingName + "-" + consensusIDStr + "-" + address
}

// Input: address that is encoded
// Output: shardingName, consensusID, rawAddress
func DecodeAddress(address string) (string, int64, string) {
	res := strings.Split(address, "-")
	if len(res) != 3 {
		println("DecodeAddress Error: Illegal parameter")
		return "", 0, ""
	}
	shardingName := res[0]
	rawAddress := res[2]

	consensusID, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		println("DecodeAddress Error: Illegal ConsensusID")
		return "", 0, ""
	}
	return shardingName, consensusID, rawAddress
}

type LatestBlockRequestMechanism struct {
	maxVHeight   uint64
	curEcho      map[string]*pb.SimpleWhirlyProof
	curEchoLock  sync.Mutex
	requestEpoch int64 // 确保一个Epoch只 request 一次
	requestLock  sync.Mutex
}

// func (sw *SimpleWhirlyImpl) RequestLatestBlock(potSignal *PoTSignal, sharding *Sharding) {
func (sw *SimpleWhirlyImpl) RequestLatestBlock(epoch int64, proof []byte, committee []string) {

	sw.latestBlockReq.requestLock.Lock()
	if epoch <= sw.latestBlockReq.requestEpoch {
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"address": sw.PublicAddress,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] request latest block!")

	sw.latestBlockReq.curEchoLock.Lock()
	sw.latestBlockReq.curEcho = make(map[string]*pb.SimpleWhirlyProof)
	sw.latestBlockReq.curEchoLock.Unlock()
	sw.latestBlockReq.maxVHeight = sw.vHeight

	sw.latestBlockReq.requestEpoch = epoch
	sw.latestBlockReq.requestLock.Unlock()

	time.Sleep(1 * time.Second)

	newLatestBlockRequest := sw.NewLatestBlockRequest(epoch, proof, committee)
	if sw.GetP2pAdaptorType() == "p2p" {
		sw.handleMsg(newLatestBlockRequest)
	}
	// broadcast
	err := sw.Broadcast(newLatestBlockRequest)
	if err != nil {
		sw.Log.WithField("error", err.Error()).Warn("Broadcast newLeaderMsg failed.")
	}
}

func (sw *SimpleWhirlyImpl) UpdateCommittee(committee []string, weight int) {

	_, _, address := DecodeAddress(sw.PublicAddress)
	if address != DaemonNodePublicAddress {
		sw.Log.WithFields(logrus.Fields{
			"address": sw.PublicAddress,
		}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] update committee tirgger!")
		sw.Weight = int64(weight)
		sw.inCommittee = true
	}

	sw.Committee = committee
}

func (sw *SimpleWhirlyImpl) SleepNode() {
	sw.Log.WithFields(logrus.Fields{
		"address": sw.PublicAddress,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] sleep node tirgger!")

	sw.Weight = 0
	sw.inCommittee = false

	// Report to the controller that this node should be stopped
	sw.stopEntrance <- sw.PublicAddress

}

func (sw *SimpleWhirlyImpl) VerifyPoTProof(epoch int64, leader int64, proof []byte) bool {
	return true
}

func (sw *SimpleWhirlyImpl) OnReceiveLatestBlockRequest(newLeaderMsg *pb.LatestBlockRequest) {
	epoch := int64(newLeaderMsg.Epoch)
	leader := int64(newLeaderMsg.Leader)
	publicAddress := newLeaderMsg.PublicAddress
	committee := newLeaderMsg.Committee
	newConsensusId := newLeaderMsg.ConsensusId

	if epoch < sw.epoch {
		return
	}

	if !sw.VerifyPoTProof(epoch, leader, newLeaderMsg.Proof) {
		return
	}

	// Enter the current epoch and record the leader
	sw.SetLeader(epoch, publicAddress)
	sw.SetEpoch(epoch)
	sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] advance Epoch success!")

	sw.voteLock.Lock()
	sw.CleanVote()
	sw.voteLock.Unlock()

	// Calculate the weight of the node
	weight := 0
	_, consensusID, address := DecodeAddress(sw.PublicAddress)
	for _, c := range committee {
		if c == address {
			weight += 1
		}
	}

	// If the weight is not 0, it indicates that the node is in the committee
	// The daemon node should never be stopped
	// 当前共识实例的 daemon node 不需要停止，但是其他节点，包括前面共识实例的 daemon node 都需要停止
	if weight > 0 || (address == DaemonNodePublicAddress && consensusID == int64(newConsensusId)) {
		sw.UpdateCommittee(committee, weight)
		// The daemon node should never echo
		if address == DaemonNodePublicAddress {
			return
		}
	} else {
		sw.SleepNode()
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"newEpoch":  epoch,
		"newLeader": publicAddress,
		"myAddress": sw.PublicAddress,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceive LatestBlockRequest.")

	block, err := sw.BlockStorage.Get(sw.lockProof.BlockHash)
	if err != nil {
		sw.Log.Trace("no block for ehco message")
	}

	// Echo leader
	echoMsg := sw.NewLatestBlockEchoMsg(leader, block, sw.lockProof, sw.epoch, sw.vHeight)

	if sw.GetLeader(sw.epoch) == sw.PublicAddress {
		// echo self
		sw.OnReceiveLatestBlockEcho(echoMsg)
	} else {
		// send vote to the leader
		if sw.GetP2pAdaptorType() == "p2p" {
			_ = sw.Unicast(sw.GetLeader(sw.epoch), echoMsg)
		} else {
			_ = sw.Unicast(publicAddress, echoMsg)
		}
	}

}

func (sw *SimpleWhirlyImpl) OnReceiveLatestBlockEcho(msg *pb.WhirlyMsg) {
	echoMsg := msg.GetLatestBlockEcho()
	senderAdress := echoMsg.PublicAddress

	if int64(echoMsg.Epoch) < sw.epoch {
		sw.Log.WithFields(logrus.Fields{
			"echoMsg.epoch": echoMsg.Epoch,
			"myEpoch":       sw.epoch,
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] the epoch of echo message is too old.")
		return
	}

	sw.latestBlockReq.curEchoLock.Lock()
	sw.Log.WithFields(logrus.Fields{
		"sender":       echoMsg.PublicAddress,
		"epoch":        echoMsg.Epoch,
		"leader":       echoMsg.Leader,
		"len(curEcho)": len(sw.latestBlockReq.curEcho),
		"VHeight":      echoMsg.VHeight,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceive LatestBlockEcho.")

	err := sw.BlockStorage.Put(echoMsg.Block)
	if err != nil {
		sw.Log.WithError(err).Info("Store the new block from echo message failed.")
		sw.latestBlockReq.curEchoLock.Unlock()
		return
	}

	if !sw.verfiySwProof(echoMsg.SwProof) {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] echo proof is wrong.")
		sw.latestBlockReq.curEchoLock.Unlock()
		return
	}

	if echoMsg.VHeight > sw.latestBlockReq.maxVHeight {
		sw.latestBlockReq.maxVHeight = echoMsg.VHeight
	}

	sw.latestBlockReq.curEcho[senderAdress] = echoMsg.SwProof

	sw.lock.Lock()
	sw.UpdateLockProof(echoMsg.SwProof)
	sw.lock.Unlock()

	if len(sw.latestBlockReq.curEcho) >= 2*sw.Config.F+1 {
		sw.AdvanceView(sw.latestBlockReq.maxVHeight)
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] begin propose.")
		sw.latestBlockReq.curEcho = make(map[string]*pb.SimpleWhirlyProof)
		sw.latestBlockReq.curEchoLock.Unlock()
		go sw.OnPropose()
		return
	}
	sw.latestBlockReq.curEchoLock.Unlock()
}

// func (sw *SimpleWhirlyImpl) testNewLeader() {
// 	for i := 1; i < 100; i++ {
// 		time.Sleep(time.Second * 8)
// 		potSignal := &PoTSignal{}

// 		potSignal.Epoch = sw.epoch + 1
// 		potSignal.Proof = nil
// 		potSignal.LeaderPublicAddress = sw.Config.Nodes[i%4].Address
// 		potSignal.Committee = make([]string, len(sw.Config.Nodes))
// 		for i := 0; i < len(sw.Config.Nodes); i++ {
// 			potSignal.Committee[i] = sw.Config.Nodes[i].Address
// 		}

// 		potSignalBytes, _ := json.Marshal(potSignal)
// 		sw.PoTByteEntrance <- potSignalBytes
// 	}
// }

// func (sw *SimpleWhirlyImpl) testNewLeader2() {
// 	for i := 1; i < 100; i++ {
// 		time.Sleep(time.Second * 5)
// 		potSignal := &PoTSignal{}

// 		potSignal.Epoch = sw.epoch + 1
// 		potSignal.Proof = nil
// 		potSignal.LeaderPublicAddress = sw.Config.Nodes[i%4].Address
// 		potSignal.Committee = make([]string, len(sw.Config.Nodes))
// 		for i := 0; i < len(sw.Config.Nodes); i++ {
// 			potSignal.Committee[i] = sw.Config.Nodes[i].Address
// 		}
// 		potSignal.Committee[(i-1)%4] = sw.Config.Nodes[(i+1)%4].Address

// 		potSignalBytes, _ := json.Marshal(potSignal)
// 		sw.PoTByteEntrance <- potSignalBytes
// 	}
// }
