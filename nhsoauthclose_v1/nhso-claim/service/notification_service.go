// service/notification_service.go
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type NotificationService struct {
	telegramBotToken string
	telegramChatID   string
	morpromAPIURL    string
	morpromAPIKey    string
	logDir           string
}

type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type MorpromMessage struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"` // success, error, warning, info
	VN      string `json:"vn,omitempty"`
	HN      string `json:"hn,omitempty"`
}

func NewNotificationService() *NotificationService {
	godotenv.Load()
	
	return &NotificationService{
		telegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		morpromAPIURL:    os.Getenv("MORPROM_API_URL"),
		morpromAPIKey:    os.Getenv("MORPROM_API_KEY"),
		logDir:           os.Getenv("LOG_DIR"),
	}
}

// SendTelegramNotification ส่งข้อความแจ้งเตือนผ่าน Telegram
// SendTelegramNotification ส่งข้อความแจ้งเตือนผ่าน Telegram
func (ns *NotificationService) SendTelegramNotification(message string) error {
	if ns.telegramBotToken == "" || ns.telegramChatID == "" {
		return fmt.Errorf("telegram configuration not set")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", ns.telegramBotToken)

	payload := TelegramMessage{
		ChatID: ns.telegramChatID,
		Text:   message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling telegram payload: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for {
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("error sending telegram message: %v", err)
		}
		defer resp.Body.Close()

		// อ่าน response body
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			// ส่งสำเร็จ ✅
			return nil
		}

		// parse response เพื่อดู error_code
		var tgResp struct {
			OK          bool   `json:"ok"`
			ErrorCode   int    `json:"error_code"`
			Description string `json:"description"`
			Parameters  struct {
				RetryAfter int `json:"retry_after"`
			} `json:"parameters"`
		}
		_ = json.Unmarshal(body, &tgResp)

		if tgResp.ErrorCode == 429 && tgResp.Parameters.RetryAfter > 0 {
			wait := time.Duration(tgResp.Parameters.RetryAfter) * time.Second
			fmt.Printf("⚠️ Telegram rate limit hit, retrying after %v...\n", wait)
			time.Sleep(wait)
			continue // retry อีกครั้ง
		}

		// ถ้าเป็น error อื่น → return เลย
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(body))
	}
}

// SendMorpromNotification ส่งข้อความแจ้งเตือนผ่าน หมอพร้อม API
func (ns *NotificationService) SendMorpromNotification(title, message, messageType, vn, hn string) error {
	if ns.morpromAPIURL == "" || ns.morpromAPIKey == "" {
		return fmt.Errorf("morprom configuration not set")
	}

	// สร้าง payload ตามรูปแบบใหม่
	payload := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"type": "text",
				"text": message, // ใช้ message ที่ฟอร์แมตไว้แล้ว
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling morprom payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://morpromt2f.moph.go.th/api/notify/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating morprom request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer "+ns.morpromAPIKey)
    req.Header.Set("client-key", "adc398af6a4b55fd668a33dc575d3157fc84e08a")
	req.Header.Set("secret-key", "Z52WR7Y2MYUH4YU5BSYYAB4NR3YY")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending morprom message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("morprom API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// LogToFile บันทึก log ลงไฟล์ตาม VN
func (ns *NotificationService) LogToFile(vn, level, message string) error {
	if ns.logDir == "" {
		ns.logDir = "./logs"
	}

	// สร้าง directory ถ้ายังไม่มี
	if err := os.MkdirAll(ns.logDir, 0755); err != nil {
		return fmt.Errorf("error creating log directory: %v", err)
	}

	// สร้างไฟล์ log สำหรับ VN นี้
	logFile := filepath.Join(ns.logDir, fmt.Sprintf("%s.log", vn))
	
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	if _, err := file.WriteString(logEntry); err != nil {
		return fmt.Errorf("error writing to log file: %v", err)
	}

	return nil
}

// NotifySuccess แจ้งเตือนเมื่อสำเร็จ
func (ns *NotificationService) NotifySuccess(vn, hn, transactionID, authenCode string, totalAmount float64) {
	message := fmt.Sprintf("✅ Claim สำเร็จ\nVN: %s\nHN: %s\nTransaction: %s\nAuthenCode: %s\nAmount: %.2f บาท",
		vn, hn, transactionID, authenCode, totalAmount)

	// ส่ง Telegram
	go func() {
		if err := ns.SendTelegramNotification(message); err != nil {
			log.Printf("Failed to send Telegram notification: %v", err)
		}
	}()

	// ส่ง หมอพร้อม
	go func() {
		if err := ns.SendMorpromNotification("Claim สำเร็จ", message, "success", vn, hn); err != nil {
			log.Printf("Failed to send Morprom notification: %v", err)
		}
	}()

	// บันทึก log
	go func() {
		if err := ns.LogToFile(vn, "SUCCESS", message); err != nil {
			log.Printf("Failed to write log file: %v", err)
		}
	}()
}

// NotifyError แจ้งเตือนเมื่อเกิด error
func (ns *NotificationService) NotifyError(vn, hn, transactionID, errorMessage string, step string) {
	message := fmt.Sprintf("❌ Claim Error (%s)\nVN: %s\nHN: %s\nTransaction: %s\nError: %s",
		step, vn, hn, transactionID, errorMessage)

	// ส่ง Telegram
	go func() {
		if err := ns.SendTelegramNotification(message); err != nil {
			log.Printf("Failed to send Telegram notification: %v", err)
		}
	}()

	// ส่ง หมอพร้อม
	go func() {
		if err := ns.SendMorpromNotification("Claim Error", message, "error", vn, hn); err != nil {
			log.Printf("Failed to send Morprom notification: %v", err)
		}
	}()

	// บันทึก log
	go func() {
		if err := ns.LogToFile(vn, "ERROR", fmt.Sprintf("%s: %s", step, errorMessage)); err != nil {
			log.Printf("Failed to write log file: %v", err)
		}
	}()
}

// NotifyStep บันทึกขั้นตอนการทำงาน
func (ns *NotificationService) NotifyStep(vn, step, message string) {
	logMessage := fmt.Sprintf("Step: %s - %s", step, message)
	
	// บันทึก log
	go func() {
		if err := ns.LogToFile(vn, "INFO", logMessage); err != nil {
			log.Printf("Failed to write log file: %v", err)
		}
	}()
}
