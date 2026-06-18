const levels = { debug: 0, info: 1, warn: 2, error: 3 };
const current = levels[process.env.LOG_LEVEL] ?? levels.info;

function ts() {
  return new Date().toISOString();
}

export const logger = {
  debug(...args) {
    if (current <= 0) console.debug(`[${ts()}] [DEBUG]`, ...args);
  },
  info(...args) {
    if (current <= 1) console.error(`[${ts()}] [INFO]`, ...args);
  },
  warn(...args) {
    if (current <= 2) console.warn(`[${ts()}] [WARN]`, ...args);
  },
  error(...args) {
    if (current <= 3) console.error(`[${ts()}] [ERROR]`, ...args);
  },
};

export default logger;
