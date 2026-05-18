# Stock Monitor

个人股票盯盘工具，支持 A 股（沪深）、港股的实时行情、K 线技术分析、持仓盈亏跟踪、价格预警和 AI 智能荐股。

## 快速开始

### 1. 编译

```bash
go build -o bin/server ./cmd/server
```

### 2. 配置

编辑 `.env` 文件：

```env
PORT=3000
JWT_SECRET=自定义密钥
ADMIN_PASSWORD=管理员密码
DEEPSEEK_API_KEY=sk-xxx    # 可选，启用 AI 荐股
```

### 3. 启动

```bash
./bin/server
```

### 4. 打开浏览器

访问 `http://localhost:3000`，首次启动会生成管理员账号密码并打印在日志中。

## 功能

### 自选股

- 添加/删除自选股，支持港股 `HK:0700`、沪市 `SH:600519`、深市 `SZ:000001`
- 实时行情：最新价、涨跌幅、今开、最高、最低、成交量
- 数据通过 WebSocket 推送到前端，每 5 秒轮询刷新

### K 线图

- 时间周期：5 分钟、日 K、周 K、月 K
- 叠加均线（周期自适应）及买卖信号标记
- 支持横纵轴缩放、十字光标

### 卖出分析

- 基于日 K 数据，8 项技术指标加权评分（RSI 背离、MACD 死叉、MA 交叉等）
- 满分 13 分，>50% 强烈卖出，25-49% 偏弱，<25% 健康

### 持仓管理

- 记录持股数量、成本价，实时计算盈亏
- 总投入、总市值、总盈亏汇总

### 价格预警

- 涨破/跌破/涨跌幅达阈值时触发通知
- 同一规则 30 分钟内不重复触发

### AI 荐股

- 输入行业关键词，DeepSeek 大模型智能推荐相关股票
- 自动去重（跳过已自选）、四因子排序（相关性 + 可交易 + 动量 + 成交量）
- 结果缓存 30 分钟

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + Gin + SQLite |
| 前端 | 原生 HTML/CSS/JS，无框架 |
| K 线图 | TradingView Lightweight Charts v4 |
| 实时推送 | WebSocket（服务器 ↔ 浏览器） |
| 行情数据 | 免费公开接口 |
| AI | DeepSeek API（OpenAI 兼容） |
| 认证 | JWT |

## 数据存储

所有数据存储在 `data/stock-monitor.db`（SQLite），仅本地保存。

## 推荐配置

```env
PORT=3000
DATA_DIR=data
DEEPSEEK_API_KEY=sk-xxx
DEEPSEEK_MODEL=deepseek-chat
RECOMMEND_LIMIT=8
```
