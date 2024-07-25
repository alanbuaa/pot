package pot

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
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

func (w *Worker) SendDci(ctx context.Context, request *pb.SendDciRequest) (*pb.SendDciResponse, error) {

	dcirewards := request.GetDciReward()
	for _, pbdciproof := range dcirewards {
		dciproof := ToDciReward(pbdciproof)
		w.mempool.AddDciReward(dciproof)
	}

	return &pb.SendDciResponse{
		IsSuccess: true,
		Height:    0,
	}, nil

}

func (w *Worker) GetBalance(ctx context.Context, request *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	addr := request.GetAddress()
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
		w.mempool.AddRawTx(tx)
	}

	return &pb.DevastateDciResponse{}, nil
}

func (w *Worker) handleDevastateDciRequest() {

}

func (w *Worker) GetUpperBlock(height int64) *pb.WhirlyBlock {
	return &pb.WhirlyBlock{
		ParentHash: nil,
		Hash:       nil,
		Height:     0,
		Txs:        nil,
		Justify:    nil,
		Committed:  false,
	}
}

func (w *Worker) VerifyUpperBlock(block *pb.WhirlyBlock) bool {
	return true
}

func (w *Worker) VerifyDciReward(reward *DciReward) {
	proof := reward.Proof
	block := w.GetUpperBlock(proof.Height)

	if w.VerifyUpperBlock(block) {

	}

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

			rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
			rand.Shuffle(len(rewards), func(i, j int) {
				rewards[i], rewards[j] = rewards[j], rewards[i]
			})

			if Selectn < len(rewards) {
				selectreward[chainID] = rewards[:Selectn]
			} else {
				selectreward[chainID] = rewards
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
