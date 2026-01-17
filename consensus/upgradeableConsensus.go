package consensus

import (
	"encoding/json"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
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
	log = log.WithField("module", "UPGRADECC").WithField("c_id", cid)
	log.Info("Initializing Upgradeable consensus")

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
	log.Debug("UpgradeableConsensus data structures initialized")
	// p2pAdaptor.SetReceiver(uc)
	p2pAdaptor.SetReceiver(uc.GetMsgByteEntrance())
	p2pAdaptor.Subscribe([]byte("consensus"))
	log.Debug("P2P adaptor configured, subscribed to 'consensus' topic")
	uc.wg.Add(1)

	cfg.Upgradeable.InitConsensus.Nodes = cfg.Nodes
	cfg.Upgradeable.InitConsensus.Keys = cfg.Keys
	cfg.Upgradeable.InitConsensus.F = cfg.F
	log.WithFields(logrus.Fields{
		"init_type": cfg.Upgradeable.InitConsensus.Type,
		"init_cid":  cfg.Upgradeable.InitConsensus.ConsensusID,
	}).Info("Building initial working consensus")
	if cfg.Upgradeable.InitConsensus.Type != "whirly" {
		uc.working, _ = BuildConsensus(
			nid,
			cfg.Upgradeable.InitConsensus.ConsensusID,
			cfg.Upgradeable.InitConsensus,
			uc,
			uc,
			log,
		)
	} else {
		uc.working, _ = BuildConsensus(
			nid,
			cfg.Upgradeable.InitConsensus.ConsensusID,
			cfg.Upgradeable.InitConsensus,
			uc,
			p2pAdaptor,
			log,
		)
	}

	if uc.working == nil {
		log.Error("Failed to initialize working consensus")
		return nil
	}
	log.Info("Initial working consensus initialized successfully")
	go uc.receiveMsg()
	log.Debug("Message receiver goroutine started")

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
	uc.log.Info("Stopping UpgradeableConsensus...")
	close(uc.closed)
	uc.log.Debug("Waiting for goroutines to finish")
	uc.wg.Wait()
	uc.log.Info("UpgradeableConsensus stopped successfully")
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
		uc.log.Warn("Transaction verification failed")
		return
	}
	tx, err := rtx.ToTx()
	if err != nil {
		uc.log.WithError(err).Warn("Failed to decode transaction")
		return
	}
	uc.log.WithField("type", tx.Type.String()).Debug("Processing transaction request")
	switch tx.Type {
	case pb.TransactionType_NORMAL:
		hash := rtx.Hash()
		uc.inputLock.Lock()
		uc.inputBuffer[hash] = request
		uc.inputLock.Unlock()
		uc.log.WithField("hash", hash).Trace("Normal transaction added to input buffer")
		uc.consensusLock.RLock()
		defer uc.consensusLock.RUnlock()
		uc.working.GetRequestEntrance() <- request
		for cid, candi := range uc.candidates {
			uc.log.WithField("candidate_cid", cid).Trace("Forwarding transaction to candidate consensus")
			candi.GetRequestEntrance() <- request
		}

	case pb.TransactionType_UPGRADE:
		uc.log.Info("Received upgrade transaction")
		uc.consensusLock.RLock()
		defer uc.consensusLock.RUnlock()
		uc.working.GetRequestEntrance() <- request
	case pb.TransactionType_LOCK:
		fallthrough
	case pb.TransactionType_TIMEVOTE:
		uc.log.WithField("type", tx.Type.String()).Warn("Transaction type not allowed for external requests")
	default:
		uc.log.WithField("type", tx.Type.String()).Warn("Unknown transaction type")
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
	uc.log.WithFields(logrus.Fields{"from_cid": uc.working.GetConsensusID(), "to_cid": nextCid}).Info("Starting consensus upgrade")
	var nextWorking model.Consensus
	candidateCount := len(uc.candidates)
	uc.log.WithField("candidate_count", candidateCount).Debug("Processing candidate consensuses")
	for cid, c := range uc.candidates {
		if cid == nextCid {
			nextWorking = c
			uc.log.WithField("cid", cid).Debug("Selected as new working consensus")
		} else {
			uc.log.WithField("cid", cid).Debug("Stopping non-selected candidate")
			c.Stop()
		}
	}
	// clear candidates
	uc.candidates = map[int64]model.Consensus{}
	uc.log.Debug("Candidate consensuses cleared")

	// execute output
	outputBlockCount := len(uc.outputBuffer[nextCid])
	uc.log.WithField("block_count", outputBlockCount).Debug("Executing buffered output blocks")
	for _, block := range uc.outputBuffer[nextCid] {
		uc.executeNormalTx(block, []byte{}, nextCid)
	}

	// redo request
	uc.inputLock.Lock()
	inputCount := len(uc.inputBuffer)
	uc.log.WithField("request_count", inputCount).Debug("Replaying buffered requests")
	for _, request := range uc.inputBuffer {
		nextWorking.GetRequestEntrance() <- request
	}
	uc.inputLock.Unlock()

	// clear outputBuffer
	uc.outputLock.Lock()
	uc.outputBuffer = map[int64][]types.ConsensusBlock{}
	uc.outputLock.Unlock()
	uc.log.Debug("Output buffer cleared")

	if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
		// clear msgBuffer
		uc.msgLock.Lock()
		msgCount := len(uc.msgBuffer)
		uc.msgBuffer = map[int64][][]byte{}
		uc.msgLock.Unlock()
		uc.log.WithField("cleared_msg_count", msgCount).Debug("Message buffer cleared (async mode)")
	}

	uc.log.WithField("old_cid", uc.working.GetConsensusID()).Debug("Stopping old working consensus")
	uc.working.Stop()
	uc.working = nextWorking
	uc.epoch++
	uc.log.WithFields(logrus.Fields{"new_cid": nextCid, "epoch": uc.epoch}).Info("Consensus upgrade completed successfully")
}

