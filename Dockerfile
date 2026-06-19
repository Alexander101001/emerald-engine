FROM node:18-slim

# Install Build Tools for Go and Rust
RUN apt-get update && apt-get install -y \
    git \
    python3 \
    golang-go \
    rustc \
    cargo \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

RUN npm install

# Build Go/Rust components if exist
RUN if [ -f main.go ]; then go build -o bin/engine main.go; fi

CMD ["node", "src/agi/brain.js"]
