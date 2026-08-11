//! CGo-compatible FFI for cggmp21 threshold signing.
//!
//! All functions are safe to call from C/CGo. Opaque session pointers must be
//! freed with the corresponding `_free` function. Buffers returned via out-params
//! must be freed with `cggmp21_free_buffer`.
//!
//! Fixed configuration: secp256k1 curve, SHA-256 digest, SecurityLevel128.
//!
//! ## Protocol loop (from the Go side)
//!
//! ```text
//! 1. session = cggmp21_signing_new(...)
//! 2. loop:
//!      // drain any initial outgoing messages
//!      while cggmp21_signing_poll_outgoing(...) == 1:
//!          send message to recipient
//!      if cggmp21_signing_is_done(session): break
//!      // receive one message from the network
//!      msg = recv_from_network()
//!      cggmp21_signing_deliver(session, msg.sender, msg.is_broadcast, msg.data)
//! 3. cggmp21_signing_result(session, sig_buf, sig_buf_len)
//! 4. cggmp21_signing_free(session)
//! ```

#![allow(clippy::missing_safety_doc)]

use std::collections::VecDeque;
use std::mem::ManuallyDrop;
use std::os::raw::c_char;

use cggmp21::{
    signing::msg::Msg, supported_curves::Secp256k1, DataToSign, ExecutionId, KeyShare,
    PartialSignature, Presignature, Signature, SigningError,
};
use cggmp21::define_security_level;
use cggmp21::generic_ec::Scalar;
use rand_core::OsRng;
use round_based::{
    state_machine::{ProceedResult, StateMachine},
    Incoming, MessageDestination, MessageType,
};
use sha2::Sha256;

// CompatLevel — a relaxed cggmp21 security level matching legacy tss-lib
// keyshares (1024-bit Paillier primes, 2048-bit modulus). The CGGMP21 protocol
// constants (epsilon/ell/ell_prime/m/q) are kept at cggmp21's stock
// SecurityLevel128 values, which oversizes the ZK proofs but does NOT weaken
// soundness — they just give more margin than needed at this prime size.
// SECURITY_BITS=256 is the only parameter that gates the Paillier prime size
// check (4·SECURITY_BITS ≤ prime bits). Net effect: ~112-bit symmetric-
// equivalent security from the Paillier modulus, vs cggmp21's intended ~128.
// Documented tradeoff for the tss-lib → cggmp21 migration.
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

type E = Secp256k1;
type D = Sha256;
type L = CompatLevel;
type SigningMsg = Msg<E, D>;
type SigningOutput = Result<Signature<E>, SigningError>;
type PresignOutput = Result<Presignature<E>, SigningError>;
type SigningSM = dyn StateMachine<Output = SigningOutput, Msg = SigningMsg>;
type PresignSM = dyn StateMachine<Output = PresignOutput, Msg = SigningMsg>;

// ── Status codes ──────────────────────────────────────────────────────────────

pub const CGGMP21_OK: i32 = 0;
pub const CGGMP21_ERR_INVALID_ARG: i32 = 1;
pub const CGGMP21_ERR_JSON: i32 = 2;
pub const CGGMP21_ERR_PROTOCOL: i32 = 3;
pub const CGGMP21_ERR_NOT_FINISHED: i32 = 4;

// ── Thread-local error storage ────────────────────────────────────────────────

std::thread_local! {
    static LAST_ERROR: std::cell::RefCell<String> = const { std::cell::RefCell::new(String::new()) };
}

fn set_last_error(msg: impl Into<String>) {
    LAST_ERROR.with(|e| *e.borrow_mut() = msg.into());
}

/// Format an error chain (top-level message + every `source()` in turn) so the
/// FFI surfaces the underlying cause instead of just "signing protocol failed".
fn format_err_chain(err: &dyn std::error::Error) -> String {
    let mut out = err.to_string();
    let mut src = err.source();
    while let Some(e) = src {
        out.push_str(": ");
        out.push_str(&e.to_string());
        src = e.source();
    }
    out
}

// ── Signing session ───────────────────────────────────────────────────────────

