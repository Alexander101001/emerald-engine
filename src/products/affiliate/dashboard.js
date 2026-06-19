import { Router } from 'express';
import { getDb } from '../../db/database.js';

const router = Router();

router.get('/api/affiliate/dashboard', async (req, res) => {
  try {
    const db = await getDb();
    const clicks = db.exec("SELECT COUNT(*) as c FROM affiliate_log WHERE type='click'")[0]?.values[0][0] || 0;
    const conversions = db.exec("SELECT COUNT(*) as c FROM affiliate_log WHERE type='conversion'")[0]?.values[0][0] || 0;
    const revenue = db.exec("SELECT COALESCE(SUM(amount),0) as r FROM affiliate_log WHERE type='conversion'")[0]?.values[0][0] || 0;

    res.json({ clicks, conversions, revenue, rate: clicks > 0 ? ((conversions / clicks) * 100).toFixed(2) : '0.00' });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

export default router;
