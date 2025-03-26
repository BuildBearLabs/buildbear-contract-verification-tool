package contract

// ContractInfo represents the contract information
type ContractInfo struct {
	ContractAddress string                 `json:"contractAddress"`
	ContractName    string                 `json:"contractName"`
	Artifact        map[string]interface{} `json:"artifact"`
}
