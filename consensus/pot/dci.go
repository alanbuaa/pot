package pot

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	BurnoutAddress    = []byte("000000")
	VsiConvertAddress = []byte("000000")
	epsilon           = 1e-9
)

//goland:noinspection ALL
var (
	CoinbaseLock = uint64(6)
	// savings       = uint64(0)
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

func (w *Worker) VerifyBciReward(reward *BciReward) (bool, *types.ExecutedTx, error) {
	//return true
	//address := reward.Address
	//amount := reward.Amount
	proof := reward.Proof

	exeheight := proof.Height
	if exeheight > w.executeheight {
		return false, nil, fmt.Errorf("height is beyond execute height")
	}
	exeblock := w.mempool.GetBlockByHash(crypto.Convert(proof.BlockHash))
	if exeblock == nil {
		return false, nil, fmt.Errorf("could not find exeblock %s", hexutil.Encode(proof.BlockHash))
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

func (w *Worker) SendBci(ctx context.Context, request *pb.SendBciRequest) (*pb.SendBciResponse, error) {
	err := w.broadcastSendBciRequest(request)
	if err != nil {
		return &pb.SendBciResponse{IsSuccess: false}, err
	}
	return w.handleSendBciRequest(request)
}

func (w *Worker) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	addr := request.GetAddress()
	w.log.Errorf("get balance of %s", hexutil.Encode(addr))
	//amount := w.chainReader.GetBalance(addr)
	utxos, err := w.chainReader.FindUTXO(addr)
	if err != nil {
		fmt.Println("fail:", err)
		return &pb.GetBalanceResponse{Balance: 0, Address: addr}, err
	}
	balance := int64(0)

	pbutxos := make([]*pb.Utxo, 0)
	//count := 0
	fmt.Println(len(utxos))
	for utxokey, utxo := range utxos {
		// var txid [32]byte
		// var voutput int64
		// _, err := fmt.Sscanf(utxokey, "%s:%d", &txid, &voutput)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return &pb.GetBalanceResponse{Balance: 0, Address: addr}, err
		// }
		// balance += utxo.Value
		// fmt.Println(hexutil.Encode(txid[:]))

		parts := strings.Split(utxokey, ":")
		txid := parts[0]
		voutput, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			w.log.Errorf("Failed to parse voutput: %v", err)
			return &pb.GetBalanceResponse{Balance: 0, Address: addr}, err
		}

		balance += utxo.Value
		txidbyte, _ := hexutil.Decode(txid)

		Utxo := &pb.Utxo{
			Txid:     txidbyte,
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

// func (w *Worker) DevastateBci(ctx context.Context, request *pb.DevastateBciRequest) (*pb.DevastateBciResponse, error) {
// 	err := w.broadcastDevastateBciRequest(request)
// 	if err != nil {
// 		return &pb.DevastateBciResponse{Flag: false}, err
// 	}

// 	return w.handleDevastateBciRequest(request)
// }

func (w *Worker) VerifyUTXO(ctx context.Context, request *pb.VerifyUTXORequest) (*pb.VerifyUTXOResponse, error) {
	return &pb.VerifyUTXOResponse{Flag: true}, nil
	// from := request.GetFrom()
	// if from != nil {
	// 	return &pb.VerifyUTXOResponse{Flag: false}, nil
	// }

	// proof := request.GetProof()

	// utxoproof := &pb.UTXOProof{}
	// err := proto.Unmarshal(proof, utxoproof)
	// if err != nil {
	// 	return &pb.VerifyUTXOResponse{Flag: false}, err
	// }

	// hash := utxoproof.GetTxHash()
	// height := w.chainReader.GetCurrentHeight()
	// for height > 0 {

	// 	block, err := w.chainReader.GetByHeight(height)
	// 	if err != nil {
	// 		return &pb.VerifyUTXOResponse{Flag: false}, err

	// 	}
	// 	rawtx := block.GetRawTx()
	// 	for _, tx := range rawtx {
	// 		if bytes.Equal(tx.Txid[:], hash) {
	// 			if len(tx.TxInput) == 0 {
	// 				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("txinput is zero")
	// 			}
	// 			if len(tx.TxOutput) < int(utxoproof.GetVoutput()) {
	// 				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("txinput is less than voutput")
	// 			}
	// 			output := tx.TxOutput[utxoproof.GetVoutput()]
	// 			outputData := output.Data
	// 			if bytes.Equal(outputData, utxoproof.GetData()) {
	// 				return &pb.VerifyUTXOResponse{Flag: true}, nil
	// 			} else {
	// 				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("txoutput data is not equal to utxoproof data")
	// 			}
	// 		}
	// 	}
	// }
	// //return &pb.VerifyUTXOResponse{Flag: false}, nil
	// return &pb.VerifyUTXOResponse{Flag: false}, nil
}

func (w *Worker) CreateLockTransaction(ctx context.Context, request *pb.CreateLockTransactionRequest) (*pb.CreateLockTransactionResponse, error) {
	txs := request.GetTx()
	rawtx := types.ToRawTx(txs)
	err := w.checkLockTransaction(rawtx)
	if err != nil {
		return &pb.CreateLockTransactionResponse{}, err
	}

	err = w.broadcastClientTransaction(rawtx, pb.TxType_CreateLockTransaction)
	if err != nil {
		return nil, err
	}
	w.mempool.AddRawTx(rawtx)
	return &pb.CreateLockTransactionResponse{IsSuccess: true}, nil
}

func (w *Worker) checkLockTransaction(rawtx *types.RawTx) error {
	height := w.getEpoch()
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		if rawtx.IsCoinBase() {
			return fmt.Errorf("coinbase tx can't be lock tx")
		}
		if !rawtx.BasicVerify() {
			return fmt.Errorf("tx %s verify failed", hexutil.Encode(rawtx.Txid[:]))
		}
		if len(rawtx.TxInput) == 0 {
			return fmt.Errorf("tx %s has no input", hexutil.Encode(rawtx.Txid[:]))
		}
		expectedtype := rawtx.TxInput[0].BciType

		inputAmountTotal := int64(0)

		for _, input := range rawtx.TxInput {
			if input.BciType != expectedtype {
				return fmt.Errorf("tx %s input type not match", hexutil.Encode(rawtx.Txid[:]))
			}

			utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(input.Txid[:]), input.Voutput)
			outsBytes := b.Get([]byte(utxokey))
			if outsBytes == nil {
				return fmt.Errorf(" tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outsBytes)

			if !output.CanBeUnlockWith(input) {
				return fmt.Errorf(" tx error for can't unlock with txinput")
			}
			if input.BciType != output.BciType {
				return fmt.Errorf(" tx error for bci type not match,get output bci type %d but input bci type %d ", output.BciType, input.BciType)
			}
			if input.Value != output.Value {
				return fmt.Errorf(" tx error for value not match")
			}
			if output.BurnLock != 0 && output.BurnLock > height {
				return fmt.Errorf(" tx error for use output burnlock of create lock transaction is not zero or before current height")
			}
			inputAmountTotal += output.Value
		}
		uniqueAddress := rawtx.TxOutput[0].Address
		OutputAmountTotal := int64(0)

		for _, output := range rawtx.TxOutput {

			if output.LockTime <= ConfirmDelay && output.LockTime != 0 {
				return fmt.Errorf(" tx error for output locktime is too short")
			}
			// outputmap[output.BciType] += output.Value
			if output.BciType != expectedtype {
				return fmt.Errorf(" tx error for output bcitype not match")
			}
			if !bytes.Equal(output.Address, uniqueAddress) {
				return fmt.Errorf(" tx error for output address is not only one")
			}
			if output.Value <= 0 {
				return fmt.Errorf(" tx error for output value is not positive")
			}
			if output.CreatedAt != 0 {
				return fmt.Errorf("tx error for time for createlock tx output from client should be zero ")
			}

			OutputAmountTotal += output.Value
		}
		if OutputAmountTotal != inputAmountTotal {
			return fmt.Errorf(" tx error for amount is not match")
		}

		return nil
	})
	return err
}

func (w *Worker) broadcastClientTransaction(rawtx *types.RawTx, types pb.TxType) error {
	pbtx := pb.ClientTransaction{
		Tx:     rawtx.ToProto(),
		TxType: types,
	}
	b, err := proto.Marshal(&pbtx)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Client_Transaction,
		MsgByte: b,
	}
	messagebyte, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	err = w.Engine.Broadcast(messagebyte)
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) CreateLockTransferTransaction(ctx context.Context, request *pb.CreateLockTransferTransactionRequest) (*pb.CreateLockTransferTransactionResponse, error) {
	pbrawtx := request.GetTx()
	rawtx := types.ToRawTx(pbrawtx)
	err := w.CheckLockTransferTransaction(rawtx)
	if err != nil {
		return nil, err
	}

	err = w.broadcastClientTransaction(rawtx, pb.TxType_LockTransferTranscation)
	if err != nil {
		return nil, err
	}
	w.mempool.AddRawTx(rawtx)
	return &pb.CreateLockTransferTransactionResponse{
		IsSuccess: true,
	}, nil
}

func (w *Worker) CheckLockTransferTransaction(rawtx *types.RawTx) error {
	height := w.getEpoch()
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		if rawtx.IsCoinBase() {
			return fmt.Errorf("coinbase tx can't be lock tx")
		}
		inputinterest := int64(0)
		outputinterest := int64(0)
		toptransactionfee := int64(0)
		count := 0
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock == 0 || output.BurnLock < height {
				return fmt.Errorf("tx is not a transferlock transaction")
			}

			for _, txoutput := range rawtx.TxOutput {
				if txoutput.BurnLock == output.BurnLock && txoutput.Value == output.Value && txoutput.CreatedAt == output.CreatedAt {
					if txoutput.Interest <= output.Interest {
						inputinterest += txoutput.Interest
						interest := CalcInterest(height, txoutput)
						inputinterest += interest
						count += 1
						break
					}
					toptransactionfee += int64(math.Floor(float64(output.Value) * savingRate * float64(height-output.BlockHeight)))
				}
				return fmt.Errorf("transferlock transaction is not valid for could not find a lock utxo corresponding to txinput")
			}
		}
		if count != len(rawtx.TxInput) {
			return fmt.Errorf("transferlock transaction is not valid")
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock == 0 {
				return fmt.Errorf("tx is not a transferlock transaction")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputinterest {
			return fmt.Errorf("transferlock transaction is not valid for use more than input interest, output interest: %d, transactionfee %d, input interest: %d", outputinterest, rawtx.TransactionFee, inputinterest)
		}
		if rawtx.TransactionFee > toptransactionfee {
			return fmt.Errorf("transferlock transaction is not valid for use more than top transaction fee,top is %d,use %d", toptransactionfee, rawtx.TransactionFee)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
func (w *Worker) CreateDevastateTransaction(ctx context.Context, request *pb.CreateDevastateTransactionRequest) (*pb.CreateDevastateTransactionResponse, error) {
	pbrawtx := request.GetTx()
	rawtx := types.ToRawTx(pbrawtx)
	err := w.CheckDevastateTransaction(rawtx)
	if err != nil {
		return nil, err
	}
	w.mempool.AddRawTx(rawtx)
	err = w.broadcastClientTransaction(rawtx, pb.TxType_DevasteTransaction)
	if err != nil {
		return nil, err
	}

	return &pb.CreateDevastateTransactionResponse{IsSuccess: true}, nil
}

func (w *Worker) CheckDevastateTransaction(rawtx *types.RawTx) error {
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputInterest := int64(0)
		outputinterest := int64(0)
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock > height {
				return fmt.Errorf("tx is not a devasted transaction")
			}
			inputInterest += output.Interest

			if output.BurnLock != 0 && output.BurnLock >= height {
				gap := height - output.BurnLock
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			} else if output.BurnLock == 0 {
				gap := height - output.BlockHeight
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if !bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not a devasted transaction for receiving address is not burnout address")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputInterest {
			return fmt.Errorf("tx is not a devasted transaction for output interest is greater than input interest")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
func (w *Worker) CreateNonLockTransferTransaction(ctx context.Context, request *pb.CreateNonLockTransferTransactionRequest) (*pb.CreateNonLockTransferTransactionResponse, error) {
	pbrawtx := request.GetTx()
	rawtx := types.ToRawTx(pbrawtx)
	err := w.CheckNonLockTransferTransaction(rawtx)
	if err != nil {
		return nil, err
	}

	err = w.broadcastClientTransaction(rawtx, pb.TxType_NonLockTransferTranscation)
	if err != nil {
		return nil, err
	}
	w.mempool.AddRawTx(rawtx)
	return &pb.CreateNonLockTransferTransactionResponse{IsSuccess: true}, nil
}

func (w *Worker) CheckNonLockTransferTransaction(rawtx *types.RawTx) error {
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputinterest := int64(0)
		outputinterest := int64(0)

		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock < height {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}

			inputinterest += output.Interest
			if output.BurnLock != 0 && output.BurnLock >= height {
				gap := height - output.BurnLock
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			} else if output.BurnLock == 0 {
				gap := height - output.BlockHeight
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			}
		}

		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock != 0 {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}
			if bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not a non-lock transfer transaction for receiving address is a burnout address")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputinterest {
			return fmt.Errorf("tx is not valid for use interest is more than input interest")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) CreateBciToVsiTransaction(ctx context.Context, request *pb.CreateBciToVsiRequest) (*pb.CreateBciToVsiResponse, error) {
	pbrawtx := request.GetTx()
	rawtx := types.ToRawTx(pbrawtx)
	err := w.CheckBciToVsiRequest(rawtx)
	if err != nil {
		return nil, err
	}
	err = w.broadcastClientTransaction(rawtx, pb.TxType_CreateBciToVsiTransaction)
	if err != nil {
		return nil, err
	}
	w.mempool.AddRawTx(rawtx)
	return &pb.CreateBciToVsiResponse{IsSuccess: true}, nil
}

func (w *Worker) GetPqcKey(ctx context.Context, request *pb.GetPqcKeyRequest) (*pb.GetPqcKeyResponse, error) {
	if request.Height != 0 {
		b, err := w.chainReader.GetByHeight(request.Height)
		if err != nil {
			return &pb.GetPqcKeyResponse{Flag: false}, err
		}
		bhash := b.GetHeader().Hash()

		flag, key := w.TryFindKey(crypto.Convert(bhash))
		if !flag {
			return &pb.GetPqcKeyResponse{Flag: false}, fmt.Errorf("can't find key for height %d for the block is not owned by this node", request.Height)
		}
		return &pb.GetPqcKeyResponse{
			Flag:      true,
			PublicKey: b.GetHeader().PublicKey,
			SecretKey: key,
		}, nil
	} else {
		b, err := w.blockStorage.Get(request.BlockHash)
		if err != nil {
			return &pb.GetPqcKeyResponse{Flag: false}, err
		}
		bhash := b.GetHeader().Hash()

		flag, key := w.TryFindKey(crypto.Convert(bhash))
		if !flag {
			return &pb.GetPqcKeyResponse{Flag: false}, fmt.Errorf("can't find key for height %d for the block is not owned by this node", request.Height)
		}
		return &pb.GetPqcKeyResponse{
			Flag:      true,
			PublicKey: b.GetHeader().PublicKey,
			SecretKey: key,
		}, nil
	}

}
func (w *Worker) CheckBciToVsiRequest(rawtx *types.RawTx) error {
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputinterest := int64(0)
		outputinterest := int64(0)

		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock < height {
				return fmt.Errorf("tx is not a transfer To Vsi transaction")
			}

			inputinterest += output.Interest
			if output.BurnLock != 0 && output.BurnLock >= height {
				gap := height - output.BurnLock
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			} else if output.BurnLock == 0 {
				gap := height - output.BlockHeight
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if !bytes.Equal(txoutput.Address, VsiConvertAddress) {
				return fmt.Errorf("tx is not a SelfLock transaction for receiving address is not burnout address")
			}
			if txoutput.Data == nil {
				return fmt.Errorf("tx is not a SelfLock transaction for txoutput data is nil")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputinterest {
			return fmt.Errorf("tx is not a SelfLock transaction for output interest is greater than input interest")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) handleSendBciRequest(request *pb.SendBciRequest) (*pb.SendBciResponse, error) {
	Bcirewards := request.GetBciReward()

	for _, pbBcireward := range Bcirewards {

		BciReward := ToBciReward(pbBcireward)
		flag, tx, err := w.VerifyBciReward(BciReward)
		if !flag {
			return &pb.SendBciResponse{
				IsSuccess: false,
				Height:    0,
			}, fmt.Errorf("the Bci reward is not valid for %s", err.Error())
		} else {
			txdata := tx.Data
			if len(txdata) < 98 {
				return &pb.SendBciResponse{
					IsSuccess: false,
					Height:    0,
				}, fmt.Errorf("the Bci reward is not valid for %s", err.Error())
			} else {
				address := txdata[2:98]
				//fmt.Println(hexutil.Encode(address))
				BciReward.Address = address
				w.mempool.AddBciReward(BciReward)
			}
		}
	}
	return &pb.SendBciResponse{
		IsSuccess: true,
		Height:    w.getEpoch(),
	}, nil
}

func (w *Worker) broadcastSendBciRequest(request *pb.SendBciRequest) error {
	requestbytes, err := proto.Marshal(request)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_SendBci_Request,
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

// func (w *Worker) handleDevastateBciRequest(request *pb.DevastateBciRequest) (*pb.DevastateBciResponse, error) {
// 	pbrawtx := request.GetTx()
// 	rawtx := types.ToRawTx(pbrawtx)
// 	if !rawtx.BasicVerify() {
// 		return &pb.DevastateBciResponse{Flag: false}, fmt.Errorf("tx is not valid")
// 	} else {

// 		db := w.chainReader.GetBoltDb()
// 		err := db.View(func(tx *bolt.Tx) error {
// 			b := tx.Bucket([]byte(types.UTXOBucket))
// 			inputmap := make(map[int32]int64)
// 			for _, input := range rawtx.TxInput {

// 				txid := input.Txid
// 				voutput := input.Voutput

// 				utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(txid[:]), voutput)
// 				fmt.Println(utxokey)
// 				// if outputs, ok := outputsmap[utxokey]; ok {
// 				// 	if !outputs.CanBeUnlockWith(input) {
// 				// 		return &pb.DevastateBciResponse{Flag: false}, fmt.Errorf("tx input cannot unlock corresponding utxo")
// 				// 	}
// 				// 	if input.Value != outputs.Value {
// 				// 		return &pb.DevastateBciResponse{Flag: false}, fmt.Errorf("tx input value is not equal to corresponding utxo value")
// 				// 	}
// 				// 	amount += input.Value
// 				// 	delete(outputsmap, utxokey)
// 				// }
// 				outsBytes := b.Get([]byte(utxokey))
// 				if len(outsBytes) == 0 {
// 					return fmt.Errorf("the input corresponding utxo not found")
// 				}

// 				output := types.DecodeByteToTxOutput(outsBytes)
// 				// TODO: add script check

// 				// if output.UseFlag {
// 				// 	return fmt.Errorf("tx input corresponding output is used but not check")
// 				// }
// 				if !output.CanBeUnlockWith(input) {
// 					return fmt.Errorf("tx input cannot unlock corresponding utxo")
// 				}
// 				if input.BciType != output.BciType || input.Value != output.Value {
// 					return fmt.Errorf(" tx error for bci type not match or value not match ")
// 				}
// 				inputmap[input.BciType] += input.Value

// 				b.Put([]byte(utxokey), output.EncodeToByte())
// 			}
// 			outputmap := make(map[int32]int64)
// 			outputcount := make(map[int32]int)
// 			for _, output := range rawtx.TxOutput {
// 				//if !output.IsCoinbase {
// 				//	return fmt.Errorf(" tx error for output is not coinbase")
// 				//}
// 				if output.LockTime <= ConfirmDelay && output.LockTime != 0 {
// 					return fmt.Errorf(" tx error for output locktime is too short")
// 				}
// 				outputmap[output.BciType] += output.Value
// 				outputcount[output.BciType]++
// 			}

// 			for bcitype, totalvalue := range inputmap {
// 				if totalvalue != outputmap[bcitype] {
// 					return fmt.Errorf(" tx error for input and output bci amount not match")
// 				}
// 			}

// 			for _, output := range rawtx.TxOutput {
// 				if output.Value != inputmap[output.BciType]/outputmap[output.BciType] {
// 					return fmt.Errorf(" tx error for output bci value and count not match")
// 				}
// 			}
// 			w.mempool.AddRawTx(rawtx)
// 			w.mempool.rawmap[rawtx.Hash()] = request.GetTransaction()
// 			return nil
// 		})
// 		if err != nil {
// 			return &pb.DevastateBciResponse{Flag: false}, err
// 		}

// 	}

// 	return &pb.DevastateBciResponse{Flag: true}, nil
// }

// func (w *Worker) broadcastDevastateBciRequest(request *pb.DevastateBciRequest) error {
// 	requestbytes, err := proto.Marshal(request)
// 	if err != nil {
// 		return err
// 	}
// 	message := &pb.PoTMessage{
// 		MsgType: pb.MessageType_DevastateBci_Request,
// 		MsgByte: requestbytes,
// 	}
// 	messageByte, err := proto.Marshal(message)
// 	if err != nil {
// 		return err
// 	}
// 	err = w.Engine.Broadcast(messageByte)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (w *Worker) GenerateCoinbaseTx(pubkeybyte []byte, vdf0res []byte, totalreward int64) *types.Tx {
	Bcirewards := w.mempool.GetAllBciRewards()
	coinbaseproof := make([]types.CoinbaseProof, 0)
	for _, Bcireward := range Bcirewards {
		proof := types.CoinbaseProof{
			TxHash:  Bcireward.Proof.TxHash,
			Address: Bcireward.Address,
			Amount:  Bcireward.Amount,
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
		LockTime: CoinbaseLock,
	}
	txouts = append(txouts, minerout)

	selectreward := make(map[int32][]*BciReward)
	if len(Bcirewards) != 0 {
		groupsdata := groupByChainID(Bcirewards)
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
					LockTime: CoinbaseLock,
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

func groupByChainID(rewards []*BciReward) map[int32][]*BciReward {
	groupData := make(map[int32][]*BciReward)
	for _, reward := range rewards {
		groupData[reward.BciType] = append(groupData[reward.BciType], reward)
	}
	return groupData
}

func (w *Worker) GenerateCoinbaseTxWithoutMinerKey(Bcirewards []*BciReward, privkey *crypto.PqcKey, totalreward int64) *types.RawTx {
	coinbaseproof := make([]types.CoinbaseProof, 0)
	pubkeybyte := privkey.PublicKeyBytes()
	minerout := types.TxOutput{
		Address:  pubkeybyte,
		Value:    int64(math.Floor(float64(totalreward) * bcimap[Miner])),
		ScriptPk: nil,
		Proof:    nil,
		LockTime: CoinbaseLock,
	}

	for _, Bcireward := range Bcirewards {
		proof := types.CoinbaseProof{
			TxHash:  Bcireward.Proof.TxHash,
			Address: Bcireward.Address,
			Amount:  Bcireward.Amount,
			Type:    Bcireward.BciType,
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
	if height < ConfirmDelay {
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
					fmt.Printf("txid %s data transfer to vm\n", hexutil.Encode(tx.Txid[:]))
					err := w.TransferTx2EVM(txoutput.Data)
					if err != nil {
						return err
					}
				}

			}
		}
	}

	return nil
}

func (w *Worker) TransferTx2EVM(data []byte) error {
	conn, err := grpc.NewClient(w.config.PoT.ExcutorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))

	if err != nil {
		return err
	}
	client := pb.NewPoTExecutorClient(conn)
	request := &pb.ExecuteTxRequest{
		Tx: data,
	}
	resp, err := client.ExecuteTxs(context.Background(), request)
	if err != nil {
		return err
	}
	if resp.GetFlag() {
		return nil
	}
	return nil
}

func (w *Worker) CheckBlockTxs(block *types.Block) (bool, error) {
	txs := block.GetRawTx()
	// header := block.GetHeader()
	for _, tx := range txs {
		flag, err := w.CheckTxWithBlock(tx, block)
		if err != nil {
			return flag, fmt.Errorf("tx %s check failed %s", hexutil.Encode(tx.Txid[:]), err.Error())
		}
		if !tx.IsCoinBase() {
			if w.IsCreateLockTransaction(tx, block) {
				fmt.Printf("tx %s is create lock transaction\n", hexutil.Encode(tx.Txid[:]))
				return w.checkCreateLockTransaction(block, tx)
			} else if w.IsTransferLockTransaction(tx, block) {
				fmt.Printf("tx %s is transfer lock transaction\n", hexutil.Encode(tx.Txid[:]))
				return w.checkTransferLockTransaction(tx, block)
			} else if w.IsNonLockTransferTransaction(tx, block) {
				fmt.Printf("tx %s is non lock transfer transaction\n", hexutil.Encode(tx.Txid[:]))
				return w.checkNonLockTransferTransaction(tx, block)
			} else if w.IsDevastedTransaction(tx, block) {
				fmt.Printf("tx %s is devasted transaction\n", hexutil.Encode(tx.Txid[:]))
				return w.checkDevastedTransaction(tx, block)
			} else {
				return false, fmt.Errorf("tx type error")
			}
		}
	}
	return true, nil
}

func (w *Worker) CheckTxWithBlock(rawtx *types.RawTx, block *types.Block) (bool, error) {
	db := w.chainReader.GetBoltDb()
	header := block.GetHeader()
	totalreward := CalcTotalReward(header.Height)
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		if !rawtx.IsCoinBase() {
			if !rawtx.BasicVerify() {
				return fmt.Errorf("tx %s verify failed", hexutil.Encode(rawtx.Txid[:]))
			}
			if len(rawtx.TxInput) == 0 {
				return fmt.Errorf("tx %s has no input", hexutil.Encode(rawtx.Txid[:]))
			}

			expectedtype := rawtx.TxInput[0].BciType

			inputAmountTotal := int64(0)
			for _, input := range rawtx.TxInput {
				if input.BciType != expectedtype {
					return fmt.Errorf("tx %s input type not match", hexutil.Encode(rawtx.Txid[:]))
				}

				utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(input.Txid[:]), input.Voutput)

				outsBytes := b.Get([]byte(utxokey))
				if outsBytes == nil {
					return fmt.Errorf(" tx error for can't find corresponding utxo %s ", utxokey)
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
						if bytes.Equal(txOutput.Address, BurnoutAddress) {
							return fmt.Errorf(" tx error for utxo has not reached burn lock height")
						}
					}
				}
				inputAmountTotal += output.Value
				b.Put([]byte(utxokey), output.EncodeToByte())
			}
			uniqueAddress := rawtx.TxOutput[0].Address
			OutputAmountTotal := int64(0)

			for _, output := range rawtx.TxOutput {

				if output.LockTime <= ConfirmDelay && output.LockTime != 0 {
					return fmt.Errorf(" tx error for output locktime is too short")
				}
				// outputmap[output.BciType] += output.Value
				if output.BciType != expectedtype {
					return fmt.Errorf(" tx error for output bcitype not match")
				}
				if !bytes.Equal(output.Address, uniqueAddress) {
					return fmt.Errorf(" tx error for output address is not only one")
				}
				if output.Value <= 0 {
					return fmt.Errorf(" tx error for output value is not positive")
				}

				OutputAmountTotal += output.Value

			}
			if OutputAmountTotal != inputAmountTotal {
				return fmt.Errorf(" tx error for amount is not match")
			}

		} else {
			coinbaseproofs := make([]types.CoinbaseProof, len(rawtx.CoinbaseProofs))
			if len(coinbaseproofs) != 0 {
				copy(coinbaseproofs, rawtx.CoinbaseProofs)
				notdrawProof := make(map[int32][]*types.CoinbaseProof)
				for _, coinbaseproof := range coinbaseproofs {
					if !coinbaseproof.DoDraw {
						if flag, err := w.CheckNotDrawCoinbaseProof(&coinbaseproof); !flag {
							return err
						}

					} else {
						if !w.mempool.HasBciRewardByCoinbaseProof(&coinbaseproof) {
							return fmt.Errorf(" coinbase tx error for can't find corresponding Bci reward")
						}
					}

				}

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
									if output.Value != int64(math.Floor(float64(totalreward)*bcimap[bcitype]/float64(lenproofs))) {
										return fmt.Errorf("coinbase tx error for output bci value is not correct")
									}
									if output.LockTime != CoinbaseLock {
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
				for bcitype, proofs := range notdrawProof {
					for _, output := range rawtx.TxOutput {
						if output.BciType == bcitype {
							flag := false
							for _, proof := range proofs {
								if bytes.Equal(proof.Address, output.Address) {
									flag = true
									if output.Value != proof.Amount {
										return fmt.Errorf("coinbase tx error for output bci value is not correct")
									}
									if output.LockTime != CoinbaseLock {
										return fmt.Errorf("coinbase tx error for output locktime is not correct")
									}
									if output.Interest != proof.Interest {
										return fmt.Errorf("coinbase tx error for output bci interest is not correct")
									}
								}
							}
							if !flag {
								return fmt.Errorf("coinbase tx error for notdrawproof does not find corresponding the txoutput")
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
				return fmt.Errorf("coinbase tx miner output does not match to header public key, expect %s but get %s", hexutil.Encode(minerout.Address), hexutil.Encode(header.PublicKey))
			}
			if minerout.Value != int64(math.Floor(float64(totalreward)*bcimap[Miner])) {
				return fmt.Errorf("coinbase tx miner output value is not correct")
			}
			if minerout.LockTime != CoinbaseLock {
				return fmt.Errorf("coinbase tx mineroutput format is not correct")
			}
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *Worker) CheckNotDrawCoinbaseProof(proof *types.CoinbaseProof) (bool, error) {
	//db := w.chainReader.GetBoltDb()
	height := w.chainReader.GetCurrentHeight()
	for height > 0 {
		block, err := w.chainReader.GetByHeight(height)
		if err != nil {
			return false, err
		}

		txs := block.GetRawTx()
		for _, tx := range txs {
			if bytes.Equal(tx.Txid[:], proof.TxHash[:]) {
				return true, nil
			}
		}
		height--
	}
	return false, fmt.Errorf("coinbase tx error for the transaction of proof is not found")
}
func (w *Worker) CheckBlockTxInterest(rawtx *types.RawTx, blockheight uint64) (bool, error) {
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
			utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(input.Txid[:]), input.Voutput)
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

func (w *Worker) CheckMinerTransacFee(block *types.Block) (bool, error) {

	txs := block.GetRawTx()
	coinbasetx := txs[0]
	if !coinbasetx.IsCoinBase() {
		return false, fmt.Errorf("first tx is not coinbase tx")
	}
	minerout := coinbasetx.TxOutput[0]
	_, err := CheckMinerOutput(minerout, block)
	if err != nil {
		return false, err
	}
	return true, nil
}

func CheckMinerOutput(minerout types.TxOutput, block *types.Block) (bool, error) {
	addr := minerout.Address
	header := block.GetHeader()
	totalreward := CalcTotalReward(block.Header.Height)
	if !bytes.Equal(addr, header.PublicKey) {
		return false, fmt.Errorf("coinbase tx miner output does not match to header public key")
	}
	if minerout.Value != int64(math.Floor(float64(totalreward)*bcimap[Miner])) {
		return false, fmt.Errorf("coinbase tx miner output value is not correct")
	}
	if minerout.LockTime != CoinbaseLock {
		return false, fmt.Errorf("coinbase tx mineroutput lockTime is not correct")
	}

	txs := block.GetRawTx()
	transactionfee := int64(0)

	for _, tx := range txs {
		if !tx.IsCoinBase() {
			transactionfee += tx.TransactionFee
		}
	}

	if minerout.Interest > transactionfee {
		return false, fmt.Errorf("coinbase tx miner output interest is more than transaction fee")
	}

	return true, nil
}

func CalcTotalReward(height uint64) int64 {
	year := float64(height) / float64(OneYear*2)
	if year < 1 {
		return TotalReward
	}

	halfTimes := math.Floor(math.Log2(year))
	return int64(math.Floor(float64(TotalReward) * math.Pow(0.5, halfTimes)))
}

func CalcInterest(height uint64, output types.TxOutput) int64 {
	if output.BlockHeight <= height {
		return 0
	}

	// check non lock transfer interest

	if output.BurnLock == 0 {
		gap := height - output.BlockHeight
		return int64(math.Floor(float64(output.Value) * float64(gap) * savingRate))
	}
	if output.BurnLock > 0 && output.BurnLock < height {
		gap := height - output.BurnLock
		return int64(math.Floor(float64(output.Value) * float64(gap) * savingRate))
	}

	// check transfer lock
	if output.BurnLock > 0 && output.BurnLock >= height {

		rate := output.Rate
		gap := height - output.BlockHeight
		return int64(math.Floor(float64(output.Value) * float64(gap) * rate))
	}
	return 0 // should not happen
}

func (w *Worker) IsCreateLockTransaction(rawtx *types.RawTx, block *types.Block) bool {
	height := block.GetHeader().Height

	boltdb := w.chainReader.GetBoltDb()
	err := boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		if !rawtx.IsCoinBase() {
			for _, input := range rawtx.TxInput {
				lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(input.Txid[:]), input.Voutput)
				outputbyte := b.Get([]byte(lockkey))
				if outputbyte == nil {
					return fmt.Errorf("lock tx error for can't find corresponding utxo ")
				}
				output := types.DecodeByteToTxOutput(outputbyte)
				if output.BurnLock != 0 && output.BurnLock > height {
					return fmt.Errorf("the tx is not a TryLock tx")
				}
			}
			for _, output := range rawtx.TxOutput {
				if output.BurnLock <= height {
					return fmt.Errorf("the tx is not a TryLock tx")
				}
				// check interest
			}
			if len(rawtx.TxOutput) != 1 {
				return fmt.Errorf("the create lock transaction only have one output")
			}
		}

		return nil
	})
	return err == nil
}

func (w *Worker) IsTransferLockTransaction(rawtx *types.RawTx, block *types.Block) bool {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock == 0 || output.BurnLock < block.Header.Height {
				return fmt.Errorf("tx is not a transferlock transaction for use input is not lock ")
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock == 0 {
				return fmt.Errorf("tx is not a transferlock transaction for output is not lock")
			}
		}
		return nil
	})

	return err == nil
}

func (w *Worker) IsDevastedTransaction(rawtx *types.RawTx, block *types.Block) bool {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock > block.Header.Height {
				return fmt.Errorf("tx is not a devasted transaction")
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if !bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not a devasted transaction for receiving address is not burnout address")
			}
			if len(txoutput.Data) == 0 {
				return fmt.Errorf("tx is not a devasted transaction for data is empty")
			}

		}
		return nil
	})

	fmt.Printf("tx %s is a devasted transaction", hexutil.Encode(rawtx.Txid[:]))
	if err != nil {
		fmt.Println(err)
	}
	return err == nil
}

func (w *Worker) IsNonLockTransferTransaction(rawtx *types.RawTx, block *types.Block) bool {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock < block.Header.Height {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock != 0 {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}
			if bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not a non-lock transfer transaction for output address is burnout address")
			}
		}
		return nil
	})

	return err == nil
}

func (w *Worker) IsConvertToVsiTransaction(rawtx *types.RawTx, block *types.Block) bool {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock < block.Header.Height {
				return fmt.Errorf("tx is not a BciToVsi transaction")
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock != 0 {
				return fmt.Errorf("tx is not a BciToVsi transaction")
			}
			if bytes.Equal(txoutput.Address, VsiConvertAddress) {
				return fmt.Errorf("tx is not a BciToVsi transaction for output address is burnout address")
			}
		}
		return nil
	})

	return err == nil
}

func (w *Worker) checkCreateLockTransaction(block *types.Block, rawtx *types.RawTx) (bool, error) {
	db := w.chainReader.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		expectedtype := rawtx.TxInput[0].BciType
		inputInterest := int64(0)
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock > block.Header.Height {
				return fmt.Errorf("tx is not a CreateLock transaction for txoutput is lock")
			}
			if output.BciType != expectedtype || txinput.BciType != expectedtype {
				return fmt.Errorf("tx is not legal for bcitype is not unified")
			}
			inputInterest += output.Interest
			if output.BurnLock != 0 {
				gap := block.Header.Height - output.BurnLock
				interest := int64(math.Floor(float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			} else {
				gap := block.Header.Height - output.BlockHeight
				interest := int64(math.Floor(float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			}

		}
		if len(rawtx.TxOutput) != 1 {
			return fmt.Errorf("tx is not legal for createlock transaction has more than one output")
		}

		for _, output := range rawtx.TxOutput {
			if output.BurnLock == 0 {
				return fmt.Errorf("tx is not a Create lock transaction for lock time is 0")
			}
			if output.BurnLock < block.Header.Height {
				return fmt.Errorf("tx is not a Create lock transaction for lock time is less than current height")
			}

			yeargap := output.BurnLock - block.Header.Height
			if yeargap < HalfYear {
				return fmt.Errorf("tx is not a Create lock transaction for lock time is less than half year")
			}

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
			if math.Abs(output.Rate-rate) > epsilon {
				return fmt.Errorf("tx is illegal for interest rate is not equal to the expected rate, which expected to be %f but get %f", rate, output.Rate)
			}
			if output.CreatedAt != block.Header.Height {
				return fmt.Errorf("tx is illegal for created at is not equal to the block height")
			}
		}
		if rawtx.TxOutput[0].Interest+rawtx.TransactionFee > inputInterest {
			return fmt.Errorf("the interest number is too large")
		}

		return nil
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *Worker) checkTransferLockTransaction(rawtx *types.RawTx, block *types.Block) (bool, error) {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputinterest := int64(0)
		outputinterest := int64(0)
		toptransactionfee := int64(0)
		count := 0
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock == 0 || output.BurnLock < block.Header.Height {
				return fmt.Errorf("tx is not a transferlock transaction for input is not lock")
			}

			for _, txoutput := range rawtx.TxOutput {
				if txoutput.BurnLock == output.BurnLock && txoutput.Value == output.Value && txoutput.CreatedAt == output.CreatedAt {
					if txoutput.Interest <= output.Interest {
						inputinterest += txoutput.Interest
						interest := CalcInterest(block.Header.Height, txoutput)
						inputinterest += interest
						count += 1
						break
					}
					toptransactionfee += int64(math.Floor(float64(output.Value) * savingRate * float64(block.Header.Height-output.BlockHeight)))
				}
				return fmt.Errorf("transferlock transaction is not valid for could not find a lock utxo corresponding to txinput")
			}
		}
		if count != len(rawtx.TxInput) {
			return fmt.Errorf("transferlock transaction is not valid for could not find some txoutput corresponding to txinput")
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock == 0 {
				return fmt.Errorf("tx is not a transferlock transaction for output is not lock")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputinterest {
			return fmt.Errorf("transferlock transaction is not valid for use more than input interest")
		}
		if rawtx.TransactionFee > toptransactionfee {
			return fmt.Errorf("transferlock transaction is not valid for use more than top transaction fee")
		}
		return nil
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *Worker) checkNonLockTransferTransaction(rawtx *types.RawTx, block *types.Block) (bool, error) {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputinterest := int64(0)
		outputinterest := int64(0)
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock != 0 && output.BurnLock >= block.Header.Height {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}

			inputinterest += output.Interest
			if output.BurnLock != 0 && output.BurnLock < block.Header.Height {
				gap := block.Header.Height - output.BurnLock
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			} else if output.BurnLock == 0 {
				gap := block.Header.Height - output.BlockHeight
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputinterest += interest
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if txoutput.BurnLock != 0 {
				return fmt.Errorf("tx is not a non-lock transfer transaction")
			}
			if bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not valid for non-lock address for txoutput address is burnout address")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputinterest {
			return fmt.Errorf("tx is not valid for use interest is more than input interest")
		}

		return nil
	})

	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *Worker) checkDevastedTransaction(rawtx *types.RawTx, block *types.Block) (bool, error) {
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		inputInterest := int64(0)
		outputinterest := int64(0)
		for _, txinput := range rawtx.TxInput {
			lockkey := fmt.Sprintf("%s:%d", hexutil.Encode(txinput.Txid[:]), txinput.Voutput)
			outpubyte := b.Get([]byte(lockkey))
			if outpubyte == nil {
				return fmt.Errorf("update tx error for can't find corresponding utxo ")
			}
			output := types.DecodeByteToTxOutput(outpubyte)
			if output.BurnLock > block.Header.Height {
				return fmt.Errorf("tx is not a devasted transaction")
			}
			inputInterest += output.Interest

			if output.BurnLock != 0 && output.BurnLock >= block.Header.Height {
				gap := block.Header.Height - output.BurnLock
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			} else if output.BurnLock == 0 {
				gap := block.Header.Height - output.BlockHeight
				interest := int64(math.Floor(savingRate * float64(output.Interest) * float64(gap) / float64(OneYear)))
				inputInterest += interest
			}
		}
		for _, txoutput := range rawtx.TxOutput {
			if !bytes.Equal(txoutput.Address, BurnoutAddress) {
				return fmt.Errorf("tx is not a devasted transaction for receiving address is not burnout address")
			}
			outputinterest += txoutput.Interest
		}
		if outputinterest+rawtx.TransactionFee > inputInterest {
			return fmt.Errorf("tx is not a devasted transaction for output interest is greater than input interest")
		}
		return nil
	})

	if err != nil {
		return false, err
	}
	return true, nil

}
