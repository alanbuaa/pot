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
	//假设服务器地址为localhost:50051
	serverAddr := "127.0.0.1:9869"

	// 连接到服务器
	conn, err := connectToServer(serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := pb.NewDciExectorClient(conn)

	for true {

		//str := "0x13017a8cdc8a5a3929fdd814c925c9db2ff9875fc0b9fd30250f9126af04d5d1decfa524d9226d5f529c0dc708dc3e2f0dcb254c75848f4f16f972712231602f04165df4a72bc8bceaf4effcc62dc59461eba9801cb66c99a9666553f1ba42d1"
		//addr, err := hexutil.Decode(str)
		////fmt.Println(addr)
		//str1 := "0xd9a26bd283cc41e6bbefea02bbb646e4f4055ab0bd1bf040f485a64f1cc1915f"
		//str2 := "0x4479d3eb7ced7d6da57cfda3897d3437fb52cb7a6805ae1935905a77dec23c67"
		//blockhash, err := hexutil.Decode(str1)
		//txhash, err := hexutil.Decode(str2)
		//dcireward := &pb.DciReward{
		//	Address: addr,
		//	Amount:  100,
		//	ChainID: 1,
		//	DciProof: &pb.DciProof{
		//		Height:    37,
		//		BlockHash: blockhash,
		//		TxHash:    txhash,
		//	},
		//}
		//
		//req := &pb.SendDciRequest{DciReward: []*pb.DciReward{dcireward}}
		//
		//resp, err := client.SendDci(context.Background(), req)
		//if err != nil {
		//	fmt.Println(err)
		//}
		//fmt.Printf("Response: %d\n", resp.IsSuccess)
		//break

		str := "0x13017a8cdc8a5a3929fdd814c925c9db2ff9875fc0b9fd30250f9126af04d5d1decfa524d9226d5f529c0dc708dc3e2f0dcb254c75848f4f16f972712231602f04165df4a72bc8bceaf4effcc62dc59461eba9801cb66c99a9666553f1ba42d1"
		addr, err := hexutil.Decode(str)
		//fmt.Println(addr)

		req := &pb.GetBalanceRequest{
			Address: addr,
		}

		resp, err := client.GetBalance(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(resp.Balance)
		//txid := resp.GetUtxos()[0].GetTxid()
		//txinput := types.TxInput{
		//	IsCoinbase: false,
		//	Txid:       crypto.Convert(txid),
		//	Voutput:    1,
		//	Scriptsig:  addr,
		//	Value:      2000,
		//	Address:    addr,
		//}
		//
		//txoutput := types.TxOutput{
		//	Address:  nil,
		//	Value:    2000,
		//	IsSpent:  false,
		//	ScriptPk: addr,
		//	Proof:    addr,
		//}
		//tx := types.RawTx{
		//	Txid:           [32]byte{},
		//	TxInput:        []types.TxInput{txinput},
		//	TxOutput:       []types.TxOutput{txoutput},
		//	CoinbaseProofs: nil,
		//}
		//request := &pb.DevastateDciRequest{
		//	Amount:      2000,
		//	Tx:          tx.ToProto(),
		//	To:          addr,
		//	Transaction: types.RandByte(),
		//}
		//
		//response, err := client.DevastateDci(context.Background(), request)
		//fmt.Println(response.Flag)
		break
	}

	//privkey := crypto.GenerateKey()
	////pubkey := privkey.PublicKey()
	//fmt.Println(hexutil.Encode(privkey.PublicKeyBytes()))CommitBlock
}
