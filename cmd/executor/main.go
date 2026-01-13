package main

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"google.golang.org/grpc"
)

func main() {
	// 加载配置文件
	cfg, err := config.NewConfig("config/config.yaml", 0)
	if err != nil {
		fmt.Println("read config.yaml failed: ", err)
	}

	// 初始化日志系统
	logging.Setup("config/config.yaml")
	logger := logging.GetLogger()

	// 创建 TCP 监听器，监听配置文件中指定的执行器地址
	listen, err := net.Listen("tcp", cfg.Consensus.PoT.ExecutorAddress)
	logger.Infof("executor is running on: %s", cfg.Consensus.PoT.ExecutorAddress)
	if err != nil {
		logger.Error("failed to runn executor: ", err)
		return
	}
	// 创建 gRPC 服务器
	rpcserver := grpc.NewServer()
	// 创建 PoT 执行器实例
	exec := NewExec()
	// 注册 PoT 执行器服务
	pb.RegisterPoTExecutorServer(rpcserver, exec)
	// 在后台启动 gRPC 服务
	go rpcserver.Serve(listen)
	defer rpcserver.Stop()
	defer listen.Close()
	// 主循环：持续生成测试交易数据
	for {
		time.Sleep(1 * time.Second)
		logger.Info("Generating test blocks for height ", exec.height)
		// 为当前高度生成测试区块
		block := exec.GenerateTxsForHeight(exec.height)
		exec.blocks = append(exec.blocks, block)
		exec.height += 1
	}
}

// PoTExecutor 是 PoT 共识的执行器实现
// 维护当前区块高度和已生成的模拟区块列表
type PoTExecutor struct {
	height uint64       // 当前区块高度
	blocks []*Mockblock // 已生成的模拟区块列表
}

// NewExec 创建并返回一个新的 PoTExecutor 实例
func NewExec() *PoTExecutor {
	return &PoTExecutor{
		height: 0,
		blocks: make([]*Mockblock, 0),
	}
}

// GetTxs 获取指定高度范围内的交易数据，由远程调用
// 参数:
//   - ctx: 上下文
//   - request: 包含起始高度的请求
//
// 返回:
//   - GetTxResponse: 包含交易区块列表的响应
//   - error: 错误信息
func (p *PoTExecutor) GetTxs(ctx context.Context, request *pb.GetTxRequest) (*pb.GetTxResponse, error) {

	logger := logging.GetLogger()
	logger.Infof("receive request, start %d, end %d\n", request.StartHeight, p.height)
	// 获取请求的起始高度
	start := request.GetStartHeight()
	// 如果请求的起始高度超过当前高度，返回空响应
	if start > p.height {
		return &pb.GetTxResponse{}, nil
	}
	// 构造执行区块列表
	execblock := make([]*pb.ExecuteBlock, 0)
	// 遍历指定范围内的区块
	for i := start; i < uint64(len(p.blocks)); i++ {
		header := &pb.ExecuteHeader{Height: i}
		txs := make([]*pb.ExecutedTx, 0)
		for _, tx := range p.blocks[i].Txs {
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

	return &pb.GetTxResponse{
		Start:  start,
		End:    p.height,
		Blocks: execblock,
	}, nil

}

// VerifyTxs 验证交易列表的有效性
// 参数:
//   - ctx: 上下文
//   - request: 包含待验证交易列表的请求
//
// 返回:
//   - VerifyTxResponse: 包含验证结果的响应
//   - error: 错误信息
func (p *PoTExecutor) VerifyTxs(ctx context.Context, request *pb.VerifyTxRequest) (*pb.VerifyTxResponse, error) {
	// 创建验证结果数组，默认所有交易都通过验证
	flag := make([]bool, len(request.GetTxs()))
	for i := 0; i < len(flag); i++ {
		flag[i] = true
	}
	reponse := &pb.VerifyTxResponse{
		Txs:  request.Txs,
		Flag: flag,
	}
	return reponse, nil
}

// ExecuteTxs 执行单个交易
// 参数:
//   - ctx: 上下文
//   - request: 包含待执行交易的请求
//
// 返回:
//   - ExecuteTxResponse: 包含执行结果的响应
//   - error: 错误信息
func (p *PoTExecutor) ExecuteTxs(ctx context.Context, request *pb.ExecuteTxRequest) (*pb.ExecuteTxResponse, error) {
	// 返回执行成功的响应（测试实现，始终返回成功）
	return &pb.ExecuteTxResponse{
		Tx:   request.Tx,
		Flag: true,
		TxID: []byte{},
	}, nil

}

// VerifyIncensentive 验证激励数据的有效性
// 参数:
//   - ctx: 上下文
//   - request: 包含待验证激励数据的请求
//
// 返回:
//   - IncensentiveVerifyResponse: 包含验证结果的响应
//   - error: 错误信息
func (p *PoTExecutor) VerifyIncensentive(ctx context.Context, request *pb.IncensentiveVerifyRequest) (*pb.IncensentiveVerifyResponse, error) {
	// 返回验证成功的响应（测试实现，始终返回成功）
	return &pb.IncensentiveVerifyResponse{
		VerifyRes: []bool{true},
	}, nil
}

// GetIncentive 获取激励数据
// 参数:
//   - ctx: 上下文
//   - request: 获取激励数据的请求
//
// 返回:
//   - GetIncentiveResponse: 包含激励数据的响应
//   - error: 错误信息
func (p *PoTExecutor) GetIncentive(ctx context.Context, request *pb.GetIncentiveRequest) (*pb.GetIncentiveResponse, error) {
	// 返回空的激励响应（测试实现）
	return &pb.GetIncentiveResponse{}, nil
}

// GenerateTxsForHeight 为指定高度生成模拟交易数据
// 参数:
//   - height: 区块高度
//
// 返回:
//   - Mockblock: 包含生成的交易列表的模拟区块
func (p *PoTExecutor) GenerateTxsForHeight(height uint64) *Mockblock {
	txs := make([][]byte, 0)
	for i := 0; i < 100; i++ {
		bigint := big.NewInt(int64(i))
		tx := crypto.Hash(bigint.Bytes())
		txs = append(txs, tx)
	}
	return &Mockblock{
		Height: height,
		Txs:    txs,
	}
}

// Mockblock 表示一个模拟区块
// 用于测试目的，包含区块高度和交易列表
type Mockblock struct {
	Height uint64   // 区块高度
	Txs    [][]byte // 交易数据列表
}
