package simpleWhirly

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"time"

	bc_api "blockchain-crypto/blockchain_api"

	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	whirlyUtilities "github.com/zzz136454872/upgradeable-consensus/consensus/whirly"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

type Event uint8

type SimpleWhirly interface {
	Update(block *pb.WhirlyBlock)
	OnCommit(block *pb.WhirlyBlock)
	OnReceiveProposal(newBlock *pb.WhirlyBlock, swProof *pb.SimpleWhirlyProof)
	OnReceiveVote(msg *pb.WhirlyMsg)
	OnPropose()
	GetPoTByteEntrance() chan<- []byte
}

type SimpleWhirlyImpl struct {
	whirlyUtilities.WhirlyUtilitiesImpl
	epoch        int64
	leader       map[int64]string
	leaderLock   sync.Mutex
	lock         sync.Mutex
	voteLock     sync.Mutex
	proposalLock sync.Mutex
	curYesVote   []*pb.WhirlyVote
	curNoVote    []*pb.SimpleWhirlyProof
	proposeView  uint64
	bLock        *pb.WhirlyBlock
	bExec        *pb.WhirlyBlock
	lockProof    *pb.SimpleWhirlyProof
	vHeight      uint64
	waitProposal *sync.Cond
	cancel       context.CancelFunc

	// New Epoch Echo
	newEpoch NewEpochMechanism

	// Latest block synchronization
	latestBlockReq LatestBlockRequestMechanism

	inCommittee    bool
	Committee      []string
	stopEntrance   chan string
	updateEntrance chan string
	incentive      whirlyUtilities.Incentive

	// Ping
	readyNodes     []string
	sendPingCancel context.CancelFunc
}

func NewSimpleWhirly(
	id int64,
	cid int64,
	epoch int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
	publicAddress string,
	stopEntrance chan string,
	updateEntrance chan string,
) *SimpleWhirlyImpl {
	log.WithFields(logrus.Fields{
		"node_id":        id,
		"consensus_id":   cid,
		"epoch":          epoch,
		"public_address": publicAddress,
	}).Info("Initializing Simple Whirly consensus")
	log.WithField("consensus_id", cid).Trace("Generating genesis block")
	genesisBlock := whirlyUtilities.GenerateGenesisBlock()
	ctx, cancel := context.WithCancel(context.Background())
	sw := &SimpleWhirlyImpl{
		bLock:     genesisBlock,
		bExec:     genesisBlock,
		lockProof: nil,
		vHeight:   genesisBlock.Height,
		// pendingUpdate: make(chan *pb.SimpleWhirlyProof, 1),
		cancel:         cancel,
		stopEntrance:   stopEntrance,
		updateEntrance: updateEntrance,
	}

	sw.Init(id, cid, cfg, exec, p2pAdaptor, log, publicAddress)
	err := sw.BlockStorage.Put(genesisBlock)
	if err != nil {
		sw.Log.Fatal("Store genesis block failed!")
	}

	// make view number equal to 0 to create genesis block proof
	sw.View = whirlyUtilities.NewView(0, 1)
	sw.lockProof = sw.Proof(sw.View.ViewNum, nil, genesisBlock.Hash)

	// view number add 1
	sw.View.ViewNum++
	sw.waitProposal = sync.NewCond(&sw.lock)
	sw.Log.WithField("replica_id", id).Debug("Block storage initialized")
	sw.Log.WithField("replica_id", id).Debug("Command cache initialized")

	sw.CleanVote()
	sw.inCommittee = false
	sw.epoch = 1
	sw.proposeView = 0
	// ensure p2pAdaptor of SimpleWhirly is same as PoT

	sw.leader = make(map[int64]string)
	// sw.PoTByteEntrance = make(chan []byte, 10)

	// go sw.updateAsync(ctx)
	go sw.receiveMsg(ctx)

	// if sw.GetP2pAdaptorType() == "p2p" {
	// 	if sw.ID == sw.leader[sw.epoch] {
	// 		// TODO: ensure all nodes is ready before OnPropose
	// 		time.Sleep(2 * time.Second)
	// 		go sw.OnPropose()
	// 	}
	// } else {
	// 	sw.readyNodes = make([]string, 0)
	// 	sendCtx, sendCancel := context.WithCancel(context.Background())
	// 	sw.sendPingCancel = sendCancel
	// 	go sw.sendPingMsg(sendCtx)
	// }

	log.Info("Simple Whirly consensus initialized successfully")
	return sw
}

