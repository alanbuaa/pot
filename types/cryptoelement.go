package types

import (
	"blockchain-crypto/proof/schnorr_proof/bls12381"
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/verifiable_draw"
	"fmt"

	"blockchain-crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/pb"
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
	group1 := bls12381.NewG1()
	if s == nil {
		return &pb.CryptoElement{}
	}
	if s.SRS != nil {
		srsbyte, err := s.SRS.ToCompressedBytes()
		if err != nil {
			return nil
		}
		// TODO test
		_, testErr := srs.FromCompressedBytes(srsbyte)
		if testErr != nil {
			fmt.Printf("test FromCompressedBytes error:%v\n", err)
		}
		return &pb.CryptoElement{
			SRS:            srsbyte,
			SrsUpdateProof: s.SrsUpdateProof.ToBytes(),
		}
	} else {
		var shuffleproof []byte
		var err error
		var drawproof []byte
		var commiteeworkheightlist []uint64

		if s.ShuffleProof != nil {
			shuffleproof, err = s.ShuffleProof.ToBytes()
			if err != nil {
				return nil
			}
		}
		if s.DrawProof != nil {

			drawproof, err = s.DrawProof.ToBytes()
			if err != nil {
				return nil
			}
		}

		holderpkilist := make([]*pb.PointG1List, 0)
		for _, list := range s.HolderPKLists {

			g1s := make([]*pb.PointG1, 0)
			for _, g1 := range list {
				g1bytes := group1.ToCompressed(g1)
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
				g1bytes := group1.ToCompressed(g1)
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
				g1bytes := group1.ToCompressed(g1)
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
			g1bytes := group1.ToCompressed(g1)
			committeePKList = append(committeePKList, &pb.PointG1{
				Pointbytes: g1bytes,
			})
		}

		if s.CommitteeWorkHeightList != nil {
			commiteeworkheightlist = s.CommitteeWorkHeightList
		}

		return &pb.CryptoElement{
			// SRS:                     nil,
			HolderPKLists:    holderpkilist,
			ShareCommitLists: sharecommitlist,
			CoeffCommitLists: coeffcommitlist,
			EncSharesLists:   encsharelist,
			CommitteePKLists: committeePKList,
			// SrsUpdateProof:          nil,
			ShuffleProof:            shuffleproof,
			DrawProof:               drawproof,
			CommitteeWorkHeightList: commiteeworkheightlist,
		}
	}
}

func ToCryptoElement(element *pb.CryptoElement) (CryptoElement, error) {
	group1 := bls12381.NewG1()
	if element == nil {
		return CryptoElement{}, nil
	}
	if element.GetSRS() != nil {
		// fmt.Println(hexutil.Encode(element.GetSRS()))
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
			point, err := group1.FromCompressed(g1.GetPointbytes())
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
			point, err := group1.FromCompressed(g1.GetPointbytes())
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
			point, err := group1.FromCompressed(g1.GetPointbytes())
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
		point, err := group1.FromCompressed(g1.GetPointbytes())
		if err != nil {
			return CryptoElement{}, err
		}
		committeePKList = append(committeePKList, point)
	}
	var err error
	var shuffleproof *verifiable_draw.DrawProof = nil
	if element.GetShuffleProof() != nil {
		shuffleproof = &verifiable_draw.DrawProof{}
		shuffleproof, err = shuffleproof.FromBytes(element.GetShuffleProof())
		if err != nil {
			return CryptoElement{}, err
		}
	}
	var drawproof *verifiable_draw.DrawProof = nil
	if element.GetDrawProof() != nil {
		drawproof = &verifiable_draw.DrawProof{}
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
