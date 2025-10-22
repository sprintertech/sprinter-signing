package signature

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	DOMAIN_NAME = "LiquidityPool"
	VERSION     = "1.0.0"
)

// BorrowUnlockHash calculates the hash that has to be signed and submitted on-chain to the liquidity
// pool contract.
func BorrowUnlockHash(
	calldata []byte,
	outputAmount *big.Int,
	outputToken common.Address,
	destinationChainId *big.Int,
	target common.Address,
	deadline uint64,
	caller common.Address,
	liquidityPool common.Address,
	nonce *big.Int,
) ([]byte, error) {
	msg := apitypes.TypedDataMessage{
		"caller":         caller.Hex(),
		"borrowToken":    outputToken.Hex(),
		"amount":         outputAmount,
		"target":         target.Hex(),
		"targetCallData": calldata,
		"nonce":          nonce,
		"deadline":       new(big.Int).SetUint64(deadline),
	}

	chainId := math.HexOrDecimal256(*destinationChainId)
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Borrow": []apitypes.Type{
				{Name: "caller", Type: "address"},
				{Name: "borrowToken", Type: "address"},
				{Name: "amount", Type: "uint256"},
				{Name: "target", Type: "address"},
				{Name: "targetCallData", Type: "bytes"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
		},
		PrimaryType: "Borrow",
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           &chainId,
			Version:           VERSION,
			VerifyingContract: liquidityPool.Hex(),
		},
		Message: msg,
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

// BorrowManyUnlockHash calculates the hash that has to be signed and submitted on-chain to the liquidity
// pool contract.
func BorrowManyUnlockHash(
	calldata []byte,
	outputAmounts []*big.Int,
	outputTokens []common.Address,
	destinationChainId *big.Int,
	target common.Address,
	deadline uint64,
	caller common.Address,
	liquidityPool common.Address,
	nonce *big.Int,
) ([]byte, error) {
	hexOutputTokens := make([]string, len(outputTokens))
	for i, token := range outputTokens {
		hexOutputTokens[i] = token.Hex()
	}

	msg := apitypes.TypedDataMessage{
		"caller":         caller.Hex(),
		"borrowTokens":   hexOutputTokens,
		"amounts":        outputAmounts,
		"target":         target.Hex(),
		"targetCallData": calldata,
		"nonce":          nonce,
		"deadline":       new(big.Int).SetUint64(deadline),
	}

	chainId := math.HexOrDecimal256(*destinationChainId)
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"BorrowMany": []apitypes.Type{
				{Name: "caller", Type: "address"},
				{Name: "borrowTokens", Type: "address[]"},
				{Name: "amounts", Type: "uint256[]"},
				{Name: "target", Type: "address"},
				{Name: "targetCallData", Type: "bytes"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
		},
		PrimaryType: "BorrowMany",
		Domain: apitypes.TypedDataDomain{
			Name:              DOMAIN_NAME,
			ChainId:           &chainId,
			Version:           VERSION,
			VerifyingContract: liquidityPool.Hex(),
		},
		Message: msg,
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
