# SAGE 프로젝트 상세 가이드 - Part 3: DID 및 블록체인 통합

> **대상 독자**: 프로그래밍 초급자부터 중급 개발자까지
> **작성일**: 2025-10-07
> **버전**: 1.0
> **이전**: [Part 2 - 암호화 시스템](./DETAILED_GUIDE_PART2_KO.md)

---

## 목차

1. [DID (Decentralized Identifier) 심층 분석](#1-did-decentralized-identifier-심층-분석)
2. [블록체인 선택과 다중 체인 전략](#2-블록체인-선택과-다중-체인-전략)
3. [Ethereum 통합](#3-ethereum-통합)
4. [DID 등록 프로세스](#4-did-등록-프로세스)
5. [DID 조회 및 해석](#5-did-조회-및-해석)
6. [DID 업데이트 및 비활성화](#6-did-업데이트-및-비활성화)
7. [캐싱 및 성능 최적화](#7-캐싱-및-성능-최적화)
8. [다중 체인 관리](#8-다중-체인-관리)
9. [실전 예제](#9-실전-예제)

---

## 1. DID (Decentralized Identifier) 심층 분석

### 1.1 DID의 필요성

**전통적인 신원 관리의 문제점**

```
중앙화된 신원 시스템:

┌─────────────────────────────────────┐
│   중앙 서버 (예: OAuth Provider)     │
│   - Facebook Login                  │
│   - Google Sign-In                  │
│   - GitHub OAuth                    │
└──────────────┬──────────────────────┘
               │
        문제점들:
        1. 단일 장애점 (Single Point of Failure)
           서버 다운 → 모든 사용자 로그인 불가

        2. 검열 위험
           계정 정지 → 연결된 모든 서비스 접근 불가

        3. 프라이버시 침해
           중앙 기관이 모든 활동 추적 가능

        4. 벤더 락인 (Vendor Lock-in)
           플랫폼 변경 시 신원 이전 불가

        5. 데이터 소유권 문제
           사용자가 자신의 데이터 통제 불가
               ↓
        [앱 1] [앱 2] [앱 3] ...
```

**DID의 해결 방법**

```
탈중앙화된 DID 시스템:

┌─────────────────────────────────────┐
│   블록체인 (Immutable Ledger)        │
│   - 변조 불가능                       │
│   - 24/7 가용성                      │
│   - 검열 저항성                       │
│   - 글로벌 접근                       │
└──────────────┬──────────────────────┘
               │
        장점들:
        Yes 자기 주권 신원 (Self-Sovereign Identity)
           개인키 소유자만 신원 통제

        Yes 영구성 (Permanence)
           블록체인에 영구 기록

        Yes 상호 운용성 (Interoperability)
           모든 플랫폼에서 동일 DID 사용

        Yes 프라이버시 보호
           필요한 정보만 선택적 공개

        Yes 검증 가능 (Verifiable)
           누구나 독립적으로 검증 가능
               ↓
        [앱 1] [앱 2] [앱 3] ...
```

### 1.2 W3C DID 표준

**DID 구조 (RFC 3986)**

```
DID Syntax:
did:method:method-specific-id

예시:
did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc454e4438f44e

파싱:
┌─────────────────────────────────────────────────────────┐
│ scheme    │ method  │ method-specific-id                │
├───────────┼─────────┼───────────────────────────────────┤
│ did       │ sage    │ ethereum:0x742d35Cc6634C05...     │
└─────────────────────────────────────────────────────────┘

상세 분해:
- scheme: "did" (고정)
- method: "sage" (SAGE 시스템)
- network: "ethereum" (블록체인 네트워크)
- address: "0x742d35Cc..." (Ethereum 주소)
```

**DID 문서 (DID Document)**

```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://w3id.org/security/suites/ed25519-2020/v1"
  ],
  "id": "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "controller": "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
  "verificationMethod": [
    {
      "id": "did:sage:ethereum:0x742d35Cc...#keys-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:sage:ethereum:0x742d35Cc...",
      "publicKeyMultibase": "zH3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV"
    }
  ],
  "authentication": ["did:sage:ethereum:0x742d35Cc...#keys-1"],
  "service": [
    {
      "id": "did:sage:ethereum:0x742d35Cc...#agent-endpoint",
      "type": "AgentService",
      "serviceEndpoint": "https://agent.example.com/api"
    }
  ]
}
```

**각 필드 설명**:

```
@context:
- DID 문서의 버전과 스펙 정의
- JSON-LD 컨텍스트

id:
- DID 자체
- 이 문서가 설명하는 주체

controller:
- DID를 통제하는 주체
- 보통 자기 자신

verificationMethod:
- 검증에 사용할 수 있는 암호화 재료
- 공개키, 인증서 등
- id: 검증 방법의 고유 식별자
- type: 키 타입 (Ed25519, Secp256k1 등)
- publicKeyMultibase: 공개키 (multibase 인코딩)

authentication:
- 인증에 사용할 검증 방법 참조
- DID 소유자임을 증명하는 데 사용

service:
- DID 주체와 상호작용할 서비스 엔드포인트
- 예: AI 에이전트 API, 메시징 서비스 등
```

### 1.3 SAGE DID 메소드 스펙

**메소드 정의**

```
Method Name: sage

Method Specific Identifier Format:
did:sage:<network>:<address>

Supported Networks:
- ethereum: Ethereum Mainnet
- sepolia: Ethereum Sepolia Testnet
- kaia: Kaia Mainnet (Cypress)
- kairos: Kaia Testnet
- solana: Solana Mainnet (planned)

Address Format:
- Ethereum/Kaia: 0x + 40 hex characters
- Solana: Base58 encoded (32 bytes)

Examples:
did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc454e4438f44e
did:sage:sepolia:0x1234567890123456789012345678901234567890
did:sage:kaia:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd
did:sage:solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d
```

**CRUD 작업**

```
Create (생성):
- 스마트 컨트랙트의 registerAgent() 호출
- 파라미터: name, endpoint, publicKey, signature
- 결과: 블록체인에 DID 문서 저장

Read (조회):
- 스마트 컨트랙트의 getAgent() 또는 agentsByDID 매핑 조회
- 파라미터: DID 또는 Ethereum 주소
- 결과: DID 문서 반환

Update (업데이트):
- updateAgent() 호출
- 소유자만 가능
- 업데이트 가능 필드: name, description, endpoint, capabilities

Deactivate (비활성화):
- deactivateAgent() 호출
- 소유자만 가능
- active 플래그를 false로 설정
- 완전 삭제는 불가능 (블록체인 불변성)
```

### 1.4 온체인 데이터 구조

**SAGE 스마트 컨트랙트의 AgentMetadata**

```solidity
struct AgentMetadata {
    string did;              // DID 문자열
    address owner;           // 소유자 주소
    bytes publicKey;         // Ed25519 공개키 (32바이트)
    string name;             // 에이전트 이름
    string description;      // 설명
    string endpoint;         // API 엔드포인트 URL
    string capabilities;     // JSON 배열 문자열
    bool active;             // 활성 상태
    uint256 createdAt;       // 생성 타임스탬프
    uint256 updatedAt;       // 업데이트 타임스탬프
}
```

**저장소 매핑**

```solidity
contract SageRegistryV2 {
    // 주요 저장소
    mapping(bytes32 => AgentMetadata) private agents;
    // agentId (keccak256(did)) → AgentMetadata

    mapping(string => bytes32) private didToAgentId;
    // did 문자열 → agentId

    mapping(address => bytes32[]) private ownerToAgents;
    // owner 주소 → agentId 배열 (한 주소가 여러 에이전트 소유 가능)

    mapping(bytes32 => uint256) private agentNonce;
    // agentId → nonce (replay 공격 방지)

    // 키 검증 관련
    mapping(bytes32 => KeyValidation) private keyValidations;
    // keyHash → KeyValidation

    mapping(address => bytes32) private addressToKeyHash;
    // address → keyHash (키 재사용 방지)
}
```

**저장소 접근 패턴**

```
1. DID로 조회:
   did → didToAgentId[did] → agentId
       → agents[agentId] → AgentMetadata

2. 주소로 조회:
   address → ownerToAgents[address] → agentId[]
          → agents[agentId] → AgentMetadata[]

3. agentId로 직접 조회:
   agentId → agents[agentId] → AgentMetadata

시간 복잡도:
- DID 조회: O(1)
- 주소로 조회: O(n), n = 해당 주소의 에이전트 수
- agentId 조회: O(1)

가스 비용:
- DID 조회: ~30,000 gas (읽기 전용)
- 등록: ~620,000 gas
- 업데이트: ~80,000 gas
```

---

## 2. 블록체인 선택과 다중 체인 전략

### 2.1 지원 블록체인 비교

| 특성                | Ethereum             | Kaia                    | Solana                |
| ------------------- | -------------------- | ----------------------- | --------------------- |
| **합의 알고리즘**   | PoS (Proof of Stake) | PoS                     | PoH + PoS             |
| **블록 시간**       | ~12초                | ~1초                    | ~400ms                |
| **TPS**             | ~30                  | ~4,000                  | ~65,000               |
| **완결성**          | 2 epochs (~13분)     | 즉시                    | ~1초                  |
| **가스 비용**       | 높음 ($5-50)         | 매우 낮음 ($0.001-0.01) | 낮음 ($0.00001-0.001) |
| **스마트 컨트랙트** | Solidity             | Solidity                | Rust/C                |
| **에코시스템**      | 가장 큼              | 중간 (한국 중심)        | 빠르게 성장           |
| **개발 도구**       | Hardhat, Foundry     | Hardhat, Foundry        | Anchor                |
| **지갑 지원**       | MetaMask 등 많음     | Kaikas, MetaMask        | Phantom, Solflare     |
| **SAGE 상태**       | Yes 완전 지원         | Yes 완전 지원            |  개발 중            |

### 2.2 Ethereum 선택 이유

**장점**:

```
1. 보안성 및 탈중앙화
   - 수천 개의 검증자 노드
   - 높은 보안 예산 (스테이킹: $수백억)
   - 오랜 역사 (2015년~)

2. 성숙한 에코시스템
   - 풍부한 개발 도구
   - 대규모 개발자 커뮤니티
   - 검증된 스마트 컨트랙트 패턴

3. 상호 운용성
   - 대부분의 DApp이 Ethereum 지원
   - 크로스 체인 브릿지 많음
   - ENS (Ethereum Name Service) 통합

4. 신뢰와 인지도
   - 기관 투자자 신뢰
   - 규제 명확성
   - 글로벌 표준
```

**단점 및 대응**:

```
1. 높은 가스 비용
   대응: Layer 2 솔루션 계획 (Arbitrum, Optimism)

2. 느린 확정 시간
   대응: 낙관적 업데이트 + 백그라운드 확인

3. 제한된 TPS
   대응: 배치 처리, 오프체인 인덱싱
```

### 2.3 Kaia 추가 이유

**Kaia (구 Klaytn)의 특징**:

```
1. 한국 시장 특화
   - Kakao, LG 등 대기업 참여
   - 한국 사용자 친화적
   - 원화 연동 서비스

2. 빠른 속도 + 낮은 비용
   - 1초 블록 시간
   - $0.001 정도의 낮은 트랜잭션 비용
   - 즉시 확정 (Instant Finality)

3. Ethereum 호환
   - EVM 호환 (Solidity 사용)
   - Ethereum 도구 그대로 사용 가능
   - 마이그레이션 용이

4. 기업 친화적
   - 서비스 체인 (프라이빗 체인)
   - 기업 지원 프로그램
   - 규제 준수
```

**SAGE의 Kaia 활용**:

```
시나리오 1: 프로덕션 배포
- 메인 DID 레지스트리: Ethereum
- 빠른 업데이트: Kaia
- 크로스 체인 검증으로 이중 보안

시나리오 2: 한국 시장 공략
- 한국 사용자를 위한 Kaia 우선
- 글로벌 확장 시 Ethereum 추가

시나리오 3: 개발/테스트
- Kairos (테스트넷)에서 무료 테스트
- Sepolia보다 빠른 피드백
```

### 2.4 다중 체인 아키텍처

**계층 구조**:

```
┌─────────────────────────────────────────────────────┐
│              Application Layer                       │
│           (AI Agent Applications)                    │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────┴───────────────────────────────┐
│              DID Manager (다중 체인 추상화)           │
│  - 통일된 API                                        │
│  - 체인 선택 로직                                    │
│  - 캐싱 및 동기화                                    │
└────┬────────────────┬────────────────┬──────────────┘
     │                │                │
┌────┴──────┐  ┌──────┴──────┐  ┌─────┴────────┐
│ Ethereum  │  │    Kaia     │  │   Solana     │
│ Resolver  │  │  Resolver   │  │  Resolver    │
└────┬──────┘  └──────┬──────┘  └─────┬────────┘
     │                │                │
┌────┴──────┐  ┌──────┴──────┐  ┌─────┴────────┐
│ Ethereum  │  │    Kaia     │  │   Solana     │
│ Mainnet   │  │  Mainnet    │  │  Mainnet     │
└───────────┘  └─────────────┘  └──────────────┘
```

**체인 선택 전략**:

```go
// did/manager.go

func (m *Manager) selectChain(did AgentDID) (Chain, error) {
    // 1. DID에서 체인 파싱
    chain, _, err := ParseDID(did)
    if err != nil {
        return "", err
    }

    // 2. 체인 설정 확인
    if !m.IsChainConfigured(chain) {
        return "", fmt.Errorf("chain not configured: %s", chain)
    }

    return chain, nil
}

// 자동 폴백 (Fallback) 전략
func (m *Manager) resolveWithFallback(did AgentDID) (*AgentMetadata, error) {
    chain, _, _ := ParseDID(did)

    // 1차 시도
    metadata, err := m.resolver.Resolve(ctx, did)
    if err == nil {
        return metadata, nil
    }

    // 2차: 캐시 확인
    if cached, ok := m.cache.Get(string(did)); ok {
        log.Warn("Using cached DID (chain unavailable)")
        return cached.(*AgentMetadata), nil
    }

    // 3차: 대체 체인 시도 (설정된 경우)
    if fallbackChain := m.getFallbackChain(chain); fallbackChain != "" {
        fallbackDID := convertDID(did, fallbackChain)
        return m.resolver.Resolve(ctx, fallbackDID)
    }

    return nil, err
}
```

---

## 3. Ethereum 통합

### 3.1 Ethereum 클라이언트 구현

**EthereumClient 구조**:

```go
// did/ethereum/client.go

type EthereumClient struct {
    client          *ethclient.Client      // geth 클라이언트
    contract        *bind.BoundContract    // 컨트랙트 바인딩
    contractAddr    common.Address         // 컨트랙트 주소
    chainID         *big.Int               // 체인 ID

    // 트랜잭션 관리
    txOpts          *bind.TransactOpts     // 트랜잭션 옵션
    gasPrice        *big.Int               // 가스 가격
    gasPriceOracle  GasPriceOracle         // 동적 가스 가격

    // 캐싱
    cache           *lru.Cache             // LRU 캐시
    cacheTTL        time.Duration          // 캐시 TTL

    mu              sync.RWMutex           // 동시성 제어
}
```

**초기화**:

```go
func NewEthereumClient(
    rpcURL string,
    contractAddr string,
    privateKey string,
) (*EthereumClient, error) {
    // 1. RPC 클라이언트 연결
    client, err := ethclient.Dial(rpcURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    // 2. 체인 ID 확인
    chainID, err := client.ChainID(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to get chain ID: %w", err)
    }

    // 3. 개인키 로드
    key, err := crypto.HexToECDSA(privateKey)
    if err != nil {
        return nil, fmt.Errorf("invalid private key: %w", err)
    }

    // 4. 트랜잭션 서명자 생성
    auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
    if err != nil {
        return nil, fmt.Errorf("failed to create transactor: %w", err)
    }

    // 5. 가스 가격 설정
    gasPrice, err := client.SuggestGasPrice(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to get gas price: %w", err)
    }
    auth.GasPrice = gasPrice

    // 6. 컨트랙트 바인딩
    addr := common.HexToAddress(contractAddr)
    contract := bind.NewBoundContract(
        addr,
        parseABI(),  // ABI 파싱
        client,
        client,
        client,
    )

    // 7. 캐시 초기화
    cache, _ := lru.New(1000)  // 최대 1000개 항목

    return &EthereumClient{
        client:       client,
        contract:     contract,
        contractAddr: addr,
        chainID:      chainID,
        txOpts:       auth,
        gasPrice:     gasPrice,
        cache:        cache,
        cacheTTL:     5 * time.Minute,
    }, nil
}

위치: did/ethereum/client.go
```

### 3.2 스마트 컨트랙트 ABI 바인딩

**ABI (Application Binary Interface)**:

```go
// did/ethereum/abi.go

const SageRegistryV2ABI = `[
    {
        "type": "function",
        "name": "registerAgent",
        "inputs": [
            {"name": "did", "type": "string"},
            {"name": "name", "type": "string"},
            {"name": "description", "type": "string"},
            {"name": "endpoint", "type": "string"},
            {"name": "publicKey", "type": "bytes"},
            {"name": "capabilities", "type": "string"},
            {"name": "signature", "type": "bytes"}
        ],
        "outputs": [
            {"name": "", "type": "bytes32"}
        ],
        "stateMutability": "nonpayable"
    },
    {
        "type": "function",
        "name": "getAgent",
        "inputs": [
            {"name": "agentId", "type": "bytes32"}
        ],
        "outputs": [
            {
                "name": "",
                "type": "tuple",
                "components": [
                    {"name": "did", "type": "string"},
                    {"name": "owner", "type": "address"},
                    {"name": "publicKey", "type": "bytes"},
                    {"name": "name", "type": "string"},
                    {"name": "description", "type": "string"},
                    {"name": "endpoint", "type": "string"},
                    {"name": "capabilities", "type": "string"},
                    {"name": "active", "type": "bool"},
                    {"name": "createdAt", "type": "uint256"},
                    {"name": "updatedAt", "type": "uint256"}
                ]
            }
        ],
        "stateMutability": "view"
    },
    {
        "type": "event",
        "name": "AgentRegistered",
        "inputs": [
            {"name": "agentId", "type": "bytes32", "indexed": true},
            {"name": "did", "type": "string", "indexed": false},
            {"name": "owner", "type": "address", "indexed": true}
        ]
    }
]`

func parseABI() abi.ABI {
    parsedABI, err := abi.JSON(strings.NewReader(SageRegistryV2ABI))
    if err != nil {
        panic(fmt.Sprintf("failed to parse ABI: %v", err))
    }
    return parsedABI
}
```

**함수 호출 헬퍼**:

```go
// 읽기 전용 호출 (call)
func (c *EthereumClient) call(
    method string,
    result interface{},
    args ...interface{},
) error {
    // ABI 인코딩
    input, err := c.contract.Abi.Pack(method, args...)
    if err != nil {
        return fmt.Errorf("failed to pack args: %w", err)
    }

    // eth_call 실행
    msg := ethereum.CallMsg{
        To:   &c.contractAddr,
        Data: input,
    }

    output, err := c.client.CallContract(
        context.Background(),
        msg,
        nil,  // latest block
    )
    if err != nil {
        return fmt.Errorf("call failed: %w", err)
    }

    // ABI 디코딩
    err = c.contract.Abi.UnpackIntoInterface(result, method, output)
    if err != nil {
        return fmt.Errorf("failed to unpack result: %w", err)
    }

    return nil
}

// 트랜잭션 전송 (sendTransaction)
func (c *EthereumClient) transact(
    method string,
    args ...interface{},
) (*types.Transaction, error) {
    // ABI 인코딩
    input, err := c.contract.Abi.Pack(method, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to pack args: %w", err)
    }

    // Nonce 가져오기
    nonce, err := c.client.PendingNonceAt(
        context.Background(),
        c.txOpts.From,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get nonce: %w", err)
    }

    // 가스 추정
    gasLimit, err := c.client.EstimateGas(context.Background(), ethereum.CallMsg{
        From: c.txOpts.From,
        To:   &c.contractAddr,
        Data: input,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to estimate gas: %w", err)
    }

    // 트랜잭션 생성
    tx := types.NewTransaction(
        nonce,
        c.contractAddr,
        big.NewInt(0),  // value
        gasLimit,
        c.gasPrice,
        input,
    )

    // 서명
    signedTx, err := c.txOpts.Signer(c.txOpts.From, tx)
    if err != nil {
        return nil, fmt.Errorf("failed to sign tx: %w", err)
    }

    // 전송
    err = c.client.SendTransaction(context.Background(), signedTx)
    if err != nil {
        return nil, fmt.Errorf("failed to send tx: %w", err)
    }

    return signedTx, nil
}
```

### 3.3 이벤트 리스닝

**이벤트 구독**:

```go
// did/ethereum/client.go

type AgentRegisteredEvent struct {
    AgentID [32]byte
    DID     string
    Owner   common.Address
}

func (c *EthereumClient) SubscribeAgentRegistered(
    handler func(AgentRegisteredEvent),
) (ethereum.Subscription, error) {
    // 1. 이벤트 쿼리 생성
    query := ethereum.FilterQuery{
        Addresses: []common.Address{c.contractAddr},
        Topics:    [][]common.Hash{
            {crypto.Keccak256Hash([]byte("AgentRegistered(bytes32,string,address)"))},
        },
    }

    // 2. 로그 채널 생성
    logs := make(chan types.Log)

    // 3. 구독
    sub, err := c.client.SubscribeFilterLogs(
        context.Background(),
        query,
        logs,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe: %w", err)
    }

    // 4. 이벤트 처리 고루틴
    go func() {
        for {
            select {
            case log := <-logs:
                event, err := c.parseAgentRegisteredLog(log)
                if err != nil {
                    continue
                }
                handler(event)

            case err := <-sub.Err():
                log.Error("subscription error", "err", err)
                return
            }
        }
    }()

    return sub, nil
}

func (c *EthereumClient) parseAgentRegisteredLog(
    vLog types.Log,
) (AgentRegisteredEvent, error) {
    var event AgentRegisteredEvent

    // Topics 파싱
    if len(vLog.Topics) < 2 {
        return event, fmt.Errorf("invalid topics")
    }

    // agentId (indexed)
    copy(event.AgentID[:], vLog.Topics[1].Bytes())

    // owner (indexed)
    event.Owner = common.BytesToAddress(vLog.Topics[2].Bytes())

    // did (non-indexed, in Data)
    err := c.contract.Abi.UnpackIntoInterface(
        &struct{ DID string }{},
        "AgentRegistered",
        vLog.Data,
    )
    if err != nil {
        return event, err
    }

    return event, nil
}

사용 예:
sub, _ := client.SubscribeAgentRegistered(func(e AgentRegisteredEvent) {
    fmt.Printf("New agent registered: %s (owner: %s)\n",
        e.DID, e.Owner.Hex())
})
defer sub.Unsubscribe()
```

---

## 4. DID 등록 프로세스

### 4.1 전체 흐름

```
┌─────────────────────────────────────────────────────┐
│                1. 준비 단계                          │
│  - Ed25519 키 쌍 생성                               │
│  - Secp256k1 키 쌍 생성 (Ethereum용)                │
│  - 메타데이터 준비 (name, endpoint 등)              │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                2. DID 생성                           │
│  did = "did:sage:ethereum:" + ethereumAddress       │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                3. 소유권 증명 서명 생성               │
│  challenge = keccak256(                             │
│      "SAGE Key Registration:",                      │
│      chainId,                                       │
│      contractAddress,                               │
│      senderAddress,                                 │
│      keyHash                                        │
│  )                                                  │
│  signature = sign(challenge, secp256k1PrivateKey)   │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                4. 트랜잭션 생성                       │
│  tx = registerAgent(                                │
│      did, name, description, endpoint,              │
│      ed25519PublicKey, capabilities, signature      │
│  )                                                  │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                5. 트랜잭션 전송                       │
│  - 가스 가격 추정                                    │
│  - Nonce 설정                                       │
│  - 서명 및 전송                                     │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                6. 트랜잭션 대기                       │
│  - 블록에 포함될 때까지 대기                         │
│  - 영수증 확인                                      │
│  - 이벤트 로그 파싱                                 │
└────────────────────┬────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────┐
│                7. 검증 및 완료                        │
│  - agentId 추출                                     │
│  - DID 조회로 등록 확인                             │
│  - 로컬 캐시 업데이트                               │
└─────────────────────────────────────────────────────┘
```

### 4.2 코드 구현

**CLI 명령어**:

```go
// cmd/sage-did/register.go

var registerCmd = &cobra.Command{
    Use:   "register",
    Short: "Register a new AI agent DID",
    RunE:  runRegister,
}

func init() {
    registerCmd.Flags().String("chain", "ethereum", "Blockchain network")
    registerCmd.Flags().String("key", "", "Path to Secp256k1 private key")
    registerCmd.Flags().String("ed-key", "", "Path to Ed25519 public key")
    registerCmd.Flags().String("name", "", "Agent name")
    registerCmd.Flags().String("endpoint", "", "Agent API endpoint")
    registerCmd.Flags().StringSlice("capabilities", nil, "Agent capabilities")
    registerCmd.MarkFlagRequired("key")
    registerCmd.MarkFlagRequired("ed-key")
    registerCmd.MarkFlagRequired("name")
}

func runRegister(cmd *cobra.Command, args []string) error {
    // 1. 플래그 파싱
    chain, _ := cmd.Flags().GetString("chain")
    keyPath, _ := cmd.Flags().GetString("key")
    edKeyPath, _ := cmd.Flags().GetString("ed-key")
    name, _ := cmd.Flags().GetString("name")
    endpoint, _ := cmd.Flags().GetString("endpoint")
    caps, _ := cmd.Flags().GetStringSlice("capabilities")

    // 2. 키 로드
    secp256k1Key, err := loadSecp256k1Key(keyPath)
    if err != nil {
        return fmt.Errorf("failed to load Secp256k1 key: %w", err)
    }

    ed25519PubKey, err := loadEd25519PublicKey(edKeyPath)
    if err != nil {
        return fmt.Errorf("failed to load Ed25519 key: %w", err)
    }

    // 3. DID 생성
    addr := crypto.PubkeyToAddress(secp256k1Key.PublicKey)
    did := fmt.Sprintf("did:sage:%s:%s", chain, addr.Hex())

    fmt.Printf("Registering DID: %s\n", did)

    // 4. DID Manager 초기화
    manager := did.NewManager()
    err = manager.Configure(did.Chain(chain), &did.RegistryConfig{
        RPCEndpoint:     getEnv("ETHEREUM_RPC_URL"),
        ContractAddress: getEnv("SAGE_REGISTRY_ADDRESS"),
        ChainID:         getChainID(chain),
    })
    if err != nil {
        return err
    }

    // Ethereum 클라이언트 설정
    ethClient, err := ethereum.NewEthereumClient(
        getEnv("ETHEREUM_RPC_URL"),
        getEnv("SAGE_REGISTRY_ADDRESS"),
        secp256k1KeyToHex(secp256k1Key),
    )
    if err != nil {
        return err
    }
    manager.SetClient(did.ChainEthereum, ethClient)

    // 5. 소유권 증명 서명 생성
    signature, err := generateOwnershipSignature(
        secp256k1Key,
        ed25519PubKey,
        ethClient,
    )
    if err != nil {
        return fmt.Errorf("failed to generate signature: %w", err)
    }

    // 6. 등록 요청
    req := &did.RegistrationRequest{
        DID:          did.AgentDID(did),
        Name:         name,
        Description:  "",
        Endpoint:     endpoint,
        PublicKey:    ed25519PubKey,
        Capabilities: caps,
        Signature:    signature,
    }

    fmt.Println("Sending registration transaction...")
    result, err := manager.RegisterAgent(context.Background(), did.ChainEthereum, req)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }

    // 7. 결과 출력
    fmt.Printf("\nYes Registration successful!\n")
    fmt.Printf("   Transaction: %s\n", result.TxHash)
    fmt.Printf("   Agent ID: %s\n", result.AgentID)
    fmt.Printf("   Block: %d\n", result.BlockNumber)
    fmt.Printf("\n")
    fmt.Printf("View on Etherscan:\n")
    fmt.Printf("   https://etherscan.io/tx/%s\n", result.TxHash)

    return nil
}
```

**소유권 증명 서명**:

```go
func generateOwnershipSignature(
    privKey *ecdsa.PrivateKey,
    pubKey []byte,
    client *ethereum.EthereumClient,
) ([]byte, error) {
    // 1. 공개키 해시
    keyHash := crypto.Keccak256Hash(pubKey)

    // 2. 챌린지 메시지 구성
    // 스마트 컨트랙트와 동일한 방식
    chainID := client.ChainID()
    contractAddr := client.ContractAddress()
    senderAddr := crypto.PubkeyToAddress(privKey.PublicKey)

    message := crypto.Keccak256Hash(
        []byte("SAGE Key Registration:"),
        chainID.Bytes(),
        contractAddr.Bytes(),
        senderAddr.Bytes(),
        keyHash.Bytes(),
    )

    // 3. EIP-191 서명 (Ethereum Signed Message)
    prefixedMsg := fmt.Sprintf(
        "\x19Ethereum Signed Message:\n32%s",
        message,
    )
    hash := crypto.Keccak256Hash([]byte(prefixedMsg))

    // 4. ECDSA 서명
    signature, err := crypto.Sign(hash.Bytes(), privKey)
    if err != nil {
        return nil, err
    }

    // 5. Recovery ID 조정 (Ethereum 표준)
    if signature[64] < 27 {
        signature[64] += 27
    }

    return signature, nil
}
```

**등록 트랜잭션 전송**:

```go
// did/registry.go

func (r *MultiChainRegistry) Register(
    ctx context.Context,
    chain Chain,
    req *RegistrationRequest,
) (*RegistrationResult, error) {
    // 1. 체인별 레지스트리 가져오기
    reg, ok := r.registries[chain]
    if !ok {
        return nil, fmt.Errorf("chain not configured: %s", chain)
    }

    // 2. Capabilities JSON 인코딩
    capsJSON, err := json.Marshal(req.Capabilities)
    if err != nil {
        return nil, fmt.Errorf("failed to encode capabilities: %w", err)
    }

    // 3. 스마트 컨트랙트 호출
    tx, err := reg.(*ethereum.EthereumClient).RegisterAgent(
        string(req.DID),
        req.Name,
        req.Description,
        req.Endpoint,
        req.PublicKey,
        string(capsJSON),
        req.Signature,
    )
    if err != nil {
        return nil, fmt.Errorf("transaction failed: %w", err)
    }

    // 4. 트랜잭션 대기
    receipt, err := bind.WaitMined(ctx, reg.Client(), tx)
    if err != nil {
        return nil, fmt.Errorf("wait mined failed: %w", err)
    }

    // 5. 상태 확인
    if receipt.Status != types.ReceiptStatusSuccessful {
        return nil, fmt.Errorf("transaction reverted")
    }

    // 6. 이벤트 로그 파싱
    var agentID [32]byte
    for _, log := range receipt.Logs {
        if len(log.Topics) > 0 {
            eventSig := log.Topics[0]
            expectedSig := crypto.Keccak256Hash(
                []byte("AgentRegistered(bytes32,string,address)"),
            )
            if eventSig == expectedSig {
                copy(agentID[:], log.Topics[1].Bytes())
                break
            }
        }
    }

    // 7. 결과 반환
    return &RegistrationResult{
        AgentID:     hex.EncodeToString(agentID[:]),
        TxHash:      tx.Hash().Hex(),
        BlockNumber: receipt.BlockNumber.Uint64(),
        GasUsed:     receipt.GasUsed,
    }, nil
}
```

### 4.3 가스 최적화

**가스 추정 및 최적화**:

```go
// did/ethereum/client.go

type GasEstimator struct {
    client  *ethclient.Client
    history []uint64  // 최근 가스 가격 히스토리
    mu      sync.Mutex
}

func (e *GasEstimator) EstimateGasPrice() (*big.Int, error) {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 1. 네트워크 제안 가격
    suggested, err := e.client.SuggestGasPrice(context.Background())
    if err != nil {
        return nil, err
    }

    // 2. 최근 블록의 평균 가스 가격
    header, err := e.client.HeaderByNumber(context.Background(), nil)
    if err != nil {
        return nil, err
    }
    baseFee := header.BaseFee

    // 3. EIP-1559: maxFeePerGas 계산
    // maxFee = baseFee * 2 + priorityFee
    priorityFee := big.NewInt(2 * params.GWei)  // 2 Gwei tip
    maxFee := new(big.Int).Mul(baseFee, big.NewInt(2))
    maxFee.Add(maxFee, priorityFee)

    // 4. 히스토리 기반 조정
    if len(e.history) > 0 {
        avg := e.averageHistory()
        // 평균보다 낮으면 조정
        if maxFee.Cmp(avg) < 0 {
            maxFee = avg
        }
    }

    // 5. 히스토리 업데이트
    e.history = append(e.history, maxFee.Uint64())
    if len(e.history) > 100 {
        e.history = e.history[1:]
    }

    return maxFee, nil
}

// 동적 가스 한도 추정
func (c *EthereumClient) EstimateGasLimit(
    method string,
    args ...interface{},
) (uint64, error) {
    // 1. ABI 인코딩
    input, err := c.contract.Abi.Pack(method, args...)
    if err != nil {
        return 0, err
    }

    // 2. eth_estimateGas 호출
    msg := ethereum.CallMsg{
        From: c.txOpts.From,
        To:   &c.contractAddr,
        Data: input,
    }

    gasLimit, err := c.client.EstimateGas(context.Background(), msg)
    if err != nil {
        return 0, err
    }

    // 3. 안전 마진 추가 (20%)
    safeLimit := gasLimit * 120 / 100

    return safeLimit, nil
}
```

**배치 등록 (가스 절약)**:

```solidity
// contracts/ethereum/contracts/SageRegistryBatch.sol

function registerAgentBatch(
    RegistrationParams[] calldata agents
) external returns (bytes32[] memory) {
    bytes32[] memory agentIds = new bytes32[](agents.length);

    for (uint i = 0; i < agents.length; i++) {
        agentIds[i] = _registerAgent(agents[i]);
    }

    return agentIds;
}

가스 비교:
- 개별 등록 10개: ~6,200,000 gas
- 배치 등록 10개: ~4,800,000 gas
- 절약: ~22%
```

---

## 5. DID 조회 및 해석

### 5.1 조회 메커니즘

**조회 프로세스**:

```
┌─────────────────────────────────────────┐
│       1. DID 파싱                        │
│  did:sage:ethereum:0x742d35Cc...        │
│  → chain: ethereum                      │
│  → address: 0x742d35Cc...               │
└──────────────┬──────────────────────────┘
               ↓
┌─────────────────────────────────────────┐
│       2. 캐시 확인                       │
│  if cached && !expired:                 │
│      return cached                      │
└──────────────┬──────────────────────────┘
               ↓ cache miss
┌─────────────────────────────────────────┐
│       3. agentId 계산                    │
│  agentId = keccak256(did)               │
└──────────────┬──────────────────────────┘
               ↓
┌─────────────────────────────────────────┐
│       4. 스마트 컨트랙트 조회             │
│  agent = contract.getAgent(agentId)     │
└──────────────┬──────────────────────────┘
               ↓
┌─────────────────────────────────────────┐
│       5. 데이터 변환                     │
│  AgentMetadata (Solidity struct)        │
│  → AgentMetadata (Go struct)            │
└──────────────┬──────────────────────────┘
               ↓
┌─────────────────────────────────────────┐
│       6. 검증                            │
│  - active == true?                      │
│  - publicKey valid?                     │
│  - endpoint reachable? (optional)       │
└──────────────┬──────────────────────────┘
               ↓
┌─────────────────────────────────────────┐
│       7. 캐시 저장 및 반환               │
│  cache.Set(did, metadata, ttl)          │
│  return metadata                        │
└─────────────────────────────────────────┘
```

**구현**:

```go
// did/resolver.go

type MultiChainResolver struct {
    resolvers map[Chain]Resolver
    cache     *Cache
    mu        sync.RWMutex
}

func (r *MultiChainResolver) Resolve(
    ctx context.Context,
    did AgentDID,
) (*AgentMetadata, error) {
    // 1. DID 파싱
    chain, identifier, err := ParseDID(did)
    if err != nil {
        return nil, fmt.Errorf("invalid DID: %w", err)
    }

    // 2. 캐시 확인
    cacheKey := string(did)
    if cached, ok := r.cache.Get(cacheKey); ok {
        return cached.(*AgentMetadata), nil
    }

    // 3. 체인별 resolver 가져오기
    resolver, ok := r.resolvers[chain]
    if !ok {
        return nil, fmt.Errorf("chain not configured: %s", chain)
    }

    // 4. 체인별 조회
    metadata, err := resolver.Resolve(ctx, did)
    if err != nil {
        return nil, err
    }

    // 5. 검증
    if err := r.validateMetadata(metadata); err != nil {
        return nil, fmt.Errorf("invalid metadata: %w", err)
    }

    // 6. 캐시 저장
    r.cache.Set(cacheKey, metadata, 5*time.Minute)

    return metadata, nil
}

func (r *MultiChainResolver) validateMetadata(
    meta *AgentMetadata,
) error {
    // 활성 상태 확인
    if !meta.Active {
        return fmt.Errorf("agent is deactivated")
    }

    // 공개키 확인
    if len(meta.PublicKey) != 32 {
        return fmt.Errorf("invalid public key length")
    }

    // DID 형식 확인
    if !strings.HasPrefix(meta.DID, "did:sage:") {
        return fmt.Errorf("invalid DID format")
    }

    return nil
}
```

**Ethereum Resolver**:

```go
// did/ethereum/resolver.go

type EthereumResolver struct {
    client *EthereumClient
}

func (r *EthereumResolver) Resolve(
    ctx context.Context,
    did did.AgentDID,
) (*did.AgentMetadata, error) {
    // 1. agentId 계산
    agentID := crypto.Keccak256Hash([]byte(did))

    // 2. 스마트 컨트랙트 호출
    var result struct {
        DID          string
        Owner        common.Address
        PublicKey    []byte
        Name         string
        Description  string
        Endpoint     string
        Capabilities string
        Active       bool
        CreatedAt    *big.Int
        UpdatedAt    *big.Int
    }

    err := r.client.call("getAgent", &result, agentID)
    if err != nil {
        return nil, fmt.Errorf("contract call failed: %w", err)
    }

    // 3. DID 존재 확인
    if result.Owner == (common.Address{}) {
        return nil, fmt.Errorf("DID not found")
    }

    // 4. Capabilities 파싱
    var capabilities []string
    if result.Capabilities != "" {
        json.Unmarshal([]byte(result.Capabilities), &capabilities)
    }

    // 5. AgentMetadata 구성
    metadata := &did.AgentMetadata{
        DID:          result.DID,
        Owner:        result.Owner.Hex(),
        PublicKey:    result.PublicKey,
        Name:         result.Name,
        Description:  result.Description,
        Endpoint:     result.Endpoint,
        Capabilities: capabilities,
        Active:       result.Active,
        CreatedAt:    time.Unix(result.CreatedAt.Int64(), 0),
        UpdatedAt:    time.Unix(result.UpdatedAt.Int64(), 0),
    }

    return metadata, nil
}

// 공개키만 조회 (최적화)
func (r *EthereumResolver) ResolvePublicKey(
    ctx context.Context,
    did did.AgentDID,
) ([]byte, error) {
    agentID := crypto.Keccak256Hash([]byte(did))

    var result struct {
        PublicKey []byte
    }

    // 부분 조회 (가스 절약)
    err := r.client.call("getAgentPublicKey", &result, agentID)
    if err != nil {
        return nil, err
    }

    return result.PublicKey, nil
}
```

### 5.2 배치 조회

**여러 DID 동시 조회**:

```go
// did/resolver.go

func (r *MultiChainResolver) ResolveBatch(
    ctx context.Context,
    dids []AgentDID,
) ([]*AgentMetadata, error) {
    // 체인별 그룹화
    byChain := make(map[Chain][]AgentDID)
    for _, did := range dids {
        chain, _, err := ParseDID(did)
        if err != nil {
            continue
        }
        byChain[chain] = append(byChain[chain], did)
    }

    // 병렬 조회
    results := make(chan *AgentMetadata, len(dids))
    errors := make(chan error, len(dids))
    var wg sync.WaitGroup

    for chain, chainDIDs := range byChain {
        wg.Add(1)
        go func(c Chain, dids []AgentDID) {
            defer wg.Done()
            for _, did := range dids {
                meta, err := r.Resolve(ctx, did)
                if err != nil {
                    errors <- err
                    continue
                }
                results <- meta
            }
        }(chain, chainDIDs)
    }

    // 결과 수집
    go func() {
        wg.Wait()
        close(results)
        close(errors)
    }()

    var metadata []*AgentMetadata
    for meta := range results {
        metadata = append(metadata, meta)
    }

    return metadata, nil
}
```

### 5.3 오프체인 인덱싱

**The Graph를 사용한 빠른 조회**:

```graphql
# subgraph/schema.graphql

type Agent @entity {
  id: ID! # agentId
  did: String! # DID 문자열
  owner: Bytes! # 소유자 주소
  publicKey: Bytes! # 공개키
  name: String!
  description: String
  endpoint: String!
  capabilities: [String!]!
  active: Boolean!
  createdAt: BigInt!
  updatedAt: BigInt!

  # 관계
  updates: [AgentUpdate!]! @derivedFrom(field: "agent")
}

type AgentUpdate @entity {
  id: ID! # txHash-logIndex
  agent: Agent!
  field: String! # 변경된 필드
  oldValue: String
  newValue: String!
  timestamp: BigInt!
  blockNumber: BigInt!
  txHash: Bytes!
}
```

**GraphQL 쿼리**:

```graphql
# 조회 예시
query GetAgent($did: String!) {
  agents(where: { did: $did, active: true }) {
    id
    did
    owner
    name
    endpoint
    publicKey
    capabilities
    createdAt
    updatedAt
  }
}

# 검색 예시
query SearchAgents($name: String!) {
  agents(
    where: { name_contains: $name, active: true }
    orderBy: createdAt
    orderDirection: desc
    first: 10
  ) {
    id
    did
    name
    endpoint
    capabilities
  }
}

# 소유자별 조회
query GetAgentsByOwner($owner: Bytes!) {
  agents(
    where: { owner: $owner, active: true }
    orderBy: createdAt
    orderDirection: desc
  ) {
    id
    did
    name
    endpoint
  }
}
```

**Go 클라이언트**:

```go
// did/indexer/graph_client.go

type GraphClient struct {
    endpoint string
    client   *http.Client
}

func (c *GraphClient) QueryAgent(did string) (*AgentMetadata, error) {
    query := `
        query($did: String!) {
            agents(where: { did: $did, active: true }) {
                id did owner name endpoint publicKey
                capabilities createdAt updatedAt
            }
        }
    `

    vars := map[string]interface{}{
        "did": did,
    }

    resp, err := c.query(query, vars)
    if err != nil {
        return nil, err
    }

    // JSON 파싱
    var result struct {
        Data struct {
            Agents []struct {
                ID           string   `json:"id"`
                DID          string   `json:"did"`
                Owner        string   `json:"owner"`
                Name         string   `json:"name"`
                Endpoint     string   `json:"endpoint"`
                PublicKey    string   `json:"publicKey"`
                Capabilities []string `json:"capabilities"`
                CreatedAt    string   `json:"createdAt"`
                UpdatedAt    string   `json:"updatedAt"`
            } `json:"agents"`
        } `json:"data"`
    }

    err = json.Unmarshal(resp, &result)
    if err != nil {
        return nil, err
    }

    if len(result.Data.Agents) == 0 {
        return nil, fmt.Errorf("agent not found")
    }

    agent := result.Data.Agents[0]
    // AgentMetadata로 변환...

    return metadata, nil
}

장점:
- 블록체인 조회보다 10-100배 빠름
- 복잡한 검색 쿼리 가능
- 히스토리 추적 가능
- 가스 비용 없음
```

---

## 6. DID 업데이트 및 비활성화

### 6.1 업데이트 프로세스

**업데이트 가능한 필드**:

```
변경 가능:
Yes name: 에이전트 이름
Yes description: 설명
Yes endpoint: API 엔드포인트
Yes capabilities: 기능 목록

변경 불가:
No did: DID는 불변
No owner: 소유권 이전 불가 (보안상)
No publicKey: 키 변경 불가 (새로 등록 필요)
No createdAt: 생성 시간 불변
```

**업데이트 코드**:

```go
// cmd/sage-did/update.go

func runUpdate(cmd *cobra.Command, args []string) error {
    did := args[0]

    // 변경할 필드들
    updates := make(map[string]interface{})

    if cmd.Flags().Changed("name") {
        name, _ := cmd.Flags().GetString("name")
        updates["name"] = name
    }

    if cmd.Flags().Changed("endpoint") {
        endpoint, _ := cmd.Flags().GetString("endpoint")
        updates["endpoint"] = endpoint
    }

    if cmd.Flags().Changed("capabilities") {
        caps, _ := cmd.Flags().GetStringSlice("capabilities")
        updates["capabilities"] = caps
    }

    // 개인키 로드 (소유자 증명)
    keyPath, _ := cmd.Flags().GetString("key")
    keyPair, err := loadKey(keyPath)
    if err != nil {
        return err
    }

    // DID Manager 초기화
    manager := initManager()

    // 업데이트 실행
    err = manager.UpdateAgent(
        context.Background(),
        did.AgentDID(did),
        updates,
        keyPair,
    )
    if err != nil {
        return fmt.Errorf("update failed: %w", err)
    }

    fmt.Printf("Yes Agent updated successfully\n")
    return nil
}
```

**스마트 컨트랙트 업데이트**:

```solidity
// contracts/ethereum/contracts/SageRegistryV2.sol

function updateAgent(
    bytes32 agentId,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    string calldata capabilities
) external onlyAgentOwner(agentId) {
    AgentMetadata storage agent = agents[agentId];

    require(agent.active, "Agent not active");

    // 변경 사항만 업데이트 (가스 절약)
    if (bytes(name).length > 0) {
        agent.name = name;
    }
    if (bytes(description).length > 0) {
        agent.description = description;
    }
    if (bytes(endpoint).length > 0) {
        agent.endpoint = endpoint;
    }
    if (bytes(capabilities).length > 0) {
        agent.capabilities = capabilities;
    }

    agent.updatedAt = block.timestamp;
    agentNonce[agentId]++;

    emit AgentUpdated(agentId, agent.did, msg.sender);
}
```

### 6.2 비활성화

**비활성화 vs 삭제**:

```
비활성화 (Deactivate):
- active 플래그를 false로 설정
- 데이터는 블록체인에 남음
- 재활성화 가능 (reactivate 함수)
- DID 조회 시 "비활성화됨" 반환

완전 삭제:
- 불가능 (블록체인 불변성)
- 프라이버시: 민감 정보는 오프체인 저장 권장
```

**비활성화 코드**:

```go
// cmd/sage-did/deactivate.go

func runDeactivate(cmd *cobra.Command, args []string) error {
    did := args[0]

    // 소유자 확인
    keyPath, _ := cmd.Flags().GetString("key")
    keyPair, err := loadKey(keyPath)
    if err != nil {
        return err
    }

    // 확인 메시지
    fmt.Printf("Warning  Warning: This will deactivate the agent:\n")
    fmt.Printf("   DID: %s\n", did)
    fmt.Printf("   This action can be reverted later.\n")
    fmt.Printf("\nContinue? (yes/no): ")

    var confirm string
    fmt.Scanln(&confirm)
    if confirm != "yes" {
        fmt.Println("Cancelled.")
        return nil
    }

    // DID Manager
    manager := initManager()

    // 비활성화
    err = manager.DeactivateAgent(
        context.Background(),
        did.AgentDID(did),
        keyPair,
    )
    if err != nil {
        return fmt.Errorf("deactivation failed: %w", err)
    }

    fmt.Printf("Yes Agent deactivated\n")
    return nil
}
```

**스마트 컨트랙트**:

```solidity
function deactivateAgent(
    bytes32 agentId
) external onlyAgentOwner(agentId) {
    AgentMetadata storage agent = agents[agentId];
    require(agent.active, "Already deactivated");

    agent.active = false;
    agent.updatedAt = block.timestamp;
    agentNonce[agentId]++;

    emit AgentDeactivated(agentId, agent.did, msg.sender);
}

// 재활성화 (선택적 기능)
function reactivateAgent(
    bytes32 agentId
) external onlyAgentOwner(agentId) {
    AgentMetadata storage agent = agents[agentId];
    require(!agent.active, "Already active");

    agent.active = true;
    agent.updatedAt = block.timestamp;
    agentNonce[agentId]++;

    emit AgentReactivated(agentId, agent.did, msg.sender);
}
```

---

## 7. 캐싱 및 성능 최적화

### 7.1 다층 캐싱 전략

```
┌─────────────────────────────────────────┐
│   Layer 1: In-Memory LRU Cache           │
│   - 가장 빠름 (~1μs)                     │
│   - 프로세스 내부                        │
│   - 1000개 항목, 5분 TTL                │
└──────────────┬──────────────────────────┘
               ↓ cache miss
┌─────────────────────────────────────────┐
│   Layer 2: Redis/Memcached              │
│   - 빠름 (~1ms)                         │
│   - 여러 인스턴스 공유                   │
│   - 10000개 항목, 30분 TTL              │
└──────────────┬──────────────────────────┘
               ↓ cache miss
┌─────────────────────────────────────────┐
│   Layer 3: The Graph (오프체인 인덱스)   │
│   - 보통 (~100ms)                       │
│   - 복잡한 쿼리 가능                     │
│   - 무제한, 실시간 동기화               │
└──────────────┬──────────────────────────┘
               ↓ cache miss
┌─────────────────────────────────────────┐
│   Layer 4: Blockchain RPC               │
│   - 느림 (~1s)                          │
│   - 가장 정확                            │
│   - 원본 데이터                          │
└─────────────────────────────────────────┘
```

**구현**:

```go
// did/cache/cache.go

type MultiLevelCache struct {
    l1 *lru.Cache           // In-memory
    l2 *redis.Client        // Redis
    l3 *GraphClient         // The Graph

    l1TTL time.Duration
    l2TTL time.Duration
}

func (c *MultiLevelCache) Get(key string) (interface{}, bool) {
    // L1: In-memory
    if val, ok := c.l1.Get(key); ok {
        return val, true
    }

    // L2: Redis
    if c.l2 != nil {
        val, err := c.l2.Get(context.Background(), key).Result()
        if err == nil {
            var metadata AgentMetadata
            json.Unmarshal([]byte(val), &metadata)
            // L1에도 저장
            c.l1.Add(key, &metadata)
            return &metadata, true
        }
    }

    // L3: The Graph
    if c.l3 != nil {
        metadata, err := c.l3.QueryAgent(key)
        if err == nil {
            // L1, L2에 저장
            c.Set(key, metadata, c.l1TTL)
            return metadata, true
        }
    }

    return nil, false
}

func (c *MultiLevelCache) Set(
    key string,
    value interface{},
    ttl time.Duration,
) {
    // L1
    c.l1.Add(key, value)

    // L2
    if c.l2 != nil {
        data, _ := json.Marshal(value)
        c.l2.Set(
            context.Background(),
            key,
            data,
            c.l2TTL,
        )
    }
}
```

### 7.2 Bloom Filter로 존재 확인

**존재하지 않는 DID 빠르게 필터링**:

```go
// did/cache/bloom.go

type BloomFilter struct {
    filter *bloom.BloomFilter
    mu     sync.RWMutex
}

func NewBloomFilter(expectedItems uint) *BloomFilter {
    // False positive rate: 0.01 (1%)
    return &BloomFilter{
        filter: bloom.NewWithEstimates(expectedItems, 0.01),
    }
}

func (bf *BloomFilter) MightExist(did string) bool {
    bf.mu.RLock()
    defer bf.mu.RUnlock()
    return bf.filter.Test([]byte(did))
}

func (bf *BloomFilter) Add(did string) {
    bf.mu.Lock()
    defer bf.mu.Unlock()
    bf.filter.Add([]byte(did))
}

// Resolver에 통합
func (r *MultiChainResolver) Resolve(
    ctx context.Context,
    did AgentDID,
) (*AgentMetadata, error) {
    // Bloom filter로 빠른 존재 확인
    if !r.bloom.MightExist(string(did)) {
        return nil, fmt.Errorf("DID not found")
    }

    // 실제 조회
    metadata, err := r.resolveFromChain(ctx, did)
    if err != nil {
        return nil, err
    }

    // Bloom filter 업데이트
    r.bloom.Add(string(did))

    return metadata, nil
}

성능:
- False positive: 1% (존재하지 않는데 존재한다고 판단)
- False negative: 0% (존재하는데 없다고 판단 절대 안함)
- 메모리: 1M DID당 ~1.2MB
- 조회 속도: O(k) ≈ O(1), k = hash 함수 개수
```

### 7.3 프리페칭 (Prefetching)

**자주 사용되는 DID 미리 로드**:

```go
// did/cache/prefetcher.go

type Prefetcher struct {
    resolver *MultiChainResolver
    cache    *Cache

    // 사용 통계
    stats    map[string]int
    mu       sync.RWMutex

    ticker   *time.Ticker
    stopCh   chan struct{}
}

func (p *Prefetcher) Start() {
    p.ticker = time.NewTicker(1 * time.Minute)
    p.stopCh = make(chan struct{})

    go func() {
        for {
            select {
            case <-p.ticker.C:
                p.prefetchPopular()
            case <-p.stopCh:
                return
            }
        }
    }()
}

func (p *Prefetcher) prefetchPopular() {
    p.mu.RLock()

    // 사용 빈도 상위 100개 DID
    type didFreq struct {
        did   string
        count int
    }

    var popular []didFreq
    for did, count := range p.stats {
        popular = append(popular, didFreq{did, count})
    }
    p.mu.RUnlock()

    // 정렬
    sort.Slice(popular, func(i, j int) bool {
        return popular[i].count > popular[j].count
    })

    // 상위 100개 프리페치
    for i := 0; i < 100 && i < len(popular); i++ {
        did := AgentDID(popular[i].did)

        // 캐시에 없으면 로드
        if _, ok := p.cache.Get(string(did)); !ok {
            metadata, err := p.resolver.Resolve(
                context.Background(),
                did,
            )
            if err == nil {
                p.cache.Set(string(did), metadata, 30*time.Minute)
            }
        }
    }
}

// 사용 통계 기록
func (p *Prefetcher) RecordAccess(did string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.stats[did]++
}
```

---

## 8. 다중 체인 관리

### 8.1 체인 추상화

**통일된 인터페이스**:

```go
// did/types.go

type Registry interface {
    Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error)
    Update(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair crypto.KeyPair) error
    Deactivate(ctx context.Context, did AgentDID, keyPair crypto.KeyPair) error
}

type Resolver interface {
    Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error)
    ResolvePublicKey(ctx context.Context, did AgentDID) ([]byte, error)
}

type ChainClient interface {
    Registry
    Resolver

    // 체인 정보
    ChainID() *big.Int
    BlockNumber() (uint64, error)

    // 트랜잭션
    WaitForTransaction(ctx context.Context, txHash string) error
}
```

**체인별 구현**:

```go
// did/ethereum/client.go
type EthereumClient struct { ... }
func (c *EthereumClient) Register(...) { ... }
func (c *EthereumClient) Resolve(...) { ... }

// did/kaia/client.go
type KaiaClient struct { ... }
func (c *KaiaClient) Register(...) { ... }
func (c *KaiaClient) Resolve(...) { ... }

// did/solana/client.go (planned)
type SolanaClient struct { ... }
func (c *SolanaClient) Register(...) { ... }
func (c *SolanaClient) Resolve(...) { ... }
```

### 8.2 크로스 체인 검증

**여러 체인에서 동일 에이전트 검증**:

```go
// did/verification.go

type CrossChainVerifier struct {
    resolver *MultiChainResolver
}

func (v *CrossChainVerifier) VerifyCrossChain(
    dids []AgentDID,
) (bool, error) {
    // 1. 모든 DID 조회
    metadataList := make([]*AgentMetadata, len(dids))
    for i, did := range dids {
        meta, err := v.resolver.Resolve(context.Background(), did)
        if err != nil {
            return false, fmt.Errorf("failed to resolve %s: %w", did, err)
        }
        metadataList[i] = meta
    }

    // 2. 일관성 검증
    first := metadataList[0]
    for i := 1; i < len(metadataList); i++ {
        meta := metadataList[i]

        // 공개키 일치 확인
        if !bytes.Equal(first.PublicKey, meta.PublicKey) {
            return false, fmt.Errorf("public key mismatch")
        }

        // 소유자 일치 확인 (주소 형식 차이 고려)
        if !v.ownerMatches(first.Owner, meta.Owner) {
            return false, fmt.Errorf("owner mismatch")
        }

        // 이름 일치 확인
        if first.Name != meta.Name {
            return false, fmt.Errorf("name mismatch")
        }
    }

    return true, nil
}

func (v *CrossChainVerifier) ownerMatches(addr1, addr2 string) bool {
    // Ethereum 주소 정규화
    if strings.HasPrefix(addr1, "0x") && strings.HasPrefix(addr2, "0x") {
        return strings.EqualFold(addr1, addr2)
    }

    // Solana 주소는 그대로 비교
    return addr1 == addr2
}
```

### 8.3 체인 선택 알고리즘

**최적의 체인 자동 선택**:

```go
// did/selector.go

type ChainSelector struct {
    preferences ChainPreferences
    monitor     *ChainMonitor
}

type ChainPreferences struct {
    PreferredChains []Chain
    CostSensitive   bool
    SpeedSensitive  bool
}

func (s *ChainSelector) SelectBestChain(
    operation string,
) Chain {
    scores := make(map[Chain]float64)

    for _, chain := range s.preferences.PreferredChains {
        score := s.calculateScore(chain, operation)
        scores[chain] = score
    }

    // 최고 점수 체인 선택
    var bestChain Chain
    var bestScore float64
    for chain, score := range scores {
        if score > bestScore {
            bestChain = chain
            bestScore = score
        }
    }

    return bestChain
}

func (s *ChainSelector) calculateScore(
    chain Chain,
    operation string,
) float64 {
    score := 0.0

    // 가용성 (0-10)
    uptime := s.monitor.GetUptime(chain)
    score += uptime * 10

    // 비용 (0-10, 낮을수록 좋음)
    if s.preferences.CostSensitive {
        avgCost := s.monitor.GetAverageCost(chain, operation)
        // $0.001 = 10점, $1 = 0점
        costScore := 10 - (math.Log10(avgCost*1000) * 2)
        score += max(0, costScore) * 2  // 가중치 2배
    }

    // 속도 (0-10)
    if s.preferences.SpeedSensitive {
        avgTime := s.monitor.GetAverageTime(chain, operation)
        // 1초 = 10점, 60초 = 0점
        timeScore := 10 - (avgTime.Seconds() / 6)
        score += max(0, timeScore) * 1.5  // 가중치 1.5배
    }

    // 확정 시간 (0-10)
    finality := s.monitor.GetFinalityTime(chain)
    finalityScore := 10 - (finality.Seconds() / 60)
    score += max(0, finalityScore)

    return score
}

예시:
preferences := ChainPreferences{
    PreferredChains: []Chain{ChainEthereum, ChainKaia},
    CostSensitive:   true,
    SpeedSensitive:  false,
}

selector := NewChainSelector(preferences, monitor)
chain := selector.SelectBestChain("register")
// → ChainKaia (비용 낮음)

preferences.SpeedSensitive = true
chain = selector.SelectBestChain("resolve")
// → ChainKaia (빠르고 저렴)
```

---

## 9. 실전 예제

### 9.1 완전한 DID 라이프사이클

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/did/ethereum"
)

func main() {
    fmt.Println("=== SAGE DID 라이프사이클 예제 ===\n")

    // 1. 키 생성
    fmt.Println("1. 키 생성")
    edKey, _ := keys.GenerateEd25519KeyPair()
    secpKey, _ := keys.GenerateSecp256k1KeyPair()
    fmt.Printf("   Ed25519 ID: %s\n", edKey.ID())
    fmt.Printf("   Secp256k1 ID: %s\n", secpKey.ID())

    // 2. DID Manager 초기화
    fmt.Println("\n2. DID Manager 초기화")
    manager := did.NewManager()

    manager.Configure(did.ChainEthereum, &did.RegistryConfig{
        RPCEndpoint:     "http://localhost:8545",
        ContractAddress: "0x...",
        ChainID:         1,
    })

    ethClient, _ := ethereum.NewEthereumClient(
        "http://localhost:8545",
        "0x...",
        "private_key_hex",
    )
    manager.SetClient(did.ChainEthereum, ethClient)

    // 3. DID 등록
    fmt.Println("\n3. DID 등록")
    edPub := edKey.PublicKey().(ed25519.PublicKey)

    signature, _ := generateOwnershipSignature(
        secpKey,
        edPub,
        ethClient,
    )

    req := &did.RegistrationRequest{
        DID:          did.GenerateDID(did.ChainEthereum, secpAddress),
        Name:         "Demo AI Agent",
        Description:  "Demonstration agent for SAGE DID",
        Endpoint:     "https://demo-agent.example.com/api",
        PublicKey:    edPub,
        Capabilities: []string{"chat", "analysis", "translation"},
        Signature:    signature,
    }

    result, err := manager.RegisterAgent(
        context.Background(),
        did.ChainEthereum,
        req,
    )
    if err != nil {
        panic(err)
    }

    agentDID := req.DID
    fmt.Printf("   Yes 등록 완료\n")
    fmt.Printf("   DID: %s\n", agentDID)
    fmt.Printf("   Tx: %s\n", result.TxHash)
    fmt.Printf("   Block: %d\n", result.BlockNumber)
    fmt.Printf("   Gas: %d\n", result.GasUsed)

    // 4. DID 조회
    fmt.Println("\n4. DID 조회")
    time.Sleep(2 * time.Second)  // 블록 확정 대기

    metadata, err := manager.ResolveAgent(
        context.Background(),
        agentDID,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("   이름: %s\n", metadata.Name)
    fmt.Printf("   엔드포인트: %s\n", metadata.Endpoint)
    fmt.Printf("   공개키: %x...\n", metadata.PublicKey[:16])
    fmt.Printf("   기능: %v\n", metadata.Capabilities)
    fmt.Printf("   활성: %v\n", metadata.Active)
    fmt.Printf("   생성: %s\n", metadata.CreatedAt.Format(time.RFC3339))

    // 5. DID 업데이트
    fmt.Println("\n5. DID 업데이트")
    updates := map[string]interface{}{
        "name":     "Updated Demo Agent",
        "endpoint": "https://updated-agent.example.com/api",
        "capabilities": []string{
            "chat", "analysis", "translation", "code-generation",
        },
    }

    err = manager.UpdateAgent(
        context.Background(),
        agentDID,
        updates,
        secpKey,
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("   Yes 업데이트 완료\n")

    // 6. 업데이트 확인
    fmt.Println("\n6. 업데이트 확인")
    time.Sleep(2 * time.Second)

    metadata, _ = manager.ResolveAgent(
        context.Background(),
        agentDID,
    )
    fmt.Printf("   새 이름: %s\n", metadata.Name)
    fmt.Printf("   새 엔드포인트: %s\n", metadata.Endpoint)
    fmt.Printf("   새 기능: %v\n", metadata.Capabilities)

    // 7. 기능 확인
    fmt.Println("\n7. 기능 확인")
    hasCodeGen, _ := manager.CheckCapabilities(
        context.Background(),
        agentDID,
        []string{"code-generation"},
    )
    fmt.Printf("   code-generation 지원: %v\n", hasCodeGen)

    hasUnknown, _ := manager.CheckCapabilities(
        context.Background(),
        agentDID,
        []string{"unknown-capability"},
    )
    fmt.Printf("   unknown-capability 지원: %v\n", hasUnknown)

    // 8. 소유자의 모든 에이전트 조회
    fmt.Println("\n8. 소유자의 모든 에이전트 조회")
    ownerAddress := crypto.PubkeyToAddress(
        secpKey.PublicKey().(*ecdsa.PublicKey),
    ).Hex()

    agents, _ := manager.ListAgentsByOwner(
        context.Background(),
        ownerAddress,
    )
    fmt.Printf("   소유자: %s\n", ownerAddress)
    fmt.Printf("   에이전트 수: %d\n", len(agents))
    for i, agent := range agents {
        fmt.Printf("   [%d] %s - %s\n", i+1, agent.Name, agent.DID)
    }

    // 9. 비활성화
    fmt.Println("\n9. 비활성화")
    fmt.Printf("   Warning  에이전트를 비활성화하시겠습니까? (yes/no): ")

    var confirm string
    fmt.Scanln(&confirm)

    if confirm == "yes" {
        err = manager.DeactivateAgent(
            context.Background(),
            agentDID,
            secpKey,
        )
        if err != nil {
            panic(err)
        }
        fmt.Printf("   Yes 비활성화 완료\n")

        // 10. 비활성화 확인
        fmt.Println("\n10. 비활성화 확인")
        time.Sleep(2 * time.Second)

        metadata, _ = manager.ResolveAgent(
            context.Background(),
            agentDID,
        )
        fmt.Printf("   활성 상태: %v\n", metadata.Active)
    } else {
        fmt.Println("   비활성화 취소")
    }

    fmt.Println("\n=== 라이프사이클 완료 ===")
}
```

### 9.2 다중 체인 배포

```go
package main

import (
    "context"
    "fmt"
    "sync"
)

func main() {
    fmt.Println("=== 다중 체인 DID 배포 ===\n")

    // 키 생성
    edKey, _ := keys.GenerateEd25519KeyPair()
    ethKey, _ := keys.GenerateSecp256k1KeyPair()  // Ethereum/Kaia 공용

    // Manager 초기화
    manager := did.NewManager()

    // Ethereum 설정
    manager.Configure(did.ChainEthereum, &did.RegistryConfig{
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/...",
        ContractAddress: "0xEthereumContract...",
        ChainID:         1,
    })

    // Kaia 설정
    manager.Configure(did.ChainKaia, &did.RegistryConfig{
        RPCEndpoint:     "https://public-en.node.kaia.io",
        ContractAddress: "0xKaiaContract...",
        ChainID:         8217,
    })

    // 클라이언트 설정...

    // 등록 요청 준비
    baseReq := &did.RegistrationRequest{
        Name:         "Multi-Chain Agent",
        Description:  "Agent deployed on multiple chains",
        Endpoint:     "https://agent.example.com/api",
        PublicKey:    edKey.PublicKey().(ed25519.PublicKey),
        Capabilities: []string{"cross-chain", "multi-network"},
    }

    // 병렬 등록
    var wg sync.WaitGroup
    results := make(chan *did.RegistrationResult, 2)
    errors := make(chan error, 2)

    chains := []did.Chain{did.ChainEthereum, did.ChainKaia}
    for _, chain := range chains {
        wg.Add(1)
        go func(c did.Chain) {
            defer wg.Done()

            // 체인별 DID 생성
            req := *baseReq
            req.DID = did.GenerateDID(c, ethAddress)

            // 서명 생성
            signature, err := generateSignatureForChain(c, ethKey, req.PublicKey)
            if err != nil {
                errors <- err
                return
            }
            req.Signature = signature

            // 등록
            fmt.Printf("Registering on %s...\n", c)
            result, err := manager.RegisterAgent(
                context.Background(),
                c,
                &req,
            )
            if err != nil {
                errors <- err
                return
            }

            results <- result
            fmt.Printf("Yes %s: %s\n", c, result.TxHash)
        }(chain)
    }

    wg.Wait()
    close(results)
    close(errors)

    // 결과 수집
    fmt.Println("\n=== 등록 결과 ===")
    for result := range results {
        fmt.Printf("Agent ID: %s\n", result.AgentID)
        fmt.Printf("Tx Hash: %s\n", result.TxHash)
        fmt.Printf("Block: %d\n", result.BlockNumber)
        fmt.Printf("Gas Used: %d\n\n", result.GasUsed)
    }

    // 에러 확인
    for err := range errors {
        fmt.Printf("No Error: %v\n", err)
    }

    // 크로스 체인 검증
    fmt.Println("=== 크로스 체인 검증 ===")
    ethereumDID := did.GenerateDID(did.ChainEthereum, ethAddress)
    kaiaDID := did.GenerateDID(did.ChainKaia, ethAddress)

    verifier := did.NewCrossChainVerifier(manager.GetResolver())
    valid, err := verifier.VerifyCrossChain([]did.AgentDID{
        ethereumDID,
        kaiaDID,
    })

    if err != nil {
        fmt.Printf("검증 실패: %v\n", err)
    } else if valid {
        fmt.Println("Yes 모든 체인에서 일관된 데이터 확인")
    } else {
        fmt.Println("No 체인 간 데이터 불일치")
    }
}
```

---

## 요약

Part 3에서 다룬 내용:

1. **DID 심층 분석**: W3C 표준, SAGE 메소드 스펙, 온체인 데이터 구조
2. **블록체인 선택**: Ethereum vs Kaia 비교, 다중 체인 전략
3. **Ethereum 통합**: 클라이언트 구현, ABI 바인딩, 이벤트 리스닝
4. **DID 등록**: 전체 프로세스, 소유권 증명, 가스 최적화
5. **DID 조회**: 조회 메커니즘, 배치 조회, 오프체인 인덱싱
6. **업데이트/비활성화**: 변경 가능 필드, 비활성화 vs 삭제
7. **캐싱 최적화**: 다층 캐싱, Bloom filter, 프리페칭
8. **다중 체인 관리**: 체인 추상화, 크로스 체인 검증, 선택 알고리즘
9. **실전 예제**: 완전한 라이프사이클, 다중 체인 배포

**다음 파트 예고**:

**Part 4: 핸드셰이크 프로토콜 및 세션 관리**에서는:

- HPKE 기반 핸드셰이크 상세 분석
- 클라이언트/서버 구현
- 세션 생성 및 관리
- 이벤트 기반 아키텍처
