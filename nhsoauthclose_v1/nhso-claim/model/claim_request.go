package model

import (
	"time"

	"gorm.io/gorm"
)

type ClaimRequest struct {
	gorm.Model
	Hcode           string    `gorm:"size:5;not null" json:"hcode"`
	MainInsclCode   string    `gorm:"size:3;not null" json:"mainInsclCode"`
	ServiceDateTime int64     `gorm:"not null" json:"serviceDateTime"` // Unix timestamp in milliseconds
//    ServiceDateTimeStr string `gorm:"column:service_date_time_str" json:"serviceDateTimeStr"` // เก็บเป็น string
	InvoiceDateTime int64     `gorm:"not null" json:"invoiceDateTime"` // Unix timestamp in milliseconds
//    InvoiceDateTimeStr string `gorm:"column:invoice_date_time_str" json:"invoiceDateTimeStr"` // เก็บเป็น string
//	TransactionID   string    `gorm:"size:255;not null;unique" json:"transactionId"
        TransactionID   string    `gorm:"size:255;not null" json:"transactionId"`
	TotalAmount     float64   `gorm:"not null" json:"totalAmount"`
	PaidAmount      float64   `gorm:"not null" json:"paidAmount"`
	PrivilegeAmount float64   `gorm:"not null" json:"privilegeAmount"`
	ClaimServiceCode string   `gorm:"size:20;not null" json:"claimServiceCode"`
	Pid             string    `gorm:"size:13;not null" json:"pid"`
	SourceID        string    `gorm:"size:50;not null" json:"sourceId"`
	RecorderPid     string    `gorm:"size:13;not null" json:"recorderPid"`
	Status          string    `gorm:"size:20;default:'PENDING'" json:"status"`
	ResponseData    string    `gorm:"type:text" json:"responseData"`
        RequestData     string    `gorm:"type:text" json:"requestData"` // เพิ่มฟิลด์นี้สำหรับเก็บ requ
	
	// Fields from database query
	Hn             string    `gorm:"size:50" json:"hn"`
	Birthday       time.Time `json:"birthday"`
	Vn             string    `gorm:"size:50" json:"vn"`
	Ptname         string    `gorm:"size:250" json:"ptname"`
	Pttype         string    `gorm:"size:10" json:"pttype"`
        //Hipdata_code   string    `gorm:"size:10" json:"pttype_hipdata_code"`
        HipdataCode string `gorm:"column:pttype_hipdata_code" json:"hipdataCode"`
	PttypeName     string    `gorm:"size:100" json:"pttypeName"`
	SpcltyName     string    `gorm:"size:100" json:"spcltyName"`
	DepartmentName string    `gorm:"size:100" json:"departmentName"`
	AuthCode       string    `gorm:"size:50" json:"authCode"`
 Telephone       string    `gorm:"size:50" json:"telephone"`
}

// TableName specifies the table name for ClaimRequest
func (ClaimRequest) TableName() string {
	return "claim_requests"
}
