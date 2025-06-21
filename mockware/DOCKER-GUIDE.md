# 🐳 QT-1 Mockware Docker Guide

Multiple Docker deployment options for the QT-1 Mockware testing suite.

## 🎯 Deployment Options

### Option 1: Unified Container (Recommended)
**Single container running both services**

```bash
# Build and run unified container
npm run docker:unified

# Or manually
docker build -t qt1-mockware .
docker run -d -p 3000:3000 -p 8081:8081 --name qt1-mockware qt1-mockware
```

### Option 2: Docker Compose (Multi-Container)
**Separate containers for each service + monitoring**

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Option 3: Individual Containers
**Build and run services separately**

```bash
# Mock Provider only
cd mock-provider
docker build -t qt1-mock-provider .
docker run -d -p 8081:8081 qt1-mock-provider

# Testing Client only
cd testing-client  
docker build -t qt1-testing-client .
docker run -d -p 3000:3000 qt1-testing-client
```

## 🛠️ Unified Container Details

### Features
- **Single Container**: Both services in one container using Supervisor
- **Process Management**: Automatic service restart and health monitoring
- **Shared Logging**: Centralized log management
- **Resource Efficient**: Lower memory footprint than multi-container
- **Easy Deployment**: Single image to deploy and manage

### Architecture
```
qt1-mockware:latest
├── Supervisor (Process Manager)
├── Mock Provider (Port 8081)
├── Testing Client (Port 3000)  
├── Shared Logs (/app/logs/)
└── Health Monitoring
```

### Environment Variables
```bash
# Service Ports
TESTING_CLIENT_PORT=3000
MOCK_PROVIDER_PORT=8081

# Mock Provider Configuration
RESPONSE_DELAY_MIN=100
RESPONSE_DELAY_MAX=500
ERROR_RATE=0.02
RATE_LIMIT_MAX=1000
ENABLE_STREAMING=true

# Testing Client Configuration  
DEFAULT_TARGET_URL=http://host.docker.internal:8080
MAX_CONCURRENT_TESTS=100
```

### Custom Configuration
```bash
docker run -d \
  -p 3000:3000 -p 8081:8081 \
  -e RESPONSE_DELAY_MIN=50 \
  -e RESPONSE_DELAY_MAX=200 \
  -e ERROR_RATE=0.05 \
  -e DEFAULT_TARGET_URL=http://my-qt1-middleware:8080 \
  --name qt1-mockware \
  qt1-mockware
```

## 📊 Health Monitoring

### Container Health Check
The unified container includes built-in health monitoring:

```bash
# Check container health
docker inspect qt1-mockware --format='{{.State.Health.Status}}'

# View health check logs
docker inspect qt1-mockware --format='{{range .State.Health.Log}}{{.Output}}{{end}}'
```

### Service Health Endpoints
```bash
# Test from outside container
curl http://localhost:3000/health  # Testing Client
curl http://localhost:8081/health  # Mock Provider

# Test from inside container  
docker exec qt1-mockware curl localhost:3000/health
docker exec qt1-mockware curl localhost:8081/health
```

## 📁 Volume Mounts

### Persistent Logs
```bash
docker run -d \
  -p 3000:3000 -p 8081:8081 \
  -v $(pwd)/container-logs:/app/logs \
  --name qt1-mockware \
  qt1-mockware
```

### Custom Configuration
```bash
docker run -d \
  -p 3000:3000 -p 8081:8081 \
  -v $(pwd)/custom-scenarios:/app/testing-client/scenarios:ro \
  -v $(pwd)/custom-responses:/app/mock-provider/responses:ro \
  --name qt1-mockware \
  qt1-mockware
```

## 🔧 Container Management

### Start/Stop/Restart
```bash
# Start
docker start qt1-mockware

# Stop
docker stop qt1-mockware

# Restart
docker restart qt1-mockware

# Remove
docker rm qt1-mockware
```

### Logs and Debugging
```bash
# View container logs
docker logs qt1-mockware

# Follow logs in real-time
docker logs -f qt1-mockware

# Execute commands inside container
docker exec -it qt1-mockware bash

# View service-specific logs
docker exec qt1-mockware cat /app/logs/mock-provider.log
docker exec qt1-mockware cat /app/logs/testing-client.log
```

### Process Monitoring
```bash
# View running processes
docker exec qt1-mockware ps aux

# View supervisor status
docker exec qt1-mockware supervisorctl status

# Restart individual service
docker exec qt1-mockware supervisorctl restart mock-provider
docker exec qt1-mockware supervisorctl restart testing-client
```

## 🌐 Networking

