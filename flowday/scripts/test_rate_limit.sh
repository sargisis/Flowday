#!/bin/bash

echo "🔒 Testing Rate Limiting on Auth Endpoints"
echo "=========================================="
echo ""

API_URL="http://localhost:8080/api/v1"

# Test Login Rate Limit (5 per minute)
echo "📝 Test 1: Login Rate Limit (Max 5/min)"
echo "Sending 7 login requests quickly..."
echo ""

for i in {1..7}; do
  echo "Request #$i:"
  response=$(curl -s -w "\nHTTP Status: %{http_code}" -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@test.com","password":"wrongpassword"}')
  
  echo "$response"
  echo "---"
  sleep 0.5
done

echo ""
echo "Expected: Requests 1-5 should return 401 (wrong password)"
echo "          Requests 6-7 should return 429 (rate limited)"
echo ""
echo "=========================================="
echo ""

# Test Register Rate Limit (3 per minute)
echo "📝 Test 2: Register Rate Limit (Max 3/min)"
echo "Sending 5 registration requests quickly..."
echo ""

for i in {1..5}; do
  echo "Request #$i:"
  response=$(curl -s -w "\nHTTP Status: %{http_code}" -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"Test$i\",\"email\":\"test$i@test.com\",\"password\":\"password123\"}")
  
  echo "$response"
  echo "---"
  sleep 0.5
done

echo ""
echo "Expected: Requests 1-3 might succeed or conflict"
echo "          Requests 4-5 should return 429 (rate limited)"
echo ""
echo "=========================================="
echo ""

# Test Forgot Password Rate Limit (3 per minute)
echo "📝 Test 3: Forgot Password Rate Limit (Max 3/min)"
echo "Sending 5 forgot-password requests quickly..."
echo ""

for i in {1..5}; do
  echo "Request #$i:"
  response=$(curl -s -w "\nHTTP Status: %{http_code}" -X POST "$API_URL/auth/forgot-password" \
    -H "Content-Type: application/json" \
    -d '{"email":"test@test.com"}')
  
  echo "$response"
  echo "---"
  sleep 0.5
done

echo ""
echo "Expected: Requests 1-3 should return 200"
echo "          Requests 4-5 should return 429 (rate limited)"
echo ""
echo "🎉 Test Complete!"
