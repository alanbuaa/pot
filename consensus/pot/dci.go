package pot

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/proto"
	"math"
	"sort"
)

const (
	ChainID0Rate = 0.1
	ChainID1Rate = 0.2
	ChainID2Rate = 0.3
	ChainID3Rate = 0.4
)

func (w *Worker) GenerateDciSendTx() {

}

func (w *Worker) ReceiveDCITx() {

}

func (w *Worker) VerifyDCITx() bool {
	return true
}

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
		return false, nil, fmt.Errorf("the height of proof is not equal to the height of block")
	}
	for _, tx := range exeblock.Txs {
		if !bytes.Equal(tx.TxHash, proof.TxHash) {
			continue
		} else {

		}

	}
	return false, nil, fmt.Errorf("not found tx in block")
}

func (w *Worker) SendDci(ctx context.Context, request *pb.SendDciRequest) (*pb.SendDciResponse, error) {

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

func (w *Worker) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	addr := request.GetAddress()
	w.log.Errorf("get balance of %s", hexutil.Encode(addr))
	amount := w.chainReader.GetBalance(addr)

	return &pb.GetBalanceResponse{
		Address:   addr,
		Balance:   amount,
		TxOutputs: nil,
	}, nil
}

func (w *Worker) DevastateDci(ctx context.Context, request *pb.DevastateDciRequest) (*pb.DevastateDciResponse, error) {

	pbrawtx := request.GetTx()
	tx := types.ToRawTx(pbrawtx)
	if !tx.BasicVerify() {
		return &pb.DevastateDciResponse{}, fmt.Errorf("tx is not valid")
	} else {
		addr := tx.TxInput[0].Address
		outputs := w.chainReader.FindUTXO(addr)
		for _, input := range tx.TxInput {
			if !bytes.Equal(addr, input.Address) {
				return &pb.DevastateDciResponse{}, fmt.Errorf("tx is not valid for address not the same")
			}
			flag := false
			for i, output := range outputs {
				if !output.CanBeUnlockWith(input.Address) {
					continue
				}
				if input.Value == output.Value {
					flag = true
					outputs = append(outputs[:i], outputs[i+1:]...)
					break
				}
			}
			if !flag {
				return &pb.DevastateDciResponse{}, fmt.Errorf("tx is not valid for not found spendable utxo")
			}
		}
		w.mempool.AddRawTx(tx)
	}

	return &pb.DevastateDciResponse{}, nil
}

func (w *Worker) VerifyUTXO(ctx context.Context, request *pb.VerifyUTXORequest) (*pb.VerifyUTXOResponse, error) {
	from := request.GetFrom()
	if from != nil {
		return &pb.VerifyUTXOResponse{Flag: false}, nil
	}
	to := request.GetTo()
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
			if !bytes.Equal(txinput.Address, to) {
				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("wrong address")
			}
			if amount != txinput.Value {
				return &pb.VerifyUTXOResponse{Flag: false}, fmt.Errorf("wrong value")
			}
			return &pb.VerifyUTXOResponse{Flag: true}, nil
		}
	}

	//return &pb.VerifyUTXOResponse{Flag: false}, nil
	return &pb.VerifyUTXOResponse{Flag: true}, nil
}

func (w *Worker) handleDevastateDciRequest() {

}

func (w *Worker) GenerateCoinbaseTx(pubkeybyte []byte, vdf0res []byte, totalreward int64) *types.Tx {
	dcirewards := w.mempool.GetAllDciRewards()

	txin := types.TxInput{
		IsCoinbase: false,
		Txid:       [32]byte{},
		Voutput:    -1,
		Scriptsig:  nil,
		Value:      0,
		Address:    []byte{},
	}
	txouts := make([]types.TxOutput, 0)
	minerout := types.TxOutput{
		Address:  pubkeybyte,
		Value:    int64(math.Floor(float64(totalreward) * ChainID0Rate)),
		IsSpent:  false,
		ScriptPk: nil,
		Proof:    nil,
	}
	txouts = append(txouts, minerout)

	selectreward := make(map[int64][]*DciReward)
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

			//seed := bytes.Join([][]byte{vdf0res},big.NewInt(chainID).Bytes())

			rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
			//rand.Shuffle(len(rewards), func(i, j int) {
			//	rewards[i], rewards[j] = rewards[j], rewards[i]
			//})
			//if Selectn < len(rewards) {
			//	selectreward[chainID] = rewards[:Selectn]
			//} else {
			//	selectreward[chainID] = rewards
			//}

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

	for chainID, rewards := range selectreward {
		lenreward := len(rewards)
		switch chainID {
		case 1:
			for _, reward := range rewards {
				txout := types.TxOutput{
					Address:  reward.Address,
					Value:    int64(math.Floor(float64(totalreward) * ChainID1Rate / float64(lenreward))),
					IsSpent:  false,
					ScriptPk: nil,
					Proof:    nil,
				}
				txouts = append(txouts, txout)
			}
		}
	}

	tx := &types.RawTx{
		Txid:     [32]byte{},
		TxInput:  []types.TxInput{txin},
		TxOutput: txouts,
	}
	tx.Txid = tx.Hash()

	txdata, _ := tx.EncodeToByte()
	return &types.Tx{Data: txdata}
}

func groupByChainID(rewards []*DciReward) map[int64][]*DciReward {
	groupData := make(map[int64][]*DciReward)
	for _, reward := range rewards {
		groupData[reward.ChainID] = append(groupData[reward.ChainID], reward)
	}
	return groupData
}