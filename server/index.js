require('dotenv').config();
const express = require('express');
const http = require('http');
const path = require('path');
const config = require('./config');
const QosClient = require('./qos-client');
const WsHandler = require('./ws-handler');
const AlertEngine = require('./alert-engine');

const app = express();
const server = http.createServer(app);

// Middleware
app.use(express.json());
app.use(express.static(path.resolve(__dirname, '..', 'public')));

// QOS client + WS handler + Alert engine
const qosClient = new QosClient();
const wsHandler = new WsHandler(server);
const alertEngine = new AlertEngine(wsHandler);

// Wire up QOS callbacks
let _reqSeq = 0;
// Need fetchQuote for REST routes — use RS (snapshot request) via WS
qosClient.fetchQuote = function (code) {
  return new Promise((resolve, reject) => {
    if (!this.connected) return reject(new Error('Not connected'));
    const reqid = Date.now() * 1000 + (_reqSeq++ % 1000);
    const handler = (raw) => {
      try {
        const msg = JSON.parse(raw.toString());
        if (msg.type === 'RS' && msg.reqid === reqid) {
          this.ws.removeListener('message', handler);
          const d = msg.data && msg.data[0];
          if (d) {
            resolve({
              code: d.c, price: parseFloat(d.lp), yp: parseFloat(d.yp),
              open: parseFloat(d.o), high: parseFloat(d.h), low: parseFloat(d.l),
              volume: parseFloat(d.v), turnover: parseFloat(d.t), timestamp: d.ts, status: d.s,
            });
          } else {
            reject(new Error('No data'));
          }
        }
      } catch {}
    };
    this.ws.on('message', handler);
    this._send({ type: 'RS', codes: [code], reqid });
    setTimeout(() => { this.ws.removeListener('message', handler); reject(new Error('Timeout')); }, 10000);
  });
};

// On quote from QOS -> broadcast to browsers + check alerts
qosClient.onQuote = (quote) => {
  wsHandler.broadcastQuote(quote);
  alertEngine.evaluate(quote);
};

// Routes
app.use('/api/watchlist', require('./routes/watchlist'));
app.use('/api/alerts', require('./routes/alerts'));
app.use('/api/holdings', require('./routes/holdings'));
app.use('/api', require('./routes/quote')(qosClient));
app.use('/api', require('./routes/kline')(qosClient));

// Connect to QOS
qosClient.connect();

// Start server
server.listen(config.port, () => {
  console.log(`Stock Monitor running at http://localhost:${config.port}`);
});

// Graceful shutdown
process.on('SIGINT', () => {
  console.log('\nShutting down...');
  qosClient.close();
  wsHandler.close();

  // Force exit after 3s if graceful shutdown hangs (e.g. open connections)
  const forceExit = setTimeout(() => process.exit(1), 3000);
  forceExit.unref();

  server.close(() => {
    clearTimeout(forceExit);
    process.exit(0);
  });
});
