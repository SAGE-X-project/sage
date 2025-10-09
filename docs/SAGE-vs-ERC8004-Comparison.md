# SAGE vs ERC-8004: 핵심 차이점 분석

## 핵심 발견사항

**ERC-8004는 Agent Management에 집중하고, SAGE는 Message Integrity까지 보장합니다.**

---

## 상세 비교

### 1. 보안 범위 (Security Scope)

| 항목 | ERC-8004 + A2A | SAGE |
|------|---------------|------|
| **Agent Identity** | Yes Identity Registry | Yes SageRegistryV2 (DID) |
| **Agent Reputation** | Yes Reputation Registry | No (향후 구현 예정) |
| **Agent Validation** | Yes Validation Registry | Warning Public Key Ownership Proof |
| **Message Signing** | Warning A2A Layer (선택적) | Yes **RFC 9421 HTTP Signatures (필수)** |
| **Message Integrity** | No **애플리케이션 레이어** | Yes **프로토콜 레벨 보장** |
| **Real-time Verification** | Warning Push Notification JWT만 | Yes **모든 메시지** |
| **Replay Attack Prevention** | Warning Push Notification만 | Yes **Nonce 관리** |
| **Message Ordering** | No | Yes **Sequence 기반** |

---

## ERC-8004의 범위

### What ERC-8004 Provides Yes

#### 1. Agent-Level Trust (에이전트 신뢰)
```
목적: "이 에이전트를 믿을 수 있는가?"
메커니즘:
  - Identity Registry: 에이전트 식별
  - Reputation Registry: 평판 기록
  - Validation Registry: 작업 결과 검증
```

#### 2. Task-Level Validation (작업 검증)
```
목적: "에이전트가 작업을 제대로 수행했는가?"
메커니즘:
  - DataHash: 작업 결과의 해시 커밋
  - Validator Agents: 재실행 또는 TEE attestation
  - 사후 검증 (Post-execution)
```

#### 3. Long-term Accountability (장기 책임성)
```
목적: "나중에 문제가 생기면 추적 가능한가?"
메커니즘:
  - On-chain audit trail
  - Permanent records
  - Dispute resolution
```

### What ERC-8004 Does NOT Provide No

#### 1. Real-time Message Integrity (실시간 메시지 무결성)
```
문제: "지금 받은 메시지가 변조되지 않았는가?"
ERC-8004: No 보장 안함
A2A Protocol: Warning TLS에 의존 (전송 계층)
```

#### 2. Message Authentication (메시지 인증)
```
문제: "이 메시지가 정말 해당 에이전트가 보낸 것인가?"
ERC-8004: No 직접 다루지 않음
A2A Protocol: Warning HTTP 인증 (Bearer Token, API Key)
```

#### 3. Message-level Cryptographic Proof (메시지 레벨 암호학적 증명)
```
문제: "메시지 내용을 부인할 수 없는 증거가 있는가?"
ERC-8004: No DataHash만 (작업 결과에 대해)
A2A Protocol: Warning Push Notification JWT만
```

---

## SAGE의 추가 보안 계층

### What SAGE Provides (Beyond ERC-8004) Yes

#### 1. RFC 9421 HTTP Message Signatures (메시지 서명)

**모든 메시지**에 대해 암호학적 서명을 제공:

```http
POST /protected HTTP/1.1
Host: server.example.com
Content-Digest: sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:
Signature-Input: sig1=("@method" "@path" "content-digest" "date");
                      keyid="session:abc123";
                      created=1618884473;
                      nonce="n-12345"
Signature: sig1=:K2qGT5srn2OGbOIDzQ6kYT+ruaycnDAAUpKv+ePFfD0RAxn/1BUeZx/Kdrq32DrfakQ6bPsvB9aqZqognNT6be4olHROIkeV879RrsrObury8L9SCEibeoHyqU/yCjphSmEdd7WD+zrchK57quskKwRefy2iEC5S2uAH0EPyOZKWlvbKmKu5q4CaB8X/I5/+HLZLGvDiezqi6/7p2Gngf+hwZ0lSdy39vyNMaaAT0tKo6nuVw0S1MVg1Q7MpWYZs0soHjttq0uLIA3DIbQfLiIvK6/l0BdWTU7+2uQj7lBkQAsFZx
```

