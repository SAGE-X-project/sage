#!/bin/bash
# SAGE 빠른 기능 검증 스크립트
# 개발 중 주요 기능만 빠르게 검증
# 작성일: 2025-10-10

set -e

# 색상
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================"
echo "SAGE 빠른 기능 검증"
echo -e "======================================${NC}"

echo -e "${YELLOW}[1/5] RFC 9421 서명/검증${NC}"
go test github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run TestIntegration -v

echo -e "${YELLOW}[2/5] 암호화 키 (Ed25519, Secp256k1)${NC}"
go test github.com/sage-x-project/sage/pkg/agent/crypto/keys -run "TestEd25519KeyPair/SignAndVerify" -v
go test github.com/sage-x-project/sage/pkg/agent/crypto/keys -run "TestSecp256k1KeyPair/SignAndVerify" -v

echo -e "${YELLOW}[3/5] 메시지 처리 (Nonce, 순서)${NC}"
go test github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run TestNonceManager/GenerateNonce -v
go test github.com/sage-x-project/sage/pkg/agent/core/message/order -run TestOrderManager/SeqMonotonicity -v

echo -e "${YELLOW}[4/5] 세션 관리${NC}"
go test github.com/sage-x-project/sage/pkg/agent/session -run "TestSessionManager_CreateSession" -v

echo -e "${YELLOW}[5/5] HPKE 핸드셰이크${NC}"
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run TestE2E_HPKE_Handshake_MockTransport

echo ""
echo -e "${GREEN}빠른 검증 완료!${NC}"
echo ""
echo "전체 검증을 실행하려면:"
echo "  ./tools/scripts/verify_all_features.sh"
