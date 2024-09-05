use ark_bls12_381::{Fr, Fq, Fq2, G1Affine, G2Affine, Bls12_381};
use ark_ff::{FromBytes, PrimeField, ToBytes};
use prost::Message;
use caulk_plus::multi_proof::MultiProof;
use caulk_plus::single_proof::SingleProof;
use crate::bls12_381::{Fr as ProtoFr, Fq as ProtoFq, Fq2 as ProtoFq2, G1Affine as ProtoG1Affine,
                       G2Affine as ProtoG2Affine};
use crate::caulk_plus_grpc::{MultiProof as ProtoMultiProof, RootsOfUnity, SingleProof as ProtoSingleProof};
pub fn convert_proto_fr_to_fr(proto_fr: &ProtoFr) -> Fr {
    let proto_fr_vec = vec![proto_fr.e1, proto_fr.e2, proto_fr.e3, proto_fr.e4];
    Fr::from_le_bytes_mod_order(unsafe { proto_fr_vec.align_to::<u8>().1 })
}

pub fn convert_fr_to_proto_fr(fr: &Fr) -> ProtoFr {
    let mut fr_vec = Vec::with_capacity(32);
    fr.write(&mut fr_vec).unwrap();
    let proto_fr_vec = unsafe { fr_vec.align_to::<u64>().1 };
    ProtoFr {
        e1: proto_fr_vec[0],
        e2: proto_fr_vec[1],
        e3: proto_fr_vec[2],
        e4: proto_fr_vec[3],
    }
}

pub fn convert_proto_fq_to_fq(proto_fq: &ProtoFq) -> Fq {
    let proto_fq_vec = vec![proto_fq.e1, proto_fq.e2, proto_fq.e3, proto_fq.e4, proto_fq.e5, proto_fq.e6];
    Fq::from_le_bytes_mod_order(unsafe { proto_fq_vec.align_to::<u8>().1 })
}

pub fn convert_fq_to_proto_fq(fq: &Fq) -> ProtoFq {
    let mut fq_vec = Vec::with_capacity(48);
    fq.write(&mut fq_vec).unwrap();
    let proto_fr_vec = unsafe { fq_vec.align_to::<u64>().1 };
    ProtoFq {
        e1: proto_fr_vec[0],
        e2: proto_fr_vec[1],
        e3: proto_fr_vec[2],
        e4: proto_fr_vec[3],
        e5: proto_fr_vec[4],
        e6: proto_fr_vec[5],
    }
}

pub fn convert_proto_fq2_to_fq2(proto_fq2: &ProtoFq2) -> Fq2 {
    Fq2::new(convert_proto_fq_to_fq(&proto_fq2.c1.clone().unwrap()), convert_proto_fq_to_fq(&proto_fq2.c2.clone().unwrap()))
}

pub fn convert_fq2_to_proto_fq2(fq2: &Fq2) -> ProtoFq2 {
    let mut fq2_vec = Vec::with_capacity(96);
    fq2.write(&mut fq2_vec).unwrap();
    let proto_fq2_vec = unsafe { fq2_vec.align_to::<u64>().1 };
    ProtoFq2 {
        c1: ProtoFq {
            e1: proto_fq2_vec[0],
            e2: proto_fq2_vec[1],
            e3: proto_fq2_vec[2],
            e4: proto_fq2_vec[3],
            e5: proto_fq2_vec[4],
            e6: proto_fq2_vec[5],
        }.into(),
        c2: ProtoFq {
            e1: proto_fq2_vec[6],
            e2: proto_fq2_vec[7],
            e3: proto_fq2_vec[8],
            e4: proto_fq2_vec[9],
            e5: proto_fq2_vec[10],
            e6: proto_fq2_vec[11],
        }.into(),
    }
}

pub fn convert_g1_to_proto_g1(g1: &G1Affine) -> ProtoG1Affine {
    ProtoG1Affine {
        x: convert_fq_to_proto_fq(&g1.x).into(),
        y: convert_fq_to_proto_fq(&g1.y).into(),
        infinity: g1.infinity,
    }
}

