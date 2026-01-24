package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"math/big"

	"github.com/zzz136454872/upgradeable-consensus/crypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/executor"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// ExecutorServiceImpl 实现 ExecutorServer 接口
type ExecutorServiceImpl struct {
	pb.UnimplementedExecutorServer
	logger *logrus.Entry
}

// 交易流程：Client -> PoT Executor -> Committee Consensus -> Client(CommitBlock) ->
func main() {
	// 初始化日志系统
	// parse command-line flags
	cfgPath := flag.String("c", "config/config.yaml", "path to config yaml file")
	flag.Parse()

	if cfgPath == nil || len(*cfgPath) == 0 {
		fmt.Printf("config path is empty. Use -c to specify the config file")
		os.Exit(2)
	}

	cfg, err := config.NewConfig(*cfgPath)
	if err != nil {
		fmt.Printf("read config failed: %v", err)
		os.Exit(1)
	}

	logging.Setup(*cfgPath)
	logger := logging.GetLogger().WithField("module", "EXECUTOR")

	logger.Info("Configuration loaded successfully")

	// 初始化交易发送客户端
	txClient, conn, err := initTxClient(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Tx client")
	}
	defer conn.Close()

	// 初始化通用 Executor 服务器
	executorServer, executorListen, err := initGeneralExecutorService(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize general Executor service")
	}
	defer executorListen.Close()
	defer executorServer.Stop()

	// 初始化 PoT Executor 服务
	rpcserver, listen, exec, err := initPoTExecutorService(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize PoT Executor service")
	}
	defer listen.Close()
	defer rpcserver.Stop()

	// 启动事务生成循环
	runTransactionLoop(txClient, exec, logger)
}

// runTransactionLoop 持续生成测试交易数据并发送普通交易
func runTransactionLoop(txClient pb.P2PClient, exec *executor.PoTExecutor, logger *logrus.Entry) {
	logger.Info("Transaction generation started (interval: 2s)")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 设置优雅关闭信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sentCount := 0
	failedCount := 0

	for {
		select {
		case <-sigChan:
			logger.Info("Shutdown signal received, stopping...")
			logger.WithFields(logrus.Fields{
				"total_sent":   sentCount,
				"total_failed": failedCount,
				"final_height": exec.Height,
			}).Info("Final statistics")
			return

		case <-ticker.C:
			// 为当前高度生成测试区块
			logger.WithField("height", exec.Height).Trace("Generating test blocks for current height")
			block := generateTxsForHeight(exec.Height)
			exec.Blocks = append(exec.Blocks, block)
			exec.Height += 1

			// 生成并发送随机普通交易（格式："数字1,数字2"）
			innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))
			tx := &pb.Transaction{Type: pb.TransactionType_NORMAL, Payload: []byte(innerTx)}
			logger.WithField("payload", innerTx).Trace("Sending transaction to consensus network")
			if err := sendTransaction(txClient, tx, logger); err != nil {
				logger.WithError(err).WithField("payload", innerTx).Warn("Transaction send failed")
				failedCount++
			} else {
				sentCount++
				if sentCount%10 == 0 {
					logger.WithFields(logrus.Fields{
						"sent":   sentCount,
						"failed": failedCount,
						"height": exec.Height,
					}).Info("Transaction statistics")
				}
			}
		}
	}
}

