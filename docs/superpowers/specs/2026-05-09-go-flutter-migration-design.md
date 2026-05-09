# Stock Monitor — Go 后端 + Flutter 移动端迁移设计

## 1. 概述

### 目标

将现有 Node.js 单机股票监控工具扩展为多端架构：

- **保留** Web 前端（`public/` 下的 HTML/JS/CSS）
- **新增** Go 后端，替代 Node.js Express 服务端
- **新增** Flutter 移动端（iOS + Android）
- **新增** SQLite 数据库，替代 JSON 文件存储

### 非目标

- 不引入用户认证系统（维持 localhost 单用户场景）
- 不改动 QOS 行情源接入方式
- 不改动前端 UI 设计

### 约束

| 约束 | 说明 |
|------|------|
| API 兼容 | HTTP API 路径、请求/响应格式尽量和现有 Node.js 版一致 |
| WS 协议兼容 | Web 和 Flutter 使用相同的 WS 消息格式 |
| 单仓库 | 所有代码在同一 Git 仓库中 |
| 零额外部署 | SQLite 嵌入式数据库，无需独立数据库服务 |

---

## 2. 总体架构

```
                         ┌──────────────────────────────────────────┐
                         │              Go 后端 (:3000)               │
                         │                                          │
  QOS 行情源              │  ┌──────────┐  ┌─────────┐  ┌─────────┐ │
 (WebSocket) ←──────────→│  │ QOS Client│  │ WS Hub  │  │ Gin HTTP│ │
                         │  │(goroutine)│  │(gorilla)│  │  Server  │ │
                         │  └────┬─────┘  └────┬────┘  └────┬────┘ │
                         │       │             │            │       │
                         │       └──────┬──────┘            │       │
                         │              │                   │       │
                         │     ┌────────┴────────┐          │       │
                         │     │   Alert Engine  │   ┌──────┴─────┐ │
                         │     └────────┬────────┘   │  Handlers  │ │
                         │              │            │  (repo)    │ │
                         │              │            └──────┬─────┘ │
                         │              │                   │       │
                         │     ┌────────┴───────────────────┴─────┐ │
                         │     │            SQLite                │ │
                         │     └──────────────────────────────────┘ │
                         └──────┬───────────────┬──────────────────┘
                                │ HTTP + WS     │ HTTP + WS
                         ┌──────┴──────┐ ┌──────┴──────┐
                         │  Web 前端    │ │ Flutter App │
                         │ public/ 目录  │ │   mobile/   │
                         │ (保留原版)    │ │ iOS + 安卓   │
                         └─────────────┘ └─────────────┘
```

---

## 3. Go 后端

### 3.1 依赖

```
github.com/gin-gonic/gin          # HTTP 框架 + 路由
github.com/gorilla/websocket       # WebSocket (QOS 客户端 + 浏览器服务端)
github.com/mattn/go-sqlite3        # SQLite 驱动
github.com/joho/godotenv           # .env 加载
```

### 3.2 目录结构

```
stock-monitor/
├── cmd/server/main.go             # 入口
├── internal/
│   ├── config/config.go           # 环境变量
│   ├── db/
│   │   ├── sqlite.go              # 连接 + 迁移
│   │   └── migrations/001_init.sql
│   ├── model/                     # 数据结构
│   │   ├── watchlist.go
│   │   ├── alert.go
│   │   ├── holding.go
│   │   └── quote.go
│   ├── repo/                      # 数据访问层
│   │   ├── watchlist.go
│   │   ├── alert.go
│   │   └── holding.go
│   ├── qos/                       # QOS WebSocket 客户端
│   │   ├── client.go              # 连接、重连、心跳
│   │   └── kline.go               # fetchHistoryKline
│   ├── ws/                        # 浏览器/App WebSocket
│   │   ├── hub.go                 # 广播管理
│   │   └── client.go              # 单连接读写
│   ├── alert/engine.go            # 告警评估引擎
│   └── handler/                   # HTTP handlers
│       ├── watchlist.go
│       ├── alert.go
│       ├── holding.go
│       ├── quote.go
│       └── kline.go
├── web/                           # 现有 public/ 迁移 → Go embed
│   ├── index.html
│   ├── css/style.css
│   └── js/*.js
├── mobile/                        # Flutter 项目
│   └── stock_monitor/
├── data/                          # SQLite 数据库文件目录
├── go.mod
├── go.sum
└── Makefile
```

