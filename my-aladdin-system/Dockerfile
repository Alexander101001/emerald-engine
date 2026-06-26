FROM python:3.13-slim-bookworm AS base

WORKDIR /app

RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends \
    curl \
    supervisor \
    git \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

RUN chmod +x scripts/*.sh 2>/dev/null || true

ENV PYTHONUNBUFFERED=1
ENV OLLAMA_HOST=http://localhost:11434
ENV OLLAMA_MODEL=qwen2.5-coder:1.5b
ENV HF_HOME=/data/huggingface
ENV ALADDIN_STATE=/data/aladdin

RUN mkdir -p /data/huggingface /data/aladdin /data/logs /var/log/supervisor

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 7860

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