**핵심 보장:**
- Yes **메시지 무결성**: Content-Digest로 본문 변조 방지
- Yes **발신자 인증**: KeyID + Signature로 신원 증명
- Yes **Replay 방지**: Nonce + Created timestamp
- Yes **부인 방지 (Non-repudiation)**: 암호학적 서명으로 부인 불가

#### 2. Session-based Encryption (세션 기반 암호화)

**Handshake 프로토콜**로 안전한 세션 수립:

```
Phase 1: Invitation (공개 메시지)
  ↓
Phase 2: Request (HPKE 암호화, X25519 ephemeral key 교환)
  ↓
Phase 3: Response (서버 ephemeral key 전달)
  ↓
Phase 4: Complete (Shared Secret 도출)
  ↓
Session Established: HKDF로 암호화/서명 키 생성
```

**보안 속성:**
- Yes **Forward Secrecy**: Ephemeral key 사용
- Yes **Mutual Authentication**: 양방향 신원 확인
- Yes **End-to-End Encryption**: 메시지 본문 암호화

#### 3. Real-time Message Validation (실시간 메시지 검증)

**모든 수신 메시지**를 즉시 검증:

```go
// core/message/validator/validator.go
func ValidateMessage(
    msg *message.Message,
    sessionID string,
    mgr SessionManager,
) (*ValidationResult, error) {
    // 1. Timestamp 검증 (Clock skew 허용)
    if !isTimestampValid(msg.Timestamp, 5*time.Minute) {
        return &ValidationResult{Valid: false, Reason: "timestamp_out_of_range"}
    }

    // 2. Nonce 중복 검사 (Replay Attack 방지)
    if nonceCache.IsUsed(msg.Nonce) {
        return &ValidationResult{Valid: false, Reason: "replay_detected"}
    }

    // 3. Sequence 검증 (Message Ordering)
    if !orderMgr.CheckSequence(sessionID, msg.Sequence, msg.Timestamp) {
        return &ValidationResult{Valid: false, Reason: "out_of_order"}
    }

    // 4. Signature 검증
    if !verifySignature(msg) {
        return &ValidationResult{Valid: false, Reason: "invalid_signature"}
    }

    return &ValidationResult{Valid: true}
}
```

#### 4. Multi-layer Defense (다층 방어)

```
Layer 1: TLS (Transport)
  ↓
Layer 2: HTTP Signatures (Message)
  ↓
Layer 3: Session Encryption (Payload)
  ↓
Layer 4: Message Validation (Nonce, Sequence, Timestamp)
  ↓
Layer 5: Application Logic
```

---

## 📊 구체적인 시나리오 비교

### 시나리오 1: 실시간 메시지 변조 공격

**공격**: 중간자(MITM)가 메시지 내용을 변경 시도

#### ERC-8004 + A2A 방어
```
1. TLS 연결 (암호화된 전송)
   Yes 네트워크 레벨 보호
   No TLS 종료 지점(Proxy) 이후 취약

2. Application-level 검증 없음
   No 메시지 본문 무결성 검증 X
   No 서명 검증 선택적

결과: Warning TLS 신뢰 필수, 종단간 보장 부족
```

#### SAGE 방어
```
1. TLS 연결 (암호화된 전송)
   Yes 네트워크 레벨 보호

2. HTTP Message Signature
   Yes Content-Digest로 본문 해시 검증
   Yes Signature로 발신자 인증
   Yes Proxy를 거쳐도 무결성 보장

3. Session Encryption
   Yes 본문 자체도 세션키로 재암호화

결과: Yes 종단간(End-to-End) 무결성 보장
```

---

### 시나리오 2: Replay Attack (재전송 공격)

**공격**: 이전에 전송된 유효한 메시지를 다시 전송

#### ERC-8004 + A2A 방어
```
1. Push Notification만 JWT + Nonce 검증
   Yes 푸시 알림은 보호됨
   No 일반 메시지는 보호 안됨

2. 애플리케이션이 직접 구현 필요
   Warning 개발자 책임

결과: Warning 표준에서 보장하지 않음
```

#### SAGE 방어
```
1. 모든 메시지에 Nonce 필수
   Yes Signature-Input의 nonce 파라미터

2. Nonce Cache로 중복 검사
   Yes core/message/nonce 패키지
   Yes 자동으로 만료된 Nonce 정리

3. Timestamp 검증
   Yes Clock skew 허용 범위 설정
   Yes 오래된 메시지 거부

결과: Yes 프로토콜 레벨에서 자동 방어
```

