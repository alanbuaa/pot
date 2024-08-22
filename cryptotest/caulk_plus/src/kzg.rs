use crate::utils::convert_to_bigints;
use crate::dft::{compute_h, group_dft};

use ark_ec::{msm::VariableBaseMSM, AffineCurve, PairingEngine, ProjectiveCurve};
use ark_ff::{Field, PrimeField};
use ark_poly::{
    univariate::DensePolynomial, EvaluationDomain, GeneralEvaluationDomain, Polynomial,
    UVPolynomial,
};
use ark_std::{One, Zero};
use std::marker::PhantomData;

pub struct KZGCommit<E: PairingEngine> {
    phantom: PhantomData<E>,
}

impl<E: PairingEngine> KZGCommit<E> {
    ////////////////////////////////////////////////////////////////////////////////////////////////
    // 承诺
    ////////////////////////////////////////////////////////////////////////////////////////////////
    pub fn commit_g1(g1_powers: &[E::G1Affine], poly: &DensePolynomial<E::Fr>) -> E::G1Affine {
        let coeffs: Vec<<E::Fr as PrimeField>::BigInt> = poly.coeffs.iter().map(|&x| x.into_repr()).collect();

        VariableBaseMSM::multi_scalar_mul(g1_powers, &coeffs).into_affine()
    }


    pub fn commit_g2(g2_powers: &[E::G2Affine], poly: &DensePolynomial<E::Fr>) -> E::G2Affine {
        let coeffs: Vec<<E::Fr as PrimeField>::BigInt> = poly.coeffs.iter().map(|&x| x.into_repr()).collect();

        VariableBaseMSM::multi_scalar_mul(g2_powers, &coeffs).into_affine()
    }

    ////////////////////////////////////////////////////////////////////////////////////////////////
    // 打开
    ////////////////////////////////////////////////////////////////////////////////////////////////
    // compute all openings to c_poly using a smart formula
    // This Code implements an algorithm for calculating n openings of a KZG vector commitment of size n in n log(n) time.
    // The algorithm is by Feist and Khovratovich. It is useful for preprocessing.
    // The full algorithm is described here https://github.com/khovratovich/Kate/blob/master/Kate_amortized.pdf
    // 使用智能公式计算c_poly的所有开口
    // 此代码实现了计算KZG向量的n个打开的算法
    // 在 n log(n) 时间内完成大小为 n 的任务。算法由Feist和Khovratovich。
    // 它对预处理很有用。
    // 完整的算法描述在这里https://github.com/khovratovich/Kate/blob/master/Kate_amortized.pdf
    pub fn multiple_open<G>(
        c_poly: &DensePolynomial<E::Fr>, // c(X)
        powers: &[G],                    // SRS
        p: usize,
    ) -> Vec<G>
    where
        G: AffineCurve<ScalarField=E::Fr> + Sized,
    {
        let degree = c_poly.coeffs.len() - 1;
        let input_domain: GeneralEvaluationDomain<E::Fr> = EvaluationDomain::new(degree).unwrap();

        let powers: Vec<G::Projective> = powers.iter().map(|x| x.into_projective()).collect();
        let h2 = compute_h(c_poly, &powers, p);

        let dom_size = input_domain.size();
        assert_eq!(1 << p, dom_size);
        assert_eq!(degree + 1, dom_size);

        let q2 = group_dft::<E::Fr, G::Projective>(&h2, p);

        let res = G::Projective::batch_normalization_into_affine(q2.as_ref());

        res
    }

    // KZG.Open( srs_KZG, f(X), deg, (alpha1, alpha2, ..., alphan) ) -> ([f(alpha1), ..., f(alphan)], pi)
    // Algorithm described in Section 4.6.1, Multiple Openings
    pub fn open_g1_batch(
        powers_of_g: &Vec<E::G1Affine>,
        poly: &DensePolynomial<E::Fr>,
        max_deg: Option<&usize>,
        points: &[E::Fr],
    ) -> (Vec<E::Fr>, E::G1Affine) {
        let mut evals = Vec::new();
        let mut proofs = Vec::new();
        for p in points.iter() {
            let (eval, pi) = Self::open_g1_single(&powers_of_g, poly, max_deg, p);
            evals.push(eval);
            proofs.push(pi);
        }

        let mut res = E::G1Projective::zero(); // default value

        for j in 0..points.len() {
            let w_j = points[j];
            // 1. Computing coefficient [1/prod]
            let mut prod = E::Fr::one();
            for (k, p) in points.iter().enumerate() {
                if k != j {
                    prod *= w_j - p;
                }
            }
            // 2. Summation
            let q_add = proofs[j].mul(prod.inverse().unwrap()); //[1/prod]Q_{j}
            res += q_add;
        }

        (evals, res.into_affine())
    }

