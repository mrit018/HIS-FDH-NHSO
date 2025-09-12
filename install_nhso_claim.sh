#!/bin/bash

# สคริปต์ติดตั้งอัตโนมัติสำหรับ NHSO Claim Auto บน Almalinux 10
echo "เริ่มต้นการติดตั้ง NHSO Claim Auto บน Almalinux 10..."

# ตรวจสอบว่าเป็นผู้ใช้ root หรือไม่
if [ "$EUID" -ne 0 ]; then
    echo "กรุณาใช้คำสั่งนี้ด้วยสิทธิ์ sudo หรือเป็นผู้ใช้ root"
    exit 1
fi

# อัพเดทระบบและติดตั้งแพ็กเกจที่จำเป็น
echo "อัพเดทระบบและติดตั้งแพ็กเกจที่จำเป็น..."
dnf update -y
dnf install -y git curl wget unzip make gcc

# ติดตั้ง Go
echo "ติดตั้ง Go..."
if ! command -v go &> /dev/null; then
    GO_VERSION="1.21.0"
    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    
    # ตั้งค่า environment variables สำหรับ Go
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export GOPATH=$HOME/go' >> /etc/profile
    echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile
    source /etc/profile
else
    echo "พบ Go ที่ติดตั้งไว้แล้ว: $(go version)"
fi

# สร้างโฟลเดอร์โปรเจค
echo "สร้างโครงสร้างโฟลเดอร์โปรเจค..."
PROJECT_DIR="/opt/nhsoauthclose"
mkdir -p $PROJECT_DIR
cd $PROJECT_DIR

# สร้างโครงสร้างโฟลเดอร์
mkdir -p config nhso-claim/model nhso-claim/service repository

# สร้างไฟล์ go.mod
cat > go.mod << EOF
module nhsoauthclose

go 1.19

require (
    github.com/joho/godotenv v1.5.1
    gorm.io/driver/mysql v1.5.2
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)

require (
    github.com/go-sql-driver/mysql v1.7.0 // indirect
    github.com/jackc/pgpassfile v1.0.0 // indirect
    github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
    github.com/jackc/pgx/v5 v5.4.3 // indirect
    github.com/jinzhu/inflection v1.0.0 // indirect
    github.com/jinzhu/now v1.1.5 // indirect
    golang.org/x/crypto v0.14.0 // indirect
    golang.org/x/text v0.13.0 // indirect
)
EOF

# สร้างไฟล์ config/database.go
cat > config/database.go << EOF
package config

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/joho/godotenv"
)

// InitDB ฟังก์ชันสำหรับเริ่มต้นการเชื่อมต่อฐานข้อมูล
func InitDB() *gorm.DB {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "mysql"
	}

	var dialector gorm.Dialector
	var dsn string

	switch dbType {
	case "postgres", "postgresql":
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "nhso_claim"
		}
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		dsn = "host=" + host + " user=" + user + " password=" + password +
			" dbname=" + dbname + " port=" + port + " sslmode=" + sslmode +
			" TimeZone=Asia/Bangkok"
		dialector = postgres.Open(dsn)

	case "mysql":
		fallthrough
	default:
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "root"
		}
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "nhso_claim"
		}

		dsn = user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname +
			"?charset=utf8mb4&parseTime=True&loc=Local"
		dialector = mysql.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Printf("Database connection established successfully (%s)", dbType)
	return db
}
EOF

# สร้างไฟล์ model/types.go
cat > nhso-claim/model/types.go << EOF
package model

// MainInsclCode mapping
const (
	MainInsclUCS = "UCS" // สิทธิ UC
	MainInsclOFC = "OFC" // สิทธิข้าราชการ
	MainInsclLGO = "LGO" // สิทธิ อปท.
	MainInsclSSS = "SSS" // ประกันสังคม
	MainInsclOTH = "OTH" // ไม่ใช้สิทธิ
	MainInsclCOR = "COR" // สิทธิคู่สัญญา
)

// ClaimServiceCode mapping
const (
	ClaimServiceGeneral     = "PG0060001" // เข้ารับบริการรักษาทั่วไป
	ClaimServiceDialysis    = "PG0130001" // บริการฟอกเลือดด้วยเครื่องไตเทียม
	ClaimServiceCommonIll   = "PG0150001" // บริการดูแลอาการเจ็บป่วยเบื้องต้น
	ClaimServiceUCEP        = "PG0120001" // UCEP PLUS
	ClaimServiceSelfIsolation = "PG0110001" // Self Isolation
)

