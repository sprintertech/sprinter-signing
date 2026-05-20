/*
 * cggmp21.h — C header for the cggmp21-ffi Rust library.
 *
 * Fixed configuration: secp256k1 curve · SHA-256 digest · SecurityLevel128.
 *
 * Usage from CGo:
 *
 *   // #cgo LDFLAGS: -L${SRCDIR}/../../rust/lib -lcggmp21_ffi -lm -ldl
 *   // #include "../../rust/include/cggmp21.h"
 *   import "C"
 */

#ifndef CGGMP21_H
#define CGGMP21_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>
#include <stdint.h>

/* ── Status codes ──────────────────────────────────────────────────────────── */

/** Returned by most functions on success. */
#define CGGMP21_OK            0
/** A required pointer argument was NULL or a numeric argument was out of range. */
#define CGGMP21_ERR_INVALID_ARG 1
/** JSON serialisation or deserialisation failed. */
#define CGGMP21_ERR_JSON      2
/** The MPC protocol returned an error (see cggmp21_last_error). */
#define CGGMP21_ERR_PROTOCOL  3
/** The requested operation is only valid after the protocol finishes. */
#define CGGMP21_ERR_NOT_FINISHED 4

/* ── Opaque types ──────────────────────────────────────────────────────────── */

/**
 * Opaque handle for a single-party signing session.
 * Must be freed with cggmp21_signing_free when no longer needed.
 */
typedef struct CggmpSigningSession CggmpSigningSession;

/* ── Session lifecycle ─────────────────────────────────────────────────────── */

/**
 * Create a new signing session for this party.
 *
 * @param key_share_json   JSON-serialised KeyShare<Secp256k1> (from cggmp21 keygen + aux-info).
 * @param key_share_len    Length of key_share_json in bytes.
 * @param eid              Execution-ID bytes. Must be unique per protocol execution.
 * @param eid_len          Length of eid in bytes.
 * @param i                This party's signing index (0-based, must be < parties_count).
 * @param parties_indexes  Array of length parties_count. parties_indexes[j] is the keygen
 *                         index of the j-th signer participating in this round.
 * @param parties_count    Number of signing parties (= threshold t).
 * @param data_hash        32-byte pre-hashed digest to sign (e.g. Keccak-256 or SHA-256
 *                         output produced by the caller). The library signs this value
 *                         as-is, reduced modulo the secp256k1 curve order. It does NOT
 *                         hash the input again.
 * @param data_hash_len    Must be 32.
 * @param out              On CGGMP21_OK, written with the new session pointer.
 *
 * @return CGGMP21_OK on success, otherwise an error code; call cggmp21_last_error() for details.
 */
int cggmp21_signing_new(
    const uint8_t *key_share_json, size_t key_share_len,
    const uint8_t *eid,            size_t eid_len,
    uint16_t       i,
    const uint16_t *parties_indexes, size_t parties_count,
    const uint8_t *data_hash,      size_t data_hash_len,
    CggmpSigningSession **out
);

/**
 * Free a session created by cggmp21_signing_new.
 * Passing NULL is a no-op.
 */
void cggmp21_signing_free(CggmpSigningSession *session);

/* ── Protocol stepping ─────────────────────────────────────────────────────── */

/**
 * Deliver an incoming protocol message from another party.
 *
 * After each call, drain any queued outgoing messages with
 * cggmp21_signing_poll_outgoing before delivering the next message.
 *
 * @param session      The session that receives the message.
 * @param sender       Sender's signing index (0-based).
 * @param is_broadcast Non-zero if the message is a broadcast; zero for point-to-point.
 * @param msg_json     JSON-serialised protocol message bytes.
 * @param msg_len      Length of msg_json in bytes.
 *
 * @return CGGMP21_OK, CGGMP21_ERR_JSON, or CGGMP21_ERR_PROTOCOL.
 */
