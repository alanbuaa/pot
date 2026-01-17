package eventdriven

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/niclabs/tcrsa"
	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/hotstuff"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

type Event uint8

const (
	QCFinish Event = iota
	ReceiveProposal
	ReceiveNewView
)

type EventDrivenHotStuff interface {
	Update(block *pb.WhirlyBlock)
	OnCommit(block *pb.WhirlyBlock)
	OnReceiveProposal(msg *pb.Prepare) (*tcrsa.SigShare, error)
	OnReceiveVote(partSig *tcrsa.SigShare)
	OnPropose()
}

type EventDrivenHotStuffImpl struct {
	hotstuff.HotStuffImpl
	lock          sync.Mutex
	pacemaker     Pacemaker
	bLeaf         *pb.WhirlyBlock
	bLock         *pb.WhirlyBlock
	bExec         *pb.WhirlyBlock
	qcHigh        *pb.QuorumCert
	vHeight       uint64
	waitProposal  *sync.Cond
	pendingUpdate chan *pb.WhirlyBlock
	eventChannels []chan Event
}

func NewEventDrivenHotStuff(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *EventDrivenHotStuffImpl {
	log = log.WithField("module", "EVENTHST").WithField("c_id", cid)
	log.Info("Initializing Event-Driven HotStuff consensus")

	log.Debug("Generating genesis block")
	genesisBlock := hotstuff.GenerateGenesisBlock()
	ehs := &EventDrivenHotStuffImpl{
		bLeaf:         genesisBlock,
		bLock:         genesisBlock,
		bExec:         genesisBlock,
		qcHigh:        nil,
		vHeight:       genesisBlock.Height,
		pendingUpdate: make(chan *pb.WhirlyBlock, 1),
		eventChannels: make([]chan Event, 0),
	}
	ehs.Init(id, cid, cfg, exec, p2pAdaptor, log)
	err := ehs.BlockStorage.Put(genesisBlock)
	if err != nil {
		log.WithError(err).Fatal("Failed to store genesis block")
	}
	log.Debug("Genesis block stored successfully")

	// make view number equal to 0 to create genesis block QC
	ehs.View = hotstuff.NewView(0, 1)
	ehs.qcHigh = ehs.QC(pb.MsgType_PREPARE_VOTE, nil, genesisBlock.Hash)
	// view number add 1
	ehs.View.ViewNum++
	ehs.waitProposal = sync.NewCond(&ehs.lock)

	log.WithField("replica_id", id).Debug("Block storage initialized")
	log.WithField("replica_id", id).Debug("Transaction cache initialized")

	// Initialize timers
	ehs.TimeChan = utils.NewTimer(time.Duration(ehs.Config.HotStuff.Timeout) * time.Second)
	ehs.TimeChan.Init()
	log.WithField("timeout_seconds", ehs.Config.HotStuff.Timeout).Debug("View timeout timer initialized")

	ehs.BatchTimeChan = utils.NewTimer(time.Duration(ehs.Config.HotStuff.BatchTimeout) * time.Second)
	ehs.BatchTimeChan.Init()
	log.WithField("batch_timeout_seconds", ehs.Config.HotStuff.BatchTimeout).Debug("Batch timer initialized")

	// Initialize current execution state
	ehs.CurExec = &hotstuff.CurProposal{
		Node:         nil,
		DocumentHash: nil,
		PrepareVote:  make([]*tcrsa.SigShare, 0),
		HighQC:       make([]*pb.QuorumCert, 0),
	}

	// Initialize pacemaker
	ehs.pacemaker = NewPacemaker(ehs, log)
	log.Info("Event-Driven HotStuff consensus initialized successfully")

	go ehs.updateAsync()
	go ehs.receiveMsg()
	go ehs.pacemaker.Run(ehs.Closed)
	return ehs
}

func (ehs *EventDrivenHotStuffImpl) emitEvent(event Event) {
	for _, c := range ehs.eventChannels {
		c <- event
	}
}

func (ehs *EventDrivenHotStuffImpl) GetHeight() uint64 {
	return ehs.bLeaf.Height
}

func (ehs *EventDrivenHotStuffImpl) GetVHeight() uint64 {
	return ehs.vHeight
}

func (ehs *EventDrivenHotStuffImpl) GetLeaf() *pb.WhirlyBlock {
	ehs.lock.Lock()
	defer ehs.lock.Unlock()
	return ehs.bLeaf
}

func (ehs *EventDrivenHotStuffImpl) SetLeaf(b *pb.WhirlyBlock) {
	ehs.lock.Lock()
	defer ehs.lock.Unlock()
	ehs.bLeaf = b
}

func (ehs *EventDrivenHotStuffImpl) GetHighQC() *pb.QuorumCert {
	ehs.lock.Lock()
	defer ehs.lock.Unlock()
	return ehs.qcHigh
}

func (ehs *EventDrivenHotStuffImpl) GetEvents() chan Event {
	c := make(chan Event)
	ehs.eventChannels = append(ehs.eventChannels, c)
	return c
}

// func (ehs *EventDrivenHotStuffImpl) Stop() {
// 	ehs.cancel()
// 	close(ehs.MsgByteEntrance)
// 	close(ehs.RequestEntrance)
// 	ehs.WhirlyBlockStorage.Close()
// 	// _ = os.RemoveAll("dbfile/node" + strconv.Itoa(int(ehs.ID)))
// }

func (ehs *EventDrivenHotStuffImpl) receiveMsg() {
	ehs.Wg.Add(1)
	for {
		select {
		case <-ehs.Closed:
			ehs.Wg.Done()
			return
		case msgByte := <-ehs.MsgByteEntrance:
			// if !ok {
			// 	return // closed
			// }
			msg, err := ehs.DecodeMsgByte(msgByte)
			if err != nil {
				ehs.Log.WithError(err).Warn("decode message failed")
				continue
			}
			ehs.handleMsg(msg)
		case request := <-ehs.RequestEntrance:
			// if !ok {
			// 	return // closed
			// }
			// ehs.Log.Debug("received request")
			ehs.handleMsg(&pb.Msg{Payload: &pb.Msg_Request{Request: request}})
		}
	}
}

func (ehs *EventDrivenHotStuffImpl) handleMsg(msg *pb.Msg) {
	switch msg.Payload.(type) {
	case *pb.Msg_Request:
		request := msg.GetRequest()
		// ehs.Log.WithField("content", request.String()).Debug("[EVENT-DRIVEN HOTSTUFF] Get request msg.")
		// put the cmd into the cmdset
		ehs.MemPool.Add(types.RawTransaction(request.Tx))
		// send the request to the leader, if the replica is not the leader
		if ehs.ID != ehs.GetLeader() {
			_ = ehs.Unicast(ehs.GetNetworkInfo()[ehs.GetLeader()], msg)
			return
		}
	case *pb.Msg_Prepare:
		prepareMsg := msg.GetPrepare()
		if prepareMsg.ViewNum < ehs.View.ViewNum {
			return
		}
		partSig, err := ehs.OnReceiveProposal(prepareMsg)
		if err != nil {
			ehs.Log.Error(err.Error())
		}
		// view change
		ehs.View.ViewNum = prepareMsg.ViewNum + 1

		// broadcast proof
		ehs.Broadcast(msg)
		ehs.View.Primary = ehs.GetLeader()
		if ehs.View.Primary == ehs.ID {
			// vote self
			// ehs.Log.WithField("view", ehs.View.ViewNum).WithField("msgview", prepareVoteMsg.ViewNum).Debug("before on receive")
			ehs.OnReceiveVote(partSig)
		} else {
			// send vote to the leader
			partSigBytes, _ := json.Marshal(partSig)
			voteMsg := ehs.VoteMsg(pb.MsgType_PREPARE_VOTE, prepareMsg.CurProposal, nil, partSigBytes)
			_ = ehs.Unicast(ehs.GetNetworkInfo()[ehs.GetLeader()], voteMsg)
			ehs.CurExec = hotstuff.NewCurProposal()
		}
	case *pb.Msg_PrepareVote:
		prepareVoteMsg := msg.GetPrepareVote()
		if ehs.CurExec.Node == nil {
			return
		}
		partSig := &tcrsa.SigShare{}
		err := json.Unmarshal(prepareVoteMsg.PartialSig, partSig)
		if err != nil {
			ehs.Log.WithField("error", err.Error()).Error("Unmarshal partSig failed.")
			return
		}
		// ehs.Log.WithField("view", ehs.View.ViewNum).WithField("msgview", prepareVoteMsg.ViewNum).Debug("before on receive")
		ehs.OnReceiveVote(partSig)
	case *pb.Msg_NewView:
		newViewMsg := msg.GetNewView()
		ehs.Log.WithFields(logrus.Fields{
			"highqc_count": len(ehs.CurExec.HighQC) + 1,
			"threshold":    ehs.Config.F,
		}).Debug("Received NEWVIEW message")
		// wait for 2f votes
		ehs.CurExec.HighQC = append(ehs.CurExec.HighQC, newViewMsg.PrepareQC)
		if len(ehs.CurExec.HighQC) == ehs.Config.F {
			ehs.Log.Debug("Sufficient NEWVIEW messages received")
			for _, cert := range ehs.CurExec.HighQC {
				if cert.ViewNum > ehs.GetHighQC().ViewNum {
					ehs.qcHigh = cert
				}
			}
			ehs.pacemaker.OnReceiverNewView(ehs.qcHigh)
		}
	default:
		ehs.Log.Warn("Received unsupported message type")
	}
}

// updateAsync receive block
func (ehs *EventDrivenHotStuffImpl) updateAsync() {
	ehs.Wg.Add(1)
	for {
		select {
		case <-ehs.Closed:
			ehs.Wg.Done()
			return
		case b := <-ehs.pendingUpdate:
			ehs.Update(b)
		}
	}
}

// Update update blocks before block
func (ehs *EventDrivenHotStuffImpl) Update(block *pb.WhirlyBlock) {
	// block1 = b'', block2 = b', block3 = b
	block1, err := ehs.BlockStorage.BlockOf(block.Justify)
	if err != nil && err != leveldb.ErrNotFound {
		ehs.Log.Fatal(err)
	}
	if block1 == nil || block1.Committed {
		return
	}

	ehs.lock.Lock()
	defer ehs.lock.Unlock()

	ehs.Log.WithFields(logrus.Fields{
		"block_hash": hex.EncodeToString(block1.Hash),
		"view":       ehs.View.ViewNum,
	}).Debug("Pre-commit phase")
	// pre-commit block1
	ehs.pacemaker.UpdateHighQC(block.Justify)

	block2, err := ehs.BlockStorage.BlockOf(block1.Justify)
	if err != nil && err != leveldb.ErrNotFound {
		ehs.Log.WithError(err).Fatal("Failed to get block from justify")
	}
	if block2 == nil || block2.Committed {
		return
	}

	if block2.Height > ehs.bLock.Height {
		ehs.bLock = block2
		ehs.Log.WithFields(logrus.Fields{
			"block_hash": hex.EncodeToString(block2.Hash),
			"view":       ehs.View.ViewNum,
		}).Debug("Commit phase")
	}

	block3, err := ehs.BlockStorage.BlockOf(block2.Justify)
	if err != nil && err != leveldb.ErrNotFound {
		ehs.Log.WithError(err).Fatal("Failed to get block from justify")
	}
	if block3 == nil || block3.Committed {
		return
	}

	if bytes.Equal(block1.ParentHash, block2.Hash) && bytes.Equal(block2.ParentHash, block3.Hash) {
		ehs.Log.WithFields(logrus.Fields{
			"block_hash": hex.EncodeToString(block3.Hash),
			"view":       ehs.View.ViewNum,
		}).Info("Decide phase")
		ehs.OnCommit(block3)
		ehs.bExec = block3
	}
}

func (ehs *EventDrivenHotStuffImpl) OnCommit(block *pb.WhirlyBlock) {
	if ehs.bExec.Height < block.Height {
		if parent, _ := ehs.BlockStorage.ParentOf(block); parent != nil {
			ehs.OnCommit(parent)
		}
		err := ehs.BlockStorage.UpdateState(block)
		if err != nil {
			ehs.Log.WithError(err).Fatal("Failed to update block state")
		}
		ehs.Log.WithFields(logrus.Fields{
			"block_hash": hex.EncodeToString(block.Hash),
			"view":       ehs.View.ViewNum,
		}).Debug("Executing block")
		go ehs.ProcessProposal(block, []byte{})
	}
}

func (ehs *EventDrivenHotStuffImpl) OnReceiveProposal(msg *pb.Prepare) (*tcrsa.SigShare, error) {
	newBlock := msg.CurProposal
	ehs.Log.WithFields(logrus.Fields{
		"block_hash": hex.EncodeToString(newBlock.Hash),
		"view":       ehs.View.ViewNum,
	}).Debug("Received proposal")

	// store the block
	err := ehs.BlockStorage.Put(newBlock)
	if err != nil {
		ehs.Log.WithError(err).WithField("block_hash", hex.EncodeToString(newBlock.Hash)).Error("Failed to store new block")
	}
	// ehs.Log.Debug("require lock")
	ehs.lock.Lock()

	// ehs.Log.Debug("start waiting block")
	qcBlock, _ := ehs.expectBlock(newBlock.Justify.BlockHash)
	// ehs.Log.Debug("end waiting block")

	if newBlock.Height <= ehs.vHeight {
		ehs.lock.Unlock()
		ehs.Log.WithFields(logrus.Fields{
			"block_height": newBlock.Height,
			"v_height":     ehs.vHeight,
		}).Warn("Block height less than or equal to vHeight, rejecting")
		return nil, errors.New("block was not accepted")
	}
	safe := false

	// changed here for extend
	if qcBlock != nil && qcBlock.Height >= ehs.bLock.Height {
		safe = true
	} else {
		if qcBlock == nil {
			ehs.Log.Warn("QC block is nil")
		} else if qcBlock.Height <= ehs.bLock.Height {
			ehs.Log.WithFields(logrus.Fields{
				"qc_height":   qcBlock.Height,
				"lock_height": ehs.bLock.Height,
			}).Warn("QC block height not greater than lock height")
		}
		ehs.Log.Warn("Liveness condition failed")
		b := newBlock
		ok := true
		for ok && b.Height > ehs.bLock.Height+1 {
			b, _ := ehs.BlockStorage.Get(b.ParentHash)
			if b == nil {
				ok = false
			}
		}
		if ok && bytes.Equal(b.ParentHash, ehs.bLock.Hash) {
			safe = true
		} else {
			ehs.Log.Warn("Safety condition failed")
		}
	}
	// unsafe, return
	if !safe {
		ehs.lock.Unlock()
		ehs.Log.Warn("Block is not safe, rejecting")
		return nil, errors.New("block was not accepted")
	}
	ehs.Log.Debug("Block accepted")
	// update vHeight
	ehs.vHeight = newBlock.Height
	ehs.MemPool.MarkProposed(types.RawTxArrayFromBytes(newBlock.Txs))
	// ehs.Log.Debug("release lock")
	ehs.lock.Unlock()
	ehs.waitProposal.Broadcast()
	ehs.emitEvent(ReceiveProposal)
	ehs.pendingUpdate <- newBlock
	marshal, _ := proto.Marshal(msg)
	ehs.CurExec.DocumentHash, _ = crypto.CreateDocumentHash(marshal, ehs.Config.Keys.PublicKey)
	ehs.CurExec.Node = newBlock
	partSig, err := crypto.TSign(ehs.CurExec.DocumentHash, ehs.Config.Keys.PrivateKey, ehs.Config.Keys.PublicKey)
	if err != nil {
		ehs.Log.WithError(err).Warn("Failed to create threshold signature")
	}
	return partSig, nil
}

func (ehs *EventDrivenHotStuffImpl) OnReceiveVote(partSig *tcrsa.SigShare) {
	// verify partSig
	err := crypto.VerifyPartSig(partSig, ehs.CurExec.DocumentHash, ehs.Config.Keys.PublicKey)
	if err != nil {
		ehs.Log.WithFields(logrus.Fields{
			"document_hash": hex.EncodeToString(ehs.CurExec.DocumentHash),
			"view":          ehs.View.ViewNum,
		}).WithError(err).Warn("Partial signature verification failed")
		return
	}
	ehs.CurExec.PrepareVote = append(ehs.CurExec.PrepareVote, partSig)
	ehs.Log.WithFields(logrus.Fields{
		"vote_count": len(ehs.CurExec.PrepareVote),
		"threshold":  2*ehs.Config.F + 1,
	}).Debug("Collected vote")
	if len(ehs.CurExec.PrepareVote) == 2*ehs.Config.F+1 {
		ehs.Log.Debug("Threshold reached, creating QC")
		// create full signature
		signature, _ := crypto.CreateFullSignature(ehs.CurExec.DocumentHash, ehs.CurExec.PrepareVote,
			ehs.Config.Keys.PublicKey)
		// create a QC
		qc := ehs.QC(pb.MsgType_PREPARE_VOTE, signature, ehs.CurExec.Node.Hash)
		// update qcHigh
		ehs.pacemaker.UpdateHighQC(qc)
		ehs.CurExec = hotstuff.NewCurProposal()
		ehs.emitEvent(QCFinish)
	}
}

func (ehs *EventDrivenHotStuffImpl) OnPropose() {
	if ehs.GetLeader() != ehs.ID {
		return
	}
	ehs.Log.WithField("view", ehs.View.ViewNum).Debug("Creating new proposal")
	time.Sleep(time.Second * time.Duration(ehs.Config.HotStuff.BatchTimeout))
	ehs.BatchTimeChan.SoftStartTimer()
	txs := ehs.MemPool.GetFirst(int(ehs.Config.HotStuff.BatchSize))
	ehs.Log.WithField("tx_count", len(txs)).Debug("Collected transactions for proposal")
	if len(txs) != 0 {
		ehs.BatchTimeChan.Stop()
	} else {
		ehs.Log.Debug("No transactions available, creating empty block")
		// Produce empty blocks when there is no tx
		// return
	}
	// create node
	proposal := ehs.createProposal(txs)
	ehs.Log.WithField("block_hash", hex.EncodeToString(proposal.Hash)).Debug("Proposal created")

	// create a new prepare msg
	msg := ehs.Msg(pb.MsgType_PREPARE, proposal, nil)
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)

	// Send to self
	ehs.MsgByteEntrance <- msgByte

	// broadcast
	ehs.Log.Debug("Broadcasting proposal")
	err = ehs.Broadcast(msg)
	if err != nil {
		ehs.Log.WithError(err).Warn("Failed to broadcast proposal")
	}
}

// expectBlock looks for a block with the given Hash, or waits for the next proposal to arrive
// ehs.lock must be locked when calling this function
func (ehs *EventDrivenHotStuffImpl) expectBlock(hash []byte) (*pb.WhirlyBlock, error) {
	block, err := ehs.BlockStorage.Get(hash)
	if err == nil {
		return block, nil
	}
	ehs.waitProposal.Wait()
	return ehs.BlockStorage.Get(hash)
}

// createProposal create a new proposal
func (ehs *EventDrivenHotStuffImpl) createProposal(txs []types.RawTransaction) *pb.WhirlyBlock {
	// create a new block
	ehs.lock.Lock()
	block := ehs.CreateLeaf(ehs.bLeaf.Hash, txs, ehs.qcHigh)
	ehs.lock.Unlock()

	// store the block
	err := ehs.BlockStorage.Put(block)
	if err != nil {
		ehs.Log.WithError(err).WithField("block_hash", hex.EncodeToString(block.Hash)).Error("Failed to store new block")
	}
	return block
}

func (ehs *EventDrivenHotStuffImpl) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (ehs *EventDrivenHotStuffImpl) GetConsensusType() string {
	return "eventdrivenhotstuff"
}