func NewSimpleWhirlyForLocalTest(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *SimpleWhirlyImpl {
	log = log.WithField("module", "SIMPLE WHIRLY").WithField("c_id", cid)
	log.Info("Initializing Simple Whirly consensus")

	log.Trace("Generate genesis block")
	genesisBlock := whirlyUtilities.GenerateGenesisBlock()
	ctx, cancel := context.WithCancel(context.Background())
	sw := &SimpleWhirlyImpl{
		bLock:     genesisBlock,
		bExec:     genesisBlock,
		lockProof: nil,
		vHeight:   genesisBlock.Height,
		// pendingUpdate: make(chan *pb.SimpleWhirlyProof, 1),
		cancel:      cancel,
		inCommittee: true,
	}

	sw.Weight = 1
	sw.inCommittee = true

	sw.InitForLocalTest(id, cid, cfg, exec, p2pAdaptor, log)
	err := sw.BlockStorage.Put(genesisBlock)
	if err != nil {
		sw.Log.Fatal("Store genesis block failed!")
	}

	// make view number equal to 0 to create genesis block proof
	sw.View = whirlyUtilities.NewView(0, 1)
	sw.lockProof = sw.Proof(sw.View.ViewNum, nil, genesisBlock.Hash)

	// view number add 1
	sw.View.ViewNum++
	sw.waitProposal = sync.NewCond(&sw.lock)
	sw.Log.WithField("replica_id", id).Debug("Block storage initialized")
	sw.Log.WithField("replica_id", id).Debug("Command cache initialized")

	sw.CleanVote()
	sw.epoch = 1
	sw.proposeView = 0
	// ensure p2pAdaptor of SimpleWhirly is same as PoT

	sw.leader = make(map[int64]string)
	// sw.PoTByteEntrance = make(chan []byte, 10)

	// go sw.updateAsync(ctx)
	go sw.receiveMsg(ctx)

	sw.SetLeader(sw.epoch, cfg.Nodes[1].P2PAddress)
	if sw.GetPeerID() == cfg.Nodes[1].P2PAddress {
		// TODO: ensure all nodes is ready before OnPropose
		sw.SetLeader(sw.epoch, sw.GetPeerID())
		go sw.OnPropose()
	} else {

	}
	// sw.testNewLeader()

	log.Info("Simple Whirly consensus initialized successfully (local test mode)")
	return sw
}

func (sw *SimpleWhirlyImpl) SetLeader(epoch int64, leaderID string) {
	sw.leaderLock.Lock()
	// sw.epoch = epoch
	sw.leader[epoch] = leaderID
	sw.leaderLock.Unlock()
}

func (sw *SimpleWhirlyImpl) SetEpoch(epoch int64) {
	if epoch > sw.epoch {
		sw.epoch = epoch
	}
}

func (sw *SimpleWhirlyImpl) SetCryptoElements(cConfig bc_api.CommitteeConfig) {
	sw.CommitteeCryptoElements = cConfig
}

func (sw *SimpleWhirlyImpl) UpdateExternalStatus(status model.ExternalStatus) {
	switch status.Command {
	case "SetLeader":
		sw.SetLeader(status.Epoch, status.Leader)
	case "SetEpoch":
		sw.SetEpoch(status.Epoch)
	case "SetCryptoElements":
		sw.SetCryptoElements(status.CryptoElements)
	default:
		sw.Log.Warnf("UpdateExternalStatus: command type not supported: %s", status.Command)
	}

}

func (sw *SimpleWhirlyImpl) GetLeader(epoch int64) string {
	sw.leaderLock.Lock()
	defer sw.leaderLock.Unlock()

	leaderID, ok := sw.leader[epoch]
	if !ok {
		return ""
	}
	return leaderID
}

func (sw *SimpleWhirlyImpl) CleanVote() {
	sw.curYesVote = make([]*pb.WhirlyVote, 0)
	sw.curNoVote = make([]*pb.SimpleWhirlyProof, 0)
	sw.incentive = *whirlyUtilities.NewIncentive(sw.PublicAddress)
}

func (sw *SimpleWhirlyImpl) GetQuorumSize() int {
	if len(sw.Committee) > 0 {
		f := (len(sw.Committee) - 1) / 3
		return 2*f + 1
	}
	return 2*int(sw.Config.Fault) + 1
}

func (sw *SimpleWhirlyImpl) GetHeight() uint64 {
	return sw.bLock.Height
}

func (sw *SimpleWhirlyImpl) GetVHeight() uint64 {
	return sw.vHeight
}

func (sw *SimpleWhirlyImpl) GetLock() *pb.WhirlyBlock {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	return sw.bLock
}

func (sw *SimpleWhirlyImpl) SetLock(b *pb.WhirlyBlock) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	sw.bLock = b
}

