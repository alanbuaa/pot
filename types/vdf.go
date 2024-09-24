// Package types defines the important functions needed by pot consensus
package types

import (
	"blockchain-crypto/vdf"
	"blockchain-crypto/vdf/wesolowski_rust"
	"errors"
)

type VDF0res struct {
	Res   []byte
	Epoch uint64
}

type VDF struct {
	ID int64
	*vdf.Vdf
	OutputChan chan *VDF0res
	Finished   bool
}

func NewVDF(outch chan *VDF0res, iteration int, id int64) *VDF {
	return &VDF{Vdf: vdf.New("wesolowski_rust", []byte(""), iteration, id), OutputChan: outch, Finished: true, ID: id}
}

func NewVDFwithInput(outch chan *VDF0res, input []byte, iteration int, id int64) *VDF {
	return &VDF{Vdf: vdf.New("wesolowski_rust", input, iteration, id), OutputChan: outch, Finished: true, ID: id}
}

func (v *VDF) Exec(epoch uint64) error {

	v.Finished = false
	defer v.setFinished()
	// start := time.Now()
	res, err := v.Vdf.Execute()
	if err != nil {
		return err
	}

	vdfRes := &VDF0res{
		Res:   res,
		Epoch: epoch,
	}
	if res != nil {
		// exectime := time.Since(start) / time.Millisecond
		// if v.ID != 3 {
		//	timestop := math.Floor(float64(exectime) * float64((10-8)/10))
		//	time.Sleep(time.Duration(timestop) * time.Millisecond)
		//	fmt.Printf("vdf execute need %d ms\n", time.Since(start)/time.Millisecond)
		// }
		v.OutputChan <- vdfRes
		// v.Finished = true
		return nil
	} else {
		return errors.New("Exec vdf0 failed for res is nil\n")
	}
}

func CheckVDF(challenge []byte, iterations int, res []byte) bool {
	return wesolowski_rust.Verify(challenge, iterations, res)
}

func (v *VDF) SetInput(input []byte, iteration int) error {
	if iteration < 1 {
		return errors.New("iteration can't be smaller than 1")
	} else if !v.IsFinished() {
		return errors.New("last epoch VDF haven't finished")
	} else {
		v.Vdf.Challenge = input
		v.Vdf.Iterations = iteration
		return nil
	}
}

func (v *VDF) Abort() error {
	err := v.Vdf.Abort()
	defer v.setFinished()
	return err
}

func (v *VDF) IsFinished() bool {
	return v.Finished
}

func (v *VDF) setFinished() {
	v.Finished = true
}
