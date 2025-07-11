fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure().build_server(true).compile(
        &[
            "../../protocols/proto/chunksink.proto",
            "../../protocols/proto/common.proto",
        ],
        &["../../protocols/proto"],
    )?;
    Ok(())
}
