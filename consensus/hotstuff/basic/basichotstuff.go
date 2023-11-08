package basic

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/niclabs/tcrsa"
	"github.com/sirupsen/logrus"
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

type BasicHotStuff struct {
	hotstuff.HotStuffImpl
	// sometimes, a new view msg will be processed before decide msg.
	// it may cause a bug. The parameter 'decided' is used to avoid it to happen temporarily.
	// TODO: find a better way to fix the bug.
	decided bool
}

func NewBasicHotStuff(
	id int64,
	cid int64,
	cfg *config.ConsensusConfig,
	exec executor.Executor,
	p2pAdaptor p2p.P2PAdaptor,
	log *logrus.Entry,
) *BasicHotStuff {
	log.WithField("cid", cid).Info("[BASICHOTSTUFF] starting")
	bhs := &BasicHotStuff{}
	bhs.Init(id, cid, cfg, exec, p2pAdaptor, log)
	bhs.View = hotstuff.NewView(1, 1)
	bhs.Log.Trace("[HOTSTUFF] Generate genesis block")
	genesisBlock := hotstuff.GenerateGenesisBlock()
	err := bhs.BlockStorage.Put(genesisBlock)
	if err != nil {
		bhs.Log.Fatal("generate genesis block failed")
	}
	bhs.PrepareQC = &pb.QuorumCert{
		BlockHash: genesisBlock.Hash,
		Type:      pb.MsgType_PREPARE_VOTE,
		ViewNum:   0,
		Signature: nil,
	}
	bhs.PreCommitQC = &pb.QuorumCert{
		BlockHash: genesisBlock.Hash,
		Type:      pb.MsgType_PRECOMMIT_VOTE,
		ViewNum:   0,
		Signature: nil,
	}
	bhs.CommitQC = &pb.QuorumCert{
		BlockHash: genesisBlock.Hash,
		Type:      pb.MsgType_COMMIT_VOTE,
		ViewNum:   0,
		Signature: nil,
	}
	bhs.Log.Tracef("[HOTSTUFF] Init command set, replica id: %d", id)

	// init timer and stop it
	bhs.TimeChan = utils.NewTimer(time.Duration(bhs.Config.HotStuff.Timeout) * time.Second)
	bhs.TimeChan.Init()

	bhs.BatchTimeChan = utils.NewTimer(time.Duration(bhs.Config.HotStuff.BatchTimeout) * time.Second)
	bhs.BatchTimeChan.Init()

	bhs.CurExec = &hotstuff.CurProposal{
		Node:          nil,
		DocumentHash:  nil,
		PrepareVote:   make([]*tcrsa.SigShare, 0),
		PreCommitVote: make([]*tcrsa.SigShare, 0),
		CommitVote:    make([]*tcrsa.SigShare, 0),
		HighQC:        make([]*pb.QuorumCert, 0),
	}
	bhs.decided = false
	go bhs.receiveMsg()
	return bhs
}

