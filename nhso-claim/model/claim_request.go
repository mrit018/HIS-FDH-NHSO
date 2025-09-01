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
	InvoiceDateTime int64     `gorm:"not null" json:"invoiceDateTime"` // Unix timestamp in milliseconds
	TransactionID   string    `gorm:"size:255;not null;unique" json:"transactionId"`
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
        Pttype_hipdata_code         string    `gorm:"size:10" json:"pttype_hipdata_code"`
	PttypeName     string    `gorm:"size:100" json:"pttypeName"`
	SpcltyName     string    `gorm:"size:100" json:"spcltyName"`
	DepartmentName string    `gorm:"size:100" json:"departmentName"`
	AuthCode       string    `gorm:"size:50" json:"authCode"`
}

// TableName specifies the table name for ClaimRequest
func (ClaimRequest) TableName() string {
	return "claim_requests"
}
