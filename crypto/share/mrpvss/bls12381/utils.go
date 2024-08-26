package mrpvss

import (
	"errors"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

// bruteForceFindExp only use for small scalar which is on the exponent of target
func bruteForceFindExp(g *PointG1, target *PointG1, step *Fr, max uint32) (*Fr, error) {
	if group1.IsZero(target) {
		return NewFr().Zero(), nil
	}
	tmp := NewFr().Set(step)
	for i := uint32(1); i < max; i++ {
		tmpPoint := group1.MulScalar(group1.New(), g, tmp)
		if group1.Equal(target, tmpPoint) {
			return tmp, nil
		}
		tmp.Add(tmp, step)
	}
	return nil, errors.New("cannot find exponent")
}
