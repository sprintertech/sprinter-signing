//! Generates 2-of-3 secp256k1 cggmp21 key shares for use as Go test fixtures.
//!
//! Run from the repo root:
//!
//!     cargo run --release --manifest-path rust/Cargo.toml \
//!         --example gen_fixtures -- tss/cggmp21/testdata
//!
//! Writes:
//!   testdata/share-0.json   serialized KeyShare<Secp256k1> for party 0
//!   testdata/share-1.json   serialized KeyShare<Secp256k1> for party 1
//!   testdata/share-2.json   serialized KeyShare<Secp256k1> for party 2
//!   testdata/pubkey.bin     compressed 33-byte secp256k1 shared public key
//!
//! Slow (~10-60s) because it generates Paillier moduli per party.

use std::{fs, path::PathBuf};

use cggmp21::{define_security_level, supported_curves::Secp256k1, trusted_dealer};
use rand_core::OsRng;

// Mirrors the CompatLevel defined in src/lib.rs. Must stay in sync; the
// fixtures are deserialised by FFI sessions parameterised on CompatLevel, so
// the levels' SECURITY_BITS must match.
#[derive(Clone)]
pub struct CompatLevel;
define_security_level!(CompatLevel {
    security_bits = 256,
    epsilon = 230,
    ell = 256,
    ell_prime = 848,
    m = 128,
    q = (cggmp21::security_level::_internal::Integer::ONE << 128_u32).into(),
});

fn main() {
    let outdir: PathBuf = std::env::args()
        .nth(1)
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from("tss/cggmp21/testdata"));
    fs::create_dir_all(&outdir).expect("create testdata dir");

    let n: u16 = 3;
    let t: u16 = 2;

    eprintln!("Generating {t}-of-{n} secp256k1 key shares — this can take 10-60s…");
    let shares = trusted_dealer::builder::<Secp256k1, CompatLevel>(n)
        .set_threshold(Some(t))
        .generate_shares(&mut OsRng)
        .expect("generate shares");

    for (i, share) in shares.iter().enumerate() {
        let path = outdir.join(format!("share-{i}.json"));
        let json = serde_json::to_vec_pretty(share).expect("serialize share");
        fs::write(&path, &json).expect("write share");
        eprintln!("  {} ({} bytes)", path.display(), json.len());
    }

    let pubkey = shares[0].shared_public_key.to_bytes(true).to_vec();
    let pubkey_path = outdir.join("pubkey.bin");
    fs::write(&pubkey_path, &pubkey).expect("write pubkey");
    eprintln!(
        "  {} ({} bytes, compressed secp256k1)",
        pubkey_path.display(),
        pubkey.len()
    );

    eprintln!("Done.");
}
