package shuffle

import (
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
	"github.com/zzz136454872/upgradeable-consensus/crypto/verifiable_draw"
)

func SimpleShuffle(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1) *verifiable_draw.DrawProof {
	size := uint32(len(pubKeyList))
	permutation, err := utils.GenRandomPermutation(size)
	if err != nil {
		return nil
	}
	shuffleProof, err := verifiable_draw.Draw(s, size, pubKeyList, size, permutation, prevRCommit)
	if err != nil {
		return nil
	}
	return shuffleProof
}

func Verify(s *srs.SRS, pubKeyList []*PointG1, prevRCommit *PointG1, shuffleProof *verifiable_draw.DrawProof) bool {
	num := uint64(len(pubKeyList))
	return verifiable_draw.Verify(s, num, pubKeyList, uint32(num), prevRCommit, shuffleProof)
}
