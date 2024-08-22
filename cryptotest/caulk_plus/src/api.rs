use std::marker::PhantomData;
use std::ops::Add;
use ark_ec::{AffineCurve, PairingEngine, ProjectiveCurve};
use ark_poly::{EvaluationDomain, GeneralEvaluationDomain};
use ark_poly::univariate::DensePolynomial;
use ark_std::{UniformRand, Zero};
use crate::kzg::KZGCommit;
use crate::multi_proof::{CaulkPlusMulti, MultiProof};
use crate::precompute::PrecomputedInput;
use crate::single_proof::{CaulkPlusSingle, CommonInput, PublicInput, SingleProof, WitnessInput};
use crate::srs::SRS;
use crate::utils::{calc_lagrange_polynomials, calc_min_limited_degree, find_indices, next_power_of_two};

pub struct API<E: PairingEngine> {
    phantom: PhantomData<E>,
}

#[allow(non_snake_case)]
impl<E: PairingEngine> API<E> {
    pub fn create_multi_proof(
        // vector size
        parent_vec_size: u32,
        // vector in u32
        parent_vector: &[E::Fr],
        // sub-vector size
        sub_vec_size: u32,
        // sub-vector in u32
        sub_vector: &[E::Fr],
    ) -> Result<MultiProof<E>, String> {
        ////////////////////////////////////////////////////////////////////////////////////////////
        // padding n and m to next power of 2
        let m_padded = next_power_of_two(sub_vec_size);
        let mut n_padded = next_power_of_two(parent_vec_size);
        if m_padded - sub_vec_size > n_padded - parent_vec_size {
            n_padded <<= 1;
        }

        let mut padded_sub_vector = Vec::with_capacity(n_padded as usize);
        padded_sub_vector.extend(sub_vector.iter().map(|v| E::Fr::from(*v)));
        let mut padded_parent_vector = Vec::with_capacity(n_padded as usize);
        padded_parent_vector.extend(parent_vector.iter().map(|v| E::Fr::from(*v)));

        // padding vector with duplicate elements
        let dup_elem = padded_sub_vector[0];
        for _ in sub_vec_size..m_padded {
            padded_sub_vector.push(dup_elem);
            padded_parent_vector.push(dup_elem);
        }
        for _ in parent_vec_size + m_padded - sub_vec_size..n_padded {
            // TODO ensure dup elem is not in origin sub vector
            padded_parent_vector.push(E::Fr::zero());
        }

        // index of sub vector elem in parent vector, start at 1
        let I = match find_indices::<E>(&padded_parent_vector, &padded_sub_vector) {
            Ok(indices) => indices,
            Err(err) => {
                return Err(err)
            }
        };

        ////////////////////////////////////////////////////////////////////////////////////////////////
        // srs
        // check srs degree
        // g1: max(n, m^2+m+2)
        // g2: max(n-1, m^2+m+2)
        let (min_g1_power_degree, min_g2_power_degree) = calc_min_limited_degree(n_padded, m_padded);

        // read srs file
        let srs = match SRS::<E>::from_binary_file_with_degree(min_g1_power_degree, min_g2_power_degree) {
            Ok(srs) => srs,
            Err(err) => return Err(format!("The degree of SRS is to low: {}", err)),
        };

        // generate domain H and V
        let domain_H = GeneralEvaluationDomain::<E::Fr>::new(n_padded as usize).unwrap();
        let domain_V = GeneralEvaluationDomain::<E::Fr>::new(m_padded as usize).unwrap();

        ////////////////////////////////////////////////////////////////////////////////////////////////
        // calculate polynomials of parent vector and sub vector

        // calc parent_poly
        // c(x) = \sum^(n-1)_(i=0) [c_i * c_l_i(x)]
        let domain_h_lagrange_poly_set = calc_lagrange_polynomials::<E>(&domain_H);
        let mut parent_poly = DensePolynomial::zero();
        for i in 0..n_padded as usize {
            parent_poly = parent_poly + &domain_h_lagrange_poly_set[i] * padded_parent_vector[i];
        }

        // calc child_poly
        // a(x) = \sum^(m-1)_(i=0) [a_i * a_l_i(x)]
        let domain_v_lagrange_poly_set = calc_lagrange_polynomials::<E>(&domain_V);
        let mut sub_poly = DensePolynomial::zero();
        for i in 0..m_padded as usize {
            sub_poly = sub_poly + &domain_v_lagrange_poly_set[i] * padded_sub_vector[i];
        }

        // kzg commitment
        let parent_vec_commit = KZGCommit::<E>::commit_g1(&srs.g1_powers, &parent_poly);
        let sub_vec_commit = KZGCommit::<E>::commit_g1(&srs.g1_powers, &sub_poly);

        let common_input = crate::multi_proof::CommonInput {
            n: parent_vec_size,
            n_padded,
            m: sub_vec_size,
            m_padded,
            c_commit: parent_vec_commit,
            a_commit: sub_vec_commit,
        };

        let witness_input = crate::multi_proof::WitnessInput {
            I,
            a_elem_set: padded_sub_vector,
            parent_poly,
            child_poly: sub_poly,
        };

        let precomputed_input =
            PrecomputedInput::gen(&witness_input.I, &witness_input.parent_poly, &witness_input.a_elem_set, &domain_H, &srs);

        Ok(CaulkPlusMulti::prove(&srs, &common_input, &witness_input, &precomputed_input))
    }

