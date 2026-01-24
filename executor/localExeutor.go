package executor

import (
	"context"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type LocalExecutor struct {
	counter int
	log     *logrus.Entry
}

func NewLocalExecutor(cfg *config.ExecutorConfig, uplog *logrus.Entry) *LocalExecutor {
	return &LocalExecutor{
		counter: 0,
		log:     uplog.WithField("module", "EXECUTOR"),
	}
}

func (e *LocalExecutor) CommitBlock(block types.ConsensusBlock, proof []byte, cid int64) {
	e.log.WithFields(logrus.Fields{
		"consensus_id": cid,
		"tx_count":     len(block.GetTxs()),
	}).Trace("[TRACE-6] LocalExecutor.CommitBlock called - EXECUTION STARTED")

	for _, rtx := range block.GetTxs() {
		tx, err := types.RawTransaction(rtx).ToTx()
		if tx.Type != pb.TransactionType_NORMAL {
			e.log.WithField("type", tx.Type.String()).Debug("Skipping non-NORMAL transaction")
			continue
		}
		utils.PanicOnError(err)
		e.log.WithField("payload", string(tx.Payload)).Trace("[TRACE-6.1] Executing transaction")
		split := strings.Split(string(tx.Payload), ",")
		arg1, _ := strconv.Atoi(split[0])
		arg2, _ := strconv.Atoi(split[1])
		e.counter++
		rawReceipt := []byte(strconv.Itoa(e.counter) + "--" + strconv.Itoa(arg1+arg2))
		msg := &pb.Msg{Payload: &pb.Msg_Reply{Reply: &pb.Reply{Tx: rtx, Receipt: rawReceipt}}}
		msgByte, err := proto.Marshal(msg)
		utils.PanicOnError(err)
		address := "localhost:9999"
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			e.log.Warn("connect to ", address, "failed")
			return
		}
		client := pb.NewP2PClient(conn)
		packet := &pb.Packet{
			Msg:         msgByte,
			ConsensusID: -1,
			Epoch:       -1,
			Type:        pb.PacketType_CLIENTPACKET,
		}

		if _, err = client.Send(context.Background(), packet); err != nil {
			e.log.Trace("send to ", address, "failed")
		}
		conn.Close()
	}
}

func (e *LocalExecutor) VerifyTx(tx types.RawTransaction) bool {
	return true
}
