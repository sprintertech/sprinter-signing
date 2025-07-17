package lifi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type Lock struct {
	LockTag [12]byte       // bytes12
	Token   common.Address // address
	Amount  *big.Int       // uint256
}

type BatchCompact struct {
	Arbiter     common.Address
	Sponsor     common.Address
	Nonce       *big.Int
	Expires     *big.Int
	Commitments []Lock
}

const LOCK_TYPEHASH = "fb7744571d97aa61eb9c2bc3c67b9b1ba047ac9e95afb2ef02bc5b3d9e64fbe5"

// VerifyCompactSignature verifies that the signature over the compact digest was made by the signer
func VerifyCompactSignature(digest []byte, signature []byte, signer common.Address) (bool, error) {
	pubkey, err := crypto.Ecrecover(digest, signature)
	if err != nil {
		return false, fmt.Errorf("ecrecover: %v", err)
	}
	pk, err := crypto.UnmarshalPubkey(pubkey)
	if err != nil {
		return false, err
	}
	recoveredSigner := crypto.PubkeyToAddress(*pk)

	return bytes.Equal(signer[:], recoveredSigner[:]), nil
}

// GenerateCompactDigest generates the EIP-712 digest of the batch compact structure
func GenerateCompactDigest(chainId *big.Int, verifyingContract common.Address, b BatchCompact) ([]byte, error) {
	var types = apitypes.Types{
		"EIP712Domain": []apitypes.Type{
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Lock": []apitypes.Type{
			{Name: "lockTag", Type: "bytes12"},
			{Name: "token", Type: "address"},
			{Name: "amount", Type: "uint256"},
		},
		"BatchCompact": []apitypes.Type{
			{Name: "arbiter", Type: "address"},
			{Name: "sponsor", Type: "address"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expires", Type: "uint256"},
			{Name: "commitments", Type: "Lock[]"},
		},
	}

	domain := apitypes.TypedDataDomain{
		Name:              "The Compact",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(chainId.Int64()),
		VerifyingContract: verifyingContract.Hex(),
	}
	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "BatchCompact",
		Domain:      domain,
		Message:     batchCompactToMessage(b),
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(messageHash)))
	return crypto.Keccak256(rawData), nil
}

func batchCompactToMessage(b BatchCompact) map[string]interface{} {
	commitments := make([]map[string]interface{}, len(b.Commitments))
	for i, lock := range b.Commitments {
		commitments[i] = lockToMap(lock)
	}

	return map[string]interface{}{
		"arbiter":     b.Arbiter.Hex(),
		"sponsor":     b.Sponsor.Hex(),
		"nonce":       b.Nonce.String(),
		"expires":     b.Expires.String(),
		"commitments": commitments,
	}
}

func lockToMap(lock Lock) map[string]interface{} {
	return map[string]interface{}{
		"lockTag": "0x" + hex.EncodeToString(lock.LockTag[:]),
		"token":   lock.Token.Hex(),
		"amount":  lock.Amount.String(),
	}
}
