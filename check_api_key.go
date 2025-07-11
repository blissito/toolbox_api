package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiKey = "tbx_-8wy90RmamaDW7v-"
	testURL = "http://localhost:8000/api/tool"
)

func main() {
	// Test payload for the screenshot tool
	payload := map[string]interface{}{
		"tool": "screenshot",
		"payload": map[string]string{
			"url": "https://example.com",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println("Response Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			fmt.Printf("Error response: %+v\n", result)
		}
	} else {
		fmt.Println("API key is valid!")
	}
}
