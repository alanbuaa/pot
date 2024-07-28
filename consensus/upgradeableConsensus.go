package consensus

import (
	"encoding/json"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

type UpgradeableConsensus struct {
	// basic items
	nid             int64
	cid             int64
	config          *config.ConsensusConfig
	executor        executor.Executor
	p2pAdaptor      p2p.P2PAdaptor
	log             *logrus.Entry
	msgByteEntrance chan []byte
	requestEntrance chan *pb.Request

	// upgrade items
	working       model.Consensus
	candidates    map[int64]model.Consensus
	consensusLock *sync.RWMutex // protects working and candidates

	inputBuffer map[types.TxHash]*pb.Request
	inputLock   *sync.Mutex // protects inputBuffer

	msgBuffer map[int64][][]byte // protects msgBuffer
	msgLock   *sync.Mutex

	outputBuffer map[int64][]types.ConsensusBlock
	outputLock   *sync.Mutex // protects outputBuffer

	upgradeBuffer map[types.TxHash]*config.ConsensusConfig
	upgradeLock   *sync.Mutex // protects upgradeBuffer

	timeWeight     map[types.TxHash]*TimeWeightRecord
	timeWeightLock *sync.Mutex // protects timeWeight

	commitedTxs map[types.TxHash][]byte
	commitLock  *sync.Mutex // protects commitedTxs and epoch
	epoch       int64

	// control items
	wg     *sync.WaitGroup
	closed chan []byte
}

func NewUpgradeableConsensus(nid int64, cid int64, cfg *config.ConsensusConfig, exec executor.Executor, p2pAdaptor p2p.P2PAdaptor, log *logrus.Entry) *UpgradeableConsensus {
	uc := &UpgradeableConsensus{
		nid:             nid,
		cid:             cid,
		config:          cfg,
		executor:        exec,
		p2pAdaptor:      p2pAdaptor,
		log:             log,
		msgByteEntrance: make(chan []byte, 10),
		requestEntrance: make(chan *pb.Request, 10),

		working:       nil,
		consensusLock: new(sync.RWMutex),
		candidates:    map[int64]model.Consensus{},

		inputBuffer: map[types.TxHash]*pb.Request{},
		inputLock:   new(sync.Mutex),

		msgBuffer: map[int64][][]byte{},
		msgLock:   new(sync.Mutex),

		outputBuffer: map[int64][]types.ConsensusBlock{},
		outputLock:   new(sync.Mutex),

		upgradeBuffer: map[types.TxHash]*config.ConsensusConfig{},
		upgradeLock:   new(sync.Mutex),

		timeWeight:     map[types.TxHash]*TimeWeightRecord{},
		timeWeightLock: new(sync.Mutex),

		commitedTxs: map[types.TxHash][]byte{},
		commitLock:  new(sync.Mutex),
		epoch:       0,

		wg:     new(sync.WaitGroup),
		closed: make(chan []byte),
	}
	// p2pAdaptor.SetReceiver(uc)
	p2pAdaptor.SetReceiver(uc.GetMsgByteEntrance())
	p2pAdaptor.Subscribe([]byte("consensus"))
	uc.wg.Add(1)

	cfg.Upgradeable.InitConsensus.Nodes = cfg.Nodes
	cfg.Upgradeable.InitConsensus.Keys = cfg.Keys
	cfg.Upgradeable.InitConsensus.F = cfg.F
	if cfg.Upgradeable.InitConsensus.Type != "whirly" {
		uc.working = BuildConsensus(
			nid,
			cfg.Upgradeable.InitConsensus.ConsensusID,
			cfg.Upgradeable.InitConsensus,
			uc,
			uc,
			log,
		)
	} else {
		uc.working = BuildConsensus(
			nid,
			cfg.Upgradeable.InitConsensus.ConsensusID,
			cfg.Upgradeable.InitConsensus,
			uc,
			p2pAdaptor,
			log,
		)
	}

	if uc.working == nil {
		log.Error("initialize working consensus failed")
		return nil
	}
	go uc.receiveMsg()

	// go func() {
	// 	for {
	// 		time.Sleep(1 * time.Second)
	// 		uc.log.WithField("count", runtime.NumGoroutine()).Info("current goroutine count")
	// 	}
	// }()
	return uc
}

// UpgradeableConsensus implements Consensus
func (uc *UpgradeableConsensus) GetConsensusID() int64 {
	return uc.cid
}

func (uc *UpgradeableConsensus) UpdateExternalStatus(status model.ExternalStatus) {
	return
}

func (uc *UpgradeableConsensus) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	return
}

