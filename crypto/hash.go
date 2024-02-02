package crypto

import "crypto/sha256"

const HashLen = 32

func Hash(input []byte) []byte {

	sum := sha256.Sum256(input)
	return sum[:]
}

func HashFixed(input []byte) [32]byte {
	return sha256.Sum256(input)
}

func Convert(input []byte) [HashLen]byte {
	var res [HashLen]byte
	if len(input) != HashLen {
		return res
	} else {
		for i := 0; i < HashLen; i++ {
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