// receiveMsg receive msg from msg channel
func (bhs *BasicHotStuff) receiveMsg() {
	bhs.Wg.Add(1)
	for {
		select {
		case msgByte := <-bhs.MsgByteEntrance:
			// if !ok {
			// 	return // closed
			// }
			msg, err := bhs.DecodeMsgByte(msgByte)
			if err != nil {
				bhs.Log.WithError(err).Warn("decode message failed")
				continue
			}
			bhs.handleMsg(msg)

		case request, ok := <-bhs.RequestEntrance:
			if !ok {
				return // closed
			}
			bhs.Log.Trace("received request")
			bhs.handleMsg(&pb.Msg{Payload: &pb.Msg_Request{Request: request}})
		case <-bhs.TimeChan.Timeout():
			bhs.Log.WithField("view", bhs.View.ViewNum).Warn("Time out, goto new view")
			// set the duration of the timeout to 2 times
			bhs.TimeChan = utils.NewTimer(time.Duration(bhs.Config.HotStuff.Timeout) * time.Second * 2)
			bhs.TimeChan.Init()
			if bhs.CurExec.Node != nil {
				bhs.MemPool.UnMark(types.RawTxArrayFromBytes(bhs.CurExec.Node.Txs))
				bhs.BlockStorage.Put(bhs.CreateLeaf(bhs.CurExec.Node.ParentHash, nil, nil))
			}
			bhs.View.ViewNum++
			bhs.View.Primary = bhs.GetLeader()
			// check if self is the next leader
			if bhs.GetLeader() != bhs.ID {
				// if not, send next view mag to the next leader
				newViewMsg := bhs.Msg(pb.MsgType_NEWVIEW, nil, bhs.PrepareQC)
				bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], newViewMsg)
				// clear curExec
				bhs.CurExec = hotstuff.NewCurProposal()
			} else {
				bhs.decided = true
			}
		case <-bhs.BatchTimeChan.Timeout():
			bhs.BatchTimeChan.Init()
			bhs.batchEvent(bhs.MemPool.GetFirst(int(bhs.Config.HotStuff.BatchSize)))
		case <-bhs.Closed:
			bhs.Wg.Done()
			return
		}
	}
}

