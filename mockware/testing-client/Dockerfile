# QT-1 Load Testing Client Dockerfile

FROM node:18-alpine

# Install system dependencies
RUN apk add --no-cache curl bash

# Set working directory
WORKDIR /app

# Create application user
RUN addgroup -g 1001 -S testclient && \
    adduser -S testclient -u 1001 -G testclient

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --only=production && npm cache clean --force

# Copy application files
COPY . .

# Create necessary directories and set permissions
RUN mkdir -p logs data && \
    chown -R testclient:testclient /app

# Switch to non-root user
USER testclient

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

# Set environment variables
ENV NODE_ENV=production \
    PORT=3000 \
    DEFAULT_TARGET_URL=http://host.docker.internal:8080 \
    MOCK_PROVIDER_URL=http://host.docker.internal:8081

# Start the application
CMD ["npm", "start"]