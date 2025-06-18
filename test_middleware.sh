#!/bin/bash

echo "🚀 QT-1 Middleware Test Script"
echo "================================"

# Start the middleware in background
echo "Starting QT-1 middleware..."
cd backend
./qt1-middleware &
MIDDLEWARE_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 3

# Test health endpoint
echo ""
echo "1. Testing health endpoint..."
curl -s http://localhost:8080/health | jq . || echo "Health check response received"

# Test status API
echo ""
echo "2. Testing status API..."
curl -s http://localhost:8080/api/status | jq . || echo "Status API response received"

# Test config API
echo ""
echo "3. Testing config API..."
curl -s http://localhost:8080/api/config | jq . || echo "Config API response received"

# Test chat endpoint (should fail since no target)
echo ""
echo "4. Testing chat endpoint (will fail - no target)..."
curl -s -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test_user","session_id":"test_session","message":"Hello world"}' \
  | jq . || echo "Chat endpoint response received"

# Test moderation (blocked word)
echo ""
echo "5. Testing moderation with blocked word..."
curl -s -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test_user","session_id":"test_session","message":"This contains violence"}' \
  | jq . || echo "Moderation test response received"

# Test killswitch API
echo ""
echo "6. Testing killswitch API..."
curl -s http://localhost:8080/api/killswitch | jq . || echo "Killswitch API response received"

# Clean up
echo ""  
echo "Stopping middleware..."
kill $MIDDLEWARE_PID

echo ""
echo "✅ Test completed! Check the logs/qt1.log file for detailed logs."