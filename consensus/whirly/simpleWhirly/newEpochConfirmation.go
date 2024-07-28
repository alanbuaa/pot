package simpleWhirly

import (
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

type NewEpochMechanism struct {
	curEcho     map[string]*pb.SimpleWhirlyProof
	curEchoLock sync.Mutex
	echoFlag    bool
	activeFlag  bool

	epoch int64
	proof []byte
}

// func (sw *SimpleWhirlyImpl) NewEpochConfirmation(potSignal *PoTSignal, sharding *Sharding) {
func (sw *SimpleWhirlyImpl) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	sw.Log.WithFields(logrus.Fields{
		"address": sw.PublicAddress,
	}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] new Epoch Confirmation!")

	sw.newEpoch.curEchoLock.Lock()
	sw.newEpoch.curEcho = make(map[string]*pb.SimpleWhirlyProof)
	sw.newEpoch.activeFlag = false
	sw.newEpoch.epoch = epoch
	sw.newEpoch.proof = proof
	sw.Committee = committee
	sw.newEpoch.curEchoLock.Unlock()

	time.Sleep(1 * time.Second)

	newLeaderMsg := sw.NewLeaderNotifyMsg(epoch, proof, committee)
	if sw.GetP2pAdaptorType() == "p2p" {
		sw.handleMsg(newLeaderMsg)
	}
	// broadcast
	// 循环发送，直到收到了足够的echo消息
	for i := 0; i < 10; i++ {
		if sw.newEpoch.activeFlag {
			break
		}
		err := sw.Broadcast(newLeaderMsg)
		if err != nil {
			sw.Log.WithField("error", err.Error()).Warn("Broadcast newLeaderMsg failed.")
		}
		time.Sleep(3 * time.Second)
	}

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

	// Calculate the weight of the node
	weight := 0
	_, _, address := DecodeAddress(sw.PublicAddress)
	for _, c := range committee {
		if c == address {
			weight += 1
		}
	}

	// 如果节点在新委员会中，则响应leader
	if weight <= 0 || address == DaemonNodePublicAddress {
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"newEpoch":  epoch,
		"newLeader": publicAddress,
		"myAddress": sw.PublicAddress,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceive Notify.")

	// block, err := sw.BlockStorage.Get(sw.lockProof.BlockHash)
	// if err != nil {
	// 	sw.Log.Trace("no block for ehco message")
	// }

	// Echo leader
	// 请注意，此时响应了新 leader，但是节点的 epoch 尚未更新，需要等到 leader 向旧委员会获取最新的区块时才更新，表示正式进行新的 epoch
	echoMsg := sw.NewLeaderEchoMsg(leader, nil, sw.lockProof, epoch, sw.vHeight)

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
	senderAdress := echoMsg.PublicAddress

	if int64(echoMsg.Epoch) < sw.epoch {
		sw.Log.WithFields(logrus.Fields{
			"echoMsg.epoch": echoMsg.Epoch,
			"myEpoch":       sw.epoch,
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] the epoch of echo message is too old.")
		return
	}

	sw.newEpoch.curEchoLock.Lock()
	_, ok := sw.newEpoch.curEcho[senderAdress]
	if ok {
		sw.newEpoch.curEchoLock.Unlock()
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"sender":       echoMsg.PublicAddress,
		"epoch":        echoMsg.Epoch,
		"leader":       echoMsg.Leader,
		"len(curEcho)": len(sw.newEpoch.curEcho),
		"VHeight":      echoMsg.VHeight,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveEcho.")

	// err := sw.BlockStorage.Put(echoMsg.Block)
	// if err != nil {
	// 	sw.Log.WithError(err).Info("Store the new block from echo message failed.")
	// 	sw.newEpoch.curEchoLock.Unlock()
	// 	return
	// }

	if !sw.verfiySwProof(echoMsg.SwProof) {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] echo proof is wrong.")
		sw.newEpoch.curEchoLock.Unlock()
		return
	}

	sw.newEpoch.curEcho[senderAdress] = echoMsg.SwProof

	// sw.lock.Lock()
	// sw.UpdateLockProof(echoMsg.SwProof)
	// sw.lock.Unlock()

	if len(sw.newEpoch.curEcho) >= 2*sw.Config.F+1 {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] begin RequestLatestBlock.")
		sw.newEpoch.curEcho = make(map[string]*pb.SimpleWhirlyProof)

		// 开始向旧委员会获取最新的区块
		go sw.RequestLatestBlock(sw.newEpoch.epoch, sw.newEpoch.proof, sw.Committee)
		sw.newEpoch.curEchoLock.Unlock()
		return
	}
	sw.newEpoch.curEchoLock.Unlock()
}
