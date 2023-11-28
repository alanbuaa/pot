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
	Vdf0Iteration = 100000
	vdf1Iteration = 60000
	cpucounter    = 1
	NoParentD     = 1
)

type Worker struct {
	//basic info
	ID     int64
	Peerid string
	log    *logrus.Entry
	config *config.ConsensusConfig
	epoch  uint64

	//vdf work
	timestamp  time.Time
	vdf0       *types.VDF
	vdf0Chan   chan *types.VDF0res
	vdf1       []*types.VDF
	vdf1Chan   chan *types.VDF0res
	vdfchecker *vdf.Vdf
	abort      chan struct{}
	wg         *sync.WaitGroup

	//rand seed
	rand         *rand.Rand
	blockcounter int

	Engine   *PoTEngine
	stopCh   chan struct{}
	synclock sync.Mutex
	storage  *types.HeaderStorage

	peerMsgQueue     chan *types.Header
	headerResponsech chan *pb.HeaderResponse
	potResponseCh    chan *pb.PoTResponse
	potstorage       *types.PoTBlockStorage
	chainreader      *types.ChainReader
	//upperconsensus
	whirly     *simpleWhirly.SimpleWhirlyImpl
	potsigchan chan<- []byte
	commitee   *orderedmap.OrderedMap
}

func NewWorker(id int64, config *config.ConsensusConfig, logger *logrus.Entry, st *types.HeaderStorage, engine *PoTEngine) *Worker {
	ch0 := make(chan *types.VDF0res, 2048)
	ch1 := make(chan *types.VDF0res, 2048)

	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil
	}
	rands := rand.New(rand.NewSource(seed.Int64()))

	vdf0 := types.NewVDF(ch0, Vdf0Iteration, id)
	vdf1 := make([]*types.VDF, cpucounter)
	for i := 0; i < cpucounter; i++ {
		vdf1[i] = types.NewVDF(ch1, vdf1Iteration, id)
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
		commitee:     orderedmap.NewOrderedMap(),
		vdfchecker:   vdf.New("wesolowski_rust", []byte(""), Vdf0Iteration, id),
		chainreader:  types.NewChainReader(st),
		Peerid:       engine.GetPeerID(),
	}
	w.Init()

	return w
}

func (w *Worker) Init() {
	// catchup
	w.vdf0.SetInput(crypto.Hash([]byte("aa")), Vdf0Iteration)
	w.SetVdf0res(0, []byte("aa"))
	w.blockcounter = 0

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
		//receive vdf0
		case res := <-w.vdf0Chan:

			epoch := w.getEpoch()
			if epoch > res.Epoch {

				w.log.Errorf("[PoT]\tthe epoch already set")
				continue
			}
			timer := time.Since(w.timestamp) / time.Millisecond
			w.log.Errorf("[PoT]\tepoch %d :Receive epoch %d vdf0 res, use %d ms\n", epoch, res.Epoch, timer)
			w.increaseEpoch()

			if w.isMinerWorking() {
				if w.abort != nil {
					close(w.abort)
					w.wg.Wait()
				}
			}

			// the last epoch is over
			// epoch increase
			res0 := res.Res
			w.SetVdf0res(res.Epoch+1, res0)

			// calculate the next epoch vdf
			inputhash := crypto.Hash(res0)

			err := w.vdf0.SetInput(inputhash, Vdf0Iteration)
			if err != nil {
				return
			}
			// TODO: handle vdf error

			w.log.Infof("[PoT]\tepoch %d:Start epoch %d vdf0", epoch+1, epoch+1)
			w.timestamp = time.Now()

			go func() {
				err = w.vdf0.Exec(epoch + 1)
				if err != nil {

				}
			}()

			backupblock, err := w.storage.GetbyHeight(epoch)

			w.log.Infof("[PoT]\tepoch %d:epoch %d block num %d", epoch+1, epoch, len(backupblock))

			if err != nil {
				w.log.Warn("[PoT]\tget backup block error :", err)
			}

			parentblock, uncleblock := w.blockSelection(backupblock, res0)
			if parentblock != nil {
				w.log.Errorf("[PoT]\tepoch %d parent block hash is : %s Difficulty %d from %d", epoch+1, hex.EncodeToString(parentblock.Hash()), parentblock.Difficulty.Int64(), parentblock.Address)
			} else {

				if len(backupblock) != 0 {
					w.chainreader.SetHeight(epoch, backupblock[0])
					parentblock = backupblock[0]
					w.log.Errorf("[PoT]\tepoch %d parent block hash is nil,set nil block %s as parent", epoch+1, hex.EncodeToString(parentblock.Hashes))
				}
			}
			if epoch > 1 {
				w.simpleLeaderUpdate(parentblock)
			}
			difficulty := w.calcDifficulty(parentblock, uncleblock)

			w.abort = make(chan struct{})

			w.wg.Add(cpucounter)
			for i := 0; i < cpucounter; i++ {
				go w.mine(res0, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg)
			}

		}
	}
}

