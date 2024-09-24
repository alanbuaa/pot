package roots_of_unity

import (
	pb2 "blockchain-crypto/pb"
	"blockchain-crypto/proof/caulk_plus"
	. "blockchain-crypto/types/curve/bls12381"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO
var (
	serviceHost = "127.0.0.1:50051"
)

func CalcRootsOfUnity(size uint32) ([]*Fr, error) {

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client := pb2.NewCpServiceClient(conn)
	res, err := client.CalcRootsOfUnity(context.TODO(), &pb2.DomainSize{Size: size})
	if err != nil {
		return nil, err
	}
	if size != res.Size {
		return nil, errors.New(fmt.Sprintf("wrong size of unity roots: %d != %d", size, res.Size))
	}
	ret := make([]*Fr, size)
	for i := uint32(0); i < size; i++ {
		ret[i] = caulk_plus.ConvertProtoFrToFr(res.Roots[i])
	}
	return ret, nil
}