pub fn convert_proto_g1_to_g1(proto_g1: &ProtoG1Affine) -> G1Affine {
    G1Affine::new(convert_proto_fq_to_fq(&proto_g1.x.clone().unwrap()), convert_proto_fq_to_fq(&proto_g1.y.clone().unwrap()), proto_g1.infinity)
}

pub fn convert_g2_to_proto_g2(g2: &G2Affine) -> ProtoG2Affine {
    ProtoG2Affine {
        x: convert_fq2_to_proto_fq2(&g2.x).into(),
        y: convert_fq2_to_proto_fq2(&g2.y).into(),
        infinity: g2.infinity,
    }
}

pub fn convert_proto_g2_to_g2(proto_g2: &ProtoG2Affine) -> G2Affine {
    G2Affine::new(convert_proto_fq2_to_fq2(&proto_g2.x.clone().unwrap()), convert_proto_fq2_to_fq2(&proto_g2.y.clone().unwrap()), proto_g2.infinity)
}

pub fn convert_proto_multi_proof_to_multi_proof(proto_multi_proof: &ProtoMultiProof) -> MultiProof<Bls12_381> {
    MultiProof {
        n: proto_multi_proof.n,
        n_padded: proto_multi_proof.n_padded,
        m: proto_multi_proof.m,
        m_padded: proto_multi_proof.m_padded,
        c_commit: convert_proto_g1_to_g1(&proto_multi_proof.c_commit.clone().unwrap()),
        a_commit: convert_proto_g1_to_g1(&proto_multi_proof.a_commit.clone().unwrap()),
        z_I: convert_proto_g1_to_g1(&proto_multi_proof.z_i.clone().unwrap()),
        c_I: convert_proto_g1_to_g1(&proto_multi_proof.c_i.clone().unwrap()),
        u: convert_proto_g1_to_g1(&proto_multi_proof.u.clone().unwrap()),
        w: convert_proto_g2_to_g2(&proto_multi_proof.w.clone().unwrap()),
        h: convert_proto_g1_to_g1(&proto_multi_proof.h.clone().unwrap()),
        v1: convert_proto_fr_to_fr(&proto_multi_proof.v1.clone().unwrap()),
        pi_1: convert_proto_g1_to_g1(&proto_multi_proof.pi_1.clone().unwrap()),
        v2: convert_proto_fr_to_fr(&proto_multi_proof.v2.clone().unwrap()),
        pi_2: convert_proto_g1_to_g1(&proto_multi_proof.pi_2.clone().unwrap()),
        pi_3: convert_proto_g1_to_g1(&proto_multi_proof.pi_3.clone().unwrap()),
    }
}

pub fn convert_multi_proof_to_proto_multi_proof(multi_proof: &MultiProof<Bls12_381>) -> ProtoMultiProof {
    ProtoMultiProof {
        n: multi_proof.n,
        n_padded: multi_proof.n_padded,
        m: multi_proof.m,
        m_padded: multi_proof.m_padded,
        c_commit: convert_g1_to_proto_g1(&multi_proof.c_commit).into(),
        a_commit: convert_g1_to_proto_g1(&multi_proof.a_commit).into(),
        z_i: convert_g1_to_proto_g1(&multi_proof.z_I).into(),
        c_i: convert_g1_to_proto_g1(&multi_proof.c_I).into(),
        u: convert_g1_to_proto_g1(&multi_proof.u).into(),
        w: convert_g2_to_proto_g2(&multi_proof.w).into(),
        h: convert_g1_to_proto_g1(&multi_proof.h).into(),
        v1: convert_fr_to_proto_fr(&multi_proof.v1).into(),
        pi_1: convert_g1_to_proto_g1(&multi_proof.pi_1).into(),
        v2: convert_fr_to_proto_fr(&multi_proof.v2).into(),
        pi_2: convert_g1_to_proto_g1(&multi_proof.pi_2).into(),
        pi_3: convert_g1_to_proto_g1(&multi_proof.pi_3).into(),
    }
}

