# SAGE Framework Improvement Plan

## 1. 현재 상태 분석

### 1.1 누락된 기능
- **Agent 등록 상태 확인**: Contract에 `IsAgentRegistered(did)` 같은 함수가 없음
- **Configuration 관리**: 설정 파일 시스템이 없음
- **Proxy Server 지원**: Gas-less transaction을 위한 프록시 서버 지원 부재
- **키 관리 확장성**: 다양한 알고리즘과 주소 생성 방식 지원 부족

### 1.2 기존 강점
- 모듈화된 구조로 확장이 용이함
- 인터페이스 기반 설계로 새로운 구현체 추가가 쉬움
- 기본적인 키 관리 시스템이 이미 구현되어 있음

## 2. 개선 사항

### 2.1 Contract 개선

#### Ethereum Smart Contract 수정
```solidity
// ISageRegistry.sol에 추가
function isAgentRegistered(string calldata did) external view returns (bool);
function getAgentRegistrationStatus(string calldata did) external view returns (bool registered, bool active);
```

#### Solana Program 수정
```rust
// sage-registry/src/lib.rs에 추가

/// Check if agent is registered by DID
pub fn is_agent_registered(ctx: Context<CheckAgent>, did: String) -> Result<bool> {
    let agent_pubkey = Pubkey::find_program_address(
        &[b"agent", did.as_bytes()],
        ctx.program_id,
    ).0;
    
    // Check if account exists
    let account_info = ctx.accounts.agent.to_account_info();
    Ok(!account_info.data_is_empty())
}

/// Get agent registration status by DID
pub fn get_agent_registration_status(
    ctx: Context<CheckAgent>, 
    did: String
) -> Result<RegistrationStatus> {
    let agent = &ctx.accounts.agent;
    
    // If account doesn't exist, return not registered
    if agent.to_account_info().data_is_empty() {
        return Ok(RegistrationStatus {
            is_registered: false,
            is_active: false,
            registered_at: 0,
            agent_pubkey: Pubkey::default(),
        });
    }
    
    Ok(RegistrationStatus {
        is_registered: true,
        is_active: agent.active,
        registered_at: agent.registered_at,
        agent_pubkey: agent.key(),
    })
}

/// Get agent by owner
pub fn get_agents_by_owner(
    ctx: Context<GetAgentsByOwner>,
    owner: Pubkey,
) -> Result<Vec<Pubkey>> {
    // This would require an index account or off-chain indexing
    // For now, return empty vec with a note
    msg!("Note: Owner-based queries require off-chain indexing");
    Ok(vec![])
}

// 새로운 Account 구조체 추가
#[derive(Accounts)]
#[instruction(did: String)]
pub struct CheckAgent<'info> {
    /// CHECK: Agent account that may or may not exist
    #[account(
        seeds = [b"agent", did.as_bytes()],
        bump,
    )]
    pub agent: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct GetAgentsByOwner<'info> {
    pub registry: Account<'info, Registry>,
}

// 새로운 구조체 정의
#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct RegistrationStatus {
    pub is_registered: bool,
    pub is_active: bool,
    pub registered_at: i64,
    pub agent_pubkey: Pubkey,
}
```

#### Go Client 수정
```go
// did/types.go에 추가
type RegistrationStatus struct {
    IsRegistered bool      `json:"is_registered"`
    IsActive     bool      `json:"is_active"`
    RegisteredAt time.Time `json:"registered_at,omitempty"`
    AgentID      string    `json:"agent_id,omitempty"`
}
```

### 2.2 Configuration Management System

#### sage-config.yaml 구조
```yaml
# sage-config.yaml
version: "1.0"

# Network configuration
networks:
  ethereum:
    mainnet:
      rpc: "https://eth-mainnet.alchemyapi.io/v2/${ALCHEMY_API_KEY}"
      contract: "0x..."
      chain_id: 1
    sepolia:
      rpc: "https://eth-sepolia.alchemyapi.io/v2/${ALCHEMY_API_KEY}"
      contract: "0x..."
      chain_id: 11155111
  
  solana:
    mainnet:
      rpc: "https://api.mainnet-beta.solana.com"
      program_id: "..."
    devnet:
      rpc: "https://api.devnet.solana.com"
      program_id: "..."

# Key management
key_management:
  default_algorithm: "ed25519"
  storage:
    type: "file" # file, memory, hsm
    path: "~/.sage/keys"
    encryption: true
  
  algorithms:
    ed25519:
      enabled: true
    secp256k1:
      enabled: true
      address_format: "ethereum" # ethereum, bitcoin
    rsa:
      enabled: false
      key_size: 2048

# Proxy server configuration
proxy:
  enabled: false
  endpoints:
    - url: "https://api.sage.network/v1/proxy"
      api_key: "${SAGE_PROXY_API_KEY}"
    - url: "https://backup.sage.network/v1/proxy"
      api_key: "${SAGE_PROXY_API_KEY}"
  
  gas_policy:
    max_gas_price: "100 gwei"
    sponsor_address: "0x..."

# Agent registration
registration:
  auto_register: true
  check_interval: "5m"
  retry_policy:
    max_attempts: 3
    backoff: "exponential"
    initial_delay: "1s"
    max_delay: "30s"

# Security
security:
  signature_algorithms:
    - "EdDSA"
    - "ECDSA"
    - "RSA-PSS"
  
  verification:
    require_active_agent: true
    max_clock_skew: "5m"
    cache_ttl: "10m"

# Logging
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### 2.3 새로운 모듈 구조

```
sage/
├── config/                    # NEW: Configuration management
│   ├── config.go             # Config loader and validator
│   ├── types.go              # Config types
│   └── env.go                # Environment variable support
├── registry/                  # NEW: Agent registry client
│   ├── client.go             # Registry client interface
│   ├── ethereum_client.go    # Ethereum-specific implementation
│   ├── solana_client.go      # Solana-specific implementation
│   └── status_checker.go     # Registration status checker
├── proxy/                     # NEW: Proxy server support
│   ├── client.go             # Proxy client
│   ├── transaction.go        # Transaction builder
│   └── gas_manager.go        # Gas management
└── crypto/
    └── algorithms/           # NEW: Algorithm extensions
        ├── registry.go       # Algorithm registry
        ├── rsa/             # RSA support
        └── bls/             # BLS signatures
