package whirly

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/niclabs/tcrsa"
	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/config"
	whirlyUtilities "github.com/zzz136454872/upgradeable-consensus/consensus/whirly"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

type Event uint8

const (
	QCFinish Event = iota
	ReceiveProposal
	ReceiveNewView
)

type Whirly interface {
	Update(qc *pb.QuorumCert)
	OnCommit(block *pb.WhirlyBlock)
	OnReceiveProposal(newBlock *pb.WhirlyBlock, qc *pb.QuorumCert)
	OnReceiveVote(msg *pb.WhirlyMsg)
	OnPropose()
}

type WhirlyImpl struct {
	whirlyUtilities.WhirlyUtilitiesImpl
	lock          sync.Mutex
	voteLock      sync.Mutex
	curYesVote    []*tcrsa.SigShare
	curNoVote     []*pb.QuorumCert
	curNewView    []*pb.QuorumCert
	proposeView   uint64
	pacemaker     Pacemaker
	bLock         *pb.WhirlyBlock
	bExec         *pb.WhirlyBlock
	lockQC        *pb.QuorumCert
	vHeight       uint64
	waitProposal  *sync.Cond
	cancel        context.CancelFunc
	eventChannels []chan Event
}

type BlockInfo struct {
	view uint64
	Hash []byte
}

func NewWhirly(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *WhirlyImpl {
	log.WithField("consensus id", cid).Debug("[WHIRLY] starting")
	log.WithField("consensus id", cid).Trace("[WHIRLY] Generate genesis block")
	genesisBlock := whirlyUtilities.GenerateGenesisBlock()
	ctx, cancel := context.WithCancel(context.Background())
	whi := &WhirlyImpl{
		bLock:         genesisBlock,
		bExec:         genesisBlock,
		lockQC:        nil,
		vHeight:       genesisBlock.Height,
		cancel:        cancel,
		eventChannels: make([]chan Event, 0),
	}

	whi.Init(id, cid, cfg, exec, p2pAdaptor, log)
	err := whi.BlockStorage.Put(genesisBlock)
	if err != nil {
		whi.Log.Fatal("Store genesis block failed!")
	}

	// make view number equal to 0 to create genesis block QC
	whi.View = whirlyUtilities.NewView(0, 1)
	whi.lockQC = whi.QC(whi.View.ViewNum, nil, genesisBlock.Hash)

	// view number add 1
	whi.View.ViewNum++
	whi.waitProposal = sync.NewCond(&whi.lock)
	whi.Log.WithField("replicaID", id).Debug("[WHIRLY] Init block storage.")
	whi.Log.WithField("replicaID", id).Debug("[WHIRLY] Init command cache.")

	// init timer and stop it
	whi.TimeChan = utils.NewTimer(time.Duration(whi.Config.Whirly.Timeout) * time.Second)
	whi.TimeChan.Init()

	whi.CleanVote()
	whi.proposeView = 0
	whi.pacemaker = NewPacemaker(whi, log.WithField("cid", cid))

	go whi.receiveMsg(ctx)
	go whi.pacemaker.Run(ctx)

	return whi
}

func (whi *WhirlyImpl) CleanVote() {
	whi.curYesVote = make([]*tcrsa.SigShare, 0)
	whi.curNoVote = make([]*pb.QuorumCert, 0)
	whi.curNewView = make([]*pb.QuorumCert, 0)
}

func (whi *WhirlyImpl) emitEvent(event Event) {
	for _, c := range whi.eventChannels {
		c <- event
	}
}

func (whi *WhirlyImpl) GetHeight() uint64 {
	return whi.bLock.Height
}

func (whi *WhirlyImpl) GetVHeight() uint64 {
	return whi.vHeight
}

func (whi *WhirlyImpl) GetLock() *pb.WhirlyBlock {
	whi.lock.Lock()
	defer whi.lock.Unlock()
	return whi.bLock
}

func (whi *WhirlyImpl) SetLock(b *pb.WhirlyBlock) {
	whi.lock.Lock()
	defer whi.lock.Unlock()
	whi.bLock = b
}

func (whi *WhirlyImpl) GetLockQC() *pb.QuorumCert {
	whi.lock.Lock()
	defer whi.lock.Unlock()
	return whi.lockQC
}

func (whi *WhirlyImpl) GetEvents() chan Event {
	c := make(chan Event)
	whi.eventChannels = append(whi.eventChannels, c)
	return c
}

func (whi *WhirlyImpl) Stop() {
	whi.cancel()
	whi.BlockStorage.Close()
	close(whi.MsgByteEntrance)
	close(whi.RequestEntrance)
	_ = os.RemoveAll("dbfile/node" + strconv.Itoa(int(whi.ID)))
}

func (whi *WhirlyImpl) receiveMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgByte, ok := <-whi.MsgByteEntrance:
			if !ok {
				return // closed
			}
			msg, err := whi.DecodeMsgByte(msgByte)
			if err != nil {
				whi.Log.WithError(err).Warn("decode message failed")
				continue
			}
			go whi.handleMsg(msg)
		case request, ok := <-whi.RequestEntrance:
			if !ok {
				return // closed
			}
			go whi.handleMsg(&pb.WhirlyMsg{Payload: &pb.WhirlyMsg_Request{Request: request}})
		}
	}
}

