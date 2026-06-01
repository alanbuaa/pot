package chained

import (
	"os"
	"testing"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/logging"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) {
	err := os.Chdir("../../../")
	utils.PanicOnError(err)
	os.Exit(m.Run())
}

func TestChainedHotStuff(t *testing.T) {
	cfg, err := config.NewConfig("config/config.yaml", 1)
	utils.PanicOnError(err)
	cfg.Consensus.Upgradeable.InitConsensus.Nodes = cfg.Consensus.Nodes
	chained := NewChainedHotStuff(1, 10001, cfg.Consensus.Upgradeable.InitConsensus, nil, nil, logging.GetLogger().WithField("test", 1))
	request := &pb.Msg_Request{Request: &pb.Request{
		Tx: []byte("1+2"),
	}}
	msg := &pb.Msg{Payload: request}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)
	chained.MsgByteEntrance <- msgByte
	time.Sleep(200 * time.Millisecond)
	defer chained.Stop()
}
