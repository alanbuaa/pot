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
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	whirlyUtilities "github.com/zzz136454872/upgradeable-consensus/consensus/whirly"
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
	log = log.WithField("module", "WHIRLY").WithField("c_id", cid)
	log.Info("Initializing Whirly consensus")
	log.WithField("consensus id", cid).Trace("Generate genesis block")
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

	whi.InitForLocalTest(id, cid, cfg, exec, p2pAdaptor, log)
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
	whi.Log.WithField("replica_id", id).Debug("Block storage initialized")
	whi.Log.WithField("replica_id", id).Debug("Command cache initialized")

	// init timer and stop it
	whi.TimeChan = utils.NewTimer(time.Duration(whi.Config.Whirly.Timeout) * time.Second)
	whi.TimeChan.Init()

	whi.CleanVote()
	whi.proposeView = 0
	whi.pacemaker = NewPacemaker(whi, log.WithField("cid", cid))

	log.Debug("Starting message receiver and pacemaker goroutines")
	go whi.receiveMsg(ctx)
	go whi.pacemaker.Run(ctx)

	log.Info("Whirly consensus initialized successfully")
	return whi
}

func (whi *WhirlyImpl) UpdateExternalStatus(status model.ExternalStatus) {
	return
}

func (whi *WhirlyImpl) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	return
}

func (whi *WhirlyImpl) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	return
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
	whi.Log.Info("Stopping Whirly consensus")
	whi.Log.Debug("Canceling context")
	whi.cancel()
	whi.Log.Debug("Closing block storage")
	whi.BlockStorage.Close()
	close(whi.MsgByteEntrance)
	close(whi.RequestEntrance)
	dbPath := "dbfile/node" + strconv.Itoa(int(whi.ID))
	whi.Log.WithField("path", dbPath).Debug("Cleaning up database files")
	_ = os.RemoveAll(dbPath)
	whi.Log.Info("Whirly consensus stopped successfully")
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
		if whi.GetLeader(int64(newViewMsg.ViewNum)) != whi.ID || newViewMsg.ViewNum < whi.View.ViewNum {
			whi.Log.WithFields(logrus.Fields{
				"replica_id":      whi.ID,
				"current_view":    whi.View.ViewNum,
				"msg_view":        newViewMsg.ViewNum,
				"expected_leader": whi.GetLeader(int64(newViewMsg.ViewNum)),
			}).Warn("Received invalid new view message")
			return
		}

		// wait for 2f+1 votes
		whi.voteLock.Lock()
		whi.curNewView = append(whi.curNewView, newViewMsg.LockQC)
		whi.voteLock.Unlock()
		whi.pacemaker.UpdateLockQC(newViewMsg.LockQC)
		if len(whi.curNewView) == 2*whi.Config.Fault+1 {
			if newViewMsg.ViewNum > whi.View.ViewNum {
				whi.View.ViewNum = newViewMsg.ViewNum
				whi.Log.WithFields(logrus.Fields{
					"replica_id": whi.ID,
					"new_view":   whi.View.ViewNum,
				}).Debug("Advanced to new view by quorum")
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

	whi.Log.WithFields(logrus.Fields{
		"replica_id":   whi.ID,
		"view":         whi.View.ViewNum,
		"block_hash":   hex.EncodeToString(block1.Hash),
		"block_height": block1.Height,
	}).Trace("Locking block (pre-commit)")
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
			"replica_id":   whi.ID,
			"view":         whi.View.ViewNum,
			"block_hash":   hex.EncodeToString(block2.Hash),
			"block_height": block2.Height,
		}).Info("Committing block")
		whi.OnCommit(block2)
		whi.bExec = block2
	}
}

