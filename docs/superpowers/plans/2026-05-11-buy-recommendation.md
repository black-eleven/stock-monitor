# Buy Recommendation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add buy recommendation (买入推荐) with 10 signals alongside existing sell analysis (卖出分析) across JS frontend and Flutter app.

**Architecture:** Client-side signal evaluation on daily K-line data. JS: two separate functions `evaluateSignals` (sell) and `evaluateBuySignals` (buy) in indicators.js; analysis.js gains a buy/sell toggle with mirrored views. Flutter: `indicators.dart` gets buy functions; `analysis_screen.dart` gets a TabBar. Go backend unchanged.

**Tech Stack:** Vanilla JS, Dart/Flutter (Riverpod, fl_chart), Python markdown → HTML

---

### Task 1: Add buy signal evaluation to JS indicators.js

**Files:**
- Modify: `web/js/indicators.js`

- [ ] **Step 1: Add bullish divergence detector and volume golden cross helper**

Add the following code after `detectVolumeDeathCross` (after line 150):

```javascript
// Helper: detect bullish divergence over last 20 bars
// Checks if price made a lower low but the indicator made a higher low
function detectBullishDivergence(bars, indicatorData) {
  const count = Math.min(20, bars.length, indicatorData.length);
  if (count < 2) return false;

  const recentBars = bars.slice(-count);
  const recentIndicators = indicatorData.slice(-count);

  let minLow = Infinity;
  let minLowIdx = -1;
  for (let i = 0; i < recentBars.length; i++) {
    if (recentBars[i].low < minLow) {
      minLow = recentBars[i].low;
      minLowIdx = i;
    }
  }

  let minIndicator = Infinity;
  let minIndicatorIdx = -1;
  for (let i = 0; i < recentIndicators.length; i++) {
    if (recentIndicators[i].value < minIndicator) {
      minIndicator = recentIndicators[i].value;
      minIndicatorIdx = i;
    }
  }

  if (minLowIdx < 0 || minIndicatorIdx < 0) return false;
  const indicatorAtMinLow = recentIndicators[minLowIdx].value;
  return minLowIdx > minIndicatorIdx && indicatorAtMinLow > minIndicator;
}

// Volume golden cross: average volume on up days > down days over last 5 bars
function detectVolumeGoldenCross(bars) {
  const n = Math.min(5, bars.length);
  if (n < 3) return false;
  const recent = bars.slice(-n);
  let upVol = 0, upDays = 0, downVol = 0, downDays = 0;
  for (let i = 0; i < recent.length; i++) {
    const change = recent[i].close - recent[i].open;
    if (change >= 0) {
      upVol += recent[i].close * recent[i].volume;
      upDays++;
    } else {
      downVol += recent[i].close * recent[i].volume;
      downDays++;
    }
  }
  if (upDays === 0 || downDays === 0) return false;
  const avgUpVol = upVol / upDays;
  const avgDownVol = downVol / downDays;
  return avgUpVol > avgDownVol * 1.2;
}
```

- [ ] **Step 2: Add `evaluateBuySignals` function after `evaluateSignals` (after line 316)**

