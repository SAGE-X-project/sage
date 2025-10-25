# SAGE (Secure Agent Guarantee Engine)
## 2025 오픈소스 개발자대회 최종 발표자료

---

## 프로젝트 개요

### "AI Agent 시대, 보안은 선택이 아닌 필수입니다"

SAGE는 **AI Agent 통신을 위한 Trust Layer**입니다.
HTTP에 HTTPS가 필요했듯이, AI Agent 시대에는 SAGE가 필요합니다.

- **문제**: AI Agent 간 통신의 보안 취약점 (중간자 공격, 메시지 변조, 신원 위조)
- **솔루션**: RFC 표준 기반 메시지 서명/암호화 + 블록체인 기반 투명한 Agent 검증
- **임팩트**: 개인정보 유출 방지, 금융 자산 보호, 신뢰할 수 있는 Agent 생태계 구축

---

## 팀 소개

**SAGE-X Project Team**

(추후 팀원 정보 추가)

---

## 개발 배경: 다가오는 위기

### 1. AI Agent 시대의 도래

**대 Agent 시대가 시작되었습니다:**

- **누구나 Agent를 만드는 시대**
  - OpenAI, Google Gemini, Claude 등 주요 AI 기업이 Agent Builder 제공
  - n8n, make 등 노코드 Agent 빌더 활성화
  - 기업들이 업무 자동화를 위해 Agent 도입 급증

- **Agent를 통한 금융 거래 시대**
  - Google의 AP2 프로젝트: Agent를 통한 Stable Coin 결제 시스템
  - AI Agent가 직접 상품 구매, 송금, 결제를 처리
  - "Appless 시대": 스마트폰의 앱이 AI 시대의 Agent로 대체

- **Agent에게 권한을 부여하는 시대**
  - MCP(Model Context Protocol)로 Agent에 도구 연결
  - 사용자가 Agent에게 개인정보, 금융정보, 파일 접근 권한 부여
  - Multi-Agent, Sub-Agent로 확장 (A2A 기술)

### 2. 현재 보안의 심각한 한계

**학계가 경고하는 Agent 보안 위험:**

최근 발표된 학술 논문들이 AI Agent의 보안 위험성을 경고하고 있습니다:

1. **"A Survey of LLM-Driven AI Agent Communication: Protocols, Security Risks, and Defense Countermeasures"**
   - AI Agent 통신 프로토콜의 보안 취약점 분석
   - 현재 Agent들이 충분한 보안 메커니즘 없이 통신 중
   - 체계적인 방어 대책 필요성 강조

2. **"The Dark Side of LLMs: Agent-based Attacks for Complete Computer Takeover"**
   - AI Agent를 통한 시스템 장악 공격 가능성 실증
   - Agent의 권한 남용 위험성 경고
   - 보안 대책 없이는 심각한 피해 발생 가능

**기술적 한계:**

- **TLS/HTTPS의 근본적 한계**
  - TLS는 구간별 암호화 (Hop-by-Hop Encryption)
  - 중간 서버(Gateway, Proxy)에서 복호화 후 재암호화
  - **종단간(End-to-End) 메시지 무결성을 보장하지 못함**
  - 중간자 공격(MITM) 가능 지점 존재

- **Agent Card만으로는 불충분**
  - A2A(Agent-to-Agent) 기술의 Agent Card는 메타데이터만 제공
  - Agent가 악의적인지, 신뢰할 수 있는지 판단 불가능
  - 공개키 검증 메커니즘 부재

- **OAuth2의 한계**
  - 사용자 인증에는 유용하나 메시지 무결성 보장 안 함
  - Agent 간 통신의 신뢰성 검증 불가능

### 3. 예상되는 피해 규모

**개인정보 유출:**
- Agent에게 전달되는 모든 대화 내용 (개인 상담, 의료 정보, 업무 기밀)
- 사용자 행동 패턴, 선호도, 금융 정보
- **피해 사례**: SKT 해킹 사태 (1,200만 고객 정보 유출, 사회적 비용 수천억 원)

**금융 자산 탈취:**
- Agent를 통한 결제 시 메시지 조작
- 송금 정보 변조 (수신자, 금액)
- Stable Coin, 암호화폐 거래 위변조

**기업 피해:**
- 업무 자동화 Agent를 통한 기밀 유출
- 내부 시스템 무단 접근
- 공급망 공격(Supply Chain Attack)

**사회적 비용:**
- 보안 사고 후 뒷수습 비용 (법적 대응, 배상, 시스템 복구)
- 신뢰 상실로 인한 산업 위축
- 국가 경쟁력 저하

### 4. 왜 지금 시작해야 하는가?

**역사는 반복됩니다:**

