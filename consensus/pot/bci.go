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
	storage "github.com/zzz136454872/upgradeable-consensus/internal/storage/pot"
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
	// VerifyBciReward: 验证一个 BciReward 的有效性。
	// 输入: reward - 待验证的 BciReward，包含 Proof(区块高度、区块哈希、交易哈希) 等信息。
	// 输出: (bool, *types.ExecutedTx, error)
	//   - bool: 验证是否通过
	//   - *types.ExecutedTx: 如果找到对应交易，返回该已执行交易信息
	//   - error: 验证失败时返回原因
	// 行为: 检查 proof 的高度不超过当前执行高度，尝试在内存池中根据区块哈希获取区块，
	// 并在区块内找到对应交易以确认 reward 的合法性。
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
	// SendBci: 接受并广播 BCI 奖励请求。
	// 输入: gRPC 上下文和 SendBciRequest 包含若干 BciReward
	// 输出: SendBciResponse 表明是否成功以及当前高度
	// 行为: 先广播请求到网络，然后在本地处理请求（将有效的 BciReward 加入内存池）。
	err := w.broadcastSendBciRequest(request)
	if err != nil {
		return &pb.SendBciResponse{IsSuccess: false}, err
	}
	return w.handleSendBciRequest(request)
}

func (w *Worker) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	// GetBalance: 查询指定地址的 UTXO 与余额并返回
	// 输入: GetBalanceRequest 包含查询地址
	// 输出: GetBalanceResponse 包含地址、余额与 UTXO 列表
	// 行为: 从链上(通过 chainReader) 查找所有 UTXO，将其转换为 pb.Utxo 格式并汇总余额

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
	// VerifyUTXO: 验证 UTXO 证明（当前为简单占位实现，总是返回 true）
	// 输入: VerifyUTXORequest（包含证明或其他字段）
	// 输出: VerifyUTXOResponse.Flag 表示证明是否有效
	// 行为: 当前实现直接返回 true；注释掉的代码显示了更完整的验证流程（遍历链上区块查找 txid 并比对 data）。
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
	// CreateLockTransaction: 创建一个锁定类型交易的 gRPC 接口实现。
	// 输入: 包含要创建交易的 pb.RawTx
	// 输出: CreateLockTransactionResponse, error
	// 行为: 将 pb 的交易转换为内部 RawTx，调用 checkLockTransaction 校验，
	// 若校验通过则广播该交易并加入内存池。
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
	// checkLockTransaction: 在提交前本地校验 CreateLock 交易的合法性。
	// 输入: rawtx - 待校验的交易
	// 输出: error - 校验失败时返回具体错误
	// 行为: 读取本地 UTXO（bolt DB），检查输入输出的类型、数额、锁定时间、签名能否解锁等规则。
	height := w.getEpoch()
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// broadcastClientTransaction: 将客户端交易封装为 PoT 消息并通过 Engine 广播到网络
	// 输入: rawtx - 内部 RawTx, types - 交易种类（pb.TxType）
	// 输出: error - 广播或编码失败时返回
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
	// CreateLockTransferTransaction: gRPC 接口，创建锁定转移交易（transfer lock）
	// 行为: 校验交易(CheckLockTransferTransaction)、广播并加入内存池。
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
	// CheckLockTransferTransaction: 校验一个 transfer-lock 类型交易是否合法。
	// 主要检查输入是否对应已存在的 lock UTXO、兴趣(interest) 使用是否合理以及费用限制等。
	height := w.getEpoch()
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// CreateDevastateTransaction: gRPC 接口，用于创建“燃烧/销毁”类型的交易（devastate）
	// 行为: 校验 CreateDevastateTransaction 后广播并加入内存池。
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
	// CheckDevastateTransaction: 校验 devastate 交易是否合法，检查输入输出地址是否为 BurnoutAddress、interest 计算等。
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// CreateNonLockTransferTransaction: gRPC 接口，创建不带锁定的转账交易（non-lock transfer）。
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
	// CheckNonLockTransferTransaction: 校验非锁定转账交易是否合法，确保输入不为锁定 UTXO、输出不为 BurnoutAddress、利息使用等。
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// CreateBciToVsiTransaction: gRPC 接口，将 BCI 转换为 VSI 的交易创建入口，经过校验后广播并加入内存池。
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
	// GetPqcKey: 根据高度或区块哈希查找并返回与区块关联的 PQC 密钥（若本节点拥有）
	// 输入: GetPqcKeyRequest 可以包含 Height 或 BlockHash
	// 输出: GetPqcKeyResponse 包含 Flag（是否成功）、PublicKey 与 SecretKey（若成功）
	// 行为: 若提供 Height 则从链中读取对应区块并尝试查找私钥；否则使用 BlockHash 从 blockStorage 获取区块并查找。
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
	// CheckBciToVsiRequest: 校验将 BCI 转换为 VSI 的交易，确保输出地址是指定的 VsiConvertAddress，且利息使用不超限。
	db := w.chainReader.GetBoltDb()
	height := w.getEpoch()

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// handleSendBciRequest: 处理本地收到的 SendBciRequest，将验证通过的 BciReward 加入内存池
	// 行为: 对请求中的每个 BciReward 调用 VerifyBciReward，提取地址并 AddBciReward 到 mempool。
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
	// broadcastSendBciRequest: 将 SendBciRequest 序列化为 PoT 消息并广播到网络。
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
// 			b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// GenerateCoinbaseTx: 根据内存池中的 BciRewards 生成 coinbase 交易。
	// 输入: pubkeybyte - 挖矿者公钥字节, vdf0res - VDF 输出种子, totalreward - 本区块应分配的总奖励
	// 输出: *types.Tx 包含序列化后的 coinbase 交易数据
	// 行为: 将 BciRewards 分组、按权重抽样、根据 bcimap 规则分配奖励到对应 TxOutput 中，并返回封装好的 coinbase tx。
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
	// groupByChainID: 将 BciReward 列表按照 BciType（链 ID）分组，返回 map[bciType] -> rewards
	groupData := make(map[int32][]*BciReward)
	for _, reward := range rewards {
		groupData[reward.BciType] = append(groupData[reward.BciType], reward)
	}
	return groupData
}

