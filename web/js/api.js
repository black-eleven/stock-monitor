class ApiClient {
  constructor() {
    this.ws = null;
    this.listeners = {};
    this.reconnectTimer = null;
  }

  _headers() {
    return {
      'Content-Type': 'application/json',
      ...auth.getAuthHeaders(),
    };
  }

  // REST API helpers
  async get(path) {
    const res = await fetch(path, { headers: auth.getAuthHeaders() });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
    return res.json();
  }

  async post(path, body) {
    const res = await fetch(path, {
      method: 'POST',
      headers: this._headers(),
      body: JSON.stringify(body),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`POST ${path} failed: ${res.status}`);
    return res.json();
  }

  async put(path, body) {
    const res = await fetch(path, {
      method: 'PUT',
      headers: this._headers(),
      body: JSON.stringify(body),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`PUT ${path} failed: ${res.status}`);
    return res.json();
  }

  async del(path) {
    const res = await fetch(path, {
      method: 'DELETE',
      headers: auth.getAuthHeaders(),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`DELETE ${path} failed: ${res.status}`);
    return res.json();
  }

  // WebSocket connection
  connectWs() {
    if (!auth.token) return;
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.ws = new WebSocket(`${protocol}//${location.host}/ws?token=${auth.token}`);

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
  async removeWatchlist(symbol) { return this.del(`/api/watchlist/${symbol}`); }

  async getAlerts() { return this.get('/api/alerts'); }
  async addAlert(symbol, type, value) { return this.post('/api/alerts', { symbol, type, value }); }
  async updateAlert(id, data) { return this.put(`/api/alerts/${id}`, data); }
  async deleteAlert(id) { return this.del(`/api/alerts/${id}`); }

  async getHoldings() { return this.get('/api/holdings'); }
  async addHolding(data) { return this.post('/api/holdings', data); }
  async updateHolding(symbol, data) { return this.put(`/api/holdings/${symbol}`, data); }
  async deleteHolding(symbol) { return this.del(`/api/holdings/${symbol}`); }

  async getKline(symbol, interval = '1d', count = 200) {
    return this.get(`/api/kline/${symbol}?interval=${interval}&count=${count}`);
  }

  async recommend(industry) {
    return this.post('/api/recommendations', { industry });
  }
}
