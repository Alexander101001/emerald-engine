FROM node:18-slim

# Install dependencies and supervisor
RUN apt-get update && apt-get install -y curl supervisor git python3 \
    && curl -fsSL https://ollama.com/install.sh | sh \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Pre-pull the model
RUN ollama serve & sleep 5 && ollama pull qwen2.5:1.5b

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

# Copy supervisor config
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Start supervisor
CMD ["/usr/bin/supervisord"]
