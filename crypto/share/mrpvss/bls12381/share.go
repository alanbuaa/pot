package mrpvss

import (
	dleq "github.com/zzz136454872/upgradeable-consensus/crypto/proof/dleq/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

type EncShare struct {
	// 加密份额
	A     *PointG1
	BList []*PointG1
	// 分发份额证明
	DealProof *dleq.Proof
}

type RoundShare struct {
	Share *PointG1
	Proof *dleq.Proof
}
