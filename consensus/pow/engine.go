package pow

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/p2p"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

// this is demo pow, do not use in production

type Engine struct {
	id          int64
	consensusID int64

	memPool         *types.MemPool
	log             *logrus.Entry
	config          *config.ConsensusConfig
	Adaptor         p2p.P2PAdaptor
	exec            executor.Executor
	msgByteEntrance chan []byte
	exitChan        chan struct{}
	requestEntrance chan *pb.Request
	blockChan       chan *pb.PoWBlock
	storage         *types.Storage
	remotePending   *atomic.Int64
	hasher          string
	wg              *sync.WaitGroup
}

func (e *Engine) GetRequestEntrance() chan<- *pb.Request {
	return e.requestEntrance
}

func (e *Engine) GetMsgByteEntrance() chan<- []byte {
	return e.msgByteEntrance
}

func (e *Engine) Stop() {
	close(e.exitChan)
	e.wg.Wait()
	e.log.Info("pow stopped")
}

func (e *Engine) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (e *Engine) GetWeight(nid int64) float64 {
	return 0.25 // to be updated
}

func (e *Engine) GetMaxAdversaryWeight() float64 {
	return 1.0 / 3
}

func (e *Engine) UpdateExternalStatus(status model.ExternalStatus) {
	return
}

func (e *Engine) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {
	return
}

func (e *Engine) RequestLatestBlock(epoch int64, proof []byte, committee []string) {
	return
}

func NewPowEngine(nid int64, cid int64, config *config.ConsensusConfig, exec executor.Executor, adaptor p2p.P2PAdaptor, log *logrus.Entry) *Engine {
	// make sure the hash function is available
	// _, err := crypto.HashFactory(config.Pow.Hash)
	// if err != nil {
	// 	log.Error("unable to find hash function")
	// 	return nil
	// }
	e := &Engine{
		id:              nid,
		consensusID:     cid,
		exec:            exec,
		memPool:         types.NewMemPool(),
		log:             log,
		Adaptor:         adaptor,
		config:          config,
		msgByteEntrance: make(chan []byte, 100),
		requestEntrance: make(chan *pb.Request, 10),
		blockChan:       make(chan *pb.PoWBlock, 10),
		storage:         types.NewStorage(fmt.Sprintf("node-%d-%d", cid, nid), config.Pow.Hash),
		remotePending:   new(atomic.Int64),
		hasher:          config.Pow.Hash,
		wg:              new(sync.WaitGroup),
	}
	// rand.Seed(0)
	// adaptor.SetReceiver(e)
	adaptor.SetReceiver(e.GetMsgByteEntrance())
	adaptor.Subscribe([]byte(config.Topic))
	go e.worker()
	go e.receiveMsg()
	e.blockChan <- e.createGenesisBlock()
	e.log.Info("pow started")
	return e
}

func (e *Engine) createGenesisBlock() *pb.PoWBlock {
	genesis := &pb.PoWBlock{
		ParentHash: []byte{},
		Height:     0,
		Txs:        [][]byte{},
		Nonce:      0,
		Commited:   false,
	}
	e.storage.Put(genesis)
	e.log.WithField("hash", genesis.Identifier(e.hasher)).Trace("create genensis block")
	return genesis
}

func (e *Engine) GetConsensusID() int64 {
	return e.consensusID
}

func (e *Engine) GetMinDifficulty(block *pb.PoWBlock) *big.Int {
	difficulty := new(big.Int)
	difficulty.SetInt64(int64(block.Height))
	return difficulty.Sub(e.config.Pow.InitDifficulty, difficulty)
}

func (e *Engine) verifyBlock(block *pb.PoWBlock) bool {
	// e.log.WithFields(logrus.Fields{
	// 	"diff":    block.GetDifficulty(),
	// 	"mindiff": e.GetMinDifficulty(block),
	// 	"block":   block.Identifier(),
	// }).Trace("verify block difficulty")
	return block.GetDifficulty(e.hasher).Cmp(e.GetMinDifficulty(block)) < 0
}

func (e *Engine) lterateBlock(coming, now, pending *pb.PoWBlock) (*pb.PoWBlock, *pb.PoWBlock) {
	// e.log.Trace("iterate block")
	build := false
	if now == nil {
		now = coming
		build = true
	} else {
		// TODO: Mark history blocks of coming
		// TODO: UnMark history block of now
		if now.Height < coming.Height {
			now = coming
			build = true
		} else if now.Height == coming.Height && now.GetDifficulty(e.hasher).Cmp(coming.GetDifficulty(e.hasher)) > 0 {
			e.memPool.UnMark(types.RawTxArrayFromBytes(now.Txs))
			now = coming
			build = true
		}
	}
	if build {
		e.log.WithFields(logrus.Fields{
			"hash":   now.Identifier(e.hasher),
			"height": now.Height,
		}).Trace("update head block")
		e.memPool.MarkProposed(types.RawTxArrayFromBytes(now.Txs))
		pending = e.buildPendingBlock(now)
		go e.commitAncient(now)
	}
	return now, pending
}

func (e *Engine) buildPendingBlock(parent *pb.PoWBlock) *pb.PoWBlock {
	txs := e.memPool.GetFirst(10)
	return &pb.PoWBlock{
		ParentHash: parent.Hash(e.hasher),
		Height:     parent.Height + 1,
		Txs:        types.RawTxArrayToBytes(txs),
		Nonce:      0,
		BlockHash:  nil,
	}
}

