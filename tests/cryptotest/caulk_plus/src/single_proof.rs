use std::collections::HashMap;
use std::io::Write;
use std::marker::PhantomData;
use ark_ec::{AffineCurve, PairingEngine, ProjectiveCurve};
use ark_poly::{univariate::DensePolynomial, EvaluationDomain, GeneralEvaluationDomain, UVPolynomial};
use ark_ff::{PrimeField, ToBytes};
use ark_serialize::CanonicalSerialize;
use ark_std::{UniformRand};
use blake2::{Blake2b, Digest};
use rand::{RngCore, thread_rng};
use crate::multi_proof;
use crate::multi_proof::{CaulkPlusMulti, MultiProof,
};
use crate::precompute::PrecomputedInput;
use crate::srs::{SRS};

pub struct CaulkPlusSingle<E: PairingEngine> {
    phantom: PhantomData<E>,
}

pub struct PublicInput<E: PairingEngine> {
    // [1]₁,[x]₁,...,[xᵈ⁻¹]₁,[1]₂,[x]₂,...,[xᵈ⁻¹]₂
    pub srs: SRS<E>,
    // 独立 𝔾₁ 生成元 𝒉 = h[1]₁
    pub h_g1_generator: E::G1Affine,
}

impl<E: PairingEngine> PublicInput<E> {
    pub fn new<R: RngCore>(
        degree: u32,
        rng: &mut R,
        h: E::Fr,
    ) -> Result<PublicInput<E>, String> {
        // 随机选取 x
        let x = E::Fr::rand(rng);
        // 随机选取 g1
        let g1_generator = E::G1Projective::rand(rng).into();
        // 随机选取 g2
        let g2_generator = E::G2Projective::rand(rng).into();

        Ok(PublicInput {
            srs: SRS::<E>::new(degree, degree, &x, &g1_generator, &g2_generator).unwrap(),
            // 独立 𝔾₁ 生成元 𝒉 = h[1]₁
            h_g1_generator: g1_generator.mul(h).into_affine(),
        })
    }
}


#[allow(non_snake_case)]
pub struct CommonInput<E: PairingEngine> {
    // size of domain H (origin)
    pub n: u32,
    // size of domain H (padded)
    pub n_padded: u32,
    // KZG commitment 𝒄 = C[x]₁
    pub c_commit: E::G1Affine,
    // Pedersen commitment 𝒗 = [v]₁ + r𝒉
    pub v_commit: E::G1Affine,
}

impl<E: PairingEngine> ToBytes for CommonInput<E> {
    fn write<W: Write>(&self, mut writer: W) -> std::io::Result<()> {
        self.n.serialize(&mut writer).expect("n serialize fail");
        self.n_padded.serialize(&mut writer).expect("n_padded serialize fail");
        self.c_commit.write(&mut writer)?;
        self.v_commit.write(&mut writer)?;
        Ok(())
    }
}

pub struct WitnessInput<E: PairingEngine> {
    // v, 𝒗 = [v]₁ + r·𝒉
    pub v: E::Fr,
    // blind factor r, 𝒗 = [v]₁ + r·𝒉
    pub r: E::Fr,
    // 位置 i
    pub i: u32,
    // 多项式 C(X)
    pub c_poly: DensePolynomial<E::Fr>,
}

#[allow(non_snake_case)]
pub struct SingleProof<E: PairingEngine> {
    // common input
    // size of domain H (origin)
    pub n: u32,
    // size of domain H (padded)
    pub n_padded: u32,
    // KZG commitment 𝒄 = C[x]₁
    pub c_commit: E::G1Affine,
    // Pedersen commitment 𝒗 = [v]₁ + r𝒉
    pub v_commit: E::G1Affine,
    // R_Link^KZG证明
    pub multi_proof: MultiProof<E>,
    // sᵥ
    pub s_v: E::Fr,
    // sᵣ
    pub s_r: E::Fr,
    // sₖ
    pub s_k: E::Fr,
    // v ̃
    pub v_tilde: E::G1Affine,
    // 𝒂
    pub a_commit: E::G1Affine,
    // ã
    pub a_tilde: E::G1Affine,
}

