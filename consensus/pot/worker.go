// Package pot implements the Proof of Time (PoT) consensus algorithm worker.
// This package provides the core functionality for mining, block creation,
// and VDF (Verifiable Delay Function) computation in the PoT blockchain system.
package pot

import (
	"blockchain-crypto/vdf"
	"bytes"
	"context"
	crand "crypto/rand"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
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

// NewWorker creates and initializes a new PoT consensus worker instance
func NewWorker(id int64, config *config.ConsensusConfig, log *logrus.Entry, bst *storage.BlockStorage, engine *PoTEngine) *Worker {

	log = log.WithField("module", "POT.WORKER").WithField("w_id", id)
	log.Info("Initializing PoT worker")

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
		log:           log,
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
		log.WithError(err).Error("Failed to write seed to bci file")
	}
	rpcserver := grpc.NewServer()
	pb.RegisterBciExectorServer(rpcserver, w)
	w.rpcserver = rpcserver
	w.listener = listen

	w.Init()

	// add pot worker here

	return w
}

// Init initializes the worker with default values and genesis block
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

// startWorking sets the worker's work flag to indicate mining is active
func (w *Worker) startWorking() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.workFlag = true
}

// Work initiates the worker's mining process by starting VDF computation
func (w *Worker) Work() {
	w.timestamp = time.Now()
	w.log.WithField("epoch", w.getEpoch()).Info("Starting VDF0 computation")

	err := w.vdfhalf.Exec(0)
	if err != nil {
		return
	}
}

// WaitandReset waits for a fixed duration before resending the VDF0 result to the channel
func (w *Worker) WaitandReset(res *types.VDF0res) {
	time.Sleep(10 * time.Second)
	w.vdf0Chan <- res
}