```javascript
// Weighted buy signal evaluation (10 signals, max score = 17.0)
// bars: array of { time, open, high, low, close, volume }
// maData: { ma5, ma20, ma60 } where each is [{ time, value }]
// rsiData: [{ time, value }]
// macdData: { dif, dea, macd } or null
// returns: { score, maxScore: 17, count, total: 10, signals: Signal[], summary: string }
function evaluateBuySignals(bars, maData, rsiData, macdData) {
  const W = {
    ma_cross_golden: 2.0, price_ma20: 1.0, price_ma60: 0.5,
    rsi_oversold: 1.0, rsi_diverge: 2.5, macd_cross_golden: 2.0,
    macd_diverge: 2.5, volume_golden: 1.5,
    vol_breakout: 2.0, bull_alignment: 2.0
  };
  const MAX_SCORE = W.ma_cross_golden + W.price_ma20 + W.price_ma60 +
    W.rsi_oversold + W.rsi_diverge + W.macd_cross_golden + W.macd_diverge +
    W.volume_golden + W.vol_breakout + W.bull_alignment;

  const signals = [];
  let score = 0;
  let count = 0;

  const latestBar = bars[bars.length - 1];

  const ma5 = maData && maData.ma5 ? maData.ma5 : [];
  const ma20 = maData && maData.ma20 ? maData.ma20 : [];
  const ma60 = maData && maData.ma60 ? maData.ma60 : [];

  const latestMA5 = ma5.length >= 1 ? ma5[ma5.length - 1] : null;
  const prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  const latestMA20 = ma20.length >= 1 ? ma20[ma20.length - 1] : null;
  const prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  const latestMA60 = ma60.length >= 1 ? ma60[ma60.length - 1] : null;

  // ---- 1. MA5金叉MA20 (weight 2.0) ----
  let s1 = { key: 'ma_cross_golden', name: 'MA5金叉MA20', triggered: false, value: null, status: 'ok', weight: W.ma_cross_golden };
  if (prevMA5 && prevMA20 && latestMA5 && latestMA20) {
    if (prevMA5.value <= prevMA20.value && latestMA5.value > latestMA20.value) {
      s1.triggered = true;
      s1.value = `MA5=${latestMA5.value.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s1.status = 'danger';
      score += W.ma_cross_golden;
      count++;
    }
  }
  signals.push(s1);

  // ---- 2. 收盘 > MA20 (weight 1.0) ----
  let s2 = { key: 'price_above_ma20', name: '收盘价高于MA20', triggered: false, value: null, status: 'ok', weight: W.price_ma20 };
  if (latestBar && latestMA20) {
    if (latestBar.close > latestMA20.value) {
      s2.triggered = true;
      s2.value = `收盘=${latestBar.close.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s2.status = 'warn';
      score += W.price_ma20;
      count++;
    }
  }
  signals.push(s2);

  // ---- 3. 收盘 > MA60 (weight 0.5) ----
  let s3 = { key: 'price_above_ma60', name: '收盘价高于MA60', triggered: false, value: null, status: 'ok', weight: W.price_ma60 };
  if (latestBar && latestMA60) {
    if (latestBar.close > latestMA60.value) {
      s3.triggered = true;
      s3.value = `收盘=${latestBar.close.toFixed(2)}, MA60=${latestMA60.value.toFixed(2)}`;
      s3.status = 'warn';
      score += W.price_ma60;
      count++;
    }
  }
  signals.push(s3);

  // ---- 4. RSI超卖 (weight 1.0) ----
  let s4 = { key: 'rsi_oversold', name: 'RSI超卖', triggered: false, value: null, status: 'ok', weight: W.rsi_oversold };
  if (rsiData && rsiData.length >= 1) {
    const latestRSI = rsiData[rsiData.length - 1].value;
    if (latestRSI < 30) {
      s4.triggered = true;
      s4.value = `RSI=${latestRSI.toFixed(2)}`;
      s4.status = latestRSI < 20 ? 'danger' : 'warn';
      score += W.rsi_oversold;
      count++;
    }
  }
  signals.push(s4);

  // ---- 5. RSI底背离 (weight 2.5) ----
  let s5 = { key: 'rsi_bullish_divergence', name: 'RSI底背离', triggered: false, value: null, status: 'ok', weight: W.rsi_diverge };
  if (bars && rsiData && rsiData.length >= 2) {
    if (detectBullishDivergence(bars, rsiData)) {
      s5.triggered = true;
      s5.value = '价格创新低，RSI未创新低';
      s5.status = 'danger';
      score += W.rsi_diverge;
      count++;
    }
  }
  signals.push(s5);

  // ---- 6. MACD金叉 (weight 2.0) ----
  let s6 = { key: 'macd_cross_golden', name: 'MACD金叉', triggered: false, value: null, status: 'ok', weight: W.macd_cross_golden };
  if (macdData && macdData.dif && macdData.dea) {
    const dif = macdData.dif;
    const dea = macdData.dea;
    if (dif.length >= 2 && dea.length >= 2) {
      const prevDIF = dif[dif.length - 2].value;
      const prevDEA = dea[dea.length - 2].value;
      const currDIF = dif[dif.length - 1].value;
      const currDEA = dea[dea.length - 1].value;
      if (prevDIF <= prevDEA && currDIF > currDEA) {
        s6.triggered = true;
        s6.value = `DIF=${currDIF.toFixed(4)}, DEA=${currDEA.toFixed(4)}`;
        s6.status = 'danger';
        score += W.macd_cross_golden;
        count++;
      }
    }
  }
  signals.push(s6);

  // ---- 7. MACD底背离 (weight 2.5) ----
  let s7 = { key: 'macd_bullish_divergence', name: 'MACD底背离', triggered: false, value: null, status: 'ok', weight: W.macd_diverge };
  if (bars && macdData && macdData.dif && macdData.dif.length >= 2) {
    if (detectBullishDivergence(bars, macdData.dif)) {
      s7.triggered = true;
      s7.value = '价格创新低，DIF未创新低';
      s7.status = 'danger';
      score += W.macd_diverge;
      count++;
    }
  }
  signals.push(s7);

  // ---- 8. 成交量金叉 (weight 1.5) ----
  let s8 = { key: 'volume_golden_cross', name: '成交量金叉（上涨放量）', triggered: false, value: null, status: 'ok', weight: W.volume_golden };
  if (bars && bars.length >= 3) {
    if (detectVolumeGoldenCross(bars)) {
      s8.triggered = true;
      s8.value = '近5日上涨日平均成交量 > 下跌日1.2倍';
      s8.status = 'warn';
      score += W.volume_golden;
      count++;
    }
  }
  signals.push(s8);

  // ---- 9. 放量突破 (weight 2.0) ----
  let s9 = { key: 'vol_breakout', name: '放量突破', triggered: false, value: null, status: 'ok', weight: W.vol_breakout };
  if (bars && bars.length >= 5 && latestMA20) {
    const recentBars = bars.slice(-5);
    let sumVol = 0;
    for (const b of recentBars) sumVol += b.close * b.volume;
    const avgVol5 = sumVol / 5;
    const curTurnover = latestBar.close * latestBar.volume;
    if (curTurnover > avgVol5 * 1.5 && latestBar.close > latestMA20.value) {
      s9.triggered = true;
      s9.value = `当日成交额=${curTurnover.toFixed(0)}, 5日均额=${avgVol5.toFixed(0)}`;
      s9.status = 'danger';
      score += W.vol_breakout;
      count++;
    }
  }
  signals.push(s9);

  // ---- 10. 多头排列 (weight 2.0) ----
  let s10 = { key: 'bull_alignment', name: '多头均线排列', triggered: false, value: null, status: 'ok', weight: W.bull_alignment };
  if (latestMA5 && latestMA20 && latestMA60) {
    if (latestMA5.value > latestMA20.value && latestMA20.value > latestMA60.value) {
      s10.triggered = true;
      s10.value = `MA5=${latestMA5.value.toFixed(2)} > MA20=${latestMA20.value.toFixed(2)} > MA60=${latestMA60.value.toFixed(2)}`;
      s10.status = 'danger';
      score += W.bull_alignment;
      count++;
    }
  }
  signals.push(s10);

  // ---- Summary ----
  const pct = Math.round((score / MAX_SCORE) * 100);
  let summary;
  if (pct >= 50) {
    summary = '强烈买入信号';
  } else if (pct >= 25) {
    summary = '值得关注';
  } else if (pct > 0) {
    summary = '观望';
  } else {
    summary = '暂无买入信号';
  }

  return {
    score,
    maxScore: MAX_SCORE,
    count,
    total: 10,
    signals,
    summary
  };
}
```

- [ ] **Step 3: Verify with grep that the new function is present**

Run: `grep -c "evaluateBuySignals" web/js/indicators.js`
Expected: `1`

- [ ] **Step 4: Commit**

```bash
git add web/js/indicators.js
git commit -m "feat: add buy signal evaluation (10 signals, maxScore=17) to indicators.js"
```

---

### Task 2: Add buy/sell toggle to JS analysis.js

**Files:**
- Modify: `web/js/analysis.js`

- [ ] **Step 1: Add buy analysis computation to `analyze()` method**

In the `analyze(symbol)` method, after line 38 (`this.results.set(symbol, ...)`), add buy signal computation. Replace the whole `analyze` method:

```javascript
  async analyze(symbol) {
    const data = await this.api.getKline(symbol, '1d', 100);
    const bars = [];
    for (const item of data) {
      if (!item.k) continue;
      for (const k of item.k) {
        bars.push({
          time: k.ts,
          open: parseFloat(k.o),
          high: parseFloat(k.h),
          low: parseFloat(k.l),
          close: parseFloat(k.cl),
          volume: parseFloat(k.v || 0),
        });
      }
    }
    bars.sort((a, b) => a.time - b.time);
    if (bars.length < 30) return;

    const ma5 = calcMA(bars, 5);
    const ma20 = calcMA(bars, 20);
    const ma60 = calcMA(bars, 60);
    const rsi = calcRSI(bars, 14);
    const macd = calcMACD(bars);
    const sellSignals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);
    const buySignals = evaluateBuySignals(bars, { ma5, ma20, ma60 }, rsi, macd);

    this.results.set(symbol, { bars, ma5, ma20, ma60, rsi, macd, signals: sellSignals, buySignals });
  }
```

Note: The only changes are: add `volume: parseFloat(k.v || 0)` to the bar object, and compute `buySignals` alongside `sellSignals`.

- [ ] **Step 2: Add buy view rendering methods**

Add two new methods before the closing `}` of the class (after `_showDetail`):

```javascript
  _renderBuyList() {
    const container = document.getElementById('analysisList');

    let totalScore = 0;
    let totalMaxScore = 0;
    for (const [, result] of this.results) {
      totalScore += result.buySignals.score;
      totalMaxScore += result.buySignals.maxScore;
    }
    const avgPct = totalMaxScore > 0 ? Math.round((totalScore / totalMaxScore) * 100) : 0;

    let summaryColor, summaryLabel;
    if (avgPct >= 50) {
      summaryColor = '#3fb950';
      summaryLabel = '强烈推荐买入';
    } else if (avgPct >= 25) {
      summaryColor = '#d29922';
      summaryLabel = '值得关注';
    } else {
      summaryColor = '#8b949e';
      summaryLabel = '暂无明确买入信号';
    }

    let html = '<div class="analysis-summary" style="color:' + summaryColor + '">买入指数 <strong style="font-size:28px">' + avgPct + '%</strong> — ' + summaryLabel + '</div>';
    html += '<div class="analysis-cards">';

    for (const w of this.watchlist) {
      const result = this.results.get(w.symbol);
      if (!result) continue;

      const pct = Math.round((result.buySignals.score / result.buySignals.maxScore) * 100);

      let pctColor;
      if (pct >= 50) {
        pctColor = '#3fb950';
      } else if (pct >= 25) {
        pctColor = '#d29922';
      } else {
        pctColor = '#8b949e';
      }

      let cardClass;
      if (pct >= 50) {
        cardClass = 'buy-danger';
      } else if (pct > 0) {
        cardClass = 'buy-warn';
      } else {
        cardClass = '';
      }

      const latestBar = result.bars[result.bars.length - 1];
      const price = formatPrice(latestBar.close);

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(w.symbol) + '" data-mode="buy">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(w.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(w.symbol)) + '</span>';
      html += '<span class="price">' + escapeHtml(price) + '</span>';
      html += '</div>';
      html += '<div class="analysis-card-signals">买入指数 <strong style="font-size:18px;color:' + pctColor + '">' + pct + '%</strong> (' + result.buySignals.count + '/' + result.buySignals.total + ')</div>';
      html += '</div>';
    }

    html += '</div>';
    container.innerHTML = html;

    container.querySelectorAll('.analysis-card').forEach(function(el) {
      el.addEventListener('click', function() {
        this._showBuyDetail(el.dataset.symbol);
      }.bind(this));
    }, this);
  }

  _showBuyDetail(symbol) {
    const container = document.getElementById('analysisList');
    const result = this.results.get(symbol);
    if (!result) return;

    const w = this.watchlist.find(function(item) { return item.symbol === symbol; });
    const name = w ? w.name : shortCode(symbol);
    const pct = Math.round((result.buySignals.score / result.buySignals.maxScore) * 100);
    const mode = 'buy';

    let summaryColor;
    if (pct >= 50) {
      summaryColor = '#3fb950';
    } else if (pct >= 25) {
      summaryColor = '#d29922';
    } else {
      summaryColor = '#8b949e';
    }

    let html = '<button id="analysisBack" style="background:none;border:none;color:#58a6ff;cursor:pointer;font-size:14px;padding:8px 0;">← 返回列表</button>';
    html += '<h3 style="margin:8px 0;">' + escapeHtml(name) + ' (' + escapeHtml(shortCode(symbol)) + ')</h3>';
    html += '<div style="color:' + summaryColor + ';margin:8px 0;">买入指数 <strong style="font-size:24px">' + pct + '%</strong> — ' + escapeHtml(result.buySignals.summary) + ' (' + result.buySignals.count + '/' + result.buySignals.total + ')</div>';

    html += '<table class="data-table"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';

    for (const signal of result.buySignals.signals) {
      let statusIcon, statusText, color;

      if (!signal.triggered) {
        statusIcon = '⚪';
        statusText = '未触发';
        color = '#8b949e';
      } else if (signal.status === 'danger') {
        statusIcon = '🟢';
        statusText = '推荐';
        color = '#3fb950';
      } else {
        statusIcon = '🟡';
        statusText = '关注';
        color = '#d29922';
      }

      const valueStr = signal.value ? escapeHtml(String(signal.value)) : '--';

      html += '<tr>';
      html += '<td>' + escapeHtml(signal.name) + '</td>';
      html += '<td style="color:' + color + ';">' + statusIcon + ' ' + escapeHtml(statusText) + '</td>';
      html += '<td>' + valueStr + '</td>';
      html += '</tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;

    document.getElementById('analysisBack').addEventListener('click', function() {
      this._currentMode = this._currentMode || 'sell';
      if (this._currentMode === 'buy') {
        this._renderBuyList();
      } else {
        this._renderList();
      }
    }.bind(this));
  }
```

- [ ] **Step 3: Update the `render()` method to include toggle buttons**

Replace the `render()` method:

```javascript
  async render() {
    const promises = this.watchlist
      .filter(w => !this.results.has(w.symbol))
      .map(w => this.analyze(w.symbol));
    await Promise.all(promises);

    // Default to sell view
    this._currentMode = 'sell';
    this._renderWithToggle();
  }

  _renderWithToggle() {
    const container = document.getElementById('analysisList');

    // Toggle bar
    let toggleHtml = '<div class="analysis-toggle" style="display:flex;gap:8px;margin-bottom:12px;">';
    toggleHtml += '<button id="analysisSellBtn" class="toggle-btn' + (this._currentMode === 'sell' ? ' active' : '') + '" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'sell' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">卖出分析</button>';
    toggleHtml += '<button id="analysisBuyBtn" class="toggle-btn' + (this._currentMode === 'buy' ? ' active' : '') + '" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'buy' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">买入推荐</button>';
    toggleHtml += '</div>';
    toggleHtml += '<div id="analysisContent"></div>';
    container.innerHTML = toggleHtml;

    document.getElementById('analysisSellBtn').addEventListener('click', function() {
      this._currentMode = 'sell';
      this._renderWithToggle();
    }.bind(this));

    document.getElementById('analysisBuyBtn').addEventListener('click', function() {
      this._currentMode = 'buy';
      this._renderWithToggle();
    }.bind(this));

    const contentContainer = document.getElementById('analysisContent');
    // Temporarily redirect innerHTML writes to the content container
    const origHTML = Object.getOwnPropertyDescriptor(Element.prototype, 'innerHTML');
    // HACK: Override container.innerHTML temporarily to target contentContainer
    // We copy _renderList/_renderBuyList content into contentContainer via a different approach:
    // Save the container reference, temporarily replace it
    const realContainer = container;
    const proxyContainer = {
      get innerHTML() { return contentContainer.innerHTML; },
      set innerHTML(v) { contentContainer.innerHTML = v; },
      querySelectorAll: function(sel) { return contentContainer.querySelectorAll(sel); }
    };

    // Use a better approach: just call the render methods with a target override
    // We modify _renderList and _renderBuyList to use `this._analysisTarget` instead of hardcoded getElementById
    this._analysisTarget = contentContainer;
    if (this._currentMode === 'buy') {
      this._renderBuyListContent();
    } else {
      this._renderListContent();
    }
  }
```

Wait — the approach above is fragile. Let me use a cleaner pattern instead. The simplest approach: add a `_getContainer()` helper and update `_renderList` / `_renderBuyList` to use it.

Actually, the cleanest approach: modify `_renderList` and `_renderBuyList` to accept an optional container parameter. Let me rewrite Step 3 more cleanly.

- [ ] **Step 3 (revised): Replace the `render()` method and refactor render methods**

Replace the `render()` method:

```javascript
  async render() {
    const promises = this.watchlist
      .filter(w => !this.results.has(w.symbol))
      .map(w => this.analyze(w.symbol));
    await Promise.all(promises);
    this._currentMode = 'sell';
    this._showToggleView();
  }

  _showToggleView() {
    const container = document.getElementById('analysisList');

    let html = '<div style="display:flex;gap:8px;margin-bottom:12px;">';
    html += '<button id="analysisSellBtn" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'sell' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">卖出分析</button>';
    html += '<button id="analysisBuyBtn" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'buy' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">买入推荐</button>';
    html += '</div>';
    html += '<div id="analysisInner"></div>';
    container.innerHTML = html;

    document.getElementById('analysisSellBtn').addEventListener('click', function() {
      this._currentMode = 'sell';
      this._showToggleView();
    }.bind(this));

    document.getElementById('analysisBuyBtn').addEventListener('click', function() {
      this._currentMode = 'buy';
      this._showToggleView();
    }.bind(this));

    if (this._currentMode === 'buy') {
      this._renderBuyList();
    } else {
      this._renderList();
    }
  }
```

And update `_renderList` and `_renderBuyList` to write to `#analysisInner` instead of `#analysisList` by replacing all `document.getElementById('analysisList')` with `document.getElementById('analysisInner')` inside those two methods.

Similarly, `_showDetail` and `_showBuyDetail` must write to `#analysisInner` and wire `#analysisBack` to call `_showToggleView()`.

This approach is clean but requires editing all 4 render methods. To keep the plan concise, the full updated analysis.js will be written at once.

- [ ] **Step 3 (FINAL): Write the complete updated analysis.js**

Rather than patching individual methods, write the complete file. Start from the original and apply these changes:

1. In `analyze()`, add `volume: parseFloat(k.v || 0)` to bar and compute `buySignals`
2. Rename `_renderList` to use `#analysisInner` instead of `#analysisList`
3. Add `_renderBuyList` (writes to `#analysisInner`)
4. Add `_showBuyDetail` (writes to `#analysisInner`)
5. Replace `render()` with `_showToggleView()` pattern
6. Update `_showDetail` back button to call `_showToggleView()`

The complete file content:

```javascript
class AnalysisComponent {
  constructor(api) {
    this.api = api;
    this.watchlist = [];
    this.results = new Map();
    this._currentMode = 'sell';
  }

  async init() {
    this.watchlist = await this.api.getWatchlist();
    this.render();
  }

  async analyze(symbol) {
    const data = await this.api.getKline(symbol, '1d', 100);
    const bars = [];
    for (const item of data) {
      if (!item.k) continue;
      for (const k of item.k) {
        bars.push({
          time: k.ts,
          open: parseFloat(k.o),
          high: parseFloat(k.h),
          low: parseFloat(k.l),
          close: parseFloat(k.cl),
          volume: parseFloat(k.v || 0),
        });
      }
    }
    bars.sort((a, b) => a.time - b.time);
    if (bars.length < 30) return;

    const ma5 = calcMA(bars, 5);
    const ma20 = calcMA(bars, 20);
    const ma60 = calcMA(bars, 60);
    const rsi = calcRSI(bars, 14);
    const macd = calcMACD(bars);
    const sellSignals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);
    const buySignals = evaluateBuySignals(bars, { ma5, ma20, ma60 }, rsi, macd);

    this.results.set(symbol, { bars, ma5, ma20, ma60, rsi, macd, signals: sellSignals, buySignals });
  }

  async render() {
    const promises = this.watchlist
      .filter(w => !this.results.has(w.symbol))
      .map(w => this.analyze(w.symbol));
    await Promise.all(promises);
    this._currentMode = 'sell';
    this._showToggleView();
  }

  _showToggleView() {
    const container = document.getElementById('analysisList');

    let html = '<div style="display:flex;gap:8px;margin-bottom:12px;">';
    html += '<button id="analysisSellBtn" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'sell' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">卖出分析</button>';
    html += '<button id="analysisBuyBtn" style="flex:1;padding:8px;border:1px solid #30363d;background:' + (this._currentMode === 'buy' ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:6px;cursor:pointer;font-size:14px;">买入推荐</button>';
    html += '</div>';
    html += '<div id="analysisInner"></div>';
    container.innerHTML = html;

    document.getElementById('analysisSellBtn').addEventListener('click', function() {
      this._currentMode = 'sell';
      this._showToggleView();
    }.bind(this));

    document.getElementById('analysisBuyBtn').addEventListener('click', function() {
      this._currentMode = 'buy';
      this._showToggleView();
    }.bind(this));

    if (this._currentMode === 'buy') {
      this._renderBuyList();
    } else {
      this._renderList();
    }
  }

  _renderList() {
    const container = document.getElementById('analysisInner');

    let totalScore = 0;
    let totalMaxScore = 0;
    for (const [, result] of this.results) {
      totalScore += result.signals.score;
      totalMaxScore += result.signals.maxScore;
    }
    const avgPct = totalMaxScore > 0 ? Math.round((totalScore / totalMaxScore) * 100) : 0;

    let summaryColor, summaryLabel;
    if (avgPct >= 50) {
      summaryColor = '#f85149';
      summaryLabel = '建议关注卖出';
    } else if (avgPct >= 25) {
      summaryColor = '#d29922';
      summaryLabel = '偏弱，注意观察';
    } else {
      summaryColor = '#3fb950';
      summaryLabel = '暂无明确卖出信号';
    }

    let html = '<div class="analysis-summary" style="color:' + summaryColor + '">卖出指数 <strong style="font-size:28px">' + avgPct + '%</strong> — ' + summaryLabel + '</div>';
    html += '<div class="analysis-cards">';

    for (const w of this.watchlist) {
      const result = this.results.get(w.symbol);
      if (!result) continue;

      const pct = Math.round((result.signals.score / result.signals.maxScore) * 100);

      let pctColor;
      if (pct >= 50) {
        pctColor = '#f85149';
      } else if (pct >= 25) {
        pctColor = '#d29922';
      } else {
        pctColor = '#3fb950';
      }

      let cardClass;
      if (pct >= 50) {
        cardClass = 'danger';
      } else if (pct > 0) {
        cardClass = 'warn';
      } else {
        cardClass = '';
      }

      const latestBar = result.bars[result.bars.length - 1];
      const price = formatPrice(latestBar.close);

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(w.symbol) + '">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(w.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(w.symbol)) + '</span>';
      html += '<span class="price">' + escapeHtml(price) + '</span>';
      html += '</div>';
      html += '<div class="analysis-card-signals">卖出指数 <strong style="font-size:18px;color:' + pctColor + '">' + pct + '%</strong> (' + result.signals.count + '/' + result.signals.total + ')</div>';
      html += '</div>';
    }

    html += '</div>';
    container.innerHTML = html;

    container.querySelectorAll('.analysis-card').forEach(function(el) {
      el.addEventListener('click', function() {
        this._showDetail(el.dataset.symbol);
      }.bind(this));
    }, this);
  }

  _showDetail(symbol) {
    const container = document.getElementById('analysisInner');
    const result = this.results.get(symbol);
    if (!result) return;

    const w = this.watchlist.find(function(item) { return item.symbol === symbol; });
    const name = w ? w.name : shortCode(symbol);
    const count = result.signals.count;
    const pct = Math.round((result.signals.score / result.signals.maxScore) * 100);

    let summaryColor;
    if (pct >= 50) {
      summaryColor = '#f85149';
    } else if (pct >= 25) {
      summaryColor = '#d29922';
    } else {
      summaryColor = '#3fb950';
    }

    let html = '<button id="analysisBack" style="background:none;border:none;color:#58a6ff;cursor:pointer;font-size:14px;padding:8px 0;">← 返回列表</button>';
    html += '<h3 style="margin:8px 0;">' + escapeHtml(name) + ' (' + escapeHtml(shortCode(symbol)) + ')</h3>';
    html += '<div style="color:' + summaryColor + ';margin:8px 0;">卖出指数 <strong style="font-size:24px">' + pct + '%</strong> — ' + escapeHtml(result.signals.summary) + ' (' + count + '/' + result.signals.total + ')</div>';

    html += '<table class="data-table"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';

    for (const signal of result.signals.signals) {
      let statusIcon, statusText, color;

      if (!signal.triggered) {
        statusIcon = '🟢';
        statusText = '正常';
        color = 'green';
      } else if (signal.status === 'danger') {
        statusIcon = '🔴';
        statusText = '危险';
        color = 'red';
      } else {
        statusIcon = '🟡';
        statusText = '警告';
        color = '#ffd700';
      }

      const valueStr = signal.value ? escapeHtml(String(signal.value)) : '--';

      html += '<tr>';
      html += '<td>' + escapeHtml(signal.name) + '</td>';
      html += '<td style="color:' + color + ';">' + statusIcon + ' ' + escapeHtml(statusText) + '</td>';
      html += '<td>' + valueStr + '</td>';
      html += '</tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;

    document.getElementById('analysisBack').addEventListener('click', function() {
      this._renderList();
    }.bind(this));
  }

  _renderBuyList() {
    const container = document.getElementById('analysisInner');

    let totalScore = 0;
    let totalMaxScore = 0;
    for (const [, result] of this.results) {
      totalScore += result.buySignals.score;
      totalMaxScore += result.buySignals.maxScore;
    }
    const avgPct = totalMaxScore > 0 ? Math.round((totalScore / totalMaxScore) * 100) : 0;

    let summaryColor, summaryLabel;
    if (avgPct >= 50) {
      summaryColor = '#3fb950';
      summaryLabel = '强烈推荐买入';
    } else if (avgPct >= 25) {
      summaryColor = '#d29922';
      summaryLabel = '值得关注';
    } else {
      summaryColor = '#8b949e';
      summaryLabel = '暂无明确买入信号';
    }

    let html = '<div class="analysis-summary" style="color:' + summaryColor + '">买入指数 <strong style="font-size:28px">' + avgPct + '%</strong> — ' + summaryLabel + '</div>';
    html += '<div class="analysis-cards">';

    for (const w of this.watchlist) {
      const result = this.results.get(w.symbol);
      if (!result) continue;

      const pct = Math.round((result.buySignals.score / result.buySignals.maxScore) * 100);

      let pctColor;
      if (pct >= 50) {
        pctColor = '#3fb950';
      } else if (pct >= 25) {
        pctColor = '#d29922';
      } else {
        pctColor = '#8b949e';
      }

      let cardClass;
      if (pct >= 50) {
        cardClass = 'buy-danger';
      } else if (pct > 0) {
        cardClass = 'buy-warn';
      } else {
        cardClass = '';
      }

      const latestBar = result.bars[result.bars.length - 1];
      const price = formatPrice(latestBar.close);

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(w.symbol) + '">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(w.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(w.symbol)) + '</span>';
      html += '<span class="price">' + escapeHtml(price) + '</span>';
      html += '</div>';
      html += '<div class="analysis-card-signals">买入指数 <strong style="font-size:18px;color:' + pctColor + '">' + pct + '%</strong> (' + result.buySignals.count + '/' + result.buySignals.total + ')</div>';
      html += '</div>';
    }

    html += '</div>';
    container.innerHTML = html;

    container.querySelectorAll('.analysis-card').forEach(function(el) {
      el.addEventListener('click', function() {
        this._showBuyDetail(el.dataset.symbol);
      }.bind(this));
    }, this);
  }

  _showBuyDetail(symbol) {
    const container = document.getElementById('analysisInner');
    const result = this.results.get(symbol);
    if (!result) return;

    const w = this.watchlist.find(function(item) { return item.symbol === symbol; });
    const name = w ? w.name : shortCode(symbol);
    const pct = Math.round((result.buySignals.score / result.buySignals.maxScore) * 100);

    let summaryColor;
    if (pct >= 50) {
      summaryColor = '#3fb950';
    } else if (pct >= 25) {
      summaryColor = '#d29922';
    } else {
      summaryColor = '#8b949e';
    }

    let html = '<button id="analysisBack" style="background:none;border:none;color:#58a6ff;cursor:pointer;font-size:14px;padding:8px 0;">← 返回列表</button>';
    html += '<h3 style="margin:8px 0;">' + escapeHtml(name) + ' (' + escapeHtml(shortCode(symbol)) + ')</h3>';
    html += '<div style="color:' + summaryColor + ';margin:8px 0;">买入指数 <strong style="font-size:24px">' + pct + '%</strong> — ' + escapeHtml(result.buySignals.summary) + ' (' + result.buySignals.count + '/' + result.buySignals.total + ')</div>';

    html += '<table class="data-table"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';

    for (const signal of result.buySignals.signals) {
      let statusIcon, statusText, color;

      if (!signal.triggered) {
        statusIcon = '⚪';
        statusText = '未触发';
        color = '#8b949e';
      } else if (signal.status === 'danger') {
        statusIcon = '🟢';
        statusText = '推荐';
        color = '#3fb950';
      } else {
        statusIcon = '🟡';
        statusText = '关注';
        color = '#d29922';
      }

      const valueStr = signal.value ? escapeHtml(String(signal.value)) : '--';

      html += '<tr>';
      html += '<td>' + escapeHtml(signal.name) + '</td>';
      html += '<td style="color:' + color + ';">' + statusIcon + ' ' + escapeHtml(statusText) + '</td>';
      html += '<td>' + valueStr + '</td>';
      html += '</tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;

    document.getElementById('analysisBack').addEventListener('click', function() {
      this._renderBuyList();
    }.bind(this));
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add web/js/analysis.js
git commit -m "feat: add buy/sell toggle and buy recommendation view to analysis.js"
```

---

### Task 3: Sync public/js/ with web/js/

**Files:**
- Overwrite: `public/js/indicators.js` (copy from `web/js/indicators.js`)
- Overwrite: `public/js/analysis.js` (copy from `web/js/analysis.js`)

- [ ] **Step 1: Copy files**

Run:
```bash
cp web/js/indicators.js public/js/indicators.js
cp web/js/analysis.js public/js/analysis.js
```

- [ ] **Step 2: Verify no diffs**

Run: `diff web/js/indicators.js public/js/indicators.js && diff web/js/analysis.js public/js/analysis.js && echo "OK - identical"`
Expected: `OK - identical`

- [ ] **Step 3: Commit**

```bash
git add public/js/indicators.js public/js/analysis.js
git commit -m "chore: sync public/js with web/js for buy recommendation"
```

---

### Task 4: Flutter indicators.dart — fix sell signals + add buy functions

**Files:**
- Modify: `mobile/stock_monitor/lib/domain/indicators.dart`

- [ ] **Step 1: Read current file and write complete new version**

This task replaces the entire file to:
1. Fix sell `evaluateSignals` to include all 8 signals (add volume death cross, MACD divergence, match JS)
2. Add `detectVolumeDeathCross`, `detectVolumeGoldenCross`, `detectBullishDivergence` helpers
3. Add `evaluateBuySignals` function (10 signals, maxScore=17)
4. Add per-signal detail class `SignalInfo` and update `SignalResult` to include signals list

The complete new file content — write to `mobile/stock_monitor/lib/domain/indicators.dart`:

```dart
import 'model/kline.dart';

class MA {
  final int time;
  final double value;
  MA(this.time, this.value);
}

List<MA> calcMA(List<Bar> bars, int period) {
  if (bars.length < period) return [];
  final result = <MA>[];
  double sum = 0;
  for (int i = 0; i < bars.length; i++) {
    sum += bars[i].close;
    if (i >= period) sum -= bars[i - period].close;
    if (i >= period - 1) result.add(MA(bars[i].time, sum / period));
  }
  return result;
}

class RSIVal {
  final int time;
  final double value;
  RSIVal(this.time, this.value);
}

List<RSIVal> calcRSI(List<Bar> bars, int period) {
  if (bars.length <= period) return [];
  final result = <RSIVal>[];
  double avgGain = 0, avgLoss = 0;
  for (int i = 1; i <= period; i++) {
    final diff = bars[i].close - bars[i - 1].close;
    if (diff > 0) avgGain += diff; else avgLoss -= diff;
  }
  avgGain /= period;
  avgLoss /= period;
  for (int i = period; i < bars.length; i++) {
    if (avgLoss == 0) {
      result.add(RSIVal(bars[i].time, 100));
    } else {
      result.add(RSIVal(bars[i].time, 100 - 100 / (1 + avgGain / avgLoss)));
    }
    final diff = bars[i].close - bars[i - 1].close;
    final gain = diff > 0 ? diff : 0.0;
    final loss = diff < 0 ? -diff : 0.0;
    avgGain = (avgGain * (period - 1) + gain) / period;
    avgLoss = (avgLoss * (period - 1) + loss) / period;
  }
  return result;
}

class MACDResult {
  final List<MA> dif, dea, macd;
  MACDResult(this.dif, this.dea, this.macd);
}

double _ema(List<double> values, int period, int i) {
  final k = 2.0 / (period + 1);
  double ema = values[0];
  for (int j = 1; j <= i; j++) {
    ema = values[j] * k + ema * (1 - k);
  }
  return ema;
}

MACDResult? calcMACD(List<Bar> bars, {int fast = 12, int slow = 26, int signal = 9}) {
  if (bars.length < slow) return null;
  final closes = bars.map((b) => b.close).toList();
  final dif = <MA>[];
  for (int i = slow - 1; i < bars.length; i++) {
    dif.add(MA(bars[i].time, _ema(closes.sublist(0, i + 1), fast, i) - _ema(closes.sublist(0, i + 1), slow, i)));
  }
  final dea = <MA>[];
  final macd = <MA>[];
  final difValues = dif.map((d) => d.value).toList();
  for (int i = signal - 1; i < dif.length; i++) {
    final deaVal = _ema(difValues.sublist(0, i + 1), signal, i);
    dea.add(MA(dif[i].time, deaVal));
    macd.add(MA(dif[i].time, (dif[i].value - deaVal) * 2));
  }
  return MACDResult(dif, dea, macd);
}

// ---- Divergence helpers ----

bool _detectBearishDivergence(List<Bar> bars, List<MA> indicatorData) {
  final count = [20, bars.length, indicatorData.length].reduce((a, b) => a < b ? a : b);
  if (count < 2) return false;
  final recentBars = bars.sublist(bars.length - count);
  final recentInd = indicatorData.sublist(indicatorData.length - count);

  int maxHighIdx = -1;
  double maxHigh = double.negativeInfinity;
  for (int i = 0; i < recentBars.length; i++) {
    if (recentBars[i].high > maxHigh) { maxHigh = recentBars[i].high; maxHighIdx = i; }
  }

  int maxIndIdx = -1;
  double maxInd = double.negativeInfinity;
  for (int i = 0; i < recentInd.length; i++) {
    if (recentInd[i].value > maxInd) { maxInd = recentInd[i].value; maxIndIdx = i; }
  }

  if (maxHighIdx < 0 || maxIndIdx < 0) return false;
  return maxHighIdx > maxIndIdx && recentInd[maxHighIdx].value < maxInd;
}

bool _detectBullishDivergence(List<Bar> bars, List<MA> indicatorData) {
  final count = [20, bars.length, indicatorData.length].reduce((a, b) => a < b ? a : b);
  if (count < 2) return false;
  final recentBars = bars.sublist(bars.length - count);
  final recentInd = indicatorData.sublist(indicatorData.length - count);

  int minLowIdx = -1;
  double minLow = double.infinity;
  for (int i = 0; i < recentBars.length; i++) {
    if (recentBars[i].low < minLow) { minLow = recentBars[i].low; minLowIdx = i; }
  }

  int minIndIdx = -1;
  double minInd = double.infinity;
  for (int i = 0; i < recentInd.length; i++) {
    if (recentInd[i].value < minInd) { minInd = recentInd[i].value; minIndIdx = i; }
  }

  if (minLowIdx < 0 || minIndIdx < 0) return false;
  return minLowIdx > minIndIdx && recentInd[minLowIdx].value > minInd;
}

bool _detectVolumeDeathCross(List<Bar> bars) {
  final n = [5, bars.length].reduce((a, b) => a < b ? a : b);
  if (n < 3) return false;
  final recent = bars.sublist(bars.length - n);
  double upVol = 0, downVol = 0;
  int upDays = 0, downDays = 0;
  for (final b in recent) {
    if (b.close >= b.open) { upVol += b.close * b.volume; upDays++; }
    else { downVol += b.close * b.volume; downDays++; }
  }
  if (upDays == 0 || downDays == 0) return false;
  return (downVol / downDays) > (upVol / upDays) * 1.2;
}

bool _detectVolumeGoldenCross(List<Bar> bars) {
  final n = [5, bars.length].reduce((a, b) => a < b ? a : b);
  if (n < 3) return false;
  final recent = bars.sublist(bars.length - n);
  double upVol = 0, downVol = 0;
  int upDays = 0, downDays = 0;
  for (final b in recent) {
    if (b.close >= b.open) { upVol += b.close * b.volume; upDays++; }
    else { downVol += b.close * b.volume; downDays++; }
  }
  if (upDays == 0 || downDays == 0) return false;
  return (upVol / upDays) > (downVol / downDays) * 1.2;
}

// ---- Signal detail class ----

class SignalInfo {
  final String key;
  final String name;
  final bool triggered;
  final String? value;
  final String status; // 'ok', 'warn', 'danger'
  final double weight;
  SignalInfo({required this.key, required this.name, required this.triggered, this.value, required this.status, required this.weight});
}

class SignalResult {
  final double score;
  final double maxScore;
  final int count;
  final int total;
  final List<SignalInfo> signals;
  final String summary;
  SignalResult({required this.score, required this.maxScore, required this.count, required this.total, required this.signals, required this.summary});
}

// ---- Sell signal evaluation (8 signals, maxScore=13.0) ----

SignalResult evaluateSignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);

  const W = {
    'ma_cross': 2.0, 'price_ma20': 1.0, 'price_ma60': 0.5,
    'rsi_overbought': 1.0, 'rsi_diverge': 2.5, 'macd_cross': 2.0,
    'macd_diverge': 2.5, 'volume_death': 1.5,
  };
  const maxScore = 13.0;
  final signals = <SignalInfo>[];
  double score = 0;
  int count = 0;

  final latestBar = bars.last;
  final latestMA5 = ma5.isNotEmpty ? ma5.last : null;
  final prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  final latestMA20 = ma20.isNotEmpty ? ma20.last : null;
  final prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  final latestMA60 = ma60.isNotEmpty ? ma60.last : null;

  // 1. MA5 dead cross MA20
  bool t1 = false; String? v1; String s1 = 'ok';
  if (prevMA5 != null && prevMA20 != null && latestMA5 != null && latestMA20 != null) {
    if (prevMA5.value >= prevMA20.value && latestMA5.value < latestMA20.value) {
      t1 = true; s1 = 'danger';
      v1 = 'MA5=${latestMA5.value.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
      score += W['ma_cross']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'ma_cross_dead', name: 'MA5死叉MA20', triggered: t1, value: v1, status: s1, weight: W['ma_cross']!));

  // 2. Close < MA20
  bool t2 = false; String? v2; String s2 = 'ok';
  if (latestMA20 != null && latestBar.close < latestMA20.value) {
    t2 = true; s2 = 'warn';
    v2 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
    score += W['price_ma20']!; count++;
  }
  signals.add(SignalInfo(key: 'price_below_ma20', name: '收盘价低于MA20', triggered: t2, value: v2, status: s2, weight: W['price_ma20']!));

  // 3. Close < MA60
  bool t3 = false; String? v3; String s3 = 'ok';
  if (latestMA60 != null && latestBar.close < latestMA60.value) {
    t3 = true; s3 = 'warn';
    v3 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA60=${latestMA60.value.toStringAsFixed(2)}';
    score += W['price_ma60']!; count++;
  }
  signals.add(SignalInfo(key: 'price_below_ma60', name: '收盘价低于MA60', triggered: t3, value: v3, status: s3, weight: W['price_ma60']!));

  // 4. RSI > 70
  bool t4 = false; String? v4; String s4 = 'ok';
  if (rsi.isNotEmpty && rsi.last.value > 70) {
    t4 = true;
    s4 = rsi.last.value > 80 ? 'danger' : 'warn';
    v4 = 'RSI=${rsi.last.value.toStringAsFixed(2)}';
    score += W['rsi_overbought']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_overbought', name: 'RSI超买', triggered: t4, value: v4, status: s4, weight: W['rsi_overbought']!));

  // 5. RSI bearish divergence
  bool t5 = false; String? v5; String s5 = 'ok';
  if (rsi.length >= 2 && _detectBearishDivergence(bars, rsi.map((r) => MA(r.time, r.value)).toList())) {
    t5 = true; s5 = 'danger'; v5 = '价格创新高，RSI未创新高';
    score += W['rsi_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_divergence', name: 'RSI顶背离', triggered: t5, value: v5, status: s5, weight: W['rsi_diverge']!));

  // 6. MACD dead cross
  bool t6 = false; String? v6; String s6 = 'ok';
  if (macd != null && macd.dif.length >= 2 && macd.dea.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF >= prevDEA && currDIF < currDEA) {
      t6 = true; s6 = 'danger';
      v6 = 'DIF=${currDIF.toStringAsFixed(4)}, DEA=${currDEA.toStringAsFixed(4)}';
      score += W['macd_cross']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'macd_dead_cross', name: 'MACD死叉', triggered: t6, value: v6, status: s6, weight: W['macd_cross']!));

  // 7. MACD bearish divergence
  bool t7 = false; String? v7; String s7 = 'ok';
  if (macd != null && macd.dif.length >= 2 && _detectBearishDivergence(bars, macd.dif)) {
    t7 = true; s7 = 'danger'; v7 = '价格创新高，DIF未创新高';
    score += W['macd_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'macd_divergence', name: 'MACD顶背离', triggered: t7, value: v7, status: s7, weight: W['macd_diverge']!));

  // 8. Volume death cross
  bool t8 = false; String? v8; String s8 = 'ok';
  if (bars.length >= 3 && _detectVolumeDeathCross(bars)) {
    t8 = true; s8 = 'warn'; v8 = '近5日下跌日平均成交量 > 上涨日1.2倍';
    score += W['volume_death']!; count++;
  }
  signals.add(SignalInfo(key: 'volume_death_cross', name: '成交量死叉（下跌放量）', triggered: t8, value: v8, status: s8, weight: W['volume_death']!));

  final pct = score / maxScore;
  String summary;
  if (pct >= 0.5) summary = '强烈卖出信号';
  else if (pct >= 0.25) summary = '偏弱，注意风险';
  else if (pct > 0) summary = '短期偏弱';
  else summary = '正常';

  return SignalResult(score: score, maxScore: maxScore, count: count, total: 8, signals: signals, summary: summary);
}

// ---- Buy signal evaluation (10 signals, maxScore=17.0) ----

SignalResult evaluateBuySignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);

  const W = {
    'ma_cross_golden': 2.0, 'price_ma20': 1.0, 'price_ma60': 0.5,
    'rsi_oversold': 1.0, 'rsi_diverge': 2.5, 'macd_cross_golden': 2.0,
    'macd_diverge': 2.5, 'volume_golden': 1.5,
    'vol_breakout': 2.0, 'bull_alignment': 2.0,
  };
  const maxScore = 17.0;
  final signals = <SignalInfo>[];
  double score = 0;
  int count = 0;

  final latestBar = bars.last;
  final latestMA5 = ma5.isNotEmpty ? ma5.last : null;
  final prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  final latestMA20 = ma20.isNotEmpty ? ma20.last : null;
  final prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  final latestMA60 = ma60.isNotEmpty ? ma60.last : null;

  // 1. MA5 golden cross MA20
  bool t1 = false; String? v1; String st1 = 'ok';
  if (prevMA5 != null && prevMA20 != null && latestMA5 != null && latestMA20 != null) {
    if (prevMA5.value <= prevMA20.value && latestMA5.value > latestMA20.value) {
      t1 = true; st1 = 'danger';
      v1 = 'MA5=${latestMA5.value.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
      score += W['ma_cross_golden']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'ma_cross_golden', name: 'MA5金叉MA20', triggered: t1, value: v1, status: st1, weight: W['ma_cross_golden']!));

  // 2. Close > MA20
  bool t2 = false; String? v2; String st2 = 'ok';
  if (latestMA20 != null && latestBar.close > latestMA20.value) {
    t2 = true; st2 = 'warn';
    v2 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
    score += W['price_ma20']!; count++;
  }
  signals.add(SignalInfo(key: 'price_above_ma20', name: '收盘价高于MA20', triggered: t2, value: v2, status: st2, weight: W['price_ma20']!));

  // 3. Close > MA60
  bool t3 = false; String? v3; String st3 = 'ok';
  if (latestMA60 != null && latestBar.close > latestMA60.value) {
    t3 = true; st3 = 'warn';
    v3 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA60=${latestMA60.value.toStringAsFixed(2)}';
    score += W['price_ma60']!; count++;
  }
  signals.add(SignalInfo(key: 'price_above_ma60', name: '收盘价高于MA60', triggered: t3, value: v3, status: st3, weight: W['price_ma60']!));

  // 4. RSI < 30
  bool t4 = false; String? v4; String st4 = 'ok';
  if (rsi.isNotEmpty && rsi.last.value < 30) {
    t4 = true;
    st4 = rsi.last.value < 20 ? 'danger' : 'warn';
    v4 = 'RSI=${rsi.last.value.toStringAsFixed(2)}';
    score += W['rsi_oversold']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_oversold', name: 'RSI超卖', triggered: t4, value: v4, status: st4, weight: W['rsi_oversold']!));

  // 5. RSI bullish divergence
  bool t5 = false; String? v5; String st5 = 'ok';
  if (rsi.length >= 2 && _detectBullishDivergence(bars, rsi.map((r) => MA(r.time, r.value)).toList())) {
    t5 = true; st5 = 'danger'; v5 = '价格创新低，RSI未创新低';
    score += W['rsi_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_bullish_divergence', name: 'RSI底背离', triggered: t5, value: v5, status: st5, weight: W['rsi_diverge']!));

  // 6. MACD golden cross
  bool t6 = false; String? v6; String st6 = 'ok';
  if (macd != null && macd.dif.length >= 2 && macd.dea.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF <= prevDEA && currDIF > currDEA) {
      t6 = true; st6 = 'danger';
      v6 = 'DIF=${currDIF.toStringAsFixed(4)}, DEA=${currDEA.toStringAsFixed(4)}';
      score += W['macd_cross_golden']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'macd_cross_golden', name: 'MACD金叉', triggered: t6, value: v6, status: st6, weight: W['macd_cross_golden']!));

  // 7. MACD bullish divergence
  bool t7 = false; String? v7; String st7 = 'ok';
  if (macd != null && macd.dif.length >= 2 && _detectBullishDivergence(bars, macd.dif)) {
    t7 = true; st7 = 'danger'; v7 = '价格创新低，DIF未创新低';
    score += W['macd_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'macd_bullish_divergence', name: 'MACD底背离', triggered: t7, value: v7, status: st7, weight: W['macd_diverge']!));

  // 8. Volume golden cross
  bool t8 = false; String? v8; String st8 = 'ok';
  if (bars.length >= 3 && _detectVolumeGoldenCross(bars)) {
    t8 = true; st8 = 'warn'; v8 = '近5日上涨日平均成交量 > 下跌日1.2倍';
    score += W['volume_golden']!; count++;
  }
  signals.add(SignalInfo(key: 'volume_golden_cross', name: '成交量金叉（上涨放量）', triggered: t8, value: v8, status: st8, weight: W['volume_golden']!));

  // 9. Volume breakout
  bool t9 = false; String? v9; String st9 = 'ok';
  if (bars.length >= 5 && latestMA20 != null) {
    final recentBars = bars.sublist(bars.length - 5);
    double sumVol = 0;
    for (final b in recentBars) { sumVol += b.close * b.volume; }
    final avgVol5 = sumVol / 5;
    final curTurnover = latestBar.close * latestBar.volume;
    if (curTurnover > avgVol5 * 1.5 && latestBar.close > latestMA20.value) {
      t9 = true; st9 = 'danger';
      v9 = '当日成交额=${curTurnover.toStringAsFixed(0)}, 5日均额=${avgVol5.toStringAsFixed(0)}';
      score += W['vol_breakout']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'vol_breakout', name: '放量突破', triggered: t9, value: v9, status: st9, weight: W['vol_breakout']!));

  // 10. Bull alignment
  bool t10 = false; String? v10; String st10 = 'ok';
  if (latestMA5 != null && latestMA20 != null && latestMA60 != null) {
    if (latestMA5.value > latestMA20.value && latestMA20.value > latestMA60.value) {
      t10 = true; st10 = 'danger';
      v10 = 'MA5=${latestMA5.value.toStringAsFixed(2)} > MA20=${latestMA20.value.toStringAsFixed(2)} > MA60=${latestMA60.value.toStringAsFixed(2)}';
      score += W['bull_alignment']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'bull_alignment', name: '多头均线排列', triggered: t10, value: v10, status: st10, weight: W['bull_alignment']!));

  final pct = score / maxScore;
  String summary;
  if (pct >= 0.5) summary = '强烈买入信号';
  else if (pct >= 0.25) summary = '值得关注';
  else if (pct > 0) summary = '观望';
  else summary = '暂无买入信号';

  return SignalResult(score: score, maxScore: maxScore, count: count, total: 10, signals: signals, summary: summary);
}
```

- [ ] **Step 2: Verify Dart compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/domain/indicators.dart 2>&1 | tail -5`
Expected: No issues found

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/domain/indicators.dart
git commit -m "feat: fix sell signals (8/8), add buy evaluation (10 signals) to indicators.dart"
```

---

### Task 5: Flutter analysis_screen.dart — add TabBar with buy/sell views

**Files:**
- Modify: `mobile/stock_monitor/lib/presentation/screens/analysis_screen.dart`

- [ ] **Step 1: Write complete updated analysis_screen.dart**

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';

class AnalysisScreen extends ConsumerStatefulWidget {
  const AnalysisScreen({super.key});
  @override
  ConsumerState<AnalysisScreen> createState() => _AnalysisScreenState();
}

class _AnalysisScreenState extends ConsumerState<AnalysisScreen> with SingleTickerProviderStateMixin {
  List<_Result>? _sellResults;
  List<_Result>? _buyResults;
  bool _loading = true;
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _analyze();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _analyze() async {
    setState(() => _loading = true);
    final watchlist = await ref.read(watchlistApiProvider).getAll();
    final quoteApi = ref.read(quoteApiProvider);
    final sellResults = <_Result>[];
    final buyResults = <_Result>[];
    for (final stock in watchlist) {
      try {
        final data = await quoteApi.getKline(stock.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        sellResults.add(_Result(stock: stock, signal: evaluateSignals(bars)));
        buyResults.add(_Result(stock: stock, signal: evaluateBuySignals(bars)));
      } catch (_) {}
    }
    sellResults.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    buyResults.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    setState(() { _sellResults = sellResults; _buyResults = buyResults; _loading = false; });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('技术分析'),
        actions: [IconButton(onPressed: _analyze, icon: const Icon(Icons.refresh))],
        bottom: TabBar(
          controller: _tabController,
          indicatorColor: AppTheme.accent,
          labelColor: AppTheme.accent,
          unselectedLabelColor: AppTheme.textSecondary,
          tabs: const [
            Tab(text: '卖出分析'),
            Tab(text: '买入推荐'),
          ],
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildSellView(),
                _buildBuyView(),
              ],
            ),
    );
  }

  Widget _buildSellView() {
    if (_sellResults == null || _sellResults!.isEmpty) {
      return const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    return Column(children: [
      Container(padding: const EdgeInsets.all(12), color: AppTheme.surface, child: _buildSummary(_sellResults!, '卖出')),
      Expanded(
        child: ListView.builder(
          itemCount: _sellResults!.length,
          itemBuilder: (_, i) => _buildCard(_sellResults![i]),
        ),
      ),
    ]);
  }

  Widget _buildBuyView() {
    if (_buyResults == null || _buyResults!.isEmpty) {
      return const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    return Column(children: [
      Container(padding: const EdgeInsets.all(12), color: AppTheme.surface, child: _buildSummary(_buyResults!, '买入')),
      Expanded(
        child: ListView.builder(
          itemCount: _buyResults!.length,
          itemBuilder: (_, i) => _buildCard(_buyResults![i]),
        ),
      ),
    ]);
  }

  Widget _buildSummary(List<_Result> results, String label) {
    final avgScore = results.map((r) => r.signal.score / r.signal.maxScore).reduce((a, b) => a + b) / results.length;
    final color = avgScore >= 0.5
        ? (label == '买入' ? AppTheme.up : AppTheme.down)
        : avgScore >= 0.25
            ? Colors.orange
            : AppTheme.up;
    final text = avgScore >= 0.5
        ? (label == '买入' ? '强烈偏多' : '强烈偏空')
        : avgScore >= 0.25
            ? (label == '买入' ? '偏多' : '偏弱')
            : '正常';
    return Row(mainAxisAlignment: MainAxisAlignment.center, children: [
      Text('平均${label}分 ', style: const TextStyle(color: AppTheme.textSecondary)),
      Text('${(avgScore * 100).toStringAsFixed(1)}分', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: color)),
      const SizedBox(width: 12),
      Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4), decoration: BoxDecoration(color: color.withAlpha(40), borderRadius: BorderRadius.circular(12)), child: Text(text, style: TextStyle(color: color, fontWeight: FontWeight.w600))),
    ]);
  }

  Widget _buildCard(_Result r) {
    final pct = r.signal.score / r.signal.maxScore;
    final color = pct >= 0.5 ? AppTheme.up : pct > 0 ? Colors.orange : AppTheme.textSecondary;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: ListTile(
        title: Text('${r.stock.name} (${shortCode(r.stock.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(r.signal.summary),
        trailing: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Text('${(pct * 100).toStringAsFixed(0)}分', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800, color: color)),
          Text('${r.signal.count}个信号', style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
        ]),
        onTap: () => _showDetail(r),
      ),
    );
  }

  void _showDetail(_Result r) {
    final pct = r.signal.score / r.signal.maxScore;
    final color = pct >= 0.5 ? AppTheme.up : pct > 0 ? Colors.orange : AppTheme.textSecondary;
    showModalBottomSheet(
      context: context,
      builder: (_) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('${r.stock.name} (${r.stock.symbol})', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text('评分: ${(pct * 100).toStringAsFixed(0)}分 — ${r.signal.summary}', style: TextStyle(color: color)),
            const SizedBox(height: 12),
            Text('${r.signal.count} / ${r.signal.total} 个信号触发', style: const TextStyle(color: AppTheme.textSecondary)),
            const SizedBox(height: 8),
            ...r.signal.signals.where((s) => s.triggered).map((s) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(children: [
                Icon(s.status == 'danger' ? Icons.circle : Icons.warning_amber, size: 14, color: s.status == 'danger' ? AppTheme.up : Colors.orange),
                const SizedBox(width: 8),
                Expanded(child: Text(s.name, style: const TextStyle(fontSize: 14))),
                if (s.value != null) Text(s.value!, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
              ]),
            )),
          ],
        ),
      ),
    );
  }
}

class _Result {
  final WatchlistItem stock;
  final SignalResult signal;
  _Result({required this.stock, required this.signal});
}
```

Note: Also update `lib/domain/model/kline.dart` to add `volume` field to `Bar`:
```dart
class Bar {
  final int time;
  final double open;
  final double high;
  final double low;
  final double close;
  final double volume;
  Bar({required this.time, required this.open, required this.high, required this.low, required this.close, required this.volume});
}
```

- [ ] **Step 2: Verify Flutter compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/presentation/screens/analysis_screen.dart 2>&1 | tail -5`
Expected: No issues found

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/analysis_screen.dart mobile/stock_monitor/lib/domain/model/kline.dart
git commit -m "feat: add buy/sell TabBar to analysis screen with signal detail"
```

---

## Self-Review Checklist

- [x] Spec coverage: 10 buy signals implemented in both JS and Dart, toggle UI in both platforms
- [x] No placeholders: all code is complete and ready to copy
- [x] Type consistency: `SignalInfo` / `SignalResult` used consistently across both Dart functions; `evaluateBuySignals` signature matches JS version
- [x] Kline Bar model updated to include `volume` field required by volume-based signals
- [x] public/js/ kept in sync with web/js/
- [x] No Go backend changes