func (sw *SimpleWhirlyImpl) GetLockProof() *pb.SimpleWhirlyProof {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	return sw.lockProof
}

// func (sw *SimpleWhirlyImpl) GetPoTByteEntrance() chan<- []byte {
// 	return sw.PoTByteEntrance
// }

func (sw *SimpleWhirlyImpl) Stop() {
	sw.Log.Info("Stopping Simple Whirly consensus")
	sw.Log.Debug("Canceling context")
	sw.cancel()
	sw.Log.Debug("Closing block storage")
	sw.BlockStorage.Close()
	close(sw.MsgByteEntrance)
	close(sw.RequestEntrance)
	// close(sw.PoTByteEntrance)
	newPeerId1 := strings.Replace(sw.PublicAddress, ".", "", -1)
	newPeerId2 := strings.Replace(newPeerId1, ":", "", -1)
	dbPath := "dbfile/node" + newPeerId2
	sw.Log.WithField("path", dbPath).Debug("Cleaning up database files")
	_ = os.RemoveAll(dbPath)
	_ = os.RemoveAll("store")
	sw.Log.Info("Simple Whirly consensus stopped successfully")
}

func (sw *SimpleWhirlyImpl) receiveMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgByte, ok := <-sw.MsgByteEntrance:
			if !ok {
				return // closed
			}
			msg, err := sw.DecodeMsgByte(msgByte)
			if err != nil {
				sw.Log.WithError(err).Warn("decode message failed")
				continue
			}
			go sw.handleMsg(msg)
		case request, ok := <-sw.RequestEntrance:
			if !ok {
				return // closed
			}
			go sw.handleMsg(&pb.WhirlyMsg{Payload: &pb.WhirlyMsg_Request{Request: request}})
			// case potSignal, ok := <-sw.PoTByteEntrance:
			// 	if !ok {
			// 		return // closed
			// 	}
			// 	//sw.Log.Info("receive pot signal")
			// 	go sw.handlePoTSignal(potSignal)
		}
	}
}

