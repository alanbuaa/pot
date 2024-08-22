package roots_of_unity

import (
	"context"
	"fmt"

	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	"github.com/zzz136454872/upgradeable-consensus/crypto/proof/caulk_plus"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
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

	client := pb.NewCpServiceClient(conn)
	res, err := client.CalcRootsOfUnity(context.TODO(), &pb.DomainSize{Size: size})
	if err != nil {
		return nil, err
	}
	if size != res.Size {
		return nil, fmt.Errorf("wrong size of unity roots: %d != %d", size, res.Size)
	}
	ret := make([]*Fr, size)
	for i := uint32(0); i < size; i++ {
		ret[i] = caulk_plus.ConvertProtoFrToFr(res.Roots[i])
	}
	return ret, nil
}