#[allow(non_snake_case)]
impl<E: PairingEngine> CaulkPlusSingle<E> {
    #[allow(non_snake_case)]
    pub fn prove(
        // 公共输入
        public_input: &PublicInput<E>,
        // 共同输入
        common_input: &CommonInput<E>,
        // 见证输入
        witness_input: &WitnessInput<E>,
    ) -> SingleProof<E> {
        // 公共输入
        let srs = &public_input.srs;
        let g1_generator = srs.g1_powers[0];
        let h_g1_generator = public_input.h_g1_generator;
        // 共同输入
        let n_padded = common_input.n_padded;
        let domain_H = GeneralEvaluationDomain::new(n_padded as usize).unwrap();
        // 见证输入
        let c_poly = &witness_input.c_poly;
        let i = witness_input.i;
        let v = witness_input.v;

        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 1 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 1.1 随机抽样盲化因子k, v ̂, r ̂, k ̂ ← F
        let (k,
            v_hat,
            r_hat,
            k_hat
        ) = CaulkPlusSingle::<E>::gen_blind_factors();

        // 1.2 计算 𝒂 = [v]₁ + k[x-1]₁
        let g1_v: E::G1Affine = g1_generator.mul(v).into_affine();
        let g1_x_sub_1: E::G1Affine = srs.g1_powers[1] + (-g1_generator);
        let a_commit: E::G1Affine = g1_v + g1_x_sub_1.mul(k).into_affine();

        // 1.3 证明者以 𝒄, 𝒂, A(X) = v + k(X-1), 𝕍 = {1}, I = {i} 生成R_Link^KZG证明
        let a_poly = DensePolynomial::from_coefficients_slice(&[v - k, k]);
        let I = vec![i];
        let a_elem_set = vec![v];

        // a_i -> c_j, i = 1,2,...,m
        let mut u: HashMap<u32, u32> = HashMap::new();
        u.insert(1, i);
        let multi_common_input = multi_proof::CommonInput {
            n: common_input.n,
            n_padded,
            m: 1,
            m_padded: 1,
            c_commit: common_input.c_commit,
            a_commit,
        };

        let precomputed_input = PrecomputedInput::gen(&I, c_poly, &a_elem_set, &domain_H, &srs);
        let mut u: HashMap<u32, u32> = HashMap::new();
        u.insert(1, i);
        let multi_witness_input = multi_proof::WitnessInput {
            I,
            a_elem_set,
            parent_poly: c_poly.clone(),
            child_poly: a_poly,
        };
        let multi_proof = CaulkPlusMulti::prove(&srs, &multi_common_input, &multi_witness_input, &precomputed_input);

        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 2 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 2.1 输出 𝒗 𝒂 ̃
        // - 计算 𝒗 ̃ = [v ̂]₁ + r ̂𝒉
        let g1_v_hat: E::G1Affine = g1_generator.mul(v_hat).into_affine();
        let v_tilde: E::G1Affine = g1_v_hat + h_g1_generator.mul(r_hat).into_affine();
        // - 计算 𝒂 ̃ = [v ̂]₁ + k ̂[x-1]₁
        let a_tilde: E::G1Affine = g1_v_hat + g1_x_sub_1.mul(k_hat).into_affine();


        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 3 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 3.0 生成挑战 𝒳 = Hash(CommonInput,R_Link^KZG证明,𝒂, 𝒗 ̃, 𝒂 ̃)
        let chi = CaulkPlusSingle::<E>::gen_challenge_chi(common_input, &multi_proof, &a_commit, &v_tilde, &a_tilde);

        // 3.1 输出 R_Link^KZG 的证明, 𝒂, 𝒂 ̃, 𝒗 ̃, sᵥ, sᵣ, sₖ, 𝒳
        // - 计算s_v = v ̂+ 𝒳v
        let s_v = v_hat + chi * v;
        // - 计算s_r = r ̂+ 𝒳r
        let s_r = r_hat + chi * witness_input.r;
        // - 计算s_k = k ̂+ 𝒳k
        let s_k = k_hat + chi * k;

        SingleProof {
            n: common_input.n,
            n_padded,
            c_commit: common_input.c_commit,
            v_commit: common_input.v_commit,
            multi_proof,
            s_v,
            s_r,
            s_k,
            v_tilde,
            a_commit,
            a_tilde,
        }
    }

