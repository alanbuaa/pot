package pot

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

type Sharding struct {
	Name            string
	Id              int32
	LeaderAddress   string
	Committee       []string
	consensusconfig config.ConsensusConfig
}

//func (w *Worker) simpleLeaderUpdate(parent *types.Header) {
//	if parent != nil {
//		// address := parent.Address
//		address := parent.Address
//		if !w.committeeCheck(address, parent) {
//			return
//		}
//		if w.committeeSizeCheck() && w.whirly == nil {
//
//			whirlyConfig := &config.ConsensusConfig{
//				Type:        "whirly",
//				ConsensusID: 1009,
//				Whirly: &config.WhirlyConfig{
//					Type:      "simple",
//					BatchSize: 10,
//					Timeout:   2000,
//				},
//				Nodes: w.config.Nodes,
//				Keys:  w.config.Keys,
//				F:     w.config.F,
//			}
//			s := simpleWhirly.NewSimpleWhirly(w.ID, 1009, whirlyConfig, w.Engine.exec, w.Engine.Adaptor, w.log, "", nil)
//			w.whirly = s
//			//w.Engine.SetWhirly(s)
//			// w.potSignalChan = w.whirly.GetPoTByteEntrance()
//			w.log.Errorf("[PoT]\t Start committee consensus at epoch %d", parent.ExecHeight+1)
//			return
//		}
//		potSignal := &simpleWhirly.PoTSignal{
//			Epoch:               int64(parent.ExecHeight),
//			Proof:               parent.PoTProof[0],
//			ID:                  parent.Address,
//			LeaderPublicAddress: parent.PeerId,
//		}
//		b, err := json.Marshal(potSignal)
//		if err != nil {
//			w.log.WithError(err)
//			return
//		}
//		if w.potSignalChan != nil {
//			w.potSignalChan <- b
//		}
//	}
//}

//func (w *Worker) committeeCheck(id int64, header *types.Header) bool {
//	if _, exist := w.committee.Get(id); !exist {
//		w.committee.Set(id, header)
//		return false
//	}
//	return true
//}
//
//func (w *Worker) committeeSizeCheck() bool {
//	return w.committee.Len() == 4
//}

func (w *Worker) GetPeerQueue() chan *types.Block {
	return w.peerMsgQueue
}

