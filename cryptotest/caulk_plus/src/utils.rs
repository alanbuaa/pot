use std::collections::HashMap;
use ark_ec::{AffineCurve, PairingEngine, ProjectiveCurve};
use ark_ff::PrimeField;
use ark_poly::{EvaluationDomain, GeneralEvaluationDomain, UVPolynomial};
use ark_poly::univariate::DensePolynomial;
use ark_std::{One, Zero};
use std::ops::Div;
use crate::srs::SRS;

/// 计算 C(X) = U(P(X))
pub(crate) fn poly_composition<E: PairingEngine>(inner_poly: &DensePolynomial<E::Fr>, outer_poly: &DensePolynomial<E::Fr>)
                                                 -> DensePolynomial<E::Fr> {
    // C(X) = U(P(X)), 初始为 0
    let mut result = DensePolynomial::<E::Fr>::zero();
    for i in 0..outer_poly.coeffs.len() {
        if outer_poly.coeffs[i].is_zero() {
            // 当前位置U的系数是0，直接跳过
            continue;
        }
        // 当前位置系数不为0，计算 coeffs * P(X) ^ i
        result = result + &pow_of_poly::<E>(i as u32, inner_poly) * outer_poly.coeffs[i];
    }
    result
}


/// 计算多项式的幂，R(X)=(P(X))^x
pub(crate) fn pow_of_poly<E: PairingEngine>(x: u32, poly: &DensePolynomial<E::Fr>) -> DensePolynomial<E::Fr> {
    if x == 0 {
        DensePolynomial::<E::Fr>::from_coefficients_slice(&[E::Fr::one()])
    } else {
        let mut result = poly.clone();
        for _ in 0..x - 1 {
            result = &result * poly
        }
        result
    }
}


/// 在G1上使用生成元g1和srs对多项式求值
pub fn evaluate_polynomial_on_g1<E: PairingEngine>(
    poly: &DensePolynomial<E::Fr>,
    srs: &SRS<E>,
) -> E::G1Affine {
    // poly(X) = a_0·X^0 + a_1·X^1 + ... + a_n·X^n
    // g^poly(x) = [g^(X^0)]^a_0 · [g^(X^1)]^a_1 · ... · [g^(X^n)]^a_n
    let mut res = E::G1Affine::zero();
    for i in 0..poly.len() {
        // [g^(X^i)]^a_i
        res = res + srs.g1_powers[i].mul(poly[i]).into_affine();
    }
    res
}


/// 在G2上使用生成元g2和srs对多项式求值
pub fn evaluate_polynomial_on_g2<E: PairingEngine>(
    poly: &DensePolynomial<E::Fr>,
    srs: &SRS<E>,
) -> E::G2Affine {
    // poly(X) = a_0·X^i0 + a_1·X^i1 + ... + a_n·X^in
    // g^poly(x) = [g^(X^i0)]^a_0 · [g^(X^i1)]^a_1 · ... · [g^(X^in)]^a_n
    let mut res = E::G2Affine::zero();
    for i in 0..poly.len() {
        // [g^(X^i)]^a_i
        res = res + srs.g2_powers[i].mul(poly[i]).into_affine();
    }
    res
}


/// 生成消失多项式
pub fn gen_vanishing_polynomial<E: PairingEngine>(degree: u32) -> DensePolynomial<E::Fr> {
    // Z_V(X) = X^m - 1
    let mut coeff_vec: Vec<E::Fr> = Vec::new();
    coeff_vec.push(-E::Fr::one());
    for _ in 0..degree - 1 {
        coeff_vec.push(E::Fr::zero());
    }
    coeff_vec.push(E::Fr::one());
    return DensePolynomial::from_coefficients_vec(coeff_vec);
}


/// 计算子集的消失多项式
pub fn calc_vanishing_polynomial<E: PairingEngine>(pos: &[u32], domain: &GeneralEvaluationDomain<E::Fr>) -> DensePolynomial<E::Fr> {
    // Z_V(X) = X^m - 1
    if pos.len() == domain.size() {
        return gen_vanishing_polynomial::<E>(pos.len() as u32);
    }

    // Z_I(X) = ∏_(i∈I) (X-𝜔^i)
    let mut vanishing_poly = DensePolynomial::from_coefficients_slice(&[E::Fr::one()]);
    let len = pos.len();
    for i in 0..len {
        // (X-𝜔^i)
        // index 0 = pos 1
        let multiplier = DensePolynomial::from_coefficients_slice(&[-domain.element(pos[i] as usize - 1), E::Fr::one()]);
        vanishing_poly = &vanishing_poly * &multiplier;
    }
    return vanishing_poly;
}