func (e *Engine) worker() {
	e.wg.Add(1)
	var nowBlock *pb.PoWBlock = nil
	var pendingBlock *pb.PoWBlock = nil
	for {
		select {
		case block := <-e.blockChan:
			nowBlock, pendingBlock = e.lterateBlock(block, nowBlock, pendingBlock)
			for e.remotePending.Load() > 0 {
				block = <-e.blockChan
				nowBlock, pendingBlock = e.lterateBlock(block, nowBlock, pendingBlock)
				e.remotePending.Add(-1)
			}
			e.mine(nowBlock, pendingBlock, e.GetMinDifficulty(pendingBlock))
		case <-e.exitChan:
			e.wg.Done()
			return
		}
	}
}

func (e *Engine) commitAncient(newest *pb.PoWBlock) {
	p := newest
	for i := 0; i < 6; i++ {
		if len(p.ParentHash) == 0 { // genesis
			return
		}
		b, err := e.storage.Get(p.ParentHash)
		if err != nil {
			e.log.WithError(err).Warn("find parent block failed")
			return
		}
		block := new(pb.PoWBlock)
		if err := proto.Unmarshal(b, block); err != nil {
			e.log.WithError(err).Warn("decode block failed")
			return
		}
		p = block
	}
	e.commitRecursive(p)
}

func (e *Engine) commitRecursive(block *pb.PoWBlock) {
	if block.Commited {
		return
	}
	if len(block.ParentHash) != 0 {
		parent := new(pb.PoWBlock)
		b, err := e.storage.Get(block.ParentHash)
		if err != nil {
			e.log.WithError(err).Warn("find parent block failed")
			return
		}
		if err := proto.Unmarshal(b, parent); err != nil {
			e.log.WithError(err).Warn("decode block failed")
			return
		}
		e.commitRecursive(parent)
	}
	block.Commited = true
	e.memPool.Remove(types.RawTxArrayFromBytes(block.Txs))
	e.storage.Put(block)
	e.exec.CommitBlock(block, []byte{}, e.consensusID)
	e.log.WithFields(logrus.Fields{
		"height":  block.Height,
		"hash":    block.Identifier(e.hasher),
		"txcount": len(block.GetTxs()),
	}).Info("commit block")
}

func (e *Engine) mine(parent *pb.PoWBlock, pending *pb.PoWBlock, target *big.Int) {
	pending.Nonce = rand.Uint64()
	for i := 0; i < 100; i++ {
		pending.Nonce += 1
		pending.ForceHash(e.hasher)
		// e.log.WithFields(logrus.Fields{
		// 	"difficulty": pending.GetDifficulty().String(),
		// 	"target":     target.String(),
		// }).Trace("block difficulty")
		if pending.GetDifficulty(e.hasher).Cmp(target) < 0 {
			e.log.WithFields(logrus.Fields{
				"height":     pending.Height,
				"hash":       pending.Identifier(e.hasher),
				"difficulty": pending.GetDifficulty(e.hasher),
				// "loc":        fmt.Sprintf("%p", pending),
				"txcount": len(pending.GetTxs()),
				// "block":      pending,
			}).Info("mined block")
			e.blockChan <- pending
			go e.broadcastBlock(pending)
			return
		}
		// time.Sleep(time.Millisecond)
	}
	e.blockChan <- parent
}

func (e *Engine) broadcastBlock(block *pb.PoWBlock) error {
	e.log.WithFields(logrus.Fields{
		"hash":   block.Identifier(e.hasher),
		"height": block.Height,
		// "loc":    fmt.Sprintf("%p", block),
		// "block":  block,
	}).Trace("broadcast block")
	msg := &pb.PoWMessage{
		Msg: &pb.PoWMessage_Block{
			Block: block,
		},
	}
	byteMsg, err := proto.Marshal(msg)
	if err != nil {
		e.log.WithError(err).Warn("encode block failed")
		return err
	}
	return e.Adaptor.Broadcast(byteMsg, e.consensusID, []byte("consensus"))
}

func (e *Engine) receiveMsg() {
	e.wg.Add(1)
	for {
		select {
		case msgByte := <-e.msgByteEntrance:
			msg := new(pb.PoWMessage)
			err := proto.Unmarshal(msgByte, msg)
			if err != nil {
				e.log.WithError(err).Warn("decode message failed")
				continue
			}
			switch msg.Msg.(type) {
			case *pb.PoWMessage_Request:
				e.requestEntrance <- msg.GetRequest()
			case *pb.PoWMessage_Block:
				go e.receiveBlock(msg.GetBlock())
			}
		case request := <-e.requestEntrance:
			if !e.memPool.Has(types.RawTransaction(request.Tx)) {
				e.log.Trace("received new tx")
				e.memPool.Add(types.RawTransaction(request.Tx))
				msg := &pb.PoWMessage{
					Msg: &pb.PoWMessage_Request{
						Request: request,
					},
				}
				msgByte, err := proto.Marshal(msg)
				if err != nil {
					e.log.WithError(err).Warn("encode message failed")
					continue
				}
				// e.log.Trace("broadcast tx")
				e.Adaptor.Broadcast(msgByte, e.consensusID, []byte("consensus"))
				// e.log.Trace("broadcast tx done")
			}
		case <-e.exitChan:
			e.wg.Done()
			return
		}
	}
}

func (e *Engine) verifyRemoteBlock(block *pb.PoWBlock) bool {
	block.ForceHash(e.hasher)
	return e.verifyBlock(block)
}

func (e *Engine) receiveBlock(block *pb.PoWBlock) {
	if !e.verifyRemoteBlock(block) {
		e.log.WithField("hash", block.Identifier(e.hasher)).Warn("verify remote block failed")
		return
	}
	block.Commited = false
	e.storage.PutIfNotExists(block)
	e.log.WithFields(logrus.Fields{
		"hash":   block.Identifier(e.hasher),
		"height": block.Height,
	}).Trace("received new block")
	e.remotePending.Add(1)
	e.blockChan <- block
}
