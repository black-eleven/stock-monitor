class DashboardComponent {
  constructor(api) {
    this.api = api;
    this.indices = [];
    this.topGainers = [];
    this.topLosers = [];
    this.topSignals = [];
    this.recentAlerts = [];
    this.quoteCache = {};   // symbol -> quote for watchlist re-ranking
    this.symbolNames = {};  // symbol -> name lookup
    // Hardcoded index codes as fallback (must match backend IndexSymbols)
    this._indexCodes = new Set(['SH:000001', 'SZ:399001', 'HK:HSI', 'US:IXIC']);
  }

  _isIndex(symbol) {
    return this._indexCodes.has(symbol);
  }

  async init() {
    await this._fetchData();
    this._setupWsListeners();
  }

  async _fetchData() {
    try {
      const data = await this.api.get('/api/dashboard');
      this.indices = data.indices || [];
      this.topGainers = data.topGainers || [];
      this.topLosers = data.topLosers || [];
      this.topSignals = data.topSignals || [];
      this.recentAlerts = data.recentAlerts || [];

      // Build name lookup from all sources
      for (const item of this.topGainers) {
        this.symbolNames[item.symbol] = item.name;
        this.quoteCache[item.symbol] = { code: item.symbol, price: item.price, yp: item.price / (1 + item.changePct / 100) };
      }
      for (const item of this.topLosers) {
        this.symbolNames[item.symbol] = item.name;
        if (!this.quoteCache[item.symbol]) {
          this.quoteCache[item.symbol] = { code: item.symbol, price: item.price, yp: item.price / (1 + item.changePct / 100) };
        }
      }
      for (const item of this.topSignals) {
        if (item.name && item.name !== item.symbol) {
          this.symbolNames[item.symbol] = item.name;
        }
      }
    } catch (err) {
      console.error('[Dashboard] Failed to fetch:', err);
    }
    this.render();
  }

  _setupWsListeners() {
    this.api.on('quote', (quote) => {
      // Update index cards
      for (const idx of this.indices) {
        if (idx.code === quote.code) {
          idx.price = quote.price;
          if (quote.yp > 0) idx.changePct = ((quote.price - quote.yp) / quote.yp) * 100;
          this._renderIndices();
          return;
        }
      }
      // Skip index quotes even if indices list is not yet populated
      if (this._isIndex(quote.code)) return;
      // Update watchlist quote cache for re-ranking
      this.quoteCache[quote.code] = quote;
      this._reRankMovers();
    });

    this.api.on('snapshot', (quotes) => {
      for (const q of quotes) {
        if (this._isIndex(q.code)) continue;
        this.quoteCache[q.code] = q;
      }
      this._reRankMovers();
    });

    this.api.on('alert', (alertEvent) => {
      this.recentAlerts.unshift(alertEvent);
      if (this.recentAlerts.length > 3) this.recentAlerts.pop();
      this._renderAlerts();
    });
  }

  // Public: register name for a symbol (called by app.js after watchlist loads)
  registerNames(watchlist) {
    for (const w of watchlist) {
      this.symbolNames[w.symbol] = w.name;
    }
    // Re-render with names
    this.render();
  }

  _resolveName(symbol) {
    return this.symbolNames[symbol] || shortCode(symbol);
  }

  _reRankMovers() {
    const entries = [];
    for (const [symbol, q] of Object.entries(this.quoteCache)) {
      if (this._isIndex(symbol)) continue;
      if (!q.price || !q.yp || q.yp === 0) continue;
      const changePct = ((q.price - q.yp) / q.yp) * 100;
      entries.push({ symbol, price: q.price, changePct });
    }
    entries.sort((a, b) => b.changePct - a.changePct);

    this.topGainers = entries.slice(0, 3);
    this.topLosers = entries.slice(-3).reverse();
    this._renderMovers();
  }

  render() {
    this._renderIndices();
    this._renderMovers();
    this._renderSignals();
    this._renderAlerts();
  }

  _renderIndices() {
    const el = document.getElementById('indexCards');
    if (!el) return;
    el.innerHTML = this.indices.map(idx => {
      const dir = idx.changePct >= 0 ? 'up' : 'down';
      const sign = idx.changePct >= 0 ? '+' : '';
      const priceStr = idx.price ? idx.price.toFixed(2) : '--';
      const pctStr = idx.price ? `${sign}${idx.changePct.toFixed(2)}%` : '--';
      return `<div class="index-card">
        <div class="index-card-name">${escapeHtml(idx.name)}</div>
        <div class="index-card-price ${dir}">${priceStr}</div>
        <div class="index-card-change ${dir}">${pctStr}</div>
      </div>`;
    }).join('');
  }

  _renderMovers() {
    this._renderMoverList('topGainersList', this.topGainers);
    this._renderMoverList('topLosersList', this.topLosers);
  }

  _renderMoverList(elId, items) {
    const el = document.getElementById(elId);
    if (!el) return;
    if (items.length === 0) {
      el.innerHTML = '<div class="dash-empty">暂无数据</div>';
      return;
    }
    el.innerHTML = items.map(item => {
      const dir = item.changePct >= 0 ? 'up' : 'down';
      const sign = item.changePct >= 0 ? '+' : '';
      const name = item.name || this._resolveName(item.symbol);
      return `<div class="dash-list-item">
        <span class="dash-list-name">${escapeHtml(name)}</span>
        <span class="dash-list-code">${escapeHtml(shortCode(item.symbol))}</span>
        <span class="dash-list-price">${formatPrice(item.price)}</span>
        <span class="dash-list-change ${dir}">${sign}${item.changePct.toFixed(2)}%</span>
      </div>`;
    }).join('');
  }

  _renderSignals() {
    const el = document.getElementById('topSignalsList');
    if (!el) return;
    if (this.topSignals.length === 0) {
      el.innerHTML = '<div class="dash-empty">暂无信号数据</div>';
      return;
    }
    el.innerHTML = this.topSignals.map(s => {
      const pct = Math.round(s.buyPct);
      let cls = 'dash-signal-none';
      if (pct >= 50) cls = 'dash-signal-strong';
      else if (pct >= 25) cls = 'dash-signal-watch';
      const name = s.name || this._resolveName(s.symbol);
      return `<div class="dash-list-item">
        <span class="dash-list-name">${escapeHtml(name)}</span>
        <span class="dash-list-code">${escapeHtml(shortCode(s.symbol))}</span>
        <span class="dash-signal-badge ${cls}">买入 ${pct}%</span>
      </div>`;
    }).join('');
  }

  _renderAlerts() {
    const el = document.getElementById('recentAlertsList');
    if (!el) return;
    if (this.recentAlerts.length === 0) {
      el.innerHTML = '<div class="dash-empty">暂无预警触发</div>';
      return;
    }
    el.innerHTML = this.recentAlerts.map(a => {
      const time = a.triggeredAt ? new Date(a.triggeredAt).toLocaleTimeString('zh-HK', { hour: '2-digit', minute: '2-digit' }) : '';
      const msg = a.message || `${a.type === 'above' ? '涨破' : a.type === 'below' ? '跌破' : '触发'} ${a.value || ''}`;
      const name = a.name || this._resolveName(a.symbol);
      return `<div class="dash-list-item">
        <span class="dash-list-name">${escapeHtml(name)}</span>
        <span class="dash-list-code">${escapeHtml(shortCode(a.symbol))}</span>
        <span class="dash-alert-msg">${escapeHtml(msg)}</span>
        <span class="dash-alert-time">${time}</span>
      </div>`;
    }).join('');
  }
}