// handleMsg handle different msg with different way
func (bhs *BasicHotStuff) handleMsg(msg *pb.Msg) {
	switch msg.Payload.(type) {
	case *pb.Msg_NewView:
		bhs.Log.Trace("[HOTSTUFF NEWVIEW] Got new view msg")
		// process highqc and node
		bhs.CurExec.HighQC = append(bhs.CurExec.HighQC, msg.GetNewView().PrepareQC)
		if bhs.decided {
			if len(bhs.CurExec.HighQC) >= 2*bhs.Config.F {
				bhs.View.ViewChanging = true
				bhs.HighQC = bhs.PrepareQC
				for _, qc := range bhs.CurExec.HighQC {
					if qc.ViewNum > bhs.HighQC.ViewNum {
						bhs.HighQC = qc
					}
				}
				// TODO sync blocks if fall behind
				bhs.CurExec = hotstuff.NewCurProposal()
				bhs.View.ViewChanging = false
				bhs.BatchTimeChan.SoftStartTimer()
				bhs.decided = false
			}
		}
	case *pb.Msg_Prepare:
		bhs.Log.Trace("[HOTSTUFF PREPARE] Got prepare msg")
		if !bhs.MatchingMsg(msg, pb.MsgType_PREPARE) {
			bhs.Log.Warn("[HOTSTUFF PREPARE] msg does not match")
			return
		}
		prepare := msg.GetPrepare()
		if !bytes.Equal(prepare.CurProposal.ParentHash, prepare.HighQC.BlockHash) ||
			!bhs.SafeNode(prepare.CurProposal, prepare.HighQC) {
			bhs.Log.Warn("[HOTSTUFF PREPARE] node is not correct")
			return
		}
		// create prepare vote msg
		marshal, _ := proto.Marshal(msg)
		bhs.CurExec.DocumentHash, _ = crypto.CreateDocumentHash(marshal, bhs.Config.Keys.PublicKey)
		bhs.CurExec.Node = prepare.CurProposal
		partSig, _ := crypto.TSign(bhs.CurExec.DocumentHash, bhs.Config.Keys.PrivateKey, bhs.Config.Keys.PublicKey)
		partSigBytes, _ := json.Marshal(partSig)
		prepareVoteMsg := bhs.VoteMsg(pb.MsgType_PREPARE_VOTE, bhs.CurExec.Node, nil, partSigBytes)
		// send msg to leader
		// bhs.Log.Debug("send prepareVote")
		bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], prepareVoteMsg)
		bhs.TimeChan.SoftStartTimer()
	case *pb.Msg_PrepareVote:
		bhs.Log.Trace("[HOTSTUFF PREPARE-VOTE] Got prepare vote msg")
		if !bhs.MatchingMsg(msg, pb.MsgType_PREPARE_VOTE) {
			bhs.Log.Warn("[HOTSTUFF PREPARE-VOTE] Msg not match")
			return
		}
		// verify
		prepareVote := msg.GetPrepareVote()
		partSig := new(tcrsa.SigShare)
		_ = json.Unmarshal(prepareVote.PartialSig, partSig)
		if err := crypto.VerifyPartSig(partSig, bhs.CurExec.DocumentHash, bhs.Config.Keys.PublicKey); err != nil {
			bhs.Log.Warn("[HOTSTUFF PREPARE-VOTE] Partial signature is not correct")
			return
		}
		// put it into preparevote slice
		bhs.CurExec.PrepareVote = append(bhs.CurExec.PrepareVote, partSig)
		bhs.Log.WithField("c", len(bhs.CurExec.PrepareVote)).WithField("F", bhs.Config.F).Trace("[HOTSTUFF PREPARE-VOTE] count prepare vote msg")
		if len(bhs.CurExec.PrepareVote) == bhs.Config.F*2+1 {
			// create full signature
			bhs.Log.Trace("[HOTSTUFF PRECOMMIT-VOTE] sending precommit")
			signature, _ := crypto.CreateFullSignature(bhs.CurExec.DocumentHash, bhs.CurExec.PrepareVote, bhs.Config.Keys.PublicKey)
			qc := bhs.QC(pb.MsgType_PREPARE_VOTE, signature, prepareVote.BlockHash)
			bhs.PrepareQC = qc
			preCommitMsg := bhs.Msg(pb.MsgType_PRECOMMIT, bhs.CurExec.Node, qc)
			// broadcast msg
			bhs.Broadcast(preCommitMsg)
			bhs.TimeChan.SoftStartTimer()
		}
	case *pb.Msg_PreCommit:
		bhs.Log.Trace("[HOTSTUFF PRECOMMIT] Got precommit msg")
		if !bhs.MatchingQC(msg.GetPreCommit().PrepareQC, pb.MsgType_PREPARE_VOTE) {
			bhs.Log.Warn("[HOTSTUFF PRECOMMIT] QC not match")
			return
		}
		bhs.PrepareQC = msg.GetPreCommit().PrepareQC
		partSig, _ := crypto.TSign(bhs.CurExec.DocumentHash, bhs.Config.Keys.PrivateKey, bhs.Config.Keys.PublicKey)
		partSigBytes, _ := json.Marshal(partSig)
		preCommitVote := bhs.VoteMsg(pb.MsgType_PRECOMMIT_VOTE, bhs.CurExec.Node, nil, partSigBytes)
		bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], preCommitVote)
		bhs.TimeChan.SoftStartTimer()
	case *pb.Msg_PreCommitVote:
		bhs.Log.Trace("[HOTSTUFF PRECOMMIT-VOTE] Got precommit vote msg")
		if !bhs.MatchingMsg(msg, pb.MsgType_PRECOMMIT_VOTE) {
			bhs.Log.Warn("[HOTSTUFF PRECOMMIT-VOTE] Msg not match")
			return
		}
		// verify
		preCommitVote := msg.GetPreCommitVote()
		partSig := new(tcrsa.SigShare)
		_ = json.Unmarshal(preCommitVote.PartialSig, partSig)
		if err := crypto.VerifyPartSig(partSig, bhs.CurExec.DocumentHash, bhs.Config.Keys.PublicKey); err != nil {
			bhs.Log.Warn("[HOTSTUFF PRECOMMIT-VOTE] Partial signature is not correct")
			return
		}
		bhs.CurExec.PreCommitVote = append(bhs.CurExec.PreCommitVote, partSig)
		if len(bhs.CurExec.PreCommitVote) == 2*bhs.Config.F+1 {
			bhs.Log.Trace("[HOTSTUFF PRECOMMIT-VOTE] sending commit")
			signature, _ := crypto.CreateFullSignature(bhs.CurExec.DocumentHash, bhs.CurExec.PreCommitVote, bhs.Config.Keys.PublicKey)
			preCommitQC := bhs.QC(pb.MsgType_PRECOMMIT_VOTE, signature, bhs.CurExec.Node.Hash)
			// vote self
			bhs.PreCommitQC = preCommitQC
			commitMsg := bhs.Msg(pb.MsgType_COMMIT, bhs.CurExec.Node, preCommitQC)
			bhs.Broadcast(commitMsg)
			bhs.TimeChan.SoftStartTimer()
		}
	case *pb.Msg_Commit:
		bhs.Log.Trace("[HOTSTUFF COMMIT] Got commit msg")
		commit := msg.GetCommit()
		if !bhs.MatchingQC(commit.PreCommitQC, pb.MsgType_PRECOMMIT_VOTE) {
			bhs.Log.Warn("[HOTSTUFF COMMIT] QC not match")
			return
		}
		bhs.PreCommitQC = commit.PreCommitQC
		partSig, _ := crypto.TSign(bhs.CurExec.DocumentHash, bhs.Config.Keys.PrivateKey, bhs.Config.Keys.PublicKey)
		partSigBytes, _ := json.Marshal(partSig)
		commitVoteMsg := bhs.VoteMsg(pb.MsgType_COMMIT_VOTE, bhs.CurExec.Node, nil, partSigBytes)
		bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], commitVoteMsg)
		bhs.TimeChan.SoftStartTimer()
	case *pb.Msg_CommitVote:
		bhs.Log.Trace("[HOTSTUFF COMMIT-VOTE] Got commit vote msg")
		if !bhs.MatchingMsg(msg, pb.MsgType_COMMIT_VOTE) {
			bhs.Log.Warn("[HOTSTUFF COMMIT-VOTE] Msg not match")
			return
		}
		commitVoteMsg := msg.GetCommitVote()
		partSig := new(tcrsa.SigShare)
		_ = json.Unmarshal(commitVoteMsg.PartialSig, partSig)
		if err := crypto.VerifyPartSig(partSig, bhs.CurExec.DocumentHash, bhs.Config.Keys.PublicKey); err != nil {
			bhs.Log.Warn("[HOTSTUFF COMMIT-VOTE] Partial signature is not correct")
			return
		}
		bhs.CurExec.CommitVote = append(bhs.CurExec.CommitVote, partSig)
		if len(bhs.CurExec.CommitVote) == 2*bhs.Config.F+1 {
			// bhs.Log.Debug("[HOTSTUFF PRECOMMIT-VOTE] sending decide")
			signature, _ := crypto.CreateFullSignature(bhs.CurExec.DocumentHash, bhs.CurExec.CommitVote, bhs.Config.Keys.PublicKey)
			commitQC := bhs.QC(pb.MsgType_COMMIT_VOTE, signature, bhs.CurExec.Node.Hash)
			// vote self
			bhs.CommitQC = commitQC
			decideMsg := bhs.Msg(pb.MsgType_DECIDE, bhs.CurExec.Node, commitQC)
			bhs.Broadcast(decideMsg)
			bhs.TimeChan.Stop()
			bhs.processProposal()
		}
	case *pb.Msg_Decide:
		decideMsg := msg.GetDecide()
		if decideMsg.ViewNum < bhs.View.ViewNum {
			// bhs.Log.Debug("[HOTSTUFF DECIDE] old msg")
			// already decided
			return
		}
		bhs.Log.WithField("view", bhs.View.ViewNum).Trace("[HOTSTUFF DECIDE] Got decide msg")
		if !bhs.MatchingQC(decideMsg.CommitQC, pb.MsgType_COMMIT_VOTE) {
			bhs.Log.Warn("[HOTSTUFF DECIDE] QC not match")
			return
		}
		bhs.CommitQC = decideMsg.CommitQC
		// broadcast proof
		bhs.Log.Trace("[HOTSTUFF DECIDE] broadcast proof")
		bhs.Broadcast(msg)
		bhs.Log.WithField("view", bhs.View.ViewNum).WithField("timeout", bhs.Config.HotStuff.Timeout).Trace("[HOTSTUFF DECIDE] stop timer")
		bhs.TimeChan.Stop()
		bhs.processProposal()
	case *pb.Msg_Request:
		request := msg.GetRequest()
		bhs.Log.Tracef("[HOTSTUFF] Got request msg, content:%s", request.String())
		// put the cmd into the cmdset
		bhs.MemPool.Add(types.RawTransaction(request.Tx))
		// send request to the leader, if the replica is not the leader
		if bhs.ID != bhs.GetLeader() {
			bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], msg)
			return
		}
		if bhs.CurExec.Node != nil || bhs.View.ViewChanging {
			return
		}
		// start batch timer
		bhs.BatchTimeChan.SoftStartTimer()
		// if the length of unprocessed cmd equals to batch size, stop timer and call handleMsg to send prepare msg
		// bhs.Log.Debugf("cmd set size: %d", len(bhs.MemPool.GetFirst(int(bhs.Config.HotStuff.BatchSize))))
		txs := bhs.MemPool.GetFirst(int(bhs.Config.HotStuff.BatchSize))
		if len(txs) == int(bhs.Config.HotStuff.BatchSize) {
			// stop timer
			bhs.BatchTimeChan.Stop()
			// create prepare msg
			bhs.batchEvent(txs)
		}
	default:
		bhs.Log.Warn("Unsupported msg type, drop it.")
	}
}

