package mock_lifi

const LifiMockResponse = `
{
	"order": {
	"user": "0x6C8A0c210C4C097270FA5df9b799d79A6887b11A",
	"nonce": "834202471",
	"inputs": [
		[
		"749071750893463290574776461331093852760741783827",
		"100000"
		]
	],
	"expires": "1760110167",
	"outputs": [
		{
		"call": "0x",
		"token": "0x000000000000000000000000af88d065e77c8cc2239327c5edb3a432268e5831",
		"amount": "100000",
		"oracle": "0x0000000000000000000000000000006ea400569c0040d6e5ba651c00848409be",
		"chainId": "42161",
		"context": "0x",
		"settler": "0x00000000000000000000000000000000d7278408ce7a490015577c41e57143a5",
		"recipient": "0x0000000000000000000000006c8a0c210c4c097270fa5df9b799d79a6887b11a"
		}
	],
	"inputOracle": "0x0000006EA400569c0040d6e5Ba651c00848409Be",
	"fillDeadline": "1760110167",
	"originChainId": "8453"
	},
	"quote": null,
	"sponsorSignature": null,
	"allocatorSignature": null,
	"inputSettler": "0x000001bf3f3175bd007f3889b50000c7006e72c0",
	"meta": {
	"submitTime": 1760102985727,
	"orderStatus": "Signed",
	"destinationAddress": "0x6c8a0c210c4c097270fa5df9b799d79a6887b11a",
	"orderIdentifier": "intent_F_MiaNxyjvyryvvh2OqsuINeqwFMjI",
	"onChainOrderId": "0xe40eb815e8fd931b07ca5bb1be759861ff2a63348624fb5c374b2ee675430638",
	"signedAt": "2025-10-10T13:29:37.000Z",
	"expiredAt": null,
	"lastCompactDepositBlockNumber": null,
	"orderInitiatedTxHash": "0xe40e3258ebfd90ce75feea3859c204f50e14ee9b28f72d971b4cfaa41eedfba4",
	"orderDeliveredTxHash": null,
	"orderVerifiedTxHash": null,
	"orderSettledTxHash": null
	}
}
`

const ExpectedLifiResponse = `    {
      "order": {
        "user": "0x6C8A0c210C4C097270FA5df9b799d79A6887b11A",
        "nonce": "834202471",
        "inputs": [
          [
            "749071750893463290574776461331093852760741783827",
            "100000"
          ]
        ],
        "expires": "1760110167",
        "outputs": [
          {
            "call": "0x",
            "token": "0x000000000000000000000000af88d065e77c8cc2239327c5edb3a432268e5831",
            "amount": "100000",
            "oracle": "0x0000000000000000000000000000006ea400569c0040d6e5ba651c00848409be",
            "chainId": "42161",
            "context": "0x",
            "settler": "0x00000000000000000000000000000000d7278408ce7a490015577c41e57143a5",
            "recipient": "0x0000000000000000000000006c8a0c210c4c097270fa5df9b799d79a6887b11a"
          }
        ],
        "inputOracle": "0x0000006EA400569c0040d6e5Ba651c00848409Be",
        "fillDeadline": "1760110167",
        "originChainId": "8453"
      },
      "quote": null,
      "sponsorSignature": null,
      "allocatorSignature": null,
      "inputSettler": "0x000001bf3f3175bd007f3889b50000c7006e72c0",
      "meta": {
        "submitTime": 1760102985727,
        "orderStatus": "Signed",
        "destinationAddress": "0x6c8a0c210c4c097270fa5df9b799d79a6887b11a",
        "orderIdentifier": "intent_F_MiaNxyjvyryvvh2OqsuINeqwFMjI",
        "onChainOrderId": "0xe40eb815e8fd931b07ca5bb1be759861ff2a63348624fb5c374b2ee675430638",
        "signedAt": "2025-10-10T13:29:37.000Z",
        "expiredAt": null,
        "lastCompactDepositBlockNumber": null,
        "orderInitiatedTxHash": "0xe40e3258ebfd90ce75feea3859c204f50e14ee9b28f72d971b4cfaa41eedfba4",
        "orderDeliveredTxHash": null,
        "orderVerifiedTxHash": null,
        "orderSettledTxHash": null
      }
    }`
