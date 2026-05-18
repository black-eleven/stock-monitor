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
	llm      *llm.Client
	emClient eastmoney.QuoteClient
	cache    map[string]*cacheEntry
	cacheTTL time.Duration
	limit    int
	mu       sync.RWMutex
}

func NewRecommender(llmClient *llm.Client, emClient eastmoney.QuoteClient, cacheTTL, limit int) *Recommender {
	return &Recommender{
		llm:      llmClient,
		emClient: emClient,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: time.Duration(cacheTTL) * time.Minute,
		limit:    limit,
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

	// 1. Call LLM for recommendations
	candidates, err := r.llm.Recommend(industry)
	if err != nil {
		return nil, fmt.Errorf("llm recommend: %w", err)
	}

	if len(candidates) == 0 {
		return []model.Recommendation{}, nil
	}

	// 2. Fetch quotes for candidates
	symbols := make([]string, 0, len(candidates))
	for _, c := range candidates {
		symbols = append(symbols, c.Symbol)
	}
	quotes := r.batchFetchQuotes(symbols)

	// 3. Build recommendations
	recs := make([]model.Recommendation, 0, len(candidates))
	for _, c := range candidates {
		price := 0.0
		changePercent := 0.0
		if q, ok := quotes[c.Symbol]; ok && q != nil {
			price = q.Price
			if q.YP != 0 {
				changePercent = ((q.Price - q.YP) / q.YP) * 100
			}
		}

		rec := model.Recommendation{
			Symbol:        c.Symbol,
			Name:          c.Name,
			Score:         1.0,
			NewsCount:     1,
			Price:         price,
			ChangePercent: changePercent,
		}
		if c.Reason != "" {
			rec.Highlights = []string{c.Reason}
		}
		recs = append(recs, rec)
	}

	// Apply limit
	if len(recs) > r.limit {
		recs = recs[:r.limit]
	}

	// Assign ranks
	for i := range recs {
		recs[i].Rank = i + 1
	}

	// 4. Cache
	r.mu.Lock()
	r.cache[industry] = &cacheEntry{
		recs:      recs,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
	r.mu.Unlock()

	return recs, nil
}

func (r *Recommender) batchFetchQuotes(symbols []string) map[string]*eastmoney.Quote {
	type result struct {
		symbol string
		quote  *eastmoney.Quote
	}

	ch := make(chan result, len(symbols))
	for _, sym := range symbols {
		go func(symbol string) {
			q, err := r.emClient.FetchQuoteCached(symbol)
			if err != nil {
				ch <- result{symbol: symbol}
			} else {
				ch <- result{symbol: symbol, quote: q}
			}
		}(sym)
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
