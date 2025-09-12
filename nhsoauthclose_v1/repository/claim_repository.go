package repository

import (
	"fmt"
	"log"
	"nhsoauthclose/nhso-claim/model"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ClaimRepository struct {
	db *gorm.DB
}

func NewClaimRepository(db *gorm.DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}






// เพิ่ม method นี้
func (r *ClaimRepository) UpdateVisitPttypeAuthCode(vn string, pttype string, authCode string) error {
	result := r.db.Exec(`
		UPDATE visit_pttype 
		SET auth_code = ? 
		WHERE vn = ? AND pttype = ?
	`, authCode, vn, pttype)
	
	return result.Error
}




// FetchClaimData ดึงข้อมูลจากฐานข้อมูลตาม query ที่กำหนด
func (r *ClaimRepository) FetchClaimData(dateFilter, pttypeCondition, spcltyCondition, hnCondition string) ([]model.ClaimRequest, error) {
	var claims []model.ClaimRequest

	// ตรวจสอบประเภทฐานข้อมูล
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "mysql" // ค่าเริ่มต้น
	}

	// ใช้ RAW SQL query ตามที่ให้มา
	query := `
		SELECT
			o.vstdate,
			o.vsttime,
			P.CID,
			o.hn,
			P.birthday,
			o.vn,
			CAST(CONCAT(P.pname, P.fname, ' ', P.lname) AS CHAR(250)) AS ptname,
                        vpt.pttype,
			T.hipdata_code  AS code_pttype,
			T.NAME AS name_pttype,
			s.NAME AS spclty_name,
			K.department AS department_name,
    v.paid_money AS paidAmount,
    v.uc_money  AS privilegeAmount,
    v.item_money AS totalAmount,



			vpt.auth_code,

 CASE 
          WHEN REGEXP_REPLACE(COALESCE(P.mobile_phone_number, ''), '[^0-9]', '', 'g') != '' 
              THEN REGEXP_REPLACE(P.mobile_phone_number, '[^0-9]', '', 'g') 
          WHEN REGEXP_REPLACE(COALESCE(P.hometel, ''), '[^0-9]', '', 'g') != '' 
              THEN REGEXP_REPLACE(P.hometel, '[^0-9]', '', 'g')
          WHEN REGEXP_REPLACE(COALESCE(P.informtel, ''), '[^0-9]', '', 'g') != '' 
              THEN REGEXP_REPLACE(P.informtel, '[^0-9]', '', 'g')
          ELSE '0'
        END AS telephone
		FROM
			ovst o
			LEFT OUTER JOIN vn_stat v ON v.vn = o.vn
			LEFT OUTER JOIN opdscreen oc ON oc.vn = o.vn
			LEFT OUTER JOIN patient P ON P.hn = o.hn
			LEFT OUTER JOIN pttype T ON T.pttype = o.pttype
			LEFT OUTER JOIN spclty s ON s.spclty = o.spclty
			LEFT OUTER JOIN kskdepartment K ON K.depcode = o.cur_dep
			LEFT OUTER JOIN visit_pttype vpt ON vpt.vn = o.vn 
				AND vpt.pttype = o.pttype
		WHERE
			` + dateFilter + `
			` + pttypeCondition + `
			` + spcltyCondition + `
			` + hnCondition + `
			AND P.nationality IN ('99') 
			AND P.citizenship IN ('99') 
			AND o.cur_dep IN ('999') 
			AND P.CID NOT LIKE '0%'
                        AND v.item_money  != 0
                        AND vpt.auth_code NOT LIKE 'EP%'
		ORDER BY
			o.vsttime DESC
	`

	// สำหรับ PostgreSQL ต้องปรับ syntax บางส่วน
	if dbType == "postgres" || dbType == "postgresql" {
		// แทนที่ CONCAT ด้วย || สำหรับ PostgreSQL
		query = strings.Replace(query, "CONCAT(P.pname, P.fname, ' ', P.lname)", "P.pname || ' ' || P.fname || ' ' || P.lname", -1)

		// แทน이 CAST AS CHAR ด้วย CAST AS VARCHAR
		query = strings.Replace(query, "AS CHAR(250)", "AS VARCHAR(250)", -1)
	}

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
            &vstdate,           // 1. o.vstdate
            &vsttime,           // 2. o.vsttime
            &claim.Pid,         // 3. P.CID
            &claim.Hn,          // 4. o.hn
            &birthday,          // 5. P.birthday
            &claim.Vn,          // 6. o.vn
            &claim.Ptname,      // 7. ptname
            &claim.Pttype,      // 8. code_pttype (T.hipdata_code)
            &claim.HipdataCode,
            &claim.PttypeName,  // 9. name_pttype (T.NAME)
            &claim.SpcltyName,  // 10. spclty_name (s.NAME)
            &claim.DepartmentName, // 11. department_name (K.department)
            &claim.PaidAmount,
            &claim.PrivilegeAmount,
            &claim.TotalAmount, // 12. totalAmount (v.item_money)

            &claim.AuthCode,    // 13. auth_code (vpt.auth_code)
            &claim.Telephone,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// ✅ ใช้ CombineToUnix (คืนค่าเป็นวินาที) แล้วคูณ 1000 ให้เป็น milliseconds
//		serviceUnix, err := CombineToUnix(vstdate.Format(time.RFC3339), vsttime)
                serviceUnix, err := CombineToUnix(vstdate.Format("2006-01-02"), vsttime)
		if err != nil {
			log.Printf("Error combining date/time: %v", err)
			continue
		}
		claim.ServiceDateTime = serviceUnix * 1000

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
		claim.PaidAmount =  claim.PrivilegeAmount
		claim.PrivilegeAmount = claim.TotalAmount

		// กำหนด claimServiceCode ตามประเภทบริการ
		claim.ClaimServiceCode = determineClaimServiceCode(claim.Pttype, claim.SpcltyName)

		// กำหนด mainInsclCode ตามประเภทสิทธิ
		claim.MainInsclCode = claim.HipdataCode

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

// CombineToUnix รวมวันที่ (RFC3339) + เวลา (HH:mm:ss) แล้วคืนค่า Unix timestamp (เวลาไทย)
func CombineToUnixOLD(dateStr, timeStr string) (int64, error) {
	// โหลด timezone Bangkok
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return 0, err
	}

	// parse dateStr เป็น UTC ก่อน
	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, err
	}

	// แปลงเป็นเวลาไทย
	dateInThai := date.In(loc)

	// รวมวันที่ไทย + เวลา
	fullStr := fmt.Sprintf("%s %s", dateInThai.Format("2006-01-02"), timeStr)

	// parse เป็นเวลาไทย
	layout := "2006-01-02 15:04:05"
	visitTime, err := time.ParseInLocation(layout, fullStr, loc)
	if err != nil {
		return 0, err
	}

	return visitTime.Unix(), nil
}


