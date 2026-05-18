# Stock Monitor — Project Context

## Overview

Personal stock monitoring tool. Go backend + vanilla JS frontend + SQLite.

## Build & Run

```bash
go build -o bin/server ./cmd/server
./bin/server                     # listens on :3000 by default
go test ./internal/...           # run tests
```

## Architecture

```
cmd/server/main.go          → entry point, wiring
internal/
  config/                   → env var loading (.env)
  db/                       → SQLite setup, schema, migrations
  model/                    → data structs (Quote, WatchlistItem, AlertRule, etc.)
  repo/                     → DB access layer (CRUD per entity)
  handler/                  → Gin HTTP handlers (REST API)
  middleware/                → JWT auth middleware
  eastmoney/                → stock quote + kline data client
  llm/                      → DeepSeek API client (stock recommendations)
  recommend/                → recommendation orchestrator (LLM + scoring)
  alert/                    → price alert evaluation engine
  ws/                       → WebSocket hub (browser push)
web/
  index.html, login.html, admin.html
  js/                       → vanilla JS components (api.js, app.js, kline.js, holdings.js, alerts.js, indicators.js)
  css/                      → styles
```

## API Endpoints

All under `/api`, JWT required except `/auth/*`:

| Method | Path | Description |
|--------|------|-------------|
| POST | /auth/register | Register with invite code |
| POST | /auth/login | Login, returns JWT |
| GET | /watchlist | User's watchlist |
| POST | /watchlist | Add to watchlist |
| DELETE | /watchlist/:symbol | Remove from watchlist |
| GET | /holdings | User's holdings |
| POST | /holdings | Add holding |
| PUT | /holdings/:symbol | Update holding |
| DELETE | /holdings/:symbol | Delete holding |
| GET | /alerts | Alert rules |
| POST | /alerts | Create alert |
| PUT | /alerts/:id | Update alert |
| DELETE | /alerts/:id | Delete alert |
| GET | /quote/:symbol | Single quote |
| GET | /quote/batch?symbols=... | Batch quotes |
| GET | /kline/:symbol?interval=&count= | K-line data |
| POST | /recommendations | AI stock recommendations |
| GET | /ws?token=... | WebSocket (real-time push) |

## Data Flow

```
Free HTTP APIs → eastmoney.Client (poll 5s) → hub.BroadcastQuote → browser WS
                                                                   → alert.Engine.Evaluate
Browser REST → handler → repo → SQLite
Browser WS   → hub (snapshot + real-time quotes)
```

## Key Design Decisions

- **No external API keys required** for quotes/klines — uses free public data
- **DeepSeek LLM** for stock recommendations (optional, needs API key)
- **JWT auth** with admin/normal roles, invite-code-based registration
- **SQLite** single-file database, auto-migrated on startup
- **5-second polling** for quotes (not WebSocket to external source)
- Quotes support SH/SZ/HK; US stocks can be recommended by LLM but have no quote data
- K-lines: SH/SZ from Sina, HK from Yahoo Finance (fallback)
