# LLM-Powered Stock Recommendation Design

## 目标

用 DeepSeek 大模型替代 NewsAPI + 正则表达式提取的股票推荐方式，通过 LLM 智能分析行业关键词并直接推荐相关股票。

## 架构

```
用户输入行业关键词
  → DeepSeek API (Prompt: 行业→推荐股票JSON)
    → 输出: [{symbol, name, reason, market}]
      → 符号标准化 + 行情验证 (SH/SZ/HK 拉实时行情, US 标记无数据)
        → 评分排序
          → 返回前端
```

## 数据源

| 提供商 | URL | 说明 |
|--------|-----|------|
| DeepSeek | `https://api.deepseek.com/v1/chat/completions` | 兼容 OpenAI 格式 |

模型：`deepseek-chat`，备选 `deepseek-reasoner`

## Prompt 策略

- System prompt：定义股票推荐助手角色，限定输出格式为 JSON 数组
- User prompt：传入用户输入的行业关键词
- `response_format: {"type": "json_object"}` 确保结构化输出
- Temperature 0.3 保证结果稳定可复现
- 每个市场最多推荐 5 只，覆盖 HK/SH/SZ/US
- 返回纯 JSON，不要 markdown 代码块

## 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| `+` | `internal/llm/client.go` | DeepSeek HTTP 客户端，发送 Chat Completion 请求 |
| `+` | `internal/llm/prompt.go` | System prompt 和消息构造 |
| `*` | `internal/recommend/recommender.go` | 替换 NewsAPI 流程为 LLM 调用 |
| `*` | `internal/config/config.go` | 新增 `DEEPSEEK_API_KEY`、`DEEPSEEK_MODEL` |
| `*` | `cmd/server/main.go` | 注入 LLM client 到 recommender |
| `-` | `internal/recommend/newsapi.go` | 删除（可选保留作为后备） |
| `-` | `internal/recommend/extractor.go` | 删除（正则提取不再需要） |
| `*` | `internal/recommend/scorer.go` | 调整评分权重，加入 LLM 排序位次 |

## Recommender 接口变更

```go
type Recommender struct {
    llmClient  *llm.Client
    emClient   eastmoney.QuoteClient
    cache      map[string]*cacheEntry
    cacheTTL   time.Duration
    limit      int
    mu         sync.RWMutex
}

func (r *Recommender) Search(industry string) ([]model.Recommendation, error) {
    // 1. 调用 LLM 获取推荐
    candidates, err := r.llmClient.Recommend(industry)
    // 2. 提取符号，拉行情
    quotes := r.batchFetchQuotes(candidates)
    // 3. 评分排序
    recs := Score(candidates, quotes, r.limit)
    // 4. 缓存
    return recs
}
```

## 配置项

```env
DEEPSEEK_API_KEY=sk-xxx      # 必填
DEEPSEEK_MODEL=deepseek-chat # 可选，默认 deepseek-chat
LLM_CACHE_TTL=30             # 缓存分钟数，默认 30
```

## API 响应格式不变

`POST /api/recommendations` 请求/响应格式保持兼容：
- 请求：`{"industry": "芯片半导体"}`
- 响应：`{"recommendations": [{"symbol", "name", "score", "price", "changePercent", "highlights", "rank"}]}`

前端无需任何改动。

## 错误处理

- LLM API 超时（10s）→ 返回错误信息
- LLM 返回非 JSON → 解析失败，返回空结果
- 推荐股票无行情 → 保留在列表中，price=0
- API Key 未配置 → 启动时 warn，推荐接口返回 503