// Status types
const (
	StatusPending   = "PENDING"
	StatusCompleted = "COMPLETED"
	StatusFailed    = "FAILED"
)
EOF

# สร้างไฟล์ model/claim_request.go
cat > nhso-claim/model/claim_request.go << EOF
package model

import (
	"time"

	"gorm.io/gorm"
)

type ClaimRequest struct {
	gorm.Model
	Hcode           string    \`gorm:"size:5;not null" json:"hcode"\`
	MainInsclCode   string    \`gorm:"size:3;not null" json:"mainInsclCode"\`
	ServiceDateTime int64     \`gorm:"not null" json:"serviceDateTime"\` // Unix timestamp in milliseconds
	InvoiceDateTime int64     \`gorm:"not null" json:"invoiceDateTime"\` // Unix timestamp in milliseconds
	TransactionID   string    \`gorm:"size:255;not null;unique" json:"transactionId"\`
	TotalAmount     float64   \`gorm:"not null" json:"totalAmount"\`
	PaidAmount      float64   \`gorm:"not null" json:"paidAmount"\`
	PrivilegeAmount float64   \`gorm:"not null" json:"privilegeAmount"\`
	ClaimServiceCode string   \`gorm:"size:20;not null" json:"claimServiceCode"\`
	Pid             string    \`gorm:"size:13;not null" json:"pid"\`
	SourceID        string    \`gorm:"size:50;not null" json:"sourceId"\`
	RecorderPid     string    \`gorm:"size:13;not null" json:"recorderPid"\`
	Status          string    \`gorm:"size:20;default:'PENDING'" json:"status"\`
	ResponseData    string    \`gorm:"type:text" json:"responseData"\`
	
	// Fields from database query
	Hn             string    \`gorm:"size:50" json:"hn"\`
	Birthday       time.Time \`json:"birthday"\`
	Vn             string    \`gorm:"size:50" json:"vn"\`
	Ptname         string    \`gorm:"size:250" json:"ptname"\`
	Pttype         string    \`gorm:"size:10" json:"pttype"\`
	PttypeName     string    \`gorm:"size:100" json:"pttypeName"\`
	SpcltyName     string    \`gorm:"size:100" json:"spcltyName"\`
	DepartmentName string    \`gorm:"size:100" json:"departmentName"\`
	AuthCode       string    \`gorm:"size:50" json:"authCode"\`
}

// TableName specifies the table name for ClaimRequest
func (ClaimRequest) TableName() string {
	return "claim_requests"
}
EOF

# สร้างไฟล์ model/claim_response.go
cat > nhso-claim/model/claim_response.go << EOF
package model

import (
	"gorm.io/gorm"
)

type ClaimResponse struct {
	gorm.Model
	ClaimRequestID  uint   \`gorm:"not null" json:"claimRequestId"\`
	ResponseCode    string \`gorm:"size:4" json:"responseCode"\`
	ResponseData    string \`gorm:"type:text" json:"responseData"\`
	IsSuccess       bool   \`gorm:"default:false" json:"isSuccess"\`
	ResponseMessage string \`gorm:"size:255" json:"responseMessage"\`
}

// TableName specifies the table name for ClaimResponse
func (ClaimResponse) TableName() string {
	return "claim_responses"
}
EOF

# สร้างไฟล์ service/types.go
cat > nhso-claim/service/types.go << EOF
package service

// ClaimRequestPayload represents the payload for NHSO claim API
type ClaimRequestPayload struct {
	Hcode            string  \`json:"hcode"\`
	MainInsclCode    string  \`json:"mainInsclCode"\`
	ServiceDateTime  int64   \`json:"serviceDateTime"\`
	InvoiceDateTime  int64   \`json:"invoiceDateTime"\`
	TransactionID    string  \`json:"transactionId"\`
	TotalAmount      float64 \`json:"totalAmount"\`
	PaidAmount       float64 \`json:"paidAmount"\`
	PrivilegeAmount  float64 \`json:"privilegeAmount"\`
	ClaimServiceCode string  \`json:"claimServiceCode"\`
	Pid              string  \`json:"pid"\`
	SourceID         string  \`json:"sourceId"\`
	RecorderPid      string  \`json:"recorderPid"\`
}

// ClaimResponse represents the response from NHSO API
type ClaimResponse struct {
	Success bool   \`json:"success"\`
	Code    string \`json:"code"\`
	Message string \`json:"message"\`
	Data    string \`json:"data"\`
}

// NHSOApiConfig represents the configuration for NHSO API
type NHSOApiConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}
EOF

# สร้างไฟล์ service/nhso_api.go
cat > nhso-claim/service/nhso_api.go << EOF
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	
	// Create request
	req, err := http.NewRequest("POST", config.BaseURL+"claim", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-Source-ID", payload.SourceID)
	
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
	
	return &claimResponse, nil
}
EOF

# สร้างไฟล์ repository/claim_repository.go
cat > repository/claim_repository.go << EOF
package repository

import (
	"log"
	"nhsoauthclose/nhso-claim/model"
	"os"
	"time"

	"gorm.io/gorm"
)

type ClaimRepository struct {
	db *gorm.DB
}

func NewClaimRepository(db *gorm.DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}

// FetchClaimData ดึงข้อมูลจากฐานข้อมูลตาม query ที่กำหนด
func (r *ClaimRepository) FetchClaimData(dateFilter, pttypeCondition, spcltyCondition, hnCondition string) ([]model.ClaimRequest, error) {
	var claims []model.ClaimRequest
	
	// ใช้ RAW SQL query ตามที่ให้มา
	query := \`
		SELECT
			o.vstdate,
			o.vsttime,
			P.CID,
			o.hn,
			P.birthday,
			o.vn,
			CAST(CONCAT(P.pname, P.fname, ' ', P.lname) AS CHAR(250)) AS ptname,
			T.pttype AS code_pttype,
			T.NAME AS name_pttype,
			s.NAME AS spclty_name,
			K.department AS department_name,
			v.item_money AS totalAmount,
			vpt.auth_code
		FROM
			ovst o
			LEFT OUTER JOIN vn_stat v ON v.vn = o.vn
			LEFT OUTER JOIN opdscreen oc ON oc.vn = o.vn
			LEFT OUTER JOIN patient P ON P.hn = o.hn
			LEFT OUTer JOIN pttype T ON T.pttype = o.pttype
			LEFT OUTER JOIN spclty s ON s.spclty = o.spclty
			LEFT OUTER JOIN kskdepartment K ON K.depcode = o.cur_dep
			LEFT OUTER JOIN visit_pttype vpt ON vpt.vn = o.vn 
				AND vpt.pttype = o.pttype
		WHERE
			\` + dateFilter + \`
			\` + pttypeCondition + \`
			\` + spcltyCondition + \`
			\` + hnCondition + \`
			AND P.nationality IN ('99') 
			AND P.citizenship IN ('99') 
			AND o.cur_dep IN ('999') 
			AND P.CID NOT LIKE '0%'
			AND v.item_money != 0
		ORDER BY
			o.vsttime DESC
	\`

	// ดึงข้อมูลจาก database
	rows, err := r.db.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var claim model.ClaimRequest
		var vstdate time.Time
		var vsttime string
		var birthday time.Time
		
		// Scan ข้อมูลจาก row
		err := rows.Scan(
			&vstdate, &vsttime, &claim.Pid, &claim.Hn, &birthday,
			&claim.Vn, &claim.Ptname, &claim.Pttype, &claim.PttypeName,
			&claim.SpcltyName, &claim.DepartmentName, &claim.TotalAmount,
			&claim.AuthCode,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// รวมวันที่และเวลาเพื่อสร้าง serviceDateTime
		serviceTime, err := time.Parse("2006-01-02 15:04:05", 
			vstdate.Format("2006-01-02") + " " + vsttime)
		if err != nil {
			log.Printf("Error parsing service time: %v", err)
			continue
		}
		claim.ServiceDateTime = serviceTime.Unix() * 1000 // Convert to milliseconds
		
		// ตั้งค่า invoiceDateTime เป็นเวลาปัจจุบัน
		claim.InvoiceDateTime = time.Now().Unix() * 1000
		
		// สร้าง transactionID จาก HCODE + VN
		claim.TransactionID = os.Getenv("HCODE") + claim.Vn
		
		// ตั้งค่าเริ่มต้น
		claim.Hcode = os.Getenv("HCODE")
		claim.SourceID = os.Getenv("SOURCE_ID")
		claim.RecorderPid = os.Getenv("RECORDER_PID")
		claim.Status = model.StatusPending
		claim.Birthday = birthday
		
		// กำหนดค่า paidAmount และ privilegeAmount (ตัวอย่าง)
		claim.PaidAmount = claim.TotalAmount * 0.3 // 30% เป็นค่าใช้จ่ายที่เบิกไม่ได้
		claim.PrivilegeAmount = claim.TotalAmount * 0.7 // 70% เป็นค่าใช้จ่ายที่เบิกได้
		
		// กำหนด claimServiceCode ตามประเภทบริการ
		claim.ClaimServiceCode = determineClaimServiceCode(claim.Pttype, claim.SpcltyName)
		
		// กำหนด mainInsclCode ตามประเภทสิทธิ
		claim.MainInsclCode = mapPttypeToMainInscl(claim.Pttype)
		
		claims = append(claims, claim)
	}

	return claims, nil
}

// determineClaimServiceCode กำหนดรหัสบริการตามประเภทสิทธิและแผนก
func determineClaimServiceCode(pttype, spcltyName string) string {
	switch pttype {
	case "10": // UCS
		return model.ClaimServiceGeneral
	case "11": // OFC
		if spcltyName == "内科" {
			return model.ClaimServiceCommonIll
		}
		return model.ClaimServiceGeneral
	default:
		return model.ClaimServiceGeneral
	}
}

// mapPttypeToMainInscl map ประเภทสิทธิเป็นรหัส MainInscl
func mapPttypeToMainInscl(pttype string) string {
	switch pttype {
	case "10":
		return model.MainInsclUCS
	case "11":
		return model.MainInsclOFC
	case "12":
		return model.MainInsclLGO
	case "13":
		return model.MainInsclSSS
	default:
		return model.MainInsclOTH
	}
}

// SaveClaim บันทึก claim ลงฐานข้อมูล
func (r *ClaimRepository) SaveClaim(claim *model.ClaimRequest) error {
	return r.db.Save(claim).Error
}

// GetPendingClaims ดึง claims ที่มีสถานะ PENDING
func (r *ClaimRepository) GetPendingClaims() ([]model.ClaimRequest, error) {
	var claims []model.ClaimRequest
	err := r.db.Where("status = ?", model.StatusPending).Find(&claims).Error
	return claims, err
}
EOF

# สร้างไฟล์ main.go
cat > main.go << EOF
package main

import (
	"encoding/json"
	"log"
	"nhsoauthclose/config"
	"nhsoauthclose/nhso-claim/model"
	"nhsoauthclose/nhso-claim/service"
	"nhsoauthclose/repository"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// โหลดค่าจากไฟล์ .env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// เริ่มต้นการเชื่อมต่อฐานข้อมูล
	db := config.InitDB()

	// Migrate โครงสร้างตาราง
	err = db.AutoMigrate(&model.ClaimRequest{}, &model.ClaimResponse{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// สร้าง repository
	claimRepo := repository.NewClaimRepository(db)

	// กำหนดเงื่อนไขการ query
	dateFilter := "o.vstdate = CURDATE()"
	pttypeCondition := "AND T.pttype IN ('10', '11')"
	spcltyCondition := "AND o.cur_dep = '001'"
	hnCondition := ""

	// ดึงข้อมูล claim จาก database
	claims, err := claimRepo.FetchClaimData(dateFilter, pttypeCondition, spcltyCondition, hnCondition)
	if err != nil {
		log.Fatal("Error fetching claim data:", err)
	}

	if len(claims) == 0 {
		log.Println("No claim data found")
		return
	}

	// บันทึก claim ลงฐานข้อมูล
	for i := range claims {
		result := db.Create(&claims[i])
		if result.Error != nil {
			log.Printf("Failed to save claim: %v", result.Error)
			continue
		}
		log.Printf("Saved claim with TransactionID: %s", claims[i].TransactionID)
	}

	// ดึง claim ที่มีสถานะ PENDING
	pendingClaims, err := claimRepo.GetPendingClaims()
	if err != nil {
		log.Fatal("Error fetching pending claims:", err)
	}

	if len(pendingClaims) == 0 {
		log.Println("No pending claims found")
		return
	}

	// Process each pending claim
	for _, claim := range pendingClaims {
		// เตรียม payload ตามโครงสร้าง NHSO API
		payload := service.ClaimRequestPayload{
			Hcode:            claim.Hcode,
			MainInsclCode:    claim.MainInsclCode,
			ServiceDateTime:  claim.ServiceDateTime,
			InvoiceDateTime:  claim.InvoiceDateTime,
			TransactionID:    claim.TransactionID,
			TotalAmount:      claim.TotalAmount,
			PaidAmount:       claim.PaidAmount,
			PrivilegeAmount:  claim.PrivilegeAmount,
			ClaimServiceCode: claim.ClaimServiceCode,
			Pid:              claim.Pid,
			SourceID:         claim.SourceID,
			RecorderPid:      claim.RecorderPid,
		}

		// ส่ง claim ไปที่ NHSO API
		apiResponse, err := service.SendClaim(payload)
		if err != nil {
			log.Printf("SendClaim error for transaction %s: %v", claim.TransactionID, err)
			
			// อัปเดตสถานะเป็น FAILED
			claim.Status = model.StatusFailed
			claim.ResponseData = err.Error()
			claimRepo.SaveClaim(&claim)
			continue
		}

		// แปลง response เป็น JSON string
		responseData, err := json.Marshal(apiResponse)
		if err != nil {
			log.Printf("Error marshaling response: %v", err)
			responseData = []byte("{}")
		}

		// อัปเดตสถานะ claim
		if apiResponse.Success {
			claim.Status = model.StatusCompleted
			log.Printf("Claim processed successfully: %s", claim.TransactionID)
		} else {
			claim.Status = model.StatusFailed
			log.Printf("Claim failed: %s - %s", claim.TransactionID, apiResponse.Message)
		}
		
		claim.ResponseData = string(responseData)
		claimRepo.SaveClaim(&claim)

		// บันทึก response
		if apiResponse.Success {
			claimResponse := model.ClaimResponse{
				ClaimRequestID:  claim.ID,
				ResponseCode:    apiResponse.Code,
				ResponseData:    string(responseData),
				IsSuccess:       apiResponse.Success,
				ResponseMessage: apiResponse.Message,
			}
			db.Create(&claimResponse)
		}
	}

	log.Println("Claim processing completed")
}
EOF

# สร้างไฟล์ .env
cat > .env << EOF
# Database Configuration
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=nhso_claim

# Application Configuration
SOURCE_ID=BAAC01
HCODE=10660
RECORDER_PID=1234567890123

# NHSO API Configuration
NHSO_API_BASE_URL=https://test.nhso.go.th/nhsoendpoint/
NHSO_API_KEY=your_api_key_here
NHSO_API_TIMEOUT=30
EOF

# สร้างไฟล์ .gitignore
cat > .gitignore << EOF
# Environment variables
.env

# Binaries
nhsoauthclose
nhsoauthclose.exe

# Database
*.db
*.sqlite

# Logs
*.log
logs/

# IDE
.vscode/
.idea/
*.swp
*.swo
EOF

# ติดตั้ง dependencies
echo "ติดตั้ง Go dependencies..."
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
go mod tidy

# Build โปรเจค
echo "Build โปรเจค..."
go build -o nhsoauthclose

# ทำให้ไฟล์実行ได้
chmod +x nhsoauthclose

# สร้าง systemd service
cat > /etc/systemd/system/nhsoauthclose.service << EOF
[Unit]
Description=NHSO Claim Auto Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$PROJECT_DIR
ExecStart=$PROJECT_DIR/nhsoauthclose
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# โหลดและเริ่มต้น systemd service
systemctl daemon-reload
systemctl enable nhsoauthclose
systemctl start nhsoauthclose

echo "การติดตั้งเสร็จสมบูรณ์!"
echo "โปรแกรมถูกติดตั้งที่: $PROJECT_DIR"
echo "Systemd service ถูกสร้างและเริ่มต้นแล้ว"
echo "แก้ไขการตั้งค่าในไฟล์ .env ก่อนใช้งาน: $PROJECT_DIR/.env"
echo "ตรวจสอบสถานะ: systemctl status nhsoauthclose"
echo "ดู log: journalctl -u nhsoauthclose -f"
