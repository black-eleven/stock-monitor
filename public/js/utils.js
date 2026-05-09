// Format price to 2 decimal places
function formatPrice(price) {
  if (price == null || isNaN(price)) return '--';
  return price.toFixed(2);
}

// Format large numbers with K/M/B suffix
function formatVolume(v) {
  if (v == null || isNaN(v)) return '--';
  if (v >= 1e8) return (v / 1e8).toFixed(1) + '亿';
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万';
  return v.toFixed(0);
}

// Format currency amount
function formatTurnover(v) {
  if (v == null || isNaN(v)) return '--';
  if (v >= 1e8) return (v / 1e8).toFixed(1) + '亿';
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万';
  return v.toFixed(0);
}

// Calculate change percentage
function calcChangePct(price, yp) {
  if (!yp || !price) return 0;
  return ((price - yp) / yp) * 100;
}

// Format change display
function formatChange(price, yp) {
  if (!yp || !price) return '+0.00 (0.00%)';
  const diff = price - yp;
  const pct = (diff / yp) * 100;
  const sign = diff >= 0 ? '+' : '';
  return `${sign}${diff.toFixed(2)} (${sign}${pct.toFixed(2)}%)`;
}

// Determine up/down class
function changeDir(price, yp) {
  if (!yp || !price) return '';
  return price >= yp ? 'up' : 'down';
}

// Format timestamp
function formatTime(ts) {
  const d = new Date(ts * 1000);
  return d.toLocaleTimeString('zh-HK', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

// Escape HTML for XSS safety
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Shorten HK stock code: "HK:700" -> "700"
function shortCode(code) {
  return code.replace(/^HK:/, '');
}
