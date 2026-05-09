const express = require('express');
const router = express.Router();
const storage = require('../storage');

// GET /api/watchlist
router.get('/', (req, res) => {
  const list = storage.read('watchlist');
  res.json(list);
});

// POST /api/watchlist  body: { symbol: "HK:700", name: "腾讯控股" }
router.post('/', (req, res) => {
  const { symbol, name } = req.body;
  if (!symbol || !name) {
    return res.status(400).json({ error: 'symbol and name required' });
  }
  const list = storage.read('watchlist');
  if (list.find(w => w.symbol === symbol)) {
    return res.status(409).json({ error: 'Symbol already in watchlist' });
  }
  const item = { symbol, name, addedAt: new Date().toISOString() };
  storage.write('watchlist', [...list, item]);
  res.status(201).json(item);
});

// DELETE /api/watchlist/:symbol
router.delete('/:symbol', (req, res) => {
  const symbol = req.params.symbol;
  const list = storage.read('watchlist');
  const filtered = list.filter(w => w.symbol !== symbol);
  if (filtered.length === list.length) {
    return res.status(404).json({ error: 'Symbol not found' });
  }
  storage.write('watchlist', filtered);
  res.json({ ok: true });
});

module.exports = router;
