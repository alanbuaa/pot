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
	"github.com/zzz136454872/upgradeable-consensus/logging"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

var logger = logging.GetLogger()

const timeoutDuration = time.Second * 2

type command string

type Client struct {
	consensusResult        map[command]reply
	config                 *config.Config
	requestTimeout         *utils.Timer
	requestTimeoutDuration time.Duration
	replyChan              chan *pb.Msg
	closeChan              chan int
	p2pClient              pb.P2PClient
	wg                     *sync.WaitGroup
	mut                    *sync.Mutex
	log                    *logrus.Entry

	pendingTx *atomic.Int64
	successTx *atomic.Int64

	// latency
	sendTime    map[command]time.Time
	sendTimeMut *sync.Mutex

	pb.UnimplementedP2PServer
}

type reply struct {
	result string
	count  int
}

// Send receive packet from consensus
func (gc *Client) Send(ctx context.Context, in *pb.Packet) (*pb.Empty, error) {
	msg := new(pb.Msg)
	err := proto.Unmarshal(in.Msg, msg)
	if err != nil {
		logger.WithError(err).Warn("received msg error")
		return &pb.Empty{}, nil
	}
	gc.replyChan <- msg
	return &pb.Empty{}, nil
}

func NewClient(log *logrus.Entry) *Client {
	cfg, err := config.NewConfig("config/configpot.yaml", 0)
	utils.PanicOnError(err)
	conn, err := grpc.Dial(cfg.Nodes[0].RpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
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
	}

	client.requestTimeout.Init()
	client.requestTimeout.Stop()
	return client
}

func (client *Client) getResults(cmd command) (re reply, b bool) {
	client.mut.Lock()
	defer client.mut.Unlock()
	re, b = client.consensusResult[cmd]
	return
}

func (client *Client) setResult(cmd command, re reply) {
	client.mut.Lock()
	defer client.mut.Unlock()
	client.consensusResult[cmd] = re
}

func (client *Client) getSendTime(cmd command) (st time.Time, b bool) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	st, b = client.sendTime[cmd]
	return
}

func (client *Client) setSendTime(cmd command, st time.Time) {
	client.sendTimeMut.Lock()
	defer client.sendTimeMut.Unlock()
	client.sendTime[cmd] = st
}

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
					if re.count == (len(client.config.Nodes)+2)/3 {
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

func (client *Client) sendTx(tx *pb.Transaction) {
	btx, err := proto.Marshal(tx)
	utils.PanicOnError(err)
	request := &pb.Request{Tx: btx}
	rawRequest, err := proto.Marshal(request)
	utils.PanicOnError(err)

	packet := &pb.Packet{
		Msg:         rawRequest,
		ConsensusID: -1,
		Epoch:       -1,
		Type:        pb.PacketType_CLIENTPACKET,
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

func (client *Client) normalRun() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	client.wg.Add(2)

	// use goroutine send msg
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		for {
			select {
			case <-client.closeChan:
				client.wg.Done()
				return
			case <-ticker.C:
				innerTx := strconv.Itoa(rand.Intn(1000)) + "," + strconv.Itoa(rand.Intn(1000))
				// client.log.WithField("content", cmd).Info("[CLIENT] Send request")
				tx := &pb.Transaction{Type: pb.TransactionType_NORMAL, Payload: []byte(innerTx)}
				client.sendTx(tx)
			}
		}
	}()

	// start client server
	clientServer := grpc.NewServer()
	pb.RegisterP2PServer(clientServer, client)
	listen, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		panic(err)
	}
	go client.receiveReply()
	go clientServer.Serve(listen)
	<-c
	client.log.Info("[CLIENT] Client exit...")
	clientServer.Stop()
}

func (c *Client) upgradeConsensus(nc string) {
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
			Snum:    2,
			SysPara: "123456789",
		}
		cf.Type = "pot"
		cf.PoT = ptcf
	default:
		logger.Warn("not supported yet")
		return
	}
	rcf, err := json.Marshal(cf)
	if err != nil {
		c.log.Warn("encode json failed")
	}
	tx := &pb.Transaction{Type: pb.TransactionType_UPGRADE, Payload: rcf}
	c.sendTx(tx)
}

func (client *Client) Stop() {
	close(client.closeChan)
	client.wg.Wait()
}

func main() {
	logger.SetOutput(os.Stdout) // only output to stdout
	client := NewClient(logger.WithField("module", "client"))
	if len(os.Args) == 1 {
		client.normalRun()
	} else if len(os.Args) == 3 {
		if os.Args[1] != "upgrade" {
			logger.Infof("method not supported")
		}
		client.upgradeConsensus(os.Args[2])
	} else {
		logger.Infof("wrong number of args")
	}
	client.Stop()
}