// CombineToUnix รวมวันที่ (RFC3339) + เวลา (HH:mm:ss) แล้วคืนค่า Unix timestamp (เวลาไทย)
func CombineToUnix(dateStr, timeStr string) (int64, error) {
	// โหลด timezone Bangkok
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return 0, err
	}

	// parse dateStr โดยตรงในรูปแบบ "2006-01-02" (ไม่ใช่ RFC3339)
	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return 0, err
	}

	// รวมวันที่ไทย + เวลา
	fullStr := fmt.Sprintf("%s %s", date.Format("2006-01-02"), timeStr)

	// parse เป็นเวลาไทย
	layout := "2006-01-02 15:04:05"
	visitTime, err := time.ParseInLocation(layout, fullStr, loc)
	if err != nil {
		return 0, err
	}

	return visitTime.Unix(), nil
}



// SaveNHSOConfirmPrivilegeWithGORM บันทึกข้อมูลโดยใช้ GORM (วิธีแนะนำ)
func (r *ClaimRepository) SaveNHSOConfirmPrivilege(data *model.NHSOConfirmPrivilege) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. ดึง serial
		var serialID int
		if err := tx.Raw(`SELECT get_serialnumber('nhso_confirm_privilege_id') AS cc`).Scan(&serialID).Error; err != nil {
			return err
		}
		data.ID = serialID

		// 2. Upsert nhso_confirm_privilege
		query := `
		INSERT INTO nhso_confirm_privilege (
			nhso_confirm_privilege_id, vn, nhso_seq, nhso_authen_code,
			nhso_request_json, nhso_reponse_json,
			nhso_requst_datetime, nhso_response_datetime,
			confirm_staff, nhso_status, debt_id, nhso_total_amount,
			nhso_cancel_response, nhso_cancel_datetime, cancel_staff,
			nhso_confirm_type_id, fdh_send_status, fdh_transaction_id,
			pttype, nhso_privilege_amount, nhso_cash_amount
		) VALUES (
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?
		)
		ON DUPLICATE KEY UPDATE
			nhso_reponse_json    = VALUES(nhso_reponse_json),
			nhso_response_datetime = VALUES(nhso_response_datetime),
			nhso_status          = VALUES(nhso_status),
			nhso_total_amount    = VALUES(nhso_total_amount),
			nhso_cancel_response = VALUES(nhso_cancel_response),
			nhso_cancel_datetime = VALUES(nhso_cancel_datetime),
			cancel_staff         = VALUES(cancel_staff),
			fdh_send_status      = VALUES(fdh_send_status),
			nhso_privilege_amount = VALUES(nhso_privilege_amount),
			nhso_cash_amount     = VALUES(nhso_cash_amount)
		`

		if err := tx.Exec(query,
			data.ID,
			data.Vn,
			data.NhsoSeq,
			data.NhsoAuthenCode,
			data.NhsoRequestJson,
			data.NhsoReponseJson,
			data.NhsoRequstDatetime,
			data.NhsoResponseDatetime,
			data.ConfirmStaff,
			data.NhsoStatus,
			data.DebtId,
			data.NhsoTotalAmount,
			data.NhsoCancelResponse,
			data.NhsoCancelDatetime,
			data.CancelStaff,
			data.NhsoConfirmTypeId,
			data.FdhSendStatus,
			data.FdhTransactionId,
			data.Pttype,
			data.NhsoPrivilegeAmount,
			data.NhsoCashAmount,
		).Error; err != nil {
			return err
		}

		// 3. Update visit_pttype
		if err := tx.Exec(`
			UPDATE visit_pttype
			SET auth_code = ?
			WHERE vn = ? AND auth_code LIKE 'PP%'`,
			data.NhsoAuthenCode, data.Vn,
		).Error; err != nil {
			return err
		}

		return nil
	})
}




