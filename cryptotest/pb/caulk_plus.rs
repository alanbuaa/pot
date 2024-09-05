// This file is @generated by prost-build.
/// The request message
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateMultiProofRequest {
    #[prost(uint32, tag = "1")]
    pub parent_vector_size: u32,
    #[prost(message, repeated, tag = "2")]
    pub parent_vector: ::prost::alloc::vec::Vec<super::bls12_381::Fr>,
    #[prost(uint32, tag = "3")]
    pub sub_vector_size: u32,
    #[prost(message, repeated, tag = "4")]
    pub sub_vector: ::prost::alloc::vec::Vec<super::bls12_381::Fr>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct MultiProof {
    /// size of parent vector
    #[prost(uint32, tag = "1")]
    pub n: u32,
    /// size of domain H (padded)
    #[prost(uint32, tag = "2")]
    pub n_padded: u32,
    /// size of sub vector
    #[prost(uint32, tag = "3")]
    pub m: u32,
    /// size of domain V (padded)
    #[prost(uint32, tag = "4")]
    pub m_padded: u32,
    #[prost(message, optional, tag = "5")]
    pub c_commit: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "6")]
    pub a_commit: ::core::option::Option<super::bls12_381::G1Affine>,
    /// response
    #[prost(message, optional, tag = "7")]
    pub z_i: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "8")]
    pub c_i: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "9")]
    pub u: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "10")]
    pub w: ::core::option::Option<super::bls12_381::G2Affine>,
    #[prost(message, optional, tag = "11")]
    pub h: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "12")]
    pub v1: ::core::option::Option<super::bls12_381::Fr>,
    #[prost(message, optional, tag = "13")]
    pub pi_1: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "14")]
    pub v2: ::core::option::Option<super::bls12_381::Fr>,
    #[prost(message, optional, tag = "15")]
    pub pi_2: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "16")]
    pub pi_3: ::core::option::Option<super::bls12_381::G1Affine>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateSingleProofRequest {
    /// independent g1 generator
    #[prost(message, optional, tag = "1")]
    pub h_g1_generator: ::core::option::Option<super::bls12_381::G1Affine>,
    /// vector size
    #[prost(uint32, tag = "2")]
    pub parent_vector_size: u32,
    /// vector in u32
    #[prost(message, repeated, tag = "3")]
    pub parent_vector: ::prost::alloc::vec::Vec<super::bls12_381::Fr>,
    /// sub-vector in u32
    #[prost(message, optional, tag = "4")]
    pub chosen_element: ::core::option::Option<super::bls12_381::Fr>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct SingleProof {
    /// common input
    /// size of domain H (origin)
    #[prost(uint32, tag = "1")]
    pub n: u32,
    /// size of domain H (padded)
    #[prost(uint32, tag = "2")]
    pub n_padded: u32,
    /// KZG commitment 𝒄 = C\[x\]₁
    #[prost(message, optional, tag = "3")]
    pub c_commit: ::core::option::Option<super::bls12_381::G1Affine>,
    /// Pedersen commitment 𝒗 = \[v\]₁ + r𝒉
    #[prost(message, optional, tag = "4")]
    pub v_commit: ::core::option::Option<super::bls12_381::G1Affine>,
    /// R_Link^KZG proof
    #[prost(message, optional, tag = "5")]
    pub multi_proof: ::core::option::Option<MultiProof>,
    /// sᵥ
    #[prost(message, optional, tag = "6")]
    pub s_v: ::core::option::Option<super::bls12_381::Fr>,
    /// sᵣ
    #[prost(message, optional, tag = "7")]
    pub s_r: ::core::option::Option<super::bls12_381::Fr>,
    /// sₖ
    #[prost(message, optional, tag = "8")]
    pub s_k: ::core::option::Option<super::bls12_381::Fr>,
    /// v ̃
    #[prost(message, optional, tag = "9")]
    pub v_tilde: ::core::option::Option<super::bls12_381::G1Affine>,
    /// 𝒂
    #[prost(message, optional, tag = "10")]
    pub a_commit: ::core::option::Option<super::bls12_381::G1Affine>,
    /// ã
    #[prost(message, optional, tag = "11")]
    pub a_tilde: ::core::option::Option<super::bls12_381::G1Affine>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct VerifySingleProofRequest {
    #[prost(message, optional, tag = "1")]
    pub h_generator: ::core::option::Option<super::bls12_381::G1Affine>,
    #[prost(message, optional, tag = "2")]
    pub single_proof: ::core::option::Option<SingleProof>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct VerifyReply {
    #[prost(bool, tag = "1")]
    pub res: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct DomainSize {
    #[prost(uint32, tag = "1")]
    pub size: u32,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct RootsOfUnity {
    #[prost(uint32, tag = "1")]
    pub size: u32,
    #[prost(message, repeated, tag = "2")]
    pub roots: ::prost::alloc::vec::Vec<super::bls12_381::Fr>,
}
/// Generated server implementations.
pub mod cp_service_server {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    /// Generated trait containing gRPC methods that should be implemented for use with CpServiceServer.
    #[async_trait]
    pub trait CpService: Send + Sync + 'static {
        async fn create_multi_proof(
            &self,
            request: tonic::Request<super::CreateMultiProofRequest>,
        ) -> std::result::Result<tonic::Response<super::MultiProof>, tonic::Status>;
        async fn verify_multi_proof(
            &self,
            request: tonic::Request<super::MultiProof>,
        ) -> std::result::Result<tonic::Response<super::VerifyReply>, tonic::Status>;
        async fn create_single_proof(
            &self,
            request: tonic::Request<super::CreateSingleProofRequest>,
        ) -> std::result::Result<tonic::Response<super::SingleProof>, tonic::Status>;
        async fn verify_single_proof(
            &self,
            request: tonic::Request<super::VerifySingleProofRequest>,
        ) -> std::result::Result<tonic::Response<super::VerifyReply>, tonic::Status>;
        async fn calc_roots_of_unity(
            &self,
            request: tonic::Request<super::DomainSize>,
        ) -> std::result::Result<tonic::Response<super::RootsOfUnity>, tonic::Status>;
    }
    #[derive(Debug)]
    pub struct CpServiceServer<T: CpService> {
        inner: _Inner<T>,
        accept_compression_encodings: EnabledCompressionEncodings,
        send_compression_encodings: EnabledCompressionEncodings,
        max_decoding_message_size: Option<usize>,
        max_encoding_message_size: Option<usize>,
    }
    struct _Inner<T>(Arc<T>);
    impl<T: CpService> CpServiceServer<T> {
        pub fn new(inner: T) -> Self {
            Self::from_arc(Arc::new(inner))
        }
        pub fn from_arc(inner: Arc<T>) -> Self {
            let inner = _Inner(inner);
            Self {
                inner,
                accept_compression_encodings: Default::default(),
                send_compression_encodings: Default::default(),
                max_decoding_message_size: None,
                max_encoding_message_size: None,
            }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> InterceptedService<Self, F>
        where
            F: tonic::service::Interceptor,
        {
            InterceptedService::new(Self::new(inner), interceptor)
        }
        /// Enable decompressing requests with the given encoding.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.accept_compression_encodings.enable(encoding);
            self
        }
        /// Compress responses with the given encoding, if the client supports it.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.send_compression_encodings.enable(encoding);
            self
        }
        /// Limits the maximum size of a decoded message.
        ///
        /// Default: `4MB`
        #[must_use]
        pub fn max_decoding_message_size(mut self, limit: usize) -> Self {
            self.max_decoding_message_size = Some(limit);
            self
        }
        /// Limits the maximum size of an encoded message.
        ///
        /// Default: `usize::MAX`
        #[must_use]
        pub fn max_encoding_message_size(mut self, limit: usize) -> Self {
            self.max_encoding_message_size = Some(limit);
            self
        }
    }
    impl<T, B> tonic::codegen::Service<http::Request<B>> for CpServiceServer<T>
    where
        T: CpService,
        B: Body + Send + 'static,
        B::Error: Into<StdError> + Send + 'static,
    {
        type Response = http::Response<tonic::body::BoxBody>;
        type Error = std::convert::Infallible;
        type Future = BoxFuture<Self::Response, Self::Error>;
        fn poll_ready(
            &mut self,
            _cx: &mut Context<'_>,
        ) -> Poll<std::result::Result<(), Self::Error>> {
            Poll::Ready(Ok(()))
        }
        fn call(&mut self, req: http::Request<B>) -> Self::Future {
            let inner = self.inner.clone();
            match req.uri().path() {
                "/caulk_plus.CpService/CreateMultiProof" => {
                    #[allow(non_camel_case_types)]
                    struct CreateMultiProofSvc<T: CpService>(pub Arc<T>);
                    impl<
                        T: CpService,
                    > tonic::server::UnaryService<super::CreateMultiProofRequest>
                    for CreateMultiProofSvc<T> {
                        type Response = super::MultiProof;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::CreateMultiProofRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as CpService>::create_multi_proof(&inner, request).await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let inner = inner.0;
                        let method = CreateMultiProofSvc(inner);
                        let codec = tonic::codec::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/caulk_plus.CpService/VerifyMultiProof" => {
                    #[allow(non_camel_case_types)]
                    struct VerifyMultiProofSvc<T: CpService>(pub Arc<T>);
                    impl<T: CpService> tonic::server::UnaryService<super::MultiProof>
                    for VerifyMultiProofSvc<T> {
                        type Response = super::VerifyReply;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::MultiProof>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as CpService>::verify_multi_proof(&inner, request).await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let inner = inner.0;
                        let method = VerifyMultiProofSvc(inner);
                        let codec = tonic::codec::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/caulk_plus.CpService/CreateSingleProof" => {
                    #[allow(non_camel_case_types)]
                    struct CreateSingleProofSvc<T: CpService>(pub Arc<T>);
                    impl<
                        T: CpService,
                    > tonic::server::UnaryService<super::CreateSingleProofRequest>
                    for CreateSingleProofSvc<T> {
                        type Response = super::SingleProof;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::CreateSingleProofRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as CpService>::create_single_proof(&inner, request).await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let inner = inner.0;
                        let method = CreateSingleProofSvc(inner);
                        let codec = tonic::codec::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/caulk_plus.CpService/VerifySingleProof" => {
                    #[allow(non_camel_case_types)]
                    struct VerifySingleProofSvc<T: CpService>(pub Arc<T>);
                    impl<
                        T: CpService,
                    > tonic::server::UnaryService<super::VerifySingleProofRequest>
                    for VerifySingleProofSvc<T> {
                        type Response = super::VerifyReply;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::VerifySingleProofRequest>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as CpService>::verify_single_proof(&inner, request).await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let inner = inner.0;
                        let method = VerifySingleProofSvc(inner);
                        let codec = tonic::codec::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                "/caulk_plus.CpService/CalcRootsOfUnity" => {
                    #[allow(non_camel_case_types)]
                    struct CalcRootsOfUnitySvc<T: CpService>(pub Arc<T>);
                    impl<T: CpService> tonic::server::UnaryService<super::DomainSize>
                    for CalcRootsOfUnitySvc<T> {
                        type Response = super::RootsOfUnity;
                        type Future = BoxFuture<
                            tonic::Response<Self::Response>,
                            tonic::Status,
                        >;
                        fn call(
                            &mut self,
                            request: tonic::Request<super::DomainSize>,
                        ) -> Self::Future {
                            let inner = Arc::clone(&self.0);
                            let fut = async move {
                                <T as CpService>::calc_roots_of_unity(&inner, request).await
                            };
                            Box::pin(fut)
                        }
                    }
                    let accept_compression_encodings = self.accept_compression_encodings;
                    let send_compression_encodings = self.send_compression_encodings;
                    let max_decoding_message_size = self.max_decoding_message_size;
                    let max_encoding_message_size = self.max_encoding_message_size;
                    let inner = self.inner.clone();
                    let fut = async move {
                        let inner = inner.0;
                        let method = CalcRootsOfUnitySvc(inner);
                        let codec = tonic::codec::ProstCodec::default();
                        let mut grpc = tonic::server::Grpc::new(codec)
                            .apply_compression_config(
                                accept_compression_encodings,
                                send_compression_encodings,
                            )
                            .apply_max_message_size_config(
                                max_decoding_message_size,
                                max_encoding_message_size,
                            );
                        let res = grpc.unary(method, req).await;
                        Ok(res)
                    };
                    Box::pin(fut)
                }
                _ => {
                    Box::pin(async move {
                        Ok(
                            http::Response::builder()
                                .status(200)
                                .header("grpc-status", "12")
                                .header("content-type", "application/grpc")
                                .body(empty_body())
                                .unwrap(),
                        )
                    })
                }
            }
        }
    }
    impl<T: CpService> Clone for CpServiceServer<T> {
        fn clone(&self) -> Self {
            let inner = self.inner.clone();
            Self {
                inner,
                accept_compression_encodings: self.accept_compression_encodings,
                send_compression_encodings: self.send_compression_encodings,
                max_decoding_message_size: self.max_decoding_message_size,
                max_encoding_message_size: self.max_encoding_message_size,
            }
        }
    }
    impl<T: CpService> Clone for _Inner<T> {
        fn clone(&self) -> Self {
            Self(Arc::clone(&self.0))
        }
    }
    impl<T: std::fmt::Debug> std::fmt::Debug for _Inner<T> {
        fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            write!(f, "{:?}", self.0)
        }
    }
    impl<T: CpService> tonic::server::NamedService for CpServiceServer<T> {
        const NAME: &'static str = "caulk_plus.CpService";
    }
}