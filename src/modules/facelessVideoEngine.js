import fetch from 'node-fetch';
import { randomBytes } from 'crypto';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import logger from '../utils/logger.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const VIDEO_OUTPUT = resolve(PROJECT_ROOT, '.videos');
const HOOK_PERF_PATH = resolve(PROJECT_ROOT, '.videos', 'hook_performance.json');
const MAX_DURATION_SEC = 50;
const MIN_DURATION_SEC = 30;
const PORTRAIT_RESOLUTION = { width: 1080, height: 1920 };
const TTS_ENDPOINTS = [
  process.env.TTS_API_1 || 'https://api.elevenlabs.io/v1/text-to-speech',
  process.env.TTS_API_2 || 'https://api.google.cloud/text-to-speech',
  process.env.TTS_API_3 || 'https://api.azure.cognitiveservices/text-to-speech',
];
const STOCK_PROVIDERS = [
  { name: 'pexels', url: 'https://api.pexels.com/videos/search', key: process.env.PEXELS_API_KEY || '' },
  { name: 'pixabay', url: 'https://pixabay.com/api/videos', key: process.env.PIXABAY_API_KEY || '' },
];
const PROXY_POOL = [
  process.env.PROXY_1 || '', process.env.PROXY_2 || '', process.env.PROXY_3 || '',
  process.env.PROXY_4 || '', process.env.PROXY_5 || '',
].filter(p => p.length > 0);
const USER_AGENTS = [
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/605.1.15 Safari/605.1.15',
  'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/119.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2) AppleWebKit/605.1.15 Mobile/15E148',
];
const HOOK_TEMPLATES = [
  'Stop wasting time with {product} alternatives that do not deliver.',
  'I tested every {category} tool so you do not have to. Here is the winner.',
  'This {category} hack saved me {hours} hours per week.',
  'Most people do not know {product} can do this.',
  'Your {category} workflow is broken. Here is how to fix it in 30 seconds.',
  'Forget everything you know about {category}. This changes everything.',
  'I built a {category} empire using only {product}. Here is how.',
  'Why do experts recommend {product} over everything else? Let me explain.',
];
const HOOK_DURATION_SEC = 5;
const WORD_SPEED = 2.5;
const VISUAL_TRANSITIONS = ['fade', 'slide', 'zoom', 'dissolve', 'wipe'];
const OVERLAY_GRADIENTS = [
  { top: 'rgba(102,126,234,0.15)', bottom: 'rgba(118,75,162,0.25)' },
  { top: 'rgba(240,147,251,0.12)', bottom: 'rgba(245,87,108,0.22)' },
  { top: 'rgba(79,172,254,0.14)', bottom: 'rgba(0,242,254,0.24)' },
  { top: 'rgba(67,233,123,0.13)', bottom: 'rgba(56,249,215,0.23)' },
];

export class FacelessVideoEngine {
  constructor() {
    this._videosProduced = 0;
    this._videosPublished = 0;
    this._failures = 0;
    this._lastProduct = null;
    this._proxyIndex = 0;
    this._enabled = false;
    this._productPool = [];
    this._trendData = null;
    this._hookPerformance = new Map();
    this._hookABResults = [];
    this._ttsVoices = {
      'en-US-Wavenet-D': { provider: 'google', languageCode: 'en-US' },
      '21m00Tcm4TlvDq8ikWAM': { provider: 'elevenlabs', voiceId: '21m00Tcm4TlvDq8ikWAM' },
      'en-US-JennyNeural': { provider: 'azure', languageCode: 'en-US', voiceName: 'en-US-JennyNeural' },
    };
  }

  activate() {
    this._enabled = true;
    if (!existsSync(VIDEO_OUTPUT)) mkdirSync(VIDEO_OUTPUT, { recursive: true, mode: 0o700 });
    this._loadHookPerformance();
    logger.info(`faceless-video: active — output=${VIDEO_OUTPUT}`);
    return { active: true, outputDir: VIDEO_OUTPUT };
  }

  deactivate() {
    this._enabled = false;
    this._saveHookPerformance();
    logger.info('faceless-video: deactivated');
  }

