# SAGE 프로젝트 상세 가이드 - Part 1: 프로젝트 개요 및 아키텍처

> **대상 독자**: 프로그래밍 초급자부터 중급 개발자까지
> **작성일**: 2025-10-07
> **버전**: 1.0

---

## 목차

1. [SAGE란 무엇인가?](#1-sage란-무엇인가)
2. [왜 SAGE가 필요한가?](#2-왜-sage가-필요한가)
3. [전체 아키텍처 개요](#3-전체-아키텍처-개요)
4. [핵심 개념 설명](#4-핵심-개념-설명)
5. [프로젝트 구조 상세 분석](#5-프로젝트-구조-상세-분석)

---

## 1. SAGE란 무엇인가?

### 1.1 프로젝트 정의

**SAGE (Secure Agent Guarantee Engine)**는 AI 에이전트들이 서로 안전하게 통신할 수 있도록 해주는 블록체인 기반 보안 프레임워크입니다.

#### 쉬운 비유로 이해하기

일반적인 이메일 시스템을 생각해봅시다:

- 이메일을 보낼 때 누구나 내 이름으로 이메일을 보낼 수 있습니다
- 이메일 내용이 중간에 가로채일 수 있습니다
- 받은 이메일이 정말 그 사람이 보낸 것인지 확신하기 어렵습니다

SAGE는 AI 에이전트들 사이의 통신에서 이런 문제들을 해결합니다:

- Yes **신원 보증**: 블록체인을 통해 각 AI 에이전트의 신원을 보증
- Yes **암호화된 통신**: 메시지가 중간에 가로채이더라도 읽을 수 없음
- Yes **서명 검증**: 메시지가 정말 그 에이전트가 보낸 것인지 검증 가능

### 1.2 핵심 기능

#### 기능 1: 종단간 암호화 핸드셰이크 (End-to-End Encrypted Handshake)

**문제**: AI 에이전트 A와 B가 처음 만났을 때 어떻게 안전한 통신 채널을 만들까?

**"TLS 핸드셰이크를 사용하면 되지 않나요?"**

좋은 질문입니다! 하지만 SAGE는 TLS와는 **다른 목적**과 **다른 계층**에서 동작합니다:

```
TLS vs HPKE 핸드셰이크 비교:

┌─────────────────────────────────────────────────────────┐
│                TLS 핸드셰이크                             │
├─────────────────────────────────────────────────────────┤
│ 목적: 전송 계층(Transport Layer) 보안 연결                 │
│ 대상: 클라이언트 ↔ 서버 (네트워크 연결)                      │
│ 특징:                                                   │
│  - 양방향 대화형 프로토콜 (Interactive)                    │
│  - 실시간 협상 필요 (Cipher Suite, 인증서 등)              │
│  - 연결 기반 (Connection-oriented)                       │
│  - 세션이 끊어지면 보안도 종료                              │
│                                                         │
│ 한계:                                                   │
│  No 중간 서버에서 TLS 종료 시 평문 노출                     │
│  No 애플리케이션 계층 데이터는 보호 못함                     │
│  No 메시지 자체의 종단간 암호화 불가능                       │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              HPKE (RFC 9180) 핸드셰이크                   │
├─────────────────────────────────────────────────────────┤
│ 목적: 애플리케이션 계층 메시지 암호화                        │
│ 대상: 에이전트 A ↔ 에이전트 B (논리적 개체)                  │
│ 특징:                                                    │
│  - 단방향 암호화 (One-way encryption)                     │
│  - 비대화형 (Non-interactive) - 상대방 응답 불필요          │
│  - 메시지 기반 (Message-oriented)                         │
│  - 중간 서버와 무관하게 종단간 보호                          │
│                                                         │
│ 장점:                                                    │
│  Yes TLS 위에서 추가 암호화 계층 제공                        │
│  Yes 중간 서버가 메시지 내용을 볼 수 없음                     │
│  Yes 비동기 통신에 최적화 (오프라인 메시지 등)                 │
└─────────────────────────────────────────────────────────┘

왜 HPKE를 사용하는가?

1. 계층 분리 (Layer Separation)
   TLS: 네트워크 연결을 보호 (TCP/IP 레벨)
   HPKE: 메시지 내용을 보호 (Application 레벨)

   Agent A → [TLS 암호화] → [중간 서버] → [TLS 재암호화] → Agent B
            └── HPKE로 메시지는 이미 암호화됨 ──┘

   중간 서버는 TLS는 볼 수 있지만, HPKE로 암호화된 메시지는 못 봄!

2. 비동기 통신 지원
   TLS: 실시간 연결 필요 (양쪽 다 온라인이어야 함)
   HPKE: 수신자 공개키만 있으면 암호화 가능

   예: 이메일처럼 상대방이 오프라인이어도 메시지 암호화 가능

3. 유연성
   TLS: 고정된 프로토콜 스택
   HPKE: 암호화 primitive (원시 도구)
        → 다양한 프로토콜에 삽입 가능
        → 심지어 TLS 내부에서도 사용 (TLS ECH)

4. 성능 최적화
   TLS: 매번 핸드셰이크 오버헤드
   HPKE: 공개키만 있으면 즉시 암호화
        → 메시지당 한 번의 암호화만 필요

실제 사용 사례:
┌─────────────────────────────────────────────────────────┐
│ Agent A가 Agent B에게 메시지 전송                          │
├─────────────────────────────────────────────────────────┤
│ 1. A가 B의 공개키를 블록체인에서 조회                        │
│ 2. HPKE로 메시지 암호화 (B의 공개키 사용)                    │
│ 3. 암호화된 메시지를 TLS로 전송                             │
│ 4. 중간 서버들: TLS는 볼 수 있지만 메시지 내용은 못 봄         │
│ 5. B가 수신: 자신의 개인키로만 HPKE 메시지 복호화 가능         │
│                                                         │
│ 결과: 이중 암호화 (Defense in Depth)                       │
│  - 외부층: TLS (전송 보안)                                 │
│  - 내부층: HPKE (메시지 보안)                              │
└─────────────────────────────────────────────────────────┘
```

**해결책**: HPKE (RFC 9180) 기반 핸드셰이크

HPKE는 TLS를 **대체**하는 것이 아니라, TLS **위에** 추가되는 애플리케이션 계층 암호화입니다.

```
SAGE 핸드셰이크 단계 (HPKE 사용):

1. Init (A → B)
   - A가 DID에서 B의 HPKE 공개키(pkB) 조회
   - HPKE Base: SetupS(pkB, info) → enc(캡슐화), exporter(세션시드)
   - A2A로 {enc, info, exportCtx, nonce, ts} 전송 (Ed25519 서명)

2. Ack (B → A)
   - 서명/ts/nonce 검증, 재생 차단
   - HPKE Base: SetupR(skB, enc, info) → exporter(세션시드)
   - kid 발급, ackTag = HMAC(exporter, ctxID‖nonce‖kid)
   - A2A로 {kid, ackTag} 응답 (Ed25519 서명)

-> 공유 비밀 생성!

3. Complete (A, 로컬)
   - A가 ackTag 검증(상수시간 비교)
   - exporter로 HKDF → 방향별 키(c2s/s2c) 유도, AEAD(ChaCha20-Poly1305) 초기화
   - kid ↔ 세션 바인딩 → 사용 시작

4. 안전한 세션 시작
   - 세션 키로 ChaCha20-Poly1305 암호화
   - Forward Secrecy: 임시 키는 즉시 삭제
```

**사용된 기술**:

- **X25519**: 키 교환 알고리즘 (매우 빠르고 안전함)
- **HPKE (RFC 9180)**: 최신 표준 하이브리드 암호화 primitive
- **Forward Secrecy**: 과거 메시지가 나중에 노출되어도 해독 불가능
- **이중 암호화**: TLS (전송) + HPKE (애플리케이션)

#### 기능 2: RFC 9421 준수 (HTTP 메시지 서명)

**문제**: HTTP 요청이 중간에 변조되지 않았다는 것을 어떻게 보장하나?

**왜 TLS/HTTPS만으로는 부족한가?**

많은 사람들이 "HTTPS를 사용하면 안전한 거 아닌가요?"라고 궁금해합니다. TLS는 훌륭한 보안 프로토콜이지만, AI 에이전트 통신에는 치명적인 한계가 있습니다:

```
TLS의 한계점:

1. 종단간(End-to-End) 보안 불가능
   문제 시나리오:
   AI Agent A ──TLS──→ [로드밸런서] ──TLS──→ AI Agent B
                        ↑
                    TLS 종료 지점

   - TLS는 각 연결 구간마다 독립적으로 동작
   - 로드밸런서에서 TLS가 종료되면 메시지가 평문으로 노출
   - 중간 서버(프록시, CDN, API 게이트웨이)가 메시지를 변조 가능
   - Agent A → Agent B의 진정한 종단간 보안이 보장되지 않음

2. 신원 검증의 한계
   TLS 인증서의 문제:
   - TLS는 "서버"의 신원만 확인 (도메인 소유권)
   - "요청을 보낸 사람"이 누구인지는 증명 못함
   - 클라이언트 인증서는 브라우저/환경에서 설정이 어려움
   - AI 에이전트 간 양방향 신원 확인 필요성을 충족 못함

3. 중간 서버 환경의 현실
   실제 프로덕션 환경:
   Agent A → [방화벽] → [로드밸런서] → [API 게이트웨이]
           → [프록시] → [캐시] → Agent B

   - 각 중간 서버에서 TLS 재암호화 필요
   - 모든 중간 서버를 신뢰해야 함 (Single Point of Trust 문제)
   - 하나의 중간 서버라도 해킹되면 전체 통신 노출

4. 비트단위 메시지 무결성 불가능
   TLS의 검증 범위:
   - TLS는 전송 계층(Transport Layer)만 보호
   - 애플리케이션 계층에서의 메시지 변조는 탐지 못함
   - 예: 프록시가 {"amount": 100}을 {"amount": 1000}으로 변경
   - TLS는 여전히 "안전한 연결"이지만 메시지는 변조됨

5. 부인방지(Non-Repudiation) 불가능
   법적/감사 문제:
   - TLS는 대칭키 암호화 사용 (핸드셰이크 후)
   - "누가 이 요청을 보냈는지" 나중에 증명 불가능
   - 분쟁 발생 시 메시지 출처 입증 어려움
   - 감사 추적(Audit Trail) 신뢰성 낮음

실제 공격 시나리오:
┌─────────────────────────────────────────────────────────┐
│ 침해된 로드밸런서를 통한 공격                               │
├─────────────────────────────────────────────────────────┤
│ 1. 사용자 → "100 USDC를 Alice에게 전송"                   │
│ 2. TLS로 안전하게 로드밸런서 도착                           │
│ 3. 로드밸런서에서 TLS 복호화 (평문 노출!)                   │
│ 4. 공격자가 로드밸런서 제어 중                              │
│ 5. 메시지 변조: "1000 USDC를 Hacker에게 전송"              │
│ 6. 새로운 TLS로 백엔드 서버에 전송                          │
│ 7. 백엔드 서버: "정상적인 TLS 요청"으로 인식                │
│ 8. 결과: 사용자 의도와 다른 거래 실행                       │
└─────────────────────────────────────────────────────────┘

이것이 AI 에이전트 통신에 치명적인 이유:
- AI 에이전트는 수백~수천 개의 중간 서버를 거칠 수 있음
- 각 중간 서버는 다른 조직이 운영할 수 있음
- 금융 거래, 계약 체결 등 중요한 결정을 자동으로 수행
- 한번의 메시지 변조로 수백만 달러 손실 가능
```

**RFC 9421이 해결하는 방법**

RFC 9421 HTTP Message Signatures는 TLS와는 **완전히 다른 계층**에서 보안을 제공합니다:

```
TLS vs RFC 9421 비교:

┌─────────────────────────────────────────────────────────┐
│                    TLS (전송 계층)                         │
├─────────────────────────────────────────────────────────┤
│ 보호 범위: 연결(Connection)                               │
│ 보호 대상: 전송 중인 데이터                                │
│ 생명주기: 연결이 끊어지면 끝                               │
│ 검증자: TLS 종료 지점까지만                                │
└─────────────────────────────────────────────────────────┘
               vs
┌─────────────────────────────────────────────────────────┐
│              RFC 9421 (애플리케이션 계층)                    │
├─────────────────────────────────────────────────────────┤
│ 보호 범위: 메시지(Message)                                │
│ 보호 대상: HTTP 요청/응답 내용                             │
│ 생명주기: 메시지가 존재하는 한 영구적                        │
│ 검증자: 최종 수신자(진짜 Agent B)                          │
└─────────────────────────────────────────────────────────┘

실제 동작 방식:
Agent A가 메시지 전송:
1. HTTP 요청 작성
2. RFC 9421로 메시지 서명 (A의 개인키)
3. TLS로 암호화하여 전송
4. 중간 서버들을 거침 (TLS 종료/재시작 반복)
5. 최종적으로 Agent B 도착
6. Agent B가 서명 검증 (A의 공개키)
   → 중간 서버에서 변조 여부 확인 가능!
   → 정말 Agent A가 보낸 것인지 확인 가능!
```

**해결책**: RFC 9421 표준에 따른 메시지 서명

이는 TLS 위에 추가되는 **보안 계층**으로, 다음을 보장합니다:

- Yes 중간 서버를 신뢰하지 않아도 됨
- Yes 메시지 변조 탐지 가능
- Yes 발신자 신원 암호학적으로 증명
- Yes 나중에 감사/분쟁 해결 가능

```
예시:
원본 HTTP 요청:
POST /api/chat HTTP/1.1
Host: agent-b.example.com
Content-Type: application/json
{"message": "Hello"}

서명 생성 과정:
1. 중요한 부분들을 추출:
   - 메서드: POST
   - 경로: /api/chat
   - 호스트: agent-b.example.com
   - 내용: {"message": "Hello"}

2. 이들을 정규화된 형식으로 결합

3. 개인키로 서명 생성

4. 서명을 헤더에 추가:
   Signature: keyId="agent-a-key-123", algorithm="ed25519",
              signature="base64encodedSignature..."

5. 수신자가 공개키로 서명 검증
```

**실제 코드 위치**: `/core/rfc9421/`

#### 기능 3: 다중 체인 지원

SAGE는 여러 블록체인 네트워크를 지원합니다:

| 블록체인           | 상태         | 용도               |
| ------------------ | ------------ | ------------------ |
| **Ethereum**       | Yes 완전 지원 | 메인넷 프로덕션    |
| **Sepolia**        | Yes 테스트넷  | 개발 및 테스트     |
| **Kaia (Cypress)** | Yes 완전 지원 | 한국 중심 네트워크 |
| **Kairos**         | Yes 테스트넷  | Kaia 테스트 환경   |
| **Solana**         |  개발 중   | 고성능 트랜잭션    |

**왜 여러 체인을 지원하나?**

- 다른 비용: Ethereum은 비싸지만 가장 안전, Kaia는 저렴하고 빠름
- 다른 에코시스템: 각 체인마다 다른 개발자 커뮤니티
- 지역적 선호: Kaia는 한국에서 많이 사용됨

#### 기능 4: 향상된 보안

**공개키 소유권 검증**:

```
문제 시나리오:
악의적인 사용자가 다른 사람의 공개키를 자기 것이라고 등록하려 함

SAGE의 해결책:
1. 공개키를 등록할 때 "챌린지-응답" 과정 필요
2. 시스템이 랜덤 메시지를 생성: "증명해보세요: XYZ123"
3. 사용자는 자신의 개인키로 이 메시지에 서명
4. 시스템이 공개키로 서명 검증
5. 검증 성공 시에만 등록 허용

코드 위치: contracts/ethereum/contracts/SageRegistryV2.sol:148-166
```

**키 취소 기능**:

```
만약 개인키가 노출되었다면?
1. 소유자가 "키 취소" 트랜잭션 전송
2. 스마트 컨트랙트에 취소 기록
3. 이후 해당 키로 서명된 모든 메시지 거부
4. 새 키 쌍 생성 및 재등록

코드 위치: contracts/ethereum/contracts/SageRegistryV2.sol (revokeKey 함수)
```

#### 기능 5: 다중 알고리즘 지원

SAGE는 3가지 암호화 알고리즘을 지원합니다:

**1. Ed25519 (Edwards-curve Digital Signature Algorithm)**

```
용도: 디지털 서명 (신원 확인)
특징:
- 매우 빠른 서명/검증 속도
- 작은 키 크기 (32바이트)
- 높은 보안성

사용 예:
- AI 에이전트의 신원 서명
- DID 문서 서명
- 메시지 인증

코드 위치: crypto/keys/ed25519.go
```

**2. Secp256k1 (Elliptic Curve)**

```
용도: Ethereum과의 호환성
특징:
- Ethereum과 Bitcoin이 사용하는 표준
- ECDSA 서명
- Ethereum 주소 파생

사용 예:
- Ethereum 트랜잭션 서명
- 스마트 컨트랙트 상호작용
- Ethereum 계정 관리

코드 위치: crypto/keys/secp256k1.go
```

**3. X25519 (Curve25519)**

```
용도: 키 교환 (암호화 키 공유)
특징:
- 서명이 아닌 키 교환 전용
- Diffie-Hellman 키 교환
- Forward Secrecy 제공

사용 예:
- 핸드셰이크 시 임시 키 교환
- 세션 키 유도
- HPKE 암호화

코드 위치: crypto/keys/x25519.go
```

**알고리즘 비교표**:

| 기능              | Ed25519      | Secp256k1   | X25519       |
| ----------------- | ------------ | ----------- | ------------ |
| **서명 생성**     | Yes           | Yes          | No           |
| **서명 검증**     | Yes           | Yes          | No           |
| **키 교환**       | No           | No          | Yes           |
| **암호화**        | Warning (변환 후) | No          | Yes           |
| **키 크기**       | 32바이트     | 33/65바이트 | 32바이트     |
| **속도**          |  매우 빠름 |  빠름     |  매우 빠름 |
| **Ethereum 호환** | No           | Yes          | No           |

---

## 2. 왜 SAGE가 필요한가?

### 2.1 AI 에이전트 시대의 보안 문제

**현재 상황**:
2024-2025년 현재, AI 에이전트들이 폭발적으로 증가하고 있습니다:

- ChatGPT, Claude 같은 대화형 AI
- 자동화된 거래 봇
- 코드 작성 에이전트
- 데이터 분석 에이전트

**문제점**:

```
시나리오 1: 신원 사칭
악의적인 에이전트 C가 합법적인 에이전트 B인 척 함
→ 에이전트 A가 C에게 중요한 정보를 보냄
→ 정보 유출!

시나리오 2: 중간자 공격 (Man-in-the-Middle)
에이전트 A ←→ [공격자] ←→ 에이전트 B
→ 공격자가 모든 메시지를 가로채고 수정 가능

시나리오 3: 재생 공격 (Replay Attack)
공격자가 과거에 보낸 합법적인 메시지를 다시 보냄
→ 같은 작업이 여러 번 실행됨 (예: 돈 송금)
```

### 2.2 SAGE의 해결 방법

#### 해결책 1: 블록체인 기반 신원 관리 (DID)

**Decentralized Identifier (DID)**:

```
개념:
- 블록체인에 각 AI 에이전트의 신원 정보 저장
- 중앙 서버 없이 누구나 검증 가능
- 한번 등록되면 변조 불가능

예시 DID:
did:sage:ethereum:0x1234567890abcdef...

구조 설명:
- did: DID 표준 프로토콜
- sage: SAGE 시스템
- ethereum: Ethereum 블록체인
- 0x1234...: 고유 식별자 (Ethereum 주소)
```

**DID 등록 과정** (초보자용 설명):

1. **키 쌍 생성**:

```bash
# 명령어 실행
./sage-crypto generate -t ed25519 -o agent-key.pem

내부 동작:
1. 랜덤한 256비트 숫자 생성 (개인키)
2. 수학 공식으로 공개키 계산
3. 파일로 안전하게 저장

결과:
- agent-key.pem (개인키 - 절대 공유하면 안됨!)
- agent-key.pub (공개키 - 공개해도 됨)
```

2. **스마트 컨트랙트에 등록**:

```bash
./sage-did register \
  --chain ethereum \
  --key agent-key.pem \
  --name "My AI Agent" \
  --endpoint "https://my-agent.com/api" \
  --capabilities "chat,code,analysis"

내부 동작:
1. 공개키 읽기
2. 소유권 증명을 위한 서명 생성
3. Ethereum 트랜잭션 생성:
   - 함수: registerAgent()
   - 파라미터: name, endpoint, publicKey, signature
4. 트랜잭션 전송 (가스비 지불)
5. 블록체인에 영구 기록

결과:
- DID 생성: did:sage:ethereum:0x...
- 누구나 조회 가능한 공개 정보:
  * 이름: "My AI Agent"
  * 엔드포인트: "https://my-agent.com/api"
  * 공개키: 0x1234...
  * 기능: ["chat", "code", "analysis"]
```

3. **DID 조회 및 검증**:

```bash
./sage-did resolve did:sage:ethereum:0x...

내부 동작:
1. Ethereum 노드에 연결
2. 스마트 컨트랙트 읽기 (getAgent 함수 호출)
3. 블록체인에서 데이터 가져오기
4. JSON 형식으로 파싱

반환 데이터:
{
  "did": "did:sage:ethereum:0x...",
  "name": "My AI Agent",
  "endpoint": "https://my-agent.com/api",
  "publicKey": "0x1234...",
  "capabilities": ["chat", "code", "analysis"],
  "active": true,
  "owner": "0xOwnerAddress..."
}
```

#### 해결책 2: 완벽한 Forward Secrecy

**Forward Secrecy란?**

```
비유:
매일 새로운 자물쇠로 바꾸는 집

일반 암호화:
하나의 열쇠로 모든 메시지 암호화
→ 열쇠 노출 시 과거 메시지 전부 노출!

Forward Secrecy:
각 세션마다 다른 임시 열쇠 사용
→ 하나의 열쇠가 노출되어도 다른 세션은 안전
→ 임시 열쇠는 사용 후 즉시 폐기
```

**SAGE의 구현**:

```
핸드셰이크 과정:
1. A가 임시 X25519 키 쌍 생성
   임시_개인키_A, 임시_공개키_A

2. B가 임시 X25519 키 쌍 생성
   임시_개인키_B, 임시_공개키_B

3. 임시 공개키 교환

4. 각자 공유 비밀 계산:
   A: HPKE SetupBaseS(pkB, info) → enc(캡슐화), exporter(공유 비밀).
   B: HPKE SetupBaseR(skB, enc, info) → exporter(공유 비밀, Client와 동일)

5. 공유 비밀에서 세션 키 유도:
   - exporter에서 HKDF로 방향별 키(c2s/s2c enc·sign) 유도 → AEAD  (ChaCha20-Poly1305) 초기화
   - kid ↔ sessionID 바인딩 후 세션 사용 시작

6. 통신 종료 시:
   - 모든 임시 키 삭제
   - 세션 키 삭제
   - 메모리에서 완전히 지움 (0으로 덮어쓰기)

결과:
- 나중에 개인키가 노출되어도 과거 세션 복호화 불가능
- 각 세션은 독립적으로 보호됨

```

#### 해결책 3: 재생 공격 방지 (Replay Protection)

**Nonce (Number used ONCE)**:

```
개념:
- 한 번만 사용되는 임의의 숫자/문자열
- 같은 nonce를 가진 메시지는 거부

예시:
요청 1: {"message": "송금 100달러", "nonce": "abc123"}
→ 처리됨

요청 2: {"message": "송금 100달러", "nonce": "abc123"}
→ 거부! (같은 nonce)

요청 3: {"message": "송금 100달러", "nonce": "def456"}
→ 처리됨 (새로운 nonce)
```

**SAGE의 Nonce 관리**:

```go
// 코드 위치: session/nonce.go

1. Nonce 캐시 생성:
   - 각 세션마다 독립적인 nonce 저장소
   - TTL (Time To Live) 설정 (예: 5분)

2. 메시지 수신 시:
   func (nc *NonceCache) Check(nonce string) bool {
       if nc.seen[nonce] {
           return false  // 이미 사용된 nonce
       }
       nc.seen[nonce] = time.Now()
       return true  // 처음 보는 nonce
   }

3. 자동 청소:
   - 5분마다 오래된 nonce 삭제
   - 메모리 효율성 유지

코드 위치: session/nonce.go:30-60
```

---

## 3. 전체 아키텍처 개요

### 3.1 계층 구조

SAGE는 5개의 주요 계층으로 구성됩니다:

```
┌─────────────────────────────────────────────────────────┐
│           Layer 5: Application Layer                     │
│         (AI 에이전트 애플리케이션)                          │
│                                                           │
│  예: 챗봇, 거래 봇, 분석 에이전트                           │
├─────────────────────────────────────────────────────────┤
│           Layer 4: Protocol Layer                        │
│         (핸드셰이크 및 세션 프로토콜)                        │
│                                                           │
│  - 2단계 핸드셰이크                                        │
│  - 세션 생성 및 관리                                       │
│  - RFC 9421 메시지 서명                                   │
├─────────────────────────────────────────────────────────┤
│           Layer 3: Crypto Layer                          │
│         (암호화 및 키 관리)                                │
│                                                           │
│  - Ed25519 서명                                           │
│  - X25519 키 교환                                         │
│  - ChaCha20-Poly1305 암호화                              │
│  - HKDF 키 유도                                           │
├─────────────────────────────────────────────────────────┤
│           Layer 2: Identity Layer                        │
│         (DID 및 신원 관리)                                │
│                                                           │
│  - DID 등록/조회/업데이트                                  │
│  - 공개키 검증                                            │
│  - 다중 체인 지원                                         │
├─────────────────────────────────────────────────────────┤
│           Layer 1: Blockchain Layer                      │
│         (스마트 컨트랙트 및 블록체인)                       │
│                                                           │
│  - Ethereum: SageRegistryV2                              │
│  - Kaia: SageRegistry                                    │
│  - Solana: (개발 중)                                      │
└─────────────────────────────────────────────────────────┘
```

### 3.2 컴포넌트 간 상호작용

**시나리오: AI 에이전트 A가 에이전트 B에게 메시지 전송**

```
단계별 흐름:

1. 초기화 단계
   ┌─────────┐
   │ Agent A │
   └────┬────┘
        │ 1. DID 조회 요청
        ↓
   ┌─────────────────┐
   │  DID Resolver   │ ← Ethereum/Kaia 블록체인에서 데이터 읽기
   └────┬────────────┘
        │ 2. Agent B의 공개키 및 엔드포인트 반환
        ↓
   ┌─────────┐
   │ Agent A │
   └─────────┘

2. 핸드셰이크 단계 (HPKE 기반)
   ┌─────────┐                    ┌─────────┐
   │ Agent A │                    │ Agent B │
   └────┬────┘                    └────┬────┘
        │                              │
        │                              │
        │ Request (A의 임시 키)          │
        │─────────────────────────────>│
        │                              │
        │ Response (B의 임시 키)         │
        │<─────────────────────────────│
        │                              │
        │ Complete (세션 확정)          │
        │─────────────────────────────>│
        │                              │
        │ ACK (KeyID 발급)             │
        │<─────────────────────────────│

3. 보안 통신 단계
   ┌─────────┐                    ┌─────────┐
   │ Agent A │                    │ Agent B │
   └────┬────┘                    └────┬────┘
        │                              │
        │ 세션 키로 암호화된 메시지      │
        │ + RFC 9421 서명              │
        │─────────────────────────────>│
        │                              │ 1. 서명 검증
        │                              │ 2. 복호화
        │                              │ 3. 메시지 처리
        │                              │
        │ 응답 (암호화 + 서명)          │
        │<─────────────────────────────│
```

---

## 4. 핵심 개념 설명

### 4.1 DID (Decentralized Identifier)

**DID란 무엇인가?**

DID는 중앙 기관 없이 자기 주권적으로 관리할 수 있는 디지털 신원입니다.

**전통적인 신원 vs DID**:

```
전통적인 신원 (중앙화):
┌──────────────────────────────────────┐
│   중앙 서버 (예: Facebook, Google)    │
│   - 사용자 정보 저장                  │
│   - 신원 발급 및 관리                 │
│   - 접근 권한 통제                    │
└──────────┬───────────────────────────┘
           │ 문제점:
           │ 1. 서버 해킹 시 모든 정보 유출
           │ 2. 회사가 망하면 신원 사라짐
           │ 3. 개인 정보 통제권 없음
           ↓
    [사용자들...]

DID (탈중앙화):
┌──────────────────────────────────────┐
│   블록체인 (Ethereum, Kaia 등)        │
│   - 공개키만 저장                     │
│   - 변조 불가능                       │
│   - 24/7 가용                        │
└──────────┬───────────────────────────┘
           │ 장점:
           │ 1. 개인키 소유자만 통제 가능
           │ 2. 영구적으로 존재
           │ 3. 글로벌하게 검증 가능
           ↓
    [사용자들...]
```

**SAGE DID 구조**:

```
DID 예시:
did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc454e4438f44e

파싱 결과:
┌─────────────────────────────────────────────┐
│ did           : DID 표준 프로토콜            │
│ sage          : SAGE 메소드                 │
│ ethereum      : Ethereum 네트워크            │
│ 0x742d35Cc... : Ethereum 주소 (식별자)       │
└─────────────────────────────────────────────┘

블록체인에 저장된 실제 데이터:
{
  "did": "did:sage:ethereum:0x742d35Cc...",
  "owner": "0x742d35Cc...",              // 소유자 주소
  "publicKey": "0x1234abcd...",          // Ed25519 공개키
  "name": "Trading Bot Alpha",           // 에이전트 이름
  "description": "Automated trading",    // 설명
  "endpoint": "https://bot.example.com", // API 엔드포인트
  "capabilities": [                      // 기능 목록
    "crypto-trading",
    "market-analysis",
    "risk-management"
  ],
  "active": true,                        // 활성 상태
  "createdAt": 1704067200,              // 생성 타임스탬프
  "updatedAt": 1704153600               // 업데이트 타임스탬프
}
```

### 4.2 HPKE (Hybrid Public Key Encryption)

**HPKE란?**

HPKE는 공개키 암호화와 대칭키 암호화의 장점을 결합한 최신 암호화 표준입니다 (RFC 9180).

**왜 Hybrid인가?**

```
공개키 암호화만 사용:
장점: 키 교환 안전
단점: 느림 (대용량 데이터 부적합)

대칭키 암호화만 사용:
장점: 매우 빠름
단점: 키를 어떻게 안전하게 공유?

HPKE (두 가지 결합):
1. 공개키 암호화로 대칭키 공유
2. 대칭키로 실제 데이터 암호화
→ 안전하면서도 빠름!
```

**SAGE에서의 HPKE 사용**:

```
핸드셰이크 Request 단계:

발신자 (Agent A):
┌──────────────────────────────────────┐
│ 1. 임시 X25519 키 쌍 생성             │
│    ephPrivA, ephPubA                 │
│                                      │
│ 2. B의 공개키로 HPKE 설정             │
│    suite = KEM_X25519_HKDF_SHA256    │
│           + KDF_HKDF_SHA256          │
│           + AEAD_ChaCha20Poly1305    │
│                                      │
│ 3. 캡슐화 (Encapsulation)            │
│    enc, ctx = Setup(B_pubKey)        │
│    - enc: 캡슐화된 키 (32바이트)      │
│    - ctx: 암호화 컨텍스트             │
│                                      │
│ 4. 메시지 암호화                      │
│    ciphertext = ctx.Seal(plaintext)  │
│                                      │
│ 5. 패킷 생성                          │
│    packet = enc || ciphertext        │
└──────────────────────────────────────┘
           │
           │ packet 전송
           ↓
수신자 (Agent B):
┌──────────────────────────────────────┐
│ 1. 패킷 파싱                          │
│    enc = packet[0:32]                │
│    ciphertext = packet[32:]          │
│                                      │
│ 2. 자신의 개인키로 HPKE 설정          │
│    ctx = Setup(B_privKey, enc)       │
│                                      │
│ 3. 복호화                             │
│    plaintext = ctx.Open(ciphertext)  │
└──────────────────────────────────────┘

코드 위치: crypto/keys/x25519.go:354-423
```

### 4.3 세션 (Session)

**세션이란?**

두 에이전트 간의 보안 통신 채널을 의미하며, 핸드셰이크를 통해 생성됩니다.

**세션의 생명주기**:

```
1. 생성 (Creation)
   ┌─────────────────────────────────────┐
   │ 핸드셰이크 완료 시:                  │
   │                                     │
   │ 입력:                                │
   │ - 공유 비밀 (sharedSecret)          │
   │ - 컨텍스트 ID (contextID)           │
   │ - 임시 공개키들 (ephA, ephB)         │
   │                                     │
   │ 처리:                                │
   │ 1. 세션 시드 유도:                   │
   │    salt = SHA256(label + ctx + ephs)│
   │    seed = HKDF-Extract(secret, salt)│
   │                                     │
   │ 2. 세션 ID 계산:                     │
   │    sid = Base64(SHA256(seed)[0:16]) │
   │                                     │
   │ 3. 방향별 키 유도:                   │
   │    C2S_Enc = HKDF(seed, "c2s|enc")  │
   │    C2S_Sign = HKDF(seed, "c2s|sign")│
   │    S2C_Enc = HKDF(seed, "s2c|enc")  │
   │    S2C_Sign = HKDF(seed, "s2c|sign")│
   │                                     │
   │ 출력:                                │
   │ - SecureSession 객체                │
   │ - 세션 ID                            │
   │ - 암호화/서명 키들                   │
   └─────────────────────────────────────┘

2. 사용 (Usage)
   ┌─────────────────────────────────────┐
   │ 메시지 전송:                         │
   │ plaintext → Encrypt → ciphertext    │
   │                                     │
   │ 서명 생성:                           │
   │ message → HMAC-SHA256 → signature   │
   │                                     │
   │ 메시지 수신:                         │
   │ ciphertext → Decrypt → plaintext    │
   │ signature → Verify → OK/FAIL        │
   │                                     │
   │ 상태 업데이트:                       │
   │ - lastUsedAt 갱신                   │
   │ - messageCount 증가                 │
   └─────────────────────────────────────┘

3. 만료 (Expiration)
   ┌─────────────────────────────────────┐
   │ 만료 조건:                           │
   │                                     │
   │ 1. MaxAge 초과                      │
   │    now > createdAt + MaxAge         │
   │    (예: 1시간 후 무조건 만료)         │
   │                                     │
   │ 2. IdleTimeout 초과                 │
   │    now > lastUsedAt + IdleTimeout   │
   │    (예: 10분간 사용 안하면 만료)      │
   │                                     │
   │ 3. MaxMessages 초과                 │
   │    messageCount >= MaxMessages      │
   │    (예: 10000개 메시지 후 만료)      │
   └─────────────────────────────────────┘

4. 정리 (Cleanup)
   ┌─────────────────────────────────────┐
   │ Close() 호출 시:                     │
   │                                     │
   │ 1. 상태 플래그 설정                  │
   │    closed = true                    │
   │                                     │
   │ 2. 키 재료 안전 삭제                 │
   │    for i in range(len(key)):        │
   │        key[i] = 0                   │
   │                                     │
   │ 3. 대상 키들:                        │
   │    - encryptKey                     │
   │    - signingKey                     │
   │    - sessionSeed                    │
   │    - outKey, inKey                  │
   │    - outSign, inSign                │
   │                                     │
   │ 4. 매니저에서 제거                   │
   │    manager.Remove(sessionID)        │
   └─────────────────────────────────────┘

코드 위치: session/session.go
```

### 4.4 RFC 9421 (HTTP Message Signatures)

**RFC 9421이란?**

HTTP 메시지에 디지털 서명을 추가하여 무결성과 출처를 보장하는 표준입니다.

**서명 과정 상세 설명**:

```
예시 HTTP 요청:
POST /api/chat HTTP/1.1
Host: agent-b.example.com
Content-Type: application/json
Content-Length: 28

{"message": "Hello, Agent B"}

단계 1: 서명할 컴포넌트 선택
┌─────────────────────────────────────┐
│ 선택한 컴포넌트들:                   │
│ - @method: POST                     │
│ - @authority: agent-b.example.com   │
│ - @path: /api/chat                  │
│ - content-type: application/json    │
│ - content-digest: SHA-256=...       │
└─────────────────────────────────────┘

단계 2: 정규화 (Canonicalization)
┌─────────────────────────────────────┐
│ "@method": POST                     │
│ "@authority": agent-b.example.com   │
│ "@path": /api/chat                  │
│ "content-type": application/json    │
│ "content-digest": sha-256=:...      │
│ "@signature-params": ("@method" ... │
│   );created=1704067200              │
└─────────────────────────────────────┘
코드 위치: core/rfc9421/canonicalizer.go

단계 3: 서명 생성
┌─────────────────────────────────────┐
│ signatureInput = [정규화된 문자열]    │
│                                     │
│ signature = Sign(                   │
│     privateKey,                     │
│     signatureInput                  │
│ )                                   │
│                                     │
│ signatureBase64 = Base64(signature) │
└─────────────────────────────────────┘
코드 위치: core/rfc9421/verifier.go:72-75

단계 4: 헤더 추가
┌─────────────────────────────────────┐
│ Signature-Input: sig1=("@method" .. │
│   );created=1704067200;keyid="k1"  │
│                                     │
│ Signature: sig1=:signatureBase64:   │
└─────────────────────────────────────┘

최종 HTTP 요청:
POST /api/chat HTTP/1.1
Host: agent-b.example.com
Content-Type: application/json
Content-Length: 28
Signature-Input: sig1=("@method" "@authority"
  "@path" "content-type" "content-digest");
  created=1704067200;keyid="agent-a-key-123"
Signature: sig1=:MEUCIQDx7+...base64...:

{"message": "Hello, Agent B"}
```

**서명 검증 과정**:

```
수신자 (Agent B):

1. Signature 헤더 파싱
   ┌─────────────────────────────────┐
   │ keyid 추출: "agent-a-key-123"   │
   │ algorithm 추출: "ed25519"       │
   │ signature 추출: [base64 디코드] │
   └─────────────────────────────────┘

2. KeyID로 세션 조회
   ┌─────────────────────────────────┐
   │ session = manager.GetByKeyID(   │
   │     "agent-a-key-123"           │
   │ )                               │
   │                                 │
   │ 세션에서 검증 키 가져오기:        │
   │ verifyKey = session.inSign      │
   └─────────────────────────────────┘

3. 서명 베이스 재구성
   ┌─────────────────────────────────┐
   │ Signature-Input에 명시된         │
   │ 컴포넌트들을 같은 방식으로 정규화  │
   │                                 │
   │ reconstructed = Canonicalize(   │
   │     request,                    │
   │     ["@method", "@authority"...]│
   │ )                               │
   └─────────────────────────────────┘

4. 서명 검증
   ┌─────────────────────────────────┐
   │ valid = Verify(                 │
   │     verifyKey,                  │
   │     reconstructed,              │
   │     signature                   │
   │ )                               │
   │                                 │
   │ if !valid:                      │
   │     return ERROR                │
   └─────────────────────────────────┘

5. 추가 검증
   ┌─────────────────────────────────┐
   │ - Timestamp 확인                │
   │   now - created < MaxSkew       │
   │                                 │
   │ - Nonce 확인 (재생 공격 방지)    │
   │   !nonces.Seen(nonce)           │
   │                                 │
   │ - 세션 만료 확인                 │
   │   !session.IsExpired()          │
   └─────────────────────────────────┘

코드 위치: core/rfc9421/verifier.go:46-70
```

---

## 5. 프로젝트 구조 상세 분석

### 5.1 디렉토리 구조

```
sage/
├── cmd/                      # 실행 가능한 CLI 도구들
│   ├── sage-crypto/         # 암호화 키 관리 CLI
│   │   ├── main.go          # 진입점
│   │   ├── generate.go      # 키 생성 명령어
│   │   ├── sign.go          # 서명 명령어
│   │   ├── verify.go        # 검증 명령어
│   │   └── ...
│   │
│   ├── sage-did/            # DID 관리 CLI
│   │   ├── main.go          # 진입점
│   │   ├── register.go      # DID 등록
│   │   ├── resolve.go       # DID 조회
│   │   ├── update.go        # DID 업데이트
│   │   └── ...
│   │
│   └── sage-verify/         # 메시지 검증 CLI
│       └── main.go
│
├── core/                    # 핵심 프로토콜 구현
│   ├── rfc9421/            # RFC 9421 HTTP 서명
│   │   ├── canonicalizer.go # 메시지 정규화
│   │   ├── verifier.go      # 서명 생성/검증
│   │   ├── parser.go        # 서명 헤더 파싱
│   │   └── types.go         # 데이터 구조
│   │
│   └── message/            # 메시지 처리
│       ├── validator/       # 메시지 검증
│       ├── order/          # 메시지 순서 관리
│       ├── nonce/          # Nonce 관리
│       └── dedupe/         # 중복 제거
│
├── crypto/                  # 암호화 모듈
│   ├── keys/               # 키 쌍 구현
│   │   ├── ed25519.go      # Ed25519 서명 키
│   │   ├── secp256k1.go    # Ethereum 호환 키
│   │   ├── x25519.go       # 키 교환용
│   │   └── constructors.go # 팩토리 함수들
│   │
│   ├── chain/              # 블록체인별 암호화
│   │   ├── ethereum/       # Ethereum 구현
│   │   └── solana/         # Solana 구현
│   │
│   ├── storage/            # 키 저장소
│   │   ├── file.go         # 파일 기반
│   │   └── memory.go       # 메모리 기반
│   │
│   ├── vault/              # 하드웨어 보안
│   │   └── secure_storage.go # OS 키체인 통합
│   │
│   └── formats/            # 키 포맷 변환
│       ├── jwk.go          # JSON Web Key
│       └── pem.go          # PEM 포맷
│
├── did/                     # 탈중앙화 신원
│   ├── manager.go          # DID 통합 관리자
│   ├── resolver.go         # DID 문서 조회
│   ├── registry.go         # DID 등록/업데이트
│   ├── verification.go     # DID 검증
│   │
│   ├── ethereum/           # Ethereum DID 클라이언트
│   │   ├── client.go       # 스마트 컨트랙트 상호작용
│   │   ├── resolver.go     # Ethereum DID 조회
│   │   └── abi.go          # 컨트랙트 ABI 바인딩
│   │
│   └── solana/             # Solana DID 클라이언트
│       ├── client.go
│       └── resolver.go
│
├── handshake/               # 보안 핸드셰이크
│   ├── client.go           # 요청 측 구현
│   ├── server.go           # 응답 측 구현
│   ├── types.go            # 메시지 타입들
│   └── utils.go            # 유틸리티 함수
│
├── session/                 # 세션 관리
│   ├── manager.go          # 세션 생명주기 관리
│   ├── session.go          # SecureSession 구현
│   ├── nonce.go            # 재생 공격 방지
│   ├── metadata.go         # 세션 메타데이터
│   └── types.go            # 인터페이스 정의
│
├── hpke/                    # HPKE 암호화
│   ├── client.go           # 송신자 (Sender)
│   ├── server.go           # 수신자 (Receiver)
│   └── common.go           # 공통 유틸리티
│
├── contracts/               # 스마트 컨트랙트
│   └── ethereum/
│       ├── contracts/       # Solidity 소스코드
│       │   ├── SageRegistryV2.sol  # V2 레지스트리
│       │   ├── interfaces/         # 인터페이스들
│       │   └── test/              # 테스트 컨트랙트
│       │
│       ├── scripts/         # 배포 스크립트
│       │   ├── deploy.js
│       │   └── verify.js
│       │
│       └── test/            # Hardhat 테스트
│           └── SageRegistryV2.test.js
│
├── config/                  # 설정 관리
│   ├── config.go           # 통합 설정 로더
│   ├── blockchain.go       # 블록체인 설정
│   └── validator.go        # 설정 검증
│
├── health/                  # 헬스 체크
│   ├── checker.go          # 컴포넌트 상태 확인
│   └── server.go           # HTTP 엔드포인트
│
├── examples/                # 사용 예제
│   └── mcp-integration/    # MCP 통합 예제
│       ├── basic-demo/     # 기본 데모
│       ├── basic-tool/     # 도구 통합
│       └── vulnerable-vs-secure/  # 보안 비교
│
├── tests/                   # 테스트 코드
│   ├── integration/        # 통합 테스트
│   ├── random/             # 무작위 테스트
│   └── handshake/          # 핸드셰이크 테스트
│
├── docs/                    # 문서
│   ├── handshake/          # 핸드셰이크 문서
│   ├── dev/                # 개발자 가이드
│   └── assets/             # 다이어그램 등
│
├── scripts/                 # 유틸리티 스크립트
│   └── deploy-sage.sh      # 배포 스크립트
│
└── internal/                # 내부 유틸리티
    └── utils/
```

### 5.2 주요 파일 설명

#### 5.2.1 암호화 키 관리

**crypto/keys/ed25519.go** (87줄)

```go
// Ed25519 키 쌍 구조체
type ed25519KeyPair struct {
    privateKey ed25519.PrivateKey  // 64바이트
    publicKey  ed25519.PublicKey   // 32바이트
    id         string               // 16진수 식별자
}

// 핵심 기능:
1. GenerateEd25519KeyPair()
   - 암호학적으로 안전한 난수 생성기 사용
   - 공개키 해시의 첫 8바이트로 ID 생성

2. Sign(message []byte)
   - 메시지에 대한 64바이트 서명 생성
   - 결정론적 (같은 메시지 = 같은 서명)

3. Verify(message, signature []byte)
   - 공개키로 서명 검증
   - O(1) 시간 복잡도

위치: /crypto/keys/ed25519.go:1-88
```

**crypto/keys/x25519.go** (550줄)

```go
// X25519 키 쌍 구조체
type X25519KeyPair struct {
    privateKey *ecdh.PrivateKey  // 32바이트
    publicKey  *ecdh.PublicKey   // 32바이트
    id         string
}

// 핵심 기능:
1. DeriveSharedSecret(peerPubBytes []byte)
   - ECDH로 공유 비밀 계산
   - SHA-256 해싱

2. EncryptWithEd25519Peer(edPeerPub, plaintext)
   - Ed25519 키를 X25519로 변환
   - HKDF로 키 유도
   - AES-256-GCM으로 암호화

3. DecryptWithEd25519Peer(privateKey, packet)
   - 역과정으로 복호화

4. HPKE 헬퍼 함수들:
   - HPKEDeriveSharedSecretToX25519Peer()
   - HPKEOpenSharedSecretWithX25519Priv()
   - HPKESealAndExportToX25519Peer()

위치: /crypto/keys/x25519.go:1-550
```

#### 5.2.2 핸드셰이크 구현

**전체흐름**

```
Client.Initialize()
   └─ HPKE 송신자(setupBaseS): enc, exporter 생성
   └─ Init(payload={enc, info, exportCtx, nonce, ts, initDID, respDID}) 전송(+Ed25519 서명)
   └─ Ack 수신(meta={kid, ackTagB64, ts})
   └─ HMAC(exporter, ctxID|nonce|kid) == ackTag ? 세션 고정 & kid 바인딩 : 실패

Server.SendMessage()
   └─ 메타 서명 검증 & DID 조회
   └─ timestamp 스큐/nonce 재사용 체크
   └─ info/exportCtx 정합성 확인
   └─ HPKE 수신자(setupBaseR): exporter 복원
   └─ 세션 생성(initiator=false), kid 발급/바인딩
   └─ ackTag=HMAC(exporter, ctxID|nonce|kid) 만들어 Ack 반환

```

**hpke/client.go** (152줄)

```go
// 클라이언트 (요청 측) 구조체
type Client struct {
    a2a.A2AServiceClient          // gRPC A2A 스텁
    signer   sagecrypto.KeyPair    // Ed25519(메타데이터 서명)
    did      string                // 클라이언트 DID
    resolver did.Resolver          // 피어 DID → HPKE 공개키 조회
    info     InfoBuilder           // info/exportCtx 빌더
    sessMgr  *session.Manager      // 세션 관리자 (키 바인딩)
}
```

**주요 메서드**:
`Initialize(ctx context.Context, ctxID, initDID, peerDID string)`

Init 전송 → Ack 검증 → 세션 생성 → kid 바인딩까지 끝내는 엔트리포인트

1. peerDID의 HPKE 공개키(X25519) 조회
2. info/exportCtx 빌드
3. HPKE Base(송신자) → enc, exporter(32B) 생성
4. 로컬 세션 선생성: EnsureSessionFromExporterWithRole(exporter, "sage/hpke v1", initiator=true)
5. nonce, ts, enc(B64) 담아 A2A Init 전송(결정론 직렬화 → Ed25519 서명 메타)
6. Ack(meta) 수신 후 kid, ackTagB64 추출
7. 검증: expect = HMAC(exporter, ctxID|nonce|kid) == ackTag
8. OK면 kid → sessionID 바인딩, kid 리턴

```go
// 서버 (응답 측) 구조체
type Server struct {
    a2a.UnimplementedA2AServiceServer

    // HPKE 수신 정적 키쌍(X25519) — enc 복호화에 사용
    hpkePriv keys.X25519KeyPair

    did       string
    resolver  did.Resolver
    sessMgr   *session.Manager
    nonces    *session.NonceCache   // 재생 공격 방지
    maxSkew   time.Duration         // 시계 오차 허용
    info      InfoBuilder
}
```

**주요 메서드**:
`SendMessage(ctx, in *a2a.SendMessageRequest)`

Init 처리 엔트리포인트: 서명/시각/nonce 검증 → HPKE Base 수신자(setupR)로 exporter 획득 → 세션 생성(initiator=false) → kid 발급/바인딩 → Ack 응답

1. TaskId가 TaskHPKEComplete인지 확인, 데이터 파트 파싱
2. 메타 서명 검증(결정론 직렬화 바이트, Ed25519) + DID 조회
3. payload 파싱: enc, info, exportCtx, nonce, ts, initDID, respDID
4. 시간 스큐(±maxSkew, 기본 2m) / 재생 방지(nonce 스토어)
5. info/exportCtx 재계산 후 정합성 검사
6. HPKE Base(수신자) → exporter 복원 (HPKEOpenSharedSecretWithPriv)
7. 세션 생성(initiator=false)
8. kid 발급(옵션: binder.IssueKeyID) & 바인딩
9. ackTag = HMAC(exporter, ctxID|nonce|kid) 생성 → Ack 메타(kid, ackTagB64, ts) 반환

#### 5.2.3 세션 관리

**session/session.go** (664줄)

```go
// 보안 세션 구조체
type SecureSession struct {
    id           string
    createdAt    time.Time
    lastUsedAt   time.Time
    messageCount int
    config       Config
    closed       bool

    // 역할 (initiator = true면 클라이언트)
    initiator    bool

    // 암호화 재료
    sessionSeed  []byte     // HKDF PRK
    encryptKey   []byte     // 단일 방향용
    signingKey   []byte

    // 방향별 키
    outKey       []byte     // 송신 암호화
    inKey        []byte     // 수신 암호화
    outSign      []byte     // 송신 서명
    inSign       []byte     // 수신 서명

    // AEAD 인스턴스
    aead         cipher.AEAD    // 레거시
    aeadOut      cipher.AEAD    // 송신용
    aeadIn       cipher.AEAD    // 수신용
}

// 핵심 함수:

1. NewSecureSessionFromExporterWithRole()
   - HPKE 익스포터 비밀에서 세션 생성
   - initiator에 따라 키 방향 결정

2. DeriveSessionSeed()
   - HKDF-Extract로 PRK 생성
   - Salt = SHA256(label + contextID + ephKeys)

3. ComputeSessionIDFromSeed()
   - 결정론적 세션 ID 생성
   - Base64(SHA256(seed)[0:16])

4. deriveDirectionalKeys()
   - c2s (client-to-server) 키 유도
   - s2c (server-to-client) 키 유도
   - 각각 암호화 키 + 서명 키

5. EncryptOutbound() / DecryptInbound()
   - ChaCha20-Poly1305 사용
   - Nonce (12바이트) + Ciphertext

6. Close()
   - 모든 키 메모리를 0으로 덮어쓰기
   - 보안 삭제

위치: /session/session.go:1-664
```

**session/manager.go**

```go
// 세션 관리자
type Manager struct {
    sessions map[string]*SecureSession  // sessionID -> Session
    keyIDs   map[string]string         // keyID -> sessionID
    mu       sync.RWMutex
    config   Config
}

// 핵심 기능:

1. CreateSession()
   - 새 세션 생성 및 등록
   - 고유 ID 할당

2. BindKeyID()
   - KeyID를 세션 ID에 바인딩
   - RFC 9421 서명에 사용

3. GetByKeyID()
   - KeyID로 세션 조회
   - 서명 검증 시 사용

4. CleanupExpired()
   - 주기적으로 만료 세션 제거
   - 메모리 관리

위치: /session/manager.go
```

### 5.3 데이터 흐름 요약

```
완전한 통신 플로우:

1. 초기화
   ┌──────────────────────────────────────┐
   │ Agent A                              │
   │ - Ed25519 키 로드                    │
   │ - DID 설정                           │
   │ - gRPC 연결 수립                     │
   └──────────────────────────────────────┘

2. DID 조회
   ┌──────────────────────────────────────┐
   │ A → Ethereum: getAgent(B의 주소)     │
   │ Ethereum → A: B의 메타데이터          │
   │   - 공개키                            │
   │   - 엔드포인트                        │
   │   - 기능                              │
   └──────────────────────────────────────┘

3. 핸드셰이크 (2단계)
   [상세는 Part 4에서 다룸]

4. 보안 통신
   ┌──────────────────────────────────────┐
   │ A: 메시지 작성                        │
   │ ↓                                    │
   │ A: 세션 키로 암호화                   │
   │ ↓                                    │
   │ A: RFC 9421 서명 생성                │
   │ ↓                                    │
   │ A → B: 암호화된 메시지 + 서명          │
   │ ↓                                    │
   │ B: 서명 검증                          │
   │ ↓                                    │
   │ B: 복호화                             │
   │ ↓                                    │
   │ B: 메시지 처리                        │
   └──────────────────────────────────────┘

5. 세션 종료
   ┌──────────────────────────────────────┐
   │ 조건 체크:                            │
   │ - MaxAge 초과?                       │
   │ - IdleTimeout 초과?                  │
   │ - MaxMessages 초과?                  │
   │ ↓                                    │
   │ YES: 세션 정리                        │
   │ - 모든 키 삭제                        │
   │ - 매니저에서 제거                     │
   │ - 리소스 해제                         │
   └──────────────────────────────────────┘
```

---

## 다음 파트 예고

**Part 2: 암호화 시스템 Deep Dive**에서는 다음 내용을 다룹니다:

1. Ed25519, Secp256k1, X25519의 수학적 원리
2. HPKE의 내부 동작 원리
3. ChaCha20-Poly1305 AEAD 암호화
4. HKDF 키 유도 함수 상세
5. 키 변환 과정 (Ed25519 ↔ X25519)
6. 실제 코드 예제와 테스트

**Part 3: DID 및 블록체인 통합**에서는:

1. Ethereum 스마트 컨트랙트 상세 분석
2. 온체인 데이터 구조
3. 가스 최적화 기법
4. 다중 체인 지원 구현
5. DID 조회 및 캐싱 전략

```

```
