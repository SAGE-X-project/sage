# Agent Discovery & Authorization 아키텍처 분석

> **작성일**: 2025-10-09
> **목적**: Agent A가 Agent B를 발견하고 사용할 때, 해당 기능을 Agent Level과 SAGE Level 중 어디에 구현해야 하는지 심도있게 분석

## 목차
1. [문제 정의](#문제-정의)
2. [비판적 분석](#비판적-분석)
3. [추천 아키텍처](#추천-아키텍처)
4. [구현 제안](#구현-제안)
5. [결론](#결론)

---

## 문제 정의

### 시나리오

사용자가 Agent A를 사용 중이고, Agent A가 특정 작업을 수행하기 위해 Agent B의 기능이 필요한 상황:

```
예시: 여행 계획 Agent A가 결제 처리를 위해 Payment Agent B를 호출

사용자: "제주도 여행 예약해줘"
Agent A (여행 계획):
  ├─ 항공권 검색 (자체 기능)
  ├─ 호텔 검색 (자체 기능)
  └─ 결제 필요 ❓ → Agent B (결제 처리)에게 위임
```

### Agent A가 Agent B를 사용하기 위해 필요한 것

1. **Discovery (발견)**
   - Agent B가 존재하는지 알아야 함
   - Agent B의 DID (Decentralized Identifier)
   - Agent B의 엔드포인트 (gRPC 주소)
   - Agent B의 기능(Capabilities) 목록

2. **Authorization (인가)**
   - **사용자 동의**: Agent A가 Agent B를 사용해도 되는지 사용자에게 물어봐야 함
   - **권한 범위**: Agent B의 어떤 기능까지 사용할 수 있는지 (예: 잔액 조회만 vs. 결제 실행까지)
   - **시간 제한**: 언제까지 유효한지 (1회성, 1시간, 1일, 영구 등)

3. **Connection (연결)**
   - Agent B와 HPKE 핸드셰이크 수행
   - 암호화된 세션 생성
   - 메시지 송수신

### 핵심 질문

**이러한 기능들을 어디에 구현해야 하는가?**
- **Option A**: Agent Level (각 Agent가 직접 구현)
- **Option B**: SAGE Level (플랫폼이 제공)

---

## 비판적 분석

두 가지 접근 방식의 장단점을 심도있게 분석합니다.

### Option A: Agent Level 코드에 구현

#### 아키텍처

```go
// my_travel_agent.go (Agent A - 여행 계획 Agent)
package main

type TravelAgent struct {
    name     string
    registry AgentRegistryClient  // Agent가 직접 레지스트리 관리
    did      string
}

func (a *TravelAgent) BookTrip(destination string) error {
    // 1. Agent가 직접 Payment Agent 검색
    paymentAgents, err := a.registry.Search(SearchCriteria{
        Capability: "payment",
        Rating: ">= 4.5",
    })
    if err != nil {
        return err
    }

    // 2. Agent가 직접 사용자 동의 UI 표시
    consent := a.showConsentDialog(
        "결제 처리를 위해 Payment Agent를 사용하시겠습니까?",
        paymentAgents[0].Capabilities,
    )
    if !consent {
        return errors.New("user denied")
    }

    // 3. Agent가 직접 핸드셰이크 수행
    conn, err := a.connectToAgent(paymentAgents[0].Endpoint, paymentAgents[0].DID)
    if err != nil {
        return err
    }

    // 4. Agent가 직접 메시지 암호화/전송
    return a.sendPaymentRequest(conn, amount)
}
```

#### 장점

1. **최대 유연성**
   - 각 Agent가 자신의 비즈니스 로직에 최적화된 discovery 구현 가능
   - 특수한 요구사항 충족 가능 (예: private registry, 특정 검색 알고리즘)

2. **독립성**
   - SAGE에 종속되지 않음
   - 다른 플랫폼으로 이식 가능
   - SAGE 프로토콜 변경에 영향 적음

3. **맞춤화된 UX**
   - Agent 특성에 맞는 동의 화면 구성
   - 도메인별 권한 정책 (예: 의료 Agent는 HIPAA 준수)

#### 단점

1. **🚨 중복 코드 (Code Duplication)**
   ```go
   // 모든 Agent가 동일한 코드를 작성해야 함
   func (a *TravelAgent) connectToAgent(...)   // 여행 Agent
   func (a *ShoppingAgent) connectToAgent(...) // 쇼핑 Agent
   func (a *HealthAgent) connectToAgent(...)   // 건강 Agent
   // ... 수백 개 Agent가 동일한 핸드셰이크 코드 중복
   ```

2. **🚨 보안 위험 (Security Risks)**
   - **잘못된 구현**: Agent 개발자가 HPKE, 서명 검증, nonce 관리를 잘못 구현할 가능성
   - **취약점 패치 어려움**: 보안 버그 발견 시 모든 Agent 업데이트 필요
   - **검증 부담**: 각 Agent의 보안 코드를 개별적으로 감사해야 함

3. **🚨 일관성 부족 (Inconsistent UX)**
   ```
   Agent A의 동의 화면: "Payment Agent를 사용하시겠습니까?"
   Agent B의 동의 화면: "다음 권한을 부여하시겠습니까?"
   Agent C의 동의 화면: "결제 서비스 연결을 허용하시겠습니까?"

   → 사용자 혼란, 신뢰도 하락
   ```

4. **🚨 감사 불가능 (No Auditing)**
   - Agent 간 호출 추적 어려움
   - 보안 사고 발생 시 원인 분석 어려움
   - Compliance 요구사항 충족 어려움 (예: GDPR, SOC2)

5. **🚨 유지보수 악몽**
   - SAGE 프로토콜 변경 시 모든 Agent 수정 필요
   - 버전 불일치 문제 (Agent A는 v1, Agent B는 v2 프로토콜 사용)

---

### Option B: SAGE Level 기능 제공

#### 아키텍처

```go
// my_travel_agent.go (Agent A - SAGE API 사용)
package main

type TravelAgent struct {
    name string
    sage *sage.Client  // ✅ SAGE가 제공하는 클라이언트
    did  string
}

func (a *TravelAgent) BookTrip(destination string) error {
    // 1. ✅ SAGE API로 Agent 검색
    consent, err := a.sage.RequestAgentConnection(context.Background(), sage.ConnectionRequest{
        CallerAgent:  a.did,
        TargetAgent:  "did:sage:payment-processor",  // 또는 Capability 기반 검색
        Purpose:      "여행 결제 처리",
        Capabilities: []string{"create_payment"},
        Duration:     1 * time.Hour,
    })
    if err != nil {
        return err
    }

    if !consent.Granted {
        return errors.New("user denied")
    }

    // 2. ✅ SAGE가 자동으로 핸드셰이크 수행 및 세션 반환
    session := consent.Session

    // 3. ✅ Agent는 비즈니스 로직에만 집중
    return a.callPaymentAPI(session, PaymentRequest{
        Amount:      totalCost,
        Description: "Jeju Trip Payment",
    })
}
```

#### SAGE Level 구현 예시

```go
// sage/agent_service.go (SAGE 플랫폼)
package sage

type AgentDiscoveryService interface {
    // 1️⃣ Agent 검색 (SAGE 레지스트리)
    DiscoverAgent(ctx context.Context, criteria SearchCriteria) (*AgentMetadata, error)

    // 2️⃣ 사용자 동의 요청 (표준화된 UI)
    RequestUserConsent(ctx context.Context, req ConsentRequest) (*ConsentResult, error)

    // 3️⃣ 자동 핸드셰이크 수행 및 세션 반환
    EstablishConnection(ctx context.Context, peerDID string) (*Session, error)

    // 4️⃣ Agent 간 호출 전체를 하나의 API로 추상화
    RequestAgentConnection(ctx context.Context, req ConnectionRequest) (*ConnectionResult, error)
}

type ConsentRequest struct {
    CallerAgent  string   // "did:sage:travel-agent"
    TargetAgent  string   // "did:sage:payment-processor"
    Purpose      string   // "여행 결제 처리"
    Capabilities []string // ["create_payment"]
    Duration     time.Duration
}

type ConnectionResult struct {
    Granted bool
    Session *Session          // 암호화된 세션 (이미 핸드셰이크 완료)
    Token   *CapabilityToken  // 권한 토큰 (재사용 가능)
}
```

#### 장점

1. **✅ 보안성 (Security by Default)**
   - **검증된 구현**: SAGE 팀이 HPKE, 서명, nonce 관리를 안전하게 구현
   - **중앙화된 패치**: 보안 버그 발견 시 SAGE만 업데이트하면 모든 Agent 자동 적용
   - **Secure by Default**: Agent 개발자가 보안을 신경 쓰지 않아도 안전

2. **✅ 재사용성 (Code Reuse)**
   - 모든 Agent가 동일한 SAGE API 사용
   - DRY (Don't Repeat Yourself) 원칙 준수
   - 개발 속도 향상 (핸드셰이크 코드 작성 불필요)

3. **✅ 일관된 사용자 경험 (Consistent UX)**
   ```
   표준화된 동의 화면:

   ┌─────────────────────────────────────────────┐
   │ 🔐 Agent 연결 요청                           │
   │                                             │
   │ 여행 Agent가 Payment Agent를 사용하려 합니다 │
   │                                             │
   │ 목적: 여행 결제 처리                         │
   │                                             │
   │ 요청 권한:                                   │
   │  ✓ 결제 생성 (create_payment)               │
   │                                             │
   │ 유효 기간: 1시간                             │
   │                                             │
   │ [허용] [거부] [세부 설정]                    │
   └─────────────────────────────────────────────┘

   → 모든 Agent에서 동일한 UI → 사용자 신뢰 증가
   ```

4. **✅ 감사 가능성 (Auditability)**
   ```go
   // SAGE가 모든 inter-agent 호출을 자동 로깅
   type AuditLog struct {
       Timestamp    time.Time
       CallerDID    string  // "did:sage:travel-agent"
       TargetDID    string  // "did:sage:payment-processor"
       Purpose      string  // "여행 결제 처리"
       Capabilities []string
       UserConsent  bool
       SessionID    string
       Result       string  // "success" | "denied" | "error"
   }
   ```
   - Compliance 요구사항 충족 (GDPR, SOC2, HIPAA)
   - 보안 사고 시 추적 가능
   - 비정상 패턴 탐지 가능

5. **✅ 정책 적용 (Policy Enforcement)**
   ```go
   // Organization-wide policies
   type OrganizationPolicy struct {
       Rules []PolicyRule
   }

   type PolicyRule struct {
       Condition string  // "medical_agent && financial_agent"
       Action    string  // "deny"
       Reason    string  // "의료 Agent는 금융 Agent 호출 금지 (규정 준수)"
   }

   // SAGE가 자동으로 정책 검사
   func (s *AgentService) RequestAgentConnection(ctx context.Context, req ConnectionRequest) (*ConnectionResult, error) {
       // 조직 정책 검사
       if s.policy.Violates(req.CallerAgent, req.TargetAgent) {
           return nil, ErrPolicyViolation
       }
       // ...
   }
   ```

6. **✅ Rate Limiting & Quota 관리**
   ```go
   // SAGE가 자동으로 호출 제한 관리
   type QuotaPolicy struct {
       MaxCallsPerHour   int
       MaxConcurrentSessions int
       MaxTokensPerDay   int
   }
   ```

#### 단점

1. **❌ Lock-in (종속성)**
   - SAGE 플랫폼에 강하게 종속
   - 다른 플랫폼으로 이식 시 코드 수정 필요

2. **❌ 유연성 제한**
   - 특수한 discovery 요구사항 충족 어려움 (예: private registry)
   - 표준화된 UI만 사용 가능

**반론**:
- 대부분의 Agent는 표준 기능만 필요 (80/20 법칙)
- 특수한 경우는 플러그인 메커니즘으로 확장 가능
- Lock-in은 보안/편의성과의 트레이드오프

---

## 추천 아키텍처: Hybrid Approach (계층적 접근)

완전히 Agent Level도, 완전히 SAGE Level도 아닌 **계층적 접근**을 제안합니다.

### 설계 원칙

```
┌──────────────────────────────────────────────┐
│         Application Layer (Agent)            │  ← 비즈니스 로직
│  "어떤 Agent를 호출할지, 어떤 데이터를 전달할지" │
├──────────────────────────────────────────────┤
│       SAGE Platform Layer (Infrastructure)   │  ← 인프라
│  "Discovery, Consent, Handshake, Encryption" │
└──────────────────────────────────────────────┘
```

**유사 사례**: 현대 클라우드 아키텍처의 **Service Mesh** 패턴
- **Istio**: 서비스 간 통신, 인증, 암호화, 감사를 관리
- **Application**: 비즈니스 로직만 처리
- **Agent ≈ Application**, **SAGE ≈ Service Mesh**

### Layer 1: SAGE Core (Platform Layer)

SAGE가 제공하는 핵심 인프라:

```go
// sage/agent_service.go
package sage

type AgentService struct {
    registry   *AgentRegistry
    handshake  *handshake.Server
    sessionMgr *session.Manager
    auditLog   *AuditLogger
    policy     *PolicyEngine
}

// 🎯 핵심 API: Agent 연결 요청 (Discovery + Consent + Handshake 통합)
func (s *AgentService) RequestAgentConnection(ctx context.Context, req ConnectionRequest) (*ConnectionResult, error) {
    // 1️⃣ Agent 검색 (DID 또는 Capability 기반)
    target, err := s.registry.Resolve(ctx, req.TargetAgent)
    if err != nil {
        return nil, fmt.Errorf("agent not found: %w", err)
    }

    // 2️⃣ 정책 검사 (조직 정책, Rate Limiting 등)
    if err := s.policy.Check(req.CallerAgent, req.TargetAgent, req.Capabilities); err != nil {
        return nil, fmt.Errorf("policy violation: %w", err)
    }

    // 3️⃣ 사용자 동의 요청 (표준화된 UI)
    consent, err := s.showConsentDialog(ctx, ConsentRequest{
        CallerAgent:  req.CallerAgent,
        TargetAgent:  req.TargetAgent,
        Purpose:      req.Purpose,
        Capabilities: req.Capabilities,
        Duration:     req.Duration,
    })
    if err != nil {
        return nil, err
    }

    if !consent.Granted {
        s.auditLog.Log(AuditLog{
            Event:       "connection_denied",
            CallerDID:   req.CallerAgent,
            TargetDID:   req.TargetAgent,
            UserConsent: false,
        })
        return &ConnectionResult{Granted: false}, nil
    }

    // 4️⃣ HPKE 핸드셰이크 자동 수행
    session, err := s.handshake.EstablishConnection(ctx, target.DID, target.Endpoint)
    if err != nil {
        return nil, fmt.Errorf("handshake failed: %w", err)
    }

    // 5️⃣ Capability Token 발급
    token := s.issueCapabilityToken(CapabilityToken{
        Issuer:       "did:sage:platform",
        Subject:      req.TargetAgent,
        Audience:     req.CallerAgent,
        Capabilities: req.Capabilities,
        NotBefore:    time.Now(),
        Expiration:   time.Now().Add(req.Duration),
    })

    // 6️⃣ 감사 로그 기록
    s.auditLog.Log(AuditLog{
        Event:        "connection_established",
        CallerDID:    req.CallerAgent,
        TargetDID:    req.TargetAgent,
        Purpose:      req.Purpose,
        Capabilities: req.Capabilities,
        UserConsent:  true,
        SessionID:    session.ID,
        TokenID:      token.JTI,
    })

    return &ConnectionResult{
        Granted: true,
        Session: session,
        Token:   token,
    }, nil
}

// 🎯 표준화된 동의 UI
func (s *AgentService) showConsentDialog(ctx context.Context, req ConsentRequest) (*ConsentResult, error) {
    // UI 렌더링 (Web/CLI/Mobile 지원)
    return s.uiRenderer.ShowConsentScreen(ConsentScreenData{
        CallerName:   s.registry.GetName(req.CallerAgent),
        TargetName:   s.registry.GetName(req.TargetAgent),
        Purpose:      req.Purpose,
        Capabilities: req.Capabilities,
        Duration:     req.Duration,
    })
}
```

### Layer 2: Agent Level (Application Layer)

Agent는 SAGE API를 사용하여 비즈니스 로직에만 집중:

```go
// my_travel_agent.go
package main

import (
    "github.com/sage-x-project/sage/client"
)

type TravelAgent struct {
    name string
    sage *client.SAGEClient
    did  string
}

func (a *TravelAgent) BookTrip(ctx context.Context, req BookingRequest) error {
    // 1. 항공권 검색 (자체 기능)
    flights := a.searchFlights(req.Destination, req.Date)

    // 2. 호텔 검색 (자체 기능)
    hotels := a.searchHotels(req.Destination, req.Date)

    totalCost := flights.Price + hotels.Price

    // 3. ✅ 결제를 위해 Payment Agent 호출 (SAGE API 사용)
    result, err := a.sage.RequestAgentConnection(ctx, client.ConnectionRequest{
        CallerAgent:  a.did,
        TargetAgent:  "did:sage:payment-processor",
        Purpose:      "여행 결제 처리",
        Capabilities: []string{"create_payment"},
        Duration:     1 * time.Hour,
    })
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    if !result.Granted {
        return errors.New("user denied payment authorization")
    }

    // 4. ✅ SAGE가 제공한 세션으로 메시지 전송
    paymentResp, err := result.Session.SendMessage(ctx, PaymentRequest{
        Amount:      totalCost,
        Description: "Jeju Trip Payment",
        Reference:   req.BookingID,
    })
    if err != nil {
        return fmt.Errorf("payment failed: %w", err)
    }

    // 5. 예약 완료
    return a.confirmBooking(req.BookingID, paymentResp.TransactionID)
}
```

### 확장성: 플러그인 메커니즘

특수한 요구사항을 위한 확장 포인트 제공:

```go
// sage/plugin.go
package sage

type DiscoveryPlugin interface {
    // Custom Agent 검색 로직
    Discover(ctx context.Context, criteria interface{}) (*AgentMetadata, error)
}

type ConsentPlugin interface {
    // Custom 동의 화면 및 로직
    RequestConsent(ctx context.Context, req ConsentRequest) (*ConsentResult, error)
}

// Agent가 플러그인 등록 가능
func (s *SAGEClient) RegisterDiscoveryPlugin(plugin DiscoveryPlugin) {
    s.discoveryPlugin = plugin
}

// 예: Private Registry 플러그인
type PrivateRegistryPlugin struct {
    registryURL string
}

func (p *PrivateRegistryPlugin) Discover(ctx context.Context, criteria interface{}) (*AgentMetadata, error) {
    // 내부 레지스트리에서 검색
    return p.queryPrivateRegistry(criteria)
}
```

---

## 구현 제안: Capability-based Authorization

OAuth 2.0 및 Capability-based Security 모델을 참고한 구현:

### Capability Token 구조

```go
// internal/authorization/capability.go
package authorization

type CapabilityToken struct {
    // Standard JWT Claims
    Issuer     string    `json:"iss"` // "did:sage:platform"
    Subject    string    `json:"sub"` // "did:sage:payment-processor" (Agent B)
    Audience   string    `json:"aud"` // "did:sage:travel-agent" (Agent A)
    IssuedAt   time.Time `json:"iat"`
    NotBefore  time.Time `json:"nbf"`
    Expiration time.Time `json:"exp"`
    JTI        string    `json:"jti"` // Token ID (UUID)

    // SAGE-specific Claims
    Capabilities []Capability `json:"cap"`     // 권한 목록
    Purpose      string       `json:"purpose"` // "여행 결제 처리"
    UserConsent  bool         `json:"consent"` // 사용자 동의 여부
    SessionID    string       `json:"sid"`     // 연결된 세션 ID

    // Signature (Ed25519)
    Signature []byte `json:"sig"`
}

type Capability struct {
    Action   string                 `json:"action"`   // "create_payment"
    Resource string                 `json:"resource"` // "payment/*"
    Metadata map[string]interface{} `json:"metadata"` // {"max_amount": 10000}
}
```

### Agent 간 호출 시 Token 사용

```go
// Agent A → Agent B 메시지 전송 시
func (s *Session) SendMessage(ctx context.Context, msg interface{}) (*Response, error) {
    // 1. Capability Token을 HTTP-like Authorization 헤더에 포함
    headers := map[string]string{
        "Authorization": "Bearer " + s.capabilityToken.Encode(),
        "Content-Type":  "application/json",
    }

    // 2. A2A 메시지 구성
    a2aMsg := &a2a.Message{
        TaskId:    "agent/request@v1",
        ContextId: s.contextID,
        Metadata:  headers,
        Content:   encryptMessage(msg, s.sessionKey),
    }

    // 3. 전송
    return s.a2aClient.SendMessage(ctx, a2aMsg)
}
```

### Agent B의 Token 검증

```go
// Agent B (Payment Processor)
func (b *PaymentAgent) HandleRequest(ctx context.Context, req *a2a.Message) (*a2a.Response, error) {
    // 1. Authorization 헤더에서 Token 추출
    authHeader := req.Metadata["Authorization"]
    tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

    // 2. SAGE가 발급한 Token인지 검증
    token, err := b.sage.VerifyCapabilityToken(ctx, tokenStr)
    if err != nil {
        return nil, ErrUnauthorized
    }

    // 3. Capability 확인
    if !token.HasCapability("create_payment") {
        return nil, ErrForbidden
    }

    // 4. 비즈니스 로직 수행
    return b.processPayment(req)
}
```

### Token 재사용 (성능 최적화)

```go
// Agent A가 Token을 캐싱하여 재사용
type TokenCache struct {
    mu     sync.RWMutex
    tokens map[string]*CapabilityToken  // key: targetDID
}

func (a *TravelAgent) callPaymentAgent(ctx context.Context, amount float64) error {
    targetDID := "did:sage:payment-processor"

    // 1. 캐시에 유효한 토큰이 있는지 확인
    a.tokenCache.mu.RLock()
    token := a.tokenCache.tokens[targetDID]
    a.tokenCache.mu.RUnlock()

    if token != nil && !token.IsExpired() {
        // ✅ 기존 토큰 재사용 (사용자에게 다시 묻지 않음)
        return a.sendPaymentRequest(token, amount)
    }

    // 2. 토큰 없음 → SAGE에 새로 요청 (사용자 동의 필요)
    result, err := a.sage.RequestAgentConnection(ctx, ConnectionRequest{
        TargetAgent: targetDID,
        // ...
    })
    if err != nil {
        return err
    }

    // 3. 토큰 캐싱
    a.tokenCache.mu.Lock()
    a.tokenCache.tokens[targetDID] = result.Token
    a.tokenCache.mu.Unlock()

    return a.sendPaymentRequest(result.Token, amount)
}
```

**이점**:
- 사용자가 한 번만 동의하면 Token 만료 전까지 재사용
- 매번 핸드셰이크 불필요 (성능 향상)
- Token은 제한된 권한만 포함 (최소 권한 원칙)

---

## 결론

### 최종 권장사항: **SAGE Level 기능 제공**

**이유**:

1. **🔒 보안이 최우선**
   - 암호화 프로토콜은 전문가가 구현해야 함
   - Agent 개발자에게 맡기면 취약점 발생 확률 높음
   - **"Secure by Default"** 원칙 준수

2. **👥 일관된 사용자 경험**
   - 모든 Agent에서 동일한 동의 화면
   - 사용자 신뢰 증가
   - 학습 곡선 감소

3. **📊 감사 및 Compliance**
   - 모든 inter-agent 호출 추적
   - GDPR, SOC2, HIPAA 등 규정 준수
   - 보안 사고 시 빠른 대응

4. **🌐 산업 표준 준수**
   - OAuth 2.0, Capability-based Security 모델
   - Service Mesh 패턴 (Istio, Linkerd)
   - Zero-Trust 아키텍처

5. **⚡ 개발 속도 향상**
   - Agent 개발자는 비즈니스 로직에만 집중
   - 인프라 코드 작성 불필요
   - Time-to-Market 단축

### 역할 분담

| Layer | 책임 | 예시 |
|-------|------|------|
| **SAGE Level** | Discovery, Consent, Handshake, Encryption, Auditing, Policy | `RequestAgentConnection()` |
| **Agent Level** | 비즈니스 로직, 도메인 지식, 사용자 인터페이스 | `BookTrip()`, `ProcessPayment()` |

### 비유

```
SAGE = 전화 시스템 (인프라)
  - 전화 연결
  - 암호화
  - 통화 품질 보장
  - 통화 기록

Agent = 사용자 (애플리케이션)
  - 누구에게 전화할지 결정
  - 무슨 말을 할지 결정
  - 전화 시스템은 그냥 사용
```

Agent 개발자가 전화 시스템의 암호화 프로토콜을 구현할 필요는 없습니다. **SAGE가 안전한 인프라를 제공**하고, **Agent는 그 위에서 비즈니스 가치를 창출**하는 것이 올바른 아키텍처입니다.

---

## 참고 자료

- **OAuth 2.0**: [RFC 6749](https://www.rfc-editor.org/rfc/rfc6749.html)
- **Capability-based Security**: [Wikipedia](https://en.wikipedia.org/wiki/Capability-based_security)
- **Service Mesh Pattern**: [Istio Architecture](https://istio.io/latest/docs/ops/deployment/architecture/)
- **Zero Trust Architecture**: [NIST SP 800-207](https://csrc.nist.gov/publications/detail/sp/800-207/final)