func (sw *SimpleWhirlyImpl) handleMsg(msg *pb.WhirlyMsg) {
	sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum}).Trace("Receiving msg")
	switch msg.Payload.(type) {
	case *pb.WhirlyMsg_Request:
		request := msg.GetRequest()
		//if sw.ID == 0 {
		//	sw.Log.WithField("content", request.String()).Error("[SIMPLE WHIRLY] Get request msg.")
		//}

		// put the cmd into the cmdset
		sw.MemPool.Add(types.RawTransaction(request.Tx))
		// send the request to the leader, if the replica is not the leader
		if sw.PublicAddress != sw.GetLeader(sw.epoch) {
			if sw.GetLeader(sw.epoch) == "" {
				sw.Log.WithFields(logrus.Fields{
					"address": sw.PublicAddress,
					"epoch":   sw.epoch,
				}).Warn("The leader of epoch: ", sw.epoch, " is null")
			}

			_ = sw.Unicast(sw.GetLeader(sw.epoch), msg)
			return
		}
	case *pb.WhirlyMsg_WhirlyProposal:
		proposalMsg := msg.GetWhirlyProposal()
		if int64(proposalMsg.Epoch) < sw.epoch {
			return
		}
		sw.OnReceiveProposal(proposalMsg.Block, proposalMsg.SwProof, proposalMsg.PublicAddress)
	case *pb.WhirlyMsg_WhirlyVote:
		whirlyVoteMsg := msg.GetWhirlyVote()
		if int64(whirlyVoteMsg.Epoch) < sw.epoch {
			return
		}
		sw.OnReceiveVote(whirlyVoteMsg)
	case *pb.WhirlyMsg_NewLeaderNotify:
		newLeaderMsg := msg.GetNewLeaderNotify()
		sw.OnReceiveNewLeaderNotify(newLeaderMsg)
	case *pb.WhirlyMsg_NewLeaderEcho:
		sw.OnReceiveNewLeaderEcho(msg)
	case *pb.WhirlyMsg_LatestBlockRequest:
		lbReq := msg.GetLatestBlockRequest()
		sw.OnReceiveLatestBlockRequest(lbReq)
	case *pb.WhirlyMsg_LatestBlockEcho:
		sw.OnReceiveLatestBlockEcho(msg)
	case *pb.WhirlyMsg_WhirlyPing:
		pingMsg := msg.GetWhirlyPing()
		sw.handlePingMsg(pingMsg)
	default:
		sw.Log.Warn("Receive unsupported msg")
	}
}

// Update update blocks before block
func (sw *SimpleWhirlyImpl) Update(swProof *pb.SimpleWhirlyProof) {
	// block1 = b'', block2 = b', block3 = b

	sw.lock.Lock()
	defer sw.lock.Unlock()

	block1, err := sw.expectBlock(swProof.BlockHash)
	if err != nil && err != leveldb.ErrNotFound {
		sw.Log.Fatal(err)
	}
	if block1 == nil || block1.Committed {
		return
	}
	// sw.lock.Unlock()

	sw.Log.WithFields(logrus.Fields{
		"epoch":     sw.epoch,
		"replica":   sw.ID,
		"view":      sw.View.ViewNum,
		"blockHash": hex.EncodeToString(block1.Hash),
	}).Trace("SIMPLE WHIRLY LOCK")
	// pre-commit block1
	sw.UpdateLockProof(swProof)

	block2, err := sw.BlockStorage.ParentOf(block1)
	if err != nil && err != leveldb.ErrNotFound {
		sw.Log.Fatal(err)
	}
	if block2 == nil || block2.Committed {
		return
	}

	if block2.Height+1 == block1.Height {
		_, _, address := DecodeAddress(sw.PublicAddress)
		if sw.View.ViewNum%2 == 0 && (sw.ID == 0 || address == DaemonNodePublicAddress) {
			sw.Log.WithFields(logrus.Fields{
				"epoch":       sw.epoch,
				"replica":     sw.ID,
				"view":        sw.View.ViewNum,
				"blockHash":   hex.EncodeToString(block2.Hash),
				"blockHeight": block2.Height,
				"myAddress":   sw.PublicAddress,
			}).Info("SIMPLE WHIRLY COMMIT")
		}
		// sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum, "blockHash": hex.EncodeToString(block2.Hash)}).Trace("SIMPLE WHIRLY COMMIT")
		sw.OnCommit(block2)
		sw.bExec = block2
	}
}

