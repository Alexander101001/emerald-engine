import { uid } from '../utils/helpers.js';
import logger from '../utils/logger.js';

export const AD_CLIENT = process.env.ADSENSE_CLIENT_ID || 'ca-pub-xxxxxxxxxxxx';

export const AD_SIZES = {
  display: { width: 728, height: 90 },
  rectangle: { width: 336, height: 280 },
  skyscraper: { width: 160, height: 600 },
  native: { width: 'auto', height: 'auto' },
};

export function createAdSlot({ slot, format = 'display', adClient } = {}) {
  const id = slot || `emerald-ad-${uid(6)}`;
  const client = adClient || AD_CLIENT;
  const size = AD_SIZES[format] || AD_SIZES.display;
  return {
    id,
    client,
    format,
    size,
    html: generateAdHtml(id, client, format),
    fallback: generateFallback(id),
    antiAdblock: generateAntiAdblock(id),
  };
}

function generateAdHtml(id, client, format) {
  const size = AD_SIZES[format] || AD_SIZES.display;
  return `
<div class="emerald-ad-wrapper" id="wrapper-${id}" style="min-height:${size.height}px;min-width:${size.width}px;margin:1rem auto;text-align:center;overflow:hidden">
  <ins class="adsbygoogle"
       style="display:inline-block;width:${size.width}px;height:${size.height}px"
       data-ad-client="${client}"
       data-ad-slot="${id}"
       data-ad-format="${format}"
       data-full-width-responsive="true"></ins>
  <script>
    (adsbygoogle = window.adsbygoogle || []).push({});
  </script>
  <noscript>
    <div class="ad-fallback" style="padding:1rem;background:#f5f5f5;border:1px solid #ddd;border-radius:4px">
      <p style="margin:0;font-size:0.9rem;color:#666">
        🔒 Ad blocked — <a href="https://support.emerald.app/enable-ads" style="color:#667eea">whitelist us</a>
        or <a href="/upgrade" style="color:#667eea">go ad-free</a>
      </p>
    </div>
  </noscript>
</div>`;
}

function generateFallback(id) {
  return `
<div id="fallback-${id}" class="emerald-fallback" style="min-height:90px;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,#f5f7fa,#c3cfe2);border-radius:8px;margin:1rem auto;padding:1rem;max-width:728px">
  <div style="text-align:center">
    <span style="font-size:1.2rem">💎</span>
    <p style="margin:0.5rem 0 0;font-size:0.9rem;color:#555">
      <strong>Emerald Pro</strong> — Remove ads for $3/mo
    </p>
    <a href="/subscribe" style="display:inline-block;margin-top:0.5rem;padding:0.4rem 1.2rem;background:#667eea;color:#fff;border-radius:4px;text-decoration:none;font-size:0.85rem">Upgrade</a>
  </div>
</div>`;
}

function generateAntiAdblock(id) {
  return `
<script>
(function() {
  var wrapper = document.getElementById('wrapper-${id}');
  if (!wrapper) return;
  var test = document.createElement('div');
  test.className = 'adsbygoogle';
  test.style.cssText = 'height:1px;width:1px;position:absolute;left:-9999px';
  wrapper.appendChild(test);
  var check = function() {
    if (test.offsetParent === null || test.offsetHeight === 0) {
      wrapper.innerHTML = \`${generateFallback(id).replace(/`/g, '\\`')}\`;
      wrapper.style.display = 'block';
      if (window.dispatchEvent) {
        window.dispatchEvent(new CustomEvent('emerald:adblock-detected', { detail: { slot: '${id}' } }));
      }
    }
    test.remove();
  };
  if (document.readyState === 'complete') check();
  else window.addEventListener('load', check);
})();
</script>`;
}

export function injectAdSlots(html, count = 3, formats = ['display', 'rectangle', 'native']) {
  let result = html;
  const slots = [];

  for (let i = 0; i < count; i++) {
    const fmt = formats[i % formats.length];
    const slot = createAdSlot({ format: fmt });
    slots.push(slot);

    const marker = `<!-- AD_SLOT_${i} -->`;
    const snippet = `${slot.html}\n${slot.antiAdblock}`;
    result = result.replace(marker, snippet);
  }

  if (!result.includes('AD_SLOT_')) {
    const ctaIdx = result.lastIndexOf('</main>');
    if (ctaIdx !== -1) {
      const injection = slots.map(s => `${s.html}\n${s.antiAdblock}`).join('\n');
      result = result.slice(0, ctaIdx) + '<section class="emerald-ads">\n' + injection + '\n</section>\n' + result.slice(ctaIdx);
    }
  }

  logger.info(`adsense: injected ${slots.length} ad slots`);
  return { html: result, slots };
}

export function generateAdBlockWall() {
  return `
<style>
.emerald-adblock-wall {
  position: fixed; top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0,0,0,0.85); z-index: 99999;
  display: flex; align-items: center; justify-content: center;
}
.emerald-adblock-wall > div {
  background: #fff; padding: 2rem; border-radius: 12px;
  max-width: 400px; text-align: center;
}
.emerald-adblock-wall button {
  padding: 0.6rem 2rem; background: #667eea; color: #fff;
  border: none; border-radius: 6px; cursor: pointer; margin-top: 1rem;
}
</style>
<div class="emerald-adblock-wall" id="emerald-adblock-wall">
  <div>
    <h2>🛑 Ad Blocker Detected</h2>
    <p>Please disable your ad blocker to support this free service.</p>
    <button onclick="document.getElementById('emerald-adblock-wall').remove()">
      I've disabled it
    </button>
    <p style="margin-top:1rem;font-size:0.8rem;color:#999">
      Or <a href="/subscribe">subscribe for $3/mo</a> to go ad-free
    </p>
  </div>
</div>`;
}

export default { createAdSlot, injectAdSlots, generateAdBlockWall, AD_CLIENT, AD_SIZES };
