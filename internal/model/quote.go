package model

import "github.com/black-eleven/stock-monitor/internal/eastmoney"

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

func FromEMQuote(q eastmoney.Quote) Quote {
	return Quote{
		Code: q.Code, Price: q.Price, YP: q.YP,
		Open: q.Open, High: q.High, Low: q.Low,
		Volume: q.Volume, Turnover: q.Turnover,
		Timestamp: q.Timestamp, Status: q.Status,
	}
}

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

type Fundamentals struct {
	Code            string  `json:"code"`
	PE              float64 `json:"pe"`
	PB              float64 `json:"pb"`
	MarketCap       float64 `json:"marketCap"`
	CirculatingCap  float64 `json:"circulatingCap"`
	ROE             float64 `json:"roe"`
	NAVPerShare     float64 `json:"navPerShare"`
	Industry        string  `json:"industry"`
	Revenue         float64 `json:"revenue"`
	NetProfitGrowth float64 `json:"netProfitGrowth"`
	RevenueGrowth   float64 `json:"revenueGrowth"`
}

func FromEMFundamentals(f eastmoney.Fundamentals) Fundamentals {
	return Fundamentals{
		Code: f.Code, PE: f.PE, PB: f.PB,
		MarketCap: f.MarketCap, CirculatingCap: f.CirculatingCap,
		ROE: f.ROE, NAVPerShare: f.NAVPerShare,
		Industry: f.Industry, Revenue: f.Revenue,
		NetProfitGrowth: f.NetProfitGrowth, RevenueGrowth: f.RevenueGrowth,
	}
}
