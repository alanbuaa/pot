package shuffle

import (
	. "blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/utils"
	"blockchain-crypto/verifiable_draw"
	"fmt"
)

func SimpleShuffle(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1, nodeId uint64) (*verifiable_draw.DrawProof, error) {
	size := uint32(len(pubKeyList))
	permutation, err := utils.GenRandomPermutation(size)
	if err != nil {
		return nil, fmt.Errorf("permutation is nil: %v\n", err)
	}
	return verifiable_draw.Draw(s, size, pubKeyList, size, permutation, prevRCommit, nodeId)
}

func Verify(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1, shuffleProof *verifiable_draw.DrawProof, nodeId uint64) error {
	num := uint32(len(pubKeyList))
	return verifiable_draw.Verify(s, num, pubKeyList, num, prevRCommit, shuffleProof, nodeId)
}
