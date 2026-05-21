use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());

    // Path to the proto files. Defaults to the repo layout, but can be
    // overridden via PROTO_DIR — needed when building in a sandbox that mounts
    // only the crate directory (e.g. `cross`), where the protos are staged
    // inside the crate.
    let proto_path = env::var("PROTO_DIR").unwrap_or_else(|_| "../../protocols/proto".to_string());

    tonic_build::configure()
        .build_server(false) // We only need client code
        .build_client(true)
        .out_dir(&out_dir)
        .compile_protos(
            &[
                format!("{proto_path}/chunksink.proto"),
                format!("{proto_path}/common.proto"),
            ],
            &[proto_path.as_str()],
        )?;

    // Tell cargo to rerun this build script if the proto files change
    println!("cargo:rerun-if-changed={proto_path}/chunksink.proto");
    println!("cargo:rerun-if-changed={proto_path}/common.proto");

    Ok(())
}
