class HoldingsComponent {
  constructor(api) {
    this.api = api;
    this.holdings = [];
    this.quotes = {}; // code -> price
  }

  async init() {
    this.holdings = await this.api.getHoldings();
    // Fetch current prices for all holdings (REST fallback for when WS isn't pushing)
    if (this.holdings.length > 0) {
      const symbols = this.holdings.map(h => h.symbol);
      try {
        const quotes = await this.api.get('/api/quote/batch?symbols=' + symbols.join(','));
        for (const [code, q] of Object.entries(quotes)) {
          if (q && q.price) this.quotes[code] = q.price;
        }
      } catch (err) {
        console.error('Failed to fetch initial quotes:', err);
      }
    }
    this.render();
  }

  updateQuote(quote) {
    this.quotes[quote.code] = quote.price;
    this.render();
  }

  render() {
    this._renderSummary();
    this._renderTable();
  }

  _renderSummary() {
    const el = document.getElementById('holdingsSummary');
    if (this.holdings.length === 0) {
      el.innerHTML = '';
      return;
    }

    let totalCost = 0, totalMarket = 0;
    for (const h of this.holdings) {
      const price = this.quotes[h.symbol];
      if (price) {
        totalCost += h.shares * h.avgCost;
        totalMarket += h.shares * price;
      }
    }
    const totalPnl = totalMarket - totalCost;
    const totalPnlPct = totalCost > 0 ? (totalPnl / totalCost) * 100 : 0;
    const dir = totalPnl >= 0 ? 'up' : 'down';

    el.innerHTML = `<div class="stat"><div class="label">总投入</div><div class="value">${formatPrice(totalCost)}</div></div>
      <div class="stat"><div class="label">总市值</div><div class="value">${formatPrice(totalMarket)}</div></div>
      <div class="stat"><div class="label">总盈亏</div><div class="value ${dir}">${totalPnl >= 0 ? '+' : ''}${formatPrice(totalPnl)}</div></div>
      <div class="stat"><div class="label">总盈亏率</div><div class="value ${dir}">${totalPnlPct >= 0 ? '+' : ''}${totalPnlPct.toFixed(2)}%</div></div>`;
  }

  _renderTable() {
    const body = document.getElementById('holdingsBody');
    if (this.holdings.length === 0) {
      body.innerHTML = '<tr><td colspan="7"><div class="empty-state">暂无持仓记录，点击上方添加</div></td></tr>';
      return;
    }

    body.innerHTML = this.holdings.map(h => {
      const price = this.quotes[h.symbol];
      const hasPrice = price != null;
      const marketValue = hasPrice ? h.shares * price : 0;
      const costValue = h.shares * h.avgCost;
      const pnl = hasPrice ? marketValue - costValue : 0;
      const pnlPct = hasPrice ? ((price - h.avgCost) / h.avgCost) * 100 : 0;
      const dir = pnl >= 0 ? 'up' : 'down';
      const priceDisplay = hasPrice ? formatPrice(price) : '--';
      const pnlDisplay = hasPrice ? (pnl >= 0 ? '+' : '') + formatPrice(pnl) : '--';
      const pnlPctDisplay = hasPrice ? (pnlPct >= 0 ? '+' : '') + pnlPct.toFixed(2) + '%' : '--';

      return `<tr>
        <td>${escapeHtml(h.name)}<br><small style="color:#8b949e">${escapeHtml(shortCode(h.symbol))}</small></td>
        <td>${h.shares}</td>
        <td>${formatPrice(h.avgCost)}</td>
        <td class="${hasPrice ? dir : ''}">${priceDisplay}</td>
        <td class="${hasPrice ? dir : ''}">${pnlDisplay}</td>
        <td class="${hasPrice ? dir : ''}">${pnlPctDisplay}</td>
        <td><button class="btn btn-danger btn-sm" data-symbol="${escapeHtml(h.symbol)}">删除</button></td>
      </tr>`;
    }).join('');

    body.querySelectorAll('[data-symbol]').forEach(btn => {
      btn.addEventListener('click', async () => {
        if (confirm('确定删除该持仓？')) {
          await this.api.deleteHolding(btn.dataset.symbol);
          this.holdings = this.holdings.filter(h => h.symbol !== btn.dataset.symbol);
          this.render();
        }
      });
    });
  }
}
