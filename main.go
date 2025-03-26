package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// ContractInfo represents the contract information
type ContractInfo struct {
	ContractAddress string                 `json:"contractAddress"`
	ContractName    string                 `json:"contractName"`
	Artifact        map[string]interface{} `json:"artifact"`
}

// readJSON reads a JSON file and returns its parsed content
func readJSON(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading JSON file at %s: %v\n", filePath, err)
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Printf("Error parsing JSON from %s: %v\n", filePath, err)
		return nil, err
	}

	return result, nil
}

// findArtifactPath finds the artifact path for a given contract name
func findArtifactPath(outDir, contractName string) (string, error) {
	var artifactPath string
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == contractName+".json" {
			artifactPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if artifactPath == "" {
		return "", logErrorf("artifact not found for contract %s", contractName)
	}

	return artifactPath, nil
}

// processSources processes the sources from the metadata
func processSources(sources map[string]interface{}) (string, error) {
	if sources == nil {
		log.Println("No sources provided")
		return "{}", nil
	}

	transformedSources := make(map[string]map[string]string)

	for filePath, sourceInfo := range sources {
		sourceMap, ok := sourceInfo.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if content is already in the metadata
		if content, ok := sourceMap["content"].(string); ok {
			transformedSources[filePath] = map[string]string{"content": content}
			continue
		}

		// If no content in metadata, try to read the file directly
		absolutePath, err := filepath.Abs(filePath)
		if err != nil {
			log.Printf("Error resolving absolute path for %s: %v\n", filePath, err)
			continue
		}

		content, err := os.ReadFile(absolutePath)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", filePath, err)

			// Try alternative path - sometimes lib paths need to be resolved differently
			if len(filePath) > 4 && filePath[:4] == "lib/" {
				nodeModulesPath := filepath.Join("node_modules", filePath[4:])
				content, err = os.ReadFile(nodeModulesPath)
				if err != nil {
					// Fall back to a placeholder
					transformedSources[filePath] = map[string]string{"content": logErrorf("// Content for %s not available", filePath).Error()}
				} else {
					transformedSources[filePath] = map[string]string{"content": string(content)}
				}
			} else {
				// Fall back to a placeholder
				transformedSources[filePath] = map[string]string{"content": logErrorf("// Content for %s not available", filePath).Error()}
			}
		} else {
			transformedSources[filePath] = map[string]string{"content": string(content)}
		}
	}

	result, err := json.MarshalIndent(transformedSources, "", "  ")
	if err != nil {
		return "{}", err
	}

	return string(result), nil
}

// processRemappings processes the remappings from the metadata
func processRemappings(remappings interface{}) (string, error) {
	if remappings == nil {
		return "[]", nil
	}

	remappingsSlice, ok := remappings.([]interface{})
	if !ok {
		return "[]", nil
	}

	result, err := json.MarshalIndent(remappingsSlice, "", "  ")
	if err != nil {
		return "[]", err
	}

	return string(result), nil
}

