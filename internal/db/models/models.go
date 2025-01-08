package models

// StatusWork - статус работы сканеров
type StatusWork struct {
	ActiveScan  string `json:"active_scan"`
	PassiveScan string `json:"passive_scan"`
	FinalResult int `json:"final_result"`
}

// PassiveScan - данные о пассивном сканировании
type PassiveScan struct {
	ID                 int    `json:"id"`
	Processed          string `json:"processed"`
	StatusReason       string `json:"StatusReason"`
	Method             string `json:"method"`
	ReasonNotProcessed string `json:"reason_not_processed"`
	MessageId          string `json:"message_id"`
	Link               string `json:"link"`
	StatusCode         string `json:"status_code"`
}

// ActiveScan - данные об активном сканировании
type ActiveScan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CweID       string `json:"cweid"`
	Risk        string `json:"risk"`
	Method      string `json:"method"`
	Link        string `json:"link"`
	Description string `json:"description"`
	Solution    string `json:"solution"`
}
