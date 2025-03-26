package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ReadJSON reads a JSON file and returns its parsed content
func ReadJSON(filePath string) (map[string]interface{}, error) {
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

// LogErrorf logs an error and returns it
func LogErrorf(format string, a ...interface{}) error {
	log.Printf(format, a...)
	return fmt.Errorf(format, a...)
}

// WriteJSONToFile writes JSON data to a file
func WriteJSONToFile(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return LogErrorf("error marshaling data: %v", err)
	}

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return LogErrorf("error writing to %s: %v", filePath, err)
	}

	log.Printf("Results written to %s\n", filePath)
	return nil
}
