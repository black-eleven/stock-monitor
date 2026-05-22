package eastmoney

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	env          string
	quoteURL     string
	klineBaseURL string

	mu      sync.RWMutex
	tracked []string
}

func NewClient(env string) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		env:        env,
	}

	if env == "hongkong" {
		c.quoteURL = "http://qt.gtimg.cn/q="
		c.klineBaseURL = "" // unused — Tencent kline via fetchTencentKline
	} else {
		c.quoteURL = "https://hq.sinajs.cn/list="
		c.klineBaseURL = "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData"
	}
	return c
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
	req.Header.Set("Referer", sinaReferer)
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

// ---- Quote: env-based routing (Sina for mainland, Tencent for HK) ----

func (c *Client) BatchFetchQuotes(codes []string) (map[string]*Quote, error) {
	if len(codes) == 0 {
		return map[string]*Quote{}, nil
	}

	if c.env == "hongkong" {
		return c.fetchTencentQuotes(codes)
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

	result := parseSinaQuote(string(body), codeMap)

	// Fallback to Eastmoney for codes Sina didn't return (common during non-trading hours).
	var missing []string
	for _, code := range codes {
		if _, ok := result[code]; !ok {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		log.Printf("[QUOTE] Sina missing %d/%d codes, falling back to Eastmoney", len(missing), len(codes))
		emQuotes, emErr := c.fetchEastmoneyQuotes(missing)
		if emErr != nil {
			log.Printf("[QUOTE] Eastmoney fallback error: %v", emErr)
		} else {
			for code, q := range emQuotes {
				result[code] = q
			}
		}
	}

	return result, nil
}

// ---- Quote via Tencent Finance (HK/overseas) ----

func (c *Client) fetchTencentQuotes(codes []string) (map[string]*Quote, error) {
	symbols := make([]string, 0, len(codes))
	codeMap := make(map[string]string)
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
	resp, err := c.doGetWithUA(url)
	if err != nil {
		return nil, fmt.Errorf("tencent quote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tencent quote read: %w", err)
	}

	return parseTencentQuote(string(body), codeMap), nil
}

func parseTencentQuote(body string, codeMap map[string]string) map[string]*Quote {
	quotes := make(map[string]*Quote, len(codeMap))
	ts := time.Now().Unix()

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "v_") {
			continue
		}

		eq := strings.Index(line, "=\"")
		if eq < 0 {
			continue
		}
		symbol := line[2:eq] // after "v_"
		payload := line[eq+2:]

		// Strip trailing \r, then "\";" suffix
		payload = strings.TrimSuffix(payload, "\r")
		payload = strings.TrimSuffix(payload, "\";")

		qosCode, ok := codeMap[symbol]
		if !ok {
			continue
		}

		fields := strings.Split(payload, "~")
		if len(fields) < 10 {
			continue
		}

		var q *Quote
		if strings.HasPrefix(symbol, "hk") {
			q = parseTencentHKFields(fields, qosCode, ts)
		} else {
			q = parseTencentASHFields(fields, qosCode, ts)
		}
		if q != nil {
			quotes[qosCode] = q
		}
	}
	return quotes
}

// Tencent fields (~ separated), same positions for SH/SZ/HK:
// [0]:market, [3]:price, [4]:prevClose, [5]:open, [6]:volume, [33]:high, [34]:low, [37]:turnover
func parseTencentASHFields(fields []string, code string, ts int64) *Quote {
	if len(fields) < 38 {
		return nil
	}
	return &Quote{
		Code:      code,
		Price:     parseFloatStr(fields[3]),
		YP:        parseFloatStr(fields[4]),
		Open:      parseFloatStr(fields[5]),
		High:      parseFloatStr(fields[33]),
		Low:       parseFloatStr(fields[34]),
		Volume:    parseFloatStr(fields[6]) * 100,   // lots → shares
		Turnover:  parseFloatStr(fields[37]) * 10000, // 万元 → 元
		Timestamp: ts,
		Status:    "OK",
	}
}

// Tencent HK: same field positions as A-shares, different units.
func parseTencentHKFields(fields []string, code string, ts int64) *Quote {
	if len(fields) < 38 {
		return nil
	}
	return &Quote{
		Code:      code,
		Price:     parseFloatStr(fields[3]),
		YP:        parseFloatStr(fields[4]),
		Open:      parseFloatStr(fields[5]),
		High:      parseFloatStr(fields[33]),
		Low:       parseFloatStr(fields[34]),
		Volume:    parseFloatStr(fields[6]),   // already in shares
		Turnover:  parseFloatStr(fields[37]),  // already in yuan
		Timestamp: ts,
		Status:    "OK",
	}
}

// ---- K-line via Tencent Finance (HK/overseas) ----

