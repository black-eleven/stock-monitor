class AnalysisComponent {
  constructor(api) {
    this.api = api;
    this.watchlist = [];
    this.results = new Map();
    this._currentMode = 'sell';
    this._sortMode = 'score';
    this._exchangeFilter = 'ALL';
    this._currentStrategy = '';
    this.quotes = {};
  }

  updateQuote(quote) {
    this.quotes[quote.code] = quote;
  }

  async init() {
    this.watchlist = await this.api.getWatchlist();
    try {
      const resp = await this.api.get('/api/strategy/list');
      this._strategies = (resp && resp.strategies) ? resp.strategies : [];
      this._displayNames = (resp && resp.displayNames) ? resp.displayNames : [];
    } catch (_) {
      this._strategies = [];
      this._displayNames = [];
    }
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

    // Record signals to backend
    this._recordSignal(symbol, buySignals, sellSignals);
  }

  async _recordSignal(symbol, buySignals, sellSignals) {
    try {
      const buyPct = Math.round((buySignals.score / buySignals.maxScore) * 100);
      const sellPct = Math.round((sellSignals.score / sellSignals.maxScore) * 100);
      const resp = await this.api.post('/api/signals/record', {
        symbol,
        buyScore: Math.round(buySignals.score * 100) / 100,
        buyPct,
        sellScore: Math.round(sellSignals.score * 100) / 100,
        sellPct,
        buyCount: buySignals.count,
        sellCount: sellSignals.count,
      });
      if (resp && resp.alert) {
        showToast('⚠ ' + resp.alert.message, 'alert');
      }
    } catch (_) {}
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

    // Sort and exchange filter bar
    html += '<div style="display:flex;align-items:center;gap:6px;margin-bottom:12px;flex-wrap:wrap;">';
    html += '<button id="analysisSortBtn" style="padding:4px 10px;border:1px solid #30363d;background:#161b22;color:#e6edf3;border-radius:12px;cursor:pointer;font-size:12px;">' + (this._sortMode === 'score' ? '按信号' : '按名称') + '</button>';
    html += '<span style="color:#484f58;margin:0 4px;">|</span>';
    var exchanges = ['ALL', 'US', 'HK', 'SH', 'SZ'];
    var exchangeLabels = {ALL: '全部', US: 'US', HK: 'HK', SH: 'SH', SZ: 'SZ'};
    for (var i = 0; i < exchanges.length; i++) {
      var ex = exchanges[i];
      var isActive = this._exchangeFilter === ex;
      html += '<button class="analysisExBtn" data-ex="' + ex + '" style="padding:4px 10px;border:1px solid #30363d;background:' + (isActive ? '#1f6feb' : '#161b22') + ';color:#e6edf3;border-radius:12px;cursor:pointer;font-size:12px;">' + exchangeLabels[ex] + '</button>';
    }
    html += '</div>';

    html += '<div id="analysisInner"></div>';
    container.innerHTML = html;

    document.getElementById('analysisSortBtn').addEventListener('click', function() {
      this._sortMode = this._sortMode === 'score' ? 'name' : 'score';
      this._showToggleView();
    }.bind(this));

    container.querySelectorAll('.analysisExBtn').forEach(function(btn) {
      btn.addEventListener('click', function() {
        this._exchangeFilter = btn.dataset.ex;
        this._showToggleView();
      }.bind(this));
    }, this);

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

  _getFilteredSorted(signalsKey) {
    var items = [];
    for (var wi = 0; wi < this.watchlist.length; wi++) {
      var w = this.watchlist[wi];
      var result = this.results.get(w.symbol);
      if (!result) continue;

      if (this._exchangeFilter !== 'ALL') {
        if (w.symbol.indexOf(this._exchangeFilter + ':') !== 0) continue;
      }

      items.push({
        symbol: w.symbol,
        name: w.name,
        result: result,
        score: result[signalsKey].score,
        maxScore: result[signalsKey].maxScore
      });
    }

    if (this._sortMode === 'score') {
      items.sort(function(a, b) {
        var pctA = a.maxScore > 0 ? a.score / a.maxScore : 0;
        var pctB = b.maxScore > 0 ? b.score / b.maxScore : 0;
        return pctB - pctA;
      });
    } else {
      items.sort(function(a, b) {
        return a.name.localeCompare(b.name, 'zh');
      });
    }

    return items;
  }

  _renderList() {
    const container = document.getElementById('analysisInner');

    const items = this._getFilteredSorted('signals');

    if (items.length === 0) {
      container.innerHTML = '<div class="empty-state">该交易所暂无数据</div>';
      return;
    }

    let totalScore = 0;
    let totalMaxScore = 0;
    for (var i = 0; i < items.length; i++) {
      totalScore += items[i].score;
      totalMaxScore += items[i].maxScore;
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

    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      var result = item.result;
      var pct = Math.round((item.score / item.maxScore) * 100);

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

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(item.symbol) + '">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(item.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(item.symbol)) + '</span>';
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

    // Top row: quote info (left) + strategy analysis (right)
    html += '<div style="display:flex;gap:12px;align-items:flex-start;margin-bottom:6px;">';

    // Quote info
    const q = this.quotes[symbol];
    html += '<div style="flex:1.5;min-width:0;">';
    if (q) {
      const changeDir = q.price >= q.yp ? '+' : '';
      const changeColor = q.price >= q.yp ? '#f85149' : '#3fb950';
      html += '<div style="display:flex;gap:12px;flex-wrap:wrap;font-size:13px;line-height:1.8;">';
      html += '<span>现价 <strong style="color:' + changeColor + '">' + formatPrice(q.price) + '</strong></span>';
      html += '<span>涨幅 <strong style="color:' + changeColor + '">' + changeDir + (q.yp ? ((q.price - q.yp) / q.yp * 100).toFixed(2) : '--') + '%</strong></span>';
      html += '<span>今开 ' + formatPrice(q.open) + '</span>';
      html += '<span>最高 ' + formatPrice(q.high) + '</span>';
      html += '<span>最低 ' + formatPrice(q.low) + '</span>';
      html += '<span>昨收 ' + formatPrice(q.yp) + '</span>';
      html += '</div>';
    }
    html += '<div style="color:' + summaryColor + ';margin:4px 0 0;">卖出指数 <strong style="font-size:18px">' + pct + '%</strong> — ' + escapeHtml(result.signals.summary) + ' (' + count + '/' + result.signals.total + ')</div>';
    html += '</div>';

    // Strategy analysis
    html += '<div style="flex:1;min-width:0;">';
    html += '<select id="strategySelect" style="width:100%;padding:2px 6px;background:#161b22;border:1px solid #30363d;color:#c9d1d9;border-radius:4px;font-size:12px;margin-bottom:4px;">';
    for (let i = 0; i < (this._strategies || []).length; i++) {
      const key = this._strategies[i];
      const label = (this._displayNames && this._displayNames[i]) ? this._displayNames[i] : key;
      html += '<option value="' + key + '"' + (key === (this._currentStrategy || 'event_driven') ? ' selected' : '') + '>' + label + '</option>';
    }
    html += '</select>';
    html += '<div id="strategyResult" style="color:#c9d1d9;line-height:1.3;white-space:pre-wrap;font-size:12px;"></div>';
    html += '</div></div>';

    // Indicator table (full width)
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

    const formatMd = (text) => {
      return escapeHtml(text)
        .replace(/^### (.+)$/gm, '<h4 style="color:#58a6ff;margin:4px 0 1px;font-size:13px;">$1</h4>')
        .replace(/^## (.+)$/gm, '<h3 style="color:#ffd700;margin:6px 0 2px;font-size:14px;">$1</h3>')
        .replace(/\*\*(.+?)\*\*/g, '<strong style="color:#ffd700">$1</strong>')
        .replace(/^- (.+)$/gm, '<span style="color:#58a6ff">•</span> $1')
        .replace(/^(\d+)\. (.+)$/gm, '<span style="color:#58a6ff">$1.</span> $2')
        .replace(/\b(1[7-9]\d{8})\b/g, (_, ts) => {
          const d = new Date(parseInt(ts) * 1000 + 8 * 3600 * 1000);
          const pad = n => String(n).padStart(2, '0');
          return d.getUTCFullYear() + '-' + pad(d.getUTCMonth() + 1) + '-' + pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' + pad(d.getUTCMinutes());
        })
        .replace(/\n/g, '<br>');
    };
    const bars = result.bars;
    const runStrategy = (strategy) => {
      document.getElementById('strategyResult').innerHTML = '分析中...';
      const barData = bars.map(b => ({ ts: b.time, o: b.open, cl: b.close, h: b.high, l: b.low, v: b.volume }));
      this.api.post('/api/strategy/analyze', { strategy: strategy, symbol: symbol, bars: barData }).then(resp => {
        document.getElementById('strategyResult').innerHTML = formatMd(resp.analysis || '无分析结果');
      }).catch(e => {
        document.getElementById('strategyResult').innerHTML = '失败: ' + e.message;
      });
    };
    document.getElementById('strategySelect').addEventListener('change', function() {
      const v = document.getElementById('strategySelect').value; this._currentStrategy = v; runStrategy(v);
    });
    runStrategy(this._currentStrategy || 'event_driven');
  }

  _renderBuyList() {
    const container = document.getElementById('analysisInner');

    const items = this._getFilteredSorted('buySignals');

    if (items.length === 0) {
      container.innerHTML = '<div class="empty-state">该交易所暂无数据</div>';
      return;
    }

    let totalScore = 0;
    let totalMaxScore = 0;
    for (var i = 0; i < items.length; i++) {
      totalScore += items[i].score;
      totalMaxScore += items[i].maxScore;
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

    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      var result = item.result;
      var pct = Math.round((item.score / item.maxScore) * 100);

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

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(item.symbol) + '">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(item.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(item.symbol)) + '</span>';
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

    // Top row: quote info (left) + strategy analysis (right)
    html += '<div style="display:flex;gap:12px;align-items:flex-start;margin-bottom:6px;">';

    const q2 = this.quotes[symbol];
    html += '<div style="flex:1.5;min-width:0;">';
    if (q2) {
      const changeDir = q2.price >= q2.yp ? '+' : '';
      const changeColor = q2.price >= q2.yp ? '#f85149' : '#3fb950';
      html += '<div style="display:flex;gap:12px;flex-wrap:wrap;font-size:13px;line-height:1.8;">';
      html += '<span>现价 <strong style="color:' + changeColor + '">' + formatPrice(q2.price) + '</strong></span>';
      html += '<span>涨幅 <strong style="color:' + changeColor + '">' + changeDir + (q2.yp ? ((q2.price - q2.yp) / q2.yp * 100).toFixed(2) : '--') + '%</strong></span>';
      html += '<span>今开 ' + formatPrice(q2.open) + '</span>';
      html += '<span>最高 ' + formatPrice(q2.high) + '</span>';
      html += '<span>最低 ' + formatPrice(q2.low) + '</span>';
      html += '<span>昨收 ' + formatPrice(q2.yp) + '</span>';
      html += '</div>';
    }
    html += '<div style="color:' + summaryColor + ';margin:4px 0 0;">买入指数 <strong style="font-size:18px">' + pct + '%</strong> — ' + escapeHtml(result.buySignals.summary) + ' (' + result.buySignals.count + '/' + result.buySignals.total + ')</div>';
    html += '</div>';

    // Strategy analysis
    html += '<div style="flex:1;min-width:0;">';
    html += '<select id="strategySelect" style="width:100%;padding:2px 6px;background:#161b22;border:1px solid #30363d;color:#c9d1d9;border-radius:4px;font-size:12px;margin-bottom:4px;">';
    for (let i = 0; i < (this._strategies || []).length; i++) {
      const key = this._strategies[i];
      const label = (this._displayNames && this._displayNames[i]) ? this._displayNames[i] : key;
      html += '<option value="' + key + '"' + (key === (this._currentStrategy || 'event_driven') ? ' selected' : '') + '>' + label + '</option>';
    }
    html += '</select>';
    html += '<div id="strategyResult" style="color:#c9d1d9;line-height:1.3;white-space:pre-wrap;font-size:12px;"></div>';
    html += '</div></div>';

    // Indicator table (full width)
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

    const formatMd = (text) => {
      return escapeHtml(text)
        .replace(/^### (.+)$/gm, '<h4 style="color:#58a6ff;margin:4px 0 1px;font-size:13px;">$1</h4>')
        .replace(/^## (.+)$/gm, '<h3 style="color:#ffd700;margin:6px 0 2px;font-size:14px;">$1</h3>')
        .replace(/\*\*(.+?)\*\*/g, '<strong style="color:#ffd700">$1</strong>')
        .replace(/^- (.+)$/gm, '<span style="color:#58a6ff">•</span> $1')
        .replace(/^(\d+)\. (.+)$/gm, '<span style="color:#58a6ff">$1.</span> $2')
        .replace(/\b(1[7-9]\d{8})\b/g, (_, ts) => {
          const d = new Date(parseInt(ts) * 1000 + 8 * 3600 * 1000);
          const pad = n => String(n).padStart(2, '0');
          return d.getUTCFullYear() + '-' + pad(d.getUTCMonth() + 1) + '-' + pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' + pad(d.getUTCMinutes());
        })
        .replace(/\n/g, '<br>');
    };
    const bars = result.bars;
    const runStrategy = (strategy) => {
      document.getElementById('strategyResult').innerHTML = '分析中...';
      const barData = bars.map(b => ({ ts: b.time, o: b.open, cl: b.close, h: b.high, l: b.low, v: b.volume }));
      this.api.post('/api/strategy/analyze', { strategy: strategy, symbol: symbol, bars: barData }).then(resp => {
        document.getElementById('strategyResult').innerHTML = formatMd(resp.analysis || '无分析结果');
      }).catch(e => {
        document.getElementById('strategyResult').innerHTML = '失败: ' + e.message;
      });
    };
    document.getElementById('strategySelect').addEventListener('change', function() {
      const v = document.getElementById('strategySelect').value; this._currentStrategy = v; runStrategy(v);
    });
    runStrategy(this._currentStrategy || 'event_driven');
  }
}
