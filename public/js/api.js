class ApiClient {
  constructor() {
    this.ws = null;
    this.listeners = {};
    this.reconnectTimer = null;
  }

  // REST API helpers
  async get(path) {
    const res = await fetch(path);
    if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
    return res.json();
  }

  async post(path, body) {
    const res = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(`POST ${path} failed: ${res.status}`);
    return res.json();
  }

  async put(path, body) {
    const res = await fetch(path, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(`PUT ${path} failed: ${res.status}`);
    return res.json();
  }

  async del(path) {
    const res = await fetch(path, { method: 'DELETE' });
    if (!res.ok) throw new Error(`DELETE ${path} failed: ${res.status}`);
    return res.json();
  }

  // WebSocket connection
  connectWs() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.ws = new WebSocket(`${protocol}//${location.host}`);

    this.ws.onopen = () => {
      console.log('[WS] Connected');
      this.emit('connected');
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this.emit(msg.type, msg.data);
      } catch (err) {
        console.error('[WS] Parse error:', err);
      }
    };

    this.ws.onclose = () => {
      console.log('[WS] Disconnected');
      this.emit('disconnected');
      // Auto reconnect after 3s
      this.reconnectTimer = setTimeout(() => this.connectWs(), 3000);
    };

    this.ws.onerror = () => {
      this.ws.close();
    };
  }

  // Event system
  on(event, callback) {
    if (!this.listeners[event]) this.listeners[event] = [];
    this.listeners[event].push(callback);
  }

  off(event, callback) {
    if (!this.listeners[event]) return;
    this.listeners[event] = this.listeners[event].filter(cb => cb !== callback);
  }

  emit(event, data) {
    if (!this.listeners[event]) return;
    for (const cb of this.listeners[event]) cb(data);
  }

  // Specific API calls
  async getWatchlist() { return this.get('/api/watchlist'); }
  async addWatchlist(symbol, name) { return this.post('/api/watchlist', { symbol, name }); }
  async removeWatchlist(symbol) { return this.del(`/api/watchlist/${encodeURIComponent(symbol)}`); }

  async getAlerts() { return this.get('/api/alerts'); }
  async addAlert(symbol, type, value) { return this.post('/api/alerts', { symbol, type, value }); }
  async updateAlert(id, data) { return this.put(`/api/alerts/${id}`, data); }
  async deleteAlert(id) { return this.del(`/api/alerts/${id}`); }

  async getHoldings() { return this.get('/api/holdings'); }
  async addHolding(data) { return this.post('/api/holdings', data); }
  async updateHolding(symbol, data) { return this.put(`/api/holdings/${encodeURIComponent(symbol)}`, data); }
  async deleteHolding(symbol) { return this.del(`/api/holdings/${encodeURIComponent(symbol)}`); }

  async getKline(symbol, interval = '1d', count = 200) {
    return this.get(`/api/kline/${encodeURIComponent(symbol)}?interval=${interval}&count=${count}`);
  }
}
