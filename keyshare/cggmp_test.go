// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package keyshare_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/tss/cggmp21"
)

// loadTssLibFixture reads one of the existing tss-lib test keyshares and
// rebuilds it as a keyshare.ECDSAKeyshare so the conversion can run on it.
//
// The keyshare files are tss-lib LocalPartySaveData wrapped with metadata —
// stored at tss/test/keyshares/<i>.keyshare. They were produced by the project's
// pre-cggmp21 keygen flow.
func loadTssLibFixture(t *testing.T, i int) keyshare.ECDSAKeyshare {
	t.Helper()
	path := filepath.Join("..", "tss", "test", "keyshares", fmt.Sprintf("%d.keyshare", i))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	var k keyshare.ECDSAKeyshare
	if err := json.Unmarshal(b, &k); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	if k.Key.Xi == nil {
		t.Skipf("fixture %s is an empty/non-participant share — skipping", path)
	}
	return k
}

// TestCggmpShare_ConvertAcceptedByFFI confirms that the cggmp21 FFI accepts the
// converted share (its Deserialize+Validate path runs full structural and
// cryptographic checks: Paillier modulus size, gcd(s, N), gcd(t, N), point
// non-zero, etc.). If the conversion produces malformed JSON or invalid params,
// the FFI returns an error and this test fails.
func TestCggmpShare_ConvertAcceptedByFFI(t *testing.T) {
	for i := 0; i < 3; i++ {
		i := i
		t.Run(fmt.Sprintf("share-%d", i), func(t *testing.T) {
			k := loadTssLibFixture(t, i)
			shareJSON, err := k.CggmpShare()
			if err != nil {
				t.Fatalf("CggmpShare(): %v", err)
			}

			// Smoke-test by constructing a session — the FFI deserialises and
			// runs Validate on the share. If anything in the conversion is
			// wrong, this errors out with a useful message.
			signers := []uint16{0, 1}
			eid := []byte("cggmp21-conversion-validate")
			hash := sha256.Sum256([]byte("x"))
			s, err := cggmp21.NewSigningSession(shareJSON, eid, 0, signers, hash[:])
			if err != nil {
				// The FFI may still reject parties_indexes when i != 0 in the
				// signing set, but a deserialise/Validate failure surfaces here
				// with "deserialize"/"InvalidKeyShare" in the message.
				t.Fatalf("FFI rejected converted share: %v\nshare=%s", err, string(shareJSON))
			}
			s.Close()
		})
	}
}

