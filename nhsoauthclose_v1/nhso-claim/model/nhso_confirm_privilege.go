// model/nhso_confirm_privilege.go
package model

import (
	"time"
)



type NHSOConfirmPrivilege struct {
    ID                  int       `gorm:"column:nhso_confirm_privilege_id;primaryKey" json:"id"`
    Vn                  string    `gorm:"column:vn" json:"vn"`
//    NhsoSeq             string    `gorm:"column:nhso_seq" json:"nhso_seq"`

	NhsoSeq              string    `gorm:"column:nhso_seq" json:"nhsoSeq"` // เปลี่ยนจาก int64 เป็น string
    NhsoAuthenCode      string    `gorm:"column:nhso_authen_code" json:"nhso_authen_code"`
    NhsoRequestJson     string    `gorm:"column:nhso_request_json" json:"nhso_request_json"`
//    NhsoReponseJson     string    `gorm:"column:nhso_reponse_json" json:"nhso_reponse_json"`   // ตรงกับ DB
//    NhsoRequstDatetime  time.Time `gorm:"column:nhso_requst_datetime" json:"nhso_requst_datetime"` // ตรงกับ DB
 // NhsoReponseJson    string    `gorm:"column:nhso_reponse_json"`    // ⚠ typo ต้องตรง DB


    NhsoReponseJson    string    `gorm:"column:nhso_reponse_json"`
    NhsoRequstDatetime time.Time `gorm:"column:nhso_requst_datetime"`
    NhsoResponseDatetime time.Time `gorm:"column:nhso_response_datetime"`

   // NhsoRequstDatetime time.Time `gorm:"column:nhso_requst_datetime"` // ⚠ typo ต้องตรง DB
  //  NhsoResponseDatetime time.Time `gorm:"column:nhso_response_datetime" json:"nhso_response_datetime"`
    ConfirmStaff        string    `gorm:"column:confirm_staff" json:"confirm_staff"`
    NhsoStatus          string    `gorm:"column:nhso_status" json:"nhso_status"`
    DebtId              int       `gorm:"column:debt_id" json:"debt_id"`
    NhsoTotalAmount     float64   `gorm:"column:nhso_total_amount" json:"nhso_total_amount"`
    NhsoCancelResponse  string    `gorm:"column:nhso_cancel_response" json:"nhso_cancel_response"`
    NhsoCancelDatetime  time.Time `gorm:"column:nhso_cancel_datetime" json:"nhso_cancel_datetime"`
    CancelStaff         string    `gorm:"column:cancel_staff" json:"cancel_staff"`
    NhsoConfirmTypeId   int       `gorm:"column:nhso_confirm_type_id" json:"nhso_confirm_type_id"`
    FdhSendStatus       string    `gorm:"column:fdh_send_status" json:"fdh_send_status"`
    FdhTransactionId    string    `gorm:"column:fdh_transaction_id" json:"fdh_transaction_id"`
    Pttype              string    `gorm:"column:pttype" json:"pttype"`
    NhsoPrivilegeAmount float64   `gorm:"column:nhso_privilege_amount" json:"nhso_privilege_amount"`
    NhsoCashAmount      float64   `gorm:"column:nhso_cash_amount" json:"nhso_cash_amount"`
}



func (NHSOConfirmPrivilege) TableName() string {
	return "nhso_confirm_privilege"
}
