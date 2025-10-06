# EIP-8004 (ERC-8004): Trustless Agents 상세 분석 리포트

## 📋 개요

### 기본 정보
- **제목**: ERC-8004: Trustless Agents
- **상태**: Draft (Standards Track: ERC)
- **제안 시기**: 2025년 8월
- **저자**:
  - Marco De Rossi (MetaMask)
  - Davide Crapis (Ethereum Foundation)
  - Jordan Ellis (독립 AI 개발자)

### 핵심 목적
Google의 Agent-to-Agent (A2A) 프로토콜을 확장하여, **사전 신뢰 관계 없이도** 조직 간 경계를 넘어 AI 에이전트들이 서로를 발견하고, 선택하고, 상호작용할 수 있는 **신뢰 레이어(Trust Layer)**를 제공합니다.

---

## 🏗️ 아키텍처 구조

### 1. Agent-to-Agent (A2A) 프로토콜 (기반 기술)

#### A2A 프로토콜이란?
- **발표**: 2025년 4월 9일 Google 발표
- **목적**: 서로 다른 벤더의 AI 에이전트들이 협업하고 통신할 수 있는 개방형 프로토콜
- **지원사**: 50개 이상의 기술 및 서비스 파트너 (Google, Atlassian, PayPal, SAP, PwC 등)

#### 기술 사양
```
통신 프로토콜: JSON-RPC 2.0 over HTTP(S)
```

**핵심 기능:**
1. **Agent Card를 통한 능력 발견**
   - JSON 형식으로 에이전트의 능력, 연결 정보 포함

2. **작업 생명주기 관리**
   - 빠른 작업부터 여러 날에 걸친 복잡한 작업까지 지원

3. **상호작용 모드**
   - 동기식 요청/응답
   - 스트리밍 (Server-Sent Events)
   - 비동기 푸시 알림

4. **데이터 교환**
   - 텍스트, 파일, 구조화된 JSON 데이터 지원

**설계 원칙:**
- ✅ 기존 표준 활용 (HTTP, SSE, JSON-RPC)
- ✅ 기본적으로 보안 우선
- ✅ 장기 실행 작업 지원
- ✅ 모달리티 불가지론적 (Modality Agnostic)

---

### 2. ERC-8004의 확장: 3가지 온체인 레지스트리

ERC-8004는 A2A 프로토콜 위에 **신뢰 레이어**를 추가하며, 3개의 경량 온체인 레지스트리를 도입합니다:

#### 🔷 1) Identity Registry (신원 레지스트리)

**목적:** 에이전트에게 휴대 가능하고 검열 저항적인 고유 식별자 제공

**구성 요소:**
- `AgentID`: 고유 식별자
- `AgentDomain`: 도메인 정보
- `AgentAddress`: 온체인 주소

**특징:**
- 최소한의 온체인 데이터만 저장
- 오프체인 AgentCard로 해결(resolve)
- 체인 불가지론적(Chain-agnostic) 주소 지정 지원

**스마트 계약 인터페이스 예시:**
```solidity
interface IIdentityRegistry {
    function registerAgent(
        string calldata agentId,
        string calldata agentDomain,
        address agentAddress
    ) external returns (bool);

    function resolveAgent(string calldata agentId)
        external view returns (AgentMetadata memory);
}
```

---

#### 🔷 2) Reputation Registry (평판 레지스트리)

**목적:** 에이전트 간 작업 피드백을 활성화하고 평판 구축

**핵심 메커니즘:**
- **사전 승인(Pre-authorization) 피드백**: 악의적인 평판 조작 방지
- **최소 온체인 데이터**: 실제 평판 점수는 오프체인에서 집계
- **영구 감사 추적(Permanent Audit Trail)**

**작동 방식:**
1. 클라이언트 에이전트가 작업 완료 후 피드백 attestation 게시
2. Attestation에는 다음 포함:
   - DataHash (작업 결과의 해시)
   - 참여자 정보
   - ERC-8004 요청/응답 ID
3. 오프체인 서비스가 이를 집계하여 평판 점수 산출

**생태계 가능성:**
- 전문 에이전트 평가 서비스
- 감사자 네트워크
- 보험 풀

