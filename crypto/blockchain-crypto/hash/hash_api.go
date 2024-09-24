package hash

import (
	"blockchain-crypto/hash/poseidon"
	"blockchain-crypto/hash/ripemd160"
	"blockchain-crypto/hash/scrypt"
	"blockchain-crypto/hash/sha256"
	"blockchain-crypto/hash/sha3"
	"blockchain-crypto/hash/sha512"
)

// Args 可选参数结构体
type Args struct {
	// salt []byte, N, r, p, keyLen int
	salt   []byte
	N      int
	r      int
	p      int
	keyLen int
}

// Option 自定义一个func
type Option func(option *Args)

func WithArgs(salt []byte, N, r, p, keyLen int) Option {
	return func(args *Args) {
		args.salt = salt
		args.N = N
		args.r = r
		args.p = p
		args.keyLen = keyLen
	}
}

func Hash(hashType string, input []byte, options ...Option) []byte {

	var bytes []byte

	switch hashType {
	case "ripemd160":
		h := ripemd160.New()
		h.Write(input)
		bytes = h.Sum(nil)
	case "sha256":
		h := sha256.New()
		h.Write(input)
		bytes = h.Sum(nil)
	case "sha512":
		h := sha512.New()
		h.Write(input)
		bytes = h.Sum(nil)
	case "sha3-256":
		h := sha3.New256()
		h.Write(input)
		bytes = h.Sum(nil)
	case "sha3-512":
		h := sha3.New512()
		h.Write(input)
		bytes = h.Sum(nil)
	case "keccak256":
		h := sha3.NewLegacyKeccak256()
		h.Write(input)
		bytes = h.Sum(nil)
	case "keccak512":
		h := sha3.NewLegacyKeccak512()
		h.Write(input)
		bytes = h.Sum(nil)
	case "poseidon256":
		res, _ := poseidon.HashBytes(input)
		bytes = res.Bytes()
	case "scrypt":
		args := &Args{}
		for _, option := range options {
			option(args)
			bytes, _ = scrypt.Key(input, args.salt, args.N, args.r, args.p, args.keyLen)
		}
	default:
		return nil
	}

	return bytes
}
