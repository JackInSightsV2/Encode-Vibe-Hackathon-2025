# QT-1 Mock AI Provider Dockerfile
FROM node:18-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --only=production

# Copy application files
COPY . .

# Azure Container Instances requires root user
# No user creation - runs as root for ACI compatibility

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "const http = require('http'); \
               http.get('http://localhost:8081/health', (res) => { \
                 process.exit(res.statusCode === 200 ? 0 : 1) \
               }).on('error', () => process.exit(1))"

# Start the application
CMD ["npm", "start"]