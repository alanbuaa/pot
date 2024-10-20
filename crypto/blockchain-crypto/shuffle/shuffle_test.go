package shuffle

import (
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/utils"
	"fmt"
	"testing"
	"time"
)

func TestSimpleShuffle(t *testing.T) {
	group1 := bls12381.NewG1()
	candidatesNum := uint32(16)
	quota := uint32(16)
	s, _ := srs.NewSRS(utils.CalcMinLimitedDegree(candidatesNum, quota))
	fmt.Println(1)
	generator := s.G1PowerOf(0)

	secretKeys := make([]*bls12381.Fr, candidatesNum)
	publicKeys := make([]*bls12381.PointG1, candidatesNum)
	for i := uint32(0); i < candidatesNum; i++ {
		secretKeys[i] = bls12381.FrFromUInt32(i + 1)
		publicKeys[i] = group1.Affine(group1.MulScalar(group1.New(), generator, secretKeys[i]))
	}
	fmt.Println(2)
	shuffleProof, err := SimpleShuffle(s, publicKeys, generator)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(3)
	for i := 0; i < 10; i++ {
		go func() {
			err := Verify(s, publicKeys, generator, shuffleProof)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(true)
			}
		}()
	}
	time.Sleep(10 * time.Second)
}
