use ark_ec::{AffineCurve, PairingEngine, ProjectiveCurve};
use ark_ff::{Field, FromBytes, PrimeField};
use ark_poly::{univariate::DensePolynomial, EvaluationDomain, GeneralEvaluationDomain};
use ark_poly_commit::{Polynomial, UVPolynomial};
use ark_std::{One, UniformRand, Zero};
use ark_ff::bytes::ToBytes;
use std::io::{Read, Result, Write};
use std::marker::PhantomData;
use std::ops::{Add, Mul, Sub};
use ark_serialize::CanonicalSerialize;
use blake2::{Blake2b, Digest};
use byteorder::{BigEndian, ReadBytesExt};
use crate::kzg::KZGCommit;
use crate::srs::SRS;
use crate::precompute::PrecomputedInput;
use crate::utils::{calc_lagrange_polynomials, calc_lagrange_polynomials_on_subset, calc_vanishing_polynomial, evaluate_polynomial_on_g1, evaluate_polynomial_on_g2, gen_vanishing_polynomial, pairing_check, poly_composition};

pub struct CaulkPlusMulti<E: PairingEngine> {
    phantom: PhantomData<E>,
}

#[allow(non_snake_case)]
pub struct CommonInput<E: PairingEngine> {
    // common input
    // size of domain H (origin)
    pub n: u32,
    // size of domain H (padded)
    pub n_padded: u32,
    // size of domain V (origin)
    pub m: u32,
    // size of domain V (padded)
    pub m_padded: u32,
    // KZG commitment of parent vector 𝒄 = [C[x]]₁
    pub c_commit: E::G1Affine,
    // KZG commitment of sub vector 𝒂 = [A(x)]₁
    pub a_commit: E::G1Affine,
}

impl<E: PairingEngine> ToBytes for CommonInput<E> {
    fn write<W: Write>(&self, mut writer: W) -> Result<()> {
        self.n.serialize(&mut writer).expect("n serialize fail");
        self.n_padded.serialize(&mut writer).expect("n_padded serialize fail");
        self.m.serialize(&mut writer).expect("m serialize fail");
        self.m_padded.serialize(&mut writer).expect("m_padded serialize fail");
        self.c_commit.write(&mut writer)?;
        self.a_commit.write(&mut writer)?;
        Ok(())
    }
}


#[allow(non_snake_case)]
impl<E: PairingEngine> FromBytes for CommonInput<E> {
    fn read<R: Read>(mut reader: R) -> Result<Self> {
        let n = reader.read_u32::<BigEndian>().expect("deserialize n fail");
        let n_padded = reader.read_u32::<BigEndian>().expect("deserialize n_padded fail");
        let m = reader.read_u32::<BigEndian>().expect("deserialize m fail");
        let m_padded = reader.read_u32::<BigEndian>().expect("deserialize m_padded fail");
        let c_commit = E::G1Affine::read(&mut reader).expect("deserialize c commit fail");
        let a_commit = E::G1Affine::read(&mut reader).expect("deserialize a commit fail");
        Ok(CommonInput {
            n,
            n_padded,
            m,
            m_padded,
            c_commit,
            a_commit,
        })
    }
}

#[allow(non_snake_case)]
pub struct WitnessInput<E: PairingEngine> {
    // I
    pub I: Vec<u32>,
    // {cᵢ} = C(ω^(i-1))
    pub a_elem_set: Vec<E::Fr>,
    // C(X)
    pub parent_poly: DensePolynomial<E::Fr>,
    // A(X)
    pub child_poly: DensePolynomial<E::Fr>,
}

