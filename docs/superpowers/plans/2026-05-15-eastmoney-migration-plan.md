# EastMoney HTTP API Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace QOS WebSocket data source with EastMoney HTTP API for stock quotes and K-line data.

**Architecture:** Replace `internal/qos/` (WebSocket client) with `internal/eastmoney/` (HTTP client). Handlers and recommender use a `QuoteClient` interface instead of `*qos.QosClient`. Real-time push replaced by polling goroutine in main.go that batch-fetches quotes every 5 seconds and broadcasts changes via the existing WebSocket hub.

**Tech Stack:** Go `net/http`, `encoding/json`, existing Gin + gorilla/websocket (frontend WS unchanged)

---

### Task 1: Create `internal/eastmoney/symbol.go` — Symbol format mapping

**Files:**
- Create: `internal/eastmoney/symbol.go`

- [ ] **Step 1: Write the file**

```go
package eastmoney

import (
	"fmt"
	"strings"
)

// toSecID converts QOS-format symbol (SH:600519, SZ:000001, HK:00700) to EastMoney secid.
func toSecID(qosSymbol string) (string, error) {
	parts := strings.SplitN(qosSymbol, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid symbol format: %s", qosSymbol)
	}
	market := strings.ToUpper(parts[0])
	code := strings.ToUpper(parts[1])

	switch market {
	case "SH":
		return "1." + code, nil
	case "SZ":
		return "0." + code, nil
	case "HK":
		return "116." + code, nil
	default:
		return "", fmt.Errorf("unsupported market: %s", market)
	}
}

// fromSecID converts EastMoney secid back to QOS-format symbol.
func fromSecID(secID string) string {
	parts := strings.SplitN(secID, ".", 2)
	if len(parts) != 2 {
		return secID
	}
	switch parts[0] {
	case "1":
		return "SH:" + parts[1]
	case "0":
		return "SZ:" + parts[1]
	case "116":
		return "HK:" + parts[1]
	default:
		return secID
	}
}

// ktToKlt converts QOS K-line type to EastMoney klt parameter.
func ktToKlt(kt int) int {
	switch kt {
	case 1:
		return 1   // 1m
	case 5:
		return 5   // 5m
	case 15:
		return 15  // 15m
	case 30:
		return 30  // 30m
	case 60:
		return 60  // 1h
	case 120:
		return 120 // 2h
	case 240:
		return 240 // 4h
	case 1001:
		return 101 // daily
	case 1007:
		return 102 // weekly
	case 1030:
		return 103 // monthly
	default:
		return 101 // default to daily
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/eastmoney/symbol.go
git commit -m "feat: add eastmoney symbol mapping and K-line type conversion"
```

---

### Task 2: Create `internal/eastmoney/client.go` — HTTP client, types, and QuoteClient interface

**Files:**
- Create: `internal/eastmoney/client.go`

- [ ] **Step 1: Write the file**

```go
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
// Returns a map of QOS-format code to *Quote.
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

	resp, err := c.httpClient.Get(url)
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

// parseQuoteResponse parses EastMoney quote API response.
// Handles both batch format (data.diff[]) and single-item format (data as flat object).
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

	resp, err := c.httpClient.Get(url)
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/eastmoney/client.go
git commit -m "feat: add eastmoney HTTP client with batch quote and kline fetching"
```

---

### Task 3: Create `internal/eastmoney/cache.go` — Caching layer

**Files:**
- Create: `internal/eastmoney/cache.go`

- [ ] **Step 1: Write the file**

