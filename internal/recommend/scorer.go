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
			Name:          r.Name,
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
	freqScore := float64(r.Count) / float64(maxCount) * 0.5
	headlineScore := float64(r.HeadlineHits) / float64(r.Count+1) * 0.2
	recencyScore := 0.3
	return math.Min(freqScore+headlineScore+recencyScore, 1.0)
}

func trendScore(symbol string, quotes map[string]*qos.Quote) float64 {
	q, ok := quotes[symbol]
	if !ok || q == nil || q.YP == 0 {
		return 0.5
	}

	changePercent := (q.Price - q.YP) / q.YP
	changeScore := 0.5 + changePercent*5
	changeScore = math.Max(0, math.Min(1, changeScore))

	return changeScore*0.6 + 0.4
}
