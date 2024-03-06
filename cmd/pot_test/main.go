package main

import (
	"github.com/zzz136454872/upgradeable-consensus/logging"
	"github.com/zzz136454872/upgradeable-consensus/node"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
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
	total := 4
	nodes := make([]*node.Node, total)

	for i := int64(0); i < int64(total); i++ {
		go func(index int64) {
			nodes[index] = node.NewNode(index)
		}(i)
	}
	<-sigChan
	logger.Info("[UpgradeableConsensus] Exit...")
	for i := 0; i < total; i++ {
		nodes[i].Stop()
	}
}

//func vdfProcess(vdf *vdf.Vdf, wg *sync.WaitGroup) {
//	defer wg.Done()
//	startTime := time.Now()
//	_ = vdf.Execute()
//	endTime := time.Since(startTime) / time.Millisecond
//	fmt.Printf("vdf-%d cpu-%d %dms\n", vdf.Iterations, vdf.Controller.CpuNo, endTime)
//}
//
//func main() {
//	startTime := time.Now()
//	var wg sync.WaitGroup
//
//	cnt := 9
//	wg.Add(cnt + 1)
//	challenge := []byte{170}
//	vdfList := make([]*vdf.Vdf, cnt)
//	for i := 0; i < cnt; i++ {
//		vdfList[i] = vdf.New("", challenge, 20001+i, int64(i+1))
//	}
//	for i := 1; i < cnt; i++ {
//		go vdfProcess(vdfList[i], &wg)
//	}
//	time.Sleep(1 * time.Second)
//	go vdfProcess(vdfList[0], &wg)
//	time.Sleep(1 * time.Second)
//	err := vdfList[0].Abort()
//	if err != nil {
//		return
//	}
//	wg.Wait()
//
//	fmt.Printf("Benchmark finished in %dms\n", time.Since(startTime)/time.Millisecond)
//}