#[allow(non_snake_case)]
pub struct MultiProof<E: PairingEngine> {
    // common input
    // size of domain H (origin)
    pub n: u32,
    // size of domain H (padded)
    pub n_padded: u32,
    // size of domain V (origin)
    pub m: u32,
    // size of domain V (padded)
    pub m_padded: u32,
    // KZG承诺 𝒄 = [C[x]]₁
    pub c_commit: E::G1Affine,
    // KZG承诺 𝒂 = [A(x)]₁
    pub a_commit: E::G1Affine,
    ////////////////////////////////////////////
    // 公开z_I = [Z_I'(x)]₁, c_I = [C_I'(x)]₁, 𝒖 = [U'(x)]₁
    pub z_I: E::G1Affine,
    pub c_I: E::G1Affine,
    pub u: E::G1Affine,
    // 公开 w = r₁^(-1)[W₁(x)+𝒳₂W₂(x)]₂-[r₂+r₃x+r₄x²]₂, 𝒉 = [H(x)]₁
    pub w: E::G2Affine,
    pub h: E::G1Affine,
    // 输出 v₁, v₂, π₁, π₂, π_3
    // (v₁, π₁) ← KZG.Open(U'(X), 𝛼)
    pub v1: E::Fr,
    pub pi_1: E::G1Affine,
    // (v₂, π₂) ← KZG.Open(P₁(X), v₁)
    pub v2: E::Fr,
    pub pi_2: E::G1Affine,
    // (0, π₃) ← KZG.Open(P₂(X), 𝛼)
    pub pi_3: E::G1Affine,
}

impl<E: PairingEngine> ToBytes for MultiProof<E> {
    fn write<W: Write>(&self, mut writer: W) -> Result<()> {
        self.n.serialize(&mut writer).expect("n serialize fail");
        self.n_padded.serialize(&mut writer).expect("n_padded serialize fail");
        self.m.serialize(&mut writer).expect("m serialize fail");
        self.m_padded.serialize(&mut writer).expect("m_padded serialize fail");
        self.c_commit.write(&mut writer)?;
        self.a_commit.write(&mut writer)?;
        self.z_I.write(&mut writer).expect("z_I serialize fail");
        self.c_I.write(&mut writer).expect("c_I serialize fail");
        self.u.write(&mut writer).expect("u serialize fail");
        self.w.write(&mut writer).expect("w serialize fail");
        self.h.write(&mut writer).expect("h serialize fail");
        self.v1.write(&mut writer).expect("v_1 serialize fail");
        self.pi_1.write(&mut writer).expect("pi_1 serialize fail");
        self.v2.write(&mut writer).expect("v_2 serialize fail");
        self.pi_2.write(&mut writer).expect("pi_2 serialize fail");
        self.pi_3.write(&mut writer).expect("pi_3 serialize fail");
        Ok(())
    }
}

#[allow(non_snake_case)]
impl<E: PairingEngine> FromBytes for MultiProof<E> {
    fn read<R: Read>(mut reader: R) -> Result<Self> {
        let n = reader.read_u32::<BigEndian>().expect("deserialize n fail");
        let n_padded = reader.read_u32::<BigEndian>().expect("deserialize n_padded fail");
        let m = reader.read_u32::<BigEndian>().expect("deserialize m fail");
        let m_padded = reader.read_u32::<BigEndian>().expect("deserialize m_padded fail");
        let c_commit = E::G1Affine::read(&mut reader).expect("deserialize c commit fail");
        let a_commit = E::G1Affine::read(&mut reader).expect("deserialize a commit fail");
        let z_I = E::G1Affine::read(&mut reader).expect("read z_I fail");
        let c_I = E::G1Affine::read(&mut reader).expect("read c_I fail");
        let u = E::G1Affine::read(&mut reader).expect("read u fail");
        let w = E::G2Affine::read(&mut reader).expect("read w fail");
        let h = E::G1Affine::read(&mut reader).expect("read h fail");
        let v1 = E::Fr::read(&mut reader).expect("read v1 fail");
        let pi_1 = E::G1Affine::read(&mut reader).expect("read pi_1 fail");
        let v2 = E::Fr::read(&mut reader).expect("read v2 fail");
        let pi_2 = E::G1Affine::read(&mut reader).expect("read pi_2 fail");
        let pi_3 = E::G1Affine::read(&mut reader).expect("read pi_3 fail");
        Ok(MultiProof {
            n,
            n_padded,
            m,
            m_padded,
            c_commit,
            a_commit,
            z_I,
            c_I,
            u,
            w,
            h,
            v1,
            pi_1,
            v2,
            pi_2,
            pi_3,
        })
    }
}

