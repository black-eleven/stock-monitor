# LLM-Powered Stock Recommendation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace NewsAPI+regex stock recommendation with DeepSeek LLM API.

**Architecture:** New `internal/llm/` package sends industry keyword to DeepSeek API, returns structured JSON with stock recommendations. Recommender calls LLM instead of NewsAPI, validates symbols against quotes, scores, and caches. Frontend unchanged.

**Tech Stack:** Go `net/http`, DeepSeek API (OpenAI-compatible), existing Gin + SQLite.

---

### Task 1: Create `internal/llm/client.go` + `internal/llm/prompt.go`

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/prompt.go`

`client.go`:
```go
package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Candidate struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Client struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = "deepseek-chat"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		model:      model,
		baseURL:    "https://api.deepseek.com/v1/chat/completions",
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *responseFmt  `json:"response_format,omitempty"`
}

type responseFmt struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) Recommend(industry string) ([]Candidate, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("行业：%s", industry)},
		},
		Temperature:    0.3,
		ResponseFormat: &responseFmt{Type: "json_object"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", c.baseURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("llm parse: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("llm api error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("llm: no choices returned")
	}

	content := chatResp.Choices[0].Message.Content

	// DeepSeek with json_object wraps in {"recommendations": [...]}
	var wrapper struct {
		Recommendations []Candidate `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
		// Try direct array fallback
		var candidates []Candidate
		if err2 := json.Unmarshal([]byte(content), &candidates); err2 != nil {
			return nil, fmt.Errorf("llm json parse: %w (content: %s)", err, truncate(content, 200))
		}
		return candidates, nil
	}
	return wrapper.Recommendations, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
```

`prompt.go`:
```go
package llm

const systemPrompt = `你是一个股票推荐助手。用户输入一个行业或主题，请你推荐与该行业最相关的上市公司股票。

要求：
1. 覆盖港股(HK)、A股(SH/SZ)、美股(US)市场，每个市场最多推荐5只，总数不超过15只
2. symbol格式严格遵守：港股"HK:0000"（4位补零），沪市"SH:000000"（6位），深市"SZ:000000"（6位），美股"US:TICKER"（大写）
3. 优先推荐行业龙头和核心受益标的
4. reason字段用中文简要说明推荐理由（20字以内）
5. 返回JSON对象，格式为{"recommendations": [...]}，不要markdown代码块

示例输出：
{"recommendations":[{"symbol":"HK:0700","name":"腾讯控股","reason":"社交和游戏龙头"},{"symbol":"SH:600519","name":"贵州茅台","reason":"白酒行业绝对龙头"}]}`
```

- [ ] **Step 1: Commit**

```bash
git add internal/llm/
git commit -m "feat: add DeepSeek LLM client for stock recommendation"
```

---

### Task 2: Modify `internal/config/config.go` — Add DeepSeek config

**Files:**
- Modify: `internal/config/config.go`

Add to `Config` struct:
```go
DeepSeekAPIKey string
DeepSeekModel  string
LLMCacheTTL    int
```

Add to `Load()`:
```go
deepSeekAPIKey := os.Getenv("DEEPSEEK_API_KEY")
if deepSeekAPIKey == "" {
    log.Printf("[CONFIG] DEEPSEEK_API_KEY not set — LLM recommendation will be unavailable")
}
deepSeekModel := os.Getenv("DEEPSEEK_MODEL")
if deepSeekModel == "" {
    deepSeekModel = "deepseek-chat"
}
llmCacheTTL := 30
if s := os.Getenv("LLM_CACHE_TTL"); s != "" {
    if n, err := strconv.Atoi(s); err == nil && n > 0 {
        llmCacheTTL = n
    }
}
```

Add to return `&Config{...}`:
```go
DeepSeekAPIKey: deepSeekAPIKey,
DeepSeekModel:  deepSeekModel,
LLMCacheTTL:    llmCacheTTL,
```

Remove NewsAPI config fields from `Config` struct and `Load()`: `NewsAPIKey`, `NewsAPIDays`, `NewsAPIPageSize`, `NewsAPILanguages`, `RecommendCandidates`. Keep `RecommendLimit`.

- [ ] **Step 1: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor: replace NewsAPI config with DeepSeek LLM config"
```

---

### Task 3: Rewrite `internal/recommend/recommender.go`

**Files:**
- Modify: `internal/recommend/recommender.go`

```go
package recommend

import (
	"fmt"
	"sync"
	"time"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/black-eleven/stock-monitor/internal/llm"
	"github.com/black-eleven/stock-monitor/internal/model"
)

type cacheEntry struct {
	recs      []model.Recommendation
	expiresAt time.Time
}

type Recommender struct {
	llmClient *llm.Client
	emClient  eastmoney.QuoteClient
	cache     map[string]*cacheEntry
	cacheTTL  time.Duration
	limit     int
	mu        sync.RWMutex
}

func NewRecommender(llmClient *llm.Client, emClient eastmoney.QuoteClient, cacheTTLMin, limit int) *Recommender {
	return &Recommender{
		llmClient: llmClient,
		emClient:  emClient,
		cache:     make(map[string]*cacheEntry),
		cacheTTL:  time.Duration(cacheTTLMin) * time.Minute,
		limit:     limit,
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

	// 1. Call LLM
	candidates, err := r.llmClient.Recommend(industry)
	if err != nil {
		return nil, fmt.Errorf("llm recommend: %w", err)
	}
	if len(candidates) == 0 {
		return []model.Recommendation{}, nil
	}

	// 2. Fetch quotes for candidates (best-effort)
	quotes := r.batchFetchQuotes(candidates)

	// 3. Build results
	recs := make([]model.Recommendation, 0, len(candidates))
	for i, c := range candidates {
		var price, changePercent float64
		if q, ok := quotes[c.Symbol]; ok && q != nil {
			price = q.Price
			if q.YP != 0 {
				changePercent = ((q.Price - q.YP) / q.YP) * 100
			}
		}
		recs = append(recs, model.Recommendation{
			Symbol:        c.Symbol,
			Name:          c.Name,
			Score:         float64(len(candidates) - i), // LLM order as score
			NewsCount:     0,
			Price:         price,
			ChangePercent: changePercent,
			Highlights:    []string{c.Reason},
			Rank:          i + 1,
		})
	}

	// 4. Cache
	r.mu.Lock()
	r.cache[industry] = &cacheEntry{recs: recs, expiresAt: time.Now().Add(r.cacheTTL)}
	r.mu.Unlock()

	if len(recs) > r.limit {
		recs = recs[:r.limit]
	}
	return recs, nil
}

func (r *Recommender) batchFetchQuotes(candidates []llm.Candidate) map[string]*eastmoney.Quote {
	symbols := make([]string, len(candidates))
	for i, c := range candidates {
		symbols[i] = c.Symbol
	}

	type result struct {
		symbol string
		quote  *eastmoney.Quote
	}
	ch := make(chan result, len(symbols))
	for _, s := range symbols {
		go func(symbol string) {
			q, err := r.emClient.FetchQuoteCached(symbol)
			if err != nil {
				ch <- result{symbol: symbol}
			} else {
				ch <- result{symbol: symbol, quote: q}
			}
		}(s)
	}

	quotes := make(map[string]*eastmoney.Quote, len(symbols))
	for i := 0; i < len(symbols); i++ {
		r := <-ch
		if r.quote != nil {
			quotes[r.symbol] = r.quote
		}
	}
	return quotes
}
```

- [ ] **Step 1: Update `internal/recommend/scorer.go`**

Remove `newsScore` and `trendScore` functions. Simplify `Score` (or remove it entirely — recommender now uses LLM order directly). The file can be deleted or left as a stub.

- [ ] **Step 2: Commit**

```bash
git add internal/recommend/
git commit -m "refactor: replace NewsAPI recommender with LLM-based recommender"
```

---

### Task 4: Update `cmd/server/main.go` — Wire LLM client

**Files:**
- Modify: `cmd/server/main.go`

Replace (lines 68-71):
```go
// Recommender
newsapiClient := recommend.NewNewsAPIClient(cfg.NewsAPIKey)
recommender := recommend.NewRecommender(newsapiClient, emClient, cfg.NewsAPIDays, cfg.NewsAPIPageSize, cfg.NewsAPILanguages, cfg.RecommendCandidates, cfg.RecommendLimit)
recommendH := handler.NewRecommendHandler(recommender)
```

With:
```go
// Recommender (LLM-based)
recommender := &recommend.Recommender{} // zero value if no API key
if cfg.DeepSeekAPIKey != "" {
    llmClient := llm.NewClient(cfg.DeepSeekAPIKey, cfg.DeepSeekModel)
    recommender = recommend.NewRecommender(llmClient, emClient, cfg.LLMCacheTTL, cfg.RecommendLimit)
}
recommendH := handler.NewRecommendHandler(recommender)
```

Add import: `"github.com/black-eleven/stock-monitor/internal/llm"`

- [ ] **Step 1: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire DeepSeek LLM client into recommender"
```

---

### Task 5: Clean up old files + verify

**Files:**
- Delete: `internal/recommend/newsapi.go`
- Delete: `internal/recommend/extractor.go`
- Delete: `internal/recommend/extractor_test.go`
- Delete: `internal/recommend/scorer.go` (if made obsolete)
- Delete: `internal/recommend/scorer_test.go` (if made obsolete)

- [ ] **Step 1: Delete old files**

```bash
rm -f internal/recommend/newsapi.go \
      internal/recommend/extractor.go \
      internal/recommend/extractor_test.go \
      internal/recommend/scorer.go \
      internal/recommend/scorer_test.go
```

- [ ] **Step 2: Tidy and build**

```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
```

Expected: All pass.

- [ ] **Step 3: Update config_test.go if needed**

Run `go test ./internal/config/ -v` and fix any references to removed NewsAPI config fields.

- [ ] **Step 4: Quick startup test**

```bash
DEEPSEEK_API_KEY=sk-test timeout 5 go run ./cmd/server 2>&1 || true
```

Expected: Server starts, logs config, no panic. If DEEPSEEK_API_KEY is empty, logs warning.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove NewsAPI, extractor, scorer; wire LLM end-to-end"
```
