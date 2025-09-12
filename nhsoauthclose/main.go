package main

import (
	"encoding/json"
        "fmt"  // เพิ่มบรรทัดนี้
         "net/http"
	"log"
	"nhsoauthclose/config"
	"nhsoauthclose/nhso-claim/model"
	"nhsoauthclose/nhso-claim/service"
	"nhsoauthclose/repository"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)


// เพิ่มฟังก์ชันนี้
func checkAPIStatus() {
    config := service.GetNHSOApiConfig()
    
    // ทดสอบ connection ไปยัง API base URL
    resp, err := http.Get(config.BaseURL)
    if err != nil {
        log.Printf("API Connection Test Failed: %v", err)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 200 {
        log.Printf("API Connection Test Passed: %s is reachable", config.BaseURL)
    } else {
        log.Printf("API Connection Test Failed: Status %d", resp.StatusCode)
    }
}


func formatUnixMillis(millis int64) string {
    return time.Unix(millis/1000, 0).In(time.FixedZone("ICT", 7*3600)).Format("2006-01-02 15:04:05")
}


// เพิ่ม function helper
func parsePostgresTimestamp(timestamp int64) string {
    // ลองตรวจสอบว่าค่าเป็น milliseconds หรือ seconds
    currentTime := time.Now()
    
    // ถ้าค่า timestamp ใกล้กับ current Unix timestamp (เป็น seconds)
    if timestamp > 1600000000 && timestamp < 2000000000 {
        return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
    }
    
    // ถ้าค่า timestamp ใหญ่กว่า (เป็น milliseconds)
    if timestamp > 1600000000000 && timestamp < 2000000000000 {
        return time.Unix(timestamp/1000, 0).Format("2006-01-02 15:04:05")
    }
    
    // ถ้าไม่ใช่รูปแบบใดเลย ให้ใช้ค่าปัจจุบัน
    log.Printf("WARNING: Invalid timestamp %d, using current time", timestamp)
    return currentTime.Format("2006-01-02 15:04:05")
}



// เพิ่ม function สำหรับแปลง string datetime เป็น Unix timestamp
func parseDateTimeToUnix(datetimeStr string) int64 {
    layout := "2006-01-02 15:04:05"
    t, err := time.Parse(layout, datetimeStr)
    if err != nil {
        log.Printf("Error parsing datetime %s: %v", datetimeStr, err)
        return time.Now().Unix()
    }
    return t.Unix()
}


// หรือถ้าต้องการเป็น milliseconds
func parseDateTimeToUnixMillis(datetimeStr string) int64 {
    layout := "2006-01-02 15:04:05"
    t, err := time.Parse(layout, datetimeStr)
    if err != nil {
        log.Printf("Error parsing datetime %s: %v", datetimeStr, err)
        return time.Now().UnixMilli()
    }
    return t.UnixMilli()
}


// isLateNightMode ตรวจสอบว่าเป็นช่วงเวลา 23:50 ถึง 00:00 หรือไม่
func isLateNightMode() bool {
    currentTime := time.Now()
    currentHour := currentTime.Hour()
    currentMinute := currentTime.Minute()

	log.Printf("เวลาปัจจุบัน: %02d:%02d", currentHour, currentMinute)

    //
	// notifier.NotifyStep("SYSTEM","Process", "กลับมาใช้เงื่อนไข cur_dep ตามปกติ")	
    
    return (currentHour == 23 && currentMinute >= 50) || (currentHour == 0 && currentMinute == 0)
}

// isAfterMidnight ตรวจสอบว่าเป็นเวลาหลัง 00:01 หรือไม่
func isAfterMidnight() bool {
    currentTime := time.Now()
    currentHour := currentTime.Hour()
    currentMinute := currentTime.Minute()

	log.Printf("เวลาปัจจุบัน: %02d:%02d", currentHour, currentMinute)
    
    return currentHour == 0 && currentMinute >= 1
}



// เพิ่มฟังก์ชันนี้ใน main.go
//func testAPIsWithVN(vn string, claimRepo *repository.ClaimRepository, notifier *service.NotificationService) {
func testAPIsWithVN(vn string, claimRepo *repository.ClaimRepository, notifier *service.NotificationService) {
    log.Printf("=== Starting API Test for VN: %s ===", vn)
    
    // ใช้ method จาก repository เพื่อค้นหา claim
    claim, err := claimRepo.GetClaimByVN(vn)
    if err != nil {
        log.Printf("Test Failed: VN %s not found: %v", vn, err)
        return
    }
    
    // ทดสอบ Auth API
    log.Printf("Testing Auth API for VN: %s", vn)
    birthDayStr := claim.Birthday.Format("2006-01-02")
    
    // ตรวจสอบและกำหนดค่า default สำหรับ field ที่จำเป็น
    serviceCode := claim.ClaimServiceCode
    if serviceCode == "" {
        serviceCode = os.Getenv("DEFAULT_SERVICE_CODE")
        if serviceCode == "" {
            serviceCode = "10000"
        }
    }
    
    hcode := claim.Hcode
    if hcode == "" {
        hcode = os.Getenv("HCODE")
        if hcode == "" {
            log.Printf("Error: HCODE is required")
            return
        }
    }
    
    recorderPid := claim.RecorderPid
    if recorderPid == "" {
        recorderPid = os.Getenv("DEFAULT_RECORDER_PID")
        if recorderPid == "" {
            recorderPid = "2728936358179"
        }
    }
    
    authPayload := service.AuthRequestPayload{
        SourceID:    claim.SourceID,
        TransId:     claim.TransactionID,
        Pid:         claim.Pid,
        //Phone:       []string{"0000000000"},
		//Phone:       claim.Telephone,
		Phone:       []string{claim.Telephone}, // ใช้หมายเลขจาก claim.Telephone
        Hcode:       hcode,
        Hn:          claim.Hn,
        RecorderPid: recorderPid,
        ServiceCode: serviceCode,
        BirthDay:    birthDayStr,
        Maininscl:   claim.MainInsclCode,
    }
    
    authResponse, err := service.GetAuthCode(authPayload)
    if err != nil {
        log.Printf("Auth API Test Failed: %v", err)
        return
    }
    
    if authResponse.AuthCode == "" {
        log.Printf("Auth API Test Failed: No auth code received")
        return
    }
    
    log.Printf("Auth API Test Passed: Auth Code = %s", authResponse.AuthCode)
    
    // ทดสอบ Claim API
    log.Printf("Testing Claim API for VN: %s", vn)
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
    
    claimResponse, err := service.SendClaim(payload)
    if err != nil {
        log.Printf("Claim API Test Failed: %v", err)
        return
    }
    
    if claimResponse.DataError != "" {
        log.Printf("Claim API Test Failed: %s", claimResponse.DataError)
        return
    }
    
    log.Printf("Claim API Test Passed: Authen Code = %s, Seq = %d", 
        claimResponse.AuthenCode, claimResponse.Seq)
    log.Printf("=== API Test Completed for VN: %s ===", vn)
}


func main() {
	// โหลดค่าจากไฟล์ .env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}



	// เริ่มต้นการเชื่อมต่อฐานข้อมูล
	db := config.InitDB()

	// สร้าง notification service
	notifier := service.NewNotificationService()  // ประกาศ notifier

	// สร้าง repository
	claimRepo := repository.NewClaimRepository(db)  // ประกาศ claimRepo


       checkAPIStatus()

	// ตรวจสอบว่ามีการส่ง VN มาทดสอบหรือไม่
	testVN := ""
	if len(os.Args) > 1 {
		testVN = os.Args[1]
		log.Printf("Running in test mode for VN: %s", testVN)
	}




    // หลังจากสร้าง claimRepo และ notifier
    if testVN != "" {
        testAPIsWithVN(testVN, claimRepo, notifier)
        return // ออกหลังจากทดสอบเสร็จ
    }

    // ... code ต่อไป ...

	// เริ่มต้นการเชื่อมต่อฐานข้อมูล
	//db := config.InitDB()

	// สร้าง notification service
	//notifier := service.NewNotificationService()

	// Migrate โครงสร้างตาราง
	err = db.AutoMigrate(
		&model.ClaimRequest{},
		&model.ClaimResponse{},
		&model.NHSOConfirmPrivilege{}, // เพิ่มตารางใหม่
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
		notifier.NotifyError("", "", "SYSTEM", "Failed to migrate database", "Database Migration")
	}

	// สร้าง repository
	//claimRepo := repository.NewClaimRepository(db)


// สร้าง notification service
//notifier := service.NewNotificationService()

// ตรวจสอบว่ามีการส่ง VN มาทดสอบหรือไม่
//testVN := ""
//if len(os.Args) > 1 {
//    testVN = os.Args[1]
//    log.Printf("Running in test mode for VN: %s", testVN)
//    testAPIsWithVN(testVN, claimRepo, notifier)
//    return // ออกหลังจากทดสอบเสร็จ
//}


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
	//pttypeCondition := ""
	//spcltyCondition := "AND o.cur_dep = '999'"





	pttypeCondition := os.Getenv("PTTYPE_CONDITION")
	if pttypeCondition == "" {
		pttypeCondition = "AND T.pttype NOT IN ('10', '11')" // ค่า default
	}


	// ในฟังก์ชัน main
	//pttypeCondition := "AND T.pttype NOT IN ('10', '11')"
	spcltyCondition := "AND o.cur_dep = '999'" // ค่าเริ่มต้น

	if isLateNightMode() {
		spcltyCondition = "" // ไม่ใช้เงื่อนไข cur_dep
		log.Printf("อยู่ในโหมดปิดสิทธิ์ทุกคนที่ยังไม่ได้ปิด โดยไม่สนใจ cur_dep")
		notifier.NotifyStep("SYSTEM","Process","อยู่ในโหมดปิดสิทธิ์ทุกคนที่ยังไม่ได้ปิด โดยไม่สนใจ cur_dep")
	} else if isAfterMidnight() {
		log.Printf("กลับมาใช้เงื่อนไข cur_dep ตามปกติ")
		notifier.NotifyStep("SYSTEM","Process", "กลับมาใช้เงื่อนไข cur_dep ตามปกติ")
		// spcltyCondition ยังคงเป็น "AND o.cur_dep = '999'" อยู่แล้ว
	}	



		
	// กำหนดเงื่อนไข HN/VN สำหรับการทดสอบ
	hnCondition := ""
	if testVN != "" {
		hnCondition = "AND o.vn = '" + testVN + "'"
		log.Printf("Filtering for VN: %s", testVN)
}


	// ดึงข้อมูล claim จาก database
	notifier.NotifyStep("SYSTEM", "Database", "เริ่มต้นดึงข้อมูล claim จากฐานข้อมูล")
	claims, err := claimRepo.FetchClaimData(dateFilter, pttypeCondition, spcltyCondition, hnCondition)
	if err != nil {
		log.Fatal("Error fetching claim data:", err)
		notifier.NotifyError("", "", "SYSTEM", err.Error(), "Fetch Claim Data")
		return
	}

	if len(claims) == 0 {
		log.Println("No claim data found")
		notifier.NotifyStep("SYSTEM", "Database", "ไม่พบข้อมูล claim สำหรับวันนี้")
		return
	}

	// บันทึก claim ลงฐานข้อมูล
	for i := range claims {
		result := db.Create(&claims[i])
		if result.Error != nil {
			log.Printf("Failed to save claim: %v", result.Error)
			notifier.NotifyError(claims[i].Vn, claims[i].Hn, claims[i].TransactionID,
				result.Error.Error(), "Save Claim to Database")
			continue
		}
		log.Printf("Saved claim with TransactionID: %s", claims[i].TransactionID)
		notifier.NotifyStep(claims[i].Vn, "Database", "บันทึก claim ลงฐานข้อมูลสำเร็จ")
	}

	// ดึง claim ที่มีสถานะ PENDING
	notifier.NotifyStep("SYSTEM", "Database", "ดึงข้อมูล claim ที่มีสถานะ PENDING")
	pendingClaims, err := claimRepo.GetPendingClaims()
	if err != nil {
		log.Fatal("Error fetching pending claims:", err)
		notifier.NotifyError("", "", "SYSTEM", err.Error(), "Fetch Pending Claims")
		return
	}

	if len(pendingClaims) == 0 {
		log.Println("No pending claims found")
		notifier.NotifyStep("SYSTEM", "Database", "ไม่พบ claim ที่มีสถานะ PENDING")
		return
	}

	// Process each pending claim
	for _, claim := range pendingClaims {

	

		// หากอยู่ในโหมดทดสอบ และ VN ไม่ตรงกับที่ระบุ ให้ข้าม
		if testVN != "" && claim.Vn != testVN {
			continue
		}
		notifier.NotifyStep(claim.Vn, "Process", "เริ่มประมวลผล claim")


		// ✅ ตรวจสอบว่าต้องการ auth code หรือไม่
		needsAuthCode := claim.AuthCode == "" || strings.HasPrefix(claim.AuthCode, "EP")

		if needsAuthCode {
			notifier.NotifyStep(claim.Vn, "Auth", "ขอ auth code ใหม่จาก NHSO")

			// แปลง birthday เป็น string format YYYY-MM-DD
			birthDayStr := claim.Birthday.Format("2006-01-02")

			// ตรวจสอบและกำหนดค่า default สำหรับ field ที่จำเป็น
			serviceCode := claim.ClaimServiceCode
			if serviceCode == "" {
				serviceCode = os.Getenv("DEFAULT_SERVICE_CODE")
				if serviceCode == "" {
					serviceCode = "10000" // default service code
				}
			}

			maininscl := claim.MainInsclCode
			if maininscl == "" {
				maininscl = os.Getenv("DEFAULT_MAININSCL")
				if maininscl == "" {
					maininscl = "10" // default maininscl
				}
			}

			recorderPid := claim.RecorderPid
			if recorderPid == "" {
				recorderPid = os.Getenv("DEFAULT_RECORDER_PID")
				if recorderPid == "" {
					recorderPid = "2728936358179" // fallback default
				}
			}

			// ตรวจสอบ HCODE
			hcode := claim.Hcode
			if hcode == "" {
				hcode = os.Getenv("HCODE")
				if hcode == "" {
					log.Printf("Error: HCODE is required but not provided for transaction %s", claim.TransactionID)
					notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, "HCODE is required", "Get Auth Code")
					continue
				}
			}

			// ตรวจสอบ PID
			if claim.Pid == "" {
				log.Printf("Error: PID is required but not provided for transaction %s", claim.TransactionID)
				notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, "PID is required", "Get Auth Code")
				continue
			}

			// ตรวจสอบ HN
			if claim.Hn == "" {
				log.Printf("Error: HN is required but not provided for transaction %s", claim.TransactionID)
				notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, "HN is required", "Get Auth Code")
				continue
			}

			// เตรียม payload สำหรับขอ auth code ตามรูปแบบ API
			authPayload := service.AuthRequestPayload{
				SourceID:    claim.SourceID,
				TransId:     claim.TransactionID, // ใช้ transactionID เป็น transId (เป็น string)
				Pid:         claim.Pid,
				Phone:       []string{claim.Telephone}, // ใช้หมายเลขจาก claim.Telephone
            //    Phone:       claim.Telephone,
				Hcode:       hcode,
				Hn:          claim.Hn,
				RecorderPid: recorderPid,
				ServiceCode: serviceCode,
				BirthDay:    birthDayStr,
				Maininscl:   maininscl,
			}

			// Log payload สำหรับ debugging
			log.Printf("Auth payload for transaction %s: %+v", claim.TransactionID, authPayload)

			// เรียก API เพื่อขอ auth code
			authResponse, err := service.GetAuthCode(authPayload)
			if err != nil {
				log.Printf("GetAuthCode error for transaction %s: %v", claim.TransactionID, err)
				notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, err.Error(), "Get Auth Code")

				// อัปเดตสถานะเป็น FAILED
				claim.Status = model.StatusFailed
				claim.ResponseData = err.Error()
				claimRepo.SaveClaim(&claim)
				continue
			}

			// ตรวจสอบว่าได้ auth code มาหรือไม่
			if authResponse.AuthCode == "" {
				errorMsg := "Failed to get auth code"
				if authResponse.DataError != "" {
					errorMsg = authResponse.DataError
				} else if authResponse.Message != "" {
					errorMsg = authResponse.Message
				}

				log.Printf("Auth code not received for transaction %s: %s", claim.TransactionID, errorMsg)
				notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, errorMsg, "Get Auth Code")

				claim.Status = model.StatusFailed
				claim.ResponseData = errorMsg
				claimRepo.SaveClaim(&claim)
				continue
			}



			// บันทึก auth code ที่ได้
			claim.AuthCode = authResponse.AuthCode
			log.Printf("Got new auth code: %s for transaction: %s", authResponse.AuthCode, claim.TransactionID)
			notifier.NotifyStep(claim.Vn, "Auth", "ได้รับ auth code: "+authResponse.AuthCode)

			// ✅ อัปเดต auth_code ในตาราง visit_pttype

			err = claimRepo.UpdateVisitPttypeAuthCode(claim.Vn, claim.Pttype, authResponse.AuthCode)
			if err != nil {
				errorMsg := fmt.Sprintf("อัปเดต auth_code ในตาราง visit_pttype ไม่สำเร็จ: %v", err)
				log.Printf(errorMsg)
				notifier.NotifyStep(claim.Vn, "Database", errorMsg)
				// ยังคงดำเนินการต่อได้
			} else {
				successMsg := fmt.Sprintf("อัปเดต auth_code ในตาราง visit_pttype สำเร็จ: VN=%s, pttype=%s, auth_code=%s", 
					claim.Vn, claim.Pttype, authResponse.AuthCode)
				log.Printf(successMsg)
				notifier.NotifyStep(claim.Vn, "Database", successMsg)
			}

			// บันทึก claim ที่มี auth code ใหม่
			err = claimRepo.SaveClaim(&claim)
			if err != nil {
				log.Printf("Error saving claim with new auth code: %v", err)
			}

			// รอสักครู่ก่อนส่ง claim เพื่อป้องกันการเรียก API ถี่เกินไป
			time.Sleep(500 * time.Millisecond)
		}

		// เตรียม payload ตามโครงสร้าง NHSO API สำหรับส่ง claim
		payload := service.ClaimRequestPayload{
			Hcode:            claim.Hcode,
			MainInsclCode:    claim.MainInsclCode,
//			ServiceDateTime:  claim.ServiceDateTime,
//			InvoiceDateTime:  claim.InvoiceDateTime,
//    ServiceDateTime:  time.Unix(claim.ServiceDateTime, 0).Format("2006-01-02 15:04:05"),
//    InvoiceDateTime:  time.Unix(claim.InvoiceDateTime, 0).Format("2006-01-02 15:04:05"),
//    ServiceDateTime:  parsePostgresTimestamp(claim.ServiceDateTime),
//    InvoiceDateTime:  parsePostgresTimestamp(claim.InvoiceDateTime),
//    ServiceDateTime:  parseDateTimeToUnix(claim.ServiceDateTimeStr), // แปลงเป็น Unix timestamp
//    InvoiceDateTime:  parseDateTimeToUnix(claim.InvoiceDateTimeStr), // แปลงเป็น Unix timestamp
//    ServiceDateTime:  fmt.Sprintf("%d", parseDateTimeToUnix(claim.ServiceDateTimeStr)),
//    InvoiceDateTime:  fmt.Sprintf("%d", parseDateTimeToUnix(claim.InvoiceDateTimeStr)),
    ServiceDateTime:  claim.ServiceDateTime,  // ส่งเป็น int64 โดยตรง
    InvoiceDateTime:  claim.InvoiceDateTime,  // ส่งเป็น int64 โดยตรง

			TransactionID:    claim.TransactionID,
			TotalAmount:      claim.TotalAmount,
			PaidAmount:       claim.PaidAmount,
			PrivilegeAmount:  claim.PrivilegeAmount,
			ClaimServiceCode: claim.ClaimServiceCode,
			Pid:              claim.Pid,
			SourceID:         claim.SourceID,
			RecorderPid:      claim.RecorderPid,
		}