func (whi *WhirlyImpl) OnCommit(block *pb.WhirlyBlock) {
	if whi.bExec.Height < block.Height {
		if parent, _ := whi.BlockStorage.ParentOf(block); parent != nil {
			whi.OnCommit(parent)
		}
		err := whi.BlockStorage.UpdateState(block)
		if err != nil {
			whi.Log.WithFields(logrus.Fields{
				"block_hash":   hex.EncodeToString(block.Hash),
				"block_height": block.Height,
			}).WithError(err).Fatal("Failed to update block state")
		}
		whi.Log.WithFields(logrus.Fields{
			"block_hash":   hex.EncodeToString(block.Hash),
			"block_height": block.Height,
		}).Trace("Executing committed block")
		go whi.ProcessProposal(block, []byte{})
	}
}

func (whi *WhirlyImpl) verfiyQc(qc *pb.QuorumCert) bool {
	return true
}

func (whi *WhirlyImpl) OnReceiveProposal(newBlock *pb.WhirlyBlock, qc *pb.QuorumCert) {
	whi.Log.WithFields(logrus.Fields{
		"replica_id":   whi.ID,
		"view":         whi.View.ViewNum,
		"block_height": newBlock.Height,
		"block_hash":   hex.EncodeToString(newBlock.Hash),
		"qc_view":      qc.ViewNum,
	}).Trace("Received proposal")

	// store the block
	err := whi.BlockStorage.Put(newBlock)
	if err != nil {
		whi.Log.WithError(err).Info("Store the new block failed.")
	}

	// verfiy proposal
	v := newBlock.Height
	if v <= whi.lockQC.ViewNum {
		// whi.Log.WithFields(logrus.Fields{"replica": whi.ID, "view": whi.View.ViewNum}).Warn("Receive old proposal")
		return
	}

	if !whi.verfiyQc(qc) {
		whi.Log.WithFields(logrus.Fields{
			"replica_id": whi.ID,
			"view":       whi.View.ViewNum,
			"qc_view":    qc.ViewNum,
		}).Warn("Proposal QC verification failed")
		return
	}

	if !bytes.Equal(qc.BlockHash, newBlock.ParentHash) {
		whi.Log.WithFields(logrus.Fields{
			"replica_id":        whi.ID,
			"view":              whi.View.ViewNum,
			"qc_view":           qc.ViewNum,
			"block_height":      newBlock.Height,
			"qc_block_hash":     hex.EncodeToString(qc.BlockHash),
			"block_parent_hash": hex.EncodeToString(newBlock.ParentHash),
		}).Warn("Proposal block and QC mismatch")
		return
	}

	// verfiy ExecHeight
	if v <= whi.vHeight {
		whi.Log.WithFields(logrus.Fields{
			"replica_id":   whi.ID,
			"view":         whi.View.ViewNum,
			"block_height": newBlock.Height,
			"v_height":     whi.vHeight,
		}).Debug("Block height not greater than vHeight, ignoring")
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
			whi.Log.WithFields(logrus.Fields{
				"replica_id": whi.ID,
				"view":       whi.View.ViewNum,
			}).WithError(err).Error("Failed to create partial signature for vote")
			return
		}
		voteQc = nil
	} else {
		voteFlag = false
		voteQc = whi.lockQC
		votePartSig = nil
	}

	whi.Log.WithFields(logrus.Fields{
		"replica_id":   whi.ID,
		"view":         whi.View.ViewNum,
		"block_height": newBlock.Height,
		"vote_flag":    voteFlag,
	}).Debug("Accepted proposal, voting")
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
			"replica_id":      whi.ID,
			"view":            whi.View.ViewNum,
			"sender_id":       whirlyVoteMsg.SenderId,
			"expected_leader": whi.GetLeader(int64(whirlyVoteMsg.BlockView) + 1),
			"block_view":      whirlyVoteMsg.BlockView,
			"vote_flag":       whirlyVoteMsg.Flag,
		}).Warn("Received vote for wrong view or non-leader")
		return
	}

	// Ignore messages from old views
	if whirlyVoteMsg.BlockView < whi.proposeView || whirlyVoteMsg.BlockView <= whi.lockQC.ViewNum {
		return
	}

	whi.Log.WithFields(logrus.Fields{
		"replica_id": whi.ID,
		"view":       whi.View.ViewNum,
		"sender_id":  whirlyVoteMsg.SenderId,
		"block_view": whirlyVoteMsg.BlockView,
		"vote_flag":  whirlyVoteMsg.Flag,
		"yes_votes":  len(whi.curYesVote),
		"no_votes":   len(whi.curNoVote),
	}).Trace("Received vote")

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
				"replica_id": whi.ID,
				"view":       whi.View.ViewNum,
				"sender_id":  whirlyVoteMsg.SenderId,
				"block_view": whirlyVoteMsg.BlockView,
				"block_hash": hex.EncodeToString(whirlyVoteMsg.BlockHash),
			}).WithError(err).Warn("Partial signature verification failed")
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

	if len(whi.curYesVote)+len(whi.curNoVote) == 2*whi.Config.Fault+1 {
		if len(whi.curYesVote) == 2*whi.Config.Fault+1 {
			// create full signature
			signature, err := crypto.CreateFullSignature(documentHash, whi.curYesVote,
				whi.Config.Keys.PublicKey)
			if err != nil {
				whi.Log.WithFields(logrus.Fields{
					"replica_id": whi.ID,
					"view":       whi.View.ViewNum,
					"block_view": whirlyVoteMsg.BlockView,
				}).WithError(err).Error("Failed to create full signature from partial signatures")
			}
			// create a QC
			qc := whi.QC(whirlyVoteMsg.BlockView, signature, whirlyVoteMsg.BlockHash)
			whi.Log.WithFields(logrus.Fields{
				"replica_id": whi.ID,
				"view":       whi.View.ViewNum,
				"qc_view":    qc.ViewNum,
				"yes_votes":  len(whi.curYesVote),
			}).Debug("Created QC from quorum of votes")
			// update qcHigh
			whi.lock.Lock()
			whi.pacemaker.UpdateLockQC(qc)
			whi.lock.Unlock()
		} else {
			whi.Log.WithFields(logrus.Fields{
				"replica_id": whi.ID,
				"view":       whi.View.ViewNum,
				"block_view": whirlyVoteMsg.BlockView,
				"no_votes":   len(whi.curNoVote),
				"yes_votes":  len(whi.curYesVote),
			}).Warn("Received no-votes, cannot form QC")
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
			"replica_id":   whi.ID,
			"current_view": whi.View.ViewNum,
			"leader":       whi.GetLeader(int64(whi.View.ViewNum)),
			"propose_view": whi.proposeView,
		}).Debug("Cannot propose: not leader or already proposed in this view")
		return
	}
	whi.Log.WithFields(logrus.Fields{
		"replica_id": whi.ID,
		"view":       whi.View.ViewNum,
	}).Trace("Starting proposal process")
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
			whi.Log.WithFields(logrus.Fields{
				"replica_id": whi.ID,
				"view":       whi.View.ViewNum,
			}).WithField("error", err).Warn("Failed to insert message into channel")
		}
	}()
	whi.MsgByteEntrance <- msgByte
	// broadcast
	err = whi.Broadcast(msg)
	if err != nil {
		whi.Log.WithFields(logrus.Fields{
			"replica_id": whi.ID,
			"view":       whi.View.ViewNum,
		}).WithError(err).Warn("Failed to broadcast proposal")
	}
}

// expectBlock looks for a block with the given Hash, or waits for the next proposal to arrive
// whi.lock must be locked when calling this function
func (whi *WhirlyImpl) expectBlock(hash []byte) (*pb.WhirlyBlock, error) {
	block, err := whi.BlockStorage.Get(hash)
	if err == nil {
		return block, nil
	} else {
		// whi.Log.WithFields(logrus.Fields{"replica": whi.ID, "error": err.Error()}).Error("Expect block failed")
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
	block := whi.CreateLeaf(whi.lockQC.BlockHash, whi.View.ViewNum, txs, nil, nil)
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

func (e *WhirlyImpl) GetConsensusType() string {
	return "whirly"
}
