// This file includes an algorithm for calculating n openings of a KZG vector
// commitment of size n in n log(n) time. The algorithm is by Feist and
// khovratovich. It is useful for preprocessing.
// The full algorithm is described here https://github.com/khovratovich/Kate/blob/master/Kate_amortized.pdf
// 这个文件包含了一个计算KZG向量的n个打开的算法
// 在nlog (n)时间内完成大小为n的任务。算法由Feist和khovratovich设计。它对预处理很有用。
// 完整的算法描述在这里https://github.com/khovratovich/Kate/blob/master/Kate_amortized.pdf


use ark_ec::ProjectiveCurve;
use ark_ff::PrimeField;
use ark_poly::{
    univariate::DensePolynomial, EvaluationDomain, GeneralEvaluationDomain, UVPolynomial,
};
use std::vec::Vec;

// 使用DFT计算所有预证明
// h_i= c_d[x^{d-i-1}]+c_{d-1}[x^{d-i-2}]+c_{d-2}[x^{d-i-3}]+\cdots +
// c_{i+2}[x]+c_{i+1}[1]
pub fn compute_h<F, G>(
    c_poly: &DensePolynomial<F>, /* c(X) degree up to d<2^p , i.e. c_poly has at most d+1 coeffs
                                  * non-zero */
    powers: &[G], // SRS
    p: usize,
) -> Vec<G>
where
    F: PrimeField,
    G: ProjectiveCurve,
{
    let mut coeffs = c_poly.coeffs().to_vec();
    let dom_size = 1 << p;
    let fpzero = F::zero();
    coeffs.resize(dom_size, fpzero);

    // 1. x_ext = [[x^(d-1)], [x^{d-2},...,[x],[1], d+2 [0]'s]
    let mut x_ext: Vec<G> = powers.iter().take(dom_size - 1).rev().copied().collect();
    x_ext.resize(2 * dom_size, G::zero()); // 填充2d+2个中性元素 filling 2d+2 neutral elements
    let y = group_dft::<F, G>(&x_ext, p + 1);

    // 2. c_ext = [c_d, d zeroes, c_d,c_{0},c_1,...,c_{d-2},c_{d-1}]
    let mut c_ext = vec![coeffs[coeffs.len() - 1]];
    c_ext.resize(dom_size, fpzero);
    c_ext.push(coeffs[coeffs.len() - 1]);
    for &e in coeffs.iter().take(coeffs.len() - 1) {
        c_ext.push(e);
    }
    assert_eq!(c_ext.len(), 2 * dom_size);
    let v = field_dft::<F>(&c_ext, p + 1);

    // 3. u = y o v
    let u: Vec<_> = y
        .into_iter()
        .zip(v.into_iter())
        .map(|(a, b)| a.mul(b.into_repr()))
        .collect();

    // 4. h_ext = idft_{2d+2}(u)
    let h_ext = group_inv_dft::<F, G>(&u, p + 1);

    h_ext[0..dom_size].to_vec()
}

// compute DFT of size @dom_size over vector of Fr elements
// q_i = h_0 + h_1w^i + h_2w^{2i}+\cdots + h_{dom_size-1}w^{(dom_size-1)i} for
// 0<= i< dom_size=2^p
pub fn group_dft<F, G>(h: &[G], p: usize) -> Vec<G>
where
    F: PrimeField,
    G: ProjectiveCurve,
{
    let dom_size = 1 << p;
    assert_eq!(h.len(), dom_size); // we do not support inputs of size not power of 2
    let input_domain: GeneralEvaluationDomain<F> = EvaluationDomain::new(dom_size).unwrap();
    let mut l = dom_size / 2;
    let mut m: usize = 1;
    // Stockham FFT
    let mut xvec = h.to_vec();
    for _ in 0..p {
        let mut xt = xvec.clone();
        for j in 0..l {
            for k in 0..m {
                let c0 = xvec[k + j * m];
                let c1 = xvec[k + j * m + l * m];
                xt[k + 2 * j * m] = c0 + c1;
                let wj_2l = input_domain.element((j * dom_size / (2 * l)) % dom_size);
                xt[k + 2 * j * m + m] = (c0 - c1).mul(wj_2l.into_repr());
            }
        }
        l /= 2;
        m *= 2;
        xvec = xt;
    }
    xvec
}

// compute DFT of size @dom_size over vector of Fr elements
// q_i = h_0 + h_1w^i + h_2w^{2i}+\cdots + h_{dom_size-1}w^{(dom_size-1)i} for
// 0<= i< dom_size=2^p
pub fn field_dft<F: PrimeField>(h: &[F], p: usize) -> Vec<F> {
    let dom_size = 1 << p;
    assert_eq!(h.len(), dom_size); // we do not support inputs of size not power of 2
    let input_domain: GeneralEvaluationDomain<F> = EvaluationDomain::new(dom_size).unwrap();
    let mut l = dom_size / 2;
    let mut m: usize = 1;
    // Stockham FFT
    let mut xvec = h.to_vec();
    for _ in 0..p {
        let mut xt = xvec.clone();
        for j in 0..l {
            for k in 0..m {
                let c0 = xvec[k + j * m];
                let c1 = xvec[k + j * m + l * m];
                xt[k + 2 * j * m] = c0 + c1;
                let wj_2l = input_domain.element((j * dom_size / (2 * l)) % dom_size);
                xt[k + 2 * j * m + m] = (c0 - c1) * (wj_2l);
            }
        }
        l /= 2;
        m *= 2;
        xvec = xt;
    }
    xvec
}