| 기술 시대 | 초기 상황 | 보안 사고 | 사후 대응 | 교훈 |
|---------|---------|---------|---------|------|
| **초기 인터넷** | HTTP (평문 통신) | 개인정보 유출, 세션 하이재킹 | HTTPS 도입 (1994→2010년대 전면 적용) | 보안 사고 후 10년+ 소요 |
| **모바일 앱** | 보안 미비 앱 난립 | 개인정보 무단 수집, 악성 앱 | 앱 심사, 권한 관리 강화 | 수많은 피해 후 규제 |
| **AI Agent** | TLS/OAuth만 사용 | **← 지금 이 시점** | ??? | **선제적 대응 필요!** |

**선제적 대응의 가치:**
- 보안 사고가 발생하기 전에 해결
- 사회적 비용 절감 (사고 대응 비용 >> 사전 예방 비용)
- 안전한 AI Agent 생태계 조성
- 국가 차원의 기술 경쟁력 확보

---

## 🎯 개발 목적

### "Trust Layer를 통한 안전한 AI Agent 생태계 구축"

### 1. 핵심 목표

**AI Agent 통신에 Trust Layer 제공**
- HTTP → HTTPS 전환처럼, Agent → Secure Agent로 진화 필요
- 표준 기반 접근으로 호환성 및 확장성 보장
- 오픈소스로 누구나 사용 가능한 보안 솔루션 제공

### 2. 기술적 목표

**종단간(End-to-End) 메시지 무결성 보장**
- RFC-9421 기반 HTTP 메시지 서명
- 중간 서버의 변조 시도를 수신자가 탐지 가능
- 서명 검증을 통한 메시지 출처 및 무결성 보증

**메시지 기밀성 보장**
- RFC-9180 (HPKE) 기반 종단간 암호화
- 중간 서버가 메시지 내용을 볼 수 없도록 보호
- Handshake 기반 세션 암호화 통신

**투명한 Agent 신원 검증**
- 블록체인 기반 DID(Decentralized Identifier) 관리
- Agent Card + 공개키를 블록체인에 등록
- 누구나 Agent의 신원과 공개키를 투명하게 검증 가능
- 조작 불가능한 Agent 정보 제공

### 3. 사회적 목표

**개인정보 유출 방지**
- Agent 간 통신 메시지 암호화 및 서명
- 중간자 공격 차단

**금융 자산 보호**
- 결제 메시지 변조 방지
- 송금 정보 무결성 보장

**신뢰할 수 있는 Agent 생태계**
- 악의적인 Agent와 정상 Agent 구분
- 블록체인 기반 투명한 Agent 등록부
- 사용자가 안심하고 Agent를 사용할 수 있는 환경

**국가 기술 경쟁력 향상**
- AI Agent 보안 분야 선도
- 글로벌 표준 기반 기술 확보
- 오픈소스로 국제 커뮤니티 기여

---

## 🏗️ 프로젝트 구성 및 기능

### 1. 아키텍처 개요

**SAGE는 Layer 기반 Trust Layer입니다:**

```
┌─────────────────────────────────────────────────────┐
│           Application Layer (AI Agent)              │
│              (ChatGPT, Gemini, Claude, etc.)        │
├─────────────────────────────────────────────────────┤
│         ┌─────────────────────────────┐             │
│         │    SAGE Trust Layer         │             │
│         │  - Message Signing/Verify   │             │
│         │  - Message En(De)cryption   │             │
│         │  - DID Verification         │             │
│         └─────────────────────────────┘             │
├─────────────────────────────────────────────────────┤
│         Transport Layer (HTTP/HTTPS)                │
└─────────────────────────────────────────────────────┘
```

**설계 원칙:**
- 기존 Agent 코드 최소 수정으로 통합 가능
- 플러그인 기반 확장성 (암호 알고리즘, 블록체인)
- 표준 준수 (RFC-9421, RFC-9180, DID Core)

### 2. 핵심 기능

#### 2.1 메시지 서명 (RFC-9421)

**HTTP Message Signatures 표준 구현:**

```
┌─────────────┐                              ┌─────────────┐
│  Agent A    │                              │  Agent B    │
│             │                              │             │
│  1. Sign    │                              │             │
│  Message    │    HTTP Request + Signature  │             │
│             ├─────────────────────────────>│  2. Verify  │
│             │    Signature-Input Header    │  Signature  │
│             │    Signature Header          │             │
│             │                              │  3. ✓ or ✗  │
└─────────────┘                              └─────────────┘
```

**구현 내용:**
- Signature-Input 헤더 생성 (created, expires, nonce)
- Signature 헤더 생성 (base64 인코딩)
- 서명 검증 (변조 탐지, 만료 확인, Nonce 중복 확인)
- 타임스탬프 유효성 검증 (±5분 허용)

**효과:**
- 메시지 변조 시도 즉시 탐지
- 중간 Gateway, Proxy에서의 메시지 조작 방지
- 메시지 출처 검증 (Non-repudiation)

#### 2.2 메시지 암호화 (RFC-9180)

