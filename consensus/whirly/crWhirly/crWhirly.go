package crWhirly

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	whirlyUtilities "github.com/zzz136454872/upgradeable-consensus/consensus/whirly"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"

	bc_api "blockchain-crypto/blockchain_api"
)

type Event uint8

type CrWhirly interface {
	Update(block *pb.WhirlyBlock)
	OnCommit(block *pb.WhirlyBlock)
	OnReceiveProposal(newBlock *pb.WhirlyBlock, crProof *pb.CrWhirlyProof)
	OnReceiveVote(msg *pb.WhirlyMsg)
	OnPropose()
	GetPoTByteEntrance() chan<- []byte
}

type CrWhirlyImpl struct {
	whirlyUtilities.WhirlyUtilitiesImpl
	epoch             int64
	leader            map[int64]string
	leaderLock        sync.Mutex
	lock              sync.Mutex
	voteLock          sync.Mutex
	proposalLock      sync.Mutex
	curYesRoundShares [][]*bc_api.RoundShare
	curYesVote        []*pb.CrWhirlyVote
	curNoVote         []*pb.CrWhirlyProof
	proposeView       uint64
	bLock             *pb.WhirlyBlock
	bExec             *pb.WhirlyBlock
	txCRs             []*bc_api.TxCR
	lockProof         *pb.CrWhirlyProof
	vHeight           uint64
	waitProposal      *sync.Cond
	cancel            context.CancelFunc

	// New Epoch Echo
	newEpoch NewEpochMechanism

	// Latest block synchronization
	latestBlockReq LatestBlockRequestMechanism

	inCommittee    bool
	Committee      []string
	stopEntrance   chan string
	updateEntrance chan string

	// Ping
	readyNodes     []string
	sendPingCancel context.CancelFunc
}

