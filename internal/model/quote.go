package model

type Quote struct {
	Code      string  `json:"code"`
	Price     float64 `json:"price"`
	YP        float64 `json:"yp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
}

func (q Quote) GetCode() string  { return q.Code }
func (q Quote) GetPrice() float64 { return q.Price }
func (q Quote) GetYP() float64    { return q.YP }

type KlineBar struct {
	Ts int64   `json:"ts"`
	O  float64 `json:"o"`
	Cl float64 `json:"cl"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	V  float64 `json:"v"`
}

type KlineItem struct {
	C string     `json:"c"`
	K []KlineBar `json:"k"`
}
