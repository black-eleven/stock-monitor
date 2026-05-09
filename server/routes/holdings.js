const express = require('express');
const router = express.Router();
const storage = require('../storage');

// GET /api/holdings
router.get('/', (req, res) => {
  res.json(storage.read('holdings'));
});

// POST /api/holdings  body: { symbol, name, shares, avgCost, buyDate }
router.post('/', (req, res) => {
  const { symbol, name, shares, avgCost } = req.body;
  if (!symbol || !shares || !avgCost) {
    return res.status(400).json({ error: 'symbol, shares, avgCost required' });
  }
  const existing = storage.readOne('holdings', h => h.symbol === symbol);
  if (existing) {
    return res.status(409).json({ error: 'Holding already exists. Use PUT to update.' });
  }
  const item = { symbol, name: name || symbol, shares: parseFloat(shares), avgCost: parseFloat(avgCost), buyDate: req.body.buyDate || new Date().toISOString().slice(0, 10) };
  storage.add('holdings', item);
  res.status(201).json(item);
});

// PUT /api/holdings/:symbol
router.put('/:symbol', (req, res) => {
  const symbol = req.params.symbol;
  const updated = storage.update('holdings', h => h.symbol === symbol, existing => ({ ...existing, ...req.body, symbol: existing.symbol }));
  if (!updated) return res.status(404).json({ error: 'Holding not found' });
  res.json(updated);
});

// DELETE /api/holdings/:symbol
router.delete('/:symbol', (req, res) => {
  const ok = storage.remove('holdings', h => h.symbol === req.params.symbol);
  if (!ok) return res.status(404).json({ error: 'Holding not found' });
  res.json({ ok: true });
});

module.exports = router;
