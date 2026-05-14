package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const timeoutDuration = time.Second * 2

// UpgradeClient 是支持共识升级的客户端
type UpgradeClient struct {
	consensusResult        map[string]reply     // 存储共识结果
	config                 *config.ClientConfig // 系统配置
	requestTimeout         *utils.Timer         // 请求超时计时器
	requestTimeoutDuration time.Duration        // 超时时长
	replyChan              chan *pb.Msg         // 接收回复消息的通道
	closeChan              chan int             // 关闭信号通道
	p2pClient              pb.P2PClient         // P2P 客户端
	wg                     *sync.WaitGroup      // 等待组
	mut                    *sync.Mutex          // 保护 consensusResult
	log                    *logrus.Entry        // 日志记录器

	pendingTx *atomic.Int64 // 待处理交易数量
	successTx *atomic.Int64 // 成功交易数量
	totalTx   int           // 总交易数量

	// 延迟统计
	sendTime    map[string]time.Time
	sendTimeMut *sync.Mutex

	// 升级相关
	currentProposalID types.TxHash // 当前提案ID
	proposalLock      *sync.Mutex  // 保护提案状态

	pb.UnimplementedP2PServer
}

// reply 表示共识回复结果
type reply struct {
	result string
	count  int
}

// NewUpgradeClient 创建客户端实例
func NewUpgradeClient(cfg *config.ClientConfig, log *logrus.Entry) *UpgradeClient {
	log.Infof("Connecting to consensus node: %s", cfg.RpcAddress)
	conn, err := grpc.NewClient(cfg.RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithError(err).Errorf("Failed to dial gRPC server at %s", cfg.RpcAddress)
		panic(err)
	}
	p2pClient := pb.NewP2PClient(conn)
	log.Info("Successfully connected to consensus node")

	client := &UpgradeClient{
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
		proposalLock:           new(sync.Mutex),
		log:                    log,
		pendingTx:              new(atomic.Int64),
		successTx:              new(atomic.Int64),
		UnimplementedP2PServer: pb.UnimplementedP2PServer{},
		totalTx:                0,
	}

	client.requestTimeout.Init()
	client.requestTimeout.Stop()
	log.Info("UpgradeClient initialized successfully")
	return client
}

// getResults 获取指定命令的共识结果
func (client *UpgradeClient) getResults(cmd string) (re reply, b bool) {
	client.mut.Lock()
	defer client.mut.Unlock()
	re, b = client.consensusResult[cmd]
	return
}

// setResult 设置指定命令的共识结果
func (client *UpgradeClient) setResult(cmd string, re reply) {
	client.mut.Lock()
	defer client.mut.Unlock()
	client.consensusResult[cmd] = re
}

// getSendTime 获取发送时间
func (client *UpgradeClient) getSendTime(cmd string) (st time.Time, b bool) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	st, b = client.sendTime[cmd]
	return
}

// setSendTime 设置发送时间
func (client *UpgradeClient) setSendTime(cmd string, st time.Time) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	client.sendTime[cmd] = st
}

// Send 接收来自共识节点的数据包
func (client *UpgradeClient) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
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

// receiveReply 持续接收并处理来自共识节点的回复
func (client *UpgradeClient) receiveReply() {
	client.log.Debug("Reply receiver started")
	for {
		select {
		case msg := <-client.replyChan:
			replyMsg := msg.GetReply()
			if replyMsg == nil {
				continue
			}
			client.log.Debug("Received reply message")
			tx := new(pb.Transaction)
			if err := proto.Unmarshal(replyMsg.Tx, tx); err != nil {
				client.log.WithError(err).Warn("Failed to unmarshal transaction from reply")
				continue
			}
			cmd := string(tx.Payload)
			var duration time.Duration
			if sTime, ok := client.getSendTime(cmd); !ok {
				client.log.WithField("cmd_len", len(cmd)).Warn("Received reply for unsent transaction")
				continue
			} else {
				duration = time.Since(sTime)
			}

			if re, ok := client.getResults(cmd); ok {
				if re.result == string(replyMsg.Receipt) {
					re.count++
					if re.count == 1 {
						client.pendingTx.Add(-1)
						client.successTx.Add(1)
						client.log.WithFields(logrus.Fields{
							"tx_type": tx.Type.String(),
							"latency": duration,
							"pending": client.pendingTx.Load(),
							"success": client.successTx.Load(),
						}).Info("Transaction confirmed")
					}
					client.setResult(cmd, re)
				}
			} else {
				re := reply{
					result: string(replyMsg.Receipt),
					count:  1,
				}
				client.setResult(cmd, re)
				client.pendingTx.Add(-1)
				client.successTx.Add(1)
				client.log.WithFields(logrus.Fields{
					"tx_type": tx.Type.String(),
					"latency": duration,
					"pending": client.pendingTx.Load(),
					"success": client.successTx.Load(),
				}).Info("First reply received, transaction confirmed")
			}
		case <-client.closeChan:
			client.log.Debug("Reply receiver stopped")
			client.wg.Done()
			return
		}
	}
}

