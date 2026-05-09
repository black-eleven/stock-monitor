const storage = require('./storage');

class AlertEngine {
  constructor(wsHandler) {
    this.wsHandler = wsHandler;
    this.alerts = [];
  }

  load() {
    this.alerts = storage.read('alerts');
  }

  // Called on every quote update from QOS
  evaluate(quote) {
    this.load(); // Reload to pick up any changes
    const now = Date.now();

    for (const alert of this.alerts) {
      if (!alert.enabled) continue;

      const price = quote.price;
      if (!price) continue;

      let triggered = false;
      switch (alert.type) {
        case 'above':
          triggered = price >= alert.value;
          break;
        case 'below':
          triggered = price <= alert.value;
          break;
        case 'change_pct':
          if (quote.yp && quote.yp > 0) {
            const pct = ((price - quote.yp) / quote.yp) * 100;
            triggered = Math.abs(pct) >= Math.abs(alert.value);
          }
          break;
      }

      if (triggered) {
        // Dedup: skip if triggered within last 30 minutes
        const lastTrigger = alert.lastTriggeredAt ? new Date(alert.lastTriggeredAt).getTime() : 0;
        if (now - lastTrigger < 30 * 60 * 1000) continue;

        // Mark triggered
        storage.update('alerts', a => a.id === alert.id, a => ({
          ...a,
          lastTriggeredAt: new Date().toISOString(),
        }));

        const event = {
          alertId: alert.id,
          symbol: quote.code,
          price: quote.price,
          type: alert.type,
          value: alert.value,
          message: `${quote.code} ${alert.type === 'above' ? '突破' : alert.type === 'below' ? '跌破' : '涨跌幅达'} ${alert.value}，当前价 ${quote.price.toFixed(2)}`,
          triggeredAt: new Date().toISOString(),
        };

        // Log to file
        const logs = storage.read('alert-logs');
        logs.push({ ...event, id: logs.length + 1 });
        storage.write('alert-logs', logs.slice(-200)); // Keep last 200

        // Push to browser
        this.wsHandler.broadcastAlert(event);
      }
    }
  }
}

module.exports = AlertEngine;