func (w *Worker) GenerateCoinbaseTxWithoutMinerKey(Bcirewards []*BciReward, privkey *crypto.PqcKey, totalreward int64) *types.RawTx {
	// GenerateCoinbaseTxWithoutMinerKey: 在没有矿工本地私钥的情况下，使用提供的 privkey 构建 RawTx coinbase（仅构造，不签名）
	// 返回: 未编码的 *types.RawTx，供上层进一步处理。
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
	// groupByType: 将 CoinbaseProof 按 Type 字段分组，便于后续按 BCI 类型处理。
	groupData := make(map[int32][]types.CoinbaseProof)
	for _, reward := range proofs {
		groupData[reward.Type] = append(groupData[reward.Type], reward)
	}
	return groupData
}

func CoinbaseProofToBytes(coinbaseproofs []types.CoinbaseProof) []byte {
	// CoinbaseProofToBytes: 将 coinbase proofs 序列化为连续字节流（pb 序列化后直接拼接）
	var buffer bytes.Buffer
	for _, coinbaseproof := range coinbaseproofs {
		pbproof := coinbaseproof.ToProto()
		proofbyte, _ := proto.Marshal(pbproof)
		buffer.Write(proofbyte)
	}
	return buffer.Bytes()
}

func (w *Worker) handleConfirmBlockTx(height uint64) error {
	// handleConfirmBlockTx: 在区块确认（达到 ConfirmDelay）时处理区块内的交易，
	// 如果交易输出包含 Data 字段（需要转发到 EVM），则调用 TransferTx2EVM。
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
	// TransferTx2EVM: 调用远程执行器（gRPC）将交易数据转发到 EVM/执行器进行执行。
	// 输入: data - 需要在执行器中执行的交易字节
	// 输出: error - 远程调用失败或执行器返回失败时返回错误
	conn, err := grpc.NewClient(w.config.PoT.ExecutorAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))

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
	// CheckBlockTxs: 校验一个区块内所有交易的类型与内容是否合法。
	// 行为: 对每笔交易调用 CheckTxWithBlock，非 coinbase 类型根据不同判断分派到不同的校验函数（create lock, transfer lock, non-lock, devasted）。
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
	// CheckTxWithBlock: 在应用区块前，针对单笔交易按当前区块头信息进行全面校验。
	// 输入: rawtx - 待校验交易; block - 当前包含该交易的区块
	// 输出: bool, error - 校验是否通过及失败原因
	// 行为: 若为非 coinbase 交易，检查输入 utxo 的存在性、解锁能力、金额一致性等；若为 coinbase，检查 coinbase 输出（矿工份额、抽样分配等）是否符合规则。
	db := w.chainReader.GetBoltDb()
	header := block.GetHeader()
	totalreward := CalcTotalReward(header.Height)
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
				return fmt.Errorf("coinbase tx miner output does not match to header public key")
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
	// CheckNotDrawCoinbaseProof: 在链上查找 proof 对应的交易是否已经存在以决定该 proof 是否已被“兑现”。
	// 行为: 向后遍历区块链，查找是否存在 txid 与 proof.TxHash 相同的交易。
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
	// CheckBlockTxInterest: 校验交易使用的 interest（利息）是否合法，不超过输入可用的利息。
	// 行为: 根据输出的产生时间和锁定期计算累计利息，并比较输出使用是否被允许。
	db := w.chainReader.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
	// CheckMinerTransacFee: 校验 coinbase 中矿工输出是否正确占比总奖励，以及矿工声明的 interest 是否不超过交易费总额。

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
	// CheckMinerOutput: 辅助函数，校验 miner output 与区块头的 public key、奖励分配和锁定时间是否一致。
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
	// CalcTotalReward: 根据区块高度计算本高度应分配的总奖励，随时间递减（每若干周期衰减一半）。
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
		b := tx.Bucket([]byte(storage.UTXOBucket))
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
