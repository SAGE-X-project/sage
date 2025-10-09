# SAGE DID Package

SAGE (Secure Agent Guarantee Engine) 프로젝트에서 AI 에이전트를 위한 탈중앙화 식별자(DID) 기능을 제공하는 Go 패키지입니다.

## 주요 기능

- **멀티체인 지원**: Ethereum (Sepolia 배포 완료) 및 Solana (개발중)
- **에이전트 등록**: 블록체인에 고유한 DID로 AI 에이전트 등록
- **DID 조회**: 블록체인에서 에이전트 메타데이터와 공개키 검색
- **메타데이터 검증**: 온체인 데이터와 에이전트 정보 검증
- **에이전트 관리**: 메타데이터 업데이트 및 에이전트 비활성화
- **소유자 기반 검색**: 특정 주소가 소유한 모든 에이전트 조회
- **RFC-9421 통합**: SAGE의 서명 검증 시스템과 연동
- **HPKE/KEM 지원**: 서명 키와 키 캡슐화 공개키 모두 저장
- **팩토리 패턴**: 다양한 블록체인을 위한 유연한 클라이언트 생성

## 설치

```bash
go get github.com/sage-x-project/sage/did
```

## 아키텍처

### 패키지 구조

```
did/
├── types.go              # 핵심 타입과 인터페이스
├── did.go                # DID 파싱 및 생성
├── client.go             # 클라이언트 인터페이스 정의
├── manager.go            # DID 매니저 (registry/resolver/verifier 조율)
├── factory.go            # 체인별 클라이언트 생성을 위한 ClientFactory
├── registry.go           # MultiChainRegistry 구현
├── resolver.go           # MultiChainResolver 구현
├── verification.go       # MetadataVerifier 구현
├── utils.go              # 유틸리티 함수
├── ethereum/             # Ethereum 블록체인 클라이언트
│   ├── client.go        # Ethereum DID 작업
│   ├── resolver.go      # Ethereum 전용 조회
│   ├── abi.go           # 컨트랙트 ABI 정의
│   └── SageRegistryV2.abi.json # 컨트랙트 ABI JSON
└── solana/              # Solana 블록체인 클라이언트 (개발중)
    ├── client.go        # Solana DID 작업
    └── resolver.go      # Solana 전용 조회
```

### Core 모듈과의 통합

DID 모듈은 SAGE core 모듈과 원활하게 작동하도록 설계되었습니다:

1. **DID 모듈**: 블록체인에서 에이전트 메타데이터와 공개키 검색
2. **Core 모듈**: DID 데이터를 사용하여 RFC-9421 서명 검증 수행
3. **검증 서비스**: DID 조회와 서명 검증을 조율

## 빌드 방법

### CLI 도구 빌드

```bash
# 프로젝트 루트에서 실행
go build -o sage-did ./cmd/sage-did

# 또는 go install 사용
go install ./cmd/sage-did
```

### 테스트 실행

```bash
# 모든 테스트 실행
go test ./did/...

# 상세 출력과 함께 테스트
go test -v ./did/...

# 특정 패키지 테스트
go test ./did
go test ./did/ethereum
go test ./did/solana
```

## 사용 방법

### 1. 프로그래밍 방식 사용

#### DID 매니저 생성

```go
package main

import (
    "context"
    "github.com/sage-x-project/sage/did"
)

func main() {
    // DID 매니저 생성
    manager := did.NewManager()
    
    // Ethereum 설정
    ethConfig := &did.RegistryConfig{
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/your-api-key",
        ContractAddress: "0x1234567890abcdef...",
        PrivateKey:      "your-private-key", // 가스비용
    }
    manager.Configure(did.ChainEthereum, ethConfig)
    
    // Solana 설정
    solConfig := &did.RegistryConfig{
        RPCEndpoint:     "https://api.mainnet-beta.solana.com",
        ContractAddress: "YourProgramID11111111111111111111",
        PrivateKey:      "your-private-key", // 트랜잭션 수수료용
    }
    manager.Configure(did.ChainSolana, solConfig)
}
```

#### AI 에이전트 등록