func (uc *UpgradeableConsensus) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	return
}

func (uc *UpgradeableConsensus) GetMsgByteEntrance() chan<- []byte {
	return uc.msgByteEntrance
}

func (uc *UpgradeableConsensus) GetRequestEntrance() chan<- *pb.Request {
	return uc.requestEntrance
}

func (uc *UpgradeableConsensus) Stop() {
	uc.log.Info("[UpgradeableConsensus] Exiting...")
	close(uc.closed)
	uc.wg.Wait()
}

func (uc *UpgradeableConsensus) VerifyBlock(block []byte, proof []byte) bool {
	p := new(Proof)
	if err := json.Unmarshal(proof, p); err != nil {
		uc.log.WithError(err).Warn("unmarshal proof failed")
		return false
	}
	uc.consensusLock.RLock()
	defer uc.consensusLock.RUnlock()
	if p.Cid == uc.working.GetConsensusID() {
		return uc.working.VerifyBlock(p.Block, p.Proof)
	} else if candi, ok := uc.candidates[p.Cid]; ok {
		return candi.VerifyBlock(p.Block, p.Proof)
	} else {
		uc.log.WithField("cid", p.Cid).Warn("verify proof candidate not found")
		return true // maybe already stopped
	}
}

func (uc *UpgradeableConsensus) GetWeight(nid int64) float64 {
	uc.consensusLock.RLock()
	defer uc.consensusLock.RUnlock()
	return uc.working.GetWeight(nid)
}

func (uc *UpgradeableConsensus) GetMaxAdversaryWeight() float64 {
	uc.consensusLock.RLock()
	defer uc.consensusLock.RUnlock()
	return uc.working.GetMaxAdversaryWeight()
}

