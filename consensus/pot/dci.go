package pot

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/proto"
)

var (
	exchequer       = int32(0)
	Miner           = int32(1)
	UncleBlockMiner = int32(2)
	CommitteeLeader = int32(3)
	CommitteeMember = int32(4)
	bcimap          = map[int32]float64{
		exchequer:       0.3,
		Miner:           0.5,
		UncleBlockMiner: 0.02,
		CommitteeLeader: 0.2,
		CommitteeMember: 0.1,
	}
	burnoutAddress    = []byte("000000")
	VsiConvertAddress = []byte("000000")
)

//goland:noinspection ALL
var (
	savings       = uint64(0)
	HalfYear      = OneYear / 2
	OneYear       = uint64(365 * 144)
	TwoYears      = OneYear * 2
	ThreeYears    = OneYear * 3
	TenYears      = 10 * OneYear
	savingRate    = 0.001
	HalfYearRate  = float64(0.005)
	OneYearRate   = float64(0.01)
	ThreeYearRate = float64(0.02)
	TenYearRate   = float64(0.05)
)

func (w *Worker) VerifyDciReward(reward *DciReward) (bool, *types.ExecutedTx, error) {
	//return true
	//address := reward.Address
	//amount := reward.Amount
	proof := reward.Proof

	exeheight := proof.Height
	if exeheight > w.executeheight {
		return false, nil, fmt.Errorf("height is beyond execute height")
	}
	exeblock, err := w.blockStorage.GetExcutedBlock(proof.BlockHash)
	if err != nil {
		return false, nil, err
	}
	if exeblock.Header.Height != proof.Height {
		return false, nil, fmt.Errorf("the height of proof %d is not equal to the height of block %d", proof.Height, exeblock.Header.Height)
	}
	for _, tx := range exeblock.Txs {
		if !bytes.Equal(tx.TxHash, proof.TxHash) {
			continue
		} else {
			return true, tx, nil
		}
	}
	return false, nil, fmt.Errorf("not found tx in block")
}

func (w *Worker) SendDci(ctx context.Context, request *pb.SendDciRequest) (*pb.SendDciResponse, error) {
	err := w.broadcastSendDciRequest(request)
	if err != nil {
		return &pb.SendDciResponse{IsSuccess: false}, err
	}
	return w.handleSendDciRequest(request)
}

func (w *Worker) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	addr := request.GetAddress()
	w.log.Errorf("get balance of %s", hexutil.Encode(addr))
	//amount := w.chainReader.GetBalance(addr)
	utxos := w.chainReader.FindUTXO(addr)
	balance := int64(0)

	pbutxos := make([]*pb.Utxo, 0)
	//count := 0
	for utxokey, utxo := range utxos {
		var txid [32]byte
		var voutput int64
		_, err := fmt.Sscanf(utxokey, "%s:%d", &txid, &voutput)
		if err != nil {
			return &pb.GetBalanceResponse{Balance: 0, Address: addr}, err
		}

		Utxo := &pb.Utxo{
			Txid:     txid[:],
			Voutput:  voutput,
			TxOutput: utxo.ToProto(),
		}
		pbutxos = append(pbutxos, Utxo)
	}
	return &pb.GetBalanceResponse{
		Address: addr,
		Balance: balance,
		Utxos:   pbutxos,
	}, nil
}

func (w *Worker) DevastateDci(ctx context.Context, request *pb.DevastateDciRequest) (*pb.DevastateDciResponse, error) {
	err := w.broadcastDevastateDciRequest(request)
	if err != nil {
		return &pb.DevastateDciResponse{Flag: false}, err
	}

	return w.handleDevastateDciRequest(request)
}

