use tonic::{Request, Response, Status};
use ark_bls12_381::{Bls12_381, Fr};
use ark_poly::{EvaluationDomain, GeneralEvaluationDomain};
use tonic::Code::Internal;

use tonic::transport::Server;
use caulk_plus::api::API;
use crate::caulk_plus_grpc::{CreateMultiProofRequest, CreateSingleProofRequest, MultiProof as ProtoMultiProof, SingleProof as ProtoSingleProof, VerifySingleProofRequest, VerifyReply, DomainSize, RootsOfUnity};
use crate::caulk_plus_grpc::cp_service_server::{CpService, CpServiceServer};
use crate::utils::{convert_fr_to_proto_fr, convert_multi_proof_to_proto_multi_proof, convert_proto_fr_to_fr, convert_proto_g1_to_g1, convert_proto_multi_proof_to_multi_proof, convert_proto_single_proof_to_single_proof, convert_single_proof_to_proto_single_proof};

pub mod utils;
pub mod caulk_plus_grpc {
    tonic::include_proto!("caulk_plus");
}

pub mod bls12_381 {
    tonic::include_proto!("bls12_381");
}

#[derive(Debug, Default)]
pub struct CaulkPlusService {}

#[allow(non_snake_case)]
#[tonic::async_trait]
impl CpService for CaulkPlusService {
    async fn create_multi_proof(
        &self,
        request: Request<CreateMultiProofRequest>,
    ) -> Result<Response<ProtoMultiProof>, Status> {
        let inner = request.into_inner();
        let n = inner.parent_vector_size;
        let proto_fr_parent_vector = inner.parent_vector;
        let mut parent_vector = Vec::with_capacity(n as usize);
        for i in 0..n {
            parent_vector.push(convert_proto_fr_to_fr(&proto_fr_parent_vector[i as usize].clone()));
        }
        let m = inner.sub_vector_size;
        let proto_fr_sub_vector = inner.sub_vector;
        let mut sub_vector = Vec::with_capacity(m as usize);
        for i in 0..m {
            sub_vector.push(convert_proto_fr_to_fr(&proto_fr_sub_vector[i as usize].clone()));
        }
        println!("Got a request: n = {}, m = {}", n, m);
        return match API::<Bls12_381>::create_multi_proof(n, &parent_vector, m, &sub_vector) {
            Ok(multi_proof) => Ok(Response::new(convert_multi_proof_to_proto_multi_proof(&multi_proof))),
            Err(err) => Err(Status::new(Internal, format!("Create Multi Proof Error: {}", err))),
        };
    }

    async fn verify_multi_proof(&self, request: Request<ProtoMultiProof>) -> Result<Response<VerifyReply>, Status> {
        let inner = request.into_inner();
        return match API::<Bls12_381>::verify_multi_proof(&convert_proto_multi_proof_to_multi_proof(&inner)) {
            Ok(res) => Ok(Response::new(VerifyReply { res })),
            Err(err) => Err(Status::new(Internal, format!("Verify Multi Proof Error: {}", err))),
        };
    }

    async fn create_single_proof(&self, request: Request<CreateSingleProofRequest>) -> Result<Response<ProtoSingleProof>, Status> {
        let inner = request.into_inner();
        let h_g1_generator = convert_proto_g1_to_g1(&inner.h_g1_generator.clone().unwrap());
        let n = inner.parent_vector_size;
        let proto_fr_parent_vector = inner.parent_vector;
        let mut parent_vector = Vec::with_capacity(n as usize);
        for i in 0..n {
            parent_vector.push(convert_proto_fr_to_fr(&proto_fr_parent_vector[i as usize].clone()));
        }
        let chosen_element = convert_proto_fr_to_fr(&inner.chosen_element.unwrap());

        return match API::<Bls12_381>::create_single_proof(h_g1_generator, n, &parent_vector, chosen_element) {
            Ok(single_proof) => Ok(Response::new(convert_single_proof_to_proto_single_proof(&single_proof))),
            Err(err) => Err(Status::new(Internal, format!("Create Single Proof Error: {}", err))),
        };
    }

    async fn verify_single_proof(&self, request: Request<VerifySingleProofRequest>) -> Result<Response<VerifyReply>, Status> {
        let inner = request.into_inner();
        let h_generator = inner.h_generator.unwrap();
        let single_proof = inner.single_proof.unwrap();
        return match API::<Bls12_381>::verify_single_proof(convert_proto_g1_to_g1(&h_generator), &convert_proto_single_proof_to_single_proof(&single_proof)) {
            Ok(res) => Ok(Response::new(VerifyReply { res })),
            Err(err) => Err(Status::new(Internal, format!("Verify Multi Proof Error: {}", err))),
        };
    }

    async fn calc_roots_of_unity(&self, request: Request<DomainSize>) -> Result<Response<RootsOfUnity>, Status> {
        let size = request.into_inner().size;
        return match GeneralEvaluationDomain::<Fr>::new(size as usize) {
            Some(domain_H) => {
                Ok(Response::new(RootsOfUnity { size, roots: (0..domain_H.size()).map(|i| convert_fr_to_proto_fr(&domain_H.element(i))).collect() }))
            }
            None => Err(Status::new(Internal, "Verify Multi Proof Error:".to_string())),
        };
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "127.0.0.1:50051".parse()?;
    let calc_proof_service = CaulkPlusService::default();

    Server::builder()
        .add_service(CpServiceServer::new(calc_proof_service))
        .serve(addr)
        .await?;

    Ok(())
}
