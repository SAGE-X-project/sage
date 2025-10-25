# SAGE 테스트 문제 체크리스트

**작성일**: 2025-10-24
**최종 업데이트**: 2025-10-24

## 📋 전체 요약

### ✅ 수정 완료된 문제 (5개)

| # | 패키지 | 테스트 | 문제 | 상태 |
|---|--------|--------|------|------|
| 1 | `pkg/agent/core` | `TestVerificationService/VerifyMessageFromHeaders` | Nonce replay attack | ✅ 수정 완료 |
| 2 | `pkg/agent/did` | `TestGenerateA2ACardWithProof_ECDSA` | ECDSA 공개키 형식 처리 | ✅ 수정 완료 |
| 3 | `pkg/agent/did` | `TestGenerateKeyProofOfPossession_ECDSA` | ECDSA 공개키 형식 처리 | ✅ 수정 완료 |
| 4 | `pkg/agent/did` | `TestVerifyAllKeyProofs` | ECDSA 공개키 형식 처리 | ✅ 수정 완료 |
| 5 | `pkg/agent/did` | `TestMarshalUnmarshalPublicKey/Secp256k1_key` | 테스트 기대값 오류 | ✅ 수정 완료 |
| 6 | `pkg/agent/did/ethereum` | `TestRegisterKeyTypeValidation` | Nil pointer panic | ✅ 수정 완료 |

### ⚠️ 환경 의존 문제 (1개)

| # | 패키지 | 테스트 | 문제 | 상태 |
|---|--------|--------|------|------|
| 7 | `tests` | `TestTransactionSendAndConfirm` | 블록체인 노드 미실행 | ⚠️ 환경 설정 필요 |

---

## ✅ 1. TestVerificationService/VerifyMessageFromHeaders

### 문제
```
Error: nonce replay attack detected: nonce nonce123 has already been used
```

### 원인
- 여러 테스트에서 동일한 nonce `"nonce123"` 사용
- Nonce 관리자가 이전 테스트의 nonce를 기억하여 replay attack으로 감지

### 수정 내용
**파일**: `pkg/agent/core/verification_service_test.go`

```diff
- "X-Nonce": "nonce123",
+ "X-Nonce": "nonce456", // Use unique nonce for this test
```

### 교훈
- 테스트 독립성: 각 테스트는 고유한 데이터 사용
- 공유 상태 주의: nonce 관리자 같은 싱글톤 서비스 사용 시 주의

---

## ✅ 2-4. ECDSA 공개키 처리 문제 (3개 테스트)

### 영향받는 테스트
1. `TestGenerateA2ACardWithProof_ECDSA`
2. `TestGenerateKeyProofOfPossession_ECDSA`
3. `TestVerifyAllKeyProofs`

### 문제
```
Error: failed to decompress public key: invalid public key
```

### 원인
- `MarshalPublicKey`가 secp256k1 공개키를 **64바이트 raw 형식**으로 반환
- V4 컨트랙트는 온체인 압축 해제 비용을 피하기 위해 uncompressed 형식 사용
- 하지만 코드가 `ethcrypto.DecompressPubkey`만 사용 (33바이트 압축 형식만 처리)

### Secp256k1 공개키 형식
| 형식 | 크기 | 구조 | 사용처 |
|------|------|------|--------|
| 압축 | 33 bytes | `0x02/0x03 + X` | 일반적인 사용 |
| Uncompressed | 65 bytes | `0x04 + X + Y` | 표준 형식 |
| Raw | 64 bytes | `X + Y` | V4 컨트랙트 |

### 수정 내용

**파일 1**: `pkg/agent/did/a2a_proof.go`

```go
// Before
pubKey, err := ethcrypto.DecompressPubkey(pubKeyBytes)

// After
var pubKey *ecdsa.PublicKey
if len(pubKeyBytes) == 64 {
    // Raw format - prepend 0x04
    pubKeyBytes = append([]byte{0x04}, pubKeyBytes...)
}

if len(pubKeyBytes) == 33 {
    pubKey, err = ethcrypto.DecompressPubkey(pubKeyBytes)
} else if len(pubKeyBytes) == 65 {
    pubKey, err = ethcrypto.UnmarshalPubkey(pubKeyBytes)
} else {
    return fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
}
```

**파일 2**: `pkg/agent/did/key_proof.go` (동일한 로직)

### 교훈
- 다양한 데이터 형식 지원 필요
- Ethereum 라이브러리 함수 차이 이해:
  - `DecompressPubkey`: 33 bytes → *ecdsa.PublicKey
  - `UnmarshalPubkey`: 65 bytes → *ecdsa.PublicKey

---

## ✅ 5. TestMarshalUnmarshalPublicKey/Secp256k1_key

### 문제
```
Error: Not equal: expected: 33, actual: 64
```

### 원인
- 테스트가 압축 형식(33 bytes) 기대
- 실제 코드는 V4 컨트랙트 요구사항에 따라 raw 형식(64 bytes) 반환

### 수정 내용
**파일**: `pkg/agent/did/utils_test.go`

```diff
- // secp256k1 compressed format is 33 bytes
- assert.Equal(t, 33, len(marshaled))
+ // secp256k1 uncompressed format is 64 bytes (without 0x04 prefix)
+ // V4 contract uses uncompressed format to avoid expensive decompression on-chain
+ assert.Equal(t, 64, len(marshaled))
```

