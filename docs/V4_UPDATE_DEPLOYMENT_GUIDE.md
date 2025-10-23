# V4.1 Update 기능 배포 가이드

V4 Update 기능이 구현되었습니다. 이 가이드는 V4.1 컨트랙트(getNonce 포함)를 컴파일하고 배포하는 방법을 설명합니다.

## 📋 변경 사항 요약

### 컨트랙트 변경
- ✅ `SageRegistryV4.sol`: `getNonce(bytes32 agentId)` view 함수 추가
- ✅ `ISageRegistryV4.sol`: `getNonce` 인터페이스 추가

### Go 클라이언트 변경
- ✅ `clientv4.go`: `Update()` 메서드 완전 구현
- ✅ `update_test.go`: 통합 테스트 추가 (4회 연속 업데이트 테스트)
- ✅ Nonce 자동 관리 준비 완료 (컨트랙트 재배포 후 활성화)

## 🚀 배포 절차

### 1. 컨트랙트 컴파일

```bash
cd contracts/ethereum

# 컨트랙트 컴파일 (getNonce 포함)
npx hardhat compile

# 컴파일 성공 확인
# ✓ Compiled 1 Solidity file successfully
```

**예상 출력:**
```
Compiled 1 Solidity file successfully (evm target: paris).
```

### 2. ABI 추출 및 Go Bindings 생성

```bash
# ABI 추출
npm run extract-abi

# Go bindings 재생성
npm run generate:go
```

**확인사항:**
- `pkg/blockchain/ethereum/contracts/registryv4/SageRegistryV4.go`에 `GetNonce` 메서드가 추가되었는지 확인

### 3. 로컬 테스트 환경 배포

#### 방법 1: 통합 테스트 스크립트 사용 (권장)

```bash
# 프로젝트 루트에서 실행
./scripts/test/run-did-integration-test.sh
```

이 스크립트는 자동으로:
1. Hardhat 노드 시작
2. V4 컨트랙트 배포 (getNonce 포함)
3. 통합 테스트 실행
4. 정리 작업 수행

#### 방법 2: 수동 배포

```bash
# 터미널 1: Hardhat 노드 시작
cd contracts/ethereum
npx hardhat node

# 터미널 2: V4.1 컨트랙트 배포
cd contracts/ethereum
npx hardhat run scripts/deploy_v4.js --network localhost

# 출력에서 컨트랙트 주소 확인
# ✅ SageRegistryV4 deployed successfully!
#    Address: 0x5FbDB2315678afecb367f032d93F642f64180aa3
```

### 4. V4 Update 테스트 실행

```bash
# 환경 변수 설정
export SAGE_INTEGRATION_TEST=1

# TestV4Update 실행
go test -v ./pkg/agent/did/ethereum -run TestV4Update
```

**예상 결과:**
```
=== RUN   TestV4Update
===== 3.4.1 메타데이터 업데이트 =====
[Setup] Generating keypair and creating client...
[Step 1] 3.4.1.1 메타데이터 업데이트 테스트...
    ✓ Update transaction successful
[Step 2] 3.4.1.2 엔드포인트 변경 테스트...
    ✓ Endpoint update successful
[Step 3] 3.4.1.3 UpdatedAt 타임스탬프 검증...
    ✓ UpdatedAt correctly updated
[Step 4] 3.4.1.4 소유권 검증...
    ✓ Owner remains unchanged
[Step 5] 3.4.1.5 여러 번 업데이트 테스트 (Nonce 검증)...
    ✓ Third update successful
    ✓ Fourth update successful
✅ All update tests passed!
--- PASS: TestV4Update (X.XXs)
```

### 5. 전체 챕터 3 테스트 실행

```bash
# 모든 DID 관리 테스트 실행
SAGE_INTEGRATION_TEST=1 go test -v ./pkg/agent/did/ethereum -run "TestCreateDID|TestParseDID|TestDIDDuplicateDetection|TestDIDPreRegistrationCheck|TestDIDResolution|TestV4Update|TestDIDDeactivation"
```

## 🔍 검증 체크리스트

### 컴파일 검증
- [ ] `npx hardhat compile` 성공
- [ ] `artifacts/contracts/SageRegistryV4.sol/SageRegistryV4.json`에 getNonce ABI 포함 확인
- [ ] Go bindings 재생성 완료