/// 计算拉格朗日插值基函数集合 {lᵢ(X)}, i∈[1,m] ,pos从1开始，对应于index 0
pub fn calc_lagrange_polynomials<E: PairingEngine>(
    domain: &GeneralEvaluationDomain<E::Fr>,
) -> Vec<DensePolynomial<E::Fr>> {
    // 计算 {lᵢ(X)}, i ∈ {1, 2, ..., n}
    // lᵢ(X) = ∏ (X - 𝜔ⁱ⁻¹) / (X - 𝜔ⁱ⁻¹) / ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
    // p(X) = ∏ (X - 𝜔ⁱ⁻¹)
    // q = ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
    // lᵢ(X) = p(X) / (X - 𝜔ⁱ⁻¹) / q
    let size = domain.size();
    let mut p_poly = DensePolynomial::from_coefficients_slice(&[E::Fr::one()]);
    for i in 0..size {
        p_poly = &p_poly * &DensePolynomial::from_coefficients_slice(&[-domain.element(i), E::Fr::one()]);
    }
    let mut lagrange_poly_set = vec![];
    for i in 0..size {
        // lᵢ(X) = p(X) / (X - 𝜔ⁱ⁻¹) / q
        let mut l_i_poly = &p_poly / &DensePolynomial::from_coefficients_slice(&[-domain.element(i), E::Fr::one()]);
        // q = ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
        let mut q = E::Fr::one();
        for j in 0..size {
            if j != i {
                q = q.div(domain.element(i) - domain.element(j));
            }
        }
        l_i_poly = &l_i_poly * q;
        lagrange_poly_set.push(l_i_poly);
    }
    lagrange_poly_set
}


/// 计算拉格朗日插值基函数集合 {lᵢ(X)}, i∈[1,m] ,pos从1开始，对应于index 0
pub fn calc_lagrange_polynomials_on_subset<E: PairingEngine>(
    pos: &[u32],
    domain: &GeneralEvaluationDomain<E::Fr>,
) -> Vec<DensePolynomial<E::Fr>> {
    // 计算 {lᵢ(X)}, 1 <= i <= m
    // lᵢ(X) = ∏ (X - 𝜔ⁱ⁻¹) / (X - 𝜔ⁱ⁻¹) / ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
    // p(X) = ∏ (X - 𝜔ⁱ⁻¹)
    // q = ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
    // lᵢ(X) = p(X) / (X - 𝜔ⁱ⁻¹) / q
    let m = pos.len();
    let mut p_poly = DensePolynomial::from_coefficients_slice(&[E::Fr::one()]);
    for i in 0..m {
        p_poly = &p_poly * &DensePolynomial::from_coefficients_slice(&[-domain.element(pos[i] as usize - 1), E::Fr::one()]);
    }
    let mut lagrange_poly_set = vec![];
    for i in 0..m {
        // lᵢ(X) = p(X) / (X - 𝜔ⁱ⁻¹) / q
        let mut l_i_poly = &p_poly / &DensePolynomial::from_coefficients_slice(&[-domain.element(pos[i] as usize - 1), E::Fr::one()]);
        // q = ∏_(j=1,j≠i)^(n) (𝜔ⁱ⁻¹ - 𝜔ʲ⁻¹)
        let mut q = E::Fr::one();
        for j in 0..m {
            if j != i {
                q = q.div(domain.element(pos[i] as usize - 1) - domain.element(pos[j] as usize - 1));
            }
        }
        l_i_poly = &l_i_poly * q;
        lagrange_poly_set.push(l_i_poly);
    }
    lagrange_poly_set
}

pub fn pairing_check<E: PairingEngine>(
    left_g1: E::G1Affine,
    left_g2: E::G2Affine,
    right_g1: E::G1Affine,
    right_g2: E::G2Affine,
) -> bool {
    let left_res = E::pairing(left_g1, left_g2);
    let right_res = E::pairing(right_g1, right_g2);
    left_res.eq(&right_res)
}

// for trim srs
pub fn calc_min_limited_degree(n: u32, m: u32) -> (u32, u32) {
    // g1: max(n, m^2+m+2)
    // g2: max(n-1, m^2+m+2)
    let m_limit = (m + 1).pow(2) + 1;
    return if m_limit >= n {
        (m_limit, m_limit)
    } else {
        (n, n - 1)
    }
}

