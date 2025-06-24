package rhinestone

type BundleStatus string

const (
	StatusCompleted = "COMPLETED"
)

type BundleData struct {
	Nonce   string `json:"nonce"`
	Expires string `json:"expires"`
}

type BundleEvent struct {
	BundleId            string               `json:"bundleId"`
	AcrossDepositEvents []AcrossDepositEvent `json:"acrossDepositEvents"`
}

type AcrossDepositEvent struct {
	Message             string `json:"message"`
	DepositId           string `json:"depositId"`
	Depositor           string `json:"depositor"`
	Recipient           string `json:"recipient"`
	InputToken          string `json:"inputToken"`
	InputAmount         string `json:"inputAmount"`
	OutputToken         string `json:"outputToken"`
	FillDeadline        string `json:"fillDeadline"`
	OutputAmount        string `json:"outputAmount"`
	QuoteTimestamp      uint64 `json:"quoteTimestamp"`
	ExclusiveRelayer    string `json:"exclusiveRelayer"`
	DestinationChainId  uint64 `json:"destinationChainId"`
	ExclusivityDeadline string `json:"exclusivityDeadline"`
}

type Bundle struct {
	Status        BundleStatus `json:"status"`
	TargetChainId uint64       `json:"targetChainId"`
	UserAddress   string       `json:"userAddress"`
	BundleData    BundleData   `json:"bundleData"`
	BundleEvent   BundleEvent  `json:"bundleEvent"`
}