### 교훈
- 테스트는 실제 구현을 반영해야 함
- 왜 특정 형식을 사용하는지 주석으로 명확히 문서화

---

## ✅ 6. TestRegisterKeyTypeValidation

### 문제
```
panic: runtime error: invalid memory address or nil pointer dereference
```

### 원인
1. 테스트가 블록체인 연결 없이 키 타입 검증만 테스트하려 함
2. `EthereumClient.contract` 필드가 nil
3. 유효한 Secp256k1 키로 테스트 → 키 타입 검증 통과 → `contract.Transact` 호출 시 panic

### 수정 내용
**파일**: `pkg/agent/did/ethereum/client.go`

```go
func (c *EthereumClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
    // 1. Validate key type first (fast fail)
    if req.KeyPair.Type() != sagecrypto.KeyTypeSecp256k1 {
        return nil, fmt.Errorf("ethereum requires Secp256k1 keys")
    }

    // 2. Validate client is initialized
    if c.contract == nil {
        return nil, fmt.Errorf("ethereum client not properly initialized: contract is nil")
    }

    // ... rest of function
}
```

### 핵심 변경
1. **검증 순서 최적화**: 빠른 검증(키 타입)을 먼저 수행
2. **방어적 프로그래밍**: nil 체크로 panic 대신 적절한 오류 반환

### 교훈
- Fail-fast 원칙: 빠른 검증을 먼저 수행
- Nil pointer 방지: 중요한 필드는 항상 nil 체크
- 유닛 테스트 가능성: 초기화 없이도 기본 검증 로직 테스트 가능

---

## ⚠️ 7. TestTransactionSendAndConfirm (환경 의존)

### 문제
```
Error: Post "http://localhost:8545": dial tcp [::1]:8545: connect: connection refused
```

### 원인
- 로컬 블록체인 노드(Hardhat/Anvil)가 실행되지 않음
- 이 테스트는 실제 블록체인 연결이 필요한 통합 테스트

### 해결 방법

#### Option 1: 로컬 노드 실행 (권장)

```bash
# Hardhat 사용
npx hardhat node

# 또는 Anvil 사용
anvil
```

#### Option 2: 테스트 스킵

```bash
# 블록체인 통합 테스트 제외하고 실행
go test ./pkg/... ./cmd/... ./internal/...
```

#### Option 3: CI/CD에서 자동화

```yaml
# .github/workflows/test.yml
- name: Start local blockchain
  run: npx hardhat node &

- name: Run tests
  run: go test ./...
```

### 테스트 분류
| 타입 | 패키지 | 블록체인 필요 | 설명 |
|------|--------|---------------|------|
| 유닛 테스트 | `pkg/...` | ❌ | 빠른 로직 검증 |
| 통합 테스트 | `tests/integration` | ⚠️ 일부 | HPKE, 세션 등 |
| 블록체인 테스트 | `tests/` | ✅ | 트랜잭션, DID 등록 |

---

## 📊 최종 상태

### 코드 수정 필요 테스트
- **수정 전**: 6개 실패
- **수정 후**: ✅ 0개 실패 (모두 통과)

### 환경 설정 필요 테스트
- **블록체인 노드 필요**: 1개 (TestTransactionSendAndConfirm)
- **해결 방법**: 로컬 노드 실행 또는 CI/CD 자동화

### 전체 패키지 테스트 결과

```bash
# 코어 패키지 (블록체인 없이 실행 가능)
go test ./pkg/... ./cmd/... ./internal/...
```
**결과**: ✅ 100% 통과

```bash
# 전체 테스트 (블록체인 노드 필요)
go test ./...
```
**결과**: ⚠️ 1개 실패 (환경 설정 필요)

---

## 🛠️ 수정된 파일 목록

| 파일 | 변경 내용 | 라인 |
|------|----------|------|
| `pkg/agent/core/verification_service_test.go` | Nonce 값 변경 | 333 |
| `pkg/agent/did/a2a_proof.go` | ECDSA 키 처리 로직 개선 | 210-245 |
| `pkg/agent/did/key_proof.go` | ECDSA 키 처리 로직 개선 | 123-144 |
| `pkg/agent/did/utils_test.go` | 테스트 기대값 수정 | 258-260 |
| `pkg/agent/did/ethereum/client.go` | 검증 순서 및 nil 체크 | 101-111 |

---

## 🎯 권장 사항

### 즉시 조치
1. ✅ **코드 수정**: 모두 완료
2. ⚠️ **CI/CD 설정**: 블록체인 노드 자동 시작 추가

### 장기 개선
1. **테스트 분리**: 유닛/통합/E2E 테스트 명확히 구분
2. **Mock 활용**: 블록체인 의존성을 mock으로 대체 가능한 부분 개선
3. **문서화**: 각 테스트의 사전 요구사항 명시

### 실행 가이드

```bash
# 1. 로컬 개발 (블록체인 없이)
go test ./pkg/... ./cmd/... ./internal/...

# 2. 블록체인 노드 시작
anvil  # 또는 npx hardhat node

# 3. 전체 테스트 (다른 터미널)
go test ./...

# 4. 특정 패키지만
go test -v ./pkg/agent/did
```

---

**작성자**: Claude Code
**검증 완료**: 2025-10-24
**커밋 권장**: ✅ 모든 코드 수정사항 커밋 준비 완료
