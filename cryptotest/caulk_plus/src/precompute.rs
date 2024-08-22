use ark_ec::PairingEngine;
use ark_poly::univariate::DensePolynomial;
use ark_poly::{EvaluationDomain, GeneralEvaluationDomain, UVPolynomial};
use ark_std::{One};
use crate::utils::{evaluate_polynomial_on_g2, gen_vanishing_polynomial};
use crate::srs::SRS;
#[allow(non_snake_case)]
pub struct PrecomputedInput<E: PairingEngine> {
    pub g2_W1_i_x_set: Vec<E::G2Affine>,
    pub g2_W2_i_x_set: Vec<E::G2Affine>,
}

#[allow(non_snake_case)]
impl<E: PairingEngine> PrecomputedInput<E> {
    pub fn gen(
        // 位置子集
        I: &Vec<u32>,
        // 承诺多项式
        c_poly: &DensePolynomial<E::Fr>,
        // C(X) = ∑_(i=1)^n [c_i · τ_i(X)]
        c_i_set: &[E::Fr],
        // ℍ = {1,𝜔,...,𝜔ⁿ⁻¹}
        domain_H: &GeneralEvaluationDomain<E::Fr>,
        // SRS: {[x]₁,...,[xᵈ⁻¹]₁,[x]₂,...,[xᵈ⁻¹]₂}
        srs: &SRS<E>,
    ) -> PrecomputedInput<E> {
        // 预计算输入：
        // - [W₁⁽ⁱ⁾(x)]₂ = [(C(x)-cᵢ)/(x-𝜔ⁱ)]₂, 对所有i∈I
        // - [W₂⁽ⁱ⁾(x)]₂ = [(Z_ℍ(x))/(x-𝜔ⁱ)]₂ = [(xⁿ-1)/(x-𝜔ⁱ)]₂, 对所有i∈I

        // [W₁⁽ⁱ⁾(x)]₂ = [(C(x)-cᵢ)/(x-𝜔ⁱ)], 对所有 i∈I
        let mut g2_W1_i_x_set = vec![];
        // [W₂⁽ⁱ⁾(x)]₂ = [(Z_ℍ(x))/(x-𝜔ⁱ)]₂ = [(xⁿ-1)/(x-𝜔ⁱ)]₂, 对所有i∈I
        let mut g2_W2_i_x_set = vec![];

        // 计算 Z_ℍ(X)
        let Z_H_poly = gen_vanishing_polynomial::<E>(domain_H.size() as u32);

        for i in 0..I.len() {
            // C(X) - cᵢ
            let c_i = DensePolynomial::from_coefficients_slice(&[c_i_set[i]]);
            let C_poly_minus_c_i = c_poly - &c_i;
            // X - 𝜔ⁱ
            let X_sub_omega_i =
                DensePolynomial::from_coefficients_slice(&[-domain_H.element((I[i] - 1) as usize), E::Fr::one()]); // index 0 == 1
            // W₁⁽ⁱ⁾(X) = (C(X)-cᵢ) / (X-𝜔ⁱ)
            let W_1_poly = &C_poly_minus_c_i / &X_sub_omega_i;
            // W₂⁽ⁱ⁾(X) = (Z_ℍ(X)) / (X-𝜔ⁱ) = (Xⁿ-1) / (X-𝜔ⁱ)
            let W_2_poly = &Z_H_poly / &X_sub_omega_i;
            // [W₁⁽ⁱ⁾(x)]₂
            g2_W1_i_x_set.push(evaluate_polynomial_on_g2::<E>(&W_1_poly, &srs));
            // [W₂⁽ⁱ⁾(x)]₂
            g2_W2_i_x_set.push(evaluate_polynomial_on_g2::<E>(&W_2_poly, &srs));
        }
        PrecomputedInput {
            g2_W1_i_x_set,
            g2_W2_i_x_set,
        }
    }
}
