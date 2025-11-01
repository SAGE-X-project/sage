# SAGE Agent Initialization Examples

이 디렉토리는 SAGE 에이전트가 프로그램 시작 시 키를 자동으로 관리하는 방법을 보여주는 예시들입니다.

##  예시 파일들

### 1. `main.go` - 완전한 에이전트 초기화
- 완전한 키 관리 시스템
- 블록체인 등록 기능
- 상세한 에러 처리
- 프로덕션 준비 코드

### 2. `../simple-agent-init/main.go` - 간단한 에이전트 초기화
- 간단하고 이해하기 쉬운 코드
- 기본적인 키 관리
- 메시지 서명 데모
- 학습용 예시

##  빠른 시작

### 간단한 예시 실행

```bash
cd simple-agent-init
go run main.go
```

### 완전한 예시 실행

```bash
# 기본 실행 (키만 생성/로드)
go run main.go

# 블록체인 등록 포함
REGISTER_ON_CHAIN=true go run main.go
```

##  키 관리 기능

### 자동 키 감지
- 프로그램 시작 시 키 파일 존재 여부 확인
- 기존 키가 있으면 자동으로 로드
- 키가 없으면 새로 생성

### 지원하는 키 타입
- **ECDSA (secp256k1)**: 이더리움 호환성
- **Ed25519**: 고성능 서명
- **X25519**: 키 교환 및 암호화

### 키 파일 저장
```
keys/
├── ecdsa.key      # ECDSA 개인키 (PEM 형식)
├── ed25519.key    # Ed25519 개인키 (바이너리)
└── x25519.key     # X25519 개인키 (바이너리)
```

##  사용법

### 1. 에이전트 생성

```go
// 간단한 방법
agent, err := NewSimpleAgent("my-agent", "./keys")
if err != nil {
    log.Fatal(err)
}

// 완전한 방법
agent, err := NewAgent("my-agent", "./keys")
if err != nil {
    log.Fatal(err)
}
```

### 2. 키 사용

```go
// 메시지 서명
message := []byte("Hello, World!")
signature := agent.SignMessage(message)

// 서명 검증
valid := agent.VerifyMessage(message, signature)

// 공개키 가져오기
ecdsaPub, ed25519Pub, x25519Pub, err := agent.GetPublicKeys()
```

### 3. 블록체인 등록

```go
// 에이전트를 블록체인에 등록
err := agent.RegisterOnBlockchain(
    "0x...",                    // 컨트랙트 주소
    "http://localhost:8545",    // RPC URL
    "0x...",                    // 개인키
)
```

##  환경 변수

### 필수 변수 (블록체인 등록 시)
```bash
export REGISTRY_ADDRESS="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
export RPC_URL="http://localhost:8545"
export PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
```

### 선택적 변수
```bash
export REGISTER_ON_CHAIN="true"  # 블록체인 등록 활성화
```

##  아키텍처

### 키 관리 플로우

```
프로그램 시작
     ↓
키 디렉토리 확인
     ↓
키 파일 존재? ──Yes──→ 기존 키 로드
     ↓ No
새 키 생성
     ↓
키 파일 저장
     ↓
에이전트 초기화 완료
```

### 에이전트 구조

```go
type Agent struct {
    Name        string              // 에이전트 이름
    DID         string              // 분산 식별자
    KeyDir      string              // 키 저장 디렉토리
    ECDSAKey    *ecdsa.PrivateKey   // ECDSA 개인키
    Ed25519Key  ed25519.PrivateKey  // Ed25519 개인키
    X25519Key   []byte              // X25519 개인키
    IsInitialized bool              // 초기화 상태
}
```

##  보안 고려사항

### 키 파일 권한
- 모든 키 파일은 `0600` 권한으로 저장
- 키 디렉토리는 `0700` 권한으로 생성

### 키 저장 위치
- 프로덕션에서는 안전한 위치 사용
- 예: `/etc/sage/keys/`, `~/.sage/keys/`

### 키 백업
- 키 파일을 안전하게 백업
- 키 손실 시 복구 불가능

##  테스트

### 첫 실행 (키 생성)
```bash
go run main.go
# 새로운 키들이 생성됩니다
```

### 두 번째 실행 (키 로드)
```bash
go run main.go
# 기존 키들이 로드됩니다
```

### 키 파일 삭제 후 실행
```bash
rm -rf keys/
go run main.go
# 새로운 키들이 다시 생성됩니다
```

##  추가 리소스

- [SAGE 메인 문서](../../../README.md)
- [DID 관리 가이드](../../../docs/did/)
- [암호화 가이드](../../../docs/crypto/)
- [A2A 통합 예시](../a2a-integration/)

##  기여하기

버그 리포트나 기능 요청은 [GitHub Issues](https://github.com/sage-x-project/sage/issues)에 제출해주세요.

##  라이선스

LGPL-3.0 - 자세한 내용은 [LICENSE](../../../LICENSE)를 참조하세요.
