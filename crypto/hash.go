package crypto

import "crypto/sha256"

const Hashlen = 32

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