```go
package eastmoney

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type klineCacheEntry struct {
	data      []json.RawMessage
	expiresAt time.Time
}

type quoteCacheEntry struct {
	data      *Quote
	expiresAt time.Time
}

type mergeEntry struct {
	result mergeResult
	done   chan struct{}
	once   sync.Once
}

type mergeResult struct {
	data []json.RawMessage
	q    *Quote
	err  error
}

func (m *mergeEntry) resolve(r mergeResult) {
	m.once.Do(func() {
		m.result = r
		close(m.done)
	})
}

var (
	klineCache   = map[string]*klineCacheEntry{}
	klineCacheMu sync.RWMutex
	klineMerge   = map[string]*mergeEntry{}
	klineMergeMu sync.Mutex
)

var (
	quoteCache   = map[string]*quoteCacheEntry{}
	quoteCacheMu sync.RWMutex
	quoteMerge   = map[string]*mergeEntry{}
	quoteMergeMu sync.Mutex
)

func klineCacheKey(code string, kt, count int) string {
	return fmt.Sprintf("%s:%d:%d", code, kt, count)
}

func cacheTTL(kt int) time.Duration {
	if kt <= 240 {
		return 30 * time.Second
	}
	return 5 * time.Minute
}

func (c *Client) FetchHistoryKlineCached(code string, kt int, count int) ([]json.RawMessage, error) {
	key := klineCacheKey(code, kt, count)

	klineCacheMu.RLock()
	if e, ok := klineCache[key]; ok && time.Now().Before(e.expiresAt) {
		data := e.data
		klineCacheMu.RUnlock()
		return data, nil
	}
	klineCacheMu.RUnlock()

	klineMergeMu.Lock()
	m, exists := klineMerge[key]
	if !exists {
		m = &mergeEntry{done: make(chan struct{})}
		klineMerge[key] = m
	}
	klineMergeMu.Unlock()

	if !exists {
		data, err := c.FetchHistoryKline(code, kt, count)
		if err == nil {
			klineCacheMu.Lock()
			klineCache[key] = &klineCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL(kt))}
			klineCacheMu.Unlock()
		}
		m.resolve(mergeResult{data: data, err: err})
		klineMergeMu.Lock()
		delete(klineMerge, key)
		klineMergeMu.Unlock()
		return data, err
	}

	<-m.done
	return m.result.data, m.result.err
}

func (c *Client) FetchQuoteCached(code string) (*Quote, error) {
	key := code

	quoteCacheMu.RLock()
	if e, ok := quoteCache[key]; ok && time.Now().Before(e.expiresAt) {
		q := e.data
		quoteCacheMu.RUnlock()
		return q, nil
	}
	quoteCacheMu.RUnlock()

	quoteMergeMu.Lock()
	m, exists := quoteMerge[key]
	if !exists {
		m = &mergeEntry{done: make(chan struct{})}
		quoteMerge[key] = m
	}
	quoteMergeMu.Unlock()

	if !exists {
		q, err := c.FetchQuote(code)
		if err == nil && q != nil {
			quoteCacheMu.Lock()
			quoteCache[key] = &quoteCacheEntry{data: q, expiresAt: time.Now().Add(30 * time.Second)}
			quoteCacheMu.Unlock()
		}
		m.resolve(mergeResult{q: q, err: err})
		quoteMergeMu.Lock()
		delete(quoteMerge, key)
		quoteMergeMu.Unlock()
		return q, err
	}

	<-m.done
	return m.result.q, m.result.err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/eastmoney/cache.go
git commit -m "feat: add eastmoney cache layer with TTL and request coalescing"
```

---

### Task 4: Modify `internal/model/quote.go` — Replace FromQosQuote with FromEMQuote

**Files:**
- Modify: `internal/model/quote.go`

- [ ] **Step 1: Replace qos import and conversion function**

Old content:
```go
package model

import "github.com/black-eleven/stock-monitor/internal/qos"

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

func FromQosQuote(q qos.Quote) Quote {
	return Quote{
		Code: q.Code, Price: q.Price, YP: q.YP,
		Open: q.Open, High: q.High, Low: q.Low,
		Volume: q.Volume, Turnover: q.Turnover,
		Timestamp: q.Timestamp, Status: q.Status,
	}
}

type KlineBar struct { ... }
type KlineItem struct { ... }
```