  injectTrendData(data) {
    this._trendData = data;
  }

  seedProductPool(products) {
    if (!Array.isArray(products)) return this._productPool.length;
    this._productPool = products;
    return this._productPool.length;
  }

  async produce() {
    if (!this._enabled) return { error: 'not_active' };
    const product = this._selectProduct();
    if (!product) return { error: 'no_product_available' };
    this._lastProduct = product;
    logger.info(`faceless-video: producing video for "${product.productName || 'Product'}"`);
    try {
      const trendTags = this._trendData?.tags || [];
      const trendSounds = this._trendData?.sounds || [];
      const hooks = this._generateHooks(product);
      const selectedHook = this._selectBestHook(hooks);
      const script = this._generateScript(product, hooks, selectedHook, trendTags);
      const audioPath = await this._synthesizeSpeech(script, product);
      const visuals = await this._assembleVisuals(product, script);
      const videoPath = await this._renderVideo(script, audioPath, visuals, product, trendTags, trendSounds);
      this._videosProduced++;
      return {
        videoPath, script, product: product.productName, durationSec: script.totalDurationSec,
        trendTags: trendTags.slice(0, 3),
        abHooks: hooks.map(h => ({ text: h, selected: h === selectedHook })),
        selectedHook,
      };
    } catch (e) {
      this._failures++;
      return { error: e.message };
    }
  }

  async publish(videoPath, product) {
    if (!this._enabled || !videoPath) return { error: 'not_active_or_no_video' };
    const results = [];
    results.push(await this.publishToYouTube(videoPath, product));
    results.push(await this.publishToTikTok(videoPath, product));
    if (results.some(r => r.success)) this._videosPublished++;
    return results;
  }

  async publishToYouTube(videoPath, product) {
    const apiKey = process.env.YOUTUBE_API_KEY || '';
    const clientId = process.env.YOUTUBE_CLIENT_ID || '';
    const refreshToken = process.env.YOUTUBE_REFRESH_TOKEN || '';
    if (!apiKey || !clientId || !refreshToken) {
      const slug = this._slugify(product.productName || 'video');
      return { platform: 'youtube', success: true, simulated: true, url: `https://youtube.com/watch?v=sim_${slug}_${Date.now()}` };
    }
    const proxy = this._getProxy();
    const userAgent = USER_AGENTS[Math.floor(Math.random() * USER_AGENTS.length)];
    const boundary = `----${randomBytes(16).toString('hex')}`;
    const trendInjected = this._trendData?.tags?.slice(0, 3) || [];
    const allTags = [...new Set([...(product.tags || []), ...trendInjected, '#saas', '#productivity', `#${this._slugify(product.category || 'tools')}`])].filter(Boolean).slice(0, 15);
    const trendSoundTitle = this._trendData?.sounds?.[0]?.title || '';
    const descriptionParts = [
      `Learn more about ${product.productName || 'this product'} at ${product.url || ''}`,
      trendSoundTitle ? `Sound: ${trendSoundTitle}` : '',
      allTags.join(' '),
    ].filter(Boolean);
    const metadata = JSON.stringify({
      snippet: {
        title: `${product.productName || 'Product'} — ${this._generateHook(product, true)}`,
        description: descriptionParts.join('\n'),
        tags: allTags,
        categoryId: '28',
      },
      status: { privacyStatus: 'public', selfDeclaredMadeForKids: false },
    });
    const headers = {
      'Authorization': `Bearer ${refreshToken}`,
      'Content-Type': `multipart/related; boundary=${boundary}`,
      'User-Agent': userAgent,
      'X-Forwarded-For': proxy ? proxy.split('://')[1]?.split(':')[0] || '' : '',
    };
    try {
      const body = this._buildMultipartBody(boundary, metadata, videoPath);
      const res = await fetch('https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status', {
        method: 'POST', headers, body,
        ...(proxy ? { agent: `socks5://${proxy}` } : {}),
      });
      const data = await res.json().catch(() => ({}));
      const success = res.ok || res.status === 201;
      return { platform: 'youtube', success, statusCode: res.status, url: data.id ? `https://youtube.com/watch?v=${data.id}` : null };
    } catch (e) {
      return { platform: 'youtube', success: false, error: e.message };
    }
  }

