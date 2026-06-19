FROM golang:1.26 AS builder
WORKDIR /build
COPY src/engine/*.go .
COPY go.mod .
RUN go build -o emerald-engine .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y git supervisor nodejs && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/emerald-engine ./emerald-engine
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY src/engine/vault.js ./src/engine/vault.js
COPY api_key ./api_key
COPY public ./public
COPY .git ./.git
COPY .netrc /root/.netrc
RUN chmod 600 /root/.netrc && git config --global user.email "alexander@emerald.app" && git config --global user.name "Alexander"
EXPOSE 8080
CMD ["/usr/bin/supervisord"]
