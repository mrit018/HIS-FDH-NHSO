package model

import (
	"gorm.io/gorm"
)

type ClaimResponse struct {
	gorm.Model
	ClaimRequestID  uint   `gorm:"not null" json:"claimRequestId"`
	ResponseCode    string `gorm:"size:4" json:"responseCode"`
	ResponseData    string `gorm:"type:text" json:"responseData"`
	IsSuccess       bool   `gorm:"default:false" json:"isSuccess"`
	ResponseMessage string `gorm:"type:text" json:"responseMessage"` // เปลี่ยนจาก size:255 เป็น type:text
	AuthenCode      string `gorm:"size:50" json:"authenCode"`        // เพิ่มฟิลด์เก็บ AuthenCode
	Seq             int64  `gorm:"default:0" json:"seq"`             // เพิ่มฟิลด์เก็บ Seq
}

// TableName specifies the table name for ClaimResponse
func (ClaimResponse) TableName() string {
	return "claim_responses"
}
