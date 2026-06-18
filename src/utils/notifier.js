import { exec } from 'child_process';

export function notifySuccess(amount, source) {
    console.log(`[EARNING] New profit of $${amount} from ${source}`);
    const token = process.env.TELEGRAM_BOT_TOKEN;
    const chatId = process.env.TELEGRAM_USER_ID;
    if (token && chatId && token !== 'your_telegram_bot_token') {
        exec(`curl -s "https://api.telegram.org/bot${token}/sendMessage?chat_id=${chatId}&text=Profit: $${amount} from ${source}"`);
    }
}