    // KZG.Open( srs_KZG, f(X), deg, alpha ) -> (f(alpha), pi)
    pub fn open_g1_single(
        g1_x_powers: &Vec<E::G1Affine>,
        poly: &DensePolynomial<E::Fr>,
        max_deg: Option<&usize>,
        point: &E::Fr,
    ) -> (E::Fr, E::G1Affine) {
        let eval = poly.evaluate(point);

        let global_max_deg = g1_x_powers.len();

        let mut d: usize = 0;
        if max_deg == None {
            d += global_max_deg;
        } else {
            d += max_deg.unwrap();
        }
        let divisor = DensePolynomial::from_coefficients_vec(vec![-*point, E::Fr::one()]);
        let witness_polynomial = poly / &divisor;

        assert!(g1_x_powers[(global_max_deg - d)..].len() >= witness_polynomial.len());
        let proof = VariableBaseMSM::multi_scalar_mul(
            &g1_x_powers[(global_max_deg - d)..],
            convert_to_bigints(&witness_polynomial.coeffs).as_slice(),
        ).into_affine();

        (eval, proof)
    }

    // KZG.Verify(srs_KZG, F, deg, (alpha1, alpha2, ..., alphan), (v1, ..., vn), pi)
    pub fn verify_g1(
        // Verify that @c_com is a commitment to C(X) such that C(x)=z
        powers_of_g1: &[E::G1Affine], // generator of G1
        powers_of_g2: &[E::G2Affine], // [1]_2, [x]_2, [x^2]_2, ...
        c_com: &E::G1Affine,          // commitment
        max_deg: Option<&usize>,      // max degree
        points: &[E::Fr],             // x such that eval = C(x)
        evals: &[E::Fr],              // evaluation
        pi: &E::G1Affine,             // proof
    ) -> bool {
        let pairing_inputs =
            Self::verify_g1_defer_pairing(powers_of_g1, powers_of_g2, c_com, max_deg, points, evals, pi);

        let prepared_pairing_inputs = vec![
            (
                E::G1Prepared::from(pairing_inputs[0].0.into_affine()),
                E::G2Prepared::from(pairing_inputs[0].1.into_affine()),
            ),
            (
                E::G1Prepared::from(pairing_inputs[1].0.into_affine()),
                E::G2Prepared::from(pairing_inputs[1].1.into_affine()),
            ),
        ];
        let res = E::product_of_pairings(prepared_pairing_inputs.iter()).is_one();

        res
    }

