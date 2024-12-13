package caulk_plus

import (
	"blockchain-crypto/pb"
	. "blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO
var (
	serviceHost = "127.0.0.1:50051"
)

func CreateMultiProof(s *srs.SRS, parentVectorSize uint32, parentVector []*Fr, subVectorSize uint32, subVector []*Fr) (*pb.MultiProof, error) {
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
		Srs:              ConvertSRSToProtoSRS(s),
	})
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func VerifyMultiProof(s *srs.SRS, proof *pb.MultiProof) (bool, error) {

	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	res, err := pb.NewCpServiceClient(conn).VerifyMultiProof(context.TODO(), &pb.VerifyMultiProofRequest{
		MultiProof: proof,
		Srs:        ConvertSRSToProtoSRS(s),
	})
	if err != nil {
		return false, err
	}
	return res.Res, nil
}

func CreateSingleProof(s *srs.SRS, hGenerator *PointG1, parentVectorSize uint32, parentVector []*Fr, chosenElement *Fr) (*pb.SingleProof, error) {

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
		Srs:              ConvertSRSToProtoSRS(s),
	})
	if err != nil {
		return nil, err
	}
	return proof, nil
}

func VerifySingleProof(s *srs.SRS, hGenerator *PointG1, proof *pb.SingleProof) (bool, error) {
	conn, err := grpc.NewClient(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	res, err := pb.NewCpServiceClient(conn).VerifySingleProof(context.TODO(), &pb.VerifySingleProofRequest{
		HGenerator:  ConvertPointG1ToProtoG1Affine(hGenerator),
		SingleProof: proof,
		Srs:         ConvertSRSToProtoSRS(s),
	})
	if err != nil {
		return false, err
	}
	return res.Res, nil
}