func (whi *WhirlyImpl) handleMsg(msg *pb.WhirlyMsg) {
	switch msg.Payload.(type) {
	case *pb.WhirlyMsg_Request:
		request := msg.GetRequest()
		// whi.Log.WithField("content", request.String()).Debug("[WHIRLY] Get request msg.")
		// put the cmd into the cmdset
		whi.MemPool.Add(types.RawTransaction(request.Tx))
		// send the request to the leader, if the replica is not the leader
		if whi.ID != whi.GetLeader(int64(whi.View.ViewNum)) {
			_ = whi.Unicast(whi.GetNetworkInfo()[whi.GetLeader(int64(whi.View.ViewNum))], msg)
			return
		}
	case *pb.WhirlyMsg_WhirlyProposal:
		proposalMsg := msg.GetWhirlyProposal()
		whi.OnReceiveProposal(proposalMsg.Block, proposalMsg.HighQC)
	case *pb.WhirlyMsg_WhirlyVote:
		whi.OnReceiveVote(msg)
	case *pb.WhirlyMsg_WhirlyNewView:
		newViewMsg := msg.GetWhirlyNewView()
		// whi.Log.Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] recevice newview")
		if whi.GetLeader(int64(newViewMsg.ViewNum)) != whi.ID || newViewMsg.ViewNum < whi.View.ViewNum {
			whi.Log.Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] recevice error newview!")
			return
		}

		// wait for 2f+1 votes
		whi.voteLock.Lock()
		whi.curNewView = append(whi.curNewView, newViewMsg.LockQC)
		whi.voteLock.Unlock()
		whi.pacemaker.UpdateLockQC(newViewMsg.LockQC)
		if len(whi.curNewView) == 2*whi.Config.F+1 {
			if newViewMsg.ViewNum > whi.View.ViewNum {
				whi.View.ViewNum = newViewMsg.ViewNum
				whi.Log.Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] advanceView by newview success!")
			}
			whi.pacemaker.OnReceiverNewView(whi.lockQC)
		}
	default:
		whi.Log.Warn("Receive unsupported msg")
	}
}

// Update update blocks before block
func (whi *WhirlyImpl) Update(qc *pb.QuorumCert) {
	// block1 = b'', block2 = b', block3 = b
	block1, err := whi.BlockStorage.BlockOf(qc)
	if err != nil && err != leveldb.ErrNotFound {
		whi.Log.Fatal(err)
	}
	if block1 == nil || block1.Committed {
		return
	}

	whi.lock.Lock()
	defer whi.lock.Unlock()

	whi.Log.WithField(
		"blockHash", hex.EncodeToString(block1.Hash)).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] [WHIRLY] LOCK.")
	// pre-commit block1
	whi.pacemaker.UpdateLockQC(qc)

	block2, err := whi.BlockStorage.ParentOf(block1)
	if err != nil && err != leveldb.ErrNotFound {
		whi.Log.Fatal(err)
	}
	if block2 == nil || block2.Committed {
		return
	}

	if block2.Height+1 == block1.Height {
		whi.Log.WithFields(logrus.Fields{
			"blockHash":   hex.EncodeToString(block2.Hash),
			"blockHeight": block2.Height,
		}).Info("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] [WHIRLY] COMMIT.")
		// whi.Log.WithField(
		// 	"blockHash", hex.EncodeToString(block2.Hash)).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] [WHIRLY] COMMIT.")
		whi.OnCommit(block2)
		whi.bExec = block2
	}
}

