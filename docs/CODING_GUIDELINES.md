# SAGE Coding Guidelines

**Last Updated**: 2025-10-10
**Version**: 1.0

이 문서는 SAGE 프로젝트의 Go 코드 작성 가이드라인을 정의합니다. 특히 HPKE 타입 어설션 버그로부터 배운 교훈을 바탕으로 타입 안전성과 에러 핸들링 베스트 프랙티스를 다룹니다.

---

## 목차

1. [타입 안전성 원칙](#타입-안전성-원칙)
2. [interface{} 사용 가이드라인](#interface-사용-가이드라인)
3. [타입 어설션 패턴](#타입-어설션-패턴)
4. [에러 핸들링 베스트 프랙티스](#에러-핸들링-베스트-프랙티스)
5. [DID Resolver 구현 가이드](#did-resolver-구현-가이드)
6. [코드 예제](#코드-예제)

---

## 타입 안전성 원칙

### 1. 명확한 타입 정의

**DO:**
```go
// 명확한 타입을 반환하는 인터페이스 정의
type KeyResolver interface {
    ResolveSigningKey(ctx context.Context, did AgentDID) (ed25519.PublicKey, error)
    ResolveEncryptionKey(ctx context.Context, did AgentDID) (*ecdh.PublicKey, error)
}
```

**DON'T:**
```go
// interface{}를 반환하는 모호한 인터페이스
type KeyResolver interface {
    ResolveKey(ctx context.Context, did AgentDID) (interface{}, error)
}
```

### 2. 단일 책임 원칙 (Single Responsibility)

각 메서드는 하나의 명확한 목적을 가져야 합니다.

**DO:**
```go
// 서명 키와 암호화 키를 분리
type Resolver interface {
    ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error)      // Ed25519 for signature verification
    ResolveKEMKey(ctx context.Context, did AgentDID) (interface{}, error)         // X25519 for HPKE encryption
}
```

**DON'T:**
```go
// 하나의 메서드로 모든 키 처리
type Resolver interface {
    ResolveKey(ctx context.Context, did AgentDID, keyType string) (interface{}, error)
}
```

---

## interface{} 사용 가이드라인

### 언제 interface{} 사용이 적절한가?

1. **제네릭이 필요한 경우** (Go 1.18+ 제네릭 사용 권장)
2. **외부 라이브러리와의 호환성**
3. **진정한 다형성이 필요한 경우**

### 언제 interface{} 사용을 피해야 하는가?

1. **타입을 미리 알 수 있는 경우**
2. **2-3개의 알려진 타입만 다루는 경우**
3. **타입 안전성이 중요한 경우**

### interface{} 대체 패턴

#### 패턴 1: 타입별 메서드 분리
```go
// BEFORE (Bad)
func (r *Resolver) ResolveKey(ctx context.Context, did AgentDID) (interface{}, error)

// AFTER (Good)
func (r *Resolver) ResolveSigningKey(ctx context.Context, did AgentDID) (ed25519.PublicKey, error)
func (r *Resolver) ResolveKEMKey(ctx context.Context, did AgentDID) (*ecdh.PublicKey, error)
```

#### 패턴 2: 타입 파라미터 (Go 1.18+)
```go
// Generic resolver (if truly needed)
type Resolver[T any] interface {
    Resolve(ctx context.Context, did AgentDID) (T, error)
}

// Usage
type Ed25519Resolver = Resolver[ed25519.PublicKey]
type X25519Resolver = Resolver[*ecdh.PublicKey]
```

#### 패턴 3: Sum Type 패턴
```go
type PublicKey struct {
    Ed25519 *ed25519.PublicKey
    X25519  *ecdh.PublicKey
}

func (pk PublicKey) AsEd25519() (ed25519.PublicKey, bool) {
    if pk.Ed25519 != nil {
        return *pk.Ed25519, true
    }
    return nil, false
}
```

---

## 타입 어설션 패턴

### 1. 안전한 타입 어설션 (Comma-OK 패턴)

**ALWAYS USE:**
```go
// Comma-OK 패턴으로 안전하게 처리
pub, ok := meta.PublicKey.(ed25519.PublicKey)
if !ok {
    return nil, fmt.Errorf("expected ed25519.PublicKey, got %T", meta.PublicKey)
}
```

**NEVER USE:**
```go
// Panic 가능성이 있는 직접 어설션
pub := meta.PublicKey.(ed25519.PublicKey)  // 런타임 패닉 위험!
```

### 2. 타입 스위치 패턴

여러 타입을 처리해야 하는 경우:

```go
func ProcessKey(key interface{}) error {
    switch k := key.(type) {
    case ed25519.PublicKey:
        // Ed25519 처리
        return verifyWithEd25519(k)
    case *ecdh.PublicKey:
        // ECDH 처리
        return processECDH(k)
    case nil:
        return errors.New("nil key")
    default:
        return fmt.Errorf("unsupported key type: %T", key)
    }
}
```

### 3. 인터페이스 타입 검사

```go
type Verifier interface {
    Verify(msg, sig []byte) error
}

// 커스텀 인터페이스 우선, fallback 제공
func VerifySignature(pub interface{}, msg, sig []byte) error {
    if v, ok := pub.(Verifier); ok {
        return v.Verify(msg, sig)
    }

    if ed, ok := pub.(ed25519.PublicKey); ok {
        if !ed25519.Verify(ed, msg, sig) {
            return errors.New("ed25519 verification failed")
        }
        return nil
    }

    return fmt.Errorf("unsupported key type: %T", pub)
}
```

### 4. 타입 어설션 헬퍼 함수

```go
// 재사용 가능한 타입 변환 헬퍼
func AsEd25519PublicKey(v interface{}) (ed25519.PublicKey, error) {
    switch key := v.(type) {
    case ed25519.PublicKey:
        return key, nil
    case *ed25519.PublicKey:
        if key == nil {
            return nil, errors.New("nil ed25519 public key pointer")
        }
        return *key, nil
    case []byte:
        if len(key) != ed25519.PublicKeySize {
            return nil, fmt.Errorf("invalid ed25519 key size: %d", len(key))
        }
        return ed25519.PublicKey(key), nil
    default:
        return nil, fmt.Errorf("cannot convert %T to ed25519.PublicKey", v)
    }
}
```

---

## 에러 핸들링 베스트 프랙티스

### 1. 명확한 에러 메시지

**DO:**
```go
if pub, ok := key.(ed25519.PublicKey); !ok {
    return fmt.Errorf("signature verification failed: expected ed25519.PublicKey for signing, got %T (key was used for encryption, not signing)", key)
}
```

**DON'T:**
```go
if pub, ok := key.(ed25519.PublicKey); !ok {
    return errors.New("invalid key")  // 너무 모호함
}
```

### 2. 에러 래핑

```go
import "fmt"

func ProcessMetadata(meta *Metadata) error {
    key, err := resolveKey(meta.DID)
    if err != nil {
        return fmt.Errorf("failed to resolve key for DID %s: %w", meta.DID, err)
    }
    // ...
}
```

### 3. 커스텀 에러 타입

```go
type KeyTypeError struct {
    Expected string
    Got      string
    Context  string
}

func (e *KeyTypeError) Error() string {
    return fmt.Sprintf("%s: expected %s, got %s", e.Context, e.Expected, e.Got)
}

// Usage
if _, ok := key.(ed25519.PublicKey); !ok {
    return &KeyTypeError{
        Expected: "ed25519.PublicKey",
        Got:      fmt.Sprintf("%T", key),
        Context:  "signature verification",
    }
}
```

### 4. 에러 체크 패턴

```go
// 에러 무시는 명시적으로
_, _ = w.Write(data)  // 의도적 무시

// 에러는 즉시 처리
if err := validate(input); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// defer에서 에러 처리
defer func() {
    if err := conn.Close(); err != nil {
        log.Printf("failed to close connection: %v", err)
    }
}()
```

---

## DID Resolver 구현 가이드

### HPKE 버그로부터의 교훈

**문제 상황:**
```go
// ❌ BAD: 하나의 메서드로 서명 키와 암호화 키 모두 처리
type Resolver interface {
    ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error)
}

// 클라이언트 코드에서 타입 혼동 발생
pub, _ := resolver.ResolvePublicKey(ctx, serverDID)
sigKey := pub.(ed25519.PublicKey)  // 실제로는 *ecdh.PublicKey가 반환되어 패닉!
```

**해결 방법:**
```go
// ✅ GOOD: 목적별로 메서드 분리
type Resolver interface {
    // Ed25519 signing key for signature verification
    ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error)

    // X25519 KEM key for HPKE encryption
    ResolveKEMKey(ctx context.Context, did AgentDID) (interface{}, error)
}

// 또는 더 명확하게:
type Resolver interface {
    ResolveSigningKey(ctx context.Context, did AgentDID) (ed25519.PublicKey, error)
    ResolveEncryptionKey(ctx context.Context, did AgentDID) (*ecdh.PublicKey, error)
}
```

### 구현 예제

```go
type AgentResolver struct {
    signingKeys    map[string]ed25519.PublicKey
    encryptionKeys map[string]*ecdh.PublicKey
}

func (r *AgentResolver) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
    key, ok := r.signingKeys[string(did)]
    if !ok {
        return nil, fmt.Errorf("signing key not found for DID: %s", did)
    }
    return key, nil
}

func (r *AgentResolver) ResolveKEMKey(ctx context.Context, did AgentDID) (interface{}, error) {
    key, ok := r.encryptionKeys[string(did)]
    if !ok {
        return nil, fmt.Errorf("encryption key not found for DID: %s", did)
    }
    return key, nil
}
```

---

## 코드 예제

### 예제 1: 안전한 키 검증

```go
func VerifyHandshakeSignature(resolver Resolver, did AgentDID, msg, sig []byte) error {
    // 1. 서명 검증용 키 조회 (명확한 목적)
    keyInterface, err := resolver.ResolvePublicKey(context.Background(), did)
    if err != nil {
        return fmt.Errorf("failed to resolve signing key: %w", err)
    }

    // 2. 안전한 타입 어설션
    pub, ok := keyInterface.(ed25519.PublicKey)
    if !ok {
        return fmt.Errorf("expected ed25519.PublicKey for signature verification, got %T", keyInterface)
    }

    // 3. 검증 수행
    if !ed25519.Verify(pub, msg, sig) {
        return errors.New("signature verification failed")
    }

    return nil
}
```

### 예제 2: HPKE 초기화

```go
func InitializeHPKE(resolver Resolver, recipientDID AgentDID, plaintext []byte) (*HPKEResult, error) {
    // 1. 암호화용 KEM 키 조회 (명확한 목적)
    kemInterface, err := resolver.ResolveKEMKey(context.Background(), recipientDID)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve KEM key: %w", err)
    }

    // 2. 안전한 타입 어설션
    recipientPub, ok := kemInterface.(*ecdh.PublicKey)
    if !ok {
        return nil, fmt.Errorf("expected *ecdh.PublicKey for HPKE, got %T", kemInterface)
    }

    // 3. HPKE 수행
    enc, ct, err := hpke.SealBase(recipientPub, plaintext, info, aad)
    if err != nil {
        return nil, fmt.Errorf("HPKE seal failed: %w", err)
    }

    return &HPKEResult{Enc: enc, Ciphertext: ct}, nil
}
```

### 예제 3: 멀티 타입 처리

```go
type KeyMaterial interface {
    Type() string
    Bytes() []byte
}

func ProcessKeyMaterial(km KeyMaterial) error {
    switch km.Type() {
    case "ed25519":
        pub := ed25519.PublicKey(km.Bytes())
        return processEd25519(pub)

    case "x25519":
        pub, err := ecdh.X25519().NewPublicKey(km.Bytes())
        if err != nil {
            return fmt.Errorf("invalid x25519 key: %w", err)
        }
        return processX25519(pub)

    default:
        return fmt.Errorf("unsupported key type: %s", km.Type())
    }
}
```

---

## 테스트 가이드라인

### 1. 타입 어설션 테스트

```go
func TestKeyTypeAssertion(t *testing.T) {
    tests := []struct {
        name    string
        key     interface{}
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid ed25519 key",
            key:     ed25519.PublicKey(make([]byte, 32)),
            wantErr: false,
        },
        {
            name:    "wrong type - ecdh key",
            key:     &ecdh.PublicKey{},
            wantErr: true,
            errMsg:  "expected ed25519.PublicKey",
        },
        {
            name:    "nil key",
            key:     nil,
            wantErr: true,
            errMsg:  "nil key",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := processKey(tt.key)
            if tt.wantErr {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### 2. Resolver 테스트

```go
func TestResolverKeyTypes(t *testing.T) {
    resolver := &TestResolver{
        signingKey: ed25519.PublicKey(make([]byte, 32)),
        kemKey:     mustGenerateX25519Key(),
    }

    // Test signing key
    sigKey, err := resolver.ResolvePublicKey(context.Background(), "did:test:123")
    require.NoError(t, err)
    _, ok := sigKey.(ed25519.PublicKey)
    require.True(t, ok, "signing key should be ed25519.PublicKey")

    // Test KEM key
    kemKey, err := resolver.ResolveKEMKey(context.Background(), "did:test:123")
    require.NoError(t, err)
    _, ok = kemKey.(*ecdh.PublicKey)
    require.True(t, ok, "KEM key should be *ecdh.PublicKey")
}
```

---

## 체크리스트

코드 리뷰 시 다음 항목을 확인하세요:

- [ ] `interface{}` 사용 시 정당한 이유가 있는가?
- [ ] 타입 어설션에 comma-ok 패턴을 사용하는가?
- [ ] 에러 메시지가 충분히 구체적인가?
- [ ] 타입 불일치 시 명확한 에러를 반환하는가?
- [ ] 테스트에서 잘못된 타입 케이스를 다루는가?
- [ ] 문서화에 예상 타입이 명시되어 있는가?
- [ ] nil 체크를 적절히 수행하는가?
- [ ] 에러를 적절히 래핑하는가?

---

## 참고 자료

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [SAGE HPKE Bug Report](https://github.com/sage-x-project/sage/commit/9cba982) - 타입 어설션 버그 수정
- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)

---

**문서 히스토리:**
- 2025-10-10: v1.0 초기 작성 (HPKE 타입 어설션 버그 교훈 반영)
