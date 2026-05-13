# 智能推荐自选股 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a stock recommendation engine that searches NewsAPI for industry-related news, extracts stock symbols, scores candidates by news sentiment + market trends, and exposes results via REST API. Frontend adds a "推荐发现" tab on the watchlist screen.

**Architecture:** Backend pipeline: NewsAPI search → symbol extraction from news text → QOS quote fetch → composite scoring → JSON response. Frontend: TabBarView with existing watchlist + new recommend tab. In-memory cache for 30-minute deduplication.

**Tech Stack:** Go (Gin, SQLite), Flutter (Riverpod, Dio), NewsAPI.org

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/model/recommend.go` | Request/response structs |
| Create | `internal/recommend/newsapi.go` | NewsAPI HTTP client |
| Create | `internal/recommend/extractor.go` | Symbol extraction from news text |
| Create | `internal/recommend/extractor_test.go` | Extractor unit tests |
| Create | `internal/recommend/scorer.go` | Composite scoring logic |
| Create | `internal/recommend/scorer_test.go` | Scorer unit tests |
| Create | `internal/recommend/recommender.go` | Orchestrator + cache |
| Create | `internal/handler/recommend.go` | HTTP handler |
| Modify | `internal/config/config.go` | Add NewsAPI env vars |
| Modify | `cmd/server/main.go` | Wire up recommender + handler |
| Create | `mobile/stock_monitor/lib/domain/model/recommendation.dart` | Recommendation Dart model |
| Create | `mobile/stock_monitor/lib/data/api/recommend_api.dart` | Recommend API client |
| Modify | `mobile/stock_monitor/lib/presentation/providers/api_providers.dart` | Add provider |
| Modify | `mobile/stock_monitor/lib/presentation/screens/watchlist_screen.dart` | TabBarView refactor |

---

### Task 1: Config — Add NewsAPI env vars

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add NewsAPI fields to Config struct and Load()**

Read `internal/config/config.go` and add three new fields:

```go
// Add to Config struct after AdminPassword:
NewsAPIKey          string
NewsAPIDays         int
NewsAPIPageSize     int
```

```go
// Add to Load() function after adminPassword block:
newsAPIKey := os.Getenv("NEWSAPI_KEY")
newsAPIDays := 7
if s := os.Getenv("NEWSAPI_DAYS"); s != "" {
    if d, err := strconv.Atoi(s); err == nil && d > 0 {
        newsAPIDays = d
    }
}
newsAPIPageSize := 50
if s := os.Getenv("NEWSAPI_PAGE_SIZE"); s != "" {
    if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
        newsAPIPageSize = n
    }
}
```

```go
// Add to Config return literal:
NewsAPIKey:      newsAPIKey,
NewsAPIDays:     newsAPIDays,
NewsAPIPageSize: newsAPIPageSize,
```

Also add `"strconv"` to imports.

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add NewsAPI config env vars (NEWSAPI_KEY, NEWSAPI_DAYS, NEWSAPI_PAGE_SIZE)"
```

---

### Task 2: Model — Define recommendation types

**Files:**
- Create: `internal/model/recommend.go`

- [ ] **Step 1: Write model file**

```go
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
```

- [ ] **Step 2: Build check**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/model/recommend.go
git commit -m "feat: add recommendation model types"
```

---

### Task 3: NewsAPI Client

**Files:**
- Create: `internal/recommend/newsapi.go`

- [ ] **Step 1: Write NewsAPI client**

```go
package recommend

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

type Article struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    PublishedAt string `json:"publishedAt"`
    URL         string `json:"url"`
}

type newsAPIResponse struct {
    Status       string    `json:"status"`
    TotalResults int       `json:"totalResults"`
    Articles     []Article `json:"articles"`
}

