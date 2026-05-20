// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

// Package cggmp21 is a Go wrapper around the cggmp21-ffi static library
// (rust/cggmp21-ffi). It exposes the threshold-ECDSA signing flow as two
// session types — SigningSession (one-shot online signing) and PresignSession
// (offline presignature generation) — plus the local PartialSign and Combine
// helpers that implement the one-round signing variant.
//
// The Rust static library at rust/lib/libcggmp21_ffi.a must be built first:
//
//	make build-ffi
package cggmp21

/*
#cgo CFLAGS: -I${SRCDIR}/../../rust/include
#cgo LDFLAGS: -L${SRCDIR}/../../rust/lib -lcggmp21_ffi -lm -ldl -lpthread
#include <stdlib.h>
#include "cggmp21.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// Message is a single protocol message produced or consumed by a session.
// Recipient == -1 means broadcast; otherwise it is the destination signer's
// 0-based index.
type Message struct {
	Recipient int32
	Payload   []byte
}

// IsBroadcast reports whether the message is destined for all parties.
func (m Message) IsBroadcast() bool { return m.Recipient == -1 }

// SignatureLen is the byte length of a serialised secp256k1 signature (r || s).
var SignatureLen = int(C.cggmp21_signature_len())

// Error is returned by FFI calls that fail with a CGGMP21_ERR_* status. The
// Code field exposes the raw status so callers can branch on specific failure
// modes (e.g. "not finished").
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("cggmp21 (rc=%d): %s", e.Code, e.Message)
}

// cgoCall invokes a C function that returns a CGGMP21_* status code. On a
// non-OK return it wraps cggmp21_last_error() into an *Error. LockOSThread
// keeps the goroutine pinned across the call+error read so the thread-local
// error storage stays consistent.
func cgoCall(call func() C.int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	rc := call()
	if rc == C.CGGMP21_OK {
		return nil
	}
	return &Error{
		Code:    int(rc),
		Message: C.GoString(C.cggmp21_last_error()),
	}
}

// isNotFinished reports whether err signals that the protocol has not yet
// produced a result.
func isNotFinished(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == int(C.CGGMP21_ERR_NOT_FINISHED)
}

// bytePtr returns a *C.uint8_t for the given Go slice. For empty slices it
// returns nil so we never deref a zero-length slice.
func bytePtr(b []byte) *C.uint8_t {
	if len(b) == 0 {
		return nil
	}
	return (*C.uint8_t)(unsafe.Pointer(&b[0]))
}

// uint16Ptr returns a *C.uint16_t for the given slice, or nil if empty.
func uint16Ptr(s []uint16) *C.uint16_t {
	if len(s) == 0 {
		return nil
	}
	return (*C.uint16_t)(unsafe.Pointer(&s[0]))
}

// copyAndFree turns an FFI-allocated (out_json, out_len) pair into a Go []byte
// and releases the C buffer.
func copyAndFree(ptr *C.uint8_t, length C.size_t) []byte {
	if ptr == nil || length == 0 {
		return nil
	}
	out := C.GoBytes(unsafe.Pointer(ptr), C.int(length))
	C.cggmp21_free_buffer(ptr, length)
	return out
}

// ErrNotFinished is returned when a result accessor is called before the
// protocol has completed.
var ErrNotFinished = errors.New("cggmp21: protocol not finished")
