package lifi_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sygmaprotocol/sygma-core/crypto/secp256k1"
)

func TestVerifyCompactSignature(t *testing.T) {
	// Sample valid BatchCompact and signature, replace with actual values
	validKp, _ := secp256k1.GenerateKeypair()
	validSponsor := common.HexToAddress(validKp.Address())

	invalidKp, _ := secp256k1.GenerateKeypair()

	validBatchCompact := lifi.BatchCompact{
		Arbiter: common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
		Sponsor: validSponsor,
		Nonce:   big.NewInt(123),
		Expires: big.NewInt(1866230400),
		Commitments: []lifi.Lock{
			{
				LockTag: [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				Token:   common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Amount:  big.NewInt(1000),
			},
		},
	}
	digest, err := lifi.GenerateCompactDigest(big.NewInt(1), common.Address{}, validBatchCompact)
	if err != nil {
		t.Fatalf("invalid digest")
	}
	validSignature, err := validKp.Sign(digest)
	if err != nil {
		t.Fatalf("invalid sig")
	}
	invalidSignature, err := invalidKp.Sign(digest)
	if err != nil {
		t.Fatalf("invalid sig")
	}

	tests := []struct {
		name        string
		compact     lifi.BatchCompact
		signature   []byte
		expectValid bool
		expectErr   bool
	}{
		{
			name:        "Valid signature",
			compact:     validBatchCompact,
			signature:   validSignature,
			expectValid: true,
			expectErr:   false,
		},
		{
			name:    "Invalid signature (wrong signer)",
			compact: validBatchCompact,
			// Tampered or made-up signature
			signature:   invalidSignature,
			expectValid: false,
			expectErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			valid, err := lifi.VerifyCompactSignature(digest, tc.signature, validSponsor)
			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectErr, err)
			}
			if valid != tc.expectValid {
				t.Fatalf("expected validity: %v, got: %v", tc.expectValid, valid)
			}
		})
	}
}