#[allow(non_snake_case)]
impl<E: PairingEngine> CaulkPlusMulti<E> {
    pub fn prove(
        // 公共输入
        srs: &SRS<E>,
        // 共同输入
        common_input: &CommonInput<E>,
        // 见证输入
        witness_input: &WitnessInput<E>,
        // 预计算输入
        precomputed_input: &PrecomputedInput<E>,
    ) -> MultiProof<E> {
        let mut rng = rand::thread_rng();

        let g1_x_powers = &srs.g1_powers;

        // 共同输入
        let domain_H = GeneralEvaluationDomain::new(common_input.n_padded as usize).unwrap();
        // check m_padded == I.size()
        let m_padded = common_input.m_padded;
        let domain_V = GeneralEvaluationDomain::new(m_padded as usize).unwrap();

        // 预计算输入
        let g2_W1_i_x_set = &precomputed_input.g2_W1_i_x_set;
        let g2_W2_i_x_set = &precomputed_input.g2_W2_i_x_set;

        // 见证输入
        let I = &witness_input.I;
        let I_len = I.len() as u32;
        let c_i_set = &witness_input.a_elem_set;
        let A_poly = &witness_input.child_poly;

        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 1 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 1.1 随机生成 r₁, r₂, r₃, r₄, r₅, r₆
        let r_1 = E::Fr::rand(&mut rng);
        let r_2 = E::Fr::rand(&mut rng);
        let r_3 = E::Fr::rand(&mut rng);
        let r_4 = E::Fr::rand(&mut rng);
        let r_5 = E::Fr::rand(&mut rng);
        let r_6 = E::Fr::rand(&mut rng);
        let r_1_inverse = r_1.inverse().unwrap();

        // 1.2 计算{ω_j}_(j∈I)上拉格朗日插值基函数{τᵢ(X)}_(i∈[1,m_padded])
        let tao_i_poly_set = calc_lagrange_polynomials_on_subset::<E>(&I, &domain_H);

        // 1.3 计算Z_I'(X) = r₁ · ∏_(i∈I)(X-𝜔ⁱ)
        let Z_I_blind_poly = &calc_vanishing_polynomial::<E>(&I, &domain_H) * r_1;


        // 1.4 计算C_I(X) = ∑_(i∈I) [cᵢ·τᵢ(X)]
        let mut C_I_poly = DensePolynomial::zero();
        for i in 0..I_len {
            C_I_poly = &tao_i_poly_set[i as usize] * c_i_set[i as usize] + C_I_poly;
        }

        // 1.5 计算C_I'(X) = C_I(X) + (r₂+r₃X+r₄X²) · Z_I'(X)
        let r2_add_r3_X_add_r4_X2 = DensePolynomial::from_coefficients_slice(&[r_2, r_3, r_4]);
        let C_I_blind_poly = &C_I_poly + &(&r2_add_r3_X_add_r4_X2 * &Z_I_blind_poly);

        // 1.6 计算 U(X) = ∑_{i=1,...,m_padded} [𝜔ᵢ · 𝜇ᵢ(X)] = ∑_{i=1,...,m_padded} [𝜔ᵘ⁽ⁱ⁾⁻¹ · 𝜇ᵢ(X)]
        // - 计算𝕍上拉格朗日插值基函数{𝜇ᵢ(X)}_(i∈[1,m_padded])
        let mut domain_V_positions = vec![];
        for i in 1..m_padded + 1 {
            domain_V_positions.push(i);
        }
        let miu_i_poly_set = calc_lagrange_polynomials::<E>(&domain_V);
        let mut U_poly = DensePolynomial::zero();
        for i in 1..m_padded + 1 {
            let index = I[i as usize - 1] - 1; // index 0 = pos 1
            U_poly = &U_poly + &(&miu_i_poly_set[i as usize - 1] * domain_H.element(index as usize));
        }

        // 1.7 计算U'(X) = U(X) + (r_5 + r_6 · X) · (Xᵐ - 1)
        let Z_V_poly = gen_vanishing_polynomial::<E>(m_padded);
        let U_blind_poly = &U_poly
            + &(&DensePolynomial::from_coefficients_slice(&[r_5, r_6]) * &Z_V_poly);

        // 1.8 生成 z_I c_I 𝒖
        // - 计算z_I = [Z_I'(x)]₁
        let z_I = evaluate_polynomial_on_g1(&Z_I_blind_poly, &srs);
        // - 计算c_I = [C_I'(x)]₁
        let c_I = evaluate_polynomial_on_g1(&C_I_blind_poly, &srs);
        // - 计算 𝒖 = [U'(x)]₁
        let u = evaluate_polynomial_on_g1(&U_blind_poly, &srs);


        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 2 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 2.0 Fiat-Shamir变换生成挑战 chi_1, chi_2 (𝜒₁, 𝜒₂)
        let (chi_1, chi_2) = gen_challenge_chi(common_input, z_I, c_I, u);

        // 2.1 计算[W₁(x) + 𝒳₂W₂(x)]₂ = ∑_(i∈I) [([W₁⁽ⁱ⁾(x)]₂+𝒳₂[W₂⁽ⁱ⁾(x)]₂) / (∏_(j∈I,i≠j)(𝜔^i-𝜔^j)]
        let mut g2_W1_x_add_chi2_plus_W2_x = E::G2Affine::zero();
        for i in 0..m_padded {
            // [W₁⁽ⁱ⁾(x)]₂+𝒳₂[W₂⁽ⁱ⁾(x)]₂
            let numerator = g2_W1_i_x_set[i as usize] + g2_W2_i_x_set[i as usize].mul(chi_2).into_affine();
            // ∏_(j∈I,i≠j) (𝜔^i-𝜔^j)
            let mut denominator = E::Fr::one();
            for j in 0..m_padded {
                if j != i {
                    denominator = denominator * &(domain_H.element(I[i as usize] as usize - 1) - &domain_H.element(I[j as usize] as usize - 1)); // index 0 == pos 1
                }
            }
            g2_W1_x_add_chi2_plus_W2_x = g2_W1_x_add_chi2_plus_W2_x + numerator.mul(denominator.inverse().unwrap()).into_affine();
        }

        // 2.2 计算 H(X) = [Z_I'(U'(X)) + 𝒳₁(C_I'(U'(X)) - A(X))] / Z_𝕍(X)
        let term1 = poly_composition::<E>(&U_blind_poly, &Z_I_blind_poly);
        let term2 = &poly_composition::<E>(&U_blind_poly, &C_I_blind_poly).sub(A_poly) * chi_1;
        let H_poly = &(&term1 + &term2) / &Z_V_poly;

        // 2.3 生成 𝒘 𝒉
        // - 计算 𝒘 = r₁⁻¹ · [W₁(x)+ 𝜒₂ W₂(x)]₂ - [r₂ + r₃ x + r₄ x²]₂
        let w: E::G2Affine = g2_W1_x_add_chi2_plus_W2_x.mul(r_1_inverse).into_affine() + (-evaluate_polynomial_on_g2::<E>(&r2_add_r3_X_add_r4_X2, &srs));
        // - 计算 𝒉 = [H(x)]₁
        let h = evaluate_polynomial_on_g1(&H_poly, &srs);


        ////////////////////////////////////////////////////////////////////////////////////////////
        // 第 3 轮
        ////////////////////////////////////////////////////////////////////////////////////////////
        // 3.0 Fiat-Shamir变换生成挑战 alpha (α)
        let alpha = gen_challenge_alpha(common_input, chi_1, chi_2, w, h);

        // 3.1 计算 P₁(X) = Z_I'(X) + 𝒳₁ * C_I'(X)
        let P1_poly = &Z_I_blind_poly + &(&C_I_blind_poly * chi_1);

        // 3.2 计算 P₂(X) = Z_I'(U'(𝛼)) + 𝒳₁(C_I'(U'(𝛼)) - A(X)) - Z_𝕍(𝛼)H(X) = Z_I'(U'(𝛼)) + 𝒳₁ · C_I'(U'(𝛼)) - 𝒳₁ · A(X) - Z_𝕍(𝛼)H(X)
        // Z_I'(U'(𝛼)) + 𝒳₁ · C_I'(U'(𝛼))
        let constant = poly_composition::<E>(&U_blind_poly, &Z_I_blind_poly).evaluate(&alpha)
            + poly_composition::<E>(&U_blind_poly, &C_I_blind_poly).evaluate(&alpha).mul(chi_1);
        let constant_item_poly: DensePolynomial<E::Fr> = DensePolynomial::from_coefficients_slice(&[constant]);
        let P2_poly = &(&constant_item_poly - &(A_poly * chi_1)) - &(&H_poly * Z_V_poly.evaluate(&alpha));

        // 3.3 计算 (v₁, π₁) ← KZG.Open(U'(X), 𝛼)
        let (v1, pi_1) = KZGCommit::<E>::open_g1_single(g1_x_powers, &U_blind_poly, None, &alpha);

        // 3.4 计算 (v₂, π₂) ← KZG.Open(P₁(X), 𝑣₁)
        let (v2, pi_2) = KZGCommit::<E>::open_g1_single(g1_x_powers, &P1_poly, None, &v1);

        // 3.5 计算 (0, π₃) ← KZG.Open(P₂(X), 𝛼)
        let (_, pi_3) = KZGCommit::<E>::open_g1_single(g1_x_powers, &P2_poly, None, &alpha);

        // 3.6 输出 z_I, c_I, u, w, h, v_1, v_2, π_1, π_2, π_3
        MultiProof {
            n: common_input.n,
            n_padded: common_input.n_padded,
            m: common_input.m,
            m_padded,
            c_commit: common_input.c_commit,
            a_commit: common_input.a_commit,
            z_I,
            c_I,
            u,
            w,
            h,
            v1,
            pi_1,
            v2,
            pi_2,
            pi_3,
        }
    }

