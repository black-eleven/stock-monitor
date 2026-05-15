# 东方财富 HTTP API 迁移设计

## 目标

将股票行情数据源从 QOS WebSocket API 切换为东方财富免费 HTTP API。

## 架构变化

```
之前:  QOS WebSocket (wss://api.qos.hk/ws)
       ↓ [长连接推送]
       qos.QosClient → OnQuote → Hub.BroadcastQuote / AlertEngine

之后:  东方财富 HTTP (push2.eastmoney.com)
       ↓ [定时轮询 5s]
       eastmoney.Client → PollQuotes → Hub.BroadcastQuote / AlertEngine
                        → FetchQuote / FetchKline (按需)
```

核心变化：WebSocket 实时推送 → HTTP 定时轮询。前端 WebSocket (`/ws`) 保持不变。

## 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| `+` | `internal/eastmoney/client.go` | HTTP 客户端，封装东方财富接口 + QuoteClient 接口 |
| `+` | `internal/eastmoney/cache.go` | 缓存/重试/请求合并，复用现有设计 |
| `+` | `internal/eastmoney/symbol.go` | QOS 格式符号 ↔ 东方财富 secid 映射 |
| `*` | `internal/model/quote.go` | `FromQosQuote` → `FromEMQuote`，数据类型不变 |
| `*` | `internal/config/config.go` | 移除 `QosKey`/`QosWsUrl` 字段 |
| `*` | `internal/handler/quote.go` | `*qos.QosClient` → `eastmoney.QuoteClient` 接口 |
| `*` | `internal/handler/kline.go` | 同上 |
| `*` | `internal/recommend/recommender.go` | `*qos.QosClient` → `eastmoney.QuoteClient` 接口 |
| `*` | `cmd/server/main.go` | 初始化 eastmoney client + 启动轮询 goroutine |
| `-` | `internal/qos/` | 整个包删除 |

## 核心接口

```go
// internal/eastmoney/client.go
type QuoteClient interface {
    FetchQuote(ctx context.Context, code string) (*Quote, error)
    FetchQuoteCached(code string) (*Quote, error)
    FetchHistoryKline(ctx context.Context, code string, kt, count int) ([]json.RawMessage, error)
    FetchHistoryKlineCached(code string, kt, count int) ([]json.RawMessage, error)
    BatchFetchQuotes(codes []string) (map[string]*Quote, error)
}
```

## 东方财富 API 映射

### 批量行情

```
GET http://push2.eastmoney.com/api/qt/stock/get
  ?secid=1.600519,0.000001,116.00700
  &fields=f43,f44,f45,f46,f47,f48,f50,f51,f52,f57,f58,f60,f116,f117,f162,f167,f168,f169,f170,f171
```

字段对应：f43=最新价, f44=最高, f45=最低, f46=开盘, f47=成交量, f48=成交额, f50=涨跌幅, f57=代码, f58=名称, f60=昨收, f116=总市值, f117=流通市值, f162=市盈率, f167=换手率, f168=量比, f169=振幅, f170=涨速, f171=5分钟涨跌

### K线数据

```
GET http://push2his.eastmoney.com/api/qt/stock/kline/get
  ?secid=1.600519
  &klt=101
  &fqt=1
  &beg=20250101
  &end=20250515
```

### 符号格式映射

```
HK:00700  → 116.00700    (港股，市场代码 116)
SH:600519 → 1.600519     (沪市，市场代码 1)
SZ:000001 → 0.000001     (深市，市场代码 0)
```

### K线周期映射

```
1m→1, 5m→5, 15m→15, 30m→30, 1h→60, 2h→120, 4h→240, 1d→101, 1w→102, 1M→103
```

## 实时推送替代

定时轮询方案：
- Goroutine 每 5 秒调用一次批量行情接口
- 维护订阅列表（用户自选股）
- 比较价格变化，有变化时通过 `Hub.BroadcastQuote` 推送
- 无变化时不推送，减少前端负担

## 缓存层

沿用现有设计（`internal/eastmoney/cache.go`）：
- 行情缓存 TTL：30 秒
- 分钟 K 线 TTL：30 秒，日 K 及以上 TTL：5 分钟
- 并发请求合并：相同 key 的并发请求共享一次上游调用
- 重试：最多 3 次，指数退避（1s, 2s, 4s）

## 依赖变化

- `github.com/gorilla/websocket` — 保留（前端 WebSocket 仍需要）
- QOS_KEY 环境变量 — 不再需要
- go.mod — 移除 mongo-driver 等未使用的间接依赖