    /// 验证
    /// KZG.Verify( srs_KZG, F, deg, (alpha1, alpha2, ..., alphan), (v1, ..., vn), pi)
    // Algorithm described in Section 4.6.1, Multiple Openings
    pub fn verify_g1_defer_pairing(
        // Verify that @c_com is a commitment to C(X) such that C(x)=z
        powers_of_g1: &[E::G1Affine], // generator of G1
        powers_of_g2: &[E::G2Affine], // [1]_2, [x]_2, [x^2]_2, ...
        c_com: &E::G1Affine,          // 承诺
        max_deg: Option<&usize>,      // 最大次数
        points: &[E::Fr],             // 打开点, x 使得 eval = C(x)
        evals: &[E::Fr],              // 求值
        pi: &E::G1Affine,             // proof
    ) -> Vec<(E::G1Projective, E::G2Projective)> {

        // 插值基函数集合
        // tau_i(X) = lagrange_tau[i] = polynomial equal to 0 at point[j] for j!= i and 1  at points[i]

        let mut lagrange_tau = DensePolynomial::zero();
        let mut prod = DensePolynomial::from_coefficients_slice(&[E::Fr::one()]);
        let mut components = vec![];
        for &p in points.iter() {
            let poly = DensePolynomial::from_coefficients_slice(&[-p, E::Fr::one()]);
            prod = &prod * (&poly);
            components.push(poly);
        }

        for i in 0..points.len() {
            let mut temp = &prod / &components[i];
            let lagrange_scalar = temp.evaluate(&points[i]).inverse().unwrap() * evals[i];
            temp.coeffs.iter_mut().for_each(|x| *x *= lagrange_scalar);
            lagrange_tau = lagrange_tau + temp;
        }

        // commit to sum evals[i] tau_i(X)
        assert!(
            powers_of_g1.len() >= lagrange_tau.len(),
            "KZG verifier doesn't have enough g1 powers"
        );
        let g1_tau = VariableBaseMSM::multi_scalar_mul(
            &powers_of_g1[..lagrange_tau.len()],
            convert_to_bigints(&lagrange_tau.coeffs).as_slice(),
        );

        // 消失多项式
        let z_tau = prod;

        // commit to z_tau(X) in g2
        assert!(
            powers_of_g2.len() >= z_tau.len(),
            "KZG verifier doesn't have enough g2 powers"
        );
        let g2_z_tau = VariableBaseMSM::multi_scalar_mul(
            &powers_of_g2[..z_tau.len()],
            convert_to_bigints(&z_tau.coeffs).as_slice(),
        );

        let global_max_deg = powers_of_g1.len();

        let mut d: usize = 0;
        if max_deg == None {
            d += global_max_deg;
        } else {
            d += max_deg.unwrap();
        }

        let res = vec![
            (
                g1_tau - c_com.into_projective(),
                powers_of_g2[global_max_deg - d].into_projective(),
            ),
            (pi.into_projective(), g2_z_tau),
        ];

        res
    }
}

mod tests {
    use ark_bls12_381::{Fr, Bls12_381, G1Projective, G2Projective};
    use ark_ec::{AffineCurve, ProjectiveCurve};
    use ark_poly::univariate::DensePolynomial;
    use ark_poly::{Polynomial, UVPolynomial};
    use ark_std::{One, UniformRand, Zero};
    use rand::thread_rng;
    use crate::kzg::KZGCommit;
    use crate::srs::SRS;


    #[test]
    fn test_kzg_g1() {
        let degree = 4;
        let rng = &mut thread_rng();
        let x = Fr::rand(rng);
        let g1_generator = G1Projective::rand(rng).into_affine();
        let g2_generator = G2Projective::rand(rng).into_affine();
        let srs = SRS::<Bls12_381>::new(degree, degree, &x, &g1_generator, &g2_generator).unwrap();
        // x^3 + x - 2
        let c_poly = DensePolynomial::<Fr>::from_coefficients_slice(&[Fr::from(-2), Fr::one(), Fr::zero(), Fr::one()]);
        let commit = KZGCommit::<Bls12_381>::commit_g1(&srs.g1_powers, &c_poly);

        let opening_point = Fr::rand(rng);

        // KZG.Open( srs_KZG, f(X), deg, alpha ) -> (f(alpha), pi)
        let (eval, pi) = KZGCommit::<Bls12_381>::open_g1_single(&srs.g1_powers, &c_poly, None, &opening_point);
        assert_eq!(c_poly.evaluate(&opening_point), eval);
        let res = KZGCommit::<Bls12_381>::verify_g1(&srs.g1_powers, &srs.g2_powers, &commit, None, &[opening_point], &[eval], &pi);
        assert!(res);
    }

    #[test]
    fn test_kzg_commit() {
        let degree = 4;
        let rng = &mut thread_rng();
        let x = Fr::from(3);
        let g1_generator = G1Projective::rand(rng).into_affine();
        let g2_generator = G2Projective::rand(rng).into_affine();
        let srs = SRS::<Bls12_381>::new(degree, 2u32, &x, &g1_generator, &g2_generator).unwrap();

        // x^3 + x - 2
        let poly = DensePolynomial::<Fr>::from_coefficients_slice(&[Fr::from(-2), Fr::one(), Fr::zero(), Fr::one()]);
        let commit = KZGCommit::<Bls12_381>::commit_g1(&srs.g1_powers, &poly);

        let expected = g1_generator.mul(Fr::from(28)).into_affine();
        println!("{}", expected.eq(&commit))
    }
}

