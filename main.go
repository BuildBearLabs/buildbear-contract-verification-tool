package main

import (
	"encoding/json"
	"flag"
	"log"

	"buildbear-contract-verification-tool/pkg/api"
	"buildbear-contract-verification-tool/pkg/contract"
)

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
	outputPath := "processed-contracts.json"

	// Process all directories with run-latest.json
	log.Println("Processing all directories with run-latest.json in broadcast folder...")
	data, err := contract.ProcessAllDirectories(broadcastDir, outDir, outputPath)
	if err != nil {
		log.Fatalf("Error processing directories: %v\n", err)
	}

	// Group contracts by name for display purposes
	dataGrouped := contract.GroupByContractName(data)

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
		// Send the original data format (before grouping) to the API
		err = api.SendRawContractsToVerificationAPI(data, apiURL)
		if err != nil {
			log.Fatalf("Error sending data to verification API: %v\n", err)
		}
		log.Println("Data sent successfully to verification API")
	} else {
		log.Println("No API URL provided. Skipping sending data to verification API.")
	}
}
