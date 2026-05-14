package main

import (
	"context"
	"encoding/json"
	"math/big"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const timeoutDuration = time.Second * 2

// Client 是区块链客户端的主要结构体
type Client struct {
	consensusResult        map[string]reply     // 存储共识结果，按命令索引
	config                 *config.ClientConfig // 系统配置
	requestTimeout         *utils.Timer         // 请求超时计时器
	requestTimeoutDuration time.Duration        // 超时时长
	replyChan              chan *pb.Msg         // 接收回复消息的通道
	closeChan              chan int             // 关闭信号通道
	p2pClient              pb.P2PClient         // P2P 客户端
	wg                     *sync.WaitGroup      // 等待组，用于协程同步
	mut                    *sync.Mutex          // 保护 consensusResult 的互斥锁
	log                    *logrus.Entry        // 日志记录器

	pendingTx *atomic.Int64 // 待处理交易数量（原子操作）
	successTx *atomic.Int64 // 成功交易数量（原子操作）
	totalTx   int           // 总交易数量

	// 延迟统计相关字段
	sendTime    map[string]time.Time // 记录每个命令的发送时间
	sendTimeMut *sync.Mutex          // 保护 sendTime 的互斥锁

	pb.UnimplementedP2PServer // gRPC 服务器接口实现
}

// reply 表示共识回复结果
type reply struct {
	result string // 回复结果内容
	count  int    // 收到该结果的次数
}

// NewClient 创建并返回一个新的客户端实例
func NewClient(cfg *config.ClientConfig, log *logrus.Entry) *Client {
	// 建立与共识节点的 gRPC 连接
	log.Infof("Connecting to consensus node: %s", cfg.RpcAddress)
	conn, err := grpc.NewClient(cfg.RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithError(err).Errorf("Failed to dial gRPC server at %s", cfg.RpcAddress)
		panic(err)
	}
	p2pClient := pb.NewP2PClient(conn)
	log.Info("Successfully connected to consensus node")

	client := &Client{
		consensusResult:        make(map[string]reply),
		sendTime:               make(map[string]time.Time),
		config:                 cfg,
		requestTimeoutDuration: timeoutDuration,
		requestTimeout:         utils.NewTimer(timeoutDuration),
		replyChan:              make(chan *pb.Msg),
		closeChan:              make(chan int),
		p2pClient:              p2pClient,
		wg:                     new(sync.WaitGroup),
		mut:                    new(sync.Mutex),
		sendTimeMut:            new(sync.Mutex),
		log:                    log,
		pendingTx:              new(atomic.Int64),
		successTx:              new(atomic.Int64),
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
		totalTx:                0,
	}

	client.requestTimeout.Init()
	client.requestTimeout.Stop()
	log.Info("Client initialized successfully")
	return client
}

// getResults 获取指定命令的共识结果（线程安全）
func (client *Client) getResults(cmd string) (re reply, b bool) {
	client.mut.Lock()
	defer client.mut.Unlock()
	re, b = client.consensusResult[cmd]
	return
}

// setResult 设置指定命令的共识结果（线程安全）
func (client *Client) setResult(cmd string, re reply) {
	client.mut.Lock()
	defer client.mut.Unlock()
	client.consensusResult[cmd] = re
}

// getSendTime 获取指定命令的发送时间（线程安全）
func (client *Client) getSendTime(cmd string) (st time.Time, b bool) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	st, b = client.sendTime[cmd]
	return
}

// setSendTime 设置指定命令的发送时间（线程安全）
func (client *Client) setSendTime(cmd string, st time.Time) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	client.sendTime[cmd] = st
}

// Send 接收来自共识节点的数据包
// 实现了 P2P 服务器接口，用于接收共识回复消息
func (client *Client) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	msg := new(pb.Msg)
	err := proto.Unmarshal(in.Msg, msg)
	if err != nil {
		client.log.WithError(err).Error("Failed to unmarshal received message")
		return &pb.Empty{}, nil
	}
	client.log.Debug("Received message from consensus node")
	client.replyChan <- msg
	return &pb.Empty{}, nil
}

