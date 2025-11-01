#!/bin/bash

echo "========================================="
echo "SAGE DID CLI 워크플로우 테스트"
echo "========================================="
echo ""

# 8.2.1: 키 생성
echo " Step 1: Secp256k1 키 생성"
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/test-eth-key.jwk
echo ""

# 키 정보 확인
echo " Step 2: 키 정보 확인"
echo "Key Type:"
cat /tmp/test-eth-key.jwk | jq -r '.key_type'
echo ""

# Ethereum 주소 생성
echo " Step 3: Ethereum 주소 생성"
./build/bin/sage-crypto address generate --key /tmp/test-eth-key.jwk --chain ethereum
echo ""

# 키 구조 확인
echo " Step 4: JWK 파일 구조 확인"
cat /tmp/test-eth-key.jwk | jq '{key_id, key_type, private_key: {kty: .private_key.kty, crv: .private_key.crv}}'
echo ""

echo " 워크플로우 Step 1-4 완료!"
echo "생성된 키 파일: /tmp/test-eth-key.jwk"
