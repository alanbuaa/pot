package pot

import (
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestPrivkey(t *testing.T) {

	conn, err := grpc.Dial("127.0.0.1:9876", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Error(err)
	}
	_ = pb.NewExecutorClient(conn)

}