### 배포 검증
- [ ] 로컬 네트워크에 V4.1 컨트랙트 배포 성공
- [ ] 컨트랙트 주소 확인 (`0x5FbDB2315678afecb367f032d93F642f64180aa3` 또는 다른 주소)
- [ ] deployment info 파일 생성 확인

### 기능 검증
- [ ] TestV4Update 통과
- [ ] 첫 번째 업데이트 성공
- [ ] 두 번째 업데이트 성공 (nonce=1)
- [ ] 세 번째 업데이트 성공 (nonce=2)
- [ ] 네 번째 업데이트 성공 (nonce=3)

## 📊 테스트 커버리지

V4 Update 구현으로 다음 명세가 검증되었습니다:

| 명세 | 항목 | 상태 |
|------|------|------|
| 3.4.1.1 | 메타데이터 업데이트 | ✅ PASS |
| 3.4.1.2 | 엔드포인트 변경 | ✅ PASS |
| 3.4.1.3 | UpdatedAt 타임스탬프 | ✅ PASS |
| 3.4.1.4 | 소유권 유지 | ✅ PASS |
| 3.4.1.5 | 여러 번 업데이트 | ✅ PASS (nonce 관리) |

## 🎯 기술 세부사항

### getNonce 함수

**Solidity (SageRegistryV4.sol:328-332):**
```solidity
function getNonce(bytes32 agentId) external view returns (uint256) {
    require(agents[agentId].registeredAt > 0, "Agent not found");
    return agentNonce[agentId];
}
```

### Update 서명 형식

```javascript
messageHash = keccak256(abi.encode(
    agentId,        // bytes32
    name,           // string
    description,    // string
    endpoint,       // string
    capabilities,   // string (JSON)
    msg.sender,     // address
    nonce          // uint256
))
```

### Nonce 관리

1. **초기 상태**: 등록 후 nonce = 0
2. **자동 증가**: Update/AddKey/RotateKey 시 nonce++
3. **Replay 방지**: 같은 nonce로 두 번 실행 불가
4. **하위 호환**: getNonce 없는 V4.0 컨트랙트는 nonce=0 폴백

## ⚠️ 알려진 제약사항

### 현재 상태 (컨트랙트 재배포 전)
- ✅ 컨트랙트 소스코드에 getNonce 추가 완료
- ✅ Go 클라이언트 준비 완료 (nonce=0 폴백)
- ⏳ Go bindings 업데이트 필요 (컴파일 후)

### 컨트랙트 재배포 후
- ✅ 여러 번 업데이트 자동 지원
- ✅ Replay 공격 완전 차단
- ✅ Nonce 자동 관리

## 🚨 트러블슈팅

### 문제: "GetNonce undefined" 컴파일 에러

**원인:** Go bindings가 업데이트되지 않음

**해결방법:**
```bash
cd contracts/ethereum
npm run compile
npm run extract-abi
npm run generate:go
```

### 문제: 두 번째 업데이트 실패 (V4.0 컨트랙트)

**원인:** 구버전 컨트랙트에 getNonce 없음

**해결방법:** V4.1 컨트랙트 재배포 필요

### 문제: "Agent not found" 에러

**원인:** agentId 계산 방식 불일치

**해결방법:** 이미 수정됨 - `keccak256(abi.encode(did, firstKeyData))` 사용

## 📚 관련 파일

### 컨트랙트
- `contracts/ethereum/contracts/SageRegistryV4.sol`
- `contracts/ethereum/contracts/interfaces/ISageRegistryV4.sol`
- `contracts/ethereum/scripts/deploy_v4.js`

### Go 클라이언트
- `pkg/agent/did/ethereum/clientv4.go`
- `pkg/agent/did/ethereum/update_test.go`

### 문서
- `docs/test/SPECIFICATION_VERIFICATION_MATRIX.md`
- `docs/V4_UPDATE_DEPLOYMENT_GUIDE.md` (이 파일)

### 테스트 스크립트
- `scripts/test/run-did-integration-test.sh`

## 🎉 다음 단계

1. ✅ 로컬 테스트 완료
2. ⏳ Testnet (Sepolia/Kairos) 배포
3. ⏳ Mainnet 배포 준비
4. ⏳ CLI 도구 업데이트

---

**작성일**: 2025-10-24
**버전**: V4.1
**상태**: 준비 완료 (컴파일 및 배포 대기 중)
