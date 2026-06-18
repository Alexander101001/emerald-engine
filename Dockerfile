FROM node:18-slim
RUN apt-get update && apt-get install -y git python3
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .

# Start the brain
CMD ["node", "src/agi/brain.js"]
