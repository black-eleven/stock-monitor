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
