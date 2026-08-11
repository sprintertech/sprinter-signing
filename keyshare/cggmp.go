// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package keyshare

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

// CggmpShare returns this key share serialised as a cggmp21 KeyShare<Secp256k1>
// JSON value, suitable for passing to tss/cggmp21.NewSigningSession (or any of
// the cggmp21 FFI entry points).
//
// The conversion is a re-encoding of the mathematical state stored by tss-lib
// (a.k.a. ChainSafe/threshlib) into cggmp21's serde JSON shape. Both libraries
// implement GG18/CGGMP-style threshold ECDSA on secp256k1 with the same
// per-party data:
//
//   - Secret share x_i        ← LocalSecrets.Xi
//   - Public shares X_j       ← LocalPartySaveData.BigXj
//   - Shared public key       ← LocalPartySaveData.ECDSAPub
//   - VSS evaluation points   ← LocalPartySaveData.Ks
//   - Paillier modulus N      ← LocalPartySaveData.NTildej (see note below)
//   - Paillier secret primes  ← 2·LocalPreParams.P+1, 2·LocalPreParams.Q+1
//   - Ring-Pedersen (s, t)    ← LocalPartySaveData.{H1j, H2j}
//
// Note on the Paillier modulus: cggmp21 requires the Paillier modulus and the
// Ring-Pedersen modulus to be the SAME number (`aux.p * aux.q == aux.parties[i].N`).
// tss-lib/threshlib generates them as TWO separate moduli (PaillierSK.N and
// NTildej[i]). We discard the original Paillier modulus and use NTildej as the
// unified modulus for cggmp21; `aux.{p, q}` are the safe primes that factor
// NTildei (recovered from LocalPreParams.{P, Q}, which threshlib stores as
// Sophie-Germain primes: SafePrime = 2·p_sg + 1).
//
// This substitution is sound because the cryptographic identity of the share is
// determined by the SECRET share x_i and the SHARED PUBLIC KEY — both are
// preserved exactly. The Paillier setup is fresh signing-protocol state;
// signatures produced from converted shares verify against the original
// ECDSAPub.
//
// Returns an error if the source share is missing required fields.
func (k ECDSAKeyshare) CggmpShare() ([]byte, error) {
	pre := k.Key.LocalPreParams
	if !pre.ValidateWithProof() {
		return nil, errors.New("keyshare: incomplete pre-params (missing Paillier SK, NTildei, H1/H2, or Sophie-Germain P/Q)")
	}
	if k.Key.Xi == nil || k.Key.ShareID == nil || k.Key.ECDSAPub == nil {
		return nil, errors.New("keyshare: missing core secrets (Xi/ShareID/ECDSAPub)")
	}

	n := len(k.Key.Ks)
	if n == 0 || len(k.Key.BigXj) != n || len(k.Key.NTildej) != n ||
		len(k.Key.H1j) != n || len(k.Key.H2j) != n {
		return nil, fmt.Errorf("keyshare: party arrays have inconsistent lengths (Ks=%d, BigXj=%d, NTildej=%d)",
			n, len(k.Key.BigXj), len(k.Key.NTildej))
	}

	myIndex := -1
	for j, kj := range k.Key.Ks {
		if kj != nil && kj.Cmp(k.Key.ShareID) == 0 {
			myIndex = j
			break
		}
	}
	if myIndex < 0 {
		return nil, errors.New("keyshare: this party's ShareID not found in Ks")
	}

	// Recover the safe primes that factor NTildei from threshlib's stored
	// Sophie-Germain primes (P, Q). By construction: NTildei = (2P+1)·(2Q+1).
	one := big.NewInt(1)
	two := big.NewInt(2)
	safeP := new(big.Int).Add(new(big.Int).Mul(two, k.Key.P), one)
	safeQ := new(big.Int).Add(new(big.Int).Mul(two, k.Key.Q), one)

	// Sanity check — the safe primes must factor NTildei, the modulus we'll be
	// using as cggmp21's unified Paillier+Ring-Pedersen modulus.
	if k.Key.NTildei == nil {
		return nil, errors.New("keyshare: missing NTildei")
	}
	if new(big.Int).Mul(safeP, safeQ).Cmp(k.Key.NTildei) != 0 {
		return nil, errors.New("keyshare: safe-prime recovery failed (2P+1)(2Q+1) ≠ NTildei — share has unexpected pre-params layout")
	}

	// Build the public-shares list.
	publicShares := make([]string, n)
	for j, bigX := range k.Key.BigXj {
		if bigX == nil {
			return nil, fmt.Errorf("keyshare: BigXj[%d] is nil", j)
		}
		hexPt, err := compressPoint(bigX.X(), bigX.Y())
		if err != nil {
			return nil, fmt.Errorf("keyshare: BigXj[%d]: %w", j, err)
		}
		publicShares[j] = hexPt
	}

	sharedPubHex, err := compressPoint(k.Key.ECDSAPub.X(), k.Key.ECDSAPub.Y())
	if err != nil {
		return nil, fmt.Errorf("keyshare: ECDSAPub: %w", err)
	}

	// VSS evaluation points (Ks). cggmp21 stores them as 32-byte hex scalars.
	iList := make([]string, n)
	for j, ks := range k.Key.Ks {
		if ks == nil {
			return nil, fmt.Errorf("keyshare: Ks[%d] is nil", j)
		}
		iList[j] = scalarHex(ks)
	}

	// Per-party aux info.
	// Ring-Pedersen mapping: tss-lib computes H2 = H1^alpha (alpha secret),
	// cggmp21 stores s = t^lambda (lambda secret) — so tss-lib's H1 plays the
	// role of cggmp21's t (generator), and H2 plays the role of s (= t^lambda).
	parties := make([]cggmpPartyAux, n)
	for j := 0; j < n; j++ {
		parties[j] = cggmpPartyAux{
			N: bigInt16(k.Key.NTildej[j]),
			S: bigInt16(k.Key.H2j[j]),
			T: bigInt16(k.Key.H1j[j]),
		}
	}

	// threshlib's signing threshold is `t` where t+1 parties cooperate.
	// cggmp21's min_signers is the count of required signers, i.e. t+1.
	minSigners := uint16(k.Threshold + 1)

	share := cggmpKeyShareJSON{
		Core: cggmpCoreJSON{
			Curve:           "secp256k1",
			I:               uint16(myIndex),
			SharedPublicKey: sharedPubHex,
			PublicShares:    publicShares,
			VssSetup: &cggmpVssJSON{
				MinSigners: minSigners,
				I:          iList,
			},
			X: scalarHex(k.Key.Xi),
		},
		Aux: cggmpAuxJSON{
			P:       bigInt16(safeP),
			Q:       bigInt16(safeQ),
			Parties: parties,
		},
	}
	return json.Marshal(share)
}