func (sw *SimpleWhirlyImpl) OnCommit(block *pb.WhirlyBlock) {
	if sw.bExec.Height < block.Height {
		if parent, _ := sw.BlockStorage.ParentOf(block); parent != nil {
			sw.OnCommit(parent)
		}
		// go func() {
		err := sw.BlockStorage.UpdateState(block)
		if err != nil {
			sw.Log.WithField("error", err.Error()).Fatal("Update block state failed")
		}
		// }()
		//sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum, "blockHash": hex.EncodeToString(block.Hash)}).Trace("SIMPLE WHIRLY EXEC")
		// 只有 DaemonNode 需要提交交易
		_, _, address := DecodeAddress(sw.PublicAddress)
		if sw.ID == 0 || address == DaemonNodePublicAddress {
			go sw.ProcessProposal(block, []byte{})
		}
	}
}

func (sw *SimpleWhirlyImpl) verfiySwProof(swProof *pb.SimpleWhirlyProof) bool {
	if swProof.ViewNum == 0 {
		return true
	}
	if len(swProof.Proof) < sw.GetQuorumSize() {
		sw.Log.WithFields(logrus.Fields{
			"epoch":       sw.epoch,
			"replica":     sw.ID,
			"view":        sw.View.ViewNum,
			"proofHeight": swProof.ViewNum,
			"len(proof)":  len(swProof.Proof),
			"blockHash":   hex.EncodeToString(swProof.BlockHash),
		}).Warn("Proof is too small")
		return false
	}
	for _, value := range swProof.Proof {
		if value.BlockView != swProof.ViewNum {
			sw.Log.WithFields(logrus.Fields{
				"epoch":           sw.epoch,
				"replica":         sw.ID,
				"view":            sw.View.ViewNum,
				"proofView":       swProof.ViewNum,
				"len(proof)":      len(swProof.Proof),
				"blockHash":       hex.EncodeToString(swProof.BlockHash),
				"value.BlockView": value.BlockView,
			}).Warn("Proof view is incorrect")
			return false
		}
		if !bytes.Equal(value.BlockHash, swProof.BlockHash) {
			sw.Log.WithFields(logrus.Fields{
				"epoch":           sw.epoch,
				"replica":         sw.ID,
				"view":            sw.View.ViewNum,
				"proofView":       swProof.ViewNum,
				"len(proof)":      len(swProof.Proof),
				"blockHash":       hex.EncodeToString(swProof.BlockHash),
				"value.BlockHash": hex.EncodeToString(value.BlockHash),
			}).Warn("Proof hash is incorrect")
		}
	}
	return true
}