### Default Ports
- **3000**: Load Testing Client Dashboard
- **8081**: Universal Mock Provider API

### Connect to QT-1 Middleware
```yaml
# If QT-1 middleware is on host machine
providers:
  openai:
    endpoint: "http://localhost:8081"
    
# If QT-1 middleware is in another container
providers:
  openai:
    endpoint: "http://qt1-mockware:8081"
```

### Docker Network Setup
```bash
# Create custom network
docker network create qt1-network

# Run mockware on custom network
docker run -d \
  --network qt1-network \
  --name qt1-mockware \
  -p 3000:3000 -p 8081:8081 \
  qt1-mockware

# Run your QT-1 middleware on same network
docker run -d \
  --network qt1-network \
  --name qt1-middleware \
  your-qt1-middleware
```

## 🚀 Production Deployment

### Production Configuration
```bash
docker run -d \
  --name qt1-mockware-prod \
  --restart unless-stopped \
  -p 3000:3000 -p 8081:8081 \
  -e NODE_ENV=production \
  -e RESPONSE_DELAY_MIN=50 \
  -e RESPONSE_DELAY_MAX=300 \
  -e ERROR_RATE=0.01 \
  -e RATE_LIMIT_MAX=5000 \
  -v /var/log/qt1-mockware:/app/logs \
  --memory=512m \
  --cpus=1.0 \
  qt1-mockware
```

### Resource Limits
```bash
# Set memory and CPU limits
docker run -d \
  --name qt1-mockware \
  --memory=256m \
  --memory-swap=512m \
  --cpus=0.5 \
  -p 3000:3000 -p 8081:8081 \
  qt1-mockware
```

### Security Considerations
```bash
# Run with read-only filesystem
docker run -d \
  --name qt1-mockware \
  --read-only \
  --tmpfs /app/logs \
  --tmpfs /app/data \
  -p 3000:3000 -p 8081:8081 \
  qt1-mockware

# Run with specific user
docker run -d \
  --name qt1-mockware \
  --user 1001:1001 \
  -p 3000:3000 -p 8081:8081 \
  qt1-mockware
```

## 🔧 Troubleshooting

### Common Issues

#### Container won't start
```bash
# Check container logs
docker logs qt1-mockware

# Check if ports are available
ss -tlnp | grep -E ':(3000|8081)'

# Check container resources
docker stats qt1-mockware
```

#### Services not responding
```bash
# Check supervisor status
docker exec qt1-mockware supervisorctl status

# Restart services
docker exec qt1-mockware supervisorctl restart all

# Check service logs
docker exec qt1-mockware tail -f /app/logs/mock-provider.log
docker exec qt1-mockware tail -f /app/logs/testing-client.log
```

#### Cannot connect to QT-1 middleware
```bash
# Test network connectivity
docker exec qt1-mockware curl -v http://host.docker.internal:8080/health

# Check DNS resolution
docker exec qt1-mockware nslookup host.docker.internal

# Use container IP instead
docker inspect qt1-mockware --format='{{.NetworkSettings.IPAddress}}'
```

### Debug Mode
```bash
# Run container in debug mode
docker run -it --rm \
  -p 3000:3000 -p 8081:8081 \
  -e DEBUG=* \
  qt1-mockware bash

# Run with verbose logging
docker run -d \
  -p 3000:3000 -p 8081:8081 \
  -e ENABLE_DETAILED_LOGGING=true \
  -e LOG_LEVEL=debug \
  --name qt1-mockware \
  qt1-mockware
```

## 📈 Performance Tuning

### Optimize for High Load
```bash
docker run -d \
  --name qt1-mockware-performance \
  -p 3000:3000 -p 8081:8081 \
  -e RATE_LIMIT_MAX=10000 \
  -e MAX_CONCURRENT_TESTS=500 \
  -e RESPONSE_DELAY_MIN=10 \
  -e RESPONSE_DELAY_MAX=50 \
  --memory=1g \
  --cpus=2.0 \
  qt1-mockware
```

### Monitor Performance
```bash
# Real-time stats
docker stats qt1-mockware

# Resource usage over time
docker exec qt1-mockware top

# Test performance
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8081/health"
```

## 🎯 Quick Commands Reference

```bash
# Build and run (one command)
npm run docker:unified

# Stop and remove
npm run docker:stop

# View logs
docker logs -f qt1-mockware

# Health check
curl http://localhost:3000/health && curl http://localhost:8081/health

# Restart services
docker exec qt1-mockware supervisorctl restart all

# Update image
docker pull qt1-mockware:latest && docker restart qt1-mockware
```

The unified Docker container provides the easiest way to deploy and manage both Mockware services together! 🐳