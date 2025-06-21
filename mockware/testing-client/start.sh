#!/bin/bash

# Create reports directory
mkdir -p reports

# Check if dependencies are installed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Start the testing client
echo "Starting QT-1 Testing Client..."
echo "Open http://localhost:3001 in your browser"
npm start