**인터페이스 예시:**
```solidity
interface IReputationRegistry {
    function submitFeedback(
        bytes32 taskId,
        address agentAddress,
        bytes32 dataHash,
        uint8 rating
    ) external;

    function getFeedback(address agentAddress)
        external view returns (Feedback[] memory);
}
```

---

#### 🔷 3) Validation Registry (검증 레지스트리)

**목적:** 작업이 올바르게 수행되었는지 독립적으로 검증

**검증 모델 (3가지):**

##### A. 평판 기반 시스템 (Reputation-based)
- 클라이언트 피드백 활용
- 낮은 위험 작업에 적합 (예: 피자 주문)

##### B. 스테이크 기반 추론 검증 (Stake-secured Inference Validation)
- **크립토-경제학적 보안**
- 검증자가 작업을 재실행
- 거짓 주장 시 경제적 패널티
- 중간 위험 작업에 적합

**검증 프로세스:**
```
1. 서버 에이전트: 작업 완료 후 DataHash 게시
2. 검증자 에이전트: 검증 요청 모니터링
3. 검증자: 스테이크 예치 후 작업 재실행
4. 검증자: 결과가 일치하면 ValidationResponse 제출
5. 불일치 시: 검증자의 스테이크 슬래싱(차감)
```

##### C. TEE 기반 Attestation (암호학적 검증)
- **Trusted Execution Environment** 활용
- 가장 높은 보안 수준
- 고위험 작업에 적합 (예: 의료 진단, 금융 자문)

**TEE 작동 방식:**
1. 에이전트가 TEE 엔클레이브 내에서 실행
2. CPU가 코드와 데이터 측정(measurement)
3. 하드웨어 키로 서명하여 Attestation 생성
4. 원격 검증자가 Attestation 검증

**인터페이스 예시:**
```solidity
interface IValidationRegistry {
    function requestValidation(
        bytes32 taskId,
        bytes32 dataHash,
        uint256 stake
    ) external payable;

    function submitValidation(
        bytes32 taskId,
        bytes32 dataHash,
        bytes calldata proof
    ) external;
}
```

---

## 🔐 신뢰 모델 비교

| 신뢰 모델 | 보안 메커니즘 | 적용 사례 | 비용 | 검증 속도 |
|----------|-------------|----------|------|----------|
| **Reputation** | 사회적 합의 | 피자 주문, 간단한 작업 | 낮음 | 빠름 |
| **Stake-based** | 경제적 인센티브 | 데이터 분석, 콘텐츠 생성 | 중간 | 중간 |
| **TEE Attestation** | 암호학적 증명 | 의료 진단, 법률 자문, 금융 거래 | 높음 | 느림 |

---

## 💡 핵심 설계 철학

### 1. 모듈화 (Modularity)
- 각 레지스트리는 독립적으로 작동
- 애플리케이션별 로직은 오프체인에서 처리
- 개발자가 자신의 사용 사례에 맞는 신뢰 모델 선택 가능

### 2. 체인 불가지론 (Chain-Agnostic)
- Ethereum뿐만 아니라 다양한 블록체인에서 작동 가능
- L2 솔루션과 호환

### 3. 확장성 (Scalability)
- 온체인에는 최소한의 데이터만 저장
- 대부분의 계산과 데이터는 오프체인
- 가스 비용 최소화

### 4. 보안 계층화 (Tiered Security)
- 위험 수준에 비례하는 보안 제공
- 낮은 위험 작업 → 가벼운 검증
- 높은 위험 작업 → 강력한 검증

---

## 🌐 에이전트 이코노미 (Agentic Economy)

### 비전
ERC-8004는 **자율적인 AI 에이전트들이 조직 간 경계를 넘어 거래하고, 협력하고, 평판을 구축하는 경제**를 가능하게 합니다.

### 사용 사례

#### 1. 저위험 작업 (Low-Stakes)
```
시나리오: AI 에이전트가 사용자를 위해 피자 주문
신뢰 모델: Reputation-based
작동 방식:
  1. 사용자의 에이전트가 Identity Registry에서 피자 배달 에이전트 검색
  2. Reputation Registry에서 평점 확인
  3. A2A 프로토콜로 주문 메시지 전송
  4. 작업 완료 후 피드백 제출
```

