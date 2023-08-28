FROM node:16-slim

WORKDIR /app

# Copy package files and install dependencies
COPY package*.json ./
RUN npm install

# Copy the rest of the app files
COPY . .

EXPOSE 3000
CMD ["npm", "run", "start"]