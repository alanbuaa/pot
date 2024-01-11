package simpleWhirly

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

func (sw *SimpleWhirlyImpl) handlePoTSignal(potSignalBytes []byte) {
	potSignal := &PoTSignal{}
	err := json.Unmarshal(potSignalBytes, potSignal)
	if err != nil {
		sw.Log.WithField("error", err.Error()).Error("Unmarshal potSignal failed.")
		return
	}

	// Ignoring pot signals from old epochs
	if potSignal.Epoch <= sw.epoch {
		return
	}

	// Determine whether the node is a leader
	if potSignal.LeaderNetworkId == sw.PeerId {
		sw.NewLeader(potSignal)
	}
}

func (sw *SimpleWhirlyImpl) UpdateCommittee(committee []string, weight int) {
	sw.Log.Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] update committee tirgger!")

	sw.Weight = int64(weight)
	sw.inCommittee = true

	if len(committee) != len(sw.Config.Nodes) {
		sw.Log.Warn("the committee size is error")
		return
	}

	sw.Committee = committee
}

func (sw *SimpleWhirlyImpl) SleepNode() {
	sw.Log.Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] sleep node tirgger!")

	sw.Weight = 0
	sw.inCommittee = false
}

func (sw *SimpleWhirlyImpl) VerifyPoTProof(epoch int64, leader int64, proof []byte) bool {
	return true
}

func (sw *SimpleWhirlyImpl) NewLeader(potSignal *PoTSignal) {
	sw.Log.Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] new Epoch tirgger!")

	sw.echoLock.Lock()
	sw.curEcho = make([]*pb.SimpleWhirlyProof, 0)
	sw.echoLock.Unlock()
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

func (sw *SimpleWhirlyImpl) OnReceiveNewLeaderNotify(newLeaderMsg *pb.NewLeaderNotify) {
	epoch := int64(newLeaderMsg.Epoch)
	leader := int64(newLeaderMsg.Leader)
	peerId := newLeaderMsg.PeerId
	committee := newLeaderMsg.Committee

	sw.Log.WithFields(logrus.Fields{
		"newEpoch":  epoch,
		"newLeader": leader,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceive Notify.")

	if epoch < sw.epoch {
		return
	}

	if !sw.VerifyPoTProof(epoch, leader, newLeaderMsg.Proof) {
		return
	}

	// Enter the current epoch and record the leader
	sw.SetLeader(epoch, peerId)
	sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] advance Epoch success!")

	sw.voteLock.Lock()
	sw.CleanVote()
	sw.voteLock.Unlock()

	// Calculate the weight of the node
	weight := 0
	for _, c := range committee {
		if c == sw.PeerId {
			weight += 1
		}
	}

	// If the weight is not 0, it indicates that the node is in the committee
	if weight > 0 {
		sw.UpdateCommittee(committee, weight)
	} else {
		sw.SleepNode()
	}

	// Echo leader
	echoMsg := sw.NewLeaderEchoMsg(leader, nil, sw.lockProof, sw.epoch, sw.vHeight)

	if sw.leader[sw.epoch] == sw.PeerId {
		// echo self
		sw.OnReceiveNewLeaderEcho(echoMsg)
	} else {
		// send vote to the leader
		if sw.GetP2pAdaptorType() == "p2p" {
			_ = sw.Unicast(sw.leader[sw.epoch], echoMsg)
		} else {
			_ = sw.Unicast(peerId, echoMsg)
		}
	}
}

func (sw *SimpleWhirlyImpl) OnReceiveNewLeaderEcho(msg *pb.WhirlyMsg) {
	echoMsg := msg.GetNewLeaderEcho()

	if int64(echoMsg.Epoch) < sw.epoch {
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"senderId":     echoMsg.SenderId,
		"epoch":        echoMsg.Epoch,
		"leader":       echoMsg.Leader,
		"len(curEcho)": len(sw.curEcho),
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveEcho.")

	if !sw.verfiySwProof(echoMsg.SwProof) {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] echo proof is wrong.")
		return
	}

	if echoMsg.VHeight > sw.maxVHeight {
		sw.maxVHeight = echoMsg.VHeight
	}

	sw.echoLock.Lock()
	sw.curEcho = append(sw.curEcho, echoMsg.SwProof)
	sw.echoLock.Unlock()
	sw.lock.Lock()
	sw.UpdateLockProof(echoMsg.SwProof)
	sw.lock.Unlock()

	if len(sw.curEcho) >= 2*sw.Config.F+1 {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] begin propose.")
		sw.AdvanceView(sw.maxVHeight)
		go sw.OnPropose()
	}
}

type PoTSignal struct {
	Epoch           int64
	Proof           []byte
	ID              int64
	LeaderNetworkId string
	Committee       []string
	CryptoElements  []byte
}

func (sw *SimpleWhirlyImpl) testNewLeader() {
	for i := 1; i < 100; i++ {
		time.Sleep(time.Second * 8)
		potSignal := &PoTSignal{}

		potSignal.Epoch = sw.epoch + 1
		potSignal.Proof = nil
		potSignal.LeaderNetworkId = sw.Config.Nodes[i%4].Address
		potSignal.Committee = make([]string, len(sw.Config.Nodes))
		for i := 0; i < len(sw.Config.Nodes); i++ {
			potSignal.Committee[i] = sw.Config.Nodes[i].Address
		}

		potSignalBytes, _ := json.Marshal(potSignal)
		sw.PoTByteEntrance <- potSignalBytes
	}
}

func (sw *SimpleWhirlyImpl) testNewLeader2() {
	for i := 1; i < 100; i++ {
		time.Sleep(time.Second * 5)
		potSignal := &PoTSignal{}

		potSignal.Epoch = sw.epoch + 1
		potSignal.Proof = nil
		potSignal.LeaderNetworkId = sw.Config.Nodes[i%4].Address
		potSignal.Committee = make([]string, len(sw.Config.Nodes))
		for i := 0; i < len(sw.Config.Nodes); i++ {
			potSignal.Committee[i] = sw.Config.Nodes[i].Address
		}
		potSignal.Committee[(i-1)%4] = sw.Config.Nodes[(i+1)%4].Address

		potSignalBytes, _ := json.Marshal(potSignal)
		sw.PoTByteEntrance <- potSignalBytes
	}
}