```

### 2.4 주요 인터페이스 설계

#### Configuration Interface
```go
// config/types.go
type Config struct {
    Version  string                     `yaml:"version"`
    Networks map[string]NetworkConfig   `yaml:"networks"`
    KeyMgmt  KeyManagementConfig       `yaml:"key_management"`
    Proxy    ProxyConfig               `yaml:"proxy"`
    Registry RegistrationConfig        `yaml:"registration"`
    Security SecurityConfig            `yaml:"security"`
}

type ConfigLoader interface {
    Load(path string) (*Config, error)
    LoadFromEnv() (*Config, error)
    Validate(config *Config) error
    Watch(path string, callback func(*Config)) error
}
```

#### Registry Client Interface
```go
// registry/client.go
type RegistryClient interface {
    // Check if agent is registered
    IsAgentRegistered(ctx context.Context, did string) (bool, error)
    
    // Get detailed registration status
    GetRegistrationStatus(ctx context.Context, did string) (*RegistrationStatus, error)
    
    // Register agent with auto-retry
    RegisterWithRetry(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error)
    
    // Watch registration status
    WatchRegistration(ctx context.Context, did string) (<-chan RegistrationStatus, error)
}
```

#### Proxy Client Interface
```go
// proxy/client.go
type ProxyClient interface {
    // Submit transaction through proxy
    SubmitTransaction(ctx context.Context, tx *Transaction) (*TransactionResult, error)
    
    // Estimate gas through proxy
    EstimateGas(ctx context.Context, tx *Transaction) (*GasEstimate, error)
    
    // Check proxy health
    HealthCheck(ctx context.Context) error
}

type Transaction struct {
    Chain       Chain
    Method      string
    Params      []interface{}
    GasLimit    uint64
    GasPrice    string
    Nonce       uint64
    Signature   []byte
}
```

### 2.5 구현 우선순위

1. **Phase 1 (높음)**: Agent 등록 상태 확인 기능
   - Ethereum Contract 수정 및 배포
   - Solana Program 수정 및 배포
   - Go client 업데이트 (양 체인 지원)
   - 테스트 작성

2. **Phase 2 (높음)**: Configuration 시스템
   - Config 로더 구현
   - 환경 변수 지원
   - Validation 로직

3. **Phase 3 (중간)**: 키 관리 확장
   - Algorithm registry
   - 새로운 알고리즘 지원
   - Address format 지원

4. **Phase 4 (중간)**: Proxy Server 지원
   - Proxy client 구현
   - Gas management
   - Transaction relay

## 3. 구현 일정

- **Week 1-2**: Contract 수정 및 등록 상태 확인 기능
- **Week 3-4**: Configuration 시스템 구현
- **Week 5-6**: 키 관리 확장 및 알고리즘 추가
- **Week 7-8**: Proxy server 지원 구현

## 4. 테스트 계획

### 4.1 단위 테스트
- 각 새로운 모듈에 대한 단위 테스트
- Mock을 사용한 독립적 테스트
- 90% 이상 커버리지 목표

### 4.2 통합 테스트
- Contract와의 통합 테스트
- Configuration 로딩 및 적용 테스트
- Proxy server 통합 테스트

### 4.3 E2E 테스트
- 전체 Agent 등록 및 확인 플로우
- Multi-chain 시나리오 테스트
- Failover 및 retry 테스트

## 5. 마이그레이션 가이드

### 5.1 기존 사용자를 위한 가이드
1. 새로운 contract 주소로 업데이트 (Ethereum 및 Solana)
2. sage-config.yaml 생성 및 설정
3. 코드에서 config 로더 사용

### 5.2 Breaking Changes
- Ethereum Contract interface 변경 (새 함수 추가)
- Solana Program 업데이트 필요 (새 instruction 추가)
- Config 기반 초기화 권장

### 5.3 Solana 특별 고려사항
- PDA (Program Derived Address) 기반 Agent 계정 조회
- Owner 기반 조회는 off-chain 인덱싱 필요
- Account 존재 여부 확인을 위한 별도 로직

## 6. 보안 고려사항

- Private key는 절대 config 파일에 직접 저장하지 않음
- 환경 변수 또는 별도의 secure storage 사용
- Proxy server 사용 시 API key 관리 철저
- Config 파일 권한 설정 (600)

## 7. 성능 최적화

- Registration status 캐싱
- Proxy server 로드 밸런싱
- Batch registration 지원
- Connection pooling