#### 2. 중위험 작업 (Medium-Stakes)
```
시나리오: 데이터 분석 AI 에이전트가 시장 보고서 생성
신뢰 모델: Stake-based Validation
작동 방식:
  1. 클라이언트 에이전트가 작업 요청 + 검증 요구사항 제출
  2. 서버 에이전트가 분석 수행 후 DataHash 게시
  3. 검증자 에이전트가 스테이크 예치 후 재분석
  4. 결과 일치 시 보상, 불일치 시 스테이크 슬래싱
  5. 피드백 attestation 온체인 기록
```

#### 3. 고위험 작업 (High-Stakes)
```
시나리오: 의료 진단 AI 에이전트
신뢰 모델: TEE Attestation
작동 방식:
  1. 진단 AI가 TEE 엔클레이브 내에서 실행
  2. 환자 데이터를 암호화된 상태로 처리
  3. CPU가 코드 무결성 측정 후 Attestation 생성
  4. 병원 시스템이 Attestation 검증
  5. 검증 성공 시 진단 결과 수용
  6. Validation Registry에 기록
```

---

## 🔗 관련 기술 스택

### 1. Model Context Protocol (MCP) vs A2A vs ERC-8004

| 프로토콜 | 역할 | 제공자 | 초점 |
|---------|------|--------|------|
| **MCP** | 도구 및 컨텍스트 접근 | Anthropic | 단일 에이전트 ↔ 도구 |
| **A2A** | 에이전트 간 통신 | Google | 에이전트 ↔ 에이전트 (Web2) |
| **ERC-8004** | 신뢰 레이어 | Ethereum Community | 에이전트 ↔ 에이전트 (Web3) |

**관계:**
```
MCP: 에이전트가 도구(tools)에 접근하는 방법
  ↓
A2A: 에이전트들이 서로 대화하는 방법
  ↓
ERC-8004: 에이전트들이 서로를 신뢰하는 방법
```

---

### 2. TEE (Trusted Execution Environment) 상세

#### TEE란?
메인 프로세서의 안전한 영역으로, 내부에 로드된 코드와 데이터의 **기밀성(Confidentiality)**과 **무결성(Integrity)**을 보호합니다.

#### 주요 TEE 기술
- **Intel SGX** (Software Guard Extensions)
- **AMD SEV** (Secure Encrypted Virtualization)
- **ARM TrustZone**

#### Attestation 프로세스
```
1. Boot Firmware → OS Kernel → Application 측정
2. 측정값을 보안 하드웨어 레지스터에 저장
3. CPU의 private attestation key로 서명
4. 암호학적 attestation 리포트 생성
5. 원격 검증자가 진위성 및 무결성 확인
```

#### Ethereum/Crypto에서의 TEE 활용
1. **Unichain (Uniswap)**
   - 블록 생성 과정에서 TEE 활용
   - MEV(Maximal Extractable Value) 보호

2. **TEEHEEHEE Agent**
   - AI 생성 결과를 TEE로 인증
   - 코드 변조 여부 확인

3. **Dark Pools & Private Trading**
   - 민감한 거래 정보를 TEE에서 처리
   - 노드 운영자도 개인 데이터 접근 불가

---

## 📊 SAGE 프로젝트와의 연관성

### 현재 SAGE 구현과 ERC-8004 비교

| 기능 | SAGE 현재 구현 | ERC-8004 표준 |
|------|--------------|--------------|
| **Identity** | SageRegistryV2 (DID 등록) | ✅ Identity Registry |
| **Reputation** | ❌ 미구현 | Reputation Registry |
| **Validation** | Enhanced Public Key Validation | ⚠️ 부분적 (TEE/Stake 미지원) |
| **A2A Protocol** | ✅ Handshake 구현 | ✅ 완전 호환 |
| **Crypto Verification** | RFC 9421 HTTP Signatures | ✅ 호환 |

---

### SAGE가 ERC-8004를 구현하기 위한 로드맵

#### Phase 1: Identity Registry 완성 ✅ (이미 구현됨)
```solidity
// SAGE의 SageRegistryV2가 이미 제공
- AgentID (DID)
- Public Key Ownership Proof
- On-chain Registration
```

