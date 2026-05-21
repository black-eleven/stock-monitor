# Stock Monitor — Project Context

## Overview

Personal stock monitoring tool. Go backend + vanilla JS frontend + Flutter mobile + SQLite.

## Build & Run

```bash
go build -o bin/server ./cmd/server
./bin/server                     # listens on :3000 by default
go test ./internal/...           # run tests
cd mobile/stock_monitor && flutter run  # mobile app
```

## Architecture

```
cmd/server/main.go          → entry point, wiring
internal/
  config/                   → env var loading (.env)
  db/                       → SQLite setup, schema, migrations
  model/                    → data structs
  repo/                     → DB access layer (CRUD per entity)
  handler/                  → Gin HTTP handlers (REST API)
    strategy.go             → LLM strategy analysis + caching
    signal_handler.go       → signal recording + threshold alerts
  middleware/                → JWT auth middleware
  eastmoney/                → stock quote + kline data client (multi-source)
  llm/                      → DeepSeek API client + strategy prompts
  recommend/                → recommendation orchestrator (LLM + scoring + dedup)
  alert/                    → price alert evaluation engine
  ws/                       → WebSocket hub (browser push)
web/
  index.html, login.html, admin.html
  js/                       → vanilla JS components
mobile/stock_monitor/       → Flutter app (Riverpod + GoRouter)
```

## API Endpoints

All under `/api`, JWT required except `/auth/*`:

| Method | Path | Description |
|--------|------|-------------|
| POST | /auth/login | Login, returns JWT |
| POST | /auth/register | Register with invite code |
| GET/POST/DELETE | /watchlist[/:symbol] | Watchlist CRUD |
| GET/POST/PUT/DELETE | /holdings[/:symbol] | Holdings CRUD |
| GET/POST/PUT/DELETE | /alerts[/:id] | Alerts CRUD |
| GET | /quote/:symbol | Single quote |
| GET | /quote/batch?symbols=... | Batch quotes |
| GET | /kline/:symbol?interval=&count= | K-line data |
| POST | /recommendations | AI stock recommendations |
| POST | /strategy/analyze | LLM strategy analysis (17 strategies + comprehensive) |
| GET | /strategy/list | Available strategies |
| POST | /signals/record | Record signal scores |
| GET | /signals/:symbol/history | Signal history |
| GET | /ws?token=... | WebSocket (real-time push) |

## Data Flow

```
Free HTTP APIs → eastmoney.Client (poll 5s) → hub.BroadcastQuote → browser WS
                                                                   → alert.Engine.Evaluate
Browser REST → handler → repo → SQLite
Browser WS   → hub (snapshot + real-time quotes)
Strategy API → llm.Chat → DeepSeek → cached (2min/24h)
```

## Key Design Decisions

- **No external API keys required** for quotes/klines
- **DeepSeek LLM** for recommendations + strategy analysis (optional)
- **17 strategy prompts** + comprehensive synthesis (parallel execution with cache)
- **Strategy cache**: 2min during trading hours, 24h otherwise
- **JWT auth** with admin/normal roles, invite-code-based registration
- **SQLite** single-file database, auto-migrated on startup
- Quotes: SH/SZ/HK from Sina; K-lines: SH/SZ from Sina, HK from Sina with Eastmoney fallback
- **Flutter mobile** shares same backend API
