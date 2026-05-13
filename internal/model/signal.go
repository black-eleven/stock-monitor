package model

type SignalRecord struct {
	Symbol     string  `json:"symbol"`
	Date       string  `json:"date"`
	BuyScore   float64 `json:"buyScore"`
	BuyPct     float64 `json:"buyPct"`
	SellScore  float64 `json:"sellScore"`
	SellPct    float64 `json:"sellPct"`
	BuyCount   int     `json:"buyCount"`
	SellCount  int     `json:"sellCount"`
}

type SignalHistoryReq struct {
	Date      string  `json:"date"`
	BuyScore  float64 `json:"buyScore"`
	BuyPct    float64 `json:"buyPct"`
	SellScore float64 `json:"sellScore"`
	SellPct   float64 `json:"sellPct"`
	BuyCount  int     `json:"buyCount"`
	SellCount int     `json:"sellCount"`
}

type SignalAlert struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Type      string  `json:"type"` // "buy" or "sell"
	OldPct    float64 `json:"oldPct"`
	NewPct    float64 `json:"newPct"`
	Message   string  `json:"message"`
}