pub fn convert_proto_single_proof_to_single_proof(proto_single_proof: &ProtoSingleProof) -> SingleProof<Bls12_381> {
    SingleProof {
        n: proto_single_proof.n,
        n_padded: proto_single_proof.n_padded,
        c_commit: convert_proto_g1_to_g1(&proto_single_proof.c_commit.clone().unwrap()).into(),
        v_commit: convert_proto_g1_to_g1(&proto_single_proof.v_commit.clone().unwrap()).into(),
        multi_proof: convert_proto_multi_proof_to_multi_proof(&proto_single_proof.multi_proof.clone().unwrap()),
        s_v: convert_proto_fr_to_fr(&proto_single_proof.s_v.clone().unwrap()),
        s_r: convert_proto_fr_to_fr(&proto_single_proof.s_r.clone().unwrap()),
        s_k: convert_proto_fr_to_fr(&proto_single_proof.s_k.clone().unwrap()),
        v_tilde: convert_proto_g1_to_g1(&proto_single_proof.v_tilde.clone().unwrap()).into(),
        a_commit: convert_proto_g1_to_g1(&proto_single_proof.a_commit.clone().unwrap()).into(),
        a_tilde: convert_proto_g1_to_g1(&proto_single_proof.a_tilde.clone().unwrap()).into(),
    }
}

pub fn convert_single_proof_to_proto_single_proof(single_proof: &SingleProof<Bls12_381>) -> ProtoSingleProof {
    ProtoSingleProof {
        n: single_proof.n,
        n_padded: single_proof.n_padded,
        c_commit: convert_g1_to_proto_g1(&single_proof.c_commit).into(),
        v_commit: convert_g1_to_proto_g1(&single_proof.v_commit).into(),
        multi_proof: convert_multi_proof_to_proto_multi_proof(&single_proof.multi_proof).into(),
        s_v: convert_fr_to_proto_fr(&single_proof.s_v).into(),
        s_r: convert_fr_to_proto_fr(&single_proof.s_r).into(),
        s_k: convert_fr_to_proto_fr(&single_proof.s_k).into(),
        v_tilde: convert_g1_to_proto_g1(&single_proof.v_tilde).into(),
        a_commit: convert_g1_to_proto_g1(&single_proof.a_commit).into(),
        a_tilde: convert_g1_to_proto_g1(&single_proof.a_tilde).into(),
    }
}

pub fn convert_roots_of_unity_to_proto_roots_of_unity(size: u32, roots_of_unity: Vec<Fr>) -> RootsOfUnity {
    RootsOfUnity {
        size,
        roots: (0..size as usize).map(|i| convert_fr_to_proto_fr(&roots_of_unity[i])).collect(),
    }
}

#[cfg(test)]
mod tests {
    use std::ops::Mul;
    use ark_bls12_381::{Bls12_381, Fq, Fq2, Fr, G1Affine, G2Affine};
    use ark_bls12_381::g1::{G1_GENERATOR_X, G1_GENERATOR_Y};
    use ark_bls12_381::g2::{G2_GENERATOR_X, G2_GENERATOR_Y};
    use ark_ec::{AffineCurve, ProjectiveCurve};
    use ark_ff::{PrimeField, ToBytes, UniformRand};
    use rand::{Rng, thread_rng};
    use caulk_plus::api::API;
    use crate::bls12_381::{Fr as ProtoFr, Fq as ProtoFq, Fq2 as ProtoFq2, G1Affine as ProtoG1Affine, G2Affine as ProtoG2Affine};
    use crate::utils::{convert_fq2_to_proto_fq2, convert_fq_to_proto_fq, convert_fr_to_proto_fr, convert_g1_to_proto_g1, convert_g2_to_proto_g2, convert_multi_proof_to_proto_multi_proof, convert_proto_fq2_to_fq2, convert_proto_fq_to_fq, convert_proto_fr_to_fr, convert_proto_g1_to_g1, convert_proto_g2_to_g2, convert_proto_multi_proof_to_multi_proof};

