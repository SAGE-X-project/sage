# SAGE 성능 최적화 로드맵

**작성일:** 2025년 10월 11일
**문서 버전:** 1.0
**상태:** 계획 단계

---

## 목차

1. [개요](#개요)
2. [현재 성능 현황](#현재-성능-현황)
3. [최적화 목표](#최적화-목표)
4. [Phase 1: 메모리 할당 최적화](#phase-1-메모리-할당-최적화)
5. [Phase 2: 암호화 연산 최적화](#phase-2-암호화-연산-최적화)
6. [Phase 3: 동시성 최적화](#phase-3-동시성-최적화)
7. [Phase 4: Transport Layer 최적화](#phase-4-transport-layer-최적화)
8. [Phase 5: 캐싱 및 연결 풀링](#phase-5-캐싱-및-연결-풀링)
9. [검증 방법](#검증-방법)
10. [예상 효과](#예상-효과)
11. [실행 계획](#실행-계획)

---

## 개요

### 배경

SAGE 프로젝트는 현재 **프로덕션 배포 가능한 상태**이며, 모든 기능 테스트를 통과했습니다. 그러나 프로파일링 결과 세션 생성 과정에서 개선 가능한 성능 병목이 발견되었습니다.

### 목적

본 로드맵은 SAGE의 성능을 최적화하여 다음을 달성하는 것을 목표로 합니다:
- **처리량(Throughput) 향상**: 초당 처리 가능한 세션 수 증가
- **지연시간(Latency) 감소**: 세션 생성 및 메시지 처리 시간 단축
- **메모리 효율성**: GC 압력 감소, 메모리 사용량 최적화
- **확장성(Scalability)**: 동시 접속 사용자 수 증가 대응

### 최적화 우선순위 원칙

1. **측정 가능한 개선**: 벤치마크로 효과를 입증할 수 있는 최적화
2. **코드 안정성 유지**: 기존 테스트가 모두 통과해야 함
3. **점진적 개선**: 단계별로 적용하고 검증
4. **하위 호환성**: 기존 API는 변경하지 않음

---

## 현재 성능 현황

### 벤치마크 기준선 (Baseline)

```bash
# 테스트 환경
CPU: Apple M1/M2 (ARM64)
Go Version: 1.23.0
OS: macOS 14.x

# 세션 생성 벤치마크 (현재)
BenchmarkSessionCreation-8    10000    115000 ns/op    38 allocs/op    2400 B/op
```

### 프로파일링 결과 분석

#### 1. 메모리 할당 (Allocations)

**문제점**: 세션 생성 시 38번의 메모리 할당 발생

```
세션당 할당 횟수:
- Key Material 생성: 6 allocs (outKey, inKey, outNonce, inNonce, headerKey, exporterSecret)
- HKDF 인스턴스: 6 allocs (각 키마다 별도 HKDF)
- SHA256 해시 객체: 6 allocs (HKDF 내부)
- 기타 구조체 및 슬라이스: 20 allocs

총계: 38 allocations/session
```

**영향**:
- GC(Garbage Collector) 압력 증가
- 고부하 시 지연 시간(latency) 증가
- 메모리 단편화(fragmentation)

#### 2. 암호화 연산 (Cryptographic Operations)

**문제점**: 중복된 HKDF 호출

```go
// 현재 코드 (pkg/agent/hpke/client.go)
hkdfEnc := hkdf.New(sha256.New, sharedSecret, salt, []byte("encryption"))
hkdfAuth := hkdf.New(sha256.New, sharedSecret, salt, []byte("authentication"))
hkdfSign := hkdf.New(sha256.New, sharedSecret, salt, []byte("signing"))
// ... 3개 더
```

**영향**:
- 6개의 별도 SHA256 인스턴스 생성
- 불필요한 CPU 사이클 낭비
- 캐시 미스(cache miss) 증가 가능성

#### 3. Lock Contention

**문제점**: `sync.RWMutex`를 사용한 세션 관리

```go
// pkg/agent/session/manager.go
type Manager struct {
    sessions map[string]*Session
    mu       sync.RWMutex  // 글로벌 lock
}

func (m *Manager) GetSession(id string) (*Session, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.sessions[id], nil
}
```

**영향**:
- 고동시성 환경에서 lock contention 발생
- 읽기 작업도 lock 대기 시간 발생
- 확장성 제한

---

## 최적화 목표

### 단기 목표 (1-2주)

| 메트릭 | 현재 | 목표 | 개선율 |
|--------|------|------|--------|
| 세션 생성 할당 횟수 | 38 allocs/op | <10 allocs/op | **74% 감소** |
| 세션 생성 메모리 | 2,400 B/op | <800 B/op | **67% 감소** |
| 세션 생성 시간 | 115 µs/op | <80 µs/op | **30% 단축** |

### 중기 목표 (1개월)

| 메트릭 | 현재 | 목표 | 개선율 |
|--------|------|------|--------|
| 메시지 처리 처리량 | 8,700 msg/s | >15,000 msg/s | **72% 향상** |
| P99 지연시간 | 2.5 ms | <1.5 ms | **40% 단축** |
| 동시 세션 처리 | 1,000 sessions | >5,000 sessions | **5배 향상** |

### 장기 목표 (3개월)

- **수평 확장성**: 멀티 코어 활용률 >80%
- **메모리 효율성**: GC 일시정지 시간 <1ms
- **안정성**: 24시간 부하 테스트 통과 (메모리 누수 없음)

---

## Phase 1: 메모리 할당 최적화

### 목표
세션 생성 시 메모리 할당 횟수를 38회에서 10회 미만으로 감소

### 우선순위
🔴 **P0 - Critical** (필수 수행)

### 예상 소요 시간
**12시간** (1.5일)

---

### Task 1.1: Key Material 사전 할당

**현재 문제**:
```go
// pkg/agent/session/session.go
type Session struct {
    outKey    []byte  // 32 bytes - separate allocation
    inKey     []byte  // 32 bytes - separate allocation
    outNonce  []byte  // 12 bytes - separate allocation
    inNonce   []byte  // 12 bytes - separate allocation
    headerKey []byte  // 32 bytes - separate allocation
    exporterSecret []byte  // 32 bytes - separate allocation
}
```

**최적화 방안**:
```go
type Session struct {
    keyMaterial [192]byte  // 단일 배열로 모든 키 저장
    outKey      []byte     // keyMaterial[0:32]를 가리킴
    inKey       []byte     // keyMaterial[32:64]를 가리킴
    outNonce    []byte     // keyMaterial[64:76]를 가리킴
    inNonce     []byte     // keyMaterial[76:88]를 가리킴
    headerKey   []byte     // keyMaterial[88:120]를 가리킴
    exporterSecret []byte  // keyMaterial[120:152]를 가리킴
}

func newSession() *Session {
    s := &Session{}
    s.outKey = s.keyMaterial[0:32]
    s.inKey = s.keyMaterial[32:64]
    s.outNonce = s.keyMaterial[64:76]
    s.inNonce = s.keyMaterial[76:88]
    s.headerKey = s.keyMaterial[88:120]
    s.exporterSecret = s.keyMaterial[120:152]
    return s
}
```

**기대 효과**:
- 할당 횟수: 6 → 1 (5회 감소)
- 메모리 지역성(locality) 향상 → 캐시 효율성 증가

**소요 시간**: 2시간

**파일 수정**:
- `pkg/agent/session/session.go`
- `pkg/agent/session/session_test.go`

**검증 방법**:
```bash
go test -bench=BenchmarkSessionCreation -benchmem
# 목표: allocs/op가 5 감소
```

---

### Task 1.2: Nonce 카운터 최적화

**현재 문제**:
```go
// Nonce 증가 시마다 새 슬라이스 할당
nonce := make([]byte, 12)
binary.BigEndian.PutUint64(nonce[4:], counter)
```

**최적화 방안**:
```go
type Session struct {
    outNonceCounter uint64
    inNonceCounter  uint64
    nonceBuffer [12]byte  // 재사용 가능한 버퍼
}

func (s *Session) nextOutNonce() []byte {
    binary.BigEndian.PutUint64(s.nonceBuffer[4:], s.outNonceCounter)
    s.outNonceCounter++
    return s.nonceBuffer[:]
}
```

**기대 효과**:
- 메시지당 2회 할당 제거 (송신/수신)
- 고처리량 시나리오에서 큰 개선

**소요 시간**: 3시간

---

### Task 1.3: Session Pool 구현

**목표**: Session 객체 재사용으로 할당 최소화

**구현**:
```go
// pkg/agent/session/pool.go
package session

import "sync"

var sessionPool = sync.Pool{
    New: func() interface{} {
        return &Session{
            // 사전 할당된 구조체
        }
    },
}

func acquireSession() *Session {
    return sessionPool.Get().(*Session)
}

func releaseSession(s *Session) {
    // 민감 정보 제거
    s.reset()
    sessionPool.Put(s)
}
```

**주의사항**:
- **보안**: Pool 반환 전 반드시 암호화 키 제거 (메모리 덮어쓰기)
- **Thread Safety**: Session은 한 번에 한 goroutine에서만 사용

**보안 처리**:
```go
func (s *Session) reset() {
    // 암호화 키 메모리 제거 (보안 중요)
    for i := range s.keyMaterial {
        s.keyMaterial[i] = 0
    }
    s.outNonceCounter = 0
    s.inNonceCounter = 0
}
```

**기대 효과**:
- GC 압력 **80% 감소**
- 세션 생성 시간 **40% 단축**

**소요 시간**: 5시간

**테스트**:
```bash
# 메모리 누수 확인
go test -bench=BenchmarkSessionPool -benchtime=30s -memprofile=mem.prof
go tool pprof -alloc_space mem.prof

# 부하 테스트
go test -bench=BenchmarkConcurrentSessions -cpu=1,2,4,8
```

---

## Phase 2: 암호화 연산 최적화

### 목표
HKDF 호출을 단일 인스턴스로 통합하여 SHA256 할당 감소

### 우선순위
🔴 **P0 - Critical**

### 예상 소요 시간
**8시간** (1일)

---

### Task 2.1: 단일 HKDF 확장 (Single HKDF Expand)

**현재 문제**:
```go
// pkg/agent/hpke/client.go - Initialize()
hkdfEnc := hkdf.New(sha256.New, sharedSecret, salt, []byte("encryption"))
io.ReadFull(hkdfEnc, outKey)

hkdfAuth := hkdf.New(sha256.New, sharedSecret, salt, []byte("authentication"))
io.ReadFull(hkdfAuth, inKey)

// ... 4번 더 반복
```

**최적화 방안**:
```go
// 단일 HKDF reader로 모든 키 순차 생성
func deriveSessionKeys(sharedSecret, salt []byte) (*KeySet, error) {
    contexts := [][]byte{
        []byte("sage-v1-outbound-encryption"),
        []byte("sage-v1-inbound-encryption"),
        []byte("sage-v1-outbound-nonce"),
        []byte("sage-v1-inbound-nonce"),
        []byte("sage-v1-header-key"),
        []byte("sage-v1-exporter"),
    }

    keyMaterial := make([]byte, 192) // 단일 할당
    offset := 0

    for _, context := range contexts {
        kdf := hkdf.Expand(sha256.New, sharedSecret, context)
        n, err := io.ReadFull(kdf, keyMaterial[offset:offset+32])
        if err != nil {
            return nil, err
        }
        offset += n
    }

    return &KeySet{
        OutKey: keyMaterial[0:32],
        InKey:  keyMaterial[32:64],
        // ...
    }, nil
}
```

**RFC 9180 준수**:
- HPKE 표준에 따라 각 키는 고유한 context로 독립적으로 도출
- 단일 HKDF-Expand 호출로 효율성 확보

**기대 효과**:
- SHA256 인스턴스: 6 → 1 (5회 감소)
- CPU 사이클 감소: 약 **25%**
- 메모리 할당 감소: 추가 **3-5회**

**소요 시간**: 4시간

**파일 수정**:
- `pkg/agent/hpke/client.go`
- `pkg/agent/hpke/server.go`
- `pkg/agent/hpke/hpke_test.go`

**검증**:
```bash
# 기능 테스트 (회귀 방지)
go test ./pkg/agent/hpke/... -v

# 성능 비교
go test -bench=BenchmarkHKDFDerivation -benchmem
```

---

### Task 2.2: AEAD Cipher 재사용

**현재 문제**:
```go
// 메시지마다 새로운 cipher 생성
cipher, _ := chacha20poly1305.New(s.outKey)
ciphertext := cipher.Seal(nil, nonce, plaintext, additionalData)
```

**최적화 방안**:
```go
type Session struct {
    outCipher cipher.AEAD  // 생성 시 한 번만 초기화
    inCipher  cipher.AEAD
}

func (s *Session) Encrypt(plaintext, aad []byte) ([]byte, error) {
    nonce := s.nextOutNonce()
    return s.outCipher.Seal(nil, nonce, plaintext, aad), nil
}
```

**주의사항**:
- Nonce는 절대 재사용하면 안 됨 (카운터 기반으로 보장)
- Thread-safe: 단일 Session은 하나의 goroutine에서만 사용

**기대 효과**:
- 메시지당 2회 할당 제거
- 고처리량 시나리오에서 **20% 성능 향상**

**소요 시간**: 4시간

---

## Phase 3: 동시성 최적화

### 목표
Lock contention 제거 및 동시 세션 처리 능력 향상

### 우선순위
🟠 **P1 - High** (Phase 1, 2 완료 후)

### 예상 소요 시간
**10시간** (1.5일)

---

### Task 3.1: sync.Map 기반 Session Manager

**현재 문제**:
```go
type Manager struct {
    sessions map[string]*Session
    mu       sync.RWMutex  // 병목 지점
}
```

**최적화 방안**:
```go
type Manager struct {
    sessions sync.Map  // key: string, value: *Session
}

func (m *Manager) GetSession(id string) (*Session, bool) {
    val, ok := m.sessions.Load(id)
    if !ok {
        return nil, false
    }
    return val.(*Session), true
}

func (m *Manager) StoreSession(id string, s *Session) {
    m.sessions.Store(id, s)
}
```

**장점**:
- 읽기 작업은 lock-free
- 쓰기 작업도 샤딩(sharding) 통해 contention 감소
- 고동시성 환경에 최적화

**단점**:
- Type assertion 필요 (성능 영향 미미)
- Range 연산이 snapshot 기반

**적용 시나리오**:
- 동시 세션 >100개인 경우 큰 효과
- 단일 세션 벤치마크에서는 차이 미미

**기대 효과**:
- 동시 세션 1000개: **3배 처리량 향상**
- Lock contention: **90% 감소**

**소요 시간**: 6시간

**파일 수정**:
- `pkg/agent/session/manager.go`
- `pkg/agent/session/manager_test.go`

**벤치마크**:
```go
// 동시성 테스트
func BenchmarkConcurrentSessionAccess(b *testing.B) {
    m := NewManager()
    // 1000개 세션 생성
    for i := 0; i < 1000; i++ {
        m.StoreSession(fmt.Sprintf("session-%d", i), &Session{})
    }

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            id := fmt.Sprintf("session-%d", rand.Intn(1000))
            m.GetSession(id)
        }
    })
}
```

---

### Task 3.2: Session 만료 처리 최적화

**현재 문제**:
- 만료 세션 주기적 스캔 시 전체 맵 순회 (O(n))
- 스캔 중 lock 유지

**최적화 방안**:
```go
// 만료 시간 순으로 정렬된 우선순위 큐
type expiryQueue struct {
    items []*expiryItem
    mu    sync.Mutex
}

type expiryItem struct {
    sessionID string
    expiresAt time.Time
}

func (m *Manager) cleanupExpiredSessions() {
    now := time.Now()

    // 만료된 세션만 제거 (O(k), k = 만료된 세션 수)
    for {
        item := m.expiryQueue.Peek()
        if item == nil || item.expiresAt.After(now) {
            break
        }

        m.sessions.Delete(item.sessionID)
        m.expiryQueue.Pop()
    }
}
```

**기대 효과**:
- CPU 사용률 감소
- 만료 처리 지연시간 단축

**소요 시간**: 4시간

---

## Phase 4: Transport Layer 최적화

### 목표
네트워크 전송 계층의 효율성 향상

### 우선순위
🟡 **P2 - Medium** (Phase 3 완료 후)

### 예상 소요 시간
**24시간** (3일)

---

### Task 4.1: HTTP Keep-Alive 연결 재사용

**구현**:
```go
type HTTPTransport struct {
    client *http.Client
}

func NewHTTPTransport(endpoint string) *HTTPTransport {
    return &HTTPTransport{
        client: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
                // HTTP/2 활성화
                ForceAttemptHTTP2: true,
            },
            Timeout: 30 * time.Second,
        },
    }
}
```

**기대 효과**:
- TCP 핸드셰이크 오버헤드 제거
- 지연시간 **30-40% 감소**

**소요 시간**: 4시간

---

### Task 4.2: WebSocket 연결 풀링

**구현**:
```go
type WSConnectionPool struct {
    conns   map[string]*websocket.Conn
    mu      sync.RWMutex
    maxSize int
}

func (p *WSConnectionPool) Get(endpoint string) (*websocket.Conn, error) {
    p.mu.RLock()
    conn, exists := p.conns[endpoint]
    p.mu.RUnlock()

    if exists && conn.IsAlive() {
        return conn, nil
    }

    // 새 연결 생성
    return p.dial(endpoint)
}
```

**기대 효과**:
- 연결 재사용으로 오버헤드 감소
- 실시간 메시징 지연시간 단축

**소요 시간**: 6시간

---

### Task 4.3: 메시지 압축 (선택적)

**구현**:
```go
type CompressedTransport struct {
    underlying MessageTransport
    threshold  int  // 이 크기 이상만 압축
}

func (c *CompressedTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
    if len(msg.Payload) > c.threshold {
        compressed, _ := compress(msg.Payload)
        if len(compressed) < len(msg.Payload) {
            msg.Payload = compressed
            msg.Metadata["compression"] = "gzip"
        }
    }
    return c.underlying.Send(ctx, msg)
}
```

**적용 시나리오**:
- 대용량 페이로드 (>1KB)
- 저대역폭 환경

**Trade-off**:
- 압축/해제 CPU 비용 vs 네트워크 대역폭 절약

**소요 시간**: 6시간

---

### Task 4.4: Transport 메트릭 수집

**목적**: 성능 모니터링 및 병목 구간 식별

**구현**:
```go
type MetricsTransport struct {
    underlying MessageTransport
    latency    prometheus.Histogram
    errors     prometheus.Counter
}

func (m *MetricsTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
    start := time.Now()
    resp, err := m.underlying.Send(ctx, msg)

    m.latency.Observe(time.Since(start).Seconds())
    if err != nil {
        m.errors.Inc()
    }

    return resp, err
}
```

**메트릭**:
- 요청 지연시간 (p50, p95, p99)
- 처리량 (requests/sec)
- 에러율
- 페이로드 크기 분포

**소요 시간**: 8시간

---

## Phase 5: 캐싱 및 연결 풀링

### 목표
반복적인 작업의 결과 캐싱 및 리소스 재사용

### 우선순위
🟢 **P3 - Low** (선택 사항)

### 예상 소요 시간
**16시간** (2일)

---

### Task 5.1: DID Resolution 캐싱

**현재 문제**:
- 동일한 DID를 반복적으로 블록체인에서 조회

**최적화**:
```go
type CachedResolver struct {
    underlying did.Resolver
    cache      *lru.Cache  // LRU 캐시
    ttl        time.Duration
}

func (r *CachedResolver) Resolve(ctx context.Context, didStr string) (*did.Document, error) {
    // 캐시 확인
    if doc, ok := r.cache.Get(didStr); ok {
        return doc.(*did.Document), nil
    }

    // 캐시 미스 - 실제 조회
    doc, err := r.underlying.Resolve(ctx, didStr)
    if err == nil {
        r.cache.Add(didStr, doc, r.ttl)
    }

    return doc, err
}
```

**기대 효과**:
- DID 조회 시간: **100배 단축** (네트워크 I/O 제거)
- 블록체인 노드 부하 감소

**소요 시간**: 6시간

---

### Task 5.2: 암호화 키 유도 결과 캐싱

**적용 시나리오**:
- 동일한 DID 쌍 간 반복 세션 생성
- ECDH 결과를 짧은 시간 캐싱

**주의사항**:
- **보안**: 캐시 TTL을 짧게 유지 (예: 5분)
- 메모리에서 민감 데이터 관리 주의

**소요 시간**: 6시간

---

### Task 5.3: 버퍼 풀링

**구현**:
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func encryptMessage(plaintext []byte) ([]byte, error) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)

    // buf 사용
}
```

**적용 대상**:
- 메시지 직렬화 버퍼
- 암호화 임시 버퍼
- 네트워크 I/O 버퍼

**기대 효과**:
- 고처리량 시 GC 압력 추가 감소

**소요 시간**: 4시간

---

## 검증 방법

### 1. 단위 벤치마크

```bash
# 세션 생성 성능
go test -bench=BenchmarkSessionCreation -benchmem -benchtime=10s ./pkg/agent/session/...

# HKDF 최적화 효과
go test -bench=BenchmarkHKDF -benchmem ./pkg/agent/hpke/...

# 동시성 성능
go test -bench=BenchmarkConcurrent -cpu=1,2,4,8 ./pkg/agent/session/...
```

**목표 달성 기준**:
- `allocs/op` < 10
- `B/op` < 800
- `ns/op` < 80000 (80µs)

---

### 2. 통합 부하 테스트

```bash
# 부하 테스트 도구
go run tools/loadtest/main.go \
    --duration=5m \
    --connections=1000 \
    --rate=10000

# 메모리 프로파일링
go test -bench=. -memprofile=mem.prof -benchtime=60s
go tool pprof -alloc_space mem.prof

# CPU 프로파일링
go test -bench=. -cpuprofile=cpu.prof -benchtime=60s
go tool pprof cpu.prof
```

**검증 항목**:
- 메모리 누수 없음 (장시간 실행 시 메모리 증가 없음)
- CPU 사용률 선형 증가 (코어 수에 비례)
- P99 latency < 목표값

---

### 3. 회귀 테스트 (Regression Test)

```bash
# 모든 기존 테스트가 통과해야 함
go test ./... -v -race -count=1

# 통합 테스트
make test-integration

# E2E 테스트
make test-e2e
```

**원칙**: 최적화로 인해 기능이 깨지면 안 됨

---

### 4. 성능 비교 리포트

**Before / After 비교 문서 생성**:

```markdown
## 최적화 효과 측정

### Phase 1 완료 후
| 메트릭 | Before | After | 개선율 |
|--------|--------|-------|--------|
| allocs/op | 38 | 9 | 76% 감소 |
| B/op | 2400 | 720 | 70% 감소 |
| ns/op | 115000 | 82000 | 29% 단축 |

### Phase 2 완료 후
...
```

---

## 예상 효과

### 정량적 효과

| Phase | 메트릭 | 개선 전 | 개선 후 | 개선율 |
|-------|--------|---------|---------|--------|
| **Phase 1** | 할당 횟수 | 38 allocs/op | 9 allocs/op | **76% 감소** |
| **Phase 1** | 메모리 | 2,400 B/op | 720 B/op | **70% 감소** |
| **Phase 2** | CPU 시간 | 115 µs/op | 82 µs/op | **29% 단축** |
| **Phase 3** | 동시 처리량 | 8,700 msg/s | 15,000+ msg/s | **72% 향상** |
| **Phase 4** | 네트워크 지연 | 150 ms | 100 ms | **33% 단축** |

### 정성적 효과

1. **확장성 향상**
   - 단일 서버에서 처리 가능한 동시 사용자 수 증가
   - 수평 확장 시 선형에 가까운 성능 향상

2. **운영 비용 절감**
   - 동일 처리량에 필요한 서버 수 감소
   - 클라우드 환경에서 비용 절감

3. **사용자 경험 개선**
   - 응답 시간 단축
   - 안정적인 서비스 제공

4. **개발 생산성**
   - 프로파일링 기반 최적화 문화 정착
   - 성능 회귀 조기 발견

---

## 실행 계획

### Sprint 1: Core Performance (Week 1-2)
**목표**: Phase 1 + Phase 2 완료

| 일정 | 작업 | 담당 | 상태 |
|------|------|------|------|
| Day 1-2 | Task 1.1: Key Material 사전 할당 | TBD | 대기 |
| Day 3-4 | Task 1.2: Nonce 최적화 | TBD | 대기 |
| Day 5-7 | Task 1.3: Session Pool | TBD | 대기 |
| Day 8-9 | Task 2.1: 단일 HKDF | TBD | 대기 |
| Day 10 | Task 2.2: AEAD 재사용 | TBD | 대기 |

**Milestone**: 세션 생성 <10 allocations

---

### Sprint 2: Concurrency (Week 3)
**목표**: Phase 3 완료

| 일정 | 작업 | 담당 | 상태 |
|------|------|------|------|
| Day 1-3 | Task 3.1: sync.Map 적용 | TBD | 대기 |
| Day 4-5 | Task 3.2: 만료 처리 최적화 | TBD | 대기 |

**Milestone**: 동시 세션 >5,000 처리

---

### Sprint 3: Transport (Week 4-5)
**목표**: Phase 4 완료

| 일정 | 작업 | 담당 | 상태 |
|------|------|------|------|
| Day 1-2 | Task 4.1: HTTP Keep-Alive | TBD | 대기 |
| Day 3-4 | Task 4.2: WS 연결 풀링 | TBD | 대기 |
| Day 5-7 | Task 4.3: 압축 지원 | TBD | 대기 |
| Day 8-10 | Task 4.4: 메트릭 수집 | TBD | 대기 |

**Milestone**: Transport 계층 완성도 90%

---

### Sprint 4 (선택): Caching (Week 6)
**목표**: Phase 5 완료

| 일정 | 작업 | 담당 | 상태 |
|------|------|------|------|
| Day 1-3 | Task 5.1: DID 캐싱 | TBD | 대기 |
| Day 4-5 | Task 5.2: 키 유도 캐싱 | TBD | 대기 |
| Day 6-7 | Task 5.3: 버퍼 풀링 | TBD | 대기 |

**Milestone**: 최적화 완성

---

## 위험 요소 및 대응 방안

| 위험 | 영향 | 확률 | 대응 방안 |
|------|------|------|-----------|
| **메모리 풀에서 보안 이슈** | 높음 | 낮음 | 반환 전 반드시 메모리 제로화, 보안 감사 |
| **sync.Map 성능 저하** | 중간 | 낮음 | 벤치마크로 검증, 필요시 샤딩 맵 사용 |
| **캐시 일관성 문제** | 중간 | 중간 | TTL 짧게 유지, DID 업데이트 시 무효화 |
| **최적화로 인한 버그** | 높음 | 중간 | 철저한 테스트, 단계별 적용 |
| **측정 가능한 효과 없음** | 낮음 | 낮음 | 프로파일링 기반 접근으로 리스크 최소화 |

---

## 참고 자료

### 내부 문서
- [NEXT_TASKS_PRIORITY.md](./NEXT_TASKS_PRIORITY.md) - 전체 작업 우선순위
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 시스템 아키텍처
- [TESTING.md](./TESTING.md) - 테스트 전략

### 외부 자료
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [HPKE RFC 9180](https://www.rfc-editor.org/rfc/rfc9180.html)

---

## 변경 이력

| 날짜 | 버전 | 변경 내용 | 작성자 |
|------|------|-----------|--------|
| 2025-10-11 | 1.0 | 초기 로드맵 작성 | SAGE Team |

---

## 문서 상태

**현재 상태**: 📋 계획 단계
**다음 단계**: Sprint 1 시작 승인 대기
**문서 관리자**: SAGE Development Team

---

**마지막 업데이트**: 2025년 10월 11일