    pub fn verify(
        public_input: &PublicInput<E>,
        // single proof with common proof
        single_proof: &SingleProof<E>,
    ) -> bool {
        // 公共输入
        let g1_generator = public_input.srs.g1_powers[0];

        // 1. 验证R_Link^KZG证明
        if !CaulkPlusMulti::verify(&public_input.srs, &single_proof.multi_proof) {
            return false;
        }
        // 生成挑战 𝒳 = Hash(CommonInput,R_Link^KZG证明,𝒂, 𝒗 ̃, 𝒂 ̃)
        let common_input = CommonInput {
            n: single_proof.n,
            n_padded: single_proof.n_padded,
            c_commit: single_proof.c_commit,
            v_commit: single_proof.v_commit,
        };
        let chi = CaulkPlusSingle::<E>::gen_challenge_chi(&common_input, &single_proof.multi_proof, &single_proof.a_commit, &single_proof.v_tilde, &single_proof.a_tilde);

        // 2. 验证 [sᵥ]₁ + sᵣ𝒉 = 𝒗 ̃+ 𝒳𝒗
        let g1_s_v: E::G1Affine = g1_generator.mul(single_proof.s_v).into_affine();
        let eq1_left_hand: E::G1Affine = g1_s_v + public_input.h_g1_generator.mul(single_proof.s_r).into_affine();
        let eq2_right_hand: E::G1Affine = single_proof.v_tilde + single_proof.v_commit.mul(chi).into_affine();
        if !eq1_left_hand.eq(&eq2_right_hand) {
            return false;
        }
        // 3. 验证 [sᵥ]₁ + sₖ[x-1]₁ = 𝒂 ̃+ 𝒳𝒂
        let g1_x_sub_1 = public_input.srs.g1_powers[1] + (-g1_generator);
        let eq2_left_hand = g1_s_v + g1_x_sub_1.mul(single_proof.s_k).into_affine();
        let eq2_right_hand = single_proof.a_tilde + single_proof.a_commit.mul(chi).into_affine();
        if !eq2_left_hand.eq(&eq2_right_hand) {
            return false;
        }
        true
    }

    fn gen_blind_factors() -> (E::Fr, E::Fr, E::Fr, E::Fr) {
        let rng = &mut thread_rng();
        (E::Fr::rand(rng), E::Fr::rand(rng), E::Fr::rand(rng), E::Fr::rand(rng))
    }

    fn gen_challenge_chi(
        common_input: &CommonInput<E>,
        proof: &MultiProof<E>,
        a_commit: &E::G1Affine,
        v_tilde: &E::G1Affine,
        a_tilde: &E::G1Affine,
    ) -> E::Fr {
        let mut bytes = vec![];
        common_input.write(&mut bytes).unwrap();
        proof.write(&mut bytes).unwrap();
        a_commit.write(&mut bytes).unwrap();
        v_tilde.write(&mut bytes).unwrap();
        a_tilde.write(&mut bytes).unwrap();
        let mut hasher = Blake2b::new();
        hasher.update(&bytes);
        hasher.update([3u8]);
        let alpha_bytes = hasher.finalize().to_vec();
        E::Fr::from_le_bytes_mod_order(&alpha_bytes)
    }
}