func (uc *UpgradeableConsensus) decodePacketByte(packetByte []byte) (*pb.Packet, error) {
	p := new(pb.Packet)
	err := proto.Unmarshal(packetByte, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (uc *UpgradeableConsensus) receiveMsg() {
	for {
		select {
		case request := <-uc.requestEntrance:
			go uc.handleRequest(request)
		case msgByte := <-uc.msgByteEntrance:
			// uc.log.Trace("[uc] received msg")
			packet, err := uc.decodePacketByte(msgByte)
			// uc.log.Debug("received packet ", pb.PacketType_name[int32(packet.Type)], " cid ", packet.ConsensusID)
			if err != nil {
				uc.log.WithError(err).Warn("decode packet failed")
				continue
			}
			go uc.handlePacket(packet)
		case <-uc.closed:
			uc.wg.Done()
			return
		}
	}
}

func (uc *UpgradeableConsensus) handleRequest(request *pb.Request) {
	rtx := types.RawTransaction(request.Tx)
	if !uc.executor.VerifyTx(rtx) {
		uc.log.Warn("tx verify failed")
		return
	}
	tx, err := rtx.ToTx()
	if err != nil {
		uc.log.WithError(err).Warn("decode into transaction failed")
		return
	}
	switch tx.Type {
	case pb.TransactionType_NORMAL:
		hash := rtx.Hash()
		uc.inputLock.Lock()
		uc.inputBuffer[hash] = request
		uc.inputLock.Unlock()
		uc.consensusLock.RLock()
		defer uc.consensusLock.RUnlock()
		uc.working.GetRequestEntrance() <- request
		for _, candi := range uc.candidates {
			candi.GetRequestEntrance() <- request
		}

	case pb.TransactionType_UPGRADE:
		uc.consensusLock.RLock()
		defer uc.consensusLock.RUnlock()
		uc.working.GetRequestEntrance() <- request
	case pb.TransactionType_LOCK:
		fallthrough
	case pb.TransactionType_TIMEVOTE:
		uc.log.Warn("transaction type not allowed", tx.Type.String())
	default:
		uc.log.Warn("transaction type unknown", tx.Type.String())
	}
}

// UpgradeableConsensus implements P2PServer
func (uc *UpgradeableConsensus) handlePacket(in *pb.Packet) {
	// msg := new(pb.Msg)
	// err := proto.Unmarshal(in.Msg, msg)
	// utils.LogOnError(err, "decode msg failed", uc.log)

	if in.Type == pb.PacketType_CLIENTPACKET {
		msg := new(pb.Msg)
		if err := proto.Unmarshal(in.Msg, msg); err != nil {
			uc.log.WithError(err).Warn("unmarshal msg failed")
			return
		}
		request := msg.GetRequest()
		if request == nil {
			uc.log.Warn("only request msg allowed in client packet")
			return
		}
		uc.handleRequest(request)
	} else {

		// uc.log.Debug("requiring lock")
		uc.consensusLock.RLock()
		// uc.log.Debug("required lock")
		cid := in.ConsensusID
		// uc.log.Debugf("received consensus msg, target consensus: %d", in.ConsensusID)
		if cid == int64(uc.working.GetConsensusID()) {
			// uc.log.Trace("[uc] msg send to consensus ", cid)
			uc.working.GetMsgByteEntrance() <- in.Msg
		} else if candi, ok := uc.candidates[cid]; ok {
			// uc.log.Trace("[uc] msg send to consensus ", cid)
			candi.GetMsgByteEntrance() <- in.Msg
		} else if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
			if in.Epoch < uc.epoch {
				uc.consensusLock.RUnlock()
				uc.log.WithFields(logrus.Fields{"msgepoch": in.Epoch, "epoch": uc.epoch, "cid": in.ConsensusID}).Trace("history msg")
				return
			}
			uc.msgLock.Lock()
			if _, ok := uc.msgBuffer[cid]; !ok {
				uc.msgBuffer[cid] = [][]byte{}
			}
			uc.msgBuffer[cid] = append(uc.msgBuffer[cid], in.Msg)
			uc.msgLock.Unlock()
		} else {
			uc.log.WithFields(logrus.Fields{
				"cid":     in.ConsensusID,
				"working": uc.working.GetConsensusID(),
			}).Warn("msg without receiver")
		}
		uc.consensusLock.RUnlock()
	}
}

/*
Upgrade consensus to nextCid

	This should be called after making sure consensusLock is required
	and nextCid exists
*/
func (uc *UpgradeableConsensus) upgradeConsensusTo(nextCid int64) {
	uc.log.WithField("cid", nextCid).Info("upgrading consensus")
	var nextWorking model.Consensus
	for cid, c := range uc.candidates {
		if cid == nextCid {
			nextWorking = c
		} else {
			c.Stop()
		}
	}
	// clear candidates
	uc.candidates = map[int64]model.Consensus{}

	// execute output
	for _, block := range uc.outputBuffer[nextCid] {
		uc.executeNormalTx(block, []byte{}, nextCid)
	}

	// redo request
	uc.inputLock.Lock()
	for _, request := range uc.inputBuffer {
		nextWorking.GetRequestEntrance() <- request
	}
	uc.inputLock.Unlock()

	// clear outputBuffer
	uc.outputLock.Lock()
	uc.outputBuffer = map[int64][]types.ConsensusBlock{}
	uc.outputLock.Unlock()

	if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
		// clear msgBuffer
		uc.msgLock.Lock()
		uc.msgBuffer = map[int64][][]byte{}
		uc.msgLock.Unlock()
	}

	uc.working.Stop()
	uc.working = nextWorking
	uc.log.WithField("new", nextCid).Info("upgrade consensus done")
	uc.epoch++
}

func (uc *UpgradeableConsensus) startNewConsensus(cc *config.ConsensusConfig) {
	uc.log.WithField("cid", cc.ConsensusID).Debug("starting new consensus")
	switch cc.Type {
	case "hotstuff":
		fallthrough
	case "whirly":
		uc.consensusLock.Lock()
		if _, ok := uc.candidates[cc.ConsensusID]; ok {
			uc.log.WithField("cid", cc.ConsensusID).Warn("consensus already started")
			uc.consensusLock.Unlock()
		}
		nc := BuildConsensus(uc.nid, cc.ConsensusID, cc, uc, uc, uc.log)

		if nc == nil {
			log.Error("start new consensus failed")
			return
		}
		uc.candidates[cc.ConsensusID] = nc
		uc.consensusLock.Unlock()

		if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
			ch := nc.GetMsgByteEntrance()
			uc.msgLock.Lock()
			if buf := uc.msgBuffer[cc.ConsensusID]; len(buf) > 0 {
				for _, msg := range buf {
					ch <- msg
				}
			}
			delete(uc.msgBuffer, cc.ConsensusID)
			uc.msgLock.Unlock()
		}
	default:
		uc.log.Warnf("consensus: %s not implemented yet", cc.Type)
		return
	}
}