#### Phase 2: Reputation Registry 구현 (권장)
```solidity
contract SageReputationRegistry {
    struct Feedback {
        bytes32 taskId;
        address clientAgent;
        address serverAgent;
        bytes32 dataHash;
        uint8 rating;
        uint256 timestamp;
    }

    mapping(address => Feedback[]) public agentFeedback;

    function submitFeedback(
        bytes32 taskId,
        address serverAgent,
        bytes32 dataHash,
        uint8 rating
    ) external {
        // Pre-authorization check
        require(isAuthorized(msg.sender, taskId), "Not authorized");

        agentFeedback[serverAgent].push(Feedback({
            taskId: taskId,
            clientAgent: msg.sender,
            serverAgent: serverAgent,
            dataHash: dataHash,
            rating: rating,
            timestamp: block.timestamp
        }));
    }
}
```

#### Phase 3: Validation Registry 구현 (고급)
```solidity
contract SageValidationRegistry {
    enum ValidationType { STAKE, TEE }

    struct ValidationRequest {
        bytes32 taskId;
        bytes32 dataHash;
        ValidationType validationType;
        uint256 stake;
        address requester;
    }

    mapping(bytes32 => ValidationRequest) public validationRequests;
    mapping(bytes32 => bool) public validatedTasks;

    uint256 public minStake = 0.1 ether;

    function requestValidation(
        bytes32 taskId,
        bytes32 dataHash,
        ValidationType validationType
    ) external payable {
        require(msg.value >= minStake, "Insufficient stake");

        validationRequests[taskId] = ValidationRequest({
            taskId: taskId,
            dataHash: dataHash,
            validationType: validationType,
            stake: msg.value,
            requester: msg.sender
        });

        emit ValidationRequested(taskId, dataHash, validationType, msg.value);
    }

    function submitStakeValidation(
        bytes32 taskId,
        bytes32 computedHash
    ) external payable {
        ValidationRequest memory req = validationRequests[taskId];
        require(req.validationType == ValidationType.STAKE, "Not stake validation");
        require(msg.value >= minStake, "Insufficient validator stake");

        if (computedHash == req.dataHash) {
            // Validation successful
            validatedTasks[taskId] = true;
            payable(msg.sender).transfer(req.stake / 10); // 10% reward
            emit ValidationSuccessful(taskId, msg.sender);
        } else {
            // Validation failed - slash validator stake
            payable(req.requester).transfer(msg.value);
            emit ValidationFailed(taskId, msg.sender);
        }
    }

    function submitTEEAttestation(
        bytes32 taskId,
        bytes calldata attestation
    ) external {
        ValidationRequest memory req = validationRequests[taskId];
        require(req.validationType == ValidationType.TEE, "Not TEE validation");

        // Verify TEE attestation
        require(verifyTEEAttestation(attestation, req.dataHash), "Invalid TEE attestation");

        validatedTasks[taskId] = true;
        emit ValidationSuccessful(taskId, msg.sender);
    }

    function verifyTEEAttestation(
        bytes calldata attestation,
        bytes32 expectedHash
    ) internal pure returns (bool) {
        // TODO: Implement TEE attestation verification
        // This would involve:
        // 1. Verify attestation signature with known TEE public keys
        // 2. Extract measurement from attestation
        // 3. Compare with expected hash
        return true; // Placeholder
    }

    event ValidationRequested(bytes32 indexed taskId, bytes32 dataHash, ValidationType validationType, uint256 stake);
    event ValidationSuccessful(bytes32 indexed taskId, address validator);
    event ValidationFailed(bytes32 indexed taskId, address validator);
}
```

---

## 🚀 산업 동향 및 전망

### 주요 지지 조직
- **Ethereum Foundation**
- **Linux Foundation**
- **Google**
- **Nethermind**
- **MetaMask**
- **50+ 기술 파트너**

### 채택 사례
- **UXLINK**: ERC-8004 프로토콜 채택 발표 (2025년)
- **QuestFlow**: Trustless Agent Economy 구축

### 타임라인
- **2024**: Internet of Agents 개념 등장
- **2025/04**: Google A2A 프로토콜 발표
- **2025/08**: ERC-8004 공식 제안
- **2025/Later**: Production-ready 버전 목표

---

## ⚠️ 보안 고려사항

### 1. Pre-authorization 메커니즘
- 악의적인 평판 조작 방지
- 작업 참여자만 피드백 제출 가능

### 2. 검증자 인센티브 관리
- 스테이크 슬래싱으로 허위 검증 방지
- 검증 보상으로 정직한 행동 유도

