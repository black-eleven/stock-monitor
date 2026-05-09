const WebSocket = require('ws');

class WsHandler {
  constructor(server) {
    this.wss = new WebSocket.Server({ server });
    this.clients = new Set();
    this.latestQuotes = new Map(); // code -> quote

    this.wss.on('connection', (ws) => {
      this.clients.add(ws);
      console.log(`[WS] Client connected (${this.clients.size} total)`);

      // Send latest cached quotes on connect
      if (this.latestQuotes.size > 0) {
        ws.send(JSON.stringify({
          type: 'snapshot',
          data: Array.from(this.latestQuotes.values()),
        }));
      }

      ws.on('close', () => {
        this.clients.delete(ws);
        console.log(`[WS] Client disconnected (${this.clients.size} total)`);
      });
    });
  }

  // Broadcast quote update to all connected browsers
  broadcastQuote(quote) {
    this.latestQuotes.set(quote.code, quote);
    const msg = JSON.stringify({ type: 'quote', data: quote });
    for (const ws of this.clients) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(msg);
      }
    }
  }

  // Close all client connections on shutdown
  close() {
    for (const ws of this.clients) {
      ws.close();
    }
    this.clients.clear();
  }

  // Broadcast alert trigger
  broadcastAlert(alert) {
    const msg = JSON.stringify({ type: 'alert', data: alert });
    for (const ws of this.clients) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(msg);
      }
    }
  }
}

module.exports = WsHandler;
