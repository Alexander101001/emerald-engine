FROM golang:1.26 AS builder
WORKDIR /build
COPY go.mod .
COPY src/engine/*.go .
RUN go build -o emerald-engine .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y \
    git supervisor nodejs ca-certificates wget gnupg \
    fonts-ipafont-gothic fonts-wqy-zenhei fonts-noto-color-emoji fonts-freefont-ttf \
    --no-install-recommends \
    && wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list \
    && apt-get update && apt-get install -y google-chrome-stable --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates
WORKDIR /app
COPY --from=builder /build/emerald-engine ./emerald-engine
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY src/engine/vault.js ./src/engine/vault.js
COPY api_key ./api_key
COPY public ./public
COPY .git ./.git
RUN git config --global user.email "alexander@emerald.app" && git config --global user.name "Alexander" && git config --global http.sslVerify false && git config --global credential.helper store
EXPOSE 8080
CMD ["/usr/bin/supervisord"]