// sendTx 发送交易到共识网络
func (client *UpgradeClient) sendTx(tx *pb.Transaction) error {
	btx, err := proto.Marshal(tx)
	if err != nil {
		return err
	}
	client.totalTx += 1

	request := &pb.Request{Tx: btx, Sharding: []byte("0x1")}
	rawRequest, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	packet := &pb.Packet{
		Msg:         rawRequest,
		ConsensusID: -1,
		Epoch:       -1,
		Type:        pb.PacketType_CLIENTPACKET,
	}
	client.pendingTx.Add(1)
	cmd := string(tx.Payload)
	client.log.WithFields(logrus.Fields{
		"type":    tx.Type.String(),
		"pending": client.pendingTx.Load(),
		"success": client.successTx.Load(),
	}).Debug("Sending transaction")
	client.setSendTime(cmd, time.Now())
	_, err = client.p2pClient.Send(context.Background(), packet)
	if err != nil {
		client.log.WithError(err).Error("Failed to send transaction")
		client.pendingTx.Add(-1)
		return err
	}
	client.log.Trace("Transaction sent successfully")
	return nil
}

// normalRun 启动正常运行模式
func (client *UpgradeClient) normalRun(duration time.Duration) {
	client.log.Info("Starting client in normal run mode")

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	client.wg.Add(2)

	// 启动定时发送交易的 goroutine
	stopChan := make(chan struct{})
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
			case <-stopChan:
				client.log.Debug("Transaction sender stopped by timer")
				client.wg.Done()
				return
			case <-ticker.C:
				innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))
				tx := &pb.Transaction{Type: pb.TransactionType_NORMAL, Payload: []byte(innerTx)}
				client.sendTx(tx)
			}
		}
	}()

	// 启动客户端 gRPC 服务器
	clientServer := grpc.NewServer()
	pb.RegisterP2PServer(clientServer, client)
	listen, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		client.log.WithError(err).Fatal("Failed to listen on port 9999")
	}
	client.log.Info("gRPC server listening on localhost:9999")

	go client.receiveReply()
	go clientServer.Serve(listen)

	client.log.Info("-------------------------------------------------")
	client.log.Info("  Client is running, sending continuous transactions")
	client.log.Info("-------------------------------------------------")

	// 根据 duration 决定是定时退出还是等待信号
	if duration > 0 {
		select {
		case <-time.After(duration):
			client.log.Info("Duration reached, stopping...")
			close(stopChan)
		case <-c:
			client.log.Info("Shutdown signal received")
		}
	} else {
		<-c
	}

	client.log.Info("Stopping services...")
	clientServer.Stop()

	// 打印最终统计
	client.log.Info("=================================================")
	client.log.WithFields(logrus.Fields{
		"total_tx":   client.totalTx,
		"success_tx": client.successTx.Load(),
		"pending_tx": client.pendingTx.Load(),
	}).Info("Transaction Statistics")
	client.log.Info("=================================================")
}

