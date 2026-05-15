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
	newsapi   *NewsAPIClient
	emClient  eastmoney.QuoteClient
	cache     map[string]*cacheEntry
	cacheTTL  time.Duration
	days      int
	pageSize  int
	languages []string
	candidates int
	limit     int
	mu        sync.RWMutex
}

func NewRecommender(newsapi *NewsAPIClient, emClient eastmoney.QuoteClient, days, pageSize int, languages []string, candidates, limit int) *Recommender {
	return &Recommender{
		newsapi:   newsapi,
		emClient:  emClient,
		cache:     make(map[string]*cacheEntry),
		cacheTTL:  30 * time.Minute,
		days:      days,
		pageSize:  pageSize,
		languages: languages,
		candidates: candidates,
		limit:     limit,
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

	// 1. Search NewsAPI for all configured languages concurrently
	articles, err := r.searchAllLanguages(industry)
	if err != nil {
		return nil, fmt.Errorf("news search: %w", err)
	}

	if len(articles) == 0 {
		return []model.Recommendation{}, nil
	}

	// 2. Extract stock symbols
	results := Extract(articles, r.candidates)
	if len(results) == 0 {
		return []model.Recommendation{}, nil
	}

	// 3. Fetch quotes for candidates
	quotes := r.batchFetchQuotes(results)

	// Filter candidates to only those QOS recognizes (has a quote)
	validResults := make([]ExtractionResult, 0, len(results))
	for _, res := range results {
		if _, ok := quotes[res.Symbol]; ok {
			validResults = append(validResults, res)
		}
	}
	// If every candidate was rejected, fall back to original results rather than returning empty
	if len(validResults) == 0 {
		validResults = results
	}

	// 4. Score
	recs := Score(validResults, quotes, r.limit)

	// 5. Cache
	r.mu.Lock()
	r.cache[industry] = &cacheEntry{
		recs:      recs,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
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
		r := <-ch
		if r.quote != nil {
			quotes[r.symbol] = r.quote
		}
	}
	return quotes
}
