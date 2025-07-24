package lifi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type Lock struct {
	LockTag [12]byte
	Token   common.Address
	Amount  *big.Int
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

func (l *Lock) Period() (time.Duration, error) {
	lastFour := binary.BigEndian.Uint32(l.LockTag[8:12])
	period := uint8((lastFour >> 2) & 0x3) // Bits 92-93: period
	return ResetPeriod(period).ToDuration()
}

type ResetPeriod uint8

func (p ResetPeriod) ToDuration() (time.Duration, error) {
	switch p {
	case OneSecond:
		return time.Second, nil
	case FifteenSeconds:
		return time.Second * 15, nil
	case OneMinute:
		return time.Minute, nil
	case TenMinutes:
		return time.Minute * 10, nil
	case OneHourAndFiveMinutes:
		return time.Minute * 65, nil
	case OneDay:
		return time.Hour * 24, nil
	case SevenDaysAndOneHour:
		return time.Hour*24*7 + time.Hour, nil
	case ThirtyDays:
		return time.Hour * 24 * 30, nil
	default:
		return time.Second, fmt.Errorf("unknown reset period")
	}
}

const (
	OneSecond             ResetPeriod = 0
	FifteenSeconds        ResetPeriod = 1
	OneMinute             ResetPeriod = 2
	TenMinutes            ResetPeriod = 3
	OneHourAndFiveMinutes ResetPeriod = 4
	OneDay                ResetPeriod = 5
	SevenDaysAndOneHour   ResetPeriod = 6
	ThirtyDays            ResetPeriod = 7
)

type BatchCompact struct {
	Arbiter       common.Address
	Sponsor       common.Address
	Nonce         *big.Int
	Expires       *big.Int
	IdsAndAmounts [][2]*big.Int
	Mandate       Mandate
}

type Mandate struct {
	FillDeadline      uint32
	LocalOracle       common.Address
	OutputDescription []Output
}

type Output struct {
	Oracle    common.Hash
	Settler   common.Hash
	ChainId   *big.Int
	Token     common.Hash
	Amount    *big.Int
	Recipient common.Hash
	Call      []byte
	Context   []byte
}

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
		"OutputDescription": []apitypes.Type{
			{Name: "oracle", Type: "bytes32"},
			{Name: "settler", Type: "bytes32"},
			{Name: "token", Type: "bytes32"},
			{Name: "recipient", Type: "bytes32"},
			{Name: "call", Type: "bytes"},
			{Name: "context", Type: "bytes"},
			{Name: "chainId", Type: "uint256"},
			{Name: "amount", Type: "uint256"},
		},
		"Mandate": []apitypes.Type{
			{Name: "fillDeadline", Type: "uint32"},
			{Name: "localOracle", Type: "address"},
			{Name: "outputs", Type: "OutputDescription[]"},
		},
		"BatchCompact": []apitypes.Type{
			{Name: "arbiter", Type: "address"},
			{Name: "sponsor", Type: "address"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expires", Type: "uint256"},
			{Name: "idsAndAmounts", Type: "uint256[2][]"},
			{Name: "mandate", Type: "Mandate"},
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

	rawData := fmt.Appendf(nil, "\x19\x01%s%s", string(domainSeparator), string(messageHash))
	return crypto.Keccak256(rawData), b, nil
}

func batchCompactToMessage(b BatchCompact) map[string]interface{} {
	outputs := make([]map[string]interface{}, len(b.Mandate.OutputDescription))
	for i, o := range b.Mandate.OutputDescription {
		outputs[i] = outputToMap(o)
	}

	mandate := map[string]interface{}{
		"fillDeadline": new(big.Int).SetUint64(uint64(b.Mandate.FillDeadline)),
		"localOracle":  b.Mandate.LocalOracle.Hex(),
		"outputs":      outputs,
	}

	return map[string]interface{}{
		"arbiter":       b.Arbiter.Hex(),
		"sponsor":       b.Sponsor.Hex(),
		"nonce":         b.Nonce.String(),
		"expires":       b.Expires.String(),
		"mandate":       mandate,
		"idsAndAmounts": idsAndAmountsToInterface(b.IdsAndAmounts),
	}
}

func outputToMap(output Output) map[string]interface{} {
	return map[string]interface{}{
		"oracle":    output.Oracle.Hex(),
		"settler":   output.Oracle.Hex(),
		"token":     output.Token.Hex(),
		"recipient": output.Recipient.Hex(),
		"call":      "0x" + hex.EncodeToString(output.Call),
		"context":   "0x" + hex.EncodeToString(output.Context),
		"chainId":   output.ChainId,
		"amount":    output.Amount,
	}
}

func idsAndAmountsToInterface(idsAndAmounts [][2]*big.Int) []interface{} {
	ids := (make([]interface{}, len(idsAndAmounts)))
	for i, amounts := range idsAndAmounts {
		ids[i] = []interface{}{amounts[0], amounts[1]}
	}
	return ids
}

// convertLifiOrderToBatchCompact calculates the EIP712 BatchCompact from the lifi order
func convertLifiOrderToBatchCompact(lifiOrder LifiOrder) (*BatchCompact, error) {
	idsAndAmounts := make([][2]*big.Int, len(lifiOrder.Order.Inputs))
	for i, idAndAmount := range lifiOrder.Order.Inputs {
		idsAndAmounts[i][0] = idAndAmount[0].Int
		idsAndAmounts[i][1] = idAndAmount[1].Int

	}

	outputs, err := extractOutputs(lifiOrder.Order.Outputs)
	if err != nil {
		return nil, err
	}

	return &BatchCompact{
		// TODO: arbiter
		Arbiter:       common.HexToAddress(lifiOrder.Order.LocalOracle),
		Sponsor:       common.HexToAddress(lifiOrder.Order.User),
		Nonce:         lifiOrder.Order.Nonce.Int,
		Expires:       big.NewInt(lifiOrder.Order.Expires),
		IdsAndAmounts: idsAndAmounts,
		Mandate: Mandate{
			FillDeadline:      uint32(lifiOrder.Order.FillDeadline),
			LocalOracle:       common.HexToAddress(lifiOrder.Order.LocalOracle),
			OutputDescription: outputs,
		},
	}, nil
}

func extractOutputs(mandateOutputs []MandateOutput) ([]Output, error) {
	outputs := make([]Output, len(mandateOutputs))
	for i, output := range mandateOutputs {
		chainID, ok := new(big.Int).SetString(output.ChainID, 10)
		if !ok {
			return outputs, fmt.Errorf("failed parsing chainID")
		}

		call, err := hex.DecodeString(output.Call[2:])
		if err != nil {
			return outputs, err
		}

		context, err := hex.DecodeString(output.Context[2:])
		if err != nil {
			return outputs, err
		}

		outputs[i] = Output{
			Oracle:    common.HexToHash(output.Oracle),
			Settler:   common.HexToHash(output.Settler),
			ChainId:   chainID,
			Token:     common.HexToHash(output.Token),
			Amount:    output.Amount.Int,
			Recipient: common.HexToHash(output.Recipient),
			Call:      call,
			Context:   context,
		}
	}
	return outputs, nil
}

func ExtractLocks(idsAndAmounts [][2]*BigInt) ([]Lock, error) {
	locks := make([]Lock, len(idsAndAmounts))

	for i, idsAndAmount := range idsAndAmounts {
		// Extract lockTag (first 12 bytes) from idsAndAmount[0]
		lockTag := extractLockTag(idsAndAmount[0].Int)

		// Extract token address (last 20 bytes) from idsAndAmount[0]
		tokenAddr := ExtractTokenAddress(idsAndAmount[0].Int)

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

func ExtractTokenAddress(packed *big.Int) common.Address {
	temp := new(big.Int).Set(packed)

	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	temp.And(temp, mask)

	var addr common.Address
	temp.FillBytes(addr[:])
	return addr
}
