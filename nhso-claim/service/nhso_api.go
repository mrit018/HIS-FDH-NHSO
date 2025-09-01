package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// GetNHSOApiConfig returns the NHSO API configuration from environment variables
func GetNHSOApiConfig() NHSOApiConfig {
	godotenv.Load()
	
	timeout, _ := strconv.Atoi(os.Getenv("NHSO_API_TIMEOUT"))
	if timeout == 0 {
		timeout = 30
	}
	
	return NHSOApiConfig{
		BaseURL: os.Getenv("NHSO_API_BASE_URL"),
		APIKey:  os.Getenv("NHSO_API_KEY"),
		Timeout: timeout,
	}
}

// SendClaim sends a claim to NHSO API
func SendClaim(payload ClaimRequestPayload) (*ClaimResponse, error) {
	config := GetNHSOApiConfig()
	
	// พิมพ์ค่า config สำหรับ debugging
	log.Printf("NHSO API Configuration:")
	log.Printf("  BaseURL: %s", config.BaseURL)
	log.Printf("  Timeout: %d seconds", config.Timeout)
	
	// ตรวจสอบและพิมพ์ API Key อย่างปลอดภัย
	if config.APIKey == "" {
		log.Printf("  API Key: NOT SET - this will cause authentication failure")
	} else {
		// แสดงเฉพาะส่วนแรกของ API Key เพื่อความปลอดภัย
		log.Printf("  API Key: %s... (length: %d)", config.APIKey[:min(8, len(config.APIKey))], len(config.APIKey))
	}
	
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}
	
	// Log request details
	log.Printf("Sending claim to NHSO API: %s", config.BaseURL+"api/nhso-claim-detail")
	log.Printf("Request payload: %s", string(jsonData))
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	// ใช้ endpoint ตามที่ระบุ
	apiEndpoint := "api/nhso-claim-detail"
	
	// Create request
	req, err := http.NewRequest("POST", config.BaseURL+apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers ตามที่ NHSO กำหนด
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-Source-ID", payload.SourceID)
	req.Header.Set("Accept", "application/json")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}
	
	// Log response details
	log.Printf("NHSO API response status: %d", resp.StatusCode)
	log.Printf("NHSO API response body: %s", string(body))
	
	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var claimResponse ClaimResponse
	err = json.Unmarshal(body, &claimResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}
	
	// ตรวจสอบว่ามี error ใน response หรือไม่
	if claimResponse.DataError != "" {
		return nil, fmt.Errorf("NHSO API returned error: %s", claimResponse.DataError)
	}
	
	return &claimResponse, nil
}

// helper function สำหรับการหาค่าน้อยที่สุด
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
