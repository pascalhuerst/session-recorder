use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());

    // Path to the proto files
    let proto_path = "../../protocols/proto";

    tonic_build::configure()
        .build_server(false) // We only need client code
        .build_client(true)
        .out_dir(&out_dir)
        .compile_protos(
            &[
                format!("{}/chunksink.proto", proto_path),
                format!("{}/common.proto", proto_path),
            ],
            &[proto_path],
        )?;

    // Tell cargo to rerun this build script if the proto files change
    println!("cargo:rerun-if-changed={}/chunksink.proto", proto_path);
    println!("cargo:rerun-if-changed={}/common.proto", proto_path);

    Ok(())
}