func NewCrWhirly(
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
) *CrWhirlyImpl {
	log.WithField("consensus id", cid).Debug("[CR WHIRLY] starting")
	log.WithField("consensus id", cid).Trace("[CR WHIRLY] Generate genesis block")
	genesisBlock := whirlyUtilities.GenerateGenesisBlock()
	ctx, cancel := context.WithCancel(context.Background())
	sw := &CrWhirlyImpl{
		bLock:     genesisBlock,
		bExec:     genesisBlock,
		lockProof: nil,
		vHeight:   genesisBlock.Height,
		// pendingUpdate: make(chan *pb.CrWhirlyProof, 1),
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
	sw.lockProof = sw.CrProof(sw.View.ViewNum, nil, nil, genesisBlock.Hash)

	// view number add 1
	sw.View.ViewNum++
	sw.waitProposal = sync.NewCond(&sw.lock)
	sw.Log.WithField("replicaID", id).Debug("[CR WHIRLY] Init block storage.")
	sw.Log.WithField("replicaID", id).Debug("[CR WHIRLY] Init command cache.")

	sw.CleanVote(0)
	sw.inCommittee = false
	sw.epoch = 1
	sw.proposeView = 0
	// ensure p2pAdaptor of CrWhirly is same as PoT

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

	sw.Log.Info("[CR WHIRLY]\tstart to work")
	return sw
}

func NewCrWhirlyForLocalTest(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *CrWhirlyImpl {
	log.WithField("consensus id", cid).Debug("[CR WHIRLY] starting")
	log.WithField("consensus id", cid).Trace("[CR WHIRLY] Generate genesis block")
	genesisBlock := whirlyUtilities.GenerateGenesisBlock()
	ctx, cancel := context.WithCancel(context.Background())
	sw := &CrWhirlyImpl{
		bLock:     genesisBlock,
		bExec:     genesisBlock,
		lockProof: nil,
		vHeight:   genesisBlock.Height,
		// pendingUpdate: make(chan *pb.CrWhirlyProof, 1),
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
	sw.lockProof = sw.CrProof(sw.View.ViewNum, nil, nil, genesisBlock.Hash)

	// view number add 1
	sw.View.ViewNum++
	sw.waitProposal = sync.NewCond(&sw.lock)
	sw.Log.WithField("replicaID", id).Debug("[CR WHIRLY] Init block storage.")
	sw.Log.WithField("replicaID", id).Debug("[CR WHIRLY] Init command cache.")

	sw.CleanVote(0)
	sw.epoch = 1
	sw.proposeView = 0
	// ensure p2pAdaptor of CrWhirly is same as PoT

	sw.leader = make(map[int64]string)
	// sw.PoTByteEntrance = make(chan []byte, 10)

	// go sw.updateAsync(ctx)
	go sw.receiveMsg(ctx)

	sw.SetLeader(sw.epoch, cfg.Nodes[1].Address)
	if sw.GetPeerID() == cfg.Nodes[1].Address {
		// TODO: ensure all nodes is ready before OnPropose
		sw.SetLeader(sw.epoch, sw.GetPeerID())
		go sw.OnPropose()
	} else {

	}
	// sw.testNewLeader()

	sw.Log.Info("[CR WHIRLY]\tstart to work")
	return sw
}

func (sw *CrWhirlyImpl) SetLeader(epoch int64, leaderID string) {
	sw.leaderLock.Lock()
	// sw.epoch = epoch
	sw.leader[epoch] = leaderID
	sw.leaderLock.Unlock()
}

func (sw *CrWhirlyImpl) SetEpoch(epoch int64) {
	if epoch > sw.epoch {
		sw.epoch = epoch
	}
}

func (sw *CrWhirlyImpl) SetCryptoElements(cConfig bc_api.CommitteeConfig) {
	sw.CommitteeCryptoElements = cConfig
}

func (sw *CrWhirlyImpl) UpdateExternalStatus(status model.ExternalStatus) {
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

func (sw *CrWhirlyImpl) GetLeader(epoch int64) string {
	sw.leaderLock.Lock()
	leaderID, ok := sw.leader[epoch]
	sw.leaderLock.Unlock()
	if !ok {
		return ""
	}
	return leaderID
}

func (sw *CrWhirlyImpl) CleanVote(txLength int) {
	sw.curYesRoundShares = make([][]*bc_api.RoundShare, txLength)
	sw.curYesVote = make([]*pb.CrWhirlyVote, txLength)
	sw.curNoVote = make([]*pb.CrWhirlyProof, 0)
}

func (sw *CrWhirlyImpl) GetHeight() uint64 {
	return sw.bLock.Height
}

func (sw *CrWhirlyImpl) GetVHeight() uint64 {
	return sw.vHeight
}

func (sw *CrWhirlyImpl) GetLock() *pb.WhirlyBlock {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	return sw.bLock
}

func (sw *CrWhirlyImpl) SetLock(b *pb.WhirlyBlock) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	sw.bLock = b
}

func (sw *CrWhirlyImpl) GetLockProof() *pb.CrWhirlyProof {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	return sw.lockProof
}

// func (sw *CrWhirlyImpl) GetPoTByteEntrance() chan<- []byte {
// 	return sw.PoTByteEntrance
// }

func (sw *CrWhirlyImpl) Stop() {
	sw.cancel()
	sw.BlockStorage.Close()
	close(sw.MsgByteEntrance)
	close(sw.RequestEntrance)
	// close(sw.PoTByteEntrance)
	newPeerId1 := strings.Replace(sw.PublicAddress, ".", "", -1)
	newPeerId2 := strings.Replace(newPeerId1, ":", "", -1)
	_ = os.RemoveAll("dbfile/node" + newPeerId2)
	_ = os.RemoveAll("store")
}

func (sw *CrWhirlyImpl) receiveMsg(ctx context.Context) {
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

func (sw *CrWhirlyImpl) handleMsg(msg *pb.WhirlyMsg) {
	// sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] recevice msg")
	switch msg.Payload.(type) {
	case *pb.WhirlyMsg_Request:
		request := msg.GetRequest()
		// if sw.ID == 0 {
		//	sw.Log.WithField("content", request.String()).Error("[SIMPLE WHIRLY] Get request msg.")
		// }

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
	case *pb.WhirlyMsg_CrWhirlyProposal:
		proposalMsg := msg.GetCrWhirlyProposal()
		if int64(proposalMsg.Epoch) < sw.epoch {
			return
		}
		sw.OnReceiveProposal(proposalMsg.Block, proposalMsg.CrProof, proposalMsg.PublicAddress)
	case *pb.WhirlyMsg_CrWhirlyVote:
		whirlyVoteMsg := msg.GetCrWhirlyVote()
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
func (sw *CrWhirlyImpl) Update(crProof *pb.CrWhirlyProof) {
	// block1 = b'', block2 = b', block3 = b

	sw.lock.Lock()
	defer sw.lock.Unlock()

	block1, err := sw.expectBlock(crProof.BlockHash)
	if err != nil && err != leveldb.ErrNotFound {
		sw.Log.Fatal(err)
	}
	if block1 == nil || block1.Committed {
		return
	}
	// sw.lock.Unlock()

	sw.Log.WithField(
		"blockHash", hex.EncodeToString(block1.Hash)).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] [SIMPLE WHIRLY] LOCK.")
	// pre-commit block1
	sw.UpdateLockProof(crProof)

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
				"blockHash":   hex.EncodeToString(block2.Hash),
				"blockHeight": block2.Height,
				"myAddress":   sw.PublicAddress,
			}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] [CR WHIRLY] COMMIT.")
		}
		// sw.Log.WithField(
		// 	"blockHash", hex.EncodeToString(block2.Hash)).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] [SIMPLE WHIRLY] COMMIT.")
		sw.OnCommit(block2)
		sw.bExec = block2
	}
}

func (sw *CrWhirlyImpl) OnCommit(block *pb.WhirlyBlock) {
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
		// sw.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] [SIMPLE WHIRLY] EXEC.")
		// 只有 DaemonNode 需要提交交易
		_, _, address := DecodeAddress(sw.PublicAddress)
		if sw.ID == 0 || address == DaemonNodePublicAddress {
			go sw.ProcessProposal(block, []byte{})
		}
	}
}