// UpdateNHSOConfirmPrivilege อัปเดตข้อมูลการยืนยันสิทธิ
func (r *ClaimRepository) UpdateNHSOConfirmPrivilege(data *model.NHSOConfirmPrivilege) error {
    result := r.db.Save(data)
    if result.Error != nil {
        return fmt.Errorf("error updating NHSO confirm privilege: %v", result.Error)
    }
    return nil
}

// GetNHSOConfirmPrivilegeByVN ดึงข้อมูลการยืนยันสิทธิโดยใช้ VN
func (r *ClaimRepository) GetNHSOConfirmPrivilegeByVN(vn string) (*model.NHSOConfirmPrivilege, error) {
    var record model.NHSOConfirmPrivilege
    err := r.db.Where("vn = ?", vn).First(&record).Error
    if err != nil {
        return nil, err
    }
    return &record, nil
}

// GetNHSOConfirmPrivilegeByTransactionID ดึงข้อมูลโดยใช้ Transaction ID
func (r *ClaimRepository) GetNHSOConfirmPrivilegeByTransactionID(transactionID string) (*model.NHSOConfirmPrivilege, error) {
    var record model.NHSOConfirmPrivilege
    err := r.db.Where("fdh_transaction_id = ?", transactionID).First(&record).Error
    if err != nil {
        return nil, err
    }
    return &record, nil
}

func (r *ClaimRepository) GetClaimByVN(vn string) (*model.ClaimRequest, error) {
    var claim model.ClaimRequest
	err := r.db.Where("vn = ?", vn).First(&claim).Error
    if err != nil {
        return nil, err
    }
    return &claim, nil
}
