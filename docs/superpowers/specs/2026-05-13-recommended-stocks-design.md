# 智能推荐自选股 — Design Spec

## Overview

基于用户指定的行业关键词，通过 NewsAPI 搜索近期新闻并自动提取相关股票代码，结合 QOS 实时行情数据综合打分，推荐值得关注的自选股。用户可在推荐结果中一键添加到自选列表。

## Entry Point

自选股页面（WatchlistScreen）改为 TabBarView 双标签：

- **我的自选** — 现有自选列表
- **推荐发现** — 新增，包含行业输入框 + 推荐结果列表

## Architecture

### Backend (Go) — New Components

```
internal/recommend/
├── recommender.go   — 编排推荐流程
├── newsapi.go       — NewsAPI.org 客户端
├── extractor.go     — 从新闻文本提取股票代码
└── scorer.go        — 综合打分

internal/handler/
└── recommend.go     — HTTP Handler: POST /api/recommendations
```

### Frontend (Flutter) — Modified/New Components

```
mobile/.../data/api/
└── recommend_api.dart           — 推荐 API 客户端

mobile/.../presentation/screens/
└── watchlist_screen.dart        — 改为 TabBarView（我的自选 / 推荐发现）
```

## API Contract

### Request

```
POST /api/recommendations
Authorization: Bearer <jwt>
Content-Type: application/json

{ "industry": "人工智能" }
```

### Response

```json
{
  "recommendations": [
    {
      "symbol": "US:NVDA",
      "name": "NVIDIA Corp",
      "score": 0.92,
      "newsCount": 15,
      "price": 128.50,
      "changePercent": 2.35,
      "highlights": ["AI芯片需求强劲", "Q1财报超预期"],
      "rank": 1
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `score` | Composite score 0–1 |
| `newsCount` | Number of recent related news articles |
| `highlights` | Key phrases from news (max 3) |
| `changePercent` | Current day change % from QOS quote |

### NewsAPI Configuration

```
NEWSAPI_KEY=your_key       # NewsAPI.org API key
NEWSAPI_DAYS=7             # Search news from last 7 days
NEWSAPI_PAGE_SIZE=50       # Max 50 articles per search
```

## Stock Symbol Extraction (extractor.go)

Match stock codes from news headlines and descriptions:

- `US:[A-Z]{1,5}` → US equities
- `HK:\d{4,5}` → Hong Kong equities
- `SH:\d{6}` / `SZ:\d{6}` → China A-shares

Enhanced patterns (no prefix, common formats):
- `$NVDA` / `$AAPL` → `US:NVDA`
- `00700.HK` / `700.HK` → `HK:0700`
- `600519.SH` → `SH:600519`

Deduplicate, sort by mention frequency, take top 15 for scoring.

Highlights extraction: for each matched stock, extract the sentence containing the mention (up to 120 chars). Prefer the headline when the mention appears there. Store up to 3 highlights per stock, from the 3 most recent articles.

## Scoring Algorithm (scorer.go)

```
Total = NewsScore × 0.6 + TrendScore × 0.4
```

### NewsScore (0–1)

| Metric | Weight | Logic |
|--------|--------|-------|
| Mention frequency ratio | 50% | count / max_count in batch |
| Recency | 30% | Decay factor for articles older than 24h |
| Headline hits | 20% | Headline mention > body mention |

### TrendScore (0–1)

Uses data available from QOS Quote snapshot — no additional kline fetch required.

| Metric | Weight | Logic |
|--------|--------|-------|
| Day change % | 60% | `(price − yp) / yp`, positive → 0.5–1, negative → 0–0.5 |
| Volume signal | 40% | `volume > 0` and price above yesterday's midpoint → bullish signal (0–1) |

## Data Flow

```
User selects/enters industry keyword
  → POST /api/recommendations { industry }
  → recommender.Search(industry):
    1. newsapi.Search(industry) — fetch news via NewsAPI (sortBy=popularity)
    2. extractor.Extract(articles) — regex match stock codes
    3. Deduplicate, top 15 candidates
    4. Call qos.FetchQuoteCached(symbol) for each candidate in goroutines (same pattern as quote handler's /quote/batch) — get live quotes
    5. scorer.Score(candidates, articles, quotes) — composite ranking
    6. Return top 10
  → Frontend renders recommendation cards with "+" button
  → Tap "+" → POST /api/watchlist { symbol, name } adds to watchlist
```

## Security & Error Handling

- NewsAPI key stored in env var, never exposed to frontend
- JWT auth required on `/api/recommendations`
- Graceful degradation: if NewsAPI fails, return error with clear message
- Empty results: return `{ "recommendations": [] }` with message "未找到相关推荐"
- Rate limiting: cache recommendations per industry for 30 minutes

## Implementation Order

1. Backend: `internal/recommend/` + `recommend handler` + route registration
2. Backend: Add `NEWSAPI_*` env vars to config
3. Frontend: `RecommendApi` client class
4. Frontend: TabBarView refactor on WatchlistScreen + RecommendTab widget
5. Integration test: search industry → get recommendations → add to watchlist
