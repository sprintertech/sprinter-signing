package lifi

import (
	"bytes"
	"encoding/binary"
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

// AllocatorID extracts the 92-bit allocator ID from lock tag.
func (l *Lock) AllocatorID() (*big.Int, error) {
	// Step 1: Interpret bytes12 as uint96 (big endian)
	// Bytes [0:11] (inclusive) are the 12 bytes of lockTag
	// Go's binary.BigEndian.Uint64 handles up to 8 bytes, so we slice into two parts:
	upper := binary.BigEndian.Uint64(l.LockTag[0:8])
	lower := binary.BigEndian.Uint32(l.LockTag[8:12])
	// Combine into a uint96 (represented as a big.Int)
	allocatorBits := new(big.Int).SetUint64(upper)
	allocatorBits.Lsh(allocatorBits, 32)
	allocatorBits.Add(allocatorBits, new(big.Int).SetUint64(uint64(lower)))

	// Step 2: Right-shift by 4 (to drop the lower 4 bits for period/scope)
	// allocatorBits >>= 4
	allocatorBits.Rsh(allocatorBits, 4)

	// The result is a 92-bit allocator ID as big.Int
	return allocatorBits, nil
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

// GenerateCompactDigest generates the EIP-712 digest of the batch compact structure from the lifi order
func GenerateCompactDigest(chainId *big.Int, verifyingContract common.Address, order LifiOrder) ([]byte, *BatchCompact, error) {
	b, err := convertLifiOrderToBatchCompact(order)
	if err != nil {
		return []byte{}, nil, err
	}

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
		Message:     batchCompactToMessage(*b),
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, nil, err
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, nil, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(messageHash)))
	return crypto.Keccak256(rawData), b, nil
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

// convertLifiOrderToBatchCompact calculates the EIP712 BatchCompact from the lifi order
func convertLifiOrderToBatchCompact(lifiOrder LifiOrder) (*BatchCompact, error) {
	commitments, err := extractCommitments(lifiOrder.Order.Inputs)
	if err != nil {
		return nil, err
	}

	sponsor := common.HexToAddress(lifiOrder.Order.User)
	arbiter := common.HexToAddress(lifiOrder.Order.LocalOracle)

	return &BatchCompact{
		Arbiter:     arbiter,
		Sponsor:     sponsor,
		Nonce:       lifiOrder.Order.Nonce.Int,
		Expires:     big.NewInt(lifiOrder.Order.Expires),
		Commitments: commitments,
	}, nil
}

func extractCommitments(idsAndAmounts [][2]*BigInt) ([]Lock, error) {
	locks := make([]Lock, len(idsAndAmounts))

	for i, idsAndAmount := range idsAndAmounts {
		// Extract lockTag (first 12 bytes) from idsAndAmount[0]
		lockTag := extractLockTag(idsAndAmount[0].Int)

		// Extract token address (last 20 bytes) from idsAndAmount[0]
		tokenAddr := extractTokenAddress(idsAndAmount[0].Int)

		locks[i] = Lock{
			LockTag: lockTag,
			Token:   tokenAddr,
			Amount:  idsAndAmount[1].Int,
		}
	}

	return locks, nil
}

func extractLockTag(packed *big.Int) [12]byte {
	var lockTag [12]byte

	bytes32 := make([]byte, 32)
	packed.FillBytes(bytes32)

	copy(lockTag[:], bytes32[:12])
	return lockTag
}

func extractTokenAddress(packed *big.Int) common.Address {
	temp := new(big.Int).Set(packed)

	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	temp.And(temp, mask)

	var addr common.Address
	temp.FillBytes(addr[:])
	return addr
}
