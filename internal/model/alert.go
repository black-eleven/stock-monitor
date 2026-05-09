package model

type AlertRule struct {
	ID              int     `json:"id"`
	Symbol          string  `json:"symbol"`
	Type            string  `json:"type"` // "above" | "below" | "change_pct"
	Value           float64 `json:"value"`
	Enabled         bool    `json:"enabled"`
	CreatedAt       string  `json:"createdAt"`
	LastTriggeredAt *string `json:"lastTriggeredAt"`
}

type AlertLog struct {
	ID          int    `json:"id"`
	AlertID     int    `json:"alertId"`
	Symbol      string `json:"symbol"`
	Price       float64 `json:"price"`
	Message     string `json:"message"`
	TriggeredAt string `json:"triggeredAt"`
}
