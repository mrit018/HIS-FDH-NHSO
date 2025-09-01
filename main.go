package main

import (
	"encoding/json"
	"log"
	"nhsoauthclose/config"
	"nhsoauthclose/nhso-claim/model"
	"nhsoauthclose/nhso-claim/service"
	"nhsoauthclose/repository"
	"os"

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

	// ตรวจสอบประเภทฐานข้อมูลและกำหนดเงื่อนไขวันที่
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "mysql" // ค่าเริ่มต้น
	}

	// กำหนดเงื่อนไขการ query
	var dateFilter string
	if dbType == "postgres" || dbType == "postgresql" {
		dateFilter = "o.vstdate = CURRENT_DATE"
	} else {
		dateFilter = "o.vstdate = CURDATE()"
	}
	
	//pttypeCondition := "AND T.pttype IN ('10', '11')"
        pttypeCondition := ""
	spcltyCondition := "AND o.cur_dep = '999'"
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

		// แปลง payload เป็น JSON string สำหรับเก็บลงฐานข้อมูล
		requestData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshaling request payload for transaction %s: %v", claim.TransactionID, err)
			continue
		}

		// เก็บ request payload ลงในฐานข้อมูล
		claim.RequestData = string(requestData)
		err = claimRepo.SaveClaim(&claim)
		if err != nil {
			log.Printf("Error saving request data for transaction %s: %v", claim.TransactionID, err)
			continue
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

		// อัปเดตสถานะ claim ตาม response จาก NHSO API
		// ตรวจสอบจาก dataError field ใน response
		if apiResponse.DataError == "" {
			claim.Status = model.StatusCompleted
			log.Printf("Claim processed successfully: %s, AuthenCode: %s", claim.TransactionID, apiResponse.AuthenCode)
		} else {
			claim.Status = model.StatusFailed
			log.Printf("Claim failed: %s - %s", claim.TransactionID, apiResponse.DataError)
		}
		
		claim.ResponseData = string(responseData)
		claimRepo.SaveClaim(&claim)

		// บันทึก response
		claimResponse := model.ClaimResponse{
			ClaimRequestID:  claim.ID,
			ResponseCode:    "", // NHSO API ใหม่ไม่ได้ส่ง response code แบบเดิม
			ResponseData:    string(responseData),
			IsSuccess:       apiResponse.DataError == "",
			ResponseMessage: apiResponse.DataError, // ใช้ DataError เป็น message
			AuthenCode:      apiResponse.AuthenCode, // เก็บ AuthenCode
			Seq:             apiResponse.Seq,        // เก็บ Seq
		}
		db.Create(&claimResponse)
	}

	log.Println("Claim processing completed")
}
