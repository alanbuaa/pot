package helper

import (
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"google.golang.org/protobuf/proto"
)

func DecodeRequest(rawRequest []byte) (*pb.Request, error) {
	request := new(pb.Request)
	err := proto.Unmarshal(rawRequest, request)
	return request, err
}
