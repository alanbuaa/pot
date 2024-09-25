package pot

import (
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/vdf"
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/nodeController"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

var bigD = new(big.Int).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))

const (
	Commiteelen        = 4
	CommiteeDelay      = 1
	cpuCounter         = 1
	NoParentD          = 2
	Batchsize          = 100
	Selectn            = 1
	TotalReward        = 10000
	BackupCommiteeSize = 64
	ConfirmDelay       = 6
	CandidateKeyLen    = 8 // 候选公钥列表大小
	Commitees          = 4 // 委员会大小
)

type Abortcontrol struct {
	abortchannel chan struct{}
	once         *sync.Once
}

func NewAbortcontrol() *Abortcontrol {
	return &Abortcontrol{
		abortchannel: make(chan struct{}),
		once:         new(sync.Once),
	}
}

type Worker struct {
	// basic info
	ID     int64
	PeerId string
	log    *logrus.Entry
	config *config.ConsensusConfig
	epoch  uint64

	// vdf work
	timestamp      time.Time
	vdf0           *types.VDF
	vdf0Chan       chan *types.VDF0res
	vdf1           []*types.VDF
	vdf1Chan       chan *types.VDF0res
	vdfChecker     *vdf.Vdf
	abort          *Abortcontrol
	wg             *sync.WaitGroup
	workFlag       bool
	blockKeyMap    map[[crypto.Hashlen]byte][]byte
	executeheight  uint64
	mempool        *Mempool
	chainresetflag bool

	// rand seed
	rand         *rand.Rand
	blockCounter int

	Engine  *PoTEngine
	stopCh  chan struct{}
	mutex   *sync.Mutex
	rwmutex *sync.RWMutex

	// communication
	peerMsgQueue      chan *types.Block
	blockResponseChan chan *pb.BlockResponse
	potResponseCh     chan *pb.PoTResponse
	blockStorage      *types.BlockStorage
	chainReader       *ChainReader
	listener          net.Listener
	rpcserver         *grpc.Server

	// upper consensus
	whirly         *nodeController.NodeController
	potSignalChan  chan<- []byte
	CommiteeNum    int32
	Commitee       [][]string
	Shardings      []Sharding
	BackupCommitee []string
	SelfAddress    []string
	CryptoSet      *CryptoSet
	CryptoSetMap   map[[crypto.Hashlen]byte]*CryptoSet
	// committee     *orderedmap.OrderedMap
}

func NewWorker(id int64, config *config.ConsensusConfig, logger *logrus.Entry, bst *types.BlockStorage, engine *PoTEngine) *Worker {
	ch0 := make(chan *types.VDF0res, 2048)
	ch1 := make(chan *types.VDF0res, 2048)
	potconfig := config.PoT
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil
	}
	rands := rand.New(rand.NewSource(seed.Int64()))

	vdf0 := types.NewVDF(ch0, potconfig.Vdf0Iteration, id)
	vdf1 := make([]*types.VDF, cpuCounter)
	for i := 0; i < cpuCounter; i++ {
		vdf1[i] = types.NewVDF(ch1, potconfig.Vdf1Iteration, id)
	}
	listen, err := net.Listen("tcp", config.Nodes[id].DciRpcAddress)
	if err != nil {
		panic(err)
	}

	peer := make(chan *types.Block, 5)
	keyblockmap := make(map[[crypto.Hashlen]byte][]byte)
	mempool := NewMempool()
	Commitee := make([][]string, 0)
	BackupCommitee := make([]string, BackupCommiteeSize)

	aborts := &Abortcontrol{
		abortchannel: make(chan struct{}),
		once:         new(sync.Once),
	}

	cryptoset := &CryptoSet{
		BigN:   BigN,
		SmallN: SmallN,
		G:      bls12381.NewG1().One(),
		// TODO edit H
		H:                     bls12381.NewG1().MulScalar(bls12381.NewG1().New(), bls12381.NewG1().One(), bls12381.FrFromInt(5731132)),
		LocalSRS:              nil,
		PrevShuffledPKList:    nil,
		PrevRCommitForShuffle: bls12381.NewG1().One(),
		PrevRCommitForDraw:    bls12381.NewG1().One(),
		CommitteeMarkQueue:    new(Queue),
		Threshold:             3,
	}

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
		wg:            new(sync.WaitGroup),
		rand:          rands,
		peerMsgQueue:  peer,
		mutex:         new(sync.Mutex),
		rwmutex:       new(sync.RWMutex),
		executeheight: uint64(0),
		// storage:      st,
		mempool:      mempool,
		blockStorage: bst,
		// committee:    orderedmap.NewOrderedMap(),
		vdfChecker:     vdf.New("wesolowski_rust", []byte(""), potconfig.Vdf0Iteration, id),
		chainReader:    NewChainReader(bst),
		PeerId:         engine.GetPeerID(),
		workFlag:       false,
		blockKeyMap:    keyblockmap,
		Commitee:       Commitee,
		CommiteeNum:    int32(1),
		BackupCommitee: BackupCommitee,
		chainresetflag: false,
		CryptoSet:      cryptoset,
		CryptoSetMap:   make(map[[crypto.Hashlen]byte]*CryptoSet),
	}
	rpcserver := grpc.NewServer()
	pb.RegisterDciExectorServer(rpcserver, w)
	w.rpcserver = rpcserver
	w.listener = listen

	w.Init()

	return w
}

