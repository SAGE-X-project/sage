# SAGE (Secure Agent Guarantee Engine)

> 블록체인 DID 기반 AI 에이전트 신뢰 통신 프레임워크

## 목차

- [프로젝트 개요](#프로젝트-개요)
- [핵심 기능](#핵심-기능)
- [시작하기](#시작하기)
- [문서](#문서)
- [기여하기](#기여하기)
- [라이선스](#라이선스)

## 프로젝트 개요

SAGE는 AI 에이전트 간 통신에서 발생하는 보안 취약점을 해결하기 위한 오픈소스 프레임워크입니다. 블록체인 기반 분산신원확인(DID)과 RFC 9421 표준에 따른 HTTP 부분 서명 기술을 도입하여, AI 에이전트 간 통신의 신뢰성과 무결성을 보장합니다.

### 주요 목표

- **신원 검증**: 블록체인 DID를 통한 에이전트 신원 확인
- **메시지 무결성**: RFC 9421 기반 디지털 서명으로 메시지 변조 방지
- **중간자 공격 방지**: 암호학적 서명을 통한 보안 통신
- **표준 준수**: W3C DID 및 RFC 9421 표준 완벽 지원

### 기대 효과

1. **보안 강화**: 에이전트 간 통신에서 위조, 변조, 중간자 공격 방지
2. **신뢰 구축**: 암호학적으로 검증 가능한 에이전트 신원
3. **투명성**: 블록체인 기반 감사 추적 가능
4. **표준화**: 업계 표준으로 발전 가능한 오픈소스 솔루션

## 핵심 기능

### DID 기반 신원 관리

- 블록체인에 에이전트 DID 및 공개키 등록
- W3C DID 표준 준수
- 다양한 DID 메소드 지원 (did:ethr, did:key, did:sol)

### 핸드셰이크 & 세션 키 합의

- X25519 임시 공개키 교환 → ECDH 공유 비밀
- HKDF-SHA256 세션 키 파생(암/서명 분리)
- AEAD(ChaCha20-Poly1305) + HMAC-SHA256
- PFS 보장, kid + nonce로 Replay 방지

### RFC 9421 메시지 서명

- HTTP 메시지 부분 서명 지원
- Ed25519, ECDSA 등 다양한 서명 알고리즘
- 메시지 무결성 및 부인 방지

### 유연한 배포 옵션

- **Direct P2P 모드**: 에이전트 간 직접 통신 (기본)
- **Gateway 모드**: 중앙 라우터를 통한 정책 기반 통신 (선택)

### 다중 언어 지원

- Go SDK (네이티브)
- TypeScript SDK (WASM 기반)
- Rust 핵심 암호화 엔진

## 시작하기

### 사전 요구사항

- Go 1.22 이상
- Rust 1.65 이상 (libsage_crypto 빌드용)
- 블록체인 RPC 접근 (DID 등록/조회)

### 빠른 시작

```bash
# 저장소 클론
git clone https://github.com/sage-project/sage.git
cd sage

# 의존성 설치
go mod download

# 핵심 암호화 라이브러리 빌드
cd rust/sage_crypto && cargo build --release

# 테스트 실행
go test ./...

# 예제 에이전트 실행
go run cmd/agent-peer/main.go -config config/example.yaml
```

자세한 설치 및 설정 방법은 [개발 가이드](development-guide.md)를 참조하세요.

## 문서

- [요구사항 명세서](requirements.md) - 기능 및 비기능 요구사항
- [아키텍처 문서](architecture.md) - 시스템 설계 및 모듈 구조
- [개발 가이드](development-guide.md) - 개발 환경 설정 및 코드 구조
- [API 명세서](api-spec.md) - SDK 및 Gateway API 문서
- [보안 설계서](security-design.md) - 보안 아키텍처 및 위협 모델

## 기여하기

SAGE는 오픈소스 프로젝트로 여러분의 기여를 환영합니다!

### 기여 방법

1. 이슈 등록 또는 기존 이슈 확인
2. Fork 후 feature 브랜치 생성
3. 변경사항 커밋 (커밋 메시지 규칙 준수)
4. Pull Request 생성
5. 코드 리뷰 및 병합

자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참조하세요.

### 커뮤니케이션

- GitHub Issues: 버그 리포트 및 기능 제안
- Discord: 실시간 논의 및 질문 (추후 공개)
- Wiki: 상세 문서 및 가이드

## 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.