```go
import (
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
)

// 키 쌍 생성 (Solana는 Ed25519, Ethereum은 Secp256k1)
keyPair, _ := keys.GenerateEd25519KeyPair()

// 등록 요청 생성
req := &did.RegistrationRequest{
    DID:         "did:sage:solana:agent001",
    Name:        "My AI Agent",
    Description: "지능형 어시스턴트",
    Endpoint:    "https://api.myagent.com",
    Capabilities: map[string]interface{}{
        "chat": true,
        "code": true,
        "search": false,
    },
    KeyPair: keyPair,
}

// 에이전트 등록
ctx := context.Background()
result, err := manager.RegisterAgent(ctx, did.ChainSolana, req)
if err != nil {
    panic(err)
}

fmt.Printf("에이전트 등록 완료! TX: %s\n", result.TransactionHash)
```

#### 에이전트 메타데이터 조회

```go
// 에이전트 DID 조회
agentDID := did.AgentDID("did:sage:ethereum:agent001")
metadata, err := manager.ResolveAgent(ctx, agentDID)
if err != nil {
    panic(err)
}

fmt.Printf("에이전트 이름: %s\n", metadata.Name)
fmt.Printf("엔드포인트: %s\n", metadata.Endpoint)
fmt.Printf("활성 상태: %v\n", metadata.IsActive)
```

#### 검증 서비스와의 통합

```go
import (
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
)

// DID resolver와 함께 검증 서비스 생성
verifier := core.NewVerificationService(manager)

// 에이전트 메시지 검증
message := &rfc9421.Message{
    AgentDID:  "did:sage:ethereum:agent001",
    Body:      []byte("AI 에이전트로부터의 메시지"),
    Signature: signature,
    // ... 기타 필드
}

result, err := verifier.VerifyAgentMessage(ctx, message, opts)
if result.Valid {
    fmt.Println("메시지가 성공적으로 검증되었습니다!")
}
```

### 2. CLI 도구 사용

#### 에이전트 등록

```bash
# Ethereum에 에이전트 등록
./sage-did register \
    --chain ethereum \
    --name "나의 어시스턴트" \
    --endpoint "https://api.myagent.com" \
    --description "AI 코딩 어시스턴트" \
    --capabilities '{"chat":true,"code":true}' \
    --key agent-key.jwk \
    --private-key "0x..." # 가스비용

# 저장소의 키로 Solana에 등록
./sage-did register \
    --chain solana \
    --name "Solana 에이전트" \
    --endpoint "https://api.solana-agent.com" \
    --storage-dir ./keys \
    --key-id my-agent-key \
    --rpc "https://api.devnet.solana.com" # 테스트용 devnet 사용
```

#### DID 조회

```bash
# 에이전트 메타데이터 조회
./sage-did resolve did:sage:ethereum:agent001

# 메타데이터를 파일로 저장
./sage-did resolve did:sage:solana:agent002 \
    --output agent-metadata.json \
    --format json

# 커스텀 RPC 엔드포인트
./sage-did resolve did:sage:ethereum:agent001 \
    --rpc "https://eth-mainnet.g.alchemy.com/v2/your-key"
```

#### 소유자별 에이전트 목록 조회

```bash
# Ethereum 주소가 소유한 모든 에이전트 조회
./sage-did list \
    --chain ethereum \
    --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

# JSON 출력으로 Solana 에이전트 조회
./sage-did list \
    --chain solana \
    --owner 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM \
    --format json \
    --output my-agents.json
```

#### 에이전트 메타데이터 업데이트

```bash
# 에이전트 이름과 엔드포인트 업데이트
./sage-did update did:sage:ethereum:agent001 \
    --name "업데이트된 에이전트 이름" \
    --endpoint "https://new-api.myagent.com" \
    --key agent-key.jwk

# 기능 업데이트
./sage-did update did:sage:solana:agent002 \
    --capabilities '{"chat":true,"code":true,"image":true}' \
    --storage-dir ./keys \
    --key-id my-agent-key
```

#### 에이전트 비활성화

```bash
# 에이전트 비활성화 (확인 필요)
./sage-did deactivate did:sage:ethereum:agent001 \
    --key agent-key.jwk

# 확인 프롬프트 건너뛰기
./sage-did deactivate did:sage:solana:agent002 \
    --storage-dir ./keys \
    --key-id my-agent-key \
    --yes
```

#### 메타데이터 검증

```bash
# 로컬 메타데이터를 블록체인과 비교 검증
./sage-did verify did:sage:ethereum:agent001 \
    --metadata local-metadata.json

# 커스텀 RPC로 검증
./sage-did verify did:sage:solana:agent002 \
    --metadata agent-data.json \
    --rpc "https://api.mainnet-beta.solana.com"
```

