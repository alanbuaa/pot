package pot

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func (w *Worker) simpleLeaderUpdate(parent *types.Header) {
	if parent != nil {
		// address := parent.Address
		address := parent.Address
		if !w.committeeCheck(address, parent) {
			return
		}
		if w.committeeSizeCheck() && w.whirly == nil {

			whirlyConfig := &config.ConsensusConfig{
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
			s := simpleWhirly.NewSimpleWhirly(w.ID, 1009, whirlyConfig, w.Engine.exec, w.Engine.Adaptor, w.log)
			w.whirly = s
			w.Engine.SetWhirly(s)
			w.potSignalChan = w.whirly.GetPoTByteEntrance()
			w.log.Errorf("[PoT]\t Start committee consensus at epoch %d", parent.Height+1)
			return
		}
		potSignal := &simpleWhirly.PoTSignal{
			Epoch:           int64(parent.Height),
			Proof:           parent.PoTProof[0],
			ID:              parent.Address,
			LeaderNetworkId: parent.PeerId,
		}
		b, err := json.Marshal(potSignal)
		if err != nil {
			w.log.WithError(err)
			return
		}
		if w.potSignalChan != nil {
			w.potSignalChan <- b
		}
	}
}

func (w *Worker) committeeCheck(id int64, header *types.Header) bool {
	if _, exist := w.committee.Get(id); !exist {
		w.committee.Set(id, header)
		return false
	}
	return true
}

func (w *Worker) committeeSizeCheck() bool {
	return w.committee.Len() == 4
}

func (w *Worker) GetPeerQueue() chan *types.Header {
	return w.peerMsgQueue
}

func (w *Worker) CommiteeUpdate(parent *types.Header, epoch uint64) {

	if parent != nil && parent.Difficulty.Cmp(common.Big0) == 0 {
		parentid := parent.PeerId

		if !w.CommiteeLenCheck() {
			w.AppendCommitee(parentid)
			if w.CommiteeLenCheck() && w.whirly == nil {
				whirlyConfig := &config.ConsensusConfig{
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
				s := simpleWhirly.NewSimpleWhirly(w.ID, 1009, whirlyConfig, w.Engine.exec, w.Engine.Adaptor, w.log)
				w.whirly = s
				w.Engine.SetWhirly(s)
				w.potSignalChan = w.whirly.GetPoTByteEntrance()
				w.log.Infof("[PoT]\t epoch %d:Start committee consensus whirly", parent.Height+1)
				return
			}
		} else {
			w.UpdateCommitee(parentid)
		}

		potsignal := &simpleWhirly.PoTSignal{
			Epoch:           int64(parent.Height),
			Proof:           parent.Hash(),
			ID:              parent.Address,
			LeaderNetworkId: parentid,
			Committee:       w.Commitee,
			CryptoElements:  nil,
		}

		b, err := json.Marshal(potsignal)
		if err != nil {
			w.log.WithError(err)
			return
		}

		if w.potSignalChan != nil {
			w.potSignalChan <- b
		}

	}
}

func (w *Worker) CommiteeLenCheck() bool {
	if len(w.Commitee) != Commiteelen {
		return false
	}
	return true
}

func (w *Worker) GetCommiteeLeader() string {
	return w.Commitee[Commiteelen-1]
}

func (w *Worker) UpdateCommitee(peerid string) []string {
	if w.CommiteeLenCheck() {
		w.Commitee = w.Commitee[1:Commiteelen]
		w.Commitee = append(w.Commitee, peerid)
	}
	if w.CommiteeLenCheck() {
		return w.Commitee
	}
	return nil
}

func (w *Worker) AppendCommitee(peerid string) {
	w.Commitee = append(w.Commitee, peerid)
}
