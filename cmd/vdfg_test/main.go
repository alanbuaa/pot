package main

import "github.com/zzz136454872/upgradeable-consensus/crypto/vdf"

//func main()  {
//
//
//	for i := 0; i < 2000; i++ {
//
//		vdfi := vdf.New("",[]byte("aa"),10000,0)
//
//		go vdfi.Execute()
//	}
//
//	time.Sleep(10 * time.Second)
//}

func vdfProcess(vdf *vdf.VDF, wg *sync.WaitGroup) {
	defer wg.Done()
	startTime := time.Now()
	_, _, _ = vdf.Execute()
	endTime := time.Since(startTime) / time.Millisecond
	fmt.Printf("vdf %d cpu-%d %dms\n", vdf.Id, vdf.Controller.CpuNo, endTime)
}

func main() {
	fmt.Println(utils.GetCurrentAbPathByCaller())

	startTime := time.Now()
	var wg sync.WaitGroup

	cnt := 7
	wg.Add(cnt)
	challenge := []byte{170}
	vdfList := make([]*vdf.VDF, cnt)
	for i := 0; i < cnt; i++ {
		vdfList[i], _ = vdf.New(uint64(i+1), "", challenge, 100000)
	}

	for i := 0; i < cnt; i++ {
		go vdfProcess(vdfList[i], &wg)
	}
	time.Sleep(1 * time.Second)
	err := vdfList[0].Abort()
	if err != nil {
		return
	}
	wg.Wait()

	fmt.Printf("Benchmark finished in %dms\n", time.Since(startTime)/time.Millisecond)
}
