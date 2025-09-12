package service

// NHSOApiConfig represents the NHSO API configuration
type NHSOApiConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}

// AuthRequestPayload represents the payload for auth code request
type AuthRequestPayload struct {
	SourceID    string   `json:"sourceId"`
	TransId     string   `json:"transId"`
	Pid         string   `json:"pid"`
	Phone       []string `json:"phone"`
    //Phone       string `json:"phone"`
	Hcode       string   `json:"hcode"`
	Hn          string   `json:"hn"`
	RecorderPid string   `json:"recorderPid"`
	ServiceCode string   `json:"serviceCode"`
	BirthDay    string   `json:"birthDay"`
	Maininscl   string   `json:"maininscl"`
}

// AuthResponse represents the response from NHSO Auth API
type AuthResponse struct {
	AuthCode  string `json:"authCode"`
	Message   string `json:"message"`
	DataError string `json:"dataError"`
}

// ClaimRequestPayload represents the payload for claim submission
type ClaimRequestPayload struct {
	Hcode            string  `json:"hcode"`
	MainInsclCode    string  `json:"mainInsclCode"`
//	ServiceDateTime  string  `json:"serviceDateTime"`
//	InvoiceDateTime  string  `json:"invoiceDateTime"
    ServiceDateTime  int64   `json:"serviceDateTime"`  // เปลี่ยนจาก string เป็น int64
    InvoiceDateTime  int64   `json:"invoiceDateTime"`  // เปลี่ยนจาก string เป็น int64`
	TransactionID    string  `json:"transactionId"`
	TotalAmount      float64 `json:"totalAmount"`
	PaidAmount       float64 `json:"paidAmount"`
	PrivilegeAmount  float64 `json:"privilegeAmount"`
	ClaimServiceCode string  `json:"claimServiceCode"`
	Pid              string  `json:"pid"`
	SourceID         string  `json:"sourceId"`
	RecorderPid      string  `json:"recorderPid"`
}

// ClaimResponse represents the response from NHSO Claim API
type ClaimResponse struct {
	Seq        int    `json:"seq"`
	AuthenCode string `json:"authenCode"`
	DataError  string `json:"dataError"`
	Message    string `json:"message"`
}
