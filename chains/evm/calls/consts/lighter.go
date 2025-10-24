package consts

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var LighterABI, _ = abi.JSON(strings.NewReader(`[{
  "name": "withdraw",
  "type": "function",
  "stateMutability": "nonpayable",
  "inputs": [
    {"name": "txHash", "type": "bytes32"},
    {"name": "toAddress", "type": "address"},
    {"name": "amount", "type": "uint256"}
  ],
  "outputs": []
}]`))
