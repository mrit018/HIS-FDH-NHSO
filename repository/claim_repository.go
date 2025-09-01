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
			T.hipdata_code  AS code_pttype,
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
			AND v.item_money != 0
		ORDER BY
			o.vsttime DESC
	`

	// สำหรับ PostgreSQL ต้องปรับ syntax บางส่วน
	if dbType == "postgres" || dbType == "postgresql" {
		// แทนที่ CONCAT ด้วย || สำหรับ PostgreSQL
		query = strings.Replace(query, "CONCAT(P.pname, P.fname, ' ', P.lname)", "P.pname || ' ' || P.fname || ' ' || P.lname", -1)

		// แทนที่ CAST AS CHAR ด้วย CAST AS VARCHAR
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
			&vstdate, &vsttime, &claim.Pid, &claim.Hn, &birthday,
			&claim.Vn, &claim.Ptname, &claim.Pttype, &claim.PttypeName,
			&claim.SpcltyName, &claim.DepartmentName, &claim.TotalAmount,
			&claim.AuthCode,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// ✅ ใช้ CombineToUnix (คืนค่าเป็นวินาที) แล้วคูณ 1000 ให้เป็น milliseconds
		serviceUnix, err := CombineToUnix(vstdate.Format(time.RFC3339), vsttime)
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
		claim.PaidAmount = 0
		claim.PrivilegeAmount = claim.TotalAmount

		// กำหนด claimServiceCode ตามประเภทบริการ
		claim.ClaimServiceCode = determineClaimServiceCode(claim.Pttype, claim.SpcltyName)

		// กำหนด mainInsclCode ตามประเภทสิทธิ
		claim.MainInsclCode = claim.Pttype

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
func CombineToUnix(dateStr, timeStr string) (int64, error) {
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