/// Opaque handle for a signing protocol session.
///
/// Each party participating in the signing protocol holds one session.
/// Drive the protocol by alternating `deliver` and `poll_outgoing` calls.
pub struct CggmpSigningSession {
    // Leaked boxes – freed in Drop after `sm` is dropped first.
    lk_key_share: *mut KeyShare<E, L>,
    lk_eid: *mut [u8],
    lk_parties: *mut [u16],
    lk_rng: *mut OsRng,

    // State machine holding 'static references into the leaked boxes.
    // SAFETY invariant: sm is dropped before the leaked boxes are freed (see Drop impl).
    sm: ManuallyDrop<Box<SigningSM>>,

    // Outgoing messages pending delivery, as (recipient, json_bytes).
    // recipient == -1 means broadcast; ≥ 0 is the target party's signing index.
    outgoing: VecDeque<(i32, Vec<u8>)>,

    // Filled once the protocol finishes.
    result: Option<Result<Vec<u8>, String>>,

    // Monotonically-increasing ID fed to round-based Incoming.
    msg_id: u64,
}

impl Drop for CggmpSigningSession {
    fn drop(&mut self) {
        unsafe {
            // Drop the state machine first – it holds 'static refs into the boxes below.
            ManuallyDrop::drop(&mut self.sm);
            drop(Box::from_raw(self.lk_key_share));
            drop(Box::from_raw(self.lk_eid));
            drop(Box::from_raw(self.lk_parties));
            drop(Box::from_raw(self.lk_rng));
        }
    }
}

/// Drive `sm.proceed()` until it blocks on a message, finishes, or errors.
/// Outgoing messages are collected into `outgoing`; the final result into `result`.
fn drive(
    sm: &mut Box<SigningSM>,
    outgoing: &mut VecDeque<(i32, Vec<u8>)>,
    result: &mut Option<Result<Vec<u8>, String>>,
) -> i32 {
    loop {
        match sm.proceed() {
            ProceedResult::SendMsg(msg) => {
                let recipient = match msg.recipient {
                    MessageDestination::AllParties => -1i32,
                    MessageDestination::OneParty(idx) => idx as i32,
                };
                match serde_json::to_vec(&msg.msg) {
                    Ok(bytes) => outgoing.push_back((recipient, bytes)),
                    Err(e) => {
                        set_last_error(format!("outgoing serialize: {e}"));
                        return CGGMP21_ERR_JSON;
                    }
                }
            }
            ProceedResult::Yielded => {
                // Protocol voluntarily paused; resume immediately.
            }
            ProceedResult::NeedsOneMoreMessage => {
                // Waiting for an incoming message – hand control back to the caller.
                return CGGMP21_OK;
            }
            ProceedResult::Output(output) => {
                match output {
                    Ok(sig) => {
                        let len = Signature::<E>::serialized_len();
                        let mut buf = vec![0u8; len];
                        sig.write_to_slice(&mut buf);
                        *result = Some(Ok(buf));
                    }
                    Err(e) => {
                        *result = Some(Err(format_err_chain(&e)));
                    }
                }
                return CGGMP21_OK;
            }
            ProceedResult::Error(e) => {
                set_last_error(format!("state machine error: {e:?}"));
                return CGGMP21_ERR_PROTOCOL;
            }
        }
    }
}

// ── Presignature session (mirrors the signing one, with Presignature output) ──

/// Opaque handle for an MPC presignature-generation session.
///
/// Use this for the offline phase of the one-round signing flow:
///   1. `cggmp21_presign_new` → drive with deliver/poll until `is_done`.
///   2. `cggmp21_presign_result` → serialized [`Presignature`] JSON.
///   3. Later: `cggmp21_partial_sign(presig_json, msg_hash, …)` → `PartialSignature` JSON.
///   4. `cggmp21_combine_partials(partials_json_array, …)` → 64-byte ECDSA signature.
pub struct CggmpPresignSession {
    lk_key_share: *mut KeyShare<E, L>,
    lk_eid: *mut [u8],
    lk_parties: *mut [u16],
    lk_rng: *mut OsRng,
    sm: ManuallyDrop<Box<PresignSM>>,
    outgoing: VecDeque<(i32, Vec<u8>)>,
    result: Option<Result<Vec<u8>, String>>,
    msg_id: u64,
}

