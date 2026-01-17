// Package pot implements the Proof of Time (PoT) consensus algorithm worker.
// This file contains all mining-related functions including block creation and difficulty calculation.
package pot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
)

// mine performs the actual mining process for a specific worker thread
func (w *Worker) mine(epoch uint64, vdf0res []byte, nonce int64, workerid int, abort *Abortcontrol,
	difficulty *big.Int, parentblock *types.Block, uncleblock []*types.Block, wg *sync.WaitGroup,
	emptyblock *types.Block, Bcirewards []*BciReward) *types.Block {
	defer wg.Done()
	defer w.setWorkFlagFalse()
	w.log.WithFields(logrus.Fields{
		"epoch":     epoch,
		"worker_id": workerid,
	}).Debug("Mining worker started")

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
		w.log.WithFields(logrus.Fields{
			"epoch":     epoch,
			"worker_id": workerid,
		}).Trace("Starting VDF1 computation for mining")
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
				w.log.WithFields(logrus.Fields{
					"epoch":      epoch,
					"worker_id":  workerid,
					"difficulty": difficulty.Int64(),
					"hash":       utils.EncodeShortPrint(tmp.Bytes()),
				}).Trace("Failed to meet difficulty target, retrying")
				//w.blockStorage.Put(block)

				nonce += 1
				tmp.SetInt64(nonce)
				noncebyte2 := tmp.Bytes()
				input2 := bytes.Join([][]byte{noncebyte2, vdf0res, mixdigest}, []byte(""))
				hashinput2 := crypto.Hash(input2)
				w.vdf1[workerid] = types.NewVDFwithInput(w.vdf1Chan, hashinput2, w.config.PoT.Vdf1Iteration, w.ID)

				go func() {
					w.log.WithFields(logrus.Fields{
						"epoch":     epoch,
						"worker_id": workerid,
					}).Trace("Restarting VDF1 with new nonce")
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
				w.log.WithFields(logrus.Fields{
					"epoch":     epoch,
					"worker_id": workerid,
				}).Trace("Starting next VDF1 round")
				err := w.vdf1[workerid].Exec(epoch)
				if err != nil {
					return
				}
			}()

			block := w.CompleteBlock(emptyblock, vdf0res, res1, coinbasetx, mixdigest, privkey)

			w.blockCounter += 1
			w.log.WithFields(logrus.Fields{
				"epoch":        epoch,
				"block_number": w.blockCounter,
			}).Info("Successfully mined new block")

			// broadcast the block
			w.peerMsgQueue <- block
			// w.workFlag = false
			continue
		case <-abort.abortchannel:
			err := w.vdf1[workerid].Abort()
			if err != nil {
				w.log.WithError(err).WithFields(logrus.Fields{
					"epoch":     epoch,
					"worker_id": workerid,
				}).Warn("VDF1 abort failed")
				return nil
			}
			w.log.WithFields(logrus.Fields{
				"epoch":     epoch,
				"worker_id": workerid,
			}).Debug("Mining aborted.")
			// w.workFlag = false
			return nil
		}

	}
}

// createBlockWithoutKey creates a template block without cryptographic keys
func (w *Worker) createBlockWithoutKey(epoch uint64, parentBlock *types.Block, uncleBlock []*types.Block, difficulty *big.Int,
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

// CompleteBlock finalizes a mined block by adding all required proofs and signatures
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

// CompleteCoinbaseTx completes the coinbase transaction with rewards distribution
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

// createBlock creates a complete block with all mining proofs and transactions
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

// createNilBlock creates an empty block (used when mining fails to meet difficulty target)
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

// calcDifficulty calculates the mining difficulty for the next block
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

// calcMixdigest computes the mix digest for a block by hashing all block components
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

// caldifficultyExp calculates difficulty using exponential formula
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