// sendTransaction 发送交易到共识网络
func sendTransaction(client pb.P2PClient, tx *pb.Transaction, logger *logrus.Entry) error {
	btx, err := proto.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// POT 共识使用固定的 sharding
	// 该名称在 consensus/pot/run_commitee.go 中通过 hexutil.EncodeUint64(1) 生成
	request := &pb.Request{Tx: btx, Sharding: []byte(hexutil.EncodeUint64(1))}

	rawRequest, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	packet := &pb.Packet{
		Msg:         rawRequest,
		ConsensusID: -1,
		Epoch:       -1,
		Type:        pb.PacketType_CLIENTPACKET,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Send(ctx, packet)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	logger.Trace("Transaction sent successfully")
	return nil
}

// initTxClient 初始化交易发送客户端，建立与共识节点的 gRPC 连接
func initTxClient(cfg *config.Config, logger *logrus.Entry) (pb.P2PClient, *grpc.ClientConn, error) {
	logger.WithField("address", cfg.Node.RpcAddress).Info("Connecting to consensus node")
	conn, err := grpc.NewClient(cfg.Node.RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.WithError(err).WithField("address", cfg.Node.RpcAddress).Error("Failed to connect to consensus node")
		return nil, nil, err
	}
	txClient := pb.NewP2PClient(conn)
	logger.Info("Tx client gRPC connection to consensus node established")
	return txClient, conn, nil
}

// initGeneralExecutorService 初始化通用 Executor 服务器
func initGeneralExecutorService(cfg *config.Config, logger *logrus.Entry) (*grpc.Server, net.Listener, error) {
	execAddr := cfg.Executor.Address
	logger.WithField("address", execAddr).Info("Initializing general Executor service")

	execServiceImpl := &ExecutorServiceImpl{
		logger: logger,
	}
	executorServer := grpc.NewServer()
	pb.RegisterExecutorServer(executorServer, execServiceImpl)

	executorListen, err := net.Listen("tcp", execAddr)
	if err != nil {
		logger.WithError(err).WithField("address", execAddr).Error("Failed to start general Executor service")
		return nil, nil, err
	}

	logger.WithField("address", execAddr).Info("General Executor service listening")

	// 在后台启动 gRPC 服务器
	go func() {
		if err := executorServer.Serve(executorListen); err != nil {
			logger.WithError(err).Error("General Executor service stopped")
		}
	}()

	return executorServer, executorListen, nil
}

// initPoTExecutorService 初始化 PoT Executor 服务
func initPoTExecutorService(cfg *config.Config, logger *logrus.Entry) (*grpc.Server, net.Listener, *executor.PoTExecutor, error) {
	potExecAddr := cfg.Consensus.PoT.ExecutorAddress
	logger.WithField("address", potExecAddr).Info("Initializing PoT Executor service")

	listen, err := net.Listen("tcp", potExecAddr)
	if err != nil {
		logger.WithError(err).WithField("address", potExecAddr).Error("Failed to start PoT Executor service")
		return nil, nil, nil, err
	}

	logger.WithField("address", potExecAddr).Info("PoT Executor service listening")

	rpcserver := grpc.NewServer()
	exec := executor.NewPoTExecutor()
	pb.RegisterPoTExecutorServer(rpcserver, exec)
	logger.Info("PoT Executor service registered")

	// 在后台启动 gRPC 服务
	go func() {
		if err := rpcserver.Serve(listen); err != nil {
			logger.WithError(err).Error("PoT Executor service stopped")
		}
	}()

	return rpcserver, listen, exec, nil
}

// generateTxsForHeight 为指定高度生成模拟交易
func generateTxsForHeight(height uint64) *executor.Mockblock {
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

	return &executor.Mockblock{
		Height: height,
		Txs:    txs,
	}
}

// CommitBlock 实现 ExecutorServer 的 CommitBlock 方法
func (e *ExecutorServiceImpl) CommitBlock(ctx context.Context, block *pb.ExecBlock) (*pb.Empty, error) {
	e.logger.WithFields(logrus.Fields{
		"sharding": string(block.ShardingName),
		"tx_count": len(block.Txs),
		"random":   block.RandomNumber,
	}).Trace("[TRACE-7] *** ExecutorServiceImpl.CommitBlock CALLED - BLOCK COMMITTED ***")
	// 这里可以调用内部的 executor.CommitBlock 方法
	// 注意：需要将 pb.ExecBlock 转换为 types.ConsensusBlock
	return &pb.Empty{}, nil
}

// VerifyTx 实现 ExecutorServer 的 VerifyTx 方法
func (e *ExecutorServiceImpl) VerifyTx(ctx context.Context, tx *pb.Transaction) (*pb.Result, error) {
	e.logger.WithFields(logrus.Fields{
		"type":    tx.Type.String(),
		"payload": string(tx.Payload),
	}).Trace("Verifying transaction")
	// 这里可以调用内部的 executor.VerifyTx 方法
	return &pb.Result{Success: true}, nil
}
