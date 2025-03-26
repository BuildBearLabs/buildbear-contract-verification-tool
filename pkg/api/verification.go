package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"buildbear-contract-verification-tool/pkg/utils"
)

// SendToVerificationAPI sends the grouped data to the verification API
func SendToVerificationAPI(data map[string]interface{}, apiURL string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return utils.LogErrorf("error marshaling data: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return utils.LogErrorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return utils.LogErrorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.LogErrorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return utils.LogErrorf("API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("Verification API response: %s\n", string(body))
	return nil
}
