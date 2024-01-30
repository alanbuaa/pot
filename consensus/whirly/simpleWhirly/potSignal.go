package simpleWhirly

import (
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

// func (sw *SimpleWhirlyImpl) handlePoTSignal(potSignalBytes []byte) {
// 	potSignal := &PoTSignal{}
// 	err := json.Unmarshal(potSignalBytes, potSignal)
// 	if err != nil {
// 		sw.Log.WithField("error", err.Error()).Error("Unmarshal potSignal failed.")
// 		return
// 	}

// 	// Ignoring pot signals from old epochs
// 	if potSignal.Epoch <= sw.epoch {
// 		return
// 	}

// 	// Determine whether the node is a leader
// 	if potSignal.LeaderNetworkId == sw.PublicAddress {
// 		sw.NewLeader(potSignal)
// 	}
// }

func (sw *SimpleWhirlyImpl) NewLeader(potSignal *PoTSignal) {
	sw.Log.WithFields(logrus.Fields{
		"address": sw.PublicAddress,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] new Epoch tirgger!")

	sw.curEchoLock.Lock()
	sw.curEcho = make([]*pb.SimpleWhirlyProof, 0)
	sw.curEchoLock.Unlock()
	sw.maxVHeight = sw.vHeight

	newLeaderMsg := sw.NewLeaderNotifyMsg(potSignal.Epoch, potSignal.Proof, potSignal.Committee)
	if sw.GetP2pAdaptorType() == "p2p" {
		sw.handleMsg(newLeaderMsg)
	}
	// broadcast
	err := sw.Broadcast(newLeaderMsg)
	if err != nil {
		sw.Log.WithField("error", err.Error()).Warn("Broadcast newLeaderMsg failed.")
	}
}

func (sw *SimpleWhirlyImpl) UpdateCommittee(committee []string, weight int) {

	if sw.PublicAddress != DaemonNodePublicAddress {
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

func (sw *SimpleWhirlyImpl) OnReceiveNewLeaderNotify(newLeaderMsg *pb.NewLeaderNotify) {
	epoch := int64(newLeaderMsg.Epoch)
	leader := int64(newLeaderMsg.Leader)
	publicAddress := newLeaderMsg.PublicAddress
	committee := newLeaderMsg.Committee

	if epoch < sw.epoch {
		return
	}

	if !sw.VerifyPoTProof(epoch, leader, newLeaderMsg.Proof) {
		return
	}

	// Enter the current epoch and record the leader
	sw.SetLeader(epoch, publicAddress)
	sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] advance Epoch success!")

	sw.voteLock.Lock()
	sw.CleanVote()
	sw.voteLock.Unlock()

	// Calculate the weight of the node
	weight := 0
	for _, c := range committee {
		if c == sw.PublicAddress {
			weight += 1
		}
	}

	// If the weight is not 0, it indicates that the node is in the committee
	// The daemon node should never be stopped
	if weight > 0 || sw.PublicAddress == DaemonNodePublicAddress {
		sw.UpdateCommittee(committee, weight)
		// The daemon node should never echo
		if sw.PublicAddress == DaemonNodePublicAddress {
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
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceive Notify.")

	block, err := sw.BlockStorage.Get(sw.lockProof.BlockHash)
	if err != nil {
		sw.Log.Trace("no block for ehco message")
	}

	// Echo leader
	echoMsg := sw.NewLeaderEchoMsg(leader, block, sw.lockProof, sw.epoch, sw.vHeight)

	if sw.GetLeader(sw.epoch) == sw.PublicAddress {
		// echo self
		sw.OnReceiveNewLeaderEcho(echoMsg)
	} else {
		// send vote to the leader
		if sw.GetP2pAdaptorType() == "p2p" {
			_ = sw.Unicast(sw.GetLeader(sw.epoch), echoMsg)
		} else {
			_ = sw.Unicast(publicAddress, echoMsg)
		}
	}

}

func (sw *SimpleWhirlyImpl) OnReceiveNewLeaderEcho(msg *pb.WhirlyMsg) {
	echoMsg := msg.GetNewLeaderEcho()

	if int64(echoMsg.Epoch) < sw.epoch {
		sw.Log.WithFields(logrus.Fields{
			"echoMsg.epoch": echoMsg.Epoch,
			"myEpoch":       sw.epoch,
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] the epoch of echo message is too old.")
		return
	}

	sw.curEchoLock.Lock()
	sw.Log.WithFields(logrus.Fields{
		"sender":       echoMsg.PublicAddress,
		"epoch":        echoMsg.Epoch,
		"leader":       echoMsg.Leader,
		"len(curEcho)": len(sw.curEcho),
		"VHeight":      echoMsg.VHeight,
		// "myAddress":    sw.PublicAddress,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveEcho.")

	err := sw.BlockStorage.Put(echoMsg.Block)
	if err != nil {
		sw.Log.WithError(err).Info("Store the new block from echo message failed.")
		sw.curEchoLock.Unlock()
		return
	}

	if !sw.verfiySwProof(echoMsg.SwProof) {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] echo proof is wrong.")
		sw.curEchoLock.Unlock()
		return
	}

	if echoMsg.VHeight > sw.maxVHeight {
		sw.maxVHeight = echoMsg.VHeight
	}

	sw.curEcho = append(sw.curEcho, echoMsg.SwProof)

	sw.lock.Lock()
	sw.UpdateLockProof(echoMsg.SwProof)
	sw.lock.Unlock()

	if len(sw.curEcho) >= 2*sw.Config.F+1 {
		sw.AdvanceView(sw.maxVHeight)
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] begin propose.")
		sw.curEcho = make([]*pb.SimpleWhirlyProof, 0)
		sw.curEchoLock.Unlock()
		go sw.OnPropose()
		return
	}
	sw.curEchoLock.Unlock()
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
