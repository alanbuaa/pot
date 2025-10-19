package crypto

import (
	"hash"

	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
)

// HashFactory is a wrapper around utils.HashFactory for backward compatibility
func HashFactory(h string) (hash.Hash, error) {
	return utils.HashFactory(h)
}
