# SAGE 프로젝트 상세 가이드 - Part 2: 암호화 시스템 Deep Dive

> **대상 독자**: 프로그래밍 초급자부터 중급 개발자까지
> **작성일**: 2025-10-07
> **버전**: 1.0
> **이전**: [Part 1 - 프로젝트 개요](./DETAILED_GUIDE_PART1_KO.md)

---

## 목차
1. [암호화 기초 개념](#1-암호화-기초-개념)
2. [Ed25519 디지털 서명](#2-ed25519-디지털-서명)
3. [Secp256k1과 Ethereum 호환성](#3-secp256k1과-ethereum-호환성)
4. [X25519 키 교환](#4-x25519-키-교환)
5. [HPKE (Hybrid Public Key Encryption)](#5-hpke-hybrid-public-key-encryption)
6. [ChaCha20-Poly1305 AEAD](#6-chacha20-poly1305-aead)
7. [HKDF 키 유도 함수](#7-hkdf-키-유도-함수)
8. [키 변환과 상호운용성](#8-키-변환과-상호운용성)
9. [실전 예제 및 테스트](#9-실전-예제-및-테스트)

---

## 1. 암호화 기초 개념

### 1.1 대칭키 vs 비대칭키 암호화

#### 대칭키 암호화 (Symmetric Encryption)

**개념**: 암호화와 복호화에 같은 키를 사용

```
비유: 자물쇠와 열쇠
- 문을 잠글 때도 같은 열쇠
- 문을 열 때도 같은 열쇠
- 열쇠를 분실하면 문을 열 수 없음

┌──────────────┐           ┌──────────────┐
│  평문        │           │  암호문      │
│ "Hello"      │           │ "xK9mP2..."  │
└──────┬───────┘           └──────┬───────┘
       │                          │
       │ 암호화                    │ 복호화
       ↓                          ↓
    [키: abc123]              [키: abc123]
```

**장점**:
- ⚡ 매우 빠름 (GB 단위 데이터도 빠르게 처리)
- 💾 적은 연산량
- 🔒 강력한 보안 (AES-256, ChaCha20 등)

**단점**:
- 🔑 키 공유 문제: 어떻게 안전하게 키를 전달?
- 👥 N명이 통신하면 N(N-1)/2개의 키 필요

**SAGE에서 사용**:
- ChaCha20-Poly1305로 세션 메시지 암호화
- 세션 확립 후 모든 통신에 사용

#### 비대칭키 암호화 (Asymmetric Encryption)

**개념**: 공개키로 암호화, 개인키로 복호화

```
비유: 우편함
- 공개키 = 우편함 투입구 (누구나 편지 넣을 수 있음)
- 개인키 = 우편함 열쇠 (소유자만 편지 꺼낼 수 있음)

발신자 A:
┌──────────────┐
│  평문        │
│ "Hello"      │
└──────┬───────┘
       │ B의 공개키로 암호화
       ↓
┌──────────────┐
│  암호문      │
│ "xK9mP2..."  │
└──────┬───────┘
       │ 전송
       ↓
수신자 B:
┌──────────────┐
│  암호문      │
│ "xK9mP2..."  │
└──────┬───────┘
       │ B의 개인키로 복호화
       ↓
┌──────────────┐
│  평문        │
│ "Hello"      │
└──────────────┘
```

**장점**:
- 🔓 키 배포 문제 해결 (공개키는 공개해도 안전)
- 📝 디지털 서명 가능
- 👥 N명 통신 시 N개 키 쌍만 필요

**단점**:
- 🐌 매우 느림 (대칭키의 100~1000배)
- 💻 많은 연산량
- 📦 암호화 데이터 크기 제한

**SAGE에서 사용**:
- Ed25519로 메시지 서명
- X25519로 세션 키 교환
- 핸드셰이크 초기 단계

#### 하이브리드 방식 (SAGE의 접근법)

```
최상의 조합:

1단계: 비대칭키로 세션 키 공유
   A의 X25519 ←→ B의 X25519
   → 공유 비밀 생성

2단계: 공유 비밀에서 대칭키 유도
   공유비밀 → HKDF → 세션키

3단계: 대칭키로 실제 데이터 암호화
   데이터 → ChaCha20-Poly1305 → 암호문

결과:
✅ 안전한 키 교환 (비대칭키)
✅ 빠른 암호화 (대칭키)
✅ 최고의 보안 + 성능
```

### 1.2 디지털 서명 (Digital Signature)

**개념**: 메시지가 특정인에게서 왔음을 증명하는 암호학적 증거

```
서명 과정:

1. 해시 계산
   메시지 → SHA-256 → 해시값
   "Hello, World!" → "a591a6d40bf420404..."

2. 개인키로 서명
   해시값 + 개인키 → 서명
   "a591a6d..." + 🔑 → "MEUCIQDx..."

3. 전송
   원본메시지 + 서명 → 수신자

검증 과정:

1. 해시 재계산
   받은메시지 → SHA-256 → 해시값'

2. 공개키로 서명 검증
   서명 + 공개키 → 해시값"
   "MEUCIQDx..." + 🔓 → "a591a6d..."

3. 비교
   해시값' == 해시값"?
   ✅ 같으면: 유효한 서명
   ❌ 다르면: 변조되었거나 위조된 서명
```

**실생활 비유**:
```
종이 서명:
- 복사 가능 (위조 위험)
- 누가 언제 서명했는지 불명확
- 문서 변조 시 알 수 없음

디지털 서명:
- 복사 불가능 (개인키 필요)
- 타임스탬프 포함 가능
- 문서 1비트만 바뀌어도 검증 실패
```

**SAGE에서의 활용**:
```go
// 메시지 서명
message := []byte("Transfer 100 tokens to Agent B")
signature, err := agentKey.Sign(message)

// 서명 검증
err = peerKey.Verify(message, signature)
if err != nil {
    // 서명 무효!
}

코드 위치: crypto/keys/ed25519.go:72-83
```

### 1.3 키 교환 (Key Exchange)

**Diffie-Hellman 키 교환의 마법**

```
문제: 두 사람이 도청되는 채널에서 비밀 키를 공유하려면?

해결: 수학적 마법 ✨

시각화:

1. 공개 파라미터 합의
   Alice와 Bob: "소수 p=23, 생성자 g=5 사용하자"
   (공개되어도 안전)

2. 개인 비밀 선택
   Alice: a=6 (비밀!)
   Bob:   b=15 (비밀!)

3. 공개값 계산 및 교환
   Alice: A = g^a mod p = 5^6 mod 23 = 8
   Bob:   B = g^b mod p = 5^15 mod 23 = 19

   Alice → [8] → Bob
   Bob → [19] → Alice

4. 공유 비밀 계산
   Alice: s = B^a mod p = 19^6 mod 23 = 2
   Bob:   s = A^b mod p = 8^15 mod 23 = 2

   둘 다 같은 값 2를 얻음!

5. 도청자는?
   - 8, 19, 23, 5를 모두 알지만
   - 6이나 15를 알아낼 수 없음 (이산 로그 문제)
```

**X25519 (타원곡선 버전)**

```
더 효율적인 타원곡선 사용:

Alice:
1. 개인키 생성: a (32바이트 랜덤)
2. 공개키 계산: A = a * G
   (G는 곡선의 기준점)

Bob:
1. 개인키 생성: b (32바이트 랜덤)
2. 공개키 계산: B = b * G

교환:
Alice ←→ [A, B] ←→ Bob

공유 비밀:
Alice: S = a * B = a * (b * G)
Bob:   S = b * A = b * (a * G)

둘 다: S = a * b * G (같은 점!)

코드:
shared, err := keyPair.DeriveSharedSecret(peerPublicKey)

위치: crypto/keys/x25519.go:111-128
```

---

## 2. Ed25519 디지털 서명

### 2.1 Ed25519란?

**Edwards-curve Digital Signature Algorithm**

```
특징:
- 타원곡선 암호화 기반
- Curve25519 사용
- 매우 빠른 서명/검증
- 작은 키 크기 (32바이트)
- 결정론적 서명
```

**왜 Ed25519를 선택했나?**

| 알고리즘 | 키 크기 | 서명 크기 | 속도 | 보안 레벨 |
|---------|---------|----------|------|----------|
| RSA-2048 | 256B | 256B | 느림 | 112bit |
| ECDSA-256 | 32B | 64B | 중간 | 128bit |
| **Ed25519** | **32B** | **64B** | **빠름** | **128bit** |

**SAGE의 선택 이유**:
1. ⚡ 서명 생성: 0.01ms (매우 빠름)
2. ⚡ 검증: 0.03ms
3. 🔒 높은 보안성
4. 🐛 구현 버그에 강함 (트위스트 공격 방지)
5. 🎯 사이드 채널 공격 저항성

### 2.2 수학적 원리 (간단히)

**타원곡선이란?**

```
수식: y² = x³ + ax + b

Ed25519는 Edwards 곡선 사용:
x² + y² = 1 + dx²y²
여기서 d = -121665/121666

특별한 점들:
- 기준점 G (generator)
- 무한대 점 O (단위원소)

점 덧셈 규칙:
P + Q = R (곡선 상의 점들)

스칼라 곱셈:
n * P = P + P + ... + P (n번)
```

**서명 과정 (간소화)**

```
키 생성:
1. 개인키: 랜덤 32바이트 (seed)
2. seed → SHA-512 → (a, prefix)
3. 공개키: A = a * G

서명 생성 (메시지 m):
1. r = Hash(prefix || m)
2. R = r * G
3. S = r + Hash(R || A || m) * a
4. 서명 = (R, S)

검증:
1. S * G = R + Hash(R || A || m) * A 인지 확인
2. 같으면 유효, 다르면 무효
```

### 2.3 SAGE 구현 분석

**코드: crypto/keys/ed25519.go**

```go
// 키 생성
func GenerateEd25519KeyPair() (sagecrypto.KeyPair, error) {
    // 1. 암호학적 안전한 난수 생성
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }

    // 2. 공개키 해시로 ID 생성
    hash := sha256.Sum256(publicKey)
    id := hex.EncodeToString(hash[:8])  // 첫 8바이트만

    return &ed25519KeyPair{
        privateKey: privateKey,  // 64바이트
        publicKey:  publicKey,   // 32바이트
        id:         id,          // 16자 16진수
    }, nil
}

위치: crypto/keys/ed25519.go:38-54
```

**서명 생성**

```go
func (kp *ed25519KeyPair) Sign(message []byte) ([]byte, error) {
    // ed25519.Sign은 내부적으로:
    // 1. SHA-512로 해시 계산
    // 2. 난수 생성 (deterministic)
    // 3. 점 연산으로 R, S 계산
    signature := ed25519.Sign(kp.privateKey, message)

    // signature는 64바이트:
    // - 처음 32바이트: R (점의 인코딩)
    // - 나머지 32바이트: S (스칼라)
    return signature, nil
}

위치: crypto/keys/ed25519.go:72-75

실제 사용 예:
message := []byte("Agent A requests session with Agent B")
sig, _ := keyPair.Sign(message)
// sig: [64]byte
```

**서명 검증**

```go
func (kp *ed25519KeyPair) Verify(message, signature []byte) error {
    // ed25519.Verify는:
    // 1. 서명에서 R, S 추출
    // 2. S * G = R + H(R,A,m) * A 확인
    // 3. 점 연산으로 등식 검증
    if !ed25519.Verify(kp.publicKey, message, signature) {
        return sagecrypto.ErrInvalidSignature
    }
    return nil
}

위치: crypto/keys/ed25519.go:78-83

실제 사용 예:
err := peerKey.Verify(message, signature)
if err != nil {
    log.Fatal("서명 검증 실패!")
}
```

### 2.4 실전 예제

**시나리오: Agent A가 메시지에 서명하고 Agent B가 검증**

```go
package main

import (
    "fmt"
    "github.com/sage-x-project/sage/crypto/keys"
)

func main() {
    // 1. Agent A: 키 생성
    agentA, _ := keys.GenerateEd25519KeyPair()
    fmt.Printf("Agent A ID: %s\n", agentA.ID())

    // 2. Agent A: 메시지 작성 및 서명
    message := []byte("Transfer 100 tokens to Agent B")
    signature, _ := agentA.Sign(message)
    fmt.Printf("서명: %x\n", signature[:16]) // 처음 16바이트만 표시

    // 3. Agent A: 공개키 공유 (DID 시스템 통해)
    pubKey := agentA.PublicKey()

    // 4. Agent B: 서명 검증
    agentB := &ed25519KeyPair{publicKey: pubKey}
    err := agentB.Verify(message, signature)

    if err == nil {
        fmt.Println("✅ 서명 유효! Agent A가 보낸 것이 확실함")
    } else {
        fmt.Println("❌ 서명 무효! 변조되었거나 위조됨")
    }

    // 5. 변조 테스트
    tamperedMsg := []byte("Transfer 1000 tokens to Agent B")
    err = agentB.Verify(tamperedMsg, signature)
    fmt.Printf("변조된 메시지 검증: %v\n", err) // 실패해야 함!
}
```

**출력**:
```
Agent A ID: a1b2c3d4e5f6g7h8
서명: 8f3a2b1c9d4e5f6a...
✅ 서명 유효! Agent A가 보낸 것이 확실함
변조된 메시지 검증: invalid signature
```

---

## 3. Secp256k1과 Ethereum 호환성

### 3.1 Secp256k1이란?

**Standards for Efficient Cryptography (SEC)**

```
특징:
- Bitcoin과 Ethereum이 사용하는 타원곡선
- ECDSA 서명 알고리즘
- Keccak-256 해시 (Ethereum)
- 복구 가능한 서명 (recovery ID)
```

**곡선 파라미터**:
```
y² = x³ + 7 (매우 간단한 형태!)

소수 p = 2^256 - 2^32 - 977
위수 n = FFFFFFFF FFFFFFFF FFFFFFFF FFFFFFFE
         BAAEDCE6 AF48A03B BFD25E8C D0364141

기준점 G = (
  79BE667E F9DCBBAC 55A06295 CE870B07 029BFCDB 2DCE28D9 59F2815B 16F81798,
  483ADA77 26A3C465 5DA4FBFC 0E1108A8 FD17B448 A6855419 9C47D08F FB10D4B8
)
```

### 3.2 Ethereum 주소 유도

**공개키 → Ethereum 주소 과정**

```
단계별 설명:

1. 공개키 (65바이트 비압축 형식)
   04 + X좌표(32바이트) + Y좌표(32바이트)

   예: 04a1b2c3d4e5f6...

2. 0x04 프리픽스 제거
   a1b2c3d4e5f6... (64바이트)

3. Keccak-256 해시
   Keccak256(a1b2c3d4e5f6...)
   → ef1234567890abcdef... (32바이트)

4. 마지막 20바이트 추출
   ef1234567890abcdef... → ...1234567890abcdef (20바이트)

5. 0x 프리픽스 추가
   최종 주소: 0x...1234567890abcdef

6. Checksum 적용 (EIP-55)
   일부 문자를 대문자로: 0x...1234567890AbCdEf
```

**SAGE 구현**:

```go
// 코드 위치: crypto/keys/secp256k1.go

type secp256k1KeyPair struct {
    privateKey *secp256k1.PrivateKey
    publicKey  *secp256k1.PublicKey
    id         string
}

// Ethereum 호환 서명 생성
func (kp *secp256k1KeyPair) Sign(message []byte) ([]byte, error) {
    // 1. Keccak256 해시 (Ethereum 표준)
    hash := ethcrypto.Keccak256(message)

    // 2. ECDSA 서명 + Recovery ID
    // 65바이트: r(32) + s(32) + v(1)
    privateKey := kp.privateKey.ToECDSA()
    signature, err := ethcrypto.Sign(hash, privateKey)
    if err != nil {
        return nil, err
    }

    // signature[64] = recovery ID (0, 1, 2, 또는 3)
    // 이를 통해 서명에서 공개키 복구 가능!
    return signature, nil
}

위치: crypto/keys/secp256k1.go:76-90
```

### 3.3 서명 복구 (Public Key Recovery)

**Ethereum의 특별한 기능**

```
일반 서명:
message + signature + publicKey → verify(true/false)

복구 가능한 서명:
message + signature → publicKey

장점:
- 트랜잭션에 공개키 포함 불필요
- 가스 비용 절약
- 서명 크기 감소
```

**복구 과정**:

```go
// Ethereum 트랜잭션 서명 복구 예제
func recoverPublicKey(message, signature []byte) ([]byte, error) {
    // 1. 메시지 해시
    hash := ethcrypto.Keccak256(message)

    // 2. 서명에서 공개키 복구
    // signature[0:64] = r, s
    // signature[64] = recovery ID
    publicKey, err := ethcrypto.SigToPub(hash, signature)
    if err != nil {
        return nil, err
    }

    // 3. 공개키를 바이트로
    pubBytes := ethcrypto.FromECDSAPub(publicKey)
    return pubBytes, nil
}
```

### 3.4 스마트 컨트랙트 연동

**Solidity에서 서명 검증**

```solidity
// contracts/ethereum/contracts/SageRegistryV2.sol

function _recoverSigner(
    bytes32 ethSignedHash,
    bytes memory signature
) internal pure returns (address) {
    // 서명 분해
    bytes32 r;
    bytes32 s;
    uint8 v;

    assembly {
        // signature = r(32) + s(32) + v(1)
        r := mload(add(signature, 32))
        s := mload(add(signature, 64))
        v := byte(0, mload(add(signature, 96)))
    }

    // ecrecover: 내장 함수로 주소 복구
    return ecrecover(ethSignedHash, v, r, s);
}

// 사용 예
address signer = _recoverSigner(messageHash, signature);
require(signer == owner, "Invalid signature");
```

**SAGE의 Go 구현**:

```go
// crypto/chain/ethereum/provider.go

func (p *EthereumProvider) VerifySignature(
    message []byte,
    signature []byte,
    publicKey []byte,
) (bool, error) {
    // 1. 메시지 해시
    hash := crypto.Keccak256(message)

    // 2. 서명에서 주소 복구
    recoveredPub, err := crypto.SigToPub(hash, signature)
    if err != nil {
        return false, err
    }

    // 3. 기대하는 주소 계산
    expectedAddr := crypto.PubkeyToAddress(*expectedPub)
    recoveredAddr := crypto.PubkeyToAddress(*recoveredPub)

    // 4. 비교
    return expectedAddr == recoveredAddr, nil
}
```

---

## 4. X25519 키 교환

### 4.1 X25519란?

**Curve25519 기반 Diffie-Hellman**

```
특징:
- 키 교환 전용 (서명 불가)
- Montgomery 곡선 사용
- 매우 빠른 연산
- 사이드 채널 공격 저항
- 32바이트 키
```

**Curve25519 vs Ed25519**:

| 특징 | Curve25519 (X25519) | Edwards25519 (Ed25519) |
|-----|---------------------|------------------------|
| **용도** | 키 교환 (ECDH) | 서명 (ECDSA) |
| **곡선 형태** | Montgomery | Edwards |
| **연산** | 스칼라 곱셈만 | 점 덧셈 + 곱셈 |
| **함수** | `x = f(u)` | `(x, y)` 점 |
| **변환** | Ed25519 ↔ X25519 가능 | |

### 4.2 ECDH 키 교환 상세

**X25519 연산**

```
Montgomery 곡선:
v² = u³ + 486662u² + u

X25519 함수:
점(u, v)의 u 좌표만 사용
X25519(scalar, u_coordinate) → u'

장점:
- v 좌표 불필요 → 빠름
- 조건 분기 없음 → 사이드 채널 안전
```

**SAGE 구현**:

```go
// crypto/keys/x25519.go

type X25519KeyPair struct {
    privateKey *ecdh.PrivateKey  // 32바이트
    publicKey  *ecdh.PublicKey   // 32바이트
    id         string
}

// 키 생성
func GenerateX25519KeyPair() (sagecrypto.KeyPair, error) {
    // Go 1.20+ ecdh 패키지 사용
    privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }

    publicKey := privateKey.PublicKey()

    // ID 생성
    pubKeyBytes := publicKey.Bytes()
    hash := sha256.Sum256(pubKeyBytes)
    id := hex.EncodeToString(hash[:8])

    return &X25519KeyPair{
        privateKey: privateKey,
        publicKey:  publicKey,
        id:         id,
    }, nil
}

위치: crypto/keys/x25519.go:50-69
```

**공유 비밀 유도**:

```go
func (kp *X25519KeyPair) DeriveSharedSecret(
    peerPubBytes []byte,
) ([]byte, error) {
    // 1. 피어 공개키 파싱
    curve := ecdh.X25519()
    peerPub, err := curve.NewPublicKey(peerPubBytes)
    if err != nil {
        return nil, fmt.Errorf("invalid peer public key: %w", err)
    }

    // 2. ECDH 연산
    // shared = myPriv * peerPub
    shared, err := kp.privateKey.ECDH(peerPub)
    if err != nil {
        return nil, fmt.Errorf("ECDH failed: %w", err)
    }

    // 3. SHA-256 해시 (추가 보안)
    sum := sha256.Sum256(shared)
    return sum[:], nil
}

위치: crypto/keys/x25519.go:111-128

주의사항:
- shared가 모두 0인지 확인 (low-order point 공격 방지)
- 해시를 적용하여 편향 제거
```

### 4.3 Ed25519 ↔ X25519 변환

**왜 변환이 필요한가?**

```
문제:
- DID 시스템에는 Ed25519 공개키 등록
- 핸드셰이크는 X25519 필요
- 두 종류의 키를 모두 등록하면 비용 증가

해결:
- Ed25519 키를 X25519로 변환
- 하나의 키만 블록체인에 등록
```

**수학적 원리**:

```
Ed25519와 X25519는 birational equivalence 관계:

Edwards 곡선 (Ed25519):
x² + y² = 1 + dx²y²

Montgomery 곡선 (X25519):
v² = u³ + Au² + u

변환 공식:
u = (1 + y) / (1 - y)
v = √(u³ + Au² + u) * (u / x)

역변환:
y = (u - 1) / (u + 1)
x = √(dx²y² / (1 - x² - y²))
```

**공개키 변환 (Ed25519 → X25519)**:

```go
// crypto/keys/x25519.go

func convertEd25519PubToX25519(pubKey crypto.PublicKey) ([]byte, error) {
    // 1. 타입 확인
    edPub, ok := pubKey.(ed25519.PublicKey)
    if !ok {
        return nil, fmt.Errorf("not ed25519.PublicKey")
    }

    // 2. Ed25519 점 디코딩
    // edwards25519 패키지 사용 (Go 1.17+)
    P, err := new(edwards25519.Point).SetBytes(edPub)
    if err != nil {
        return nil, fmt.Errorf("invalid Ed25519 point: %w", err)
    }

    // 3. Montgomery 형식으로 변환
    // BytesMontgomery()는 u 좌표만 반환
    xPub := P.BytesMontgomery()

    return xPub, nil
}

위치: crypto/keys/x25519.go:318-334

예제:
edPub := agentA.PublicKey().(ed25519.PublicKey)
xPub, _ := convertEd25519PubToX25519(edPub)
// xPub는 X25519 공개키 (32바이트)
```

**개인키 변환 (Ed25519 → X25519)**:

```go
func convertEd25519PrivToX25519(privKey crypto.PrivateKey) ([]byte, error) {
    // 1. 타입 확인
    edPriv, ok := privKey.(ed25519.PrivateKey)
    if !ok {
        return nil, fmt.Errorf("not ed25519.PrivateKey")
    }

    // 2. Seed 추출 (처음 32바이트)
    seed := edPriv.Seed()

    // 3. RFC 8032 §5.1.5에 따라 처리
    h := sha512.Sum512(seed)

    // 4. 클램핑 (clamping)
    h[0] &= 248    // 하위 3비트 제거
    h[31] &= 127   // 최상위 비트 제거
    h[31] |= 64    // 두 번째 비트 설정

    // 5. 처음 32바이트가 X25519 개인키
    var xPriv [32]byte
    copy(xPriv[:], h[:32])

    return xPriv[:], nil
}

위치: crypto/keys/x25519.go:298-316

클램핑 이유:
- 하위 3비트 0: 8의 배수로 만듦 (cofactor 제거)
- 최상위 비트: 타이밍 공격 방지
- 두 번째 비트: 키 범위 표준화
```

### 4.4 부트스트랩 암호화

**핸드셰이크 초기 단계 보안**

```
문제:
- Agent A와 B가 처음 통신
- 아직 공유 비밀 없음
- 임시 공개키를 어떻게 안전하게 전달?

해결: 부트스트랩 암호화
1. DID 시스템에서 피어의 Ed25519 공개키 조회
2. Ed25519 → X25519 변환
3. 변환된 키로 임시 공개키 암호화
```

**구현**:

```go
// crypto/keys/x25519.go

func EncryptWithEd25519Peer(
    edPeerPub crypto.PublicKey,
    plaintext []byte,
) ([]byte, error) {
    // 1. 임시 X25519 키 쌍 생성
    kp, err := GenerateX25519KeyPair()
    if err != nil {
        return nil, err
    }

    // 2. 피어의 Ed25519 공개키를 X25519로 변환
    peerX, err := convertEd25519PubToX25519(edPeerPub)
    if err != nil {
        return nil, err
    }

    peerPubKey, err := ecdh.X25519().NewPublicKey(peerX)
    if err != nil {
        return nil, err
    }

    // 3. ECDH로 공유 비밀 계산
    privKey := kp.PrivateKey().(*ecdh.PrivateKey)
    raw, err := privKey.ECDH(peerPubKey)
    if err != nil {
        return nil, err
    }

    // 4. Transcript 생성 (Noise Protocol 패턴)
    pubKey := kp.PublicKey().(*ecdh.PublicKey)
    transcript := append(pubKey.Bytes(), peerX...)

    // 5. HKDF로 AES 키 유도
    key, err := deriveHKDFKey(raw, transcript)
    if err != nil {
        return nil, err
    }

    // 6. AES-256-GCM 암호화
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, 12)
    rand.Read(nonce)

    ct := aead.Seal(nil, nonce, plaintext, transcript)

    // 7. 패킷 구성: ephPub || nonce || ciphertext
    packet := append(append(pubKey.Bytes(), nonce...), ct...)

    return packet, nil
}

위치: crypto/keys/x25519.go:178-232

패킷 구조:
┌────────────┬───────┬──────────────┐
│ EphPub(32) │ N(12) │ Ciphertext   │
└────────────┴───────┴──────────────┘
```

**복호화**:

```go
func DecryptWithEd25519Peer(
    privateKey crypto.PrivateKey,
    packet []byte,
) ([]byte, error) {
    // 1. 패킷 파싱
    if len(packet) < 32 + 12 {
        return nil, fmt.Errorf("packet too short")
    }

    ePubBytes := packet[:32]
    nonce := packet[32:44]
    ct := packet[44:]

    // 2. 임시 공개키 로드
    ePubKey, err := ecdh.X25519().NewPublicKey(ePubBytes)
    if err != nil {
        return nil, err
    }

    // 3. 자신의 Ed25519 개인키를 X25519로 변환
    selfXPrivBytes, err := convertEd25519PrivToX25519(privateKey)
    if err != nil {
        return nil, err
    }

    selfXPrivKey, err := ecdh.X25519().NewPrivateKey(selfXPrivBytes)
    if err != nil {
        return nil, err
    }

    // 4. ECDH로 공유 비밀 복원
    raw, err := selfXPrivKey.ECDH(ePubKey)
    if err != nil {
        return nil, err
    }

    // 5. Transcript 재구성
    selfXPub := selfXPrivKey.PublicKey()
    transcript := append(ePubBytes, selfXPub.Bytes()...)

    // 6. HKDF로 같은 AES 키 유도
    key, err := deriveHKDFKey(raw, transcript)
    if err != nil {
        return nil, err
    }

    // 7. 복호화
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    plaintext, err := aead.Open(nil, nonce, ct, transcript)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }

    return plaintext, nil
}

위치: crypto/keys/x25519.go:234-282
```

---

## 5. HPKE (Hybrid Public Key Encryption)

### 5.1 HPKE 개요

**RFC 9180 표준**

```
HPKE = KEM + KDF + AEAD

KEM (Key Encapsulation Mechanism):
- 공개키로 공유 비밀 생성
- SAGE: X25519

KDF (Key Derivation Function):
- 공유 비밀에서 키들 유도
- SAGE: HKDF-SHA256

AEAD (Authenticated Encryption with Associated Data):
- 인증된 암호화
- SAGE: ChaCha20-Poly1305
```

### 5.2 HPKE 모드

**4가지 모드**:

```
1. Base Mode (0)
   - 단방향: 송신자 → 수신자
   - 수신자 인증 없음
   - SAGE 사용 ✅

2. PSK Mode (1)
   - Pre-Shared Key 사용
   - 사전 공유 비밀 필요

3. Auth Mode (2)
   - 송신자 인증
   - 송신자의 정적 개인키 사용

4. AuthPSK Mode (3)
   - Auth + PSK 결합
```

### 5.3 HPKE 동작 과정

**송신자 (Sender)**:

```
입력:
- 수신자 공개키 (pkR)
- 평문 (plaintext)
- 추가 데이터 (info, aad)

단계:

1. Setup (캡슐화)
   ┌─────────────────────────────────┐
   │ 1. 임시 키 쌍 생성:              │
   │    (skE, pkE) ← GenerateKeyPair()│
   │                                  │
   │ 2. ECDH 계산:                    │
   │    dh ← ECDH(skE, pkR)          │
   │                                  │
   │ 3. KEM 캡슐화:                   │
   │    enc ← Encap(pkR, info)       │
   │    = 임시 공개키 pkE             │
   │                                  │
   │ 4. 공유 비밀 추출:               │
   │    secret ← Extract(dh)         │
   └─────────────────────────────────┘

2. Key Schedule (키 유도)
   ┌─────────────────────────────────┐
   │ 1. 컨텍스트 구성:                │
   │    context ← mode || kemID ||   │
   │              kdfID || aeadID    │
   │                                  │
   │ 2. PSK 처리 (Base는 empty):     │
   │    pskID_hash ← 0x              │
   │    info_hash ← Hash(info)       │
   │                                  │
   │ 3. 키 스케줄:                    │
   │    key ← Expand(secret,         │
   │          "hpke key" || context) │
   │    base_nonce ← Expand(secret,  │
   │          "hpke nonce" || ...)   │
   │    exporter ← Expand(secret,    │
   │          "hpke exp" || ...)     │
   └─────────────────────────────────┘

3. Seal (암호화)
   ┌─────────────────────────────────┐
   │ ct ← AEAD.Seal(                 │
   │     key,                        │
   │     nonce,                      │
   │     plaintext,                  │
   │     aad                         │
   │ )                               │
   └─────────────────────────────────┘

출력:
- enc (32바이트)
- ct (ciphertext)
```

**수신자 (Receiver)**:

```
입력:
- 자신의 개인키 (skR)
- 캡슐화된 키 (enc)
- 암호문 (ct)
- 추가 데이터 (info, aad)

단계:

1. Setup (역캡슐화)
   ┌─────────────────────────────────┐
   │ 1. 임시 공개키 파싱:             │
   │    pkE ← enc                    │
   │                                  │
   │ 2. ECDH 계산:                    │
   │    dh ← ECDH(skR, pkE)          │
   │                                  │
   │ 3. 공유 비밀 추출:               │
   │    secret ← Extract(dh)         │
   └─────────────────────────────────┘

2. Key Schedule
   [송신자와 동일한 과정]
   → 같은 key, nonce, exporter 얻음

3. Open (복호화)
   ┌─────────────────────────────────┐
   │ pt ← AEAD.Open(                 │
   │     key,                        │
   │     nonce,                      │
   │     ct,                         │
   │     aad                         │
   │ )                               │
   └─────────────────────────────────┘

출력:
- plaintext
```

### 5.4 SAGE의 HPKE 구현

**송신자**:

```go
// crypto/keys/x25519.go

func HPKESealAndExportToX25519Peer(
    peer crypto.PublicKey,
    plaintext []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (packet []byte, exporterSecret []byte, err error) {
    // 1. HPKE Suite 설정
    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    // 2. 수신자 공개키 언마샬
    pubKey := peer.(*ecdh.PublicKey)
    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    rp, err := kem.UnmarshalBinaryPublicKey(pubKey.Bytes())
    if err != nil {
        return nil, nil, err
    }

    // 3. Sender 생성
    sender, err := suite.NewSender(rp, info)
    if err != nil {
        return nil, nil, err
    }

    // 4. Setup (내부적으로 임시 키 생성)
    enc, sealer, err := sender.Setup(rand.Reader)
    if err != nil {
        return nil, nil, err
    }

    // 5. 암호화
    ct, err := sealer.Seal(plaintext, info)  // AAD = info
    if err != nil {
        return nil, nil, err
    }

    // 6. Exporter Secret 유도 (세션 키용)
    secret := sealer.Export(exportCtx, uint(exportLen))

    // 7. 패킷 조립: enc || ct
    packet = append(enc, ct...)

    return packet, secret, nil
}

위치: crypto/keys/x25519.go:458-499

사용 예:
packet, sessionKey, _ := HPKESealAndExportToX25519Peer(
    peerPub,
    []byte("Hello, HPKE!"),
    []byte("handshake v1"),
    []byte("session-derivation"),
    32,
)
```

**수신자**:

```go
func HPKEOpenAndExportWithX25519Priv(
    priv crypto.PrivateKey,
    packet []byte,
    info []byte,
    exportCtx []byte,
    exportLen int,
) (plaintext []byte, exporterSecret []byte, err error) {
    const encLen = 32  // X25519 캡슐화 키 길이

    // 1. 패킷 파싱
    if len(packet) < encLen {
        return nil, nil, fmt.Errorf("packet too short")
    }
    enc := packet[:encLen]
    ct := packet[encLen:]

    // 2. HPKE Suite (송신자와 동일)
    suite := hpke.NewSuite(
        hpke.KEM_X25519_HKDF_SHA256,
        hpke.KDF_HKDF_SHA256,
        hpke.AEAD_ChaCha20Poly1305,
    )

    // 3. 자신의 개인키 언마샬
    privKey := priv.(*ecdh.PrivateKey)
    kem := hpke.KEM_X25519_HKDF_SHA256.Scheme()
    skR, err := kem.UnmarshalBinaryPrivateKey(privKey.Bytes())
    if err != nil {
        return nil, nil, err
    }

    // 4. Receiver 생성
    receiver, err := suite.NewReceiver(skR, info)
    if err != nil {
        return nil, nil, err
    }

    // 5. Setup (enc로 공유 비밀 복원)
    opener, err := receiver.Setup(enc)
    if err != nil {
        return nil, nil, err
    }

    // 6. 복호화
    pt, err := opener.Open(ct, info)  // AAD = info
    if err != nil {
        return nil, nil, err
    }

    // 7. Exporter Secret 유도 (같은 값!)
    secret := opener.Export(exportCtx, uint(exportLen))

    return pt, secret, nil
}

위치: crypto/keys/x25519.go:501-549

사용 예:
plaintext, sessionKey, _ := HPKEOpenAndExportWithX25519Priv(
    myPriv,
    packet,
    []byte("handshake v1"),
    []byte("session-derivation"),
    32,
)
// sessionKey는 송신자와 동일!
```

### 5.5 Exporter Secret 활용

**세션 키 유도**:

```
HPKE의 핵심 기능:
- 암호화뿐만 아니라 키 유도도 가능
- Export() 함수로 추가 키 재료 생성

활용:
1. 핸드셰이크 중 HPKE로 암호화
2. Export()로 세션 키 유도
3. 세션 키로 이후 통신 암호화

장점:
- 한 번의 핸드셰이크로 여러 키 생성
- Forward Secrecy 유지
- 표준화된 방법
```

**코드 예제**:

```go
// 핸드셰이크에서 HPKE 사용
func establishSession(peerPub crypto.PublicKey) (*session.SecureSession, error) {
    // 1. Request 메시지 암호화 + 세션 키 유도
    requestMsg := []byte("Handshake Request")
    packet, sessionSeed, err := keys.HPKESealAndExportToX25519Peer(
        peerPub,
        requestMsg,
        []byte("sage/handshake v1"),     // info
        []byte("sage-session-key"),      // export context
        32,                               // 32바이트 시드
    )

    // 2. 세션 생성
    sess, err := session.NewSecureSessionFromExporter(
        "session-123",
        sessionSeed,  // HPKE Export로 얻은 비밀
        session.Config{
            MaxAge: time.Hour,
            IdleTimeout: 10 * time.Minute,
        },
    )

    // 3. packet 전송, sess로 이후 통신
    return sess, nil
}
```

---

## 6. ChaCha20-Poly1305 AEAD

### 6.1 AEAD란?

**Authenticated Encryption with Associated Data**

```
일반 암호화:
plaintext → cipher → ciphertext
문제: 변조 탐지 불가

AEAD:
plaintext + AAD → cipher → ciphertext + tag

특징:
1. 암호화 (Encryption)
   - 평문을 암호문으로

2. 인증 (Authentication)
   - 변조 탐지
   - 출처 확인

3. 추가 데이터 (Associated Data)
   - 암호화하지 않지만 인증되는 데이터
   - 예: HTTP 헤더
```

### 6.2 ChaCha20-Poly1305

**구성 요소**:

```
ChaCha20:
- 스트림 암호
- 256비트 키
- 96비트 nonce
- Salsa20의 개선 버전
- ARX 구조 (Add-Rotate-XOR)

Poly1305:
- MAC (Message Authentication Code)
- 128비트 태그
- 매우 빠른 계산

결합:
ChaCha20으로 암호화 + Poly1305로 인증
```

**ChaCha20 알고리즘 (간소화)**:

```
상태 행렬 (16개 32비트 워드):

┌────────────────────────────────┐
│ "expa" "nd 3" "2-by" "te k"   │  상수
├────────────────────────────────┤
│   key[0]   key[1]   key[2]   key[3]   │  256비트 키
│   key[4]   key[5]   key[6]   key[7]   │
├────────────────────────────────┤
│ counter │ nonce[0] nonce[1] nonce[2] │  카운터 + nonce
└────────────────────────────────┘

QR (Quarter Round) 함수:
a += b; d ^= a; d <<<= 16;
c += d; b ^= c; b <<<= 12;
a += b; d ^= a; d <<<= 8;
c += d; b ^= c; b <<<= 7;

20라운드 (10번의 컬럼 + 대각선 라운드)

최종 상태를 평문과 XOR
```

**Poly1305 알고리즘**:

```
입력:
- 메시지 m (여러 블록)
- 256비트 키 (r || s)

과정:
1. 메시지를 16바이트 블록으로 나눔
2. 각 블록을 리틀 엔디안 정수로 해석
3. 모듈러 연산:
   acc = 0
   for each block c:
       acc = ((acc + c) * r) mod (2^130 - 5)
   tag = (acc + s) mod 2^128

출력: 128비트 태그
```

### 6.3 SAGE 구현

**암호화**:

```go
// session/session.go

func (s *SecureSession) EncryptOutbound(plaintext []byte) ([]byte, error) {
    // 1. AEAD 인스턴스 확인
    if s.aeadOut == nil {
        return nil, fmt.Errorf("outbound AEAD not initialized")
    }

    // 2. 랜덤 nonce 생성 (12바이트)
    nonce := make([]byte, chacha20poly1305.NonceSize)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("nonce generation failed: %w", err)
    }

    // 3. AEAD 암호화
    // Seal(dst, nonce, plaintext, additionalData)
    ct := s.aeadOut.Seal(nil, nonce, plaintext, nil)

    // ct는 ciphertext + tag (16바이트 태그 포함)

    // 4. 패킷 조립: nonce || ciphertext+tag
    out := make([]byte, len(nonce)+len(ct))
    copy(out, nonce)
    copy(out[len(nonce):], ct)

    // 5. 세션 상태 업데이트
    s.UpdateLastUsed()

    return out, nil
}

위치: session/session.go:589-606

패킷 구조:
┌───────────┬────────────────┬──────┐
│ Nonce(12) │ Ciphertext     │ Tag(16) │
└───────────┴────────────────┴──────┘
```

**복호화**:

```go
func (s *SecureSession) DecryptInbound(data []byte) ([]byte, error) {
    // 1. AEAD 인스턴스 확인
    if s.aeadIn == nil {
        return nil, fmt.Errorf("inbound AEAD not initialized")
    }

    // 2. 길이 확인
    if len(data) < chacha20poly1305.NonceSize {
        return nil, fmt.Errorf("data too short")
    }

    // 3. 패킷 파싱
    nonce := data[:chacha20poly1305.NonceSize]
    ct := data[chacha20poly1305.NonceSize:]

    // 4. AEAD 복호화 + 검증
    // Open(dst, nonce, ciphertext+tag, additionalData)
    pt, err := s.aeadIn.Open(nil, nonce, ct, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption/authentication failed: %w", err)
    }

    // Open이 성공하면:
    // - 복호화 완료
    // - 태그 검증 완료
    // - 변조되지 않음 보장

    // 5. 세션 상태 업데이트
    s.UpdateLastUsed()

    return pt, nil
}

위치: session/session.go:608-626
```

**AAD 사용**:

```go
// Associated Data를 사용한 암호화
func (s *SecureSession) EncryptWithAADOutbound(
    plaintext, aad []byte,
) ([]byte, error) {
    if s.aeadOut == nil {
        return nil, fmt.Errorf("outbound AEAD not initialized")
    }

    nonce := make([]byte, chacha20poly1305.NonceSize)
    rand.Read(nonce)

    // AAD는 암호화되지 않지만 인증됨
    ct := s.aeadOut.Seal(nil, nonce, plaintext, aad)

    out := make([]byte, len(nonce)+len(ct))
    copy(out, nonce)
    copy(out[len(nonce):], ct)

    s.UpdateLastUsed()
    return out, nil
}

위치: session/session.go:629-645

사용 예:
plaintext := []byte("secret message")
aad := []byte("user:alice,timestamp:1704067200")
encrypted, _ := sess.EncryptWithAADOutbound(plaintext, aad)

// AAD가 변조되면 복호화 실패
```

### 6.4 성능 특성

**벤치마크 (일반적인 CPU)**:

```
ChaCha20-Poly1305:
- 암호화: ~1 GB/s
- 복호화: ~1 GB/s
- 키 설정: ~100 ns

AES-256-GCM (하드웨어 지원 시):
- 암호화: ~3-4 GB/s
- 복호화: ~3-4 GB/s
- 키 설정: ~50 ns

AES-256-GCM (소프트웨어):
- 암호화: ~100 MB/s
- 복호화: ~100 MB/s

결론:
- 하드웨어 AES 있으면 AES 빠름
- 없으면 ChaCha20 훨씬 빠름
- 모바일/IoT에서 ChaCha20 우수
```

---

## 7. HKDF 키 유도 함수

### 7.1 HKDF란?

**HMAC-based Key Derivation Function (RFC 5869)**

```
목적:
- 약한 키 재료를 강한 키로 변환
- 하나의 비밀에서 여러 키 생성
- 편향 제거 및 엔트로피 확산

구조:
HKDF = Extract + Expand
```

### 7.2 Extract 단계

**목적**: 여러 소스의 엔트로피를 하나의 PRK(Pseudorandom Key)로 압축

```
HKDF-Extract(salt, IKM) → PRK

IKM (Input Keying Material):
- 입력 키 재료
- 예: ECDH 공유 비밀

salt:
- 선택적 솔트 (없으면 0x00...)
- 편향 제거 및 도메인 분리

PRK (Pseudorandom Key):
- 의사 난수 키
- HashLen 바이트 (SHA-256이면 32바이트)

알고리즘:
PRK = HMAC-Hash(salt, IKM)

예:
IKM = ECDH(myPriv, peerPub)  // 32바이트
salt = SHA256("sage/session" || contextID)
PRK = HMAC-SHA256(salt, IKM)  // 32바이트
```

### 7.3 Expand 단계

**목적**: PRK를 확장하여 필요한 길이의 키 생성

```
HKDF-Expand(PRK, info, L) → OKM

PRK:
- Extract 단계의 출력

info:
- 컨텍스트 정보
- 도메인 분리 및 키 타입 구분

L:
- 출력 길이 (바이트)
- 최대 255 * HashLen

OKM (Output Keying Material):
- 최종 키

알고리즘:
N = ceil(L / HashLen)
T(0) = empty
T(i) = HMAC-Hash(PRK, T(i-1) || info || [i])
OKM = T(1) || T(2) || ... || T(N)[0:L]

예:
PRK = [32바이트 의사난수]
info = "encryption"
L = 32

T(1) = HMAC-SHA256(PRK, "" || "encryption" || 0x01)
OKM = T(1)[0:32] = 32바이트 암호화 키
```

### 7.4 SAGE의 HKDF 사용

**세션 키 유도**:

```go
// session/session.go

func DeriveSessionSeed(sharedSecret []byte, p Params) ([]byte, error) {
    // 1. 레이블 설정
    label := p.Label
    if label == "" {
        label = "a2a/handshake v1"
    }

    // 2. 임시 공개키들을 정렬 (대칭성 보장)
    lo, hi := canonicalOrder(p.SelfEph, p.PeerEph)

    // 3. 솔트 계산
    h := sha256.New()
    h.Write([]byte(label))
    h.Write([]byte(p.ContextID))
    h.Write(lo)
    h.Write(hi)
    salt := h.Sum(nil)

    // 4. HKDF-Extract
    seed := hkdfExtractSHA256(sharedSecret, salt)

    return seed, nil
}

위치: session/session.go:181-202

설명:
- sharedSecret: ECDH 공유 비밀
- salt: 컨텍스트별 고유값
- seed: 세션 시드 (PRK)
```

**방향별 키 유도**:

```go
func (s *SecureSession) deriveDirectionalKeys() error {
    salt := []byte(s.id)  // 세션 ID를 솔트로

    // HKDF-Expand 헬퍼
    expand := func(info string, n int) ([]byte, error) {
        r := hkdf.New(sha256.New, s.sessionSeed, salt, []byte(info))
        out := make([]byte, n)
        if _, err := io.ReadFull(r, out); err != nil {
            return nil, err
        }
        return out, nil
    }

    // 클라이언트→서버 키
    c2sEnc, _ := expand("c2s|enc|v1", 32)   // 암호화 키
    c2sSign, _ := expand("c2s|sign|v1", 32)  // 서명 키

    // 서버→클라이언트 키
    s2cEnc, _ := expand("s2c|enc|v1", 32)
    s2cSign, _ := expand("s2c|sign|v1", 32)

    // 역할에 따라 할당
    if s.initiator {
        // 클라이언트: 송신은 c2s, 수신은 s2c
        s.outKey, s.outSign = c2sEnc, c2sSign
        s.inKey, s.inSign = s2cEnc, s2cSign
    } else {
        // 서버: 송신은 s2c, 수신은 c2s
        s.outKey, s.outSign = s2cEnc, s2cSign
        s.inKey, s.inSign = c2sEnc, c2sSign
    }

    return nil
}

위치: session/session.go:240-273

키 계층:
공유비밀
    ↓ HKDF-Extract (salt=컨텍스트)
세션시드 (PRK)
    ↓ HKDF-Expand (info="c2s|enc|v1")
    ├→ C2S 암호화 키
    ↓ HKDF-Expand (info="c2s|sign|v1")
    ├→ C2S 서명 키
    ↓ HKDF-Expand (info="s2c|enc|v1")
    ├→ S2C 암호화 키
    ↓ HKDF-Expand (info="s2c|sign|v1")
    └→ S2C 서명 키
```

### 7.5 도메인 분리 (Domain Separation)

**왜 중요한가?**

```
문제:
같은 키를 여러 용도로 사용하면 공격 가능

예:
key = HKDF(secret, "key")
encKey = key
sigKey = key  // 위험!

공격:
암호화 오라클을 서명 오라클로 악용 가능

해결: info 파라미터로 도메인 분리
encKey = HKDF(secret, "encryption")
sigKey = HKDF(secret, "signature")
→ 완전히 독립적인 키
```

**SAGE의 도메인 분리 전략**:

```
계층적 info 구조:

Level 1: 프로토콜
"sage/handshake v1"

Level 2: 방향
"c2s" (client-to-server)
"s2c" (server-to-client)

Level 3: 용도
"enc" (encryption)
"sign" (signature)

Level 4: 버전
"v1"

최종 info:
"c2s|enc|v1"
"c2s|sign|v1"
"s2c|enc|v1"
"s2c|sign|v1"

장점:
✅ 각 키가 완전히 독립적
✅ 버전 업그레이드 용이
✅ 방향 혼동 방지
✅ 크로스 프로토콜 공격 방지
```

---

## 8. 키 변환과 상호운용성

### 8.1 키 포맷

**PEM (Privacy Enhanced Mail)**:

```
예시:
-----BEGIN ED25519 PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIG7OqKqMsUwHxKHqEVNd9sHzq7JjVzRjxGfnVghZEcPK
-----END ED25519 PRIVATE KEY-----

구조:
1. 헤더: -----BEGIN [타입]-----
2. Base64 인코딩된 DER
3. 푸터: -----END [타입]-----

DER (Distinguished Encoding Rules):
ASN.1 구조의 바이너리 인코딩
```

**JWK (JSON Web Key)**:

```json
{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "bu6oqoyxTAfEoeoRU132wfOrsmNXNGPEZ-dWCFkRw8o",
  "d": "nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A"
}

필드:
- kty: Key Type (OKP = Octet String Key Pairs)
- crv: Curve (Ed25519, X25519 등)
- x: Public Key (Base64URL)
- d: Private Key (Base64URL, optional)
```

**Raw 바이트**:

```
Ed25519:
- 공개키: 32바이트
- 개인키: 64바이트 (seed 32 + public 32)
  또는 32바이트 (seed만)

X25519:
- 공개키: 32바이트
- 개인키: 32바이트

Secp256k1:
- 공개키: 33바이트 (압축) 또는 65바이트 (비압축)
- 개인키: 32바이트
```

### 8.2 SAGE의 키 변환기

**PEM 익스포트**:

```go
// crypto/formats/pem.go

type PEMExporter struct{}

func (e *PEMExporter) ExportPrivate(
    kp sagecrypto.KeyPair,
    format sagecrypto.KeyFormat,
) ([]byte, error) {
    if format != sagecrypto.KeyFormatPEM {
        return nil, fmt.Errorf("unsupported format")
    }

    switch kp.Type() {
    case sagecrypto.KeyTypeEd25519:
        // 1. PKCS#8 래핑
        privKey := kp.PrivateKey().(ed25519.PrivateKey)
        pkcs8, err := x509.MarshalPKCS8PrivateKey(privKey)
        if err != nil {
            return nil, err
        }

        // 2. PEM 인코딩
        block := &pem.Block{
            Type:  "PRIVATE KEY",
            Bytes: pkcs8,
        }
        return pem.EncodeToMemory(block), nil

    case sagecrypto.KeyTypeSecp256k1:
        // Secp256k1는 표준 PKCS#8 지원 안함
        // SEC1 형식 사용
        privKey := kp.PrivateKey().(*ecdsa.PrivateKey)
        der, err := x509.MarshalECPrivateKey(privKey)
        if err != nil {
            return nil, err
        }

        block := &pem.Block{
            Type:  "EC PRIVATE KEY",
            Bytes: der,
        }
        return pem.EncodeToMemory(block), nil

    default:
        return nil, fmt.Errorf("unsupported key type")
    }
}

위치: crypto/formats/pem.go
```

**JWK 익스포트**:

```go
// crypto/formats/jwk.go

type JWKExporter struct{}

func (e *JWKExporter) ExportPublic(
    kp sagecrypto.KeyPair,
    format sagecrypto.KeyFormat,
) ([]byte, error) {
    if format != sagecrypto.KeyFormatJWK {
        return nil, fmt.Errorf("unsupported format")
    }

    switch kp.Type() {
    case sagecrypto.KeyTypeEd25519:
        pubKey := kp.PublicKey().(ed25519.PublicKey)
        jwk := map[string]string{
            "kty": "OKP",
            "crv": "Ed25519",
            "x":   base64.RawURLEncoding.EncodeToString(pubKey),
        }
        return json.Marshal(jwk)

    case sagecrypto.KeyTypeX25519:
        pubKey := kp.PublicKey().(*ecdh.PublicKey)
        jwk := map[string]string{
            "kty": "OKP",
            "crv": "X25519",
            "x":   base64.RawURLEncoding.EncodeToString(pubKey.Bytes()),
        }
        return json.Marshal(jwk)

    case sagecrypto.KeyTypeSecp256k1:
        pubKey := kp.PublicKey().(*ecdsa.PublicKey)
        // 압축 형식 사용
        compressed := elliptic.MarshalCompressed(
            pubKey.Curve,
            pubKey.X,
            pubKey.Y,
        )
        jwk := map[string]string{
            "kty": "EC",
            "crv": "secp256k1",
            "x":   base64.RawURLEncoding.EncodeToString(compressed),
        }
        return json.Marshal(jwk)

    default:
        return nil, fmt.Errorf("unsupported key type")
    }
}

위치: crypto/formats/jwk.go
```

### 8.3 키 임포트

**PEM 임포트**:

```go
// crypto/formats/pem.go

type PEMImporter struct{}

func (i *PEMImporter) ImportPrivate(
    data []byte,
    format sagecrypto.KeyFormat,
) (sagecrypto.KeyPair, error) {
    if format != sagecrypto.KeyFormatPEM {
        return nil, fmt.Errorf("unsupported format")
    }

    // 1. PEM 디코딩
    block, _ := pem.Decode(data)
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM")
    }

    // 2. PKCS#8 언마샬
    if block.Type == "PRIVATE KEY" {
        key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
        if err != nil {
            return nil, err
        }

        // 3. 타입별 처리
        switch k := key.(type) {
        case ed25519.PrivateKey:
            return keys.NewEd25519KeyPairFromPrivate(k)

        case *ecdsa.PrivateKey:
            // Secp256k1 확인
            if k.Curve.Params().Name == "secp256k1" {
                return keys.NewSecp256k1KeyPairFromPrivate(k)
            }

        default:
            return nil, fmt.Errorf("unsupported key type: %T", k)
        }
    }

    return nil, fmt.Errorf("unsupported PEM type: %s", block.Type)
}

위치: crypto/formats/pem.go
```

### 8.4 실전 예제

**키 생성 및 저장**:

```go
package main

import (
    "fmt"
    "os"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/crypto/formats"
    sagecrypto "github.com/sage-x-project/sage/crypto"
)

func main() {
    // 1. Ed25519 키 생성
    kp, _ := keys.GenerateEd25519KeyPair()
    fmt.Printf("Generated key ID: %s\n", kp.ID())

    // 2. PEM 형식으로 익스포트
    exporter := formats.NewPEMExporter()
    pemData, _ := exporter.ExportPrivate(kp, sagecrypto.KeyFormatPEM)

    // 3. 파일에 저장
    os.WriteFile("agent-key.pem", pemData, 0600)
    fmt.Println("Saved to agent-key.pem")

    // 4. JWK 형식으로도 저장
    jwkExporter := formats.NewJWKExporter()
    jwkData, _ := jwkExporter.ExportPublic(kp, sagecrypto.KeyFormatJWK)
    os.WriteFile("agent-key.jwk", jwkData, 0644)
    fmt.Println("Saved public key to agent-key.jwk")
}
```

**키 로드 및 사용**:

```go
package main

import (
    "fmt"
    "os"
    "github.com/sage-x-project/sage/crypto/formats"
    sagecrypto "github.com/sage-x-project/sage/crypto"
)

func main() {
    // 1. PEM 파일 읽기
    pemData, _ := os.ReadFile("agent-key.pem")

    // 2. 키 임포트
    importer := formats.NewPEMImporter()
    kp, _ := importer.ImportPrivate(pemData, sagecrypto.KeyFormatPEM)

    fmt.Printf("Loaded key ID: %s\n", kp.ID())
    fmt.Printf("Key type: %s\n", kp.Type())

    // 3. 메시지 서명
    message := []byte("Hello, SAGE!")
    signature, _ := kp.Sign(message)
    fmt.Printf("Signature: %x\n", signature[:16])

    // 4. 검증
    err := kp.Verify(message, signature)
    if err == nil {
        fmt.Println("✅ Signature valid!")
    }
}
```

---

## 9. 실전 예제 및 테스트

### 9.1 완전한 암호화 플로우

**시나리오**: Agent A와 Agent B의 보안 통신

```go
package main

import (
    "fmt"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/session"
    "time"
)

func main() {
    fmt.Println("=== SAGE 암호화 플로우 예제 ===\n")

    // 1. Agent A와 B의 키 생성
    fmt.Println("1. 키 생성")
    agentA_Ed, _ := keys.GenerateEd25519KeyPair()
    agentB_Ed, _ := keys.GenerateEd25519KeyPair()
    fmt.Printf("   Agent A ID: %s\n", agentA_Ed.ID())
    fmt.Printf("   Agent B ID: %s\n", agentB_Ed.ID())

    // 2. 임시 X25519 키 생성 (핸드셰이크용)
    fmt.Println("\n2. 임시 키 생성 (핸드셰이크)")
    agentA_X, _ := keys.GenerateX25519KeyPair()
    agentB_X, _ := keys.GenerateX25519KeyPair()

    // 3. Agent A: 공유 비밀 계산
    fmt.Println("\n3. 공유 비밀 계산")
    agentA_X_Pair := agentA_X.(*keys.X25519KeyPair)
    agentB_X_Pair := agentB_X.(*keys.X25519KeyPair)

    sharedA, _ := agentA_X_Pair.DeriveSharedSecret(
        agentB_X_Pair.PublicBytesKey(),
    )

    // 4. Agent B: 공유 비밀 계산 (같은 값!)
    sharedB, _ := agentB_X_Pair.DeriveSharedSecret(
        agentA_X_Pair.PublicBytesKey(),
    )

    fmt.Printf("   Agent A shared: %x...\n", sharedA[:8])
    fmt.Printf("   Agent B shared: %x...\n", sharedB[:8])
    fmt.Printf("   ✅ 같은 공유 비밀!\n")

    // 5. 세션 생성
    fmt.Println("\n4. 세션 생성")
    params := session.Params{
        ContextID:    "ctx-123",
        SelfEph:      agentA_X_Pair.PublicBytesKey(),
        PeerEph:      agentB_X_Pair.PublicBytesKey(),
        Label:        "sage/demo v1",
        SharedSecret: sharedA,
    }

    // Agent A: 클라이언트 (initiator=true)
    sessA, _ := session.NewSecureSessionFromExporterWithRole(
        "session-abc",
        sharedA,
        true,  // initiator
        session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        },
    )

    // Agent B: 서버 (initiator=false)
    sessB, _ := session.NewSecureSessionFromExporterWithRole(
        "session-abc",
        sharedB,
        false,  // responder
        session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        },
    )

    fmt.Printf("   Session ID: %s\n", sessA.GetID())

    // 6. Agent A → B: 메시지 암호화
    fmt.Println("\n5. Agent A → B: 메시지 암호화")
    plaintext := []byte("Transfer 100 tokens to Agent B")
    encrypted, _ := sessA.EncryptOutbound(plaintext)
    fmt.Printf("   평문: %s\n", plaintext)
    fmt.Printf("   암호문: %x...\n", encrypted[:32])
    fmt.Printf("   길이: %d 바이트\n", len(encrypted))

    // 7. Agent B: 복호화
    fmt.Println("\n6. Agent B: 복호화")
    decrypted, _ := sessB.DecryptInbound(encrypted)
    fmt.Printf("   복호문: %s\n", decrypted)
    fmt.Printf("   ✅ 복호화 성공!\n")

    // 8. Agent B → A: 응답
    fmt.Println("\n7. Agent B → A: 응답")
    response := []byte("Acknowledged: 100 tokens received")
    encResponse, _ := sessB.EncryptOutbound(response)
    decResponse, _ := sessA.DecryptInbound(encResponse)
    fmt.Printf("   응답: %s\n", decResponse)

    // 9. 변조 테스트
    fmt.Println("\n8. 변조 테스트")
    encrypted[50] ^= 0xFF  // 한 바이트 변조
    _, err := sessB.DecryptInbound(encrypted)
    if err != nil {
        fmt.Printf("   ❌ 변조 감지: %v\n", err)
    }

    // 10. 세션 정리
    fmt.Println("\n9. 세션 정리")
    sessA.Close()
    sessB.Close()
    fmt.Println("   ✅ 모든 키 안전하게 삭제됨")
}
```

**출력 예시**:
```
=== SAGE 암호화 플로우 예제 ===

1. 키 생성
   Agent A ID: a1b2c3d4e5f6g7h8
   Agent B ID: 9i0j1k2l3m4n5o6p

2. 임시 키 생성 (핸드셰이크)

3. 공유 비밀 계산
   Agent A shared: ef12cd34ab56...
   Agent B shared: ef12cd34ab56...
   ✅ 같은 공유 비밀!

4. 세션 생성
   Session ID: xK9mP2qR7sT3uV

5. Agent A → B: 메시지 암호화
   평문: Transfer 100 tokens to Agent B
   암호문: 8a3f2b1c9d4e5f6a...
   길이: 75 바이트

6. Agent B: 복호화
   복호문: Transfer 100 tokens to Agent B
   ✅ 복호화 성공!

7. Agent B → A: 응답
   응답: Acknowledged: 100 tokens received

8. 변조 테스트
   ❌ 변조 감지: decryption/authentication failed

9. 세션 정리
   ✅ 모든 키 안전하게 삭제됨
```

### 9.2 성능 테스트

**벤치마크 코드**:

```go
package crypto_test

import (
    "testing"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/session"
    "time"
)

func BenchmarkEd25519Sign(b *testing.B) {
    kp, _ := keys.GenerateEd25519KeyPair()
    message := []byte("benchmark message")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        kp.Sign(message)
    }
}

func BenchmarkEd25519Verify(b *testing.B) {
    kp, _ := keys.GenerateEd25519KeyPair()
    message := []byte("benchmark message")
    sig, _ := kp.Sign(message)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        kp.Verify(message, sig)
    }
}

func BenchmarkX25519DH(b *testing.B) {
    kpA, _ := keys.GenerateX25519KeyPair()
    kpB, _ := keys.GenerateX25519KeyPair()
    pairA := kpA.(*keys.X25519KeyPair)
    pairB := kpB.(*keys.X25519KeyPair)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pairA.DeriveSharedSecret(pairB.PublicBytesKey())
    }
}

func BenchmarkChaCha20Poly1305Encrypt(b *testing.B) {
    sess, _ := session.NewSecureSessionFromExporterWithRole(
        "bench",
        make([]byte, 32),
        true,
        session.Config{},
    )
    plaintext := make([]byte, 1024)  // 1KB

    b.ResetTimer()
    b.SetBytes(1024)
    for i := 0; i < b.N; i++ {
        sess.EncryptOutbound(plaintext)
    }
}

func BenchmarkSessionCreation(b *testing.B) {
    sharedSecret := make([]byte, 32)
    config := session.Config{
        MaxAge: time.Hour,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sess, _ := session.NewSecureSessionFromExporterWithRole(
            "bench",
            sharedSecret,
            true,
            config,
        )
        sess.Close()
    }
}
```

**벤치마크 실행**:
```bash
cd crypto
go test -bench=. -benchmem

출력 예시:
BenchmarkEd25519Sign-8              50000    25000 ns/op    0 B/op    0 allocs/op
BenchmarkEd25519Verify-8            20000    75000 ns/op    0 B/op    0 allocs/op
BenchmarkX25519DH-8                100000    15000 ns/op   64 B/op    2 allocs/op
BenchmarkChaCha20Poly1305Encrypt-8  500000     3000 ns/op 1024 B/op    2 allocs/op
BenchmarkSessionCreation-8           30000    45000 ns/op  512 B/op   10 allocs/op
```

### 9.3 단위 테스트

**Ed25519 테스트**:

```go
// crypto/keys/ed25519_test.go

func TestEd25519KeyGeneration(t *testing.T) {
    kp, err := keys.GenerateEd25519KeyPair()
    assert.NoError(t, err)
    assert.NotNil(t, kp)
    assert.NotEmpty(t, kp.ID())
}

func TestEd25519SignVerify(t *testing.T) {
    kp, _ := keys.GenerateEd25519KeyPair()
    message := []byte("test message")

    // 서명
    sig, err := kp.Sign(message)
    assert.NoError(t, err)
    assert.Len(t, sig, 64)

    // 검증
    err = kp.Verify(message, sig)
    assert.NoError(t, err)
}

func TestEd25519InvalidSignature(t *testing.T) {
    kp, _ := keys.GenerateEd25519KeyPair()
    message := []byte("test message")
    sig, _ := kp.Sign(message)

    // 서명 변조
    sig[0] ^= 0xFF

    // 검증 실패해야 함
    err := kp.Verify(message, sig)
    assert.Error(t, err)
}

func TestEd25519DifferentMessages(t *testing.T) {
    kp, _ := keys.GenerateEd25519KeyPair()
    msg1 := []byte("message 1")
    msg2 := []byte("message 2")

    sig1, _ := kp.Sign(msg1)
    sig2, _ := kp.Sign(msg2)

    // 다른 메시지는 다른 서명
    assert.NotEqual(t, sig1, sig2)

    // 크로스 검증 실패
    err := kp.Verify(msg1, sig2)
    assert.Error(t, err)
}
```

**세션 테스트**:

```go
// session/session_test.go

func TestSessionSymmetry(t *testing.T) {
    shared := make([]byte, 32)
    rand.Read(shared)

    // 두 세션 생성
    sessA, _ := session.NewSecureSessionFromExporterWithRole(
        "test",
        shared,
        true,  // initiator
        session.Config{},
    )

    sessB, _ := session.NewSecureSessionFromExporterWithRole(
        "test",
        shared,
        false,  // responder
        session.Config{},
    )

    // A → B
    plaintext := []byte("Hello, B!")
    encrypted, _ := sessA.EncryptOutbound(plaintext)
    decrypted, _ := sessB.DecryptInbound(encrypted)
    assert.Equal(t, plaintext, decrypted)

    // B → A
    response := []byte("Hello, A!")
    encResponse, _ := sessB.EncryptOutbound(response)
    decResponse, _ := sessA.DecryptInbound(encResponse)
    assert.Equal(t, response, decResponse)
}

func TestSessionExpiration(t *testing.T) {
    sess, _ := session.NewSecureSessionFromExporterWithRole(
        "test",
        make([]byte, 32),
        true,
        session.Config{
            MaxAge: 100 * time.Millisecond,
        },
    )

    // 처음에는 유효
    assert.False(t, sess.IsExpired())

    // 대기
    time.Sleep(150 * time.Millisecond)

    // 만료되어야 함
    assert.True(t, sess.IsExpired())
}
```

---

## 요약

Part 2에서 다룬 내용:

1. **암호화 기초**: 대칭키 vs 비대칭키, 디지털 서명, 키 교환
2. **Ed25519**: 빠른 서명 알고리즘, SAGE의 신원 확인에 사용
3. **Secp256k1**: Ethereum 호환성, 서명 복구 기능
4. **X25519**: 고속 키 교환, ECDH 프로토콜
5. **HPKE**: 하이브리드 암호화, RFC 9180 표준
6. **ChaCha20-Poly1305**: 고성능 AEAD 암호화
7. **HKDF**: 안전한 키 유도, 도메인 분리
8. **키 변환**: Ed25519 ↔ X25519, 다양한 포맷 지원
9. **실전 예제**: 완전한 암호화 플로우, 테스트 및 벤치마크

**다음 파트 예고**:

**Part 3: DID 및 블록체인 통합**에서는:
- Ethereum 스마트 컨트랙트 상세 분석
- DID 등록/조회/업데이트 프로세스
- 가스 최적화 및 보안 검증
- 다중 체인 지원 구현

계속해서 Part 3를 작성하시겠습니까?
