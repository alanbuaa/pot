package types

import (
	"fmt"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	mrpvss "github.com/zzz136454872/upgradeable-consensus/crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/verifiable_draw"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

var (
	G1 = bls12381.NewG1()
)

type CryptoElement struct {
	SRS            *srs.SRS
	SrsUpdateProof *schnorr_proof.SchnorrProof // SRS更新证明
	// 置换阶段
	ShuffleProof *verifiable_draw.DrawProof // 公钥列表置换证明（包含置换输出的公钥列表）
	// 抽签阶段
	DrawProof *verifiable_draw.DrawProof // 抽签证明（包含抽签输出的公钥列表）
	// DPVSS阶段
	HolderPKLists           [][]*bls12381.PointG1 // 多个PVSS参与者公钥列表
	ShareCommitLists        [][]*bls12381.PointG1 // 多个PVSS份额承诺列表
	CoeffCommitLists        [][]*bls12381.PointG1 // 多个PVSS系数承诺列表
	EncShareLists           [][]*mrpvss.EncShare  // 多个PVSS加密份额列表
	CommitteePKList         []*bls12381.PointG1   // 多个PVSS对应的委员会公钥（公钥y = g^s，s为秘密）
	CommitteeWorkHeightList []uint64              // 每个PVSS所对应委员会的工作高度
}

func (s *CryptoElement) ToProto() *pb.CryptoElement {
	if s == nil {
		return &pb.CryptoElement{}
	}
	if s.SRS != nil {
		srsbyte, err := s.SRS.ToCompressedBytes()
		if err != nil {
			return nil
		}
		return &pb.CryptoElement{
			SRS:            srsbyte,
			SrsUpdateProof: s.SrsUpdateProof.ToBytes(),
		}
	} else {

		shuffleproof, err := s.ShuffleProof.ToBytes()
		if err != nil {
			return nil
		}

		drawproof, err := s.DrawProof.ToBytes()
		if err != nil {
			return nil
		}
		holderpkilist := make([]*pb.PointG1List, 0)
		for _, list := range s.HolderPKLists {

			g1s := make([]*pb.PointG1, 0)
			for _, g1 := range list {
				g1bytes := G1.ToCompressed(g1)
				pointg1 := &pb.PointG1{
					Pointbytes: g1bytes,
				}
				g1s = append(g1s, pointg1)
			}
			holderpkilist = append(holderpkilist, &pb.PointG1List{PointG1S: g1s})
		}

		sharecommitlist := make([]*pb.PointG1List, 0)
		for _, list := range s.ShareCommitLists {
			g1s := make([]*pb.PointG1, 0)
			for _, g1 := range list {
				g1bytes := G1.ToCompressed(g1)
				pointg1 := &pb.PointG1{
					Pointbytes: g1bytes,
				}
				g1s = append(g1s, pointg1)
			}
			sharecommitlist = append(sharecommitlist, &pb.PointG1List{PointG1S: g1s})
		}

		coeffcommitlist := make([]*pb.PointG1List, 0)
		for _, list := range s.CoeffCommitLists {
			g1s := make([]*pb.PointG1, 0)
			for _, g1 := range list {
				g1bytes := G1.ToCompressed(g1)
				pointg1 := &pb.PointG1{
					Pointbytes: g1bytes,
				}
				g1s = append(g1s, pointg1)
			}
			coeffcommitlist = append(coeffcommitlist, &pb.PointG1List{PointG1S: g1s})
		}
		encsharelist := make([]*pb.EncShareList, 0)
		for _, list := range s.EncShareLists {
			shares := make([]*pb.EncShare, 0)
			for _, share := range list {
				sharebyte := share.ToBytes()

				shares = append(shares, &pb.EncShare{
					Sharebytes: sharebyte,
				})
			}
			encsharelist = append(encsharelist, &pb.EncShareList{EncShares: shares})
		}

		committeePKList := make([]*pb.PointG1, 0)
		for _, g1 := range s.CommitteePKList {
			g1bytes := G1.ToCompressed(g1)
			committeePKList = append(committeePKList, &pb.PointG1{
				Pointbytes: g1bytes,
			})
		}

		return &pb.CryptoElement{
			//SRS:                     nil,
			HolderPKLists:    holderpkilist,
			ShareCommitLists: sharecommitlist,
			CoeffCommitLists: coeffcommitlist,
			EncSharesLists:   encsharelist,
			CommitteePKLists: committeePKList,
			//SrsUpdateProof:          nil,
			ShuffleProof:            shuffleproof,
			DrawProof:               drawproof,
			CommitteeWorkHeightList: s.CommitteeWorkHeightList,
		}
	}
}

func ToCryptoElement(element *pb.CryptoElement) (CryptoElement, error) {
	if element == nil {
		return CryptoElement{}, fmt.Errorf("element is nil")
	}
	if element.GetSRS() != nil {
		Srs, err := srs.FromCompressedBytes(element.GetSRS())
		if err != nil {
			return CryptoElement{}, err
		}
		srsproof := &schnorr_proof.SchnorrProof{}
		srsproofs, err := srsproof.FromBytes(element.GetSrsUpdateProof())
		if err != nil {
			return CryptoElement{}, err
		}
		return CryptoElement{
			SRS:            Srs,
			SrsUpdateProof: srsproofs,
		}, nil
	}

	holderpkilist := make([][]*bls12381.PointG1, 0)
	for _, list := range element.GetHolderPKLists() {
		g1s := make([]*bls12381.PointG1, 0)
		for _, g1 := range list.GetPointG1S() {
			point, err := G1.FromCompressed(g1.GetPointbytes())
			if err != nil {
				return CryptoElement{}, err
			}
			g1s = append(g1s, point)
		}
		holderpkilist = append(holderpkilist, g1s)
	}

	sharecommitlist := make([][]*bls12381.PointG1, 0)
	for _, list := range element.GetShareCommitLists() {
		g1s := make([]*bls12381.PointG1, 0)
		for _, g1 := range list.GetPointG1S() {
			point, err := G1.FromCompressed(g1.GetPointbytes())
			if err != nil {
				return CryptoElement{}, err
			}
			g1s = append(g1s, point)
		}
		sharecommitlist = append(sharecommitlist, g1s)
	}

	coeffcommitlist := make([][]*bls12381.PointG1, 0)
	for _, list := range element.GetCoeffCommitLists() {
		g1s := make([]*bls12381.PointG1, 0)
		for _, g1 := range list.GetPointG1S() {
			point, err := G1.FromCompressed(g1.GetPointbytes())
			if err != nil {
				return CryptoElement{}, err
			}
			g1s = append(g1s, point)
		}
		coeffcommitlist = append(coeffcommitlist, g1s)
	}
	encsharelist := make([][]*mrpvss.EncShare, 0)
	for _, list := range element.GetEncSharesLists() {
		shares := make([]*mrpvss.EncShare, 0)
		for _, share := range list.GetEncShares() {
			sharebytes := share.GetSharebytes()
			encshare, err := mrpvss.NewEmptyEncShare().FromBytes(sharebytes)
			if err != nil {
				return CryptoElement{}, err
			}
			shares = append(shares, encshare)
		}
		encsharelist = append(encsharelist, shares)
	}

	committeePKList := make([]*bls12381.PointG1, 0)
	for _, g1 := range element.GetCommitteePKLists() {
		point, err := G1.FromCompressed(g1.GetPointbytes())
		if err != nil {
			return CryptoElement{}, err
		}
		committeePKList = append(committeePKList, point)
	}
	var err error
	shuffleproof := &verifiable_draw.DrawProof{}
	if element.GetShuffleProof() != nil {
		shuffleproof, err = shuffleproof.FromBytes(element.GetShuffleProof())
		if err != nil {
			return CryptoElement{}, err
		}
	}
	drawproof := &verifiable_draw.DrawProof{}
	if element.GetDrawProof() != nil {
		drawproof, err = drawproof.FromBytes(element.GetDrawProof())
		if err != nil {
			return CryptoElement{}, err
		}
	}
	return CryptoElement{
		SRS:                     nil,
		SrsUpdateProof:          nil,
		ShuffleProof:            shuffleproof,
		DrawProof:               drawproof,
		HolderPKLists:           holderpkilist,
		ShareCommitLists:        sharecommitlist,
		CoeffCommitLists:        coeffcommitlist,
		EncShareLists:           encsharelist,
		CommitteePKList:         committeePKList,
		CommitteeWorkHeightList: element.GetCommitteeWorkHeightList(),
	}, nil
}
