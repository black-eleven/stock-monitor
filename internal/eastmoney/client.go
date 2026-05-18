package eastmoney

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const eastMoneyReferer = "https://quote.eastmoney.com/"

// Quote represents a real-time stock quote.
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

// QuoteClient is the interface for fetching stock data.
type QuoteClient interface {
	FetchQuoteCached(code string) (*Quote, error)
	FetchHistoryKlineCached(code string, kt int, count int) ([]json.RawMessage, error)
}

type Client struct {
	httpClient   *http.Client
	baseURL      string
	klineBaseURL string

	mu      sync.RWMutex
	tracked []string
}

func NewClient() *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		baseURL:      "http://push2.eastmoney.com/api/qt/stock/get",
		klineBaseURL: "http://push2his.eastmoney.com/api/qt/stock/kline/get",
	}
}

func (c *Client) doGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", eastMoneyReferer)
	return c.httpClient.Do(req)
}

// SetTrackedCodes sets the list of codes to poll for real-time quotes.
func (c *Client) SetTrackedCodes(codes []string) {
	c.mu.Lock()
	c.tracked = append([]string{}, codes...)
	c.mu.Unlock()
}

// GetTrackedCodes returns the current tracked codes.
func (c *Client) GetTrackedCodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string{}, c.tracked...)
}

// BatchFetchQuotes fetches quotes for multiple codes at once using EastMoney API.
func (c *Client) BatchFetchQuotes(codes []string) (map[string]*Quote, error) {
	if len(codes) == 0 {
		return map[string]*Quote{}, nil
	}

	secIDs := make([]string, 0, len(codes))
	codeMap := make(map[string]string, len(codes)) // secid to qosCode
	for _, code := range codes {
		secID, err := toSecID(code)
		if err != nil {
			continue
		}
		secIDs = append(secIDs, secID)
		codeMap[secID] = code
	}

	if len(secIDs) == 0 {
		return map[string]*Quote{}, nil
	}

	fields := "f43,f44,f45,f46,f47,f48,f57,f60"
	url := fmt.Sprintf("%s?secid=%s&fields=%s", c.baseURL, strings.Join(secIDs, ","), fields)

	resp, err := c.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("eastmoney batch quote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eastmoney batch quote read: %w", err)
	}

	return c.parseQuoteResponse(body, codeMap)
}