// receiveReply 持续接收并处理来自共识节点的回复消息
// 该方法在独立的 goroutine 中运行，处理交易确认和延迟统计
func (client *Client) receiveReply() {
	client.log.Debug("Reply receiver started")
	for {
		select {
		case msg := <-client.replyChan:
			replyMsg := msg.GetReply()
			client.log.Debug("Received reply message")
			tx := new(pb.Transaction)
			if err := proto.Unmarshal(replyMsg.Tx, tx); err != nil {
				client.log.WithError(err).Warn("Failed to unmarshal transaction from reply")
				continue
			}
			cmd := string(tx.Payload)
			var duration time.Duration
			if sTime, ok := client.getSendTime(cmd); !ok {
				client.log.WithField("cmd", cmd).Warn("Received reply for unsent transaction")
				continue
			} else {
				duration = time.Since(sTime)
			}

			if re, ok := client.getResults(cmd); ok {
				if re.result == string(replyMsg.Receipt) {
					re.count++
					// if re.count == (len(client.config.Nodes)+2)/3 {
					if re.count == 2 {
						client.pendingTx.Add(-1)
						client.successTx.Add(1)
						client.log.WithFields(logrus.Fields{
							"cmd":     cmd,
							"result":  re.result,
							"latency": duration,
							"total":   client.pendingTx.Load() + client.successTx.Load(),
							"pending": client.pendingTx.Load(),
							"success": client.successTx.Load(),
						}).Info("Transaction consensus reached")
					}
					client.setResult(cmd, re)
				}
			} else {
				re := reply{
					result: string(replyMsg.Receipt),
					count:  1,
				}
				client.setResult(cmd, re)
				client.log.WithFields(logrus.Fields{
					"payload": string(tx.Payload),
					"latency": duration,
				}).Debug("First reply received for transaction")
			}
		case <-client.closeChan:
			client.log.Debug("Reply receiver stopped")
			client.wg.Done()
			return
		}
	}
}

// sendTx 发送交易到共识网络
func (client *Client) sendTx(tx *pb.Transaction) {
	btx, err := proto.Marshal(tx)
	utils.PanicOnError(err)
	client.totalTx += 1
	// POT 共识使用固定的 sharding 名称 "0x1"
	// 该名称在 consensus/pot/run_commitee.go 中通过 hexutil.EncodeUint64(1) 生成
	request := &pb.Request{Tx: btx, Sharding: []byte("0x1")}

	rawRequest, err := proto.Marshal(request)
	utils.PanicOnError(err)

	packet := &pb.Packet{
		Msg:         rawRequest,
		ConsensusID: -1,
		Epoch:       -1,
		Type:        pb.PacketType_CLIENTPACKET,
		// ReceiverPublicAddress: BroadcastToAll,
	}
	client.pendingTx.Add(1)
	cmd := string(tx.Payload)
	client.log.WithFields(logrus.Fields{
		"type":    tx.Type.String(),
		"payload": string(tx.Payload),
		"total":   client.pendingTx.Load() + client.successTx.Load(),
		"pending": client.pendingTx.Load(),
		"success": client.successTx.Load(),
	}).Debug("Sending transaction")
	client.setSendTime(cmd, time.Now())
	_, err = client.p2pClient.Send(context.Background(), packet)
	if err != nil {
		client.log.WithError(err).WithField("payload", string(tx.Payload)).Error("Failed to send transaction")
		client.pendingTx.Add(-1) // 发送失败，回滚 pending 计数
	} else {
		client.log.WithField("payload", string(tx.Payload)).Trace("Transaction sent successfully")
	}
}

