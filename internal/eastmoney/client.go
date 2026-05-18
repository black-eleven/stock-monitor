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

const sinaReferer = "https://finance.sina.com.cn/"

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
	quoteURL     string
	klineBaseURL string

	mu      sync.RWMutex
	tracked []string
}

func NewClient() *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		quoteURL:     "https://hq.sinajs.cn/list=",
		klineBaseURL: "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData",
	}
}

func (c *Client) doGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", sinaReferer)
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

// ---- Quote via Sina Finance (all markets) ----

func (c *Client) BatchFetchQuotes(codes []string) (map[string]*Quote, error) {
	if len(codes) == 0 {
		return map[string]*Quote{}, nil
	}

	symbols := make([]string, 0, len(codes))
	codeMap := make(map[string]string, len(codes)) // sinaSymbol → qosCode
	for _, code := range codes {
		sym, err := toSinaSymbol(code)
		if err != nil {
			continue
		}
		symbols = append(symbols, sym)
		codeMap[sym] = code
	}
	if len(symbols) == 0 {
		return map[string]*Quote{}, nil
	}

	url := c.quoteURL + strings.Join(symbols, ",")
	resp, err := c.doGet(url)
	if err != nil {
		return nil, fmt.Errorf("sina quote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sina quote read: %w", err)
	}

	return parseSinaQuote(string(body), codeMap), nil
}

func parseSinaQuote(body string, codeMap map[string]string) map[string]*Quote {
	quotes := make(map[string]*Quote, len(codeMap))
	ts := time.Now().Unix()

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "var hq_str_") {
			continue
		}

		// Extract symbol: var hq_str_sh600519="..."
		eq := strings.Index(line, "=\"")
		if eq < 0 {
			continue
		}
		symbol := line[12:eq] // after "var hq_str_"
		payload := line[eq+2:]
		if len(payload) == 0 || payload[len(payload)-1] != '"' {
			continue
		}
		payload = payload[:len(payload)-1]

		qosCode, ok := codeMap[symbol]
		if !ok {
			continue
		}

		fields := strings.Split(payload, ",")
		if len(fields) < 10 {
			continue
		}

		var q *Quote
		if strings.HasPrefix(symbol, "hk") {
			q = parseSinaHKFields(fields, qosCode, ts)
		} else {
			q = parseSinaASHFields(fields, qosCode, ts)
		}
		if q != nil {
			quotes[qosCode] = q
		}
	}
	return quotes
}

// Sina SH/SZ fields: 0=name, 1=open, 2=yp, 3=price, 4=high, 5=low, 8=volume, 9=turnover
func parseSinaASHFields(fields []string, code string, ts int64) *Quote {
	return &Quote{
		Code:      code,
		Open:      parseFloatStr(fields[1]),
		YP:        parseFloatStr(fields[2]),
		Price:     parseFloatStr(fields[3]),
		High:      parseFloatStr(fields[4]),
		Low:       parseFloatStr(fields[5]),
		Volume:    parseFloatStr(fields[8]),
		Turnover:  parseFloatStr(fields[9]),
		Timestamp: ts,
		Status:    "OK",
	}
}

// Sina HK fields: 0=engName, 1=chiName, 2=open, 3=yp, 4=high, 5=low, 6=price, 11=turnover, 12=volume
func parseSinaHKFields(fields []string, code string, ts int64) *Quote {
	if len(fields) < 13 {
		return nil
	}
	return &Quote{
		Code:      code,
		Open:      parseFloatStr(fields[2]),
		YP:        parseFloatStr(fields[3]),
		High:      parseFloatStr(fields[4]),
		Low:       parseFloatStr(fields[5]),
		Price:     parseFloatStr(fields[6]),
		Volume:    parseFloatStr(fields[12]),
		Turnover:  parseFloatStr(fields[11]),
		Timestamp: ts,
		Status:    "OK",
	}
}

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
	data, err := c.fetchSinaKline(code, kt, count)
	if err == nil && len(data) > 0 {
		return data, nil
	}
	if strings.HasPrefix(code, "HK:") {
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
			"ts": ts, "o": parseFloatStr(r.Open),
			"cl": parseFloatStr(r.Close), "h": parseFloatStr(r.High),
			"l": parseFloatStr(r.Low), "v": parseFloatStr(r.Volume),
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
			"ts": ts, "o": quote.Open[i], "cl": quote.Close[i],
			"h": quote.High[i], "l": quote.Low[i], "v": quote.Volume[i],
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
}

// ---- Helpers ----

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