// compute idft of size @dom_size over vector of G1 elements
// q_i = (h_0 + h_1w^-i + h_2w^{-2i}+\cdots +
// h_{dom_size-1}w^{-(dom_size-1)i})/dom_size for 0<= i< dom_size=2^p
pub fn group_inv_dft<F, G>(h: &[G], p: usize) -> Vec<G>
where
    F: PrimeField,
    G: ProjectiveCurve,
{
    let dom_size = 1 << p;
    assert_eq!(h.len(), dom_size); // we do not support inputs of size not power of 2
    let input_domain: GeneralEvaluationDomain<F> = EvaluationDomain::new(dom_size).unwrap();
    let mut l = dom_size / 2;
    let mut m: usize = 1;
    // Stockham FFT
    let mut xvec = h.to_vec();
    for _ in 0..p {
        let mut xt = xvec.clone();
        for j in 0..l {
            for k in 0..m {
                let c0 = xvec[k + j * m];
                let c1 = xvec[k + j * m + l * m];
                xt[k + 2 * j * m] = c0 + c1;
                let wj_2l = input_domain
                    .element((dom_size - (j * dom_size / (2 * l)) % dom_size) % dom_size);
                xt[k + 2 * j * m + m] = (c0 - c1).mul(wj_2l.into_repr()); // Difference #1 to forward DFT
            }
        }
        l /= 2;
        m *= 2;
        xvec = xt;
    }

    let domain_inverse = F::from(1u64 << p).inverse().unwrap().into_repr();
    let res = xvec.iter().map(|x| x.mul(domain_inverse)).collect();

    res
}

// compute idft of size @dom_size over vector of G1 elements
// q_i = (h_0 + h_1w^-i + h_2w^{-2i}+\cdots +
// h_{dom_size-1}w^{-(dom_size-1)i})/dom_size for 0<= i< dom_size=2^p
pub fn field_inv_dft<F: PrimeField>(h: &[F], p: usize) -> Vec<F> {
    let dom_size = 1 << p;

    assert_eq!(h.len(), dom_size); // we do not support inputs of size not power of 2
    let input_domain: GeneralEvaluationDomain<F> = EvaluationDomain::new(dom_size).unwrap();
    let mut l = dom_size / 2;
    let mut m: usize = 1;
    // Stockham FFT
    let mut xvec = h.to_vec();
    for _ in 0..p {
        let mut xt = xvec.clone();
        for j in 0..l {
            for k in 0..m {
                let c0 = xvec[k + j * m];
                let c1 = xvec[k + j * m + l * m];
                xt[k + 2 * j * m] = c0 + c1;
                let wj_2l = input_domain
                    .element((dom_size - (j * dom_size / (2 * l)) % dom_size) % dom_size);
                xt[k + 2 * j * m + m] = (c0 - c1) * wj_2l; // Difference #1 to
                                                           // forward DFT
            }
        }
        l /= 2;
        m *= 2;
        xvec = xt;
    }

    let domain_inverse = F::from(1u64 << p).inverse().unwrap();
    let res = xvec.iter().map(|&x| x * domain_inverse).collect();

    res
}

#[cfg(test)]
pub mod tests {
    use super::*;
    use ark_bls12_381::Bls12_381;
    use ark_ec::PairingEngine;
    use ark_std::{test_rng, UniformRand};

    #[test]
    fn test_dft() {
        test_dft_helper::<Bls12_381>();
    }

    fn test_dft_helper<E: PairingEngine>() {
        let mut rng = test_rng();
        for i in 2..6 {
            let size = 1 << i;

            let h: Vec<E::Fr> = (0..size).map(|_| E::Fr::rand(&mut rng)).collect();

            let c_dft = field_dft::<E::Fr>(&h, i);
            let c_back = field_inv_dft::<E::Fr>(&c_dft, i);
            assert_eq!(h, c_back);

            let h: Vec<E::G1Projective> =
                (0..size).map(|_| E::G1Projective::rand(&mut rng)).collect();

            let c_dft = group_dft::<E::Fr, E::G1Projective>(&h, i);
            let c_back = group_inv_dft::<E::Fr, E::G1Projective>(&c_dft, i);
            assert_eq!(h, c_back);

            let h: Vec<E::G2Projective> =
                (0..size).map(|_| E::G2Projective::rand(&mut rng)).collect();

            let c_dft = group_dft::<E::Fr, E::G2Projective>(&h, i);
            let c_back = group_inv_dft::<E::Fr, E::G2Projective>(&c_dft, i);
            assert_eq!(h, c_back);
        }
    }
}