func (whi *WhirlyImpl) OnCommit(block *pb.WhirlyBlock) {
	if whi.bExec.Height < block.Height {
		if parent, _ := whi.BlockStorage.ParentOf(block); parent != nil {
			whi.OnCommit(parent)
		}
		// go func() {
		err := whi.BlockStorage.UpdateState(block)
		if err != nil {
			whi.Log.WithField("error", err.Error()).Fatal("Update block state failed")
		}
		// }()
		// whi.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] [WHIRLY] EXEC.")
		go whi.ProcessProposal(block, []byte{})
	}
}

func (whi *WhirlyImpl) verfiyQc(qc *pb.QuorumCert) bool {
	return true
}

func (whi *WhirlyImpl) OnReceiveProposal(newBlock *pb.WhirlyBlock, qc *pb.QuorumCert) {
	whi.Log.WithFields(logrus.Fields{
		"blockHeight": newBlock.Height,
		"qcView":      qc.ViewNum,
	}).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveProposal.")
	// whi.Log.WithField("blockHash", hex.EncodeToString(newBlock.Hash)).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveProposal.")

	// store the block
	err := whi.BlockStorage.Put(newBlock)
	if err != nil {
		whi.Log.WithError(err).Info("Store the new block failed.")
	}

	// verfiy proposal
	v := newBlock.Height
	if v <= whi.lockQC.ViewNum {
		// whi.Log.Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] receive old proposal.")
		return
	}

	if !whi.verfiyQc(qc) {
		whi.Log.Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] proposal qc is wrong.")
		return
	}

	if !bytes.Equal(qc.BlockHash, newBlock.ParentHash) {
		whi.Log.WithFields(logrus.Fields{
			"qcHeight":    qc.ViewNum,
			"blockHeight": newBlock.Height,
			"qcHash":      hex.EncodeToString(qc.BlockHash),
			"blockHash":   hex.EncodeToString(newBlock.ParentHash),
		}).Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveProposal: proposal block and qc is incongruous.")
		// whi.Log.Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] proposal block and qc is incongruous.")
		return
	}

	// verfiy Height
	if v <= whi.vHeight {
		// info log rathee than warn
		whi.Log.WithFields(logrus.Fields{
			"blockHeight": newBlock.Height,
			"vHeight":     whi.vHeight,
		}).Info("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveProposal: WhirlyBlock height less than vHeight.")
		return
	}

	// vote for proposal
	var voteFlag bool
	var votePartSig *tcrsa.SigShare
	var voteQc *pb.QuorumCert
	if qc.ViewNum >= whi.lockQC.ViewNum {
		voteFlag = true
		data := BlockInfo{
			view: newBlock.Height,
			Hash: newBlock.Hash,
		}
		marshal, _ := json.Marshal(data)
		documentHash, _ := crypto.CreateDocumentHash(marshal, whi.Config.Keys.PublicKey)
		votePartSig, err = crypto.TSign(documentHash, whi.Config.Keys.PrivateKey, whi.Config.Keys.PublicKey)
		if err != nil {
			whi.Log.WithField("error", err.Error()).Error("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveProposal: create partSig failed!")
			return
		}
		voteQc = nil
	} else {
		voteFlag = false
		voteQc = whi.lockQC
		votePartSig = nil
	}

	whi.Log.Debug("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "]  OnReceiveProposal: Accepted block.")
	// update vHeight
	whi.vHeight = newBlock.Height
	whi.MemPool.MarkProposed(types.RawTxArrayFromBytes(newBlock.Txs))

	whi.waitProposal.Broadcast()

	whi.emitEvent(ReceiveProposal)
	go whi.Update(qc)

	partSigBytes, _ := json.Marshal(votePartSig)
	voteMsg := whi.VoteMsg(newBlock.Height, newBlock.Hash, voteFlag, voteQc, partSigBytes, nil, 0)

	whi.pacemaker.AdvanceView(newBlock.Height)

	if whi.GetLeader(int64(newBlock.Height)+1) == whi.ID {
		// vote self
		whi.OnReceiveVote(voteMsg)
	} else {
		// send vote to the leader
		_ = whi.Unicast(whi.GetNetworkInfo()[whi.GetLeader(int64(newBlock.Height)+1)], voteMsg)
	}
}

