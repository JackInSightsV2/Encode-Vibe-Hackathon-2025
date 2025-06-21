#!/bin/bash

# QT-1 Mockware Startup Script
# Starts both the Load Testing Client and Mock Provider

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
TESTING_CLIENT_PORT=3000
MOCK_PROVIDER_PORT=8081
LOG_DIR="./logs"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[MOCKWARE]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

# Function to check if port is available
check_port() {
    local port=$1
    local service=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "Port $port is already in use (needed for $service)"
        print_info "Attempting to stop existing process on port $port..."
        
        # Try to kill the process using the port
        local pid=$(lsof -Pi :$port -sTCP:LISTEN -t)
        if [ ! -z "$pid" ]; then
            kill -TERM $pid 2>/dev/null || kill -KILL $pid 2>/dev/null
            sleep 2
            
            # Check if port is now free
            if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
                print_error "Could not free port $port. Please manually stop the process and try again."
                return 1
            else
                print_success "Freed port $port"
            fi
        fi
    fi
    return 0
}

# Function to create log directory
setup_logging() {
    if [ ! -d "$LOG_DIR" ]; then
        mkdir -p "$LOG_DIR"
        print_info "Created log directory: $LOG_DIR"
    fi
}

# Function to check dependencies
check_dependencies() {
    print_status "Checking dependencies..."
    
    # Check if Node.js is installed
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed. Please install Node.js and try again."
        exit 1
    fi
    
    # Check if npm is installed
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed. Please install npm and try again."
        exit 1
    fi
    
    print_success "Node.js $(node --version) and npm $(npm --version) are available"
}

# Function to install dependencies
install_dependencies() {
    print_status "Installing dependencies..."
    
    # Install testing client dependencies
    if [ -d "testing-client" ]; then
        print_info "Installing testing client dependencies..."
        cd testing-client
        if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules" ]; then
            npm install > "$LOG_DIR/testing-client-install.log" 2>&1
            if [ $? -eq 0 ]; then
                print_success "Testing client dependencies installed"
            else
                print_error "Failed to install testing client dependencies. Check $LOG_DIR/testing-client-install.log"
                exit 1
            fi
        else
            print_info "Testing client dependencies already installed"
        fi
        cd ..
    else
        print_error "testing-client directory not found!"
        exit 1
    fi
    
    # Install mock provider dependencies
    if [ -d "mock-provider" ]; then
        print_info "Installing mock provider dependencies..."
        cd mock-provider
        if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules" ]; then
            npm install > "$LOG_DIR/mock-provider-install.log" 2>&1
            if [ $? -eq 0 ]; then
                print_success "Mock provider dependencies installed"
            else
                print_error "Failed to install mock provider dependencies. Check $LOG_DIR/mock-provider-install.log"
                exit 1
            fi
        else
            print_info "Mock provider dependencies already installed"
        fi
        cd ..
    else
        print_error "mock-provider directory not found!"
        exit 1
    fi
}

# Function to start mock provider
start_mock_provider() {
    print_status "Starting Mock Provider..."
    
    cd mock-provider
    
    # Start the mock provider in background
    nohup npm start > "$LOG_DIR/mock-provider.log" 2>&1 &
    MOCK_PROVIDER_PID=$!
    
    # Give it time to start
    sleep 3
    
    # Check if it's running
    if kill -0 $MOCK_PROVIDER_PID 2>/dev/null; then
        # Test the health endpoint
        if curl -s "http://localhost:$MOCK_PROVIDER_PORT/health" > /dev/null 2>&1; then
            print_success "Mock Provider started on port $MOCK_PROVIDER_PORT (PID: $MOCK_PROVIDER_PID)"
        else
            print_warning "Mock Provider process started but health check failed"
        fi
    else
        print_error "Failed to start Mock Provider"
        cat "$LOG_DIR/mock-provider.log"
        exit 1
    fi
    
    cd ..
}

