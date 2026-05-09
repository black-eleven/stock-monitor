class AnalysisComponent {
  constructor(api) {
    this.api = api;
    this.watchlist = [];
    this.results = new Map(); // symbol -> analysis result
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
    const signals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);

    this.results.set(symbol, { bars, ma5, ma20, ma60, rsi, macd, signals });
  }

  async render() {
    const promises = this.watchlist
      .filter(w => !this.results.has(w.symbol))
      .map(w => this.analyze(w.symbol));
    await Promise.all(promises);
    this._renderList();
  }

  _renderList() {
    const container = document.getElementById('analysisList');

    // Calculate summary statistics
    let totalTriggered = 0;
    let totalPossible = 0;
    for (const [, result] of this.results) {
      totalTriggered += result.signals.count;
      totalPossible += result.signals.total;
    }
    const numResults = this.results.size;
    const avgTriggered = numResults > 0 ? totalTriggered / numResults : 0;

    let stars;
    if (avgTriggered >= 3) {
      stars = '★★★'; // ★★★
    } else if (avgTriggered >= 1) {
      stars = '★★☆'; // ★★☆
    } else {
      stars = '★☆☆'; // ★☆☆
    }

    let html = '<div class="analysis-summary">综合卖出信号: ' + stars + ' (' + totalTriggered + '/' + totalPossible + ' 项触发)</div>';
    html += '<div class="analysis-cards">';

    for (const w of this.watchlist) {
      const result = this.results.get(w.symbol);
      if (!result) continue;

      const count = result.signals.count;
      let cardClass;
      if (count >= 3) {
        cardClass = 'danger';
      } else if (count > 0) {
        cardClass = 'warn';
      } else {
        cardClass = '';
      }

      let icon;
      if (count === 0) {
        icon = '✅'; // ✅
      } else if (count >= 3) {
        icon = '🔴'; // 🔴
      } else {
        icon = '⚠'; // ⚠
      }

      const latestBar = result.bars[result.bars.length - 1];
      const price = formatPrice(latestBar.close);

      html += '<div class="analysis-card' + (cardClass ? ' ' + cardClass : '') + '" data-symbol="' + escapeHtml(w.symbol) + '">';
      html += '<div class="analysis-card-header">';
      html += '<span class="name">' + escapeHtml(w.name) + '</span>';
      html += '<span class="symbol">' + escapeHtml(shortCode(w.symbol)) + '</span>';
      html += '<span class="price">' + escapeHtml(price) + '</span>';
      html += '</div>';
      html += '<div class="analysis-card-signals">' + icon + ' 卖出信号 ' + count + '/' + result.signals.total + '</div>';
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
    const container = document.getElementById('analysisList');
    const result = this.results.get(symbol);
    if (!result) return;

    const w = this.watchlist.find(function(item) { return item.symbol === symbol; });
    const name = w ? w.name : shortCode(symbol);
    const count = result.signals.count;

    let summaryColor;
    if (count === 0) {
      summaryColor = 'green';
    } else if (count >= 3) {
      summaryColor = 'red';
    } else {
      summaryColor = '#ffd700';
    }

    let html = '<button id="analysisBack" style="background:none;border:none;color:#58a6ff;cursor:pointer;font-size:14px;padding:8px 0;">← 返回列表</button>';
    html += '<h3 style="margin:8px 0;">' + escapeHtml(name) + ' (' + escapeHtml(shortCode(symbol)) + ')</h3>';
    html += '<div style="color:' + summaryColor + ';margin:8px 0;font-weight:bold;">' + escapeHtml(result.signals.summary) + ' (' + count + '/' + result.signals.total + ')</div>';

    html += '<table class="data-table"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';

    for (const signal of result.signals.signals) {
      let statusIcon, statusText, color;

      if (!signal.triggered) {
        statusIcon = '🟢'; // 🟢
        statusText = '正常'; // 正常
        color = 'green';
      } else if (signal.status === 'danger') {
        statusIcon = '🔴'; // 🔴
        statusText = '危险'; // 危险
        color = 'red';
      } else {
        statusIcon = '🟡'; // 🟡
        statusText = '警告'; // 警告
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
      this.render();
    }.bind(this));
  }
}
