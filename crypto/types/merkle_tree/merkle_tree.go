package merkle_tree

import (
	"sort"

	"github.com/zzz136454872/upgradeable-consensus/crypto/hash"
)

// ComputeMerkleRoot compute the root hash of a merkle tree composed of input hashes.
// If sortedRoot is true, compute the root hash of a sorted merkle tree.
// The function does not check the validity of the hash ( i.e. duplicate hashes).
// Relevant checks should be done beforehand.
func ComputeMerkleRoot(hashes [][]byte, sortedRoot bool) []byte {
	if sortedRoot {
		sort.Slice(hashes, func(i, j int) bool { return string(hashes[i]) < string(hashes[j]) })
	}
	if len(hashes) == 0 {
		return nil
	}
	for len(hashes) > 1 {
		// padding for odd num
		if len(hashes)&1 == 1 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		var nextLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			h := hash.Hash("sha256", append(hashes[i], hashes[i+1]...), nil)
			nextLevel = append(nextLevel, h)
		}
		hashes = nextLevel
	}
	return hashes[0]
}
