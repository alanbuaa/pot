package pot

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/elliotchance/orderedmap"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/whirly/simpleWhirly"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

var bigD = new(big.Int).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))

const (
	Commiteelen = 4
	cpuCounter  = 1
	NoParentD   = 2
)

type Worker struct {
	// basic info
	ID     int64
	PeerId string
	log    *logrus.Entry
	config *config.ConsensusConfig
	epoch  uint64

	// vdf work
	timestamp  time.Time
	vdf0       *types.VDF
	vdf0Chan   chan *types.VDF0res
	vdf1       []*types.VDF
	vdf1Chan   chan *types.VDF0res
	vdfChecker *vdf.Vdf
	abort      chan struct{}
	wg         *sync.WaitGroup
	workFlag   bool
	// rand seed
	rand         *rand.Rand
	blockCounter int

	Engine  *PoTEngine
	stopCh  chan struct{}
	mutex   sync.Mutex
	storage *types.HeaderStorage

	peerMsgQueue       chan *types.Header
	headerResponseChan chan *pb.HeaderResponse
	potResponseCh      chan *pb.PoTResponse
	potStorage         *types.PoTBlockStorage
	chainReader        *types.ChainReader
	// upper consensus
	whirly        *simpleWhirly.SimpleWhirlyImpl
	potSignalChan chan<- []byte
	committee     *orderedmap.OrderedMap
	Commitee      []string
}

func NewWorker(id int64, config *config.ConsensusConfig, logger *logrus.Entry, st *types.HeaderStorage, engine *PoTEngine) *Worker {
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

	peer := make(chan *types.Header, 5)

	w := &Worker{
		abort:        make(chan struct{}),
		Engine:       engine,
		config:       config,
		ID:           id,
		log:          logger,
		epoch:        0,
		vdf0:         vdf0,
		vdf0Chan:     ch0,
		vdf1:         vdf1,
		vdf1Chan:     ch1,
		wg:           new(sync.WaitGroup),
		rand:         rands,
		peerMsgQueue: peer,
		storage:      st,
		committee:    orderedmap.NewOrderedMap(),
		vdfChecker:   vdf.New("wesolowski_rust", []byte(""), potconfig.Vdf0Iteration, id),
		chainReader:  types.NewChainReader(st),
		PeerId:       engine.GetPeerID(),
		workFlag:     false,
	}
	w.Init()

	return w
}