### 3.3 SQLite 数据模型

```sql
CREATE TABLE watchlist (
    symbol   TEXT PRIMARY KEY,   -- "HK:700", "SH:600519"
    name     TEXT NOT NULL,      -- "腾讯控股"
    added_at TEXT NOT NULL       -- ISO 8601
);

CREATE TABLE holdings (
    symbol   TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    shares   REAL NOT NULL,
    avg_cost REAL NOT NULL,
    buy_date TEXT               -- "YYYY-MM-DD"
);

CREATE TABLE alerts (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol             TEXT NOT NULL,
    type               TEXT NOT NULL,     -- "above" / "below" / "change_pct"
    value              REAL NOT NULL,
    enabled            INTEGER DEFAULT 1,
    created_at         TEXT NOT NULL,
    last_triggered_at  TEXT
);

CREATE TABLE alert_logs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id     INTEGER NOT NULL,
    symbol       TEXT NOT NULL,
    price        REAL NOT NULL,
    message      TEXT NOT NULL,
    triggered_at TEXT NOT NULL
);
```

### 3.4 API 路由

完全兼容现有 API：

```
# 自选股
GET    /api/watchlist
POST   /api/watchlist          body: { symbol, name }
DELETE /api/watchlist/:symbol

# 提醒
GET    /api/alerts
POST   /api/alerts             body: { symbol, type, value }
PUT    /api/alerts/:id         body: { enabled?, value? }
DELETE /api/alerts/:id

# 持仓
GET    /api/holdings
POST   /api/holdings           body: { symbol, name, shares, avgCost, buyDate? }
PUT    /api/holdings/:symbol   body: { shares?, avgCost?, buyDate? }
DELETE /api/holdings/:symbol

# 行情
GET    /api/quote/batch?symbols=HK:700,HK:1810
GET    /api/quote/:symbol      # symbol 格式: ^(HK|SH|SZ|US):[A-Z0-9]{1,10}$

# K线
GET    /api/kline/:symbol?interval=1d&count=100

# WebSocket
GET    /ws                     # 实时行情推送 + 告警推送
```

### 3.5 QOS Client 设计

从 Node.js 版 `qos-client.js` 翻译为 Go，关键改进：

| Node.js (旧) | Go (新) |
|---|---|
| `Date.now()` + seq 做 reqid | `atomic.AddInt64` 自增计数器 |
| `on('message')` 临时处理器 | `sync.Map` 做 `reqid → chan response` 映射 |
| Promise + 闭包 reject | `select` + `case <-time.After(10s)` 超时 |
| 单 goroutine (Node 事件循环) | 读 goroutine + 写 goroutine + 重连控制 |

结构：

```go
type QosClient struct {
    conn      *websocket.Conn
    sendCh    chan []byte          // 写 goroutine
    pending   sync.Map             // reqid → chan response
    reqSeq    int64                // 原子自增
    onQuote   func(Quote)          // 行情回调
    onKline   func(Kline)          // K线回调
    // ... 重连、心跳、订阅管理
}
```

### 3.6 WebSocket Hub

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    quotes     sync.Map           // code → Quote (snapshot 缓存)
}

