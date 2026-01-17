package executor

import (
	"context"
	"math/big"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// Mockblock 模拟区块，用于测试
type Mockblock struct {
	Height uint64   // 区块高度
	Txs    [][]byte // 交易列表
}

// PoTExecutor PoT 共识执行器
type PoTExecutor struct {
	Height uint64       // 当前区块高度
	Blocks []*Mockblock // 区块列表
}

// NewPoTExecutor 创建 PoTExecutor 实例
func NewPoTExecutor() *PoTExecutor {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	logger.Info("Initializing PoTExecutor")
	return &PoTExecutor{
		Height: 0,
		Blocks: make([]*Mockblock, 0),
	}
}

// GetTxs 获取指定高度范围的交易数据
func (p *PoTExecutor) GetTxs(ctx context.Context, request *pb.GetTxRequest) (*pb.GetTxResponse, error) {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	start := request.GetStartHeight()

	logger.WithFields(map[string]interface{}{
		"start_height":   start,
		"current_height": p.Height,
	}).Debug("Received GetTxs request")

	if start > p.Height {
		logger.WithFields(map[string]interface{}{
			"start_height":   start,
			"current_height": p.Height,
		}).Warn("Requested start height exceeds current height")
		return &pb.GetTxResponse{}, nil
	}

	execblock := make([]*pb.ExecuteBlock, 0)
	for i := start; i < uint64(len(p.Blocks)); i++ {
		header := &pb.ExecuteHeader{Height: i}
		txs := make([]*pb.ExecutedTx, 0)
		for _, tx := range p.Blocks[i].Txs {
			etx := &pb.ExecutedTx{
				TxHash: tx,
				Height: i,
				Data:   nil,
			}
			txs = append(txs, etx)
		}
		blocks := &pb.ExecuteBlock{
			Header: header,
			Txs:    txs,
		}
		execblock = append(execblock, blocks)
	}

	logger.WithFields(map[string]interface{}{
		"start":       start,
		"end":         p.Height,
		"block_count": len(execblock),
	}).Info("GetTxs completed")

	return &pb.GetTxResponse{
		Start:  start,
		End:    p.Height,
		Blocks: execblock,
	}, nil
}

// VerifyTxs 验证交易列表
func (p *PoTExecutor) VerifyTxs(ctx context.Context, request *pb.VerifyTxRequest) (*pb.VerifyTxResponse, error) {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	txCount := len(request.GetTxs())

	logger.WithField("tx_count", txCount).Debug("Verifying transactions")

	flag := make([]bool, txCount)
	for i := 0; i < len(flag); i++ {
		flag[i] = true
	}

	logger.WithField("tx_count", txCount).Debug("All transactions verified successfully")

	reponse := &pb.VerifyTxResponse{
		Txs:  request.Txs,
		Flag: flag,
	}
	return reponse, nil
}

// ExecuteTxs 执行单个交易
func (p *PoTExecutor) ExecuteTxs(ctx context.Context, request *pb.ExecuteTxRequest) (*pb.ExecuteTxResponse, error) {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	logger.Debug("Executing transaction")

	return &pb.ExecuteTxResponse{
		Tx:   request.Tx,
		Flag: true,
		TxID: []byte{},
	}, nil
}

// VerifyIncensentive 验证激励数据
func (p *PoTExecutor) VerifyIncensentive(ctx context.Context, request *pb.IncensentiveVerifyRequest) (*pb.IncensentiveVerifyResponse, error) {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	logger.Debug("Verifying incentive")

	return &pb.IncensentiveVerifyResponse{
		VerifyRes: []bool{true},
	}, nil
}

// GetIncentive 获取激励数据
func (p *PoTExecutor) GetIncentive(ctx context.Context, request *pb.GetIncentiveRequest) (*pb.GetIncentiveResponse, error) {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	logger.Debug("Getting incentive")

	return &pb.GetIncentiveResponse{}, nil
}

// GenerateTxsForHeight 为指定高度生成模拟交易
func (p *PoTExecutor) GenerateTxsForHeight(height uint64) *Mockblock {
	logger := logging.GetLogger().WithField("module", "POTEXECUTOR")
	txs := make([][]byte, 0)
	for i := 0; i < 100; i++ {
		bigint := big.NewInt(int64(i))
		tx := crypto.Hash(bigint.Bytes())
		txs = append(txs, tx)
	}

	logger.WithFields(map[string]interface{}{
		"height":   height,
		"tx_count": len(txs),
	}).Debug("Generated mock transactions")

	return &Mockblock{
		Height: height,
		Txs:    txs,
	}
}