func (w *Worker) VerifyUTXO(ctx context.Context, request *pb.VerifyUTXORequest) (*pb.VerifyUTXOResponse, error) {
	from := request.GetFrom()
	if from != nil {
		return &pb.VerifyUTXOResponse{Flag: false}, nil
	}
	amount := request.GetValue()
	proof := request.GetProof()

	utxoproof := &pb.UTXOProof{}
	err := proto.Unmarshal(proof, utxoproof)
	if err != nil {
		return &pb.VerifyUTXOResponse{Flag: false}, err
	}
	height := utxoproof.GetHeight()
	hash := utxoproof.GetTxHash()

	block, err := w.chainReader.GetByHeight(height)
	if err != nil {
		return &pb.VerifyUTXOResponse{Flag: false}, err
	}

	rawtx := block.GetRawTx()
	for _, tx := range rawtx {
		if bytes.Equal(tx.Txid[:], hash) {
			if len(tx.TxInput) == 0 {
				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("txinput is zero")
			}
			txinput := tx.TxInput[0]

			if amount != txinput.Value {
				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("wrong value")
			}
			// return &pb.VerifyUTXOResponse{Flag: true}, nil
		}
	}

	//return &pb.VerifyUTXOResponse{Flag: false}, nil
	return &pb.VerifyUTXOResponse{Flag: true}, nil
}

func (w *Worker) handleSendDciRequest(request *pb.SendDciRequest) (*pb.SendDciResponse, error) {
	dcirewards := request.GetDciReward()

	for _, pbdcireward := range dcirewards {

		dciReward := ToDciReward(pbdcireward)
		flag, tx, err := w.VerifyDciReward(dciReward)
		if !flag {
			return &pb.SendDciResponse{
				IsSuccess: false,
				Height:    0,
			}, fmt.Errorf("the dci reward is not valid for %s", err.Error())
		} else {
			txdata := tx.Data
			if len(txdata) < 98 {
				return &pb.SendDciResponse{
					IsSuccess: false,
					Height:    0,
				}, fmt.Errorf("the dci reward is not valid for %s", err.Error())
			} else {
				address := txdata[2:98]
				//fmt.Println(hexutil.Encode(address))
				dciReward.Address = address
				w.mempool.AddDciReward(dciReward)
			}
		}
	}
	return &pb.SendDciResponse{
		IsSuccess: true,
		Height:    w.getEpoch(),
	}, nil
}

func (w *Worker) broadcastSendDciRequest(request *pb.SendDciRequest) error {
	requestbytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_SendDci_Request,
		MsgByte: requestbytes,
	}
	messageByte, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	err = w.Engine.Broadcast(messageByte)
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) handleDevastateDciRequest(request *pb.DevastateDciRequest) (*pb.DevastateDciResponse, error) {
	pbrawtx := request.GetTx()
	rawtx := types.ToRawTx(pbrawtx)
	if !rawtx.BasicVerify() {
		return &pb.DevastateDciResponse{Flag: false}, fmt.Errorf("tx is not valid")
	} else {

		db := w.chainReader.GetBoltDb()
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(types.UTXOBucket))
			inputmap := make(map[int32]int64)
			for _, input := range rawtx.TxInput {

				txid := input.Txid
				voutput := input.Voutput

				utxokey := fmt.Sprintf("%s:%d", txid, voutput)
				// if outputs, ok := outputsmap[utxokey]; ok {
				// 	if !outputs.CanBeUnlockWith(input) {
				// 		return &pb.DevastateDciResponse{Flag: false}, fmt.Errorf("tx input cannot unlock corresponding utxo")
				// 	}
				// 	if input.Value != outputs.Value {
				// 		return &pb.DevastateDciResponse{Flag: false}, fmt.Errorf("tx input value is not equal to corresponding utxo value")
				// 	}
				// 	amount += input.Value
				// 	delete(outputsmap, utxokey)
				// }
				outsBytes := b.Get([]byte(utxokey))
				if outsBytes == nil || len(outsBytes) == 0 {
					return fmt.Errorf("the input corresponding utxo not found")
				}

				output := types.DecodeByteToTxOutput(outsBytes)
				// TODO: add script check

				// if output.UseFlag {
				// 	return fmt.Errorf("tx input corresponding output is used but not check")
				// }
				if !output.CanBeUnlockWith(input) {
					return fmt.Errorf("tx input cannot unlock corresponding utxo")
				}
				if input.BciType != output.BciType || input.Value != output.Value {
					return fmt.Errorf(" tx error for bci type not match or value not match ")
				}
				inputmap[input.BciType] += input.Value

				b.Put([]byte(utxokey), output.EncodeToByte())
			}
			outputmap := make(map[int32]int64)
			outputcount := make(map[int32]int)
			for _, output := range rawtx.TxOutput {
				//if !output.IsCoinbase {
				//	return fmt.Errorf(" tx error for output is not coinbase")
				//}
				if output.LockTime <= ConfirmDelay && output.LockTime != 0 {
					return fmt.Errorf(" tx error for output locktime is too short")
				}
				outputmap[output.BciType] += output.Value
				outputcount[output.BciType]++
			}

			for bcitype, totalvalue := range inputmap {
				if totalvalue != outputmap[bcitype] {
					return fmt.Errorf(" tx error for input and output bci amount not match")
				}
			}

			for _, output := range rawtx.TxOutput {
				if output.Value != inputmap[output.BciType]/outputmap[output.BciType] {
					return fmt.Errorf(" tx error for output bci value and count not match")
				}
			}
			w.mempool.AddRawTx(rawtx)
			w.mempool.rawmap[rawtx.Hash()] = request.GetTransaction()
			return nil
		})
		if err != nil {
			return &pb.DevastateDciResponse{Flag: false}, err
		}

	}

	return &pb.DevastateDciResponse{Flag: true}, nil
}