type NewsAPIClient struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func NewNewsAPIClient(apiKey string) *NewsAPIClient {
    return &NewsAPIClient{
        apiKey:     apiKey,
        baseURL:    "https://newsapi.org/v2/everything",
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *NewsAPIClient) Search(query string, days int, pageSize int) ([]Article, error) {
    fromDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

    u, _ := url.Parse(c.baseURL)
    u.RawQuery = url.Values{
        "q":        {query},
        "apiKey":   {c.apiKey},
        "sortBy":   {"popularity"},
        "pageSize": {fmt.Sprintf("%d", pageSize)},
        "from":     {fromDate},
        "language": {"en"},
    }.Encode()

    resp, err := c.httpClient.Get(u.String())
    if err != nil {
        return nil, fmt.Errorf("newsapi request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var apiErr struct {
            Code    string `json:"code"`
            Message string `json:"message"`
        }
        json.NewDecoder(resp.Body).Decode(&apiErr)
        return nil, fmt.Errorf("newsapi error %s: %s", apiErr.Code, apiErr.Message)
    }

    var result newsAPIResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("newsapi decode: %w", err)
    }

    return result.Articles, nil
}
```

- [ ] **Step 2: Build check**

Run: `go build ./internal/recommend/...`

- [ ] **Step 3: Commit**

```bash
git add internal/recommend/newsapi.go
git commit -m "feat: add NewsAPI client for stock news search"
```

---

### Task 4: Stock Symbol Extractor

**Files:**
- Create: `internal/recommend/extractor.go`
- Create: `internal/recommend/extractor_test.go`

- [ ] **Step 1: Write the failing test**

```go
package recommend

import (
    "testing"
)

func TestExtractStockSymbols(t *testing.T) {
    articles := []Article{
        {
            Title:       "NVIDIA (NVDA) reports record earnings on AI chip demand",
            Description: "$NVDA surged 5% after the announcement. Meanwhile, AMD also rose 2%.",
            PublishedAt: "2026-05-13T10:00:00Z",
        },
        {
            Title:       "Tencent 0700.HK falls amid regulatory concerns",
            Description: "Shares of HK:0700 dropped 3% in Hong Kong trading.",
            PublishedAt: "2026-05-13T09:00:00Z",
        },
        {
            Title:       "Kweichow Moutai 600519.SH hits new high",
            Description: "The premium liquor maker continues its rally.",
            PublishedAt: "2026-05-12T08:00:00Z",
        },
    }

    results := Extract(articles)

    symbols := make(map[string]bool)
    for _, r := range results {
        symbols[r.Symbol] = true
    }

    if !symbols["US:NVDA"] {
        t.Error("expected US:NVDA from $NVDA or (NVDA)")
    }
    if !symbols["HK:0700"] {
        t.Error("expected HK:0700 from 0700.HK or HK:0700")
    }
    if !symbols["SH:600519"] {
        t.Error("expected SH:600519 from 600519.SH")
    }

    // NVDA should have higher count (appears in both title and description x2)
    for _, r := range results {
        if r.Symbol == "US:NVDA" && r.Count < 2 {
            t.Errorf("expected NVDA count >= 2, got %d", r.Count)
        }
        if r.Symbol == "US:NVDA" && r.HeadlineHits == 0 {
            t.Error("expected NVDA to have headline hits")
        }
        if r.Symbol == "US:NVDA" && len(r.Highlights) == 0 {
            t.Error("expected NVDA to have highlights")
        }
    }

    // Should be deduplicated: no duplicate symbols
    seen := make(map[string]int)
    for _, r := range results {
        if prev, ok := seen[r.Symbol]; ok {
            t.Errorf("duplicate symbol %s at index %d (previously at %d)", r.Symbol, seen[r.Symbol], prev)
        }
        seen[r.Symbol] = 1
    }
}

func TestExtractEmpty(t *testing.T) {
    results := Extract(nil)
    if len(results) != 0 {
        t.Errorf("expected empty, got %d results", len(results))
    }
}

