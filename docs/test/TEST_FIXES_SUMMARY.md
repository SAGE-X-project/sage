# SAGE 테스트 수정 요약

**작성일**: 2025-10-24
**상태**: ✅ 모든 테스트 통과

## 개요

SPECIFICATION_VERIFICATION_MATRIX.md 문서의 모든 테스트를 실행하여 실패하는 케이스를 찾고 수정했습니다.

## 발견된 실패 케이스

### 1. pkg/agent/core - VerifyMessageFromHeaders 테스트 실패

**문제**: Nonce replay attack 오류
```
Error: Should be empty, but was nonce replay attack detected: nonce nonce123 has already been used
```

**원인**:
- 여러 테스트에서 동일한 nonce `"nonce123"`을 사용
- 이전 테스트에서 사용한 nonce가 nonce 관리자에 기록되어 있어 다음 테스트에서 replay attack으로 감지됨

**수정 위치**: `pkg/agent/core/verification_service_test.go:333`

**수정 내용**:
```go
// Before
"X-Nonce": "nonce123",

// After
"X-Nonce": "nonce456", // Use unique nonce for this test
```

**결과**: ✅ 테스트 통과

---

### 2. pkg/agent/did - ECDSA 공개키 처리 실패 (3개 테스트)

**실패 테스트**:
1. `TestGenerateA2ACardWithProof_ECDSA`
2. `TestGenerateKeyProofOfPossession_ECDSA`
3. `TestVerifyAllKeyProofs`

**문제**:
```
Error: failed to decompress public key: invalid public key
```

**원인**:
- `MarshalPublicKey` 함수가 secp256k1 공개키를 64바이트 uncompressed 형식(0x04 prefix 없이)으로 반환
- V4 컨트랙트가 온체인에서 비용이 많이 드는 압축 해제를 피하기 위해 uncompressed 형식 사용
- 하지만 `DecompressPubkey` 함수는 33바이트 압축 형식만 처리 가능
- 64바이트 raw 형식을 받으면 "invalid public key" 오류 발생

**수정 위치 1**: `pkg/agent/did/a2a_proof.go:210-245`

**수정 내용**:
```go
// Before
// Decompress public key
pubKey, err := ethcrypto.DecompressPubkey(pubKeyBytes)
if err != nil {
    return false, fmt.Errorf("failed to decompress public key: %w", err)
}

// After
// Handle both compressed (33 bytes) and uncompressed formats (64 or 65 bytes)
var pubKey *ecdsa.PublicKey
if len(pubKeyBytes) == 64 {
    // Raw format (64 bytes: x || y) - prepend 0x04 for standard uncompressed format
    pubKeyBytes = append([]byte{0x04}, pubKeyBytes...)
}

if len(pubKeyBytes) == 33 {
    // Compressed format - use DecompressPubkey
    pubKey, err = ethcrypto.DecompressPubkey(pubKeyBytes)
    if err != nil {
        return false, fmt.Errorf("failed to decompress public key: %w", err)
    }
} else if len(pubKeyBytes) == 65 {
    // Uncompressed format - use UnmarshalPubkey
    pubKey, err = ethcrypto.UnmarshalPubkey(pubKeyBytes)
    if err != nil {
        return false, fmt.Errorf("failed to unmarshal public key: %w", err)
    }
} else {
    return false, fmt.Errorf("invalid public key length: %d (expected 33, 64, or 65 bytes)", len(pubKeyBytes))
}
```

**수정 위치 2**: `pkg/agent/did/key_proof.go:123-144`

**수정 내용**: a2a_proof.go와 동일한 로직 적용

**결과**: ✅ 모든 ECDSA 테스트 통과

---

### 3. pkg/agent/did - MarshalUnmarshalPublicKey 테스트 실패

**문제**:
```
Error: Not equal:
        expected: 33
        actual  : 64
```

**원인**:
- 테스트가 secp256k1 공개키가 33바이트(압축 형식)일 것으로 기대
- 하지만 `MarshalPublicKey`는 V4 컨트랙트 요구사항에 따라 64바이트(uncompressed 형식) 반환

**수정 위치**: `pkg/agent/did/utils_test.go:258-260`

**수정 내용**:
```go
// Before
// secp256k1 compressed format is 33 bytes
assert.Equal(t, 33, len(marshaled))

// After
// secp256k1 uncompressed format is 64 bytes (without 0x04 prefix)
// V4 contract uses uncompressed format to avoid expensive decompression on-chain
assert.Equal(t, 64, len(marshaled))
```

**결과**: ✅ 테스트 통과

---

### 4. pkg/agent/did/ethereum - RegisterKeyTypeValidation 테스트 실패

**문제**: Nil pointer dereference panic
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**원인**:
- 테스트가 블록체인 연결 없이 키 타입 검증만 테스트하려고 함
- `EthereumClient`의 `contract` 필드가 nil
- 유효한 Secp256k1 키로 테스트할 때 키 타입 검증을 통과하고 `contract.Transact` 호출 시 nil pointer dereference 발생

**수정 위치**: `pkg/agent/did/ethereum/client.go:101-111`