int cggmp21_signing_deliver(
    CggmpSigningSession *session,
    uint16_t sender,
    uint8_t  is_broadcast,
    const uint8_t *msg_json, size_t msg_len
);

/**
 * Returns non-zero if the protocol has finished (successfully or with an error).
 * Once this returns non-zero, call cggmp21_signing_result to get the signature.
 */
int cggmp21_signing_is_done(const CggmpSigningSession *session);

/* ── Outgoing messages ─────────────────────────────────────────────────────── */

/**
 * Pop the next pending outgoing protocol message.
 *
 * Returns 1 if a message was available; 0 if the queue is empty.
 * Call in a loop until it returns 0 to drain all queued messages.
 *
 * On success (return value 1):
 *   - *out_recipient is set to -1 for a broadcast message, or to the target
 *     party's signing index for a point-to-point message.
 *   - *out_json points to a heap-allocated buffer of *out_len bytes containing
 *     the JSON-serialised protocol message.  The caller MUST free it with
 *     cggmp21_free_buffer(*out_json, *out_len).
 *
 * @param session       The session to poll.
 * @param out_recipient Written with the recipient index or -1.
 * @param out_json      Written with a pointer to the message buffer.
 * @param out_len       Written with the length of the message buffer.
 */
int cggmp21_signing_poll_outgoing(
    CggmpSigningSession *session,
    int32_t  *out_recipient,
    uint8_t **out_json,
    size_t   *out_len
);

/* ── Result retrieval ──────────────────────────────────────────────────────── */

/**
 * Retrieve the signature after the protocol finishes successfully.
 *
 * The signature is written as r || s in big-endian encoding.
 * Use cggmp21_signature_len() to obtain the required buffer size.
 *
 * @param session      A finished session (cggmp21_signing_is_done returns non-zero).
 * @param out_sig      Buffer of at least cggmp21_signature_len() bytes.
 * @param out_sig_len  Length of out_sig in bytes.
 *
 * @return CGGMP21_OK on success.
 *         CGGMP21_ERR_NOT_FINISHED if the protocol is not yet complete.
 *         CGGMP21_ERR_PROTOCOL if the protocol completed with an error.
 *         CGGMP21_ERR_INVALID_ARG if out_sig_len is too small.
 */
int cggmp21_signing_result(
    const CggmpSigningSession *session,
    uint8_t *out_sig, size_t out_sig_len
);

/**
 * Returns the byte length of a serialised secp256k1 signature (r || s).
 * Typically 64 bytes.
 */
size_t cggmp21_signature_len(void);

/* ── Utilities ─────────────────────────────────────────────────────────────── */

/**
 * Free a buffer that was allocated by a cggmp21 FFI function.
 * Passing NULL or len=0 is a no-op.
 *
 * @param buf Pointer returned in an out-parameter by a cggmp21 function.
 * @param len Byte length reported alongside the pointer.
 */
void cggmp21_free_buffer(uint8_t *buf, size_t len);

/**
 * Returns a pointer to a null-terminated string describing the most recent
 * error on this thread.  The pointer is valid until the next cggmp21 call on
 * this thread.  Returns an empty string if no error has occurred.
 */
const char *cggmp21_last_error(void);

/**
 * Returns the cggmp21-ffi library version as a null-terminated string.
 */
const char *cggmp21_version(void);

/* ── One-round signing: presignature + partial sign + combine ─────────────── */

/**
 * Opaque handle for a presignature-generation MPC session.
 * Must be freed with cggmp21_presign_free.
 */
typedef struct CggmpPresignSession CggmpPresignSession;

/**
 * Start a presignature-generation session.
 *
 * Drive the protocol with cggmp21_presign_deliver / cggmp21_presign_poll_outgoing,
 * just like a signing session. The presignature is message-independent — no
 * data_hash is required at this stage.
 *
 * Once cggmp21_presign_is_done returns non-zero, fetch the serialized
 * Presignature with cggmp21_presign_result.
 *
 * Parameters mirror cggmp21_signing_new minus data_hash.
 *
 * @return CGGMP21_OK on success; otherwise call cggmp21_last_error().
 */
