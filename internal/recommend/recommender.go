package recommend

import (
	"fmt"
	"math"
	"sort"
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

func (r *Recommender) Search(industry string, exclude []string) ([]model.Recommendation, error) {
	r.mu.RLock()
	if e, ok := r.cache[industry]; ok && time.Now().Before(e.expiresAt) {
		recs := e.recs
		r.mu.RUnlock()
		return recs, nil
	}
	r.mu.RUnlock()

	candidates, err := r.llm.Recommend(industry)
	if err != nil {
		return nil, fmt.Errorf("llm recommend: %w", err)
	}
	if len(candidates) == 0 {
		return []model.Recommendation{}, nil
	}

	// Filter out watchlist duplicates
	if len(exclude) > 0 {
		excluded := make(map[string]bool, len(exclude))
		for _, s := range exclude {
			excluded[s] = true
		}
		filtered := make([]llm.Candidate, 0, len(candidates))
		for _, c := range candidates {
			if !excluded[c.Symbol] {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
		if len(candidates) == 0 {
			return []model.Recommendation{}, nil
		}
	}

	symbols := make([]string, 0, len(candidates))
	for _, c := range candidates {
		symbols = append(symbols, c.Symbol)
	}
	quotes := r.batchFetchQuotes(symbols)

	// Find max volume for normalization
	maxVol := 0.0
	for _, q := range quotes {
		if q != nil && q.Volume > maxVol {
			maxVol = q.Volume
		}
	}

	// Score and build
	total := len(candidates)
	recs := make([]model.Recommendation, 0, total)
	for i, c := range candidates {
		price := 0.0
		changePercent := 0.0
		volume := 0.0
		hasQuote := false
		if q, ok := quotes[c.Symbol]; ok && q != nil {
			hasQuote = true
			price = q.Price
			volume = q.Volume
			if q.YP != 0 {
				changePercent = ((q.Price - q.YP) / q.YP) * 100
			}
		}

		// Composite: LLM 30% + Quote 25% + Momentum 25% + Volume 20%
		llmScore := 1.0 - float64(i)/float64(total)
		quoteScore := 0.0
		if hasQuote {
			quoteScore = 1.0
		}
		momentumScore := math.Max(0, math.Min(1, 0.5+changePercent*0.05))
		volumeScore := 0.0
		if maxVol > 0 && volume > 0 {
			volumeScore = math.Min(1, volume/maxVol)
		}
		finalScore := math.Round((llmScore*0.3+quoteScore*0.25+momentumScore*0.25+volumeScore*0.2)*100) / 100

		rec := model.Recommendation{
			Symbol:        c.Symbol,
			Name:          c.Name,
			Score:         finalScore,
			NewsCount:     1,
			Price:         price,
			ChangePercent: changePercent,
		}
		if c.Reason != "" {
			rec.Highlights = []string{c.Reason}
		}
		recs = append(recs, rec)
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score > recs[j].Score
	})

	if len(recs) > r.limit {
		recs = recs[:r.limit]
	}
	for i := range recs {
		recs[i].Rank = i + 1
	}

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
