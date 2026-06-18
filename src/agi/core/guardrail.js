import { PROTECTED_FILES } from '../../config/systemConfig.js';
import path from 'path';

export function isSafeToModify(filePath) {
    const fileName = path.basename(filePath);
    if (PROTECTED_FILES.includes(fileName)) {
        console.error(`SECURITY ALERT: Attempted unauthorized modification of ${fileName}`);
        return false;
    }
    return true;
}

export function validateCode(code) {
    // هنا نضيف منطق فحص الكود قبل تشغيله
    if (code.includes('rm -rf /') || code.includes('process.exit()')) {
        return false;
    }
    return true;
}
