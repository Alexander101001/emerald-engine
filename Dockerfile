FROM node:18-slim

RUN apt-get update && apt-get install -y git python3 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

RUN npm install

CMD ["node", "src/agi/brain.js"]