### 3. 영구 감사 추적
- 모든 작업과 피드백이 온체인 기록
- 사후 감사 및 분쟁 해결 가능

### 4. 사용자 주도 도메인 검증
- 에이전트 도메인의 진위성 확인
- DNS 기반 검증 메커니즘

### 5. Sybil Attack 방어
- 스테이크 요구사항으로 대량 계정 생성 비용 증가
- 평판 시스템에서 시간 가중치 적용

### 6. 데이터 프라이버시
- TEE를 통한 민감 데이터 보호
- 오프체인 데이터 최소화

---

## 📈 ERC-8004의 의의

### 1. **개방형 에이전트 경제 실현**
- 중앙 집중식 플랫폼 없이도 에이전트 간 거래 가능
- 조직 간 경계를 넘는 협업

### 2. **신뢰 메커니즘의 민주화**
- 대기업만이 아닌 모든 개발자가 신뢰 가능한 에이전트 구축 가능
- 저비용으로 높은 수준의 신뢰 달성

### 3. **Web3 + AI의 융합**
- 블록체인의 투명성 + AI의 자율성
- 새로운 비즈니스 모델 창출

### 4. **상호운용성 (Interoperability)**
- 다양한 프레임워크와 벤더의 에이전트 통합
- 기술적 장벽 제거

---

## 🎯 결론 및 권장사항

### SAGE 프로젝트를 위한 제안

#### 즉시 실행 가능 ✅
1. **ERC-8004 호환성 검증**
   - 현재 SageRegistryV2가 Identity Registry 역할 수행 확인
   - A2A 프로토콜 메시지 형식과 호환성 테스트

#### 단기 목표 (1-3개월)
2. **Reputation Registry 구현**
   - 간단한 피드백 시스템 추가
   - 오프체인 평판 집계 서비스 개발 (선택)

3. **ERC-8004 표준 준수 인증**
   - Ethereum Magicians 포럼에 구현 공유
   - 커뮤니티 피드백 수렴

#### 중기 목표 (3-6개월)
4. **Stake-based Validation 구현**
   - 검증자 스테이크 메커니즘 개발
   - 슬래싱 조건 및 보상 구조 설계

5. **통합 테스트 및 문서화**
   - End-to-end 사용 사례 테스트
   - 개발자 가이드 및 API 문서 작성

#### 장기 목표 (6-12개월)
6. **TEE Attestation 지원**
   - Intel SGX 또는 AMD SEV 통합 연구
   - 고위험 작업을 위한 암호학적 검증 구현

7. **크로스체인 호환성**
   - L2 솔루션 (Optimism, Arbitrum) 지원
   - 멀티체인 에이전트 ID 관리

### 생태계 참여
- **Ethereum Magicians 포럼**: ERC-8004 논의 참여
- **A2A Working Group**: 프로토콜 개발에 기여
- **Early Adopter Program**: 표준 형성 과정에 참여

### 비즈니스 기회
1. **에이전트 마켓플레이스**: 신뢰할 수 있는 에이전트 발견 플랫폼
2. **평판 집계 서비스**: 오프체인 평판 점수 계산 및 제공
3. **검증자 네트워크**: 스테이크 기반 작업 검증 서비스
4. **TEE 인프라**: 고위험 작업을 위한 TEE 환경 제공

---

## 📚 참고 자료

### 공식 문서
- [EIP-8004 사양](https://eips.ethereum.org/EIPS/eip-8004)
- [A2A Protocol GitHub](https://github.com/a2aproject/A2A)
- [Google A2A 발표](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)

### 커뮤니티
- [Ethereum Magicians Discussion](https://ethereum-magicians.org/t/erc-8004-trustless-agents/25098)
- [A2A Protocol Website](https://a2a-protocol.org/)

### 관련 기술
- [Intel SGX Documentation](https://www.intel.com/content/www/us/en/developer/tools/software-guard-extensions/overview.html)
- [RFC 9421: HTTP Message Signatures](https://datatracker.ietf.org/doc/html/rfc9421)

---

**ERC-8004는 단순한 기술 표준이 아니라, 자율적인 AI 에이전트들이 신뢰를 기반으로 협업하는 새로운 경제 패러다임의 시작입니다.**

---

*문서 작성일: 2025-10-06*
*작성자: SAGE Development Team*
*버전: 1.0*