func (c *Client) fetchTencentKline(code string, kt int, count int) ([]json.RawMessage, error) {
	symbol, err := toSinaSymbol(code)
	if err != nil {
		return nil, err
	}

	isHK := strings.HasPrefix(code, "HK:")
	var url string
	var keys []string

	if kt >= 1001 {
		// Daily/weekly/monthly: fqkline/get for all markets.
		// A-shares: qfqday, HK: day (no qfq prefix).
		ktype, fq := ktToTencentParams(kt)
		url = fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,%s,,,%d,%s", symbol, ktype, count, fq)
		if fq != "" {
			keys = []string{fq + ktype, ktype}
		} else {
			keys = []string{ktype}
		}
	} else if isHK {
		// HK intraday: fqkline/get (5m, 15m, etc.)
		ktype, _ := ktToTencentParams(kt)
		url = fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,%s,,,%d,", symbol, ktype, count)
		keys = []string{ktype}
	} else {
		// A-share intraday: mkline endpoint (m1, m5, m15, etc.)
		mktype := ktToMklineType(kt)
		url = fmt.Sprintf("https://ifzq.gtimg.cn/appstock/app/kline/mkline?param=%s,%s,,%d", symbol, mktype, count)
		keys = []string{mktype}
	}

	resp, err := c.doGetWithUA(url)
	if err != nil {
		return nil, fmt.Errorf("tencent kline: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tencent kline read: %w", err)
	}

	return parseTencentKline(body, symbol, keys, count)
}

func ktToTencentParams(kt int) (ktype, fq string) {
	switch kt {
	case 1:
		return "1m", ""
	case 5:
		return "5m", ""
	case 15:
		return "15m", ""
	case 30:
		return "30m", ""
	case 60:
		return "60m", ""
	case 120, 240:
		return "60m", ""
	case 1001:
		return "day", "qfq"
	case 1007:
		return "week", "qfq"
	case 1030:
		return "month", "qfq"
	default:
		return "day", "qfq"
	}
}

func ktToMklineType(kt int) string {
	switch kt {
	case 1:
		return "m1"
	case 5:
		return "m5"
	case 15:
		return "m15"
	case 30:
		return "m30"
	default:
		return "m60"
	}
}

