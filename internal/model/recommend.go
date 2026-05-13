package model

type RecommendReq struct {
	Industry string `json:"industry"`
}

type Recommendation struct {
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	Score         float64  `json:"score"`
	NewsCount     int      `json:"newsCount"`
	Price         float64  `json:"price"`
	ChangePercent float64  `json:"changePercent"`
	Highlights    []string `json:"highlights"`
	Rank          int      `json:"rank"`
}

type RecommendResp struct {
	Recommendations []Recommendation `json:"recommendations"`
}
