class AnalysisComponent {
  constructor(api) {
    this.api = api;
    this.watchlist = [];
    this.results = new Map();
    this._currentMode = 'sell';
    this._sortMode = 'score';
    this._exchangeFilter = 'ALL';
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
