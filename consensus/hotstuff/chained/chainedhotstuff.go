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
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
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
	chs := &ChainedHotStuff{}
	chs.Init(id, cid, cfg, exec, p2pAdaptor, log)
	chs.View = hotstuff.NewView(1, 1)
	chs.Log.Debugf("[HOTSTUFF] Init block storage, replica id: %d", id)
	chs.Log.Debugf("[HOTSTUFF] Generate genesis block")
	genesisBlock := hotstuff.GenerateGenesisBlock()
	err := chs.BlockStorage.Put(genesisBlock)
	if err != nil {
		chs.Log.Fatal("generate genesis block failed")
	}
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
	chs.Log.Debugf("[HOTSTUFF] Init command set, replica id: %d", id)

	// init timer and stop it
	chs.TimeChan = utils.NewTimer(time.Duration(chs.Config.HotStuff.Timeout) * time.Second)
	chs.TimeChan.Init()

	chs.BatchTimeChan = utils.NewTimer(time.Duration(chs.Config.HotStuff.BatchTimeout) * time.Second)
	chs.BatchTimeChan.Init()

	chs.CurExec = &hotstuff.CurProposal{
		Node:         nil,
		DocumentHash: nil,
		PrepareVote:  make([]*tcrsa.SigShare, 0),
		HighQC:       make([]*pb.QuorumCert, 0),
	}
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
		chs.Log.Debugf("[CHAINED HOTSTUFF] Get request msg, content:%s", request.String())
		// put the cmd into the txset
		chs.MemPool.Add(types.RawTransaction(request.Tx))
		if chs.GetLeader() != chs.ID {
			// redirect to the leader
			chs.Unicast(chs.GetNetworkInfo()[chs.GetLeader()], msg)
			return
		}
		if chs.CurExec.Node != nil {
			return
		}
		chs.BatchTimeChan.SoftStartTimer()
		// if the length of unprocessed cmd equals to batch size, stop timer and call handleMsg to send prepare msg
		chs.Log.Debugf("Command cache size: %d", len(chs.MemPool.GetFirst(int(chs.Config.HotStuff.BatchSize))))
		txs := chs.MemPool.GetFirst(int(chs.Config.HotStuff.BatchSize))
		if len(txs) == int(chs.Config.HotStuff.BatchSize) {
			// stop timer
			chs.BatchTimeChan.Stop()
			// create prepare msg
			chs.batchEvent(txs)
		}
	case *pb.Msg_Prepare:
		chs.Log.Info("[CHAINED HOTSTUFF] Get generic msg")
		if !chs.MatchingMsg(msg, pb.MsgType_PREPARE) {
			chs.Log.Info("[CHAINED HOTSTUFF] Prepare msg not match")
			return
		}
		// get prepare msg
		prepare := msg.GetPrepare()

		if !chs.SafeNode(prepare.CurProposal, prepare.CurProposal.Justify) {
			chs.Log.Warn("[CHAINED HOTSTUFF] Unsafe node")
			return
		}
		// add view number and change leader
		chs.View.ViewNum++
		chs.View.Primary = chs.GetLeader()

		marshal, _ := proto.Marshal(msg)
		chs.CurExec.DocumentHash, _ = crypto.CreateDocumentHash(marshal, chs.Config.Keys.PublicKey)
		chs.CurExec.Node = prepare.CurProposal
		partSig, _ := crypto.TSign(chs.CurExec.DocumentHash, chs.Config.Keys.PrivateKey, chs.Config.Keys.PublicKey)

		if chs.View.Primary == chs.ID {
			// vote self
			chs.CurExec.PrepareVote = append(chs.CurExec.PrepareVote, partSig)
		} else {
			// send vote msg to the next leader
			partSigBytes, _ := json.Marshal(partSig)
			voteMsg := chs.VoteMsg(pb.MsgType_PREPARE_VOTE, prepare.CurProposal, nil, partSigBytes)
			chs.Unicast(chs.GetNetworkInfo()[chs.GetLeader()], voteMsg)
			chs.CurExec = hotstuff.NewCurProposal()
		}
		chs.update(prepare.CurProposal)
	case *pb.Msg_PrepareVote:
		chs.Log.Info("[CHAINED HOTSTUFF] Get generic vote msg")
		if !chs.MatchingMsg(msg, pb.MsgType_PREPARE_VOTE) {
			chs.Log.Warn("[CHAINED HOTSTUFF] Msg not match")
			return
		}
		prepareVote := msg.GetPrepareVote()
		chs.Log.Debug("get preparevote")
		partSig := new(tcrsa.SigShare)
		chs.Log.Debug("verify partial sig")
		_ = json.Unmarshal(prepareVote.PartialSig, partSig)
		if err := crypto.VerifyPartSig(partSig, chs.CurExec.DocumentHash, chs.Config.Keys.PublicKey); err != nil {
			chs.Log.Warn("[CHAINED HOTSTUFF GENERIC-VOTE] Partial signature is not correct")
			return
		}
		chs.Log.Debug("got generic vote: ", len(chs.CurExec.PrepareVote))
		chs.CurExec.PrepareVote = append(chs.CurExec.PrepareVote, partSig)
		if len(chs.CurExec.PrepareVote) == chs.Config.F*2+1 {
			signature, _ := crypto.CreateFullSignature(chs.CurExec.DocumentHash, chs.CurExec.PrepareVote, chs.Config.Keys.PublicKey)
			qc := chs.QC(pb.MsgType_PREPARE_VOTE, signature, prepareVote.BlockHash)
			chs.genericQC = qc
			chs.CurExec = hotstuff.NewCurProposal()
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
		return
	}
	// block = b*, block1 = b'', block2 = b', block3 = b
	block1, err := chs.BlockStorage.BlockOf(block.Justify)
	if err == leveldb.ErrNotFound || block1.Committed {
		return
	}
	// start pre-commit phase on block’s parent
	if bytes.Equal(block.ParentHash, block1.Hash) {
		chs.Log.Infof("[CHAINED HOTSTUFF] Start pre-commit phase on block %s", hex.EncodeToString(block1.Hash))
		chs.genericQC = block.Justify
	}

	block2, err := chs.BlockStorage.BlockOf(block1.Justify)
	if err == leveldb.ErrNotFound || block2.Committed {
		return
	}
	// start commit phase on block1’s parent
	if bytes.Equal(block.ParentHash, block1.Hash) && bytes.Equal(block1.ParentHash, block2.Hash) {
		chs.Log.Infof("[CHAINED HOTSTUFF] Start commit phase on block %s", hex.EncodeToString(block2.Hash))
		chs.lockQC = block1.Justify
	}

	block3, err := chs.BlockStorage.BlockOf(block2.Justify)
	if err == leveldb.ErrNotFound || block3.Committed {
		return
	}

	if bytes.Equal(block.ParentHash, block1.Hash) && bytes.Equal(block1.ParentHash, block2.Hash) &&
		bytes.Equal(block2.ParentHash, block3.Hash) {
		//decide
		chs.Log.Infof("[CHAINED HOTSTUFF] Start decide phase on block %s", hex.EncodeToString(block3.Hash))
		chs.processProposal()
	}
}

func (chs *ChainedHotStuff) batchEvent(txs []types.RawTransaction) {
	// if batch timeout, check size
	if len(txs) == 0 {
		chs.BatchTimeChan.SoftStartTimer()
		return
	}
	// create prepare msg
	node := chs.CreateLeaf(chs.BlockStorage.GetLastBlockHash(), txs, chs.genericQC)
	chs.CurExec.Node = node
	chs.MemPool.MarkProposed(txs)
	if chs.HighQC == nil {
		chs.HighQC = chs.genericQC
	}
	prepareMsg := chs.Msg(pb.MsgType_PREPARE, node, chs.HighQC)
	// broadcast prepare msg
	chs.Broadcast(prepareMsg)
	chs.TimeChan.SoftStartTimer()
}

func (chs *ChainedHotStuff) processProposal() {
	// process proposal
	go chs.ProcessProposal(chs.CurExec.Node, []byte{})
	// store block
	chs.CurExec.Node.Committed = true
}

func (chs *ChainedHotStuff) VerifyBlock(block []byte, proof []byte) bool {
	return true
}
