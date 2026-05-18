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
		klineBaseURL: "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData",
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

func (c *Client) doGetWithUA(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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

// BatchFetchQuotes fetches quotes using EastMoney (SH/SZ) + Tencent (HK) as fallback.
func (c *Client) BatchFetchQuotes(codes []string) (map[string]*Quote, error) {
	if len(codes) == 0 {
		return map[string]*Quote{}, nil
	}

	emCodes := make([]string, 0, len(codes))
	hkCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		if isHKCode(code) {
			hkCodes = append(hkCodes, code)
		} else {
			emCodes = append(emCodes, code)
		}
	}

	result := make(map[string]*Quote, len(codes))

	// SH/SZ via EastMoney
	if len(emCodes) > 0 {
		emQuotes, _ := c.batchFetchEastMoney(emCodes)
		for k, v := range emQuotes {
			result[k] = v
		}
	}

	// HK via Tencent
	if len(hkCodes) > 0 {
		hkQuotes, _ := c.batchFetchTencent(hkCodes)
		for k, v := range hkQuotes {
			result[k] = v
		}
	}

	return result, nil
}

func isHKCode(code string) bool {
	return strings.HasPrefix(code, "HK:")
}

// ---- EastMoney quote ----

func (c *Client) batchFetchEastMoney(codes []string) (map[string]*Quote, error) {
	secIDs := make([]string, 0, len(codes))
	codeMap := make(map[string]string, len(codes))
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

func (c *Client) parseQuoteResponse(body []byte, codeMap map[string]string) (map[string]*Quote, error) {
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

	for secID, qosCode := range codeMap {
		if strings.HasSuffix(secID, "."+singleResp.Data.F57) {
			return map[string]*Quote{qosCode: {
				Code: qosCode, Price: singleResp.Data.F43 / 100,
				YP: singleResp.Data.F60 / 100, Open: singleResp.Data.F46 / 100,
				High: singleResp.Data.F44 / 100, Low: singleResp.Data.F45 / 100,
				Volume: singleResp.Data.F47, Turnover: singleResp.Data.F48,
				Timestamp: time.Now().Unix(), Status: "OK",
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
			Code: qosCode, Price: d.F43 / 100, YP: d.F60 / 100,
			Open: d.F46 / 100, High: d.F44 / 100, Low: d.F45 / 100,
			Volume: d.F47, Turnover: d.F48, Timestamp: ts, Status: "OK",
		}
	}
	return quotes
}

// ---- Tencent quote (HK fallback) ----

func (c *Client) batchFetchTencent(codes []string) (map[string]*Quote, error) {
	txSymbols := make([]string, 0, len(codes))
	for _, code := range codes {
		sym, err := toTencentSymbol(code)
		if err != nil {
			continue
		}
		txSymbols = append(txSymbols, sym)
	}
	if len(txSymbols) == 0 {
		return map[string]*Quote{}, nil
	}

	url := fmt.Sprintf("http://qt.gtimg.cn/q=%s", strings.Join(txSymbols, ","))
	resp, err := c.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("tencent quote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tencent quote read: %w", err)
	}

	return parseTencentQuote(string(body), codes)
}

func parseTencentQuote(body string, codes []string) (map[string]*Quote, error) {
	// Response: v_hk00700="100~Tencent~00700~451.400~456.400~..."
	result := make(map[string]*Quote, len(codes))
	ts := time.Now().Unix()

	codeIndex := make(map[string]string, len(codes))
	for _, code := range codes {
		parts := strings.SplitN(code, ":", 2)
		if len(parts) == 2 {
			codeIndex[strings.ToLower(parts[1])] = code
		}
	}

	lines := strings.Split(body, ";")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=\"") {
			continue
		}
		// Extract v_hkXXXX="..." or v_szXXXX="..."
		idx := strings.Index(line, "=\"")
		if idx < 0 {
			continue
		}
		payload := line[idx+2:]
		if len(payload) == 0 || payload[len(payload)-1] != '"' {
			continue
		}
		payload = payload[:len(payload)-1]

		fields := strings.Split(payload, "~")
		if len(fields) < 35 {
			continue
		}
		// idx 2: code, idx 3: price, idx 4: yp, idx 5: open, idx 33: high, idx 34: low, idx 6: volume, idx 37: turnover
		txCode := strings.TrimSpace(fields[2])
		qosCode, ok := codeIndex[strings.ToLower(txCode)]
		if !ok {
			continue
		}

		result[qosCode] = &Quote{
			Code:      qosCode,
			Price:     parseFloatStr(fields[3]),
			YP:        parseFloatStr(fields[4]),
			Open:      parseFloatStr(fields[5]),
			High:      parseFloatStr(fields[33]),
			Low:       parseFloatStr(fields[34]),
			Volume:    parseFloatStr(fields[6]),
			Turnover:  parseFloatStr(fields[37]),
			Timestamp: ts,
			Status:    "OK",
		}
	}
	return result, nil
}