int cggmp21_presign_new(
    const uint8_t *key_share_json, size_t key_share_len,
    const uint8_t *eid,            size_t eid_len,
    uint16_t       i,
    const uint16_t *parties_indexes, size_t parties_count,
    CggmpPresignSession **out
);

/** Free a presignature session. NULL is a no-op. */
void cggmp21_presign_free(CggmpPresignSession *session);

/** Deliver an incoming protocol message to a presignature session. */
int cggmp21_presign_deliver(
    CggmpPresignSession *session,
    uint16_t sender,
    uint8_t  is_broadcast,
    const uint8_t *msg_json, size_t msg_len
);

/**
 * Pop the next outgoing message from a presignature session.
 * Same contract as cggmp21_signing_poll_outgoing: returns 1 if a message was
 * available, 0 otherwise; *out_json must be freed with cggmp21_free_buffer.
 */
int cggmp21_presign_poll_outgoing(
    CggmpPresignSession *session,
    int32_t  *out_recipient,
    uint8_t **out_json,
    size_t   *out_len
);

/** Returns non-zero once the presignature protocol has finished. */
int cggmp21_presign_is_done(const CggmpPresignSession *session);

/**
 * Retrieve the JSON-serialised Presignature once the protocol finishes.
 *
 * On success, *out_json is a heap-allocated buffer that the caller MUST free
 * with cggmp21_free_buffer(*out_json, *out_len).
 *
 * @return CGGMP21_OK, CGGMP21_ERR_NOT_FINISHED, or CGGMP21_ERR_PROTOCOL.
 */
int cggmp21_presign_result(
    const CggmpPresignSession *session,
    uint8_t **out_json, size_t *out_len
);

/**
 * Locally compute a PartialSignature from a presignature and a 32-byte hash.
 *
 * **Never reuse a presignature** — signing two different messages with the
 * same presignature leaks the private key.
 *
 * @param presignature_json  JSON-serialised Presignature (from cggmp21_presign_result).
 * @param data_hash          32-byte pre-hashed digest (e.g. Keccak-256 or SHA-256 output).
 *                           Signed as-is mod curve order; the library does NOT re-hash.
 * @param out_json           Receives a heap-allocated JSON PartialSignature buffer.
 * @param out_len            Receives the buffer length.
 *
 * On success, free *out_json with cggmp21_free_buffer(*out_json, *out_len).
 *
 * @return CGGMP21_OK, CGGMP21_ERR_INVALID_ARG, or CGGMP21_ERR_JSON.
 */
int cggmp21_partial_sign(
    const uint8_t *presignature_json, size_t presignature_len,
    const uint8_t *data_hash,         size_t data_hash_len,
    uint8_t **out_json, size_t *out_len
);

/**
 * Combine a JSON array of PartialSignatures into a final ECDSA signature.
 *
 * @param partials_json_array  JSON array bytes: [partial_sig_1, partial_sig_2, ...].
 *                             Must contain >= threshold partials, all derived from
 *                             the same presignature run.
 * @param out_sig              Buffer of at least cggmp21_signature_len() bytes.
 * @param out_sig_len          Length of out_sig in bytes.
 *
 * The signature is written as r || s, big-endian. May produce an invalid
 * signature if some signer cheated — verify before trusting.
 *
 * @return CGGMP21_OK, CGGMP21_ERR_INVALID_ARG, CGGMP21_ERR_JSON,
 *         or CGGMP21_ERR_PROTOCOL (empty/malformed input).
 */
int cggmp21_combine_partials(
    const uint8_t *partials_json_array, size_t partials_len,
    uint8_t *out_sig, size_t out_sig_len
);

#ifdef __cplusplus
}
#endif

#endif /* CGGMP21_H */
