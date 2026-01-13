package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type command string

// Client 是区块链客户端的主要结构体
// 负责发送交易、接收回复、统计交易状态等功能
type Client struct {
	consensusResult        map[command]reply // 存储共识结果，按命令索引
	config                 *config.Config    // 系统配置
	requestTimeout         *utils.Timer      // 请求超时计时器
	requestTimeoutDuration time.Duration     // 超时时长
	replyChan              chan *pb.Msg      // 接收回复消息的通道
	closeChan              chan int          // 关闭信号通道
	p2pClient              pb.P2PClient      // P2P 客户端
	wg                     *sync.WaitGroup   // 等待组，用于协程同步
	mut                    *sync.Mutex       // 保护 consensusResult 的互斥锁
	log                    *logrus.Entry     // 日志记录器

	pendingTx *atomic.Int64 // 待处理交易数量（原子操作）
	successTx *atomic.Int64 // 成功交易数量（原子操作）
	totalTx   int           // 总交易数量

	// 延迟统计相关字段
	sendTime    map[command]time.Time // 记录每个命令的发送时间
	sendTimeMut *sync.Mutex           // 保护 sendTime 的互斥锁

	pb.UnimplementedP2PServer // gRPC 服务器接口实现
}

// reply 表示共识回复结果
// 用于统计相同结果的回复数量
type reply struct {
	result string // 回复结果内容
	count  int    // 收到该结果的次数
}

// Send 接收来自共识节点的数据包
// 实现了 P2P 服务器接口，用于接收共识回复消息
// 参数:
//   - ctx: 上下文
//   - in: 接收到的数据包
//
// 返回:
//   - *pb.Empty: 空响应
//   - error: 错误信息
func (gc *Client) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	msg := new(pb.Msg)
	err := proto.Unmarshal(in.Msg, msg)
	if err != nil {
		gc.log.WithError(err).Warn("received msg error")
		return &pb.Empty{}, nil
	}
	gc.replyChan <- msg
	return &pb.Empty{}, nil
}

// NewClient 创建并返回一个新的客户端实例
// 参数:
//   - log: 日志记录器
//
// 返回:
//   - *Client: 客户端实例
func NewClient(cfg *config.Config, log *logrus.Entry) *Client {
	// 加载配置文件
	// 建立与共识节点的 gRPC 连接
	conn, err := grpc.Dial(cfg.Nodes[0].RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("failed to dial grpc: %v", err)
		panic(err)
	}
	p2pClient := pb.NewP2PClient(conn)
	// rand.Seed(time.Now().UnixNano())

	client := &Client{
		consensusResult:        make(map[command]reply),
		sendTime:               make(map[command]time.Time),
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
	return client
}

// getResults 获取指定命令的共识结果（线程安全）
// 参数:
//   - cmd: 命令标识
//
// 返回:
//   - reply: 回复结果
//   - bool: 是否存在该结果
func (client *Client) getResults(cmd command) (re reply, b bool) {
	client.mut.Lock()
	defer client.mut.Unlock()
	re, b = client.consensusResult[cmd]
	return
}

// setResult 设置指定命令的共识结果（线程安全）
// 参数:
//   - cmd: 命令标识
//   - re: 回复结果
func (client *Client) setResult(cmd command, re reply) {
	client.mut.Lock()
	defer client.mut.Unlock()
	client.consensusResult[cmd] = re
}

// getSendTime 获取指定命令的发送时间（线程安全）
// 用于计算交易延迟
// 参数:
//   - cmd: 命令标识
//
// 返回:
//   - time.Time: 发送时间
//   - bool: 是否存在该记录
func (client *Client) getSendTime(cmd command) (st time.Time, b bool) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	st, b = client.sendTime[cmd]
	return
}

// setSendTime 设置指定命令的发送时间（线程安全）
// 参数:
//   - cmd: 命令标识
//   - st: 发送时间
func (client *Client) setSendTime(cmd command, st time.Time) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	client.sendTime[cmd] = st
}