New content:
```go
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/model/quote.go
git commit -m "refactor: replace FromQosQuote with FromEMQuote in model layer"
```

---

### Task 5: Modify `internal/config/config.go` — Remove QOS config fields

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Remove QOS fields from Config struct and Load()**

Remove `QosKey` and `QosWsUrl` from the Config struct:
```go
type Config struct {
	Port           string
	DataDir        string
	JwtSecret             string
	AdminPassword          string
	ExplicitAdminPassword  bool
	NewsAPIKey          string
	NewsAPIDays         int
	NewsAPIPageSize     int
	NewsAPILanguages    []string
	RecommendCandidates int
	RecommendLimit      int
}
```

Remove QOS-specific lines from `Load()`:
```go
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	absDataDir, _ := filepath.Abs(dataDir)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = generateRandomSecret(32)
		log.Printf("[CONFIG] JWT_SECRET not set, generated random secret (first 8 chars): %s...", jwtSecret[:8])
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	explicitAdminPassword := adminPassword != ""
	if adminPassword == "" {
		adminPassword = generateRandomSecret(16)
		log.Printf("[CONFIG] ADMIN_PASSWORD not set, generated: %s", adminPassword)
	}

	// ... NewsAPI config unchanged ...

	return &Config{
		Port:          port,
		DataDir:       absDataDir,
		JwtSecret:     jwtSecret,
		AdminPassword:          adminPassword,
		ExplicitAdminPassword:  explicitAdminPassword,
		NewsAPIKey:          newsAPIKey,
		NewsAPIDays:         newsAPIDays,
		NewsAPIPageSize:     newsAPIPageSize,
		NewsAPILanguages:    newsAPILanguages,
		RecommendCandidates: recommendCandidates,
		RecommendLimit:      recommendLimit,
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor: remove QOS config fields, EastMoney requires no API key"
```

---

### Task 6: Modify `internal/handler/quote.go` — Use eastmoney.QuoteClient interface

**Files:**
- Modify: `internal/handler/quote.go`

- [ ] **Step 1: Replace all `*qos.QosClient` references with `eastmoney.QuoteClient`**

```go
package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

var symbolRegex = regexp.MustCompile(`^(HK|SH|SZ):[A-Z0-9]{1,10}$`)

type QuoteHandler struct {
	client eastmoney.QuoteClient
}

func NewQuoteHandler(client eastmoney.QuoteClient) *QuoteHandler {
	return &QuoteHandler{client: client}
}

func (h *QuoteHandler) Register(api *gin.RouterGroup) {
	api.GET("/quote/batch", h.batch)
	api.GET("/quote/:symbol", h.single)
}

func (h *QuoteHandler) batch(c *gin.Context) {
	symbolsStr := c.Query("symbols")
	symbols := strings.Split(symbolsStr, ",")
	trimmed := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No symbols provided"})
		return
	}

	type result struct {
		symbol string
		quote  *eastmoney.Quote
	}
	results := make(chan result, len(trimmed))

	for _, s := range trimmed {
		go func(symbol string) {
			q, err := h.client.FetchQuoteCached(symbol)
			if err != nil {
				results <- result{symbol: symbol}
			} else {
				results <- result{symbol: symbol, quote: q}
			}
		}(s)
	}

	data := make(map[string]interface{})
	for i := 0; i < len(trimmed); i++ {
		r := <-results
		if r.quote != nil {
			data[r.symbol] = r.quote
		}
	}
	c.JSON(http.StatusOK, data)
}

func (h *QuoteHandler) single(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if !symbolRegex.MatchString(symbol) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol format. Use HK:700 / SH:600519 / SZ:000001"})
		return
	}
	quote, err := h.client.FetchQuoteCached(symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch quote"})
		return
	}
	c.JSON(http.StatusOK, quote)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handler/quote.go
git commit -m "refactor: quote handler uses eastmoney.QuoteClient interface"
```

