# SAGE 핸드셰이크 문서

이 폴더는 AI 에이전트 간 인증되고 암호화된 세션을 수립하기 위한 SAGE의 보안 핸드셰이크 프로토콜 문서를 포함합니다.

## 빠른 네비게이션

### 처음 사용하시는 분

1. **여기서 시작**: [암호학적 개요](./cryptographic-ko.md) - SAGE의 보안 기반 이해
2. **접근 방식 선택**:
   - **전통적 방식**: [4단계 핸드셰이크 가이드](./handshake-ko.md) - 성숙하고 검증된 방식
   - **현대적 방식**: [HPKE 기반 핸드셰이크 가이드](./hpke-based-handshake-ko.md) - 1-RTT, 신규 프로젝트 권장

### 개발자용

- **구현 가이드**: [HPKE 상세 튜토리얼](./hpke-detailed-ko.md) - 코드 예제가 포함된 단계별 설명
- **API 레퍼런스**: `/handshake` 및 `/hpke` 패키지의 코드 문서 참조

---

## 두 가지 핸드셰이크 프로토콜

SAGE는 두 가지 핸드셰이크 프로토콜을 지원합니다. 요구사항에 따라 선택하세요:

| 특징 | 전통적 방식 (4단계) | HPKE 기반 (2단계) |
|-----|-------------------|------------------|
| **패키지** | `handshake/` | `hpke/` |
| **왕복 횟수** | 2 RTT (4개 메시지) | 1 RTT (2개 메시지) |
| **키 교환** | X25519 ECDH | HPKE Base + E2E X25519 |
| **Forward Secrecy** |  (임시 키) |  (HPKE + E2E 애드온) |
| **성숙도** | 안정적 | 안정적 |
| **권장 사용처** | 기존 통합 | 신규 프로젝트 |

### 전통적 4단계 핸드셰이크

**단계**: Invitation → Request → Response → Complete

```
클라이언트                                   서버
  │                                          │
  │  1. Invitation (서명됨, 평문)             │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  2. Request (부트스트랩 암호화)            │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  3. Response (세션으로 암호화)            │
  │ <──────────────────────────────────────  │
  │                                          │
  │  4. Complete (최종 확인)                 │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  암호화된 세션 수립 완료                │
```

**사용 시기**: 기존 A2A 프로토콜 통합과의 호환성이 필요한 경우

**문서**: [handshake-ko.md](./handshake-ko.md)

### HPKE 기반 2단계 핸드셰이크 (권장)

**단계**: Initialize → Acknowledge

```
클라이언트                                   서버
  │                                          │
  │  1. Init (HPKE enc + ephC)              │
  │ ──────────────────────────────────────>  │
  │                                          │
  │  2. Ack (kid + ackTag + ephS)           │
  │ <──────────────────────────────────────  │
  │                                          │
  │  암호화된 세션 수립 완료                │
```

**사용 시기**: 신규 프로젝트 시작, 낮은 지연시간 필요

**문서**: [hpke-based-handshake-ko.md](./hpke-based-handshake-ko.md)

---

## 문서 설명

### 핵심 문서

| 문서 | 언어 | 대상 | 설명 |
|-----|------|------|------|
| **cryptographic-en.md** | English | 모든 사용자 | 종합 암호학적 설계 및 보안 모델 |
| **cryptographic-ko.md** | 한국어 | 모든 사용자 | 종합 암호학적 설계 및 보안 모델 (한국어) |
| **handshake-en.md** | English | 개발자 | 전통적 4단계 핸드셰이크 구현 가이드 |
| **handshake-ko.md** | 한국어 | 개발자 | 전통적 4단계 핸드셰이크 구현 가이드 (한국어) |
| **hpke-based-handshake-en.md** | English | 개발자 | HPKE 기반 핸드셰이크 구현 가이드 |
| **hpke-based-handshake-ko.md** | 한국어 | 개발자 | HPKE 기반 핸드셰이크 구현 가이드 (한국어) |

### 튜토리얼

| 문서 | 언어 | 대상 | 설명 |
|-----|------|------|------|
| **hpke-detailed-en.md** | English | 초보자 | 코드 예제가 포함된 단계별 HPKE 튜토리얼 |
| **hpke-detailed-ko.md** | 한국어 | 초보자 | 코드 예제가 포함된 단계별 HPKE 튜토리얼 (한국어) |

