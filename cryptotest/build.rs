fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("pb/caulk_plus.proto").unwrap_or_else(|e| panic!("Failed to compile protos {:?}", e));
    tonic_build::compile_protos("pb/bls12_381.proto").unwrap_or_else(|e| panic!("Failed to compile protos {:?}", e));
    Ok(())
}
