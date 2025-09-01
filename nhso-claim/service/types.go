package service

// ClaimRequestPayload represents the payload for NHSO claim API
type ClaimRequestPayload struct {
	Hcode            string  `json:"hcode"`
	MainInsclCode    string  `json:"mainInsclCode"`
	ServiceDateTime  int64   `json:"serviceDateTime"`
	InvoiceDateTime  int64   `json:"invoiceDateTime"`
	TransactionID    string  `json:"transactionId"`
	TotalAmount      float64 `json:"totalAmount"`
	PaidAmount       float64 `json:"paidAmount"`
	PrivilegeAmount  float64 `json:"privilegeAmount"`
	ClaimServiceCode string  `json:"claimServiceCode"`
	Pid              string  `json:"pid"`
	SourceID         string  `json:"sourceId"`
	RecorderPid      string  `json:"recorderPid"`
}

// ClaimResponse represents the response from NHSO API
//type ClaimResponse struct {
	//Success bool   `json:"success"`
	//Code    string `json:"code"`
	//Message string `json:"message"`
	//Data    string `json:"data"`
//}

// ClaimResponse represents the response from NHSO API
type ClaimResponse struct {
	Seq        int64  `json:"seq"`
	AuthenCode string `json:"authenCode"`
	DataError  string `json:"dataError"`
}

// NHSOApiConfig represents the configuration for NHSO API
type NHSOApiConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}