---

## 학습 경로

### 경로 1: 전통적 핸드셰이크

1. [cryptographic-ko.md](./cryptographic-ko.md) 읽기 - 보안 기반
2. [handshake-ko.md](./handshake-ko.md) 읽기 - 4단계 프로토콜
3. `/handshake` 패키지 코드 검토
4. 통합 구현

### 경로 2: HPKE 핸드셰이크 (권장)

1. [cryptographic-ko.md](./cryptographic-ko.md) 읽기 - 보안 기반 (HPKE 섹션 집중)
2. [hpke-based-handshake-ko.md](./hpke-based-handshake-ko.md) 읽기 - 2단계 프로토콜
3. [hpke-detailed-ko.md](./hpke-detailed-ko.md) 따라하기 - 실습 튜토리얼
4. `/hpke` 패키지 코드 검토
5. 통합 구현

---

## 핵심 개념

### DID 신원 바인딩

두 프로토콜 모두 **분산 식별자(DIDs)**를 사용하여 세션을 에이전트 신원에 바인딩합니다:

- Ed25519 서명 키로 에이전트 신원 검증
- X25519 키 (Ed25519에서 유도)로 키 교환 수행
- DID 메타데이터는 블록체인(Ethereum, Solana, Kaia)에 저장

### Forward Secrecy (전방향 안전성)

두 프로토콜 모두 **전방향 안전성**을 제공합니다:

- **전통적 방식**: 세션 수립 후 임시 X25519 키 삭제
- **HPKE**: HPKE Base 모드 + E2E 임시 X25519 애드온

### 세션 보안

세션 수립 후 다음을 사용합니다:

- **암호화**: ChaCha20-Poly1305 AEAD (인증 암호화)
- **MAC**: HMAC-SHA256으로 추가 무결성 보호
- **키 유도**: HKDF-SHA256 with 방향성 키 (C2S/S2C 분리)
- **논스 관리**: 세션별 논스 추적으로 재전송 공격 방지

---

## 보안 속성

두 핸드셰이크 프로토콜 모두 다음을 제공합니다:

-  **상호 인증**: 양쪽 에이전트가 서로의 DID를 검증
-  **Forward Secrecy**: 장기 키가 노출되어도 과거 세션은 안전
-  **재전송 보호**: 논스와 타임스탬프로 메시지 재전송 방지
-  **중간자 공격 방어**: DID 서명으로 중간자 공격 방지
-  **End-to-End 암호화**: 통신하는 에이전트만 메시지 복호화 가능

---

## 구현 참고사항

### 프로토콜 선택

**전통적 방식 (4단계) 사용 조건**:
- 기존 A2A 프로토콜 시스템과 통합
- 명시적인 초대/수락 흐름 필요
- 하위 호환성 필요

**HPKE 기반 (2단계) 사용 조건**:
- 새 프로젝트 시작
- 낮은 지연시간 필요 (1 RTT vs 2 RTT)
- 최신 암호화 방식 선호 (HPKE/RFC 9180)
- 단순한 상태 머신 선호

### 코드 패키지

- **전통적 방식**: `github.com/sage-x-project/sage/handshake`
- **HPKE**: `github.com/sage-x-project/sage/hpke`
- **세션 관리**: `github.com/sage-x-project/sage/session`
- **암호화**: `github.com/sage-x-project/sage/crypto/keys`

---

## 관련 문서

- **[RFC 9421 HTTP Message Signatures](../core/rfc9421-ko.md)**: HTTP 레벨 인증
- **[Crypto Package Guide](../crypto/crypto-ko.md)**: 암호화 기본 요소
- **[DID Documentation](../did/)**: 분산 식별자 시스템

---

## 기여하기

핸드셰이크 문서 업데이트 시:

1. 영문 및 한국어 버전 모두 업데이트
2. 코드 예제가 실제 구현과 일치하는지 확인
3. 테스트 실행: `go test ./handshake/... ./hpke/...`
4. 새 문서 추가 시 이 README 업데이트

---

## 지원

질문이나 문제가 있는 경우:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- 문서: https://docs.sage-x-project.org
