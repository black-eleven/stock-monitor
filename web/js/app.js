// Global instances
const api = new ApiClient();
let dashboardComp, watchlistComp, klineComp, holdingsComp, alertsComp, analysisComp, recommendComp;
let componentsInited = false;
let pendingStock = null; // stock symbol to navigate to after components init

// Switch to a specific tab programmatically
function switchToTab(tabName) {
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
  const btn = document.querySelector(`.tab-btn[data-tab="${tabName}"]`);
  if (btn) btn.classList.add('active');
  const pane = document.getElementById(`tab-${tabName}`);
  if (pane) pane.classList.add('active');

  if (!componentsInited && tabName !== 'dashboard') {
    initLazyComponents();
  }

  if (tabName === 'watchlist' && klineComp) {
    setTimeout(() => klineComp.resize(), 300);
  }
}

// Navigate to stock detail page — switches to watchlist tab and selects the stock
function navigateToStock(symbol) {
  if (!symbol) return;
  // Update URL without reload
  const url = new URL(location);
  url.searchParams.set('stock', symbol);
  history.pushState(null, '', url);

  if (!componentsInited) {
    pendingStock = symbol;
    switchToTab('watchlist');
    return;
  }
  switchToTab('watchlist');
  // Ensure "my" sub-tab is active
  document.querySelectorAll('.subtab-btn').forEach(b => b.classList.remove('active'));
  const myBtn = document.querySelector('.subtab-btn[data-subtab="my"]');
  if (myBtn) myBtn.classList.add('active');
  document.getElementById('watchlistMy').style.display = 'block';
  document.getElementById('watchlistRecommend').style.display = 'none';
  document.getElementById('addWatchlistBtn').style.display = '';

  if (watchlistComp) {
    // Check if stock is in watchlist
    const inList = watchlistComp.watchlist.find(w => w.symbol === symbol);
    if (inList) {
      watchlistComp.selectStock(symbol);
    } else {
      // Auto-add to watchlist, then select (need name from quote)
      watchlistComp._promptAddThenSelect(symbol);
    }
  }
}

// Tab switching via click
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    switchToTab(btn.dataset.tab);
  });
});

// Handle browser back/forward
window.addEventListener('popstate', () => {
  const params = new URLSearchParams(location.search);
  const symbol = params.get('stock');
  if (symbol && componentsInited && watchlistComp) {
    switchToTab('watchlist');
    const inList = watchlistComp.watchlist.find(w => w.symbol === symbol);
    if (inList) watchlistComp.selectStock(symbol);
  }
});

// Init dashboard on page load
async function init() {
  // Auth guard
  if (!auth.isLoggedIn()) {
    window.location.href = '/login.html';
    return;
  }

  // Check for stock param in URL
  const params = new URLSearchParams(location.search);
  pendingStock = params.get('stock');

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

  // Init dashboard immediately
  dashboardComp = new DashboardComponent(api);
  await dashboardComp.init();

  // Handle alert toasts globally
  api.on('alert', (alert) => {
    showToast(`⚠ ${escapeHtml(alert.message)}`, 'alert');
  });

  // If URL has stock param, navigate after dashboard is ready
  if (pendingStock) {
    switchToTab('watchlist');
  }
}

// Lazy-init remaining components when user first switches tabs
async function initLazyComponents() {
  if (componentsInited) return;
  componentsInited = true;

  // Create kline component for watchlist inline chart
  klineComp = new KlineComponent(api, 'watchlistChartContainer', 'watchlistKlineIntervals');
  klineComp.init();

  // Create analysis component (populates signal cache, used by dashboard + watchlist)
  analysisComp = new AnalysisComponent(api);
  await analysisComp.init();

  // Create watchlist component with integrated kline + analysis
  watchlistComp = new WatchlistComponent(api, klineComp, analysisComp);
  await watchlistComp.init();

  // Register watchlist names with dashboard for display
  if (dashboardComp) {
    dashboardComp.registerNames(watchlistComp.watchlist);
  }

  recommendComp = new RecommendComponent(api, async (symbol, name) => {
    await api.addWatchlist(symbol, name);
    watchlistComp.watchlist.push({ symbol, name });
    watchlistComp.renderTabs();
    if (watchlistComp.watchlist.length === 1) {
      watchlistComp.selectStock(symbol);
    }
    if (watchlistComp.onWatchlistChange) watchlistComp.onWatchlistChange(watchlistComp.watchlist);
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
      if (target === 'my' && watchlistComp && watchlistComp.selectedSymbol) {
        setTimeout(() => { if (klineComp) klineComp.resize(); }, 300);
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

  document.getElementById('recommendIndustry').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') document.getElementById('recommendSearchBtn').click();
  });

  alertsComp = new AlertsComponent(api);
  holdingsComp = new HoldingsComponent(api, alertsComp);
  await holdingsComp.init();

  // Wire signal provider for watchlist tab badges
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

  watchlistComp.renderTabs();
  if (watchlistComp.selectedSymbol) {
    const quote = watchlistComp.quotes[watchlistComp.selectedSymbol];
    watchlistComp.renderDetail(watchlistComp.selectedSymbol, quote);
  }

  // If URL had a stock param, navigate to it (overrides default first-stock selection)
  if (pendingStock) {
    const inList = watchlistComp.watchlist.find(w => w.symbol === pendingStock);
    if (inList) {
      watchlistComp.selectStock(pendingStock);
    } else {
      watchlistComp._promptAddThenSelect(pendingStock);
    }
    pendingStock = null;
  }

  // Wire up real-time quote -> components
  api.on('snapshot', (quotes) => {
    for (const quote of quotes) {
      watchlistComp.updateQuote(quote);
      analysisComp.updateQuote(quote);
      holdingsComp.updateQuote(quote);
    }
  });

  api.on('quote', (quote) => {
    watchlistComp.updateQuote(quote);
    analysisComp.updateQuote(quote);
    holdingsComp.updateQuote(quote);
  });

  api.on('alert', (alert) => {
    alertsComp.addLog(alert);
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
    } catch (err) { alert('添加失败: ' + err.message); }
  });

  document.getElementById('cancelAlert').addEventListener('click', () => {
    const symbolInput = document.querySelector('#alertForm input[name="symbol"]');
    if (symbolInput) {
      symbolInput.readOnly = false;
      symbolInput.style.background = '';
    }
    hideModal('alertModal');
  });
  document.getElementById('alertForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = Object.fromEntries(new FormData(e.target));
    try {
      await api.addAlert(data.symbol, data.type, parseFloat(data.value));
      hideModal('alertModal');
      const form = e.target;
      const si = form.querySelector('input[name="symbol"]');
      if (si) { si.readOnly = false; si.style.background = ''; }
      form.reset();
      alertsComp.alerts = await api.getAlerts();
      holdingsComp.render();
    } catch (err) { alert('添加失败: ' + err.message); }
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
