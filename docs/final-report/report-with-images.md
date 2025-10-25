# SAGE (Secure Agent Guarantee Engine)
## 2025 오픈소스 개발자대회 최종 발표자료 (시각화 버전)

---

## 목차

1. [프로젝트 개요](#프로젝트-개요)
2. [팀 소개](#팀-소개)
3. [개발 배경: 다가오는 위기](#개발-배경-다가오는-위기)
4. [개발 목적](#개발-목적)
5. [프로젝트 구성 및 기능](#프로젝트-구성-및-기능)
6. [추후 활용 방안](#추후-활용-방안-및-계획)
7. [시연 계획](#시연-계획)
8. [Appendix](#appendix-사용된-오픈소스-목록)

---

## 프로젝트 개요

### "AI Agent 시대, 보안은 선택이 아닌 필수입니다"

![Diagram 1](assets/diagrams/diagram-01.png)

**핵심 메시지:**
- **문제**: AI Agent 간 통신의 보안 취약점 (중간자 공격, 메시지 변조, 신원 위조)
- **솔루션**: RFC 표준 기반 메시지 서명/암호화 + 블록체인 기반 투명한 Agent 검증
- **임팩트**: 개인정보 유출 방지, 금융 자산 보호, 신뢰할 수 있는 Agent 생태계 구축

### 핵심 가치 제안

![Diagram 2](assets/diagrams/diagram-02.png)

---

## 팀 소개

**SAGE-X Project Team**

(추후 팀원 정보 추가)

---

## 개발 배경: 다가오는 위기

### 1. AI Agent 시대의 도래

![Diagram 3](assets/diagrams/diagram-03.png)

#### 1.1 누구나 Agent를 만드는 시대

![Diagram 4](assets/diagrams/diagram-04.png)

#### 1.2 Agent 사용 확산 통계

![Diagram 5](assets/diagrams/diagram-05.png)

### 2. 현재 보안의 심각한 한계

#### 2.1 TLS/HTTPS의 근본적 한계

![Diagram 6](assets/diagrams/diagram-06.png)

**vs SAGE의 종단간 보안:**

![Diagram 7](assets/diagrams/diagram-07.png)

#### 2.2 학술 논문이 경고하는 위험

![Diagram 8](assets/diagrams/diagram-08.png)

### 3. 예상되는 피해 규모

#### 3.1 보안 사고 비용 비교

![Diagram 9](assets/diagrams/diagram-09.png)

#### 3.2 실제 사례: SKT 해킹 사태

| 항목 | 내용 |
|------|------|
| **피해 규모** | 약 1,200만 고객 정보 유출 |
| **직접 비용** | 수백억 원 (배상, 복구) |
| **간접 비용** | 신뢰 상실, 브랜드 훼손 |
| **사회적 비용** | 개인정보 보호 인식 저하 |
| **교훈** | **사전 예방 > 사후 대응** |

### 4. 왜 지금 시작해야 하는가?

#### 4.1 역사는 반복된다

![Diagram 10](assets/diagrams/diagram-10.png)

**핵심 메시지:**
- HTTP → HTTPS 전환: **보안 사고 후 10년+ 소요**
- 모바일 앱 보안: **수많은 피해 후 규제**
- AI Agent: **지금 선제적 대응 필요!**

#### 4.2 선제적 대응의 가치

![Diagram 11](assets/diagrams/diagram-11.png)

---

## 개발 목적

### Trust Layer로 안전한 AI Agent 생태계 구축

![Diagram 12](assets/diagrams/diagram-12.png)

### 핵심 목표

![Diagram 13](assets/diagrams/diagram-13.png)

---

## 🏗️ 프로젝트 구성 및 기능

### 1. 전체 아키텍처

```mermaid
graph TB
    subgraph "Application Layer"
        APP[AI Agents<br/>ChatGPT, Gemini, Claude, etc.]
    end

    subgraph "SAGE Trust Layer"
        SIGN[RFC-9421<br/>Message Signing]
        ENCRYPT[RFC-9180<br/>HPKE Encryption]
        DID[DID Management<br/>Blockchain]
        CRYPTO[Crypto Engine<br/>Secp256k1, Ed25519]
        SESSION[Session Manager]
        STORAGE[Secure Storage<br/>Vault]

        SIGN --- ENCRYPT
        ENCRYPT --- DID
        DID --- CRYPTO
        CRYPTO --- SESSION
        SESSION --- STORAGE
    end

    subgraph "Integration Layer"
        BC[Blockchain<br/>Ethereum, ...]
        SDK[Multi-Lang SDKs<br/>Go, Python, TS, Java]
        CLI[CLI Tools<br/>sage-crypto, sage-did]
    end

    APP --> SIGN
    SIGN --> BC
    ENCRYPT --> SDK
    DID --> CLI

    style "SAGE Trust Layer" fill:#e3f2fd
```

### 2. 핵심 기능

#### 2.1 RFC-9421 메시지 서명

```mermaid
sequenceDiagram
    autonumber
    participant A as Agent A
    participant S as SAGE Signer
    participant G as Gateway
    participant V as SAGE Verifier
    participant B as Agent B

    Note over A,B: RFC-9421 HTTP Message Signatures

    A->>S: 메시지 전송 요청
    S->>S: 1. Signature Base 생성<br/>2. 개인키로 서명<br/>3. Signature-Input 헤더 생성<br/>4. Signature 헤더 생성
    S->>G: HTTP Request<br/>+ Signature-Input<br/>+ Signature

    Note over G: Gateway는 내용을<br/>변조할 수 있지만<br/>서명은 재생성 불가

    G->>V: 메시지 전달
    V->>V: 1. Signature Base 재생성<br/>2. 공개키로 서명 검증<br/>3. Timestamp 확인<br/>4. Nonce 검증

    alt 검증 성공
        V->>B: ✅ 메시지 전달
    else 검증 실패
        V->>A: ❌ 거부 (변조 감지)
    end

    style V fill:#51cf66
```

**서명 헤더 구조:**

```
Signature-Input: sig1=("@method" "@path" "content-type" "content-digest");
                 created=1618884473;
                 expires=1618884773;
                 nonce="b3k6t4n7"

Signature: sig1=:K2qGT5srn2OGbOIDzQ6kYT+ruaycnDAAUpKv+ePFfD6/...:
```

#### 2.2 RFC-9180 HPKE 암호화

![Diagram 16](assets/diagrams/diagram-16.png)

**HPKE 암호화 과정:**

![Diagram 17](assets/diagrams/diagram-17.png)

#### 2.3 블록체인 기반 DID 관리

![Diagram 18](assets/diagrams/diagram-18.png)

**DID Document 구조:**

```json
{
  "id": "did:sage:ethereum:0x1234567890abcdef",
  "publicKey": [{
    "id": "did:sage:ethereum:0x1234567890abcdef#keys-1",
    "type": "Secp256k1VerificationKey2019",
    "controller": "did:sage:ethereum:0x1234567890abcdef",
    "publicKeyHex": "04abc123..."
  }],
  "service": [{
    "id": "did:sage:ethereum:0x1234567890abcdef#agent-service",
    "type": "AgentService",
    "serviceEndpoint": "https://agent.example.com"
  }],
  "metadata": {
    "name": "Payment Agent",
    "description": "Secure payment processing agent",
    "version": "1.0.0",
    "category": "finance"
  }
}
```

### 3. 시스템 구성도 (상세)

```mermaid
graph TB
    subgraph "SAGE Core Engine"
        direction TB

        subgraph "Security Layer"
            RFC9421[RFC-9421 Signer/Verifier]
            RFC9180[RFC-9180 HPKE]
            NONCE[Nonce Manager]
        end

        subgraph "Identity Layer"
            DID[DID Manager]
            DIDRES[DID Resolver]
            META[Metadata Handler]
        end

        subgraph "Crypto Layer"
            SECP[Secp256k1 Engine]
            ED[Ed25519 Engine]
            X25519[X25519 ECDH]
            CHACHA[ChaCha20Poly1305]
        end

        subgraph "Storage Layer"
            VAULT[Encrypted Vault]
            CACHE[Cache Manager]
            SESSION[Session Store]
        end
    end

    subgraph "External Integration"
        BC[Blockchain<br/>Ethereum]
        RPC[RPC Provider<br/>Alchemy/Infura]
    end

    subgraph "Developer Interface"
        SDK_GO[Go SDK]
        SDK_PY[Python SDK]
        SDK_TS[TypeScript SDK]
        SDK_JAVA[Java SDK]
        CLI[CLI Tools]
    end

    RFC9421 --> SECP
    RFC9421 --> ED
    RFC9180 --> X25519
    RFC9180 --> CHACHA

    DID --> BC
    DIDRES --> RPC

    RFC9421 --> VAULT
    SESSION --> CACHE

    SDK_GO --> RFC9421
    SDK_PY --> RFC9180
    SDK_TS --> DID
    SDK_JAVA --> RFC9421
    CLI --> DID

    style "SAGE Core Engine" fill:#e3f2fd
    style "Security Layer" fill:#fff3e0
    style "Identity Layer" fill:#f3e5f5
```

### 4. 플러그인 아키텍처

![Diagram 20](assets/diagrams/diagram-20.png)

### 5. 검증 및 품질 보증

#### 5.1 테스트 커버리지

![Diagram 21](assets/diagrams/diagram-21.png)

#### 5.2 품질 메트릭

![Diagram 22](assets/diagrams/diagram-22.png)

---

## 추후 활용 방안 및 계획

### 로드맵 타임라인

![Diagram 23](assets/diagrams/diagram-23.png)

### 주요 마일스톤

![Diagram 24](assets/diagrams/diagram-24.png)

### SAGE-ADK 개념도

![Diagram 25](assets/diagrams/diagram-25.png)

### Agent Marketplace 개념도

![Diagram 26](assets/diagrams/diagram-26.png)

---

## 시연 계획

### 시연 인프라 구성

![Diagram 27](assets/diagrams/diagram-27.png)

### 시연 시나리오 플로우

#### 시나리오 1: 보안 미적용 (공격 성공)

![Diagram 28](assets/diagrams/diagram-28.png)

#### 시나리오 2: SAGE 적용 (공격 차단)

![Diagram 29](assets/diagrams/diagram-29.png)

#### 시나리오 3: 개인정보 암호화

![Diagram 30](assets/diagrams/diagram-30.png)

#### 시나리오 4: Agent 신원 검증

![Diagram 31](assets/diagrams/diagram-31.png)

### 시연 영상 구성 (3분)

![Diagram 32](assets/diagrams/diagram-32.png)

---

## SAGE의 차별성

### 1. 기술 비교 매트릭스

| 비교 항목 | HTTP | HTTPS (TLS) | SAGE |
|---------|------|-------------|------|
| **전송 암호화** | ❌ 없음 | ✅ 구간별 | ✅ 종단간 |
| **메시지 무결성** | ❌ 없음 | ⚠️  구간별 | ✅ 종단간 |
| **메시지 서명** | ❌ 없음 | ❌ 없음 | ✅ RFC-9421 |
| **종단간 암호화** | ❌ 없음 | ❌ 없음 | ✅ RFC-9180 |
| **신원 검증** | ❌ 없음 | ⚠️  인증서 | ✅ 블록체인 DID |
| **변조 탐지** | ❌ 불가능 | ⚠️  구간별만 | ✅ 즉시 탐지 |
| **투명성** | N/A | ⚠️  CA 신뢰 | ✅ 블록체인 |
| **표준 준수** | RFC-1945 | RFC-8446 | RFC-9421, 9180, DID |

### 2. 보안 레벨 비교

![Diagram 33](assets/diagrams/diagram-33.png)

### 3. 경쟁 우위

![Diagram 34](assets/diagrams/diagram-34.png)

---

## Appendix

### A. 사용된 오픈소스 목록

#### A.1 핵심 기술 스택

![Diagram 35](assets/diagrams/diagram-35.png)

#### A.2 오픈소스 라이선스 분포

![Diagram 36](assets/diagrams/diagram-36.png)

### B. 참조 표준

![Diagram 37](assets/diagrams/diagram-37.png)

### C. 프로젝트 통계

#### C.1 코드 통계

![Diagram 38](assets/diagrams/diagram-38.png)

#### C.2 검증 완료 현황

![Diagram 39](assets/diagrams/diagram-39.png)

---

## 결론

### SAGE의 비전

![Diagram 40](assets/diagrams/diagram-40.png)

### 핵심 메시지

> **"AI Agent 시대, 보안은 선택이 아닌 필수입니다"**
>
> HTTP에 HTTPS가 필요했듯이, AI Agent에는 SAGE가 필요합니다.
>
> **SAGE와 함께, 안전한 AI Agent 시대를 만들어갑니다.**

---

## 📞 연락처

- **GitHub**: https://github.com/sage-x-project/sage
- **문의**: (추후 추가)
- **데모**: (시연용 도메인 추가 예정)

---

*본 프로젝트는 2025 오픈소스 개발자대회 출품작입니다.*
*과학기술정보통신부 · 정보통신산업진흥원 주최*
