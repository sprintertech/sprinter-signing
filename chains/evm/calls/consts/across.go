package consts

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var HubPoolABI, _ = abi.JSON(strings.NewReader(`
[
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "destinationChainId",
        "type": "uint256"
      },
      {
        "internalType": "address",
        "name": "l1Token",
        "type": "address"
      }
    ],
    "name": "poolRebalanceRoute",
    "outputs": [
      {
        "internalType": "address",
        "name": "destinationToken",
        "type": "address"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`))

var SpokePoolABI, _ = abi.JSON(strings.NewReader(`
[
  {
    "inputs": [
      {
        "components": [
          {
            "internalType": "bytes32",
            "name": "depositor",
            "type": "bytes32"
          },
          {
            "internalType": "bytes32",
            "name": "recipient",
            "type": "bytes32"
          },
          {
            "internalType": "bytes32",
            "name": "exclusiveRelayer",
            "type": "bytes32"
          },
          {
            "internalType": "bytes32",
            "name": "inputToken",
            "type": "bytes32"
          },
          {
            "internalType": "bytes32",
            "name": "outputToken",
            "type": "bytes32"
          },
          {
            "internalType": "uint256",
            "name": "inputAmount",
            "type": "uint256"
          },
          {
            "internalType": "uint256",
            "name": "outputAmount",
            "type": "uint256"
          },
          {
            "internalType": "uint256",
            "name": "originChainId",
            "type": "uint256"
          },
          {
            "internalType": "uint256",
            "name": "depositId",
            "type": "uint256"
          },
          {
            "internalType": "uint32",
            "name": "fillDeadline",
            "type": "uint32"
          },
          {
            "internalType": "uint32",
            "name": "exclusivityDeadline",
            "type": "uint32"
          },
          {
            "internalType": "bytes",
            "name": "message",
            "type": "bytes"
          }
        ],
        "internalType": "struct V3SpokePoolInterface.V3RelayData",
        "name": "relayData",
        "type": "tuple"
      },
      {
        "internalType": "uint256",
        "name": "repaymentChainId",
        "type": "uint256"
      },
      {
        "internalType": "bytes32",
        "name": "repaymentAddress",
        "type": "bytes32"
      }
    ],
    "name": "fillRelay",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
{
  "anonymous": false,
  "inputs": [
    {
      "indexed": false,
      "internalType": "bytes32",
      "name": "inputToken",
      "type": "bytes32"
    },
    {
      "indexed": false,
      "internalType": "bytes32",
      "name": "outputToken",
      "type": "bytes32"
    },
    {
      "indexed": false,
      "internalType": "uint256",
      "name": "inputAmount",
      "type": "uint256"
    },
    {
      "indexed": false,
      "internalType": "uint256",
      "name": "outputAmount",
      "type": "uint256"
    },
    {
      "indexed": true,
      "internalType": "uint256",
      "name": "destinationChainId",
      "type": "uint256"
    },
    {
      "indexed": true,
      "internalType": "uint256",
      "name": "depositId",
      "type": "uint256"
    },
    {
      "indexed": false,
      "internalType": "uint32",
      "name": "quoteTimestamp",
      "type": "uint32"
    },
    {
      "indexed": false,
      "internalType": "uint32",
      "name": "fillDeadline",
      "type": "uint32"
    },
    {
      "indexed": false,
      "internalType": "uint32",
      "name": "exclusivityDeadline",
      "type": "uint32"
    },
    {
      "indexed": true,
      "internalType": "bytes32",
      "name": "depositor",
      "type": "bytes32"
    },
    {
      "indexed": false,
      "internalType": "bytes32",
      "name": "recipient",
      "type": "bytes32"
    },
    {
      "indexed": false,
      "internalType": "bytes32",
      "name": "exclusiveRelayer",
      "type": "bytes32"
    },
    {
      "indexed": false,
      "internalType": "bytes",
      "name": "message",
      "type": "bytes"
    }
  ],
  "name": "FundsDeposited",
  "type": "event"
}
]
`))