// ---- FetchQuote (single) ----

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

// ---- K-line: Sina (SH/SZ) + Yahoo (HK fallback) ----

func (c *Client) FetchHistoryKline(code string, kt int, count int) ([]json.RawMessage, error) {
	// Try Sina first
	data, err := c.fetchSinaKline(code, kt, count)
	if err == nil && len(data) > 0 {
		return data, nil
	}

	// Fall back to Yahoo for HK stocks
	if isHKCode(code) {
		return c.fetchYahooKline(code, kt, count)
	}

	return data, err
}

func (c *Client) fetchSinaKline(code string, kt int, count int) ([]json.RawMessage, error) {
	symbol, err := toSinaSymbol(code)
	if err != nil {
		return nil, err
	}

	scale := ktToSinaScale(kt)
	url := fmt.Sprintf("%s?symbol=%s&scale=%d&datalen=%d", c.klineBaseURL, symbol, scale, count)

	resp, err := c.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("sina kline: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sina kline read: %w", err)
	}

	return parseSinaKline(body)
}

func parseSinaKline(body []byte) ([]json.RawMessage, error) {
	if string(body) == "null" || len(body) == 0 {
		return nil, fmt.Errorf("sina: no data")
	}
	var records []struct {
		Day    string `json:"day"`
		Open   string `json:"open"`
		High   string `json:"high"`
		Low    string `json:"low"`
		Close  string `json:"close"`
		Volume string `json:"volume"`
	}
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("sina kline parse: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("sina: empty records")
	}

	bars := make([]json.RawMessage, 0, len(records))
	for _, r := range records {
		ts := parseDateToUnix(r.Day)
		bar := map[string]interface{}{
			"ts": ts,
			"o":  parseFloatStr(r.Open),
			"cl": parseFloatStr(r.Close),
			"h":  parseFloatStr(r.High),
			"l":  parseFloatStr(r.Low),
			"v":  parseFloatStr(r.Volume),
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
}

// ---- Yahoo Finance kline (HK fallback) ----

func (c *Client) fetchYahooKline(code string, kt int, count int) ([]json.RawMessage, error) {
	symbol, err := toYahooSymbol(code)
	if err != nil {
		return nil, err
	}

	interval, yrange := yahooParams(kt, count)
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=%s",
		symbol, yrange, interval)

	resp, err := c.doGetWithUA(url)
	if err != nil {
		return nil, fmt.Errorf("yahoo kline: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo kline read: %w", err)
	}

	return parseYahooKline(body)
}

func yahooParams(kt int, count int) (interval, yrange string) {
	switch {
	case kt <= 60:
		return "1m", "5d"
	case kt <= 240:
		return "5m", "5d"
	case kt == 1001:
		days := count * 2
		if days < 5 {
			days = 5
		}
		if days <= 30 {
			return "1d", "1mo"
		}
		if days <= 90 {
			return "1d", "3mo"
		}
		if days <= 180 {
			return "1d", "6mo"
		}
		if days <= 365 {
			return "1d", "1y"
		}
		return "1d", "2y"
	case kt == 1007:
		return "1wk", "6mo"
	case kt == 1030:
		return "1mo", "2y"
	default:
		return "1d", "1mo"
	}
}

func parseYahooKline(body []byte) ([]json.RawMessage, error) {
	var resp struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []float64 `json:"open"`
						High   []float64 `json:"high"`
						Low    []float64 `json:"low"`
						Close  []float64 `json:"close"`
						Volume []float64 `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("yahoo kline parse: %w", err)
	}
	if len(resp.Chart.Result) == 0 || len(resp.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo: no data")
	}

	q := resp.Chart.Result[0]
	timestamps := q.Timestamp
	quote := q.Indicators.Quote[0]
	if len(timestamps) == 0 || len(quote.Open) == 0 {
		return nil, fmt.Errorf("yahoo: empty series")
	}

	bars := make([]json.RawMessage, 0, len(timestamps))
	for i, ts := range timestamps {
		if i >= len(quote.Open) {
			break
		}
		bar := map[string]interface{}{
			"ts": ts,
			"o":  quote.Open[i],
			"cl": quote.Close[i],
			"h":  quote.High[i],
			"l":  quote.Low[i],
			"v":  quote.Volume[i],
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
}

// ---- Date/float helpers ----

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
