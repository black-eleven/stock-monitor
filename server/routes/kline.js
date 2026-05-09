const express = require('express');
const router = express.Router();

const KT_MAP = {
  '1m': 1, '5m': 5, '15m': 15, '30m': 30,
  '1h': 60, '2h': 120, '4h': 240,
  '1d': 1001, '1w': 1007, '1M': 1030,
};

module.exports = function (qosClient) {
  // GET /api/kline/HK:700?interval=1d&count=100
  router.get('/kline/:symbol', async (req, res) => {
    const symbol = req.params.symbol.toUpperCase();
    const interval = req.query.interval || '1d';
    const count = parseInt(req.query.count || '100', 10);
    const kt = KT_MAP[interval];

    if (!kt) {
      return res.status(400).json({ error: `Invalid interval: ${interval}. Supported: ${Object.keys(KT_MAP).join(', ')}` });
    }

    try {
      const data = await qosClient.fetchHistoryKline(symbol, Math.floor(Date.now() / 1000), kt, count);
      res.json(data);
    } catch (err) {
      res.status(502).json({ error: 'Failed to fetch kline data' });
    }
  });

  return router;
};
