import logger from '../utils/logger.js';

const MAX_BATCH_TOKENS = 32000;
const MIN_BATCH_SIZE = 3;

class ContextBatcher {
  constructor() {
    this._queue = [];
    this._batches = 0;
    this._tokensSaved = 0;
    this._enabled = false;
  }

  activate() {
    this._enabled = true;
    logger.info('context-batcher: active');
    return { active: true, maxBatchTokens: MAX_BATCH_TOKENS };
  }

  deactivate() {
    this._enabled = false;
    logger.info('context-batcher: deactivated');
  }

  enqueue(task) {
    if (!this._enabled) return task;
    this._queue.push({
      ...task,
      _enqueuedAt: Date.now(),
      _id: `task-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    });
    return this._queue.length >= MIN_BATCH_SIZE
      ? this.flush()
      : null;
  }

  flush() {
    if (this._queue.length === 0) return [];
    const batches = [];
    while (this._queue.length > 0) {
      const batch = this._queue.splice(0, Math.min(5, this._queue.length));
      batches.push(this._buildBatch(batch));
    }
    this._batches += batches.length;
    const savings = batches.reduce((a, b) => a + b.estimatedTokensSaved, 0);
    this._tokensSaved += savings;
    logger.info(`context-batcher: flushed ${batches.length} batches, saved ~${savings} tokens`);
    return batches;
  }

  batchTasks(tasks) {
    if (!this._enabled || tasks.length < MIN_BATCH_SIZE) return tasks.map(t => ({ ...t, type: 'single' }));
    const batches = [];
    const sorted = [...tasks].sort((a, b) => (b.priority || 0) - (a.priority || 0));
    for (let i = 0; i < sorted.length; i += 5) {
      const slice = sorted.slice(i, i + 5);
      batches.push(this._buildBatch(slice));
    }
    this._batches += batches.length;
    const savings = batches.reduce((a, b) => a + b.estimatedTokensSaved, 0);
    this._tokensSaved += savings;
    logger.info(`context-batcher: batched ${tasks.length} tasks into ${batches.length} groups, saved ~${savings} tokens`);
    return batches;
  }

  buildPromptBatch(tasks) {
    if (tasks.length === 0) return '';
    if (tasks.length === 1) return tasks[0].prompt || '';
    const header = `Process the following ${tasks.length} tasks in order. Respond with a JSON array where each element corresponds to the task index.\n\n`;
    const sections = tasks.map((t, i) => {
      return `[TASK ${i + 1}]\nType: ${t.type || 'general'}\nContext: ${t.context || ''}\nInstruction: ${t.prompt || t.instruction || ''}\n`;
    });
    return header + sections.join('\n') + `\n\nRespond ONLY with a valid JSON array of ${tasks.length} results.`;
  }

  parseBatchResponse(response, expectedCount) {
    if (!response) return [];
    try {
      const parsed = JSON.parse(response);
      if (Array.isArray(parsed) && parsed.length === expectedCount) {
        return parsed.map((r, i) => ({ index: i, result: r }));
      }
    } catch {}
    const fallback = response.split('\n').filter(l => l.trim()).map(l => ({ result: l }));
    return fallback.slice(0, expectedCount);
  }

  queueLength() {
    return this._queue.length;
  }

  getBatcherStatus() {
    return {
      enabled: this._enabled,
      batchesCreated: this._batches,
      tokensSaved: this._tokensSaved,
      queueDepth: this._queue.length,
      minBatchSize: MIN_BATCH_SIZE,
      maxBatchTokens: MAX_BATCH_TOKENS,
    };
  }

  _buildBatch(tasks) {
    const estimatedTokens = tasks.reduce((a, t) => {
      const tc = ((t.prompt || t.context || t.instruction || '') + (JSON.stringify(t) || '')).length / 4;
      return a + Math.ceil(tc);
    }, 0);
    const individualOverhead = estimatedTokens * 2;
    const batchTokens = estimatedTokens + 50;
    const tokensSaved = Math.max(0, individualOverhead - batchTokens);
    return {
      id: `batch-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
      tasks: tasks.map(t => t._id || t.id),
      taskCount: tasks.length,
      estimatedTokens,
      tokensSaved,
      prompt: this.buildPromptBatch(tasks),
      createdAt: Date.now(),
    };
  }
}

const contextBatcher = new ContextBatcher();
export default contextBatcher;