// 单 goroutine Run()，select 多路复用：
//   register   → 加入 clients
//   unregister → 移除 + 清理
//   broadcast  → 遍历 clients 发送
```

- 新连接：先发送 snapshot（所有缓存行情），再增量推送
- 消息格式：`{ "type": "quote", "data": {...} }` / `{ "type": "alert", "data": {...} }` / `{ "type": "snapshot", "data": [...] }`

### 3.7 Alert Engine

和现有 Node.js 版逻辑完全一致：

1. `OnQuote(quote)` → 从 DB 查询该 symbol 的启用的告警规则（每次 evaluate 重新查询，和 Node.js 版每次 `storage.read('alerts')` 行为一致）
2. 逐条判断：`above`（`price >= value`）、`below`（`price <= value`）、`change_pct`（`abs((price-yp)/yp*100) >= abs(value)`）
3. 30 分钟去重：检查 `last_triggered_at`，未超过阈值则跳过
4. 触发：更新 `last_triggered_at`、写入 `alert_logs`、`Hub.BroadcastAlert()`

### 3.8 Web 前端适配

现有 `public/` 只需改动一处：`api.js` 中 WebSocket 连接地址：

```diff
- this.ws = new WebSocket(`${protocol}//${location.host}`);
+ this.ws = new WebSocket(`${protocol}//${location.host}/ws`);
```

消息格式（`{ type, data }`）不变。HTML/CSS/JS 代码完全保留。

其他注意事项：
- Gin 路由 `/api/*` 精确匹配优先于 `StaticFS("/", ...)`，不会冲突
- 数据迁移脚本 `cmd/migrate/main.go` 作为独立二进制运行，不在 `main.go` 中

### 3.9 启动流程

```go
func main() {
    cfg  := config.Load()
    db   := db.Open(cfg.DataDir)
    hub  := ws.NewHub()
    go hub.Run()

    qos    := qos.NewClient(cfg.QosKey)
    engine := alert.NewEngine(db, hub)

    qos.OnQuote = func(q model.Quote) {
        hub.BroadcastQuote(q)
        engine.Evaluate(q)
    }

    r := gin.Default()
    r.StaticFS("/", http.FS(webFS))  // go:embed web/
    handler.RegisterAll(r.Group("/api"), db, qos)
    r.GET("/ws", hub.ServeWS)

    go qos.Connect()
    r.Run(":" + cfg.Port)
}
```

---

## 4. Flutter 移动端

### 4.1 技术栈

| 维度 | 选择 | 理由 |
|------|------|------|
| 状态管理 | **Riverpod** | 类型安全、无全局单例、适合中等复杂度 |
| HTTP 客户端 | **dio** | 拦截器链、超时重试、拦截器易于调试 |
| WebSocket | **web_socket_channel** | Dart 官方维护 |
| K线图 | **fl_chart** + 自定义指标 | 蜡烛图原生支持、LineChart 叠加 MA、交互缩放 |
| 本地缓存 | **shared_preferences** | 存设置（后端地址、默认周期） |
| 路由 | **go_router** | 声明式路由、BottomNav 支持好 |
| 架构 | 精简 Clean Architecture | data / domain / presentation 三层 |

### 4.2 目录结构

```
mobile/stock_monitor/
├── lib/
│   ├── main.dart                    # ProviderScope + App
│   ├── app.dart                     # MaterialApp.router + 暗色主题
│   │
│   ├── core/
│   │   ├── config.dart              # 后端地址、API 前缀
│   │   ├── theme.dart               # 暗色主题（#0d1117 背景，绿涨红跌）
│   │   └── utils.dart               # formatPrice, shortCode, calcChangePct
│   │
│   ├── data/
│   │   ├── api/
│   │   │   ├── api_client.dart      # dio 实例（baseUrl, 超时, 拦截器）
│   │   │   ├── watchlist_api.dart
│   │   │   ├── alert_api.dart
│   │   │   ├── holding_api.dart
│   │   │   └── quote_api.dart
│   │   ├── ws/
│   │   │   └── ws_client.dart       # WebSocket 连接 + 自动重连
│   │   └── repo/
│   │       ├── watchlist_repo.dart
│   │       ├── alert_repo.dart
│   │       ├── holding_repo.dart
│   │       └── quote_repo.dart
│   │
│   ├── domain/
│   │   └── model/
│   │       ├── stock.dart           # WatchlistItem, Quote
│   │       ├── alert.dart           # AlertRule, AlertLog
│   │       ├── holding.dart         # Holding
│   │       └── kline.dart           # KlineBar
│   │
│   └── presentation/
│       ├── providers/
│       │   ├── watchlist_provider.dart
│       │   ├── quote_provider.dart
│       │   ├── holding_provider.dart
│       │   ├── alert_provider.dart
│       │   ├── kline_provider.dart
│       │   └── analysis_provider.dart
│       ├── screens/
│       │   ├── watchlist_screen.dart
│       │   ├── kline_screen.dart
│       │   ├── holdings_screen.dart
│       │   ├── alerts_screen.dart
│       │   └── analysis_screen.dart
│       └── widgets/
│           ├── stock_card.dart
│           ├── quote_badge.dart
│           ├── kline_chart.dart      # fl_chart 封装 + MA/RSI/MACD
│           ├── holding_row.dart
│           └── alert_item.dart
├── pubspec.yaml
└── analysis_options.yaml
```

### 4.3 页面设计

5 Tab 布局（BottomNavigationBar）：

**自选股 Tab**
- 列表：每行显示股票名、代码、最新价（颜色涨跌）、涨跌幅
- 点击展开详情：今开/最高/最低/昨收/成交量/成交额
- 右上角 + 按钮添加自选（symbol + name 输入）
- 长按或滑动删除

**K线 Tab**
- 顶部 Symbol 下拉选择器
- 周期切换行：分时/5M/15M/30M/1H/2H/4H/日K/周K/月K
- 主图：蜡烛图 + MA5(黄)/MA20(蓝)/MA60(紫) 叠加
- 副图1：RSI(14)，超买线(70)/超卖线(30)
- 副图2：MACD 柱状图 + DIF/DEA 线
- 支持双指缩放、拖拽、十字光标
- 顶部显示 OHLC 四价
- 卖出信号标记（红色向下箭头）

**持仓 Tab**
- 汇总栏：总成本 / 总市值 / 总盈亏 / 总盈亏率（颜色标注）
- 列表：股票名、持仓量、成本价、现价、盈亏额、盈亏率
- 右上角 + 添加持仓（表单：symbol, shares, avgCost, buyDate）
- 滑动编辑/删除

**提醒 Tab**
- 上半部分：提醒规则列表，每条显示 symbol、类型（涨破/跌破/涨跌幅）、阈值、启用开关、删除按钮
- 下半部分：触发日志（最近 50 条），时间 + 内容
- + 按钮弹 BottomSheet 添加规则

**分析 Tab**
- 全选股卖出信号扫描结果
- 顶部：平均卖出分 + 市场情绪标签（绿/黄/红）
- 列表卡片：股票名、现价、卖出评分、触发信号数、颜色编码
- 点击卡片进入详情：8 维度信号逐条展示（信号名、状态指示灯、数值）

### 4.4 实时行情数据流

```
Go WS Server (/ws)
      │
      ├── { "type": "snapshot", "data": [...] }     ← 连接时一次性下发
      ├── { "type": "quote",    "data": {...} }     ← 增量推送
      └── { "type": "alert",    "data": {...} }     ← 告警推送
      │
      ▼
WsClient (data/ws/)
      │
      ├──→ QuoteProvider    ──→ WatchlistScreen（价格更新）
      │                     ──→ HoldingsScreen（盈亏更新）
      │
      └──→ AlertProvider    ──→ AlertsScreen（日志+Toast）
```

### 4.5 K线图实现方案

使用 `fl_chart` 的 `CandlestickChart` + `LineChart` 叠加：

- 主图区域 60%：`CandlestickChart` 渲染 K 线 + 3 条 `LineChart` 叠加（MA5/MA20/MA60）
- RSI 区域 20%：`LineChart` 渲染 RSI 曲线 + 水平参考线（70/30）
- MACD 区域 20%：`BarChart` 渲染红绿柱 + `LineChart` 渲染 DIF/DEA
- 手势：`GestureDetector` + `ScaleGestureRecognizer` 实现缩放/拖拽
- 十字光标：`LongPressMoveUpdate` 显示 OHLC tooltip

指标计算：将 `public/js/indicators.js` 中的 `calcMA`、`calcRSI`、`calcMACD`、`evaluateSignals` 翻译为 Dart 纯函数。

---

## 5. 迁移策略

### 5.1 数据迁移

Node.js `data/*.json` → Go SQLite：

- 提供一次性迁移脚本 `cmd/migrate/main.go`
- 读取 JSON 文件 → 插入 SQLite 表
- 迁移后 JSON 文件保留为备份

### 5.2 实施顺序（推荐 4 阶段）

```
Phase 1: Go 后端核心
  ├── 项目骨架、配置、SQLite、migration
  ├── repo 层（watchlist, alert, holding CRUD）
  ├── Gin routes + handlers
  └── 单元测试 → 可独立运行

Phase 2: QOS 接入 + WebSocket
  ├── qos/client.go（连接、fetchHistoryKline、fetchQuote）
  ├── ws/hub.go + ws/client.go（浏览器 WS）
  ├── alert/engine.go（告警评估）
  └── 集成测试 → 替代 Node.js

Phase 3: Web 前端适配
  ├── 移动 public/ → web/，go:embed 嵌入
  ├── 验证所有 API 兼容
  └── 端到端测试

Phase 4: Flutter 移动端
  ├── 项目骨架、主题、路由
  ├── 5 个页面逐步开发
  ├── WebSocket 集成
  └── iOS/Android 真机测试
```

### 5.3 旧代码处理

- `server/` 目录重命名为 `server-node-legacy/`，README 标注已废弃
- `public/` 移动到 `web/`，由 Go embed 静态文件服务
- `data/*.json` 迁移后保留为备份
- `package.json` 移除 `express`、`ws` 依赖，保留仅作文档参考

---

## 6. 测试策略

| 层级 | Go 后端 | Flutter |
|------|---------|---------|
| 单元测试 | repo 层（SQLite in-memory）、handler 层（httptest）、qos mock | indicator 计算函数、repo 层（mock API） |
| 集成测试 | QOS mock server + WS client | 无（依赖后端） |
| E2E | Playwright（Web） | integration_test（Flutter） |

---

## 7. 已知风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| QOS API 无官方文档，协议靠逆向 | 保留现有 Node.js 代码参考，逐字段验证 |
| fl_chart 不支持副图 | 主图和副图用独立 Chart 组件，GestureDetector 同步缩放 |
| 单仓库内 Go + Flutter 构建协调 | Makefile 统一入口：`make run` / `make build-mobile` |
| SQLite 无并发写入 | Go 后端是单进程，所有写入串行化在 HTTP handler；未来若需扩展再引入连接池 |

---

## 8. 项目结构总览（最终态）

```
stock-monitor/
├── cmd/
│   ├── server/main.go              # Go 后端入口
│   └── migrate/main.go             # JSON → SQLite 迁移工具
├── internal/                       # Go 内部包
│   ├── config/
│   ├── db/
│   ├── model/
│   ├── repo/
│   ├── qos/
│   ├── ws/
│   ├── alert/
│   └── handler/
├── web/                            # Web 前端（go:embed）
│   ├── index.html
│   ├── css/
│   └── js/
├── mobile/                         # Flutter 移动端
│   └── stock_monitor/
├── server-node-legacy/             # 原 Node.js 代码（归档）
├── data/                           # SQLite 数据库
├── docs/                           # 文档
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .env
```
