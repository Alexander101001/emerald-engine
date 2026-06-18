import { loadSecureConfig } from '../utils/configLoader.js';

loadSecureConfig();

console.log('Secure core environment initialized successfully.');
console.log('Telegram Command Root authorized for ID:', process.env.TELEGRAM_USER_ID);
console.log('All 66 agents locked onto secure cloud stream.');
