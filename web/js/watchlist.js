class WatchlistComponent {
  constructor(api, onSelectStock, onWatchlistChange) {
    this.api = api;
    this.onSelectStock = onSelectStock;
    this.onWatchlistChange = onWatchlistChange;
    this.watchlist = [];
    this.quotes = {};        // code -> quote
    this.selectedSymbol = null;
    this.signalProvider = null;
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

  selectStock(symbol) {
    this.selectedSymbol = symbol;
    this.renderTabs();
    const quote = this.quotes[symbol];
    this.renderDetail(symbol, quote);
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

      // Signal badge
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

    // Delete button always shown
    el.insertAdjacentHTML('beforeend', '<button class="btn btn-danger btn-sm" style="margin-top:12px" id="removeStockBtn">删除自选</button>');

    // Signal summary from analysis component (if available)
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

    document.getElementById('removeStockBtn')?.addEventListener('click', async () => {
      const code = symbol;
      await this.api.removeWatchlist(code);
      this.watchlist = this.watchlist.filter(w => w.symbol !== code);
      if (this.onWatchlistChange) this.onWatchlistChange(this.watchlist);
      if (this.watchlist.length > 0) {
        this.selectStock(this.watchlist[0].symbol);
      } else {
        this.selectedSymbol = null;
        this.renderTabs();
        el.innerHTML = '<div class="empty-state">点击上方 + 添加自选股</div>';
      }
    });
  }

  _showAddDialog() {
    const symbol = prompt('输入股票代码（如 HK:700 / SH:600519 / US:AAPL）:');
    if (!symbol) return;
    const name = prompt('输入股票名称（如 腾讯控股）:');
    if (!name) return;
    this.api.addWatchlist(symbol.toUpperCase(), name).then(item => {
      this.watchlist.push(item);
      if (this.onWatchlistChange) this.onWatchlistChange(this.watchlist);
      this.renderTabs();
      this.selectStock(item.symbol);
    }).catch(err => {
      alert('添加失败: ' + err.message);
    });
  }
}