// auto find mapping between parent vector and sub vector, 
// return sub vector elements indices in parent vector (start at 1)
pub fn find_indices<E: PairingEngine>(parent: &[E::Fr], child: &[E::Fr]) -> Result<Vec<u32>, String> {
    // 创建一个哈希映射表用于存储父向量中每个元素的索引
    let mut parent_index_map = HashMap::new();
    for (index, &value) in parent.iter().enumerate() {
        parent_index_map.entry(value).or_insert(vec![]).push(index as u32);
    }

    // 创建一个数组用于存储子向量中元素的父向量索引
    let mut indices = Vec::with_capacity(child.len());

    // 遍历子向量，并查找每个元素在父向量中的索引
    for &child_value in child {
        match parent_index_map.get_mut(&child_value) {
            Some(indices_for_value) if !indices_for_value.is_empty() => {
                // 如果找到，将其索引添加到数组中，并从哈希映射中移除这个索引
                let parent_index = indices_for_value.pop().unwrap();
                indices.push(parent_index + 1);
            }
            _ => {
                // 如果没有找到足够的元素，返回错误
                return Err("Element in child vector not found in parent vector".to_string());
            }
        }
    }

    Ok(indices)
}
// copied from arkworks
pub(crate) fn convert_to_bigints<F: PrimeField>(p: &[F]) -> Vec<F::BigInt> {
    ark_std::cfg_iter!(p)
        .map(|s| s.into_repr())
        .collect::<Vec<_>>()
}

pub fn next_power_of_two(n: u32) -> u32 {
    if n == 0 || n > (u32::MAX >> 1) + 1 {
        return 0;
    }
    let mut tmp = n - 1;
    tmp |= tmp >> 1;
    tmp |= tmp >> 2;
    tmp |= tmp >> 4;
    tmp |= tmp >> 8;
    tmp |= tmp >> 16;
    tmp += 1;
    tmp
}


#[cfg(test)]
mod tests {
    use ark_bls12_381::{Bls12_381, Fr, G1Projective, G2Projective};
    use ark_ec::{AffineCurve, ProjectiveCurve};
    use ark_ff::Field;
    use ark_poly::{EvaluationDomain, GeneralEvaluationDomain, Polynomial, UVPolynomial};
    use ark_poly::univariate::DensePolynomial;
    use ark_std::{One, UniformRand, Zero};
    use crate::utils::{calc_lagrange_polynomials, calc_lagrange_polynomials_on_subset, calc_vanishing_polynomial, evaluate_polynomial_on_g1, evaluate_polynomial_on_g2, find_indices, gen_vanishing_polynomial, next_power_of_two, pairing_check, SRS};

    #[test]
    fn test_find_mapping_and_indices() {
        let parent = vec![Fr::from(5), Fr::from(2), Fr::from(3), Fr::from(1), Fr::from(4), Fr::from(3)];
        let child = vec![Fr::from(3), Fr::from(4), Fr::from(3), Fr::from(5)];
        match find_indices::<Bls12_381>(&parent, &child) {
            Ok(indices) => {
                println!("{:?}", indices); // 输出: [6, 5, 3, 1]
            }
            Err(err) => {
                println!("{}", err);
            }
        }
    }

    #[test]
    fn test_gen_vanishing_polynomial() {
        let degree = 8;
        let vanishing_poly = gen_vanishing_polynomial::<Bls12_381>(degree);
        // x^8 - 1
        assert_eq!(vanishing_poly, DensePolynomial::from_coefficients_slice(
            &[Fr::from(-1), Fr::zero(), Fr::zero(), Fr::zero(), Fr::zero(), Fr::zero(), Fr::zero(), Fr::zero(), Fr::one()]));
    }

    #[test]
    fn test_calc_vanishing_polynomial() {
        let degree = 8;
        let domain = GeneralEvaluationDomain::<Fr>::new(degree).unwrap();

        let pos = vec![1, 2, 6, 7, 8];
        let vanishing_poly = calc_vanishing_polynomial::<Bls12_381>(&pos, &domain);

        // C_I'(𝜔^(j_k)) = C_I(𝜔^(j_k))
        for i in 0..pos.len() {
            // index 0 == pos 1
            assert_eq!(Fr::zero(), vanishing_poly.evaluate(&domain.element(pos[i] as usize - 1)));
        }
    }

    #[test]
    fn test_calc_lagrange_polynomials_subset() {
        let degree = 2;
        let domain = GeneralEvaluationDomain::<Fr>::new(degree).unwrap();
        // domain = {1, -1}
        let mut positions = vec![]; // {0, 1}
        for i in 0..degree as u32 {
            positions.push(i + 1);
        }
        let lagrange_polynomials_subset = calc_lagrange_polynomials_on_subset::<Bls12_381>(&positions, &domain);

        let mut sum_of_lagrange_polynomials_subset = DensePolynomial::<Fr>::zero();
        for i in 0..lagrange_polynomials_subset.len() {
            sum_of_lagrange_polynomials_subset += &lagrange_polynomials_subset[i];
        }
        let a_half = Fr::one() * (Fr::from(2).inverse().unwrap());
        // 1/2 · x + 1/2
        assert_eq!(lagrange_polynomials_subset[0], DensePolynomial::from_coefficients_slice(&[a_half, a_half]));
        // -1/2 · x + 1/2
        assert_eq!(lagrange_polynomials_subset[1], DensePolynomial::from_coefficients_slice(&[a_half, -a_half]));
        // 求和为1
        assert_eq!(sum_of_lagrange_polynomials_subset.coeffs[0], Fr::one());
    }

