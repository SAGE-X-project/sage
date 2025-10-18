# Phase 7: Sepolia 테스트넷 배포 계획

**날짜**: 2025-10-07
**대상 네트워크**: Ethereum Sepolia Testnet
**상태**: 📋 배포 준비 중
**우선순위**: HIGH (보안 개선사항 적용 후 재배포 필요)

---

## 요약

Phase 1-6에서 완료한 모든 보안 개선사항이 반영된 컨트랙트를 Sepolia 테스트넷에 배포합니다. 이전 배포(2025-10-06)는 보안 개선 전 버전이므로 최신 버전으로 재배포가 필요합니다.

**주요 변경사항**:
- ✅ 3개 CRITICAL 이슈 해결
- ✅ 8개 HIGH 이슈 해결
- ✅ 4개 MEDIUM 이슈 해결 (핵심)
- ✅ 157개 테스트 통과

---

## 배포 대상 컨트랙트

### 1. SAGE Core Contracts (최신 보안 버전)

| 컨트랙트 | 버전 | 주요 개선사항 |
|---------|------|-------------|
| **SageRegistryV2** | Security Enhanced | ReentrancyGuard, Ownable2Step, Pausable, Hook gas limit, DID validation |
| **SageRegistryV3** | NEW | Commit-reveal 프론트러닝 방지 |
| **SageVerificationHook** | Updated | 향상된 검증 로직 |

### 2. ERC-8004 Contracts (최신 보안 버전)

| 컨트랙트 | 버전 | 주요 개선사항 |
|---------|------|-------------|
| **ERC8004IdentityRegistry** | Adapter | O(1) deactivation, standalone 독립성 |
| **ERC8004ReputationRegistry** | Legacy | 기본 기능 |
| **ERC8004ReputationRegistryV2** | NEW | Commit-reveal, 데드라인 검증 (1시간-30일) |
| **ERC8004ValidationRegistry** | Security Enhanced | ReentrancyGuard, Pull Payment, 만료 처리, 평판 기반 스테이킹 |

### 3. ERC-8004 Standalone Contracts (완전 독립)

| 컨트랙트 | 설명 |
|---------|------|
| **ERC8004IdentityRegistry (Standalone)** | Sage 의존성 없는 완전 독립 구현 |
| **ERC8004ReputationRegistry (Standalone)** | 독립적인 평판 시스템 |
| **ERC8004ValidationRegistry (Standalone)** | 독립적인 검증 시스템 |

### 4. Governance Contracts (선택적)

| 컨트랙트 | 용도 |
|---------|------|
| **SimpleMultiSig** | 다중 서명 지갑 |
| **TEEKeyRegistry** | TEE 키 거버넌스 |
| **TimelockController** | 시간 지연 거버넌스 |

---

## 배포 전 체크리스트

### ✅ 코드 준비 상태

- [x] 로컬 테스트 통과 (157/157)
- [x] 보안 감사 검증 완료
- [x] 모든 CRITICAL 이슈 해결
- [x] 모든 HIGH 이슈 해결
- [x] 컨트랙트 컴파일 확인
- [x] 배포 스크립트 검증

### 🔧 환경 설정 확인

- [ ] Sepolia RPC 엔드포인트 설정
- [ ] 배포 계정 개인키 설정 (.env)
- [ ] 충분한 Sepolia ETH 보유 확인 (~0.5 ETH 권장)
- [ ] Etherscan API 키 설정 (컨트랙트 검증용)
- [ ] hardhat.config.js 네트워크 설정 확인

### 📝 배포 후 작업 준비

- [ ] 컨트랙트 검증 스크립트
- [ ] 초기 설정 스크립트 (Hook 연결 등)
- [ ] 배포 결과 문서화 템플릿
- [ ] 모니터링 도구 준비

---

## 배포 순서

### Phase 7.1: Core Infrastructure 배포

