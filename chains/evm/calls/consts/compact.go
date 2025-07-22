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
  }
]
`))