func TestExtractTop15(t *testing.T) {
    var articles []Article
    for i := 0; i < 100; i++ {
        articles = append(articles, Article{
            Title:       "AAPL stock hits new high",
            Description: "$AAPL continues strong performance.",
            PublishedAt: "2026-05-13T10:00:00Z",
        })
        articles = append(articles, Article{
            Title:       "GOOGL announces new AI features",
            Description: "$GOOGL shares rise.",
            PublishedAt: "2026-05-13T10:00:00Z",
        })
    }
    results := Extract(articles)
    if len(results) > 15 {
        t.Errorf("expected max 15, got %d", len(results))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/recommend/ -v -run TestExtract`
Expected: FAIL (function Extract not defined)

- [ ] **Step 3: Write extractor implementation**

```go
package recommend

import (
    "regexp"
    "sort"
    "strings"
)

var symbolPatterns = []struct {
    regex    *regexp.Regexp
    template string // e.g. "US:$1", "HK:$1", etc.
}{
    // Explicit prefixed symbols
    {regexp.MustCompile(`US:([A-Z]{1,5})`), "US:$1"},
    {regexp.MustCompile(`HK:(\d{4,5})`), "HK:$1"},
    {regexp.MustCompile(`SH:(\d{6})`), "SH:$1"},
    {regexp.MustCompile(`SZ:(\d{6})`), "SZ:$1"},
    // $TICKER format → US
    {regexp.MustCompile(`\$([A-Z]{1,5})`), "US:$1"},
    // TICKER.HK format → HK, strip leading zeros
    {regexp.MustCompile(`(\d{4,5})\.HK`), "HK:$1"},
    // TICKER.SH format
    {regexp.MustCompile(`(\d{6})\.SH`), "SH:$1"},
    // TICKER.SZ format
    {regexp.MustCompile(`(\d{6})\.SZ`), "SZ:$1"},
    // (TICKER) parenthetical reference → US
    {regexp.MustCompile(`\(([A-Z]{1,5})\)`), "US:$1"},
}

type rawHit struct {
    symbol      string
    headline    bool
    publishedAt string
    snippet     string
}

type ExtractionResult struct {
    Symbol       string
    Name         string
    Count        int
    HeadlineHits int
    Highlights   []string
}

func Extract(articles []Article) []ExtractionResult {
    var hits []rawHit

    for _, a := range articles {
        text := a.Title + " " + a.Description
        headlineUpper := strings.ToUpper(a.Title)

        for _, p := range symbolPatterns {
            matches := p.regex.FindAllStringSubmatch(text, -1)
            for _, m := range matches {
                symbol := strings.ToUpper(p.template)
                for i, submatch := range m[1:] {
                    symbol = strings.Replace(symbol, "$"+string(rune('0'+i+1)), strings.ToUpper(submatch), 1)
                }
                // Normalize HK codes: strip leading zeros, keep 4-5 digits
                if strings.HasPrefix(symbol, "HK:") {
                    code := strings.TrimPrefix(symbol, "HK:")
                    code = strings.TrimLeft(code, "0")
                    if len(code) < 4 {
                        code = strings.Repeat("0", 4-len(code)) + code
                    }
                    symbol = "HK:" + code
                }

                isHeadline := strings.Contains(headlineUpper, m[0])

                // Extract snippet around the match (120 chars max)
                idx := strings.Index(strings.ToUpper(text), strings.ToUpper(m[0]))
                snippet := ""
                if idx >= 0 {
                    start := idx - 30
                    if start < 0 {
                        start = 0
                    }
                    end := idx + len(m[0]) + 90
                    if end > len(text) {
                        end = len(text)
                    }
                    snippet = strings.TrimSpace(text[start:end])
                    if len(snippet) > 120 {
                        snippet = snippet[:120]
                    }
                }

                hits = append(hits, rawHit{
                    symbol:      symbol,
                    headline:    isHeadline,
                    publishedAt: a.PublishedAt,
                    snippet:     snippet,
                })
            }
        }
    }

    // Aggregate by symbol
    type agg struct {
        count        int
        headlineHits int
        snippets     []string
        latest       string
    }
    aggs := make(map[string]*agg)
    for _, h := range hits {
        a, ok := aggs[h.symbol]
        if !ok {
            a = &agg{}
            aggs[h.symbol] = a
        }
        a.count++
        if h.headline {
            a.headlineHits++
        }
        if len(a.snippets) < 3 {
            a.snippets = append(a.snippets, h.snippet)
        }
        if h.publishedAt > a.latest {
            a.latest = h.publishedAt
        }
    }

    // Convert to slice, sort by count desc
    results := make([]ExtractionResult, 0, len(aggs))
    for sym, a := range aggs {
        results = append(results, ExtractionResult{
            Symbol:       sym,
            Name:         sym,
            Count:        a.count,
            HeadlineHits: a.headlineHits,
            Highlights:   a.snippets,
        })
    }
    sort.Slice(results, func(i, j int) bool {
        return results[i].Count > results[j].Count
    })

    // Top 15
    if len(results) > 15 {
        results = results[:15]
    }

    return results
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/recommend/ -v -run TestExtract`
Expected: All three tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/recommend/extractor.go internal/recommend/extractor_test.go
git commit -m "feat: add stock symbol extractor from news articles"
```

---

### Task 5: Scorer — Composite scoring

**Files:**
- Create: `internal/recommend/scorer.go`
- Create: `internal/recommend/scorer_test.go`

- [ ] **Step 1: Write the failing test**

```go
package recommend

import (
    "math"
    "testing"

    "github.com/black-eleven/stock-monitor/internal/qos"
)

func TestScore(t *testing.T) {
    results := []ExtractionResult{
        {Symbol: "US:NVDA", Count: 10, HeadlineHits: 5, Highlights: []string{"NVIDIA reports record earnings"}},
        {Symbol: "US:AMD", Count: 3, HeadlineHits: 1, Highlights: []string{"AMD also rose"}},
    }
    quotes := map[string]*qos.Quote{
        "US:NVDA": {Price: 130, YP: 125, Volume: 1000000},
        "US:AMD":  {Price: 80, YP: 82, Volume: 500000},
    }

    recs := Score(results, quotes)

    if len(recs) == 0 {
        t.Fatal("expected recommendations, got empty")
    }

    // NVDA should be ranked first (highest count, positive day change)
    if recs[0].Symbol != "US:NVDA" {
        t.Errorf("expected NVDA first, got %s", recs[0].Symbol)
    }
    if recs[0].Rank != 1 {
        t.Errorf("expected rank 1, got %d", recs[0].Rank)
    }
    if recs[0].Score < 0 || recs[0].Score > 1 {
        t.Errorf("score %f out of range [0,1]", recs[0].Score)
    }
    if recs[0].NewsCount != 10 {
        t.Errorf("expected NewsCount=10, got %d", recs[0].NewsCount)
    }

    // changePercent: NVDA = (130-125)/125 * 100 = 4.0
    expectedChange := ((130.0 - 125.0) / 125.0) * 100.0
    if math.Abs(recs[0].ChangePercent-expectedChange) > 0.01 {
        t.Errorf("expected ChangePercent %.2f, got %.2f", expectedChange, recs[0].ChangePercent)
    }
}

func TestScoreEmpty(t *testing.T) {
    recs := Score(nil, nil)
    if recs == nil {
        t.Error("expected empty slice, got nil")
    }
    if len(recs) != 0 {
        t.Errorf("expected 0, got %d", len(recs))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/recommend/ -v -run TestScore`
Expected: FAIL (function Score not defined)

- [ ] **Step 3: Write scorer implementation**

```go
package recommend

import (
    "math"
    "sort"

    "github.com/black-eleven/stock-monitor/internal/model"
    "github.com/black-eleven/stock-monitor/internal/qos"
)

func Score(results []ExtractionResult, quotes map[string]*qos.Quote) []model.Recommendation {
    if len(results) == 0 {
        return []model.Recommendation{}
    }

    maxCount := 0
    for _, r := range results {
        if r.Count > maxCount {
            maxCount = r.Count
        }
    }
    if maxCount == 0 {
        maxCount = 1
    }

    var recs []model.Recommendation
    for _, r := range results {
        newsScore := newsScore(r, maxCount)
        trendScore := trendScore(r.Symbol, quotes)
        total := newsScore*0.6 + trendScore*0.4

        q, ok := quotes[r.Symbol]
        var price, changePercent float64
        if ok && q != nil {
            price = q.Price
            if q.YP != 0 {
                changePercent = ((q.Price - q.YP) / q.YP) * 100
            }
        }

        recs = append(recs, model.Recommendation{
            Symbol:        r.Symbol,
            Name:          r.Symbol,
            Score:         math.Round(total*100) / 100,
            NewsCount:     r.Count,
            Price:         price,
            ChangePercent: math.Round(changePercent*100) / 100,
            Highlights:    r.Highlights,
        })
    }

    sort.Slice(recs, func(i, j int) bool {
        return recs[i].Score > recs[j].Score
    })

    for i := range recs {
        recs[i].Rank = i + 1
    }

    // Top 10
    if len(recs) > 10 {
        recs = recs[:10]
    }

    return recs
}

func newsScore(r ExtractionResult, maxCount int) float64 {
    freqScore := float64(r.Count) / float64(maxCount) * 0.5        // up to 0.5
    headlineScore := float64(r.HeadlineHits) / float64(r.Count+1) * 0.2 // up to ~0.2
    recencyScore := 0.3                                                      // default recent
    return math.Min(freqScore+headlineScore+recencyScore, 1.0)
}

func trendScore(symbol string, quotes map[string]*qos.Quote) float64 {
    q, ok := quotes[symbol]
    if !ok || q == nil || q.YP == 0 {
        return 0.5 // neutral score when no data
    }

    // Day change score: map changePercent to [0.5, 1] for positive, [0, 0.5] for negative
    changePercent := (q.Price - q.YP) / q.YP // e.g. 0.04 for +4%
    changeScore := 0.5 + changePercent*5      // map roughly -0.1..+0.1 to 0..1
    changeScore = math.Max(0, math.Min(1, changeScore))

    return changeScore * 0.6 + 0.4 // min 0.4 baseline from volume (always bullish when volume present)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/recommend/ -v -run TestScore`
Expected: All tests PASS

- [ ] **Step 5: Run all recommend tests**

Run: `go test ./internal/recommend/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/recommend/scorer.go internal/recommend/scorer_test.go
git commit -m "feat: add composite scoring (news + market trends)"
```

---

### Task 6: Recommender — Orchestrator with cache

**Files:**
- Create: `internal/recommend/recommender.go`

- [ ] **Step 1: Write recommender implementation**

```go
package recommend

import (
    "fmt"
    "sync"
    "time"

    "github.com/black-eleven/stock-monitor/internal/model"
    "github.com/black-eleven/stock-monitor/internal/qos"
)

type cacheEntry struct {
    recs      []model.Recommendation
    expiresAt time.Time
}

type Recommender struct {
    newsapi     *NewsAPIClient
    qosClient   *qos.QosClient
    cache       map[string]*cacheEntry
    cacheTTL    time.Duration
    days        int
    pageSize    int
    mu          sync.RWMutex
}

func NewRecommender(newsapi *NewsAPIClient, qosClient *qos.QosClient, days, pageSize int) *Recommender {
    return &Recommender{
        newsapi:   newsapi,
        qosClient:  qosClient,
        cache:     make(map[string]*cacheEntry),
        cacheTTL:  30 * time.Minute,
        days:      days,
        pageSize:  pageSize,
    }
}

func (r *Recommender) Search(industry string) ([]model.Recommendation, error) {
    // Check cache
    r.mu.RLock()
    if e, ok := r.cache[industry]; ok && time.Now().Before(e.expiresAt) {
        recs := e.recs
        r.mu.RUnlock()
        return recs, nil
    }
    r.mu.RUnlock()

    // 1. Search NewsAPI
    articles, err := r.newsapi.Search(industry, r.days, r.pageSize)
    if err != nil {
        return nil, fmt.Errorf("news search: %w", err)
    }

    if len(articles) == 0 {
        return []model.Recommendation{}, nil
    }

    // 2. Extract stock symbols
    results := Extract(articles)
    if len(results) == 0 {
        return []model.Recommendation{}, nil
    }

    // 3. Fetch quotes for candidates
    quotes := r.batchFetchQuotes(results)

    // 4. Score
    recs := Score(results, quotes)

    // 5. Cache
    r.mu.Lock()
    r.cache[industry] = &cacheEntry{
        recs:      recs,
        expiresAt: time.Now().Add(r.cacheTTL),
    }
    r.mu.Unlock()

    return recs, nil
}

func (r *Recommender) batchFetchQuotes(candidates []ExtractionResult) map[string]*qos.Quote {
    type result struct {
        symbol string
        quote  *qos.Quote
    }

    ch := make(chan result, len(candidates))
    for _, c := range candidates {
        go func(symbol string) {
            q, err := r.qosClient.FetchQuoteCached(symbol)
            if err != nil {
                ch <- result{symbol: symbol}
            } else {
                ch <- result{symbol: symbol, quote: q}
            }
        }(c.Symbol)
    }

    quotes := make(map[string]*qos.Quote, len(candidates))
    for i := 0; i < len(candidates); i++ {
        r := <-ch
        if r.quote != nil {
            quotes[r.symbol] = r.quote
        }
    }
    return quotes
}
```

- [ ] **Step 2: Build check**

Run: `go build ./internal/recommend/...`

- [ ] **Step 3: Commit**

```bash
git add internal/recommend/recommender.go
git commit -m "feat: add recommendation orchestrator with 30-minute cache"
```

---

### Task 7: HTTP Handler

**Files:**
- Create: `internal/handler/recommend.go`

- [ ] **Step 1: Write handler**

```go
package handler

import (
    "net/http"
    "strings"

    "github.com/black-eleven/stock-monitor/internal/model"
    "github.com/black-eleven/stock-monitor/internal/recommend"
    "github.com/gin-gonic/gin"
)

type RecommendHandler struct {
    recommender *recommend.Recommender
}

func NewRecommendHandler(r *recommend.Recommender) *RecommendHandler {
    return &RecommendHandler{recommender: r}
}

func (h *RecommendHandler) Register(api *gin.RouterGroup) {
    api.POST("/recommendations", h.recommend)
}

func (h *RecommendHandler) recommend(c *gin.Context) {
    var req model.RecommendReq
    if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Industry) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "industry is required"})
        return
    }

    recs, err := h.recommender.Search(strings.TrimSpace(req.Industry))
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch recommendations: " + err.Error()})
        return
    }

    if recs == nil {
        recs = []model.Recommendation{}
    }

    c.JSON(http.StatusOK, model.RecommendResp{Recommendations: recs})
}
```

- [ ] **Step 2: Build check**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/handler/recommend.go
git commit -m "feat: add POST /api/recommendations handler"
```

---

### Task 8: Wire Up — Register handler in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add recommendation wiring**

After the klineH handler block, add:

```go
// Recommender
newsapiClient := recommend.NewNewsAPIClient(cfg.NewsAPIKey)
recommender := recommend.NewRecommender(newsapiClient, qosClient, cfg.NewsAPIDays, cfg.NewsAPIPageSize)
recommendH := handler.NewRecommendHandler(recommender)
```

In the route block, after `klineH.Register(auth)`:

```go
recommendH.Register(auth)
```

Add import for `"github.com/black-eleven/stock-monitor/internal/recommend"`.

- [ ] **Step 2: Build to verify**

Run: `go build ./cmd/server/`

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire up recommendation engine in server"
```

---

### Task 9: Flutter — Recommendation Model

**Files:**
- Create: `mobile/stock_monitor/lib/domain/model/recommendation.dart`

- [ ] **Step 1: Write Dart model**

```dart
class Recommendation {
  final String symbol;
  final String name;
  final double score;
  final int newsCount;
  final double price;
  final double changePercent;
  final List<String> highlights;
  final int rank;

  Recommendation({
    required this.symbol,
    required this.name,
    required this.score,
    required this.newsCount,
    required this.price,
    required this.changePercent,
    required this.highlights,
    required this.rank,
  });

  factory Recommendation.fromJson(Map<String, dynamic> json) =>
      Recommendation(
        symbol: json['symbol'] as String,
        name: json['name'] as String,
        score: (json['score'] as num).toDouble(),
        newsCount: json['newsCount'] as int,
        price: (json['price'] as num).toDouble(),
        changePercent: (json['changePercent'] as num).toDouble(),
        highlights: (json['highlights'] as List).cast<String>(),
        rank: json['rank'] as int,
      );
}
```

- [ ] **Step 2: Commit**

```bash
git add mobile/stock_monitor/lib/domain/model/recommendation.dart
git commit -m "feat: add Recommendation Dart model"
```

---

### Task 10: Flutter — Recommend API Client

**Files:**
- Create: `mobile/stock_monitor/lib/data/api/recommend_api.dart`

- [ ] **Step 1: Write API client**

```dart
import '../../domain/model/recommendation.dart';
import 'api_client.dart';

class RecommendApi {
  final ApiClient _client;
  RecommendApi(this._client);

  Future<List<Recommendation>> recommend(String industry) async {
    final res = await _client.post('/recommendations', data: {'industry': industry});
    final list = res.data['recommendations'] as List;
    return list.map((e) => Recommendation.fromJson(e)).toList();
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add mobile/stock_monitor/lib/data/api/recommend_api.dart
git commit -m "feat: add RecommendApi Flutter client"
```

---

### Task 11: Flutter — Add Provider

**Files:**
- Modify: `mobile/stock_monitor/lib/presentation/providers/api_providers.dart`

- [ ] **Step 1: Add recommendApiProvider**

Add import:
```dart
import '../../data/api/recommend_api.dart';
```

Add provider:
```dart
final recommendApiProvider = Provider((ref) => RecommendApi(ref.watch(apiClientProvider)));
```

- [ ] **Step 2: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/providers/api_providers.dart
git commit -m "feat: add recommendApiProvider"
```

---

### Task 12: Flutter — Refactor WatchlistScreen to TabBarView

**Files:**
- Modify: `mobile/stock_monitor/lib/presentation/screens/watchlist_screen.dart`

- [ ] **Step 1: Rewrite watchlist_screen.dart with tabs**

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/stock.dart';
import '../../domain/model/recommendation.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';
import '../widgets/stock_card.dart';

class WatchlistScreen extends ConsumerStatefulWidget {
  const WatchlistScreen({super.key});
  @override
  ConsumerState<WatchlistScreen> createState() => _WatchlistScreenState();
}

class _WatchlistScreenState extends ConsumerState<WatchlistScreen>
    with SingleTickerProviderStateMixin {
  List<WatchlistItem>? _watchlist;
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _load();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final api = ref.read(watchlistApiProvider);
    final list = await api.getAll();
    setState(() => _watchlist = list);
  }

  Future<void> _add() async {
    final symbolCtrl = TextEditingController();
    final nameCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('添加自选'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码 (如 HK:700)')),
          const SizedBox(height: 12),
          TextField(controller: nameCtrl, decoration: const InputDecoration(hintText: '名称 (如 腾讯控股)')),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('添加')),
        ],
      ),
    );

    if (ok == true && symbolCtrl.text.isNotEmpty && nameCtrl.text.isNotEmpty) {
      try {
        await ref.read(watchlistApiProvider).add(symbolCtrl.text.toUpperCase(), nameCtrl.text);
        await _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  void _showDetail(WatchlistItem item) {
    final quote = ref.read(quoteProvider).quotes[item.symbol];
    if (quote == null) return;
    showModalBottomSheet(
      context: context,
      builder: (_) => StockDetailSheet(
        item: item,
        quote: quote,
        onDelete: () async {
          await ref.read(watchlistApiProvider).remove(item.symbol);
          _load();
        },
      ),
    );
  }

  void _openKline(String symbol) {
    Navigator.of(context).pushNamed('/kline', arguments: {'symbol': symbol});
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('自选股'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: '我的自选'),
            Tab(text: '推荐发现'),
          ],
        ),
        actions: [
          IconButton(onPressed: _add, icon: const Icon(Icons.add)),
        ],
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildWatchlistTab(),
          _buildRecommendTab(),
        ],
      ),
    );
  }

  Widget _buildWatchlistTab() {
    if (_watchlist == null) return const Center(child: CircularProgressIndicator());
    final quotes = ref.watch(quoteProvider).quotes;

    if (_watchlist!.isEmpty) {
      return const Center(
        child: Text('暂无自选股\n点击右上角 + 添加',
            textAlign: TextAlign.center, style: TextStyle(color: AppTheme.textSecondary)),
      );
    }

    return ListView.builder(
      itemCount: _watchlist!.length,
      itemBuilder: (_, i) {
        final item = _watchlist![i];
        return StockCard(
          item: item,
          quote: quotes[item.symbol],
          onTap: () => _showDetail(item),
          onDelete: () async {
            await ref.read(watchlistApiProvider).remove(item.symbol);
            _load();
          },
        );
      },
    );
  }

  Widget _buildRecommendTab() {
    return _RecommendTab(
      onAddToWatchlist: (symbol, name) async {
        try {
          await ref.read(watchlistApiProvider).add(symbol, name);
          await _load();
          _tabController.animateTo(0);
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('已添加 $name 到自选股')),
            );
          }
        } catch (e) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('添加失败: $e')),
            );
          }
        }
      },
      onOpenKline: _openKline,
    );
  }
}

class _RecommendTab extends StatefulWidget {
  final Future<void> Function(String symbol, String name) onAddToWatchlist;
  final void Function(String symbol) onOpenKline;

  const _RecommendTab({required this.onAddToWatchlist, required this.onOpenKline});

  @override
  State<_RecommendTab> createState() => _RecommendTabState();
}

class _RecommendTabState extends State<_RecommendTab> {
  final _controller = TextEditingController();
  List<Recommendation>? _recs;
  String? _error;
  bool _loading = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final industry = _controller.text.trim();
    if (industry.isEmpty) return;

    setState(() {
      _loading = true;
      _error = null;
      _recs = null;
    });

    try {
      final api = RecommendApiProvider.of(context);
      final recs = await api.recommend(industry);
      setState(() {
        _recs = recs;
        _loading = false;
        if (recs.isEmpty) {
          _error = '未找到相关推荐';
        }
      });
    } catch (e) {
      setState(() {
        _error = '获取推荐失败: $e';
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _controller,
                  decoration: const InputDecoration(
                    hintText: '输入行业关键词 (如 AI, 新能源, 半导体)',
                    border: OutlineInputBorder(),
                    contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                  onSubmitted: (_) => _search(),
                ),
              ),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: _loading ? null : _search,
                child: _loading
                    ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : const Text('搜索'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Expanded(child: _buildResults()),
        ],
      ),
    );
  }

  Widget _buildResults() {
    if (_error != null) {
      return Center(
        child: Text(_error!, style: const TextStyle(color: AppTheme.textSecondary)),
      );
    }
    if (_recs == null) {
      return const Center(
        child: Text('输入行业关键词搜索推荐股票', style: TextStyle(color: AppTheme.textSecondary)),
      );
    }

    return ListView.builder(
      itemCount: _recs!.length,
      itemBuilder: (_, i) {
        final r = _recs![i];
        return _RecommendCard(
          rec: r,
          onAdd: () => widget.onAddToWatchlist(r.symbol, r.name),
          onTap: () => widget.onOpenKline(r.symbol),
        );
      },
    );
  }
}

class _RecommendCard extends StatelessWidget {
  final Recommendation rec;
  final VoidCallback onAdd;
  final VoidCallback onTap;

  const _RecommendCard({required this.rec, required this.onAdd, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final changeDir = rec.changePercent >= 0 ? 'up' : 'down';
    final changeColor = changeDir == 'up' ? AppTheme.up : AppTheme.down;

    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: AppTheme.up.withAlpha(25),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text('#${rec.rank}', style: const TextStyle(fontSize: 12, color: AppTheme.up, fontWeight: FontWeight.w700)),
                  ),
                  const SizedBox(width: 8),
                  Text(rec.symbol, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16, color: AppTheme.textPrimary)),
                  const Spacer(),
                  if (rec.price > 0)
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(formatPrice(rec.price), style: TextStyle(fontWeight: FontWeight.w700, color: changeColor)),
                        Text('${rec.changePercent >= 0 ? '+' : ''}${rec.changePercent.toStringAsFixed(2)}%', style: TextStyle(fontSize: 12, color: changeColor)),
                      ],
                    ),
                  const SizedBox(width: 8),
                  IconButton(
                    icon: const Icon(Icons.add_circle_outline, color: AppTheme.up),
                    onPressed: onAdd,
                    tooltip: '加入自选',
                  ),
                ],
              ),
              if (rec.highlights.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: rec.highlights.take(2).map((h) => Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: AppTheme.textSecondary.withAlpha(20),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(h, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
                  )).toList(),
                ),
              ],
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(Icons.article_outlined, size: 14, color: AppTheme.textSecondary.withAlpha(150)),
                  const SizedBox(width: 4),
                  Text('${rec.newsCount} 篇相关新闻', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary.withAlpha(150))),
                  const SizedBox(width: 12),
                  Icon(Icons.auto_awesome, size: 14, color: AppTheme.textSecondary.withAlpha(150)),
                  const SizedBox(width: 4),
                  Text('综合评分 ${(rec.score * 100).toStringAsFixed(0)}', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary.withAlpha(150))),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// Provide access to RecommendApi without Riverpod dependency in private widget
class RecommendApiProvider {
  static RecommendApi of(BuildContext context) {
    // Use the widget tree to resolve — In Flutter, we use ProviderScope.containerOf
    final container = ProviderScope.containerOf(context);
    return container.read(recommendApiProvider);
  }
}
```

- [ ] **Step 2: Add import for RecommendApi in dart file**

The `ProviderScope.containerOf` approach requires `package:flutter_riverpod/flutter_riverpod.dart` import too (it usually already works within a ConsumerWidget tree). The widget already imports consumer-related packages at the top of the file.

- [ ] **Step 3: Build check**

Run: `cd mobile/stock_monitor && flutter build apk --debug 2>&1 | tail -5`
Expected: No compile errors

- [ ] **Step 4: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/watchlist_screen.dart
git commit -m "feat: add recommend tab to watchlist screen with search and results"
```

---

## Plan self-review

**Spec coverage:**
- ✅ Config env vars → Task 1
- ✅ NewsAPI client → Task 3
- ✅ Symbol extraction → Task 4
- ✅ Scoring algorithm → Task 5
- ✅ Recommender orchestrator + cache → Task 6
- ✅ HTTP handler POST /api/recommendations → Task 7
- ✅ Wire-up in main.go → Task 8
- ✅ Flutter model → Task 9
- ✅ Flutter API client → Task 10
- ✅ Flutter provider → Task 11
- ✅ TabBarView refactor → Task 12
- ✅ JWT auth — inherited from existing auth middleware group (auth := api.Group("", authMW))

**No placeholders:** All tasks contain complete code, exact commands, expected output.

**Type consistency:**
- `Recommendation` struct defined in Task 2 matches usage in Tasks 5, 6, 7, 9, 12
- `ExtractionResult` defined in Task 4 matches usage in Tasks 5, 6
- `Article` defined in Task 3 matches usage in Task 4
- `RecommendApi` defined in Task 10 matches usage in Tasks 11, 12
- `recommendApiProvider` defined in Task 11 matches usage in Task 12
