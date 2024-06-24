package main

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/grpc"
)

func main() {

	listen, err := net.Listen("tcp", "127.0.0.1:9877")
	if err != nil {
		return
	}
	rpcserver := grpc.NewServer()
	exec := NewExec()
	pb.RegisterPoTExecutorServer(rpcserver, exec)
	go rpcserver.Serve(listen)
	defer rpcserver.Stop()
	defer listen.Close()
	for {
		time.Sleep(20 * time.Second)
		blocks := exec.GenerateTxsForHeight(exec.height)
		exec.blocks = append(exec.blocks, blocks)
		exec.height += 1
	}
}

type PoTexecutor struct {
	height uint64
	blocks []*Testblock
}

// mustEmbedUnimplementedPoTExecutorServer implements pb.PoTExecutorServer.
func (p *PoTexecutor) mustEmbedUnimplementedPoTExecutorServer() {
	panic("unimplemented")
}

func NewExec() *PoTexecutor {
	return &PoTexecutor{
		height: 0,
		blocks: make([]*Testblock, 0),
	}
}

func (p *PoTexecutor) GetTxs(ctx context.Context, request *pb.GetTxRequest) (*pb.GetTxResponse, error) {
	fmt.Printf("receive request, start %d, end %d\n", request.StartHeight, p.height)
	start := request.GetStartHeight()
	if start > p.height {
		return &pb.GetTxResponse{}, nil
	}
	execblock := make([]*pb.ExecuteBlock, 0)
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
		End:    p.height - 1,
		Blocks: execblock,
	}, nil

}

func (p *PoTexecutor) VerifyTxs(ctx context.Context, request *pb.VerifyTxRequest) (*pb.VerifyTxResponse, error) {
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

func (p *PoTexecutor) CommitTxs(ctx context.Context, request *pb.CommitTxsRequest) (*pb.CommitTxsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PoTexecutor) GenerateTxsForHeight(height uint64) *Testblock {
	txs := make([][]byte, 0)
	for i := 0; i < 1000; i++ {
		bigint := big.NewInt(int64(i))
		tx := crypto.Hash(bigint.Bytes())
		txs = append(txs, tx)
	}
	return &Testblock{
		Height: height,
		Txs:    txs,
	}
}

type Testblock struct {
	Height uint64
	Txs    [][]byte
}