// ── cggmp21 KeyShare<Secp256k1> JSON shape (matches the serde output) ────────

type cggmpKeyShareJSON struct {
	Core cggmpCoreJSON `json:"core"`
	Aux  cggmpAuxJSON  `json:"aux"`
}

type cggmpCoreJSON struct {
	Curve           string        `json:"curve"`
	I               uint16        `json:"i"`
	SharedPublicKey string        `json:"shared_public_key"`
	PublicShares    []string      `json:"public_shares"`
	VssSetup        *cggmpVssJSON `json:"vss_setup,omitempty"`
	X               string        `json:"x"`
}

type cggmpVssJSON struct {
	MinSigners uint16   `json:"min_signers"`
	I          []string `json:"I"`
}

type cggmpAuxJSON struct {
	P       cggmpBigInt     `json:"p"`
	Q       cggmpBigInt     `json:"q"`
	Parties []cggmpPartyAux `json:"parties"`
}

type cggmpBigInt struct {
	Radix int    `json:"radix"`
	Value string `json:"value"`
}

type cggmpPartyAux struct {
	N cggmpBigInt `json:"N"`
	S cggmpBigInt `json:"s"`
	T cggmpBigInt `json:"t"`
}

// ── Encoding helpers ─────────────────────────────────────────────────────────

// secp256k1 curve order N (= 2^256 - 0x14551231950b75fc4402da1732fc9bebf).
// Used to reduce scalars before encoding.
var secp256k1N, _ = new(big.Int).SetString(
	"fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)

// scalarHex returns x mod N as a fixed 32-byte big-endian lowercase hex string.
func scalarHex(x *big.Int) string {
	r := new(big.Int).Mod(x, secp256k1N)
	buf := make([]byte, 32)
	r.FillBytes(buf)
	return hex.EncodeToString(buf)
}

// compressPoint returns the 33-byte SEC1 compressed encoding of (x, y) as hex.
// Parity prefix: 0x02 if y is even, 0x03 if y is odd.
func compressPoint(x, y *big.Int) (string, error) {
	if x == nil || y == nil {
		return "", errors.New("nil point coordinate")
	}
	xb := make([]byte, 32)
	x.FillBytes(xb)
	prefix := byte(0x02)
	if y.Bit(0) == 1 {
		prefix = 0x03
	}
	out := make([]byte, 33)
	out[0] = prefix
	copy(out[1:], xb)
	return hex.EncodeToString(out), nil
}

// bigInt16 wraps a non-negative big.Int as cggmp21's {"radix":16, "value":"<hex>"}
// JSON shape used for Paillier moduli and Ring-Pedersen parameters.
func bigInt16(x *big.Int) cggmpBigInt {
	if x == nil {
		return cggmpBigInt{Radix: 16, Value: "0"}
	}
	// rug::Integer JSON omits the leading "0x" and uses lowercase.
	return cggmpBigInt{Radix: 16, Value: x.Text(16)}
}