// เพิ่มก่อนส่ง API
log.Printf("ServiceDateTime (unix): %d", claim.ServiceDateTime)
log.Printf("InvoiceDateTime (unix): %d", claim.InvoiceDateTime)
log.Printf("ServiceDateTime (formatted): %s", time.Unix(claim.ServiceDateTime, 0).Format("2006-01-02 15:04:05"))
log.Printf("InvoiceDateTime (formatted): %s", time.Unix(claim.InvoiceDateTime, 0).Format("2006-01-02 15:04:05"))


// ใช้เช่น
log.Printf("ServiceDateTime: %s", formatUnixMillis(claim.ServiceDateTime))
log.Printf("InvoiceDateTime: %s", formatUnixMillis(claim.InvoiceDateTime))

		// แปลง payload เป็น JSON string สำหรับเก็บลงฐานข้อมูล
		requestData, err := json.Marshal(payload)

		if err != nil {
			log.Printf("Error marshaling request payload for transaction %s: %v", claim.TransactionID, err)
			notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, err.Error(), "Marshal Request Payload")
			continue
		}


// Log payload ที่จะส่งไปยัง API
log.Printf("Payload to send: %s", string(requestData))

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

    // ✅ อัปเดต auth_code ในตาราง visit_pttype ด้วย authenCode จาก response การส่ง claim
    if apiResponse.AuthenCode != "" {
        err = claimRepo.UpdateVisitPttypeAuthCode(claim.Vn, claim.Pttype, apiResponse.AuthenCode)
        if err != nil {
            log.Printf("Error updating visit_pttype auth_code with authenCode: %v", err)
            notifier.NotifyStep(claim.Vn, "Database", "อัปเดต auth_code ในตาราง visit_pttype ด้วย authenCode ไม่สำเร็จ: "+err.Error())
        } else {
            log.Printf("Updated visit_pttype auth_code with authenCode: %s", apiResponse.AuthenCode)
            notifier.NotifyStep(claim.Vn, "Database", "อัปเดต auth_code ในตาราง visit_pttype ด้วย authenCode สำเร็จ: "+apiResponse.AuthenCode)
        }
    }
    
    // ✅ ส่งการแจ้งเตือนความสำเร็จ
    notifier.NotifySuccess(claim.Vn, claim.Hn, claim.TransactionID, apiResponse.AuthenCode, claim.TotalAmount)




		} else {
			claim.Status = model.StatusFailed
			log.Printf("Claim failed: %s - %s", claim.TransactionID, apiResponse.DataError)


    // ✅ ส่งการแจ้งเตือนข้อผิดพลาด
    notifier.NotifyError(claim.Vn, claim.Hn, claim.TransactionID, apiResponse.DataError, "Claim Submission")

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
			Seq:             int64(apiResponse.Seq),         // เก็บ Seq
		}
		db.Create(&claimResponse)
		notifier.NotifyStep(claim.Vn, "Process", "ประมวลผล claim เสร็จสิ้น")

		// ✅ บันทึกข้อมูลลงตาราง nhso_confirm_privilege
		if apiResponse.DataError == "" {
			now := time.Now()
			nhsoRecord := &model.NHSOConfirmPrivilege{
				Vn:                   claim.Vn,
//				NhsoSeq:              fmt.Sprintf("%d", apiResponse.Seq),
//                NhsoSeq:              fmt.Sprintf("%d", apiResponse.Seq),  // แปลง int เป็น string
//NhsoSeq              string    `gorm:"column:nhso_seq" json:"nhsoSeq"`, // เปลี่ยนจาก int64 เป็น 

				NhsoAuthenCode:       apiResponse.AuthenCode,
				NhsoRequestJson:      string(requestData),
				NhsoReponseJson:      string(responseData),
				NhsoRequstDatetime:   now,
				NhsoResponseDatetime: now,
				ConfirmStaff:         getSystemUser(),
				NhsoStatus:           "1", // 1 = สำเร็จ, 0 = ไม่สำเร็จ
				NhsoTotalAmount:      claim.TotalAmount,
				Pttype:               claim.Pttype,
				NhsoPrivilegeAmount:  claim.PrivilegeAmount,
				NhsoCashAmount:       claim.TotalAmount - claim.PrivilegeAmount,
				FdhTransactionId:     claim.TransactionID,
			}

			err := claimRepo.SaveNHSOConfirmPrivilege(nhsoRecord)
			if err != nil {
				log.Printf("Warning: Failed to save NHSO confirm privilege record: %v", err)
			} else {
				log.Printf("Successfully saved NHSO confirm privilege for VN: %s", claim.Vn)
			}
		}
	}

	notifier.NotifyStep("SYSTEM", "Process", "การประมวลผล claim ทั้งหมดเสร็จสิ้น")
	log.Println("Claim processing completed")
}

// helper function สำหรับดึง system user
func getSystemUser() string {
	user := os.Getenv("SYSTEM_USER")
	if user == "" {
		return "SYSTEM"
	}
	return user
}