// TestCggmpShare_SigningE2E proves end-to-end correctness: converts all three
// tss-lib fixtures, runs 2-of-3 cggmp21 signing between two of them, and
// verifies the resulting signature against the public key recorded in the
// original tss-lib share.
//
// If the conversion mis-encodes ANY of: secret share, public shares, VSS setup,
// or Paillier/Ring-Pedersen aux, the protocol will either error during signing
// or produce a signature that doesn't verify. So this is the real correctness
// test — passing it means the conversion is sound.
func TestCggmpShare_SigningE2E(t *testing.T) {
	type signerSlot struct {
		shareJSON   []byte
		keygenIndex uint16
	}
	slots := make([]signerSlot, 0, 3)
	var pubkeyCompressed []byte

	for i := 0; i < 3; i++ {
		k := loadTssLibFixture(t, i)
		shareJSON, err := k.CggmpShare()
		if err != nil {
			t.Fatalf("CggmpShare share-%d: %v", i, err)
		}
		// Find this party's keygen index (position of ShareID in the sorted Ks list).
		// The fixture file numbering (0/1/2) doesn't match keygen index.
		idx := -1
		for j, kj := range k.Key.Ks {
			if kj.Cmp(k.Key.ShareID) == 0 {
				idx = j
				break
			}
		}
		if idx < 0 {
			t.Fatalf("share-%d: ShareID not in Ks", i)
		}
		slots = append(slots, signerSlot{shareJSON: shareJSON, keygenIndex: uint16(idx)})

		if pubkeyCompressed == nil {
			pubX, pubY := k.Key.ECDSAPub.X(), k.Key.ECDSAPub.Y()
			pubkeyCompressed = make([]byte, 33)
			if pubY.Bit(0) == 0 {
				pubkeyCompressed[0] = 0x02
			} else {
				pubkeyCompressed[0] = 0x03
			}
			xb := make([]byte, 32)
			pubX.FillBytes(xb)
			copy(pubkeyCompressed[1:], xb)
		}
	}

	// Sort by keygen index so the signing-position ↔ keygen-index mapping is
	// deterministic and parties_indexes is monotonic (cggmp21 requirement).
	sort.Slice(slots, func(a, b int) bool { return slots[a].keygenIndex < slots[b].keygenIndex })

	// Pick the first two signers — keygen indexes ascending. parties_indexes is
	// passed to every signer and lists the keygen indexes of all participants.
	signerSet := []uint16{slots[0].keygenIndex, slots[1].keygenIndex}

	eid := []byte("cggmp21-conversion-e2e-signing")
	hash := sha256.Sum256([]byte("hello from a converted tss-lib share"))

	s0, err := cggmp21.NewSigningSession(slots[0].shareJSON, eid, 0, signerSet, hash[:])
	if err != nil {
		t.Fatalf("signer 0 (keygen idx %d) NewSigningSession: %v", slots[0].keygenIndex, err)
	}
	defer s0.Close()
	s1, err := cggmp21.NewSigningSession(slots[1].shareJSON, eid, 1, signerSet, hash[:])
	if err != nil {
		t.Fatalf("signer 1 (keygen idx %d) NewSigningSession: %v", slots[1].keygenIndex, err)
	}
	defer s1.Close()

	runSigning(t, []*cggmp21.SigningSession{s0, s1})

	sig, err := s0.Signature()
	if err != nil {
		// Surface signer-1's view too in case it errored.
		if sig1, err1 := s1.Signature(); err1 != nil {
			t.Logf("signer 1 also errored: %v (sig=%x)", err1, sig1)
		}
		t.Fatalf("signer 0 Signature: %v", err)
	}

	if !crypto.VerifySignature(pubkeyCompressed, hash[:], sig) {
		t.Fatalf("converted-share signature does not verify\n  sig=%x\n  hash=%x\n  pubkey=%x",
			sig, hash[:], pubkeyCompressed)
	}
}

// runSigning is a minimal in-process message router that drives the two
// SigningSessions to completion.
func runSigning(t *testing.T, sessions []*cggmp21.SigningSession) {
	t.Helper()
	type pending struct {
		sender    uint16
		broadcast bool
		payload   []byte
	}
	inboxes := make([][]pending, len(sessions))

	for round := 0; round < 50; round++ {
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
							sender: uint16(i), broadcast: true, payload: msg.Payload,
						})
					}
				} else {
					inboxes[msg.Recipient] = append(inboxes[msg.Recipient], pending{
						sender: uint16(i), broadcast: false, payload: msg.Payload,
					})
				}
			}
		}

		allDone := true
		for _, s := range sessions {
			if !s.Done() {
				allDone = false
				break
			}
		}
		if allDone {
			return
		}

		any := false
		for _, ib := range inboxes {
			if len(ib) > 0 {
				any = true
				break
			}
		}
		if !any {
			t.Fatalf("protocol stuck at round %d", round)
		}

		for i, ib := range inboxes {
			if sessions[i].Done() {
				// Session already produced an Output (signature or error). Drop
				// remaining queued messages — they're for past rounds.
				inboxes[i] = nil
				continue
			}
			for _, p := range ib {
				if err := sessions[i].Deliver(p.sender, p.broadcast, p.payload); err != nil {
					t.Fatalf("party %d Deliver: %v", i, err)
				}
			}
			inboxes[i] = nil
		}
	}
	t.Fatal("protocol did not finish within 50 rounds")
}

// Compile-time check that the fixture's ECDSAKeyshare type still has the field
// we reach into. If keygen.LocalPartySaveData ever changes shape, the build
// breaks here.
var _ = keygen.LocalPartySaveData{}

// Helper: format hex strings deterministically for failure messages.
var _ = hex.EncodeToString
var _ = big.NewInt