// proposeUpgrade 发起共识升级提案
func (client *UpgradeClient) proposeUpgrade(targetConsensus string) {
	client.log.WithField("target_consensus", targetConsensus).Info("=================================================")
	client.log.Info("  Initiating Consensus Upgrade Proposal")
	client.log.Info("=================================================")

	// 创建共识配置
	cf := new(config.ConsensusConfig)
	cf.ConsensusID = rand.Int63()%100000 + 10000

	switch targetConsensus {
	case "basichotstuff", "hotstuff":
		client.log.Info("Configuring Basic HotStuff consensus")
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
	case "basicwhirly", "whirly":
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
	default:
		client.log.WithField("consensus_type", targetConsensus).Warn("Unsupported consensus type, using default hotstuff")
		hcf := &config.HotStuffConfig{
			Type:         "basic",
			BatchSize:    10,
			BatchTimeout: 1,
			Timeout:      2,
		}
		cf.Type = "hotstuff"
		cf.HotStuff = hcf
	}

	// 创建升级提案
	proposal, err := upgrade.CreateUpgradeProposal(
		cf.Type,
		nil,                   // CDL descriptor
		10,                    // current height (会被节点覆盖)
		1,                     // threshold (测试用，只需要1个确认)
		0,                     // incentive
		[]byte("client"),      // proposer
		"Upgrade to "+cf.Type, // description
	)
	if err != nil {
		client.log.WithError(err).Error("Failed to create upgrade proposal")
		return
	}

	// 设置具体的共识参数
	proposal.ConsensusID = cf.ConsensusID
	if cf.HotStuff != nil {
		proposal.ConsensusParams = map[string]interface{}{
			"type":          cf.HotStuff.Type,
			"batch_size":    cf.HotStuff.BatchSize,
			"batch_timeout": cf.HotStuff.BatchTimeout,
			"timeout":       cf.HotStuff.Timeout,
		}
	} else if cf.Whirly != nil {
		proposal.ConsensusParams = map[string]interface{}{
			"type":       cf.Whirly.Type,
			"batch_size": cf.Whirly.BatchSize,
			"timeout":    cf.Whirly.Timeout,
		}
	}

	// 保存当前提案ID
	client.proposalLock.Lock()
	client.currentProposalID = proposal.ProposalID
	client.proposalLock.Unlock()

	// 打包并发送交易
	tx, err := upgrade.PackUpgradeTransaction(proposal)
	if err != nil {
		client.log.WithError(err).Error("Failed to pack upgrade transaction")
		return
	}

	client.log.WithFields(logrus.Fields{
		"proposal_id":      hex.EncodeToString(proposal.ProposalID[:8]),
		"target_consensus": proposal.TargetConsensus,
		"consensus_id":     proposal.ConsensusID,
		"fork_height":      proposal.ForkHeight,
		"switch_height":    proposal.SwitchHeight,
	}).Info("Sending upgrade proposal transaction")

	err = client.sendTx(tx)
	if err != nil {
		client.log.WithError(err).Error("Failed to send upgrade proposal")
		return
	}

	client.log.Info("-------------------------------------------------")
	client.log.Info("  Upgrade proposal sent successfully!")
	client.log.WithField("proposal_id", hex.EncodeToString(proposal.ProposalID[:8])).Info("  Use -confirm to confirm this proposal")
	client.log.Info("-------------------------------------------------")
}

// confirmUpgrade 确认升级提案
func (client *UpgradeClient) confirmUpgrade(proposalIDHex string) {
	client.log.WithField("proposal_id", proposalIDHex).Info("=================================================")
	client.log.Info("  Confirming Upgrade Proposal")
	client.log.Info("=================================================")

	// 解析提案ID
	var proposalID types.TxHash
	if proposalIDHex == "auto" {
		// 使用自动生成的提案ID（测试用）
		client.proposalLock.Lock()
		proposalID = client.currentProposalID
		client.proposalLock.Unlock()
		if proposalID == (types.TxHash{}) {
			client.log.Error("No active proposal to confirm. Send a proposal first with -upgrade flag")
			return
		}
	} else {
		idBytes, err := hex.DecodeString(proposalIDHex)
		if err != nil {
			client.log.WithError(err).Error("Invalid proposal ID format")
			return
		}
		copy(proposalID[:], idBytes)
	}

	// 创建确认交易
	confirmTx, err := upgrade.CreateConfirmTransaction(
		proposalID,
		true,     // approved
		[]byte{}, // signature (简化测试，不签名)
		0,        // confirmer ID
	)
	if err != nil {
		client.log.WithError(err).Error("Failed to create confirm transaction")
		return
	}

	// 打包交易
	tx, err := upgrade.PackConfirmTransaction(confirmTx)
	if err != nil {
		client.log.WithError(err).Error("Failed to pack confirm transaction")
		return
	}

	client.log.WithFields(logrus.Fields{
		"proposal_id":  hex.EncodeToString(proposalID[:8]),
		"confirmer_id": confirmTx.ConfirmerId,
		"approved":     confirmTx.Approved,
	}).Info("Sending confirm transaction")

	err = client.sendTx(tx)
	if err != nil {
		client.log.WithError(err).Error("Failed to send confirm transaction")
		return
	}

	client.log.Info("-------------------------------------------------")
	client.log.Info("  Confirm transaction sent successfully!")
	client.log.Info("  Node will switch consensus at the target height")
	client.log.Info("-------------------------------------------------")
}

// Stop 停止客户端
func (client *UpgradeClient) Stop() {
	client.log.Debug("Stopping client...")
	close(client.closeChan)
	client.wg.Wait()
	client.log.Debug("All goroutines stopped")
}

// printStatus 打印当前状态（用于调试）
func (client *UpgradeClient) printStatus() {
	client.log.Info("=== Client Status ===")
	client.log.WithFields(logrus.Fields{
		"total_tx":   client.totalTx,
		"pending_tx": client.pendingTx.Load(),
		"success_tx": client.successTx.Load(),
	}).Info("Transaction Statistics")

	client.proposalLock.Lock()
	if client.currentProposalID != (types.TxHash{}) {
		client.log.WithField("proposal_id", hex.EncodeToString(client.currentProposalID[:8])).Info("Current Proposal")
	}
	client.proposalLock.Unlock()
}

// marshalJSON 辅助函数
func marshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
