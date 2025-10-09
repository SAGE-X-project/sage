# SAGE 프로젝트 상세 가이드 - Part 5: 스마트 컨트랙트 및 온체인 레지스트리

## 목차
1. [스마트 컨트랙트 개요](#1-스마트-컨트랙트-개요)
2. [SageRegistry 컨트랙트 상세 분석](#2-sageregistry-컨트랙트-상세-분석)
3. [Hook 시스템과 확장성](#3-hook-시스템과-확장성)
4. [가스 최적화 기법](#4-가스-최적화-기법)
5. [컨트랙트 배포 프로세스](#5-컨트랙트-배포-프로세스)
6. [Go 언어와의 통합](#6-go-언어와의-통합)
7. [다국어 바인딩 (Python, JavaScript)](#7-다국어-바인딩)
8. [보안 고려사항](#8-보안-고려사항)
9. [실전 예제](#9-실전-예제)

---

## 1. 스마트 컨트랙트 개요

### 1.1 스마트 컨트랙트란?

스마트 컨트랙트는 블록체인 상에서 실행되는 프로그램입니다. 일종의 "자동 실행 계약"으로 생각할 수 있습니다.

#### 일반 프로그램과의 차이점

```
┌─────────────────────────────────────────────────────────────┐
│                  일반 프로그램 vs 스마트 컨트랙트              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  일반 프로그램 (예: Python 스크립트)                          │
│  ┌────────────────────────────────────┐                    │
│  │ 1. 중앙 서버에서 실행                │                    │
│  │ 2. 관리자가 언제든 수정 가능          │                    │
│  │ 3. 데이터가 변조될 수 있음            │                    │
│  │ 4. 서버가 다운되면 중단됨             │                    │
│  └────────────────────────────────────┘                    │
│                                                             │
│  스마트 컨트랙트 (예: Solidity)                               │
│  ┌────────────────────────────────────┐                    │
│  │ 1. 수천 개의 노드에서 분산 실행       │                    │
│  │ 2. 배포 후 코드 수정 불가 (불변성)    │                    │
│  │ 3. 모든 거래가 검증되고 기록됨        │                    │
│  │ 4. 단일 장애 지점 없음                │                    │
│  └────────────────────────────────────┘                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 SAGE에서의 스마트 컨트랙트 역할

SAGE 프로젝트에서 스마트 컨트랙트는 **AI 에이전트의 신원 증명 시스템**입니다.

#### 핵심 역할

1. **에이전트 등록** - 누가 정식 에이전트인지 기록
2. **공개키 저장** - 각 에이전트의 암호화 키 보관
3. **신원 검증** - 통신하려는 에이전트가 진짜인지 확인
4. **불변성 보장** - 등록된 정보는 임의로 변경 불가

#### 실생활 비유

```
스마트 컨트랙트 = 정부의 주민등록 시스템

┌─────────────────────────────────────────┐
│  일반 주민등록                            │
│  • 정부 서버에 신분 정보 저장             │
│  • 주민등록증 발급                        │
│  • 다른 기관이 신원 조회 가능             │
└─────────────────────────────────────────┘
         ↓ 블록체인 버전 ↓
┌─────────────────────────────────────────┐
│  SAGE 스마트 컨트랙트                     │
│  • 블록체인에 에이전트 정보 저장          │
│  • DID (탈중앙 신분증) 발급               │
│  • 누구나 에이전트 신원 검증 가능         │
└─────────────────────────────────────────┘
```

### 1.3 SAGE 컨트랙트 구조

```
contracts/ethereum/contracts/
├── SageRegistry.sol              ← 메인 레지스트리 컨트랙트
├── SageVerificationHook.sol      ← 검증 훅 컨트랙트
└── interfaces/
    ├── ISageRegistry.sol         ← 레지스트리 인터페이스
    └── IRegistryHook.sol         ← 훅 인터페이스
```

---

## 2. SageRegistry 컨트랙트 상세 분석

### 2.1 컨트랙트 구조 개요

SageRegistry는 약 380줄의 Solidity 코드로 구성된 핵심 컨트랙트입니다.

#### 파일 위치
- `contracts/ethereum/contracts/SageRegistry.sol`

#### 핵심 구성 요소

```solidity
contract SageRegistry is ISageRegistry {
    // 1. 데이터 구조
    struct RegistrationParams { ... }

    // 2. 상태 변수 (블록체인에 영구 저장되는 데이터)
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(bytes32 => uint256) private agentNonce;

    // 3. 보안 제약 조건
    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MIN_PUBLIC_KEY_LENGTH = 32;
    uint256 private constant MAX_PUBLIC_KEY_LENGTH = 65;

    // 4. 핵심 함수들
    function registerAgent(...) external returns (bytes32) { ... }
    function updateAgent(...) external { ... }
    function deactivateAgent(...) external { ... }
    function getAgent(...) external view returns (AgentMetadata memory) { ... }
}
```

### 2.2 데이터 저장 구조 (Mapping의 이해)

#### Mapping이란?

Mapping은 블록체인의 "거대한 데이터베이스 테이블"이라고 생각하면 됩니다.

```
일반 프로그래밍의 딕셔너리/맵과 유사:
Python: dict = {"key1": "value1", "key2": "value2"}
Go:     map[string]string{"key1": "value1"}
Solidity: mapping(string => string) data;
```

#### SAGE의 4가지 핵심 Mapping

**1. agents - 에이전트 전체 정보 저장**
```solidity
mapping(bytes32 => AgentMetadata) private agents;

// 사용 예시:
bytes32 agentId = 0x1234...;
AgentMetadata memory agent = agents[agentId];
// agent.name, agent.publicKey 등에 접근 가능
```

**비유:**
```
agentId를 "학번"이라고 생각하면
agents 는 "학생 정보 데이터베이스"

agents[학번20230001] = {
    이름: "김철수",
    연락처: "010-1234-5678",
    학과: "컴퓨터공학",
    ...
}
```

**2. didToAgentId - DID로 에이전트 찾기**
```solidity
mapping(string => bytes32) private didToAgentId;

// 사용 예시:
string memory did = "did:sage:kaia:alice";
bytes32 agentId = didToAgentId[did];
// 이제 agents[agentId]로 전체 정보 조회 가능
```

**비유:**
```
DID = "이메일 주소"
agentId = "학번"

didToAgentId["alice@sage.ai"] = "학번20230001"
→ 이메일로 학번을 찾고
→ 학번으로 학생 정보를 찾음
```

**3. ownerToAgents - 소유자별 에이전트 목록**
```solidity
mapping(address => bytes32[]) private ownerToAgents;

// 사용 예시:
address owner = 0xABCD...;
bytes32[] memory myAgents = ownerToAgents[owner];
// myAgents[0], myAgents[1] 등으로 접근
```

**비유:**
```
한 사람이 여러 에이전트를 소유할 수 있음

ownerToAgents[0xABCD...] = [
    agent1_id,
    agent2_id,
    agent3_id
]

실생활 예: 한 사람이 여러 전화번호를 가진 것과 유사
```

**4. agentNonce - 재생 공격 방지 카운터**
```solidity
mapping(bytes32 => uint256) private agentNonce;

// 사용 예시:
bytes32 agentId = 0x1234...;
uint256 currentNonce = agentNonce[agentId];
// 매번 업데이트할 때마다 nonce++
```

**비유:**
```
Nonce = "거래 일련번호"

거래 내역:
1회: 계좌 개설 (nonce=0)
2회: 정보 수정 (nonce=1)
3회: 정보 수정 (nonce=2)

오래된 서명(nonce=1)으로 새 요청을 보낼 수 없음
→ 재생 공격 방지
```

### 2.3 에이전트 등록 프로세스 상세 분석

#### 등록 프로세스 전체 흐름

```
User Application
     ↓
     ↓ (1) registerAgent 호출
     ↓
┌────────────────────────────────────────────────────────────┐
│  SageRegistry.registerAgent()                              │
│  (contracts/SageRegistry.sol:65-85)                        │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  Step 1: 파라미터 검증 (validPublicKey modifier)           │
│  ├─ 공개키 길이 체크 (32-65 bytes)                         │
│  └─ 잘못된 키 즉시 거부                                     │
│                                                            │
│  Step 2: 구조체로 파라미터 패킹                             │
│  ├─ Stack too deep 에러 방지                               │
│  └─ RegistrationParams 구조체 생성                         │
│                                                            │
│  Step 3: _registerAgent 내부 함수 호출                     │
│       ↓                                                    │
└───────┼────────────────────────────────────────────────────┘
        ↓
┌───────────────────────────────────────────────────────────┐
│  _registerAgent() 내부 처리                                │
│  (contracts/SageRegistry.sol:90-110)                       │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  Step 4: _validateRegistrationInputs()                    │
│  ├─ DID 중복 검사                                          │
│  ├─ 소유 에이전트 수 제한 (최대 100개)                      │
│  └─ 필수 필드 존재 확인                                     │
│                                                           │
│  Step 5: _generateAgentId()                               │
│  ├─ keccak256(DID + publicKey + timestamp)               │
│  └─ 고유한 32바이트 ID 생성                                │
│                                                           │
│  Step 6: _verifyRegistrationSignature()                   │
│  ├─ 메시지 해시 계산                                       │
│  ├─ ECDSA 서명 검증 (Ethereum)                            │
│  └─ 소유권 증명 완료                                       │
│                                                           │
│  Step 7: _executeBeforeHook() [선택적]                    │
│  ├─ 외부 검증 컨트랙트 호출                                │
│  └─ 블랙리스트 체크 등                                     │
│                                                           │
│  Step 8: _storeAgentMetadata()                            │
│  ├─ agents[agentId] 저장                                  │
│  ├─ didToAgentId[did] 매핑                                │
│  ├─ ownerToAgents 배열에 추가                             │
│  ├─ nonce 증가                                            │
│  └─ AgentRegistered 이벤트 발생                           │
│                                                           │
│  Step 9: _executeAfterHook() [선택적]                     │
│  └─ 등록 후 추가 처리                                      │
│                                                           │
└───────────────────────────────────────────────────────────┘
        ↓
    Return agentId
```

#### 코드 레벨 상세 분석

**Step 1-2: 진입점과 파라미터 검증**

```solidity
// contracts/SageRegistry.sol:65-85
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature
) external validPublicKey(publicKey) returns (bytes32) {
    // ↑ validPublicKey modifier가 먼저 실행됨
    //   (contracts/SageRegistry.sol:48-55)

    // Modifier 동작:
    modifier validPublicKey(bytes memory publicKey) {
        require(
            publicKey.length >= MIN_PUBLIC_KEY_LENGTH &&  // 32 bytes
            publicKey.length <= MAX_PUBLIC_KEY_LENGTH,     // 65 bytes
            "Invalid public key length"
        );
        _;  // 원래 함수 실행
    }

    // Stack too deep 에러 방지를 위한 구조체 사용
    RegistrationParams memory params = RegistrationParams({
        did: did,
        name: name,
        description: description,
        endpoint: endpoint,
        publicKey: publicKey,
        capabilities: capabilities,
        signature: signature
    });

    return _registerAgent(params);
}
```

**왜 구조체를 사용하나?**

Solidity의 EVM은 함수 내에서 사용할 수 있는 로컬 변수 개수에 제한이 있습니다 (Stack depth 제한). 파라미터가 7개 이상일 때 "Stack too deep" 에러가 발생할 수 있습니다.

```
No 에러 발생 예시:
function manyParams(
    string a, string b, string c,
    string d, string e, string f,
    string g, string h
) {
    // Stack too deep!
}

Yes 해결 방법:
struct Params { ... }
function betterParams(Params memory params) {
    // 구조체 하나만 스택에 올라감
}
```

**Step 4: 입력 검증**

```solidity
// contracts/SageRegistry.sol:115-123
function _validateRegistrationInputs(
    string memory did,
    string memory name
) private view {
    // 1. 필수 필드 체크
    require(bytes(did).length > 0, "DID required");
    require(bytes(name).length > 0, "Name required");

    // 2. 중복 등록 방지
    require(didToAgentId[did] == bytes32(0), "DID already registered");
    //      ↑ 이미 등록된 DID면 bytes32(0)이 아닌 값이 있음

    // 3. 소유 제한 (DoS 공격 방지)
    require(
        ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER,  // 100개
        "Too many agents"
    );
}
```

**msg.sender란?**

`msg.sender`는 Solidity의 특별한 전역 변수로, 현재 함수를 호출한 주소를 나타냅니다.

```
사용자 지갑 (0xABCD...)
    ↓ 트랜잭션 전송
SageRegistry.registerAgent()
    ↓ 함수 내부에서
msg.sender = 0xABCD...  // 자동으로 설정됨
```

**Step 5: 에이전트 ID 생성**

```solidity
// contracts/SageRegistry.sol:128-133
function _generateAgentId(
    string memory did,
    bytes memory publicKey
) private view returns (bytes32) {
    return keccak256(abi.encodePacked(
        did,              // "did:sage:kaia:alice"
        publicKey,        // 0x1234...
        block.timestamp   // 현재 블록 타임스탬프
    ));
}
```

**keccak256 해시 함수란?**

```
입력: 임의 길이의 데이터
출력: 고정된 32바이트 해시

예시:
입력1: "did:sage:kaia:alice" + publicKey + 1234567890
출력1: 0x3f7a... (32 bytes)

입력2: "did:sage:kaia:alice" + publicKey + 1234567891  (1초 차이)
출력2: 0x9c2b... (32 bytes, 완전히 다른 값)

특징:
• 같은 입력 → 항상 같은 출력
• 다른 입력 → 완전히 다른 출력 (눈사태 효과)
• 해시로부터 원본 복원 불가능 (일방향)
```

**abi.encodePacked란?**

여러 데이터를 하나의 바이트 배열로 합치는 함수입니다.

```
예시:
string a = "hello"
string b = "world"
uint c = 123

abi.encodePacked(a, b, c) =
    0x68656c6c6f776f726c647b  (hex)
    = "helloworld{" (ASCII + 123의 인코딩)
```

**Step 6: 서명 검증**

```solidity
// contracts/SageRegistry.sol:138-154
function _verifyRegistrationSignature(
    bytes32 agentId,
    RegistrationParams memory params
) private view {
    // 1. 서명할 메시지 생성
    bytes32 messageHash = keccak256(abi.encodePacked(
        params.did,
        params.name,
        params.description,
        params.endpoint,
        params.publicKey,
        params.capabilities,
        msg.sender,           // 호출자 주소 포함
        agentNonce[agentId]   // 재생 공격 방지
    ));

    // 2. 서명 검증
    require(
        _verifySignature(
            messageHash,
            params.signature,
            params.publicKey,
            msg.sender
        ),
        "Invalid signature"
    );
}
```

**서명 검증의 원리**

```
┌─────────────────────────────────────────────────────────┐
│  클라이언트 측 (등록 전)                                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 등록 데이터 준비                                      │
│     data = {did, name, description, ...}                │
│                                                         │
│  2. 메시지 해시 계산                                      │
│     messageHash = keccak256(data)                       │
│                                                         │
│  3. 개인키로 서명 생성                                    │
│     signature = sign(messageHash, privateKey)           │
│                                                         │
│  4. 데이터 + 서명을 컨트랙트에 전송                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
                    ↓ 블록체인 전송 ↓
┌─────────────────────────────────────────────────────────┐
│  스마트 컨트랙트 측 (검증)                                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 받은 데이터로 동일한 해시 계산                         │
│     messageHash' = keccak256(data)                      │
│                                                         │
│  2. 서명과 공개키로 서명자 복원                            │
│     signer = ecrecover(messageHash', signature)         │
│                                                         │
│  3. 복원된 주소가 msg.sender와 일치하는지 확인             │
│     require(signer == msg.sender)                       │
│                                                         │
│  Yes 검증 성공 → 개인키 소유자임을 증명                     │
│  No 검증 실패 → 거래 취소 (revert)                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**ECDSA 서명 복원 코드**

```solidity
// contracts/SageRegistry.sol:331-356
function _verifySignature(
    bytes32 messageHash,
    bytes memory signature,
    bytes memory publicKey,
    address expectedSigner
) private pure returns (bool) {
    // Ethereum 서명 (secp256k1)
    if (publicKey.length == 64 || publicKey.length == 65) {
        // Ethereum 서명 메시지 형식
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked(
                "\x19Ethereum Signed Message:\n32",
                messageHash
            )
        );

        // 서명으로부터 서명자 주소 복원
        address recovered = _recoverSigner(ethSignedHash, signature);

        // 복원된 주소가 예상 주소와 일치하는지 확인
        return recovered == expectedSigner;
    }

    // Ed25519 지원 (향후 추가 예정)
    if (publicKey.length == 32) {
        return true;  // Placeholder
    }

    return false;
}

// contracts/SageRegistry.sol:361-379
function _recoverSigner(
    bytes32 messageHash,
    bytes memory signature
) private pure returns (address) {
    require(signature.length == 65, "Invalid signature length");

    // ECDSA 서명 파싱 (r, s, v)
    bytes32 r;
    bytes32 s;
    uint8 v;

    assembly {
        r := mload(add(signature, 32))   // 첫 32바이트
        s := mload(add(signature, 64))   // 다음 32바이트
        v := byte(0, mload(add(signature, 96)))  // 마지막 1바이트
    }

    // ecrecover: EVM의 내장 함수
    return ecrecover(messageHash, v, r, s);
}
```

**ECDSA 서명 구조 (65 bytes)**

```
┌─────────────────────────────────────────────────┐
│  ECDSA Signature (65 bytes)                     │
├─────────────────────────────────────────────────┤
│                                                 │
│  r (32 bytes): 0x3f7a2b... (서명 첫 부분)       │
│  s (32 bytes): 0x9c4e1d... (서명 둘째 부분)     │
│  v (1 byte):   27 or 28    (복구 ID)            │
│                                                 │
│  r, s: 타원곡선 서명의 주요 값                   │
│  v: 공개키 복원을 위한 힌트                      │
│                                                 │
└─────────────────────────────────────────────────┘
```

**Step 8: 데이터 저장**

```solidity
// contracts/SageRegistry.sol:176-199
function _storeAgentMetadata(
    bytes32 agentId,
    RegistrationParams memory params
) private {
    // 1. 메인 저장소에 에이전트 정보 저장
    agents[agentId] = AgentMetadata({
        did: params.did,
        name: params.name,
        description: params.description,
        endpoint: params.endpoint,
        publicKey: params.publicKey,
        capabilities: params.capabilities,
        owner: msg.sender,              // 소유자 기록
        registeredAt: block.timestamp,  // 등록 시각
        updatedAt: block.timestamp,     // 수정 시각
        active: true                    // 활성 상태
    });

    // 2. DID → agentId 매핑
    didToAgentId[params.did] = agentId;

    // 3. 소유자 목록에 추가
    ownerToAgents[msg.sender].push(agentId);

    // 4. Nonce 증가 (다음 업데이트 대비)
    agentNonce[agentId]++;

    // 5. 이벤트 발생 (블록체인 로그)
    emit AgentRegistered(
        agentId,
        msg.sender,
        params.did,
        block.timestamp
    );
}
```

**블록체인 이벤트란?**

이벤트는 스마트 컨트랙트가 발생시키는 "로그"입니다. 상태 저장소보다 훨씬 저렴한 비용으로 데이터를 기록할 수 있습니다.

```
이벤트의 용도:
1. 프론트엔드 알림
   - 웹앱이 "등록 완료" 알림 표시

2. 데이터 인덱싱
   - The Graph 같은 서비스가 이벤트 수집
   - 빠른 검색을 위한 오프체인 DB 구축

3. 감사 로그
   - 모든 변경 사항 추적 가능

비용 비교:
• Storage 저장: ~20,000 gas
• Event 발생: ~375 gas
  (약 50배 저렴!)
```

**이벤트 정의 및 사용**

```solidity
// contracts/interfaces/ISageRegistry.sol:24-29
event AgentRegistered(
    bytes32 indexed agentId,    // indexed: 검색 가능
    address indexed owner,       // indexed: 검색 가능
    string did,                  // 일반 데이터
    uint256 timestamp            // 일반 데이터
);

// 사용 예시
emit AgentRegistered(
    0x1234...,
    0xABCD...,
    "did:sage:kaia:alice",
    1234567890
);
```

**indexed 키워드의 의미**

```
indexed 필드:
• 최대 3개까지 가능
• 검색 필터로 사용 가능
• 블록체인 인덱스에 저장됨

예시: 특정 owner의 이벤트만 필터링
events = contract.queryFilter(
    contract.filters.AgentRegistered(null, owner_address)
)
```

### 2.4 에이전트 업데이트 및 비활성화

#### 업데이트 함수

```solidity
// contracts/SageRegistry.sol:220-257
function updateAgent(
    bytes32 agentId,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    string calldata capabilities,
    bytes calldata signature
) external onlyAgentOwner(agentId) {
    // 1. 활성 상태 체크
    require(agents[agentId].active, "Agent not active");
    require(bytes(name).length > 0, "Name required");

    // 2. 새로운 nonce로 서명 검증
    bytes32 messageHash = keccak256(abi.encodePacked(
        agentId,
        name,
        description,
        endpoint,
        capabilities,
        msg.sender,
        agentNonce[agentId]  // 현재 nonce
    ));

    require(
        _verifySignature(
            messageHash,
            signature,
            agents[agentId].publicKey,  // 저장된 공개키 사용
            msg.sender
        ),
        "Invalid signature"
    );

    // 3. 메타데이터 업데이트 (공개키는 변경 불가!)
    agents[agentId].name = name;
    agents[agentId].description = description;
    agents[agentId].endpoint = endpoint;
    agents[agentId].capabilities = capabilities;
    agents[agentId].updatedAt = block.timestamp;

    // 4. Nonce 증가
    agentNonce[agentId]++;

    // 5. 이벤트 발생
    emit AgentUpdated(agentId, msg.sender, block.timestamp);
}
```

**onlyAgentOwner Modifier**

```solidity
// contracts/SageRegistry.sol:43-46
modifier onlyAgentOwner(bytes32 agentId) {
    require(
        agents[agentId].owner == msg.sender,
        "Not agent owner"
    );
    _;  // 원래 함수 실행
}
```

Modifier는 함수 실행 전에 조건을 체크하는 "문지기" 역할을 합니다.

```
함수 호출 흐름:

updateAgent(agentId, ...)
    ↓
onlyAgentOwner 체크
    ├─ agents[agentId].owner == msg.sender?
    │   ├─ Yes → 함수 실행
    │   └─ No  → Revert (거래 취소)
    ↓
실제 updateAgent 로직 실행
```

#### 비활성화 함수

```solidity
// contracts/SageRegistry.sol:262-269
function deactivateAgent(bytes32 agentId)
    external
    onlyAgentOwner(agentId)
{
    require(agents[agentId].active, "Agent already inactive");

    // 활성 상태를 false로 변경
    agents[agentId].active = false;
    agents[agentId].updatedAt = block.timestamp;

    emit AgentDeactivated(agentId, msg.sender, block.timestamp);
}
```

**삭제 vs 비활성화**

```
No 왜 삭제하지 않나?

1. 블록체인의 불변성
   - 한번 저장된 데이터는 영구 보존
   - delete 해도 과거 상태는 조회 가능

2. 참조 무결성
   - 다른 컨트랙트가 이 agentId를 참조할 수 있음
   - 삭제하면 다른 시스템이 오작동

3. 감사 추적
   - 과거 에이전트 활동 이력 보존
   - 보안 조사 시 필요

Yes 비활성화의 장점:
• 데이터 보존
• 필요시 재활성화 가능
• 역사적 기록 유지
```

### 2.5 조회 함수들

#### agentId로 조회

```solidity
// contracts/SageRegistry.sol:274-277
function getAgent(bytes32 agentId)
    external
    view
    returns (AgentMetadata memory)
{
    require(agents[agentId].registeredAt > 0, "Agent not found");
    return agents[agentId];
}
```

**view 함수란?**

```
Solidity 함수 종류:

1. 일반 함수 (State-changing)
   - 블록체인 상태 변경
   - 가스 비용 발생
   - 트랜잭션 필요

2. view 함수
   - 상태 읽기만 가능
   - 가스 비용 없음
   - 로컬 노드에서 즉시 실행

3. pure 함수
   - 상태 접근도 불가
   - 순수 계산만
   - 가스 비용 없음
```

#### DID로 조회

```solidity
// contracts/SageRegistry.sol:282-286
function getAgentByDID(string calldata did)
    external
    view
    returns (AgentMetadata memory)
{
    bytes32 agentId = didToAgentId[did];
    require(agentId != bytes32(0), "Agent not found");
    return agents[agentId];
}
```

**2단계 조회 과정**

```
Step 1: DID → agentId 변환
didToAgentId["did:sage:kaia:alice"] = 0x1234...

Step 2: agentId → 전체 정보
agents[0x1234...] = {
    name: "Alice Agent",
    publicKey: 0xABCD...,
    ...
}
```

#### 소유자별 조회

```solidity
// contracts/SageRegistry.sol:291-293
function getAgentsByOwner(address _owner)
    external
    view
    returns (bytes32[] memory)
{
    return ownerToAgents[_owner];
}
```

**배열 반환의 가스 비용**

```
Warning 주의사항:

ownerToAgents[address]가 100개 에이전트를 포함하면
모든 agentId를 한번에 반환

→ view 함수라 가스 비용은 없지만
→ RPC 응답 크기가 커질 수 있음

대안:
• 오프체인 인덱서 사용 (The Graph)
• 페이지네이션 구현
• 이벤트 기반 조회
```

---

## 3. Hook 시스템과 확장성

### 3.1 Hook 패턴이란?

Hook은 특정 시점에 외부 로직을 "끼워넣을" 수 있는 확장 메커니즘입니다.

#### 실생활 비유

```
온라인 쇼핑몰 주문 프로세스:

기본 프로세스:
1. 상품 선택
2. 결제
3. 배송

Hook 추가:
1. 상품 선택
2. [Before Hook] 재고 확인, 쿠폰 적용
3. 결제
4. [After Hook] 포인트 적립, 이메일 발송
5. 배송

→ Hook 덕분에 기본 프로세스 수정 없이 기능 추가 가능
```

### 3.2 SAGE Hook 시스템 구조

```
┌────────────────────────────────────────────────────────┐
│  SageRegistry.registerAgent()                          │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ... 입력 검증 ...                                      │
│                                                        │
│  ┌──────────────────────────────────────────┐         │
│  │  Before Hook (선택적)                     │         │
│  │  ────────────────────────────────────────│         │
│  │  • 추가 검증 로직                          │         │
│  │  • 블랙리스트 체크                         │         │
│  │  • 속도 제한                               │         │
│  │  • KYC 검증                                │         │
│  │                                           │         │
│  │  실패 시 → 등록 중단                       │         │
│  └──────────────────────────────────────────┘         │
│                                                        │
│  ... 데이터 저장 ...                                    │
│                                                        │
│  ┌──────────────────────────────────────────┐         │
│  │  After Hook (선택적)                      │         │
│  │  ────────────────────────────────────────│         │
│  │  • 알림 전송                               │         │
│  │  • 외부 시스템 동기화                       │         │
│  │  • 통계 업데이트                           │         │
│  │  • 보상 지급                               │         │
│  └──────────────────────────────────────────┘         │
│                                                        │
└────────────────────────────────────────────────────────┘
```

### 3.3 Hook 인터페이스

```solidity
// contracts/interfaces/IRegistryHook.sol
interface IRegistryHook {
    /**
     * @notice 등록 전 호출
     * @return success - 검증 통과 여부
     * @return reason - 실패 시 이유
     */
    function beforeRegister(
        bytes32 agentId,
        address agentOwner,
        bytes calldata data
    ) external returns (bool success, string memory reason);

    /**
     * @notice 등록 후 호출
     */
    function afterRegister(
        bytes32 agentId,
        address agentOwner,
        bytes calldata data
    ) external;
}
```

### 3.4 Before Hook 구현 예시

```solidity
// contracts/SageVerificationHook.sol:37-68
function beforeRegister(
    bytes32,  // agentId (미사용)
    address agentOwner,
    bytes calldata data
) external override returns (bool success, string memory reason) {
    // 1. 블랙리스트 체크
    if (blacklisted[agentOwner]) {
        return (false, "Address blacklisted");
    }

    // 2. 등록 쿨다운 체크 (1분)
    if (block.timestamp < lastRegistrationTime[agentOwner] + REGISTRATION_COOLDOWN) {
        return (false, "Registration cooldown active");
    }

    // 3. 일일 등록 제한 체크
    if (_isNewDay(agentOwner)) {
        registrationAttempts[agentOwner] = 0;  // 카운터 리셋
    }

    if (registrationAttempts[agentOwner] >= MAX_REGISTRATIONS_PER_DAY) {  // 5개
        return (false, "Daily registration limit reached");
    }

    // 4. DID 형식 검증
    (string memory did, ) = abi.decode(data, (string, bytes));
    if (!_isValidDID(did)) {
        return (false, "Invalid DID format");
    }

    return (true, "");  // 모든 검증 통과
}
```

**DID 형식 검증**

```solidity
// contracts/SageVerificationHook.sol:110-121
function _isValidDID(string memory did) private pure returns (bool) {
    bytes memory didBytes = bytes(did);

    // 최소 길이 체크
    if (didBytes.length < 10) return false;

    // "did:" 접두사 체크
    if (didBytes[0] != 'd' ||
        didBytes[1] != 'i' ||
        didBytes[2] != 'd' ||
        didBytes[3] != ':')
    {
        return false;
    }

    return true;
}
```

**일일 제한 체크**

```solidity
// contracts/SageVerificationHook.sol:103-105
function _isNewDay(address user) private view returns (bool) {
    return block.timestamp / 1 days > lastRegistrationTime[user] / 1 days;
}
```

이 코드는 "날짜가 바뀌었는지" 체크합니다.

```
예시:
현재 시각: 1234567890 (타임스탬프)
마지막 등록: 1234481290

1234567890 / 86400 = 14284  (현재 날짜 인덱스)
1234481290 / 86400 = 14283  (마지막 등록 날짜 인덱스)

14284 > 14283 → true (새로운 날)
→ 등록 카운터 리셋
```

### 3.5 After Hook 구현

```solidity
// contracts/SageVerificationHook.sol:73-84
function afterRegister(
    bytes32,  // agentId (미사용)
    address agentOwner,
    bytes calldata  // data (미사용)
) external override {
    // 등록 카운터 증가
    registrationAttempts[agentOwner]++;

    // 마지막 등록 시각 업데이트
    lastRegistrationTime[agentOwner] = block.timestamp;

    // 여기에 추가 로직 가능:
    // • 이벤트 발생
    // • 외부 서비스 API 호출
    // • 자동 테스트 트리거
}
```

### 3.6 Hook 설정 및 관리

```solidity
// contracts/SageRegistry.sol:316-325
function setBeforeRegisterHook(address hook) external onlyOwner {
    beforeRegisterHook = hook;
}

function setAfterRegisterHook(address hook) external onlyOwner {
    afterRegisterHook = hook;
}
```

**Hook 설정 프로세스**

```
1. Hook 컨트랙트 배포
   $ npx hardhat run scripts/deploy-hook.js
   → 주소: 0xHOOK_ADDRESS

2. SageRegistry에 Hook 주소 설정
   registry.setBeforeRegisterHook("0xHOOK_ADDRESS")

3. 이후 모든 등록 요청에 Hook 적용됨
```

**Hook 비활성화**

```solidity
// Hook을 0x0 주소로 설정하면 비활성화
registry.setBeforeRegisterHook(address(0))
```

### 3.7 Hook의 실전 활용 예시

#### 예시 1: KYC 검증 Hook

```solidity
contract KYCVerificationHook is IRegistryHook {
    mapping(address => bool) public kycVerified;

    function beforeRegister(
        bytes32,
        address agentOwner,
        bytes calldata
    ) external override returns (bool, string memory) {
        if (!kycVerified[agentOwner]) {
            return (false, "KYC verification required");
        }
        return (true, "");
    }

    function afterRegister(bytes32, address, bytes calldata) external override {
        // 등록 완료 이메일 발송 등
    }
}
```

#### 예시 2: 수수료 징수 Hook

```solidity
contract FeeCollectionHook is IRegistryHook {
    uint256 public constant REGISTRATION_FEE = 0.1 ether;

    function beforeRegister(
        bytes32,
        address,
        bytes calldata
    ) external payable override returns (bool, string memory) {
        if (msg.value < REGISTRATION_FEE) {
            return (false, "Insufficient registration fee");
        }
        return (true, "");
    }

    function afterRegister(
        bytes32,
        address agentOwner,
        bytes calldata
    ) external override {
        // 수수료를 treasury로 전송
        payable(treasury).transfer(REGISTRATION_FEE);
    }
}
```

---

## 4. 가스 최적화 기법

### 4.1 가스란?

가스는 Ethereum 및 EVM 기반 블록체인에서 연산 비용을 나타내는 단위입니다.

```
┌────────────────────────────────────────────────────┐
│  가스 비용 구조                                      │
├────────────────────────────────────────────────────┤
│                                                    │
│  최종 비용 = Gas Used × Gas Price                  │
│                                                    │
│  Gas Used: 실행한 연산의 복잡도                     │
│  Gas Price: 사용자가 지불할 가격 (Gwei 단위)        │
│                                                    │
│  예시:                                              │
│  • 등록 함수: 200,000 gas                          │
│  • Gas Price: 250 gwei                             │
│  • 비용: 0.05 KAIA (~$5)                           │
│                                                    │
└────────────────────────────────────────────────────┘
```

### 4.2 SAGE 컨트랙트의 가스 최적화 기법

#### 1. calldata vs memory

```solidity
// No 비효율적
function register(string memory did) external {
    // memory: 메모리에 복사 (비용 발생)
}

// Yes 효율적
function register(string calldata did) external {
    // calldata: 복사 없이 직접 읽기
}
```

**calldata의 장점**

```
calldata:
• 트랜잭션 데이터 영역에 직접 접근
• 메모리 복사 비용 없음
• 읽기 전용 (수정 불가)

memory:
• 데이터를 메모리로 복사
• 복사 비용 발생
• 수정 가능

gas 절약: 약 3,000-5,000 gas per string
```

#### 2. 구조체 패킹

```solidity
// No 비효율적 (3 slots)
struct Agent {
    bool active;        // 1 byte → 32 bytes slot
    address owner;      // 20 bytes → 32 bytes slot
    uint256 timestamp;  // 32 bytes → 32 bytes slot
}
// Total: 96 bytes (3 storage slots)

// Yes 효율적 (2 slots)
struct Agent {
    address owner;      // 20 bytes ┐
    bool active;        // 1 byte   │ → 32 bytes slot
    uint96 timestamp;   // 11 bytes ┘
    uint256 extra;      // 32 bytes → 32 bytes slot
}
// Total: 64 bytes (2 storage slots)
```

**Storage Slot 이해**

```
EVM Storage:
• 32바이트 단위로 저장
• 1 slot = 32 bytes
• Slot 읽기/쓰기 비용:
  - SLOAD: 2,100 gas
  - SSTORE: 20,000 gas (첫 쓰기)

패킹 효과:
• 3 slots → 2 slots
• 20,000 gas 절약 (한 slot 절약)
```

#### 3. 상수 사용 (constant, immutable)

```solidity
// No 비효율적
uint256 public maxAgents = 100;  // Storage에 저장 (SLOAD 필요)

// Yes 효율적
uint256 public constant MAX_AGENTS = 100;  // 바이트코드에 직접 삽입
```

**Constant vs Immutable vs 일반 변수**

```
constant:
• 컴파일 타임에 값 결정
• 바이트코드에 직접 포함
• Storage 사용 안 함
• Gas 비용: 거의 0

immutable:
• 생성자에서 한 번 설정
• 바이트코드에 포함
• Storage 사용 안 함
• Gas 비용: 거의 0

일반 변수:
• Storage에 저장
• 읽기마다 2,100 gas
• 쓰기마다 20,000 gas
```

#### 4. 이벤트 활용

```solidity
// No 비효율적: Storage에 모든 변경 기록 저장
struct Agent {
    ...
    uint256[] updateHistory;  // 매우 비쌈!
}

// Yes 효율적: 이벤트로 기록
event AgentUpdated(bytes32 indexed agentId, uint256 timestamp);
emit AgentUpdated(agentId, block.timestamp);
```

**Storage vs Event 비용 비교**

```
32바이트 데이터 저장:

Storage:
• SSTORE: 20,000 gas (첫 쓰기)
• SSTORE: 5,000 gas (재쓰기)

Event:
• LOG0: 375 gas (base)
• + 8 gas per byte
• Total: ~1,000 gas

→ Event가 20배 저렴!
```

#### 5. 짧은 에러 메시지

```solidity
// No 비효율적
require(condition, "This is a very long error message that costs more gas");
// 긴 문자열은 더 많은 gas 소비

// Yes 효율적
require(condition, "Invalid input");
// 짧고 명확한 메시지
```

#### 6. 함수 가시성 최적화

```solidity
// Private 함수는 internal보다 약간 저렴
function _validate(...) private {  // 외부 호출 불가
    ...
}

function _validate(...) internal {  // 상속 컨트랙트 호출 가능
    ...
}
```

#### 7. 배열 길이 캐싱

```solidity
// No 비효율적
for (uint i = 0; i < arr.length; i++) {  // 매번 length 읽기
    ...
}

// Yes 효율적
uint len = arr.length;  // 한 번만 읽기
for (uint i = 0; i < len; i++) {
    ...
}
```

### 4.3 Hardhat 가스 리포터

SAGE 프로젝트는 가스 사용량을 자동으로 측정합니다.

```javascript
// hardhat.config.js:168-176
const gasReporterConfig = {
  enabled: process.env.REPORT_GAS === "true",
  currency: "USD",
  gasPrice: 250,  // Kaia 기준
  coinmarketcap: process.env.COINMARKETCAP_API_KEY,
  excludeContracts: [],
  src: "./contracts",
  outputFile: process.env.GAS_REPORT_FILE
};
```

**가스 리포트 실행**

```bash
# .env 파일 설정
REPORT_GAS=true
COINMARKETCAP_API_KEY=your_api_key

# 테스트 실행
npm test

# 출력 예시:
┌─────────────────────┬─────────┬──────────┬──────────┬─────────┐
│ Contract            │ Method  │ Min      │ Max      │ Avg     │
├─────────────────────┼─────────┼──────────┼──────────┼─────────┤
│ SageRegistry        │ register│ 185,432  │ 205,876  │ 195,654 │
│ SageRegistry        │ update  │ 45,123   │ 52,987   │ 49,055  │
│ SageRegistry        │ deactiv │ 28,765   │ 28,765   │ 28,765  │
└─────────────────────┴─────────┴──────────┴──────────┴─────────┘
```

---

## 5. 컨트랙트 배포 프로세스

### 5.1 배포 환경 설정

#### Hardhat 설정 파일

```javascript
// hardhat.config.js:27-78
const networks = {
  // 로컬 개발
  hardhat: {
    chainId: 31337,
    mining: { auto: true, interval: 0 }
  },

  localhost: {
    url: "http://127.0.0.1:8545",
    chainId: 31337
  },

  // Kaia 테스트넷 (Kairos)
  kairos: {
    url: "https://public-en-kairos.node.kaia.io",
    chainId: 1001,
    accounts: [process.env.PRIVATE_KEY],
    gasPrice: 250 * 1e9,  // 250 Gwei
    gas: 3000000
  },

  // Kaia 메인넷
  kaia: {
    url: "https://public-en.node.kaia.io",
    chainId: 8217,
    accounts: [process.env.MAINNET_PRIVATE_KEY],
    gasPrice: 250 * 1e9
  }
};
```

#### 환경 변수 설정

```bash
# .env 파일
PRIVATE_KEY=0x...                  # 배포자 개인키
KAIROS_RPC_URL=https://...         # RPC 엔드포인트
GAS_PRICE_GWEI=250                 # 가스 가격
GAS_LIMIT=3000000                  # 가스 한도
```

### 5.2 컴파일 과정

```bash
# 컨트랙트 컴파일
npm run compile

# 실행 내용:
# 1. Solidity 파일을 bytecode로 변환
# 2. ABI (Application Binary Interface) 생성
# 3. artifacts/ 디렉토리에 저장
```

**컴파일 출력물**

```
artifacts/contracts/SageRegistry.sol/
├── SageRegistry.json              ← 메인 아티팩트
│   ├── abi: [...]                 ← 함수 인터페이스
│   ├── bytecode: "0x608060..."    ← 배포할 바이트코드
│   ├── deployedBytecode: "0x..." ← 배포 후 온체인 코드
│   └── metadata: {...}            ← 컴파일 메타데이터
└── SageRegistry.dbg.json          ← 디버그 정보
```

**ABI (Application Binary Interface)**

ABI는 컨트랙트와 외부 세계를 연결하는 "설명서"입니다.

```json
[
  {
    "type": "function",
    "name": "registerAgent",
    "inputs": [
      {"name": "did", "type": "string"},
      {"name": "name", "type": "string"},
      ...
    ],
    "outputs": [
      {"name": "agentId", "type": "bytes32"}
    ],
    "stateMutability": "nonpayable"
  },
  ...
]
```

### 5.3 배포 스크립트 구조

전체 테스트 및 배포 스크립트는 다음과 같습니다:

```bash
# contracts/ethereum/scripts/full-test.sh

#!/bin/bash

# 1. 컴파일
npm run compile

# 2. 로컬 노드 시작
npx hardhat node &
NODE_PID=$!

# 3. 컨트랙트 배포
npm run deploy:unified:local

# 4. 배포 검증
npx hardhat run scripts/verify-deployment.js

# 5. 테스트 실행
npm test

# 6. 정리
kill $NODE_PID
```

### 5.4 배포 프로세스 상세

#### Step 1: 로컬 노드 시작

```bash
npx hardhat node

# 출력:
Started HTTP and WebSocket JSON-RPC server at http://127.0.0.1:8545/

Accounts:
========
Account #0: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (10000 ETH)
Private Key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
...
```

Hardhat 노드는 개발용 로컬 블록체인입니다.

```
특징:
• 즉시 블록 생성 (0초 블록 타임)
• 무한 ETH 제공
• 트랜잭션 즉시 확인
• 디버깅 기능 내장
```

#### Step 2: 컨트랙트 배포

배포는 일반 트랜잭션과 유사하지만 `to` 주소가 없고 bytecode를 전송합니다.

```
일반 트랜잭션:
from: 0xABCD...
to: 0x1234...       ← 수신자 주소
value: 1 ETH
data: ""

컨트랙트 배포 트랜잭션:
from: 0xABCD...
to: null            ← 수신자 없음
value: 0
data: "0x608060..." ← 컨트랙트 bytecode
```

**배포 과정**

```
1. 트랜잭션 생성
   ├─ bytecode 준비
   ├─ 생성자 인자 인코딩
   └─ 개인키로 서명

2. 블록체인 전송
   ├─ RPC 노드로 전송
   └─ 트랜잭션 해시 수신

3. 마이닝 대기
   ├─ 블록에 포함 대기
   └─ 확인 블록 수 체크

4. 컨트랙트 주소 생성
   ├─ 주소 = keccak256(sender + nonce)[12:]
   └─ 주소 반환
```

#### Step 3: 배포 정보 저장

```json
// deployments/kairos-v2-latest.json
{
  "network": "kairos",
  "chainId": 1001,
  "timestamp": "2025-01-15T10:30:00.000Z",
  "deployer": "0xABCD...",
  "contracts": {
    "SageRegistry": {
      "address": "0x1234...",
      "transactionHash": "0x5678...",
      "blockNumber": 12345678,
      "gasUsed": "1234567"
    },
    "SageVerificationHook": {
      "address": "0x9ABC...",
      "transactionHash": "0xDEF0...",
      "blockNumber": 12345679,
      "gasUsed": "567890"
    }
  }
}
```

### 5.5 컨트랙트 검증 (Verification)

컨트랙트 검증은 소스 코드를 블록체인 탐색기에 공개하는 과정입니다.

```
검증 전:
└─ 탐색기에 bytecode만 표시
   └─ 0x608060405234801561001057600080fd5b50...

검증 후:
├─ 소스 코드 전체 표시
├─ Read/Write 함수 UI 제공
└─ 다른 개발자가 쉽게 통합 가능
```

**자동 검증**

```bash
# Kairos 테스트넷 검증
npx hardhat verify --network kairos 0x1234... "constructor_arg1" "arg2"

# 실행 내용:
# 1. 소스 코드 수집
# 2. 컴파일 설정 준비
# 3. Kaiascan API에 제출
# 4. 검증 완료 대기
```

**수동 검증**

```
1. Kaiascan 방문
   https://kairos.kaiascan.io/account/0x1234...

2. "Contract" 탭 → "Verify and Publish" 클릭

3. 정보 입력:
   ├─ Compiler Version: v0.8.19
   ├─ Optimization: Yes (200 runs)
   ├─ Source Code: [복사/붙여넣기]
   └─ Constructor Arguments: [인코딩된 값]

4. 제출 → 검증 완료
```

**검증 문서 위치**: `contracts/ethereum/docs/VERIFICATION_GUIDE.md`

---

## 6. Go 언어와의 통합

### 6.1 Go 바인딩 생성

SAGE는 Go 언어로 작성된 백엔드와 Solidity 컨트랙트를 연결하기 위해 **abigen** 도구를 사용합니다.

#### abigen이란?

```
abigen (ABI Generator):
• Ethereum Foundation 공식 도구
• ABI를 Go 구조체로 변환
• 타입 안전한 컨트랙트 호출 제공

변환 과정:
Solidity Contract
    ↓ compile
ABI JSON
    ↓ abigen
Go Binding Code
```

#### 바인딩 생성 프로세스

**Step 1: ABI 추출**

```bash
# 컴파일 후 ABI 파일 생성
npm run compile

# ABI 위치
artifacts/contracts/SageRegistry.sol/SageRegistry.json
```

**Step 2: abigen 실행**

```bash
# abigen 설치
go install github.com/ethereum/go-ethereum/cmd/abigen@latest

# 바인딩 생성
abigen \
  --abi=artifacts/contracts/SageRegistry.sol/SageRegistry.json \
  --pkg=registry \
  --type=SageRegistry \
  --out=bindings/SageRegistry.go
```

**Step 3: 생성된 Go 코드**

```go
// bindings/SageRegistry.go (자동 생성)

package registry

import (
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

// SageRegistry는 컨트랙트의 Go 바인딩입니다
type SageRegistry struct {
    SageRegistryCaller     // 읽기 전용 함수
    SageRegistryTransactor // 쓰기 함수
    SageRegistryFilterer   // 이벤트 필터링
    contract *bind.BoundContract
}

// SageRegistryCaller는 읽기 전용 바인딩입니다
type SageRegistryCaller struct {
    contract *bind.BoundContract
}

// GetAgent는 view 함수를 호출합니다
func (_SageRegistry *SageRegistryCaller) GetAgent(
    opts *bind.CallOpts,
    agentId [32]byte,
) (struct {
    Did          string
    Name         string
    Description  string
    Endpoint     string
    PublicKey    []byte
    Capabilities string
    Owner        common.Address
    RegisteredAt *big.Int
    UpdatedAt    *big.Int
    Active       bool
}, error) {
    // 내부 구현...
}

// RegisterAgent는 트랜잭션을 전송합니다
func (_SageRegistry *SageRegistryTransactor) RegisterAgent(
    opts *bind.TransactOpts,
    did string,
    name string,
    description string,
    endpoint string,
    publicKey []byte,
    capabilities string,
    signature []byte,
) (*types.Transaction, error) {
    // 내부 구현...
}
```

### 6.2 Go에서 컨트랙트 호출

#### 초기화

```go
package main

import (
    "context"
    "log"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/sage-x-project/sage/bindings/registry"
)

func main() {
    // 1. RPC 클라이언트 연결
    client, err := ethclient.Dial("https://public-en-kairos.node.kaia.io")
    if err != nil {
        log.Fatal(err)
    }

    // 2. 컨트랙트 주소
    contractAddr := common.HexToAddress("0x1234...")

    // 3. 컨트랙트 인스턴스 생성
    registry, err := registry.NewSageRegistry(contractAddr, client)
    if err != nil {
        log.Fatal(err)
    }

    // 이제 registry를 사용할 수 있음
}
```

#### 읽기 작업 (Call)

읽기 작업은 가스 비용이 없고 즉시 실행됩니다.

```go
// did/registry.go에서 발췌

// DID로 에이전트 조회
func (r *EthereumRegistry) GetAgentByDID(
    ctx context.Context,
    did string,
) (*AgentMetadata, error) {
    // CallOpts: 읽기 옵션
    opts := &bind.CallOpts{
        Context: ctx,
        Pending: false,  // 확정된 상태만 조회
    }

    // 컨트랙트 view 함수 호출
    agent, err := r.contract.GetAgentByDID(opts, did)
    if err != nil {
        return nil, fmt.Errorf("failed to get agent: %w", err)
    }

    // Solidity 구조체를 Go 구조체로 변환
    return &AgentMetadata{
        DID:          agent.Did,
        Name:         agent.Name,
        Description:  agent.Description,
        Endpoint:     agent.Endpoint,
        PublicKey:    agent.PublicKey,
        Capabilities: agent.Capabilities,
        Owner:        agent.Owner.Hex(),
        RegisteredAt: agent.RegisteredAt.Uint64(),
        UpdatedAt:    agent.UpdatedAt.Uint64(),
        Active:       agent.Active,
    }, nil
}
```

#### 쓰기 작업 (Transaction)

쓰기 작업은 가스 비용이 발생하고 블록 확인이 필요합니다.

```go
// 에이전트 등록
func (r *EthereumRegistry) Register(
    ctx context.Context,
    req *RegistrationRequest,
) (*RegistrationResult, error) {
    // 1. 서명 생성
    signature, err := r.signRegistrationData(req)
    if err != nil {
        return nil, err
    }

    // 2. TransactOpts: 트랜잭션 옵션
    auth, err := r.getTransactOpts(ctx)
    if err != nil {
        return nil, err
    }

    // 3. 컨트랙트 함수 호출
    tx, err := r.contract.RegisterAgent(
        auth,
        req.DID,
        req.Name,
        req.Description,
        req.Endpoint,
        req.KeyPair.PublicKey().Bytes(),
        req.Capabilities,
        signature,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to send transaction: %w", err)
    }

    // 4. 트랜잭션 확인 대기
    receipt, err := bind.WaitMined(ctx, r.client, tx)
    if err != nil {
        return nil, fmt.Errorf("failed to mine transaction: %w", err)
    }

    // 5. 실행 결과 확인
    if receipt.Status != types.ReceiptStatusSuccessful {
        return nil, fmt.Errorf("transaction failed")
    }

    // 6. 이벤트에서 agentId 추출
    agentId, err := r.extractAgentIdFromReceipt(receipt)
    if err != nil {
        return nil, err
    }

    return &RegistrationResult{
        AgentID:         agentId,
        TransactionHash: tx.Hash().Hex(),
        BlockNumber:     receipt.BlockNumber.Uint64(),
        GasUsed:         receipt.GasUsed,
    }, nil
}
```

**TransactOpts 구성**

```go
func (r *EthereumRegistry) getTransactOpts(
    ctx context.Context,
) (*bind.TransactOpts, error) {
    // 1. 개인키 로드
    privateKey, err := crypto.HexToECDSA(r.config.PrivateKey)
    if err != nil {
        return nil, err
    }

    // 2. Chain ID 가져오기
    chainID, err := r.client.ChainID(ctx)
    if err != nil {
        return nil, err
    }

    // 3. TransactOpts 생성
    auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
    if err != nil {
        return nil, err
    }

    // 4. 가스 설정
    auth.GasPrice = big.NewInt(int64(r.config.GasPrice))
    auth.GasLimit = r.config.GasLimit
    auth.Context = ctx

    return auth, nil
}
```

### 6.3 이벤트 모니터링

블록체인 이벤트를 실시간으로 감지할 수 있습니다.

```go
// 에이전트 등록 이벤트 구독
func (r *EthereumRegistry) WatchRegistrations(
    ctx context.Context,
) (<-chan *AgentRegisteredEvent, error) {
    // 1. 이벤트 채널 생성
    events := make(chan *registry.SageRegistryAgentRegistered)

    // 2. 이벤트 구독
    sub, err := r.contract.WatchAgentRegistered(
        &bind.WatchOpts{Context: ctx},
        events,
        nil,  // agentId 필터 (nil = 모든 이벤트)
        nil,  // owner 필터
    )
    if err != nil {
        return nil, err
    }

    // 3. 변환된 이벤트 채널
    out := make(chan *AgentRegisteredEvent)

    // 4. 백그라운드 처리
    go func() {
        defer close(out)
        defer sub.Unsubscribe()

        for {
            select {
            case event := <-events:
                // 이벤트 변환
                out <- &AgentRegisteredEvent{
                    AgentID:   hex.EncodeToString(event.AgentId[:]),
                    Owner:     event.Owner.Hex(),
                    DID:       event.Did,
                    Timestamp: event.Timestamp.Uint64(),
                }

            case err := <-sub.Err():
                log.Printf("Event subscription error: %v", err)
                return

            case <-ctx.Done():
                return
            }
        }
    }()

    return out, nil
}
```

**이벤트 사용 예시**

```go
func main() {
    ctx := context.Background()

    // 이벤트 구독
    events, err := registry.WatchRegistrations(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 이벤트 처리
    for event := range events {
        fmt.Printf("New agent registered:\n")
        fmt.Printf("  DID: %s\n", event.DID)
        fmt.Printf("  Owner: %s\n", event.Owner)
        fmt.Printf("  Time: %d\n", event.Timestamp)

        // 알림 전송, 데이터베이스 업데이트 등
    }
}
```

### 6.4 MultiChainRegistry 구현

SAGE는 여러 블록체인을 동시에 지원합니다.

```go
// did/registry.go:62-67

type MultiChainRegistry struct {
    registries map[Chain]Registry
    configs    map[Chain]*RegistryConfig
}

func NewMultiChainRegistry() *MultiChainRegistry {
    return &MultiChainRegistry{
        registries: make(map[Chain]Registry),
        configs:    make(map[Chain]*RegistryConfig),
    }
}
```

**다중 체인 등록**

```go
// did/registry.go:75-93

func (m *MultiChainRegistry) Register(
    ctx context.Context,
    chain Chain,
    req *RegistrationRequest,
) (*RegistrationResult, error) {
    // 1. 체인별 레지스트리 선택
    registry, exists := m.registries[chain]
    if !exists {
        return nil, fmt.Errorf("no registry for chain %s", chain)
    }

    // 2. 입력 검증
    if err := validateRegistrationRequest(req); err != nil {
        return nil, err
    }

    // 3. Chain prefix 추가
    if !hasChainPrefix(req.DID, chain) {
        req.DID = addChainPrefix(req.DID, chain)
    }

    // 4. 체인별 등록 실행
    return registry.Register(ctx, req)
}
```

**사용 예시**

```go
// 다중 체인 레지스트리 초기화
mcr := did.NewMultiChainRegistry()

// Kaia 체인 추가
kaiaRegistry := NewEthereumRegistry(kaiaConfig)
mcr.AddRegistry(did.ChainKaia, kaiaRegistry, kaiaConfig)

// Ethereum 체인 추가
ethRegistry := NewEthereumRegistry(ethConfig)
mcr.AddRegistry(did.ChainEthereum, ethRegistry, ethConfig)

// Kaia에 등록
result, err := mcr.Register(ctx, did.ChainKaia, req)

// Ethereum에 등록
result, err := mcr.Register(ctx, did.ChainEthereum, req)
```

---

## 7. 다국어 바인딩

### 7.1 Python 바인딩

SAGE 컨트랙트를 Python에서 사용할 수 있습니다.

#### 설치

```bash
pip install web3
```

#### 사용 예시

```python
# contracts/ethereum/bindings/python/example.py

from web3 import Web3
import json

# 1. RPC 연결
w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))

# 2. ABI 로드
with open('abi/SageRegistry.abi.json', 'r') as f:
    abi = json.load(f)

# 3. 컨트랙트 인스턴스
contract_address = '0x1234...'
contract = w3.eth.contract(address=contract_address, abi=abi)

# 4. 읽기 작업
agent = contract.functions.getAgentByDID('did:sage:kaia:alice').call()
print(f"Agent Name: {agent[1]}")  # agent[1] = name
print(f"Endpoint: {agent[3]}")    # agent[3] = endpoint

# 5. 쓰기 작업
from eth_account import Account

# 계정 로드
account = Account.from_key('0x...')

# 트랜잭션 구성
tx = contract.functions.registerAgent(
    did='did:sage:kaia:bob',
    name='Bob Agent',
    description='Python registered agent',
    endpoint='https://bob.example.com',
    publicKey=b'...',
    capabilities='{"chat": true}',
    signature=b'...'
).build_transaction({
    'from': account.address,
    'gas': 300000,
    'gasPrice': w3.to_wei('250', 'gwei'),
    'nonce': w3.eth.get_transaction_count(account.address),
})

# 서명 및 전송
signed_tx = w3.eth.account.sign_transaction(tx, account.key)
tx_hash = w3.eth.send_raw_transaction(signed_tx.rawTransaction)

# 확인 대기
receipt = w3.eth.wait_for_transaction_receipt(tx_hash)
print(f"Transaction successful: {receipt.status == 1}")
```

### 7.2 JavaScript/TypeScript 바인딩

#### 설치

```bash
npm install ethers
```

#### 사용 예시

```typescript
// contracts/examples/javascript/register.ts

import { ethers } from 'ethers';
import SageRegistryABI from './abi/SageRegistry.abi.json';

async function main() {
    // 1. Provider 연결
    const provider = new ethers.JsonRpcProvider(
        'https://public-en-kairos.node.kaia.io'
    );

    // 2. Wallet 로드
    const wallet = new ethers.Wallet(process.env.PRIVATE_KEY!, provider);

    // 3. 컨트랙트 연결
    const contractAddress = '0x1234...';
    const contract = new ethers.Contract(
        contractAddress,
        SageRegistryABI,
        wallet
    );

    // 4. 읽기 작업
    const agent = await contract.getAgentByDID('did:sage:kaia:alice');
    console.log('Agent:', {
        name: agent.name,
        endpoint: agent.endpoint,
        active: agent.active
    });

    // 5. 쓰기 작업
    const tx = await contract.registerAgent(
        'did:sage:kaia:charlie',
        'Charlie Agent',
        'TypeScript registered agent',
        'https://charlie.example.com',
        '0x...',  // publicKey
        '{"chat": true}',  // capabilities
        '0x...'   // signature
    );

    console.log('Transaction sent:', tx.hash);

    // 6. 확인 대기
    const receipt = await tx.wait();
    console.log('Transaction confirmed:', receipt.status === 1);

    // 7. 이벤트 파싱
    const event = receipt.logs
        .map(log => contract.interface.parseLog(log))
        .find(e => e?.name === 'AgentRegistered');

    if (event) {
        console.log('AgentID:', event.args.agentId);
        console.log('Owner:', event.args.owner);
    }
}

main().catch(console.error);
```

#### 이벤트 리스닝

```typescript
// 실시간 이벤트 모니터링

contract.on('AgentRegistered', (agentId, owner, did, timestamp, event) => {
    console.log('New agent registered!');
    console.log('  AgentID:', agentId);
    console.log('  Owner:', owner);
    console.log('  DID:', did);
    console.log('  Time:', new Date(timestamp.toNumber() * 1000));
});

// 과거 이벤트 조회
const filter = contract.filters.AgentRegistered(null, owner_address);
const events = await contract.queryFilter(filter, fromBlock, toBlock);

for (const event of events) {
    console.log('Historical event:', event.args);
}
```

### 7.3 Rust 바인딩

Rust는 `ethers-rs` 라이브러리를 사용합니다.

```rust
// contracts/ethereum/bindings/rust/src/main.rs

use ethers::prelude::*;
use std::sync::Arc;

abigen!(
    SageRegistry,
    "./abi/SageRegistry.abi.json"
);

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. Provider 연결
    let provider = Provider::<Http>::try_from(
        "https://public-en-kairos.node.kaia.io"
    )?;

    // 2. Wallet 로드
    let wallet: LocalWallet = "0x...".parse()?;
    let client = SignerMiddleware::new(provider, wallet);
    let client = Arc::new(client);

    // 3. 컨트랙트 인스턴스
    let address = "0x1234...".parse::<Address>()?;
    let contract = SageRegistry::new(address, client);

    // 4. 에이전트 조회
    let agent = contract
        .get_agent_by_did("did:sage:kaia:alice".to_string())
        .call()
        .await?;

    println!("Agent name: {}", agent.name);

    // 5. 에이전트 등록
    let tx = contract
        .register_agent(
            "did:sage:kaia:dave".to_string(),
            "Dave Agent".to_string(),
            "Rust registered".to_string(),
            "https://dave.example.com".to_string(),
            vec![0u8; 32].into(),  // publicKey
            "{}".to_string(),       // capabilities
            vec![0u8; 65].into(),   // signature
        )
        .send()
        .await?
        .await?;

    println!("Transaction: {:?}", tx);

    Ok(())
}
```

---

## 8. 보안 고려사항

### 8.1 재진입 공격 (Reentrancy) 방지

재진입 공격은 스마트 컨트랙트의 가장 유명한 취약점입니다.

#### 공격 원리

```solidity
// No 취약한 컨트랙트
contract Vulnerable {
    mapping(address => uint256) public balances;

    function withdraw() external {
        uint256 amount = balances[msg.sender];

        // 1. 먼저 송금 (위험!)
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success);

        // 2. 나중에 잔액 업데이트
        balances[msg.sender] = 0;
    }
}

// 공격자 컨트랙트
contract Attacker {
    Vulnerable target;

    receive() external payable {
        // withdraw() 재호출!
        if (address(target).balance > 0) {
            target.withdraw();  // 무한 반복
        }
    }

    function attack() external {
        target.withdraw();  // 최초 호출
    }
}
```

**공격 흐름**

```
1. Attacker.attack() 호출
2. Vulnerable.withdraw() 실행
3.   amount = balances[attacker] (예: 1 ETH)
4.   attacker.call{value: 1 ETH}()
5.     → Attacker.receive() 트리거
6.       → Vulnerable.withdraw() 재호출!
7.         amount = balances[attacker] (여전히 1 ETH!)
8.         attacker.call{value: 1 ETH}()
9.           → 무한 반복...
10. 컨트랙트 잔액 전부 탈취
```

#### SAGE의 방어 기법

**1. Checks-Effects-Interactions 패턴**

```solidity
// Yes 안전한 패턴
function withdraw() external {
    uint256 amount = balances[msg.sender];

    // 1. Checks - 조건 검증
    require(amount > 0, "No balance");

    // 2. Effects - 상태 먼저 변경
    balances[msg.sender] = 0;

    // 3. Interactions - 마지막에 외부 호출
    (bool success, ) = msg.sender.call{value: amount}("");
    require(success);
}
```

**SAGE 적용 예시**

```solidity
// contracts/SageRegistry.sol:176-199
function _storeAgentMetadata(...) private {
    // 1. Checks: 이미 검증 완료

    // 2. Effects: 상태 변경 먼저
    agents[agentId] = AgentMetadata({...});
    didToAgentId[params.did] = agentId;
    ownerToAgents[msg.sender].push(agentId);
    agentNonce[agentId]++;

    // 3. Interactions: 이벤트만 발생 (안전)
    emit AgentRegistered(...);
}
```

**2. ReentrancyGuard 사용**

```solidity
// OpenZeppelin의 ReentrancyGuard
abstract contract ReentrancyGuard {
    uint256 private _status;

    modifier nonReentrant() {
        require(_status != 2, "ReentrancyGuard: reentrant call");
        _status = 2;  // ENTERED
        _;
        _status = 1;  // NOT_ENTERED
    }
}

// 사용
contract Safe is ReentrancyGuard {
    function withdraw() external nonReentrant {
        // 재진입 불가능
    }
}
```

### 8.2 정수 오버플로우/언더플로우

Solidity 0.8.0 이상은 자동으로 오버플로우를 체크합니다.

```solidity
// Solidity 0.8.0+
uint256 a = type(uint256).max;  // 2^256 - 1
uint256 b = a + 1;  // No Panic! Overflow

// 이전 버전 (0.7.6 이하)
uint256 a = type(uint256).max;
uint256 b = a + 1;  // b = 0 (오버플로우, 버그!)
```

**SAGE 사용 버전**

```solidity
// contracts/SageRegistry.sol:2
pragma solidity ^0.8.19;  // Yes 자동 오버플로우 체크
```

### 8.3 서명 재사용 공격 방지

#### Nonce 시스템

```solidity
// contracts/SageRegistry.sol:27
mapping(bytes32 => uint256) private agentNonce;

// 등록 시 서명 검증
function _verifyRegistrationSignature(...) private view {
    bytes32 messageHash = keccak256(abi.encodePacked(
        ...,
        agentNonce[agentId]  // 현재 nonce 포함
    ));
    // ...
}

// 업데이트 후 nonce 증가
agentNonce[agentId]++;  // contracts/SageRegistry.sol:254
```

**Nonce의 작동 원리**

```
시나리오: Alice가 에이전트 정보 업데이트

1회차 업데이트:
├─ nonce: 0
├─ 서명: sign(data + nonce=0, privateKey)
├─ 컨트랙트: 검증 성공 → nonce++ (nonce=1)
└─ 결과: 성공

공격 시도: 이전 서명 재사용
├─ nonce: 1 (이미 증가함)
├─ 서명: sign(data + nonce=0, privateKey)  ← 이전 서명
├─ 컨트랙트: nonce 불일치 (0 ≠ 1)
└─ 결과: No 검증 실패

→ 오래된 서명으로 재공격 불가능
```

#### EIP-712 구조화된 서명 (향후 개선)

```solidity
// EIP-712 도메인 분리
bytes32 DOMAIN_SEPARATOR = keccak256(abi.encode(
    keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
    keccak256("SageRegistry"),
    keccak256("1"),
    block.chainid,
    address(this)
));

// 구조화된 메시지
bytes32 structHash = keccak256(abi.encode(
    keccak256("RegisterAgent(string did,string name,...)"),
    keccak256(bytes(did)),
    keccak256(bytes(name)),
    ...
));

bytes32 digest = keccak256(abi.encodePacked(
    "\x19\x01",
    DOMAIN_SEPARATOR,
    structHash
));
```

**EIP-712의 장점**

```
1. 체인별 분리
   - Ethereum 서명이 Kaia에서 재사용 불가

2. 컨트랙트별 분리
   - 다른 컨트랙트 서명 재사용 불가

3. 사용자 친화적
   - MetaMask 등에서 읽기 쉬운 서명 프롬프트
```

### 8.4 접근 제어

#### Modifier 기반 접근 제어

```solidity
// contracts/SageRegistry.sol:38-41
modifier onlyOwner() {
    require(msg.sender == owner, "Only owner");
    _;
}

modifier onlyAgentOwner(bytes32 agentId) {
    require(agents[agentId].owner == msg.sender, "Not agent owner");
    _;
}
```

**활용 예시**

```solidity
// 컨트랙트 소유자만 호출 가능
function setBeforeRegisterHook(address hook)
    external
    onlyOwner  // ← 접근 제어
{
    beforeRegisterHook = hook;
}

// 에이전트 소유자만 호출 가능
function updateAgent(bytes32 agentId, ...)
    external
    onlyAgentOwner(agentId)  // ← 접근 제어
{
    // 업데이트 로직
}
```

#### Role-Based Access Control (RBAC)

더 복잡한 권한 관리를 위해 OpenZeppelin의 AccessControl을 사용할 수 있습니다.

```solidity
// 향후 확장 예시
import "@openzeppelin/contracts/access/AccessControl.sol";

contract SageRegistryV3 is AccessControl {
    bytes32 public constant ADMIN_ROLE = keccak256("ADMIN_ROLE");
    bytes32 public constant MODERATOR_ROLE = keccak256("MODERATOR_ROLE");

    constructor() {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
    }

    function blacklistAgent(bytes32 agentId)
        external
        onlyRole(MODERATOR_ROLE)
    {
        // Moderator만 블랙리스트 가능
    }
}
```

### 8.5 가스 제한 공격 방지

#### 배열 순회 제한

```solidity
// No 위험: 무제한 배열
function getAllAgents(address owner) external view returns (AgentMetadata[] memory) {
    bytes32[] memory agentIds = ownerToAgents[owner];
    AgentMetadata[] memory agents = new AgentMetadata[](agentIds.length);

    // owner가 1000개 에이전트를 가지면?
    // → 가스 한도 초과로 실행 불가!
    for (uint i = 0; i < agentIds.length; i++) {
        agents[i] = agents[agentIds[i]];
    }

    return agents;
}

// Yes 안전: 최대 개수 제한
uint256 private constant MAX_AGENTS_PER_OWNER = 100;

require(
    ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER,
    "Too many agents"
);
```

#### 페이지네이션 패턴

```solidity
// 페이지네이션으로 대량 데이터 조회
function getAgentsPaginated(
    address owner,
    uint256 offset,
    uint256 limit
) external view returns (bytes32[] memory) {
    bytes32[] storage allAgents = ownerToAgents[owner];

    uint256 end = offset + limit;
    if (end > allAgents.length) {
        end = allAgents.length;
    }

    bytes32[] memory page = new bytes32[](end - offset);
    for (uint256 i = offset; i < end; i++) {
        page[i - offset] = allAgents[i];
    }

    return page;
}
```

### 8.6 타임스탬프 의존성

```solidity
// Warning 주의: block.timestamp는 조작 가능
// 마이너가 ~15초 범위 내에서 조작 가능

// No 위험한 사용
function lottery() external {
    uint256 winner = uint256(keccak256(abi.encodePacked(block.timestamp))) % players.length;
    // 마이너가 우승자 조작 가능!
}

// Yes 안전한 사용 (SAGE)
function _storeAgentMetadata(...) private {
    agents[agentId].registeredAt = block.timestamp;  // 정확한 시각 불필요
    agents[agentId].updatedAt = block.timestamp;     // 대략적 순서만 중요
}
```

**안전한 타임스탬프 사용 원칙**

```
1. 15초 이상의 시간 간격 사용
   Yes 일일 제한 (86400초)
   Yes 쿨다운 (60초)
   No 정밀한 타이밍 (1초 이하)

2. 순서 보장용으로만 사용
   Yes "이 거래가 저 거래보다 나중"
   No "정확히 10:30:00에 실행"

3. 난수 생성에 단독 사용 금지
   No random = hash(timestamp)
   Yes random = VRF (Chainlink VRF 등)
```

---

## 9. 실전 예제

### 9.1 전체 등록 플로우 (Go 클라이언트)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
)

func main() {
    ctx := context.Background()

    // 1. 암호화 키 생성
    keyPair, err := keys.GenerateEd25519KeyPair()
    if err != nil {
        log.Fatal(err)
    }

    // 2. DID 생성
    didStr := fmt.Sprintf(
        "did:sage:kaia:%s",
        keyPair.PublicKey().Fingerprint()[:16],
    )

    // 3. 레지스트리 초기화
    config := &did.RegistryConfig{
        Chain:           did.ChainKaia,
        Network:         did.NetworkKairos,
        ContractAddress: "0x1234...",
        RPCEndpoint:     "https://public-en-kairos.node.kaia.io",
        PrivateKey:      "0x...",  // 가스 지불용 키
        GasPrice:        250000000000,  // 250 Gwei
    }

    registry, err := did.NewEthereumRegistry(config)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 등록 요청 준비
    req := &did.RegistrationRequest{
        DID:          didStr,
        Name:         "My Agent",
        Description:  "Test agent for SAGE",
        Endpoint:     "https://my-agent.example.com",
        KeyPair:      keyPair,
        Capabilities: `{"chat": true, "image": false}`,
    }

    // 5. 블록체인에 등록
    fmt.Println("Registering agent on blockchain...")
    result, err := registry.Register(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Yes Registration successful!\n")
    fmt.Printf("   Agent ID: %s\n", result.AgentID)
    fmt.Printf("   Tx Hash: %s\n", result.TransactionHash)
    fmt.Printf("   Block: %d\n", result.BlockNumber)
    fmt.Printf("   Gas Used: %d\n", result.GasUsed)

    // 6. 등록 확인
    agent, err := registry.GetAgentByDID(ctx, didStr)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\n Agent Details:\n")
    fmt.Printf("   Name: %s\n", agent.Name)
    fmt.Printf("   Endpoint: %s\n", agent.Endpoint)
    fmt.Printf("   Active: %v\n", agent.Active)
    fmt.Printf("   Registered: %d\n", agent.RegisteredAt)
}
```

### 9.2 에이전트 업데이트

```go
func updateAgentEndpoint(
    ctx context.Context,
    registry did.Registry,
    agentDID did.AgentDID,
    newEndpoint string,
    keyPair crypto.KeyPair,
) error {
    // 업데이트 내용
    updates := map[string]interface{}{
        "endpoint": newEndpoint,
    }

    // 블록체인 업데이트
    err := registry.Update(ctx, agentDID, updates, keyPair)
    if err != nil {
        return fmt.Errorf("update failed: %w", err)
    }

    fmt.Printf("Yes Agent endpoint updated to: %s\n", newEndpoint)
    return nil
}
```

### 9.3 이벤트 기반 모니터링

```go
func monitorRegistrations(ctx context.Context, registry *EthereumRegistry) {
    // 이벤트 구독
    events, err := registry.WatchRegistrations(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("👀 Monitoring new agent registrations...")

    // 실시간 처리
    for event := range events {
        fmt.Printf("\n🆕 New Agent Registered!\n")
        fmt.Printf("   DID: %s\n", event.DID)
        fmt.Printf("   Owner: %s\n", event.Owner)
        fmt.Printf("   Time: %s\n", time.Unix(int64(event.Timestamp), 0))

        // 추가 작업
        go processNewAgent(event)
    }
}

func processNewAgent(event *AgentRegisteredEvent) {
    // 데이터베이스에 저장
    // 이메일 알림 전송
    // 자동 검증 시작
    // ...
}
```

### 9.4 Python으로 대시보드 만들기

```python
# dashboard.py

from web3 import Web3
import json
from datetime import datetime

class SageMonitor:
    def __init__(self, contract_address, abi_path):
        self.w3 = Web3(Web3.HTTPProvider(
            'https://public-en-kairos.node.kaia.io'
        ))

        with open(abi_path) as f:
            abi = json.load(f)

        self.contract = self.w3.eth.contract(
            address=contract_address,
            abi=abi
        )

    def get_agent_count(self, owner):
        """소유자의 에이전트 개수"""
        agents = self.contract.functions.getAgentsByOwner(owner).call()
        return len(agents)

    def get_recent_registrations(self, from_block='latest', count=10):
        """최근 등록 내역"""
        # 이벤트 필터
        event_filter = self.contract.events.AgentRegistered.create_filter(
            fromBlock=from_block
        )

        # 이벤트 조회
        events = event_filter.get_all_entries()

        # 최신 10개
        recent = events[-count:]

        result = []
        for event in recent:
            result.append({
                'did': event['args']['did'],
                'owner': event['args']['owner'],
                'timestamp': datetime.fromtimestamp(event['args']['timestamp']),
                'block': event['blockNumber'],
                'tx': event['transactionHash'].hex()
            })

        return result

    def print_dashboard(self):
        """대시보드 출력"""
        print("=" * 60)
        print(" SAGE Registry Dashboard")
        print("=" * 60)

        registrations = self.get_recent_registrations()

        print(f"\n📊 Recent Registrations ({len(registrations)}):\n")

        for i, reg in enumerate(registrations, 1):
            print(f"{i}. {reg['did']}")
            print(f"   Owner: {reg['owner']}")
            print(f"   Time: {reg['timestamp']}")
            print(f"   Tx: {reg['tx'][:16]}...")
            print()

# 사용
if __name__ == '__main__':
    monitor = SageMonitor(
        contract_address='0x1234...',
        abi_path='abi/SageRegistry.abi.json'
    )

    monitor.print_dashboard()
```

### 9.5 JavaScript 프론트엔드 통합

```typescript
// frontend/src/sage-registry.ts

import { ethers } from 'ethers';
import SageRegistryABI from './abi/SageRegistry.abi.json';

export class SageRegistryClient {
    private contract: ethers.Contract;
    private provider: ethers.Provider;

    constructor(contractAddress: string, providerUrl: string) {
        this.provider = new ethers.JsonRpcProvider(providerUrl);
        this.contract = new ethers.Contract(
            contractAddress,
            SageRegistryABI,
            this.provider
        );
    }

    // MetaMask 연결
    async connectWallet(): Promise<ethers.Signer> {
        if (!window.ethereum) {
            throw new Error('MetaMask not installed');
        }

        await window.ethereum.request({ method: 'eth_requestAccounts' });
        const provider = new ethers.BrowserProvider(window.ethereum);
        return provider.getSigner();
    }

    // 에이전트 조회
    async getAgent(did: string) {
        const agent = await this.contract.getAgentByDID(did);
        return {
            name: agent.name,
            description: agent.description,
            endpoint: agent.endpoint,
            active: agent.active,
            registeredAt: new Date(agent.registeredAt.toNumber() * 1000)
        };
    }

    // 에이전트 등록 (MetaMask 사용)
    async registerAgent(params: {
        did: string;
        name: string;
        description: string;
        endpoint: string;
        publicKey: Uint8Array;
        capabilities: string;
        signature: Uint8Array;
    }) {
        const signer = await this.connectWallet();
        const contractWithSigner = this.contract.connect(signer);

        const tx = await contractWithSigner.registerAgent(
            params.did,
            params.name,
            params.description,
            params.endpoint,
            params.publicKey,
            params.capabilities,
            params.signature
        );

        console.log('Transaction sent:', tx.hash);

        const receipt = await tx.wait();
        console.log('Transaction confirmed:', receipt);

        return receipt;
    }

    // 실시간 이벤트 구독
    subscribeToRegistrations(callback: (event: any) => void) {
        this.contract.on('AgentRegistered', (agentId, owner, did, timestamp) => {
            callback({
                agentId: agentId,
                owner: owner,
                did: did,
                timestamp: new Date(timestamp.toNumber() * 1000)
            });
        });
    }
}

// React 컴포넌트에서 사용
export function AgentRegistrationForm() {
    const [client] = useState(
        () => new SageRegistryClient(
            '0x1234...',
            'https://public-en-kairos.node.kaia.io'
        )
    );

    const handleRegister = async () => {
        try {
            const receipt = await client.registerAgent({
                did: formData.did,
                name: formData.name,
                description: formData.description,
                endpoint: formData.endpoint,
                publicKey: new Uint8Array(32),
                capabilities: '{}',
                signature: new Uint8Array(65)
            });

            alert(`Registration successful! Tx: ${receipt.transactionHash}`);
        } catch (error) {
            console.error('Registration failed:', error);
        }
    };

    // ...
}
```

---

## 결론

Part 5에서는 SAGE 프로젝트의 스마트 컨트랙트 시스템을 상세히 살펴보았습니다:

### 핵심 내용 요약

1. **스마트 컨트랙트 기초**
   - 블록체인 상의 불변 프로그램
   - AI 에이전트의 신원 증명 시스템
   - 탈중앙화된 레지스트리

2. **SageRegistry 컨트랙트**
   - 4가지 핵심 Mapping 구조
   - 9단계 등록 프로세스
   - 서명 기반 소유권 증명

3. **Hook 시스템**
   - Before/After Hook 패턴
   - 확장 가능한 검증 로직
   - 플러그인 아키텍처

4. **가스 최적화**
   - calldata vs memory
   - 구조체 패킹
   - 이벤트 활용

5. **배포 및 검증**
   - Hardhat 기반 개발 환경
   - 다중 네트워크 지원
   - 블록체인 탐색기 검증

6. **다국어 통합**
   - Go (abigen)
   - Python (web3.py)
   - JavaScript/TypeScript (ethers.js)
   - Rust (ethers-rs)

7. **보안**
   - 재진입 공격 방지
   - Nonce 기반 재생 공격 차단
   - 접근 제어 시스템

8. **실전 활용**
   - 전체 등록 플로우
   - 이벤트 모니터링
   - 대시보드 구축

### 다음 단계

Part 6에서는 전체 시스템의 데이터 플로우와 통합 가이드를 다룰 예정입니다:
- 암호화 → DID → 핸드셰이크 → 세션 → 스마트 컨트랙트의 완전한 통합
- 실제 AI 에이전트 간 통신 시나리오
- 프로덕션 배포 가이드
- 문제 해결 및 디버깅

---

**문서 정보**
- 작성일: 2025-01-15
- 버전: 1.0
- Part: 5/6
- 다음: [Part 6 - Complete Data Flow and Integration Guide](DETAILED_GUIDE_PART6_KO.md)
