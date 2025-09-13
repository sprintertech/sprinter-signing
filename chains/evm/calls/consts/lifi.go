package consts

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var LifiABI, _ = abi.JSON(strings.NewReader(`[{
  "name": "fillOrderOutputs",
  "type": "function",
  "stateMutability": "nonpayable",
  "inputs": [
    {"name": "fillDeadline", "type": "uint32"},
    {"name": "orderId", "type": "bytes32"},
    {
      "name": "outputs",
      "type": "tuple[]",
      "components": [
        {"name": "oracle", "type": "bytes32"},
        {"name": "settler", "type": "bytes32"},
        {"name": "chainId", "type": "uint256"},
        {"name": "token", "type": "bytes32"},
        {"name": "amount", "type": "uint256"},
        {"name": "recipient", "type": "bytes32"},
        {"name": "call", "type": "bytes"},
        {"name": "context", "type": "bytes"}
      ]
    },
    {"name": "proposedSolver", "type": "bytes32"}
  ],
  "outputs": []
}]`))