func (sw *CrWhirlyImpl) verfiyCrProof(crProof *pb.CrWhirlyProof) bool {
	if crProof.ViewNum == 0 {
		return true
	}
	if len(crProof.VoteProof) < 2*sw.Config.F+1 {
		sw.Log.WithFields(logrus.Fields{
			"proofHeight": crProof.ViewNum,
			"len(proof)":  len(crProof.VoteProof),
			"blockHash":   hex.EncodeToString(crProof.BlockHash),
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] proof is too small.")
		return false
	}
	// TODO: verfig proofs
	return true
}

func (sw *CrWhirlyImpl) OnReceiveProposal(newBlock *pb.WhirlyBlock, crProof *pb.CrWhirlyProof, publicAddress string) {
	sw.Log.WithFields(logrus.Fields{
		"blockHeight": newBlock.Height,
		"proofView":   crProof.ViewNum,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveProposal.")
	// sw.Log.WithField("blockHash", hex.EncodeToString(newBlock.Hash)).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveProposal.")

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
		// sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] receive old proposal.")
		return
	}

	if !sw.verfiyCrProof(crProof) {
		sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] proposal proof is wrong.")
		return
	}

	if !bytes.Equal(crProof.BlockHash, newBlock.ParentHash) {
		sw.Log.WithFields(logrus.Fields{
			"proofHeight": crProof.ViewNum,
			"blockHeight": newBlock.Height,
			"proofHash":   hex.EncodeToString(crProof.BlockHash),
			"blockHash":   hex.EncodeToString(newBlock.ParentHash),
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveProposal: proposal block and proof is incongruous.")
		// sw.Log.Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] proposal block and proof is incongruous.")
		return
	}

	// verfiy ExecHeight
	if v <= sw.vHeight {
		// info log rathee than warn
		sw.Log.WithFields(logrus.Fields{
			"blockHeight": newBlock.Height,
			"vHeight":     sw.vHeight,
			"myAddress":   sw.PublicAddress,
		}).Info("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveProposal: WhirlyBlock height less than vHeight.")
		return
	}

	sw.Log.Debug("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "]  OnReceiveProposal: Accepted block.")
	// update vHeight
	sw.vHeight = newBlock.Height
	sw.MemPool.MarkProposed(types.RawTxArrayFromBytes(newBlock.Txs))

	sw.waitProposal.Broadcast()

	sw.Update(crProof)
	sw.AdvanceView(newBlock.Height)

	if !sw.inCommittee {
		sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "]  OnReceiveProposal: Not in Committee.")
		return
	}

	// vote for proposal
	var voteFlag bool
	var voteProof *pb.CrWhirlyProof
	var roundShares [][]byte
	if crProof.ViewNum >= sw.lockProof.ViewNum {
		voteFlag = true
		voteProof = nil

		for _, rtx := range newBlock.GetTxs() {
			tx, err := types.RawTransaction(rtx).ToTx()
			if err != nil {
				sw.Log.Warn("Invalid transaction")
				return
			}
			if tx.Type != pb.TransactionType_CENSORSHIP_RESISTANT {
				continue
			}
			// Threshold Decryption
			txCR, err := bc_api.DecodeBytesToTxCR(tx.Payload)
			if err != nil {
				sw.Log.Warn("Invalid txCR in raw transaction: ", err)
				continue
			}

			roundShare, roundShareProof, err := bc_api.CalcRoundShareOfDPVSS(
				uint32(sw.ID+1),
				sw.CommitteeCryptoElements.H,
				txCR.C,
				sw.CommitteeCryptoElements.DPVSSConfig.ShareCommits[sw.ID],
				sw.CommitteeCryptoElements.DPVSSConfig.Share,
			)
			if err != nil {
				sw.Log.Warn("CalcRoundShareOfDPVSS error: ", err)
				continue
			}
			rs := bc_api.RoundShare{
				Index: uint32(sw.ID + 1),
				Piece: roundShare,
				Proof: roundShareProof,
			}
			rsBytes, err := bc_api.EncodeRoundShareToBytes(rs)
			if err != nil {
				sw.Log.Warn("EncodeRoundShareToBytes error: ", err)
				continue
			}
			roundShares = append(roundShares, rsBytes)
		}

	} else {
		voteFlag = false
		voteProof = sw.lockProof
		roundShares = nil
	}
	// voteMsg := sw.VoteMsg(newBlock.Height, newBlock.Hash, voteFlag, nil, nil, voteProof, sw.epoch)
	voteMsg := &pb.WhirlyMsg{}

	voteMsg.Payload = &pb.WhirlyMsg_CrWhirlyVote{CrWhirlyVote: &pb.CrWhirlyVote{
		SenderId:        uint64(sw.ID),
		BlockView:       newBlock.Height,
		BlockHash:       newBlock.Hash,
		Flag:            voteFlag,
		CrProof:         voteProof,
		Epoch:           uint64(sw.epoch),
		PublicAddress:   sw.PublicAddress,
		Weight:          uint64(sw.Weight),
		DecryptedShares: roundShares,
	}}

	if sw.GetLeader(sw.epoch) == sw.PublicAddress {
		// if sw.GetLeader(int64(newBlock.ExecHeight)) == sw.ID {
		// vote self
		vote := voteMsg.GetCrWhirlyVote()
		sw.OnReceiveVote(vote)
	} else {
		// send vote to the leader
		_ = sw.Unicast(publicAddress, voteMsg)

	}
}

func (sw *CrWhirlyImpl) OnReceiveVote(whirlyVoteMsg *pb.CrWhirlyVote) {
	if !sw.inCommittee {
		sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "]  OnReceiveVote: Not in Committee.")
		return
	}

	if int64(whirlyVoteMsg.Epoch) < sw.epoch {
		return
	}

	if sw.GetLeader(sw.epoch) != sw.PublicAddress {
		// if sw.GetLeader(int64(whirlyVoteMsg.BlockView)) != sw.ID {
		sw.Log.WithFields(logrus.Fields{
			"senderId":  whirlyVoteMsg.SenderId,
			"getleader": sw.GetLeader(sw.epoch),
			"blockView": whirlyVoteMsg.BlockView,
			"sw.ID":     sw.ID,
			"Flag":      whirlyVoteMsg.Flag,
		}).Warn("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveVote with error node.")
		// sw.Log.Warn("OnReceiveVote with error block view")
		return
	}

	// Ignore messages from old views
	if whirlyVoteMsg.BlockView < sw.proposeView || whirlyVoteMsg.BlockView <= sw.lockProof.ViewNum {
		// sw.Log.Warn("ignore vote from old views")
		return
	}

	sw.Log.WithFields(logrus.Fields{
		"senderId":     whirlyVoteMsg.SenderId,
		"blockView":    whirlyVoteMsg.BlockView,
		"flag":         whirlyVoteMsg.Flag,
		"len(YesVote)": len(sw.curYesVote),
		"weight":       whirlyVoteMsg.Weight,
	}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveVote.")
	// sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnReceiveVote.")

	if whirlyVoteMsg.Flag {
		sw.voteLock.Lock()
		sw.proposalLock.Lock()
		if whirlyVoteMsg.BlockView >= sw.proposeView {
			if whirlyVoteMsg.BlockHash == nil {
				sw.Log.Warn("Invaild vote: BlockHash is null")
				sw.proposalLock.Unlock()
				sw.voteLock.Unlock()
				return
			}

			sw.curYesVote = append(sw.curYesVote, whirlyVoteMsg)
			if sw.txCRs != nil && len(sw.txCRs) != 0 {
				if whirlyVoteMsg.DecryptedShares == nil {
					sw.Log.Warn("Invaild vote: DecryptedShares is null")
					sw.proposalLock.Unlock()
					sw.voteLock.Unlock()
					return
				}

				if len(whirlyVoteMsg.DecryptedShares) != len(sw.txCRs) {
					sw.Log.Warn("Invaild vote: The length of decrypted shares is invaild")
					sw.proposalLock.Unlock()
					sw.voteLock.Unlock()
					return
				}

				for i := 0; i < len(sw.txCRs); i++ {
					roundShare, err := bc_api.DecodeBytesToRoundShare(whirlyVoteMsg.DecryptedShares[i])
					if err != nil {
						sw.Log.Warn("DecodeBytesToRoundShare error: ", err)
						sw.proposalLock.Unlock()
						sw.voteLock.Unlock()
						return
					}

					verifyRoundShareRes := bc_api.VerifyRoundShareOfDPVSS(
						roundShare.Index,
						sw.CommitteeCryptoElements.H,
						sw.txCRs[i].C,
						sw.CommitteeCryptoElements.DPVSSConfig.ShareCommits[i],
						roundShare.Piece, roundShare.Proof)
					// 有效则使用
					if verifyRoundShareRes {
						sw.curYesRoundShares[i] = append(sw.curYesRoundShares[i], roundShare)
						// validRoundSharesBytes = append(validRoundSharesBytes, roundShare)
					}
				}
			}
		}
		sw.proposalLock.Unlock()

	} else {
		// TODO: verfiycrProof
		sw.voteLock.Lock()
		for i := 0; i < int(whirlyVoteMsg.Weight); i++ {
			sw.curNoVote = append(sw.curNoVote, whirlyVoteMsg.CrProof)
		}
		sw.lock.Lock()
		sw.UpdateLockProof(whirlyVoteMsg.CrProof)
		sw.lock.Unlock()
	}

	if len(sw.curYesVote)+len(sw.curNoVote) >= 2*sw.Config.F+1 {
		if len(sw.curYesVote) >= 2*sw.Config.F+1 {
			var plaintextTxsBytes [][]byte
			if sw.txCRs == nil || len(sw.txCRs) == 0 {
				plaintextTxsBytes = nil
			} else {
				plaintextTxsBytes = make([][]byte, len(sw.txCRs))
				for i := 0; i < len(sw.txCRs); i++ {
					// 领导者重构门限加密密文
					roundSecretBytes := bc_api.RecoverRoundSecret(uint32(2*sw.Config.F+1), sw.curYesRoundShares[i])
					if roundSecretBytes == nil {
						// 共识失败，丢弃交易
						sw.voteLock.Unlock()
						return
					}

					// 领导者解密门限加密的交易密钥密文，得到交易密钥
					decryptedTxKeyBytes, err := bc_api.DecryptIBVE(roundSecretBytes, sw.txCRs[i].CipherTxKey)
					if err != nil {
						sw.Log.Warn("DecryptIBVE error: ", err)
						sw.voteLock.Unlock()
						return
					}
					// 领导者对称解密交易密文，得到交易明文
					decryptedTx, err := bc_api.DecryptTx(decryptedTxKeyBytes, sw.txCRs[i].CipherTx)
					if err != nil {
						sw.Log.Warn("DecryptTx error: ", err)
						sw.voteLock.Unlock()
						return
					}
					// 领导者记录交易明文
					sw.txCRs[i].Tx = decryptedTx

					plaintextTxsBytes[i], err = bc_api.EncodeTxCRToBytes(*sw.txCRs[i])
					if err != nil {
						sw.Log.Warn("EncodeTxCRToBytes error: ", err)
						sw.voteLock.Unlock()
						return
					}
				}
			}

			proof := sw.CrProof(whirlyVoteMsg.BlockView, plaintextTxsBytes, sw.curYesVote, whirlyVoteMsg.BlockHash)
			sw.Log.WithFields(logrus.Fields{
				"proofView": proof.ViewNum,
			}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] create full proof!")
			// update proofHigh
			sw.lock.Lock()
			sw.UpdateLockProof(proof)
			sw.lock.Unlock()
		} else {
			sw.Log.WithFields(logrus.Fields{
				"msgView":     whirlyVoteMsg.BlockView,
				"len(NoVote)": len(sw.curNoVote),
			}).Error("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] have NoVote!")
		}

		sw.AdvanceView(whirlyVoteMsg.BlockView)
		go sw.OnPropose()
	}
	sw.voteLock.Unlock()
}

func (sw *CrWhirlyImpl) OnPropose() {
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
			"nowView": sw.View.ViewNum,
			"leader":  sw.GetLeader(sw.epoch),
		}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnPropose: not allow!")
		sw.proposalLock.Unlock()
		return
	}
	sw.proposeView = sw.View.ViewNum
	sw.proposalLock.Unlock()

	sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] OnPropose")
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
	msg := &pb.WhirlyMsg{}

	msg.Payload = &pb.WhirlyMsg_CrWhirlyProposal{CrWhirlyProposal: &pb.CrWhirlyProposal{
		SenderId:      uint64(sw.ID),
		Block:         proposal,
		CrProof:       sw.lockProof,
		Epoch:         uint64(sw.epoch),
		PublicAddress: sw.PublicAddress,
	}}

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
func (sw *CrWhirlyImpl) expectBlock(hash []byte) (*pb.WhirlyBlock, error) {
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
func (sw *CrWhirlyImpl) createProposal(txs []types.RawTransaction) *pb.WhirlyBlock {
	// create a new block
	sw.lock.Lock()
	// do not use proof fields for blocks
	block := sw.CreateLeaf(sw.lockProof.BlockHash, sw.View.ViewNum, txs, nil)
	sw.lock.Unlock()
	// store the block
	err := sw.BlockStorage.Put(block)
	if err != nil {
		sw.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Error("Store new block failed!")
	}
	sw.voteLock.Lock()
	sw.CleanVote(len(block.GetTxs()))
	sw.voteLock.Unlock()

	sw.txCRs = make([]*bc_api.TxCR, len(txs))
	sw.Log.WithField("blockHash", hex.EncodeToString(block.Hash)).Warn("Store new block: ", len(sw.txCRs))
	for i := 0; i < len(txs); i++ {
		tx, err := txs[i].ToTx()
		if err != nil {
			sw.Log.Warn("Invalid transaction")
			continue
		}
		if tx.Type != pb.TransactionType_CENSORSHIP_RESISTANT {
			continue
		}
		sw.txCRs[i], err = bc_api.DecodeBytesToTxCR(tx.Payload)
		if err != nil {
			sw.Log.Warn("Invalid txCR in raw transaction: ", err)
			continue
		}
	}

	return block
}

func (sw *CrWhirlyImpl) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (sw *CrWhirlyImpl) UpdateLockProof(crProof *pb.CrWhirlyProof) {
	block, _ := sw.expectBlock(crProof.BlockHash)
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
			"old":                 sw.lockProof.ViewNum,
			"new":                 crProof.ViewNum,
			"lockProof.blockHash": hex.EncodeToString(crProof.BlockHash),
			"bLock.Hash":          hex.EncodeToString(block.Hash),
		}).Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] UpdateLockProof.")
		// p.log.Trace("[SIMPLE WHIRLY] UpdateLockProof.")
		sw.lockProof = crProof
		sw.bLock = block
	}

	txs := block.GetTxs()
	newTxs := make([][]byte, len(txs))

	if txs == nil || crProof.Proof == nil {
		return
	}

	if len(txs) != len(crProof.Proof) {
		sw.Log.Warn("UpdateLockProof: Invalid crProof")
		return
	}

	for i := 0; i < len(txs); i++ {
		tx, err := types.RawTransaction(txs[i]).ToTx()
		if err != nil {
			sw.Log.Warn("Invalid transaction")
			continue
		}
		if tx.Type != pb.TransactionType_CENSORSHIP_RESISTANT {
			continue
		}
		txCR, err := bc_api.DecodeBytesToTxCR(tx.Payload)
		if err != nil {
			sw.Log.Warn("Invalid txCR in raw transaction: ", err)
			continue
		}
		plaintextTx, err := bc_api.DecodeBytesToTxCR(crProof.Proof[i])
		if err != nil {
			sw.Log.Warn("Invalid txCR in crProof: ", err)
			continue
		}
		if !bytes.Equal(txCR.CipherTx, plaintextTx.CipherTx) {
			sw.Log.Warn("UpdateLockProof: Invalid proof in crProof")
			return
		}
		tx.Payload = crProof.Proof[i] // tx is pb.Transaction type, Transaction.Paylod is marshal value of txCR
		btx, err := proto.Marshal(tx) // btx is RawTransaction, it is a marshal value of pb.Transaction
		utils.PanicOnError(err)
		newTxs[i] = btx
	}
	block.Txs = newTxs
	sw.BlockStorage.UpdateState(block)
}

func (sw *CrWhirlyImpl) AdvanceView(viewNum uint64) {
	if viewNum >= sw.View.ViewNum {
		sw.View.ViewNum = viewNum
		sw.View.ViewNum++
		sw.Log.Trace("[epoch_" + strconv.Itoa(int(sw.epoch)) + "] [replica_" + strconv.Itoa(int(sw.ID)) + "] [view_" + strconv.Itoa(int(sw.View.ViewNum)) + "] advanceView success!")
	}
}
