use rand::Rng;
use crate::bls12_381::{Fr as ProtoFr};
use crate::caulk_plus_grpc::{CreateMultiProofRequest, CreateSingleProofRequest,  MultiProof, SingleProof, VerifyReply};
use crate::caulk_plus_grpc::cp_service_client::CpServiceClient;

mod utils;


pub mod caulk_plus_grpc {
    tonic::include_proto!("caulk_plus");
}

pub mod bls12_381 {
    tonic::include_proto!("bls12_381");
}


#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = CpServiceClient::connect("http://127.0.0.1:50051").await?;
    let mut rng = rand::thread_rng();

    let parent_vector_size = 30u32;
    let mut parent_vector = Vec::with_capacity(parent_vector_size as usize);
    for _ in 0..parent_vector_size {
        parent_vector.push(ProtoFr {
            e1: rng.gen(),
            e2: rng.gen(),
            e3: rng.gen(),
            e4: rng.gen(),
        });
    }
    let sub_vector_size = 11u32;
    let mut sub_vector = Vec::with_capacity(sub_vector_size as usize);
    for i in 0..sub_vector_size {
        sub_vector.push(ProtoFr {
            e1: i as u64 + 1,
            e2: i as u64 * 100 + 1,
            e3: i as u64 * 10000 + 1,
            e4: i as u64 * 1000000 + 1,
        });
    }
    let I = vec![1usize, 3, 4, 7, 23, 8, 9, 10, 12, 17, 21, 22, 30, 24, 16, 26, 27, 29, 25, 31, 2, 11, 5, 6, 20, 14, 28, 13, 19, 18, 15];
    for i in 0..sub_vector_size as usize {
        parent_vector[I[i]] = sub_vector[i].clone()
    }
    for i in 0..parent_vector_size as usize {
        println!("parent [{}]: {:?}", i, parent_vector[i]);
    }
    for i in 0..sub_vector_size as usize {
        println!("sub_vc [{}]: {:?}", i, sub_vector[i]);
    }

    let request = tonic::Request::new(CreateMultiProofRequest {
        parent_vector_size,
        parent_vector,
        sub_vector_size,
        sub_vector,
    });

    let response = client.create_multi_proof(request).await?;
    let multi_proof = response.into_inner();
    println!("{:?}", multi_proof);

    let verify_response = client.verify_multi_proof(multi_proof).await?;
    println!("{}", verify_response.into_inner().res);
    Ok(())
}