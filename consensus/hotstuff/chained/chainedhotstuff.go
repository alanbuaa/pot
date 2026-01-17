package chained

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

type ChainedHotStuff struct {
	hotstuff.HotStuffImpl
	genericQC *pb.QuorumCert
	lockQC    *pb.QuorumCert
	lock      sync.Mutex
}

func NewChainedHotStuff(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *ChainedHotStuff {
	log = log.WithField("module", "CHAINEDHST").WithField("c_id", cid)
	log.Info("Initializing Chained HotStuff consensus")

	chs := &ChainedHotStuff{}
	chs.Init(id, cid, cfg, exec, p2pAdaptor, log)
	chs.View = hotstuff.NewView(1, 1)

	log.WithField("replica_id", id).Debug("Initializing block storage")
	log.Debug("Generating genesis block")
	genesisBlock := hotstuff.GenerateGenesisBlock()
	err := chs.BlockStorage.Put(genesisBlock)
	if err != nil {
		log.WithError(err).Fatal("Failed to store genesis block")
	}
	log.Debug("Genesis block stored successfully")
	chs.genericQC = &pb.QuorumCert{
		BlockHash: genesisBlock.Hash,
		Type:      pb.MsgType_PREPARE_VOTE,
		ViewNum:   0,
		Signature: nil,
	}
	chs.lockQC = &pb.QuorumCert{
		BlockHash: genesisBlock.Hash,
		Type:      pb.MsgType_PREPARE_VOTE,
		ViewNum:   0,
		Signature: nil,
	}
	log.WithField("replica_id", id).Debug("Quorum certificates initialized")

	// Initialize timers
	chs.TimeChan = utils.NewTimer(time.Duration(chs.Config.HotStuff.Timeout) * time.Second)
	chs.TimeChan.Init()
	log.WithField("timeout_seconds", chs.Config.HotStuff.Timeout).Debug("View timeout timer initialized")

	chs.BatchTimeChan = utils.NewTimer(time.Duration(chs.Config.HotStuff.BatchTimeout) * time.Second)
	chs.BatchTimeChan.Init()
	log.WithField("batch_timeout_seconds", chs.Config.HotStuff.BatchTimeout).Debug("Batch timer initialized")

	// Initialize current execution state
	chs.CurExec = &hotstuff.CurProposal{
		Node:         nil,
		DocumentHash: nil,
		PrepareVote:  make([]*tcrsa.SigShare, 0),
		HighQC:       make([]*pb.QuorumCert, 0),
	}

	log.Info("Chained HotStuff consensus initialized successfully")
	go chs.receiveMsg()
	return chs
}

func (chs *ChainedHotStuff) receiveMsg() {
	chs.Wg.Add(1)
	for {
		select {
		case msgByte := <-chs.MsgByteEntrance:
			msg, err := chs.DecodeMsgByte(msgByte)
			if err != nil {
				chs.Log.WithError(err).Warn("decode message failed")
				continue
			}
			go chs.handleMsg(msg)
		case request := <-chs.RequestEntrance:
			// if !ok {
			// 	return // closed
			// }
			chs.handleMsg(&pb.Msg{Payload: &pb.Msg_Request{Request: request}})
		case <-chs.BatchTimeChan.Timeout():
			chs.BatchTimeChan.Init()
			chs.batchEvent(chs.MemPool.GetFirst(int(chs.Config.HotStuff.BatchSize)))
		case <-chs.TimeChan.Timeout():
			chs.Config.HotStuff.Timeout = chs.Config.HotStuff.Timeout * 2
			chs.TimeChan.Init()
			// create dummy node
			//chs.CreateLeaf()
			// send new view msg
		case <-chs.Closed:
			chs.Wg.Done()
			return
		}
	}
}

func (chs *ChainedHotStuff) handleMsg(msg *pb.Msg) {
	switch msg.Payload.(type) {
	case *pb.Msg_Request:
		request := msg.GetRequest()
		chs.Log.WithField("tx_size", len(request.Tx)).Debug("Received client request")
		// put the cmd into the txset
		chs.MemPool.Add(types.RawTransaction(request.Tx))
		if chs.GetLeader() != chs.ID {
			// redirect to the leader
			chs.Log.Debug("Forwarding request to leader")
			chs.Unicast(chs.GetNetworkInfo()[chs.GetLeader()], msg)
			return
		}
		if chs.CurExec.Node != nil {
			return
		}
		chs.BatchTimeChan.SoftStartTimer()
		// if the length of unprocessed cmd equals to batch size, stop timer and call handleMsg to send prepare msg
		txs := chs.MemPool.GetFirst(int(chs.Config.HotStuff.BatchSize))
		chs.Log.WithField("cache_size", len(txs)).Debug("Current transaction cache size")
		if len(txs) == int(chs.Config.HotStuff.BatchSize) {
			// stop timer
			chs.Log.WithField("batch_size", len(txs)).Debug("Batch size reached, stopping batch timer")
			chs.BatchTimeChan.Stop()
			// create prepare msg
			chs.batchEvent(txs)
		}
	case *pb.Msg_Prepare:
		chs.Log.Debug("Received PREPARE message")
		if !chs.MatchingMsg(msg, pb.MsgType_PREPARE) {
			chs.Log.Warn("PREPARE message does not match current view")
			return
		}
		// get prepare msg
		prepare := msg.GetPrepare()

		if !chs.SafeNode(prepare.CurProposal, prepare.CurProposal.Justify) {
			chs.Log.Warn("PREPARE proposal is unsafe")
			return
		}
		// add view number and change leader
		chs.View.ViewNum++
		chs.View.Primary = chs.GetLeader()
		chs.Log.WithFields(logrus.Fields{
			"view":   chs.View.ViewNum,
			"leader": chs.View.Primary,
		}).Debug("View updated")

		marshal, _ := proto.Marshal(msg)
		chs.CurExec.DocumentHash, _ = crypto.CreateDocumentHash(marshal, chs.Config.Keys.PublicKey)
		chs.CurExec.Node = prepare.CurProposal
		partSig, _ := crypto.TSign(chs.CurExec.DocumentHash, chs.Config.Keys.PrivateKey, chs.Config.Keys.PublicKey)

		if chs.View.Primary == chs.ID {
			// vote self
			chs.Log.Debug("This node is the new leader, voting for self")
			chs.CurExec.PrepareVote = append(chs.CurExec.PrepareVote, partSig)
		} else {
			// send vote msg to the next leader
			chs.Log.WithField("leader", chs.GetLeader()).Debug("Sending PREPARE_VOTE to leader")
			partSigBytes, _ := json.Marshal(partSig)
			voteMsg := chs.VoteMsg(pb.MsgType_PREPARE_VOTE, prepare.CurProposal, nil, partSigBytes)
			chs.Unicast(chs.GetNetworkInfo()[chs.GetLeader()], voteMsg)
			chs.CurExec = hotstuff.NewCurProposal()
		}
		chs.update(prepare.CurProposal)
	case *pb.Msg_PrepareVote:
		chs.Log.Debug("Received PREPARE_VOTE message")
		if !chs.MatchingMsg(msg, pb.MsgType_PREPARE_VOTE) {
			chs.Log.Warn("PREPARE_VOTE message does not match")
			return
		}
		prepareVote := msg.GetPrepareVote()
		partSig := new(tcrsa.SigShare)
		chs.Log.Debug("Verifying partial signature")
		_ = json.Unmarshal(prepareVote.PartialSig, partSig)
		if err := crypto.VerifyPartSig(partSig, chs.CurExec.DocumentHash, chs.Config.Keys.PublicKey); err != nil {
			chs.Log.WithError(err).Warn("PREPARE_VOTE partial signature verification failed")
			return
		}
		chs.CurExec.PrepareVote = append(chs.CurExec.PrepareVote, partSig)
		chs.Log.WithFields(logrus.Fields{
			"vote_count": len(chs.CurExec.PrepareVote),
			"threshold":  chs.Config.F*2 + 1,
		}).Debug("Collected PREPARE_VOTE")
		if len(chs.CurExec.PrepareVote) == chs.Config.F*2+1 {
			chs.Log.Debug("Threshold reached, creating generic QC")
			signature, _ := crypto.CreateFullSignature(chs.CurExec.DocumentHash, chs.CurExec.PrepareVote, chs.Config.Keys.PublicKey)
			qc := chs.QC(pb.MsgType_PREPARE_VOTE, signature, prepareVote.BlockHash)
			chs.genericQC = qc
			chs.CurExec = hotstuff.NewCurProposal()
			chs.Log.Debug("Generic QC created, restarting batch timer")
			chs.BatchTimeChan.SoftStartTimer()
		}
	case *pb.Msg_NewView:
	}
}

func (chs *ChainedHotStuff) SafeNode(node *pb.WhirlyBlock, qc *pb.QuorumCert) bool {
	return bytes.Equal(node.ParentHash, chs.lockQC.BlockHash) || //safety rule
		qc.ViewNum > chs.lockQC.ViewNum // liveness rule
}

func (chs *ChainedHotStuff) update(block *pb.WhirlyBlock) {
	chs.lock.Lock()
	defer chs.lock.Unlock()
	if block.Justify == nil {
		chs.Log.Debug("Block has no justify, skipping update")
		return
	}
	// block = b*, block1 = b'', block2 = b', block3 = b
	block1, err := chs.BlockStorage.BlockOf(block.Justify)
	if err == leveldb.ErrNotFound || block1.Committed {
		return
	}
	// start pre-commit phase on block’s parent
	if bytes.Equal(block.ParentHash, block1.Hash) {
		chs.Log.WithField("block_hash", hex.EncodeToString(block1.Hash)).Info("Starting pre-commit phase")
		chs.genericQC = block.Justify
	}

	block2, err := chs.BlockStorage.BlockOf(block1.Justify)
	if err == leveldb.ErrNotFound || block2.Committed {
		return
	}
	// start commit phase on block1’s parent
	if bytes.Equal(block.ParentHash, block1.Hash) && bytes.Equal(block1.ParentHash, block2.Hash) {
		chs.Log.WithField("block_hash", hex.EncodeToString(block2.Hash)).Info("Starting commit phase")
		chs.lockQC = block1.Justify
	}

	block3, err := chs.BlockStorage.BlockOf(block2.Justify)
	if err == leveldb.ErrNotFound || block3.Committed {
		return
	}

	if bytes.Equal(block.ParentHash, block1.Hash) && bytes.Equal(block1.ParentHash, block2.Hash) &&
		bytes.Equal(block2.ParentHash, block3.Hash) {
		//decide
		chs.Log.WithField("block_hash", hex.EncodeToString(block3.Hash)).Info("Starting decide phase")
		chs.processProposal()
	}
}

func (chs *ChainedHotStuff) batchEvent(txs []types.RawTransaction) {
	// if batch timeout, check size
	if len(txs) == 0 {
		chs.Log.Debug("No transactions to batch, restarting timer")
		chs.BatchTimeChan.SoftStartTimer()
		return
	}

	chs.Log.WithField("tx_count", len(txs)).Debug("Creating proposal with batched transactions")

	// create prepare msg
	node := chs.CreateLeaf(chs.BlockStorage.GetLastBlockHash(), txs, chs.genericQC)
	chs.CurExec.Node = node
	chs.MemPool.MarkProposed(txs)
	if chs.HighQC == nil {
		chs.HighQC = chs.genericQC
	}
	prepareMsg := chs.Msg(pb.MsgType_PREPARE, node, chs.HighQC)
	// broadcast prepare msg
	chs.Log.Debug("Broadcasting PREPARE message")
	chs.Broadcast(prepareMsg)
	chs.TimeChan.SoftStartTimer()
}

func (chs *ChainedHotStuff) processProposal() {
	chs.Log.Debug("Processing committed proposal")
	// process proposal
	go chs.ProcessProposal(chs.CurExec.Node, []byte{})
	// store block
	chs.CurExec.Node.Committed = true
	chs.Log.Debug("Proposal marked as committed")
}

func (chs *ChainedHotStuff) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (ehs *ChainedHotStuff) GetConsensusType() string {
	return "chainedhotstuff"
}