# Function to start testing client
start_testing_client() {
    print_status "Starting Load Testing Client..."
    
    cd testing-client
    
    # Start the testing client in background
    nohup npm start > "$LOG_DIR/testing-client.log" 2>&1 &
    TESTING_CLIENT_PID=$!
    
    # Give it time to start
    sleep 3
    
    # Check if it's running
    if kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
        # Test the health endpoint
        if curl -s "http://localhost:$TESTING_CLIENT_PORT/health" > /dev/null 2>&1; then
            print_success "Load Testing Client started on port $TESTING_CLIENT_PORT (PID: $TESTING_CLIENT_PID)"
        else
            print_warning "Testing Client process started but health check failed"
        fi
    else
        print_error "Failed to start Load Testing Client"
        cat "$LOG_DIR/testing-client.log"
        exit 1
    fi
    
    cd ..
}

# Function to display status
show_status() {
    echo ""
    echo -e "${PURPLE}🤖 QT-1 Mockware Status${NC}"
    echo -e "${PURPLE}========================${NC}"
    echo ""
    
    if kill -0 $MOCK_PROVIDER_PID 2>/dev/null; then
        echo -e "${GREEN}✅ Mock Provider${NC}     : Running on http://localhost:$MOCK_PROVIDER_PORT (PID: $MOCK_PROVIDER_PID)"
    else
        echo -e "${RED}❌ Mock Provider${NC}     : Not running"
    fi
    
    if kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
        echo -e "${GREEN}✅ Testing Client${NC}    : Running on http://localhost:$TESTING_CLIENT_PORT (PID: $TESTING_CLIENT_PID)"
    else
        echo -e "${RED}❌ Testing Client${NC}    : Not running"
    fi
    
    echo ""
    echo -e "${CYAN}📊 Quick Access Links:${NC}"
    echo "  • Testing Dashboard  : http://localhost:$TESTING_CLIENT_PORT"
    echo "  • Mock Provider API  : http://localhost:$MOCK_PROVIDER_PORT"
    echo "  • Provider Health    : http://localhost:$MOCK_PROVIDER_PORT/health"
    echo "  • Provider Dashboard : mock-provider/dashboard/index.html"
    echo ""
    echo -e "${CYAN}📁 Log Files:${NC}"
    echo "  • Mock Provider      : $LOG_DIR/mock-provider.log"
    echo "  • Testing Client     : $LOG_DIR/testing-client.log"
    echo ""
}

# Function to save PIDs
save_pids() {
    echo "MOCK_PROVIDER_PID=$MOCK_PROVIDER_PID" > .mockware-pids
    echo "TESTING_CLIENT_PID=$TESTING_CLIENT_PID" >> .mockware-pids
    echo "MOCK_PROVIDER_PORT=$MOCK_PROVIDER_PORT" >> .mockware-pids
    echo "TESTING_CLIENT_PORT=$TESTING_CLIENT_PORT" >> .mockware-pids
}

# Function to handle cleanup on exit
cleanup() {
    echo ""
    print_status "Shutting down Mockware..."
    
    if [ ! -z "$MOCK_PROVIDER_PID" ] && kill -0 $MOCK_PROVIDER_PID 2>/dev/null; then
        print_info "Stopping Mock Provider (PID: $MOCK_PROVIDER_PID)..."
        kill -TERM $MOCK_PROVIDER_PID 2>/dev/null
    fi
    
    if [ ! -z "$TESTING_CLIENT_PID" ] && kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
        print_info "Stopping Testing Client (PID: $TESTING_CLIENT_PID)..."
        kill -TERM $TESTING_CLIENT_PID 2>/dev/null
    fi
    
    # Wait a moment for graceful shutdown
    sleep 2
    
    # Force kill if still running
    if [ ! -z "$MOCK_PROVIDER_PID" ] && kill -0 $MOCK_PROVIDER_PID 2>/dev/null; then
        kill -KILL $MOCK_PROVIDER_PID 2>/dev/null
    fi
    
    if [ ! -z "$TESTING_CLIENT_PID" ] && kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
        kill -KILL $TESTING_CLIENT_PID 2>/dev/null
    fi
    
    # Clean up PID file
    rm -f .mockware-pids
    
    print_success "Mockware shutdown complete"
}

