# QT-1 Mockware - Unified Container
# Runs both Load Testing Client and Universal Mock Provider in a single container

FROM node:18-alpine

# Install system dependencies (including dos2unix)
RUN apk add --no-cache \
    bash \
    curl \
    supervisor \
    dos2unix \
    && rm -rf /var/cache/apk/*

# Set working directory
WORKDIR /app

# Create application user
RUN addgroup -g 1001 -S mockware && \
    adduser -S mockware -u 1001 -G mockware

# Copy package files for both services
COPY testing-client/package*.json ./testing-client/
COPY mock-provider/package*.json ./mock-provider/
COPY package.json ./

# Install dependencies for both services
RUN cd testing-client && npm install --only=production && \
    cd ../mock-provider && npm install --only=production && \
    cd .. && npm cache clean --force

# Copy application source code
COPY testing-client/ ./testing-client/
COPY mock-provider/ ./mock-provider/
COPY start-mockware.sh ./
COPY docker-entrypoint.sh ./

# CONVERT LINE ENDINGS FOR SCRIPTS
RUN dos2unix start-mockware.sh docker-entrypoint.sh

# Create necessary directories
RUN mkdir -p /app/logs /app/data && \
    chown -R mockware:mockware /app

# Create supervisor configuration
COPY supervisord.conf /etc/supervisord.conf

# Make scripts executable
RUN chmod +x start-mockware.sh docker-entrypoint.sh

# Expose ports
EXPOSE 3000 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:3000/health && curl -f http://localhost:8081/health

# Switch to non-root user
USER mockware

# Set environment variables
ENV NODE_ENV=production \
    TESTING_CLIENT_PORT=3000 \
    MOCK_PROVIDER_PORT=8081 \
    DEFAULT_TARGET_URL=http://host.docker.internal:8080 \
    RESPONSE_DELAY_MIN=10 \
    RESPONSE_DELAY_MAX=50 \
    ERROR_RATE=0.001 \
    ENABLE_ERROR_SIMULATION=false \
    ENABLE_STREAMING=true

# Use supervisor to run both services
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisord.conf", "-n"]