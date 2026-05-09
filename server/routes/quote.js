const express = require('express');
const router = express.Router();

module.exports = function (qosClient) {
  // GET /api/quote/batch?symbols=HK:700,HK:9988 — MUST be before :symbol route
  router.get('/quote/batch', async (req, res) => {
    const symbols = (req.query.symbols || '').split(',').map(s => s.trim().toUpperCase()).filter(Boolean);
    if (symbols.length === 0) {
      return res.status(400).json({ error: 'No symbols provided' });
    }
    const results = await Promise.allSettled(symbols.map(s => qosClient.fetchQuote(s)));
    const data = symbols.reduce((acc, s, i) => {
      if (results[i].status === 'fulfilled') acc[s] = results[i].value;
      return acc;
    }, {});
    res.json(data);
  });

  // GET /api/quote/:symbol — e.g. /api/quote/HK:700
  router.get('/quote/:symbol', async (req, res) => {
    const symbol = req.params.symbol.toUpperCase();
    if (!/^HK:\d{2,5}$/.test(symbol)) {
      return res.status(400).json({ error: 'Invalid symbol format. Use HK:700' });
    }
    try {
      const data = await qosClient.fetchQuote(symbol);
      res.json(data);
    } catch (err) {
      res.status(502).json({ error: 'Failed to fetch quote' });
    }
  });

  return router;
};
