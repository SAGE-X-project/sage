# SAGE 프로젝트 상세 가이드 - Part 6B: 실전 통합 가이드

## 목차
1. [시작하기 전에](#1-시작하기-전에)
2. [CLI 도구 사용법](#2-cli-도구-사용법)
3. [Go 프로젝트 통합](#3-go-프로젝트-통합)
4. [Node.js/TypeScript 프로젝트 통합](#4-nodejstypescript-프로젝트-통합)
5. [Python 프로젝트 통합](#5-python-프로젝트-통합)
6. [MCP Tool 보안 추가](#6-mcp-tool-보안-추가)
7. [프로덕션 배포 가이드](#7-프로덕션-배포-가이드)

---

## 1. 시작하기 전에

### 1.1 필수 요구사항

#### 소프트웨어 요구사항

```
┌─────────────────────────────────────────────────────────┐
│  필수 도구 체크리스트                                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [ ] Go 1.24 이상                                        │
│      $ go version                                       │
│      go version go1.24.0 linux/amd64                    │
│                                                         │
│  [ ] Node.js 18 이상 (선택적)                            │
│      $ node --version                                   │
│      v18.17.0                                           │
│                                                         │
│  [ ] Git                                                │
│      $ git --version                                    │
│      git version 2.39.0                                 │
│                                                         │
│  [ ] 블록체인 RPC 접근 (Kaia/Ethereum)                   │
│      - 공개 RPC: https://public-en-kairos.node.kaia.io │
│      - 또는 Alchemy, Infura 등 서비스                    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 블록체인 계정 준비

```bash
# 1. 테스트넷 지갑 주소 필요
# Kairos (Kaia 테스트넷)에서 테스트 토큰 받기

# 방법 1: Kaia Wallet 사용
# https://wallet.kaia.io 에서 지갑 생성

# 방법 2: Metamask 사용
# Network 추가:
# - Network Name: Kaia Kairos Testnet
# - RPC URL: https://public-en-kairos.node.kaia.io
# - Chain ID: 1001
# - Currency Symbol: KAIA

# 2. Faucet에서 테스트 토큰 받기
# https://faucet.kaia.io
# → 주소 입력 → 무료 테스트 KAIA 받기
```

### 1.2 프로젝트 구조 이해

SAGE를 통합할 때 권장하는 디렉토리 구조:

```
your-project/
├── sage/                    # SAGE 관련 설정
│   ├── keys/               # 암호화 키 저장 (Warning .gitignore에 추가!)
│   │   ├── agent.jwk       # Ed25519 서명 키
│   │   └── ethereum.jwk    # Secp256k1 블록체인 키
│   ├── config.yaml         # SAGE 설정 파일
│   └── did.json            # DID 정보 (등록 후 자동 생성)
│
├── src/                    # 애플리케이션 코드
│   ├── main.go             # (Go 예시)
│   └── agent.go
│
├── .env                    # 환경 변수 (Warning .gitignore에 추가!)
├── .gitignore
└── README.md
```

**중요한 보안 설정:**

```bash
# .gitignore 에 반드시 추가!
echo "sage/keys/*.jwk" >> .gitignore
echo "sage/keys/*.pem" >> .gitignore
echo ".env" >> .gitignore
echo "sage/did.json" >> .gitignore
```

---

## 2. CLI 도구 사용법

### 2.1 SAGE CLI 설치

```bash
# 1. SAGE 저장소 클론
git clone https://github.com/sage-x-project/sage.git
cd sage

# 2. CLI 도구 빌드
make build

# 또는 개별 빌드
go build -o build/bin/sage-crypto ./cmd/sage-crypto
go build -o build/bin/sage-did ./cmd/sage-did

# 3. PATH에 추가 (선택적)
export PATH=$PATH:$(pwd)/build/bin

# 4. 설치 확인
sage-crypto --help
sage-did --help
```

### 2.2 sage-crypto: 키 관리

#### 새 키 생성

```bash
# Ed25519 키 생성 (서명용)
sage-crypto generate \
  --type ed25519 \
  --name "my-agent" \
  --output ./sage/keys

# 출력:
# Yes Key generated successfully!
# Key ID: abc123def456
# Type: Ed25519
# Location: ./sage/keys/abc123def456.jwk
```

**생성된 파일 확인:**

```bash
cat ./sage/keys/abc123def456.jwk
```

```json
{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "7v8Ag3MQ...",
  "d": "9x2Bh4Kp...",
  "kid": "abc123def456",
  "alg": "EdDSA"
}
```

#### Ethereum/Kaia용 키 생성

```bash
# Secp256k1 키 생성 (블록체인용)
sage-crypto generate \
  --type secp256k1 \
  --name "blockchain-key" \
  --output ./sage/keys

# Yes Key generated successfully!
# Key ID: xyz789abc012
# Type: Secp256k1
# Ethereum Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb9
```

#### 키 목록 조회

```bash
sage-crypto list --dir ./sage/keys

# 출력:
#  Keys in ./sage/keys:
#
# 1. abc123def456
#    Type: Ed25519
#    Created: 2025-01-15 10:30:00
#    Purpose: signing
#
# 2. xyz789abc012
#    Type: Secp256k1
#    Created: 2025-01-15 10:31:00
#    Purpose: ethereum
#    Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb9
```

#### Ethereum 주소 조회

```bash
sage-crypto address \
  --key ./sage/keys/xyz789abc012.jwk

# 출력:
# Ethereum Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb9
```

#### 메시지 서명

```bash
sage-crypto sign \
  --key ./sage/keys/abc123def456.jwk \
  --message "Hello, SAGE!" \
  --output signature.bin

# Yes Message signed
# Signature: 0x3f7a9e2b...
```

#### 서명 검증

```bash
sage-crypto verify \
  --key ./sage/keys/abc123def456.jwk \
  --message "Hello, SAGE!" \
  --signature signature.bin

# Yes Signature valid!
```

### 2.3 sage-did: DID 관리

#### 환경 변수 설정

```bash
# .env 파일 생성
cat > .env << 'EOF'
# Blockchain RPC
KAIA_RPC_URL=https://public-en-kairos.node.kaia.io
KAIA_CHAIN_ID=1001

# Contract Address (Kairos 테스트넷)
SAGE_REGISTRY_ADDRESS=0x1234567890123456789012345678901234567890

# Gas Payer Private Key
# Warning 테스트넷용만! 메인넷에서는 절대 평문 저장 금지
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE
EOF

# .env 로드
source .env
```

#### DID 등록

```bash
# 블록체인에 에이전트 등록
sage-did register \
  --chain kaia \
  --name "My AI Agent" \
  --description "A helpful AI assistant" \
  --endpoint "https://my-agent.example.com" \
  --capabilities '{"chat":true,"tools":["calculator","weather"]}' \
  --key ./sage/keys/xyz789abc012.jwk \
  --rpc $KAIA_RPC_URL \
  --contract $SAGE_REGISTRY_ADDRESS \
  --private-key $PRIVATE_KEY

# 출력:
# Registering agent did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku on kaia...
#
# Yes Agent registered successfully!
# DID: did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku
# Transaction: 0x7f8a9b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0
# Block: 12,345,678
# Gas Used: 187,432
```

**등록 정보가 자동 저장됨:**

```bash
cat ./sage/keys/did_sage_kaia_5HueCGU8rMjxEXxiPuD5BDku.json
```

```json
{
  "did": "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku",
  "transactionHash": "0x7f8a9b...",
  "blockNumber": 12345678,
  "timestamp": 1705123456,
  "gasUsed": 187432
}
```

#### DID 조회

```bash
# DID Document 조회
sage-did resolve \
  --did "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku" \
  --rpc $KAIA_RPC_URL \
  --contract $SAGE_REGISTRY_ADDRESS

# 출력:
# {
#   "id": "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku",
#   "verificationMethod": [{
#     "id": "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku#key-1",
#     "type": "Ed25519VerificationKey2020",
#     "controller": "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku",
#     "publicKeyMultibase": "z6Mk..."
#   }],
#   "service": [{
#     "id": "#agent-endpoint",
#     "type": "AgentService",
#     "serviceEndpoint": "https://my-agent.example.com"
#   }]
# }
```

#### DID 업데이트

```bash
# 엔드포인트나 capabilities 변경
sage-did update \
  --did "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku" \
  --endpoint "https://new-endpoint.example.com" \
  --capabilities '{"chat":true,"tools":["calculator","weather","image"]}' \
  --key ./sage/keys/xyz789abc012.jwk \
  --rpc $KAIA_RPC_URL \
  --contract $SAGE_REGISTRY_ADDRESS \
  --private-key $PRIVATE_KEY

# Yes Agent updated successfully!
# Transaction: 0x9a8b7c...
```

#### DID 비활성화

```bash
# 에이전트 비활성화 (삭제는 아님)
sage-did deactivate \
  --did "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku" \
  --key ./sage/keys/xyz789abc012.jwk \
  --rpc $KAIA_RPC_URL \
  --contract $SAGE_REGISTRY_ADDRESS \
  --private-key $PRIVATE_KEY

# Yes Agent deactivated successfully!
```

---

## 3. Go 프로젝트 통합

### 3.1 기본 프로젝트 설정

```bash
# 1. 새 Go 프로젝트 생성
mkdir my-sage-agent
cd my-sage-agent
go mod init github.com/myorg/my-sage-agent

# 2. SAGE 의존성 추가
go get github.com/sage-x-project/sage
```

### 3.2 최소 구현 예제

**main.go:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/handshake"
    "github.com/sage-x-project/sage/session"
)

func main() {
    ctx := context.Background()

    // 1. 설정 로드
    config := loadConfig()

    // 2. 키 로드
    keyPair, err := loadKeys()
    if err != nil {
        log.Fatal(err)
    }

    // 3. DID Manager 초기화
    didManager := did.NewManager()
    err = didManager.Configure(did.ChainKaia, &did.RegistryConfig{
        RPCEndpoint:     config.KAIARPC,
        ContractAddress: config.RegistryAddress,
        PrivateKey:      config.PrivateKey,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 4. Session Manager 생성
    sessionManager := session.NewManager(session.DefaultConfig())
    sessionManager.StartCleanupRoutine(10 * time.Minute)

    // 5. Handshake Client/Server 생성
    handshakeClient := handshake.NewClient(
        keyPair,
        didManager,
        nil, // A2A client (실제 구현 필요)
    )

    handshakeServer := handshake.NewServer(
        keyPair,
        didManager,
        nil, // Events (선택적)
    )

    // 6. 다른 에이전트와 통신
    err = communicateWithPeer(
        ctx,
        "did:sage:kaia:OtherAgent",
        handshakeClient,
        sessionManager,
    )
    if err != nil {
        log.Printf("Communication failed: %v", err)
    }

    fmt.Println("Yes Agent initialized successfully!")
}

func communicateWithPeer(
    ctx context.Context,
    peerDID string,
    client *handshake.Client,
    sessionMgr *session.Manager,
) error {
    // 1. 핸드셰이크 수행
    fmt.Printf(" Starting handshake with %s...\n", peerDID)

    // Invitation
    invMsg := handshake.InvitationMessage{
        From: "did:sage:kaia:MyAgent",
        // ... 추가 필드
    }

    _, err := client.Invitation(ctx, invMsg, peerDID)
    if err != nil {
        return fmt.Errorf("invitation failed: %w", err)
    }

    // (Request, Response, Complete 단계 생략 - Part 4 참조)

    // 2. 세션 생성 (핸드셰이크 완료 후)
    sess, err := session.NewSecureSession(/* ... */)
    if err != nil {
        return fmt.Errorf("session creation failed: %w", err)
    }

    sessionMgr.AddSession(sess)

    // 3. 메시지 전송
    plaintext := []byte("Hello from SAGE!")
    encrypted, err := sess.EncryptMessage(plaintext)
    if err != nil {
        return fmt.Errorf("encryption failed: %w", err)
    }

    fmt.Printf(" Sent encrypted message: %d bytes\n", len(encrypted))

    return nil
}

func loadConfig() *Config {
    // 환경 변수나 설정 파일에서 로드
    return &Config{
        KAIARPC:         "https://public-en-kairos.node.kaia.io",
        RegistryAddress: "0x...",
        PrivateKey:      "0x...",
    }
}

func loadKeys() (crypto.KeyPair, error) {
    // JWK 파일에서 키 로드
    store, err := storage.NewFileKeyStorage("./sage/keys")
    if err != nil {
        return nil, err
    }

    return store.Load("my-agent-key-id")
}

type Config struct {
    KAIARPC         string
    RegistryAddress string
    PrivateKey      string
}
```

### 3.3 HTTP 서버 통합

**server.go:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage/did"
)

type Server struct {
    didResolver *did.Resolver
    verifier    *rfc9421.HTTPVerifier
}

func NewServer() *Server {
    return &Server{
        didResolver: did.NewResolver(/* ... */),
        verifier:    rfc9421.NewHTTPVerifier(),
    }
}

// SAGE 인증 미들웨어
func (s *Server) SAGEAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. DID 추출
        agentDID := r.Header.Get("X-Agent-DID")
        if agentDID == "" {
            http.Error(w, "Missing X-Agent-DID header", http.StatusUnauthorized)
            return
        }

        // 2. DID Document 조회
        didDoc, err := s.didResolver.Resolve(r.Context(), agentDID)
        if err != nil {
            http.Error(w, "Failed to resolve DID", http.StatusUnauthorized)
            return
        }

        // 3. RFC 9421 서명 검증
        publicKey := didDoc.VerificationMethod[0].PublicKey
        err = s.verifier.VerifyRequest(r, publicKey)
        if err != nil {
            http.Error(w, fmt.Sprintf("Signature verification failed: %v", err),
                http.StatusUnauthorized)
            return
        }

        // 4. 검증 성공 - 다음 핸들러 호출
        next(w, r)
    }
}

// 보호된 API 엔드포인트
func (s *Server) protectedHandler(w http.ResponseWriter, r *http.Request) {
    // 이 함수에 도달하면 이미 인증됨

    // 비즈니스 로직
    result := map[string]interface{}{
        "message": "This is a protected resource",
        "data":    []string{"item1", "item2"},
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func (s *Server) Start() {
    // 보호된 엔드포인트 등록
    http.HandleFunc("/api/protected",
        s.SAGEAuthMiddleware(s.protectedHandler))

    // 공개 엔드포인트 (인증 불필요)
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    fmt.Println(" Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}
```

### 3.4 완전한 예제

**full_agent.go:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/crypto/storage"
    "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/handshake"
    "github.com/sage-x-project/sage/session"
)

type SAGEAgent struct {
    keyPair         crypto.KeyPair
    myDID           did.AgentDID
    didManager      *did.Manager
    sessionManager  *session.Manager
    handshakeClient *handshake.Client
    handshakeServer *handshake.Server
}

func NewSAGEAgent(keyStoragePath, myDID string, registryConfig *did.RegistryConfig) (*SAGEAgent, error) {
    // 1. 키 로드
    keyStorage, err := storage.NewFileKeyStorage(keyStoragePath)
    if err != nil {
        return nil, fmt.Errorf("failed to create key storage: %w", err)
    }

    keyPair, err := keyStorage.Load("default")
    if err != nil {
        return nil, fmt.Errorf("failed to load key: %w", err)
    }

    // 2. DID Manager 초기화
    didManager := did.NewManager()
    err = didManager.Configure(did.ChainKaia, registryConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to configure DID manager: %w", err)
    }

    // 3. Session Manager
    sessionManager := session.NewManager(&session.Config{
        MaxSessions:        100,
        SessionTTL:         24 * time.Hour,
        CleanupInterval:    10 * time.Minute,
        EnableHealthChecks: true,
    })

    // 4. Handshake Client & Server
    handshakeClient := handshake.NewClient(keyPair, didManager, nil)
    handshakeServer := handshake.NewServer(keyPair, didManager, nil)

    return &SAGEAgent{
        keyPair:         keyPair,
        myDID:           did.AgentDID(myDID),
        didManager:      didManager,
        sessionManager:  sessionManager,
        handshakeClient: handshakeClient,
        handshakeServer: handshakeServer,
    }, nil
}

func (a *SAGEAgent) Start(ctx context.Context) error {
    log.Printf(" Starting SAGE Agent: %s", a.myDID)

    // 세션 정리 루틴 시작
    a.sessionManager.StartCleanupRoutine(10 * time.Minute)

    // 메시지 수신 루프 (실제로는 gRPC 서버 등)
    go a.messageReceiveLoop(ctx)

    log.Println("Yes Agent started successfully")
    return nil
}

func (a *SAGEAgent) SendMessage(ctx context.Context, peerDID string, message []byte) error {
    // 1. 세션 확인 또는 생성
    sess, err := a.getOrCreateSession(ctx, peerDID)
    if err != nil {
        return fmt.Errorf("failed to get session: %w", err)
    }

    // 2. 메시지 암호화
    encrypted, err := sess.EncryptMessage(message)
    if err != nil {
        return fmt.Errorf("failed to encrypt: %w", err)
    }

    // 3. 전송 (실제 구현 필요)
    log.Printf(" Sending %d bytes to %s", len(encrypted), peerDID)

    return nil
}

func (a *SAGEAgent) getOrCreateSession(ctx context.Context, peerDID string) (*session.SecureSession, error) {
    // 1. 기존 세션 확인
    sess := a.sessionManager.GetSessionByPeerDID(peerDID)
    if sess != nil {
        return sess, nil
    }

    // 2. 새 핸드셰이크 시작
    log.Printf(" Starting handshake with %s", peerDID)

    // (핸드셰이크 4단계 수행 - Part 4 참조)
    // ...

    // 3. 세션 생성
    // ...

    return sess, nil
}

func (a *SAGEAgent) messageReceiveLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // 메시지 수신 처리
            time.Sleep(100 * time.Millisecond)
        }
    }
}

func main() {
    // Context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("\n Shutting down...")
        cancel()
    }()

    // Agent 생성
    agent, err := NewSAGEAgent(
        "./sage/keys",
        "did:sage:kaia:MyAgent",
        &did.RegistryConfig{
            RPCEndpoint:     "https://public-en-kairos.node.kaia.io",
            ContractAddress: "0x...",
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Agent 시작
    if err := agent.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // 예제: 메시지 전송
    err = agent.SendMessage(ctx, "did:sage:kaia:OtherAgent", []byte("Hello!"))
    if err != nil {
        log.Printf("Failed to send message: %v", err)
    }

    // 종료 대기
    <-ctx.Done()
    log.Println(" Goodbye!")
}
```

---

## 4. Node.js/TypeScript 프로젝트 통합

### 4.1 프로젝트 설정

```bash
# 1. 새 프로젝트 생성
mkdir my-sage-agent-ts
cd my-sage-agent-ts
npm init -y

# 2. TypeScript 설정
npm install -D typescript @types/node ts-node
npx tsc --init

# 3. SAGE SDK 설치 (npm 패키지 출시 후)
npm install @sage-x-project/sdk

# 또는 로컬 빌드 사용
npm install ethers @noble/ed25519 @noble/curves
```

### 4.2 TypeScript 구현

**src/agent.ts:**

```typescript
import { ethers } from 'ethers';
import * as ed25519 from '@noble/ed25519';

// SAGE 타입 정의
interface SAGEConfig {
    did: string;
    keyPath: string;
    blockchain: {
        chain: 'kaia' | 'ethereum';
        rpcUrl: string;
        contractAddress: string;
    };
}

interface SecureSession {
    id: string;
    peerDID: string;
    encryptionKey: Uint8Array;
    authKey: Uint8Array;
    sendMessage(message: string): Promise<void>;
    close(): void;
}

class SAGEClient {
    private config: SAGEConfig;
    private provider: ethers.Provider;
    private contract: ethers.Contract;
    private sessions: Map<string, SecureSession>;

    constructor(config: SAGEConfig) {
        this.config = config;
        this.sessions = new Map();

        // Blockchain 연결
        this.provider = new ethers.JsonRpcProvider(config.blockchain.rpcUrl);

        // 컨트랙트 연결 (ABI 필요)
        const abi = [
            "function getAgentByDID(string) view returns (tuple(string did, string name, string endpoint, bytes publicKey, bool active))",
            "event AgentRegistered(bytes32 indexed agentId, address indexed owner, string did, uint256 timestamp)"
        ];
        this.contract = new ethers.Contract(
            config.blockchain.contractAddress,
            abi,
            this.provider
        );
    }

    async initialize(): Promise<void> {
        console.log(' Initializing SAGE client...');

        // DID Document 검증
        await this.resolveDID(this.config.did);

        console.log('Yes SAGE client initialized');
    }

    async resolveDID(did: string): Promise<any> {
        try {
            const agent = await this.contract.getAgentByDID(did);

            return {
                did: agent.did,
                name: agent.name,
                endpoint: agent.endpoint,
                publicKey: agent.publicKey,
                active: agent.active,
            };
        } catch (error) {
            throw new Error(`Failed to resolve DID ${did}: ${error}`);
        }
    }

    async getOrCreateSession(peerDID: string): Promise<SecureSession> {
        // 기존 세션 확인
        const existing = this.sessions.get(peerDID);
        if (existing) {
            return existing;
        }

        // 새 세션 생성
        console.log(` Creating session with ${peerDID}`);

        // 1. DID Resolution
        const peerInfo = await this.resolveDID(peerDID);

        // 2. Handshake 수행
        const session = await this.performHandshake(peerDID, peerInfo);

        // 3. 세션 저장
        this.sessions.set(peerDID, session);

        return session;
    }

    private async performHandshake(
        peerDID: string,
        peerInfo: any
    ): Promise<SecureSession> {
        // 핸드셰이크 구현 (Part 4 참조)

        // 임시 키 생성
        const ephemeralPrivate = ed25519.utils.randomPrivateKey();
        const ephemeralPublic = await ed25519.getPublicKey(ephemeralPrivate);

        // ... (Invitation, Request, Response, Complete)

        // 세션 객체 반환
        return {
            id: 'session-id',
            peerDID,
            encryptionKey: new Uint8Array(32),
            authKey: new Uint8Array(32),
            sendMessage: async (message: string) => {
                console.log(` Sending: ${message}`);
                // 암호화 및 전송 구현
            },
            close: () => {
                this.sessions.delete(peerDID);
            },
        };
    }

    on(event: string, callback: Function): void {
        // 이벤트 리스너 등록
        if (event === 'message') {
            // 메시지 수신 처리
        } else if (event === 'session_created') {
            // 세션 생성 이벤트
        }
    }
}

// 사용 예시
async function main() {
    const sage = new SAGEClient({
        did: 'did:sage:kaia:MyAgent',
        keyPath: './sage/keys/agent.jwk',
        blockchain: {
            chain: 'kaia',
            rpcUrl: 'https://public-en-kairos.node.kaia.io',
            contractAddress: '0x...',
        },
    });

    await sage.initialize();

    // 메시지 수신 핸들러
    sage.on('message', async (msg: any) => {
        console.log(` Received: ${msg.plaintext}`);

        // 응답 전송
        const session = await sage.getOrCreateSession(msg.senderDID);
        await session.sendMessage('Response message');
    });

    // 메시지 전송
    const session = await sage.getOrCreateSession('did:sage:kaia:OtherAgent');
    await session.sendMessage('Hello from TypeScript!');
}

main().catch(console.error);
```

### 4.3 Express.js 통합

**src/server.ts:**

```typescript
import express from 'express';
import { SAGEClient } from './agent';

const app = express();
app.use(express.json());

const sage = new SAGEClient({
    did: 'did:sage:kaia:MyServerAgent',
    keyPath: './sage/keys/server.jwk',
    blockchain: {
        chain: 'kaia',
        rpcUrl: 'https://public-en-kairos.node.kaia.io',
        contractAddress: '0x...',
    },
});

// 초기화
await sage.initialize();

// SAGE 인증 미들웨어
const sageAuth = async (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const agentDID = req.header('X-Agent-DID');
    const signature = req.header('Signature');

    if (!agentDID || !signature) {
        return res.status(401).json({ error: 'Missing authentication' });
    }

    try {
        // DID Resolution
        const agentInfo = await sage.resolveDID(agentDID);

        // 서명 검증 (RFC 9421)
        // ... 검증 로직 ...

        // 검증 성공 - 요청 객체에 DID 추가
        (req as any).agentDID = agentDID;
        next();
    } catch (error) {
        res.status(401).json({ error: 'Authentication failed' });
    }
};

// 보호된 엔드포인트
app.post('/api/secure-chat', sageAuth, async (req, res) => {
    const { message } = req.body;
    const agentDID = (req as any).agentDID;

    // 세션 가져오기
    const session = await sage.getOrCreateSession(agentDID);

    // 메시지 처리
    console.log(` Message from ${agentDID}: ${message}`);

    // AI 모델 호출 등...
    const response = `Echo: ${message}`;

    // 암호화된 응답 전송
    await session.sendMessage(response);

    res.json({ success: true });
});

app.listen(3000, () => {
    console.log(' Server running on http://localhost:3000');
});
```

---

## 5. Python 프로젝트 통합

### 5.1 프로젝트 설정

```bash
# 1. 가상환경 생성
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 2. 필수 패키지 설치
pip install web3 pynacl cryptography
```

### 5.2 Python 구현

**sage_agent.py:**

```python
from web3 import Web3
from nacl.signing import SigningKey, VerifyKey
from nacl.public import PrivateKey, PublicKey, Box
import json
import hashlib
import time

class SAGEAgent:
    def __init__(self, did: str, key_path: str, blockchain_config: dict):
        self.did = did
        self.key_path = key_path
        self.blockchain_config = blockchain_config
        self.sessions = {}

        # Web3 연결
        self.w3 = Web3(Web3.HTTPProvider(blockchain_config['rpc_url']))

        # 컨트랙트 연결
        with open('abi/SageRegistry.abi.json', 'r') as f:
            abi = json.load(f)

        self.contract = self.w3.eth.contract(
            address=blockchain_config['contract_address'],
            abi=abi
        )

        # 키 로드
        with open(key_path, 'r') as f:
            key_data = json.load(f)
            # JWK 파싱 및 키 생성
            self.signing_key = self._load_signing_key(key_data)

    def _load_signing_key(self, jwk: dict) -> SigningKey:
        # JWK에서 Ed25519 키 로드
        import base64
        d = base64.urlsafe_b64decode(jwk['d'] + '==')
        return SigningKey(d)

    def resolve_did(self, did: str) -> dict:
        """DID Document 조회"""
        try:
            agent = self.contract.functions.getAgentByDID(did).call()

            return {
                'did': agent[0],
                'name': agent[1],
                'endpoint': agent[3],
                'public_key': agent[4],
                'active': agent[9]
            }
        except Exception as e:
            raise Exception(f"Failed to resolve DID: {e}")

    def get_or_create_session(self, peer_did: str):
        """세션 가져오기 또는 생성"""
        if peer_did in self.sessions:
            return self.sessions[peer_did]

        print(f" Creating session with {peer_did}")

        # DID Resolution
        peer_info = self.resolve_did(peer_did)

        # Handshake 수행
        session = self._perform_handshake(peer_did, peer_info)

        self.sessions[peer_did] = session
        return session

    def _perform_handshake(self, peer_did: str, peer_info: dict):
        """핸드셰이크 수행"""
        # X25519 임시 키 생성
        ephemeral_key = PrivateKey.generate()

        # ... Invitation, Request, Response, Complete ...

        # 세션 객체 생성
        return SAGESession(
            session_id='session-id',
            peer_did=peer_did,
            encryption_key=b'\x00' * 32,  # 실제 키 유도
            auth_key=b'\x00' * 32,
        )

    def send_message(self, peer_did: str, message: str):
        """메시지 전송"""
        session = self.get_or_create_session(peer_did)

        # 암호화
        encrypted = session.encrypt(message.encode())

        print(f" Sending {len(encrypted)} bytes to {peer_did}")

        # 전송 (실제 네트워크 구현 필요)
        # ...

class SAGESession:
    def __init__(self, session_id: str, peer_did: str, encryption_key: bytes, auth_key: bytes):
        self.session_id = session_id
        self.peer_did = peer_did
        self.encryption_key = encryption_key
        self.auth_key = auth_key
        self.seq_number = 0

    def encrypt(self, plaintext: bytes) -> bytes:
        """ChaCha20-Poly1305 암호화"""
        from cryptography.hazmat.primitives.ciphers.aead import ChaCha20Poly1305

        aead = ChaCha20Poly1305(self.encryption_key)

        # Nonce 생성
        nonce = os.urandom(12)

        # AAD 구성
        aad = f"{self.session_id}{self.seq_number}".encode()

        # 암호화
        ciphertext = aead.encrypt(nonce, plaintext, aad)

        self.seq_number += 1

        return nonce + ciphertext

    def decrypt(self, encrypted: bytes) -> bytes:
        """ChaCha20-Poly1305 복호화"""
        from cryptography.hazmat.primitives.ciphers.aead import ChaCha20Poly1305

        aead = ChaCha20Poly1305(self.encryption_key)

        # Nonce 추출
        nonce = encrypted[:12]
        ciphertext = encrypted[12:]

        # AAD 재구성
        aad = f"{self.session_id}{self.seq_number}".encode()

        # 복호화 및 검증
        plaintext = aead.decrypt(nonce, ciphertext, aad)

        return plaintext

# 사용 예시
if __name__ == '__main__':
    agent = SAGEAgent(
        did='did:sage:kaia:MyPythonAgent',
        key_path='./sage/keys/agent.jwk',
        blockchain_config={
            'rpc_url': 'https://public-en-kairos.node.kaia.io',
            'contract_address': '0x...',
        }
    )

    # 메시지 전송
    agent.send_message('did:sage:kaia:OtherAgent', 'Hello from Python!')
```

---

## 6. MCP Tool 보안 추가

### 6.1 MCP란?

**MCP (Model Context Protocol)**은 AI 모델이 외부 도구를 호출할 수 있게 하는 프로토콜입니다.

```
일반적인 MCP 흐름:

AI Model (ChatGPT, Claude)
    ↓
    "날씨를 알려줘"
    ↓
MCP Tool (Weather API)
    ↓
    {"temperature": 72, "conditions": "sunny"}
    ↓
AI Model
    ↓
    "현재 기온은 72도이고 맑습니다"
```

**문제점:** 기본 MCP는 보안이 없음!
- 누구나 Tool을 호출 가능
- 데이터 변조 가능
- 신원 확인 불가

### 6.2 SAGE로 MCP Tool 보안하기

#### Before: 보안 없음

```go
// 보안 없는 MCP Tool
func weatherHandler(w http.ResponseWriter, r *http.Request) {
    var req ToolRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 누구나 호출 가능! 
    location := req.Arguments["location"].(string)

    weather := getWeather(location)
    json.NewEncoder(w).Encode(weather)
}
```

#### After: SAGE 보안 적용

```go
// SAGE로 보안된 MCP Tool
func secureWeatherHandler(w http.ResponseWriter, r *http.Request) {
    // 1. SAGE 검증 추가 (3줄!)
    if err := verifySAGERequest(r); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }

    // 2. 나머지 코드는 동일
    var req ToolRequest
    json.NewDecoder(r.Body).Decode(&req)

    location := req.Arguments["location"].(string)
    weather := getWeather(location)
    json.NewEncoder(w).Encode(weather)
}

func verifySAGERequest(r *http.Request) error {
    // DID 추출
    agentDID := r.Header.Get("X-Agent-DID")
    if agentDID == "" {
        return fmt.Errorf("missing DID")
    }

    // DID Resolution (블록체인에서 공개키 조회)
    didDoc, err := resolver.Resolve(r.Context(), agentDID)
    if err != nil {
        return err
    }

    // RFC 9421 서명 검증
    publicKey := didDoc.VerificationMethod[0].PublicKey
    return verifier.VerifyRequest(r, publicKey)
}
```

### 6.3 완전한 MCP Tool 예제

**secure_calculator.go:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage/did"
)

type CalculatorTool struct {
    resolver *did.Resolver
    verifier *rfc9421.HTTPVerifier
}

func NewCalculatorTool() *CalculatorTool {
    return &CalculatorTool{
        resolver: did.NewResolver(/* config */),
        verifier: rfc9421.NewHTTPVerifier(),
    }
}

// SAGE 검증 미들웨어
func (c *CalculatorTool) SAGEAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        agentDID := r.Header.Get("X-Agent-DID")
        if agentDID == "" {
            http.Error(w, "Missing X-Agent-DID", 401)
            return
        }

        // DID Resolution
        didDoc, err := c.resolver.Resolve(r.Context(), agentDID)
        if err != nil {
            http.Error(w, "DID resolution failed", 401)
            return
        }

        // Capability 체크 (선택적)
        if !hasCapability(didDoc, "calculator") {
            http.Error(w, "Agent not authorized for calculator", 403)
            return
        }

        // 서명 검증
        publicKey := didDoc.VerificationMethod[0].PublicKey
        if err := c.verifier.VerifyRequest(r, publicKey); err != nil {
            http.Error(w, "Signature verification failed", 401)
            return
        }

        // 검증 성공
        next(w, r)
    }
}

// Calculator Tool 핸들러
func (c *CalculatorTool) addHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        A float64 `json:"a"`
        B float64 `json:"b"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }

    result := req.A + req.B

    json.NewEncoder(w).Encode(map[string]interface{}{
        "result": result,
    })
}

func (c *CalculatorTool) Start() {
    // 보호된 엔드포인트
    http.HandleFunc("/add", c.SAGEAuth(c.addHandler))
    http.HandleFunc("/subtract", c.SAGEAuth(c.subtractHandler))
    http.HandleFunc("/multiply", c.SAGEAuth(c.multiplyHandler))
    http.HandleFunc("/divide", c.SAGEAuth(c.divideHandler))

    // Health check (공개)
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    fmt.Println(" Secure Calculator Tool running on :8080")
    http.ListenAndServe(":8080", nil)
}

func hasCapability(didDoc *did.DIDDocument, capability string) bool {
    // DID Document의 capabilities 확인
    // (구현 생략)
    return true
}
```

---

## 7. 프로덕션 배포 가이드

### 7.1 배포 전 체크리스트

```
┌─────────────────────────────────────────────────────────┐
│  프로덕션 배포 체크리스트                                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [ ] 보안                                                │
│      [ ] 개인키를 환경 변수나 비밀 관리자에 저장          │
│      [ ] .gitignore에 키 파일 추가 확인                  │
│      [ ] HTTPS/TLS 사용                                  │
│      [ ] Rate limiting 구현                              │
│      [ ] DDoS 방어 설정                                  │
│                                                         │
│  [ ] 블록체인                                            │
│      [ ] 메인넷 컨트랙트 주소 확인                        │
│      [ ] 충분한 가스 잔액 확보                            │
│      [ ] RPC 엔드포인트 이중화                            │
│      [ ] 트랜잭션 재시도 로직 구현                        │
│                                                         │
│  [ ] 성능                                                │
│      [ ] DID Resolution 캐싱 활성화                      │
│      [ ] Session 정리 루틴 실행                          │
│      [ ] 메모리 누수 확인                                 │
│      [ ] Load balancing 설정                             │
│                                                         │
│  [ ] 모니터링                                            │
│      [ ] Health check 엔드포인트 구현                    │
│      [ ] 로그 수집 (Sentry, Datadog 등)                 │
│      [ ] 메트릭 모니터링 (Prometheus)                    │
│      [ ] 알람 설정                                       │
│                                                         │
│  [ ] 테스트                                              │
│      [ ] 단위 테스트 작성                                │
│      [ ] 통합 테스트 작성                                │
│      [ ] 부하 테스트 수행                                │
│      [ ] 장애 복구 테스트                                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 7.2 환경 변수 관리

**프로덕션 .env 예시:**

```bash
# .env.production (Warning 절대 Git에 커밋 금지!)

# Application
NODE_ENV=production
PORT=8080

# SAGE Configuration
SAGE_DID=did:sage:kaia:ProductionAgent
SAGE_KEY_STORAGE=/var/secrets/sage/keys

# Blockchain - Mainnet
KAIA_RPC_URL=https://public-en.node.kaia.io
KAIA_CHAIN_ID=8217
SAGE_REGISTRY_ADDRESS=0x0000000000000000000000000000000000000000

# Secrets (use secret manager!)
PRIVATE_KEY=${SECRET_PRIVATE_KEY}  # AWS Secrets Manager
ENCRYPTION_KEY=${SECRET_ENCRYPTION_KEY}

# Monitoring
SENTRY_DSN=https://...
DATADOG_API_KEY=${SECRET_DATADOG_API_KEY}

# Performance
DID_CACHE_TTL=86400  # 24 hours
SESSION_TTL=86400    # 24 hours
MAX_SESSIONS=10000
```

**AWS Secrets Manager 사용:**

```bash
# AWS CLI로 secret 저장
aws secretsmanager create-secret \
    --name sage/production/private-key \
    --secret-string "0x..."

# 애플리케이션에서 로드
PRIVATE_KEY=$(aws secretsmanager get-secret-value \
    --secret-id sage/production/private-key \
    --query SecretString \
    --output text)
```

### 7.3 Docker 배포

**Dockerfile:**

```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /sage-agent ./cmd/agent

# Runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary
COPY --from=builder /sage-agent .

# Copy config (without keys!)
COPY config.yaml .

# Keys는 volume으로 마운트
VOLUME ["/root/keys"]

EXPOSE 8080

CMD ["./sage-agent"]
```

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  sage-agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - KAIA_RPC_URL=${KAIA_RPC_URL}
      - SAGE_REGISTRY_ADDRESS=${SAGE_REGISTRY_ADDRESS}
      - PRIVATE_KEY=${PRIVATE_KEY}
    volumes:
      # 키 파일을 안전하게 마운트
      - ./keys:/root/keys:ro
    restart: unless-stopped
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 7.4 Kubernetes 배포

**deployment.yaml:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sage-agent
spec:
  replicas: 3  # High availability
  selector:
    matchLabels:
      app: sage-agent
  template:
    metadata:
      labels:
        app: sage-agent
    spec:
      containers:
      - name: sage-agent
        image: your-registry/sage-agent:latest
        ports:
        - containerPort: 8080
        env:
        - name: KAIA_RPC_URL
          value: "https://public-en.node.kaia.io"
        - name: SAGE_REGISTRY_ADDRESS
          value: "0x..."
        - name: PRIVATE_KEY
          valueFrom:
            secretKeyRef:
              name: sage-secrets
              key: private-key
        volumeMounts:
        - name: sage-keys
          mountPath: /root/keys
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: sage-keys
        secret:
          secretName: sage-keys
          defaultMode: 0400  # Read-only
---
apiVersion: v1
kind: Service
metadata:
  name: sage-agent
spec:
  selector:
    app: sage-agent
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

**secrets.yaml:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sage-secrets
type: Opaque
data:
  # Base64 인코딩된 값
  private-key: <base64-encoded-key>
---
apiVersion: v1
kind: Secret
metadata:
  name: sage-keys
type: Opaque
data:
  agent.jwk: <base64-encoded-jwk>
```

### 7.5 모니터링 설정

**Prometheus 메트릭:**

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    HandshakesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "sage_handshakes_total",
            Help: "Total number of handshakes",
        },
        []string{"status"},  // success, failed
    )

    ActiveSessions = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "sage_active_sessions",
            Help: "Number of active sessions",
        },
    )

    MessageEncryptionDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "sage_message_encryption_duration_seconds",
            Help:    "Message encryption duration",
            Buckets: prometheus.DefBuckets,
        },
    )

    DIDResolutionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "sage_did_resolution_duration_seconds",
            Help:    "DID resolution duration",
            Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0},
        },
        []string{"cache_hit"},  // true, false
    )
)
```

**사용 예시:**

```go
func (s *SessionManager) AddSession(sess *SecureSession) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.sessions[sess.ID] = sess

    // 메트릭 업데이트
    metrics.ActiveSessions.Set(float64(len(s.sessions)))
}

