FROM golang:1.26 AS builder
WORKDIR /build
COPY src/engine/main.go .
RUN go build -o emerald-engine main.go

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y git supervisor && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/emerald-engine ./emerald-engine
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY public ./public
COPY .git ./.git
COPY .netrc /root/.netrc
RUN chmod 600 /root/.netrc && git config --global user.email "alexander@emerald.app" && git config --global user.name "Alexander"
CMD ["/usr/bin/supervisord"]
