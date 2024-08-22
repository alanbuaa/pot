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
    use ark_bls12_381::{Fq, Fq2, Fr, G1Affine, G2Affine};
    use ark_bls12_381::g1::{G1_GENERATOR_X, G1_GENERATOR_Y};
    use ark_bls12_381::g2::{G2_GENERATOR_X, G2_GENERATOR_Y};
    use ark_ff::{PrimeField, UniformRand};
    use rand::{Rng, thread_rng};
    use crate::bls12_381::{Fr as ProtoFr, Fq as ProtoFq, Fq2 as ProtoFq2, G1Affine as ProtoG1Affine, G2Affine as ProtoG2Affine};
    use crate::utils::{convert_fq2_to_proto_fq2, convert_fq_to_proto_fq, convert_fr_to_proto_fr, convert_g1_to_proto_g1, convert_g2_to_proto_g2, convert_proto_fq2_to_fq2, convert_proto_fq_to_fq, convert_proto_fr_to_fr, convert_proto_g1_to_g1, convert_proto_g2_to_g2};

    #[test]
    fn test_convert_between_proto_fr_and_fr() {
        let fr = Fr::from(1234567890123456789u64);
        println!("fr: {}", fr);
        let convert_proto_fr = convert_fr_to_proto_fr(&fr);
        println!("convert_proto_fr: {:?}", convert_proto_fr);
        let res_fr = convert_proto_fr_to_fr(&convert_proto_fr);
        println!("res_fr: {}\n", res_fr);
        assert!(fr.eq(&res_fr));

        let proto_fr = ProtoFr { e1: 1234567890123456789, e2: 0, e3: 0, e4: 0 };
        println!("proto_fr: {}", fr);
        let conv_fr = convert_proto_fr_to_fr(&proto_fr);
        println!("conv_fr: {}", conv_fr);
        let res_proto_fr = convert_fr_to_proto_fr(&conv_fr);
        println!("res_proto_fr: {:?}", res_proto_fr);
        assert!(proto_fr.eq(&res_proto_fr));
    }

    #[test]
    fn test_convert_between_proto_fq_and_fq() {
        let rng = &mut thread_rng();

        let fq = Fq::from(1234567890123456789u64);
        println!("fq: {}", fq);
        let convert_proto_fq = convert_fq_to_proto_fq(&fq);
        println!("convert_proto_fq: {:?}", convert_proto_fq);
        let res_fq = convert_proto_fq_to_fq(&convert_proto_fq);
        println!("res_fq: {}\n", res_fq);
        assert!(fq.eq(&res_fq));

        let proto_fq = ProtoFq {
            e1: 1234567890123456789,
            e2: 0,
            e3: 0,
            e4: 0,
            e5: 0,
            e6: 0,
        };
        println!("proto_fq: {:?}", proto_fq);
        let convert_fq = convert_proto_fq_to_fq(&proto_fq);
        println!("convert_fq: {}", convert_fq);
        let res_proto_fq = convert_fq_to_proto_fq(&convert_fq);
        println!("res_proto_fq: {:?}", res_proto_fq);
        assert!(res_proto_fq.eq(&proto_fq));
    }

    #[test]
    fn test_convert_between_proto_fq2_and_fq2() {
        let rng = &mut thread_rng();

        let fq2 = Fq2::rand(rng);
        println!("fq2: {}", fq2);
        let convert_proto_fq2 = convert_fq2_to_proto_fq2(&fq2);
        println!("convert_proto_fq2: {:?}", convert_proto_fq2);
        let res_fq2 = convert_proto_fq2_to_fq2(&convert_proto_fq2);
        println!("res_fq2: {}\n", res_fq2);
        assert!(fq2.eq(&res_fq2));

        let proto_fq2 = ProtoFq2 {
            c1: ProtoFq {
                e1: 15312334153293348280,
                e2: 841050694974028783,
                e3: 12993178926126977399,
                e4: 14331714969349929730,
                e5: 2740446039084699729,
                e6: 165123225776229009,
            }.into(),
            c2: ProtoFq {
                e1: 16549740192668593022,
                e2: 3696594454104530263,
                e3: 13103893525273989193,
                e4: 6443473286224459290,
                e5: 9055845637167730533,
                e6: 1432192374203850592,
            }.into(),
        };
        println!("proto_fq2: {:?}", proto_fq2);
        let convert_fq2 = convert_proto_fq2_to_fq2(&proto_fq2);
        println!("convert_fq2: {}", convert_fq2);
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

    // fn test_convert_between_proto_multi_proof_and_multi_proof() {
    //     let mut rng = rand::thread_rng();
    // 
    //     let parent_vector_size = 32u32;
    //     let mut parent_vector = Vec::with_capacity(parent_vector_size as usize);
    //     for _ in 0..parent_vector_size {
    //         parent_vector.push(ProtoFr {
    //             e1: rng.gen(),
    //             e2: rng.gen(),
    //             e3: rng.gen(),
    //             e4: rng.gen(),
    //         });
    //     }
    //     let sub_vector_size = 16u32;
    //     let mut sub_vector = Vec::with_capacity(sub_vector_size as usize);
    //     for i in 0..sub_vector_size {
    //         sub_vector.push(ProtoFr {
    //             e1: i as u64 + 1,
    //             e2: i as u64 * 100 + 1,
    //             e3: i as u64 * 10000 + 1,
    //             e4: i as u64 * 1000000 + 1,
    //         });
    //     }
    //     let I = vec![1usize, 3, 4, 7, 23, 8, 9, 10, 12, 17, 21, 22, 30, 24, 16, 26, 27, 29, 25, 31, 2, 11, 5, 6, 20, 14, 28, 13, 19, 18, 15];
    //     for i in 0..sub_vector_size as usize {
    //         parent_vector[I[i]] = sub_vector[i].clone()
    //     }
    //     for i in 0..parent_vector_size as usize {
    //         println!("parent [{}]: {:?}", i, parent_vector[i]);
    //     }
    //     for i in 0..sub_vector_size as usize {
    //         println!("sub_vc [{}]: {:?}", i, sub_vector[i]);
    //     }
    // 
    //     let request = API::<Bls12_381>::create_multi_proof(parent_vector_size)
    // 
    //     let g2 = G2Affine::new(G2_GENERATOR_X, G2_GENERATOR_Y, false);
    //     let res_g2 = convert_proto_g2_to_g2(&convert_g2_to_proto_g2(&g2));
    //     assert!(res_g2.eq(&g2));
    // 
    //     let proto_g2 = ProtoG2Affine {
    //         x: convert_fq2_to_proto_fq2(&G2_GENERATOR_X).into(),
    //         y: convert_fq2_to_proto_fq2(&G2_GENERATOR_Y).into(),
    //         infinity: false,
    //     };
    //     let res_proto_g2 = &convert_g2_to_proto_g2(&convert_proto_g2_to_g2(&proto_g2));
    //     assert!(res_proto_g2.eq(&proto_g2));
    // }
}