    pub fn verify(
        // SRS
        srs: &SRS<E>,
        // proof (contains common input)
        proof: &MultiProof<E>,
    ) -> bool {
        let g1_x_powers = &srs.g1_powers;
        let g2_x_powers = &srs.g2_powers;
        let g1_generator = g1_x_powers[0];
        let g2_generator = g2_x_powers[0];
        // 共同输入
        let n_padded = proof.n_padded;
        let m_padded = proof.m_padded;
        let a_commit = proof.a_commit;
        let c_commit = proof.c_commit;
        // multi proof

        let z_I = proof.z_I;
        let c_I = proof.c_I;
        let u = proof.u;
        let w = proof.w;
        let h = proof.h;
        let v1 = proof.v1;
        let v2 = proof.v2;
        let pi_1 = proof.pi_1;
        let pi_2 = proof.pi_2;
        let pi_3 = proof.pi_3;
        let common_input = CommonInput {
            n: proof.n,
            n_padded,
            m: proof.m,
            m_padded,
            a_commit,
            c_commit,
        };

        // 挑战 χ_1, χ_2, α
        // TODO calc challenge 
        let (chi_1, chi_2) = gen_challenge_chi::<E>(&common_input, z_I, c_I, u);
        let alpha = gen_challenge_alpha::<E>(&common_input, chi_1, chi_2, w, h);

        // 1. 计算𝒑₁ = 𝒛_I + 𝜒₁ · c_I
        let p_1 = z_I.add(c_I.mul(chi_1).into_affine());

        // 2. 计算𝒑₂ = [v₂]₁ - 𝜒₁ · 𝒂 - Z_𝕍(α) · 𝒉
        let Z_V_alpha = gen_vanishing_polynomial::<E>(m_padded).evaluate(&alpha);
        let p_2 = g1_generator.mul(v2).into_affine() + (-a_commit.mul(chi_1).into_affine()) + (-h.mul(Z_V_alpha).into_affine());

        // 3. 验证 1 ← KZG.Verify(u, 𝛼, v₁, π₁)
        if !KZGCommit::<E>::verify_g1(g1_x_powers, g2_x_powers, &u, None, &[alpha], &[v1], &pi_1) {
            println!("1");
            return false;
        }

        // 4. 验证 1 ← KZG.Verify(𝒑₁, v₁, v₂, π₂)
        if !KZGCommit::<E>::verify_g1(g1_x_powers, g2_x_powers, &p_1, None, &[v1], &[v2], &pi_2) {
            println!("2");
            return false;
        }

        // 5. 验证 1 ← KZG.Verify(𝒑₂, 𝛼, 0, π_3 )
        if !KZGCommit::<E>::verify_g1(g1_x_powers, g2_x_powers, &p_2, None, &[alpha], &[E::Fr::zero()], &pi_3) {
            println!("3");
            return false;
        }

        // 6. 验证 e(𝒄 - 𝒄_I + 𝜒₂[xⁿ - 1]₁, [1]₂) = e(z_I,w)
        let g1_x_pow_n_minus_1 = g1_x_powers[n_padded as usize].add(-g1_generator);
        let left_g1 = c_commit.add(-c_I).add(g1_x_pow_n_minus_1.mul(chi_2).into_affine());
        pairing_check::<E>(left_g1, g2_generator, z_I, w)
    }
}