func parseTencentKline(body []byte, symbol string, keys []string, count int) ([]json.RawMessage, error) {
	var resp struct {
		Code int `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("tencent kline parse: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("tencent kline error code: %d", resp.Code)
	}

	raw, ok := resp.Data[symbol]
	if !ok {
		return nil, fmt.Errorf("tencent kline: no data for %s", symbol)
	}

	var stockData map[string]json.RawMessage
	if err := json.Unmarshal(raw, &stockData); err != nil {
		return nil, fmt.Errorf("tencent kline stock parse: %w", err)
	}

	var rows [][]json.RawMessage
	for _, key := range keys {
		if data, ok := stockData[key]; ok {
			if err := json.Unmarshal(data, &rows); err == nil && len(rows) > 0 {
				break
			}
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("tencent kline: no data")
	}

	// Limit to requested count (API may return more)
	start := len(rows) - count
	if start < 0 {
		start = 0
	}
	rows = rows[start:]

	// Each row: [date, open, close, high, low, volume, ...optional extra fields]
	bars := make([]json.RawMessage, 0, len(rows))
	for _, r := range rows {
		if len(r) < 6 {
			continue
		}
		ts := parseDateToUnix(rawToString(r[0]))
		bar := map[string]interface{}{
			"ts": ts,
			"o":  parseFloatStr(rawToString(r[1])),
			"cl": parseFloatStr(rawToString(r[2])),
			"h":  parseFloatStr(rawToString(r[3])),
			"l":  parseFloatStr(rawToString(r[4])),
			"v":  parseFloatStr(rawToString(r[5])),
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
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
		symbol := line[11:eq] // after "var hq_str_"
		payload := line[eq+2:]
		if len(payload) < 2 || payload[len(payload)-2:] != "\";" {
			continue
		}
		payload = payload[:len(payload)-2]

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

// ---- K-line: env-based routing ----

func (c *Client) FetchHistoryKline(code string, kt int, count int) ([]json.RawMessage, error) {
	if c.env == "hongkong" {
		return c.fetchTencentKline(code, kt, count)
	}

	data, err := c.fetchSinaKline(code, kt, count)
	if err == nil && len(data) > 0 {
		return data, nil
	}
	if strings.HasPrefix(code, "HK:") {
		log.Printf("[KLINE] Sina failed for %s, falling back to Eastmoney", code)
		return c.fetchEastmoneyKline(code, kt, count)
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

// ---- Eastmoney kline (HK fallback) ----

func (c *Client) fetchEastmoneyKline(code string, kt int, count int) ([]json.RawMessage, error) {
	secid, err := toEastmoneySecID(code)
	if err != nil {
		return nil, err
	}

	klt := ktToEastmoneyKlt(kt)
	fqt := 0
	if kt >= 1001 {
		fqt = 1 // 前复权 for daily/weekly/monthly
	}
	url := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56&klt=%d&fqt=%d&end=20500101&lmt=%d",
		secid, klt, fqt, count,
	)
	log.Printf("[EASTMONEY] %s", url)

	resp, err := c.doGetWithUA(url)
	if err != nil {
		return nil, fmt.Errorf("eastmoney kline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("eastmoney kline HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eastmoney kline read: %w", err)
	}
	return parseEastmoneyKline(body)
}

func parseEastmoneyKline(body []byte) ([]json.RawMessage, error) {
	var resp struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney kline parse: %w", err)
	}
	if len(resp.Data.Klines) == 0 {
		return nil, fmt.Errorf("eastmoney: no data")
	}

	bars := make([]json.RawMessage, 0, len(resp.Data.Klines))
	for _, line := range resp.Data.Klines {
		// f51=date, f52=open, f53=close, f54=high, f55=low, f56=volume
		fields := strings.Split(line, ",")
		if len(fields) < 6 {
			continue
		}
		ts := parseDateToUnix(fields[0])
		bar := map[string]interface{}{
			"ts": ts,
			"o":  parseFloatStr(fields[1]),
			"cl": parseFloatStr(fields[2]),
			"h":  parseFloatStr(fields[3]),
			"l":  parseFloatStr(fields[4]),
			"v":  parseFloatStr(fields[5]),
		}
		data, _ := json.Marshal(bar)
		bars = append(bars, json.RawMessage(data))
	}
	return bars, nil
}

func rawToString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// ---- Eastmoney quote (mainland fallback) ----

func (c *Client) fetchEastmoneyQuotes(codes []string) (map[string]*Quote, error) {
	if len(codes) == 0 {
		return map[string]*Quote{}, nil
	}

	secids := make([]string, 0, len(codes))
	secidToCode := make(map[string]string, len(codes))
	for _, code := range codes {
		secid, err := toEastmoneySecID(code)
		if err != nil {
			continue
		}
		secids = append(secids, secid)
		secidToCode[secid] = code
	}
	if len(secids) == 0 {
		return map[string]*Quote{}, nil
	}

	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/ulist.np/get?fltt=2&invt=2&fields=f2,f5,f6,f12,f15,f16,f17,f18&secids=%s",
		strings.Join(secids, ","),
	)

	resp, err := c.doGetWithUA(url)
	if err != nil {
		return nil, fmt.Errorf("eastmoney quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("eastmoney quote HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eastmoney quote read: %w", err)
	}

	return parseEastmoneyQuote(body, secidToCode)
}

func parseEastmoneyQuote(body []byte, secidToCode map[string]string) (map[string]*Quote, error) {
	var resp struct {
		Data struct {
			Diff []map[string]interface{} `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney quote parse: %w", err)
	}

	quotes := make(map[string]*Quote, len(resp.Data.Diff))
	ts := time.Now().Unix()

	for _, item := range resp.Data.Diff {
		code := getStringField(item, "f12")
		if code == "" {
			continue
		}

		qosCode := findCodeByRaw(secidToCode, code)
		if qosCode == "" {
			continue
		}

		q := &Quote{
			Code:      qosCode,
			Price:     getFloatField(item, "f2"),
			Open:      getFloatField(item, "f17"),
			YP:        getFloatField(item, "f18"),
			High:      getFloatField(item, "f15"),
			Low:       getFloatField(item, "f16"),
			Volume:    getFloatField(item, "f5") * 100, // 手 → 股
			Turnover:  getFloatField(item, "f6"),
			Timestamp: ts,
			Status:    "OK",
		}
		quotes[qosCode] = q
	}
	return quotes, nil
}

// findCodeByRaw finds the qosCode (e.g. "SH:600519") matching a raw numeric code.
func findCodeByRaw(secidToCode map[string]string, rawCode string) string {
	for _, qosCode := range secidToCode {
		if strings.HasSuffix(qosCode, ":"+rawCode) {
			return qosCode
		}
	}
	return ""
}

func getStringField(item map[string]interface{}, key string) string {
	if v, ok := item[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloatField(item map[string]interface{}, key string) float64 {
	if v, ok := item[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case string:
			return parseFloatStr(n)
		}
	}
	return 0
}

// ---- Helpers ----

func parseDateToUnix(dateStr string) int64 {
	layouts := []string{"2006-01-02 15:04", "2006-01-02", "20060102"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.Unix()
		}
	}
	return 0
}

func parseFloatStr(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}
