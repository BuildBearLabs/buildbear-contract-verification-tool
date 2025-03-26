package contract

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"buildbear-contract-verification-tool/pkg/utils"
)

// ProcessDirectory processes a single directory
func ProcessDirectory(broadcastDir, dirName, outDir string, allContracts map[string][]ContractInfo) (map[string][]ContractInfo, error) {
	dirPath := filepath.Join(broadcastDir, dirName)
	runLatestPath := filepath.Join(dirPath, "run-latest.json")

	// Initialize this directory in allContracts if it doesn't exist
	if _, ok := allContracts[dirName]; !ok {
		allContracts[dirName] = []ContractInfo{}
	}

	// Read the run-latest.json file
	runLatest, err := utils.ReadJSON(runLatestPath)
	if err != nil {
		log.Printf("Failed to read run-latest.json in directory %s\n", dirName)
		return allContracts, err
	}

	// Process each transaction in run-latest.json
	transactions, ok := runLatest["transactions"].([]interface{})
	if !ok {
		log.Printf("Invalid transactions in directory %s\n", dirName)
		return allContracts, utils.LogErrorf("invalid transactions format")
	}

	for _, txInterface := range transactions {
		tx, ok := txInterface.(map[string]interface{})
		if !ok {
			continue
		}

		contractName, hasContractName := tx["contractName"].(string)
		contractAddress, hasContractAddress := tx["contractAddress"].(string)

		if hasContractName && hasContractAddress {
			artifactPath, err := FindArtifactPath(outDir, contractName)
			if err != nil {
				log.Printf("Artifact for contract %s not found: %v\n", contractName, err)
				continue
			}

			artifactContent, err := utils.ReadJSON(artifactPath)
			if err != nil {
				log.Printf("Failed to read artifact for contract %s: %v\n", contractName, err)
				continue
			}

			metadata, ok := artifactContent["metadata"].(map[string]interface{})
			if !ok {
				log.Printf("Failed to get metadata for contract %s\n", contractName)
				continue
			}

			// Process sources
			sources := "{}"
			if metadataSources, ok := metadata["sources"].(map[string]interface{}); ok {
				sources, err = ProcessSources(metadataSources)
				if err != nil {
					log.Printf("Error processing sources for %s: %v\n", contractName, err)
				}
			}

			// Process remappings
			remappings := "[]"
			if settings, ok := metadata["settings"].(map[string]interface{}); ok {
				if settingsRemappings, ok := settings["remappings"]; ok {
					remappings, err = ProcessRemappings(settingsRemappings)
					if err != nil {
						log.Printf("Error processing remappings for %s: %v\n", contractName, err)
					}
				}
			}

			// Prepare the settings map
			settings := make(map[string]interface{})
			if metadataSettings, ok := metadata["settings"].(map[string]interface{}); ok {
				settings["evmVersion"] = metadataSettings["evmVersion"]
				settings["metadata"] = metadataSettings["metadata"]
				settings["libraries"] = metadataSettings["libraries"]
				settings["optimizer"] = metadataSettings["optimizer"]
				settings["outputSelection"] = map[string]map[string][]string{
					"*": {
						"*": []string{
							"abi",
							"devdoc",
							"userdoc",
							"storageLayout",
							"evm.bytecode.object",
							"evm.bytecode.sourceMap",
							"evm.bytecode.linkReferences",
							"evm.deployedBytecode.object",
							"evm.deployedBytecode.sourceMap",
							"evm.deployedBytecode.linkReferences",
							"evm.deployedBytecode.immutableReferences",
							"metadata",
						},
					},
				}
			}

			// Unmarshal remappings from JSON string
			var remappingsInterface interface{}
			err = json.Unmarshal([]byte(remappings), &remappingsInterface)
			if err != nil {
				log.Printf("Error unmarshaling remappings for %s: %v\n", contractName, err)
			} else {
				settings["remappings"] = remappingsInterface
			}

			// Unmarshal sources from JSON string
			var sourcesInterface interface{}
			err = json.Unmarshal([]byte(sources), &sourcesInterface)
			if err != nil {
				log.Printf("Error unmarshaling sources for %s: %v\n", contractName, err)
			}

			// Create the artifact
			artifact := map[string]interface{}{
				"deployedBytecode": artifactContent["bytecode"],
				"abi":              artifactContent["abi"],
				"language":         metadata["language"],
				"settings":         settings,
				"sources":          sourcesInterface,
			}

			// Add contract to the directory's array
			allContracts[dirName] = append(allContracts[dirName], ContractInfo{
				ContractAddress: contractAddress,
				ContractName:    contractName,
				Artifact:        artifact,
			})
		}
	}

	return allContracts, nil
}

// ProcessAllDirectories processes all directories in the broadcast folder
func ProcessAllDirectories(broadcastDir, outDir string, outputPath string) (map[string][]ContractInfo, error) {
	// Get all directories in the broadcast folder
	broadcastDirs, err := os.ReadDir(broadcastDir)
	if err != nil {
		return nil, utils.LogErrorf("error reading broadcast directory: %v", err)
	}

	allContracts := make(map[string][]ContractInfo)

	// Process each directory under broadcast
	for _, scriptDir := range broadcastDirs {
		if !scriptDir.IsDir() {
			continue
		}

		scriptDirPath := filepath.Join(broadcastDir, scriptDir.Name())
		chainDirs, err := os.ReadDir(scriptDirPath)
		if err != nil {
			log.Printf("Error reading directory %s: %v\n", scriptDir.Name(), err)
			continue
		}

		// Check each subdirectory for run-latest.json
		for _, chainDir := range chainDirs {
			if !chainDir.IsDir() {
				continue
			}

			runLatestPath := filepath.Join(scriptDirPath, chainDir.Name(), "run-latest.json")
			if _, err := os.Stat(runLatestPath); err == nil {
				// Found a directory with run-latest.json
				log.Printf("Found run-latest.json in %s/%s\n", scriptDir.Name(), chainDir.Name())
				allContracts, err = ProcessDirectory(scriptDirPath, chainDir.Name(), outDir, allContracts)
				if err != nil {
					log.Printf("Error processing directory %s: %v\n", chainDir.Name(), err)
				}
			}
		}
	}

	// Write to a file
	err = utils.WriteJSONToFile(outputPath, allContracts)
	if err != nil {
		return nil, utils.LogErrorf("error writing to %s: %v", outputPath, err)
	}

	return allContracts, nil
}

// GroupByContractName groups contracts by their name
func GroupByContractName(data map[string][]ContractInfo) map[string]interface{} {
	grouped := make(map[string]interface{})

	for key, contracts := range data {
		for _, contract := range contracts {
			if _, ok := grouped[contract.ContractName]; !ok {
				// Initialize the contract entry
				grouped[contract.ContractName] = map[string]interface{}{
					"artifact":          contract.Artifact,
					"contractAddresses": map[string]string{},
				}
			}

			// Add the contract address to the contractAddresses map
			contractInfo := grouped[contract.ContractName].(map[string]interface{})
			contractAddresses := contractInfo["contractAddresses"].(map[string]string)
			contractAddresses[key] = contract.ContractAddress
		}
	}

	return grouped
}