**HPKE (Hybrid Public Key Encryption) 구현:**

```
┌─────────────┐                              ┌─────────────┐
│  Agent A    │                              │  Agent B    │
│             │    1. Handshake Request      │             │
│             ├─────────────────────────────>│             │
│             │    (Ephemeral Public Key)    │             │
│             │                              │             │
│             │    2. Handshake Response     │             │
│             │<─────────────────────────────┤             │
│             │    (Encrypted Session Key)   │             │
│             │                              │             │
│  3. Encrypt │    4. Encrypted Message      │  5. Decrypt │
│             ├─────────────────────────────>│             │
│             │    (ChaCha20Poly1305)        │             │
└─────────────┘                              └─────────────┘
```

**구현 내용:**
- DHKEM (X25519) 키 교환
- AEAD 암호화 (ChaCha20Poly1305)
- 세션 관리 (Session ID, Nonce, TTL)
- Exporter를 통한 추가 키 유도

**효과:**
- 중간 서버가 메시지 내용 확인 불가
- 개인정보, 금융정보 안전하게 전송
- Forward Secrecy 보장

#### 2.3 DID 관리

**블록체인 기반 분산 신원 인증:**

```
┌─────────────────────────────────────────────────────┐
│              Blockchain (Ethereum)                  │
│                                                     │
│  DID Document:                                      │
│  {                                                  │
│    "id": "did:sage:ethereum:0x123...",              │
│    "publicKey": [{                                  │
│      "id": "did:sage:ethereum:0x123...#keys-1",     │
│      "type": "Secp256k1VerificationKey2019",        │
│      "publicKeyHex": "04ab3f..."                    │
│    }],                                              │
│    "service": [{                                    │
│      "type": "AgentService",                        │
│      "serviceEndpoint": "https://agent.example.com" │
│    }],                                              │
│    "metadata": {                                    │
│      "name": "Payment Agent",                       │
│      "description": "...",                          │
│      "version": "1.0.0"                             │
│    }                                                │
│  }                                                  │
└─────────────────────────────────────────────────────┘
```

**기능:**
- DID 생성 및 등록 (did:sage:ethereum: 형식)
- 공개키 블록체인에 등록 (조작 불가)
- Agent 메타데이터 관리 (이름, 설명, 엔드포인트)
- DID 조회 및 검증
- DID 업데이트 및 비활성화

**효과:**
- 투명한 Agent 정보 공개
- 공개키 위변조 방지
- 신뢰할 수 있는 Agent 식별
- 악의적 Agent 차단 가능

#### 2.4 암호화 키 관리

**다양한 암호 알고리즘 지원:**

| 알고리즘 | 용도 | 키 길이 | 특징 |
|---------|------|---------|------|
| **Secp256k1** | ECDSA 서명 | 32 bytes (private), 33/65 bytes (public) | Ethereum 호환 |
| **Ed25519** | EdDSA 서명 | 32 bytes (private), 32 bytes (public) | 고속, 안전 |
| **X25519** | ECDH 키 교환 | 32 bytes | HPKE 사용 |
| **ChaCha20Poly1305** | AEAD 암호화 | - | 고속, 안전 |

**키 저장:**
- PEM 형식 파일 저장 (파일 권한 0600)
- Vault 암호화 저장 (암호화된 키 저장소)

### 3. 시스템 구성

```
┌────────────────────────────────────────────────────────┐
│                    SAGE Core                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  RFC-9421    │  │  RFC-9180    │  │  DID Core    │  │
│  │  (Signing)   │  │  (HPKE)      │  │  (Identity)  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  Crypto      │  │  Session     │  │  Storage     │  │
│  │  Management  │  │  Management  │  │  (Vault)     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
┌───────▼───────┐ ┌───────▼───────┐ ┌───────▼───────┐
│  Blockchain   │ │  Multi-Lang   │ │  CLI Tools    │
│  Integration  │ │  SDKs         │ │               │
│               │ │               │ │               │
│  - Ethereum   │ │  - Go         │ │  - sage-crypto│
│  - (확장 가능)  │ │  - Python     │ │  - sage-did   │
│               │ │  - TypeScript │ │  - sage-verify│
│               │ │  - Java       │ │               │
└───────────────┘ └───────────────┘ └───────────────┘
```

### 4. 확장성 및 플러그인 아키텍처

**암호 알고리즘 플러그인:**
- 새로운 암호 알고리즘 추가 가능
- 인터페이스 기반 설계
- 레거시 알고리즘 교체 용이

**블록체인 플러그인:**
- Ethereum 외 다른 블록체인 지원 가능
- 체인별 Adapter 패턴 적용
- 멀티체인 DID 지원 준비

**SDK 다양성:**
- Go, Python, TypeScript, Java SDK 제공
- 각 언어별 관용적 API 설계
- 쉬운 통합 (5줄 이내 코드로 적용 가능)

