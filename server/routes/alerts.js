const express = require('express');
const router = express.Router();
const storage = require('../storage');

function nextId() {
  const alerts = storage.read('alerts');
  return alerts.length > 0 ? Math.max(...alerts.map(a => a.id)) + 1 : 1;
}

// GET /api/alerts
router.get('/', (req, res) => {
  res.json(storage.read('alerts'));
});

// POST /api/alerts  body: { symbol, type: "above"|"below"|"change_pct", value }
router.post('/', (req, res) => {
  const { symbol, type, value } = req.body;
  if (!symbol || !type || value === undefined) {
    return res.status(400).json({ error: 'symbol, type, value required' });
  }
  if (!['above', 'below', 'change_pct'].includes(type)) {
    return res.status(400).json({ error: 'type must be above, below, or change_pct' });
  }
  const alert = { id: nextId(), symbol, type, value: parseFloat(value), enabled: true, createdAt: new Date().toISOString(), lastTriggeredAt: null };
  storage.add('alerts', alert);
  res.status(201).json(alert);
});

// PUT /api/alerts/:id
router.put('/:id', (req, res) => {
  const id = parseInt(req.params.id, 10);
  const updated = storage.update('alerts', a => a.id === id, existing => ({
    ...existing,
    ...req.body,
    id: existing.id,
    createdAt: existing.createdAt,
  }));
  if (!updated) return res.status(404).json({ error: 'Alert not found' });
  res.json(updated);
});

// DELETE /api/alerts/:id
router.delete('/:id', (req, res) => {
  const id = parseInt(req.params.id, 10);
  const ok = storage.remove('alerts', a => a.id === id);
  if (!ok) return res.status(404).json({ error: 'Alert not found' });
  res.json({ ok: true });
});

module.exports = router;