    pub fn verify_multi_proof(
        proof: &MultiProof<E>,
    ) -> Result<bool, String> {
        ////////////////////////////////////////////////////////////////////////////////////////////////
        // TODO: check padded n and m to next power of 2

        ////////////////////////////////////////////////////////////////////////////////////////////////
        // srs
        // check srs degree
        // g1: max(n, m^2+m+2)
        // g2: max(n-1, m^2+m+2)
        let (min_g1_power_degree, min_g2_power_degree) = calc_min_limited_degree(proof.n_padded, proof.m_padded);

        // read srs file
        let srs = match SRS::<E>::from_binary_file_with_degree(min_g1_power_degree, min_g2_power_degree) {
            Ok(srs) => srs,
            Err(err) => return Err(format!("The degree of SRS is to low: {}", err)),
        };

        Ok(CaulkPlusMulti::verify(&srs, &proof))
    }

    pub fn create_single_proof(
        // independent g1 generator
        h_g1_generator: E::G1Affine,
        // vector size
        parent_vec_size: u32,
        // vector in u32
        parent_vector: &[E::Fr],
        // sub-vector in u32
        chosen_element: E::Fr,
    ) -> Result<SingleProof<E>, String> {
        ////////////////////////////////////////////////////////////////////////////////////////////////
        // padding n and m to next power of 2
        let n_padded = next_power_of_two(parent_vec_size);

        let mut padded_parent_vector = Vec::with_capacity(n_padded as usize);
        padded_parent_vector.extend(parent_vector.iter().map(|v| E::Fr::from(*v)));

        for _ in parent_vec_size..n_padded {
            // TODO ensure dup elem is not in origin sub vector
            padded_parent_vector.push(E::Fr::zero());
        }

        // index of chosen elem in parent vector, start at 1
        let mut index = parent_vec_size + 1;
        for i in 0..parent_vec_size {
            if chosen_element.eq(&parent_vector[i as usize]) {
                index = i+1;
                break;
            }
        }
        if index == parent_vec_size + 1 {
            return Err("The single element is not in parent vector.".to_string());
        }

        let rng = &mut rand::thread_rng();


        ////////////////////////////////////////////////////////////////////////////////////////////////
        // srs
        // check srs degree
        // g1: max(n, m^2+m+2)
        // g2: max(n-1, m^2+m+2)
        let (min_g1_power_degree, min_g2_power_degree) = calc_min_limited_degree(n_padded, 1);

        // read srs file
        let srs = match SRS::<E>::from_binary_file_with_degree(min_g1_power_degree, min_g2_power_degree) {
            Ok(srs) => srs,
            Err(err) => return Err(format!("The degree of SRS is to low: {}", err)),
        };

        // generate domain H
        let domain_H = GeneralEvaluationDomain::<E::Fr>::new(n_padded as usize).unwrap();

        ////////////////////////////////////////////////////////////////////////////////////////////////
        // calculate polynomials of parent vector and sub vector

        // calc parent_poly
        // c(x) = \sum^(n-1)_(i=0) [c_i * c_l_i(x)]
        let domain_h_lagrange_poly_set = calc_lagrange_polynomials::<E>(&domain_H);
        let mut parent_poly = DensePolynomial::zero();
        for i in 0..n_padded as usize {
            parent_poly = parent_poly + &domain_h_lagrange_poly_set[i] * padded_parent_vector[i];
        }

        // kzg commitment
        let parent_vec_commit = KZGCommit::<E>::commit_g1(&srs.g1_powers, &parent_poly);

        // pedersen commit
        // blind factor
        let r = E::Fr::rand(rng);
        let chosen_elem_commit = srs.g1_powers[0].mul(chosen_element).add(h_g1_generator.mul(r)).into_affine();

        let public_input = PublicInput {
            srs,
            h_g1_generator,
        };

        let common_input = CommonInput {
            n: parent_vec_size,
            n_padded,
            c_commit: parent_vec_commit,
            v_commit: chosen_elem_commit,
        };

        let witness_input = WitnessInput {
            v: chosen_element,
            r,
            i: index,
            c_poly: parent_poly,
        };

        Ok(CaulkPlusSingle::prove(&public_input, &common_input, &witness_input))
    }

    pub fn verify_single_proof(
        // independent g1 generator
        h_g1_generator: E::G1Affine,
        proof: &SingleProof<E>,
    ) -> Result<bool, String> {
        ////////////////////////////////////////////////////////////////////////////////////////////////
        // srs
        // check srs degree
        // g1: max(n, m^2+m+2)
        // g2: max(n-1, m^2+m+2)
        let (min_g1_power_degree, min_g2_power_degree) = calc_min_limited_degree(proof.n_padded, 1);

        // read srs file
        let srs = match SRS::<E>::from_binary_file_with_degree(min_g1_power_degree, min_g2_power_degree) {
            Ok(srs) => srs,
            Err(err) => return Err(format!("The degree of SRS is to low: {}", err)),
        };

        let public_input = PublicInput::<E> {
            srs,
            h_g1_generator,
        };
        Ok(CaulkPlusSingle::verify(&public_input, &proof))
    }
}