---

### 시나리오 3: Out-of-Order Message (순서 뒤바뀜)

**공격**: 메시지 순서를 바꿔서 혼란 유발

#### ERC-8004 + A2A 방어
```
1. 메시지 순서 보장 없음
   No A2A Protocol에 sequence 개념 없음

2. Task ID로만 연관성 추적
   Warning 작업 단위 추적만 가능
   No 메시지 순서 보장 X

결과: No 순서 보장 안됨
```

#### SAGE 방어
```
1. Sequence Number 기반 순서 관리
   Yes core/message/order 패키지

2. Timestamp와 Sequence 조합 검증
   Yes 단조증가(Monotonic) 검증
   Yes 시간 역행 감지

3. Session 별 격리
   Yes 세션마다 독립적인 Sequence

결과: Yes 엄격한 메시지 순서 보장
```

---

## 역할 구분

### ERC-8004의 역할: "Agent Marketplace & Reputation"

```
목표: 신뢰할 수 있는 에이전트 발견 및 선택
초점:
  - "어떤 에이전트를 선택할까?" (Identity)
  - "이 에이전트는 신뢰할 수 있나?" (Reputation)
  - "작업 결과가 맞나?" (Validation)
시간축: 사전 선택 + 사후 검증
```

### SAGE의 역할: "Secure Communication Channel"

```
목표: 실시간 안전한 메시지 전송
초점:
  - "지금 받은 메시지가 진짜인가?" (Authentication)
  - "내용이 변조되지 않았나?" (Integrity)
  - "재전송 공격은 아닌가?" (Replay Prevention)
  - "순서가 맞나?" (Ordering)
시간축: 실시간 통신 중
```

---

## 상호 보완성

ERC-8004와 SAGE는 **경쟁 관계가 아니라 상호 보완 관계**입니다:

```
┌─────────────────────────────────────────────┐
│         ERC-8004: Agent Trust Layer         │
│  (누구를 신뢰할 것인가? - Long-term)          │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  Identity Registry: 에이전트 식별     │   │
│  │  Reputation Registry: 평판 관리      │   │
│  │  Validation Registry: 작업 검증      │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↓
        에이전트 선택 및 신뢰 결정
                    ↓
┌─────────────────────────────────────────────┐
│       SAGE: Secure Message Protocol         │
│  (메시지가 안전한가? - Real-time)             │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  RFC 9421: 메시지 서명               │   │
│  │  Handshake: 세션 수립                │   │
│  │  Encryption: 본문 암호화             │   │
│  │  Validation: Nonce/Sequence 검증    │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↓
        안전한 메시지 전송 및 수신
                    ↓
┌─────────────────────────────────────────────┐
│         Application Business Logic          │
│  (작업 수행 및 결과 생성)                     │
└─────────────────────────────────────────────┘
                    ↓
        작업 완료 후
                    ↓
┌─────────────────────────────────────────────┐
│      ERC-8004: Post-execution Feedback      │
│  (작업이 제대로 수행되었나? - Post-validation) │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  DataHash: 결과 커밋                 │   │
│  │  Validation: 재실행 또는 TEE         │   │
│  │  Reputation: 피드백 기록             │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

---

## 완전한 보안 스택 구축

### SAGE + ERC-8004 통합 시나리오

```typescript
// 1. ERC-8004로 신뢰할 수 있는 에이전트 발견
const agents = await identityRegistry.searchAgents({
  capability: "medical-diagnosis",
  minReputation: 4.5
});

// 2. 평판 확인
const reputation = await reputationRegistry.getReputation(agents[0].address);
if (reputation.score < 4.5) {
  throw new Error("Insufficient reputation");
}

// 3. SAGE Handshake로 안전한 세션 수립
const session = await sageClient.handshake(agents[0].endpoint);

// 4. SAGE로 안전하게 메시지 전송 (RFC 9421 서명 + 암호화)
const response = await session.sendMessage({
  type: "diagnosis-request",
  data: encryptedPatientData,
  // SAGE가 자동으로 처리:
  // - Content-Digest 생성
  // - Signature 생성
  // - Nonce 추가
  // - Sequence 관리
});

// 5. SAGE가 자동으로 응답 검증
// - Signature 검증
// - Nonce 중복 확인
// - Sequence 순서 확인
// - Timestamp 유효성 확인

