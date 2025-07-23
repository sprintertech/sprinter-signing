package consts

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var CompactABI, _ = abi.JSON(strings.NewReader(`
[
  {
    "inputs": [
      {
        "internalType": "uint96",
        "name": "allocatorId",
        "type": "uint96"
      },
    ],
    "name": "toRegisteredAllocator",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "name": "getForcedWithdrawalStatus",
    "type": "function",
    "stateMutability": "view",
    "inputs": [
      { "name": "account", "type": "address" },
      { "name": "id",      "type": "uint256" }
    ],
    "outputs": [
      { "name": "status",           "type": "uint8" },
      { "name": "withdrawableAt",   "type": "uint256" }
    ]
  },
  {
    "name": "hasConsumedAllocatorNonce",
    "type": "function",
    "stateMutability": "view",
    "inputs": [
      { "name": "allocator", "type": "address" },
      { "name": "nonce",     "type": "uint256" }
    ],
    "outputs": [
      { "name": "", "type": "bool" }
    ]
  }
]
`))
