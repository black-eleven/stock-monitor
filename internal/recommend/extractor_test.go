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
