// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cggmp21

/*
#include "cggmp21.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"runtime"
	"unsafe"
)

// PresignSession drives the cggmp21 offline presignature-generation protocol
// (the message-independent half of the one-round signing flow). Drive it the
// same way as SigningSession and fetch the JSON-encoded Presignature with
// Presignature once Done reports true.
type PresignSession struct {
	ptr *C.CggmpPresignSession
}

// NewPresignSession starts a presignature-generation session. Inputs match
// NewSigningSession minus the message hash — presignatures are
// message-independent.
func NewPresignSession(keyShare, eid []byte, i uint16, partiesIndexes []uint16) (*PresignSession, error) {
	if len(partiesIndexes) == 0 {
		return nil, errors.New("cggmp21: partiesIndexes must not be empty")
	}

	var ptr *C.CggmpPresignSession
	err := cgoCall(func() C.int {
		return C.cggmp21_presign_new(
			bytePtr(keyShare), C.size_t(len(keyShare)),
			bytePtr(eid), C.size_t(len(eid)),
			C.uint16_t(i),
			uint16Ptr(partiesIndexes), C.size_t(len(partiesIndexes)),
			&ptr,
		)
	})
	if err != nil {
		return nil, err
	}

	s := &PresignSession{ptr: ptr}
	runtime.SetFinalizer(s, (*PresignSession).finalize)
	return s, nil
}

// Deliver hands an incoming protocol message to the state machine.
func (s *PresignSession) Deliver(sender uint16, broadcast bool, msg []byte) error {
	if s.ptr == nil {
		return errors.New("cggmp21: presign session closed")
	}
	var b C.uint8_t
	if broadcast {
		b = 1
	}
	return cgoCall(func() C.int {
		return C.cggmp21_presign_deliver(
			s.ptr,
			C.uint16_t(sender),
			b,
			bytePtr(msg), C.size_t(len(msg)),
		)
	})
}

// NextOutgoing pops the next queued outgoing message.
func (s *PresignSession) NextOutgoing() (Message, bool, error) {
	if s.ptr == nil {
		return Message{}, false, errors.New("cggmp21: presign session closed")
	}
	var (
		recipient C.int32_t
		buf       *C.uint8_t
		length    C.size_t
	)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	rc := C.cggmp21_presign_poll_outgoing(s.ptr, &recipient, &buf, &length)
	if rc == 0 {
		return Message{}, false, nil
	}
	return Message{
		Recipient: int32(recipient),
		Payload:   copyAndFree(buf, length),
	}, true, nil
}

// Done reports whether the presignature protocol has finished.
func (s *PresignSession) Done() bool {
	if s.ptr == nil {
		return false
	}
	return C.cggmp21_presign_is_done(s.ptr) != 0
}

// Presignature returns the JSON-serialised Presignature once Done is true.
// Pass this to PartialSign together with the message hash to compute a
// PartialSignature. **Never reuse a presignature** — doing so leaks the
// private key.
func (s *PresignSession) Presignature() ([]byte, error) {
	if s.ptr == nil {
		return nil, errors.New("cggmp21: presign session closed")
	}
	var (
		buf    *C.uint8_t
		length C.size_t
	)
	err := cgoCall(func() C.int {
		return C.cggmp21_presign_result(s.ptr, &buf, &length)
	})
	if err != nil {
		if isNotFinished(err) {
			return nil, ErrNotFinished
		}
		return nil, err
	}
	return copyAndFree(buf, length), nil
}

// Close releases the C-side session. Safe to call multiple times.
func (s *PresignSession) Close() {
	runtime.SetFinalizer(s, nil)
	s.free()
}

func (s *PresignSession) finalize() {
	s.free()
}

func (s *PresignSession) free() {
	if s.ptr == nil {
		return
	}
	C.cggmp21_presign_free(s.ptr)
	s.ptr = nil
}

// PartialSign locally computes a PartialSignature from a presignature and a
// 32-byte message hash. The returned bytes are JSON-serialised and meant to be
// gathered (one per signer) and passed to Combine.
//
// **Never reuse a presignature.**
func PartialSign(presignature, dataHash []byte) ([]byte, error) {
	if len(dataHash) != 32 {
		return nil, errors.New("cggmp21: dataHash must be 32 bytes")
	}
	if len(presignature) == 0 {
		return nil, errors.New("cggmp21: presignature must not be empty")
	}

	var (
		buf    *C.uint8_t
		length C.size_t
	)
	err := cgoCall(func() C.int {
		return C.cggmp21_partial_sign(
			bytePtr(presignature), C.size_t(len(presignature)),
			bytePtr(dataHash), C.size_t(len(dataHash)),
			&buf, &length,
		)
	})
	if err != nil {
		return nil, err
	}
	return copyAndFree(buf, length), nil
}

// Combine combines a threshold-sized set of PartialSignatures (each JSON-
// serialised, as returned by PartialSign) into a final ECDSA signature
// (r || s, SignatureLen bytes).
//
// The returned signature may still be invalid for the public key and message
// if some signer cheated; callers should verify it before trusting.
func Combine(partials [][]byte) ([]byte, error) {
	if len(partials) == 0 {
		return nil, errors.New("cggmp21: at least one partial signature required")
	}

	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, p := range partials {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(p)
	}
	buf.WriteByte(']')
	arr := buf.Bytes()

	out := make([]byte, SignatureLen)
	err := cgoCall(func() C.int {
		return C.cggmp21_combine_partials(
			bytePtr(arr), C.size_t(len(arr)),
			(*C.uint8_t)(unsafe.Pointer(&out[0])), C.size_t(len(out)),
		)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
