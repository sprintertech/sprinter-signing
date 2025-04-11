package mayan

import "errors"

// Convert Wormhole chain ID to EVM chain ID
func WormholeToEVMChainID(whChainID uint16) (uint64, error) {
	switch whChainID {
	case 2: // Ethereum
		return 1, nil
	case 4: // BSC
		return 56, nil
	case 5: // Polygon
		return 137, nil
	case 6: // Avalanche
		return 43114, nil
	case 7: // Oasis
		return 42262, nil
	case 10: // Fantom
		return 250, nil
	case 14: // Celo
		return 42220, nil
	case 16: // Moonbeam
		return 1284, nil
	case 23: // Arbitrum
		return 42161, nil
	case 24: // Optimism
		return 10, nil
	case 25: // Gnosis
		return 100, nil
	case 30: // Base
		return 8453, nil
	case 35: // Mantle
		return 5000, nil
	case 38: // Linea
		return 59144, nil
	default:
		return 0, errors.New("unknown Wormhole chain ID")
	}
}