func (w *Worker) CommitteeUpdate(epoch uint64) {

	if epoch >= CommiteeDelay+Commiteelen {
		committee := make([]string, Commiteelen)
		selfaddress := make([]string, 0)
		for i := uint64(0); i < Commiteelen; i++ {
			block, err := w.chainReader.GetByHeight(epoch - CommiteeDelay - i)
			if err != nil {
				return
			}
			if block != nil {
				header := block.GetHeader()
				committee[i] = hexutil.Encode(header.PublicKey)
				flag, _ := w.TryFindKey(crypto.Convert(header.Hash()))
				if flag {
					selfaddress = append(selfaddress, hexutil.Encode(header.PublicKey))
				}
			}
		}
		// potsignal := &simpleWhirly.PoTSignal{
		// 	Epoch:               int64(epoch),
		// 	Proof:               nil,
		// 	ID:                  0,
		// 	LeaderPublicAddress: committee[0],
		// 	Committee:           committee,
		// 	SelfPublicAddress:   selfaddress,
		// 	CryptoElements:      nil,
		// }
		whilyConsensus := &config.WhirlyConfig{
			Type:      "simple",
			BatchSize: 2,
			Timeout:   2,
		}

		consensus := config.ConsensusConfig{
			Type:        "whirly",
			ConsensusID: 1201,
			Whirly:      whilyConsensus,
			Nodes:       w.config.Nodes,
			Topic:       w.config.Topic,
			F:           w.config.F,
		}

		sharding1 := simpleWhirly.PoTSharding{
			Name:                "default",
			ParentSharding:      nil,
			LeaderPublicAddress: committee[0],
			Committee:           committee,
			CryptoElements:      nil,
			SubConsensus:        consensus,
		}

		w.log.Error(len(committee))

		sharding2 := simpleWhirly.PoTSharding{
			Name:                "hello_world",
			ParentSharding:      nil,
			LeaderPublicAddress: committee[0],
			Committee:           committee,
			CryptoElements:      nil,
			SubConsensus:        consensus,
		}
		shardings := []simpleWhirly.PoTSharding{sharding1, sharding2}

		potsignal := &simpleWhirly.PoTSignal{
			Epoch:             int64(epoch),
			Proof:             make([]byte, 0),
			ID:                0,
			SelfPublicAddress: selfaddress,
			Shardings:         shardings,
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
	//if epoch > 10 && w.ID == 1 {
	//	block, err := w.chainReader.GetByHeight(epoch - 1)
	//	if err != nil {
	//		return
	//	}
	//	header := block.GetHeader()
	//	fill, err := os.OpenFile("difficulty", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	_, err = fill.WriteString(fmt.Sprintf("%d\n", header.Difficulty.Int64()))
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	fill.Close()
	//}
}

func (w *Worker) ImprovedCommitteeUpdate(epoch uint64, committeeNum int) {
	backupcommitee, selfaddr := w.GetBackupCommitee(epoch)
	commitee := w.ShuffleCommitee(epoch, backupcommitee, committeeNum)
	for i := 0; i < committeeNum; i++ {
		selfaddress := make([]string, 0)
		for _, s := range commitee[i] {
			if IsContain(selfaddr, s) {
				selfaddress = append(selfaddress, s)
			}
		}
	}

}

func IsContain(parent []string, son string) bool {
	for _, s := range parent {
		if s == son {
			return true
		}
	}
	return false
}

func (w *Worker) SetWhirly(impl *simpleWhirly.NodeController) {
	w.whirly = impl
	w.potSignalChan = impl.GetPoTByteEntrance()
}

//func (w *Worker) CommiteeLenCheck() bool {
//	if len(w.Commitee) != Commiteelen {
//		return false
//	}
//	return true
//}

//func (w *Worker) GetCommiteeLeader() string {
//	return w.Commitee[Commiteelen-1]
//}

//func (w *Worker) UpdateCommitee(commiteeid int, peerid string) []string {
//	if w.CommiteeLenCheck() {
//		w.Commitee[commiteeid] = w.Commitee[commiteeid][1:Commiteelen]
//		w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
//	}
//	if w.CommiteeLenCheck() {
//		return w.Commitee[commiteeid]
//	}
//	return nil
//}
//
//func (w *Worker) AppendCommitee(commiteeid int, peerid string) {
//	w.Commitee[commiteeid] = append(w.Commitee[commiteeid], peerid)
//}

func (w *Worker) GetBackupCommitee(epoch uint64) ([]string, []string) {
	commitee := make([]string, Commiteelen)
	selfAddr := make([]string, 0)
	if epoch > BackupCommiteeSize && epoch%BackupCommiteeSize == 0 {
		for i := uint64(1); i <= BackupCommiteeSize; i++ {
			getepoch := epoch - 64 + i
			block, err := w.chainReader.GetByHeight(getepoch)
			if err != nil {
				return nil, nil
			}
			if block != nil {
				header := block.GetHeader()
				commiteekey := header.PublicKey
				commitee = append(commitee, hexutil.Encode(commiteekey))
				flag, _ := w.TryFindKey(crypto.Convert(header.Hash()))
				if flag {
					selfAddr = append(selfAddr, hexutil.Encode(header.PublicKey))
				}
			}
		}
		w.BackupCommitee = commitee
		w.SelfAddress = selfAddr
	} else {
		commitee = w.BackupCommitee
		selfAddr = w.SelfAddress
	}
	return commitee, selfAddr
}

// TODO: Shuffle function need to complete
func (w *Worker) ShuffleCommitee(epoch uint64, backupcommitee []string, comitteenum int) [][]string {
	commitees := make([][]string, comitteenum)
	for i := 0; i < comitteenum; i++ {
		if len(backupcommitee) > 4 {
			commitees[i] = backupcommitee[:4]
		} else {
			commitees[i] = backupcommitee
		}
	}
	return commitees
}
