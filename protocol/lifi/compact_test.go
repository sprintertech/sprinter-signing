package lifi_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sprintertech/sprinter-signing/protocol/lifi/mock"
	"github.com/sygmaprotocol/sygma-core/crypto/secp256k1"
)

func TestVerifyCompactSignature(t *testing.T) {
	// Sample valid BatchCompact and signature, replace with actual values
	validKp, _ := secp256k1.GenerateKeypair()
	validSponsor := common.HexToAddress(validKp.Address())

	invalidKp, _ := secp256k1.GenerateKeypair()

	var validOrder *lifi.LifiOrder
	_ = json.Unmarshal([]byte(mock.ExpectedLifiResponse), &validOrder)

	digest, validBatchCompact, err := lifi.GenerateCompactDigest(big.NewInt(1), common.Address{}, *validOrder)
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
			compact:     *validBatchCompact,
			signature:   validSignature,
			expectValid: true,
			expectErr:   false,
		},
		{
			name:    "Invalid signature (wrong signer)",
			compact: *validBatchCompact,
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
