// package main
//
// import (
//
//	"context"
//	"fmt"
//	"github.com/zzz136454872/upgradeable-consensus/crypto"
//	"github.com/zzz136454872/upgradeable-consensus/pb"
//	"google.golang.org/grpc"
//	"math/big"
//	"net"
//	"time"
//
// )
//
// func main() {
//
//		listen, err := net.Listen("tcp", "127.0.0.1:9877")
//		if err != nil {
//			return
//		}
//		rpcserver := grpc.NewServer()
//		exec := NewExec()
//		pb.RegisterPoTExecutorServer(rpcserver, exec)
//		go rpcserver.Serve(listen)
//		defer rpcserver.Stop()
//		defer listen.Close()
//		for {
//			time.Sleep(20 * time.Second)
//			blocks := exec.GenerateTxsForHeight(exec.height)
//			exec.blocks = append(exec.blocks, blocks)
//			exec.height += 1
//		}
//	}
//
//	type PoTexecutor struct {
//		height uint64
//		blocks []*Testblock
//	}
//
//	func NewExec() *PoTexecutor {
//		return &PoTexecutor{
//			height: 0,
//			blocks: make([]*Testblock, 0),
//		}
//	}
//
// func (p *PoTexecutor) GetTxs(ctx context.Context, request *pb.GetTxRequest) (*pb.GetTxResponse, error) {
//
//	fmt.Printf("receive request, start %d, end %d\n", request.StartHeight, p.height)
//	start := request.GetStartHeight()
//	if start > p.height {
//		return &pb.GetTxResponse{}, nil
//	}
//	execblock := make([]*pb.ExecuteBlock, 0)
//	for i := start; i < uint64(len(p.blocks)); i++ {
//		header := &pb.ExecuteHeader{Height: i}
//		txs := make([]*pb.ExecutedTx, 0)
//		for _, tx := range p.blocks[i].Txs {
//			etx := &pb.ExecutedTx{
//				TxHash: tx,
//				Height: i,
//				Data:   nil,
//			}
//			txs = append(txs, etx)
//		}
//		blocks := &pb.ExecuteBlock{
//			Header: header,
//			Txs:    txs,
//		}
//		execblock = append(execblock, blocks)
//	}
//
//	return &pb.GetTxResponse{
//		Start:  start,
//		End:    p.height - 1,
//		Blocks: execblock,
//	}, nil
//
// }
//
//	func (p *PoTexecutor) VerifyTxs(ctx context.Context, request *pb.VerifyTxRequest) (*pb.VerifyTxResponse, error) {
//		flag := make([]bool, len(request.GetTxs()))
//		for i := 0; i < len(flag); i++ {
//			flag[i] = true
//		}
//		reponse := &pb.VerifyTxResponse{
//			Txs:  request.Txs,
//			Flag: flag,
//		}
//		return reponse, nil
//	}
//
//	func (p *PoTexecutor) GenerateTxsForHeight(height uint64) *Testblock {
//		txs := make([][]byte, 0)
//		for i := 0; i < 1000; i++ {
//			bigint := big.NewInt(int64(i))
//			tx := crypto.Hash(bigint.Bytes())
//			txs = append(txs, tx)
//		}
//		return &Testblock{
//			Height: height,
//			Txs:    txs,
//		}
//	}
//
//	type Testblock struct {
//		Height uint64
//		Txs    [][]byte
//	}
package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/grpc"
	"log"
)

// 连接到gRPC服务器
func connectToServer(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	return conn, nil
}

// 创建客户端并调用服务
func main() {
	// 假设服务器地址为localhost:50051
	serverAddr := "127.0.0.1:9867"

	// 连接到服务器
	conn, err := connectToServer(serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := pb.NewDciExectorClient(conn)

	for true {

		str := "0x06c5a8ddb574b9a5497cb6cd20bad05a05da85332a227f988c285d1b848b5195725e1089e2c8302916b4d167df7fb0280de64605119388401122c16b28a921c1099da44d988e381e0515802ce8af50b64962ea1f91c2bf2f76a0f8d646c5b742"
		addr, err := hexutil.Decode(str)
		fmt.Println(addr)
		req := &pb.GetBalanceRequest{
			Address:   addr,
			Signature: nil,
		}

		resp, err := client.GetBalance(context.Background(), req)
		if err != nil {
			break
		}
		fmt.Printf("Response: %d\n", resp.Balance)
		break
	}
}
