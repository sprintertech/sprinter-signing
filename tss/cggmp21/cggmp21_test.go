// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cggmp21

import (
	"errors"
	"strings"
	"testing"
)

// ── Constants ────────────────────────────────────────────────────────────────

func TestSignatureLen(t *testing.T) {
	if SignatureLen != 64 {
		t.Fatalf("SignatureLen = %d, want 64 (secp256k1 r||s)", SignatureLen)
	}
}

// ── Message ─────────────────────────────────────────────────────────────────

func TestMessage_IsBroadcast(t *testing.T) {
	for _, tc := range []struct {
		name      string
		recipient int32
		want      bool
	}{
		{"broadcast", -1, true},
		{"party 0", 0, false},
		{"party 5", 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := Message{Recipient: tc.recipient}
			if got := m.IsBroadcast(); got != tc.want {
				t.Fatalf("Recipient=%d → IsBroadcast()=%v, want %v", tc.recipient, got, tc.want)
			}
		})
	}
}

// ── Error type ──────────────────────────────────────────────────────────────

func TestError_FormatAndUnwrap(t *testing.T) {
	e := &Error{Code: 2, Message: "boom"}
	if msg := e.Error(); !strings.Contains(msg, "rc=2") || !strings.Contains(msg, "boom") {
		t.Fatalf("Error string %q missing expected fields", msg)
	}

	var target *Error
	if !errors.As(e, &target) {
		t.Fatal("errors.As did not unwrap *Error")
	}
	if target.Code != 2 {
		t.Fatalf("unwrapped Code = %d, want 2", target.Code)
	}
}

// ── Input validation ────────────────────────────────────────────────────────

func TestNewSigningSession_RejectsBadInputs(t *testing.T) {
	for _, tc := range []struct {
		name           string
		keyShare       []byte
		eid            []byte
		i              uint16
		parties        []uint16
		dataHash       []byte
		wantSubstring  string
		wantErrorClass string // "go" for Go-side check, "ffi" for FFI error code
	}{
		{
			name:           "empty parties",
			keyShare:       []byte("{}"),
			eid:            []byte("eid"),
			parties:        nil,
			dataHash:       make([]byte, 32),
			wantSubstring:  "partiesIndexes",
			wantErrorClass: "go",
		},
		{
			name:           "wrong-size hash",
			keyShare:       []byte("{}"),
			eid:            []byte("eid"),
			parties:        []uint16{0},
			dataHash:       make([]byte, 31),
			wantSubstring:  "32 bytes",
			wantErrorClass: "go",
		},
		{
			name:           "malformed keyshare JSON",
			keyShare:       []byte("not json"),
			eid:            []byte("eid"),
			parties:        []uint16{0, 1},
			dataHash:       make([]byte, 32),
			wantSubstring:  "deserialize",
			wantErrorClass: "ffi",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSigningSession(tc.keyShare, tc.eid, tc.i, tc.parties, tc.dataHash)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSubstring) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSubstring)
			}
			var ffiErr *Error
			isFFI := errors.As(err, &ffiErr)
			if tc.wantErrorClass == "ffi" && !isFFI {
				t.Fatalf("expected *Error, got %T", err)
			}
			if tc.wantErrorClass == "go" && isFFI {
				t.Fatalf("expected Go-side error, got *Error: %v", ffiErr)
			}
		})
	}
}

func TestNewPresignSession_RejectsBadInputs(t *testing.T) {
	if _, err := NewPresignSession([]byte("{}"), []byte("eid"), 0, nil); err == nil {
		t.Fatal("expected error for empty partiesIndexes")
	}
	if _, err := NewPresignSession([]byte("not json"), []byte("eid"), 0, []uint16{0, 1}); err == nil {
		t.Fatal("expected error for malformed keyshare")
	}
}

func TestPartialSign_RejectsBadInputs(t *testing.T) {
	for _, tc := range []struct {
		name         string
		presig       []byte
		hash         []byte
		wantContains string
	}{
		{"empty presig", nil, make([]byte, 32), "presignature"},
		{"short hash", []byte("{}"), make([]byte, 16), "32 bytes"},
		{"malformed presig", []byte("garbage"), make([]byte, 32), "deserialize"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PartialSign(tc.presig, tc.hash)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantContains)
			}
		})
	}
}

func TestCombine_RejectsBadInputs(t *testing.T) {
	if _, err := Combine(nil); err == nil {
		t.Fatal("expected error for nil partials")
	}
	if _, err := Combine([][]byte{}); err == nil {
		t.Fatal("expected error for empty partials slice")
	}
	if _, err := Combine([][]byte{[]byte("not json")}); err == nil {
		t.Fatal("expected error for malformed partial")
	}
}

// ── Session lifecycle (no live session — methods on a nil-ptr session) ─────

func TestSigningSession_MethodsAfterClose(t *testing.T) {
	// Build a session shell with a nil ptr — equivalent to a closed session.
	s := &SigningSession{}

	if err := s.Deliver(0, false, []byte("x")); err == nil {
		t.Error("Deliver should error on closed session")
	}
	if _, _, err := s.NextOutgoing(); err == nil {
		t.Error("NextOutgoing should error on closed session")
	}
	if s.Done() {
		t.Error("Done should be false on closed session")
	}
	if _, err := s.Signature(); err == nil {
		t.Error("Signature should error on closed session")
	}
	// Close on already-closed must be a no-op.
	s.Close()
	s.Close()
}

func TestPresignSession_MethodsAfterClose(t *testing.T) {
	s := &PresignSession{}

	if err := s.Deliver(0, false, []byte("x")); err == nil {
		t.Error("Deliver should error on closed session")
	}
	if _, _, err := s.NextOutgoing(); err == nil {
		t.Error("NextOutgoing should error on closed session")
	}
	if s.Done() {
		t.Error("Done should be false on closed session")
	}
	if _, err := s.Presignature(); err == nil {
		t.Error("Presignature should error on closed session")
	}
	s.Close()
	s.Close()
}
