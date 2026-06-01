use std::env;
use std::fs::File;
use std::io::{BufReader, Read, Write};
use ark_ec::{AffineCurve, PairingEngine, ProjectiveCurve};
use ark_ff::{FromBytes, ToBytes};
use ark_serialize::CanonicalDeserialize;
use ark_std::{One, Zero};
use blake2::{Blake2b, Digest};
use byteorder::{BigEndian, ReadBytesExt};

/// SRS
#[derive(Clone)]
pub struct SRS<E: PairingEngine> {
    // degree of g1 powers
    pub g1_degree: u32,
    // degree of g2 powers
    pub g2_degree: u32,
    // [1]₁,[x]₁,...,[xᵈ]₁
    pub g1_powers: Vec<E::G1Affine>,
    // [1]₂,[x]₂,...,[xᵈ]₂
    pub g2_powers: Vec<E::G2Affine>,
}

// Implement ToBytes trait for SRS
impl<E: PairingEngine> ToBytes for SRS<E> {
    fn write<W: Write>(&self, mut writer: W) -> Result<(), std::io::Error> {
        self.g1_degree.write(&mut writer).expect("write g1 degree error");
        self.g2_degree.write(&mut writer).expect("write g2 degree error");
        self.g1_powers.write(&mut writer).expect("write g1_x_powers error");
        self.g2_powers.write(&mut writer).expect("write g2_x_powers error");
        Ok(())
    }
}

impl<E: PairingEngine> SRS<E> {
    pub fn new(
        g1_degree: u32,
        g2_degree: u32,
        tau: &E::Fr,
        g1: &E::G1Affine,
        g2: &E::G2Affine,
    ) -> Result<SRS<E>, String> {
        if g1_degree < 1 || g2_degree < 1 {
            return Err(String::from("invalid max degree"));
        }
        let mut g1_powers = vec![];
        let mut g2_powers = vec![];
        let mut lower_len = g2_degree;
        let mut extra_len = 0u32;
        if lower_len > g1_degree {
            lower_len = g1_degree + 1;
            extra_len = g2_degree - g1_degree;
        } else {
            lower_len += 1;
            extra_len = g1_degree - g2_degree
        }
        let mut tau_term = E::Fr::one();
        for _ in 0..lower_len {
            // g1 ^ x_term
            g1_powers.push(g1.mul(tau_term).into_affine());
            // g2 ^ x_term
            g2_powers.push(g2.mul(tau_term).into_affine());
            tau_term *= tau;
        }
        if lower_len == g2_degree + 1 {
            for _ in 0..extra_len {
                // g1 ^ x_term
                g1_powers.push(g1.mul(tau_term).into_affine());
                tau_term *= tau;
            }
        } else {
            for _ in 0..extra_len {
                // g2 ^ x_term
                g2_powers.push(g2.mul(tau_term).into_affine());
                tau_term *= tau;
            }
        }
        Ok(SRS {
            g1_degree,
            g2_degree,
            g1_powers,
            g2_powers,
        })
    }

    // trim srs to needed degree
    pub fn trim(mut self, g1_degree: u32, g2_degree: u32) -> Self {
        self.g1_degree = g1_degree;
        self.g2_degree = g2_degree;
        self.g1_powers.truncate(g1_degree as usize + 1);
        self.g2_powers.truncate(g2_degree as usize + 1);
        self
    }

    fn read<R: Read>(mut reader: R) -> Result<Self, String> {
        let g1_degree = reader.read_u32::<BigEndian>().expect("read g1 degree fail");
        let g2_degree = reader.read_u32::<BigEndian>().expect("read g2 degree fail");

        let g1_power_len = g1_degree as usize + 1;
        let g2_power_len = g2_degree as usize + 1;

        let mut g1_powers: Vec<E::G1Affine> = Vec::with_capacity(g1_power_len);
        g1_powers.resize_with(g1_power_len, || E::G1Affine::zero());
        let mut g2_powers: Vec<E::G2Affine> = Vec::with_capacity(g2_degree as usize);
        g2_powers.resize_with(g2_power_len, || E::G2Affine::zero());

        let mut compressed_g1_bytes: Vec<u8> = vec![0u8; 48];
        for i in 0..g1_power_len {
            reader.read_exact(&mut compressed_g1_bytes).expect("read g1 power fail");
            compressed_g1_bytes.reverse();
            g1_powers[i] = E::G1Affine::deserialize(&mut compressed_g1_bytes.as_slice()).expect("parse g1 power fail");
        }

        let mut compressed_g2_bytes: Vec<u8> = vec![0u8; 96];
        for i in 0..g2_power_len {
            reader.read_exact(&mut compressed_g2_bytes).expect("read g2 power fail");
            compressed_g2_bytes.reverse();
            g2_powers[i] = E::G2Affine::deserialize(&mut compressed_g2_bytes.as_slice()).expect("parse g2 power fail");
        }
        drop(reader);

        Ok(SRS {
            g1_degree,
            g2_degree,
            g1_powers,
            g2_powers,
        })
    }

