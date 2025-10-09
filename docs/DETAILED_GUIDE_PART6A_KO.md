# SAGE 프로젝트 상세 가이드 - Part 6A: 완전한 데이터 플로우

## 목차
1. [전체 시스템 데이터 플로우 개요](#1-전체-시스템-데이터-플로우-개요)
2. [에이전트 등록부터 통신까지 완전한 흐름](#2-에이전트-등록부터-통신까지-완전한-흐름)
3. [키 생성에서 세션 종료까지](#3-키-생성에서-세션-종료까지)
4. [블록체인 레이어와 애플리케이션 레이어 통합](#4-블록체인-레이어와-애플리케이션-레이어-통합)
5. [에러 처리 및 복구 플로우](#5-에러-처리-및-복구-플로우)
6. [타이밍 다이어그램](#6-타이밍-다이어그램)

---

## 1. 전체 시스템 데이터 플로우 개요

### 1.1 SAGE의 전체 아키텍처 레이어

SAGE는 여러 레이어가 상호작용하는 복잡한 시스템입니다. 각 레이어의 역할을 이해하는 것이 중요합니다.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Application Layer                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  AI Agent Application                                    │   │
│  │  • LLM Integration (OpenAI, Claude, etc.)               │   │
│  │  • Business Logic                                        │   │
│  │  • User Interface                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                            ↕ API Calls
┌─────────────────────────────────────────────────────────────────┐
│                    SAGE Security Layer                          │
│  ┌─────────────┬──────────────┬─────────────┬────────────────┐ │
│  │  Session    │  Handshake   │  RFC 9421   │  Message       │ │
│  │  Manager    │  Protocol    │  Signatures │  Validation    │ │
│  └─────────────┴──────────────┴─────────────┴────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                            ↕ Encryption/Signing
┌─────────────────────────────────────────────────────────────────┐
│                  Cryptography Layer                             │
│  ┌─────────────┬──────────────┬─────────────┬────────────────┐ │
│  │  Ed25519    │  X25519      │  Secp256k1  │  ChaCha20      │ │
│  │  (Signing)  │  (Key Agree) │  (Ethereum) │  (Encryption)  │ │
│  └─────────────┴──────────────┴─────────────┴────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                            ↕ DID Operations
┌─────────────────────────────────────────────────────────────────┐
│                  Identity Layer (DID)                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  DID Manager                                             │   │
│  │  • DID Resolution (with multi-level caching)            │   │
│  │  • Key Discovery                                         │   │
│  │  • Trust Verification                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                            ↕ Blockchain RPC
┌─────────────────────────────────────────────────────────────────┐
│                  Blockchain Layer                               │
│  ┌────────────────┬─────────────────┬─────────────────────┐    │
│  │  Ethereum      │  Kaia           │  Solana             │    │
│  │  SageRegistry  │  SageRegistry   │  DID Program        │    │
│  │  Contract      │  Contract       │                     │    │
│  └────────────────┴─────────────────┴─────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 데이터 흐름의 3가지 주요 경로

SAGE에서 데이터는 크게 3가지 경로로 흐릅니다:

#### 경로 1: 등록 플로우 (Registration Flow)

```
Developer → SAGE CLI → Blockchain → Registry Contract
```

새로운 AI 에이전트를 시스템에 등록하는 경로입니다.

**특징:**
- 한 번만 실행 (에이전트당)
- 블록체인 트랜잭션 필요 (가스 비용 발생)
- 영구 기록 생성

**단계:**
1. 개발자가 키 생성
2. DID 생성
3. 서명 생성
4. 블록체인 전송
5. 확인 대기

#### 경로 2: 해결 플로우 (Resolution Flow)

```
Agent A → DID Resolver → Cache → Blockchain → Agent A
```

다른 에이전트의 정보를 조회하는 경로입니다.

**특징:**
- 자주 실행 (통신할 때마다)
- 읽기 전용 (가스 비용 없음)
- 다중 레벨 캐싱 사용

**단계:**
1. 메모리 캐시 확인
2. 로컬 DB 캐시 확인
3. 블록체인 조회
4. 결과 캐싱
5. 반환

#### 경로 3: 통신 플로우 (Communication Flow)

```
Agent A → Handshake → Session → Encrypted Message → Agent B
```

실제 에이전트 간 암호화된 메시지를 주고받는 경로입니다.

**특징:**
- 지속적으로 실행
- 블록체인 접근 불필요 (세션 수립 후)
- 최고 성능 요구

**단계:**
1. 핸드셰이크 (4단계)
2. 세션 생성
3. 메시지 암호화
4. 전송
5. 복호화 및 검증

### 1.3 컴포넌트 간 상호작용 맵

```
┌─────────────────────────────────────────────────────────────┐
│  전체 시스템 상호작용 다이어그램                              │
└─────────────────────────────────────────────────────────────┘

    [1] Key Generation          [2] DID Registration
         ↓                              ↓
    ┌─────────┐                  ┌──────────┐
    │ Ed25519 │ ←───────────────→│   DID    │
    │ Secp256k1│                 │ Manager  │
    └─────────┘                  └──────────┘
         ↓                              ↓
         └──────────────┬───────────────┘
                        ↓
              [3] Blockchain Registration
                        ↓
                 ┌─────────────┐
                 │  Registry   │
                 │  Contract   │
                 └─────────────┘
                        ↑
                        │ [4] DID Resolution
                        │
              ┌─────────┴─────────┐
              ↓                   ↓
        [5] Agent A         [6] Agent B
              ↓                   ↓
         ┌─────────┐         ┌─────────┐
         │Handshake│ ←──────→│Handshake│
         │ Client  │         │ Server  │
         └─────────┘         └─────────┘
              ↓                   ↓
         [7] X25519 Key Exchange
              ↓                   ↓
         ┌─────────┐         ┌─────────┐
         │ Session │ ←──────→│ Session │
         │ Manager │         │ Manager │
         └─────────┘         └─────────┘
              ↓                   ↓
         [8] Encrypted Communication
              ↓                   ↓
         ┌─────────┐         ┌─────────┐
         │ChaCha20 │ ←──────→│ChaCha20 │
         │Poly1305 │         │Poly1305 │
         └─────────┘         └─────────┘
```

---

## 2. 에이전트 등록부터 통신까지 완전한 흐름

### 2.1 Phase 1: 에이전트 준비 (Agent Preparation)

이 단계에서는 AI 에이전트가 SAGE 시스템에 참여하기 위한 준비를 합니다.

#### Step 1-1: 암호화 키 생성

```
시작: 개발자가 새 에이전트 생성 결정
↓

┌─────────────────────────────────────────────────────────┐
│  $ sage-crypto generate --type ed25519                  │
│    --name "my-agent" --output ./keys                    │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│  crypto/keys/ed25519.go:32-55                           │
│  GenerateEd25519KeyPair()                               │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 난수 생성기 초기화                                   │
│     rand.Reader (OS 제공 cryptographically secure)     │
│                                                         │
│  2. Ed25519 키 쌍 생성                                   │
│     publicKey, privateKey, err :=                       │
│         ed25519.GenerateKey(rand.Reader)                │
│                                                         │
│     • Public Key: 32 bytes                              │
│     • Private Key: 64 bytes                             │
│                                                         │
│  3. 키 ID 계산                                           │
│     keyID = SHA256(publicKey)[:16]                      │
│     → 16-character hex string                           │
│                                                         │
│  4. Ed25519KeyPair 구조체 생성                          │
│     kp := &Ed25519KeyPair{                              │
│         privateKey: privateKey,                         │
│         publicKey:  publicKey,                          │
│         keyID:      keyID,                              │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│  crypto/storage/file.go:45-89                           │
│  FileKeyStorage.Store()                                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. JWK 형식으로 변환                                    │
│     jwk := map[string]interface{}{                      │
│         "kty": "OKP",                                   │
│         "crv": "Ed25519",                               │
│         "x":   base64(publicKey),                       │
│         "d":   base64(privateKey),                      │
│     }                                                   │
│                                                         │
│  2. JSON 직렬화                                          │
│     data, _ := json.MarshalIndent(jwk, "", "  ")       │
│                                                         │
│  3. 안전한 파일 저장 (0600 권한)                         │
│     filename := fmt.Sprintf("%s.jwk", keyID)           │
│     ioutil.WriteFile(filename, data, 0600)             │
│                                                         │
│  4. 메타데이터 저장                                      │
│     metadata := KeyMetadata{                            │
│         KeyID:     keyID,                               │
│         Algorithm: "Ed25519",                           │
│         CreatedAt: time.Now(),                          │
│         Purpose:   "signing",                           │
│     }                                                   │
│     → keys/metadata.json에 추가                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
결과: ./keys/abc123def456.jwk 파일 생성 완료
```

**생성된 파일 예시:**

```json
// keys/abc123def456.jwk
{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "7v8Ag3...",  // Public key (Base64URL)
  "d": "9x2Bh4...",  // Private key (Base64URL)
  "kid": "abc123def456",
  "alg": "EdDSA",
  "use": "sig"
}
```

#### Step 1-2: DID 생성

```
입력: 생성된 Ed25519 공개키
↓

┌─────────────────────────────────────────────────────────┐
│  did/did.go:45-78                                       │
│  GenerateAgentDID()                                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 공개키 핑거프린트 계산                               │
│     publicKeyBytes := keyPair.PublicKey().Bytes()      │
│     hash := sha256.Sum256(publicKeyBytes)              │
│     fingerprint := base58.Encode(hash[:16])            │
│     → 예: "5HueCGU8rMjxEXxiPuD5BDku"                   │
│                                                         │
│  2. 체인 식별자 추가                                     │
│     chain := "kaia"  // 또는 "ethereum", "solana"      │
│                                                         │
│  3. DID 문자열 조합                                      │
│     did := fmt.Sprintf(                                 │
│         "did:sage:%s:%s",                               │
│         chain,                                          │
│         fingerprint,                                    │
│     )                                                   │
│                                                         │
│  4. DID 검증                                             │
│     if !isValidDIDFormat(did) {                         │
│         return error                                    │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
결과: "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku"
```

**DID 구조 분석:**

```
did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku
│   │    │    │
│   │    │    └─ Agent Identifier (공개키 기반)
│   │    └────── Blockchain Network (kaia/ethereum/solana)
│   └─────────── SAGE Method
└─────────────── DID Scheme (W3C 표준)
```

#### Step 1-3: 블록체인 등록 준비

```
입력: DID, 키 쌍, 에이전트 메타데이터
↓

┌─────────────────────────────────────────────────────────┐
│  $ sage-did register \                                  │
│      --chain kaia \                                     │
│      --name "Alice Agent" \                             │
│      --endpoint "https://alice.example.com" \           │
│      --key ./keys/abc123def456.jwk                      │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│  cmd/sage-did/register.go:88-174                        │
│  runRegister()                                          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 키 로드 (loadKeyPair)                               │
│     data := ioutil.ReadFile("./keys/abc123def456.jwk") │
│     keyPair := parseJWK(data)                          │
│                                                         │
│  2. 체인 검증 (validateKeyForChain)                     │
│     if chain == "kaia" && keyPair.Type() != Secp256k1: │
│         return error  // Kaia는 Secp256k1 필요        │
│                                                         │
│  3. Capabilities 파싱                                   │
│     capabilities := map[string]interface{}{            │
│         "chat": true,                                   │
│         "image": false,                                 │
│         "tools": ["calculator", "weather"],            │
│     }                                                   │
│                                                         │
│  4. 등록 요청 구조체 생성                                │
│     req := &did.RegistrationRequest{                   │
│         DID:          "did:sage:kaia:...",             │
│         Name:         "Alice Agent",                    │
│         Description:  "AI assistant",                   │
│         Endpoint:     "https://alice.example.com",     │
│         Capabilities: capabilities,                     │
│         KeyPair:      keyPair,                          │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
다음 단계: 블록체인 트랜잭션 생성
```

### 2.2 Phase 2: 블록체인 등록 (Blockchain Registration)

#### Step 2-1: 서명 생성

```
입력: RegistrationRequest
↓

┌─────────────────────────────────────────────────────────┐
│  did/registry.go (EthereumRegistry 내부)                │
│  signRegistrationData()                                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 서명할 데이터 준비                                   │
│     message := abi.encodePacked(                        │
│         req.DID,           // "did:sage:kaia:..."      │
│         req.Name,          // "Alice Agent"             │
│         req.Description,   // "AI assistant"            │
│         req.Endpoint,      // "https://..."             │
│         req.PublicKey,     // 32 bytes                  │
│         req.Capabilities,  // JSON string               │
│         msg.sender,        // 0xABCD... (지갑 주소)     │
│         nonce,             // 0 (첫 등록)               │
│     )                                                   │
│                                                         │
│  2. 메시지 해시 계산                                     │
│     messageHash := keccak256(message)                   │
│     → 32 bytes hash                                     │
│                                                         │
│  3. Ethereum 서명 형식 적용                              │
│     ethHash := keccak256(                               │
│         "\x19Ethereum Signed Message:\n32",            │
│         messageHash                                     │
│     )                                                   │
│                                                         │
│  4. 개인키로 서명                                        │
│     signature, err := crypto.Sign(                      │
│         ethHash,                                        │
│         privateKey,                                     │
│     )                                                   │
│     → 65 bytes (r: 32, s: 32, v: 1)                    │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
결과: 65-byte ECDSA signature
```

**서명 프로세스 상세:**

```
원본 데이터 (가변 길이)
    ↓
ABI 인코딩
    ↓
Keccak256 해시 (32 bytes)
    ↓
Ethereum 접두사 추가
    ↓
다시 Keccak256 해시 (32 bytes)
    ↓
ECDSA 서명 (개인키 사용)
    ↓
65-byte 서명 (r||s||v)
```

#### Step 2-2: 트랜잭션 전송

```
입력: RegistrationRequest + Signature
↓

┌─────────────────────────────────────────────────────────┐
│  did/registry.go:150-200                                │
│  EthereumRegistry.Register()                            │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. RPC 클라이언트 연결                                  │
│     client, err := ethclient.Dial(                      │
│         "https://public-en-kairos.node.kaia.io"        │
│     )                                                   │
│                                                         │
│  2. 컨트랙트 인스턴스 생성                               │
│     contract, err := bindings.NewSageRegistry(         │
│         contractAddress,  // 0x1234...                 │
│         client,                                         │
│     )                                                   │
│                                                         │
│  3. TransactOpts 준비                                   │
│     auth, err := bind.NewKeyedTransactor(              │
│         gasPayerPrivateKey,  // 가스 지불용 키          │
│         chainID,             // 1001 (Kairos)          │
│     )                                                   │
│     auth.GasPrice = big.NewInt(250 * 1e9)  // 250 Gwei │
│     auth.GasLimit = 300000                             │
│                                                         │
│  4. 컨트랙트 함수 호출                                   │
│     tx, err := contract.RegisterAgent(                 │
│         auth,                                           │
│         req.DID,              // string                 │
│         req.Name,             // string                 │
│         req.Description,      // string                 │
│         req.Endpoint,         // string                 │
│         req.PublicKey,        // bytes                  │
│         req.Capabilities,     // string (JSON)          │
│         signature,            // bytes (65)             │
│     )                                                   │
│                                                         │
│  5. 트랜잭션 전송 완료                                   │
│     fmt.Printf("Tx Hash: %s\n", tx.Hash().Hex())       │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
트랜잭션이 mempool에 전송됨
```

#### Step 2-3: 블록체인 처리

```
Kaia Blockchain Network
↓

┌─────────────────────────────────────────────────────────┐
│  Blockchain Processing                                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Mempool에서 트랜잭션 대기                            │
│     • 가스 가격 순으로 정렬                              │
│     • 검증: nonce, 서명, 잔액                           │
│                                                         │
│  2. 블록 생성자(Validator)가 선택                        │
│     • 새 블록에 포함                                     │
│     • Block Number: 12,345,678                          │
│     • Timestamp: 1705123456                             │
│                                                         │
│  3. EVM에서 컨트랙트 실행                                │
│     ┌─────────────────────────────────────────────┐   │
│     │ SageRegistry.registerAgent() 실행           │   │
│     │ (contracts/SageRegistry.sol:65-110)         │   │
│     ├─────────────────────────────────────────────┤   │
│     │                                             │   │
│     │ • validPublicKey modifier 체크              │   │
│     │ • _validateRegistrationInputs()             │   │
│     │   - DID 중복 체크                            │   │
│     │   - 소유자당 최대 100개 제한                 │   │
│     │                                             │   │
│     │ • _generateAgentId()                        │   │
│     │   agentId = keccak256(                      │   │
│     │       did + publicKey + timestamp           │   │
│     │   )                                         │   │
│     │   → 0x9a7b...                               │   │
│     │                                             │   │
│     │ • _verifyRegistrationSignature()            │   │
│     │   - 메시지 해시 재계산                       │   │
│     │   - ecrecover로 서명자 복원                 │   │
│     │   - msg.sender와 비교                       │   │
│     │                                             │   │
│     │ • _storeAgentMetadata()                     │   │
│     │   agents[agentId] = {                       │   │
│     │       did: "did:sage:kaia:...",            │   │
│     │       name: "Alice Agent",                  │   │
│     │       publicKey: 0x...,                     │   │
│     │       owner: 0xABCD...,                     │   │
│     │       registeredAt: 1705123456,            │   │
│     │       active: true                          │   │
│     │   }                                         │   │
│     │                                             │   │
│     │ • emit AgentRegistered(...)                 │   │
│     │                                             │   │
│     └─────────────────────────────────────────────┘   │
│                                                         │
│  4. State 변경 적용                                      │
│     • Storage에 영구 기록                                │
│     • Merkle Tree 업데이트                              │
│     • State Root 계산                                   │
│                                                         │
│  5. 블록 확정 (Finalization)                            │
│     • 네트워크 전파                                      │
│     • 2-3개 추가 블록 생성 (확인)                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
등록 완료 - 블록체인에 영구 기록됨
```

#### Step 2-4: 확인 및 결과 반환

```
Go 클라이언트 측
↓

┌─────────────────────────────────────────────────────────┐
│  did/registry.go (계속)                                 │
│  Register() - 확인 대기                                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  6. 트랜잭션 확인 대기                                   │
│     receipt, err := bind.WaitMined(                     │
│         ctx,                                            │
│         client,                                         │
│         tx,                                             │
│     )                                                   │
│                                                         │
│     • 블록에 포함될 때까지 대기                          │
│     • Timeout: 60초 (기본값)                            │
│                                                         │
│  7. 실행 결과 확인                                       │
│     if receipt.Status != 1 {                            │
│         // 트랜잭션 실패                                 │
│         return nil, fmt.Errorf("tx failed")            │
│     }                                                   │
│                                                         │
│  8. 이벤트에서 agentId 추출                              │
│     logs := receipt.Logs                                │
│     for _, log := range logs {                          │
│         event := contract.ParseAgentRegistered(log)     │
│         if event != nil {                               │
│             agentId = event.AgentId                     │
│             break                                       │
│         }                                               │
│     }                                                   │
│                                                         │
│  9. 결과 구조체 생성                                     │
│     result := &RegistrationResult{                      │
│         AgentID:         hex.Encode(agentId),          │
│         TransactionHash: tx.Hash().Hex(),              │
│         BlockNumber:     receipt.BlockNumber.Uint64(), │
│         GasUsed:         receipt.GasUsed,              │
│         Timestamp:       time.Now().Unix(),            │
│     }                                                   │
│                                                         │
│  10. 로컬 캐시 업데이트                                  │
│      r.cache.Set(req.DID, agentMetadata)               │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│  CLI 출력                                                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Yes Agent registered successfully!                      │
│  DID: did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku           │
│  Transaction: 0x7f8a9b...                               │
│  Block: 12,345,678                                      │
│  Gas Used: 187,432                                      │
│  Cost: 0.0468 KAIA (~$2.34)                            │
│                                                         │
│  Registration info saved to:                            │
│  ./keys/did_sage_kaia_5HueCGU8rMjxEXxiPuD5BDku.json    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 2.3 Phase 3: 에이전트 검색 (Agent Discovery)

Agent A가 Agent B와 통신하려면 먼저 B의 정보를 조회해야 합니다.

#### Step 3-1: DID Resolution 시작

```
Agent A Application
    ↓
"Agent B의 DID: did:sage:kaia:9xYz... 로 정보 조회"
    ↓

┌─────────────────────────────────────────────────────────┐
│  did/resolver.go:78-125                                 │
│  Resolver.Resolve(did)                                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Level 1 캐시: 메모리 (sync.Map)                     │
│     if doc, ok := r.memoryCache.Load(did); ok {        │
│         return doc, nil  // 즉시 반환 (< 1ms)           │
│     }                                                   │
│                                                         │
│  2. Level 2 캐시: 로컬 DB (BoltDB/SQLite)               │
│     if r.dbCache != nil {                               │
│         doc, err := r.dbCache.Get(did)                  │
│         if err == nil {                                 │
│             r.memoryCache.Store(did, doc)  // L1 캐시   │
│             return doc, nil  // 반환 (~10ms)            │
│         }                                               │
│     }                                                   │
│                                                         │
│  3. Level 3: 블록체인 조회                               │
│     chain := extractChainFromDID(did)                   │
│     client := r.getClientForChain(chain)                │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
캐시 미스 - 블록체인 조회 필요
```

#### Step 3-2: 블록체인 조회

```
┌─────────────────────────────────────────────────────────┐
│  Blockchain Query (Read-Only)                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. RPC 연결 (읽기 전용 - 가스 불필요)                   │
│     client := ethclient.Dial(rpcURL)                    │
│     contract := NewSageRegistry(address, client)        │
│                                                         │
│  2. view 함수 호출                                       │
│     agent, err := contract.GetAgentByDID(               │
│         &bind.CallOpts{},                               │
│         "did:sage:kaia:9xYz...",                        │
│     )                                                   │
│                                                         │
│     • 네트워크 왕복: ~100-500ms                          │
│     • 가스 비용: 0                                       │
│                                                         │
│  3. 응답 파싱                                            │
│     agent = {                                           │
│         Did:          "did:sage:kaia:9xYz...",         │
│         Name:         "Bob Agent",                      │
│         Description:  "Weather assistant",              │
│         Endpoint:     "https://bob.example.com",       │
│         PublicKey:    0x4f3e...,  // 32 bytes          │
│         Capabilities: "{\"weather\": true}",           │
│         Owner:        0x8765...,                        │
│         RegisteredAt: 1704567890,                       │
│         UpdatedAt:    1704567890,                       │
│         Active:       true,                             │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
에이전트 정보 획득
```

#### Step 3-3: DID Document 생성 및 캐싱

```
┌─────────────────────────────────────────────────────────┐
│  did/resolver.go (계속)                                 │
│  buildDIDDocument()                                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  4. DID Document 구조체 생성                             │
│     doc := &DIDDocument{                                │
│         Context: "https://www.w3.org/ns/did/v1",       │
│         ID:      "did:sage:kaia:9xYz...",              │
│         VerificationMethod: [{                          │
│             ID:   "did:sage:kaia:9xYz...#key-1",       │
│             Type: "Ed25519VerificationKey2020",        │
│             Controller: "did:sage:kaia:9xYz...",       │
│             PublicKeyMultibase: "z6Mk...",  // Base58  │
│         }],                                             │
│         Service: [{                                     │
│             ID:              "#agent-endpoint",         │
│             Type:            "AgentService",            │
│             ServiceEndpoint: "https://bob.example.com",│
│         }],                                             │
│     }                                                   │
│                                                         │
│  5. 다단계 캐싱                                          │
│     // Level 1: 메모리 캐시 (즉시 접근)                  │
│     r.memoryCache.Store(did, doc)                       │
│                                                         │
│     // Level 2: DB 캐시 (재시작 후에도 유지)             │
│     if r.dbCache != nil {                               │
│         r.dbCache.Set(did, doc, 24*time.Hour)          │
│     }                                                   │
│                                                         │
│  6. 반환                                                 │
│     return doc, nil                                     │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
Agent A가 Agent B의 공개키와 엔드포인트를 알게 됨
```

**캐시 성능 비교:**

```
┌──────────────┬──────────┬────────────────────┐
│ Cache Level  │ Latency  │ Storage Duration   │
├──────────────┼──────────┼────────────────────┤
│ L1 (Memory)  │ < 1ms    │ 프로세스 종료시까지 │
│ L2 (DB)      │ ~10ms    │ 24시간 (설정 가능)  │
│ L3 (Chain)   │ ~200ms   │ 영구 (블록체인)     │
└──────────────┴──────────┴────────────────────┘

두 번째 조회부터는 L1 캐시에서 즉시 반환
→ 200배 성능 향상!
```

### 2.4 Phase 4: 핸드셰이크 (Handshake)

이제 Agent A와 B가 서로를 알게 되었으니 안전한 통신 채널을 만듭니다.

#### Step 4-1: Invitation (초대)

```
Agent A
    ↓
"Agent B와 보안 세션을 시작하고 싶어"
    ↓

┌─────────────────────────────────────────────────────────┐
│  handshake/client.go:49-85                              │
│  Client.Invitation()                                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 클라이언트 임시 키 생성 (X25519)                     │
│     clientEphemeralPrivate, clientEphemeralPublic :=    │
│         x25519.GenerateKeyPair()                        │
│                                                         │
│     • Private: 32 bytes (비공개 저장)                    │
│     • Public:  32 bytes (상대방에게 전송)                │
│                                                         │
│  2. Nonce 생성 (재생 공격 방지)                          │
│     nonce := make([]byte, 16)                           │
│     rand.Read(nonce)                                    │
│     → 예: 0x7a3f9e2b...                                 │
│                                                         │
│  3. Invitation 메시지 생성                               │
│     invMsg := InvitationMessage{                        │
│         From:              "did:sage:kaia:5Hue...",    │
│         ClientEphemeralPK: clientEphemeralPublic,       │
│         Nonce:             hex.Encode(nonce),           │
│         Timestamp:         time.Now().Unix(),          │
│     }                                                   │
│                                                         │
│  4. RFC 9421 서명 생성                                   │
│     signature := signMessage(invMsg, edPrivateKey)      │
│     invMsg.Signature = signature                        │
│                                                         │
│  5. gRPC로 전송                                          │
│     response, err := c.a2aClient.SendMessage(ctx, &a2a.SendMessageRequest{│
│         To:      "did:sage:kaia:9xYz...",              │
│         Type:    "invitation",                          │
│         Payload: json.Marshal(invMsg),                  │
│     })                                                  │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
네트워크 전송 →
```

**RFC 9421 서명 프로세스:**

```
┌─────────────────────────────────────────────────────────┐
│  core/rfc9421/signer.go                                 │
│  Sign HTTP Message                                      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Signature Base 생성                                  │
│     base := ""                                          │
│     base += "\"@method\": POST\n"                       │
│     base += "\"@path\": /a2a.Agent/SendMessage\n"       │
│     base += "\"content-type\": application/json\n"      │
│     base += "\"content-digest\": sha-256=...\n"         │
│     base += "\"@signature-params\": (...)"              │
│                                                         │
│  2. Ed25519 서명                                         │
│     signature := ed25519.Sign(                          │
│         privateKey,                                     │
│         []byte(base),                                   │
│     )                                                   │
│     → 64 bytes                                          │
│                                                         │
│  3. HTTP Header에 추가                                   │
│     Signature: sig=:base64(signature):                  │
│     Signature-Input: sig=("@method" "@path" ...);      │
│                      keyid="did:sage:kaia:5Hue...#key-1"│
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### Step 4-2: Request (요청)

```
네트워크 →
    ↓
Agent B 수신
    ↓

┌─────────────────────────────────────────────────────────┐
│  handshake/server.go:159-241                            │
│  Server.HandleInvitation()                              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 서명 검증 (RFC 9421)                                 │
│     valid := verifySignature(                           │
│         message,                                        │
│         signature,                                      │
│         senderDID,                                      │
│     )                                                   │
│     if !valid {                                         │
│         return error("Invalid signature")              │
│     }                                                   │
│                                                         │
│  2. Nonce 중복 체크 (재생 공격 방지)                     │
│     if s.nonceCache.Exists(nonce) {                     │
│         return error("Nonce reused")                   │
│     }                                                   │
│     s.nonceCache.Add(nonce, 5*time.Minute)             │
│                                                         │
│  3. 타임스탬프 검증 (시간 동기화 공격 방지)              │
│     if abs(now - timestamp) > 60 {  // 1분 허용         │
│         return error("Timestamp out of range")         │
│     }                                                   │
│                                                         │
│  4. 임시 상태 저장                                       │
│     s.pendingState[senderDID] = &PendingState{         │
│         ClientEphemeralPK: invMsg.ClientEphemeralPK,    │
│         Nonce:             nonce,                       │
│         Timestamp:         timestamp,                   │
│     }                                                   │
│                                                         │
│  5. 서버 임시 키 생성                                    │
│     serverEphPriv, serverEphPub :=                      │
│         x25519.GenerateKeyPair()                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
```

**HPKE를 사용한 암호화 요청 생성:**

```
┌─────────────────────────────────────────────────────────┐
│  hpke/server.go:45-89                                   │
│  Server.Encapsulate()                                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Recipient Public Key 가져오기                        │
│     recipientPK := didDocument.VerificationMethod[0]    │
│         .PublicKeyMultibase                             │
│                                                         │
│  2. HPKE Context 생성                                    │
│     sender, err := hpke.SetupSender(                    │
│         suite,              // DHKEM(X25519, HKDF-SHA256)│
│         recipientPK,        // Agent A의 X25519 공개키   │
│         []byte("sage-handshake"),  // info             │
│     )                                                   │
│                                                         │
│     sender는 다음을 포함:                                │
│     • enc: 캡슐화된 키 (32 bytes)                        │
│     • 암호화/복호화를 위한 AEAD                          │
│                                                         │
│  3. 요청 메시지 암호화                                   │
│     reqMsg := RequestMessage{                           │
│         ServerEphemeralPK: serverEphPub,                │
│         Nonce:             newNonce,                    │
│     }                                                   │
│                                                         │
│     plaintext := json.Marshal(reqMsg)                   │
│     ciphertext := sender.Seal(                          │
│         plaintext,                                      │
│         []byte("handshake-request"),  // AAD           │
│     )                                                   │
│                                                         │
│  4. 응답 구성                                            │
│     response := &HpkeResponse{                          │
│         EncapsulatedKey: sender.enc,  // 32 bytes      │
│         Ciphertext:      ciphertext,                    │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
← 네트워크 전송
```

**HPKE 암호화 과정 상세:**

```
Agent B                          암호화 과정                     Agent A
  │                                                               │
  │  1. X25519 임시 키 쌍 생성                                     │
  │     (ephPriv, ephPub)                                        │
  │                                                               │
  │  2. DH 키 교환                                                │
  │     sharedSecret = X25519(ephPriv, clientEphPub) ────────────→ clientEphPub 알고 있음
  │     → 32 bytes 공유 비밀                                       │
  │                                                               │
  │  3. KDF로 암호화 키 유도                                       │
  │     key = HKDF-Extract-and-Expand(                           │
  │         sharedSecret,                                        │
  │         "sage-handshake"  // info                            │
  │     )                                                         │
  │     → 32 bytes AES-GCM 키                                     │
  │                                                               │
  │  4. 메시지 암호화                                             │
  │     ciphertext = AES-GCM-Encrypt(                            │
  │         key,                                                 │
  │         reqMsg,                                              │
  │         "handshake-request"  // AAD                          │
  │     )                                                         │
  │                                                               │
  │  5. 전송                                                      │
  │     send(ephPub || ciphertext) ──────────────────────────────→ 수신
  │                                                               │
  │                                                               6. 복호화
  │                                                                  sharedSecret = X25519(
  │                                                                      clientEphPriv,
  │                                                                      ephPub
  │                                                                  )
  │                                                                  key = HKDF(...)
  │                                                                  plaintext = AES-GCM-Decrypt(
  │                                                                      key,
  │                                                                      ciphertext
  │                                                                  )
```

#### Step 4-3: Response (응답) - 생략 (Part 4에서 이미 다룸)

#### Step 4-4: Complete (완료) - 생략 (Part 4에서 이미 다룸)

### 2.5 Phase 5: 세션 생성 및 통신

핸드셰이크가 완료되면 양측 모두 세션을 생성합니다.

```
Agent A & Agent B
    ↓

┌─────────────────────────────────────────────────────────┐
│  session/session.go:181-240                             │
│  NewSecureSession()                                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 공유 비밀 계산 (X25519 DH)                           │
│     sharedSecret := x25519.ComputeSharedSecret(         │
│         myEphemeralPrivate,                             │
│         peerEphemeralPublic,                            │
│     )                                                   │
│     → 32 bytes                                          │
│                                                         │
│  2. Session Seed 유도 (HKDF)                            │
│     seed := hkdf.Extract(                               │
│         sha256.New,                                     │
│         sharedSecret,                                   │
│         []byte("sage-session-v1"),  // salt            │
│     )                                                   │
│                                                         │
│     info := myDID || peerDID || timestamp              │
│     sessionSeed := hkdf.Expand(                         │
│         sha256.New,                                     │
│         seed,                                           │
│         info,                                           │
│         48,  // 384 bits                                │
│     )                                                   │
│                                                         │
│  3. Session ID 계산                                      │
│     hash := sha256.Sum256(                              │
│         sessionSeed || "session-id-v1"                  │
│     )                                                   │
│     sessionID := base58.Encode(hash[:16])              │
│     → "7vH3Jq9KmN2p..."                                 │
│                                                         │
│  4. 방향별 키 유도                                        │
│     c2sEncKey := deriveKey(sessionSeed, "c2s-enc")     │
│     c2sAuthKey := deriveKey(sessionSeed, "c2s-auth")   │
│     s2cEncKey := deriveKey(sessionSeed, "s2c-enc")     │
│     s2cAuthKey := deriveKey(sessionSeed, "s2c-auth")   │
│                                                         │
│     각 키는 32 bytes                                     │
│                                                         │
│  5. ChaCha20-Poly1305 AEAD 초기화                       │
│     c2sCipher, _ := chacha20poly1305.New(c2sEncKey)    │
│     s2cCipher, _ := chacha20poly1305.New(s2cEncKey)    │
│                                                         │
│  6. SecureSession 구조체 생성                            │
│     session := &SecureSession{                          │
│         sessionID:     sessionID,                       │
│         localDID:      myDID,                           │
│         remoteDID:     peerDID,                         │
│         c2sEncKey:     c2sEncKey,                       │
│         c2sAuthKey:    c2sAuthKey,                      │
│         s2cEncKey:     s2cEncKey,                       │
│         s2cAuthKey:    s2cAuthKey,                      │
│         c2sCipher:     c2sCipher,                       │
│         s2cCipher:     s2cCipher,                       │
│         createdAt:     time.Now(),                     │
│         lastUsed:      time.Now(),                     │
│     }                                                   │
│                                                         │
│  7. Session Manager에 등록                               │
│     manager.AddSession(session)                         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**키 유도 트리 (Key Derivation Tree):**

```
Shared Secret (32 bytes)
    │
    ├─ HKDF-Extract ───→ PRK (32 bytes)
    │
    └─ HKDF-Expand
         │
         ├─ info: "myDID||peerDID||timestamp"
         │
         └─ Session Seed (48 bytes)
              │
              ├─ Session ID ← SHA256(seed || "session-id-v1")
              │
              ├─ c2s-enc-key ← HKDF-Expand(seed, "c2s-enc", 32)
              │
              ├─ c2s-auth-key ← HKDF-Expand(seed, "c2s-auth", 32)
              │
              ├─ s2c-enc-key ← HKDF-Expand(seed, "s2c-enc", 32)
              │
              └─ s2c-auth-key ← HKDF-Expand(seed, "s2c-auth", 32)

결과: 각 방향마다 독립적인 암호화키와 인증키
```

#### 암호화된 메시지 전송

```
Agent A가 Agent B에게 메시지 전송
    ↓

┌─────────────────────────────────────────────────────────┐
│  session/session.go:300-345                             │
│  SecureSession.EncryptMessage()                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. 메시지 준비                                          │
│     plaintext := "Hello, Agent B!"                      │
│                                                         │
│  2. Nonce 생성 (12 bytes for ChaCha20-Poly1305)        │
│     nonce := make([]byte, 12)                           │
│     rand.Read(nonce)                                    │
│                                                         │
│  3. AAD (Additional Authenticated Data) 구성            │
│     aad := sessionID || seqNumber || timestamp          │
│                                                         │
│  4. AEAD 암호화                                          │
│     ciphertext := s.c2sCipher.Seal(                     │
│         nil,        // dst                              │
│         nonce,      // 12 bytes                         │
│         plaintext,  // 메시지                            │
│         aad,        // 인증할 추가 데이터                 │
│     )                                                   │
│                                                         │
│     결과:                                                │
│     ciphertext = encrypted_data || auth_tag (16 bytes) │
│                                                         │
│  5. Encrypted Message 구조체                            │
│     encMsg := &EncryptedMessage{                        │
│         SessionID:   sessionID,                         │
│         Nonce:       nonce,                             │
│         Ciphertext:  ciphertext,                        │
│         SeqNumber:   s.seqNumber++,                     │
│         Timestamp:   time.Now().Unix(),                │
│     }                                                   │
│                                                         │
│  6. 전송                                                 │
│     send(encMsg)                                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        ↓
네트워크 전송 →
```

#### 메시지 복호화 및 검증

```
← 네트워크 수신
    ↓
Agent B

┌─────────────────────────────────────────────────────────┐
│  session/session.go:347-395                             │
│  SecureSession.DecryptMessage()                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Session ID 확인                                      │
│     if encMsg.SessionID != s.sessionID {                │
│         return error("Session mismatch")               │
│     }                                                   │
│                                                         │
│  2. Sequence Number 검증 (재생 공격 방지)                │
│     if encMsg.SeqNumber <= s.lastSeqNumber {            │
│         return error("Old sequence number")            │
│     }                                                   │
│     s.lastSeqNumber = encMsg.SeqNumber                  │
│                                                         │
│  3. Timestamp 검증                                       │
│     if abs(now - encMsg.Timestamp) > 300 {  // 5분      │
│         return error("Message too old")                │
│     }                                                   │
│                                                         │
│  4. AAD 재구성                                           │
│     aad := encMsg.SessionID ||                          │
│           encMsg.SeqNumber ||                           │
│           encMsg.Timestamp                              │
│                                                         │
│  5. AEAD 복호화 및 인증                                  │
│     plaintext, err := s.s2cCipher.Open(                 │
│         nil,                 // dst                     │
│         encMsg.Nonce,        // 12 bytes                │
│         encMsg.Ciphertext,   // encrypted + tag         │
│         aad,                 // AAD (검증용)             │
│     )                                                   │
│                                                         │
│     인증 태그가 일치하지 않으면 error 반환               │
│     → 변조 감지!                                         │
│                                                         │
│  6. 성공                                                 │
│     return plaintext, nil                               │
│     → "Hello, Agent B!"                                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**AEAD 동작 원리 (ChaCha20-Poly1305):**

```
암호화 (Seal):
plaintext: "Hello, Agent B!"
    ↓
ChaCha20 Stream Cipher
    ↓
ciphertext: 0x3f7a9e... (암호화된 데이터)
    ↓
Poly1305 MAC
    ├─ Input: ciphertext + AAD
    └─ Output: 16-byte authentication tag
    ↓
Result: ciphertext || tag


복호화 (Open):
ciphertext || tag
    ↓
1. Poly1305 검증
   ├─ 재계산: MAC' = Poly1305(ciphertext + AAD)
   └─ 비교: MAC' == tag?
       ├─ Yes → 계속
       └─ No  → ERROR (변조됨!)
    ↓
2. ChaCha20 복호화
   plaintext = ChaCha20_Decrypt(ciphertext)
    ↓
Result: "Hello, Agent B!"
```

---

## 3. 키 생성에서 세션 종료까지

### 3.1 키 생명주기 (Key Lifecycle)

```
┌─────────────────────────────────────────────────────────┐
│  Key Lifecycle Timeline                                 │
└─────────────────────────────────────────────────────────┘

[1] Key Generation
    │
    ├─ Ed25519 (Signing)
    │  • Purpose: DID 등록, 메시지 서명
    │  • Lifetime: 영구 (또는 수동 교체)
    │  • Storage: 암호화된 파일 (0600 권한)
    │
    ├─ Secp256k1 (Ethereum)
    │  • Purpose: 블록체인 트랜잭션 서명
    │  • Lifetime: 영구
    │  • Storage: 암호화된 파일
    │
    └─ X25519 (Key Agreement)
       • Purpose: HPKE, 세션 키 교환
       • Lifetime: 단일 핸드셰이크 (임시)
       • Storage: 메모리만 (디스크 저장 안 함)

        ↓

[2] Key Registration (블록체인)
    │
    • Ed25519/Secp256k1 공개키만 등록
    • 개인키는 절대 공유/전송 안 함
    • 블록체인에 영구 기록

        ↓

[3] Key Exchange (핸드셰이크)
    │
    • X25519 임시 키 쌍 생성
    • 공개키만 교환
    • DH로 공유 비밀 계산
    • 사용 후 즉시 삭제

        ↓

[4] Session Key Derivation
    │
    • HKDF로 세션 키 유도
    • 4개의 독립 키:
    │  - c2s 암호화 키
    │  - c2s 인증 키
    │  - s2c 암호화 키
    │  - s2c 인증 키
    │
    • Lifetime: 세션 지속 시간 (기본 24시간)

        ↓

[5] Key Usage (암호화 통신)
    │
    • ChaCha20-Poly1305로 메시지 암호화
    • 각 메시지마다 고유 nonce
    • Sequence number로 재생 공격 방지

        ↓

[6] Key Rotation (선택적)
    │
    • 주기적으로 세션 재협상
    • 새 임시 키로 핸드셰이크
    • 이전 세션 폐기
    • Forward Secrecy 보장

        ↓

[7] Key Revocation (필요시)
    │
    • 개인키 유출 의심 시
    • 블록체인에 revocation 트랜잭션
    • 새 키 쌍 생성 및 등록
    • DID Document 업데이트

        ↓

[8] Key Cleanup
    │
    • 세션 종료 시 메모리 삭제
    • 로그에 키 정보 남기지 않음
    • 안전한 메모리 소거 (zeroing)
```

### 3.2 세션 생명주기

```
┌─────────────────────────────────────────────────────────┐
│  Session Lifecycle                                      │
└─────────────────────────────────────────────────────────┘

Time: T=0
    ↓
[Creation] 핸드셰이크 완료
    │
    • NewSecureSession() 호출
    • Session ID 생성
    • 4개 방향별 키 유도
    • Manager에 등록
    │
    State: ACTIVE
    Expiry: T + 24h (기본값)

        ↓ (사용)

Time: T=1h
    ↓
[Active Communication]
    │
    • 메시지 암호화/복호화
    • lastUsed 타임스탬프 업데이트
    • seqNumber 증가
    │
    State: ACTIVE

        ↓ (계속 사용)

Time: T=12h
    ↓
[Health Check] (선택적)
    │
    • Session Manager가 주기적으로 체크
    • 만료 임박 세션 감지
    • 경고 로그 출력
    │
    if (now + 1h > expiry) {
        log.Warn("Session expiring soon")
        // 선택적: 자동 갱신 트리거
    }

        ↓

Time: T=23h
    ↓
[Renewal] (선택적)
    │
    • 새 핸드셰이크 시작
    • 새 Session ID로 교체
    • 이전 세션은 잠시 유지 (graceful transition)
    │
    State: RENEWING

        ↓

Time: T=24h
    ↓
[Expiration]
    │
    • 자동 만료
    • Session Manager가 제거
    • 메모리에서 키 삭제
    • 이벤트 발생: SessionExpired
    │
    State: EXPIRED

        ↓

[Cleanup]
    │
    • 메모리 영점화 (zero out keys)
    • 관련 리소스 해제
    • 통계 업데이트
```

**Session Manager의 자동 정리 로직:**

```go
// session/manager.go:200-245

func (m *Manager) StartCleanupRoutine(interval time.Duration) {
    ticker := time.NewTicker(interval)  // 기본: 10분
    go func() {
        for range ticker.C {
            m.cleanupExpiredSessions()
        }
    }()
}

func (m *Manager) cleanupExpiredSessions() {
    now := time.Now()

    m.mu.Lock()
    defer m.mu.Unlock()

    for sessionID, session := range m.sessions {
        // 만료 체크
        if now.After(session.ExpiresAt) {
            // 1. 이벤트 발생
            if m.events != nil {
                m.events.OnSessionExpired(session)
            }

            // 2. 메모리에서 제거
            delete(m.sessions, sessionID)
            delete(m.sessionsByKeyID, session.KeyID)

            // 3. 통계 업데이트
            m.stats.ExpiredSessions++

            // 4. 로그
            log.Info("Session expired and cleaned up",
                "sessionID", sessionID,
                "peerDID", session.RemoteDID,
            )
        }
    }
}
```

---

## 4. 블록체인 레이어와 애플리케이션 레이어 통합

### 4.1 레이어 간 인터페이스

```
┌─────────────────────────────────────────────────────────┐
│  Application Layer                                      │
│  ┌───────────────────────────────────────────────────┐ │
│  │  AI Agent (예: ChatGPT Plugin)                    │ │
│  │                                                   │ │
│  │  func HandleUserMessage(msg string) {            │ │
│  │      // 1. SAGE 세션 확인                         │ │
│  │      session := sage.GetSession(peerDID)         │ │
│  │                                                   │ │
│  │      // 2. 메시지 암호화                          │ │
│  │      encrypted := session.Encrypt(msg)           │ │
│  │                                                   │ │
│  │      // 3. 전송                                   │ │
│  │      Send(encrypted)                             │ │
│  │  }                                                │ │
│  └───────────────────────────────────────────────────┘ │
└────────────────────────┬────────────────────────────────┘
                         │ SAGE SDK API
                         ↓
┌─────────────────────────────────────────────────────────┐
│  SAGE SDK Layer                                         │
│  ┌───────────────────────────────────────────────────┐ │
│  │  High-Level API                                   │ │
│  │                                                   │ │
│  │  type SAGEClient struct {                        │ │
│  │      didManager     *did.Manager                 │ │
│  │      sessionManager *session.Manager             │ │
│  │      handshake      *handshake.Client            │ │
│  │  }                                                │ │
│  │                                                   │ │
│  │  func (c *SAGEClient) SecureConnect(             │ │
│  │      peerDID string,                             │ │
│  │  ) (*SecureSession, error)                       │ │
│  └───────────────────────────────────────────────────┘ │
└────────────────────────┬────────────────────────────────┘
                         │
      ┌──────────────────┼──────────────────┐
      │                  │                  │
      ↓                  ↓                  ↓
┌──────────┐      ┌──────────┐      ┌──────────┐
│   DID    │      │Handshake │      │ Session  │
│ Manager  │      │ Protocol │      │ Manager  │
└────┬─────┘      └──────────┘      └──────────┘
     │
     │ Blockchain Access
     ↓
┌─────────────────────────────────────────────────────────┐
│  Blockchain Abstraction Layer                           │
│  ┌───────────────────────────────────────────────────┐ │
│  │  type BlockchainClient interface {               │ │
│  │      RegisterDID(...)                             │ │
│  │      ResolveDID(...)                              │ │
│  │      UpdateDID(...)                               │ │
│  │  }                                                │ │
│  └───────────────────────────────────────────────────┘ │
└────────────────────────┬────────────────────────────────┘
                         │
      ┌──────────────────┼──────────────────┐
      │                  │                  │
      ↓                  ↓                  ↓
┌──────────┐      ┌──────────┐      ┌──────────┐
│ Ethereum │      │   Kaia   │      │ Solana   │
│  Client  │      │  Client  │      │  Client  │
└────┬─────┘      └────┬─────┘      └────┬─────┘
     │                 │                 │
     │                 │                 │
     ↓                 ↓                 ↓
┌─────────────────────────────────────────────────────────┐
│  Blockchain Networks                                    │
│  • Ethereum Mainnet/Sepolia                            │
│  • Kaia Mainnet/Kairos                                 │
│  • Solana Mainnet/Devnet                               │
└─────────────────────────────────────────────────────────┘
```

### 4.2 실전 통합 예시

#### 예시 1: Express.js 백엔드 통합

```typescript
// server.ts

import express from 'express';
import { SAGEClient } from '@sage-x-project/sdk';

const app = express();
const sage = new SAGEClient({
    did: 'did:sage:kaia:MyServerAgent',
    keyPath: './keys/server.jwk',
    blockchain: {
        chain: 'kaia',
        rpcUrl: 'https://public-en-kairos.node.kaia.io',
        contractAddress: '0x1234...',
    },
});

// 초기화
await sage.initialize();

// API 엔드포인트
app.post('/api/secure-message', async (req, res) => {
    const { peerDID, message } = req.body;

    try {
        // 1. 세션 가져오기 (없으면 자동 핸드셰이크)
        const session = await sage.getOrCreateSession(peerDID);

        // 2. 메시지 암호화 및 전송
        const encrypted = await session.sendMessage(message);

        res.json({
            success: true,
            sessionID: session.id,
            messageID: encrypted.id,
        });
    } catch (error) {
        res.status(500).json({
            success: false,
            error: error.message,
        });
    }
});

// 수신 메시지 핸들러
sage.on('message', async (msg) => {
    console.log(`Received from ${msg.senderDID}:`, msg.plaintext);

    // 비즈니스 로직 처리
    const response = await processMessage(msg.plaintext);

    // 응답 전송
    const session = await sage.getSession(msg.sessionID);
    await session.sendMessage(response);
});

app.listen(3000);
```

#### 예시 2: Python AI Agent 통합

```python
# ai_agent.py

from sage_sdk import SAGEClient, SecureSession
import openai

class SecureAIAgent:
    def __init__(self, did: str, key_path: str):
        self.sage = SAGEClient(
            did=did,
            key_path=key_path,
            blockchain={
                'chain': 'kaia',
                'rpc_url': 'https://public-en-kairos.node.kaia.io',
                'contract_address': '0x1234...',
            }
        )
        self.sage.initialize()

    async def handle_user_request(self, user_did: str, request: str):
        """사용자 요청을 안전하게 처리"""

        # 1. SAGE 세션 확보
        session = await self.sage.get_or_create_session(user_did)

        # 2. 요청 복호화 (이미 복호화된 상태로 수신됨)
        # request는 이미 plaintext

        # 3. AI 모델로 처리
        response = openai.ChatCompletion.create(
            model="gpt-4",
            messages=[
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": request}
            ]
        )

        answer = response.choices[0].message.content

        # 4. 응답 암호화 및 전송
        encrypted = await session.send_message(answer)

        return {
            'session_id': session.id,
            'message_id': encrypted.id,
            'status': 'sent'
        }

    def start(self):
        """에이전트 시작"""

        # 메시지 수신 핸들러 등록
        @self.sage.on_message
        async def on_message(msg):
            print(f"📨 Message from {msg.sender_did}")
            response = await self.handle_user_request(
                msg.sender_did,
                msg.plaintext
            )
            print(f"Yes Response sent: {response['message_id']}")

        # 세션 이벤트 핸들러
        @self.sage.on_session_created
        async def on_session_created(session):
            print(f"🔐 Secure session created with {session.peer_did}")

        # 시작
        self.sage.start()
        print("🤖 AI Agent is running...")

# 실행
if __name__ == '__main__':
    agent = SecureAIAgent(
        did='did:sage:kaia:MyAIAgent',
        key_path='./keys/agent.jwk'
    )
    agent.start()
```

---

## 5. 에러 처리 및 복구 플로우

### 5.1 일반적인 에러 시나리오

#### 시나리오 1: 핸드셰이크 실패

```
Agent A → Invitation → Agent B
                         ↓
                    (서명 검증 실패)
                         ↓
                    ← Rejection

┌─────────────────────────────────────────────────────────┐
│  에러 처리 플로우                                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Agent B가 에러 응답 전송                             │
│     response := &HandshakeError{                        │
│         Code:    "INVALID_SIGNATURE",                   │
│         Message: "Signature verification failed",       │
│         Retry:   true,  // 재시도 가능                  │
│     }                                                   │
│                                                         │
│  2. Agent A가 에러 수신                                  │
│     if error.Retry {                                    │
│         // DID Document 재조회 (키가 변경되었을 수 있음) │
│         doc := resolver.Resolve(peerDID)                │
│         // 재시도                                        │
│         retry()                                         │
│     } else {                                            │
│         return error                                    │
│     }                                                   │
│                                                         │
│  3. 로깅 및 모니터링                                     │
│     metrics.IncrementHandshakeFailure(                  │
│         "invalid_signature",                            │
│         peerDID,                                        │
│     )                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 시나리오 2: 네트워크 타임아웃

```
Agent A → Request → [Network Timeout] × Agent B
    ↓
(재시도 로직)

┌─────────────────────────────────────────────────────────┐
│  Exponential Backoff 재시도                              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  attempt := 0                                           │
│  maxRetries := 3                                        │
│                                                         │
│  for attempt < maxRetries {                             │
│      err := sendMessage(msg)                            │
│      if err == nil {                                    │
│          return success                                 │
│      }                                                   │
│                                                         │
│      if !isRetriable(err) {                             │
│          return error  // 재시도 불가능                  │
│      }                                                   │
│                                                         │
│      // Exponential backoff                             │
│      delay := time.Second * (1 << attempt)              │
│      // 1초, 2초, 4초                                    │
│                                                         │
│      time.Sleep(delay + jitter())                       │
│      attempt++                                          │
│  }                                                       │
│                                                         │
│  return error("Max retries exceeded")                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 시나리오 3: 세션 만료

```
Agent A → Encrypted Message → Agent B
                                 ↓
                         (세션 ID 없음)
                                 ↓
                         ← SessionExpired Error

┌─────────────────────────────────────────────────────────┐
│  세션 재협상                                             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Agent A가 SessionExpired 수신                        │
│     if error.Code == "SESSION_EXPIRED" {                │
│         // 로컬 세션도 제거                              │
│         sessionManager.RemoveSession(sessionID)         │
│                                                         │
│         // 새 핸드셰이크 시작                            │
│         newSession := handshake.InitiateHandshake(      │
│             peerDID,                                    │
│         )                                               │
│                                                         │
│         // 원래 메시지 재전송                            │
│         newSession.SendMessage(originalMessage)         │
│     }                                                   │
│                                                         │
│  2. 자동 갱신 (proactive)                                │
│     // 만료 1시간 전에 자동 갱신                          │
│     if session.ExpiresAt - now < 1*time.Hour {          │
│         initiateRenewal(session)                        │
│     }                                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 5.2 재해 복구 (Disaster Recovery)

```
┌─────────────────────────────────────────────────────────┐
│  Backup & Recovery Strategy                             │
└─────────────────────────────────────────────────────────┘

[1] Key Backup
    │
    ├─ Primary: 암호화된 로컬 파일
    │  • AES-256-GCM 암호화
    │  • Password-based KDF
    │  • 위치: ./keys/
    │
    ├─ Secondary: Cloud Backup (선택적)
    │  • AWS Secrets Manager
    │  • Google Cloud KMS
    │  • Vault by HashiCorp
    │
    └─ Tertiary: 하드웨어 백업
       • USB 키
       • Hardware Security Module (HSM)
       • Paper wallet (BIP39 mnemonic)

[2] Session State Recovery
    │
    ├─ 세션은 복구 불가능 (설계상)
    │  • 임시 키 기반
    │  • Forward Secrecy 보장
    │
    └─ 대신: 새 핸드셰이크
       • 자동으로 새 세션 생성
       • 사용자는 인지 못 함 (transparent)

[3] Blockchain Data Recovery
    │
    ├─ DID 정보는 블록체인에서 복구
    │  • 영구 저장
    │  • 언제든 재조회 가능
    │
    └─ 로컬 캐시 재구축
       • DID Resolution으로 재구축
       • 시간이 걸릴 수 있음 (수 분)

[4] Application State Recovery
    │
    ├─ 메시지 히스토리
    │  • 애플리케이션 레벨 백업
    │  • 데이터베이스 스냅샷
    │
    └─ Configuration
       • config.yaml 백업
       • 환경 변수 문서화
```

---

## 6. 타이밍 다이어그램

### 6.1 정상 플로우 타이밍

```
┌─────────────────────────────────────────────────────────┐
│  Complete Agent Communication Timeline                  │
│  (Total: ~2-3 seconds for first message)                │
└─────────────────────────────────────────────────────────┘

T=0ms
  │
  │  [Agent A] User 요청 수신
  │  "Agent B에게 메시지 전송"
  │
  ↓
T=10ms
  │
  │  [Agent A] DID Resolution 시작
  │  Resolver.Resolve("did:sage:kaia:AgentB")
  │
  ├─ L1 Cache miss
  ├─ L2 Cache miss
  └─ Blockchain query
  │
  ↓
T=210ms (+200ms - RPC call)
  │
  │  [Agent A] DID Document 수신
  │  • Agent B 공개키: 0x4f3e...
  │  • Endpoint: https://agent-b.com
  │  • Cache 저장
  │
  ↓
T=220ms
  │
  │  [Agent A] Handshake - Invitation 생성
  │  • X25519 임시 키 생성 (1ms)
  │  • Invitation 메시지 구성 (1ms)
  │  • Ed25519 서명 (5ms)
  │  • gRPC 전송 시작
  │
  ↓
T=250ms (+30ms - 네트워크 전송)
  │
  │  [Agent B] Invitation 수신
  │  • 서명 검증 (5ms)
  │  • Nonce 체크 (1ms)
  │  • 임시 키 생성 (1ms)
  │
  ↓
T=257ms
  │
  │  [Agent B] Handshake - Request 생성
  │  • HPKE 암호화 (3ms)
  │  • 응답 전송
  │
  ↓
T=287ms (+30ms - 네트워크 전송)
  │
  │  [Agent A] Request 수신
  │  • HPKE 복호화 (3ms)
  │  • Response 생성 (2ms)
  │  • 전송
  │
  ↓
T=322ms (+30ms - 네트워크)
  │
  │  [Agent B] Response 수신
  │  • 검증 (2ms)
  │  • Complete 전송
  │
  ↓
T=354ms (+30ms - 네트워크)
  │
  │  [Agent A] Complete 수신
  │  • 세션 생성 시작
  │  • 공유 비밀 계산 (X25519 DH) - 1ms
  │  • HKDF 키 유도 - 2ms
  │  • Session ID 생성 - 1ms
  │  • 세션 저장
  │
  ↓
T=358ms
  │
  │  [Agent A & B] 세션 확립 완료! 🎉
  │  Session ID: "7vH3Jq9KmN2p..."
  │
  ↓
T=360ms
  │
  │  [Agent A] 원래 메시지 암호화
  │  • Plaintext: "Hello, Agent B!"
  │  • ChaCha20-Poly1305 암호화 (1ms)
  │  • 전송
  │
  ↓
T=390ms (+30ms - 네트워크)
  │
  │  [Agent B] 암호화 메시지 수신
  │  • Sequence number 체크 (0.1ms)
  │  • ChaCha20-Poly1305 복호화 (1ms)
  │  • Poly1305 MAC 검증 (0.5ms)
  │  • Plaintext 추출: "Hello, Agent B!"
  │
  ↓
T=392ms
  │
  │  [Agent B] 응답 처리
  │  • 비즈니스 로직 실행 (varies)
  │  • 응답 메시지 암호화 (1ms)
  │  • 전송
  │
  ↓
T=422ms (+30ms - 네트워크)
  │
  │  [Agent A] 응답 수신
  │  • 복호화 (1ms)
  │  • User에게 반환
  │
  ↓
T=423ms
  │
  │  Yes 완료!
  │  Total: 423ms (첫 번째 메시지)
  │
  ↓
  │
  │  [이후 메시지들]
  │  • DID Resolution: 캐시에서 즉시 (< 1ms)
  │  • 핸드셰이크: 생략 (기존 세션 사용)
  │  • 암호화/복호화만: ~60ms
  │
  ↓
```

### 6.2 에러 시나리오 타이밍

```
┌─────────────────────────────────────────────────────────┐
│  Error Recovery Timeline                                │
└─────────────────────────────────────────────────────────┘

T=0ms
  │
  │  [Agent A] 메시지 전송 시도
  │
  ↓
T=210ms
  │
  │  [Network] Timeout! ⏱️
  │  (Agent B 응답 없음)
  │
  ↓
T=210ms
  │
  │  [Agent A] Retry 로직 시작
  │  • Attempt 1 실패 감지
  │  • Backoff: 1초 대기
  │
  ↓
T=1210ms (+1초)
  │
  │  [Agent A] Retry Attempt 2
  │  • 메시지 재전송
  │
  ↓
T=1240ms (+30ms)
  │
  │  [Agent B] 메시지 수신
  │  (이번에는 성공)
  │  • 정상 처리
  │
  ↓
T=1270ms
  │
  │  Yes 복구 완료
  │  Total delay: 1270ms (원래 210ms + retry 1060ms)
  │
```

---

## 결론

Part 6A에서는 SAGE의 완전한 데이터 플로우를 다루었습니다:

### 핵심 내용 요약

1. **전체 시스템 레이어**
   - Application → SAGE SDK → Crypto → DID → Blockchain
   - 각 레이어의 역할과 상호작용

2. **완전한 통신 플로우**
   - Phase 1: 에이전트 준비 (키 생성, DID 생성)
   - Phase 2: 블록체인 등록 (서명, 트랜잭션, 확인)
   - Phase 3: 에이전트 검색 (DID Resolution, 캐싱)
   - Phase 4: 핸드셰이크 (4단계 프로토콜)
   - Phase 5: 암호화 통신 (세션 기반)

3. **생명주기 관리**
   - 키 생명주기 (생성 → 사용 → 교체 → 폐기)
   - 세션 생명주기 (생성 → 활성 → 갱신 → 만료)

4. **레이어 통합**
   - 블록체인과 애플리케이션의 통합
   - 실전 예시 (Node.js, Python)

5. **에러 처리**
   - 핸드셰이크 실패 처리
   - 네트워크 타임아웃 재시도
   - 세션 만료 복구
   - 재해 복구 전략

6. **성능 최적화**
   - 다단계 캐싱으로 200배 성능 향상
   - 타이밍 분석 및 최적화 지점

### 다음 단계

**Part 6B**에서 다룰 내용:
- 실제 프로젝트에 SAGE 통합하는 단계별 가이드
- CLI 도구 사용법
- SDK 통합 예제
- MCP (Model Context Protocol) 통합
- 프로덕션 배포 체크리스트

---

**문서 정보**
- 작성일: 2025-01-15
- 버전: 1.0
- Part: 6A/6C
- 이전: [Part 5 - Smart Contracts and On-Chain Registry](DETAILED_GUIDE_PART5_KO.md)
- 다음: [Part 6B - Practical Integration Guide](DETAILED_GUIDE_PART6B_KO.md)