func (whi *WhirlyImpl) OnReceiveVote(msg *pb.WhirlyMsg) {
	whirlyVoteMsg := msg.GetWhirlyVote()

	if whi.GetLeader(int64(whirlyVoteMsg.BlockView)+1) != whi.ID {
		whi.Log.WithFields(logrus.Fields{
			"senderId":  whirlyVoteMsg.SenderId,
			"getleader": whi.GetLeader(int64(whirlyVoteMsg.BlockView) + 1),
			"blockView": whirlyVoteMsg.BlockView,
			"whi.ID":    whi.ID,
			"Flag":      whirlyVoteMsg.Flag,
		}).Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveVote with error block view.")
		// whi.Log.Warn("OnReceiveVote with error block view")
		return
	}

	// Ignore messages from old views
	if whirlyVoteMsg.BlockView < whi.proposeView || whirlyVoteMsg.BlockView <= whi.lockQC.ViewNum {
		// whi.Log.Warn("ignore vote from old views")
		return
	}

	whi.Log.WithFields(logrus.Fields{
		"senderId":     whirlyVoteMsg.SenderId,
		"blockView":    whirlyVoteMsg.BlockView,
		"flag":         whirlyVoteMsg.Flag,
		"len(YesVote)": len(whi.curYesVote),
	}).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveVote.")
	// whi.Log.Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveVote.")

	var documentHash []byte
	if whirlyVoteMsg.Flag {
		partSig := &tcrsa.SigShare{}
		err := json.Unmarshal(whirlyVoteMsg.PartialSig, partSig)
		if err != nil {
			whi.Log.WithField("error", err.Error()).Error("Unmarshal partSig failed.")
			return
		}

		// verify partSig
		data := BlockInfo{
			view: whirlyVoteMsg.BlockView,
			Hash: whirlyVoteMsg.BlockHash,
		}
		marshal, _ := json.Marshal(data)
		documentHash, _ = crypto.CreateDocumentHash(marshal, whi.Config.Keys.PublicKey)
		err = crypto.VerifyPartSig(partSig, documentHash, whi.Config.Keys.PublicKey)
		if err != nil {
			whi.Log.WithFields(logrus.Fields{
				"error":     err.Error(),
				"blockView": whirlyVoteMsg.BlockView,
				"blockHash": hex.EncodeToString(whirlyVoteMsg.BlockHash),
				"view":      whi.View.ViewNum,
			}).Warn("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnReceiveVote: signature not verified!")
			return
		}
		whi.voteLock.Lock()
		if whirlyVoteMsg.BlockView >= whi.proposeView && whirlyVoteMsg.BlockView > whi.lockQC.ViewNum {
			whi.curYesVote = append(whi.curYesVote, partSig)
		}
		whi.voteLock.Unlock()

	} else {
		whi.voteLock.Lock()
		whi.curNoVote = append(whi.curNoVote, whirlyVoteMsg.Qc)
		whi.voteLock.Unlock()
		whi.lock.Lock()
		whi.pacemaker.UpdateLockQC(whirlyVoteMsg.Qc)
		whi.lock.Unlock()
	}

	if len(whi.curYesVote)+len(whi.curNoVote) == 2*whi.Config.F+1 {
		if len(whi.curYesVote) == 2*whi.Config.F+1 {
			// create full signature
			signature, err := crypto.CreateFullSignature(documentHash, whi.curYesVote,
				whi.Config.Keys.PublicKey)
			if err != nil {
				whi.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] create full signature error!")
			}
			// create a QC
			qc := whi.QC(whirlyVoteMsg.BlockView, signature, whirlyVoteMsg.BlockHash)
			whi.Log.WithFields(logrus.Fields{
				"qcView": qc.ViewNum,
			}).Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] create full signature!")
			// update qcHigh
			whi.lock.Lock()
			whi.pacemaker.UpdateLockQC(qc)
			whi.lock.Unlock()
		} else {
			whi.Log.WithFields(logrus.Fields{
				"msgView":     whirlyVoteMsg.BlockView,
				"len(NoVote)": len(whi.curNoVote),
			}).Error("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] have NoVote!")
		}

		whi.pacemaker.AdvanceView(whirlyVoteMsg.BlockView)
		go whi.OnPropose()
	}
}

