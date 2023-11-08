package basic

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

func TestBasicHotStuff_ReceiveMsg(t *testing.T) {
	wd, err := os.Getwd()
	utils.PanicOnError(err)
	t.Log("pwd", wd)
	cfg, err := config.NewConfig("config/config.yaml", 1)
	utils.PanicOnError(err)
	cfg.Consensus.Upgradeable.InitConsensus.Nodes = cfg.Consensus.Nodes
	stuff := NewBasicHotStuff(2, 10001, cfg.Consensus.Upgradeable.InitConsensus, nil, nil, logging.GetLogger().WithField("test", true))
	request := &pb.Msg_Request{Request: &pb.Request{
		Tx: []byte("1+2"),
	}}
	msg := &pb.Msg{Payload: request}
	msgByte, err := proto.Marshal(msg)
	utils.PanicOnError(err)
	stuff.MsgByteEntrance <- msgByte
	stuff.MsgByteEntrance <- msgByte
	stuff.MsgByteEntrance <- msgByte
	stuff.Stop()
	time.Sleep(time.Millisecond * 200)
}

func TestRemoveDBFile(t *testing.T) {
	err := os.RemoveAll("dbfile/")
	if err != nil {
		t.Fatal(err)
	}
}