func (w *Worker) broadcastDevastateDciRequest(request *pb.DevastateDciRequest) error {
	requestbytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_DevastateDci_Request,
		MsgByte: requestbytes,
	}
	messageByte, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	err = w.Engine.Broadcast(messageByte)
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) GenerateCoinbaseTx(pubkeybyte []byte, vdf0res []byte, totalreward int64) *types.Tx {
	dcirewards := w.mempool.GetAllDciRewards()
	coinbaseproof := make([]types.CoinbaseProof, 0)
	for _, dcireward := range dcirewards {
		proof := types.CoinbaseProof{
			TxHash:  dcireward.Proof.TxHash,
			Address: dcireward.Address,
			Amount:  dcireward.Amount,
		}
		coinbaseproof = append(coinbaseproof, proof)
	}
	txin := types.TxInput{

		Txid:      [32]byte{},
		Voutput:   -1,
		Scriptsig: nil,
		Value:     0,
		Address:   []byte{},
	}
	txouts := make([]types.TxOutput, 0)
	minerout := types.TxOutput{
		Address: pubkeybyte,
		Value:   int64(math.Floor(float64(totalreward) * bcimap[Miner])),

		ScriptPk: nil,
		Proof:    nil,
		LockTime: 144,
	}
	txouts = append(txouts, minerout)

	selectreward := make(map[int32][]*DciReward)
	if len(dcirewards) != 0 {
		groupsdata := groupByChainID(dcirewards)
		for _, rewards := range groupsdata {
			total := int64(0)
			for _, reward := range rewards {
				total += reward.Amount
			}
			for _, reward := range rewards {
				reward.weight = float64(reward.Amount) / float64(total)
			}
		}

		for chainID, rewards := range groupsdata {
			sort.Slice(rewards, func(i, j int) bool {
				return bytes.Compare(rewards[i].Address, rewards[j].Address) < 0
			})

			//seed := bytes.Join([][]byte{vdf0res},big.NewInt(bcitype).Bytes())

			rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
			for i := 0; i < Selectn; i++ {
				r := rand.Float64()
				acnum := 0.0
				for _, reward := range rewards {
					acnum += reward.weight
					if r <= acnum {
						selectreward[chainID] = append(selectreward[chainID], reward)
					}
				}
			}
		}
	}

	for bcitype, rewards := range selectreward {

		lenreward := len(rewards)
		switch bcitype {
		case UncleBlockMiner:
			for _, reward := range rewards {
				txout := types.TxOutput{
					Address: reward.Address,
					Value:   int64(math.Floor(float64(totalreward) * bcimap[UncleBlockMiner] / float64(lenreward))),

					ScriptPk: nil,
					Proof:    nil,
					LockTime: 144,
				}
				txouts = append(txouts, txout)
			}
		}
	}

	tx := &types.RawTx{
		Txid:           [32]byte{},
		TxInput:        []types.TxInput{txin},
		TxOutput:       txouts,
		CoinbaseProofs: coinbaseproof,
	}
	tx.Txid = tx.Hash()

	if len(txouts) == 2 {
		fmt.Println(hexutil.Encode(tx.Txid[:]))
	}

	txdata, _ := tx.EncodeToByte()
	return &types.Tx{Data: txdata}
}

