package utils

import (
	"os"
	"path"
	"runtime"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
)

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func PanicOnLogError(err error, log *logrus.Entry) {
	if err != nil {
		log.Fatalf("panic: %v", err)
	}
}

func LogOnError(err error, msg string, log *logrus.Entry) {
	if err != nil {
		log.WithError(err).Warn(msg)
	}
}

func ChangeWD2ProjectRoot(relativePath string) {
	_, filename, _, _ := runtime.Caller(1)
	dir := path.Join(path.Dir(filename), relativePath)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

// EncodeShortPrint encodes b as a hex string without 0x prefix.
// If b is longer than 8 bytes, only the first 8 bytes are encoded.
func EncodeShortPrint(b []byte) string {
	data := b
	if len(b) > 8 {
		data = b[:8]
	}
	return hexutil.Encode(data)
}
