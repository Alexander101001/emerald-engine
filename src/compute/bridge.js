import { execSync, exec } from 'child_process';
import path from 'path';
import fs from 'fs';

const RUST_BIN = path.resolve('src/compute/rust/target/release/analyzer');
const GO_BIN = path.resolve('src/network/go/rotator');

export async function analyzeWithRust(csvData) {
  try {
    if (!fs.existsSync(RUST_BIN)) {
      console.warn('[BRIDGE] Rust binary not found — compiling...');
      execSync('cd src/compute/rust && cargo build --release 2>/dev/null || echo skip', { stdio: 'pipe' });
    }
    if (!fs.existsSync(RUST_BIN)) {
      console.warn('[BRIDGE] Rust unavailable — using JS fallback');
      return jsAnalyze(csvData);
    }
    const result = execSync(`echo "${csvData.replace(/"/g, '\\"')}\nEOF" | ${RUST_BIN}`, {
      timeout: 10000,
      encoding: 'utf-8',
    });
    return JSON.parse(result);
  } catch (err) {
    console.warn('[BRIDGE] Rust error:', err.message);
    return jsAnalyze(csvData);
  }
}

export async function fetchWithGo(urls) {
  try {
    if (!fs.existsSync(GO_BIN)) {
      console.warn('[BRIDGE] Go binary not found — building...');
      execSync('cd src/network/go && go build -o rotator rotator.go 2>/dev/null || echo skip', { stdio: 'pipe' });
    }
    if (!fs.existsSync(GO_BIN)) {
      console.warn('[BRIDGE] Go unavailable — using JS fallback');
      return jsFetch(urls);
    }
    const input = Array.isArray(urls) ? urls.join('\n') : urls;
    const result = execSync(`echo "${input}" | ${GO_BIN}`, {
      timeout: 30000,
      encoding: 'utf-8',
    });
    return JSON.parse(result);
  } catch (err) {
    console.warn('[BRIDGE] Go error:', err.message);
    return jsFetch(urls);
  }
}

function jsAnalyze(csvData) {
  const scores = {};
  for (const line of csvData.split('\n')) {
    const parts = line.split(',');
    if (parts.length >= 3) {
      const keyword = parts[0].trim().toLowerCase();
      const mentions = parseFloat(parts[1]) || 0;
      const growth = parseFloat(parts[2]) || 0;
      scores[keyword] = mentions * growth * 100;
    }
  }
  return scores;
}

async function jsFetch(urls) {
  const list = Array.isArray(urls) ? urls : [urls];
  const results = {};
  for (const url of list) {
    try {
      const res = await fetch(url, { signal: AbortSignal.timeout(10000) });
      results[url] = `${res.status} — ${(await res.text()).length} bytes`;
    } catch (err) {
      results[url] = `error: ${err.message}`;
    }
  }
  return results;
}