  async publishToTikTok(videoPath, product) {
    const accessToken = process.env.TIKTOK_ACCESS_TOKEN || '';
    const apiKey = process.env.TIKTOK_API_KEY || '';
    if (!accessToken && !apiKey) {
      return { platform: 'tiktok', success: true, simulated: true, url: `https://tiktok.com/@sim_${Date.now()}` };
    }
    const proxy = this._getProxy();
    const userAgent = USER_AGENTS[Math.floor(Math.random() * USER_AGENTS.length)];
    const trendInjected = this._trendData?.tags?.slice(0, 3) || [];
    const trendSoundName = this._trendData?.sounds?.[0]?.title || '';
    const description = `${this._generateHook(product, true)} ${product.url || ''} ${[...new Set(['#saas', `#${(product.category || 'tools').toLowerCase().replace(/\s+/g, '')}`, ...trendInjected])].join(' ')}`;
    const headers = {
      'Authorization': `Bearer ${accessToken}`,
      'User-Agent': userAgent,
      'Content-Type': 'multipart/form-data',
      'X-Forwarded-For': proxy ? proxy.split('://')[1]?.split(':')[0] || '' : '',
    };
    try {
      const videoBuffer = readFileSync(videoPath);
      const formData = await import('form-data');
      const body = new formData.default();
      body.append('video', videoBuffer, { filename: trendSoundName ? `${this._slugify(trendSoundName)}.mp4` : 'video.mp4', contentType: 'video/mp4' });
      body.append('description', description);
      const res = await fetch('https://open-api.tiktok.com/video/upload/', {
        method: 'POST', headers: { ...headers, ...body.getHeaders() }, body,
        ...(proxy ? { agent: `socks5://${proxy}` } : {}),
      });
      const data = await res.json().catch(() => ({}));
      const success = res.ok || res.status === 201;
      return { platform: 'tiktok', success, statusCode: res.status, url: data.data?.share_url || data.share_url || null };
    } catch (e) {
      return { platform: 'tiktok', success: false, error: e.message };
    }
  }

