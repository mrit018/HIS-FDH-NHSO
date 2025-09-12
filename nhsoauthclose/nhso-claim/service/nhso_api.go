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

	"nhsoauthclose/nhso-claim/model"
	"nhsoauthclose/repository"

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

// GetAuthCode requests an auth code from NHSO API
func GetAuthCode(payload AuthRequestPayload) (AuthResponse, error) {
	config := GetNHSOApiConfig()
	
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("error marshaling auth payload: %v", err)
	}
	
	// Log request details
	log.Printf("Getting auth code from NHSO API: %s", config.BaseURL+"api/AuthenCode")
	log.Printf("Auth request payload: %s", string(jsonData))
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	// Create request
	req, err := http.NewRequest("POST", config.BaseURL+"api/AuthenCode", bytes.NewBuffer(jsonData))
	if err != nil {
		return AuthResponse{}, fmt.Errorf("error creating auth request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-Source-ID", payload.SourceID)
	req.Header.Set("Accept", "application/json")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("error sending auth request: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("error reading auth response: %v", err)
	}
	
	// Log response details
	log.Printf("NHSO Auth API response status: %d", resp.StatusCode)
	log.Printf("NHSO Auth API response body: %s", string(body))
	
	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return AuthResponse{}, fmt.Errorf("Auth API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var authResponse AuthResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("error parsing auth response: %v", err)
	}
	
	return authResponse, nil
}

// SendClaim sends a claim to NHSO API
func SendClaim(payload ClaimRequestPayload) (ClaimResponse, error) {
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
		return ClaimResponse{}, fmt.Errorf("error marshaling payload: %v", err)
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
		return ClaimResponse{}, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers ตามที่ NHSO กำหนด
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-Source-ID", payload.SourceID)
	req.Header.Set("Accept", "application/json")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error reading response: %v", err)
	}
	
	// Log response details
	log.Printf("NHSO API response status: %d", resp.StatusCode)
	log.Printf("NHSO API response body: %s", string(body))
	
	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return ClaimResponse{}, fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var claimResponse ClaimResponse
	err = json.Unmarshal(body, &claimResponse)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error parsing response: %v", err)
	}
	
	// ตรวจสอบว่ามี error ใน response หรือไม่
	if claimResponse.DataError != "" {
		return claimResponse, fmt.Errorf("NHSO API returned error: %s", claimResponse.DataError)
	}
	
	return claimResponse, nil
}

// SendClaim2 sends a claim to NHSO API and saves the response
func SendClaim2(payload ClaimRequestPayload, claimRepo *repository.ClaimRepository, claimRequest *model.ClaimRequest) (ClaimResponse, error) {
	config := GetNHSOApiConfig()
	
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error marshaling payload: %v", err)
	}
	
	requestJSON := string(jsonData)
	
	// Log request details
	log.Printf("Sending claim to NHSO API: %s", config.BaseURL+"api/nhso-claim-detail")
	log.Printf("Request payload: %s", requestJSON)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	// ใช้ endpoint ตามที่ระบุ
	apiEndpoint := "api/nhso-claim-detail"
	
	// Create request
	req, err := http.NewRequest("POST", config.BaseURL+apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-Source-ID", payload.SourceID)
	req.Header.Set("Accept", "application/json")
	
	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error reading response: %v", err)
	}
	
	responseJSON := string(body)
	
	// Log response details
	log.Printf("NHSO API response status: %d", resp.StatusCode)
	log.Printf("NHSO API response body: %s", responseJSON)
	
	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return ClaimResponse{}, fmt.Errorf("API returned non-200 status: %d, body: %s", resp.StatusCode, responseJSON)
	}
	
	// Parse response
	var claimResponse ClaimResponse
	err = json.Unmarshal(body, &claimResponse)
	if err != nil {
		return ClaimResponse{}, fmt.Errorf("error parsing response: %v", err)
	}
	
	// ตรวจสอบว่ามี error ใน response หรือไม่
	if claimResponse.DataError != "" {
		return claimResponse, fmt.Errorf("NHSO API returned error: %s", claimResponse.DataError)
	}
	
	// ✅ บันทึกข้อมูลลงตาราง nhso_confirm_privilege
	if claimRepo != nil && claimRequest != nil {
		now := time.Now()
		
		nhsoRecord := &model.NHSOConfirmPrivilege{
			Vn:                   claimRequest.Vn,
			NhsoSeq:              fmt.Sprintf("%d", claimResponse.Seq),
			NhsoAuthenCode:       claimResponse.AuthenCode,
			NhsoRequestJson:      requestJSON,
			NhsoReponseJson:      responseJSON,
			NhsoRequstDatetime:   now,
			NhsoResponseDatetime: now,
			ConfirmStaff:         getSystemUser(),
			NhsoStatus:           "1",
			NhsoTotalAmount:      claimRequest.TotalAmount,
			Pttype:               claimRequest.Pttype,
			NhsoPrivilegeAmount:  claimRequest.PrivilegeAmount,
			NhsoCashAmount:       claimRequest.TotalAmount - claimRequest.PrivilegeAmount,
			FdhTransactionId:     claimRequest.TransactionID,
		}

		// ใช้ method จาก repository
		err := claimRepo.SaveNHSOConfirmPrivilege(nhsoRecord)
		if err != nil {
			log.Printf("Warning: Failed to save NHSO confirm privilege record: %v", err)
		}
	}
	
	return claimResponse, nil
}

// helper function สำหรับการหาค่าน้อยที่สุด
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// helper function สำหรับดึง system user
func getSystemUser() string {
	user := os.Getenv("SYSTEM_USER")
	if user == "" {
		return "SYSTEM"
	}
	return user
}