func (whi *WhirlyImpl) OnPropose() {
	// =====================================
	// == Testing the Byzantine situation ==
	// =====================================
	// if whi.ID == 3 {
	// 	return
	// }

	if whi.GetLeader(int64(whi.View.ViewNum)) != whi.ID || whi.proposeView >= whi.View.ViewNum {
		whi.Log.WithFields(logrus.Fields{
			"nowView":     whi.View.ViewNum,
			"leader":      whi.GetLeader(int64(whi.View.ViewNum)),
			"proposeView": whi.proposeView,
		}).Error("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "] OnPropose: not allow!")
		return
	}
	whi.Log.Trace("[replica_" + strconv.Itoa(int(whi.ID)) + "] [view_" + strconv.Itoa(int(whi.View.ViewNum)) + "]  OnPropose")
	txs := whi.MemPool.GetFirst(int(whi.Config.Whirly.BatchSize))
	if len(txs) != 0 {
		_ = 1
	} else {
		_ = 1
		// Produce empty blocks when there is no tx
		// return
	}

	whi.proposeView = whi.View.ViewNum
	whi.voteLock.Lock()
	whi.CleanVote()
	whi.voteLock.Unlock()

	// create node
	proposal := whi.createProposal(txs)
	// create a new prepare msg
	msg := whi.ProposalMsg(proposal, whi.lockQC, nil, 0)

	// if whi.ID == 3 {
	// 	_ = whi.Unicast(whi.GetNetworkInfo()[0], msg)
	// 	_ = whi.Unicast(whi.GetNetworkInfo()[1], msg)
	// 	return
	// }

	// the old leader should vote too
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)

	// the following line may panic
	defer func() {
		if err := recover(); err != nil {
			whi.Log.WithField("error", err).Warn("[WHIRLY] insert into channel failed")
		}
	}()
	whi.MsgByteEntrance <- msgByte
	// broadcast
	err = whi.Broadcast(msg)
	if err != nil {
		whi.Log.WithField("error", err.Error()).Warn("Broadcast proposal failed.")
	}
}

// expectBlock looks for a block with the given Hash, or waits for the next proposal to arrive
// whi.lock must be locked when calling this function
func (whi *WhirlyImpl) expectBlock(hash []byte) (*pb.WhirlyBlock, error) {
	block, err := whi.BlockStorage.Get(hash)
	if err == nil {
		return block, nil
	} else {
		// whi.Log.WithField("error", err.Error()).Error("[replica_" + strconv.Itoa(int(whi.ID)) + "] expect block failed.")
		_ = 1
	}
	whi.waitProposal.Wait()
	return whi.BlockStorage.Get(hash)
}

// createProposal create a new proposal
func (whi *WhirlyImpl) createProposal(txs []types.RawTransaction) *pb.WhirlyBlock {
	// create a new block
	whi.lock.Lock()
	// do not use qc fields for blocks
	block := whi.CreateLeaf(whi.lockQC.BlockHash, whi.View.ViewNum, txs, nil)
	whi.lock.Unlock()
	// store the block
	err := whi.BlockStorage.Put(block)
	if err != nil {
		whi.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Error("Store new block failed!")
	}
	return block
}

func (whi *WhirlyImpl) VerifyBlock(block []byte, proof []byte) bool {
	return true
}