func (uc *UpgradeableConsensus) startNewConsensus(cc *config.ConsensusConfig) {
	uc.log.WithFields(logrus.Fields{"cid": cc.ConsensusID, "type": cc.Type}).Info("Starting new candidate consensus")
	switch cc.Type {
	case "hotstuff":
		fallthrough
	case "whirly":
		uc.consensusLock.Lock()
		if _, ok := uc.candidates[cc.ConsensusID]; ok {
			uc.log.WithField("cid", cc.ConsensusID).Warn("Candidate consensus already exists, skipping creation")
			uc.consensusLock.Unlock()
			return
		}
		uc.log.WithField("cid", cc.ConsensusID).Debug("Building candidate consensus instance")
		nc, _ := BuildConsensus(uc.nid, cc.ConsensusID, cc, uc, uc, uc.log)

		if nc == nil {
			uc.log.WithField("cid", cc.ConsensusID).Error("Failed to build candidate consensus")
			uc.consensusLock.Unlock()
			return
		}
		uc.candidates[cc.ConsensusID] = nc
		uc.consensusLock.Unlock()
		uc.log.WithField("cid", cc.ConsensusID).Info("Candidate consensus started successfully")

		if uc.config.Upgradeable.NetworkType == config.NetworkAsync {
			ch := nc.GetMsgByteEntrance()
			uc.msgLock.Lock()
			if buf := uc.msgBuffer[cc.ConsensusID]; len(buf) > 0 {
				uc.log.WithFields(logrus.Fields{"cid": cc.ConsensusID, "msg_count": len(buf)}).Debug("Replaying buffered messages")
				for _, msg := range buf {
					ch <- msg
				}
			}
			delete(uc.msgBuffer, cc.ConsensusID)
			uc.msgLock.Unlock()
		}
	default:
		uc.log.WithField("type", cc.Type).Warn("Consensus type not supported for candidate creation")
		return
	}
}

func (e *UpgradeableConsensus) GetConsensusType() string {
	return "upgradeable"
}
