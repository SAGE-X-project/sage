# SageRegistry v1 → v2 마이그레이션 가이드

##  개요

SageRegistryV2는 향상된 공개키 검증 기능을 제공하는 업그레이드 버전입니다. 이 가이드는 v1에서 v2로의 원활한 마이그레이션을 도와드립니다.

##  주요 개선사항

### 1. **공개키 검증 강화**
-  **형식 검증**: 0x04 (비압축), 0x02/0x03 (압축) 형식 확인
-  **제로키 방지**: 모든 바이트가 0인 무효한 키 거부
-  **소유권 증명**: 서명을 통한 개인키 소유 증명
-  **키 폐기 기능**: 손상된 키 폐기 및 관련 에이전트 비활성화
-  **Ed25519 거부**: 온체인 검증 불가능한 Ed25519 키 명시적 거부

### 2. **보안 개선**
- 챌린지-응답 방식의 키 소유권 증명
- 폐기된 키 재사용 방지
- 키 유효성 상태 추적

### 3. **가스 최적화**
- 검증 로직 최적화로 가스 사용량 관리
- 등록: ~620K gas
- 업데이트: ~50K gas
- 폐기: ~30K gas

##  마이그레이션 방법

### Option 1: 완전 교체 (권장)

#### 1단계: 새 컨트랙트 배포
```bash
# 배포 스크립트 실행
./deploy-v2.sh

# 또는 직접 실행
npx hardhat run scripts/deploy-v2.js --network kairos
```

#### 2단계: 기존 데이터 마이그레이션 (선택사항)
```javascript
// 마이그레이션 스크립트 예제
async function migrateAgents(oldRegistry, newRegistry) {
  const agents = await oldRegistry.getAllAgents();
  
  for (const agent of agents) {
    // 새로운 서명 생성 필요
    const signature = await createRegistrationSignature(
      agent.owner,
      agent.publicKey
    );
    
    await newRegistry.registerAgent(
      agent.did,
      agent.name,
      agent.description,
      agent.endpoint,
      agent.publicKey,
      agent.capabilities,
      signature // 새로운 요구사항
    );
  }
}
```

#### 3단계: 프론트엔드 업데이트
```javascript
// 서명 생성 함수 추가
async function createRegistrationSignature(signer, publicKey) {
  const contractAddress = registry.address;
  const chainId = await signer.getChainId();
  const keyHash = ethers.keccak256(publicKey);
  
  const packedData = ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    [
      "SAGE Key Registration:",
      chainId,
      contractAddress,
      signer.address,
      keyHash
    ]
  );
  
  const challenge = ethers.keccak256(packedData);
  return await signer.signMessage(ethers.getBytes(challenge));
}

// 에이전트 등록 호출 업데이트
const signature = await createRegistrationSignature(signer, publicKey);
await registry.registerAgent(
  did,
  name,
  description,
  endpoint,
  publicKey,
  capabilities,
  signature // 추가된 매개변수
);
```

### Option 2: 점진적 마이그레이션

기존 v1을 유지하면서 새 등록만 v2로 진행:

```javascript
// 듀얼 레지스트리 관리
class DualRegistry {
  constructor(v1Address, v2Address) {
    this.v1 = new ethers.Contract(v1Address, V1_ABI, provider);
    this.v2 = new ethers.Contract(v2Address, V2_ABI, provider);
  }
  
  async getAgent(did) {
    // v2에서 먼저 검색
    try {
      return await this.v2.getAgentByDID(did);
    } catch {
      // v1에서 검색
      return await this.v1.getAgentByDID(did);
    }
  }
  
  async registerAgent(...params) {
    // 새 등록은 v2로만
    return await this.v2.registerAgent(...params);
  }
}
```

##  코드 변경사항

### 컨트랙트 인터페이스 변경

#### v1 (이전)
```solidity
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature
) external returns (bytes32);
```

#### v2 (현재)
```solidity
function registerAgent(
    string calldata did,
    string calldata name,
    string calldata description,
    string calldata endpoint,
    bytes calldata publicKey,
    string calldata capabilities,
    bytes calldata signature  // 서명 형식 변경
) external returns (bytes32);

// 새로운 함수들
function isKeyValid(bytes calldata publicKey) external view returns (bool);
function revokeKey(bytes calldata publicKey) external;
```