func (c *Client) Invitation(...) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        metrics.HandshakesTotal.WithLabelValues("success").Inc()
    }()

    // ... handshake logic ...
}
```

### 7.6 로깅 Best Practices

```go
package main

import (
    "go.uber.org/zap"
)

func initLogger() *zap.Logger {
    config := zap.NewProductionConfig()

    // 프로덕션 설정
    config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    config.OutputPaths = []string{
        "stdout",
        "/var/log/sage-agent/app.log",
    }

    logger, _ := config.Build()
    return logger
}

func main() {
    logger := initLogger()
    defer logger.Sync()

    logger.Info("Agent starting",
        zap.String("did", "did:sage:kaia:MyAgent"),
        zap.String("version", "1.0.0"),
    )

    // 구조화된 로깅
    logger.Info("Handshake initiated",
        zap.String("peer_did", "did:sage:kaia:OtherAgent"),
        zap.Duration("timeout", 30*time.Second),
    )

    // 에러 로깅
    if err != nil {
        logger.Error("Handshake failed",
            zap.Error(err),
            zap.String("peer_did", peerDID),
            zap.Int("retry_count", retries),
        )
    }
}
```

---

## 결론

Part 6B에서는 SAGE의 실전 통합 방법을 다루었습니다:

### 핵심 내용 요약

1. **시작하기**
   - 필수 요구사항 및 준비
   - 프로젝트 구조 설계
   - 보안 설정

2. **CLI 도구**
   - sage-crypto: 키 생성 및 관리
   - sage-did: DID 등록 및 관리
   - 실전 명령어 예시

3. **언어별 통합**
   - Go: 완전한 에이전트 구현
   - TypeScript/Node.js: Express.js 통합
   - Python: Web3.py 활용

4. **MCP Tool 보안**
   - 3줄 코드로 보안 추가
   - Before/After 비교
   - 완전한 예제

5. **프로덕션 배포**
   - 배포 체크리스트
   - Docker & Kubernetes
   - 모니터링 및 로깅

### 다음 단계

**Part 6C**에서 다룰 내용:
- 일반적인 문제 및 해결 방법
- 성능 최적화 기법
- 보안 Best Practices
- FAQ

---

**문서 정보**
- 작성일: 2025-01-15
- 버전: 1.0
- Part: 6B/6C
- 이전: [Part 6A - Complete Data Flow](DETAILED_GUIDE_PART6A_KO.md)
- 다음: [Part 6C - Troubleshooting and Best Practices](DETAILED_GUIDE_PART6C_KO.md)