---

### Task 7: Modify `internal/handler/kline.go` — Use eastmoney.QuoteClient interface

**Files:**
- Modify: `internal/handler/kline.go`

- [ ] **Step 1: Replace all `*qos.QosClient` references with `eastmoney.QuoteClient`**

```go
package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

var ktMap = map[string]int{
	"1m": 1, "5m": 5, "15m": 15, "30m": 30,
	"1h": 60, "2h": 120, "4h": 240,
	"1d": 1001, "1w": 1007, "1M": 1030,
}

type KlineHandler struct {
	client eastmoney.QuoteClient
}

func NewKlineHandler(client eastmoney.QuoteClient) *KlineHandler {
	return &KlineHandler{client: client}
}

func (h *KlineHandler) Register(api *gin.RouterGroup) {
	api.GET("/kline/:symbol", h.getKline)
}

func (h *KlineHandler) getKline(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	interval := c.DefaultQuery("interval", "1d")
	countStr := c.DefaultQuery("count", "100")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 100
	}

	kt, ok := ktMap[interval]
	if !ok {
		keys := make([]string, 0, len(ktMap))
		for k := range ktMap {
			keys = append(keys, k)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval: " + interval + ". Supported: " + strings.Join(keys, ", "),
		})
		return
	}

	data, err := h.client.FetchHistoryKlineCached(symbol, kt, count)
	if err != nil {
		log.Printf("[Kline] Failed to fetch kline for %s (kt=%d): %v", symbol, kt, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch kline data"})
		return
	}
	c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handler/kline.go
git commit -m "refactor: kline handler uses eastmoney.QuoteClient interface"
```

---

### Task 8: Modify `internal/recommend/recommender.go` — Use eastmoney.QuoteClient interface

**Files:**
- Modify: `internal/recommend/recommender.go`

- [ ] **Step 1: Replace all `*qos.QosClient` references with `eastmoney.QuoteClient`**

```go
package recommend

import (
	"fmt"
	"sync"
	"time"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/black-eleven/stock-monitor/internal/model"
)

type cacheEntry struct {
	recs      []model.Recommendation
	expiresAt time.Time
}

type Recommender struct {
	newsapi    *NewsAPIClient
	emClient   eastmoney.QuoteClient
	cache      map[string]*cacheEntry
	cacheTTL   time.Duration
	days       int
	pageSize   int
	languages  []string
	candidates int
	limit      int
	mu         sync.RWMutex
}

func NewRecommender(newsapi *NewsAPIClient, emClient eastmoney.QuoteClient, days, pageSize int, languages []string, candidates, limit int) *Recommender {
	return &Recommender{
		newsapi:    newsapi,
		emClient:   emClient,
		cache:      make(map[string]*cacheEntry),
		cacheTTL:   30 * time.Minute,
		days:       days,
		pageSize:   pageSize,
		languages:  languages,
		candidates: candidates,
		limit:      limit,
	}
}

func (r *Recommender) Search(industry string) ([]model.Recommendation, error) {
	r.mu.RLock()
	if e, ok := r.cache[industry]; ok && time.Now().Before(e.expiresAt) {
		recs := e.recs
		r.mu.RUnlock()
		return recs, nil
	}
	r.mu.RUnlock()

	articles, err := r.searchAllLanguages(industry)
	if err != nil {
		return nil, fmt.Errorf("news search: %w", err)
	}

	if len(articles) == 0 {
		return []model.Recommendation{}, nil
	}

	results := Extract(articles, r.candidates)
	if len(results) == 0 {
		return []model.Recommendation{}, nil
	}

	quotes := r.batchFetchQuotes(results)

	validResults := make([]ExtractionResult, 0, len(results))
	for _, res := range results {
		if _, ok := quotes[res.Symbol]; ok {
			validResults = append(validResults, res)
		}
	}
	if len(validResults) == 0 {
		validResults = results
	}

	recs := Score(validResults, quotes, r.limit)

	r.mu.Lock()
	r.cache[industry] = &cacheEntry{recs: recs, expiresAt: time.Now().Add(r.cacheTTL)}
	r.mu.Unlock()

	return recs, nil
}

func (r *Recommender) searchAllLanguages(industry string) ([]Article, error) {
	type result struct {
		articles []Article
		err      error
	}

	ch := make(chan result, len(r.languages))
	for _, lang := range r.languages {
		go func(language string) {
			arts, err := r.newsapi.Search(industry, r.days, r.pageSize, language)
			ch <- result{articles: arts, err: err}
		}(lang)
	}

	var all []Article
	var lastErr error
	for i := 0; i < len(r.languages); i++ {
		r := <-ch
		if r.err != nil {
			lastErr = r.err
			continue
		}
		all = append(all, r.articles...)
	}

	if len(all) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return all, nil
}

func (r *Recommender) batchFetchQuotes(candidates []ExtractionResult) map[string]*eastmoney.Quote {
	type result struct {
		symbol string
		quote  *eastmoney.Quote
	}

	ch := make(chan result, len(candidates))
	for _, c := range candidates {
		go func(symbol string) {
			q, err := r.emClient.FetchQuoteCached(symbol)
			if err != nil {
				ch <- result{symbol: symbol}
			} else {
				ch <- result{symbol: symbol, quote: q}
			}
		}(c.Symbol)
	}

	quotes := make(map[string]*eastmoney.Quote, len(candidates))
	for i := 0; i < len(candidates); i++ {
		res := <-ch
		if res.quote != nil {
			quotes[res.symbol] = res.quote
		}
	}
	return quotes
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/recommend/recommender.go
git commit -m "refactor: recommender uses eastmoney.QuoteClient interface"
```