// receiveReply 持续接收并处理来自共识节点的回复消息
// 该方法在独立的 goroutine 中运行，处理交易确认和延迟统计
func (client *Client) receiveReply() {
	for {
		select {
		case msg := <-client.replyChan:
			replyMsg := msg.GetReply()
			client.log.Info("get reply message: ", string(replyMsg.Tx))
			tx := new(pb.Transaction)
			if proto.Unmarshal(replyMsg.Tx, tx) != nil {
				continue
			}
			cmd := command(tx.Payload)
			var duration time.Duration
			if sTime, ok := client.getSendTime(cmd); !ok {
				client.log.Trace("the reply message has not been sent")
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
							"total":   client.pendingTx.Load() + client.successTx.Load(),
							"pending": client.pendingTx.Load(),
							"success": client.successTx.Load(),
						}).Infof("Consensus success.")
					}
					client.setResult(cmd, re)
				}
			} else {
				re := reply{
					result: string(replyMsg.Receipt),
					count:  1,
				}
				client.setResult(cmd, re)
				client.log.Infof("the message %s latency is %s", string(replyMsg.Tx), duration)
			}
		case <-client.closeChan:
			client.wg.Done()
			return
		}
	}
}

// sendTx 发送交易到共识网络
// 参数:
//   - tx: 待发送的交易
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
	client.log.WithFields(logrus.Fields{
		"total":   client.pendingTx.Load() + client.successTx.Load(),
		"pending": client.pendingTx.Load(),
		"success": client.successTx.Load(),
	}).Info("sending tx")
	cmd := command(tx.Payload)
	client.setSendTime(cmd, time.Now())
	_, err = client.p2pClient.Send(context.Background(), packet)
	utils.LogOnError(err, "send packet failed", client.log)
}

// normalRun 启动客户端的正常运行模式
// 定期发送随机交易到共识网络，并启动 gRPC 服务器接收回复
func (client *Client) normalRun() {
	// 设置信号处理，用于优雅退出
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	client.wg.Add(2)

	// 启动定时发送交易的 goroutine
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		for {
			select {
			case <-client.closeChan:
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
		client.log.Errorf("failed to listen on port 9999: %v", err)
		panic(err)
	}
	// 启动回复接收 goroutine
	go client.receiveReply()
	// 启动 gRPC 服务
	go clientServer.Serve(listen)
	// 等待退出信号
	<-c
	client.log.Info("[CLIENT] Client exit...")
	clientServer.Stop()
}

// upgradeConsensus 发起共识升级交易
// 根据指定的共识类型构造升级配置并发送到网络
// 参数:
//   - nc: 新的共识类型名称（如 "basichotstuff", "pot" 等）
func (c *Client) upgradeConsensus(nc string) {
	// 创建新的共识配置
	cf := new(config.ConsensusConfig)
	cf.ConsensusID = rand.Int63()%100000 + 10000
	switch nc {
	case "basichotstuff":
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
		c.log.Warn("not supported yet")
		return
	}
	rcf, err := json.Marshal(cf)
	if err != nil {
		c.log.Warn("encode json failed")
	}
	tx := &pb.Transaction{Type: pb.TransactionType_UPGRADE, Payload: rcf}
	c.sendTx(tx)
}

// Stop 优雅地停止客户端
// 关闭所有 goroutine 并等待它们结束
func (client *Client) Stop() {
	close(client.closeChan)
	client.wg.Wait()
}

// main 是客户端程序的入口函数
// 支持两种运行模式：
//  1. 无参数：正常运行模式，持续发送交易
//  2. 带参数 "upgrade <consensus_type>"：发起共识升级
func main() {
	// 配置日志输出到标准输出
	logrus.SetOutput(os.Stdout)
	// 加载配置文件
	cfg, err := config.NewConfig("config/config.yaml", 0)
	if err != nil {
		fmt.Println("read config.yaml failed: ", err)
	}
	// 创建客户端实例
	client := NewClient(cfg, logrus.WithField("module", "client"))

	// 根据命令行参数选择运行模式
	if len(os.Args) == 1 {
		// 正常运行模式
		client.normalRun()
	} else if len(os.Args) == 3 {
		// 共识升级模式
		if os.Args[1] != "upgrade" {
			logrus.Infof("method not supported")
		}
		client.upgradeConsensus(os.Args[2])
	} else {
		logrus.Infof("wrong number of args")
	}
	// 停止客户端
	client.Stop()
}
