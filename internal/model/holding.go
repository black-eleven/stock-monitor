package model

type Holding struct {
	Symbol  string  `json:"symbol"`
	Name    string  `json:"name"`
	Shares  float64 `json:"shares"`
	AvgCost float64 `json:"avgCost"`
	BuyDate string  `json:"buyDate"`
}
