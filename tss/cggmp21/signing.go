// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cggmp21

/*
#include "cggmp21.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// SigningSession drives the cggmp21 online ECDSA signing protocol for one
// party. Construct it with NewSigningSession, drive it by alternating Deliver
// and NextOutgoing until Done reports true, then fetch the result with
// Signature.
//
// A session owns C-side resources; call Close (or rely on the finalizer, which
// is best-effort) to release them. The session is not safe for concurrent use
// from multiple goroutines.
type SigningSession struct {
	mu  sync.Mutex
	ptr *C.CggmpSigningSession
}

// NewSigningSession starts a signing session for this party.
//
// Parameters:
//   - keyShare: JSON-serialised KeyShare<Secp256k1>.
//   - eid: execution ID bytes (must be unique per protocol run).
//   - i: this party's signing index (0-based, must be < len(partiesIndexes)).
//   - partiesIndexes: partiesIndexes[j] is the keygen index of the j-th signer.
//   - dataHash: 32-byte big-endian message hash (e.g. Keccak-256 output).
func NewSigningSession(keyShare, eid []byte, i uint16, partiesIndexes []uint16, dataHash []byte) (*SigningSession, error) {
	if len(dataHash) != 32 {
		return nil, errors.New("cggmp21: dataHash must be 32 bytes")
	}
	if len(partiesIndexes) == 0 {
		return nil, errors.New("cggmp21: partiesIndexes must not be empty")
	}

	var ptr *C.CggmpSigningSession
	err := cgoCall(func() C.int {
		return C.cggmp21_signing_new(
			bytePtr(keyShare), C.size_t(len(keyShare)),
			bytePtr(eid), C.size_t(len(eid)),
			C.uint16_t(i),
			uint16Ptr(partiesIndexes), C.size_t(len(partiesIndexes)),
			bytePtr(dataHash), C.size_t(len(dataHash)),
			&ptr,
		)
	})
	if err != nil {
		return nil, err
	}

	s := &SigningSession{ptr: ptr}
	runtime.SetFinalizer(s, (*SigningSession).finalize)
	return s, nil
}

// Deliver hands an incoming protocol message to the state machine. Drain any
// queued outgoing messages with NextOutgoing after each Deliver.
func (s *SigningSession) Deliver(sender uint16, broadcast bool, msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return errors.New("cggmp21: signing session closed")
	}
	var b C.uint8_t
	if broadcast {
		b = 1
	}
	return cgoCall(func() C.int {
		return C.cggmp21_signing_deliver(
			s.ptr,
			C.uint16_t(sender),
			b,
			bytePtr(msg), C.size_t(len(msg)),
		)
	})
}

// NextOutgoing pops the next queued outgoing message. The boolean is false
// when the queue is drained.
func (s *SigningSession) NextOutgoing() (Message, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return Message{}, false, errors.New("cggmp21: signing session closed")
	}
	var (
		recipient C.int32_t
		buf       *C.uint8_t
		length    C.size_t
	)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	rc := C.cggmp21_signing_poll_outgoing(s.ptr, &recipient, &buf, &length)
	if rc == 0 {
		return Message{}, false, nil
	}
	return Message{
		Recipient: int32(recipient),
		Payload:   copyAndFree(buf, length),
	}, true, nil
}

// Done reports whether the protocol has finished (successfully or with an
// error). Once Done returns true, call Signature to fetch the result.
func (s *SigningSession) Done() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return false
	}
	return C.cggmp21_signing_is_done(s.ptr) != 0
}

// Signature returns the final ECDSA signature as r || s (big-endian,
// SignatureLen bytes). Returns ErrNotFinished if Done has not yet reported true.
func (s *SigningSession) Signature() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return nil, errors.New("cggmp21: signing session closed")
	}
	out := make([]byte, SignatureLen)
	err := cgoCall(func() C.int {
		return C.cggmp21_signing_result(s.ptr, (*C.uint8_t)(unsafe.Pointer(&out[0])), C.size_t(len(out)))
	})
	if err != nil {
		// Distinguish "not finished" from other errors for callers that want
		// to retry until the protocol completes.
		if isNotFinished(err) {
			return nil, ErrNotFinished
		}
		return nil, err
	}
	return out, nil
}

// Close releases the C-side session. Safe to call multiple times.
func (s *SigningSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	runtime.SetFinalizer(s, nil)
	s.freeLocked()
}

func (s *SigningSession) finalize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.freeLocked()
}

func (s *SigningSession) freeLocked() {
	if s.ptr == nil {
		return
	}
	C.cggmp21_signing_free(s.ptr)
	s.ptr = nil
}