    #[test]
    fn test_convert_between_proto_fr_and_fr() {
        let fr123 = Fr::from(123);
        println!("fr: {}", fr123);
        println!("fr: {:?}", fr123);
        let convert_proto_fr = convert_fr_to_proto_fr(&fr123);
        println!("convert_proto_fr: {:?}", convert_proto_fr);
        let res_fr = convert_proto_fr_to_fr(&convert_proto_fr);
        println!("res_fr: {}\n", res_fr);
        assert!(fr123.eq(&res_fr));

        let proto_fr = ProtoFr { e1: 12995873202578071081, e2: 16055258479306406044, e3: 12580165684082946707, e4: 5686935703723219923 };
        println!("proto_fr: {:?}", proto_fr);
        let conv_fr = convert_proto_fr_to_fr(&proto_fr);
        println!("conv_fr: {:?}", conv_fr);
        println!("conv_fr: {}", conv_fr);
        let res_proto_fr = convert_fr_to_proto_fr(&conv_fr);
        println!("res_proto_fr: {:?}", res_proto_fr);
        assert!(proto_fr.eq(&res_proto_fr));
    }

    #[test]
    fn test_convert_between_proto_fq_and_fq() {
        let g1: G1Affine = G1Affine::new(G1_GENERATOR_X, G1_GENERATOR_Y, false);
        let g123 = g1.mul(123).into_affine();
        let fq = g123.x;
        println!("fq: {}", fq);
        println!("fq: {:?}", fq);
        let convert_proto_fq = convert_fq_to_proto_fq(&fq);
        println!("convert_proto_fq: {:?}", convert_proto_fq);
        let res_fq = convert_proto_fq_to_fq(&convert_proto_fq);
        println!("res_fq: {}\n", res_fq);
        assert!(fq.eq(&res_fq));

        let proto_fq = ProtoFq {
            e1: 9643578314753161704,
            e2: 12363969365937116593,
            e3: 17370378380101614273,
            e4: 10525188256555326244,
            e5: 625377410555126400,
            e6: 66496752359416402,
        };
        println!("proto_fq: {:?}", proto_fq);
        let convert_fq = convert_proto_fq_to_fq(&proto_fq);
        println!("convert_fq: {}", convert_fq);
        println!("convert_fq: {:?}", convert_fq);
        let res_proto_fq = convert_fq_to_proto_fq(&convert_fq);
        println!("res_proto_fq: {:?}", res_proto_fq);
        assert!(res_proto_fq.eq(&proto_fq));
    }

    #[test]
    fn test_convert_between_proto_fq2_and_fq2() {
        let g1: G2Affine = G2Affine::new(G2_GENERATOR_X, G2_GENERATOR_Y, false);
        let g123 = g1.mul(123).into_affine();
        let fq2 = g123.x;
        println!("fq2: {}", fq2);
        println!("fq2: {:?}", fq2);
        let convert_proto_fq2 = convert_fq2_to_proto_fq2(&fq2);
        println!("convert_proto_fq2: {:?}", convert_proto_fq2);
        let res_fq2 = convert_proto_fq2_to_fq2(&convert_proto_fq2);
        println!("res_fq2: {}", res_fq2);
        println!("res_fq2: {:?}\n", res_fq2);
        assert!(fq2.eq(&res_fq2));

        let proto_fq2 = ProtoFq2 {
            c1: ProtoFq {
                e1: 1159306823142176013,
                e2: 10447810949987217124,
                e3: 3083478465137856249,
                e4: 5899801630485792087,
                e5: 1445135372160517734,
                e6: 678458133818818382,
            }.into(),
            c2: ProtoFq {
                e1: 5594160341619071078,
                e2: 8955073851498511001,
                e3: 8812007552358678521,
                e4: 17602816475186738576,
                e5: 16890608467270166650,
                e6: 1576694991520513337,
            }.into(),
        };
        println!("proto_fq2: {:?}", proto_fq2);
        let convert_fq2 = convert_proto_fq2_to_fq2(&proto_fq2);
        println!("convert_fq2: {}", convert_fq2);
        println!("convert_fq2: {:?}", convert_fq2);
        let res_proto_fq2 = convert_fq2_to_proto_fq2(&convert_fq2);
        println!("res_proto_fq2: {:?}", res_proto_fq2);
        assert!(res_proto_fq2.eq(&proto_fq2));
    }