## 블록체인 설정

### Ethereum 설정

| 네트워크 | RPC 엔드포인트 | SageRegistryV2 주소 | 상태 |
|---------|---------------|-------------------|------|
| Mainnet | https://eth-mainnet.g.alchemy.com/v2/{key} | TBD | 계획됨 |
| Sepolia | https://eth-sepolia.g.alchemy.com/v2/{key} | `0x487d45a678eb947bbF9d8f38a67721b13a0209BF` | **배포 완료** |
| Holesky | https://eth-holesky.g.alchemy.com/v2/{key} | TBD | 계획됨 |

**참고**: 현재 테스트는 Sepolia 테스트넷 사용을 권장합니다.

### Solana 설정

| 네트워크 | RPC 엔드포인트 | 프로그램 ID | 상태 |
|---------|---------------|------------|------|
| Mainnet | https://api.mainnet-beta.solana.com | TBD | 계획됨 |
| Devnet | https://api.devnet.solana.com | TBD | 개발중 |
| Testnet | https://api.testnet.solana.com | TBD | 개발중 |

**참고**: Solana 통합은 현재 개발 중입니다. 기본 클라이언트 구현은 존재하지만 온체인 프로그램 배포가 필요합니다.

## DID 형식

SAGE DID는 다음 형식을 따릅니다:
```
did:sage:<chain>:<agent-id>
```

예시:
- `did:sage:ethereum:agent001`
- `did:sage:solana:agent_abc123`

## 실제 사용 예제

### 1. 전체 에이전트 생명주기

```bash
# 1. 블록체인에 맞는 키 생성
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./keys --key-id solana-agent

# 2. Solana에 에이전트 등록
./sage-did register \
    --chain solana \
    --name "AI Assistant v1" \
    --endpoint "https://assistant.example.com/api" \
    --description "범용 AI 어시스턴트" \
    --capabilities '{"chat":true,"code":true,"search":true}' \
    --storage-dir ./keys \
    --key-id solana-agent

# 3. 등록 확인 및 조회
./sage-did resolve did:sage:solana:agent_12345 --format json

# 4. 마이그레이션 후 엔드포인트 업데이트
./sage-did update did:sage:solana:agent_12345 \
    --endpoint "https://new.assistant.example.com/api" \
    --storage-dir ./keys \
    --key-id solana-agent

# 5. 주소가 소유한 모든 에이전트 조회
./sage-did list --chain solana \
    --owner YourSolanaAddress111111111111111111111111111

# 6. 더 이상 필요없을 때 에이전트 비활성화
./sage-did deactivate did:sage:solana:agent_12345 \
    --storage-dir ./keys \
    --key-id solana-agent \
    --yes
```

### 2. 멀티체인 에이전트 관리

```bash
# 동일한 에이전트를 여러 체인에 등록
# 먼저 Ethereum에 등록
./sage-crypto generate --type secp256k1 --format storage \
    --storage-dir ./keys --key-id eth-agent

./sage-did register \
    --chain ethereum \
    --name "CrossChain AI" \
    --endpoint "https://api.crosschain-ai.com" \
    --storage-dir ./keys \
    --key-id eth-agent

# 그 다음 Solana에 등록
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./keys --key-id sol-agent

./sage-did register \
    --chain solana \
    --name "CrossChain AI" \
    --endpoint "https://api.crosschain-ai.com" \
    --storage-dir ./keys \
    --key-id sol-agent
```

### 3. 프로그래밍 통합 예제