// OnGetVdf0Response handles VDF0 computation results and manages epoch transitions
func (w *Worker) OnGetVdf0Response() {
	// Start block handling in a separate goroutine
	go w.handleBlock()

	for {
		select {
		// receive vdf0
		case res := <-w.vdf0Chan:
			epoch := w.getEpoch()
			timer := time.Since(w.timestamp) / time.Millisecond
			w.log.WithFields(logrus.Fields{
				"current_epoch": epoch,
				"result_epoch":  res.Epoch,
				"vdf_hash":      utils.EncodeShortPrint(crypto.Hash(res.Res)),
				"duration_ms":   timer,
			}).Debug("Received VDF0 result")
			//time.Sleep(10 * time.Second)

			if epoch > res.Epoch {
				w.log.WithFields(logrus.Fields{
					"current_epoch": epoch,
					"result_epoch":  res.Epoch,
				}).Warn("VDF0 result epoch already passed")
				continue
			}
			if len(res.Res) == 0 {
				w.log.Warn("Received empty VDF0 result")
			}

			//timestop := math.Floor(float64(timer) * float64(10-w.config.PoT.Slowrate) / float64(w.config.PoT.Slowrate))
			//time.Sleep(time.Duration(timestop) * time.Millisecond)
			//
			//timer = time.Since(w.timestamp) / time.Millisecond
			//w.log.Infof("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\Commitees", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)

			flag, err := w.CheckParentBlockEnough(epoch)

			if !flag {
				w.log.WithError(err).WithField("epoch", epoch).Warn("Insufficient parent blocks, retrying later")
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
				w.log.WithField("epoch", epoch+1).Trace("Mining aborted for new epoch")
			}

			// calculate the next epoch vdf
			inputHash := crypto.Hash(res0)

			if !w.vdf0.IsFinished() {

				err := w.vdf0.Abort()
				if err != nil {
					w.log.WithError(err).WithField("epoch", epoch+1).Warn("VDF0 abort failed")
				}
				w.log.WithField("epoch", epoch+1).Debug("VDF0 aborted for new epoch")
			}

			w.increaseEpoch()
			//w.vdf0 = types.NewVDFwithInput(w.getVDF0chan(), inputHash, w.config.PoT.Vdf0Iteration, w.ID)
			w.vdfhalf = types.NewVDFwithInput(w.vdfhalfchan, inputHash, w.config.PoT.Vdf1Iteration, w.ID)
			if err != nil {
				w.log.WithError(err).WithField("epoch", epoch+1).Warn("Failed to initialize VDF0")
				continue
			}

			w.log.WithField("epoch", epoch+1).Debug("Starting new epoch VDF0")
			w.timestamp = time.Now()
			go func() {
				err = w.vdfhalf.Exec(epoch + 1)
				if err != nil {
					w.log.WithError(err).Warn("VDF execution failed")
				}
			}()

			backupblock, err := w.blockStorage.GetbyHeight(epoch)
			w.log.WithFields(logrus.Fields{
				"next_epoch":    epoch + 1,
				"current_epoch": epoch,
				"block_count":   len(backupblock),
			}).Trace("Retrieved blocks for epoch")

			parentblock, uncleblock := w.blockSelection(backupblock, res0, epoch)

			if parentblock != nil {
				w.log.WithFields(logrus.Fields{
					"epoch":      epoch + 1,
					"hash":       utils.EncodeShortPrint(parentblock.Hash()),
					"difficulty": parentblock.GetHeader().Difficulty.Int64(),
					"from_node":  parentblock.GetHeader().Address,
				}).Debug("Selected parent block")
			} else {
				if len(backupblock) != 0 {
					//w.chainReader.SetHeight(epoch, backupblock[0])
					//parentblock = backupblock[0]
					//w.log.Infof("[PoT]\tepoch %d:parent block hash is nil,execset nil block %s as parent", epoch+1, hex.EncodeToString(parentblock.GetHeader().Hashes))
					continue
				} else {
					w.log.WithField("epoch", epoch+1).Error("No parent block found")
					panic(fmt.Errorf("[PoT]\tepoch %d:dont't find any parent block", epoch+1))
				}
			}

			_, err = w.GetExcutedTxsFromExecutor(epoch)
			if err != nil {
				w.log.WithError(err).WithField("epoch", epoch+1).Warn("Failed to get transactions from executor")
			} else {
				w.log.WithField("epoch", epoch+1).Trace("Retrieved transactions from executor")
			}

			_ = w.handleBlockExecutedHeader(parentblock)

			_, err = w.GetIncentiveTxFromExecutor(epoch)
			if err != nil {
				w.log.WithError(err).WithField("epoch", epoch+1).Warn("Failed to get incentive transactions from executor")
			} else {
				w.log.WithField("epoch", epoch+1).Trace("Retrieved incentive transactions from executor")
			}

			err = w.handleBlockRawTx(parentblock)
			if err != nil {
				w.log.WithError(err).WithFields(logrus.Fields{
					"epoch":       epoch + 1,
					"parent_hash": hexutil.Encode(parentblock.Hash()),
				}).Warn("Failed to handle transactions for block")
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

// handleVdfhalf processes VDF half computation results and manages the transition to VDF0 computation
func (w *Worker) handleVdfhalf() {
	for {
		select {
		case res := <-w.vdfhalfchan:
			epoch := w.getEpoch()
			w.log.WithFields(logrus.Fields{
				"epoch":    epoch,
				"vdf_hash": utils.EncodeShortPrint(crypto.Hash(res.Res)),
			}).Debug("Received VDF half result")

			// Validate epoch consistency
			if res.Epoch != epoch {
				w.log.WithFields(logrus.Fields{
					"current_epoch": epoch,
					"result_epoch":  res.Epoch,
				}).Warn("VDF half result epoch mismatch")
			}

			// Store VDF half result for later use in block completion
			w.blockStorage.SetVDFHalf(epoch, res.Res)
			w.log.WithField("epoch", epoch).Trace("VDF half result stored")

			// Abort any running VDF0 computation as we transition to new VDF0
			if !w.vdf0.IsFinished() {
				err := w.vdf0.Abort()
				if err != nil {
					w.log.WithError(err).WithField("epoch", epoch+1).Warn("VDF0 abort failed during half transition")
				}
				w.log.WithField("epoch", epoch+1).Debug("VDF0 aborted for VDF half transition")
			}

			// Start VDF0 computation with VDF half result as input
			inputhash := crypto.Hash(res.Res)
			w.vdf0 = types.NewVDFwithInput(w.getVDF0chan(), inputhash, w.config.PoT.Vdf0Iteration-w.config.PoT.Vdf1Iteration, w.ID)
			w.log.WithFields(logrus.Fields{
				"epoch":      epoch,
				"input_hash": utils.EncodeShortPrint(inputhash),
			}).Debug("Starting VDF0 computation from VDF half")

			// Execute VDF0 computation in a separate goroutine
			go func() {
				err := w.vdf0.Exec(epoch)
				if err != nil {
					w.log.WithError(err).WithField("epoch", epoch).Error("VDF0 execution failed")
				}
			}()
		}
	}
}

// GetExcutedTxsFromExecutor retrieves executed transaction blocks from the executor service
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
		w.log.WithError(err).WithFields(logrus.Fields{
			"epoch":        epoch,
			"start_height": w.executeheight,
		}).Error("Failed to get transactions from executor")
		return nil, err
	}

	executeblocks := response.GetBlocks()
	excuteheight := response.GetEnd()

	if excuteheight > w.executeheight {
		w.log.WithFields(logrus.Fields{
			"old_height": w.executeheight,
			"new_height": excuteheight,
		}).Trace("Updated execute height")
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

	w.log.WithFields(logrus.Fields{
		"epoch":       epoch,
		"block_count": len(blocks),
	}).Debug("Retrieved executed blocks from executor")
	return blocks, nil
}

// GetExecutedBlockFromMempool retrieves a batch of executed blocks from the mempool
func (w *Worker) GetExecutedBlockFromMempool() []*types.ExecutedBlock {
	ExecutedBlocks := w.mempool.GetFirstN(w.config.PoT.Batchsize)
	return ExecutedBlocks
}

// GetIncentiveTxFromExecutor retrieves incentive transactions from the executor service
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
	w.log.WithFields(logrus.Fields{
		"epoch":        epoch,
		"reward_count": len(bcireward),
	}).Trace("Processing BCI rewards from executor")

	for _, pbBcireward := range bcireward {
		BciReward := ToBciReward(pbBcireward)
		flag, _, err := w.VerifyBciReward(BciReward)
		if !flag {
			w.log.WithError(err).WithField("amount", BciReward.Amount).Warn("BCI reward verification failed")
			return nil, fmt.Errorf("the Bci reward is not valid for %s", err.Error())
		} else {
			w.log.WithFields(logrus.Fields{
				"amount":  BciReward.Amount,
				"address": hexutil.Encode(BciReward.Address),
			}).Debug("Adding BCI reward to mempool")
			w.mempool.AddBciReward(BciReward)
		}
	}
	if w.incentiveheight < response.GetEnd() {
		w.log.WithFields(logrus.Fields{
			"old_height": w.incentiveheight,
			"new_height": response.GetEnd(),
		}).Trace("Updated incentive height")
		w.incentiveheight = response.GetEnd()
	}
	return nil, nil
}

// SetBlockKeyMap stores the private key associated with a block hash
func (w *Worker) SetBlockKeyMap(privatekey []byte, blockhash [crypto.Hashlen]byte) {
	w.rwmutex.Lock()
	defer w.rwmutex.Unlock()
	w.blockKeyMap[blockhash] = privatekey
	w.log.WithField("block_hash", utils.EncodeShortPrint(blockhash[:])).Trace("Stored block private key")
}

// TryFindKey attempts to retrieve the private key for a given block hash
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

// TryFindCommiteeKey attempts to derive the committee key for a given block hash
func (w *Worker) TryFindCommiteeKey(blockhash [crypto.Hashlen]byte) (bool, *crypto.PrivateKey) {
	w.rwmutex.RLock()
	prikey := w.blockKeyMap[blockhash]
	w.rwmutex.RUnlock()

	if prikey != nil {
		// Retrieve block to get public key and PoT proof
		block, _ := w.blockStorage.Get(blockhash[:])
		if block == nil {
			w.log.WithField("block_hash", utils.EncodeShortPrint(blockhash[:])).Warn("Block not found for committee key generation")
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
			w.log.WithField("block_hash", utils.EncodeShortPrint(blockhash[:])).Warn("Failed to generate committee key")
			return false, nil
		}
		w.log.WithField("block_hash", utils.EncodeShortPrint(blockhash[:])).Trace("Generated committee key")
		return true, commiteekey
	} else {
		return false, nil
	}
}

// SetVdf0res stores the VDF0 result for a specific epoch in persistent storage
func (w *Worker) SetVdf0res(epoch uint64, vdf0 []byte) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.blockStorage.SetVDFres(epoch, vdf0)
}

// GetVdf0byEpoch retrieves the VDF0 result for a specific epoch from storage
func (w *Worker) GetVdf0byEpoch(epoch uint64) ([]byte, error) {
	return w.blockStorage.GetVDFresbyEpoch(epoch)
}

// getVDF0chan returns the channel for receiving VDF0 computation results
func (w *Worker) getVDF0chan() chan *types.VDF0res {
	return w.vdf0Chan
}

// checkVDFforepoch validates a VDF result for a specific epoch
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

// setVDF0epoch sets the worker's current epoch to a specific value
func (w *Worker) setVDF0epoch(epoch uint64) error {
	epochnow := w.getEpoch()
	if epochnow > epoch {
		return fmt.Errorf("could not execset for a outdated epoch %d", epoch)
	}

	// Abort any running VDF1 work before epoch change
	if w.IsVDF1Working() {
		err := w.vdf0.Abort()
		if err != nil {
			w.log.WithError(err).WithField("target_epoch", epoch).Error("VDF0 abort failed during epoch reset")
			return err
		}
		w.log.WithFields(logrus.Fields{
			"current_epoch": epochnow,
			"target_epoch":  epoch,
		}).Debug("VDF0 aborted for epoch reset")
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.epoch = epoch
	w.log.WithField("new_epoch", epoch).Info("Epoch reset complete")

	return nil
}

// getEpoch returns the current epoch number in a thread-safe manner
func (w *Worker) getEpoch() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	epoch := w.epoch
	return epoch
}

// increaseEpoch increments the current epoch number by 1 in a thread-safe manner
func (w *Worker) increaseEpoch() {
	w.mutex.Lock()
	w.epoch += 1
	w.mutex.Unlock()
}

func (w *Worker) self() string {
	return w.PeerId
}

// IsVDF1Working checks if the worker is currently performing VDF1 mining operations
func (w *Worker) IsVDF1Working() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Return the work flag status indicating if mining is active
	return w.workFlag
}