func groupByChainID(rewards []*DciReward) map[int32][]*DciReward {
	groupData := make(map[int32][]*DciReward)
	for _, reward := range rewards {
		groupData[reward.BciType] = append(groupData[reward.BciType], reward)
	}
	return groupData
}

func (w *Worker) GenerateCoinbaseTxWithoutMinerKey(dcirewards []*DciReward, privkey *crypto.PrivateKey, totalreward int64) *types.RawTx {
	coinbaseproof := make([]types.CoinbaseProof, 0)
	pubkeybyte := privkey.PublicKeyBytes()
	minerout := types.TxOutput{
		Address:  pubkeybyte,
		Value:    int64(math.Floor(float64(totalreward) * bcimap[Miner])),
		ScriptPk: nil,
		Proof:    nil,
		LockTime: 144,
	}

	for _, dcireward := range dcirewards {
		proof := types.CoinbaseProof{
			TxHash:  dcireward.Proof.TxHash,
			Address: dcireward.Address,
			Amount:  dcireward.Amount,
			Type:    dcireward.BciType,
		}
		coinbaseproof = append(coinbaseproof, proof)
	}
	txin := types.TxInput{
		Txid:      [32]byte{},
		Voutput:   -1,
		Scriptsig: nil,
		Value:     0,
		Address:   []byte{},
	}
	txouts := make([]types.TxOutput, 0)

	txouts = append(txouts, minerout)

	tx := &types.RawTx{
		Txid:           [32]byte{},
		TxInput:        []types.TxInput{txin},
		TxOutput:       txouts,
		CoinbaseProofs: coinbaseproof,
	}
	return tx
}
func groupByType(proofs []types.CoinbaseProof) map[int32][]types.CoinbaseProof {
	groupData := make(map[int32][]types.CoinbaseProof)
	for _, reward := range proofs {
		groupData[reward.Type] = append(groupData[reward.Type], reward)
	}
	return groupData
}

func CoinbaseProofToBytes(coinbaseproofs []types.CoinbaseProof) []byte {
	var buffer bytes.Buffer
	for _, coinbaseproof := range coinbaseproofs {
		pbproof := coinbaseproof.ToProto()
		proofbyte, _ := proto.Marshal(pbproof)
		buffer.Write(proofbyte)
	}
	return buffer.Bytes()
}

func (w *Worker) handleConfirmBlockTx(height uint64) error {
	if height-ConfirmDelay > 0 {
		return nil
	}
	block, err := w.chainReader.GetByHeight(height - ConfirmDelay)
	if err != nil {
		return err
	}

	txs := block.GetRawTx()
	for _, tx := range txs {
		if !tx.IsCoinBase() {
			for _, txoutput := range tx.TxOutput {
				if len(txoutput.Data) != 0 {

				}

			}

		}
	}

	return nil
}

func (w *Worker) transferTx2EVM([]byte) error {

	return nil
}

