# Stock Monitor

个人股票盯盘工具，支持 A 股（沪深）、港股的实时行情、K 线技术分析、持仓盈亏跟踪、价格预警、AI 智能荐股和策略分析。提供 Web 前端 + Flutter 移动端。

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
DEEPSEEK_API_KEY=sk-xxx    # 可选，启用 AI 功能
```

### 3. 启动

```bash
./bin/server
```

访问 `http://localhost:3000`，首次启动会生成管理员账号密码并打印在日志中。

## 功能

### 自选股 + 实时行情

- 添加/删除自选股，支持港股 `HK:0700`、沪市 `SH:600519`、深市 `SZ:000001`
- 实时行情：最新价、涨跌幅、今开、最高、最低、成交量
- WebSocket 推送，5 秒轮询刷新

### K 线图

- 时间周期：5 分钟、日 K、周 K、月 K
- 均线周期自适应，买卖信号标记
- 缩放、十字光标，北京时间显示

### 技术分析

- 基于日 K 数据，8 项卖出信号 + 10 项买入信号加权评分
- 按交易所筛选、按信号/名称排序
- 信号持久化到后端，跨设备可查

### AI 策略分析

- 17 种专业策略模板：均线金叉、趋势跟踪、放量突破、缠论、波浪理论等
- **综合分析**：并行运行全部策略，聚合输出多空对比、共识分歧、1-10 评分、操作建议
- 缓存策略：盘中 2 分钟，盘后 24 小时；综合分析缓存命中秒出
- Markdown 渲染：粗体高亮、标题、时间戳转北京时间

### AI 荐股

- 输入行业关键词，DeepSeek 大模型推荐相关股票
- 自动去重、四因子排序（相关性 + 可交易 + 涨跌动量 + 成交量热度）
- 结果缓存 30 分钟

### 持仓管理

- 记录持股数量、成本价，实时计算盈亏
- 总投入、总市值、总盈亏汇总

### 价格预警

- 涨破/跌破/涨跌幅达阈值时触发通知
- 同一规则 30 分钟内不重复触发

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + Gin + SQLite |
| 前端 | 原生 HTML/CSS/JS，无框架 |
| 移动端 | Flutter + Riverpod |
| K 线图 | TradingView Lightweight Charts v4（Web）/ CustomPaint（Flutter） |
| 实时推送 | WebSocket（服务器 ↔ 浏览器） |
| 行情数据 | 免费公开接口 |
| AI | DeepSeek API（OpenAI 兼容） |
| 认证 | JWT |

## 数据存储

所有数据存储在 `data/stock-monitor.db`（SQLite），仅本地保存。

## 推荐配置

```env
PORT=3000
JWT_SECRET=your_random_secret
ADMIN_PASSWORD=your_admin_password
DEEPSEEK_API_KEY=sk-xxx
DEEPSEEK_MODEL=deepseek-chat
RECOMMEND_LIMIT=8
```
