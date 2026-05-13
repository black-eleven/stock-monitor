package recommend

import (
	"regexp"
	"sort"
	"strings"
)

// nonStockWords are common acronyms that look like tickers but aren't.
var nonStockWords = map[string]bool{
	"EU": true, "UK": true, "US": true, "IPO": true, "ETF": true,
	"GDP": true, "PMI": true, "CPI": true, "PPI": true, "CEO": true, "CFO": true,
	"FDA": true, "SEC": true, "IRS": true, "FBI": true, "CIA": true, "WHO": true,
	"API": true, "URL": true, "HTML": true, "CSS": true, "JSON": true, "XML": true,
	"HTTP": true, "HTTPS": true, "SSH": true, "VPN": true, "DNS": true, "TCP": true,
	"ISO": true, "OCR": true, "NLP": true, "LLM": true, "AGI": true, "GPU": true,
	"CPU": true, "RAM": true, "SSD": true, "HDD": true, "USB": true, "LCD": true,
	"LED": true, "OLED": true, "IOT": true,
	"ROI": true, "ROE": true, "YOY": true, "QOQ": true, "FYI": true, "FAQ": true,
}

var symbolPatterns = []struct {
	regex    *regexp.Regexp
	template string
}{
	// Explicit prefixed symbols (already precise)
	{regexp.MustCompile(`US:([A-Z]{1,5})`), "US:$1"},
	{regexp.MustCompile(`HK:(\d{4,5})`), "HK:$1"},
	{regexp.MustCompile(`SH:(\d{6})`), "SH:$1"},
	{regexp.MustCompile(`SZ:(\d{6})`), "SZ:$1"},
	// $TICKER format → US (require 2-5 chars, skip non-stock words)
	{regexp.MustCompile(`\$([A-Z]{2,5})`), "US:$1"},
	// TICKER.HK format → HK; reject years (19xx/20xx) to avoid news dates
	{regexp.MustCompile(`(\d{1,5})\.HK`), "HK:$1"},
	// TICKER.SH format
	{regexp.MustCompile(`(\d{6})\.SH`), "SH:$1"},
	// TICKER.SZ format
	{regexp.MustCompile(`(\d{6})\.SZ`), "SZ:$1"},
	// (TICKER) parenthetical reference → US; require 2-5 chars, skip non-stock words
	{regexp.MustCompile(`\(([A-Z]{2,5})\)`), "US:$1"},
}

// isFakeHKCode filters out numbers unlikely to be real HK stock codes.
func isFakeHKCode(code string) bool {
	// Years like 1900-2099 in news are almost never HK stock codes
	if len(code) == 4 {
		if code >= "1900" && code <= "2099" {
			return true
		}
	}
	return false
}

type rawHit struct {
	symbol      string
	headline    bool
	publishedAt string
	snippet     string
}

// ExtractionResult represents a single extracted stock symbol with its metadata.
type ExtractionResult struct {
	Symbol       string
	Name         string
	Count        int
	HeadlineHits int
	Highlights   []string
}

// Extract scans a slice of Articles and returns deduplicated ExtractionResults
// sorted by frequency (descending), capped at limit.
func Extract(articles []Article, limit int) []ExtractionResult {
	if len(articles) == 0 {
		return nil
	}

	var hits []rawHit

	for _, a := range articles {
		text := a.Title + " " + a.Description
		headlineUpper := strings.ToUpper(a.Title)

		for _, p := range symbolPatterns {
			matches := p.regex.FindAllStringSubmatch(text, -1)
			for _, m := range matches {
				rawCode := strings.ToUpper(m[1])

				// Skip non-stock words for US ticker patterns
				if strings.Contains(p.template, "US:") && nonStockWords[rawCode] {
					continue
				}

				// Skip fake HK codes (years, etc.)
				if strings.Contains(p.template, "HK:") && isFakeHKCode(rawCode) {
					continue
				}

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

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}
