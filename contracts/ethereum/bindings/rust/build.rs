use std::env;
use std::path::PathBuf;

fn main() {
    // Re-run build script if ABI files change
    println!("cargo:rerun-if-changed=abi/");
    
    // Set ABI directory path for runtime
    let abi_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap())
        .join("abi");
    println!("cargo:rustc-env=ABI_DIR={}", abi_dir.display());
}
