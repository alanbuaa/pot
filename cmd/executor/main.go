package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

func main() {
	// 初始化日志系统
	logging.Setup("config/config.yaml")
	logger := logging.GetLogger().WithField("module", "EXECUTOR")
	logger.Info("Executor starting...")

	// 加载配置文件
	logger.Debug("Loading configuration from config/config.yaml")
	cfg, err := config.NewConfig("config/config.yaml", 0)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}
	logger.Info("Configuration loaded successfully")

	// 建立与共识节点的 gRPC 连接
	logger.WithField("address", cfg.Nodes[0].RpcAddress).Info("Connecting to consensus node")
	conn, err := grpc.NewClient(cfg.Nodes[0].RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.WithError(err).WithField("address", cfg.Nodes[0].RpcAddress).Fatal("Failed to connect to consensus node")
	}
	defer conn.Close()
	p2pClient := pb.NewP2PClient(conn)
	logger.Info("gRPC connection to consensus node established")

	// 创建通用 Executor 服务器，监听 cfg.Executor.Address

	execAddr := cfg.Executor.Address
	logger.WithField("address", execAddr).Info("Initializing general Executor service")
	execServiceImpl := &ExecutorServiceImpl{
		logger: logger,
	}
	executorServer := grpc.NewServer()
	pb.RegisterExecutorServer(executorServer, execServiceImpl)

	executorListen, err := net.Listen("tcp", execAddr)
	if err != nil {
		logger.WithError(err).WithField("address", execAddr).Fatal("Failed to start general Executor service")
	}
	defer executorListen.Close()
	logger.WithField("address", execAddr).Info("General Executor service listening")

	// 启动 Executor gRPC 服务器
	go func() {
		if err := executorServer.Serve(executorListen); err != nil {
			logger.WithError(err).Error("General Executor service stopped")
		}
	}()
	defer executorServer.Stop()

	potExecAddr := cfg.Consensus.PoT.ExecutorAddress
	logger.WithField("address", potExecAddr).Info("Initializing PoT Executor service")
	listen, err := net.Listen("tcp", potExecAddr)
	if err != nil {
		logger.WithError(err).WithField("address", potExecAddr).Fatal("Failed to start PoT Executor service")
	}
	defer listen.Close()
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
	defer rpcserver.Stop()

	// 主循环：持续生成测试交易数据并发送普通交易
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
			block := exec.GenerateTxsForHeight(exec.Height)
			exec.Blocks = append(exec.Blocks, block)
			exec.Height += 1

			// 生成并发送随机普通交易（格式："数字1,数字2"）
			innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))
			tx := &pb.Transaction{Type: pb.TransactionType_NORMAL, Payload: []byte(innerTx)}
			logger.WithField("payload", innerTx).Trace("Sending transaction to consensus network")
			if err := SendTransaction(p2pClient, tx, logger); err != nil {
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

// SendTransaction 发送交易到共识网络
func SendTransaction(client pb.P2PClient, tx *pb.Transaction, logger *logrus.Entry) error {
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