    fn read_with_degree<R: Read>(mut reader: R, max_g1_degree: u32, max_g2_degree: u32) -> Result<Self, String> {
        let g1_degree = reader.read_u32::<BigEndian>().expect("read g1 degree fail");
        let g2_degree = reader.read_u32::<BigEndian>().expect("read g2 degree fail");
        if max_g1_degree > g1_degree {
            return Err("The g1 degree of SRS is too low".to_string());
        } else if max_g2_degree > g2_degree {
            return Err("The g2 degree of SRS is too low".to_string());
        }

        let g1_power_len = max_g1_degree as usize + 1;
        let g2_power_len = max_g2_degree as usize + 1;

        let mut g1_powers: Vec<E::G1Affine> = Vec::with_capacity(g1_power_len);
        g1_powers.resize_with(g1_power_len, || E::G1Affine::zero());
        let mut g2_powers: Vec<E::G2Affine> = Vec::with_capacity(g2_degree as usize);
        g2_powers.resize_with(g2_power_len, || E::G2Affine::zero());

        let mut compressed_g1_bytes: Vec<u8> = vec![0u8; 48];
        for i in 0..g1_power_len {
            reader.read_exact(&mut compressed_g1_bytes).expect("read g1 power fail");
            compressed_g1_bytes.reverse();
            g1_powers[i] = E::G1Affine::deserialize(&mut compressed_g1_bytes.as_slice()).expect("parse g1 power fail");
        }
        for _ in max_g1_degree..g1_degree {
            reader.read_exact(&mut compressed_g1_bytes).expect("read g1 power fail");
        }

        let mut compressed_g2_bytes: Vec<u8> = vec![0u8; 96];
        for i in 0..g2_power_len {
            reader.read_exact(&mut compressed_g2_bytes).expect("read g2 power fail");
            compressed_g2_bytes.reverse();
            g2_powers[i] = E::G2Affine::deserialize(&mut compressed_g2_bytes.as_slice()).expect("parse g2 power fail");
        }
        drop(reader);

        Ok(SRS {
            g1_degree,
            g2_degree,
            g1_powers,
            g2_powers,
        })
    }

    pub fn from_binary_file() -> Result<Self, String> {
        let file = match File::open("srs.binary") {
            Ok(file) => file,
            Err(err) => return Err(format!("Failed to open file: {}", err)),
        };
        let mut reader = BufReader::new(file);
        return match SRS::<E>::read(&mut reader) {
            Ok(srs) => Ok(srs),
            Err(err) => Err(format!("Failed to read SRS: {}", err)),
        };
    }

    pub fn from_binary_file_with_degree(g1_degree: u32, g2_degree: u32) -> Result<Self, String> {
        let file = match File::open("srs.binary") {
            Ok(file) => file,
            Err(err) => return Err(format!("Failed to open file: {}", err)),
        };
        let mut reader = BufReader::new(file);
        return match SRS::<E>::read_with_degree(&mut reader, g1_degree, g2_degree) {
            Ok(srs) => Ok(srs),
            Err(err) => Err(format!("Failed to read SRS: {}", err)),
        };
    }


    pub fn hash(&self) -> Vec<u8> {
        let mut hasher = Blake2b::new();
        let mut bytes = vec![];
        self.write(&mut bytes).unwrap();
        hasher.update(bytes);
        hasher.finalize().to_vec()
    }
}


#[cfg(test)]
mod tests {
    use std::fs::File;
    use std::io::BufReader;
    use std::time::Instant;

