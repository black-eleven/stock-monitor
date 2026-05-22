class HoldingsComponent {
  constructor(api, alertsComp) {
    this.api = api;
    this.alertsComp = alertsComp;
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
    // Init alerts
    if (this.alertsComp) {
      await this.alertsComp.init();
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
      body.innerHTML = '<tr><td colspan="8"><div class="empty-state">暂无持仓记录，点击上方添加</div></td></tr>';
      return;
    }

    const typeLabels = { above: '涨破', below: '跌破', change_pct: '涨跌幅≥' };
    const self = this;

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

      // Render alerts for this symbol
      const symbolAlerts = this.alertsComp ? this.alertsComp.alerts.filter(a => a.symbol === h.symbol) : [];
      let alertsHtml = '';
      if (symbolAlerts.length > 0) {
        alertsHtml = symbolAlerts.map(a => {
          const enabledColor = a.enabled ? '#3fb950' : '#484f58';
          const label = typeLabels[a.type] || a.type;
          return `<div style="display:flex;align-items:center;gap:4px;margin-bottom:2px;font-size:12px;">
            <span style="color:${enabledColor};cursor:pointer;" data-toggle-alert="${a.id}" title="点击${a.enabled ? '禁用' : '启用'}">${label} ${a.value}</span>
            <span data-del-alert="${a.id}" style="color:#484f58;cursor:pointer;font-size:10px;" title="删除">✕</span>
          </div>`;
        }).join('');
      }
      alertsHtml += `<button class="btn btn-sm" data-add-alert="${escapeHtml(h.symbol)}" style="font-size:11px;padding:2px 6px;margin-top:2px;" title="添加预警">+</button>`;

      return `<tr>
        <td>${escapeHtml(h.name)}<br><small style="color:#8b949e">${escapeHtml(shortCode(h.symbol))}</small></td>
        <td>${h.shares}</td>
        <td>${formatPrice(h.avgCost)}</td>
        <td class="${hasPrice ? dir : ''}">${priceDisplay}</td>
        <td class="${hasPrice ? dir : ''}">${pnlDisplay}</td>
        <td class="${hasPrice ? dir : ''}">${pnlPctDisplay}</td>
        <td style="min-width:100px;">${alertsHtml}</td>
        <td><button class="btn btn-danger btn-sm" data-symbol="${escapeHtml(h.symbol)}">删除</button></td>
      </tr>`;
    }).join('');

    // Delete holding
    body.querySelectorAll('[data-symbol]').forEach(btn => {
      btn.addEventListener('click', async () => {
        if (confirm('确定删除该持仓？')) {
          await this.api.deleteHolding(btn.dataset.symbol);
          this.holdings = this.holdings.filter(h => h.symbol !== btn.dataset.symbol);
          this.render();
        }
      });
    });

    // Toggle alert enabled/disabled
    body.querySelectorAll('[data-toggle-alert]').forEach(el => {
      el.addEventListener('click', async () => {
        const id = parseInt(el.dataset.toggleAlert, 10);
        const alert = self.alertsComp.alerts.find(a => a.id === id);
        if (!alert) return;
        await self.api.updateAlert(id, { enabled: !alert.enabled });
        alert.enabled = !alert.enabled;
        self.render();
      });
    });

    // Delete alert
    body.querySelectorAll('[data-del-alert]').forEach(el => {
      el.addEventListener('click', async () => {
        const id = parseInt(el.dataset.delAlert, 10);
        self.alertsComp.alerts = self.alertsComp.alerts.filter(a => a.id !== id);
        await self.api.deleteAlert(id);
        self.render();
      });
    });

    // Add alert for this symbol (opens modal pre-filled)
    body.querySelectorAll('[data-add-alert]').forEach(btn => {
      btn.addEventListener('click', () => {
        self._openAlertModal(btn.dataset.addAlert);
      });
    });
  }

  _openAlertModal(symbol) {
    const form = document.getElementById('alertForm');
    form.reset();
    const symbolInput = form.querySelector('input[name="symbol"]');
    symbolInput.value = symbol;
    symbolInput.readOnly = true;
    symbolInput.style.background = '#21262d';
    showModal('alertModal');
  }

  refreshAlerts() {
    // Re-render to show updated alert badges in table
    this._renderTable();
  }
}
