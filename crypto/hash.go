package crypto

import "crypto/sha256"

const (
	Hashlen = 32
)

var (
	niltx      = make([]byte, 0)
	NilTxsHash = Hash(niltx)
)

func Hash(input []byte) []byte {

	sum := sha256.Sum256(input)
	return sum[:]
}

func HashFixed(input []byte) [32]byte {
	return sha256.Sum256(input)
}

func Convert(input []byte) [Hashlen]byte {
	var res [Hashlen]byte
	if len(input) != Hashlen {
		return res
	} else {
		for i := 0; i < Hashlen; i++ {
			res[i] = input[i]
		}
		return res
	}
}

// ComputeMerkleRoot compute the root hash of a merkle tree composed of input hashes.
// The function does not check the validity of the hash ( i.e. duplicate hashes).
// Relevant checks should be done beforehand.
func ComputeMerkleRoot(hashes [][]byte) []byte {
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
			hash := sha256.Sum256(append(hashes[i], hashes[i+1]...))
			nextLevel = append(nextLevel, hash[:])
		}
		hashes = nextLevel
	}
	return hashes[0]
}
