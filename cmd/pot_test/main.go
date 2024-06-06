package main

import (
	"context"
	config "github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/logging"
	"github.com/zzz136454872/upgradeable-consensus/node"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/grpc"
	"math/big"
	"net"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	logger  = logging.GetLogger()
	sigChan = make(chan os.Signal)
)

//
// func main() {
//	outChan := make(chan []byte, 500)
//
//	vdf := types.NewVDF(outChan, pot.Vdf0Iteration)
//	vdf.SetInput([]byte("aa"), pot.Vdf0Iteration)
//	go receiveChan(outChan, vdf)
//	var wg sync.WaitGroup
//	wg.Add(1)
//	vdf.Exec()
//	wg.Wait()
//
// }
//
// func receiveChan(outpu chan []byte, vdf *types.VDF) {
//	epoch := 0
//	in := []byte("aa")
//	for {
//		select {
//		case res := <-outpu:
//			fmt.Println(epoch)
//			fmt.Println(hex.EncodeToString(res))
//			fmt.Println(types.CheckVDF(in, pot.Vdf0Iteration, res))
//			epoch += 1
//
//			vdfin := crypto.Hash(res)
//
//			vdf.SetInput(vdfin, pot.Vdf0Iteration)
//			in = vdfin
//			go vdf.Exec()
//		}
//	}
// }

func main() {
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	cfg, _ := config.NewConfig("config/configpot.yaml", 0)
	total := cfg.Total
	nodes := make([]*node.Node, total)

	for i := int64(0); i < int64(total); i++ {
		go func(index int64) {

			nodes[index] = node.NewNode(index)
		}(i)
	}
	go testexecutor()
	<-sigChan
	logger.Info("[UpgradeableConsensus] Exit...")
	for i := 0; i < total; i++ {
		nodes[i].Stop()
	}
}

//	func vdfProcess(vdf *vdf.Vdf, wg *sync.WaitGroup) {
//		defer wg.Done()
//		startTime := time.Now()
//		_ = vdf.Execute()
//		endTime := time.Since(startTime) / time.Millisecond
//		fmt.Printf("vdf-%d cpu-%d %dms\n", vdf.Iterations, vdf.Controller.CpuNo, endTime)
//	}
//
//	func main() {
//		startTime := time.Now()
//		var wg sync.WaitGroup
//
//		cnt := 9
//		wg.Add(cnt + 1)
//		challenge := []byte{170}
//		vdfList := make([]*vdf.Vdf, cnt)
//		for i := 0; i < cnt; i++ {
//			vdfList[i] = vdf.New("", challenge, 20001+i, int64(i+1))
//		}
//		for i := 1; i < cnt; i++ {
//			go vdfProcess(vdfList[i], &wg)
//		}
//		time.Sleep(1 * time.Second)
//		go vdfProcess(vdfList[0], &wg)
//		time.Sleep(1 * time.Second)
//		err := vdfList[0].Abort()
//		if err != nil {
//			return
//		}
//		wg.Wait()
//
//		fmt.Printf("Benchmark finished in %dms\n", time.Since(startTime)/time.Millisecond)
//	}
func testexecutor() {
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

func NewExec() *PoTexecutor {
	return &PoTexecutor{
		height: 0,
		blocks: make([]*Testblock, 0),
	}
}

func (p *PoTexecutor) GetTxs(ctx context.Context, request *pb.GetTxRequest) (*pb.GetTxResponse, error) {
	//fmt.Printf("receive request, start %d, end %d\n", request.StartHeight, p.height)
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
