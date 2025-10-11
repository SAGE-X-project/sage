# SAGE 프로젝트 상세 가이드 - Part 6C: 문제 해결 및 모범 사례

## 목차
1. [일반적인 문제 및 해결 방법](#1-일반적인-문제-및-해결-방법)
2. [성능 최적화](#2-성능-최적화)
3. [보안 Best Practices](#3-보안-best-practices)
4. [디버깅 가이드](#4-디버깅-가이드)
5. [FAQ](#5-faq)
6. [전체 시리즈 요약](#6-전체-시리즈-요약)

---

## 1. 일반적인 문제 및 해결 방법

### 1.1 키 관리 문제

#### 문제 1: "Key file not found"

```
No Error: failed to load key: open ./sage/keys/abc123.jwk: no such file or directory
```

**원인:**
- 키 파일 경로가 잘못됨
- 키가 아직 생성되지 않음

**해결 방법:**

```bash
# 1. 키 디렉토리 확인
ls -la ./sage/keys

# 2. 키가 없다면 생성
sage-crypto generate --type ed25519 --name my-agent --output ./sage/keys

# 3. 생성된 키 ID 확인
sage-crypto list --dir ./sage/keys

# 4. 올바른 키 ID 사용
# 예: abc123def456.jwk
```

#### 문제 2: "Invalid key format"

```
No Error: failed to parse JWK: invalid character 'x' looking for beginning of value
```

**원인:**
- JWK 파일이 손상됨
- 잘못된 형식의 파일

**해결 방법:**

```bash
# 1. 파일 내용 확인
cat ./sage/keys/abc123.jwk

# 올바른 JWK 형식:
# {
#   "kty": "OKP",
#   "crv": "Ed25519",
#   "x": "...",
#   "d": "..."
# }

# 2. 파일이 손상되었다면 백업에서 복원
cp ./sage/keys/backup/abc123.jwk ./sage/keys/

# 3. 백업이 없다면 새 키 생성
sage-crypto generate --type ed25519 --name new-agent --output ./sage/keys

# Warning 주의: 새 키를 생성하면 블록체인에 다시 등록해야 함
```

#### 문제 3: "Permission denied"

```
No Error: open ./sage/keys/abc123.jwk: permission denied
```

**원인:**
- 파일 권한이 잘못됨

**해결 방법:**

```bash
# 1. 현재 권한 확인
ls -l ./sage/keys/abc123.jwk

# 2. 올바른 권한 설정 (소유자만 읽기/쓰기)
chmod 600 ./sage/keys/abc123.jwk

# 3. 디렉토리 권한도 확인
chmod 700 ./sage/keys
```

### 1.2 블록체인 연결 문제

#### 문제 4: "Connection timeout"

```
No Error: Post "https://public-en-kairos.node.kaia.io": context deadline exceeded
```

**원인:**
- 네트워크 연결 문제
- RPC 엔드포인트 다운
- 방화벽 차단

**해결 방법:**

```bash
# 1. RPC 엔드포인트 테스트
curl -X POST https://public-en-kairos.node.kaia.io \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 응답이 있으면 RPC는 정상:
# {"jsonrpc":"2.0","id":1,"result":"0xbc614e"}

# 2. 대체 RPC 사용
# config.yaml:
blockchain:
  kaia:
    rpc_url: "https://kaia-kairos.blockpi.network/v1/rpc/public"
    # 또는
    rpc_url: "https://kaia-kairos-rpc.allthatnode.com:8551"

# 3. 타임아웃 증가
# Go 코드:
client, err := ethclient.DialContext(
    context.WithTimeout(ctx, 30*time.Second),  // 30초로 증가
    rpcURL,
)
```

#### 문제 5: "Insufficient funds for gas"

```
No Error: insufficient funds for gas * price + value
```

**원인:**
- 지갑에 KAIA/ETH가 부족

**해결 방법:**

```bash
# 1. 잔액 확인
sage-crypto address --key ./sage/keys/blockchain.jwk
# Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb9

# 2. 테스트넷 Faucet에서 토큰 받기
# Kairos: https://faucet.kaia.io
# Sepolia: https://sepoliafaucet.com

# 3. 잔액 확인 (etherscan/kaiascan)
# https://kairos.kaiascan.io/account/0x742d35...

# 4. 또는 다른 지갑에서 전송
```

#### 문제 6: "Transaction underpriced"

```
No Error: replacement transaction underpriced
```

**원인:**
- 가스 가격이 너무 낮음
- 같은 nonce로 재전송 시 가스 가격이 낮음

**해결 방법:**

```go
// 1. 가스 가격을 10% 인상
gasPrice := big.NewInt(250 * 1e9)  // 250 Gwei
newGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(110))
newGasPrice.Div(newGasPrice, big.NewInt(100))  // 275 Gwei

auth.GasPrice = newGasPrice

// 2. 또는 네트워크 권장 가스 가격 사용
gasPrice, err := client.SuggestGasPrice(ctx)
if err != nil {
    return err
}
auth.GasPrice = gasPrice

// 3. EIP-1559 사용 (최신 네트워크)
gasTipCap, _ := client.SuggestGasTipCap(ctx)
auth.GasTipCap = gasTipCap
auth.GasFeeCap = new(big.Int).Mul(gasPrice, big.NewInt(2))
```

### 1.3 DID Resolution 문제

#### 문제 7: "DID not found"

```
No Error: DID not found: did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku
```

**원인:**
- DID가 아직 등록되지 않음
- 잘못된 체인에서 조회
- 컨트랙트 주소가 잘못됨

**해결 방법:**

```bash
# 1. DID 형식 확인
# 올바른 형식: did:sage:{chain}:{identifier}
# 예: did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku

# 2. 체인 확인
# DID의 chain 부분과 RPC URL이 일치하는지 확인

# 3. 블록체인에서 직접 조회
sage-did resolve \
  --did "did:sage:kaia:5HueCGU8rMjxEXxiPuD5BDku" \
  --rpc "https://public-en-kairos.node.kaia.io" \
  --contract "0x..."

# 4. DID가 등록되지 않았다면 등록
sage-did register \
  --chain kaia \
  --name "My Agent" \
  --endpoint "https://my-agent.com" \
  --key ./sage/keys/agent.jwk
```

#### 문제 8: "Cache poisoning suspected"

```
Warning Warning: DID resolution returned different results
```

**원인:**
- 캐시가 오래됨
- DID가 업데이트됨

**해결 방법:**

```go
// 1. 캐시 강제 갱신
resolver.ClearCache(did)
freshDoc, err := resolver.Resolve(ctx, did)

// 2. 캐시 TTL 확인 및 조정
config := &did.ResolverConfig{
    CacheTTL: 1 * time.Hour,  // 기본 24시간에서 1시간으로 단축
}

// 3. 캐시 완전히 비활성화 (디버깅용)
config := &did.ResolverConfig{
    EnableCache: false,
}
```

### 1.4 핸드셰이크 문제

#### 문제 9: "Handshake timeout"

```
No Error: handshake timeout after 30s
```

**원인:**
- 네트워크 지연
- 상대방 에이전트 응답 없음
- 방화벽 차단

**해결 방법:**

```go
// 1. 타임아웃 증가
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

// 2. 재시도 로직 추가
var sess *session.SecureSession
var err error
for attempt := 0; attempt < 3; attempt++ {
    sess, err = performHandshake(ctx, peerDID)
    if err == nil {
        break
    }

    log.Printf("Handshake attempt %d failed: %v", attempt+1, err)
    time.Sleep(time.Second * time.Duration(1<<attempt))  // Exponential backoff
}

// 3. 상대방 엔드포인트 확인
didDoc, _ := resolver.Resolve(ctx, peerDID)
endpoint := didDoc.Service[0].ServiceEndpoint
log.Printf("Peer endpoint: %s", endpoint)

// ping 테스트
resp, err := http.Get(endpoint + "/health")
if err != nil {
    log.Printf("Peer unreachable: %v", err)
}
```

#### 문제 10: "Invalid signature"

```
No Error: signature verification failed
```

**원인:**
- 잘못된 키로 서명
- 메시지가 변조됨
- 클라이언트와 서버의 메시지 형식 불일치

**해결 방법:**

```go
// 1. 서명 디버깅 활성화
verifier := rfc9421.NewHTTPVerifier()
verifier.SetDebug(true)  // 상세 로그 출력

// 2. 서명 베이스 확인
signatureBase := verifier.GetSignatureBase(req)
log.Printf("Signature Base:\n%s", signatureBase)

// 3. 공개키 확인
didDoc, _ := resolver.Resolve(ctx, agentDID)
pubKey := didDoc.VerificationMethod[0].PublicKey
log.Printf("Public Key: %x", pubKey)

// 4. 올바른 알고리즘 사용 확인
// Ed25519: 64 bytes signature
// Secp256k1: 65 bytes signature
log.Printf("Signature length: %d", len(signature))
```

#### 문제 11: "Nonce reuse detected"

```
No Error: nonce has already been used
```

**원인:**
- 메시지 재전송 (재생 공격 방지 작동)
- 클라이언트 버그로 같은 nonce 재사용

**해결 방법:**

```go
// 1. 항상 새 nonce 생성
nonce := make([]byte, 16)
_, err := rand.Read(nonce)
if err != nil {
    return fmt.Errorf("failed to generate nonce: %w", err)
}

// 2. Nonce 캐시 크기 확인
nonceCache := NewNonceCache(10000)  // 10,000개 nonce 저장

// 3. Nonce TTL 확인
nonceCache.Add(nonce, 5*time.Minute)  // 5분 동안 유효
```

### 1.5 세션 문제

#### 문제 12: "Session expired"

```
No Error: session not found or expired
```

**원인:**
- 세션이 만료됨 (기본 24시간)
- 세션이 수동으로 삭제됨

**해결 방법:**

```go
// 1. 세션 TTL 확인
config := &session.Config{
    SessionTTL: 48 * time.Hour,  // 48시간으로 연장
}

// 2. 자동 갱신 활성화
if session.ExpiresAt.Sub(time.Now()) < 1*time.Hour {
    log.Println("Session expiring soon, renewing...")
    newSession, err := renewSession(ctx, peerDID)
    if err != nil {
        return err
    }
    sessionManager.ReplaceSession(oldSessionID, newSession)
}

// 3. 만료 시 자동 재협상
sess, err := sessionManager.GetSession(sessionID)
if err != nil {
    if errors.Is(err, session.ErrSessionNotFound) {
        // 새 핸드셰이크 시작
        sess, err = performHandshake(ctx, peerDID)
    }
}
```

#### 문제 13: "Sequence number out of order"

```
No Error: received sequence number 5, expected 3
```

**원인:**
- 메시지가 순서대로 도착하지 않음
- 네트워크 패킷 재정렬

**해결 방법:**

```go
// 1. Sequence number 윈도우 허용
const SequenceWindow = 10

if msg.SeqNumber > sess.LastSeqNumber &&
   msg.SeqNumber <= sess.LastSeqNumber + SequenceWindow {
    // 허용 범위 내
    sess.LastSeqNumber = msg.SeqNumber
} else {
    return fmt.Errorf("sequence number out of window")
}

// 2. 또는 재정렬 버퍼 사용
type ReorderBuffer struct {
    messages map[uint64]*EncryptedMessage
    nextSeq  uint64
}

func (b *ReorderBuffer) Add(msg *EncryptedMessage) []byte {
    if msg.SeqNumber == b.nextSeq {
        // 순서대로 도착
        b.nextSeq++
        return processMessage(msg)
    } else {
        // 버퍼에 저장
        b.messages[msg.SeqNumber] = msg
        return nil
    }
}
```

---

## 2. 성능 최적화

### 2.1 DID Resolution 최적화

#### 다단계 캐싱 전략

```go
type OptimizedResolver struct {
    // L1: 메모리 캐시 (가장 빠름)
    memCache *sync.Map

    // L2: Redis 캐시 (여러 인스턴스 공유)
    redisCache *redis.Client

    // L3: 로컬 DB 캐시
    dbCache *sql.DB

    // L4: 블록체인 (가장 느림)
    blockchain BlockchainClient
}

func (r *OptimizedResolver) Resolve(ctx context.Context, did string) (*DIDDocument, error) {
    // L1: 메모리 (~1ms)
    if doc, ok := r.memCache.Load(did); ok {
        metrics.CacheHits.WithLabelValues("L1").Inc()
        return doc.(*DIDDocument), nil
    }

    // L2: Redis (~5ms)
    if r.redisCache != nil {
        docBytes, err := r.redisCache.Get(ctx, "did:"+did).Bytes()
        if err == nil {
            var doc DIDDocument
            json.Unmarshal(docBytes, &doc)

            // L1에도 캐싱
            r.memCache.Store(did, &doc)

            metrics.CacheHits.WithLabelValues("L2").Inc()
            return &doc, nil
        }
    }

    // L3: Local DB (~10ms)
    var docJSON string
    err := r.dbCache.QueryRow("SELECT doc FROM did_cache WHERE did = ? AND expires_at > ?",
        did, time.Now()).Scan(&docJSON)
    if err == nil {
        var doc DIDDocument
        json.Unmarshal([]byte(docJSON), &doc)

        // L1, L2에도 캐싱
        r.memCache.Store(did, &doc)
        if r.redisCache != nil {
            r.redisCache.Set(ctx, "did:"+did, docJSON, 24*time.Hour)
        }

        metrics.CacheHits.WithLabelValues("L3").Inc()
        return &doc, nil
    }

    // L4: 블록체인 (~200ms)
    doc, err := r.blockchain.GetAgentByDID(ctx, did)
    if err != nil {
        return nil, err
    }

    // 모든 레벨에 캐싱
    r.cacheDocument(ctx, did, doc)

    metrics.CacheMisses.Inc()
    return doc, nil
}

func (r *OptimizedResolver) cacheDocument(ctx context.Context, did string, doc *DIDDocument) {
    // L1: 메모리
    r.memCache.Store(did, doc)

    // L2: Redis (비동기)
    if r.redisCache != nil {
        go func() {
            docBytes, _ := json.Marshal(doc)
            r.redisCache.Set(ctx, "did:"+did, docBytes, 24*time.Hour)
        }()
    }

    // L3: DB (비동기)
    go func() {
        docBytes, _ := json.Marshal(doc)
        r.dbCache.Exec(
            "INSERT INTO did_cache (did, doc, expires_at) VALUES (?, ?, ?) ON CONFLICT(did) DO UPDATE SET doc=?, expires_at=?",
            did, string(docBytes), time.Now().Add(24*time.Hour),
            string(docBytes), time.Now().Add(24*time.Hour),
        )
    }()
}
```

**성능 비교:**

```
┌──────────────┬──────────┬────────────┐
│ Cache Level  │ Latency  │ Hit Rate   │
├──────────────┼──────────┼────────────┤
│ L1 (Memory)  │ ~1ms     │ 60%        │
│ L2 (Redis)   │ ~5ms     │ 30%        │
│ L3 (DB)      │ ~10ms    │ 8%         │
│ L4 (Chain)   │ ~200ms   │ 2%         │
└──────────────┴──────────┴────────────┘

평균 응답 시간:
0.6*1 + 0.3*5 + 0.08*10 + 0.02*200 = 7.9ms
→ 캐싱 없이는 200ms, 25배 향상!
```

### 2.2 암호화 성능 최적화

#### 키 재사용 및 풀링

```go
type CryptoPool struct {
    cipherPool sync.Pool
}

func NewCryptoPool() *CryptoPool {
    return &CryptoPool{
        cipherPool: sync.Pool{
            New: func() interface{} {
                return &CipherContext{
                    buffer: make([]byte, 0, 4096),
                }
            },
        },
    }
}

func (p *CryptoPool) Encrypt(key []byte, plaintext []byte) ([]byte, error) {
    // 풀에서 cipher 컨텍스트 가져오기
    ctx := p.cipherPool.Get().(*CipherContext)
    defer p.cipherPool.Put(ctx)

    // 재사용 가능한 버퍼 사용
    ctx.buffer = ctx.buffer[:0]

    cipher, _ := chacha20poly1305.New(key)
    nonce := make([]byte, 12)
    rand.Read(nonce)

    ciphertext := cipher.Seal(ctx.buffer, nonce, plaintext, nil)

    // 결과 복사 (버퍼는 재사용됨)
    result := make([]byte, len(nonce)+len(ciphertext))
    copy(result, nonce)
    copy(result[12:], ciphertext)

    return result, nil
}
```

#### 병렬 처리

```go
func (s *SecureSession) EncryptBatch(messages [][]byte) ([][]byte, error) {
    results := make([][]byte, len(messages))
    errs := make([]error, len(messages))

    var wg sync.WaitGroup
    sem := make(chan struct{}, runtime.NumCPU())  // CPU 코어 수만큼 병렬

    for i, msg := range messages {
        wg.Add(1)
        go func(idx int, plaintext []byte) {
            defer wg.Done()

            sem <- struct{}{}  // 세마포어 획득
            defer func() { <-sem }()  // 세마포어 해제

            encrypted, err := s.EncryptMessage(plaintext)
            results[idx] = encrypted
            errs[idx] = err
        }(i, msg)
    }

    wg.Wait()

    // 에러 체크
    for _, err := range errs {
        if err != nil {
            return nil, err
        }
    }

    return results, nil
}
```

### 2.3 네트워크 최적화

#### Connection Pooling

```go
type ConnectionPool struct {
    conns chan *grpc.ClientConn
    mu    sync.Mutex
    opts  []grpc.DialOption
}

func NewConnectionPool(target string, size int) *ConnectionPool {
    p := &ConnectionPool{
        conns: make(chan *grpc.ClientConn, size),
        opts: []grpc.DialOption{
            grpc.WithInsecure(),
            grpc.WithKeepaliveParams(keepalive.ClientParameters{
                Time:                10 * time.Second,
                Timeout:             3 * time.Second,
                PermitWithoutStream: true,
            }),
        },
    }

    // 풀 초기화
    for i := 0; i < size; i++ {
        conn, _ := grpc.Dial(target, p.opts...)
        p.conns <- conn
    }

    return p
}

func (p *ConnectionPool) Get() *grpc.ClientConn {
    return <-p.conns
}

func (p *ConnectionPool) Put(conn *grpc.ClientConn) {
    select {
    case p.conns <- conn:
        // 성공
    default:
        // 풀이 꽉 찼으면 연결 닫기
        conn.Close()
    }
}
```

#### 메시지 배치 처리

```go
type MessageBatcher struct {
    messages chan *Message
    flush    chan struct{}

    batchSize     int
    flushInterval time.Duration
}

func (b *MessageBatcher) Start() {
    ticker := time.NewTicker(b.flushInterval)
    defer ticker.Stop()

    batch := make([]*Message, 0, b.batchSize)

    for {
        select {
        case msg := <-b.messages:
            batch = append(batch, msg)

            if len(batch) >= b.batchSize {
                b.sendBatch(batch)
                batch = batch[:0]
            }

        case <-ticker.C:
            if len(batch) > 0 {
                b.sendBatch(batch)
                batch = batch[:0]
            }

        case <-b.flush:
            if len(batch) > 0 {
                b.sendBatch(batch)
                batch = batch[:0]
            }
        }
    }
}

func (b *MessageBatcher) sendBatch(messages []*Message) {
    // 한 번의 RPC 호출로 여러 메시지 전송
    log.Printf("Sending batch of %d messages", len(messages))
    // ... gRPC call ...
}
```

### 2.4 메모리 최적화

#### 메모리 풀 사용

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func processMessage(msg []byte) error {
    // 풀에서 버퍼 가져오기
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)

    // 버퍼 사용
    n := copy(buf, msg)
    processedData := buf[:n]

    // ... 처리 ...

    return nil
}
```

#### 메모리 누수 방지

```go
// No 메모리 누수 예시
type SessionManager struct {
    sessions map[string]*SecureSession
    // cleanup 루틴이 없으면 세션이 계속 쌓임!
}

// Yes 올바른 구현
type SessionManager struct {
    sessions map[string]*SecureSession
    mu       sync.RWMutex
}

func (m *SessionManager) StartCleanupRoutine(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            m.cleanupExpiredSessions()
        }
    }()
}

func (m *SessionManager) cleanupExpiredSessions() {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now()
    for sessionID, session := range m.sessions {
        if now.After(session.ExpiresAt) {
            delete(m.sessions, sessionID)
            log.Printf("Cleaned up expired session: %s", sessionID)
        }
    }
}
```

---

## 3. 보안 Best Practices

### 3.1 키 관리 Best Practices

#### Yes DO: 하드웨어 키 저장소 사용

```go
// macOS Keychain 사용
import "github.com/keybase/go-keychain"

func saveKeyToKeychain(keyID string, keyData []byte) error {
    item := keychain.NewItem()
    item.SetSecClass(keychain.SecClassGenericPassword)
    item.SetService("sage-agent")
    item.SetAccount(keyID)
    item.SetData(keyData)
    item.SetAccessible(keychain.AccessibleWhenUnlocked)

    return keychain.AddItem(item)
}

func loadKeyFromKeychain(keyID string) ([]byte, error) {
    query := keychain.NewItem()
    query.SetSecClass(keychain.SecClassGenericPassword)
    query.SetService("sage-agent")
    query.SetAccount(keyID)
    query.SetReturnData(true)

    results, err := keychain.QueryItem(query)
    if err != nil {
        return nil, err
    }

    return results[0].Data, nil
}
```

#### No DON'T: 키를 평문으로 저장

```go
// No 절대 하지 마세요!
ioutil.WriteFile("private_key.txt", privateKey, 0644)

// No 환경 변수에 직접 저장도 피하세요
os.Setenv("PRIVATE_KEY", "0x1234...")

// Yes 대신 암호화하거나 비밀 관리자 사용
```

#### Yes DO: 키 회전 (Key Rotation)

```go
func rotateKey(agent *SAGEAgent) error {
    // 1. 새 키 생성
    newKeyPair, err := keys.GenerateEd25519KeyPair()
    if err != nil {
        return err
    }

    // 2. 블록체인에 새 키 등록 (DID 업데이트)
    err = agent.didManager.UpdateDID(ctx, agent.myDID, map[string]interface{}{
        "publicKey": newKeyPair.PublicKey().Bytes(),
    }, newKeyPair)
    if err != nil {
        return err
    }

    // 3. 기존 키 백업
    backupKey(agent.keyPair, "backup/old-key-"+time.Now().Format("20060102"))

    // 4. 새 키로 교체
    agent.keyPair = newKeyPair

    // 5. 모든 활성 세션 무효화 (재협상 필요)
    agent.sessionManager.InvalidateAllSessions()

    log.Println("Yes Key rotation completed")
    return nil
}

// 정기적 실행 (예: 90일마다)
func scheduleKeyRotation(agent *SAGEAgent) {
    ticker := time.NewTicker(90 * 24 * time.Hour)
    go func() {
        for range ticker.C {
            if err := rotateKey(agent); err != nil {
                log.Printf("No Key rotation failed: %v", err)
            }
        }
    }()
}
```

### 3.2 네트워크 보안

#### Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    visitors map[string]*rate.Limiter
    mu       sync.RWMutex
    r        rate.Limit
    b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        visitors: make(map[string]*rate.Limiter),
        r:        r,
        b:        b,
    }
}

func (rl *RateLimiter) GetLimiter(did string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.visitors[did]
    if !exists {
        limiter = rate.NewLimiter(rl.r, rl.b)
        rl.visitors[did] = limiter
    }

    return limiter
}

// HTTP 미들웨어
func RateLimitMiddleware(rateLimiter *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            agentDID := r.Header.Get("X-Agent-DID")

            limiter := rateLimiter.GetLimiter(agentDID)
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// 사용 예시:
// 분당 60 요청, 버스트 100
rateLimiter := NewRateLimiter(1, 100)  // 1 req/sec = 60 req/min
http.Handle("/api/", RateLimitMiddleware(rateLimiter)(apiHandler))
```

#### TLS/HTTPS 강제

```go
func startSecureServer() {
    // TLS 설정
    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS13,  // TLS 1.3 이상만
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
    }

    server := &http.Server{
        Addr:      ":443",
        Handler:   router,
        TLSConfig: tlsConfig,
    }

    log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
}

// HTTP → HTTPS 리다이렉트
func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "https://"+r.Host+r.RequestURI,
        http.StatusMovedPermanently)
}

http.HandleFunc("/", redirectToHTTPS)
go http.ListenAndServe(":80", nil)
```

### 3.3 입력 검증

```go
func validateDID(did string) error {
    // 1. 형식 검증
    if !strings.HasPrefix(did, "did:sage:") {
        return fmt.Errorf("invalid DID prefix")
    }

    parts := strings.Split(did, ":")
    if len(parts) != 4 {
        return fmt.Errorf("invalid DID format")
    }

    // 2. 체인 검증
    chain := parts[2]
    validChains := map[string]bool{
        "kaia": true, "ethereum": true, "solana": true,
    }
    if !validChains[chain] {
        return fmt.Errorf("unsupported chain: %s", chain)
    }

    // 3. Identifier 검증 (Base58)
    identifier := parts[3]
    if len(identifier) < 10 || len(identifier) > 50 {
        return fmt.Errorf("invalid identifier length")
    }

    // Base58 문자 검증
    validChars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
    for _, c := range identifier {
        if !strings.ContainsRune(validChars, c) {
            return fmt.Errorf("invalid Base58 character: %c", c)
        }
    }

    return nil
}

func validateEndpoint(endpoint string) error {
    // 1. URL 파싱
    u, err := url.Parse(endpoint)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }

    // 2. HTTPS만 허용
    if u.Scheme != "https" {
        return fmt.Errorf("only HTTPS endpoints allowed")
    }

    // 3. localhost 금지 (프로덕션)
    if strings.Contains(u.Host, "localhost") || strings.Contains(u.Host, "127.0.0.1") {
        return fmt.Errorf("localhost endpoints not allowed")
    }

    return nil
}
```

### 3.4 감사 로깅

```go
type AuditLogger struct {
    logger *zap.Logger
}

func (a *AuditLogger) LogHandshake(event string, did string, success bool, metadata map[string]interface{}) {
    fields := []zap.Field{
        zap.String("event", event),
        zap.String("peer_did", did),
        zap.Bool("success", success),
        zap.Time("timestamp", time.Now()),
    }

    for k, v := range metadata {
        fields = append(fields, zap.Any(k, v))
    }

    if success {
        a.logger.Info("Handshake event", fields...)
    } else {
        a.logger.Warn("Handshake failed", fields...)
    }
}

func (a *AuditLogger) LogDIDResolution(did string, success bool, cacheHit bool) {
    a.logger.Info("DID resolution",
        zap.String("did", did),
        zap.Bool("success", success),
        zap.Bool("cache_hit", cacheHit),
        zap.Time("timestamp", time.Now()),
    )
}

func (a *AuditLogger) LogMessageSent(sessionID, peerDID string, messageSize int) {
    a.logger.Info("Message sent",
        zap.String("session_id", sessionID),
        zap.String("peer_did", peerDID),
        zap.Int("size_bytes", messageSize),
        zap.Time("timestamp", time.Now()),
    )
}

// 보안 이벤트 (별도 로그 파일)
func (a *AuditLogger) LogSecurityEvent(eventType string, severity string, details map[string]interface{}) {
    fields := []zap.Field{
        zap.String("event_type", eventType),
        zap.String("severity", severity),
        zap.Time("timestamp", time.Now()),
    }

    for k, v := range details {
        fields = append(fields, zap.Any(k, v))
    }

    switch severity {
    case "critical":
        a.logger.Error("Security event", fields...)
    case "high":
        a.logger.Warn("Security event", fields...)
    default:
        a.logger.Info("Security event", fields...)
    }
}
```

---

## 4. 디버깅 가이드

### 4.1 로깅 레벨 조정

```go
// 개발 환경: 상세 로깅
func initDevLogger() *zap.Logger {
    config := zap.NewDevelopmentConfig()
    config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

    logger, _ := config.Build()
    return logger
}

// 프로덕션: 선택적 상세 로깅
func initProdLogger() *zap.Logger {
    config := zap.NewProductionConfig()

    // 환경 변수로 레벨 제어
    level := os.Getenv("LOG_LEVEL")
    switch level {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    case "info":
        config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    default:
        config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
    }

    logger, _ := config.Build()
    return logger
}
```

### 4.2 디버그 엔드포인트

```go
func setupDebugEndpoints() {
    // pprof 프로파일링
    http.HandleFunc("/debug/pprof/", pprof.Index)
    http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    http.HandleFunc("/debug/pprof/profile", pprof.Profile)
    http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    http.HandleFunc("/debug/pprof/trace", pprof.Trace)

    // 상태 조회
    http.HandleFunc("/debug/sessions", debugSessionsHandler)
    http.HandleFunc("/debug/cache", debugCacheHandler)

    go http.ListenAndServe(":6060", nil)
}

func debugSessionsHandler(w http.ResponseWriter, r *http.Request) {
    sessions := sessionManager.GetAllSessions()

    data := make([]map[string]interface{}, len(sessions))
    for i, sess := range sessions {
        data[i] = map[string]interface{}{
            "session_id": sess.ID,
            "peer_did":   sess.RemoteDID,
            "created_at": sess.CreatedAt,
            "expires_at": sess.ExpiresAt,
            "active":     time.Now().Before(sess.ExpiresAt),
        }
    }

    json.NewEncoder(w).Encode(data)
}
```

### 4.3 패킷 덤프

```go
type PacketDumper struct {
    enabled bool
    mu      sync.Mutex
    file    *os.File
}

func NewPacketDumper(filename string) *PacketDumper {
    file, _ := os.Create(filename)
    return &PacketDumper{
        enabled: true,
        file:    file,
    }
}

func (d *PacketDumper) DumpRequest(req *http.Request) {
    if !d.enabled {
        return
    }

    d.mu.Lock()
    defer d.mu.Unlock()

    fmt.Fprintf(d.file, "\n=== REQUEST %s ===\n", time.Now().Format(time.RFC3339))
    fmt.Fprintf(d.file, "%s %s\n", req.Method, req.URL)

    // 헤더
    fmt.Fprintln(d.file, "Headers:")
    for k, v := range req.Header {
        fmt.Fprintf(d.file, "  %s: %s\n", k, v)
    }

    // 바디
    if req.Body != nil {
        body, _ := ioutil.ReadAll(req.Body)
        req.Body = ioutil.NopCloser(bytes.NewBuffer(body))  // 복원

        fmt.Fprintln(d.file, "Body:")
        fmt.Fprintf(d.file, "%s\n", string(body))
    }
}
```

### 4.4 추적 (Tracing)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func performHandshakeWithTracing(ctx context.Context, peerDID string) error {
    tracer := otel.Tracer("sage-agent")

    ctx, span := tracer.Start(ctx, "handshake")
    defer span.End()

    span.SetAttributes(
        attribute.String("peer.did", peerDID),
    )

    // Invitation
    ctx, invSpan := tracer.Start(ctx, "handshake.invitation")
    err := sendInvitation(ctx, peerDID)
    invSpan.End()
    if err != nil {
        span.RecordError(err)
        return err
    }

    // Request
    ctx, reqSpan := tracer.Start(ctx, "handshake.request")
    err = sendRequest(ctx, peerDID)
    reqSpan.End()
    if err != nil {
        span.RecordError(err)
        return err
    }

    // ... Response, Complete ...

    return nil
}
```

---

## 5. FAQ

### Q1: SAGE는 어떤 블록체인을 지원하나요?

**A:** 현재 다음 블록체인을 지원합니다:
- **Ethereum** (Mainnet, Sepolia)
- **Kaia** (Mainnet, Kairos testnet)
- **Solana** (Mainnet, Devnet)

추가 EVM 호환 체인도 쉽게 통합 가능합니다.

### Q2: 가스 비용은 얼마나 드나요?

**A:** 네트워크별 예상 비용:

```
Kaia (Kairos Testnet):
- DID 등록: ~187,000 gas × 250 Gwei = 0.047 KAIA (~$2-5)
- DID 업데이트: ~49,000 gas × 250 Gwei = 0.012 KAIA (~$0.5-1)
- DID 조회: 무료 (view 함수)

Ethereum (Mainnet):
- DID 등록: ~200,000 gas × 50 Gwei = 0.01 ETH (~$30-50)
- 훨씬 비싸므로 L2 사용 권장

Solana:
- DID 등록: ~0.001 SOL (~$0.1)
- 가장 저렴!
```

### Q3: 세션은 얼마나 유지되나요?

**A:** 기본 24시간이며, 설정으로 변경 가능합니다:

```go
config := &session.Config{
    SessionTTL: 48 * time.Hour,  // 48시간
}
```

만료 전 자동 갱신도 구현 가능합니다.

### Q4: Forward Secrecy가 보장되나요?

**A:** 네! SAGE는 완벽한 Forward Secrecy를 제공합니다:

1. **임시 키 사용**: 각 핸드셰이크마다 새 X25519 키 쌍 생성
2. **세션 후 폐기**: 임시 키는 세션 후 메모리에서 삭제
3. **과거 복호화 불가능**: 개인키가 유출되어도 과거 세션은 안전

### Q5: 프로덕션 환경에서 몇 개의 세션을 처리할 수 있나요?

**A:** 하드웨어에 따라 다르지만:

```
일반 서버 (4 CPU, 8GB RAM):
- 동시 세션: ~10,000개
- 초당 메시지: ~5,000개
- 초당 핸드셰이크: ~100개

고성능 서버 (16 CPU, 32GB RAM):
- 동시 세션: ~100,000개
- 초당 메시지: ~50,000개
- 초당 핸드셰이크: ~1,000개
```

### Q6: 오프라인 에이전트와 통신할 수 있나요?

**A:** 직접 통신은 불가능하지만, 메시지 큐 패턴을 사용할 수 있습니다:

```go
// 중간 서버 (항상 온라인)
type MessageQueue struct {
    messages map[string][]*EncryptedMessage
}

// Agent A → 중간 서버
func (q *MessageQueue) StoreMessage(recipientDID string, msg *EncryptedMessage) {
    q.messages[recipientDID] = append(q.messages[recipientDID], msg)
}

// Agent B (온라인 복귀) → 중간 서버
func (q *MessageQueue) GetMessages(myDID string) []*EncryptedMessage {
    msgs := q.messages[myDID]
    delete(q.messages, myDID)
    return msgs
}
```

### Q7: SAGE를 기존 시스템에 점진적으로 통합할 수 있나요?

**A:** 물론입니다! 권장 단계:

```
Phase 1: 인증만 추가 (1주)
├─ RFC 9421 서명 검증만 구현
└─ 기존 통신 방식 유지

Phase 2: DID 통합 (2주)
├─ 블록체인에 에이전트 등록
└─ DID Resolution 추가

Phase 3: 암호화 통신 (2주)
├─ 핸드셰이크 구현
└─ 세션 기반 암호화

Phase 4: 최적화 (진행형)
├─ 캐싱 개선
└─ 성능 튜닝
```

### Q8: 멀티 테넌시를 지원하나요?

**A:** 네, 각 에이전트가 독립적인 DID를 가지므로 자연스럽게 지원됩니다:

```go
type MultiTenantSAGE struct {
    agents map[string]*SAGEAgent  // tenantID → agent
}

func (m *MultiTenantSAGE) GetAgent(tenantID string) *SAGEAgent {
    return m.agents[tenantID]
}

// 각 테넌트별 독립 처리
agent := multiTenant.GetAgent("tenant-123")
agent.SendMessage(ctx, peerDID, message)
```

### Q9: 모바일 앱에서 사용할 수 있나요?

**A:** 가능하지만 고려사항이 있습니다:

```
Yes 장점:
- Go Mobile로 iOS/Android 바인딩 가능
- 경량 암호화 라이브러리

Warning 주의사항:
- 블록체인 조회 시 데이터 사용량
- 키 저장소 보안 (Keychain/Keystore 필수)
- 백그라운드 세션 관리

권장: 백엔드 프록시 사용
Mobile App → Backend (SAGE) → Other Agents
```

### Q10: 테스트는 어떻게 하나요?

**A:** 여러 레벨의 테스트 가능:

```bash
# 1. 단위 테스트
go test ./crypto/...
go test ./did/...

# 2. 통합 테스트
go test ./tests/integration/...

# 3. 로컬 테스트넷
cd contracts/ethereum
npx hardhat node  # 로컬 블록체인 시작
npm run deploy:localhost  # 컨트랙트 배포

# 4. 테스트넷 (Kairos)
npm run deploy:kairos

# 5. End-to-End 테스트
./scripts/full-test.sh
```

---

## 6. 전체 시리즈 요약

### 6.1 전체 문서 구조

```
SAGE 상세 가이드 시리즈 (총 8개 파트)
│
├── Part 1: 프로젝트 개요 및 아키텍처 (~800줄)
│   ├─ SAGE란 무엇인가
│   ├─ 왜 필요한가
│   ├─ 전체 아키텍처
│   ├─ 핵심 개념 (DID, HPKE, RFC 9421)
│   └─ 프로젝트 구조
│
├── Part 2: 암호화 시스템 상세 (~1,200줄)
│   ├─ Ed25519 (서명)
│   ├─ X25519 (키 교환)
│   ├─ Secp256k1 (Ethereum)
│   ├─ HPKE (암호화)
│   ├─ ChaCha20-Poly1305 (AEAD)
│   └─ 키 형식 변환 (JWK, PEM)
│
├── Part 3: DID 및 블록체인 통합 (~1,300줄)
│   ├─ DID 표준 및 SAGE 메서드
│   ├─ Ethereum vs Kaia 비교
│   ├─ DID 등록/조회/업데이트
│   ├─ 다단계 캐싱
│   ├─ 크로스체인 검증
│   └─ The Graph 통합
│
├── Part 4: 핸드셰이크 및 세션 관리 (~1,400줄)
│   ├─ 4단계 핸드셰이크 (Invitation, Request, Response, Complete)
│   ├─ 클라이언트 구현
│   ├─ 서버 구현
│   ├─ 세션 생성 및 키 유도
│   ├─ 세션 매니저
│   ├─ 이벤트 기반 아키텍처
│   └─ 보안 고려사항
│
├── Part 5: 스마트 컨트랙트 및 온체인 레지스트리 (~1,500줄)
│   ├─ SageRegistry 컨트랙트 상세
│   ├─ Hook 시스템
│   ├─ 가스 최적화
│   ├─ 컨트랙트 배포
│   ├─ Go 언어 통합
│   ├─ 다국어 바인딩 (Python, JavaScript, Rust)
│   ├─ 보안 (재진입, 오버플로우, 서명 재사용)
│   └─ 실전 예제
│
├── Part 6A: 완전한 데이터 플로우 (~1,400줄)
│   ├─ 전체 시스템 레이어
│   ├─ 등록부터 통신까지 완전한 흐름
│   ├─ 키 및 세션 생명주기
│   ├─ 레이어 통합
│   ├─ 에러 처리
│   └─ 타이밍 다이어그램
│
├── Part 6B: 실전 통합 가이드 (~1,600줄)
│   ├─ 시작하기 및 요구사항
│   ├─ CLI 도구 완전 가이드
│   ├─ Go 프로젝트 통합
│   ├─ Node.js/TypeScript 통합
│   ├─ Python 통합
│   ├─ MCP Tool 보안 추가
│   └─ 프로덕션 배포 (Docker, K8s)
│
└── Part 6C: 문제 해결 및 모범 사례 (~1,400줄) ← 현재 문서
    ├─ 일반적인 문제 및 해결 (13가지)
    ├─ 성능 최적화
    ├─ 보안 Best Practices
    ├─ 디버깅 가이드
    └─ FAQ (10가지)

총 분량: 약 10,600줄
```

### 6.2 학습 경로 추천

```
┌─────────────────────────────────────────────────────────┐
│  초급자 → 중급자 → 고급자 학습 경로                        │
└─────────────────────────────────────────────────────────┘

Level 1: 기초 이해 (1-2일)
├─ Part 1: 프로젝트 개요 읽기
│  • SAGE가 무엇인지 이해
│  • 왜 필요한지 이해
│  • 전체 구조 파악
│
└─ CLI 도구 실습 (Part 6B)
   • 키 생성해보기
   • 테스트넷에 DID 등록
   • DID 조회해보기

Level 2: 핵심 개념 (3-5일)
├─ Part 2: 암호화 시스템
│  • Ed25519, X25519 이해
│  • HPKE 동작 원리
│
├─ Part 3: DID 및 블록체인
│  • DID Document 구조
│  • 캐싱 전략
│
└─ Part 4: 핸드셰이크
   • 4단계 프로토콜 이해
   • 세션 관리

Level 3: 실전 구현 (1-2주)
├─ Part 6B: 통합 가이드
│  • 간단한 에이전트 만들기
│  • MCP Tool에 보안 추가
│
├─ Part 5: 스마트 컨트랙트
│  • 컨트랙트 배포
│  • Go 바인딩 사용
│
└─ Part 6A: 데이터 플로우
   • 전체 흐름 이해
   • 타이밍 최적화

Level 4: 최적화 및 프로덕션 (진행형)
├─ Part 6C: Best Practices
│  • 성능 최적화
│  • 보안 강화
│  • 문제 해결
│
└─ 실전 배포
   • 프로덕션 배포
   • 모니터링 설정
   • 지속적 개선
```

### 6.3 핵심 개념 요약

#### 암호화

```
Ed25519: 서명 (메시지 인증)
  • 32-byte 공개키, 64-byte 서명
  • RFC 9421 HTTP 서명에 사용

X25519: 키 교환 (Forward Secrecy)
  • ECDH로 공유 비밀 생성
  • 각 핸드셰이크마다 새 키

ChaCha20-Poly1305: 메시지 암호화
  • AEAD (인증 암호화)
  • 빠르고 안전
```

#### 블록체인

```
DID 등록: 한 번만 (가스 비용 발생)
DID 조회: 무제한 (무료, 캐싱 활용)
DID 업데이트: 필요시 (가스 비용 발생)

캐싱 전략: L1(메모리) → L2(Redis) → L3(DB) → L4(Chain)
→ 평균 응답 시간: 7.9ms (캐싱 없이 200ms)
```

#### 핸드셰이크

```
4단계: Invitation → Request → Response → Complete

각 단계:
1. Invitation: "안전하게 대화하고 싶어요"
2. Request: 임시 키 교환 (암호화됨)
3. Response: 세션 파라미터 합의
4. Complete: 세션 확립 완료

결과: 양방향 암호화 세션 생성
```

#### 세션

```
세션 키 유도:
Shared Secret (X25519 DH)
    ↓ HKDF
Session Seed
    ↓ HKDF-Expand
4개 독립 키: c2s-enc, c2s-auth, s2c-enc, s2c-auth

메시지 암호화:
Plaintext → ChaCha20-Poly1305 → Ciphertext + Auth Tag
```

### 6.4 다음 단계

SAGE를 학습하고 통합했다면:

```
1. 커뮤니티 참여
   - GitHub Discussions
   - Discord/Telegram
   - 이슈 리포팅

2. 기여하기
   - 문서 개선
   - 버그 수정
   - 새 기능 제안

3. 프로덕션 사용
   - 실전 프로젝트에 적용
   - 성능 데이터 공유
   - Best Practices 공유

4. 확장하기
   - 새 블록체인 지원 추가
   - SDK 다른 언어로 포팅
   - MCP Tool 생태계 구축
```

---

## 결론

Part 6C에서는 SAGE 사용 시 겪을 수 있는 문제들과 해결 방법을 다루었습니다.

### 핵심 내용

1. **문제 해결**: 13가지 일반적인 문제와 해결 방법
2. **성능 최적화**: 캐싱, 병렬 처리, 메모리 관리
3. **보안**: 키 관리, 네트워크 보안, 감사 로깅
4. **디버깅**: 로깅, 추적, 패킷 덤프
5. **FAQ**: 10가지 자주 묻는 질문

### 전체 시리즈 완료

**8개 파트 총 10,600줄**의 상세 문서가 완성되었습니다!

이 가이드가 SAGE를 이해하고 사용하는 데 도움이 되기를 바랍니다.

---

**문서 정보**
- 작성일: 2025-01-15
- 버전: 1.0
- Part: 6C/6C (최종)
- 이전: [Part 6B - Practical Integration Guide](DETAILED_GUIDE_PART6B_KO.md)

** 전체 시리즈 완성! **

프로그래밍을 모르는 초급자도 SAGE를 완전히 이해하고 사용할 수 있는 완전한 가이드입니다.
