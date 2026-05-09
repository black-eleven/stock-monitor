const WebSocket = require('ws');
const config = require('./config');

class QosClient {
  constructor() {
    this.ws = null;
    this.subscribedCodes = [];
    this.onQuote = null;     // callback(quote)
    this.onKline = null;     // callback(kline)
    this.connected = false;
    this.reconnectTimer = null;
    this.heartbeatTimer = null;
    this._reconnectAttempt = 0;
    this._reqSeq = 0;
    this._pendingRejects = [];
  }

  connect() {
    if (!config.qosKey) {
      console.warn('[QOS] No API key configured');
      return;
    }

    // Guard against concurrent connect calls
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.ws = new WebSocket(config.qosWsUrl);

    this.ws.on('open', () => {
      console.log('[QOS] Connected');
      this.connected = true;
      this._reconnectAttempt = 0;
      this._pendingRejects = [];
      this._startHeartbeat();
      // Re-subscribe after reconnect
      if (this.subscribedCodes.length > 0) {
        this._send({ type: 'S', codes: [this.subscribedCodes.join(',')] });
      }
    });

    this.ws.on('message', (raw) => {
      try {
        const msg = JSON.parse(raw.toString());
        this._handleMessage(msg);
      } catch (err) {
        console.error('[QOS] Parse error:', err.message);
      }
    });

    this.ws.on('close', () => {
      console.log('[QOS] Disconnected');
      this.connected = false;
      this._stopHeartbeat();
      this._rejectPending();
      this._scheduleReconnect();
    });

    this.ws.on('error', (err) => {
      console.error('[QOS] Error:', err.message);
      // WebSocket already closes itself on error
    });
  }

  subscribe(codes) {
    this.subscribedCodes = [...new Set([...this.subscribedCodes, ...codes])];
    if (this.connected) {
      this._send({ type: 'S', codes: [this.subscribedCodes.join(',')] });
    }
  }

  unsubscribe(codes) {
    this.subscribedCodes = this.subscribedCodes.filter(c => !codes.includes(c));
    if (this.connected && this.subscribedCodes.length > 0) {
      this._send({ type: 'S', codes: [this.subscribedCodes.join(',')] });
    }
    if (this.subscribedCodes.length === 0) {
      this._send({ type: 'SC', codes: [codes.join(',')] });
    }
  }

  fetchKline(code, kt, count, adjust = 0) {
    return new Promise((resolve, reject) => {
      if (!this.connected) return reject(new Error('Not connected'));

      const reqid = Date.now() * 1000 + (++this._reqSeq % 1000);
      this._pendingRejects.push(reject);

      const handler = (raw) => {
        try {
          const msg = JSON.parse(raw.toString());
          if (msg.type === 'RK' && msg.reqid === reqid) {
            this.ws.removeListener('message', handler);
            const idx = this._pendingRejects.indexOf(reject);
            if (idx !== -1) this._pendingRejects.splice(idx, 1);
            resolve(msg.data || []);
          }
        } catch {}
      };
      this.ws.on('message', handler);

      this._send({
        type: 'RK',
        kline_reqs: [{ c: code, co: count, a: adjust, kt }],
        reqid,
      });

      setTimeout(() => {
        this.ws.removeListener('message', handler);
        const idx = this._pendingRejects.indexOf(reject);
        if (idx !== -1) this._pendingRejects.splice(idx, 1);
        reject(new Error('K-line request timeout'));
      }, 10000);
    });
  }

  fetchHistoryKline(code, endTs, kt, count, adjust = 0) {
    return new Promise((resolve, reject) => {
      if (!this.connected) return reject(new Error('Not connected'));

      const reqid = Date.now() * 1000 + (++this._reqSeq % 1000);
      this._pendingRejects.push(reject);

      const handler = (raw) => {
        try {
          const msg = JSON.parse(raw.toString());
          if (msg.type === 'RH' && msg.reqid === reqid) {
            this.ws.removeListener('message', handler);
            const idx = this._pendingRejects.indexOf(reject);
            if (idx !== -1) this._pendingRejects.splice(idx, 1);
            resolve(msg.data || []);
          }
        } catch {}
      };
      this.ws.on('message', handler);

      this._send({
        type: 'RH',
        kline_reqs: [{ c: code, e: endTs, co: count, a: adjust, kt }],
        reqid,
      });

      setTimeout(() => {
        this.ws.removeListener('message', handler);
        const idx = this._pendingRejects.indexOf(reject);
        if (idx !== -1) this._pendingRejects.splice(idx, 1);
        reject(new Error('History K-line request timeout'));
      }, 10000);
    });
  }

  _handleMessage(msg) {
    if (msg.tp === 'S' && this.onQuote) {
      this.onQuote({
        code: msg.c,
        price: parseFloat(msg.lp),
        yp: parseFloat(msg.yp),
        open: parseFloat(msg.o),
        high: parseFloat(msg.h),
        low: parseFloat(msg.l),
        volume: parseFloat(msg.v),
        turnover: parseFloat(msg.t),
        timestamp: msg.ts,
        status: msg.s,
      });
    }
    if (msg.tp === 'K' && this.onKline) {
      this.onKline({
        code: msg.c,
        open: parseFloat(msg.o),
        close: parseFloat(msg.cl),
        high: parseFloat(msg.h),
        low: parseFloat(msg.l),
        volume: parseFloat(msg.v),
        timestamp: msg.ts,
        kt: msg.kt,
      });
    }
  }

  _send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  _startHeartbeat() {
    this._stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      this._send({ type: 'H' });
    }, 15000);
  }

  _stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  _rejectPending() {
    for (const reject of this._pendingRejects) {
      reject(new Error('Connection lost'));
    }
    this._pendingRejects = [];
  }

  _scheduleReconnect() {
    const delay = Math.min(1000 * Math.pow(2, this._reconnectAttempt), 30000);
    this._reconnectAttempt += 1;
    console.log(`[QOS] Reconnecting in ${delay}ms (attempt ${this._reconnectAttempt})`);
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  close() {
    clearTimeout(this.reconnectTimer);
    this._stopHeartbeat();
    if (this.ws) {
      this.ws.removeAllListeners();
      this.ws.close();
      this.ws = null;
    }
  }
}

module.exports = QosClient;
