// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package cggmp21

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// loadFixtures returns the three pre-generated 2-of-3 key shares and the
// 33-byte compressed shared public key. Regenerate with:
//
//	cargo run --release --manifest-path rust/Cargo.toml --example gen_fixtures \
//	    -- tss/cggmp21/testdata
func loadFixtures(t *testing.T) (shares [3][]byte, pubkey []byte) {
	t.Helper()
	for i := 0; i < 3; i++ {
		path := filepath.Join("testdata", "share-"+string(rune('0'+i))+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("load fixture %s: %v (run gen_fixtures example to regenerate)", path, err)
		}
		shares[i] = b
	}
	pk, err := os.ReadFile(filepath.Join("testdata", "pubkey.bin"))
	if err != nil {
		t.Fatalf("load pubkey fixture: %v", err)
	}
	if len(pk) != 33 {
		t.Fatalf("pubkey fixture: want 33 bytes (compressed secp256k1), got %d", len(pk))
	}
	return shares, pk
}

// session is the common interface satisfied by both *SigningSession and
// *PresignSession — lets us share the message-routing loop.
type session interface {
	Deliver(sender uint16, broadcast bool, msg []byte) error
	NextOutgoing() (Message, bool, error)
	Done() bool
}

// runProtocol drives a set of sessions in lock-step, routing every outgoing
// message to its destination(s) until every session reports Done. Returns the
// total number of rounds it took, for the test log.
func runProtocol(t *testing.T, sessions []session) int {
	t.Helper()
	type pending struct {
		sender    uint16
		broadcast bool
		payload   []byte
	}
	inboxes := make([][]pending, len(sessions))

	const maxRounds = 50 // safety cap so a stuck test fails instead of hanging
	for round := 0; round < maxRounds; round++ {
		// Drain outgoing from each party.
		for i, sess := range sessions {
			for {
				msg, ok, err := sess.NextOutgoing()
				if err != nil {
					t.Fatalf("party %d NextOutgoing: %v", i, err)
				}
				if !ok {
					break
				}
				if msg.IsBroadcast() {
					for j := range sessions {
						if j == i {
							continue
						}
						inboxes[j] = append(inboxes[j], pending{
							sender:    uint16(i),
							broadcast: true,
							payload:   msg.Payload,
						})
					}
				} else {
					inboxes[msg.Recipient] = append(inboxes[msg.Recipient], pending{
						sender:    uint16(i),
						broadcast: false,
						payload:   msg.Payload,
					})
				}
			}
		}

		// All done?
		allDone := true
		for _, s := range sessions {
			if !s.Done() {
				allDone = false
				break
			}
		}
		if allDone {
			return round
		}

		// Detect deadlock — nothing to deliver, nobody done.
		anyPending := false
		for _, ib := range inboxes {
			if len(ib) > 0 {
				anyPending = true
				break
			}
		}
		if !anyPending {
			t.Fatalf("protocol stuck at round %d: no outgoing messages and no parties done", round)
		}

		// Deliver everything queued.
		for i, ib := range inboxes {
			for _, p := range ib {
				if err := sessions[i].Deliver(p.sender, p.broadcast, p.payload); err != nil {
					t.Fatalf("party %d Deliver: %v", i, err)
				}
			}
			inboxes[i] = nil
		}
	}
	t.Fatalf("protocol did not finish within %d rounds", maxRounds)
	return -1
}

// TestSigningSession_E2E runs the full 2-of-3 cggmp21 online signing protocol
// between two parties, in-process, and verifies the resulting ECDSA signature
// against the shared public key.
func TestSigningSession_E2E(t *testing.T) {
	shares, pubkey := loadFixtures(t)

	// Parties at keygen-index 0 and 1 cooperate; party with share-2 sits out.
	signers := []uint16{0, 1}
	eid := []byte("cggmp21-go-test-signing-eid-v1")
	hash := sha256.Sum256([]byte("hello, threshold signing"))

	s0, err := NewSigningSession(shares[0], eid, 0, signers, hash[:])
	if err != nil {
		t.Fatalf("party 0 NewSigningSession: %v", err)
	}
	defer s0.Close()
	s1, err := NewSigningSession(shares[1], eid, 1, signers, hash[:])
	if err != nil {
		t.Fatalf("party 1 NewSigningSession: %v", err)
	}
	defer s1.Close()

	rounds := runProtocol(t, []session{s0, s1})
	t.Logf("signing finished in %d rounds", rounds)

	sig, err := s0.Signature()
	if err != nil {
		t.Fatalf("Signature: %v", err)
	}
	if len(sig) != SignatureLen {
		t.Fatalf("signature length = %d, want %d", len(sig), SignatureLen)
	}

	// Both parties should yield the same signature.
	sig1, err := s1.Signature()
	if err != nil {
		t.Fatalf("party 1 Signature: %v", err)
	}
	if string(sig) != string(sig1) {
		t.Fatal("party 0 and party 1 produced different signatures")
	}

	if !crypto.VerifySignature(pubkey, hash[:], sig) {
		t.Fatalf("signature does not verify against shared pubkey\n  sig=%x\n  hash=%x\n  pubkey=%x", sig, hash[:], pubkey)
	}
}