// parseQuoteResponse handles both batch format (data.diff[]) and single-item format.
func (c *Client) parseQuoteResponse(body []byte, codeMap map[string]string) (map[string]*Quote, error) {
	// Try batch format: data.diff[]
	var batchResp struct {
		RC   int `json:"rc"`
		Data struct {
			Total int               `json:"total"`
			Diff  []json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &batchResp); err == nil && batchResp.RC == 0 && len(batchResp.Data.Diff) > 0 {
		return c.parseDiffItems(batchResp.Data.Diff, codeMap), nil
	}

	// Try single-item format: data is a flat object with f57 code
	var singleResp struct {
		RC   int `json:"rc"`
		Data struct {
			F43 float64 `json:"f43"`
			F44 float64 `json:"f44"`
			F45 float64 `json:"f45"`
			F46 float64 `json:"f46"`
			F47 float64 `json:"f47"`
			F48 float64 `json:"f48"`
			F57 string  `json:"f57"`
			F60 float64 `json:"f60"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &singleResp); err != nil || singleResp.RC != 0 || singleResp.Data.F57 == "" {
		return map[string]*Quote{}, nil
	}

	// Match code (f57) against known secid suffixes in codeMap
	for secID, qosCode := range codeMap {
		if strings.HasSuffix(secID, "."+singleResp.Data.F57) {
			return map[string]*Quote{qosCode: {
				Code:      qosCode,
				Price:     singleResp.Data.F43 / 100,
				YP:        singleResp.Data.F60 / 100,
				Open:      singleResp.Data.F46 / 100,
				High:      singleResp.Data.F44 / 100,
				Low:       singleResp.Data.F45 / 100,
				Volume:    singleResp.Data.F47,
				Turnover:  singleResp.Data.F48,
				Timestamp: time.Now().Unix(),
				Status:    "OK",
			}}, nil
		}
	}
	return map[string]*Quote{}, nil
}

func (c *Client) parseDiffItems(items []json.RawMessage, codeMap map[string]string) map[string]*Quote {
	quotes := make(map[string]*Quote, len(items))
	ts := time.Now().Unix()

	for _, item := range items {
		var d struct {
			F43 float64 `json:"f43"`
			F44 float64 `json:"f44"`
			F45 float64 `json:"f45"`
			F46 float64 `json:"f46"`
			F47 float64 `json:"f47"`
			F48 float64 `json:"f48"`
			F57 string  `json:"f57"`
			F60 float64 `json:"f60"`
		}
		if err := json.Unmarshal(item, &d); err != nil {
			continue
		}

		var qosCode string
		for secID, code := range codeMap {
			if strings.HasSuffix(secID, "."+d.F57) {
				qosCode = code
				break
			}
		}
		if qosCode == "" {
			continue
		}

		quotes[qosCode] = &Quote{
			Code:      qosCode,
			Price:     d.F43 / 100,
			YP:        d.F60 / 100,
			Open:      d.F46 / 100,
			High:      d.F44 / 100,
			Low:       d.F45 / 100,
			Volume:    d.F47,
			Turnover:  d.F48,
			Timestamp: ts,
			Status:    "OK",
		}
	}
	return quotes
}

// FetchQuote fetches a single quote via EastMoney API.
func (c *Client) FetchQuote(code string) (*Quote, error) {
	quotes, err := c.BatchFetchQuotes([]string{code})
	if err != nil {
		return nil, err
	}
	if q, ok := quotes[code]; ok {
		return q, nil
	}
	return nil, fmt.Errorf("no quote data for %s", code)
}

// FetchHistoryKline fetches K-line data from EastMoney API.
func (c *Client) FetchHistoryKline(code string, kt int, count int) ([]json.RawMessage, error) {
	secID, err := toSecID(code)
	if err != nil {
		return nil, err
	}

	klt := ktToKlt(kt)
	end := time.Now().Format("20060102")
	beg := estimateStartDate(kt, count)

	url := fmt.Sprintf("%s?secid=%s&klt=%d&fqt=1&beg=%s&end=%s&lmt=%d",
		c.klineBaseURL, secID, klt, beg, end, count)

	resp, err := c.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("eastmoney kline: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eastmoney kline read: %w", err)
	}

	return c.parseKlineResponse(body)
}

func estimateStartDate(kt int, count int) string {
	now := time.Now()
	var days int
	switch {
	case kt <= 240:
		days = (count*kt + 1440 - 1) / 1440 // minutes to days, ceiling
	case kt == 1001:
		days = count * 2 // 2x margin for weekends
	case kt == 1007:
		days = count * 10
	case kt == 1030:
		days = count * 45
	default:
		days = count * 2
	}
	return now.AddDate(0, 0, -days).Format("20060102")
}

func (c *Client) parseKlineResponse(body []byte) ([]json.RawMessage, error) {
	var resp struct {
		RC   int `json:"rc"`
		Data struct {
			Code   string   `json:"code"`
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("kline parse: %w", err)
	}
	if resp.RC != 0 || resp.Data.Klines == nil {
		return []json.RawMessage{}, nil
	}

	bars := make([]json.RawMessage, 0, len(resp.Data.Klines))
	for _, line := range resp.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		// Format: date,open,close,high,low,volume,amount,...
		ts := parseDateToUnix(parts[0])
		o := parseFloatStr(parts[1])
		cl := parseFloatStr(parts[2])
		h := parseFloatStr(parts[3])
		l := parseFloatStr(parts[4])
		v := parseFloatStr(parts[5])

		bar := map[string]interface{}{
			"ts": ts,
			"o":  o,
			"cl": cl,
			"h":  h,
			"l":  l,
			"v":  v,
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
}

func parseDateToUnix(dateStr string) int64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t, err = time.Parse("20060102", dateStr)
		if err != nil {
			return 0
		}
	}
	return t.Unix()
}

func parseFloatStr(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}
