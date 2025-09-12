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