impl Drop for CggmpPresignSession {
    fn drop(&mut self) {
        unsafe {
            ManuallyDrop::drop(&mut self.sm);
            drop(Box::from_raw(self.lk_key_share));
            drop(Box::from_raw(self.lk_eid));
            drop(Box::from_raw(self.lk_parties));
            drop(Box::from_raw(self.lk_rng));
        }
    }
}

/// Drive the presignature state machine. Mirrors [`drive`] but serializes the
/// final [`Presignature`] to JSON instead of writing raw signature bytes.
fn drive_presign(
    sm: &mut Box<PresignSM>,
    outgoing: &mut VecDeque<(i32, Vec<u8>)>,
    result: &mut Option<Result<Vec<u8>, String>>,
) -> i32 {
    loop {
        match sm.proceed() {
            ProceedResult::SendMsg(msg) => {
                let recipient = match msg.recipient {
                    MessageDestination::AllParties => -1i32,
                    MessageDestination::OneParty(idx) => idx as i32,
                };
                match serde_json::to_vec(&msg.msg) {
                    Ok(bytes) => outgoing.push_back((recipient, bytes)),
                    Err(e) => {
                        set_last_error(format!("outgoing serialize: {e}"));
                        return CGGMP21_ERR_JSON;
                    }
                }
            }
            ProceedResult::Yielded => {}
            ProceedResult::NeedsOneMoreMessage => return CGGMP21_OK,
            ProceedResult::Output(output) => {
                match output {
                    Ok(presig) => match serde_json::to_vec(&presig) {
                        Ok(bytes) => *result = Some(Ok(bytes)),
                        Err(e) => {
                            set_last_error(format!("presignature serialize: {e}"));
                            return CGGMP21_ERR_JSON;
                        }
                    },
                    Err(e) => {
                        *result = Some(Err(format_err_chain(&e)));
                    }
                }
                return CGGMP21_OK;
            }
            ProceedResult::Error(e) => {
                set_last_error(format!("state machine error: {e:?}"));
                return CGGMP21_ERR_PROTOCOL;
            }
        }
    }
}

// ── Constructor / destructor ──────────────────────────────────────────────────

