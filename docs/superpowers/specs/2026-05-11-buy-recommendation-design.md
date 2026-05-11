# 买入推荐功能 — 设计文档

## 1. 概述

在现有卖出分析功能基础上，新增对称的买入推荐功能。采用混合方案：反转卖出 8 个信号 + 新增 2 个买入专属信号，满分 17 分。

所有计算保持客户端执行（JS / Dart），Go 后端不做改动。

---

## 2. 信号设计（10 个信号，满分 17）

### 2.1 反转信号（8 个，权重 13.0）

| # | 信号 | 权重 | 触发条件 | 状态分级 |
|---|------|------|----------|----------|
| 1 | MA5 金叉 MA20 | 2.0 | 上周期 MA5 ≤ MA20，当前 MA5 > MA20 | 触发=推荐，无细分 |
| 2 | 收盘 > MA20 | 1.0 | 最新收盘价 > MA20 | 触发=warn |
| 3 | 收盘 > MA60 | 0.5 | 最新收盘价 > MA60 | 触发=warn |
| 4 | RSI 超卖 | 1.0 | RSI(14) < 30，RSI(14) < 20 升级为 danger | warn/danger |
| 5 | RSI 底背离 | 2.5 | 价格创新低，RSI 未创新低（20 周期窗口） | 触发=danger |
| 6 | MACD 金叉 | 2.0 | 上周期 DIF ≤ DEA，当前 DIF > DEA | 触发=danger |
| 7 | MACD 底背离 | 2.5 | 价格创新低，DIF 未创新低（20 周期窗口） | 触发=danger |
| 8 | 成交量金叉 | 1.5 | 上涨日平均成交量 > 下跌日 × 1.2（5 日窗口） | 触发=warn |

### 2.2 新增信号（2 个，权重 4.0）

| # | 信号 | 权重 | 触发条件 | 状态分级 |
|---|------|------|----------|----------|
| 9 | 放量突破 | 2.0 | 当日成交量 > 5日均量 × 1.5，且收盘 > MA20 | 触发=danger |
| 10 | 多头排列 | 2.0 | MA5 > MA20 > MA60 三线顺向 | 触发=danger |

---

## 3. 评分与阈值

**公式：**
```
买入指数 = floor(score / 17.0 × 100) %
```

**阈值：**

| 买入指数 | 颜色 | 含义 |
|----------|------|------|
| ≥ 50% | 绿色 | 强烈推荐，多个买入信号共振 |
| 25–49% | 黄色 | 值得关注，部分信号触发 |
| < 25% | 灰色 | 暂无明确买入信号 |

和卖出分析保持一致的 25%/50% 分界，颜色语义相反（卖出红=卖出，买入绿=买入）。

---

## 4. 实现范围

### 4.1 JavaScript 前端

**indicators.js：**
- 新增 `detectBullishDivergence(bars, indicatorData)` — 底部背离检测
- 新增 `evaluateBuySignals(bars, ma5, ma20, ma60, rsi, macd)` — 买入信号评估
- 返回结构同 `evaluateSignals`：`{ score, maxScore: 17, count, total: 10, signals: [...], summary }`

**analysis.js：**
- 在 `AnalysisComponent` 中添加买入/卖出切换按钮
- 买入视图复用卖出视图的结构（平均分栏 + 卡片列表 + 信号详情弹窗）
- 默认选中卖出分析（保持向后兼容）

### 4.2 Flutter 移动端

**indicators.dart：**
- 先补全卖出分析缺失的信号（成交量死叉、MACD 背离）
- 新增 `detectBullishDivergence`、`evaluateBuySignals`

**analysis_screen.dart：**
- 顶部添加 `TabBar`，两个标签页：「买入推荐」「卖出分析」
- 买入 Tab 组件结构和卖出 Tab 一致

### 4.3 Go 后端

不修改。继续提供 `GET /api/kline/:symbol` 原始 K 线数据。

---

## 5. 不做的

- 不在 Go 后端实现分析逻辑
- 不引入新的技术指标（布林带、筹码集中度等）
- 不接入外部推荐/AI 模型
- 不新增 API 路由

---

## 6. 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `web/js/indicators.js` | 修改 | 新增买入信号评估函数 |
| `public/js/indicators.js` | 修改 | 同步 web/ |
| `web/js/analysis.js` | 修改 | 买入/卖出切换 UI |
| `public/js/analysis.js` | 修改 | 同步 web/ |
| `mobile/stock_monitor/lib/domain/indicators.dart` | 修改 | 补全卖出信号 + 新增买入函数 |
| `mobile/stock_monitor/lib/presentation/screens/analysis_screen.dart` | 修改 | TabBar 切换买入/卖出 |
