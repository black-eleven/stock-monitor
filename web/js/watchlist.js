class WatchlistComponent {
  constructor(api, klineComp, analysisComp) {
    this.api = api;
    this.klineComp = klineComp;
    this.analysisComp = analysisComp;
    this.watchlist = [];
    this.quotes = {};        // code -> quote
    this.selectedSymbol = null;
    this.signalProvider = null;
    this.onWatchlistChange = null;  // external listener (e.g. recommend component)
  }

  async init() {
    this.watchlist = await this.api.getWatchlist();
    document.getElementById('addWatchlistBtn').addEventListener('click', () => this._showAddDialog());
    this.renderTabs();

    // Fetch initial quotes from REST API (fallback for when WS snapshot hasn't arrived yet)
    if (this.watchlist.length > 0) {
      try {
        const symbols = this.watchlist.map(w => w.symbol);
        const quotes = await this.api.get('/api/quote/batch?symbols=' + symbols.join(','));
        for (const [code, q] of Object.entries(quotes)) {
          if (q && q.code) this.quotes[code] = q;
        }
      } catch (err) {
        console.error('Failed to fetch initial quotes:', err);
      }
    }

    if (this.watchlist.length > 0) {
      this.selectStock(this.watchlist[0].symbol);
    }
  }

  // Called when new quote arrives from WS
  updateQuote(quote) {
    this.quotes[quote.code] = quote;
    this.renderTabs();
    if (this.selectedSymbol === quote.code) {
      this.renderDetail(this.selectedSymbol, quote);
    }
  }

  selectStock(symbol, updateUrl = true) {
    this.selectedSymbol = symbol;
    this.renderTabs();
    const quote = this.quotes[symbol];
    this.renderDetail(symbol, quote);

    // Update URL for deep linking
    if (updateUrl) {
      const url = new URL(location);
      url.searchParams.set('stock', symbol);
      history.replaceState(null, '', url);
    }

    // Show kline chart
    const klineIntervals = document.getElementById('watchlistKlineIntervals');
    const chartContainer = document.getElementById('watchlistChartContainer');
    const analysisContainer = document.getElementById('watchlistAnalysis');

    if (klineIntervals) klineIntervals.style.display = 'flex';
    if (chartContainer) chartContainer.style.display = 'block';
    if (analysisContainer) analysisContainer.style.display = 'block';

    if (this.klineComp) {
      this.klineComp.setSymbol(symbol);
      setTimeout(() => this.klineComp.resize(), 300);
    }

    if (this.analysisComp) {
      this.analysisComp.renderStockDetail(symbol, analysisContainer);
    }
  }

  _hidePanels() {
    const klineIntervals = document.getElementById('watchlistKlineIntervals');
    const chartContainer = document.getElementById('watchlistChartContainer');
    const analysisContainer = document.getElementById('watchlistAnalysis');
    if (klineIntervals) klineIntervals.style.display = 'none';
    if (chartContainer) chartContainer.style.display = 'none';
    if (analysisContainer) analysisContainer.style.display = 'none';
  }

  renderTabs() {
    const container = document.getElementById('stockTabs');
    if (this.watchlist.length === 0) {
      container.innerHTML = '<div class="empty-state">点击右上角 + 添加自选股</div>';
      return;
    }

    container.innerHTML = this.watchlist.map(w => {
      const q = this.quotes[w.symbol];
      const changePct = q ? calcChangePct(q.price, q.yp) : 0;
      const dir = q ? changeDir(q.price, q.yp) : '';
      const changeText = q ? (changePct >= 0 ? '+' : '') + changePct.toFixed(2) + '%' : '--';
      const active = w.symbol === this.selectedSymbol ? 'active' : '';

      let signalBadge = '';
      if (this.signalProvider) {
        const sig = this.signalProvider(w.symbol);
        if (sig) {
          if (sig.buyPct >= 50) {
            signalBadge = '<span style="background:#1a7f37;color:#fff;font-size:10px;padding:1px 4px;border-radius:3px;margin-left:4px;">B' + sig.buyPct + '</span>';
          } else if (sig.sellPct >= 50) {
            signalBadge = '<span style="background:#da3633;color:#fff;font-size:10px;padding:1px 4px;border-radius:3px;margin-left:4px;">S' + sig.sellPct + '</span>';
          } else if (sig.buyPct >= 25) {
            signalBadge = '<span style="background:#9e6a03;color:#fff;font-size:10px;padding:1px 4px;border-radius:3px;margin-left:4px;">B' + sig.buyPct + '</span>';
          } else if (sig.sellPct >= 25) {
            signalBadge = '<span style="background:#9e6a03;color:#fff;font-size:10px;padding:1px 4px;border-radius:3px;margin-left:4px;">S' + sig.sellPct + '</span>';
          }
        }
      }

      return `<div class="stock-tab ${active}" data-symbol="${escapeHtml(w.symbol)}">
        <span class="name">${escapeHtml(w.name)}${signalBadge}</span>
        <span class="symbol">${escapeHtml(shortCode(w.symbol))}</span>
        <span class="change ${dir}">${changeText}</span>
      </div>`;
    }).join('');

    container.querySelectorAll('.stock-tab').forEach(el => {
      el.addEventListener('click', () => this.selectStock(el.dataset.symbol));
    });
  }

  renderDetail(symbol, quote) {
    const el = document.getElementById('stockDetail');

    if (quote) {
      const dir = changeDir(quote.price, quote.yp);
      const changeStr = formatChange(quote.price, quote.yp);
      const changePct = calcChangePct(quote.price, quote.yp);

      el.innerHTML = `<div class="price-row">
        <div>
          <div style="font-size:14px;color:#8b949e">${escapeHtml(shortCode(quote.code))}</div>
          <span class="current-price ${dir}">${formatPrice(quote.price)}</span>
          <span class="change-value ${dir}">${escapeHtml(changeStr)}</span>
        </div>
      </div>
      <div class="info-grid">
        <div class="info-item">今开 <span class="value">${formatPrice(quote.open)}</span></div>
        <div class="info-item">最高 <span class="value">${formatPrice(quote.high)}</span></div>
        <div class="info-item">最低 <span class="value">${formatPrice(quote.low)}</span></div>
        <div class="info-item">昨收 <span class="value">${formatPrice(quote.yp)}</span></div>
        <div class="info-item">成交量 <span class="value">${formatVolume(quote.volume)}</span></div>
        <div class="info-item">成交额 <span class="value">${formatTurnover(quote.turnover)}</span></div>
      </div>`;
    } else {
      const name = this.watchlist.find(w => w.symbol === symbol)?.name || shortCode(symbol);
      el.innerHTML = `<div class="price-row">
        <div>
          <div style="font-size:14px;color:#8b949e">${escapeHtml(shortCode(symbol))}</div>
          <div style="font-size:24px;font-weight:700;color:#8b949e">${escapeHtml(name)}</div>
          <div style="color:#8b949e;margin-top:4px">暂无实时数据（休市或连接中）</div>
        </div>
      </div>`;
    }

    // Delete button + signal badge
    el.insertAdjacentHTML('beforeend', '<button class="btn btn-danger btn-sm" style="margin-top:12px" id="removeStockBtn">删除自选</button>');

    if (this.signalProvider) {
      const signals = this.signalProvider(symbol);
      if (signals) {
        var signalHtml = '';
        if (signals.buyPct >= 50) {
          signalHtml = '<span style="background:#1a7f37;color:#fff;padding:4px 10px;border-radius:12px;font-size:12px;">买入 ' + signals.buyPct + '% &middot; ' + signals.buyCount + '信号</span>';
        } else if (signals.sellPct >= 50) {
          signalHtml = '<span style="background:#da3633;color:#fff;padding:4px 10px;border-radius:12px;font-size:12px;">卖出 ' + signals.sellPct + '% &middot; ' + signals.sellCount + '信号</span>';
        } else if (signals.buyPct >= 25) {
          signalHtml = '<span style="background:#9e6a03;color:#fff;padding:4px 10px;border-radius:12px;font-size:12px;">关注买入 ' + signals.buyPct + '%</span>';
        } else if (signals.sellPct >= 25) {
          signalHtml = '<span style="background:#9e6a03;color:#fff;padding:4px 10px;border-radius:12px;font-size:12px;">偏弱 ' + signals.sellPct + '%</span>';
        } else {
          signalHtml = '<span style="color:#8b949e;font-size:12px;">暂无明确信号</span>';
        }
        el.insertAdjacentHTML('beforeend', '<div style="margin-top:12px;display:flex;align-items:center;gap:8px;">' + signalHtml + '</div>');
      }
    }

    // Load fundamentals asynchronously
    this.renderFundamentals(symbol);

    document.getElementById('removeStockBtn')?.addEventListener('click', async () => {
      const code = symbol;
      await this.api.removeWatchlist(code);
      this.watchlist = this.watchlist.filter(w => w.symbol !== code);
      this._notifyWatchlistChange();
      if (this.watchlist.length > 0) {
        this.selectStock(this.watchlist[0].symbol);
      } else {
        this.selectedSymbol = null;
        this.renderTabs();
        el.innerHTML = '<div class="empty-state">点击上方 + 添加自选股</div>';
        this._hidePanels();
      }
    });
  }

  async renderFundamentals(symbol) {
    const el = document.getElementById('stockDetail');
    // Remove previous fundamentals grid if any
    const prev = document.getElementById('fundGrid');
    if (prev) prev.remove();

    var fundEl = document.createElement('div');
    fundEl.id = 'fundGrid';
    fundEl.className = 'info-grid fundamentals-grid';
    fundEl.innerHTML = '<div class="info-item" style="color:#8b949e;">基本面数据加载中...</div>';
    el.appendChild(fundEl);

    try {
      const f = await this.api.getFundamentals(symbol);
      if (!f || (!f.pe && !f.pb && !f.roe && !f.marketCap && !f.industry)) {
        fundEl.innerHTML = '<div class="info-item" style="color:#8b949e;">暂无基本面数据</div>';
        return;
      }
      var items = [];
      if (f.pe) items.push('<div class="info-item">市盈率(PE) <span class="value">' + f.pe.toFixed(2) + '</span></div>');
      if (f.pb) items.push('<div class="info-item">市净率(PB) <span class="value">' + f.pb.toFixed(2) + '</span></div>');
      if (f.roe) items.push('<div class="info-item">ROE <span class="value">' + f.roe.toFixed(2) + '%</span></div>');
      if (f.marketCap) items.push('<div class="info-item">总市值 <span class="value">' + formatMarketCap(f.marketCap) + '</span></div>');
      if (f.navPerShare) items.push('<div class="info-item">每股净资产 <span class="value">' + f.navPerShare.toFixed(2) + '</span></div>');
      if (f.industry) items.push('<div class="info-item">行业 <span class="value">' + escapeHtml(f.industry) + '</span></div>');
      if (f.netProfitGrowth) items.push('<div class="info-item">净利增长率 <span class="value">' + f.netProfitGrowth.toFixed(2) + '%</span></div>');
      if (f.revenueGrowth) items.push('<div class="info-item">营收增长率 <span class="value">' + f.revenueGrowth.toFixed(2) + '%</span></div>');
      fundEl.innerHTML = items.join('');
    } catch (_) {
      fundEl.innerHTML = '<div class="info-item" style="color:#8b949e;">基本面数据获取失败</div>';
    }
  }

  _notifyWatchlistChange() {
    if (this.klineComp) this.klineComp.updateSymbols(this.watchlist);
    if (this.onWatchlistChange) this.onWatchlistChange(this.watchlist);
  }

  // Prompt user to add a stock (not in watchlist) then select it
  async _promptAddThenSelect(symbol) {
    const name = prompt(`该股票不在自选列表中，输入名称以添加并查看详情（如 腾讯控股）:\n\n${symbol}`);
    if (!name) return;
    try {
      const item = await this.api.addWatchlist(symbol.toUpperCase(), name);
      this.watchlist.push(item);
      this._notifyWatchlistChange();
      this.renderTabs();
      this.selectStock(item.symbol);
    } catch (err) {
      alert('添加失败: ' + err.message);
    }
  }

  _showAddDialog() {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `<div class="modal" style="width:480px;">
      <h3>添加自选股</h3>
      <input id="addSearchInput" placeholder="搜索股票代码或名称..." autofocus>
      <div id="addSearchResults" style="max-height:240px;overflow-y:auto;margin-bottom:8px;"></div>
      <div>已选：<span id="addSelectedInfo" style="color:#8b949e;">--</span></div>
      <div class="modal-actions">
        <button id="addConfirmBtn" class="btn btn-primary" disabled>确认添加</button>
        <button id="addCancelBtn" class="btn">取消</button>
      </div>
    </div>`;
    document.body.appendChild(overlay);

    const searchInput = overlay.querySelector('#addSearchInput');
    const resultsEl = overlay.querySelector('#addSearchResults');
    const selectedInfo = overlay.querySelector('#addSelectedInfo');
    const confirmBtn = overlay.querySelector('#addConfirmBtn');
    let selected = null; // { symbol, name }
    let searchTimer = null;

    const doSearch = async (keyword) => {
      if (!keyword.trim()) { resultsEl.innerHTML = ''; return; }
      try {
        const results = await this.api.get('/api/search?q=' + encodeURIComponent(keyword.trim()));
        if (!results || results.length === 0) {
          resultsEl.innerHTML = '<div style="padding:8px;color:#8b949e;font-size:13px;">无匹配结果</div>';
          return;
        }
        resultsEl.innerHTML = results.map(r => {
          const marketTag = { 'SH:': '沪', 'SZ:': '深', 'HK:': '港', 'US:': '美' }[r.market] || r.market;
          return `<div class="search-result-item" data-code="${escapeHtml(r.code)}" data-name="${escapeHtml(r.name)}" style="padding:8px 10px;cursor:pointer;border-bottom:1px solid #21262d;display:flex;justify-content:space-between;align-items:center;">
            <span><strong style="color:#e6edf3;">${escapeHtml(r.name)}</strong> <small style="color:#8b949e;">${escapeHtml(shortCode(r.code))}</small></span>
            <span style="background:#21262d;color:#8b949e;padding:1px 6px;border-radius:4px;font-size:11px;">${marketTag}</span>
          </div>`;
        }).join('');

        resultsEl.querySelectorAll('.search-result-item').forEach(el => {
          el.addEventListener('click', () => {
            selected = { symbol: el.dataset.code, name: el.dataset.name };
            selectedInfo.innerHTML = `<strong style="color:#e6edf3;">${escapeHtml(el.dataset.name)}</strong> <small style="color:#8b949e;">${escapeHtml(shortCode(el.dataset.code))}</small>`;
            confirmBtn.disabled = false;
            resultsEl.querySelectorAll('.search-result-item').forEach(e => e.style.background = '');
            el.style.background = '#1a2744';
          });
        });
      } catch (_) {
        resultsEl.innerHTML = '<div style="padding:8px;color:#8b949e;font-size:13px;">搜索失败</div>';
      }
    };

    searchInput.addEventListener('input', () => {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(() => doSearch(searchInput.value), 300);
    });
    searchInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !selected) {
        e.preventDefault();
        doSearch(searchInput.value);
      }
    });

    const close = () => overlay.remove();
    overlay.querySelector('#addCancelBtn').addEventListener('click', close);
    overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });

    confirmBtn.addEventListener('click', async () => {
      if (!selected) return;
      try {
        const item = await this.api.addWatchlist(selected.symbol, selected.name);
        this.watchlist.push(item);
        this._notifyWatchlistChange();
        this.renderTabs();
        this.selectStock(item.symbol);
        close();
      } catch (err) {
        alert('添加失败: ' + err.message);
      }
    });

    setTimeout(() => searchInput.focus(), 100);
  }
}
