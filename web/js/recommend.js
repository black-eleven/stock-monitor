class RecommendComponent {
  constructor(api, onAddToWatchlist) {
    this.api = api;
    this.onAddToWatchlist = onAddToWatchlist;
  }

  async search(industry) {
    try {
      const resp = await this.api.recommend(industry);
      return resp.recommendations || [];
    } catch (err) {
      throw new Error('获取推荐失败: ' + err.message);
    }
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

      return `<div class="recommend-card" data-symbol="${escapeHtml(r.symbol)}" data-name="${escapeHtml(r.name)}">
        <div class="rec-main">
          <div class="rec-rank">#${r.rank}</div>
          <div class="rec-info">
            <div class="rec-symbol">${escapeHtml(r.symbol)}</div>
            ${highlightsHtml ? `<div class="rec-highlights">${highlightsHtml}</div>` : ''}
          </div>
          ${hasPrice ? `<div class="rec-price-col">
            <span class="rec-price ${changeDir}">${formatPrice(r.price)}</span>
            <span class="rec-change ${changeDir}">${sign}${r.changePercent.toFixed(2)}%</span>
          </div>` : ''}
          <button class="btn btn-primary btn-sm rec-add-btn" data-symbol="${escapeHtml(r.symbol)}" data-name="${escapeHtml(r.name)}">+ 自选</button>
        </div>
        <div class="rec-meta">
          <span>📰 ${r.newsCount} 篇相关新闻</span>
          <span>⭐ 综合评分 ${(r.score * 100).toFixed(0)}</span>
        </div>
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
  }
}