**순서 1: SageRegistryV2 (보안 강화 버전)**
```bash
npx hardhat run scripts/deploy-sage-v2-security.js --network sepolia
```
- ReentrancyGuard 적용
- Ownable2Step 적용
- Pausable 적용
- Hook 가스 제한 (50,000)
- DID 검증 강화

**순서 2: SageRegistryV3 (프론트러닝 방지)**
```bash
npx hardhat run scripts/deploy-sage-v3.js --network sepolia
```
- Commit-reveal 패턴
- 타이밍 검증 (1분-1시간)

**순서 3: SageVerificationHook**
```bash
# SageRegistryV2 주소 필요
npx hardhat run scripts/deploy-verification-hook.js --network sepolia
```

**순서 4: Hook 설정**
```bash
# SageRegistryV2에 Hook 연결
# setBeforeRegisterHook()
# setAfterRegisterHook()
```

### Phase 7.2: ERC-8004 Adapter Contracts 배포

**순서 5: ERC8004IdentityRegistry (Adapter)**
```bash
npx hardhat run scripts/deploy-erc8004-identity.js --network sepolia
```
- SageRegistryV2 주소 필요
- O(1) deactivation 적용

**순서 6: ERC8004ReputationRegistryV2**
```bash
npx hardhat run scripts/deploy-erc8004-reputation-v2.js --network sepolia
```
- IdentityRegistry 주소 필요
- Commit-reveal 적용
- 데드라인 검증 (1시간-30일)

**순서 7: ERC8004ValidationRegistry**
```bash
npx hardhat run scripts/deploy-erc8004-validation.js --network sepolia
```
- IdentityRegistry, ReputationRegistry 주소 필요
- ReentrancyGuard 적용
- Pull Payment 패턴
- 만료 처리 함수
- 평판 기반 스테이킹

**순서 8: ValidationRegistry 연결**
```bash
# ReputationRegistry에 ValidationRegistry 설정
# setValidationRegistry()
```

### Phase 7.3: ERC-8004 Standalone Contracts 배포 (선택)

**순서 9-11: Standalone Contracts**
```bash
npx hardhat run scripts/deploy-erc8004-standalone.js --network sepolia
```
- 완전 독립적인 ERC-8004 구현
- Sage 의존성 없음
- 표준 준수 검증용

### Phase 7.4: Governance Contracts 배포 (선택)

**순서 12-14: Governance Infrastructure**
```bash
npx hardhat run scripts/deploy-governance.js --network sepolia
```
- SimpleMultiSig
- TEEKeyRegistry
- TimelockController

---

## 예상 배포 비용

### Gas 추정

| 컨트랙트 | 예상 Gas | 비용 (@10 gwei) |
|---------|---------|----------------|
| SageRegistryV2 | ~3,500,000 | ~0.035 ETH |
| SageRegistryV3 | ~3,800,000 | ~0.038 ETH |
| SageVerificationHook | ~500,000 | ~0.005 ETH |
| ERC8004IdentityRegistry | ~1,200,000 | ~0.012 ETH |
| ERC8004ReputationRegistryV2 | ~2,500,000 | ~0.025 ETH |
| ERC8004ValidationRegistry | ~4,000,000 | ~0.040 ETH |
| Hook 설정 (4 트랜잭션) | ~200,000 | ~0.002 ETH |
| **합계 (Core + ERC8004)** | **~15,700,000** | **~0.157 ETH** |

**추가 (Standalone + Governance)**:
| 카테고리 | 예상 Gas | 비용 (@10 gwei) |
|---------|---------|----------------|
| Standalone (3개) | ~4,000,000 | ~0.040 ETH |
| Governance (3개) | ~3,000,000 | ~0.030 ETH |
| **총 합계 (모든 컨트랙트)** | **~22,700,000** | **~0.227 ETH** |

**권장 잔액**: 최소 0.3 ETH (여유분 포함)

---

## 배포 스크립트 준비

### 필요한 스크립트

