// Package pot implements the Proof of Time (PoT) consensus algorithm worker.
// This package provides the core functionality for mining, block creation,
// and VDF (Verifiable Delay Function) computation in the PoT blockchain system.
package pot

import (
	"blockchain-crypto/vdf"
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	storage "github.com/zzz136454872/upgradeable-consensus/internal/storage/pot"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"net"
	"os"
	"sync"
	"time"
)

// bigD represents the maximum difficulty value (2^256 - 1) used in mining calculations.
var bigD = new(big.Int).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))

const (
	// Commiteelen defines the length of committee members
	Commiteelen = 4
	// CommiteeDelay specifies the delay time for committee operations
	CommiteeDelay = 1
	// cpuCounter defines the number of CPU workers for parallel mining
	cpuCounter = 1
	// NoParentD is the default difficulty when no parent block exists
	NoParentD = 2
	// Batchsize defines the batch size for transaction processing
	Batchsize = 100
	// Selectn specifies the number of selections in reward distribution
	Selectn = 1
	// TotalReward is the total reward amount for mining
	TotalReward = 65536
	// BackupCommiteeSize defines the size of backup committee
	BackupCommiteeSize = 64
	// ConfirmDelay specifies the confirmation delay in blocks
	ConfirmDelay = 6
	// CandidateKeyLen defines the length of candidate public key
	CandidateKeyLen = 32
	// Commitees defines the number of committee members
	Commitees = 4
)

// Worker represents a PoT consensus worker that handles mining, block creation,
// and VDF computations. It manages the entire lifecycle of the PoT consensus process
// including block mining, committee management, and network communication.
type Worker struct {
	// basic info - core identification and configuration
	ID     int64                   // unique identifier for this worker node
	PeerId string                  // peer ID for network identification
	log    *logrus.Entry           // structured logger for this worker
	config *config.ConsensusConfig // consensus configuration parameters
	epoch  uint64                  // current epoch number

	// vdf work - VDF computation management
	timestamp time.Time // timestamp for VDF computation tracking

	vdf0        *types.VDF          // VDF0 instance for epoch progression
	vdf0Chan    chan *types.VDF0res // channel for VDF0 results
	vdf1        []*types.VDF        // VDF1 instances for mining (parallel workers)
	vdf1Chan    chan *types.VDF0res // channel for VDF1 results
	vdfhalf     *types.VDF          // VDF half computation instance
	vdfhalfchan chan *types.VDF0res // channel for VDF half results

	vdfChecker      *vdf.Vdf                        // VDF verification instance
	abort           *Abortcontrol                   // control structure for aborting operations
	wg              *sync.WaitGroup                 // wait group for goroutine synchronization
	workFlag        bool                            // flag indicating if worker is currently mining
	blockKeyMap     map[[crypto.Hashlen]byte][]byte // mapping of block hashes to private keys
	CommitteeKeyMap map[[crypto.Hashlen]byte][]byte // mapping for committee keys
	executeheight   uint64                          // height of last executed block
	incentiveheight uint64                          // height of last processed incentive
	mempool         *Mempool                        // transaction mempool
	chainresetflag  bool                            // flag for chain reset operations

	// rand seed - randomization and counting
	rand         *rand.Rand // random number generator
	blockCounter int        // counter for created blocks
	keyseed      []byte     // seed for key generation

	Engine  *PoTEngine    // reference to the PoT engine
	mutex   *sync.Mutex   // mutex for thread-safe operations
	rwmutex *sync.RWMutex // read-write mutex for concurrent access

	// communication - network and RPC interfaces
	peerMsgQueue      chan *types.Block      // channel for incoming peer messages
	blockResponseChan chan *pb.BlockResponse // channel for block responses
	potResponseCh     chan *pb.PoTResponse   // channel for PoT responses
	blockStorage      *storage.BlockStorage  // persistent block storage
	chainReader       *ChainReader           // chain state reader
	listener          net.Listener           // network listener for RPC
	rpcserver         *grpc.Server           // gRPC server instance
	httpserver        *gin.Engine            // HTTP server for web interface

	// upper consensus - integration with higher-level consensus
	whirly         *nodeController.NodeController // whirly consensus controller
	potSignalChan  chan<- []byte                  // channel for signaling to upper layer
	CommiteeNum    int32                          // current committee number
	Commitee       [][]string                     // committee member lists
	Shardings      []Sharding                     // sharding configuration
	BackupCommitee []string                       // backup committee members
	SelfAddress    []string                       // own addresses in different contexts
	Cryptoset      *CryptoSet                     // cryptographic parameter set
	//committee     *orderedmap.OrderedMap       // ordered committee mapping (commented out)
}

// NewWorker creates and initializes a new PoT consensus worker instance.
// It sets up all necessary components including VDF channels, networking,
// storage, and cryptographic elements required for PoT consensus operation.
//
// Parameters:
//   - id: unique identifier for this worker node
//   - config: consensus configuration containing PoT parameters
//   - logger: structured logger for this worker instance
//   - bst: block storage for persistent block data
//   - engine: PoT engine instance for core operations
//
// Returns:
//   - *Worker: initialized worker instance ready for consensus participation
//   - nil if initialization fails (e.g., network setup error)
func NewWorker(id int64, config *config.ConsensusConfig, logger *logrus.Entry, bst *storage.BlockStorage, engine *PoTEngine) *Worker {
	ch0 := make(chan *types.VDF0res, 2048)
	ch1 := make(chan *types.VDF0res, 2048)
	ch2 := make(chan *types.VDF0res, 2048)
	potconfig := config.PoT

	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil
	}

	rands := rand.New(rand.NewSource(seed.Uint64()))

	vdf0 := types.NewVDF(ch0, potconfig.Vdf0Iteration, id)
	vdf1 := make([]*types.VDF, cpuCounter)
	for i := 0; i < cpuCounter; i++ {
		vdf1[i] = types.NewVDF(ch1, potconfig.Vdf1Iteration, id)
	}
	vdfhalf := types.NewVDF(ch2, potconfig.Vdf1Iteration, id)

	listen, err := net.Listen("tcp", config.PoT.BciRpcAddress)
	if err != nil {
		panic(err)
	}

	peer := make(chan *types.Block, 5)
	keyblockmap := make(map[[crypto.Hashlen]byte][]byte)
	commiteemap := make(map[[crypto.Hashlen]byte][]byte)
	mempool := NewMempool()
	Commitee := make([][]string, 0)
	BackupCommitee := make([]string, BackupCommiteeSize)

	aborts := &Abortcontrol{
		abortchannel: make(chan struct{}),
		once:         new(sync.Once),
	}
	randseed := make([]byte, 32)
	_, err = rand.Read(randseed)
	if err != nil {
		return nil
	}

	vdfInstance := vdf.New(potconfig.VdfType, []byte(""), potconfig.Vdf0Iteration, id)

	w := &Worker{
		abort:         aborts,
		Engine:        engine,
		config:        config,
		ID:            id,
		log:           logger,
		epoch:         0,
		vdf0:          vdf0,
		vdf0Chan:      ch0,
		vdf1:          vdf1,
		vdf1Chan:      ch1,
		vdfhalf:       vdfhalf,
		vdfhalfchan:   ch2,
		wg:            new(sync.WaitGroup),
		rand:          rands,
		peerMsgQueue:  peer,
		mutex:         new(sync.Mutex),
		rwmutex:       new(sync.RWMutex),
		executeheight: uint64(0),
		//storage:      st,
		mempool:      mempool,
		blockStorage: bst,
		//committee:    orderedmap.NewOrderedMap(),
		vdfChecker:      vdfInstance,
		chainReader:     NewChainReader(bst),
		PeerId:          engine.GetPeerID(),
		workFlag:        false,
		blockKeyMap:     keyblockmap,
		CommitteeKeyMap: commiteemap,
		Commitee:        Commitee,
		CommiteeNum:     int32(1),
		BackupCommitee:  BackupCommitee,
		chainresetflag:  false,
		keyseed:         randseed,
	}
	// bci info record
	if err := utils.AppendToFile("bci", "[seed]"+hexutil.Encode(randseed)+"\n"); err != nil {
		logger.WithError(err).Error("Failed to write seed to bci file")
	}
	rpcserver := grpc.NewServer()
	pb.RegisterBciExectorServer(rpcserver, w)
	w.rpcserver = rpcserver
	w.listener = listen

	w.Init()

	// add pot worker here

	return w
}