func (w *Worker) Init() {
	// catchup
	// w.log.Infof("%d %d", w.config.PoT.Snum, w.config.PoT.Vdf1Iteration)
	w.vdf0.SetInput(crypto.Hash([]byte("aa")), w.config.PoT.Vdf0Iteration)
	w.SetVdf0res(0, []byte("aa"))
	w.blockStorage.Put(types.DefaultGenesisBlock())
	w.CryptoSetMap[crypto.Convert(types.DefaultGenesisBlock().Hash())] = w.CryptoSet
	w.blockCounter = 0

}
func (w *Worker) startWorking() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.workFlag = true
}

func (w *Worker) Work() {
	w.timestamp = time.Now()
	w.log.Infof("[PoT]\tStart epoch %d vdf0", w.getEpoch())

	err := w.vdf0.Exec(0)
	if err != nil {
		return
	}
}
func (w *Worker) WaitandReset(res *types.VDF0res) {
	time.Sleep(10 * time.Second)
	w.vdf0Chan <- res
}

func (w *Worker) OnGetVdf0Response() {
	go w.handleBlock()

	for {
		select {
		// receive vdf0
		case res := <-w.vdf0Chan:
			epoch := w.getEpoch()
			timer := time.Since(w.timestamp) / time.Millisecond
			w.log.Infof("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\n", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)
			// time.Sleep(10 * time.Second)

			if epoch > res.Epoch {
				w.log.Errorf("[PoT]\tthe epoch already execset")
				continue
			}
			if len(res.Res) == 0 {
				w.log.Errorf("[PoT]\treceive vdf0 res is empty")
			}

			// timestop := math.Floor(float64(timer) * float64(10-w.config.PoT.Slowrate) / float64(w.config.PoT.Slowrate))
			// time.Sleep(time.Duration(timestop) * time.Millisecond)
			//
			// timer = time.Since(w.timestamp) / time.Millisecond
			// w.log.Infof("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\Commitees", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)

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
			w.vdf0 = types.NewVDFwithInput(w.getVDF0chan(), inputHash, w.config.PoT.Vdf0Iteration, w.ID)
			if err != nil {
				w.log.Warnf("[PoT]\tepoch %d:execset vdf0 error for %t", epoch+1, err)
				continue
			}

			w.log.Debugf("[PoT]\tepoch %d:Start epoch %d vdf0", epoch+1, epoch+1)
			w.timestamp = time.Now()
			go func() {
				err = w.vdf0.Exec(epoch + 1)
				if err != nil {
					w.log.Info("[PoT]\texecute vdf error for :", err)
				}
			}()

			backupblock, err := w.blockStorage.GetbyHeight(epoch)
			w.log.Infof("[PoT]\tepoch %d:epoch %d block num %d", epoch+1, epoch, len(backupblock))

			parentblock, uncleblock := w.blockSelection(backupblock, res0, epoch)

			if parentblock != nil {
				w.log.Infof("[PoT]\tepoch %d:parent block hash is : %s Difficulty %d from %d", epoch+1, hex.EncodeToString(parentblock.Hash()), parentblock.GetHeader().Difficulty.Int64(), parentblock.GetHeader().Address)
			} else {
				if len(backupblock) != 0 {
					// w.chainReader.SetHeight(epoch, backupblock[0])
					// parentblock = backupblock[0]
					// w.log.Infof("[PoT]\tepoch %d:parent block hash is nil,execset nil block %s as parent", epoch+1, hex.EncodeToString(parentblock.GetHeader().Hashes))
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
			err = w.handleBlockRawTx(parentblock)
			if err != nil {
				w.log.Errorf("[PoT]\tepoch %d: Handle Txs for block %s err for %s", epoch+1, hexutil.Encode(parentblock.Hash()), err)
			}

			if parentblock.Header.Height > 0 {
				err = w.UpdateLocalCryptoSetByBlock(parentblock.GetHeader().Height, parentblock)
				if err != nil {
					w.log.Errorf("[PoT]\tepoch %v: update local crypto set err: for %v", epoch+1, err)
				}
			}

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

			for i := 0; i < cpuCounter; i++ {
				go w.mine(epoch+1, vdf0rescopy, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg)
			}

		}
	}
}

func (w *Worker) mine(epoch uint64, vdf0res []byte, nonce int64, workerid int, abort *Abortcontrol, difficulty *big.Int, parentblock *types.Block, uncleblock []*types.Block, wg *sync.WaitGroup) *types.Block {
	defer wg.Done()
	defer w.setWorkFlagFalse()

	w.log.Infof("[PoT]\tepoch %d:workerid %d start to mine", epoch, workerid)
	mixdigest := w.calcMixdigest(epoch, parentblock, uncleblock, difficulty, w.PeerId)
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

				// block := w.createNilBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
				w.log.Infof("[PoT]\tepoch %d:workerid %d fail to find a %d block", epoch, workerid, difficulty.Int64())
				// w.blockStorage.Put(block)

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
			block := w.createBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
			w.blockCounter += 1
			w.log.Infof("[PoT]\tepoch %d:get new block %d", epoch, w.blockCounter)

			// broadcast the block
			w.peerMsgQueue <- block

			// begin new work
			nonce += 1
			tmp.SetInt64(nonce)
			noncebyte := tmp.Bytes()
			input := bytes.Join([][]byte{noncebyte, vdf0res, mixdigest}, []byte(""))
			hashinput := crypto.Hash(input)
			w.vdf1[workerid] = types.NewVDFwithInput(w.vdf1Chan, hashinput, w.config.PoT.Vdf1Iteration, w.ID)

			go func() {
				w.log.Debugf("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
				err := w.vdf1[workerid].Exec(epoch)
				if err != nil {
					return
				}
			}()

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

func (w *Worker) createBlock(epoch uint64, parentBlock *types.Block, uncleBlock []*types.Block, difficulty *big.Int, mixdigest []byte, nonce int64, vdf0res []byte, vdf1res []byte) *types.Block {
	exeblocks := w.GetExecutedBlockFromMempool()
	exeheader := make([]*types.ExecuteHeader, 0)
	for _, exeblock := range exeblocks {
		exeheader = append(exeheader, exeblock.Header)
	}
	// w.log.Infof("Txs len %d", len(Txs))
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
	// publicKeyBytes32 := crypto.Convert(publicKeyBytes)

	coinbasetx := w.GenerateCoinbaseTx(publicKeyBytes, vdf0res, TotalReward)
	Txs := []*types.Tx{coinbasetx}
	rawtxs := w.mempool.GetRawTx()
	for _, rawtx := range rawtxs {
		fmt.Println(hexutil.Encode(rawtx.Txid[:]), 123456)
		txbyte, err := rawtx.EncodeToByte()
		if err != nil {
			return nil
		}
		Txs = append(Txs, &types.Tx{Data: txbyte})
	}
	txshash := crypto.ComputeMerkleRoot(types.Txs2Bytes(Txs))
	id := w.ID
	peerid := w.PeerId

	cryptoset, err := w.GenerateCryptoSetFromLocal(epoch)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	h := &types.Header{
		Height:        epoch,
		ParentHash:    parentblockhash,
		UncleHash:     uncleBlockhash,
		Mixdigest:     mixdigest,
		Difficulty:    difficulty,
		Nonce:         nonce,
		Timestamp:     time.Now(),
		PoTProof:      PotProof,
		Address:       id,
		PeerId:        peerid,
		TxHash:        txshash,
		Hashes:        nil,
		PublicKey:     publicKeyBytes,
		CryptoElement: cryptoset,
	}

	h.Hashes = h.Hash()
	w.SetKeyBlockMap(privateKey.Private(), crypto.Convert(h.Hash()))

	return &types.Block{
		Header:     h,
		Txs:        Txs,
		ExeHeaders: exeheader,
	}
}

func (w *Worker) GetExcutedTxsFromExecutor(epoch uint64) ([]*types.ExecutedBlock, error) {
	conn, err := grpc.Dial(w.config.PoT.ExcutorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))
	if err != nil {
		return nil, err
	}
	client := pb.NewPoTExecutorClient(conn)
	request := &pb.GetTxRequest{
		StartHeight: w.executeheight,
		Des:         w.config.PoT.ExcutorAddress,
	}
	response, err := client.GetTxs(context.Background(), request)
	if err != nil {
		// w.log.Errorf("[PoT]\tGet Txs from executor error for %s",err)
		return nil, err
	}

	executeblocks := response.GetBlocks()
	excuteheight := response.GetEnd()

	if excuteheight > w.executeheight {
		w.executeheight = excuteheight
	}

	// executedtxs := make([]*types.ExecutedTxData, 0)
	// for i := 0; i < len(executeblocks); i++ {
	// height := executeblocks[i].GetHeader().GetHeight()
	// executedTxs := executeblocks[i].GetTxs()
	// for _, executedtx := range executedTxs {
	//	exectx := &types.ExecutedTxData{
	//		ExecutedHeight: height,
	//		TxHash:         executedtx.GetTxHash(),
	//	}
	//	//exectxdata, _ := exectx.EncodeToByte()
	//	executedtxs = append(executedtxs, exectx)
	//	w.mempool.Add(exectx)
	// }

	// }
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

func (w *Worker) GetExecutedBlockFromMempool() []*types.ExecutedBlock {
	ExecutedBlocks := w.mempool.GetFirstN(w.config.PoT.Batchsize)

	return ExecutedBlocks
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
	// privkeybyte32 := crypto.Convert(privateKey.Private())

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

	w.SetKeyBlockMap(privateKey.Private(), crypto.Convert(h.Hash()))

	return &types.Block{
		Header: h,
		Txs:    make([]*types.Tx, 0),
	}
}

func (w *Worker) SetKeyBlockMap(privatekey []byte, blockhash [crypto.Hashlen]byte) {
	w.rwmutex.Lock()
	defer w.rwmutex.Unlock()
	w.blockKeyMap[blockhash] = privatekey
}

func (w *Worker) TryFindKey(blockhash [crypto.PrivateKeyLen]byte) (bool, []byte) {
	w.rwmutex.RLock()
	prikey := w.blockKeyMap[blockhash]
	w.rwmutex.RUnlock()
	if prikey != nil {
		return true, prikey
	} else {
		return false, nil
	}
}

func (w *Worker) SetVdf0res(epocch uint64, vdf0 []byte) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.blockStorage.SetVDFres(epocch, vdf0)

}

func (w *Worker) GetVdf0byEpoch(epoch uint64) ([]byte, error) {
	return w.blockStorage.GetVDFresbyEpoch(epoch)
}

func (w *Worker) getVDF0chan() chan *types.VDF0res {
	return w.vdf0Chan
}

func (w *Worker) checkVDFforepoch(epoch uint64, vdfres []byte) bool {
	epoch1, err := w.blockStorage.GetVDFresbyEpoch(epoch + 1)
	// we have next epoch vdfres
	if err == nil && len(epoch1) != 0 {
		return bytes.Equal(vdfres, epoch1)
	}
	epoch0, err := w.blockStorage.GetVDFresbyEpoch(epoch)
	// we don't have next epoch vdfres, but we have now vdfres
	if err == nil && len(epoch0) != 0 {
		return w.vdfChecker.CheckVDF(epoch0, vdfres)
	}

	return false
}

// TODO:
func (w *Worker) setVDF0epoch(epoch uint64) error {
	epochnow := w.getEpoch()
	if epochnow > epoch {
		return fmt.Errorf("could not execset for a outdated epoch %d", epoch)
	}

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

func (w *Worker) getEpoch() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	epoch := w.epoch
	return epoch
}

func (w *Worker) increaseEpoch() {
	w.mutex.Lock()
	w.epoch += 1
	w.mutex.Unlock()
}

func (w *Worker) calcDifficulty(parentblock *types.Block, uncleBlock []*types.Block) *big.Int {
	if parentblock == nil || parentblock.GetHeader().Difficulty.Cmp(common.Big0) == 0 {
		return big.NewInt(NoParentD)
	}

	diff := new(big.Int)
	diff.Set(parentblock.GetHeader().Difficulty)
	for _, block := range uncleBlock {
		header := block.GetHeader()
		diff.Add(diff, header.Difficulty)
	}

	snum := new(big.Int)
	snum.SetInt64(w.config.PoT.Snum)
	diffculty := diff.Div(diff, snum)
	// w.log.Infof("[PoT]\tSum difficulty is %d and blocks num is %d, next is %d", diff.Int64(), len(uncleBlock)+1, diffculty.Int64())

	if diffculty.Cmp(big.NewInt(0)) == 0 {
		return diffculty.SetInt64(1)
	}
	return diffculty
}

func (w *Worker) calcMixdigest(epoch uint64, parentblock *types.Block, uncleblock []*types.Block, difficulty *big.Int, peerid string) []byte {
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
	// tmp.SetInt64(ID)
	IDBytes := []byte(peerid)
	tmp.SetInt64(int64(epoch))
	epochBytes := tmp.Bytes()
	hashinput := bytes.Join([][]byte{epochBytes, parentblockhash, uncleblockhash, difficultyBytes, IDBytes}, []byte(""))
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
	if count < w.config.PoT.Snum/2 {
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

			readyblocks = append(readyblocks, block)
			hashinput := append(block.Hash(), sr...)

			tmp := new(big.Int).Mul(bigD, blockheader.Difficulty)
			weight := new(big.Int).Div(tmp, new(big.Int).SetBytes(crypto.Hash(hashinput)))
			if weight.Cmp(maxweight) > 0 {
				max = len(readyblocks)
				maxweight.Set(weight)
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

func (w *Worker) IsVDF1Working() bool {

	// for i := 0; i < len(w.vdf1); i++ {
	//	if !w.vdf1[i].Finished {
	//		return true
	//	}
	// }
	w.mutex.Lock()
	defer w.mutex.Unlock()
	// for i := 0; i < cpuCounter; i++ {
	//	_, ok := <-w.vdf1[i].OutputChan
	//	if ok {
	//		return true
	//	}
	// }

	if w.workFlag {
		return true
	}
	return false
}

func (w *Worker) setWorkFlagFalse() {
	w.mutex.Lock()
	w.mutex.Unlock()
	w.workFlag = false
}

func (w *Worker) SetEngine(engine *PoTEngine) {
	w.Engine = engine
	w.PeerId = w.Engine.GetPeerID()
}

func (w *Worker) handleBlockExecutedHeader(block *types.Block) error {
	executedHeaders := block.GetExecutedHeaders()
	// fmt.Println("len of executed headers is :", len(executedHeaders))
	for _, header := range executedHeaders {
		hash := header.Hash()
		exeblock := w.mempool.GetBlockByHash(hash)
		if exeblock != nil {
			err := w.blockStorage.PutExcutedBlock(exeblock)
			// w.log.Error("the exe block height ", exeblock.Header.Height)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("handle block excuted tx error for executedblock %s does not exist in mempool", hexutil.Encode(hash[:]))
		}

	}
	w.mempool.MarkProposedByHeader(executedHeaders)
	return nil
}

func (w *Worker) handleBlockRawTx(block *types.Block) error {
	// w.log.Errorf("len of rawtx is %d", len(block.GetRawTx()))
	// if len(block.GetRawTx()) != 0 {
	//	fmt.Println(hexutil.Encode(block.GetRawTx()[0].Txid[:]))
	// }

	txs := block.GetRawTx()
	for _, tx := range txs {
		if tx.IsCoinBase() {
			dciproofs := tx.CoinbaseProofs
			w.mempool.MarkDciRewardProposed(dciproofs)
			if len(tx.TxOutput) == 2 {
				fmt.Println(hexutil.Encode(tx.Txid[:]), tx.TxOutput[1].Value)
			}
		}
	}
	err := w.chainReader.UpdateTxForBlock(block)
	if err != nil {
		return err
	}
	if block.Header.Height > ConfirmDelay {
		block, err := w.chainReader.GetByHeight(block.Header.Height - ConfirmDelay)
		if err != nil {
			return err
		}
		rawtxs := block.GetRawTx()
		for _, tx := range rawtxs {
			if !tx.IsCoinBase() {
				txid := tx.Txid
				transactionbyte, ok := w.mempool.rawmap[txid]
				if !ok {
					return fmt.Errorf("handle block raw tx error for tx %s does not exist in mempool", hexutil.Encode(txid[:]))
				}
				ptx := &pb.Transaction{
					Type:    pb.TransactionType_NORMAL,
					Payload: transactionbyte,
				}

				btx, err := proto.Marshal(ptx)
				if err != nil {
					return err
				}

				sharding := []byte(hexutil.EncodeUint64(1))
				request := &pb.Request{
					Tx:       btx,
					Sharding: sharding,
				}
				w.Engine.UpperConsensus.GetRequestEntrance() <- request
			}
		}

	}

	return nil
}

func (w *Worker) stop() {
	fill, err := os.OpenFile(fmt.Sprintf("./logs/blockmyself-%d", w.ID), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	fill.WriteString("")
	w.rpcserver.Stop()
	w.listener.Close()
}