/// Create a new signing session.
///
/// # Parameters
/// - `key_share_json` / `key_share_len`: JSON-serialized `KeyShare<Secp256k1>`.
/// - `eid` / `eid_len`: Execution ID bytes (must be unique per protocol run).
/// - `i`: This party's signing index (0-based).
/// - `parties_indexes` / `parties_count`: `parties_indexes[j]` is the keygen
///   index of the j-th signer participating in this round.
/// - `data_hash` / `data_hash_len`: 32-byte big-endian hash of the message to sign.
/// - `out`: On success, written with a pointer to the new session.
///
/// Returns `CGGMP21_OK` on success; sets `cggmp21_last_error()` on failure.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_new(
    key_share_json: *const u8,
    key_share_len: usize,
    eid: *const u8,
    eid_len: usize,
    i: u16,
    parties_indexes: *const u16,
    parties_count: usize,
    data_hash: *const u8,
    data_hash_len: usize,
    out: *mut *mut CggmpSigningSession,
) -> i32 {
    if key_share_json.is_null()
        || eid.is_null()
        || parties_indexes.is_null()
        || data_hash.is_null()
        || out.is_null()
    {
        set_last_error("null pointer argument");
        return CGGMP21_ERR_INVALID_ARG;
    }

    let ks_bytes = std::slice::from_raw_parts(key_share_json, key_share_len);
    let key_share: KeyShare<E, L> = match serde_json::from_slice(ks_bytes) {
        Ok(ks) => ks,
        Err(e) => {
            set_last_error(format!("key_share deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };

    let hash_bytes = std::slice::from_raw_parts(data_hash, data_hash_len);
    if hash_bytes.len() != 32 {
        set_last_error(format!(
            "data_hash must be 32 bytes, got {}",
            hash_bytes.len()
        ));
        return CGGMP21_ERR_INVALID_ARG;
    }

    // The 32-byte input is treated as the final pre-hash digest (e.g. Keccak-256
    // or SHA-256 output produced by the caller) and signed as-is, modulo the
    // curve order. The FFI does NOT hash it again.
    let scalar = Scalar::<E>::from_be_bytes_mod_order(hash_bytes);
    let data_to_sign = DataToSign::<E>::from_scalar(scalar);

    let eid_vec: Vec<u8> = std::slice::from_raw_parts(eid, eid_len).to_vec();
    let parties_vec: Vec<u16> =
        std::slice::from_raw_parts(parties_indexes, parties_count).to_vec();

    // Leak inputs that need 'static references for the state machine.
    let lk_key_share: &'static KeyShare<E, L> = &*Box::into_raw(Box::new(key_share));
    let lk_eid: &'static [u8] = Box::leak(eid_vec.into_boxed_slice());
    let lk_parties: &'static [u16] = Box::leak(parties_vec.into_boxed_slice());
    let rng_ptr: *mut OsRng = Box::into_raw(Box::new(OsRng));
    let lk_rng: &'static mut OsRng = &mut *rng_ptr;

    let sm_impl = cggmp21::signing(ExecutionId::new(lk_eid), i, lk_parties, lk_key_share)
        .sign_sync(lk_rng, data_to_sign);

    let sm: Box<SigningSM> = Box::new(sm_impl);

    let mut session = Box::new(CggmpSigningSession {
        lk_key_share: lk_key_share as *const _ as *mut _,
        lk_eid: lk_eid as *const _ as *mut _,
        lk_parties: lk_parties as *const _ as *mut _,
        lk_rng: rng_ptr,
        sm: ManuallyDrop::new(sm),
        outgoing: VecDeque::new(),
        result: None,
        msg_id: 0,
    });

    // Advance until the protocol asks for its first incoming message.
    let rc = drive(&mut session.sm, &mut session.outgoing, &mut session.result);
    if rc != CGGMP21_OK {
        // drive already called set_last_error; let Drop clean up the session
        return rc;
    }

    *out = Box::into_raw(session);
    CGGMP21_OK
}

/// Free a signing session. Passing NULL is a no-op.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_free(session: *mut CggmpSigningSession) {
    if !session.is_null() {
        drop(Box::from_raw(session));
    }
}

// ── Protocol stepping ─────────────────────────────────────────────────────────

/// Deliver an incoming protocol message from another party.
///
/// Call `cggmp21_signing_poll_outgoing` in a loop after each deliver to drain
/// any outgoing messages before delivering the next one.
///
/// - `sender`: Sender's signing index (0-based).
/// - `is_broadcast`: Non-zero if broadcast; zero for point-to-point.
/// - `msg_json` / `msg_len`: JSON-serialized protocol message.
///
/// Returns `CGGMP21_OK`, `CGGMP21_ERR_JSON`, or `CGGMP21_ERR_PROTOCOL`.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_deliver(
    session: *mut CggmpSigningSession,
    sender: u16,
    is_broadcast: u8,
    msg_json: *const u8,
    msg_len: usize,
) -> i32 {
    let s = &mut *session;

    let bytes = std::slice::from_raw_parts(msg_json, msg_len);
    let msg: SigningMsg = match serde_json::from_slice(bytes) {
        Ok(m) => m,
        Err(e) => {
            set_last_error(format!("msg deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };

    let msg_type = if is_broadcast != 0 {
        MessageType::Broadcast
    } else {
        MessageType::P2P
    };

    s.msg_id += 1;
    let incoming = Incoming {
        id: s.msg_id,
        sender,
        msg_type,
        msg,
    };

    // received_msg returns Err(msg) if the message is rejected (wrong round / unexpected).
    if let Err(_rejected) = s.sm.received_msg(incoming) {
        set_last_error("message rejected by state machine (unexpected sender or round)");
        return CGGMP21_ERR_PROTOCOL;
    }

    drive(&mut s.sm, &mut s.outgoing, &mut s.result)
}

/// Returns non-zero if the protocol has finished (successfully or with an error).
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_is_done(session: *const CggmpSigningSession) -> i32 {
    if (*session).result.is_some() { 1 } else { 0 }
}

// ── Outgoing messages ─────────────────────────────────────────────────────────

/// Pop the next pending outgoing protocol message.
///
/// Returns 1 if a message is available; 0 when the queue is empty.
/// Call repeatedly after each `cggmp21_signing_new` or `cggmp21_signing_deliver`
/// to drain all pending messages before waiting for the next incoming one.
///
/// On return value 1:
/// - `*out_recipient` is -1 for broadcast, or the target party's signing index.
/// - `*out_json` / `*out_len` hold the JSON message; **must** be freed with
///   `cggmp21_free_buffer(*out_json, *out_len)`.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_poll_outgoing(
    session: *mut CggmpSigningSession,
    out_recipient: *mut i32,
    out_json: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    let s = &mut *session;
    match s.outgoing.pop_front() {
        None => 0,
        Some((recipient, bytes)) => {
            *out_recipient = recipient;
            *out_len = bytes.len();
            let mut boxed = bytes.into_boxed_slice();
            *out_json = boxed.as_mut_ptr();
            std::mem::forget(boxed);
            1
        }
    }
}

// ── Result retrieval ──────────────────────────────────────────────────────────

/// Retrieve the signature once the protocol completes successfully.
///
/// Writes `r || s` (big-endian) into `out_sig`. Use `cggmp21_signature_len()`
/// to get the required buffer size (64 bytes for secp256k1).
///
/// Returns `CGGMP21_ERR_NOT_FINISHED` if the protocol is still running,
/// `CGGMP21_ERR_PROTOCOL` if it failed (see `cggmp21_last_error`).
#[no_mangle]
pub unsafe extern "C" fn cggmp21_signing_result(
    session: *const CggmpSigningSession,
    out_sig: *mut u8,
    out_sig_len: usize,
) -> i32 {
    let s = &*session;
    match &s.result {
        None => {
            set_last_error("protocol not finished yet");
            CGGMP21_ERR_NOT_FINISHED
        }
        Some(Err(e)) => {
            set_last_error(e.clone());
            CGGMP21_ERR_PROTOCOL
        }
        Some(Ok(sig_bytes)) => {
            let expected = Signature::<E>::serialized_len();
            if out_sig_len < expected {
                set_last_error(format!(
                    "output buffer too small: need {expected}, got {out_sig_len}"
                ));
                return CGGMP21_ERR_INVALID_ARG;
            }
            std::ptr::copy_nonoverlapping(sig_bytes.as_ptr(), out_sig, sig_bytes.len());
            CGGMP21_OK
        }
    }
}

/// Byte length of a serialized secp256k1 signature (`r || s`). Typically 64.
#[no_mangle]
pub extern "C" fn cggmp21_signature_len() -> usize {
    Signature::<E>::serialized_len()
}

// ── Utilities ─────────────────────────────────────────────────────────────────

/// Free a buffer allocated by a cggmp21 FFI function. NULL / len=0 is a no-op.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_free_buffer(buf: *mut u8, len: usize) {
    if !buf.is_null() && len > 0 {
        drop(Vec::from_raw_parts(buf, len, len));
    }
}

/// Returns a null-terminated string describing the most recent error on this
/// thread. Valid until the next cggmp21 call on this thread.
#[no_mangle]
pub extern "C" fn cggmp21_last_error() -> *const c_char {
    LAST_ERROR.with(|e| {
        let s = e.borrow();
        let c_str = std::ffi::CString::new(s.as_str()).unwrap_or_default();
        c_str.into_raw() as *const c_char
    })
}

/// cggmp21-ffi version string (null-terminated).
#[no_mangle]
pub extern "C" fn cggmp21_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr() as *const c_char
}

// ─────────────────────────────────────────────────────────────────────────────
//  One-round signing flow
//  ----------------------
//  Three pieces:
//    1. Presignature MPC session   (multi-round, message-independent)
//    2. Local partial signature    (single party, given presig + message)
//    3. Combine partial signatures (one broadcast round → final ECDSA sig)
// ─────────────────────────────────────────────────────────────────────────────

/// Allocate a heap buffer (Box<[u8]>) into the FFI out-pointer pair.
/// The caller must free it with `cggmp21_free_buffer`.
unsafe fn alloc_out_buffer(bytes: Vec<u8>, out_json: *mut *mut u8, out_len: *mut usize) {
    *out_len = bytes.len();
    let mut boxed = bytes.into_boxed_slice();
    *out_json = boxed.as_mut_ptr();
    std::mem::forget(boxed);
}

/// Create a new presignature-generation session.
///
/// Inputs match `cggmp21_signing_new` except there is no message hash:
/// presignatures are message-independent.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_new(
    key_share_json: *const u8,
    key_share_len: usize,
    eid: *const u8,
    eid_len: usize,
    i: u16,
    parties_indexes: *const u16,
    parties_count: usize,
    out: *mut *mut CggmpPresignSession,
) -> i32 {
    if key_share_json.is_null() || eid.is_null() || parties_indexes.is_null() || out.is_null() {
        set_last_error("null pointer argument");
        return CGGMP21_ERR_INVALID_ARG;
    }

    let ks_bytes = std::slice::from_raw_parts(key_share_json, key_share_len);
    let key_share: KeyShare<E, L> = match serde_json::from_slice(ks_bytes) {
        Ok(ks) => ks,
        Err(e) => {
            set_last_error(format!("key_share deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };

    let eid_vec: Vec<u8> = std::slice::from_raw_parts(eid, eid_len).to_vec();
    let parties_vec: Vec<u16> =
        std::slice::from_raw_parts(parties_indexes, parties_count).to_vec();

    let lk_key_share: &'static KeyShare<E, L> = &*Box::into_raw(Box::new(key_share));
    let lk_eid: &'static [u8] = Box::leak(eid_vec.into_boxed_slice());
    let lk_parties: &'static [u16] = Box::leak(parties_vec.into_boxed_slice());
    let rng_ptr: *mut OsRng = Box::into_raw(Box::new(OsRng));
    let lk_rng: &'static mut OsRng = &mut *rng_ptr;

    let sm_impl = cggmp21::signing(ExecutionId::new(lk_eid), i, lk_parties, lk_key_share)
        .generate_presignature_sync(lk_rng);
    let sm: Box<PresignSM> = Box::new(sm_impl);

    let mut session = Box::new(CggmpPresignSession {
        lk_key_share: lk_key_share as *const _ as *mut _,
        lk_eid: lk_eid as *const _ as *mut _,
        lk_parties: lk_parties as *const _ as *mut _,
        lk_rng: rng_ptr,
        sm: ManuallyDrop::new(sm),
        outgoing: VecDeque::new(),
        result: None,
        msg_id: 0,
    });

    let rc = drive_presign(&mut session.sm, &mut session.outgoing, &mut session.result);
    if rc != CGGMP21_OK {
        return rc;
    }

    *out = Box::into_raw(session);
    CGGMP21_OK
}

/// Free a presignature session. Passing NULL is a no-op.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_free(session: *mut CggmpPresignSession) {
    if !session.is_null() {
        drop(Box::from_raw(session));
    }
}

/// Deliver an incoming message to a presignature session.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_deliver(
    session: *mut CggmpPresignSession,
    sender: u16,
    is_broadcast: u8,
    msg_json: *const u8,
    msg_len: usize,
) -> i32 {
    let s = &mut *session;
    let bytes = std::slice::from_raw_parts(msg_json, msg_len);
    let msg: SigningMsg = match serde_json::from_slice(bytes) {
        Ok(m) => m,
        Err(e) => {
            set_last_error(format!("msg deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };
    let msg_type = if is_broadcast != 0 {
        MessageType::Broadcast
    } else {
        MessageType::P2P
    };
    s.msg_id += 1;
    let incoming = Incoming {
        id: s.msg_id,
        sender,
        msg_type,
        msg,
    };
    if s.sm.received_msg(incoming).is_err() {
        set_last_error("message rejected by state machine (unexpected sender or round)");
        return CGGMP21_ERR_PROTOCOL;
    }
    drive_presign(&mut s.sm, &mut s.outgoing, &mut s.result)
}

/// Pop the next outgoing message from a presignature session. Same contract as
/// `cggmp21_signing_poll_outgoing`.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_poll_outgoing(
    session: *mut CggmpPresignSession,
    out_recipient: *mut i32,
    out_json: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    let s = &mut *session;
    match s.outgoing.pop_front() {
        None => 0,
        Some((recipient, bytes)) => {
            *out_recipient = recipient;
            alloc_out_buffer(bytes, out_json, out_len);
            1
        }
    }
}

/// Returns non-zero once the presignature protocol has finished.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_is_done(session: *const CggmpPresignSession) -> i32 {
    if (*session).result.is_some() { 1 } else { 0 }
}

/// Retrieve the JSON-serialized [`Presignature`] after the protocol completes.
///
/// On success the caller MUST free `*out_json` with `cggmp21_free_buffer`.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_presign_result(
    session: *const CggmpPresignSession,
    out_json: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    let s = &*session;
    match &s.result {
        None => {
            set_last_error("presignature not finished yet");
            CGGMP21_ERR_NOT_FINISHED
        }
        Some(Err(e)) => {
            set_last_error(e.clone());
            CGGMP21_ERR_PROTOCOL
        }
        Some(Ok(bytes)) => {
            alloc_out_buffer(bytes.clone(), out_json, out_len);
            CGGMP21_OK
        }
    }
}

/// Locally compute a [`PartialSignature`] from a presignature and a 32-byte
/// message hash. **A presignature must NEVER be reused** — doing so leaks the
/// private key.
///
/// Output is JSON; caller frees `*out_json` with `cggmp21_free_buffer`.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_partial_sign(
    presignature_json: *const u8,
    presignature_len: usize,
    data_hash: *const u8,
    data_hash_len: usize,
    out_json: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    if presignature_json.is_null() || data_hash.is_null() || out_json.is_null() || out_len.is_null()
    {
        set_last_error("null pointer argument");
        return CGGMP21_ERR_INVALID_ARG;
    }

    let presig_bytes = std::slice::from_raw_parts(presignature_json, presignature_len);
    let presig: Presignature<E> = match serde_json::from_slice(presig_bytes) {
        Ok(p) => p,
        Err(e) => {
            set_last_error(format!("presignature deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };

    let hash_bytes = std::slice::from_raw_parts(data_hash, data_hash_len);
    if hash_bytes.len() != 32 {
        set_last_error(format!(
            "data_hash must be 32 bytes, got {}",
            hash_bytes.len()
        ));
        return CGGMP21_ERR_INVALID_ARG;
    }
    let scalar = Scalar::<E>::from_be_bytes_mod_order(hash_bytes);
    let data_to_sign = DataToSign::<E>::from_scalar(scalar);

    let partial = presig.issue_partial_signature(data_to_sign);
    match serde_json::to_vec(&partial) {
        Ok(bytes) => {
            alloc_out_buffer(bytes, out_json, out_len);
            CGGMP21_OK
        }
        Err(e) => {
            set_last_error(format!("partial signature serialize: {e}"));
            CGGMP21_ERR_JSON
        }
    }
}

/// Combine a threshold-sized JSON array of [`PartialSignature`]s into a final
/// ECDSA signature. Writes `r || s` (big-endian, 64 bytes on secp256k1) to
/// `out_sig`. Use `cggmp21_signature_len()` for the required size.
///
/// `partials_json_array` must deserialize to `[PartialSignature, …]`.
///
/// Returns `CGGMP21_ERR_PROTOCOL` if `combine` rejects the input (e.g. mixed
/// presignatures, malformed scalars, or empty array). The returned signature
/// may still be invalid for the public key/message if a signer cheated —
/// callers should verify before trusting it.
#[no_mangle]
pub unsafe extern "C" fn cggmp21_combine_partials(
    partials_json_array: *const u8,
    partials_len: usize,
    out_sig: *mut u8,
    out_sig_len: usize,
) -> i32 {
    if partials_json_array.is_null() || out_sig.is_null() {
        set_last_error("null pointer argument");
        return CGGMP21_ERR_INVALID_ARG;
    }

    let bytes = std::slice::from_raw_parts(partials_json_array, partials_len);
    let partials: Vec<PartialSignature<E>> = match serde_json::from_slice(bytes) {
        Ok(v) => v,
        Err(e) => {
            set_last_error(format!("partials deserialize: {e}"));
            return CGGMP21_ERR_JSON;
        }
    };

    let sig = match PartialSignature::<E>::combine(&partials) {
        Some(s) => s,
        None => {
            set_last_error("combine rejected partials (empty or malformed)");
            return CGGMP21_ERR_PROTOCOL;
        }
    };

    let expected = Signature::<E>::serialized_len();
    if out_sig_len < expected {
        set_last_error(format!(
            "output buffer too small: need {expected}, got {out_sig_len}"
        ));
        return CGGMP21_ERR_INVALID_ARG;
    }
    let mut buf = vec![0u8; expected];
    sig.write_to_slice(&mut buf);
    std::ptr::copy_nonoverlapping(buf.as_ptr(), out_sig, buf.len());
    CGGMP21_OK
}