// Init initializes the worker with default values and starts the HTTP server.
// This method sets up the initial VDF state, genesis block, and web interface.
// It should be called once after worker creation to prepare for consensus operations.
func (w *Worker) Init() {
	// Initialize VDF half computation with default input
	w.vdfhalf.SetInput(crypto.Hash([]byte("aa")), w.config.PoT.Vdf1Iteration)

	// Set initial VDF0 result for epoch 0
	w.SetVdf0res(0, ([]byte("aa")))

	// Store the genesis block in block storage
	w.blockStorage.Put(types.DefaultGenesisBlock())

	// Initialize block counter
	w.blockCounter = 0
}

// startWorking sets the worker's work flag to indicate mining is active.
// This method is thread-safe and should be called when starting mining operations.
func (w *Worker) startWorking() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.workFlag = true
}

// Work initiates the worker's mining process by starting VDF computation.
// This method records the start timestamp and begins the VDF half computation
// for the current epoch. It's the entry point for the mining workflow.
func (w *Worker) Work() {
	w.timestamp = time.Now()
	w.log.Infof("[PoT]\tStart epoch %d vdf0", w.getEpoch())

	err := w.vdfhalf.Exec(0)
	if err != nil {
		return
	}
}

// WaitandReset implements a delay mechanism for VDF result processing.
// It waits for a fixed duration before resending the VDF0 result to the channel.
// This is used for handling timing-related issues in epoch transitions.
//
// Parameters:
//   - res: VDF0 result to be resent after delay
func (w *Worker) WaitandReset(res *types.VDF0res) {
	time.Sleep(10 * time.Second)
	w.vdf0Chan <- res
}

// OnGetVdf0Response handles VDF0 computation results and manages epoch transitions.
// This is the main event loop that processes VDF0 results, validates blocks,
// updates epochs, and initiates mining for new blocks. It runs continuously
// and coordinates the entire PoT consensus workflow.
func (w *Worker) OnGetVdf0Response() {
	// Start block handling in a separate goroutine
	go w.handleBlock()

	for {
		select {
		// receive vdf0
		case res := <-w.vdf0Chan:
			epoch := w.getEpoch()
			timer := time.Since(w.timestamp) / time.Millisecond
			w.log.Infof("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\n", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)
			//time.Sleep(10 * time.Second)

			if epoch > res.Epoch {
				w.log.Errorf("[PoT]\tthe epoch already execset")
				continue
			}
			if len(res.Res) == 0 {
				w.log.Errorf("[PoT]\treceive vdf0 res is empty")
			}

			//timestop := math.Floor(float64(timer) * float64(10-w.config.PoT.Slowrate) / float64(w.config.PoT.Slowrate))
			//time.Sleep(time.Duration(timestop) * time.Millisecond)
			//
			//timer = time.Since(w.timestamp) / time.Millisecond
			//w.log.Infof("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\Commitees", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)

			flag, err := w.CheckParentBlockEnough(epoch)

			if !flag {
				w.log.Infof("[PoT]\tepoch %d: check parent block enough fail for %s", epoch, err)
				go w.WaitandReset(res)
				continue
			}
			// the last epoch is over
			// epoch increase
			res0 := res.Res
			w.SetVdf0res(res.Epoch+1, res0)

			if w.IsVDF1Working() {
				w.abort.once.Do(func() {
					close(w.abort.abortchannel)
				})

				w.setWorkFlagFalse()
				w.wg.Wait()
				w.log.Debugf("[PoT]\tepoch %d:the miner got abort for get in new epoch", epoch+1)
			}

			// calculate the next epoch vdf
			inputHash := crypto.Hash(res0)

			if !w.vdf0.IsFinished() {

				err := w.vdf0.Abort()
				if err != nil {
					w.log.Warnf("[PoT]\tepoch %d: vdf0 abort error for %s", epoch+1, err)
				}
				w.log.Warnf("[PoT]\tepoch %d:vdf0 got abort for new epoch ", epoch+1)
			}

			w.increaseEpoch()
			//w.vdf0 = types.NewVDFwithInput(w.getVDF0chan(), inputHash, w.config.PoT.Vdf0Iteration, w.ID)
			w.vdfhalf = types.NewVDFwithInput(w.vdfhalfchan, inputHash, w.config.PoT.Vdf1Iteration, w.ID)
			if err != nil {
				w.log.Warnf("[PoT]\tepoch %d:execset vdf0 error for %t", epoch+1, err)
				continue
			}

			w.log.Debugf("[PoT]\tepoch %d:Start epoch %d vdf0", epoch+1, epoch+1)
			w.timestamp = time.Now()
			go func() {
				err = w.vdfhalf.Exec(epoch + 1)
				if err != nil {
					w.log.Info("[PoT]\texecute vdf error for :", err)
				}
			}()

			backupblock, err := w.blockStorage.GetbyHeight(epoch)
			w.log.Infof("[PoT]\tepoch %d: epoch %d block num %d", epoch+1, epoch, len(backupblock))

			parentblock, uncleblock := w.blockSelection(backupblock, res0, epoch)

			if parentblock != nil {
				w.log.Infof("[PoT]\tepoch %d:parent block hash: %s Difficulty %d from %d", epoch+1, hex.EncodeToString(parentblock.Hash()), parentblock.GetHeader().Difficulty.Int64(), parentblock.GetHeader().Address)
			} else {
				if len(backupblock) != 0 {
					//w.chainReader.SetHeight(epoch, backupblock[0])
					//parentblock = backupblock[0]
					//w.log.Infof("[PoT]\tepoch %d:parent block hash is nil,execset nil block %s as parent", epoch+1, hex.EncodeToString(parentblock.GetHeader().Hashes))
					continue
				} else {
					w.log.Errorf("[PoT]\tepoch %d:dont't find any parent block", epoch+1)
					panic(fmt.Errorf("[PoT]\tepoch %d:dont't find any parent block", epoch+1))
				}
			}

			_, err = w.GetExcutedTxsFromExecutor(epoch)
			if err != nil {
				w.log.Warnf("[PoT]\tepoch %d: Get Tx from executor error for %s", epoch+1, err)
			} else {
				w.log.Debugf("[PoT]\tepoch %d: Get Txs from executor", epoch+1)
			}

			_ = w.handleBlockExecutedHeader(parentblock)

			_, err = w.GetIncentiveTxFromExecutor(epoch)
			if err != nil {
				w.log.Errorf("[PoT]\tepoch %d: Get Incentive Tx from executor error for %s", epoch+1, err)
			} else {
				w.log.Infof("[PoT]\tepoch %d: Get Incentive Txs from executor", epoch+1)
			}

			err = w.handleBlockRawTx(parentblock)
			if err != nil {
				w.log.Errorf("[PoT]\tepoch %d: Handle Txs for block %s err for %s", epoch+1, hexutil.Encode(parentblock.Hash()), err)
			}

			//w.UpdateLocalCryptoSetByBlock(parentblock.GetHeader().Height, parentblock)

			// if epoch > 1 {

			// 	w.simpleLeaderUpdate(parentblock)
			// }

			w.CommitteeUpdate(epoch)
			difficulty := w.calcDifficulty(parentblock, uncleblock)
			w.startWorking()
			w.abort = NewAbortcontrol()

			w.wg.Add(cpuCounter)

			vdf0rescopy := make([]byte, len(res0))
			copy(vdf0rescopy, res0)

			exeblocks := w.GetExecutedBlockFromMempool()
			rawtxs := w.mempool.GetRawTx()
			Bcirewards := w.mempool.GetAllBciRewards()

			for i := 0; i < cpuCounter; i++ {
				block := w.createBlockWithoutKey(epoch+1, parentblock, uncleblock, difficulty, exeblocks, rawtxs)
				go w.mine(epoch+1, vdf0rescopy, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg, block, Bcirewards)
			}
		}
	}
}