### 5. 검증 및 품질 보증

**100% 명세서 검증 완료:**

| 섹션 | 항목 수 | 검증 상태 |
|------|---------|----------|
| 1. RFC-9421 구현 | 11 | ✅ 100% |
| 2. 암호화 키 관리 | 13 | ✅ 100% |
| 3. DID 관리 | 12 | ✅ 100% |
| 4. 블록체인 연동 | 10 | ✅ 100% |
| 5. 메시지 처리 | 10 | ✅ 100% |
| 6. CLI 도구 | 13 | ✅ 100% |
| 7. 세션 관리 | 6 | ✅ 100% |
| 8. HPKE | 5 | ✅ 100% |
| 9. 헬스체크 | 3 | ✅ 100% |
| **총계** | **83** | **✅ 100%** |

**테스트 현황:**
- 총 61개 테스트 함수
- 19개 테스트 파일 (7,848 라인)
- 단위 테스트, 통합 테스트, 명세 검증 테스트
- 지속적 통합(CI) 자동화

**문서화:**
- 5,128 라인 검증 매트릭스 문서
- 섹션별 상세 검증 문서 (9개 파일)
- 테스트 실행 명령어, 예상 결과, 검증 방법 포함

---

## 🌟 추후 활용 방안 및 계획

### 1. 단기 계획 (6개월)

#### 1.1 rs-sage-core (Rust Core Library)
**목적**: 고성능 코어 라이브러리 제공

- Rust로 Core 기능 재구현
- WebAssembly 지원 (브라우저에서 실행 가능)
- 성능 최적화 (Go 대비 2-3배 향상 목표)
- 다양한 플랫폼 지원 (iOS, Android 포함)

**활용 방안:**
- 모바일 Agent 앱 개발
- 브라우저 확장 프로그램
- 엣지 디바이스 (IoT Agent)

#### 1.2 다중 언어 SDK 완성도 향상

**현재 상태:**
- Go SDK: ✅ 완료
- Python SDK: 🔄 진행 중
- TypeScript SDK: 🔄 진행 중
- Java SDK: 📋 계획

**개선 사항:**
- 각 언어별 예제 코드 및 튜토리얼
- 패키지 매니저 등록 (PyPI, npm, Maven)
- 상세 API 문서 (Sphinx, JSDoc, Javadoc)

### 2. 중기 계획 (1년)

#### 2.1 SAGE-ADK (Agent Development Kit)
**목적**: 보안이 강화된 Agent를 쉽게 개발할 수 있는 통합 개발 도구

**기능:**
- SAGE + A2A(Agent-to-Agent) 통합
- Agent Builder UI (드래그 앤 드롭 방식)
- 템플릿 기반 Agent 생성 (결제, 예약, 상담 등)
- 자동 SAGE 통합 (보안 코드 자동 생성)
- 로컬 테스트 환경 제공

**기대 효과:**
- 개발자가 보안 걱정 없이 Agent 개발에 집중
- Agent 개발 시간 단축 (몇 주 → 며칠)
- 안전한 Agent 생태계 조성

#### 2.2 Agent Dashboard (Agent Marketplace)
**목적**: 신뢰할 수 있는 Agent 정보를 제공하는 투명한 마켓플레이스

**기능:**
- 블록체인에 등록된 모든 Agent 목록 표시
- Agent 정보 실시간 조회 (DID, 공개키, 메타데이터)
- Agent 평점 및 리뷰 시스템
- 보안 검증 배지 (SAGE 인증 Agent)
- Agent 검색 및 필터링
- Agent 사용 가이드

