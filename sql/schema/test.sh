#!/bin/bash

echo "=== 1. Reset databázy ==="
curl -X POST http://localhost:8080/admin/reset
echo -e "\n"

echo "=== 2. Vytvoriť usera ==="
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "saul@bettercall.com", "password": "123456"}'
echo -e "\n"

echo "=== 3. Prihlásiť usera ==="
response=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "saul@bettercall.com", "password": "123456"}')
echo "$response"

saulAccessToken=$(echo "$response" | jq -r '.token')
saulRefreshToken=$(echo "$response" | jq -r '.refresh_token')

echo "Access Token: $saulAccessToken"
echo "Refresh Token: $saulRefreshToken"
echo -e "\n"

echo "=== 4. Test chirps s refresh tokenom (401) ==="
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $saulRefreshToken" \
  -d '{"body": "Let'\''s just say I know a guy... who knows a guy... who knows another guy."}'
echo -e "\n"

echo "=== 5. Test chirps s access tokenom (201) ==="
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $saulAccessToken" \
  -d '{"body": "Let'\''s just say I know a guy... who knows a guy... who knows another guy."}'
echo -e "\n"

echo "=== 6. Refresh access token ==="
refresh_response=$(curl -s -X POST http://localhost:8080/api/refresh \
  -H "Authorization: Bearer $saulRefreshToken")
echo "$refresh_response"

saulAccessToken2=$(echo "$refresh_response" | jq -r '.token')
echo "New Access Token: $saulAccessToken2"
echo -e "\n"

echo "=== 7. Test chirps s novým access tokenom ==="
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $saulAccessToken2" \
  -d '{"body": "I'\''m the guy who'\''s gonna win you this case."}'
echo -e "\n"

echo "=== 8. Revoke refresh token ==="
curl -X POST http://localhost:8080/api/revoke \
  -H "Authorization: Bearer $saulRefreshToken"
echo -e "\n"

echo "=== 9. Test refresh s revoknutým tokenom (401) ==="
curl -X POST http://localhost:8080/api/refresh \
  -H "Authorization: Bearer $saulRefreshToken"
echo -e "\n"

echo "=== Hotovo! ==="