---

### Task 9: Modify `cmd/server/main.go` — Wire eastmoney client + polling, and add GetAllSymbols to repo

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/repo/watchlist.go`

- [ ] **Step 1: Add `GetAllSymbols` method to `internal/repo/watchlist.go`**

Add this method to the existing file:
```go
// GetAllSymbols returns distinct symbols across all user watchlists.
func (r *WatchlistRepo) GetAllSymbols() ([]string, error) {
	rows, err := r.db.Query("SELECT DISTINCT symbol FROM watchlist")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}
```

- [ ] **Step 2: Rewrite main.go to use eastmoney client**

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/black-eleven/stock-monitor/internal/alert"
	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/recommend"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Repositories
	watchlistRepo := repo.NewWatchlistRepo(database)
	alertRepo := repo.NewAlertRepo(database)
	holdingRepo := repo.NewHoldingRepo(database)
	userRepo := repo.NewUserRepo(database)
	inviteCodeRepo := repo.NewInviteCodeRepo(database)

	// WebSocket Hub
	hub := ws.NewHub(cfg.JwtSecret)
	go hub.Run()

	// Init admin user if first run
	adminID, err := db.InitAdmin(database, cfg.AdminPassword, cfg.ExplicitAdminPassword)
	if err != nil {
		log.Fatalf("Failed to init admin: %v", err)
	}
	if adminID > 0 {
		log.Printf("[MAIN] Initial admin created (id=%d), password printed in config logs above", adminID)
	}

	// EastMoney Client
	emClient := eastmoney.NewClient()

	// Alert Engine
	alertEngine := alert.NewEngine(alertRepo, hub)

	// HTTP handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo, nil)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)
	quoteH := handler.NewQuoteHandler(emClient)
	klineH := handler.NewKlineHandler(emClient)

	// Recommender
	newsapiClient := recommend.NewNewsAPIClient(cfg.NewsAPIKey)
	recommender := recommend.NewRecommender(newsapiClient, emClient, cfg.NewsAPIDays, cfg.NewsAPIPageSize, cfg.NewsAPILanguages, cfg.RecommendCandidates, cfg.RecommendLimit)
	recommendH := handler.NewRecommendHandler(recommender)

	signalRepo := repo.NewSignalRepo(database)
	signalH := handler.NewSignalHandler(signalRepo, hub)

	authH := handler.NewAuthHandler(userRepo, inviteCodeRepo, cfg.JwtSecret)
	adminH := handler.NewAdminHandler(inviteCodeRepo)

	r := gin.Default()
	api := r.Group("/api")

	// Public routes — no auth required
	authH.Register(api)

	// Protected routes — JWT required
	authMW := middleware.AuthMiddleware(cfg.JwtSecret)
	auth := api.Group("", authMW)
	watchlistH.Register(auth)
	alertH.Register(auth)
	holdingH.Register(auth)
	quoteH.Register(auth)
	klineH.Register(auth)
	recommendH.Register(auth)
	signalH.Register(auth)

	// Admin routes
	admin := auth.Group("/admin", middleware.AdminRequired())
	adminH.Register(admin)

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) { hub.ServeWS(c.Writer, c.Request) })

	// Static files
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/index.html", "./web/index.html")
	r.StaticFile("/login.html", "./web/login.html")
	r.StaticFile("/admin.html", "./web/admin.html")

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Polling goroutine — fetch quotes every 5s for tracked stocks
	go pollQuotes(emClient, watchlistRepo, hub, alertEngine)

	// Periodic watchlist sync — refresh tracked codes every 30s
	go syncTrackedCodes(emClient, watchlistRepo)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func pollQuotes(emClient *eastmoney.Client, watchlistRepo *repo.WatchlistRepo, hub *ws.Hub, alertEngine *alert.Engine) {
	time.Sleep(2 * time.Second) // Wait for server to start
	for {
		codes := emClient.GetTrackedCodes()
		if len(codes) > 0 {
			quotes, err := emClient.BatchFetchQuotes(codes)
			if err != nil {
				log.Printf("[POLL] Batch fetch error: %v", err)
			} else {
				for _, q := range quotes {
					mq := model.FromEMQuote(*q)
					hub.BroadcastQuote(mq)
					alertEngine.Evaluate(mq)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func syncTrackedCodes(emClient *eastmoney.Client, watchlistRepo *repo.WatchlistRepo) {
	for {
		time.Sleep(30 * time.Second)
		symbols, err := watchlistRepo.GetAllSymbols()
		if err != nil {
			log.Printf("[SYNC] Failed to load watchlist symbols: %v", err)
			continue
		}
		if len(symbols) > 0 {
			emClient.SetTrackedCodes(symbols)
		}
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go internal/repo/watchlist.go
git commit -m "feat: wire eastmoney client with polling and watchlist tracking"
```

---

### Task 10: Remove QOS package and clean up

**Files:**
- Delete: `internal/qos/` (entire directory)

- [ ] **Step 1: Delete the qos package**

```bash
rm -rf internal/qos/
```

- [ ] **Step 2: Tidy go.mod**

```bash
go mod tidy
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove QOS package, replaced by eastmoney"
```

---

### Task 11: Build verification

**Files:** None (verification only)

- [ ] **Step 1: Build the project**

```bash
go build ./...
```

Expected: Build succeeds with no errors.

- [ ] **Step 2: Run vet**

```bash
go vet ./...
```

Expected: No warnings or errors.

- [ ] **Step 3: Run existing tests**

```bash
go test ./internal/... -v
```

Expected: All existing tests pass (test repos, config, recommend — not qos).

- [ ] **Step 4: Quick API smoke test**

Start the server briefly to verify it starts without panic:
```bash
timeout 5 go run ./cmd/server 2>&1 || true
```

Expected: Server starts, logs config, no panic or fatal errors.

- [ ] **Step 5: Commit any fixes**

```bash
git add -A && git commit -m "fix: build verification fixes"
```