1. **deploy-sage-v2-security.js** - SageRegistryV2 (보안 강화)
2. **deploy-sage-v3.js** - SageRegistryV3 (commit-reveal)
3. **deploy-verification-hook.js** - SageVerificationHook
4. **deploy-erc8004-adapter.js** - ERC8004 Adapter 3종
5. **deploy-erc8004-standalone.js** - ERC8004 Standalone 3종
6. **deploy-governance.js** - Governance 3종
7. **configure-contracts.js** - 초기 설정 (Hook 연결 등)

### 통합 배포 스크립트 (권장)

```javascript
// scripts/deploy-sepolia-phase7.js
async function main() {
  console.log("🚀 Phase 7: Sepolia Deployment Starting...\n");

  // Step 1: Deploy Core
  const sageV2 = await deploySageRegistryV2();
  const sageV3 = await deploySageRegistryV3();
  const hook = await deployVerificationHook(sageV2.address);

  // Step 2: Configure Hooks
  await configureHooks(sageV2, hook);

  // Step 3: Deploy ERC8004 Adapters
  const identity = await deployERC8004Identity(sageV2.address);
  const reputationV2 = await deployERC8004ReputationV2(identity.address);
  const validation = await deployERC8004Validation(identity.address, reputationV2.address);

  // Step 4: Link Contracts
  await linkContracts(reputationV2, validation);

  // Step 5: Verify on Etherscan
  await verifyContracts([sageV2, sageV3, hook, identity, reputationV2, validation]);

  // Step 6: Generate Report
  await generateDeploymentReport();

  console.log("✅ Phase 7 Deployment Complete!\n");
}
```

---

## 배포 후 검증

### 1. 컨트랙트 검증 (Etherscan)

모든 컨트랙트를 Etherscan에 검증하여 소스코드 공개:
```bash
npx hardhat verify --network sepolia <CONTRACT_ADDRESS> <CONSTRUCTOR_ARGS>
```

### 2. 기능 테스트

**SageRegistryV2 테스트**:
- [ ] 에이전트 등록 가능
- [ ] Hook 동작 확인
- [ ] Pause/Unpause 작동
- [ ] 소유권 이전 (2단계) 작동

**SageRegistryV3 테스트**:
- [ ] Commit 등록
- [ ] Reveal 등록 (타이밍 검증)
- [ ] 프론트러닝 방지 확인

**ERC8004ValidationRegistry 테스트**:
- [ ] 검증 요청 생성
- [ ] 스테이크 검증 제출
- [ ] 합의 도달 확인
- [ ] 보상 인출 (Pull Payment)
- [ ] 만료 처리 동작

### 3. 보안 검증

- [ ] ReentrancyGuard 작동 확인
- [ ] Pull Payment 패턴 작동
- [ ] Hook 가스 제한 작동
- [ ] Ownable2Step 작동
- [ ] Pausable 작동

### 4. 통합 테스트

- [ ] 전체 에이전트 라이프사이클
- [ ] 검증 요청 → 응답 → 합의 → 보상
- [ ] Hook 통합 동작
- [ ] 이벤트 발행 확인

---

## 배포 후 모니터링

### 1. 모니터링 대상

**이벤트 모니터링**:
- `AgentRegistered` - 에이전트 등록
- `ValidationCompleted` - 검증 완료
- `WithdrawalProcessed` - 보상 인출
- `HookFailed` - Hook 실패
- `Paused`/`Unpaused` - 긴급 정지
- `OwnershipTransferStarted` - 소유권 이전 시작

**컨트랙트 상태 모니터링**:
- 총 등록 에이전트 수
- 활성 검증 요청 수
- 평균 가스 비용
- 실패율

### 2. 알림 설정

**Etherscan Alerts**:
- 컨트랙트 트랜잭션 알림
- 대량 가스 소비 알림

**Discord/Telegram Webhook**:
- 중요 이벤트 실시간 알림
- 에러 발생 즉시 알림

