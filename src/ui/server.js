import express from 'express';
import { getTotalProfits } from '../db/database.js';

const app = express();
const PORT = process.env.PORT || 3000;

app.get('/api/status', async (req, res) => {
    const profits = await getTotalProfits().catch(() => 0);
    res.json({ status: 'ACTIVE', mode: 'AUTONOMOUS', profits });
});

app.use(express.static('public'));

app.listen(PORT, () => {
    console.log(`[UI] Dashboard at http://localhost:${PORT}`);
});