    #[test]
    fn test_convert_between_proto_g1_and_g1() {
        let rng = &mut thread_rng();
        let g1 = G1Affine::new(G1_GENERATOR_X, G1_GENERATOR_Y, false);
        println!("g1: {}", g1);
        let convert_proto_g1 = convert_g1_to_proto_g1(&g1);
        println!("convert_proto_g1: {:?}", convert_proto_g1);
        let res_g1 = convert_proto_g1_to_g1(&convert_proto_g1);
        println!("{}\n", res_g1);
        assert!(res_g1.eq(&g1));

        let proto_g1 = ProtoG1Affine {
            x: convert_fq_to_proto_fq(&G1_GENERATOR_X).into(),
            y: convert_fq_to_proto_fq(&G1_GENERATOR_Y).into(),
            infinity: false,
        };
        println!("proto_g1: {:?}", proto_g1);
        let convert_g1 = convert_proto_g1_to_g1(&proto_g1);
        println!("convert_g1: {}", convert_g1);
        let res_proto_g1 = &convert_g1_to_proto_g1(&convert_g1);
        println!("res_proto_g1: {:?}", res_proto_g1);
        assert!(res_proto_g1.eq(&proto_g1));
    }

    #[test]
    fn test_convert_between_proto_g2_and_g2() {
        let g2 = G2Affine::new(G2_GENERATOR_X, G2_GENERATOR_Y, false);
        let res_g2 = convert_proto_g2_to_g2(&convert_g2_to_proto_g2(&g2));
        assert!(res_g2.eq(&g2));

        let proto_g2 = ProtoG2Affine {
            x: convert_fq2_to_proto_fq2(&G2_GENERATOR_X).into(),
            y: convert_fq2_to_proto_fq2(&G2_GENERATOR_Y).into(),
            infinity: false,
        };
        let res_proto_g2 = &convert_g2_to_proto_g2(&convert_proto_g2_to_g2(&proto_g2));
        assert!(res_proto_g2.eq(&proto_g2));
    }

    #[test]
    fn test_convert_between_proto_multi_proof_and_multi_proof() {
        let rng = &mut thread_rng();
        let parent_vec_size = 16u32;
        let sub_vec_size = 8u32;

        let mut parent_vec = vec![];
        for _ in 0..parent_vec_size {
            parent_vec.push(Fr::rand(rng));
        }
        let I = vec![6u32, 5, 3, 1, 4, 7, 2, 8];
        let mut sub_vec = vec![];
        for i in 0..sub_vec_size {
            sub_vec.push(parent_vec[I[i as usize] as usize - 1]);
        }

        let proof = API::<Bls12_381>::create_multi_proof(16, &parent_vec, 8, &sub_vec).unwrap();
        println!("{:?}", proof);
        let proto_proof = convert_multi_proof_to_proto_multi_proof(&proof);
        println!("{:?}", proto_proof);
        let convert_proof = convert_proto_multi_proof_to_multi_proof(&proto_proof);
        println!("{:?}", convert_proof);
        let res = API::<Bls12_381>::verify_multi_proof(&convert_proof).unwrap();
        println!("res = {}", res)
    }
    #[test]
    fn test_g1_bytes() {
        let g1: G1Affine = G1Affine::new(G1_GENERATOR_X, G1_GENERATOR_Y, false);
        let g = g1.mul(123).into_affine();
        let mut buf: Vec<u8> = vec![];
        g.x.write(&mut buf).expect("TODO: panic message");
        println!("{:?}", g.x);
        println!("{:?}", buf);
        println!("{:?}\n", Fq::from_le_bytes_mod_order(&buf));
        println!("{:?}", convert_fq_to_proto_fq(&g.x));
    }
}