// processDirectory processes a single directory
func processDirectory(broadcastDir, dirName, outDir string, allContracts map[string][]ContractInfo) (map[string][]ContractInfo, error) {
	dirPath := filepath.Join(broadcastDir, dirName)
	runLatestPath := filepath.Join(dirPath, "run-latest.json")

	// Initialize this directory in allContracts if it doesn't exist
	if _, ok := allContracts[dirName]; !ok {
		allContracts[dirName] = []ContractInfo{}
	}

	// Read the run-latest.json file
	runLatest, err := readJSON(runLatestPath)
	if err != nil {
		log.Printf("Failed to read run-latest.json in directory %s\n", dirName)
		return allContracts, err
	}

	// Process each transaction in run-latest.json
	transactions, ok := runLatest["transactions"].([]interface{})
	if !ok {
		log.Printf("Invalid transactions in directory %s\n", dirName)
		return allContracts, logErrorf("invalid transactions format")
	}

	for _, txInterface := range transactions {
		tx, ok := txInterface.(map[string]interface{})
		if !ok {
			continue
		}

		contractName, hasContractName := tx["contractName"].(string)
		contractAddress, hasContractAddress := tx["contractAddress"].(string)

		if hasContractName && hasContractAddress {
			artifactPath, err := findArtifactPath(outDir, contractName)
			if err != nil {
				log.Printf("Artifact for contract %s not found: %v\n", contractName, err)
				continue
			}

			artifactContent, err := readJSON(artifactPath)
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
				sources, err = processSources(metadataSources)
				if err != nil {
					log.Printf("Error processing sources for %s: %v\n", contractName, err)
				}
			}

			// Process remappings
			remappings := "[]"
			if settings, ok := metadata["settings"].(map[string]interface{}); ok {
				if settingsRemappings, ok := settings["remappings"]; ok {
					remappings, err = processRemappings(settingsRemappings)
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

// processAllDirectories processes all directories in the broadcast folder
func processAllDirectories(broadcastDir, outDir string) (map[string][]ContractInfo, error) {
	// Get all directories in the broadcast folder
	broadcastDirs, err := os.ReadDir(broadcastDir)
	if err != nil {
		return nil, logErrorf("error reading broadcast directory: %v", err)
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
				allContracts, err = processDirectory(scriptDirPath, chainDir.Name(), outDir, allContracts)
				if err != nil {
					log.Printf("Error processing directory %s: %v\n", chainDir.Name(), err)
				}
			}
		}
	}

	// Write to a file
	data, err := json.MarshalIndent(allContracts, "", "  ")
	if err != nil {
		return nil, logErrorf("error marshaling allContracts: %v", err)
	}

	err = os.WriteFile("processed-contracts.json", data, 0644)
	if err != nil {
		return nil, logErrorf("error writing to processed-contracts.json: %v", err)
	}

	log.Println("Results written to processed-contracts.json")
	return allContracts, nil
}

// groupByContractName groups contracts by their name
func groupByContractName(data map[string][]ContractInfo) map[string]interface{} {
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

// sendToVerificationAPI sends the grouped data to the verification API
func sendToVerificationAPI(data map[string]interface{}, apiURL string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return logErrorf("error marshaling data: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return logErrorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return logErrorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return logErrorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return logErrorf("API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("Verification API response: %s\n", string(body))
	return nil
}

func logErrorf(format string, a ...interface{}) error {
	log.Printf(format, a...)
	return fmt.Errorf(format, a...)
}

func main() {
	// Define command line flags
	broadcastDirFlag := flag.String("broadcast", "./broadcast", "Path to the broadcast directory")
	outDirFlag := flag.String("out", "./out", "Path to the output directory")
	apiURLFlag := flag.String("api", "", "URL of the verification API")
	flag.Parse()

	// Set the directory paths
	broadcastDir := *broadcastDirFlag
	outDir := *outDirFlag
	apiURL := *apiURLFlag

	log.Println("Processing all directories with run-latest.json in broadcast folder...")
	data, err := processAllDirectories(broadcastDir, outDir)
	if err != nil {
		log.Fatalf("Error processing directories: %v\n", err)
	}

	dataGrouped := groupByContractName(data)

	// Output the grouped data
	result, err := json.MarshalIndent(dataGrouped, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling grouped data: %v\n", err)
	}

	log.Println("Grouped data:")
	log.Println(string(result))

	// Send data to backend route for verification if API URL is provided
	if apiURL != "" {
		log.Printf("Sending data to verification API at %s\n", apiURL)
		err = sendToVerificationAPI(dataGrouped, apiURL)
		if err != nil {
			log.Fatalf("Error sending data to verification API: %v\n", err)
		}
		log.Println("Data sent successfully to verification API")
	} else {
		log.Println("No API URL provided. Skipping sending data to verification API.")
	}
}