// normalRun 启动客户端的正常运行模式
// 定期发送随机交易到共识网络，并启动 gRPC 服务器接收回复
func (client *Client) normalRun() {
	client.log.Info("Starting client in normal run mode")
	// 设置信号处理，用于优雅退出
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	client.wg.Add(2)

	// 启动定时发送交易的 goroutine
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()
		client.log.Debug("Transaction sender started (interval: 1s)")
		for {
			select {
			case <-client.closeChan:
				client.log.Debug("Transaction sender stopped")
				client.wg.Done()
				return
			case <-ticker.C:
				// 生成随机交易内容（格式："数字1,数字2"）
				innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))
				// 创建普通类型交易
				tx := &pb.Transaction{Type: pb.TransactionType_NORMAL, Payload: []byte(innerTx)}
				client.sendTx(tx)
			}
		}
	}()

	// 启动客户端 gRPC 服务器，用于接收共识节点的回复
	clientServer := grpc.NewServer()
	pb.RegisterP2PServer(clientServer, client)
	listen, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		client.log.WithError(err).Fatal("Failed to listen on port 9999")
	}
	client.log.Info("gRPC server listening on localhost:9999")
	// 启动回复接收 goroutine
	go client.receiveReply()
	// 启动 gRPC 服务
	go clientServer.Serve(listen)
	// 等待退出信号
	client.log.Info("Client is running, press Ctrl+C to exit")
	<-c
	client.log.Info("Shutdown signal received, stopping services...")
	clientServer.Stop()
	client.log.Debug("gRPC server stopped")
}

// upgradeConsensus 发起共识升级交易
// 根据指定的共识类型构造升级配置并发送到网络
func (c *Client) upgradeConsensus(nc string) {
	c.log.WithField("target_consensus", nc).Info("Initiating consensus upgrade")
	// 创建新的共识配置
	cf := new(config.ConsensusConfig)
	cf.ConsensusID = rand.Int63()%100000 + 10000
	switch nc {
	case "basichotstuff":
		c.log.Info("Configuring Basic HotStuff consensus")
		hcf := &config.HotStuffConfig{
			Type:         "basic",
			BatchSize:    10,
			BatchTimeout: 1,
			Timeout:      2,
		}
		cf.Type = "hotstuff"
		cf.HotStuff = hcf
	case "eventdrivenhotstuff":
		hcf := &config.HotStuffConfig{
			Type:         "event-driven",
			BatchSize:    10,
			BatchTimeout: 1,
			Timeout:      2,
		}
		cf.Type = "hotstuff"
		cf.HotStuff = hcf
	case "basicwhirly":
		wcf := &config.WhirlyConfig{
			Type:      "basic",
			BatchSize: 10,
			Timeout:   2,
		}
		cf.Type = "whirly"
		cf.Whirly = wcf
	case "simplewhirly":
		wcf := &config.WhirlyConfig{
			Type:      "simple",
			BatchSize: 10,
			Timeout:   2,
		}
		cf.Type = "whirly"
		cf.Whirly = wcf
	case "pow":
		bi := new(big.Int)
		pcf := &config.PoWConfig{
			InitDifficulty: bi.SetBytes([]byte(" 0x0000010000000000000000000000000000000000000000000000000000000000")),
		}
		cf.Type = "pow"
		cf.Pow = pcf
	case "pot":
		ptcf := &config.PoTConfig{
			Snum:          2,
			SysPara:       "123456789",
			Vdf0Iteration: 200000,
			Vdf1Iteration: 160000,
		}
		cf.Type = "pot"
		cf.PoT = ptcf
	default:
		c.log.WithField("consensus_type", nc).Warn("Unsupported consensus type for upgrade")
		return
	}
	rcf, err := json.Marshal(cf)
	if err != nil {
		c.log.WithError(err).Error("Failed to marshal consensus config to JSON")
		return
	}
	c.log.WithFields(logrus.Fields{
		"consensus_type": cf.Type,
		"consensus_id":   cf.ConsensusID,
	}).Info("Sending consensus upgrade transaction")
	tx := &pb.Transaction{Type: pb.TransactionType_UPGRADE, Payload: rcf}
	c.sendTx(tx)
}

// Stop 优雅地停止客户端
// 关闭所有 goroutine 并等待它们结束
func (client *Client) Stop() {
	client.log.Debug("Stopping client...")
	close(client.closeChan)
	client.wg.Wait()
	client.log.Debug("All goroutines stopped")
}
