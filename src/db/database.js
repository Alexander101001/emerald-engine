import initSqlJs from 'sql.js';
import fs from 'fs';
import path from 'path';

const DB_PATH = path.join(process.cwd(), 'emerald.db');

let db;

export async function getDb() {
  if (db) return db;

  const SQL = await initSqlJs();
  if (fs.existsSync(DB_PATH)) {
    const buffer = fs.readFileSync(DB_PATH);
    db = new SQL.Database(buffer);
  } else {
    db = new SQL.Database();
  }

  db.run(`CREATE TABLE IF NOT EXISTS profits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT,
    amount REAL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
  )`);
  db.run(`CREATE TABLE IF NOT EXISTS strategies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    tier TEXT,
    revenue REAL DEFAULT 0,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  )`);

  save();
  return db;
}

export function save() {
  if (!db) return;
  const data = db.export();
  const buffer = Buffer.from(data);
  fs.writeFileSync(DB_PATH, buffer);
}

export async function logProfit(source, amount) {
  const d = await getDb();
  d.run('INSERT INTO profits (source, amount) VALUES (?, ?)', [source, amount]);
  save();
}

export async function getTotalProfits() {
  const d = await getDb();
  const stmt = d.exec('SELECT SUM(amount) as total FROM profits');
  return stmt.length > 0 ? stmt[0].values[0][0] || 0 : 0;
}

export default { getDb, save, logProfit, getTotalProfits };