// 6. 작업 완료 후 ERC-8004에 결과 기록
const dataHash = keccak256(response.diagnosisResult);
await validationRegistry.requestValidation(
  taskId,
  dataHash,
  ValidationType.TEE
);

// 7. 검증 완료 후 피드백
await reputationRegistry.submitFeedback(
  taskId,
  agents[0].address,
  dataHash,
  5 // 5-star rating
);
```

---

## SAGE의 차별화 가치

### 1. 즉시 사용 가능한 보안 (Out-of-the-box Security)

**ERC-8004:**
```javascript
// 개발자가 직접 구현 필요
app.post('/message', async (req, res) => {
  // Warning 메시지 검증 로직을 직접 작성해야 함
  // Warning Nonce 관리를 직접 구현해야 함
  // Warning Signature 검증을 직접 구현해야 함

  // ... 비즈니스 로직
});
```

**SAGE:**
```javascript
// 프레임워크가 자동으로 처리
app.post('/message', sageMiddleware.verify, async (req, res) => {
  // Yes 이미 검증된 메시지만 도달
  // Yes Signature 자동 검증 완료
  // Yes Nonce 자동 검사 완료
  // Yes Sequence 자동 확인 완료

  // ... 비즈니스 로직만 작성
});
```

### 2. 표준 준수 (Standards Compliance)

**ERC-8004:**
- Yes Ethereum ERC 표준
- Warning 메시지 보안은 별도 표준 필요

**SAGE:**
- Yes RFC 9421 (HTTP Message Signatures) - **IETF 표준**
- Yes HPKE (RFC 9180) - 하이브리드 공개키 암호화
- Yes HKDF (RFC 5869) - 키 도출 함수
- Yes Ed25519 (RFC 8032) - 디지털 서명

### 3. 감사 가능성 (Auditability)

**ERC-8004:**
```
감사 대상: 작업 결과 (Task output)
시점: 사후 (Post-execution)
방법: On-chain DataHash 비교
```

**SAGE:**
```
감사 대상: 모든 메시지 (All messages)
시점: 실시간 (Real-time) + 사후 (Post-execution)
방법:
  1. HTTP Signature logs
  2. Session encryption metadata
  3. Nonce/Sequence audit trail
```

---

## 결론

### 핵심 차이점 요약

| 측면 | ERC-8004 | SAGE |
|------|----------|------|
| **보안 계층** | Agent-level | **Message-level** |
| **보장 시점** | Pre-selection + Post-validation | **Real-time** |
| **무결성 보장** | Task output (DataHash) | **Every message** |
| **표준 준수** | Ethereum ERC | **IETF RFC** |
| **구현 부담** | 개발자가 메시지 보안 구현 | **프레임워크 제공** |
| **감사 범위** | 작업 결과 | **모든 통신** |

### 상호 보완성

```
ERC-8004: "누구와 통신할 것인가?" (WHO)
SAGE: "어떻게 안전하게 통신할 것인가?" (HOW)

함께 사용 시:
Yes 신뢰할 수 있는 에이전트 선택 (ERC-8004)
Yes 안전한 실시간 통신 (SAGE)
Yes 작업 결과 검증 (ERC-8004)
Yes 완전한 감사 추적 (Both)
```

### SAGE의 독자적 가치

1. **실시간 메시지 무결성** - ERC-8004가 다루지 않는 영역
2. **표준 기반 구현** - IETF RFC 준수로 상호운용성 보장
3. **개발자 경험** - 복잡한 암호학적 보안을 프레임워크가 처리
4. **종단간 보안** - TLS 종료 지점 이후에도 보안 유지

### 권장 사항

**SAGE 프로젝트는:**
1. Yes ERC-8004 Identity Registry 구현 (이미 완료)
2. Yes ERC-8004 Reputation Registry 추가 (향후)
3. Yes **메시지 보안을 핵심 차별화 요소로 강조**
4. Yes "ERC-8004 호환 + 메시지 무결성 보장" 마케팅

**왜냐하면:**
- ERC-8004는 Agent Management에 집중
- SAGE는 Secure Communication에 집중
- 둘은 상호 보완적이며, SAGE가 ERC-8004의 부족한 부분을 채움

---

*문서 작성일: 2025-10-06*
*작성자: SAGE Development Team*
*버전: 1.0*