// TestPresignPartialCombine_E2E exercises the one-round signing flow: run the
// presign MPC, locally compute partial signatures with PartialSign, combine
// them with Combine, then verify the final signature.
func TestPresignPartialCombine_E2E(t *testing.T) {
	shares, pubkey := loadFixtures(t)

	signers := []uint16{0, 1}
	eid := []byte("cggmp21-go-test-presign-eid-v1")

	p0, err := NewPresignSession(shares[0], eid, 0, signers)
	if err != nil {
		t.Fatalf("party 0 NewPresignSession: %v", err)
	}
	defer p0.Close()
	p1, err := NewPresignSession(shares[1], eid, 1, signers)
	if err != nil {
		t.Fatalf("party 1 NewPresignSession: %v", err)
	}
	defer p1.Close()

	rounds := runProtocol(t, []session{p0, p1})
	t.Logf("presign finished in %d rounds", rounds)

	presig0, err := p0.Presignature()
	if err != nil {
		t.Fatalf("party 0 Presignature: %v", err)
	}
	presig1, err := p1.Presignature()
	if err != nil {
		t.Fatalf("party 1 Presignature: %v", err)
	}

	// Sign a message that wasn't known at presign time.
	hash := sha256.Sum256([]byte("offline-signed message"))

	partial0, err := PartialSign(presig0, hash[:])
	if err != nil {
		t.Fatalf("PartialSign party 0: %v", err)
	}
	partial1, err := PartialSign(presig1, hash[:])
	if err != nil {
		t.Fatalf("PartialSign party 1: %v", err)
	}

	sig, err := Combine([][]byte{partial0, partial1})
	if err != nil {
		t.Fatalf("Combine: %v", err)
	}
	if len(sig) != SignatureLen {
		t.Fatalf("signature length = %d, want %d", len(sig), SignatureLen)
	}

	if !crypto.VerifySignature(pubkey, hash[:], sig) {
		t.Fatalf("combined signature does not verify against shared pubkey\n  sig=%x\n  hash=%x\n  pubkey=%x", sig, hash[:], pubkey)
	}
}

// TestPartialSign_RejectsReusedPresignature is informational: it documents the
// "never reuse a presignature" invariant. The library currently does not
// enforce this — reusing a presignature succeeds at the FFI level — so this
// test simply checks that PartialSign on the same presignature twice produces
// outputs (proving the leak path exists). The real protection is at the
// caller layer: discard a presignature after one use.
func TestPartialSign_DoesNotReusePresignature(t *testing.T) {
	shares, _ := loadFixtures(t)

	signers := []uint16{0, 1}
	eid := []byte("cggmp21-go-test-reuse-eid-v1")

	p0, err := NewPresignSession(shares[0], eid, 0, signers)
	if err != nil {
		t.Fatalf("party 0 NewPresignSession: %v", err)
	}
	defer p0.Close()
	p1, err := NewPresignSession(shares[1], eid, 1, signers)
	if err != nil {
		t.Fatalf("party 1 NewPresignSession: %v", err)
	}
	defer p1.Close()

	runProtocol(t, []session{p0, p1})

	presig, err := p0.Presignature()
	if err != nil {
		t.Fatalf("Presignature: %v", err)
	}

	h1 := sha256.Sum256([]byte("first message"))
	h2 := sha256.Sum256([]byte("second message"))
	if _, err := PartialSign(presig, h1[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := PartialSign(presig, h2[:]); err != nil {
		// Library does not enforce single-use — second call still returns a
		// partial. This is documented in the API; the caller must discard.
		t.Fatal(err)
	}
}