func (sw *SimpleWhirlyImpl) OnReceiveProposal(newBlock *pb.WhirlyBlock, swProof *pb.SimpleWhirlyProof, publicAddress string) {
	sw.Log.WithFields(logrus.Fields{
		"epoch":        sw.epoch,
		"replica":      sw.ID,
		"view":         sw.View.ViewNum,
		"block_height": newBlock.Height,
		"proof_view":   swProof.ViewNum,
	}).Trace("OnReceiveProposal")
	// sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum, "blockHash": hex.EncodeToString(newBlock.Hash)}).Trace("OnReceiveProposal")

	// if newBlock.ExecHeight == 1 && sw.GetP2pAdaptorType() != "p2p" {
	// 	// TODO: ensure to cancel send pingMsg when a proposal is received the first time
	// 	sw.stopSendPing()
	// }

	// store the block
	err := sw.BlockStorage.Put(newBlock)
	if err != nil {
		sw.Log.WithError(err).Info("Store the new block failed.")
	}

	// verfiy proposal
	v := newBlock.Height
	if v <= sw.lockProof.ViewNum {
		// sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum}).Warn("Receive old proposal")
		return
	}

	if !sw.verfiySwProof(swProof) {
		sw.Log.WithFields(logrus.Fields{
			"epoch":   sw.epoch,
			"replica": sw.ID,
			"view":    sw.View.ViewNum,
		}).Warn("Proposal proof is wrong")
		return
	}

	if !bytes.Equal(swProof.BlockHash, newBlock.ParentHash) {
		sw.Log.WithFields(logrus.Fields{
			"epoch":        sw.epoch,
			"replica":      sw.ID,
			"view":         sw.View.ViewNum,
			"proof_height": swProof.ViewNum,
			"block_height": newBlock.Height,
			"proof_hash":   hex.EncodeToString(swProof.BlockHash),
			"block_hash":   hex.EncodeToString(newBlock.ParentHash),
		}).Warn("OnReceiveProposal: proposal block and proof is incongruous")
		// sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum}).Warn("Proposal block and proof is incongruous")
		return
	}

	// verfiy ExecHeight
	if v <= sw.vHeight {
		// info log rathee than warn
		sw.Log.WithFields(logrus.Fields{
			"epoch":        sw.epoch,
			"replica":      sw.ID,
			"view":         sw.View.ViewNum,
			"block_height": newBlock.Height,
			"v_height":     sw.vHeight,
			"my_address":   sw.PublicAddress,
		}).Info("OnReceiveProposal: WhirlyBlock height less than vHeight")
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"epoch":   sw.epoch,
		"replica": sw.ID,
		"view":    sw.View.ViewNum,
	}).Debug("OnReceiveProposal: Accepted block")
	// update vHeight
	sw.vHeight = newBlock.Height
	sw.MemPool.MarkProposed(types.RawTxArrayFromBytes(newBlock.Txs))

	sw.waitProposal.Broadcast()

	sw.Update(swProof)
	sw.AdvanceView(newBlock.Height)

	if !sw.inCommittee {
		sw.Log.WithFields(logrus.Fields{
			"epoch":   sw.epoch,
			"replica": sw.ID,
			"view":    sw.View.ViewNum,
		}).Trace("OnReceiveProposal: Not in Committee")
		return
	}

	// vote for proposal
	var voteFlag bool
	var voteProof *pb.SimpleWhirlyProof
	if swProof.ViewNum >= sw.lockProof.ViewNum {
		voteFlag = true
		voteProof = nil
	} else {
		voteFlag = false
		voteProof = sw.lockProof
	}
	voteMsg := sw.VoteMsg(newBlock.Height, newBlock.Hash, voteFlag, nil, nil, voteProof, sw.epoch)

	if sw.GetLeader(sw.epoch) == sw.PublicAddress {
		// if sw.GetLeader(int64(newBlock.ExecHeight)) == sw.ID {
		// vote self
		vote := voteMsg.GetWhirlyVote()
		sw.OnReceiveVote(vote)
	} else {
		// send vote to the leader
		_ = sw.Unicast(publicAddress, voteMsg)

	}
}

