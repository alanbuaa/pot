// This file is @generated by prost-build.
/// bls12-381 Fr in big endian
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Fr {
    #[prost(uint64, tag = "1")]
    pub e1: u64,
    #[prost(uint64, tag = "2")]
    pub e2: u64,
    #[prost(uint64, tag = "3")]
    pub e3: u64,
    #[prost(uint64, tag = "4")]
    pub e4: u64,
}
/// bls12-381 Fq q = 384 in big endian
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Fq {
    #[prost(uint64, tag = "1")]
    pub e1: u64,
    #[prost(uint64, tag = "2")]
    pub e2: u64,
    #[prost(uint64, tag = "3")]
    pub e3: u64,
    #[prost(uint64, tag = "4")]
    pub e4: u64,
    #[prost(uint64, tag = "5")]
    pub e5: u64,
    #[prost(uint64, tag = "6")]
    pub e6: u64,
}
/// bls12-381 Fq2 q = 384 in big endian
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Fq2 {
    #[prost(message, optional, tag = "1")]
    pub c1: ::core::option::Option<Fq>,
    #[prost(message, optional, tag = "2")]
    pub c2: ::core::option::Option<Fq>,
}
/// bls12-381 g1 affine in big endian
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct G1Affine {
    #[prost(message, optional, tag = "1")]
    pub x: ::core::option::Option<Fq>,
    #[prost(message, optional, tag = "2")]
    pub y: ::core::option::Option<Fq>,
    #[prost(bool, tag = "3")]
    pub infinity: bool,
}
/// bls12-381 g2 affine in big endian
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct G2Affine {
    #[prost(message, optional, tag = "1")]
    pub x: ::core::option::Option<Fq2>,
    #[prost(message, optional, tag = "2")]
    pub y: ::core::option::Option<Fq2>,
    #[prost(bool, tag = "3")]
    pub infinity: bool,
}
