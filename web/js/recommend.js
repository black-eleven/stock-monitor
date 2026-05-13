class RecommendComponent {
  constructor(api, onAddToWatchlist) {
    this.api = api;
    this.onAddToWatchlist = onAddToWatchlist;
    this.signals = new Map(); // symbol -> { buySignals, sellSignals }
  }

  async search(industry) {
    try {
      const resp = await this.api.recommend(industry);
      const recs = resp.recommendations || [];
      if (recs.length > 0) {
        await this.analyzeSignals(recs);
      }
      return recs;
    } catch (err) {
      throw new Error('获取推荐失败: ' + err.message);
    }
  }

  async analyzeSignals(recs) {
    this.signals.clear();
    const promises = recs.map(async (r) => {
      try {
        const data = await this.api.getKline(r.symbol, '1d', 100);
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
        const buySignals = evaluateBuySignals(bars, { ma5, ma20, ma60 }, rsi, macd);
        const sellSignals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);
        this.signals.set(r.symbol, { buySignals, sellSignals });
      } catch (_) {}
    });
    await Promise.all(promises);
  }

  renderResults(recs) {
    const container = document.getElementById('recommendResults');
    if (!recs || recs.length === 0) {
      container.innerHTML = '<div class="empty-state">未找到相关推荐，换个关键词试试</div>';
      return;
    }

    container.innerHTML = recs.map(r => {
      const changeDir = r.changePercent >= 0 ? 'up' : 'down';
      const sign = r.changePercent >= 0 ? '+' : '';
      const hasPrice = r.price > 0;
      const highlightsHtml = (r.highlights || []).slice(0, 2).map(h =>
        `<span class="rec-highlight-tag">${escapeHtml(h)}</span>`
      ).join('');

      const signalRow = this._buildSignalRow(r);

      return `<div class="recommend-card" data-symbol="${escapeHtml(r.symbol)}" data-name="${escapeHtml(r.name)}">
        <div class="rec-main">
          <div class="rec-rank">#${r.rank}</div>
          <div class="rec-info">
            <div class="rec-symbol">${escapeHtml(r.symbol)}</div>
            ${escapeHtml(r.name) !== escapeHtml(r.symbol) ? `<div class="rec-name-sub">${escapeHtml(r.name)}</div>` : ''}
            ${highlightsHtml ? `<div class="rec-highlights">${highlightsHtml}</div>` : ''}
          </div>
          ${hasPrice ? `<div class="rec-price-col">
            <span class="rec-price ${changeDir}">${formatPrice(r.price)}</span>
            <span class="rec-change ${changeDir}">${sign}${r.changePercent.toFixed(2)}%</span>
          </div>` : ''}
          <button class="btn btn-primary btn-sm rec-add-btn" data-symbol="${escapeHtml(r.symbol)}" data-name="${escapeHtml(r.name)}">+ 自选</button>
        </div>
        <div class="rec-meta">
          <span>&#x1F4F0; ${r.newsCount} 篇相关新闻</span>
          <span>&#x2B50; 综合评分 ${(r.score * 100).toFixed(0)}</span>
        </div>
        ${signalRow}
      </div>`;
    }).join('');

    // Bind add-to-watchlist buttons
    container.querySelectorAll('.rec-add-btn').forEach(btn => {
      btn.addEventListener('click', async () => {
        const symbol = btn.dataset.symbol;
        const name = btn.dataset.name;
        btn.disabled = true;
        btn.textContent = '...';
        try {
          if (this.onAddToWatchlist) await this.onAddToWatchlist(symbol, name);
          btn.textContent = '✓ 已添加';
          btn.classList.remove('btn-primary');
          btn.classList.add('btn-success');
        } catch (err) {
          btn.disabled = false;
          btn.textContent = '+ 自选';
          alert('添加失败: ' + err.message);
        }
      });
    });

    // Bind signal detail clicks
    const self = this;
    container.querySelectorAll('.rec-signal-row').forEach(row => {
      row.addEventListener('click', function() {
        const symbol = this.dataset.symbol;
        self._showDetailModal(symbol);
      });
    });
  }

  _buildSignalRow(r) {
    const sig = this.signals.get(r.symbol);
    if (!sig) {
      return '<div class="rec-signal-row rec-signal-loading" data-symbol="' + escapeHtml(r.symbol) + '">&#x1F4CA; 加载技术信号...</div>';
    }

    const buyPct = Math.round((sig.buySignals.score / sig.buySignals.maxScore) * 100);
    const sellPct = Math.round((sig.sellSignals.score / sig.sellSignals.maxScore) * 100);

    const hasBuy = buyPct >= 25;
    const hasSell = sellPct >= 25;

    let cls, text;
    if (hasBuy && buyPct >= 50) {
      cls = 'rec-signal-strong-buy';
      text = `强烈买入 ${buyPct}% · ${sig.buySignals.count}信号`;
    } else if (hasSell && sellPct >= 50) {
      cls = 'rec-signal-strong-sell';
      text = `强烈卖出 ${sellPct}% · ${sig.sellSignals.count}信号`;
    } else if (hasBuy) {
      cls = 'rec-signal-watch';
      text = `值得关注 ${buyPct}% · ${sig.buySignals.count}信号`;
    } else if (hasSell) {
      cls = 'rec-signal-watch';
      text = `偏弱 ${sellPct}% · ${sig.sellSignals.count}信号`;
    } else {
      cls = 'rec-signal-none';
      text = '暂无明确信号';
    }

    return `<div class="rec-signal-row ${cls}" data-symbol="${escapeHtml(r.symbol)}">&#x1F4CA; ${text} <span style="margin-left:auto;">&#x203A;</span></div>`;
  }

  _showDetailModal(symbol) {
    const sig = this.signals.get(symbol);
    if (!sig) return;

    const recEl = document.querySelector(`.recommend-card[data-symbol="${CSS.escape(symbol)}"]`);
    const name = recEl ? recEl.dataset.name : shortCode(symbol);

    const buyPct = Math.round((sig.buySignals.score / sig.buySignals.maxScore) * 100);
    const sellPct = Math.round((sig.sellSignals.score / sig.sellSignals.maxScore) * 100);

    const buyColor = buyPct >= 50 ? '#3fb950' : buyPct >= 25 ? '#d29922' : '#8b949e';
    const sellColor = sellPct >= 50 ? '#f85149' : sellPct >= 25 ? '#d29922' : '#8b949e';

    let html = '<div class="rec-detail-overlay" id="recDetailOverlay">';
    html += '<div class="rec-detail-modal">';
    html += '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;">';
    html += '<h3 style="margin:0;">' + escapeHtml(name) + ' <span style="font-weight:normal;font-size:14px;color:#8b949e;">' + escapeHtml(shortCode(symbol)) + '</span></h3>';
    html += '<button id="recDetailClose" style="background:none;border:none;color:#8b949e;font-size:24px;cursor:pointer;">&times;</button>';
    html += '</div>';

    // Buy signals table
    html += '<h4 style="color:' + buyColor + ';margin:12px 0 4px;">买入信号分析 <span style="font-size:12px;background:' + buyColor + '22;padding:2px 8px;border-radius:8px;">' + buyPct + '% · ' + sig.buySignals.summary + '</span></h4>';
    html += '<div style="color:#8b949e;font-size:12px;margin-bottom:8px;">' + sig.buySignals.count + ' / ' + sig.buySignals.total + ' 个信号触发</div>';
    html += '<table class="data-table" style="margin-bottom:16px;"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';
    for (const s of sig.buySignals.signals) {
      let icon, label, color;
      if (!s.triggered) { icon = '&#x26AA;'; label = '未触发'; color = '#8b949e'; }
      else if (s.status === 'danger') { icon = '&#x1F7E2;'; label = '推荐'; color = '#3fb950'; }
      else { icon = '&#x1F7E1;'; label = '关注'; color = '#d29922'; }
      html += '<tr><td>' + escapeHtml(s.name) + '</td><td style="color:' + color + ';">' + icon + ' ' + label + '</td><td>' + escapeHtml(s.value || '--') + '</td></tr>';
    }
    html += '</tbody></table>';

    // Sell signals table
    html += '<h4 style="color:' + sellColor + ';margin:12px 0 4px;">卖出信号分析 <span style="font-size:12px;background:' + sellColor + '22;padding:2px 8px;border-radius:8px;">' + sellPct + '% · ' + sig.sellSignals.summary + '</span></h4>';
    html += '<div style="color:#8b949e;font-size:12px;margin-bottom:8px;">' + sig.sellSignals.count + ' / ' + sig.sellSignals.total + ' 个信号触发</div>';
    html += '<table class="data-table"><thead><tr><th>指标</th><th>状态</th><th>数值</th></tr></thead><tbody>';
    for (const s of sig.sellSignals.signals) {
      let icon, label, color;
      if (!s.triggered) { icon = '&#x1F7E2;'; label = '正常'; color = '#3fb950'; }
      else if (s.status === 'danger') { icon = '&#x1F534;'; label = '危险'; color = '#f85149'; }
      else { icon = '&#x1F7E1;'; label = '警告'; color = '#d29922'; }
      html += '<tr><td>' + escapeHtml(s.name) + '</td><td style="color:' + color + ';">' + icon + ' ' + label + '</td><td>' + escapeHtml(s.value || '--') + '</td></tr>';
    }
    html += '</tbody></table>';

    html += '</div></div>';

    // Remove existing modal if any
    const existing = document.getElementById('recDetailOverlay');
    if (existing) existing.remove();

    document.body.insertAdjacentHTML('beforeend', html);

    document.getElementById('recDetailClose').addEventListener('click', () => {
      document.getElementById('recDetailOverlay').remove();
    });
    document.getElementById('recDetailOverlay').addEventListener('click', (e) => {
      if (e.target.id === 'recDetailOverlay') document.getElementById('recDetailOverlay').remove();
    });
  }
}