func (w *Worker) mine(vdf0res []byte, nonce int64, workerid int, abort chan struct{}, difficulty *big.Int, parentblock *types.Header, uncleblock []*types.Header, wg *sync.WaitGroup) *types.Header {
	defer wg.Done()

	epoch := w.getEpoch()
	w.log.Infof("[PoT]\tepoch %d:Start run vdf1 %d to mine", epoch, workerid)
	mixdigest := w.calcMixdigest(epoch, parentblock, uncleblock, difficulty, w.ID)
	tmp := new(big.Int)
	tmp.SetInt64(nonce)
	noncebyte := tmp.Bytes()
	input := bytes.Join([][]byte{noncebyte, vdf0res, mixdigest}, []byte(""))
	hashinput := crypto.Hash(input)
	err := w.vdf1[workerid].SetInput(hashinput, vdf1Iteration)
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
				return block
			}
			// w.createBlock
			block := w.createBlock(epoch, parentblock, uncleblock, difficulty, mixdigest, nonce, vdf0res, res1)
			w.blockcounter += 1
			w.log.Infof("[PoT]\tepoch %d: get new block %d", epoch, w.blockcounter)

			// broadcast the block
			w.peerMsgQueue <- block
			return block
		case <-abort:
			err := w.vdf1[workerid].Abort()
			if err != nil {
				return nil
			}
			w.log.Infof("[PoT]\tepoch %d:vdf1 workerid %d got abort", epoch, workerid)
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
		PeerId:     w.Peerid,
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
		PeerId:     w.Peerid,
	}

	h.Hashes = h.Hash()
	return h
}

func (w *Worker) SetVdf0res(epocch uint64, vdf0 []byte) {
	w.synclock.Lock()
	defer w.synclock.Unlock()
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
	//we have next epoch vdfres
	if err == nil && len(epoch1) != 0 {
		return bytes.Equal(vdfres, epoch1)
	}
	epoch0, err := w.storage.GetPoTbyEpoch(epoch)
	//we don't have next epoch vdfres, but we have now vdfres
	if err == nil && len(epoch0) != 0 {
		return w.vdfchecker.CheckVDF(epoch0, vdfres)
	}

	return false
}

// TODO:
func (w *Worker) setVDF0epoch(epoch uint64) error {
	epochnow := w.getEpoch()
	if epochnow > epoch {
		return fmt.Errorf("[PoT]\tepoch %d: could not set for a outdated epoch %d", epochnow, epoch)
	}

	if !w.vdf0.IsFinished() {
		err := w.vdf0.Abort()
		if err != nil {
			return err
		}
		w.log.Errorf("[PoT]\tVDF0 got abort")
	}
	w.synclock.Lock()
	defer w.synclock.Unlock()
	w.epoch = epoch

	return nil
}

func (w *Worker) getEpoch() uint64 {
	w.synclock.Lock()
	defer w.synclock.Unlock()
	epoch := w.epoch
	return epoch
}
func (w *Worker) increaseEpoch() {
	w.synclock.Lock()
	w.epoch += 1
	w.synclock.Unlock()
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

func (w *Worker) blockSelection(headers []*types.Header, vdf0res []byte) (parent *types.Header, uncle []*types.Header) {
	sr := crypto.Hash(vdf0res)
	maxweight := big.NewInt(0)
	max := -1

	w.synclock.Lock()
	defer w.synclock.Unlock()

	if len(headers) == 0 {
		return nil, nil
	}

	readyblocks := make([]*types.Header, 0)
	for _, block := range headers {
		if w.chainreader.IsBehindCurrent(block) || block.ParentHash == nil {
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

	w.chainreader.SetHeight(parent.Height, parent)

	return parent, uncle
}

func (w *Worker) self() int64 {
	return w.ID
}

func (w *Worker) isMinerWorking() bool {

	for i := 0; i < len(w.vdf1); i++ {
		if !w.vdf1[i].Finished {
			return true
		}
	}
	return false
}

func (w *Worker) SetEngine(engine *PoTEngine) {
	w.Engine = engine
	w.Peerid = w.Engine.GetPeerID()
}
