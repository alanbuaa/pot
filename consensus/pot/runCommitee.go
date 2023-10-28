package pot

import (
	"encoding/json"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func (w *Worker) simpleLeaderUpdate(parent *types.Header) {
	if parent != nil {
		//address := parent.Address
		address := parent.Address
		if !w.commiteeCheck(address, parent) {
			return
		}
		if w.commiteeLencheck() && w.whirly == nil {

			whirlyconfig := &config.ConsensusConfig{
				Type:        "whirly",
				ConsensusID: 1009,
				Whirly: &config.WhirlyConfig{
					Type:      "simple",
					BatchSize: 10,
					Timeout:   2000,
				},
				Nodes: w.config.Nodes,
				Keys:  w.config.Keys,
				F:     w.config.F,
			}
			s := simpleWhirly.NewSimpleWhirly(w.ID, 1009, whirlyconfig, w.Engine.exec, w.Engine.Adaptor, w.log)
			w.whirly = s
			w.Engine.Setwhirly(s)
			w.potsigchan = w.whirly.GetPoTByteEntrance()
			w.log.Errorf("[PoT]\t Start commitee consensus at epoch %d", parent.Height+1)
			return
		}
		potsig := &simpleWhirly.PoTSignal{
			Epoch:           int64(parent.Height),
			Proof:           parent.PoTProof[0],
			ID:              parent.Address,
			LeaderNetworkId: parent.PeerId,
		}
		b, err := json.Marshal(potsig)
		if err != nil {
			w.log.WithError(err)
			return
		}
		if w.potsigchan != nil {
			w.potsigchan <- b
		}
	}
}

func (w *Worker) commiteeCheck(id int64, header *types.Header) bool {
	if _, exist := w.commitee.Get(id); !exist {
		w.commitee.Set(id, header)
		return false
	}
	return true
}

func (w *Worker) commiteeLencheck() bool {
	return w.commitee.Len() == 4
}

func (w *Worker) GetPeerqueue() chan *types.Header {
	return w.peerMsgQueue
}