**운영:**
- 24/7 실시간 블록체인 동기화
- 웹 기반 (https://dashboard.sage.dev)
- API 제공 (다른 서비스 통합 가능)

**기대 효과:**
- 사용자가 안전한 Agent를 쉽게 찾을 수 있음
- 악의적인 Agent 식별 및 회피
- Agent 개발자에게 신뢰성 증명 제공

#### 2.3 MCP (Model Context Protocol) Integration
**목적**: MCP 생태계에 SAGE 보안 적용

**배경:**
- MCP는 AI 모델에 도구를 연결하는 표준 프로토콜
- 최근 MCP에서도 Agent 연동 기능 추가
- MCP 도구도 보안 위협 대상

**통합 방안:**
- MCP Server에 SAGE Trust Layer 적용
- MCP 도구 호출 시 메시지 서명 검증
- MCP 서버 DID 등록 및 검증
- SAGE-MCP SDK 제공

**기대 효과:**
- MCP 생태계에 보안 강화
- ChatGPT, Claude, Gemini에서 안전한 도구 사용

### 3. 장기 계획 (2-3년)

#### 3.1 글로벌 표준화 활동

**IETF (Internet Engineering Task Force) 참여:**
- RFC-9421, RFC-9180 확장 제안
- AI Agent 보안 표준 Working Group 참여
- "RFC: Secure AI Agent Communication" 제안

**W3C (World Wide Web Consortium) 참여:**
- DID Core 확장 (Agent 특화 DID Method)
- Verifiable Credentials for AI Agents 제안

**기대 효과:**
- 국제 표준에 한국 기술 반영
- 글로벌 커뮤니티 주도권 확보

#### 3.2 주요 AI 플랫폼 통합

**목표 플랫폼:**
- OpenAI GPT (ChatGPT, Custom GPTs)
- Google Gemini (Gemini Agent)
- Anthropic Claude (Claude Agent)
- Microsoft Copilot
- 국내: Naver HyperCLOVA X, Kakao KoGPT

**통합 방안:**
- 각 플랫폼별 Easy Integration 라이브러리
- 플랫폼 네이티브 기능으로 통합 (Plugin, Extension)
- 공식 파트너십 체결

**기대 효과:**
- 수억 명의 사용자에게 안전한 Agent 제공
- SAGE가 사실상의 표준(De facto Standard)으로 자리잡음

#### 3.3 산업별 특화 솔루션

**금융 산업:**
- 금융 거래 Agent 보안 강화
- 금융위원회, 금융보안원과 협력
- ISO 27001, PCI-DSS 준수 검증

**의료 산업:**
- 의료 상담 Agent 개인정보 보호
- HIPAA (미국), 개인정보보호법 준수
- 환자 데이터 암호화 및 무결성 보장

**공공 산업:**
- 정부 서비스 Agent 보안
- 행정안전부, 국가정보원과 협력
- 국가 사이버 보안 체계 통합

#### 3.4 학술 및 교육

**대학 교육 과정:**
- AI Agent 보안 교재 집필
- 대학 강의 커리큘럼 제공
- 학생 프로젝트 지원

**보안 연구:**
- Agent 보안 취약점 연구
- 새로운 공격 기법 및 방어 기법 연구
- 학술 논문 발표 (국제 학회)

**커뮤니티 활동:**
- 정기 세미나 및 컨퍼런스 개최
- 오픈소스 기여자 양성
- Bug Bounty 프로그램 운영

### 4. 오픈소스 생태계 기여

#### 4.1 커뮤니티 구축

**GitHub 생태계:**
- 이슈 트래킹 및 Feature Request 수렴
- Pull Request 환영 및 기여자 가이드 제공
- 월간 릴리스 및 Changelog 관리

**개발자 커뮤니티:**
- Discord/Slack 채널 운영
- Stack Overflow 태그 생성
- 정기 온라인 밋업 (월 1회)

**문서화:**
- 다국어 문서 (한국어, 영어, 일본어, 중국어)
- 튜토리얼 영상 (YouTube)
- 블로그 포스트 (Medium, Dev.to)

#### 4.2 라이선스 전략

**현재 라이선스:**
- Go 코드: LGPL-v3 (라이브러리로 사용 가능)
- Smart Contract: MIT (자유로운 배포)

**목적:**
- 상업적 사용 허용 (기업이 SAGE를 제품에 통합 가능)
- 오픈소스 기여 장려
- Fork 및 개선 환영

#### 4.3 지속 가능성

**재정 모델:**
- 기업 후원 (Enterprise Support)
- 공공 기관 프로젝트 (정부 R&D)
- 교육 및 컨설팅 서비스

**거버넌스:**
- 오픈 거버넌스 모델
- Technical Steering Committee 구성
- 투명한 의사 결정

---

## 🎬 시연 계획

### 1. 시연 시나리오 개요

**목표**: "SAGE가 없으면 위험하고, SAGE가 있으면 안전하다"를 명확히 보여주기

**비교 시연:**
1. **보안 미적용 Agent** (취약점 노출)
2. **SAGE 적용 Agent** (공격 방어 성공)

### 2. 시연 환경 구성

#### 2.1 인프라

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   Frontend   │      │   Gateway    │      │ Payment Agent│
│  (Chat UI)   │─────>│  (Infected)  │─────>│  (AP2)       │
│              │      │   MitM       │      │              │
└──────────────┘      └──────────────┘      └──────────────┘
                              │
                              v
                      ┌──────────────┐
                      │  Blockchain  │
                      │  (Sepolia)   │
                      └──────────────┘
```

**배포:**
- Frontend: Vercel (https://sage-demo.vercel.app)
- Gateway: Supabase/AWS
- Agents: Supabase/AWS
- Blockchain: Ethereum Sepolia Testnet
- RPC: Alchemy API

**도메인:**
- 시연용 도메인 구매 (예: sage-demo.com)
- SSL 인증서 적용

#### 2.2 컨트랙트 배포

**Ethereum Sepolia Testnet:**
- AgentRegistry 컨트랙트 배포
- 테스트 Agent DID 등록
- 블록체인 탐색기에서 확인 가능 (Etherscan)

### 3. 시연 시나리오

#### 시나리오 1: 보안 미적용 Agent의 위험성

**상황**: 사용자가 일반 Agent를 통해 상품 구매

**절차:**
1. 사용자: "iPhone 15 Pro를 구매해줘"
2. Agent: 결제 정보 요청
3. 사용자: 결제 정보 입력
4. **Gateway가 메시지 변조** (수신자 주소 변경)
5. Payment Agent: 변조된 정보로 결제 처리
6. **해커에게 송금 완료**

**결과 화면:**
```
[Frontend]
사용자: iPhone 15 Pro 구매 요청
Agent: 결제 정보를 입력해주세요.
사용자: [결제 정보 입력]

[Gateway - MitM Attack Log]
⚠️  메시지 변조 중...
원본: { to: "Apple Store", amount: 1500000 }
변조: { to: "Hacker Wallet", amount: 1500000 }

[Payment Agent]
✅ 결제 완료: 1,500,000원
수신자: Hacker Wallet (0x...)

❌ 자산 탈취 발생!
```

**강조점:**
- TLS/HTTPS만으로는 Gateway의 변조를 막을 수 없음
- 메시지 무결성 검증 없이는 위험

#### 시나리오 2: SAGE (RFC-9421) - 메시지 서명 검증

**상황**: SAGE 적용 Agent 사용 (메시지 서명)

**절차:**
1. 사용자: "iPhone 15 Pro를 구매해줘"
2. Agent: 결제 정보 요청 (메시지 서명)
3. 사용자: 결제 정보 입력
4. **Gateway가 메시지 변조 시도**
5. Payment Agent: **서명 검증 실패** → 거래 거부
6. **공격 차단 성공**

**결과 화면:**
```
[Frontend]
사용자: iPhone 15 Pro 구매 요청
Agent: 결제 정보를 입력해주세요. [SAGE 서명 생성]
사용자: [결제 정보 입력]

[Gateway - MitM Attack Attempt]
⚠️  메시지 변조 시도...
원본: { to: "Apple Store", amount: 1500000 }
변조: { to: "Hacker Wallet", amount: 1500000 }
서명: [변경 없음] ← 공격자는 서명을 재생성할 수 없음

[Payment Agent - SAGE 검증]
🔍 메시지 서명 검증 중...
원본 메시지: { to: "Apple Store", amount: 1500000 }
서명 대상: { to: "Hacker Wallet", amount: 1500000 }
결과: ❌ 서명 불일치!

⛔ 거래 거부: 메시지 무결성 위반 감지
✅ 공격 차단 성공!
```

**강조점:**
- RFC-9421 메시지 서명으로 변조 탐지
- 수신자가 직접 검증하므로 안전

#### 시나리오 3: SAGE (RFC-9180) - 메시지 암호화

**상황**: 민감한 개인정보 전송 (의료 상담)

**절차:**
1. 사용자: "건강 상담 Agent에 연결"
2. 사용자: "최근 당뇨병 진단을 받았는데..." (개인 의료 정보)
3. **SAGE HPKE 암호화**
4. Gateway: 암호화된 메시지 확인 불가
5. 상담 Agent: 복호화 후 정상 응답

**결과 화면:**
```
[Frontend]
사용자: 건강 상담 Agent 연결 요청
시스템: SAGE HPKE 암호화 세션 시작...
        Handshake 완료 ✓

사용자: [개인 의료 정보 입력]
        "최근 당뇨병 진단을 받았는데,
         혈당 수치가 180mg/dL입니다..."

[Gateway - 중간 서버]
📦 암호화된 메시지 수신:
   Ciphertext: a9f3e2b1c4d5... (읽을 수 없음)
   → Gateway는 내용을 볼 수 없음 ✓

[상담 Agent - SAGE 복호화]
🔓 메시지 복호화 성공
   "최근 당뇨병 진단을 받았는데,
    혈당 수치가 180mg/dL입니다..."

Agent: 당뇨병 관리에 대해 안내드리겠습니다...

✅ 개인정보 보호 성공!
```

**강조점:**
- RFC-9180 HPKE로 종단간 암호화
- 중간 서버가 내용을 볼 수 없음
- 개인정보 유출 방지

#### 시나리오 4: SAGE DID - Agent 신원 검증

**상황**: 사용자가 결제 Agent 선택

**절차:**
1. 사용자: Agent 목록 조회
2. 시스템: 블록체인에서 Agent DID 조회
3. **정상 Agent vs 미등록 Agent 비교**
4. 사용자: 신뢰할 수 있는 Agent 선택

**결과 화면:**
```
[Agent Dashboard]

┌────────────────────────────────────────────────┐
│ Agent 목록                                      │
├────────────────────────────────────────────────┤
│ 1. Official Payment Agent           ✅ 인증됨    │
│    DID: did:sage:ethereum:0xabc...             │
│    등록일: 2025-01-15                            │
│    공개키: 04f3e2a1... (블록체인 검증 완료)          │
│    평점: ★★★★★ (1,234 reviews)                  │
│    [선택]                                       │
├────────────────────────────────────────────────┤
│ 2. Fake Payment Agent               ⚠️  미인증  │
│    DID: (등록되지 않음)                           │
│    경고: 블록체인에 등록되지 않은 Agent입니다.          │
│    [사용 불가]                                   │
└────────────────────────────────────────────────┘

[블록체인 조회 - Etherscan]
Contract: AgentRegistry (0x123...)
Function: getAgent(did:sage:ethereum:0xabc...)
Returns:
  - publicKey: 04f3e2a1...
  - metadata: {"name": "Official Payment Agent", ...}
  - status: active
  - registeredAt: 1705334400

✅ 신뢰할 수 있는 Agent 식별 성공!
```

**강조점:**
- 블록체인 기반 투명한 Agent 검증
- 악의적인 Agent 사전 차단
- 사용자가 안심하고 선택 가능

### 4. 시연 영상 구성 (3분 이내)

**타임라인:**
- 00:00-00:30: 문제 제기 (Agent 보안 위험성)
- 00:30-01:00: 시나리오 1 (보안 미적용 - 공격 성공)
- 01:00-01:40: 시나리오 2 (SAGE 서명 - 공격 차단)
- 01:40-02:10: 시나리오 3 (SAGE 암호화 - 개인정보 보호)
- 02:10-02:40: 시나리오 4 (SAGE DID - Agent 검증)
- 02:40-03:00: 마무리 (SAGE의 가치 및 오픈소스 기여)

**촬영 스타일:**
- 화면 녹화 + 나레이션
- 깔끔한 UI/UX (프로페셔널한 인상)
- 공격 차단 시점 강조 (빨간색 → 초록색 전환)
- 블록체인 탐색기 실제 화면 포함 (신뢰성)

### 5. 현장 시연 준비

**백업 계획:**
- 인터넷 연결 실패 대비: 로컬 환경 준비
- 블록체인 지연 대비: 사전 트랜잭션 준비
- 시연 영상 대기 (현장 시연 실패 시)

**관객 참여:**
- QR 코드로 시연 사이트 접속 가능
- 직접 메시지 전송 체험
- 블록체인 탐색기에서 실시간 확인

---

## Appendix: 사용된 오픈소스 목록

### 1. 핵심 기술 스택

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **Go** | SAGE 핵심 구현 언어 | BSD-3-Clause | https://golang.org/ |
| **Ethereum** | 블록체인 플랫폼 (DID 등록) | LGPL-3.0 | https://ethereum.org/ |
| **Solidity** | 스마트 컨트랙트 언어 | GPL-3.0 | https://soliditylang.org/ |

### 2. 암호화 라이브러리

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **crypto/ecdsa** | Secp256k1 ECDSA 서명 | BSD-3-Clause (Go stdlib) | https://pkg.go.dev/crypto/ecdsa |
| **crypto/ed25519** | Ed25519 EdDSA 서명 | BSD-3-Clause (Go stdlib) | https://pkg.go.dev/crypto/ed25519 |
| **golang.org/x/crypto** | X25519 키 교환, ChaCha20Poly1305 | BSD-3-Clause | https://pkg.go.dev/golang.org/x/crypto |
| **github.com/cloudflare/circl** | HPKE 구현 | BSD-3-Clause | https://github.com/cloudflare/circl |

### 3. 블록체인 관련

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **go-ethereum (geth)** | Ethereum 클라이언트 라이브러리 | LGPL-3.0 | https://github.com/ethereum/go-ethereum |
| **ethclient** | Ethereum RPC 클라이언트 | LGPL-3.0 | https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient |
| **Hardhat** | 스마트 컨트랙트 개발 도구 | MIT | https://hardhat.org/ |
| **OpenZeppelin** | 스마트 컨트랙트 라이브러리 | MIT | https://openzeppelin.com/ |

### 4. DID 관련

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **DID Core Spec** | DID 표준 규격 참조 | W3C | https://www.w3.org/TR/did-core/ |
| **did:ethr** | Ethereum DID Method 참조 | Apache-2.0 | https://github.com/decentralized-identity/ethr-did-resolver |

### 5. HTTP 및 웹 프레임워크

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **net/http** | HTTP 서버/클라이언트 | BSD-3-Clause (Go stdlib) | https://pkg.go.dev/net/http |
| **github.com/gorilla/mux** | HTTP 라우팅 | BSD-3-Clause | https://github.com/gorilla/mux |

### 6. 테스트 및 개발 도구

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **testing** | Go 테스트 프레임워크 | BSD-3-Clause (Go stdlib) | https://pkg.go.dev/testing |
| **github.com/stretchr/testify** | 테스트 어설션 라이브러리 | MIT | https://github.com/stretchr/testify |
| **golangci-lint** | Go 린터 | GPL-3.0 | https://golangci-lint.run/ |

### 7. 유틸리티

| 오픈소스 이름 | 활용 내용 | 라이선스 | 링크 |
|------------|---------|---------|------|
| **github.com/spf13/cobra** | CLI 도구 프레임워크 | Apache-2.0 | https://github.com/spf13/cobra |
| **github.com/spf13/viper** | 설정 관리 | MIT | https://github.com/spf13/viper |
| **gopkg.in/yaml.v3** | YAML 파싱 | MIT | https://github.com/go-yaml/yaml |

### 8. 참조 표준 (RFC)

| 표준 이름 | 활용 내용 | 출처 | 링크 |
|---------|---------|------|------|
| **RFC-9421** | HTTP Message Signatures | IETF | https://datatracker.ietf.org/doc/rfc9421/ |
| **RFC-9180** | HPKE (Hybrid Public Key Encryption) | IETF | https://datatracker.ietf.org/doc/rfc9180/ |
| **RFC-8032** | EdDSA (Ed25519) | IETF | https://datatracker.ietf.org/doc/rfc8032/ |
| **RFC-5869** | HKDF (HMAC-based Key Derivation) | IETF | https://datatracker.ietf.org/doc/rfc5869/ |

### 9. 학술 자료 (배경 연구)

| 논문 제목 | 활용 내용 | 출처 |
|---------|---------|------|
| **A Survey of LLM-Driven AI Agent Communication** | Agent 통신 보안 위험성 분석 | arXiv:2506.19676v3 |
| **The Dark Side of LLMs** | Agent 기반 공격 기법 연구 | arXiv:2507.06850v3 |

---

## SAGE의 차별성 및 우수성

### 1. 기술적 우수성

**표준 기반 접근:**
- RFC-9421, RFC-9180 등 국제 표준 준수
- 검증된 암호화 알고리즘 사용 (Secp256k1, Ed25519, ChaCha20Poly1305)
- 블록체인 기술 활용 (투명성, 불변성)

**아키텍처 우수성:**
- Layer 기반 설계 (기존 코드 최소 수정)
- 플러그인 아키텍처 (확장 가능)
- 다양한 언어 SDK 제공

**검증 완료:**
- 100% 명세서 검증 (83개 항목)
- 철저한 테스트 (61개 테스트 함수)
- 상세한 문서화 (5,128 라인)

### 2. 실용성

**즉시 사용 가능:**
- 라이브러리 형태 제공 (간단한 통합)
- CLI 도구 제공 (개발자 친화적)
- 예제 코드 및 튜토리얼 제공

**실제 시연 가능:**
- 완전히 작동하는 시스템
- 블록체인 실 배포
- 실시간 공격 방어 시연

### 3. 사회적 가치

**선제적 보안 대응:**
- 보안 사고 발생 전 예방
- 사회적 비용 절감
- 안전한 AI Agent 생태계 조성

**오픈소스 기여:**
- 누구나 사용 가능 (LGPL-v3)
- 커뮤니티 기여 환영
- 국가 기술 경쟁력 향상

**교육 및 인식:**
- Agent 보안의 중요성 알림
- 개발자 교육 자료 제공
- 학술 연구 기여

### 4. 지속 가능성

**명확한 로드맵:**
- 단기, 중기, 장기 계획 수립
- 구체적인 실행 방안

**커뮤니티 생태계:**
- 개발자 커뮤니티 구축
- 기업 파트너십 추진
- 국제 표준화 활동

**확장 가능성:**
- 다양한 산업 적용 가능 (금융, 의료, 공공)
- 글로벌 시장 진출 가능
- 장기적 성장 가능성

---

## 결론

### SAGE는 단순한 프로젝트가 아닙니다

**문제 제기:**
- AI Agent 시대의 보안 위협은 실재하며 시급합니다
- 학술 논문들이 경고하고 있습니다
- 보안 사고 후 대응은 너무 늦습니다

**해결책:**
- SAGE는 표준 기반의 완벽한 솔루션입니다
- 종단간 무결성, 암호화, 투명한 검증 제공
- 100% 검증 완료된 실용적인 시스템

**사회적 기여:**
- 개인정보 유출 방지
- 금융 자산 보호
- 신뢰할 수 있는 AI Agent 생태계 구축
- 국가 기술 경쟁력 향상

**오픈소스 가치:**
- 누구나 사용 가능
- 커뮤니티 기여 환영
- 지속 가능한 생태계

### SAGE와 함께, 안전한 AI Agent 시대를 만들어갑니다

**"Trust Layer for AI Agent Era"**

---

*본 프로젝트는 2025 오픈소스 개발자대회 출품작입니다.*
*과학기술정보통신부 · 정보통신산업진흥원 주최*