// mine performs the actual mining process for a specific worker thread.
// It generates a new block by computing VDF1 proofs and checking against the target difficulty.
// The function runs until a valid block is found or an abort signal is received.
//
// Parameters:
//   - epoch: current epoch number for mining
//   - vdf0res: VDF0 result from epoch transition
//   - nonce: starting nonce value for mining attempts
//   - workerid: identifier for this mining worker thread
//   - abort: control structure for aborting mining operation
//   - difficulty: target difficulty for the block
//   - parentblock: parent block for the new block
//   - uncleblock: uncle blocks to include in the new block
//   - wg: wait group for synchronization with other workers
//   - emptyblock: template block to be completed with mining results
//   - Bcirewards: BCI rewards to be included in coinbase transaction
//
// Returns:
//   - *types.Block: successfully mined block, or nil if aborted
func (w *Worker) mine(epoch uint64, vdf0res []byte, nonce int64, workerid int, abort *Abortcontrol,
	difficulty *big.Int, parentblock *types.Block, uncleblock []*types.Block, wg *sync.WaitGroup,
	emptyblock *types.Block, Bcirewards []*BciReward) *types.Block {
	defer wg.Done()
	defer w.setWorkFlagFalse()
	w.log.Infof("[PoT]\tepoch %d:workerid %d start to mine", epoch, workerid)

	privkey, _ := crypto.GeneratePqcKey()
	pubkeybyte := privkey.PublicKeyBytes()

	totalreward := CalcTotalReward(epoch)
	coinbasetx := w.GenerateCoinbaseTxWithoutMinerKey(Bcirewards, privkey, totalreward)
	coinbaseproofs := coinbasetx.CoinbaseProofs
	coinbaseProofsbyte := CoinbaseProofToBytes(coinbaseproofs)
	mixdigest := w.calcMixdigest(epoch, parentblock, uncleblock, difficulty, w.PeerId, pubkeybyte, coinbaseProofsbyte)
	tmp := new(big.Int)
	tmp.SetInt64(nonce)
	noncebyte := tmp.Bytes()
	input := bytes.Join([][]byte{noncebyte, vdf0res, mixdigest}, []byte(""))
	hashinput := crypto.Hash(input)

	w.vdf1[workerid] = types.NewVDFwithInput(w.vdf1Chan, hashinput, w.config.PoT.Vdf1Iteration, w.ID)
	vdfCh := w.vdf1Chan

	go func() {
		w.log.Debugf("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
		err := w.vdf1[workerid].Exec(epoch)
		if err != nil {
			return
		}
	}()

	target := new(big.Int).Set(bigD)
	target = target.Div(bigD, difficulty)

	for {
		select {
		case res := <-vdfCh:
			res1 := res.Res
			// compare to target

			tmp.SetBytes(crypto.Hash(res1))
			if tmp.Cmp(target) >= 0 {

				//block := w.createNilBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
				w.log.Infof("[PoT]\tepoch %d: workerid %d fail to find a %d block %s", epoch, workerid, difficulty.Int64(), hexutil.Encode(tmp.Bytes()))
				//w.blockStorage.Put(block)

				nonce += 1
				tmp.SetInt64(nonce)
				noncebyte2 := tmp.Bytes()
				input2 := bytes.Join([][]byte{noncebyte2, vdf0res, mixdigest}, []byte(""))
				hashinput2 := crypto.Hash(input2)
				w.vdf1[workerid] = types.NewVDFwithInput(w.vdf1Chan, hashinput2, w.config.PoT.Vdf1Iteration, w.ID)

				go func() {
					w.log.Infof("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
					err := w.vdf1[workerid].Exec(epoch)
					if err != nil {
						return
					}
				}()

				continue
			}
			// w.createBlock

			// begin new work
			nonce = rand.Int63()
			tmp.SetInt64(nonce)
			noncebyte := tmp.Bytes()
			privkey2, _ := crypto.GeneratePqcKey()
			pubkey2byte := privkey2.PublicKeyBytes()
			coinbasetx2 := w.GenerateCoinbaseTxWithoutMinerKey(Bcirewards, privkey2, totalreward)
			coinbaseproofs2 := coinbasetx2.CoinbaseProofs
			coinbaseProofsbyte2 := CoinbaseProofToBytes(coinbaseproofs2)
			mix2digest := w.calcMixdigest(epoch, parentblock, uncleblock, difficulty, w.PeerId, pubkey2byte, coinbaseProofsbyte2)

			in2put := bytes.Join([][]byte{noncebyte, vdf0res, mix2digest}, []byte(""))
			hash2input := crypto.Hash(in2put)

			w.vdf1[workerid] = types.NewVDFwithInput(w.vdf1Chan, hash2input, w.config.PoT.Vdf1Iteration, w.ID)

			go func() {
				w.log.Debugf("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
				err := w.vdf1[workerid].Exec(epoch)
				if err != nil {
					return
				}
			}()

			block := w.CompleteBlock(emptyblock, vdf0res, res1, coinbasetx, mixdigest, privkey)

			w.blockCounter += 1
			w.log.Infof("[PoT]\tepoch %d:get new block %d", epoch, w.blockCounter)

			// broadcast the block
			w.peerMsgQueue <- block
			// w.workFlag = false
			continue
		case <-abort.abortchannel:
			err := w.vdf1[workerid].Abort()
			if err != nil {
				w.log.Errorf("[PoT]\tepoch %d:vdf1 %d mine abort error for %t", epoch, workerid, err)
				return nil
			}
			w.log.Debugf("[PoT]\tepoch %d:vdf1 workerid %d got abort", epoch, workerid)
			// w.workFlag = false
			return nil
		}

	}
}

// handleVdfhalf processes VDF half computation results and manages the transition
// to VDF0 computation. This function handles the two-stage VDF process where
// VDF half completion triggers the start of VDF0 computation for epoch progression.
func (w *Worker) handleVdfhalf() {
	for {
		select {
		case res := <-w.vdfhalfchan:
			epoch := w.getEpoch()
			w.log.Infof("[PoT]\tepoch %d:vdfhalf got res %s", epoch, hexutil.Encode(crypto.Hash(res.Res)))

			// Validate epoch consistency
			if res.Epoch != epoch {
				w.log.Errorf("[PoT]\tepoch %d:vdfhalf got res but epoch is %d", epoch, res.Epoch)
			}

			// Store VDF half result for later use in block completion
			w.blockStorage.SetVDFHalf(epoch, res.Res)

			// Abort any running VDF0 computation as we transition to new VDF0
			if !w.vdf0.IsFinished() {
				err := w.vdf0.Abort()
				if err != nil {
					w.log.Warnf("[PoT]\tepoch %d: vdf0 abort error for %s", epoch+1, err)
				}
				w.log.Warnf("[PoT]\tepoch %d:vdf0 got abort for new epoch ", epoch+1)
			}

			// Start VDF0 computation with VDF half result as input
			inputhash := crypto.Hash(res.Res)
			w.vdf0 = types.NewVDFwithInput(w.getVDF0chan(), inputhash, w.config.PoT.Vdf0Iteration-w.config.PoT.Vdf1Iteration, w.ID)

			// Execute VDF0 computation in a separate goroutine
			go func() {
				err := w.vdf0.Exec(epoch)
				if err != nil {
					w.log.Warnf("[PoT]\tepoch %d: vdf0 run error for %s", epoch+1, err)
				}
			}()
		}
	}
}

// createBlockWithoutKey creates a template block without cryptographic keys.
// This creates the basic block structure with transactions and headers that
// will later be completed with mining proofs and keys during the mining process.
//
// Parameters:
//   - epoch: block epoch/height
//   - parentBlock: parent block reference
//   - uncleBlock: uncle blocks to include
//   - difficulty: target difficulty for mining
//   - exeblocks: executed blocks from executor
//   - rawtxs: raw transactions to include
//
// Returns:
//   - *types.Block: template block ready for mining completion
func (w *Worker) createBlockWithoutKey(epoch uint64, parentBlock *types.Block,
	uncleBlock []*types.Block, difficulty *big.Int,
	exeblocks []*types.ExecutedBlock, rawtxs []*types.RawTx) *types.Block {
	exeheader := make([]*types.ExecuteHeader, 0)
	for _, exeblock := range exeblocks {
		exeheader = append(exeheader, exeblock.Header)
	}
	//w.log.Infof("Txs len %d", len(Txs))
	parentblockhash := make([]byte, 0)
	if parentBlock != nil {
		parentblockhash = parentBlock.Hash()
	}
	uncleBlockhash := make([][]byte, len(uncleBlock))
	for i := 0; i < len(uncleBlock); i++ {
		uncleBlockhash[i] = uncleBlock[i].Hash()
	}

	Txs := make([]*types.Tx, 0)
	for _, rawtx := range rawtxs {
		if err := w.checkLockTransaction(rawtx); err == nil {
			txoutput := &rawtx.TxOutput[0]
			txoutput.CreatedAt = epoch
			if math.Abs(txoutput.Rate-float64(0)) < epsilon && txoutput.BurnLock != 0 {
				yeargap := txoutput.BurnLock - epoch
				rate := float64(0)
				if yeargap/TenYears >= 1 {
					rate = TenYearRate

				} else if yeargap/TenYears < 1 && yeargap/ThreeYears >= 1 {
					rate = ThreeYearRate
				} else if yeargap/ThreeYears < 1 && yeargap/OneYear >= 1 {
					rate = OneYearRate
				} else if yeargap/OneYear < 1 && yeargap/HalfYear >= 1 {
					rate = HalfYearRate
				}
				txoutput.Rate = rate
			}
		}
		//fmt.Println(hexutil.Encode(rawtx.Txid[:]), 123456)
		txbyte, err := rawtx.EncodeToByte()
		if err != nil {
			return nil
		}
		Txs = append(Txs, &types.Tx{Data: txbyte})
	}

	id := w.ID
	peerid := w.PeerId

	h := &types.Header{
		Height:     epoch,
		ParentHash: parentblockhash,
		UncleHash:  uncleBlockhash,

		Difficulty: difficulty,

		Timestamp: time.Now(),

		Address: id,
		PeerId:  peerid,

		Hashes: nil,

		//CryptoElement:  cryptoset,
	}
	return &types.Block{
		Header:     h,
		Txs:        Txs,
		ExeHeaders: exeheader,
	}
}

// CompleteBlock finalizes a mined block by adding all required proofs and signatures.
// This function takes a template block and completes it with VDF proofs, coinbase transaction,
// cryptographic keys, and all necessary hashes to create a valid block for the blockchain.
//
// Parameters:
//   - emptyblock: template block to be completed
//   - vdf0res: VDF0 computation result
//   - vdf1res: VDF1 computation result from mining
//   - coinbasetx: coinbase transaction template
//   - mixdigest: computed mix digest for the block
//   - privkey: private key for block signing
//
// Returns:
//   - *types.Block: completed and signed block ready for broadcast
func (w *Worker) CompleteBlock(emptyblock *types.Block, vdf0res []byte, vdf1res []byte, coinbasetx *types.RawTx, mixdigest []byte, privkey *crypto.PqcKey) *types.Block {
	// Create copies of VDF results to avoid modification
	vdf0rescopy := make([]byte, len(vdf0res))
	copy(vdf0rescopy, vdf0res)
	vdf1rescopy := make([]byte, len(vdf1res))
	copy(vdf1rescopy, vdf1res)

	vdfhalfch := make(chan []byte, 2048)
	go func(epoch uint64) []byte {
		for {
			b, err := w.blockStorage.GetVDFHalf(epoch)
			if err == nil && b != nil {
				vdfhalfch <- b
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}(emptyblock.Header.Height)

	vdfhalf := <-vdfhalfch
	if len(vdfhalf) == 0 {
		w.log.Error("[PoT]\tget vdfhalf error for result is 0")
	}
	PotProof := [][]byte{vdf0rescopy, vdf1rescopy, vdfhalf}

	block := emptyblock
	cointx := w.CompleteCoinbaseTx(vdf1res, coinbasetx, emptyblock.Header.Height)
	txs := make([]*types.Tx, 0)
	txs = append(txs, cointx)
	txs = append(txs, emptyblock.Txs...)
	pubkeybyte := privkey.PublicKeyBytes()
	block.Header.PoTProof = PotProof
	block.Header.Mixdigest = mixdigest
	block.Header.PublicKey = pubkeybyte
	block.Txs = txs

	txhash := crypto.ComputeMerkleRoot(types.Txs2Bytes(txs))
	block.Header.TxHash = txhash

	block.Header.Hashes = block.Header.Hash()
	w.SetBlockKeyMap(privkey.PrivateKeyBytes(), crypto.Convert(block.Header.Hash()))

	// commiteekey := crypto.GenerateKey()
	// keybyte := commiteekey.PublicKeyBytes()
	// block.Header.CommiteePubkey = keybyte
	// w.SetCommiteeKeyMap(commiteekey.Private(), crypto.Convert(block.Header.Hash()))
	commiteekey := crypto.GenerateCommiteeKey(privkey, w.keyseed, vdf0rescopy)
	block.Header.CommiteePubkey = commiteekey.PublicKeyBytes()
	block.Header.Hashes = block.Header.Hash()

	w.SetBlockKeyMap(privkey.PrivateKeyBytes(), crypto.Convert(block.Header.Hash()))

	return block
}
func (w *Worker) CompleteCoinbaseTx(vdf1res []byte, coinbasetx *types.RawTx, height uint64) *types.Tx {
	coinbaseproofs := coinbasetx.CoinbaseProofs
	selectproofs := make(map[int32][]*types.CoinbaseProof)
	notdrawProof := make(map[int32][]*types.CoinbaseProof)
	totalreward := CalcTotalReward(height)
	if len(coinbaseproofs) != 0 {
		groupsdata := groupByType(coinbaseproofs)
		for _, proofs := range groupsdata {
			total := int64(0)
			for _, proof := range proofs {
				if !proof.DoDraw {
					notdrawProof[proof.Type] = append(notdrawProof[proof.Type], &proof)
				} else {
					total += proof.Amount
				}
			}
			for _, proof := range proofs {
				if proof.DoDraw {
					proof.Weight = float64(proof.Amount) / float64(total)
				}
			}
		}

		for bcitypes, proofs := range groupsdata {
			sort.Slice(proofs, func(i, j int) bool {
				return bytes.Compare(proofs[i].Address, proofs[j].Address) < 0
			})

			rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf1res)[:8]))

			for i := 0; i < Selectn; i++ {
				r := rand.Float64()
				acnum := 0.0
				for _, proof := range proofs {
					acnum += proof.Weight
					if r < acnum {
						selectproofs[bcitypes] = append(selectproofs[bcitypes], &proof)
					}
				}
			}
		}

		for bcitype, proofs := range selectproofs {
			rate, ok := bcimap[bcitype]
			if !ok {
				return nil
			}
			lenproofs := len(proofs)
			for _, proof := range proofs {
				txout := types.TxOutput{
					Address:  proof.Address,
					Value:    int64(math.Floor(float64(totalreward) * rate / float64(lenproofs))),
					Interest: 0,
					ScriptPk: nil,
					Proof:    nil,
					LockTime: CoinbaseLock,
					BciType:  bcitype,
					Data:     nil,
				}
				coinbasetx.TxOutput = append(coinbasetx.TxOutput, txout)
			}
		}

		for bcitype, proofs := range notdrawProof {
			for _, proof := range proofs {
				txout := types.TxOutput{
					Address:  proof.Address,
					Value:    proof.Amount,
					Interest: 0,
					ScriptPk: nil,
					Proof:    nil,
					LockTime: CoinbaseLock,
					BciType:  bcitype,
					Data:     nil,
				}
				coinbasetx.TxOutput = append(coinbasetx.TxOutput, txout)
			}
		}

		sort.Slice(coinbasetx.TxOutput, func(i, j int) bool { return coinbasetx.TxOutput[i].BciType < coinbasetx.TxOutput[j].BciType })

	}

	coinbasetx.Txid = coinbasetx.Hash()
	if len(coinbasetx.TxOutput) > 1 {
		for _, output := range coinbasetx.TxOutput {
			w.log.Infof("bcitx to %s with amount %d ", hexutil.Encode(output.Address), output.Value)
		}
	}
	txdata, _ := coinbasetx.EncodeToByte()
	return &types.Tx{Data: txdata}
}

// createBlock creates a complete block with all mining proofs and transactions.
// This is an older version of block creation that includes VDF proofs and keys
// directly in the creation process, unlike createBlockWithoutKey which creates a template.
//
// Parameters:
//   - epoch: block epoch/height
//   - parentBlock: parent block reference
//   - uncleBlock: uncle blocks to include
//   - difficulty: mining difficulty
//   - mixdigest: computed mix digest
//   - nonce: mining nonce value
//   - vdf0res: VDF0 computation result
//   - vdf1res: VDF1 mining result
//
// Returns:
//   - *types.Block: complete block with all proofs and signatures
func (w *Worker) createBlock(epoch uint64, parentBlock *types.Block, uncleBlock []*types.Block, difficulty *big.Int, mixdigest []byte, nonce int64, vdf0res []byte, vdf1res []byte) *types.Block {
	exeblocks := w.GetExecutedBlockFromMempool()
	exeheader := make([]*types.ExecuteHeader, 0)
	for _, exeblock := range exeblocks {
		exeheader = append(exeheader, exeblock.Header)
	}
	//w.log.Infof("Txs len %d", len(Txs))
	parentblockhash := make([]byte, 0)
	if parentBlock != nil {
		parentblockhash = parentBlock.Hash()
	}
	uncleBlockhash := make([][]byte, len(uncleBlock))
	for i := 0; i < len(uncleBlock); i++ {
		uncleBlockhash[i] = uncleBlock[i].Hash()
	}

	vdf0rescopy := make([]byte, len(vdf0res))
	copy(vdf0rescopy, vdf0res)
	vdf1rescopy := make([]byte, len(vdf1res))
	copy(vdf1rescopy, vdf1res)

	PotProof := [][]byte{vdf0rescopy, vdf1rescopy}

	privateKey := crypto.GenerateKey()
	publicKeyBytes := privateKey.PublicKeyBytes()
	//publicKeyBytes32 := crypto.Convert(publicKeyBytes)
	totalreward := CalcTotalReward(epoch)

	coinbasetx := w.GenerateCoinbaseTx(publicKeyBytes, vdf0res, totalreward)
	Txs := []*types.Tx{coinbasetx}
	rawtxs := w.mempool.GetRawTx()
	for _, rawtx := range rawtxs {
		//fmt.Println(hexutil.Encode(rawtx.Txid[:]), 123456)
		txbyte, err := rawtx.EncodeToByte()
		if err != nil {
			return nil
		}
		Txs = append(Txs, &types.Tx{Data: txbyte})
	}
	txshash := crypto.ComputeMerkleRoot(types.Txs2Bytes(Txs))
	id := w.ID
	peerid := w.PeerId

	h := &types.Header{
		Height:     epoch,
		ParentHash: parentblockhash,
		UncleHash:  uncleBlockhash,
		Mixdigest:  mixdigest,
		Difficulty: difficulty,
		Nonce:      nonce,
		Timestamp:  time.Now(),
		PoTProof:   PotProof,
		Address:    id,
		PeerId:     peerid,
		TxHash:     txshash,
		Hashes:     nil,
		PublicKey:  publicKeyBytes,
		//CryptoElement:  cryptoset,
	}

	h.Hashes = h.Hash()
	w.SetBlockKeyMap(privateKey.Private(), crypto.Convert(h.Hash()))

	return &types.Block{
		Header:     h,
		Txs:        Txs,
		ExeHeaders: exeheader,
	}
}

// GetExcutedTxsFromExecutor retrieves executed transaction blocks from the executor service.
// This function communicates with the external executor to fetch blocks that have been
// executed and validated, which are then added to the mempool for inclusion in new blocks.
//
// Parameters:
//   - epoch: current epoch for which to retrieve executed transactions
//
// Returns:
//   - []*types.ExecutedBlock: list of executed blocks from the executor
//   - error: error if communication with executor fails
func (w *Worker) GetExcutedTxsFromExecutor(epoch uint64) ([]*types.ExecutedBlock, error) {
	conn, err := grpc.NewClient(w.config.PoT.ExecutorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))

	if err != nil {
		return nil, err
	}
	client := pb.NewPoTExecutorClient(conn)
	request := &pb.GetTxRequest{
		StartHeight: w.executeheight,
		Des:         w.config.PoT.ExecutorAddress,
	}
	response, err := client.GetTxs(context.Background(), request)
	if err != nil {
		w.log.Errorf("[PoT]\tGet Txs from executor error for %s", err)
		return nil, err
	}

	executeblocks := response.GetBlocks()
	excuteheight := response.GetEnd()

	if excuteheight > w.executeheight {
		w.executeheight = excuteheight
	}

	//executedtxs := make([]*types.ExecutedTxData, 0)
	//for i := 0; i < len(executeblocks); i++ {
	//height := executeblocks[i].GetHeader().GetHeight()
	//executedTxs := executeblocks[i].GetTxs()
	//for _, executedtx := range executedTxs {
	//	exectx := &types.ExecutedTxData{
	//		ExecutedHeight: height,
	//		TxHash:         executedtx.GetTxHash(),
	//	}
	//	//exectxdata, _ := exectx.EncodeToByte()
	//	executedtxs = append(executedtxs, exectx)
	//	w.mempool.Add(exectx)
	//}

	//}
	blocks := make([]*types.ExecutedBlock, 0)
	for _, executeblock := range executeblocks {
		pbheader := executeblock.GetHeader()
		header := &types.ExecuteHeader{
			Height:    pbheader.GetHeight(),
			BlockHash: pbheader.GetBlockHash(),
			TxsHash:   pbheader.GetTxsHash(),
		}
		pbtxs := executeblock.GetTxs()
		executedtxs := make([]*types.ExecutedTx, 0)
		for _, pbtx := range pbtxs {
			executedtx := &types.ExecutedTx{
				Height: pbtx.GetHeight(),
				TxHash: pbtx.GetTxHash(),
				Data:   pbtx.GetData(),
			}
			executedtxs = append(executedtxs, executedtx)
		}
		block := &types.ExecutedBlock{
			Header: header,
			Txs:    executedtxs,
		}
		w.mempool.Add(block)
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetExecutedBlockFromMempool retrieves a batch of executed blocks from the mempool.
// This function gets the first N executed blocks (where N is the configured batch size)
// that are ready to be included in the next block.
//
// Returns:
//   - []*types.ExecutedBlock: batch of executed blocks from mempool
func (w *Worker) GetExecutedBlockFromMempool() []*types.ExecutedBlock {
	ExecutedBlocks := w.mempool.GetFirstN(w.config.PoT.Batchsize)
	return ExecutedBlocks
}

// GetIncentiveTxFromExecutor retrieves incentive transactions from the executor service.
// This function fetches BCI (Blockchain Incentive) rewards that should be distributed
// to miners and other network participants based on their contributions.
//
// Parameters:
//   - epoch: current epoch for which to retrieve incentive transactions
//
// Returns:
//   - []*types.ExecutedBlock: list of executed blocks (currently returns nil)
//   - error: error if communication with executor fails or reward validation fails
func (w *Worker) GetIncentiveTxFromExecutor(epoch uint64) ([]*types.ExecutedBlock, error) {
	conn, err := grpc.NewClient(w.config.PoT.ExecutorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))

	if err != nil {
		return nil, err
	}

	client := pb.NewPoTExecutorClient(conn)
	response, err := client.GetIncentive(context.Background(), &pb.GetIncentiveRequest{
		Begin: w.incentiveheight,
		End:   w.executeheight,
	})

	if err != nil {
		return nil, err
	}

	bcireward := response.BciReward
	for _, pbBcireward := range bcireward {

		BciReward := ToBciReward(pbBcireward)
		flag, _, err := w.VerifyBciReward(BciReward)
		if !flag {
			return nil, fmt.Errorf("the Bci reward is not valid for %s", err.Error())
		} else {
			// txdata := tx.Data
			// if len(txdata) < 98 {
			// 	return nil, fmt.Errorf("the Bci reward is not valid for %s", err.Error())
			// } else {
			// 	address := txdata[2:98]
			// 	//fmt.Println(hexutil.Encode(address))
			// 	BciReward.Address = address
			// 	w.mempool.AddBciReward(BciReward)
			// }
			w.log.Infof("[PoT]\tAdd %d Bci reward to %s ", BciReward.Amount, hexutil.Encode(BciReward.Address))
			w.mempool.AddBciReward(BciReward)
		}
	}
	if w.incentiveheight < response.GetEnd() {
		w.incentiveheight = response.GetEnd()
	}
	return nil, nil
}

func (w *Worker) createNilBlock(epoch uint64, parentBlock *types.Block, uncleBlock []*types.Block, difficulty *big.Int, mixdigest []byte, nonce int64, vdf0res []byte, vdf1res []byte) *types.Block {
	parentblockhash := make([]byte, 0)
	if parentBlock != nil {
		parentblockhash = parentBlock.Hash()
	}
	uncleBlockhash := make([][]byte, len(uncleBlock))
	for i := 0; i < len(uncleBlock); i++ {
		uncleBlockhash[i] = uncleBlock[i].Hash()
	}

	Potproof := [][]byte{vdf0res, vdf1res}
	privateKey := crypto.GenerateKey()
	publicKeyBytes := privateKey.PublicKeyBytes()
	//privkeybyte32 := crypto.Convert(privateKey.Private())

	h := &types.Header{
		Height:     epoch,
		ParentHash: parentblockhash,
		UncleHash:  uncleBlockhash,
		Mixdigest:  mixdigest,
		Difficulty: big.NewInt(0),
		Nonce:      0,
		Timestamp:  time.Now(),
		PoTProof:   Potproof,
		Address:    w.ID,
		PeerId:     w.PeerId,
		TxHash:     crypto.NilTxsHash,
		Hashes:     nil,
		PublicKey:  publicKeyBytes,
	}
	h.Hashes = h.Hash()

	w.SetBlockKeyMap(privateKey.Private(), crypto.Convert(h.Hash()))

	return &types.Block{
		Header: h,
		Txs:    make([]*types.Tx, 0),
	}
}

// SetBlockKeyMap stores the private key associated with a block hash.
// This mapping is used to retrieve keys for blocks that this worker has mined,
// enabling block signing and committee participation.
//
// Parameters:
//   - privatekey: private key bytes to store
//   - blockhash: block hash as the key for the mapping
func (w *Worker) SetBlockKeyMap(privatekey []byte, blockhash [crypto.Hashlen]byte) {
	w.rwmutex.Lock()
	defer w.rwmutex.Unlock()
	w.blockKeyMap[blockhash] = privatekey
}

// TryFindKey attempts to retrieve the private key for a given block hash.
// This is used to check if the worker has the key for a specific block,
// which indicates the worker mined that block.
//
// Parameters:
//   - blockhash: block hash to look up
//
// Returns:
//   - bool: true if key found, false otherwise
//   - []byte: private key bytes if found, nil otherwise
func (w *Worker) TryFindKey(blockhash [crypto.Hashlen]byte) (bool, []byte) {
	w.rwmutex.RLock()
	prikey := w.blockKeyMap[blockhash]
	w.rwmutex.RUnlock()
	if prikey != nil {
		return true, prikey
	} else {
		return false, nil
	}
}

// func (w *Worker) SetCommiteeKeyMap(privatekey []byte, blockhash [crypto.Hashlen]byte) {
// 	w.rwmutex.Lock()
// 	defer w.rwmutex.Unlock()
// 	w.CommitteeKeyMap[blockhash] = privatekey
// }

// TryFindCommiteeKey attempts to derive the committee key for a given block hash.
// This function reconstructs the committee key from the stored private key and
// block information, enabling committee operations for blocks mined by this worker.
//
// Parameters:
//   - blockhash: block hash to derive committee key for
//
// Returns:
//   - bool: true if committee key can be derived, false otherwise
//   - *crypto.PrivateKey: derived committee private key if successful, nil otherwise
func (w *Worker) TryFindCommiteeKey(blockhash [crypto.Hashlen]byte) (bool, *crypto.PrivateKey) {
	w.rwmutex.RLock()
	prikey := w.blockKeyMap[blockhash]
	w.rwmutex.RUnlock()

	if prikey != nil {
		// Retrieve block to get public key and PoT proof
		block, _ := w.blockStorage.Get(blockhash[:])
		if block == nil {
			return false, nil
		}

		// Reconstruct PQC key pair
		pubkey := block.GetHeader().PublicKey
		pqckey := &crypto.PqcKey{
			Privkey: prikey,
			Pubkey:  pubkey,
			Scheme:  crypto.PqcScheme,
		}

		// Generate committee key using PQC key, seed, and VDF0 proof
		commiteekey := crypto.GenerateCommiteeKey(pqckey, w.keyseed, block.GetHeader().PoTProof[0])
		if commiteekey == nil {
			return false, nil
		}
		return true, commiteekey
	} else {
		return false, nil
	}
}

// SetVdf0res stores the VDF0 result for a specific epoch in persistent storage.
// This function is thread-safe and ensures consistent epoch-to-VDF result mapping.
//
// Parameters:
//   - epoch: epoch number for the VDF result
//   - vdf0: VDF0 computation result to store
func (w *Worker) SetVdf0res(epoch uint64, vdf0 []byte) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.blockStorage.SetVDFres(epoch, vdf0)
}

// GetVdf0byEpoch retrieves the VDF0 result for a specific epoch from storage.
//
// Parameters:
//   - epoch: epoch number to query
//
// Returns:
//   - []byte: VDF0 result for the specified epoch
//   - error: error if epoch not found or storage error
func (w *Worker) GetVdf0byEpoch(epoch uint64) ([]byte, error) {
	return w.blockStorage.GetVDFresbyEpoch(epoch)
}

// getVDF0chan returns the channel for receiving VDF0 computation results.
// This is used internally for VDF0 result communication between goroutines.
//
// Returns:
//   - chan *types.VDF0res: channel for VDF0 results
func (w *Worker) getVDF0chan() chan *types.VDF0res {
	return w.vdf0Chan
}

// checkVDFforepoch validates a VDF result for a specific epoch by comparing
// with stored results or using cryptographic verification.
//
// Parameters:
//   - epoch: epoch number to validate against
//   - vdfres: VDF result to validate
//
// Returns:
//   - bool: true if VDF result is valid for the epoch, false otherwise
func (w *Worker) checkVDFforepoch(epoch uint64, vdfres []byte) bool {
	// Check against next epoch VDF result if available
	epoch1, err := w.blockStorage.GetVDFresbyEpoch(epoch + 1)
	if err == nil && len(epoch1) != 0 {
		return bytes.Equal(vdfres, epoch1)
	}

	// Check against current epoch VDF result using cryptographic verification
	epoch0, err := w.blockStorage.GetVDFresbyEpoch(epoch)
	if err == nil && len(epoch0) != 0 {
		return w.vdfChecker.CheckVDF(epoch0, vdfres)
	}

	return false
}

// setVDF0epoch sets the worker's current epoch to a specific value.
// This function validates that the epoch is not outdated and aborts any
// running VDF1 computations before updating the epoch.
//
// Parameters:
//   - epoch: target epoch number to set
//
// Returns:
//   - error: error if epoch is outdated or VDF abort fails
func (w *Worker) setVDF0epoch(epoch uint64) error {
	epochnow := w.getEpoch()
	if epochnow > epoch {
		return fmt.Errorf("could not execset for a outdated epoch %d", epoch)
	}

	// Abort any running VDF1 work before epoch change
	if w.IsVDF1Working() {
		err := w.vdf0.Abort()
		if err != nil {
			return err
		}
		w.log.Errorf("[PoT]\tVDF0 got abort for reset")
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.epoch = epoch

	return nil
}

// getEpoch returns the current epoch number in a thread-safe manner.
//
// Returns:
//   - uint64: current epoch number
func (w *Worker) getEpoch() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	epoch := w.epoch
	return epoch
}

// increaseEpoch increments the current epoch number by 1 in a thread-safe manner.
// This is called when transitioning to the next epoch in the consensus process.
func (w *Worker) increaseEpoch() {
	w.mutex.Lock()
	w.epoch += 1
	w.mutex.Unlock()
}

// calcDifficulty calculates the mining difficulty for the next block based on
// the parent block and uncle blocks. The difficulty is computed as the average
// of all included blocks' difficulties, with a minimum value of 1.
//
// Parameters:
//   - parentblock: parent block for difficulty calculation
//   - uncleBlock: uncle blocks to include in difficulty calculation
//
// Returns:
//   - *big.Int: calculated difficulty for the next block
func (w *Worker) calcDifficulty(parentblock *types.Block, uncleBlock []*types.Block) *big.Int {
	// Use default difficulty if no parent block or parent has zero difficulty
	if parentblock == nil || parentblock.GetHeader().Difficulty.Cmp(common.Big0) == 0 {
		return big.NewInt(NoParentD)
	}

	// Sum difficulties from parent and uncle blocks
	diff := new(big.Int)
	diff.Set(parentblock.GetHeader().Difficulty)
	for _, block := range uncleBlock {
		header := block.GetHeader()
		diff.Add(diff, header.Difficulty)
	}

	// Calculate average difficulty based on system parameter
	snum := new(big.Int)
	snum.SetInt64(w.config.PoT.Snum)
	diffculty := diff.Div(diff, snum)

	// Ensure minimum difficulty of 1
	if diffculty.Cmp(big.NewInt(0)) == 0 {
		return diffculty.SetInt64(1)
	}
	return diffculty
}

// calcMixdigest computes the mix digest for a block by hashing together all
// block components including epoch, parent hash, uncle hashes, difficulty,
// peer ID, public key, and coinbase proof.
//
// Parameters:
//   - epoch: current epoch number
//   - parentblock: parent block reference
//   - uncleblock: uncle blocks
//   - difficulty: mining difficulty
//   - peerid: peer identifier
//   - pubkeybyte: public key bytes
//   - coinbaseproof: coinbase proof bytes
//
// Returns:
//   - []byte: computed mix digest hash
func (w *Worker) calcMixdigest(epoch uint64, parentblock *types.Block, uncleblock []*types.Block,
	difficulty *big.Int, peerid string, pubkeybyte []byte, coinbaseproof []byte) []byte {
	parentblockhash := make([]byte, 0)
	if parentblock != nil {
		parentblockhash = parentblock.Hash()
	}
	uncleblockhash := make([]byte, 0)
	for i := 0; i < len(uncleblock); i++ {
		hash := uncleblock[i].Hash()
		uncleblockhash = append(uncleblockhash, hash[:]...)
	}
	tmp := new(big.Int)
	tmp.Set(difficulty)
	difficultyBytes := tmp.Bytes()
	//tmp.SetInt64(ID)
	IDBytes := []byte(peerid)
	tmp.SetInt64(int64(epoch))
	epochBytes := tmp.Bytes()
	hashinput := bytes.Join([][]byte{epochBytes, parentblockhash, uncleblockhash, difficultyBytes, IDBytes, pubkeybyte, coinbaseproof}, []byte(""))
	res := sha256.Sum256(hashinput)
	return res[:]
}

func (w *Worker) caldifficultyExp(parentblock *types.Block, uncleBlock []*types.Block) *big.Int {

	D := new(big.Int).Set(parentblock.GetHeader().Difficulty)
	diff := new(big.Int).Set(parentblock.GetHeader().Difficulty)
	for _, block := range uncleBlock {
		diff = diff.Add(diff, block.GetHeader().Difficulty)
	}
	snum := big.NewInt(w.config.PoT.Snum)
	sys := w.config.PoT.SysPara
	b, err := hex.DecodeString(sys)
	if err != nil {
		w.log.Warn("calculate difficulty warning")
	}
	syspara := big.NewInt(0).SetBytes(b)

	tmp := new(big.Int).Set(diff)
	tmp.Mul(tmp, big.NewInt(-1))

	exponent := new(big.Rat).SetFrac(tmp, syspara)
	exponentFloat, _ := exponent.Float64()
	expres := decimal.NewFromFloat(math.E).Pow(decimal.NewFromFloat(exponentFloat))

	tmp2 := expres.BigInt()
	tmp2.Div(tmp2, syspara)
	tmp2.Mul(tmp2, D)

	tmp.Mul(tmp, big.NewInt(-1))
	tmp.Div(tmp, snum)

	newD := tmp2.Add(tmp2, tmp)

	D.Div(D, big.NewInt(4))

	if newD.Cmp(D) < 0 {
		return D
	} else {
		return newD
	}
}

func (w *Worker) CheckParentBlockEnough(height uint64) (bool, error) {
	if height == 0 {
		return true, nil
	}

	backupblock, err := w.blockStorage.GetbyHeight(height)

	if err != nil {
		return false, err
	}

	if len(backupblock) == 0 {
		return false, nil
	}

	current, err := w.chainReader.GetByHeight(height - 1)
	if err != nil {
		return false, err
	}
	/* */
	count := int64(0)
	for _, block := range backupblock {
		blockheader := block.GetHeader()
		if bytes.Equal(current.GetHeader().Hashes, blockheader.ParentHash) {
			count += 1
		}
	}
	if count*2 < w.config.PoT.Snum {
		return false, fmt.Errorf("not enough parent block for height %d", height)
	}

	return true, nil
}

func (w *Worker) blockSelection(blocks []*types.Block, vdf0res []byte, height uint64) (parent *types.Block, uncle []*types.Block) {
	if height == 0 {
		return types.DefaultGenesisBlock(), nil
	}
	sr := crypto.Hash(vdf0res)
	maxweight := big.NewInt(0)
	max := -1

	current, err := w.chainReader.GetByHeight(height - 1)

	if err != nil {
		return nil, nil
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	readyblocks := make([]*types.Block, 0)
	for _, block := range blocks {
		blockheader := block.GetHeader()
		if (bytes.Equal(current.GetHeader().Hashes, blockheader.ParentHash) && block.GetHeader().Difficulty.Cmp(common.Big0) != 0) || blockheader.ParentHash == nil {

			if len(block.Header.PoTProof) >= 2 {
				vdf1res := blockheader.PoTProof[1]

				readyblocks = append(readyblocks, block)
				hashinput := append(vdf1res, sr...)

				tmp := new(big.Int).Mul(bigD, blockheader.Difficulty)
				weight := new(big.Int).Div(tmp, new(big.Int).SetBytes(crypto.Hash(hashinput)))
				if weight.Cmp(maxweight) > 0 {
					max = len(readyblocks)
					maxweight.Set(weight)
				}
			}
		}
	}
	if max == -1 {
		return nil, nil
	}

	parent = readyblocks[max-1]
	uncle = append(readyblocks[:max-1], readyblocks[max:]...)
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.chainReader.SetHeight(parent.GetHeader().Height, parent)

	return parent, uncle
}

func (w *Worker) self() string {
	return w.PeerId
}

// IsVDF1Working checks if the worker is currently performing VDF1 mining operations.
// This is used to determine if mining should be aborted when starting new epochs.
//
// Returns:
//   - bool: true if VDF1 mining is active, false otherwise
func (w *Worker) IsVDF1Working() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Return the work flag status indicating if mining is active
	return w.workFlag
}

// setWorkFlagFalse sets the worker's mining flag to false, indicating mining has stopped.
// This function is thread-safe and called when mining operations complete or are aborted.
func (w *Worker) setWorkFlagFalse() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.workFlag = false
}