### 서명 생성 변경

#### v1 (이전)
```javascript
// 단순 메시지 서명
const messageHash = ethers.keccak256(
  ethers.AbiCoder.defaultAbiCoder().encode(
    ["string", "string", "..."],
    [did, name, ...]
  )
);
const signature = await signer.signMessage(messageHash);
```

#### v2 (현재)
```javascript
// 챌린지-응답 방식
const challenge = createChallenge(publicKey, contractAddress, chainId);
const signature = await signer.signMessage(challenge);
```

##  주의사항

### 1. **Breaking Changes**
- 서명 생성 방식이 변경되어 기존 서명은 호환되지 않음
- Ed25519 키는 명시적으로 거부됨 (32바이트)
- 제로키 검증이 더 엄격해짐

### 2. **가스 사용량 증가**
- v1: ~400K gas
- v2: ~620K gas
- 보안 향상으로 인한 불가피한 증가

### 3. **새로운 요구사항**
- 등록 시 개인키 소유 증명 필수
- 공개키 형식 준수 필수 (0x04, 0x02, 0x03)

##  테스트

### 마이그레이션 전 테스트
```bash
# v2 컨트랙트 테스트
npx hardhat test test/SageRegistryV2.test.js

# 가스 사용량 확인
npx hardhat test test/SageRegistryV2.test.js --grep "Gas"
```

### 통합 테스트
```javascript
// 통합 테스트 예제
describe("Migration Test", function() {
  it("Should migrate agent from v1 to v2", async function() {
    // 1. v1에서 에이전트 정보 가져오기
    const v1Agent = await v1Registry.getAgent(agentId);
    
    // 2. v2용 서명 생성
    const signature = await createRegistrationSignature(
      signer,
      v1Agent.publicKey
    );
    
    // 3. v2에 등록
    await v2Registry.registerAgent(
      v1Agent.did,
      v1Agent.name,
      v1Agent.description,
      v1Agent.endpoint,
      v1Agent.publicKey,
      v1Agent.capabilities,
      signature
    );
    
    // 4. 검증
    const v2Agent = await v2Registry.getAgentByDID(v1Agent.did);
    expect(v2Agent.name).to.equal(v1Agent.name);
  });
});
```

##  성능 비교

| 항목 | v1 | v2 | 변화 |
|------|-----|-----|------|
| 등록 가스 | ~400K | ~620K | +55% |
| 업데이트 가스 | ~45K | ~50K | +11% |
| 보안 수준 | 기본 | 향상 | ⬆️⬆️⬆️ |
| 키 검증 | 길이만 | 5단계 | ⬆️⬆️⬆️⬆️ |
| 키 폐기 |  |  | 새기능 |

## 🆘 문제 해결

### 문제: "Key ownership not proven" 에러
**해결**: 서명 생성 시 올바른 챌린지 메시지 형식 사용
```javascript
const challenge = keccak256(abi.encodePacked(
  "SAGE Key Registration:",
  chainId,
  contractAddress,
  signerAddress,
  keyHash
));
```

### 문제: "Invalid uncompressed key format" 에러
**해결**: 공개키가 올바른 형식인지 확인
- 비압축: 0x04 + 64바이트
- 압축: 0x02 또는 0x03 + 32바이트

### 문제: "Ed25519 not supported on-chain" 에러
**해결**: secp256k1 키 사용 (Ethereum 표준)

## 📚 참고 자료

- [SageRegistryV2.sol](contracts/SageRegistryV2.sol)
- [테스트 코드](test/SageRegistryV2.test.js)
- [배포 스크립트](scripts/deploy-v2.js)
- [공개키 검증 개선 문서](KEY_VALIDATION_IMPROVEMENTS.md)

## 🤝 지원

마이그레이션 중 문제가 발생하면:
1. 테스트넷에서 먼저 테스트
2. 이슈 트래커에 문제 보고
3. 커뮤니티 채널에서 도움 요청

---

**⚡ 중요**: 메인넷 배포 전 반드시 테스트넷에서 충분한 테스트를 진행하세요!