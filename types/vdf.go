// Package types defines the important functions needed by pot consensus
package types

import (
	"errors"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf/wesolowski_rust"
)

type VDF0res struct {
	Res   []byte
	Epoch uint64
}

type VDF struct {
	*vdf.Vdf
	OutputChan chan *VDF0res
	Finished   bool
}

func NewVDF(outch chan *VDF0res, iteration int, id int64) *VDF {

	return &VDF{Vdf: vdf.New("wesolowski_rust", []byte(""), iteration, id), OutputChan: outch, Finished: true}
}

func (v *VDF) Exec(epoch uint64) error {
	v.Finished = false
	res := v.Vdf.Execute()
	vdfRes := &VDF0res{
		Res:   res,
		Epoch: epoch,
	}
	if res != nil {
		v.OutputChan <- vdfRes
		v.Finished = true
		return nil
	} else {
		return errors.New("Exec vdf0 failed\n")
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
	v.Finished = true
	return err
}

func (v *VDF) IsFinished() bool {
	return v.Finished
}