    use ark_bls12_381::{Bls12_381, Fr, G1Affine, G1Projective, G2Affine, G2Projective};
    use ark_bls12_381::g1::{G1_GENERATOR_X, G1_GENERATOR_Y};
    use ark_bls12_381::g2::{G2_GENERATOR_X, G2_GENERATOR_Y};
    use ark_ec::{AffineCurve, ProjectiveCurve};
    use ark_ff::{FromBytes};
    use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
    use ark_std::{UniformRand};

    use crate::srs::SRS;

    #[test]
    fn test_srs_new() {
        let rng = &mut rand::thread_rng();
        let x = Fr::rand(rng);
        let degree = 40;
        let g1 = G1Projective::rand(rng).into_affine();
        let g2 = G2Projective::rand(rng).into_affine();
        let srs = SRS::<Bls12_381>::new(degree, degree, &x, &g1, &g2).unwrap();
        let x3 = x * x * x;
        let g1_x3 = g1.mul(x3).into_affine();
        let g2_x3 = g2.mul(x3).into_affine();
        assert_eq!(g1, srs.g1_powers[0]);
        assert_eq!(g2, srs.g2_powers[0]);
        assert_eq!(g1_x3, srs.g1_powers[3]);
        assert_eq!(g2_x3, srs.g2_powers[3]);
        assert_eq!(degree as usize, srs.g1_powers.len());
        assert_eq!(degree as usize, srs.g2_powers.len());
    }

    #[test]
    fn test_g1_serialize() {
        let g1: G1Affine = G1Affine::new(G1_GENERATOR_X, G1_GENERATOR_Y, false);
        for i in 0..1024 {
            let g: G1Affine = g1.mul(i).into_affine();
            let mut buf: Vec<u8> = vec![];
            g.serialize(&mut buf).expect("TODO: panic message");
            println!("{:4} {:3} {:08b}", i, buf[47], buf[47]);
        }
    }

    #[test]
    fn test_g2_serialize() {
        let g2: G2Affine = G2Affine::new(G2_GENERATOR_X, G2_GENERATOR_Y, false);
        for i in 0..1024 {
            let g: G2Affine = g2.mul(i).into_affine();
            let mut buf: Vec<u8> = vec![];
            g.serialize(&mut buf).expect("TODO: panic message");
            // println!("{:4} {:?} {:08b}", i, buf, buf[47])
            //  len = 96
            println!(" {:4} {:3} {:08b}", i, buf[95], buf[95]);
        }
    }

    #[test]
    fn test_compressed_power_parse() {
        let mut compressed_g1_power: Vec<u8> =
            vec![
                128, 236, 62, 113, 167, 25, 162, 82,
                8, 173, 201, 113, 6, 177, 34, 128,
                146, 16, 250, 244, 90, 23, 219, 36,
                241, 15, 251, 26, 192, 20, 250, 193,
                171, 149, 164, 161, 150, 126, 85, 177,
                133, 212, 223, 98, 38, 133, 185, 232,
            ];
        compressed_g1_power.reverse();
        let g1_power = G1Affine::deserialize(&mut compressed_g1_power.as_slice()).expect("deserialize error");
        println!("{:?}", g1_power);
        let mut buf: Vec<u8> = vec![];
        g1_power.serialize(&mut buf).expect("serialize error");
        println!("{:?}", buf);
    }

    #[test]
    fn test_srs_read_with_degree() {
        let time_start = Instant::now();
        let srs: SRS<Bls12_381> = SRS::from_binary_file_with_degree(2 << 12, 2 << 10)
            .expect("Failed to read SRS from file");
        println!("time: {:?}", time_start.elapsed());
        println!("g1_degree: {}", srs.g1_degree);
        println!("g2_degree: {}", srs.g2_degree);
        println!("g1_powers[0]: {:?}", srs.g1_powers[0]);
        println!("g2_powers[0]: {:?}", srs.g2_powers[0]);
    }

    #[test]
    fn test_srs_read() {
        let time_start = Instant::now();
        let srs: SRS<Bls12_381> = SRS::from_binary_file()
            .expect("Failed to read SRS from file");
        println!("time: {:?}", time_start.elapsed());
        println!("g1_degree: {}", srs.g1_degree);
        println!("g2_degree: {}", srs.g2_degree);
        println!("g1_powers[0]: {:?}", srs.g1_powers[0]);
        println!("g2_powers[0]: {:?}", srs.g2_powers[0]);
    }
}