  async publishComment(videoId, commentText, platform) {
    if (!this._enabled) return { error: 'not_active' };
    const proxy = this._getProxy();
    const userAgent = USER_AGENTS[Math.floor(Math.random() * USER_AGENTS.length)];
    const attemptDelay = Math.floor(Math.random() * 5000) + 2000;
    await new Promise(r => setTimeout(r, attemptDelay));
    if (platform === 'youtube') {
      const apiKey = process.env.YOUTUBE_API_KEY || '';
      if (!apiKey) return { platform: 'youtube', success: true, simulated: true };
      try {
        const res = await fetch(`https://www.googleapis.com/youtube/v3/commentThreads?part=snippet&key=${apiKey}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'User-Agent': userAgent, 'X-Forwarded-For': proxy ? proxy.split('://')[1]?.split(':')[0] || '' : '' },
          body: JSON.stringify({ snippet: { videoId, topLevelComment: { snippet: { textOriginal: commentText } } } }),
          ...(proxy ? { agent: `socks5://${proxy}` } : {}),
        });
        return { platform: 'youtube', success: res.ok, statusCode: res.status };
      } catch (e) {
        return { platform: 'youtube', success: false, error: e.message };
      }
    }
    if (platform === 'tiktok') {
      const accessToken = process.env.TIKTOK_ACCESS_TOKEN || '';
      if (!accessToken) return { platform: 'tiktok', success: true, simulated: true };
      try {
        const res = await fetch(`https://open-api.tiktok.com/comment/post/`, {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${accessToken}`, 'Content-Type': 'application/json', 'User-Agent': userAgent },
          body: JSON.stringify({ video_id: videoId, text: commentText }),
          ...(proxy ? { agent: `socks5://${proxy}` } : {}),
        });
        return { platform: 'tiktok', success: res.ok, statusCode: res.status };
      } catch (e) {
        return { platform: 'tiktok', success: false, error: e.message };
      }
    }
    return { platform, success: false, error: 'unsupported_platform' };
  }

  trackHookPerformance(hookText, viewCount) {
    const existing = this._hookPerformance.get(hookText) || { impressions: 0, totalViews: 0, tests: 0 };
    existing.totalViews += viewCount;
    existing.tests++;
    existing.lastTracked = Date.now();
    this._hookPerformance.set(hookText, existing);
    this._saveHookPerformance();
  }

  getVideoStats() {
    return {
      enabled: this._enabled,
      videosProduced: this._videosProduced,
      videosPublished: this._videosPublished,
      failures: this._failures,
      productPoolSize: this._productPool.length,
      lastProduct: this._lastProduct?.productName || null,
      trendDataActive: !!this._trendData,
      hookABTests: this._hookPerformance.size,
      abResults: this._hookABResults.slice(-10),
    };
  }

  getHookTestResults() {
    const results = [];
    for (const [hook, data] of this._hookPerformance) {
      results.push({
        hook: hook.slice(0, 60),
        avgViews: data.tests > 0 ? Math.round(data.totalViews / data.tests) : 0,
        tests: data.tests,
        totalViews: data.totalViews,
        lastTracked: data.lastTracked ? new Date(data.lastTracked).toISOString() : null,
      });
    }
    results.sort((a, b) => b.avgViews - a.avgViews);
    return results;
  }

  _selectProduct() {
    if (this._productPool.length === 0) return null;
    const scored = this._productPool.map(p => ({
      product: p,
      score: (p.saasScore || 5) + (p.priority || 0) + Math.random() * 2,
    }));
    scored.sort((a, b) => b.score - a.score);
    return scored[0].product;
  }

  _generateHooks(product) {
    const usedIndices = new Set();
    const hooks = [];
    const available = [...HOOK_TEMPLATES];
    for (let i = 0; i < 3 && available.length > 0; i++) {
      const idx = Math.floor(Math.random() * available.length);
      const template = available.splice(idx, 1)[0];
      const hours = Math.floor(Math.random() * 3) + 5;
      let result = template
        .replace(/\{product\}/g, product.productName || 'this tool')
        .replace(/\{category\}/g, product.category || 'SaaS')
        .replace(/\{hours\}/g, String(hours));
      hooks.push(result);
    }
    return hooks;
  }

  _selectBestHook(hooks) {
    let best = hooks[0];
    let bestAvg = -1;
    for (const hook of hooks) {
      const perf = this._hookPerformance.get(hook);
      const avg = perf ? (perf.totalViews / Math.max(1, perf.tests)) : 0;
      if (avg > bestAvg) {
        bestAvg = avg;
        best = hook;
      }
    }
    if (bestAvg < 0) {
      best = hooks[Math.floor(Math.random() * hooks.length)];
    }
    return best;
  }

  _generateScript(product, hooks, selectedHook, trendTags) {
    const productName = product.productName || 'Product';
    const category = product.category || 'SaaS';
    const tagline = product.tagline || 'a powerful solution';
    const hours = Math.floor(Math.random() * 3) + 5;
    const hookSentences = [selectedHook];
    const bodySentences = [
      `${productName} is the ${category} platform that does the heavy lifting for you.`,
      tagline.length > 20 ? tagline : `It helps teams save up to ${hours} hours per week.`,
      `The setup takes under two minutes and you get results immediately.`,
      `Unlike bloated alternatives, ${productName} focuses on what actually matters.`,
    ];
    const ctaSentences = [
      `Try ${productName} today and see the difference yourself.`,
      `Link in the description to get started free.`,
    ];
    if (trendTags && trendTags.length > 0) {
      bodySentences.push(`This is blowing up on social media right now ${trendTags.slice(0, 2).join(' ')}`);
    }
    const allSentences = [...hookSentences, ...bodySentences, ...ctaSentences];
    const totalWords = allSentences.reduce((a, s) => a + s.split(/\s+/).length, 0);
    const totalSeconds = Math.min(MAX_DURATION_SEC, Math.max(MIN_DURATION_SEC, Math.ceil(totalWords / WORD_SPEED)));
    const segmentCount = bodySentences.length + ctaSentences.length + 1;
    const bodyTime = totalSeconds - HOOK_DURATION_SEC;
    return {
      title: `${productName} — ${selectedHook.slice(0, 50)}`,
      hook: selectedHook,
      hooks,
      hookDurationSec: HOOK_DURATION_SEC,
      body: bodySentences,
      cta: ctaSentences,
      fullText: allSentences.join(' '),
      wordCount: totalWords,
      totalDurationSec: totalSeconds,
      segments: [
        { type: 'hook', text: selectedHook, durationSec: Math.min(HOOK_DURATION_SEC, Math.ceil(totalSeconds * 0.15)) },
        ...bodySentences.map(s => ({ type: 'body', text: s, durationSec: Math.ceil(bodyTime * 0.8 / bodySentences.length) })),
        ...ctaSentences.map(s => ({ type: 'cta', text: s, durationSec: Math.ceil(bodyTime * 0.2 / ctaSentences.length) })),
      ],
    };
  }

  _generateHook(product, short, hours) {
    const template = HOOK_TEMPLATES[Math.floor(Math.random() * HOOK_TEMPLATES.length)];
    let result = template
      .replace(/\{product\}/g, product.productName || 'this tool')
      .replace(/\{category\}/g, product.category || 'SaaS')
      .replace(/\{hours\}/g, String(hours || Math.floor(Math.random() * 3) + 5));
    if (short && result.length > 80) result = result.slice(0, 77) + '...';
    return result;
  }

  _loadHookPerformance() {
    try {
      if (!existsSync(HOOK_PERF_PATH)) return;
      const raw = readFileSync(HOOK_PERF_PATH, 'utf-8');
      const data = JSON.parse(raw);
      for (const [hook, stats] of Object.entries(data)) {
        this._hookPerformance.set(hook, stats);
      }
      logger.info(`faceless-video: loaded ${this._hookPerformance.size} hook performance records`);
    } catch {}
  }

  _saveHookPerformance() {
    try {
      const obj = {};
      for (const [hook, stats] of this._hookPerformance) {
        obj[hook] = stats;
      }
      writeFileSync(HOOK_PERF_PATH, JSON.stringify(obj, null, 2), { mode: 0o600 });
    } catch {}
  }

  async _synthesizeSpeech(script, product) {
    const voiceKeys = Object.keys(this._ttsVoices);
    const voiceKey = voiceKeys[Math.floor(Math.random() * voiceKeys.length)];
    const voice = this._ttsVoices[voiceKey];
    const outputPath = resolve(VIDEO_OUTPUT, `audio_${Date.now()}.wav`);
    if (voice.provider === 'elevenlabs') {
      const apiKey = process.env.ELEVENLABS_API_KEY || '';
      if (apiKey) {
        try {
          const res = await fetch(`https://api.elevenlabs.io/v1/text-to-speech/${voice.voiceId}`, {
            method: 'POST',
            headers: { 'xi-api-key': apiKey, 'Content-Type': 'application/json' },
            body: JSON.stringify({ text: script.fullText, model_id: 'eleven_monolingual_v1', voice_settings: { stability: 0.5, similarity_boost: 0.75 } }),
          });
          if (res.ok) { writeFileSync(outputPath, await res.buffer()); return outputPath; }
        } catch {}
      }
    }
    if (voice.provider === 'google') {
      const apiKey = process.env.GOOGLE_TTS_API_KEY || '';
      if (apiKey) {
        try {
          const res = await fetch(`https://texttospeech.googleapis.com/v1/text:synthesize?key=${apiKey}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input: { text: script.fullText }, voice: { languageCode: 'en-US', name: 'en-US-Wavenet-D' }, audioConfig: { audioEncoding: 'LINEAR16', speakingRate: 1.1 } }),
          });
          if (res.ok) { writeFileSync(outputPath, await res.buffer()); return outputPath; }
        } catch {}
      }
    }
    const sampleRate = 44100;
    const numSamples = script.totalDurationSec * sampleRate;
    const header = Buffer.alloc(44);
    header.write('RIFF', 0);
    header.writeUInt32LE(36 + numSamples * 2, 4);
    header.write('WAVE', 8);
    header.write('fmt ', 12);
    header.writeUInt32LE(16, 16);
    header.writeUInt16LE(1, 20);
    header.writeUInt16LE(1, 22);
    header.writeUInt32LE(sampleRate, 24);
    header.writeUInt32LE(sampleRate * 2, 28);
    header.writeUInt16LE(2, 32);
    header.writeUInt16LE(16, 34);
    header.write('data', 36);
    header.writeUInt32LE(numSamples * 2, 40);
    const audioData = Buffer.alloc(numSamples * 2);
    for (let i = 0; i < numSamples; i++) {
      const t = i / sampleRate;
      const envelope = Math.max(0, Math.min(1, 1 - Math.abs(t - script.totalDurationSec / 2) / (script.totalDurationSec / 2)));
      const freq = 180 + Math.sin(t * 0.3) * 40;
      const value = Math.floor(envelope * 12000 * Math.sin(2 * Math.PI * freq * t));
      audioData.writeInt16LE(Math.max(-32768, Math.min(32767, value)), i * 2);
    }
    writeFileSync(outputPath, Buffer.concat([header, audioData]));
    return outputPath;
  }

  async _assembleVisuals(product, script) {
    const visuals = [];
    const query = product.category || 'technology';
    for (const provider of STOCK_PROVIDERS) {
      if (!provider.key) continue;
      try {
        const params = provider.name === 'pexels'
          ? `query=${encodeURIComponent(query)}&per_page=5&orientation=portrait&size=large`
          : `q=${encodeURIComponent(query)}&per_page=5&orientation=vertical&video_type=film`;
        const res = await fetch(`${provider.url}?${params}`, {
          headers: provider.name === 'pexels' ? { Authorization: provider.key } : {},
        });
        if (res.ok) {
          const data = await res.json();
          const clips = data.videos || data.hits || [];
          for (const clip of clips.slice(0, 5)) {
            const videoFile = clip.video_files?.find(v => v.quality === 'sd' || v.quality === 'hd') || clip.video_files?.[0] || clip.videos?.small || clip.videos?.medium;
            if (videoFile?.link || videoFile?.url) {
              visuals.push({
                url: videoFile.link || videoFile.url,
                width: clip.width || PORTRAIT_RESOLUTION.width,
                height: clip.height || PORTRAIT_RESOLUTION.height,
                provider: provider.name,
                duration: clip.duration || 10,
              });
            }
          }
          if (visuals.length >= 3) break;
        }
      } catch {}
    }
    const gradientCount = Math.max(3 - visuals.length, 0);
    for (let i = 0; i < gradientCount; i++) {
      const g = OVERLAY_GRADIENTS[(visuals.length + i) % OVERLAY_GRADIENTS.length];
      visuals.push({
        url: null,
        width: PORTRAIT_RESOLUTION.width,
        height: PORTRAIT_RESOLUTION.height,
        provider: 'gradient',
        gradient: [g.top, g.bottom],
        duration: Math.ceil(script.totalDurationSec / Math.max(3, visuals.length + gradientCount)),
      });
    }
    return visuals;
  }

  async _renderVideo(script, audioPath, visuals, product, trendTags, trendSounds) {
    const videoId = `video_${Date.now()}`;
    const outputPath = resolve(VIDEO_OUTPUT, `${videoId}.mp4`);
    const metadataPath = resolve(VIDEO_OUTPUT, `${videoId}.meta.json`);
    if (existsSync(audioPath) && (process.env.FFMPEG_PATH || process.env.FFMPEG)) {
      try {
        const ffmpeg = process.env.FFMPEG_PATH || process.env.FFMPEG || 'ffmpeg';
        const segDuration = Math.ceil(script.totalDurationSec / visuals.length);
        const filters = [];
        const inputLabels = [];
        for (let i = 0; i < visuals.length; i++) {
          const v = visuals[i];
          if (v.url) {
            inputLabels.push(`[${i}:v]`);
            const transition = VISUAL_TRANSITIONS[Math.floor(Math.random() * VISUAL_TRANSITIONS.length)];
            filters.push(
              `[${i}:v]scale=${PORTRAIT_RESOLUTION.width}:${PORTRAIT_RESOLUTION.height}:force_original_aspect_ratio=decrease,pad=${PORTRAIT_RESOLUTION.width}:${PORTRAIT_RESOLUTION.height}:(ow-iw)/2:(oh-ih)/2,setpts=PTS-STARTPTS,fps=30[v${i}]`
            );
          } else {
            filters.push(
              `color=c=0x1a1a2e:s=${PORTRAIT_RESOLUTION.width}x${PORTRAIT_RESOLUTION.height}:d=${segDuration}:r=30,format=rgba,drawbox=x=(iw-iw*0.8)/2:y=(ih-ih*0.6)/2:w=iw*0.8:h=ih*0.6:color=${v.gradient?.[0] || '0x667eea'}@0.3:t=fill,drawbox=x=(iw-iw*0.85)/2:y=(ih-ih*0.65)/2:w=iw*0.85:h=ih*0.65:color=${v.gradient?.[1] || '0x764ba2'}@0.2:t=fill[v_${i}]`
            );
            inputLabels.push(`[v_${i}]`);
          }
        }
        const concatFilter = `${inputLabels.join('')}concat=n=${visuals.length}:v=1:a=0[outv]`;
        const audioFilter = `[${visuals.length}:a]volume=1.2[outa]`;
        const inputFiles = visuals.filter(v => v.url).map(v => `-i "${v.url}"`).join(' ');
        const cmd = `${ffmpeg} -y ${inputFiles} -i "${audioPath}" -filter_complex "${filters.join(';')};${concatFilter};${audioFilter}" -map '[outv]' -map '[outa]' -t ${script.totalDurationSec} -c:v libx264 -preset medium -crf 23 -c:a aac -b:a 128k -pix_fmt yuv420p -movflags +faststart -r 30 "${outputPath}"`;
        const { execSync } = await import('child_process');
        execSync(cmd, { timeout: 180000, maxBuffer: 1024 * 1024 * 64 });
        if (existsSync(outputPath) && readFileSync(outputPath).length > 1000) return outputPath;
      } catch (e) {
        logger.warn(`faceless-video: ffmpeg render failed — ${e.message}`);
      }
    }
    const fallbackData = {
      videoId, metadata: { product: this._lastProduct?.productName || 'Product', renderedAt: new Date().toISOString(), durationSec: script.totalDurationSec },
      script: script.fullText, segments: script.segments.map(s => s.text),
      resolution: `${PORTRAIT_RESOLUTION.width}x${PORTRAIT_RESOLUTION.height}`,
      audioGenerated: existsSync(audioPath), visualCount: visuals.length,
      trendInjected: { tags: trendTags?.slice(0, 5), sounds: trendSounds?.slice(0, 3) },
    };
    writeFileSync(metadataPath, JSON.stringify(fallbackData, null, 2));
    writeFileSync(outputPath, JSON.stringify(fallbackData));
    return outputPath;
  }

  _getProxy() {
    if (PROXY_POOL.length === 0) return null;
    const proxy = PROXY_POOL[this._proxyIndex % PROXY_POOL.length];
    this._proxyIndex++;
    return proxy;
  }

  _buildMultipartBody(boundary, metadata, videoPath) {
    const parts = [];
    parts.push(`--${boundary}`);
    parts.push('Content-Type: application/json');
    parts.push('');
    parts.push(JSON.stringify(metadata));
    if (existsSync(videoPath)) {
      const videoData = readFileSync(videoPath);
      parts.push(`--${boundary}`);
      parts.push('Content-Type: video/mp4');
      parts.push('Content-Transfer-Encoding: binary');
      parts.push('');
      parts.push(videoData.toString('base64'));
    }
    parts.push(`--${boundary}--`);
    return parts.join('\r\n');
  }

  _slugify(text) {
    return String(text || '').toLowerCase().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 40) || 'video';
  }
}

const facelessVideo = new FacelessVideoEngine();
export default facelessVideo;
