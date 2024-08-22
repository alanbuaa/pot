package caulk_plus

import (
	"context"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO
var (
	serviceHost = "127.0.0.1:50051"
)

func CreateMultiProof(parentVectorSize uint32, parentVector []*Fr, subVectorSize uint32, subVector []*Fr) (*pb.MultiProof, error) {
	parentVectorPb := make([]*pb.Fr, parentVectorSize)
	for i := uint32(0); i < parentVectorSize; i++ {
		parentVectorPb[i] = ConvertFrToProtoFr(parentVector[i])
	}
	subVectorPb := make([]*pb.Fr, subVectorSize)
	for i := uint32(0); i < subVectorSize; i++ {
		subVectorPb[i] = ConvertFrToProtoFr(subVector[i])
	}

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client := pb.NewCpServiceClient(conn)
	proof, err := client.CreateMultiProof(context.TODO(), &pb.CreateMultiProofRequest{
		ParentVectorSize: parentVectorSize,
		ParentVector:     parentVectorPb,
		SubVectorSize:    subVectorSize,
		SubVector:        subVectorPb,
	})
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func VerifyMultiProof(proof *pb.MultiProof) (bool, error) {

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client := pb.NewCpServiceClient(conn)
	res, err := client.VerifyMultiProof(context.TODO(), proof)
	if err != nil {
		return false, err
	}
	return res.Res, nil
}

func CreateSingleProof(hGenerator *PointG1, parentVectorSize uint32, parentVector []*Fr, chosenElement *Fr) (*pb.SingleProof, error) {

	parentVectorPb := make([]*pb.Fr, parentVectorSize)
	for i := uint32(0); i < parentVectorSize; i++ {
		parentVectorPb[i] = ConvertFrToProtoFr(parentVector[i])
	}

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client := pb.NewCpServiceClient(conn)
	proof, err := client.CreateSingleProof(context.TODO(), &pb.CreateSingleProofRequest{
		HG1Generator:     ConvertPointG1ToProtoG1Affine(hGenerator),
		ParentVectorSize: parentVectorSize,
		ParentVector:     parentVectorPb,
		ChosenElement:    ConvertFrToProtoFr(chosenElement),
	})
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func VerifySingleProof(hGenerator *PointG1, proof *pb.SingleProof) (bool, error) {

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client := pb.NewCpServiceClient(conn)
	res, err := client.VerifySingleProof(context.TODO(), &pb.VerifySingleProofRequest{
		HGenerator:  ConvertPointG1ToProtoG1Affine(hGenerator),
		SingleProof: proof,
	})
	if err != nil {
		return false, err
	}
	return res.Res, nil
}
