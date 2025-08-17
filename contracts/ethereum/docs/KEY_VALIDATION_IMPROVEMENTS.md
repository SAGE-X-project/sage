# 공개키 검증 개선 가이드

## 📋 개선 사항 요약

### 1. **5단계 검증 프로세스**

#### 현재 (SageRegistry.sol)
```solidity
// 단순히 길이만 체크
modifier validPublicKey(bytes memory publicKey) {
    require(publicKey.length >= 32 && publicKey.length <= 65);
}
```

#### 개선 (SageRegistryV2.sol)
```solidity
function _validatePublicKey(bytes calldata publicKey, bytes calldata signature) {
    // 1️⃣ 길이 검증 (32-65 bytes)
    // 2️⃣ 형식 검증 (0x04 for uncompressed, 0x02/0x03 for compressed)
    // 3️⃣ 제로키 방지 (모든 바이트가 0인 경우 거부)
    // 4️⃣ 소유권 증명 (서명으로 개인키 소유 증명)
    // 5️⃣ 검증 데이터 저장 (추후 검증용)
}
```

### 2. **주요 개선점**

#### ✅ **형식 검증**
```solidity
if (publicKey.length == 65) {
    // 비압축 형식: 0x04로 시작해야 함
    require(publicKey[0] == 0x04, "Invalid uncompressed key");
} else if (publicKey.length == 33) {
    // 압축 형식: 0x02 또는 0x03으로 시작
    require(publicKey[0] == 0x02 || publicKey[0] == 0x03, "Invalid compressed key");
}
```

#### ✅ **제로키 방지**
```solidity
// 모든 바이트가 0인 무효한 키 거부
bytes32 keyHash = keccak256(publicKey);
require(keyHash != keccak256(new bytes(publicKey.length)), "Invalid zero key");
```

#### ✅ **소유권 증명 (핵심!)**
```solidity
// 챌린지 메시지 생성
bytes32 challenge = keccak256(abi.encodePacked(
    "SAGE Key Registration:\n",
    "Chain ID: ", block.chainid,
    "Contract: ", address(this),
    "Owner: ", msg.sender,
    "Key Hash: ", keyHash,
    "Timestamp: ", block.timestamp
));

// 서명 검증으로 개인키 소유 증명
address recovered = ecrecover(ethSignedHash, signature);
require(recovered == msg.sender, "Key ownership not proven");
```

### 3. **추가 보안 기능**

#### 🔐 **키 폐기 기능**
```solidity
function revokeKey(bytes calldata publicKey) external {
    // 손상된 키를 폐기하고 관련 에이전트 비활성화
}
```

#### 🔐 **키 유효성 확인**
```solidity
function isKeyValid(bytes calldata publicKey) external view returns (bool) {
    // 키가 유효하고 폐기되지 않았는지 확인
}
```

## 🚀 마이그레이션 가이드

### Option 1: 완전 교체 (권장)
```bash
# 새 컨트랙트 배포
1. SageRegistryV2.sol 배포
2. 기존 데이터 마이그레이션 (필요시)
3. 프론트엔드/백엔드 업데이트
```

### Option 2: 부분 개선
기존 SageRegistry.sol에 최소한 다음 개선사항 적용:

```solidity
// 최소 개선안
function registerAgent(...) {
    // 1. 형식 체크 추가
    if (publicKey.length == 65) {
        require(publicKey[0] == 0x04, "Bad format");
    }
    
    // 2. 제로키 체크
    require(keccak256(publicKey) != keccak256(new bytes(publicKey.length)), "Zero key");
    
    // 3. 서명으로 소유권 증명 (중요!)
    bytes32 proof = keccak256(abi.encodePacked("I own this key", publicKey, msg.sender));
    address signer = ecrecover(proof, signature);
    require(signer == msg.sender, "Not key owner");
    
    // ... 기존 로직
}
```

## 📊 비교표

| 검증 항목 | 현재 (v1) | 개선 (v2) | 중요도 |
|---------|-----------|-----------|--------|
| 길이 검증 | ✅ | ✅ | ⭐⭐ |
| 형식 검증 | ❌ | ✅ | ⭐⭐⭐ |
| 제로키 방지 | ❌ | ✅ | ⭐⭐⭐ |
| 소유권 증명 | ❌ | ✅ | ⭐⭐⭐⭐⭐ |
| 키 폐기 | ❌ | ✅ | ⭐⭐⭐⭐ |
| 가스 비용 | ~500 | ~3,500 | - |

## 💡 구현 시 주의사항

### 1. **서명 생성 (클라이언트)**
```javascript
// Web3.js 예제
async function registerWithKeyProof(publicKey) {
    const challenge = web3.utils.soliditySha3(
        "SAGE Key Registration:\n",
        "Chain ID: ", chainId, "\n",
        "Contract: ", contractAddress, "\n",
        "Owner: ", account, "\n",
        "Key Hash: ", web3.utils.keccak256(publicKey), "\n",
        "Timestamp: ", Math.floor(Date.now() / 1000)
    );
    
    const signature = await web3.eth.sign(challenge, account);
    
    return contract.methods.registerAgent(
        did, name, description, endpoint,
        publicKey, capabilities, signature
    ).send({ from: account });
}
```

### 2. **Ed25519 지원**
- 온체인 검증 불가 (Ethereum 제약)
- 오프체인 검증 후 오라클 활용
- 또는 ZK Proof 사용 (고급)

### 3. **가스 최적화**
- 챌린지 메시지 단순화 가능
- 검증 데이터 최소화
- 이벤트로 오프체인 인덱싱

## 🎯 결론

**최소 권장사항**:
1. ✅ 형식 검증 (0x04, 0x02, 0x03)
2. ✅ 제로키 체크
3. ✅ 서명으로 소유권 증명

**추가 권장사항**:
4. ✅ 키 폐기 기능
5. ✅ 유효성 상태 추적

이 정도면 **실용적이면서도 안전한** 공개키 검증이 가능합니다!

## 📚 참고자료
- [EIP-191: Signed Data Standard](https://eips.ethereum.org/EIPS/eip-191)
- [Secp256k1 Key Format](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition)
- [ECDSA Signature Verification](https://docs.openzeppelin.com/contracts/4.x/api/utils#ECDSA)