    #[test]
    fn test_calc_lagrange_polynomials_set() {
        let degree = 16;
        let domain = GeneralEvaluationDomain::<Fr>::new(degree).unwrap();
        // domain = {1, -1}
        let lagrange_polynomials_subset = calc_lagrange_polynomials::<Bls12_381>(&domain);

        let mut sum_of_lagrange_polynomials_subset = DensePolynomial::<Fr>::zero();
        for i in 0..lagrange_polynomials_subset.len() {
            sum_of_lagrange_polynomials_subset += &lagrange_polynomials_subset[i];
        }
        if degree == 2 {
            let a_half = Fr::one() * (Fr::from(2).inverse().unwrap());
            // 1/2 · x + 1/2
            assert_eq!(lagrange_polynomials_subset[0], DensePolynomial::from_coefficients_slice(&[a_half, a_half]));
            // -1/2 · x + 1/2
            assert_eq!(lagrange_polynomials_subset[1], DensePolynomial::from_coefficients_slice(&[a_half, -a_half]));
        }
        // 求和为1
        assert_eq!(sum_of_lagrange_polynomials_subset.coeffs[0], Fr::one());
    }

    #[test]
    fn test_evaluate_polynomial_on_g1() {
        let rng = &mut rand::thread_rng();
        let x = Fr::from(2);
        let g1 = G1Projective::rand(rng).into_affine();
        let g2 = G2Projective::rand(rng).into_affine();
        let srs = SRS::<Bls12_381>::new(5, 5, &x, &g1, &g2).unwrap();

        // x^4 + x + 1
        let poly = DensePolynomial::from_coefficients_slice(&[
            Fr::one(), Fr::one(), Fr::zero(), Fr::zero(), Fr::one()
        ]);

        let g1_poly = evaluate_polynomial_on_g1::<Bls12_381>(&poly, &srs);
        // x^4 + x + 1
        let g1_x4_x_1 = srs.g1_powers[4] + srs.g1_powers[1] + srs.g1_powers[0];
        assert_eq!(g1_poly, g1_x4_x_1);
    }

    #[test]
    fn test_evaluate_polynomial_on_g2() {
        let rng = &mut rand::thread_rng();
        let x = Fr::from(2);
        let g1 = G1Projective::rand(rng).into_affine();
        let g2 = G2Projective::rand(rng).into_affine();
        let srs = SRS::<Bls12_381>::new(5, 5, &x, &g1, &g2).unwrap();

        // x^4 + x + 1
        let poly = DensePolynomial::from_coefficients_slice(&[
            Fr::one(), Fr::one(), Fr::zero(), Fr::zero(), Fr::one()
        ]);

        let g2_poly = evaluate_polynomial_on_g2::<Bls12_381>(&poly, &srs);
        // x^4 + x + 1
        let g2_x4_x_1 = srs.g2_powers[4] + srs.g2_powers[1] + srs.g2_powers[0];
        assert_eq!(g2_poly, g2_x4_x_1);
    }

    #[test]
    fn test_pairing_check() {
        let rng = &mut rand::thread_rng();
        let a = Fr::rand(rng);
        let b = Fr::rand(rng);
        let ab = a * b;

        let g1 = G1Projective::rand(rng).into_affine();
        let g2 = G2Projective::rand(rng).into_affine();
        let g1_a = g1.mul(a).into_affine();

        let g2_b = g2.mul(b).into_affine();
        let g2_ab = g2.mul(ab).into_affine();

        let res = pairing_check::<Bls12_381>(g1_a, g2_b, g1, g2_ab);
        assert_eq!(true, res);
    }

    #[test]
    fn test_next_power_of_two() {
        assert_eq!(0, next_power_of_two(0));
        assert_eq!(1, next_power_of_two(1));
        assert_eq!(2, next_power_of_two(2));
        assert_eq!(1024, next_power_of_two(1023));
        assert_eq!(1024, next_power_of_two(1024));
        assert_eq!(2048, next_power_of_two(1025));
        assert_eq!(0, next_power_of_two(u32::MAX));
        assert_eq!(0, next_power_of_two(u32::MAX - 1));
        assert_eq!(2147483648, next_power_of_two(u32::MAX >> 1));
        assert_eq!(2147483648, next_power_of_two((u32::MAX >> 1) + 1));
        assert_eq!(0, next_power_of_two((u32::MAX >> 1) + 2));
    }
}