func (sw *SimpleWhirlyImpl) OnReceiveVote(whirlyVoteMsg *pb.WhirlyVote) {
	if !sw.inCommittee {
		sw.Log.WithFields(logrus.Fields{
			"epoch":   sw.epoch,
			"replica": sw.ID,
			"view":    sw.View.ViewNum,
		}).Trace("OnReceiveVote: Not in Committee")
		return
	}

	if int64(whirlyVoteMsg.Epoch) < sw.epoch {
		return
	}

	if sw.GetLeader(sw.epoch) != sw.PublicAddress {
		// if sw.GetLeader(int64(whirlyVoteMsg.BlockView)) != sw.ID {
		sw.Log.WithFields(logrus.Fields{
			"epoch":     sw.epoch,
			"replica":   sw.ID,
			"view":      sw.View.ViewNum,
			"senderId":  whirlyVoteMsg.SenderId,
			"getleader": sw.GetLeader(sw.epoch),
			"blockView": whirlyVoteMsg.BlockView,
			"sw.ID":     sw.ID,
			"Flag":      whirlyVoteMsg.Flag,
		}).Warn("OnReceiveVote with error node")
		// sw.Log.Warn("OnReceiveVote with error block view")
		return
	}

	// Ignore messages from old views
	if whirlyVoteMsg.BlockView < sw.proposeView || whirlyVoteMsg.BlockView <= sw.lockProof.ViewNum {
		// sw.Log.Warn("ignore vote from old views")
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"epoch":        sw.epoch,
		"replica":      sw.ID,
		"view":         sw.View.ViewNum,
		"senderId":     whirlyVoteMsg.SenderId,
		"blockView":    whirlyVoteMsg.BlockView,
		"flag":         whirlyVoteMsg.Flag,
		"len(YesVote)": len(sw.curYesVote),
		"weight":       whirlyVoteMsg.Weight,
	}).Trace("OnReceiveVote")
	// sw.Log.WithFields(logrus.Fields{"epoch": sw.epoch, "replica": sw.ID, "view": sw.View.ViewNum}).Trace("OnReceiveVote")

	if whirlyVoteMsg.Flag {
		sw.voteLock.Lock()
		sw.proposalLock.Lock()
		if whirlyVoteMsg.BlockView >= sw.proposeView {
			for i := 0; i < int(whirlyVoteMsg.Weight); i++ {
				sw.curYesVote = append(sw.curYesVote, whirlyVoteMsg)
				sw.incentive.InsertYesVote(whirlyVoteMsg.PublicAddress)
			}
		}
		sw.proposalLock.Unlock()

	} else {
		// TODO: verfiySwProof
		sw.voteLock.Lock()
		for i := 0; i < int(whirlyVoteMsg.Weight); i++ {
			sw.curNoVote = append(sw.curNoVote, whirlyVoteMsg.SwProof)
			sw.incentive.InsertNoVote(whirlyVoteMsg.PublicAddress)
		}
		sw.lock.Lock()
		sw.UpdateLockProof(whirlyVoteMsg.SwProof)
		sw.lock.Unlock()
	}

	if len(sw.curYesVote)+len(sw.curNoVote) >= sw.GetQuorumSize() {
		if len(sw.curYesVote) >= sw.GetQuorumSize() {
			proof := sw.Proof(whirlyVoteMsg.BlockView, sw.curYesVote, whirlyVoteMsg.BlockHash)
			sw.Log.WithFields(logrus.Fields{
				"epoch":     sw.epoch,
				"replica":   sw.ID,
				"view":      sw.View.ViewNum,
				"proofView": proof.ViewNum,
			}).Trace("Create full proof")
			// update proofHigh
			sw.lock.Lock()
			sw.UpdateLockProof(proof)
			sw.lock.Unlock()
		} else {
			sw.Log.WithFields(logrus.Fields{
				"epoch":       sw.epoch,
				"replica":     sw.ID,
				"view":        sw.View.ViewNum,
				"msgView":     whirlyVoteMsg.BlockView,
				"len(NoVote)": len(sw.curNoVote),
			}).Error("Have NoVote")
		}

		sw.AdvanceView(whirlyVoteMsg.BlockView)
		go sw.OnPropose()
	}
	sw.voteLock.Unlock()
}

func (sw *SimpleWhirlyImpl) OnPropose() {
	// =====================================
	// == Testing the Byzantine situation ==
	// =====================================
	// if sw.ID == 3 {
	// 	return
	// }
	time.Sleep(2 * time.Second)
	sw.proposalLock.Lock()
	if sw.GetLeader(sw.epoch) != sw.PublicAddress || sw.proposeView >= sw.View.ViewNum {
		// if sw.GetLeader(int64(1)) != sw.ID {
		sw.Log.WithFields(logrus.Fields{
			"epoch":   sw.epoch,
			"replica": sw.ID,
			"view":    sw.View.ViewNum,
			"nowView": sw.View.ViewNum,
			"leader":  sw.GetLeader(sw.epoch),
		}).Trace("OnPropose: not allow")
		sw.proposalLock.Unlock()
		return
	}
	sw.proposeView = sw.View.ViewNum
	sw.proposalLock.Unlock()

	sw.Log.WithFields(logrus.Fields{
		"epoch":   sw.epoch,
		"replica": sw.ID,
		"view":    sw.View.ViewNum,
	}).Trace("OnPropose")
	txs := sw.MemPool.GetFirst(int(sw.Config.Whirly.BatchSize))
	if len(txs) != 0 {
		_ = 1
	} else {
		_ = 1
		// Produce empty blocks when there is no tx
		// return
	}

	// create node
	proposal := sw.createProposal(txs)
	// create a new prepare msg
	msg := sw.ProposalMsg(proposal, nil, sw.lockProof, sw.epoch)

	// 创建完区块后才能clean
	sw.voteLock.Lock()
	sw.CleanVote()
	sw.voteLock.Unlock()

	// the old leader should vote too
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)

	// the following line may panic
	defer func() {
		if err := recover(); err != nil {
			sw.Log.WithField("error", err).Warn("[SIMPLE WHIRLY] insert into channel failed")
		}
	}()

	if sw.GetP2pAdaptorType() == "p2p" {
		sw.MsgByteEntrance <- msgByte
	}
	// broadcast
	err = sw.Broadcast(msg)
	if err != nil {
		sw.Log.WithField("error", err.Error()).Warn("Broadcast proposal failed.")
	}
}

