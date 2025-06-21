#!/bin/bash

# QT-1 Mockware Docker Entry Point
# Prepares the container environment and starts both services

set -e

echo "🤖 QT-1 Mockware Container Starting..."
echo "======================================"

# Function to print colored output
print_info() {
    echo -e "\033[0;36m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[0;32m[SUCCESS]\033[0m $1"
}

print_warning() {
    echo -e "\033[1;33m[WARNING]\033[0m $1"
}

print_error() {
    echo -e "\033[0;31m[ERROR]\033[0m $1"
}

# Create necessary directories
print_info "Setting up container environment..."
mkdir -p /app/logs /app/data

# Set permissions
chown -R mockware:mockware /app/logs /app/data

# Display configuration
print_info "Container Configuration:"
echo "  • Testing Client Port: ${TESTING_CLIENT_PORT:-3000}"
echo "  • Mock Provider Port: ${MOCK_PROVIDER_PORT:-8081}"
echo "  • Target URL: ${DEFAULT_TARGET_URL:-http://host.docker.internal:8080}"
echo "  • Response Delay: ${RESPONSE_DELAY_MIN:-100}-${RESPONSE_DELAY_MAX:-500}ms"
echo "  • Error Rate: ${ERROR_RATE:-0.02}"
echo "  • Rate Limit: ${RATE_LIMIT_MAX:-1000} requests"
echo "  • Streaming: ${ENABLE_STREAMING:-true}"

# Function to wait for service
wait_for_service() {
    local service_name=$1
    local port=$2
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $service_name to be ready on port $port..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
            print_success "$service_name is ready!"
            return 0
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            print_error "$service_name failed to start after $max_attempts attempts"
            return 1
        fi
        
        print_info "  Attempt $attempt/$max_attempts - waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done
}

# Function to display service status
show_status() {
    echo ""
    echo "🚀 QT-1 Mockware Container Ready!"
    echo "================================="
    echo ""
    echo "📊 Services:"
    
    if curl -s -f "http://localhost:8081/health" > /dev/null 2>&1; then
        echo "  ✅ Mock Provider     : http://localhost:8081"
    else
        echo "  ❌ Mock Provider     : Not responding"
    fi
    
    if curl -s -f "http://localhost:3000/health" > /dev/null 2>&1; then
        echo "  ✅ Testing Client    : http://localhost:3000"
    else
        echo "  ❌ Testing Client    : Not responding"
    fi
    
    echo ""
    echo "🌐 Access Points:"
    echo "  • Testing Dashboard  : http://localhost:3000"
    echo "  • Mock Provider API  : http://localhost:8081"
    echo "  • Provider Health    : http://localhost:8081/health"
    echo "  • Provider Dashboard : http://localhost:8081 (see docs)"
    echo ""
    echo "📁 Container Logs:"
    echo "  • Mock Provider      : /app/logs/mock-provider.log"
    echo "  • Testing Client     : /app/logs/testing-client.log"
    echo "  • Supervisor         : /app/logs/supervisord.log"
    echo ""
    echo "🎯 Ready for QT-1 middleware testing!"
}

# Function to handle shutdown
handle_shutdown() {
    print_info "Received shutdown signal, stopping services..."
    
    # Stop supervisor and all services
    if [ -f /app/data/supervisord.pid ]; then
        supervisorctl stop all
        kill -TERM $(cat /app/data/supervisord.pid) 2>/dev/null || true
    fi
    
    print_success "Container shutdown complete"
    exit 0
}

# Set up signal handlers
trap handle_shutdown SIGTERM SIGINT

# If running in interactive mode, show status periodically
if [ -t 0 ]; then
    echo ""
    print_info "Container started in interactive mode"
    print_info "Services will start automatically via supervisor"
    
    # Start supervisor in background
    exec "$@" &
    SUPERVISOR_PID=$!
    
    # Wait a moment for services to start
    sleep 15
    
    # Show initial status
    show_status
    
    # Monitor services periodically
    while kill -0 $SUPERVISOR_PID 2>/dev/null; do
        sleep 30
        
        # Check if both services are healthy
        mock_healthy=$(curl -s -f "http://localhost:8081/health" > /dev/null 2>&1 && echo "✅" || echo "❌")
        client_healthy=$(curl -s -f "http://localhost:3000/health" > /dev/null 2>&1 && echo "✅" || echo "❌")
        
        echo "$(date '+%H:%M:%S') - Status: Mock Provider $mock_healthy | Testing Client $client_healthy"
    done
    
else
    # Non-interactive mode - just exec supervisor
    print_info "Container started in daemon mode"
    print_info "Services will start automatically"
    
    # Execute the command (supervisor)
    exec "$@"
fi