// SetEngine updates the worker's PoT engine reference and peer ID.
// This is used when the engine needs to be replaced or updated after initialization.
//
// Parameters:
//   - engine: new PoT engine instance to set
func (w *Worker) SetEngine(engine *PoTEngine) {
	w.Engine = engine
	w.PeerId = w.Engine.GetPeerID()
}

// handleBlockExecutedHeader processes the executed headers in a block by storing
// the corresponding executed blocks and marking them as proposed in the mempool.
//
// Parameters:
//   - block: block containing executed headers to process
//
// Returns:
//   - error: error if executed block not found in mempool or storage fails
func (w *Worker) handleBlockExecutedHeader(block *types.Block) error {
	executedHeaders := block.GetExecutedHeaders()

	// Process each executed header
	for _, header := range executedHeaders {
		hash := header.Hash()
		exeblock := w.mempool.GetBlockByHash(hash)
		if exeblock != nil {
			// Store executed block in persistent storage
			err := w.blockStorage.PutExcutedBlock(exeblock)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("handle block excuted tx error for executedblock %s does not exist in mempool", hexutil.Encode(hash[:]))
		}
	}

	// Mark executed headers as proposed in mempool
	w.mempool.MarkProposedByHeader(executedHeaders)
	return nil
}

// handleBlockRawTx processes the raw transactions in a block and validates
// that non-genesis blocks contain transactions.
//
// Parameters:
//   - block: block containing raw transactions to process
//
// Returns:
//   - error: error if non-genesis block has no transactions
func (w *Worker) handleBlockRawTx(block *types.Block) error {
	txs := block.GetRawTx()

	// Validate that non-genesis blocks contain transactions
	if len(txs) == 0 && block.Header.Height != 0 {
		return fmt.Errorf("block %s at %d has no tx", block.Hash(), block.Header.Height)
	}

	return nil

	err := w.chainReader.UpdateTxForBlock(block)
	if err != nil {
		return err
	}
	for _, tx := range txs {

		if tx.IsCoinBase() {
			Bciproofs := tx.CoinbaseProofs
			w.mempool.MarkBciRewardProposed(Bciproofs)
			if len(tx.TxOutput) == 2 {
				fmt.Println(hexutil.Encode(tx.Txid[:]), tx.TxOutput[1].Value)
			}
		}

	}
	w.mempool.MarkRawTxProposed(txs)

	if block.Header.Height > ConfirmDelay {
		handleblock, err := w.chainReader.GetByHeight(block.Header.Height - ConfirmDelay)
		if err != nil {
			return err
		}
		rawtxs := handleblock.GetRawTx()
		for _, tx := range rawtxs {
			if tx.IsCoinBase() {

			}

			for _, txoutput := range tx.TxOutput {
				if len(txoutput.Data) != 0 {
					testbyte, _ := hexutil.Decode("0x00")
					if bytes.Equal(txoutput.Data, testbyte) {
						continue
					}
					fmt.Println("find tx data not zero")
					if true {
						fmt.Printf("txid %s data transfer to vm\n", hexutil.Encode(tx.Txid[:]))
						err := w.TransferTx2EVM(txoutput.Data)
						if err != nil {
							fmt.Println("transfer err:", err)
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (w *Worker) Stop() {
	fill, err := os.OpenFile(fmt.Sprintf("./logs/blockmyself-%d", w.ID), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fill.WriteString(err.Error())
	}
	fill.WriteString("")
	fill.Close()
	w.rpcserver.Stop()
	w.listener.Close()
}

// GetMempool returns the mempool instance
func (w *Worker) GetMempool() *Mempool {
	return w.mempool
}

// GetChainReader returns the chainReader instance
func (w *Worker) GetChainReader() *ChainReader {
	return w.chainReader
}

// BroadcastClientTransaction broadcasts a client transaction
// This is a public wrapper for the private broadcastClientTransaction method
func (w *Worker) BroadcastClientTransaction(rawtx *types.RawTx, txType pb.TxType) error {
	return w.broadcastClientTransaction(rawtx, txType)
}

// CheckLockTransaction validates a lock transaction
// This is a public wrapper for the private checkLockTransaction method
func (w *Worker) CheckLockTransaction(rawtx *types.RawTx) error {
	return w.checkLockTransaction(rawtx)
}
