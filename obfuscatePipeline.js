const fs = require('fs');
const path = require('path');
const JavaScriptObfuscator = require('javascript-obfuscator');

const targetFile = path.join(__dirname, 'src/agi/lifecycle.js');
const rawCode = fs.readFileSync(targetFile, 'utf8');

const obfuscatedObject = JavaScriptObfuscator.obfuscate(rawCode, {
    compact: true,
    controlFlowFlattening: true,
    controlFlowFlatteningThreshold: 1,
    numbersToExpressions: true,
    simplify: true,
    stringArrayShuffle: true,
    splitStrings: true,
    stringArrayThreshold: 1
});

fs.writeFileSync(targetFile, obfuscatedObject.getObfuscatedCode(), 'utf8');
console.log('Target pipeline matrix has been totally randomized and abstractly compiled.');