```go
package main

import (
    "context"
    "log"
    
    "github.com/sage-x-project/sage/core"
    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
)

func main() {
    ctx := context.Background()
    
    // DID 매니저 설정
    manager := did.NewManager()
    manager.Configure(did.ChainEthereum, &did.RegistryConfig{
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/key",
        ContractAddress: "0x...",
    })
    
    // 에이전트 등록
    keyPair, _ := keys.GenerateSecp256k1KeyPair()
    req := &did.RegistrationRequest{
        DID:      did.GenerateDID(did.ChainEthereum, keyPair),
        Name:     "나의 에이전트",
        Endpoint: "https://agent.example.com",
        KeyPair:  keyPair,
    }
    
    result, err := manager.RegisterAgent(ctx, did.ChainEthereum, req)
    if err != nil {
        log.Fatal(err)
    }
    
    // 검증 서비스와 함께 사용
    verifier := core.NewVerificationService(manager)
    
    // 메시지 생성 및 서명
    message := &rfc9421.Message{
        AgentDID: req.DID,
        Body:     []byte("에이전트로부터의 메시지"),
    }
    
    // 메시지 서명
    signer := rfc9421.NewSigner()
    signature, _ := signer.SignMessage(keyPair, message)
    message.Signature = signature
    
    // 메시지 검증
    verifyResult, _ := verifier.VerifyAgentMessage(ctx, message, nil)
    if verifyResult.Valid {
        log.Println("메시지가 검증되었습니다!")
    }
}
```

## 보안 고려사항

1. **개인키 관리**: 개인키를 절대 노출하지 마세요. 환경 변수나 안전한 키 관리 시스템을 사용하세요.

2. **트랜잭션 수수료**: Ethereum과 Solana 모두 트랜잭션 수수료를 위한 네이티브 토큰(ETH/SOL)이 필요합니다.

3. **에이전트 비활성화**: 비활성화된 에이전트는 재활성화할 수 없습니다. 비활성화 전에 확실히 결정하세요.

4. **메타데이터 업데이트**: 에이전트 소유자(키 보유자)만 에이전트를 업데이트하거나 비활성화할 수 있습니다.

## 오류 처리

### 일반적인 오류

#### DID를 찾을 수 없음
```
Error: DID not found in registry
```
지정된 DID가 블록체인에 존재하지 않습니다.

#### 잘못된 키 타입
```
Error: Ethereum requires Secp256k1 keys, got Ed25519
```
각 블록체인에 맞는 키 타입을 사용하세요:
- Ethereum: Secp256k1
- Solana: Ed25519

#### 잔액 부족
```
Error: insufficient funds for gas
```
트랜잭션 서명자가 충분한 ETH/SOL을 보유하고 있는지 확인하세요.

#### 권한 거부
```
Error: only agent owner can update metadata
```
에이전트를 등록한 동일한 키를 사용하세요.

## 고급 기능

### 커스텀 컨트랙트 배포

프라이빗 배포를 위해 자체 DID 레지스트리 컨트랙트를 배포할 수 있습니다:

1. 블록체인에 맞는 컨트랙트 배포
2. 컨트랙트 주소로 DID 매니저 설정
3. 커스텀 `--contract` 플래그와 함께 동일한 CLI 명령 사용

### 오프체인 인덱싱

대규모 쿼리의 성능 향상을 위해:

1. 이벤트 리스너를 사용하여 DID 등록 인덱싱
2. 인덱싱된 데이터를 데이터베이스에 저장
3. `SearchAgents` 기능 구현

## 구현 상태 및 로드맵

### 완료
- Ethereum Sepolia 통합 (SageRegistryV2 배포됨)
- 팩토리 패턴을 활용한 멀티체인 아키텍처
- DID 조회 및 검증
- 에이전트 등록 및 메타데이터 관리
- crypto 패키지 통합 (Ed25519, Secp256k1, X25519)
- RFC-9421 알고리즘 매핑

### 진행중
- Solana 온체인 프로그램 개발
- 핸드셰이크 프로토콜을 위한 HPKE/KEM 키 통합
- 효율적인 쿼리를 위한 오프체인 인덱싱
- 향상된 검색 기능

### 계획됨
- Ethereum 메인넷 배포
- Kaia 블록체인 통합
- 다중 서명 에이전트 소유권
- 위임 및 권한 프레임워크
- 에이전트 기능 검증 시스템

## 키 타입 지원

DID 패키지는 SAGE crypto 패키지와 통합되어 다음을 지원합니다:

| 블록체인 | 서명 키 | KEM 키 (HPKE) | RFC 9421 알고리즘 |
|----------|---------|---------------|------------------|
| Ethereum | Secp256k1 | X25519      | es256k           |
| Solana   | Ed25519   | X25519      | ed25519          |

**참고**: `AgentMetadata`의 `PublicKEMKey` 필드는 HPKE 기반 안전한 핸드셰이크 프로토콜에 사용되는 X25519 공개키를 저장합니다.

## 라이선스

SAGE 프로젝트의 일부로 제공됩니다.