#[allow(non_snake_case)]
fn gen_challenge_chi<E: PairingEngine>(
    common_input: &CommonInput<E>,
    z_I: <E as PairingEngine>::G1Affine,
    c_I: <E as PairingEngine>::G1Affine,
    u: <E as PairingEngine>::G1Affine,
) -> (E::Fr, E::Fr) {
    let mut bytes = vec![];
    common_input.write(&mut bytes).unwrap();
    z_I.write(&mut bytes).unwrap();
    c_I.write(&mut bytes).unwrap();
    u.write(&mut bytes).unwrap();
    let mut hasher1 = Blake2b::new();
    let mut hasher2 = Blake2b::new();
    hasher1.update(&bytes);
    hasher1.update([1u8]);
    let chi_1_bytes = hasher1.finalize().to_vec();
    hasher2.update(&bytes);
    hasher2.update([2u8]);
    let chi_2_bytes = hasher2.finalize().to_vec();
    (E::Fr::from_le_bytes_mod_order(&chi_1_bytes), E::Fr::from_le_bytes_mod_order(&chi_2_bytes))
}

fn gen_challenge_alpha<E: PairingEngine>(
    common_input: &CommonInput<E>,
    chi_1: E::Fr,
    chi_2: E::Fr,
    w: E::G2Affine,
    h: E::G1Affine,
) -> E::Fr {
    let mut bytes = vec![];
    common_input.write(&mut bytes).unwrap();
    chi_1.write(&mut bytes).unwrap();
    chi_2.write(&mut bytes).unwrap();
    w.write(&mut bytes).unwrap();
    h.write(&mut bytes).unwrap();
    let mut hasher = Blake2b::new();
    hasher.update(&bytes);
    hasher.update([3u8]);
    let alpha_bytes = hasher.finalize().to_vec();
    E::Fr::from_le_bytes_mod_order(&alpha_bytes)
}