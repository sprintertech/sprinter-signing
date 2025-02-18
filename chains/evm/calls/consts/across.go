package consts

const SpokePoolABI = `
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
  }
]
`
