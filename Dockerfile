FROM golang:1.26 AS builder
WORKDIR /build
COPY go.mod .
COPY src/engine/*.go .
RUN go build -o emerald-engine .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y git supervisor nodejs ca-certificates && rm -rf /var/lib/apt/lists/* && update-ca-certificates
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
