FROM node:18-slim
RUN apt-get update && apt-get install -y curl supervisor \
    && curl -fsSL https://ollama.com/install.sh | sh
WORKDIR /app
COPY . .
RUN npm install
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
CMD ["/usr/bin/supervisord"]