func (bhs *BasicHotStuff) processProposal() {
	// process proposal
	bhs.Log.WithField("view", bhs.View.ViewNum).Info("[BASICHOTSTUFF] process proposal")
	// store block
	bhs.CurExec.Node.Committed = true
	bhs.BlockStorage.Put(bhs.CurExec.Node)
	go bhs.ProcessProposal(bhs.CurExec.Node, []byte{})
	// add view number
	bhs.View.ViewNum++
	bhs.View.Primary = bhs.GetLeader()
	// check if self is the next leader
	if bhs.View.Primary != bhs.ID {
		// if not, send next view mag to the next leader
		newViewMsg := bhs.Msg(pb.MsgType_NEWVIEW, nil, bhs.PrepareQC)
		bhs.Unicast(bhs.GetNetworkInfo()[bhs.GetLeader()], newViewMsg)
		// clear curExec
		bhs.CurExec = hotstuff.NewCurProposal()
	} else {
		bhs.decided = true
	}
}

func (bhs *BasicHotStuff) batchEvent(txs []types.RawTransaction) {
	if len(txs) == 0 {
		bhs.BatchTimeChan.SoftStartTimer()
		return
	}
	// create prepare msg
	node := bhs.CreateLeaf(bhs.BlockStorage.GetLastBlockHash(), txs, nil)
	bhs.CurExec.Node = node
	bhs.MemPool.MarkProposed(txs)
	if bhs.HighQC == nil {
		bhs.HighQC = bhs.PrepareQC
	}
	prepareMsg := bhs.Msg(pb.MsgType_PREPARE, node, bhs.HighQC)
	// vote self
	marshal, _ := proto.Marshal(prepareMsg)
	bhs.CurExec.DocumentHash, _ = crypto.CreateDocumentHash(marshal, bhs.Config.Keys.PublicKey)
	partSig, _ := crypto.TSign(bhs.CurExec.DocumentHash, bhs.Config.Keys.PrivateKey, bhs.Config.Keys.PublicKey)
	bhs.CurExec.PrepareVote = append(bhs.CurExec.PrepareVote, partSig)
	// broadcast prepare msg
	// bhs.Log.Debug("broadcast prepare")
	bhs.Broadcast(prepareMsg)
	bhs.TimeChan.SoftStartTimer()
}

func (bhs *BasicHotStuff) VerifyBlock(block []byte, proof []byte) bool {
	return true
}
