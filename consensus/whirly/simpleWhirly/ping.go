package simpleWhirly

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
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
			sw.Log.Trace("Broadcasting ping message to discover ready nodes")
			err := sw.Broadcast(pingMsg)
			if err != nil {
				sw.Log.WithError(err).Warn("Failed to broadcast ping message")
			}
		}
	}
}

func (sw *SimpleWhirlyImpl) handlePingMsg(msg *pb.WhirlyPing) {
	if sw.PublicAddress != sw.leader[sw.epoch] {
		return
	}
	sw.Log.Trace("Received ping message from node")
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
			"node_id":        id,
			"public_address": publicAddress,
			"ready_count":    len(sw.readyNodes) + 1,
			"required":       2*sw.Config.F + 1,
		}).Debug("Discovered new ready node")
		sw.readyNodes = append(sw.readyNodes, publicAddress)
	}

	if len(sw.readyNodes) == 2*sw.Config.F+1 {
		if sw.PublicAddress == sw.leader[sw.epoch] && sw.proposeView == 0 {
			sw.Log.WithField("ready_nodes", len(sw.readyNodes)).Info("Quorum reached, starting proposal process")
			go sw.OnPropose()
		}
	}
}

func (sw *SimpleWhirlyImpl) stopSendPing() {
	sw.Log.Debug("Stopping ping message broadcast")
	sw.sendPingCancel()
}