**수정 내용**:
```go
// Register registers a new agent on Ethereum
func (c *EthereumClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
    // Validate key type first (before checking client initialization)
    if req.KeyPair.Type() != sagecrypto.KeyTypeSecp256k1 {
        return nil, fmt.Errorf("ethereum requires Secp256k1 keys")
    }

    // Validate client is initialized
    if c.contract == nil {
        return nil, fmt.Errorf("ethereum client not properly initialized: contract is nil")
    }

    // ... rest of function
}
```

**주요 변경사항**:
1. 키 타입 검증을 먼저 수행 (contract 초기화 체크보다 먼저)
2. contract nil 체크 추가하여 panic 대신 적절한 오류 메시지 반환

**결과**: ✅ 모든 서브테스트 통과
- Valid Secp256k1 key: PASS
- Invalid Ed25519 key: PASS
- Invalid X25519 key: PASS

---

## 수정 파일 요약

| 파일 | 수정 내용 | 이유 |
|------|----------|------|
| `pkg/agent/core/verification_service_test.go` | Nonce 값 변경 (nonce123 → nonce456) | 테스트 독립성 확보 |
| `pkg/agent/did/a2a_proof.go` | ECDSA 공개키 처리 로직 개선 | 64바이트 uncompressed 형식 지원 |
| `pkg/agent/did/key_proof.go` | ECDSA 공개키 처리 로직 개선 | 64바이트 uncompressed 형식 지원 |
| `pkg/agent/did/utils_test.go` | 테스트 기대값 수정 (33 → 64 bytes) | V4 컨트랙트 요구사항 반영 |
| `pkg/agent/did/ethereum/client.go` | 키 타입 검증 순서 및 nil 체크 추가 | Panic 방지 및 적절한 오류 처리 |

## 테스트 결과

### 수정 전

```
FAIL    github.com/sage-x-project/sage/pkg/agent/core                0.749s
FAIL    github.com/sage-x-project/sage/pkg/agent/did                 1.012s
FAIL    github.com/sage-x-project/sage/pkg/agent/did/ethereum        0.742s
```

**총 실패**: 3개 패키지, 8개 테스트

### 수정 후

```
ok      github.com/sage-x-project/sage/pkg/agent/core                (cached)
ok      github.com/sage-x-project/sage/pkg/agent/did                 0.531s
ok      github.com/sage-x-project/sage/pkg/agent/did/ethereum        (cached)
```

**총 성공**: ✅ 모든 테스트 통과

### 전체 테스트 suite 결과

```bash
go test ./...
```

**결과**: ✅ 모든 패키지 테스트 통과 (0 failures)

## 기술적 학습 사항

### 1. Secp256k1 공개키 형식

Ethereum의 secp256k1 공개키는 3가지 형식이 있습니다:

1. **압축 형식** (33 bytes):
   - 형식: `0x02/0x03 + X`
   - Y 좌표를 복구 가능
   - 저장 공간 효율적

2. **비압축 형식** (65 bytes):
   - 형식: `0x04 + X + Y`
   - 표준 uncompressed 형식

3. **Raw 형식** (64 bytes):
   - 형식: `X + Y` (prefix 없음)
   - V4 컨트랙트가 사용하는 형식
   - 온체인 압축 해제 비용 절감

### 2. Ethereum crypto 라이브러리 함수

- `ethcrypto.DecompressPubkey(data)`: 33바이트 압축 형식 → *ecdsa.PublicKey
- `ethcrypto.UnmarshalPubkey(data)`: 65바이트 uncompressed 형식 → *ecdsa.PublicKey
- 64바이트 raw 형식은 0x04를 prepend하여 65바이트로 변환 후 UnmarshalPubkey 사용

### 3. 테스트 독립성

테스트는 다음을 보장해야 합니다:
- 각 테스트가 독립적으로 실행 가능
- 공유 상태(nonce 관리자 등)를 사용할 때 고유한 값 사용
- 테스트 순서에 의존하지 않음

### 4. 방어적 프로그래밍

Public API 함수는 다음을 검증해야 합니다:
1. 입력 파라미터 검증 (키 타입 등)
2. 내부 상태 검증 (contract != nil 등)
3. 검증 순서 최적화 (빠른 실패 원칙)

## 향후 고려사항

1. **통합 테스트 환경**:
   - 블록체인 연결이 필요한 테스트를 위한 mock 환경 구축
   - 키 타입 검증과 같은 유닛 테스트와 실제 블록체인 통합 테스트 분리

2. **공개키 형식 통일**:
   - V4 컨트랙트가 64바이트 raw 형식을 사용하는 이유 문서화
   - 모든 관련 함수가 다양한 형식을 지원하도록 일관성 유지

3. **테스트 커버리지**:
   - 공개키 형식 변환 로직에 대한 추가 테스트
   - Edge case 처리 검증

## 참고 자료

- SPECIFICATION_VERIFICATION_MATRIX.md - 전체 검증 매트릭스
- CLAUDE.md - AI 지원 개발 가이드라인
- Ethereum Yellow Paper - ECDSA 및 secp256k1 명세
- RFC 9421 - HTTP Message Signatures

---

**작성**: Claude Code
**검증 상태**: ✅ 모든 테스트 통과
**커밋 권장**: 이 수정사항들은 테스트 안정성과 코드 품질을 크게 향상시킵니다.
