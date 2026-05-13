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
	newsapi   *NewsAPIClient
	qosClient *qos.QosClient
	cache     map[string]*cacheEntry
	cacheTTL  time.Duration
	days      int
	pageSize  int
	mu        sync.RWMutex
}

func NewRecommender(newsapi *NewsAPIClient, qosClient *qos.QosClient, days, pageSize int) *Recommender {
	return &Recommender{
		newsapi:   newsapi,
		qosClient: qosClient,
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