// expectBlock looks for a block with the given Hash, or waits for the next proposal to arrive
// sw.lock must be locked when calling this function
func (sw *SimpleWhirlyImpl) expectBlock(hash []byte) (*pb.WhirlyBlock, error) {
	for {
		block, err := sw.BlockStorage.Get(hash)
		if err == nil {
			return block, nil
		} else {
			_ = 1
		}
		sw.waitProposal.Wait()
	}
}

// createProposal create a new proposal
func (sw *SimpleWhirlyImpl) createProposal(txs []types.RawTransaction) *pb.WhirlyBlock {
	// create a new block
	sw.lock.Lock()
	// do not use proof fields for blocks
	incentiveBytes, err := whirlyUtilities.EncodeIncentive(sw.incentive)
	if err != nil {
		sw.Log.Error("Encode Incentive Error: ", err)
	}
	// sw.Log.Info("create Incentive", sw.incentive)

	block := sw.CreateLeaf(sw.lockProof.BlockHash, sw.View.ViewNum, txs, nil, incentiveBytes)
	sw.lock.Unlock()
	// store the block
	err = sw.BlockStorage.Put(block)
	if err != nil {
		sw.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Error("Store new block failed!")
	}
	return block
}

func (sw *SimpleWhirlyImpl) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (sw *SimpleWhirlyImpl) UpdateLockProof(swProof *pb.SimpleWhirlyProof) {
	block, _ := sw.expectBlock(swProof.BlockHash)
	if block == nil {
		sw.Log.Warn("Could not find block of new proof.")
		return
	}
	oldProofHighBlock, _ := sw.BlockStorage.Get(sw.lockProof.BlockHash)
	if oldProofHighBlock == nil {
		sw.Log.Error("WhirlyBlock from the old proofHigh missing from storage.")
		return
	}
	if block.Height > oldProofHighBlock.Height {
		sw.Log.WithFields(logrus.Fields{
			"epoch":                 sw.epoch,
			"replica":               sw.ID,
			"view":                  sw.View.ViewNum,
			"old":                   sw.lockProof.ViewNum,
			"new":                   swProof.ViewNum,
			"lock_proof_block_hash": utils.EncodeShortPrint(swProof.BlockHash),
			"block_hash":            utils.EncodeShortPrint(block.Hash),
		}).Trace("UpdateLockProof")
		// p.log.Trace("[SIMPLE WHIRLY] UpdateLockProof.")
		sw.lockProof = swProof
		sw.bLock = block
	}
}

func (sw *SimpleWhirlyImpl) AdvanceView(viewNum uint64) {
	if viewNum >= sw.View.ViewNum {
		sw.View.ViewNum = viewNum
		sw.View.ViewNum++
		sw.Log.WithFields(logrus.Fields{
			"epoch":   sw.epoch,
			"replica": sw.ID,
			"view":    sw.View.ViewNum,
		}).Trace("AdvanceView success")
	}
}

func (e *SimpleWhirlyImpl) GetConsensusType() string {
	return "whirly"
}
