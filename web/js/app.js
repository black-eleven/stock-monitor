// Global instances
const api = new ApiClient();
let watchlistComp, klineComp, holdingsComp, alertsComp, analysisComp, recommendComp;

// Tab switching
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');

    // Resize chart when switching to kline tab
    if (btn.dataset.tab === 'kline' && klineComp) {
      setTimeout(() => klineComp.resize(), 300);
    }
    if (btn.dataset.tab === 'analysis' && analysisComp) {
      analysisComp.render();
    }
  });
});

// Init all components
async function init() {
  // Auth guard
  if (!auth.isLoggedIn()) {
    window.location.href = '/login.html';
    return;
  }

  // Connection status
  api.on('connected', () => {
    document.getElementById('connStatus').textContent = '已连接';
    document.getElementById('connStatus').className = 'connection-status connected';
  });
  api.on('disconnected', () => {
    document.getElementById('connStatus').textContent = '已断开';
    document.getElementById('connStatus').className = 'connection-status disconnected';
  });

  // Admin link visibility
  if (auth.isAdmin()) {
    const adminLink = document.getElementById('adminLink');
    if (adminLink) adminLink.style.display = 'inline';
  }

  // Connect WebSocket
  api.connectWs();

  // Init components
  klineComp = new KlineComponent(api);
  klineComp.init();

  watchlistComp = new WatchlistComponent(api, (symbol) => {
    // When stock selected in watchlist, update kline tab
    klineComp.setSymbol(symbol);
  }, (watchlist) => {
    // When watchlist changes, update kline dropdown
    klineComp.updateSymbols(watchlist);
  });
  await watchlistComp.init();

  // Recommend component
  recommendComp = new RecommendComponent(api, async (symbol, name) => {
    await api.addWatchlist(symbol, name);
    watchlistComp.watchlist.push({ symbol, name });
    watchlistComp.renderTabs();
    if (watchlistComp.watchlist.length === 1) {
      watchlistComp.selectStock(symbol);
    }
    if (watchlistComp.onWatchlistChange) watchlistComp.onWatchlistChange(watchlistComp.watchlist);
    klineComp.updateSymbols(watchlistComp.watchlist);
  });

  // Watchlist sub-tab switching
  document.querySelectorAll('.subtab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.subtab-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      const target = btn.dataset.subtab;
      document.getElementById('watchlistMy').style.display = target === 'my' ? 'block' : 'none';
      document.getElementById('watchlistRecommend').style.display = target === 'recommend' ? 'block' : 'none';
      document.getElementById('addWatchlistBtn').style.display = target === 'my' ? '' : 'none';
      // Resize chart if switching back to my tab
      if (target === 'my' && watchlistComp && watchlistComp.selectedSymbol) {
        setTimeout(() => {
          window.dispatchEvent(new Event('resize'));
        }, 100);
      }
    });
  });

  // Recommend search button
  document.getElementById('recommendSearchBtn').addEventListener('click', async () => {
    const input = document.getElementById('recommendIndustry');
    const industry = input.value.trim();
    if (!industry) return;

    const statusEl = document.getElementById('recommendStatus');
    const resultsEl = document.getElementById('recommendResults');
    statusEl.innerHTML = '<div style="padding:20px;text-align:center;color:#8b949e;">搜索中...</div>';
    resultsEl.innerHTML = '';

    try {
      const recs = await recommendComp.search(industry);
      statusEl.style.display = 'none';
      recommendComp.renderResults(recs);
    } catch (err) {
      statusEl.innerHTML = `<div class="empty-state">${escapeHtml(err.message)}</div>`;
      statusEl.style.display = 'block';
    }
  });

  // Recommend search on Enter key
  document.getElementById('recommendIndustry').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') document.getElementById('recommendSearchBtn').click();
  });

  holdingsComp = new HoldingsComponent(api);
  await holdingsComp.init();

  alertsComp = new AlertsComponent(api);
  await alertsComp.init();

  analysisComp = new AnalysisComponent(api);
  await analysisComp.init();

  // Wire analysis signal data into watchlist detail view
  watchlistComp.signalProvider = (symbol) => {
    const result = analysisComp.results.get(symbol);
    if (!result) return null;
    return {
      buyPct: Math.round((result.buySignals.score / result.buySignals.maxScore) * 100),
      sellPct: Math.round((result.signals.score / result.signals.maxScore) * 100),
      buyCount: result.buySignals.count,
      sellCount: result.signals.count,
    };
  };

  // Refresh watchlist tabs and detail to show signal badges after analysis completes
  watchlistComp.renderTabs();
  if (watchlistComp.selectedSymbol) {
    const quote = watchlistComp.quotes[watchlistComp.selectedSymbol];
    watchlistComp.renderDetail(watchlistComp.selectedSymbol, quote);
  }

  // Update kline symbols from watchlist
  klineComp.updateSymbols(watchlistComp.watchlist);

  // Handle initial snapshot
  api.on('snapshot', (quotes) => {
    for (const quote of quotes) {
      watchlistComp.updateQuote(quote);
      analysisComp.updateQuote(quote);
      holdingsComp.updateQuote(quote);
    }
  });

  // Handle real-time quotes
  api.on('quote', (quote) => {
    watchlistComp.updateQuote(quote);
      analysisComp.updateQuote(quote);
    holdingsComp.updateQuote(quote);
  });

  // Handle alerts
  api.on('alert', (alert) => {
    alertsComp.addLog(alert);
    showToast(`⚠ ${escapeHtml(alert.message)}`, 'alert');
  });

  // Modal handlers
  document.getElementById('addHoldingBtn').addEventListener('click', () => showModal('holdingModal'));
  document.getElementById('cancelHolding').addEventListener('click', () => hideModal('holdingModal'));
  document.getElementById('holdingForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = Object.fromEntries(new FormData(e.target));
    data.shares = parseFloat(data.shares) || 0;
    data.avgCost = parseFloat(data.avgCost) || 0;
    try {
      await api.addHolding(data);
      hideModal('holdingModal');
      e.target.reset();
      holdingsComp.holdings = await api.getHoldings();
      holdingsComp.render();
    } catch (err) {
      alert('添加失败: ' + err.message);
    }
  });

  document.getElementById('addAlertBtn').addEventListener('click', () => showModal('alertModal'));
  document.getElementById('cancelAlert').addEventListener('click', () => hideModal('alertModal'));
  document.getElementById('alertForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = Object.fromEntries(new FormData(e.target));
    try {
      await api.addAlert(data.symbol, data.type, parseFloat(data.value));
      hideModal('alertModal');
      e.target.reset();
      alertsComp.alerts = await api.getAlerts();
      alertsComp.render();
    } catch (err) {
      alert('添加失败: ' + err.message);
    }
  });
}

function showModal(id) {
  document.getElementById(id).classList.remove('hidden');
}

function hideModal(id) {
  document.getElementById(id).classList.add('hidden');
}

function showToast(message, type = '') {
  const container = document.getElementById('toastContainer');
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.textContent = message;
  container.appendChild(toast);
  setTimeout(() => toast.remove(), 5000);
}

// Logout
document.getElementById('logoutBtn').addEventListener('click', () => auth.logout());

// Start
document.addEventListener('DOMContentLoaded', init);
