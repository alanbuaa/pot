package simpleWhirly

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

func (sw *SimpleWhirlyImpl) sendPingMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(2 * time.Second)
			pingMsg := sw.PingMsg()
			if sw.GetP2pAdaptorType() == "p2p" {
				msgByte, err := proto.Marshal(pingMsg)
				utils.PanicOnError(err)
				sw.MsgByteEntrance <- msgByte
			}
			// broadcast
			sw.Log.Info("Broadcast pingMsg.")
			err := sw.Broadcast(pingMsg)
			if err != nil {
				sw.Log.WithField("error", err.Error()).Warn("Broadcast pingMsg failed.")
			}
		}
	}
}

func (sw *SimpleWhirlyImpl) handlePingMsg(msg *pb.WhirlyPing) {
	if sw.PublicAddress != sw.leader[sw.epoch] {
		return
	}
	sw.Log.Info("Receive ping msg.")
	id := msg.Id
	publicAddress := msg.PublicAddress
	temp := 0
	for i := 0; i < len(sw.readyNodes); i++ {
		if sw.readyNodes[i] == publicAddress {
			temp = 1
			break
		}
	}

	if temp == 0 && len(sw.readyNodes) < 2*sw.Config.F+1 {
		sw.Log.WithFields(logrus.Fields{
			"newNode":       id,
			"publicAddress": publicAddress,
		}).Info("find new node is alive.")
		sw.readyNodes = append(sw.readyNodes, publicAddress)
	}

	if len(sw.readyNodes) == 2*sw.Config.F+1 {
		if sw.PublicAddress == sw.leader[sw.epoch] && sw.proposeView == 0 {
			go sw.OnPropose()
		}
	}
}

func (sw *SimpleWhirlyImpl) stopSendPing() {
	sw.sendPingCancel()
}
