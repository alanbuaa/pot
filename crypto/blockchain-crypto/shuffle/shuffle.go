package shuffle

import (
	. "blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/utils"
	"blockchain-crypto/verifiable_draw"
	"fmt"
)

func SimpleShuffle(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1) *verifiable_draw.DrawProof {
	size := uint32(len(pubKeyList))
	permutation, err := utils.GenRandomPermutation(size)
	if err != nil {
		fmt.Println("permutation is nil", err)
		return nil
	}
	shuffleProof, err := verifiable_draw.Draw(s, size, pubKeyList, size, permutation, prevRCommit)
	if err != nil {
		fmt.Println("draw fail", err)
		return nil
	}

	return shuffleProof
}

func Verify(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1, shuffleProof *verifiable_draw.DrawProof) bool {
	num := uint32(len(pubKeyList))
	return verifiable_draw.Verify(s, num, pubKeyList, num, prevRCommit, shuffleProof)
}