func (w *Worker) CheckBlockTxs(block *types.Block) (bool, error) {
	txs := block.GetRawTx()
	header := block.GetHeader()
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, rawtx := range txs {
			if !rawtx.IsCoinBase() {
				if !rawtx.BasicVerify() {
					return fmt.Errorf("tx %s verify failed", hexutil.Encode(rawtx.Txid[:]))
				}
				inputmap := make(map[int32]int64)
				for _, input := range rawtx.TxInput {
					utxokey := fmt.Sprintf("%s:%d", input.Txid, input.Voutput)
					outsBytes := b.Get([]byte(utxokey))
					if outsBytes == nil {
						return fmt.Errorf(" tx error for can't find corresponding utxo ")
					}
					output := types.DecodeByteToTxOutput(outsBytes)

					if !output.CanBeUnlockWith(input) {
						return fmt.Errorf(" tx error for can't unlock with txinput")
					}
					if input.BciType != output.BciType || input.Value != output.Value {
						return fmt.Errorf(" tx error for bci type not match or value not match ")
					}
					if output.BurnLock != 0 && output.BurnLock > header.Height {
						for _, txOutput := range rawtx.TxOutput {
							if bytes.Equal(txOutput.Address, burnoutAddress) {
								return fmt.Errorf(" tx error for utxo has not reached burn lock height")
							}
							if txOutput.BurnLock != output.BurnLock {
								return fmt.Errorf("tx error for output burn lock not match to the input")
							}
						}
					}
					inputmap[input.BciType] += input.Value
				}

				outputmap := make(map[int32]int64)

				for _, output := range rawtx.TxOutput {
					//if !output.IsCoinbase {
					//	return fmt.Errorf(" tx error for output is not coinbase")
					//}
					if output.LockTime <= ConfirmDelay && output.LockTime != 0 {
						return fmt.Errorf(" tx error for output locktime is too short")
					}
					outputmap[output.BciType] += output.Value

				}

				for bcitype, totalvalue := range inputmap {
					if totalvalue != outputmap[bcitype] {
						return fmt.Errorf(" tx error for input and output bci amount not match")
					}
				}

			} else {
				coinbaseproofs := make([]types.CoinbaseProof, len(rawtx.CoinbaseProofs))
				if len(coinbaseproofs) != 0 {
					copy(coinbaseproofs, rawtx.CoinbaseProofs)
					for _, coinbaseproof := range coinbaseproofs {
						if !w.mempool.HasDciRewardByCoinbaseProof(&coinbaseproof) {
							return fmt.Errorf(" coinbase tx error for can't find corresponding dci reward")
						}
					}
					groupsdata := groupByType(coinbaseproofs)
					for _, proofs := range groupsdata {
						total := int64(0)
						for _, proof := range proofs {
							total += proof.Amount
						}
						for _, proof := range proofs {
							proof.Weight = float64(proof.Amount) / float64(total)
						}
					}
					selectproofs := make(map[int32][]*types.CoinbaseProof)
					for bcitypes, proofs := range groupsdata {
						sort.Slice(proofs, func(i, j int) bool {
							return bytes.Compare(proofs[i].Address, proofs[j].Address) < 0
						})
						if len(header.PoTProof) < 2 {
							return fmt.Errorf("pot proof len is not enough")
						}
						vdf1res := header.PoTProof[1]
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
						lenproofs := len(proofs)
						if _, ok := bcimap[bcitype]; !ok {
							return fmt.Errorf("proof has illegal bci type")
						}
						for _, output := range rawtx.TxOutput {
							if output.BciType == bcitype {
								flag := false
								for _, proof := range proofs {
									if bytes.Equal(proof.Address, output.Address) {
										flag = true
										if output.Value != int64(math.Floor(float64(TotalReward)*bcimap[bcitype]/float64(lenproofs))) {
											return fmt.Errorf("coinbase tx error for output bci value is not correct")
										}
										if output.LockTime != 144 {
											return fmt.Errorf("coinbase tx error for output locktime is not correct")
										}
										//TODO: Scriptcheck
										break
									}
								}
								if !flag {
									return fmt.Errorf("coinbase tx error for shuffle result does not match to the txoutput")
								}
							}
						}
					}
				}

				if len(rawtx.TxOutput) == 0 {
					return fmt.Errorf("coinbase tx without miner output")
				}

				minerout := rawtx.TxOutput[0]
				if !bytes.Equal(minerout.Address, header.PublicKey) {
					return fmt.Errorf("coinbase tx miner output does not match to header public key")
				}
				if minerout.Value != int64(math.Floor(float64(TotalReward)*bcimap[Miner])) {
					return fmt.Errorf("coinbase tx miner output value is not correct")
				}
				if minerout.LockTime != 144 {
					return fmt.Errorf("coinbase tx mineroutput format is not correct")
				}
			}
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (w *Worker) CheckTxInterest(rawtx *types.RawTx, blockheight uint64) (bool, error) {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		if rawtx.IsCoinBase() {
			for _, output := range rawtx.TxOutput {
				if output.Interest != 0 {
					if output.BciType != Miner {
						return fmt.Errorf("not miner coinbase output does not have interest")
					}
				}
			}
			return nil
		}
		interestmap := make(map[int32]int64)
		inputtotal := int64(0)
		for _, input := range rawtx.TxInput {
			utxokey := fmt.Sprintf("%s:%d", input.Txid, input.Voutput)
			outsBytes := b.Get([]byte(utxokey))
			if outsBytes == nil {
				return fmt.Errorf(" tx error for can't find corresponding utxo ")
			}

			output := types.DecodeByteToTxOutput(outsBytes)
			if output.BlockHeight == 0 {
				return fmt.Errorf("tx has not corresponding blockheight")
			}

			outputinterest := output.Interest
			interestmap[output.BciType] += outputinterest

			inputtotal += outputinterest
			yeargap := blockheight - output.BlockHeight
			if yeargap/TenYears >= 1 {
				interest := int64(math.Floor(float64(yeargap) * float64(output.Value) * TenYearRate / float64(OneYear)))
				interestmap[output.BciType] += interest
				inputtotal += interest
			} else if yeargap/TenYears < 1 && yeargap/ThreeYears >= 1 {
				interest := int64(math.Floor(float64(yeargap) * float64(output.Value) * ThreeYearRate / float64(OneYear)))
				interestmap[output.BciType] += interest
				inputtotal += interest
			} else if yeargap/ThreeYears < 1 && yeargap/OneYear >= 1 {
				interest := int64(math.Floor(float64(yeargap) * float64(output.Value) * OneYearRate / float64(OneYear)))
				interestmap[output.BciType] += interest
				inputtotal += interest
			} else if yeargap/OneYear < 1 && yeargap/HalfYear >= 1 {
				interest := int64(math.Floor(float64(yeargap) * float64(output.Value) * HalfYearRate / float64(OneYear)))
				interestmap[output.BciType] += interest
				inputtotal += interest
			} else if yeargap/HalfYear < 1 {
				interest := int64(math.Floor(float64(yeargap) * float64(output.Value) * savingRate / float64(OneYear)))
				interestmap[output.BciType] += interest
				inputtotal += interest
			}
		}
		outputmap := make(map[int32]int64)
		outputtotal := int64(0)
		for _, output := range rawtx.TxOutput {
			outputmap[output.BciType] += output.Interest
			outputtotal += output.Interest
		}

		if outputtotal+rawtx.TransactionFee > inputtotal {
			return fmt.Errorf("use total interest is more than input total interest")
		}

		for bcitype, useinterest := range outputmap {
			if haveinterest, ok := interestmap[bcitype]; ok {
				if haveinterest < useinterest {
					return fmt.Errorf("txoutput use more than corresponding interest")
				}
			} else {
				return fmt.Errorf("could not find corresponding interest in txinput")
			}
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return true, nil
}