// setWorkFlagFalse sets the worker's mining flag to false
func (w *Worker) setWorkFlagFalse() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.workFlag = false
}

// SetEngine updates the worker's PoT engine reference and peer ID
func (w *Worker) SetEngine(engine *PoTEngine) {
	w.Engine = engine
	w.PeerId = w.Engine.GetPeerID()
}

// handleBlockExecutedHeader processes the executed headers in a block
func (w *Worker) handleBlockExecutedHeader(block *types.Block) error {
	executedHeaders := block.GetExecutedHeaders()
	w.log.WithFields(logrus.Fields{
		"block_height": block.GetHeader().Height,
		"header_count": len(executedHeaders),
	}).Trace("Processing executed headers")

	// Process each executed header
	for _, header := range executedHeaders {
		hash := header.Hash()
		exeblock := w.mempool.GetBlockByHash(hash)
		if exeblock != nil {
			// Store executed block in persistent storage
			err := w.blockStorage.PutExcutedBlock(exeblock)
			if err != nil {
				w.log.WithError(err).WithField("block_hash", utils.EncodeShortPrint(hash[:])).Error("Failed to store executed block")
				return err
			}
			w.log.WithField("block_hash", utils.EncodeShortPrint(hash[:])).Trace("Stored executed block")
		} else {
			w.log.WithField("block_hash", utils.EncodeShortPrint(hash[:])).Warn("Executed block not found in mempool")
			return fmt.Errorf("handle block excuted tx error for executedblock %s does not exist in mempool", utils.EncodeShortPrint(hash[:]))
		}
	}

	// Mark executed headers as proposed in mempool
	w.mempool.MarkProposedByHeader(executedHeaders)
	return nil
}