func (w *Worker) Init() {
	// catchup
	// w.log.Infof("%d %d", w.config.PoT.Snum, w.config.PoT.Vdf1Iteration)
	w.vdf0.SetInput(crypto.Hash([]byte("aa")), w.config.PoT.Vdf0Iteration)
	w.SetVdf0res(0, []byte("aa"))
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

func (w *Worker) OnGetVdf0Response() {
	go w.handleBlock()

	for {
		select {
		// receive vdf0
		case res := <-w.vdf0Chan:
			epoch := w.getEpoch()
			timer := time.Since(w.timestamp) / time.Millisecond
			w.log.Errorf("[PoT]\tepoch %d:Receive epoch %d vdf0 res %s, use %d ms\n", epoch, res.Epoch, hexutil.Encode(crypto.Hash(res.Res)), timer)
			if epoch > res.Epoch {
				w.log.Errorf("[PoT]\tthe epoch already set")
				continue
			}

			// w.log.Errorf("[PoT]\tepoch %d:Receive epoch %d vdf0 res, use %d ms\n", epoch, res.Epoch, timer)
			w.increaseEpoch()

			// if epoch == 5 && w.ID == 3 {
			//	time.Sleep(time.Second * 15)
			// }
			// if epoch == 7 && w.ID == 2 {
			//	time.Sleep(time.Second * 20)
			// }

			if w.isMinerWorking() {
				close(w.abort)
				w.setWorkFlagFalse()
				w.wg.Wait()
				w.log.Infof("[PoT]\tepoch %d:the miner got abort for get in new epoch", epoch+1)
			}

			// the last epoch is over
			// epoch increase
			res0 := res.Res
			w.SetVdf0res(res.Epoch+1, res0)

			// calculate the next epoch vdf
			inputHash := crypto.Hash(res0)

			if !w.vdf0.Finished {
				err := w.vdf0.Abort()
				if err != nil {
					w.log.Errorf("[PoT]\tepoch %d: vdf0 abort error for %s", epoch+1, err)
				}
				w.log.Warnf("[PoT]\tepoch %d:vdf0 got abort for new epoch ", epoch+1)
			}
			backupblock, err := w.storage.GetbyHeight(epoch)

			w.log.Infof("[PoT]\tepoch %d:epoch %d block num %d", epoch+1, epoch, len(backupblock))

			if err != nil {
				w.log.Warn("[PoT]\tget backup block error :", err)
				continue
			}

			err = w.vdf0.SetInput(inputHash, w.config.PoT.Vdf0Iteration)
			if err != nil {
				w.log.Errorf("[PoT]\tepoch %d:set vdf0 error for %t", epoch+1, err)
				continue
			}

			w.log.Infof("[PoT]\tepoch %d:Start epoch %d vdf0", epoch+1, epoch+1)
			w.timestamp = time.Now()
			go func() {
				err = w.vdf0.Exec(epoch + 1)
				if err != nil {
					w.log.Info("[PoT]\texecute vdf error for :", err)
				}
			}()

			parentblock, uncleblock := w.blockSelection(backupblock, res0, epoch)
			if parentblock != nil {
				w.log.Errorf("[PoT]\tepoch %d:parent block hash is : %s Difficulty %d from %d", epoch+1, hex.EncodeToString(parentblock.Hash()), parentblock.Difficulty.Int64(), parentblock.Address)
			} else {
				if len(backupblock) != 0 {
					w.chainReader.SetHeight(epoch, backupblock[0])
					parentblock = backupblock[0]
					w.log.Errorf("[PoT]\tepoch %d:parent block hash is nil,set nil block %s as parent", epoch+1, hex.EncodeToString(parentblock.Hashes))
				} else {
					grandblock, err := w.chainReader.GetByHeight(epoch - 1)
					if err != nil {
						continue
					}
					parentblock = &types.Header{
						Height:     epoch - 1,
						ParentHash: grandblock.Hash(),
						UncleHash:  nil,
						Mixdigest:  nil,
						Difficulty: nil,
						Nonce:      0,
						Timestamp:  time.Time{},
						PoTProof:   nil,
						Address:    0,
						Hashes:     nil,
						PeerId:     w.PeerId,
					}
					parentblock.Hash()
				}
			}

			// if epoch > 1 {
			// 	w.simpleLeaderUpdate(parentblock)
			// }

			difficulty := w.calcDifficulty(parentblock, uncleblock)

			w.startWorking()
			w.abort = make(chan struct{})
			w.wg.Add(cpuCounter)
			for i := 0; i < cpuCounter; i++ {
				go w.mine(res0, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg)
			}

		}
	}
}

func (w *Worker) mine(vdf0res []byte, nonce int64, workerid int, abort chan struct{}, difficulty *big.Int, parentblock *types.Header, uncleblock []*types.Header, wg *sync.WaitGroup) *types.Header {
	defer w.setWorkFlagFalse()
	defer wg.Done()

	epoch := w.getEpoch()
	w.log.Infof("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
	mixdigest := w.calcMixdigest(epoch, parentblock, uncleblock, difficulty, w.ID)
	tmp := new(big.Int)
	tmp.SetInt64(nonce)
	noncebyte := tmp.Bytes()
	input := bytes.Join([][]byte{noncebyte, vdf0res, mixdigest}, []byte(""))
	hashinput := crypto.Hash(input)
	err := w.vdf1[workerid].SetInput(hashinput, w.config.PoT.Vdf1Iteration)
	if err != nil {
		return nil
	}
	vdfCh := w.vdf1Chan
	go w.vdf1[workerid].Exec(epoch)
	for {
		select {
		case res := <-vdfCh:
			res1 := res.Res
			// compare to target
			target := new(big.Int).Set(bigD)
			target = target.Div(bigD, difficulty)
			tmp.SetBytes(crypto.Hash(res1))
			if tmp.Cmp(target) >= 0 {

				block := w.createnilBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
				w.log.Infof("[PoT]\tepoch %d:fail to find a %d block, create a nil block %s", epoch, difficulty.Int64(), hexutil.Encode(block.Hash()))
				w.storage.Put(block)
				w.peerMsgQueue <- block
				// w.workFlag = false
				return block
			}
			// w.createBlock
			block := w.createBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
			w.blockCounter += 1
			w.log.Infof("[PoT]\tepoch %d:get new block %d", epoch, w.blockCounter)

			// broadcast the block
			w.peerMsgQueue <- block
			// w.workFlag = false
			return block
		case <-abort:
			err := w.vdf1[workerid].Abort()
			if err != nil {
				return nil
			}
			w.log.Infof("[PoT]\tepoch %d:vdf1 workerid %d got abort", epoch, workerid)
			// w.workFlag = false
			return nil
		}

	}
}

func (w *Worker) createBlock(epoch uint64, parentBlock *types.Header, uncleBlock []*types.Header, difficulty *big.Int, mixdigest []byte, nonce int64, vdf0res []byte, vdf1res []byte) *types.Header {
	parentblockhash := make([]byte, 0)
	if parentBlock != nil {
		parentblockhash = parentBlock.Hash()
	}
	uncleBlockhash := make([][]byte, len(uncleBlock))
	for i := 0; i < len(uncleBlock); i++ {
		uncleBlockhash[i] = uncleBlock[i].Hash()
	}
	Potproof := [][]byte{vdf0res, vdf1res}
	h := &types.Header{
		Height:     epoch,
		ParentHash: parentblockhash,
		UncleHash:  uncleBlockhash,
		Nonce:      nonce,
		Difficulty: difficulty,
		Mixdigest:  mixdigest,
		Address:    w.ID,
		PoTProof:   Potproof,
		PeerId:     w.PeerId,
	}
	h.Hashes = h.Hash()
	return h
}

func (w *Worker) createnilBlock(epoch uint64, parentBlock *types.Header, uncleBlock []*types.Header, difficulty *big.Int, mixdigest []byte, nonce int64, vdf0res []byte, vdf1res []byte) *types.Header {
	parentblockhash := make([]byte, 0)
	if parentBlock != nil {
		parentblockhash = parentBlock.Hash()
	}
	uncleBlockhash := make([][]byte, len(uncleBlock))
	for i := 0; i < len(uncleBlock); i++ {
		uncleBlockhash[i] = uncleBlock[i].Hash()
	}
	Potproof := [][]byte{vdf0res, vdf1res}

	h := &types.Header{
		Height:     epoch,
		ParentHash: parentblockhash,
		UncleHash:  uncleBlockhash,
		Nonce:      0,
		Difficulty: big.NewInt(0),
		Mixdigest:  mixdigest,
		Address:    w.ID,
		PoTProof:   Potproof,
		PeerId:     w.PeerId,
	}

	h.Hashes = h.Hash()
	return h
}

func (w *Worker) SetVdf0res(epocch uint64, vdf0 []byte) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.storage.SetVdfRes(epocch, vdf0)

}

func (w *Worker) GetVdf0byEpoch(epoch uint64) ([]byte, error) {
	return w.storage.GetPoTbyEpoch(epoch)
}

func (w *Worker) getVDF0chan() chan *types.VDF0res {
	return w.vdf0Chan
}

func (w *Worker) checkVDFforepoch(epoch uint64, vdfres []byte) bool {
	epoch1, err := w.storage.GetPoTbyEpoch(epoch + 1)
	// we have next epoch vdfres
	if err == nil && len(epoch1) != 0 {
		return bytes.Equal(vdfres, epoch1)
	}
	epoch0, err := w.storage.GetPoTbyEpoch(epoch)
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
		return fmt.Errorf("[PoT]\tepoch %d: could not set for a outdated epoch %d", epochnow, epoch)
	}

	if w.isMinerWorking() {
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

func (w *Worker) calcDifficulty(parentblock *types.Header, uncleBlock []*types.Header) *big.Int {
	if parentblock == nil || parentblock.Difficulty.Cmp(common.Big0) == 0 {
		return big.NewInt(NoParentD)
	}

	diff := new(big.Int)
	diff.Set(parentblock.Difficulty)
	for _, block := range uncleBlock {
		diff.Add(diff, block.Difficulty)
	}
	snum := new(big.Int)
	snum.SetInt64(w.config.PoT.Snum)
	diffculty := diff.Div(diff, snum)
	if diffculty.Cmp(big.NewInt(0)) == 0 {
		return diffculty.SetInt64(1)
	}
	return diffculty
}

func (w *Worker) calcMixdigest(epoch uint64, parentblock *types.Header, uncleblock []*types.Header, difficulty *big.Int, ID int64) []byte {
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
	tmp.SetInt64(ID)
	IDBytes := tmp.Bytes()
	tmp.SetInt64(int64(epoch))
	epochBytes := tmp.Bytes()
	hashinput := bytes.Join([][]byte{epochBytes, parentblockhash, uncleblockhash, difficultyBytes, IDBytes}, []byte(""))
	res := sha256.Sum256(hashinput)
	return res[:]
}

func (w *Worker) caldifficultyExp(parentblock *types.Header, uncleBlock []*types.Header) *big.Int {

	D := new(big.Int).Set(parentblock.Difficulty)
	diff := new(big.Int).Set(parentblock.Difficulty)
	for _, block := range uncleBlock {
		diff = diff.Add(diff, block.Difficulty)
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

func (w *Worker) blockSelection(headers []*types.Header, vdf0res []byte, height uint64) (parent *types.Header, uncle []*types.Header) {
	sr := crypto.Hash(vdf0res)
	maxweight := big.NewInt(0)
	max := -1

	current, err := w.chainReader.GetByHeight(height - 1)

	if err != nil {
		return nil, nil
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(headers) == 0 {
		return nil, nil
	}

	readyblocks := make([]*types.Header, 0)
	for _, block := range headers {
		if bytes.Equal(current.Hashes, block.ParentHash) || block.ParentHash == nil {
			readyblocks = append(readyblocks, block)
			hashinput := append(block.Hash(), sr...)
			tmp := new(big.Int).Div(bigD, new(big.Int).SetBytes(crypto.Hash(hashinput)))
			weight := new(big.Int).Mul(block.Difficulty, tmp)
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

	w.chainReader.SetHeight(parent.Height, parent)

	return parent, uncle
}

func (w *Worker) self() int64 {
	return w.ID
}

func (w *Worker) isMinerWorking() bool {

	// for i := 0; i < len(w.vdf1); i++ {
	//	if !w.vdf1[i].Finished {
	//		return true
	//	}
	// }
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.workFlag
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
