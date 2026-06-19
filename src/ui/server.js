import express from 'express';
import { getTotalProfits } from '../db/database.js';
import affiliateLanding from '../products/affiliate/index.js';
import affiliateDashboard from '../products/affiliate/dashboard.js';
import affiliatePayment from '../products/affiliate/payment.js';

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());
app.use(affiliateLanding);
app.use(affiliateDashboard);
app.use(affiliatePayment);

app.get('/api/status', async (req, res) => {
    const profits = await getTotalProfits().catch(() => 0);
    res.json({ status: 'ACTIVE', mode: 'AUTONOMOUS', profits });
});

app.post('/api/cron', async (req, res) => {
    try {
        const { execSync } = await import('child_process');
        execSync('cd /app && git pull origin main 2>/dev/null', { timeout: 15000 });
        res.json({ status: 'synced' });
    } catch (err) {
        res.json({ status: 'busy' });
    }
});

app.use(express.static('public'));

app.listen(PORT, () => {
    console.log(`[UI] Dashboard at http://localhost:${PORT}`);
});