// handleBlockRawTx processes the raw transactions in a block
func (w *Worker) handleBlockRawTx(block *types.Block) error {

	txs := block.GetRawTx()
	//
	if len(block.GetRawTx()) != 0 {
		//fmt.Println(hexutil.Encode(block.GetRawTx()[0].Txid[:]))
	} else if len(txs) == 0 && block.Header.Height != 0 {
		return fmt.Errorf("block %s at %d has no tx", block.Hash(), block.Header.Height)
	} else {
		return nil
	}

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

// GetEpoch returns the current epoch
func (w *Worker) GetEpoch() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.epoch
}

// GetWorkFlag returns the current work flag status
func (w *Worker) GetWorkFlag() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.workFlag
}

// GetVDF0 returns the VDF0 instance
func (w *Worker) GetVDF0() *types.VDF {
	return w.vdf0
}

// GetVDF1 returns the VDF1 instances
func (w *Worker) GetVDF1() []*types.VDF {
	return w.vdf1
}

// GetVDFHalf returns the VDFHalf instance
func (w *Worker) GetVDFHalf() *types.VDF {
	return w.vdfhalf
}

// GetVDF0Chan returns the VDF0 channel
func (w *Worker) GetVDF0Chan() chan *types.VDF0res {
	return w.vdf0Chan
}

// GetVDFHalfChan returns the VDFHalf channel
func (w *Worker) GetVDFHalfChan() chan *types.VDF0res {
	return w.vdfhalfchan
}

// GetAbort returns the abort control
func (w *Worker) GetAbort() *Abortcontrol {
	return w.abort
}

// GetWhirly returns the whirly controller
func (w *Worker) GetWhirly() *nodeController.NodeController {
	return w.whirly
}

// GetExecuteHeight returns the execute height
func (w *Worker) GetExecuteHeight() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.executeheight
}

// GetIncentiveHeight returns the incentive height
func (w *Worker) GetIncentiveHeight() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.incentiveheight
}