---

## 위험 요소 및 대응

### 잠재적 문제

| 위험 | 확률 | 영향 | 대응 방안 |
|-----|------|------|----------|
| 배포 실패 | 낮음 | 중간 | 로컬에서 사전 테스트 완료 |
| 가스 부족 | 중간 | 낮음 | 충분한 ETH 준비 (0.3+) |
| 검증 실패 | 낮음 | 낮음 | Etherscan API 키 사전 확인 |
| Hook 설정 오류 | 중간 | 높음 | 설정 스크립트 사전 검증 |
| 통합 테스트 실패 | 중간 | 높음 | 단계별 테스트 수행 |

### 긴급 대응 계획

**문제 발생 시**:
1. 즉시 Pausable 활성화 (필요 시)
2. 문제 분석 및 로그 수집
3. 필요시 새 버전 배포
4. 커뮤니티 공지

---

## 배포 타임라인

### 예상 소요 시간

| 단계 | 소요 시간 | 누적 시간 |
|-----|----------|----------|
| 환경 설정 및 확인 | 30분 | 30분 |
| Core 배포 (4개) | 1시간 | 1시간 30분 |
| ERC8004 Adapter 배포 (3개) | 1시간 | 2시간 30분 |
| Hook 설정 및 연결 | 30분 | 3시간 |
| Etherscan 검증 | 1시간 | 4시간 |
| 기능 테스트 | 1시간 | 5시간 |
| 통합 테스트 | 1시간 | 6시간 |
| 문서화 | 30분 | 6시간 30분 |

**총 예상 시간**: **6-7시간**

### 권장 일정

**Day 1** (4시간):
- 환경 설정
- Core 배포
- ERC8004 Adapter 배포
- 초기 검증

**Day 2** (3시간):
- 기능 테스트
- 통합 테스트
- 문서화
- 모니터링 설정

---

## 성공 기준

### 필수 조건 (Phase 7 완료)

- [ ] 모든 Core 컨트랙트 배포 완료
- [ ] 모든 ERC8004 Adapter 배포 완료
- [ ] Etherscan 검증 완료
- [ ] 기본 기능 테스트 통과
- [ ] Hook 연결 정상 작동
- [ ] 배포 문서 작성 완료

### 선택 조건 (Phase 8 준비)

- [ ] Standalone 컨트랙트 배포
- [ ] Governance 컨트랙트 배포
- [ ] 고급 통합 테스트 완료
- [ ] 모니터링 대시보드 구축
- [ ] 커뮤니티 테스트 가이드 작성

---

## 다음 단계 (Phase 8)

Phase 7 완료 후:

1. **커뮤니티 테스팅** (2주)
   - 테스트넷 공개
   - 피드백 수집
   - 버그 수정

2. **외부 감사 준비**
   - 감사 기관 선정
   - 감사 자료 준비
   - 감사 비용 확보

3. **메인넷 배포 준비**
   - 최종 보안 검토
   - 거버넌스 설정
   - 배포 계획 수립

---

## 체크리스트 요약

### 배포 전
- [ ] 로컬 테스트 157/157 통과 확인
- [ ] Sepolia RPC 설정
- [ ] 배포 계정 개인키 설정
- [ ] 0.3+ ETH 준비
- [ ] Etherscan API 키 설정
- [ ] 배포 스크립트 준비

### 배포 중
- [ ] Core 컨트랙트 배포
- [ ] ERC8004 Adapter 배포
- [ ] Hook 설정
- [ ] 컨트랙트 연결

### 배포 후
- [ ] Etherscan 검증
- [ ] 기능 테스트
- [ ] 통합 테스트
- [ ] 모니터링 설정
- [ ] 문서 업데이트

---

**문서 버전**: 1.0
**작성일**: 2025-10-07
**상태**: 📋 배포 준비 중
**다음 단계**: 환경 설정 및 배포 스크립트 실행
