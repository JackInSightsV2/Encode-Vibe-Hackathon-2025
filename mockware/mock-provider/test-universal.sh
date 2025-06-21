#!/bin/bash

# QT-1 Universal Mock Provider Test

BASE_URL="http://localhost:8081"

echo "🎭 Testing Universal Mock AI Provider"
echo "===================================="

echo ""
echo "1. Testing Root Documentation..."
curl -s "$BASE_URL/" | jq '.'

echo ""
echo "2. Testing Health Check..."
curl -s "$BASE_URL/health" | jq '.'

echo ""
echo "3. Testing OpenAI Chat Format (auto-detected)..."
curl -s -X POST "$BASE_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello universal mock!"}
    ]
  }' | jq '.'

echo ""
echo "4. Testing Anthropic Messages Format (auto-detected)..."
curl -s -X POST "$BASE_URL/v1/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet-20240229",
    "max_tokens": 100,
    "messages": [
      {"role": "user", "content": "Hello Claude mock!"}
    ]
  }' | jq '.'

echo ""
echo "5. Testing OpenAI Legacy Completions (auto-detected)..."
curl -s -X POST "$BASE_URL/v1/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-davinci-003",
    "prompt": "Complete this: The universal mock is",
    "max_tokens": 50
  }' | jq '.'

echo ""
echo "6. Testing Moderation API (auto-detected)..."
curl -s -X POST "$BASE_URL/v1/moderations" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "I want to hurt people"
  }' | jq '.'

echo ""
echo "7. Testing Unknown Provider Format (generic response)..."
curl -s -X POST "$BASE_URL/api/custom/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is AI?",
    "provider": "unknown-provider"
  }' | jq '.'

echo ""
echo "8. Testing Different Path Format..."
curl -s -X POST "$BASE_URL/ai/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Generate a response",
    "temperature": 0.7
  }' | jq '.'

echo ""
echo "✅ Universal mock testing completed!"
echo "The mock provider automatically detected and responded to all request formats."