# Function to show help
show_help() {
    echo -e "${PURPLE}🤖 QT-1 Mockware Startup Script${NC}"
    echo ""
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  start     Start both Mock Provider and Testing Client (default)"
    echo "  stop      Stop both services"
    echo "  restart   Restart both services"
    echo "  status    Show status of running services"
    echo "  logs      Show live logs from both services"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                # Start both services"
    echo "  $0 start          # Start both services"
    echo "  $0 stop           # Stop both services"
    echo "  $0 status         # Check service status"
    echo ""
}

# Function to stop services
stop_services() {
    if [ -f ".mockware-pids" ]; then
        source .mockware-pids
        
        print_status "Stopping Mockware services..."
        
        if [ ! -z "$MOCK_PROVIDER_PID" ] && kill -0 $MOCK_PROVIDER_PID 2>/dev/null; then
            print_info "Stopping Mock Provider (PID: $MOCK_PROVIDER_PID)..."
            kill -TERM $MOCK_PROVIDER_PID
        fi
        
        if [ ! -z "$TESTING_CLIENT_PID" ] && kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
            print_info "Stopping Testing Client (PID: $TESTING_CLIENT_PID)..."
            kill -TERM $TESTING_CLIENT_PID
        fi
        
        sleep 2
        rm -f .mockware-pids
        print_success "Services stopped"
    else
        print_warning "No running services found"
    fi
}

# Function to show service status
show_service_status() {
    if [ -f ".mockware-pids" ]; then
        source .mockware-pids
        show_status
    else
        print_warning "No running services found"
        echo "Use '$0 start' to start Mockware services"
    fi
}

# Function to show logs
show_logs() {
    if [ -f "$LOG_DIR/mock-provider.log" ] && [ -f "$LOG_DIR/testing-client.log" ]; then
        print_info "Showing live logs (Ctrl+C to exit)..."
        tail -f "$LOG_DIR/mock-provider.log" "$LOG_DIR/testing-client.log"
    else
        print_warning "Log files not found. Start the services first."
    fi
}

# Main execution
main() {
    # Parse command line arguments
    COMMAND=${1:-start}
    
    case $COMMAND in
        "start")
            echo -e "${PURPLE}🚀 Starting QT-1 Mockware Suite${NC}"
            echo -e "${PURPLE}================================${NC}"
            echo ""
            
            # Setup
            setup_logging
            check_dependencies
            
            # Check ports
            check_port $TESTING_CLIENT_PORT "Testing Client" || exit 1
            check_port $MOCK_PROVIDER_PORT "Mock Provider" || exit 1
            
            # Install dependencies
            install_dependencies
            
            # Start services
            start_mock_provider
            start_testing_client
            
            # Save PIDs for later management
            save_pids
            
            # Show status
            show_status
            
            # Setup signal handlers for graceful shutdown
            trap cleanup INT TERM
            
            print_success "Mockware started successfully!"
            print_info "Press Ctrl+C to stop all services"
            
            # Keep script running
            while true; do
                sleep 5
                
                # Check if services are still running
                if ! kill -0 $MOCK_PROVIDER_PID 2>/dev/null || ! kill -0 $TESTING_CLIENT_PID 2>/dev/null; then
                    print_warning "One or more services stopped unexpectedly"
                    show_status
                fi
            done
            ;;
            
        "stop")
            stop_services
            ;;
            
        "restart")
            stop_services
            sleep 2
            exec $0 start
            ;;
            
        "status")
            show_service_status
            ;;
            
        "logs")
            show_logs
            ;;
            
        "help"|"-h"|"--help")
            show_help
            ;;
            
        *)
            print_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"