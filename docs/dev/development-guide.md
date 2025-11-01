# SAGE 개발 가이드

>  **PARTIALLY OUTDATED**: 일부 내용이 오래되었습니다.
>
> **주의사항**:
> -  Rust 설치 불필요 (Go 네이티브 구현 사용)
> -  디렉터리 구조 변경됨 (rust/, server/ 디렉터리 없음)
> -  일반적인 Go 개발 가이드는 여전히 유효
>
> 실제 빌드 명령: `make build`, `make test`

## 목차

- [1. 개발 환경 설정](#1-개발-환경-설정)
- [2. 프로젝트 구조](#2-프로젝트-구조)
- [3. 코드 작성 가이드](#3-코드-작성-가이드)
- [4. 빌드 및 테스트](#4-빌드-및-테스트)
- [5. 디버깅](#5-디버깅)
- [6. 기여 가이드](#6-기여-가이드)

## 1. 개발 환경 설정

### 1.1 필수 도구

```bash
# Go 설치 (1.19 이상)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Rust 설치
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
rustup default stable
rustup target add wasm32-unknown-unknown

# Node.js 설치 (TypeScript SDK용)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 개발 도구
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
cargo install wasm-pack
npm install -g typescript @types/node
```

### 1.2 프로젝트 클론 및 설정

```bash
# 저장소 클론
git clone https://github.com/sage-project/sage.git
cd sage

# Go 모듈 초기화
go mod download

# Rust 의존성 설치
cd rust/sage_crypto
cargo build

# pre-commit 훅 설정
cp scripts/pre-commit .git/hooks/
chmod +x .git/hooks/pre-commit
```

### 1.3 IDE 설정

**VS Code 추천 확장**:
- Go (공식)
- rust-analyzer
- ESLint
- GitLens

**.vscode/settings.json**:
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "[go]": {
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    },
    "[rust]": {
        "editor.formatOnSave": true
    }
}
```

## 2. 프로젝트 구조

### 2.1 전체 구조

```
sage/
├── core/                 # 핵심 비즈니스 로직
│   ├── agent/           # Agent 구현
│   ├── did/             # DID 처리
│   ├── message/         # 메시지 구조체
│   ├── resolver/        # DID Resolver
│   └── signature/       # RFC 9421 구현
│
├── server/              # Gateway 서버 (선택)
│   ├── handler/         # HTTP 핸들러
│   ├── middleware/      # 미들웨어
│   └── router.go        # 라우팅
│
├── cmd/                 # 실행 파일
│   ├── agent-peer/      # P2P 에이전트
│   └── gateway/         # Gateway 서버
│
├── config/              # 설정 관리
│   ├── config.go        # 설정 로더
│   └── schema.go        # 설정 구조체
│
├── pkg/                 # 공개 API
│   └── sage.go          # 외부 사용 인터페이스
│
├── rust/                # Rust 모듈
│   └── sage_crypto/     # 암호화 엔진
│       ├── src/
│       └── Cargo.toml
│
├── scripts/             # 유틸리티 스크립트
├── testdata/            # 테스트 데이터
└── docs/                # 문서
```

### 2.2 모듈별 책임

| 모듈 | 책임 | 주요 파일 |
|------|------|-----------|
| core/agent | Agent 생명주기 관리 | agent.go, transport.go |
| core/signature | RFC 9421 서명 | signature.go, canonicalize.go |
| core/did | DID Document 처리 | document.go, parser.go |
| core/resolver | 블록체인 통신 | resolver.go, cache.go |
| rust/sage_crypto | 암호화 연산 | lib.rs, ffi.rs, wasm.rs |

## 3. 코드 작성 가이드

### 3.1 Go 코딩 규칙

**명명 규칙**:
```go
// 공개 함수/타입은 대문자로 시작
func SignMessage(msg []byte) []byte

// 내부 함수는 소문자로 시작
func canonicalizeHeaders(headers map[string]string) string

// 인터페이스는 -er 접미사
type Signer interface {
    Sign(data []byte) ([]byte, error)
}
```

**에러 처리**:
```go
// 에러는 항상 마지막 반환값
func Resolve(did string) (*DIDDocument, error) {
    doc, err := fetchFromBlockchain(did)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch DID: %w", err)
    }
    return doc, nil
}

// 센티널 에러 정의
var (
    ErrDIDNotFound = errors.New("DID not found")
    ErrInvalidSignature = errors.New("invalid signature")
)
```

**컨텍스트 사용**:
```go
func (r *Resolver) ResolveWithContext(ctx context.Context, did string) (*DIDDocument, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        return r.resolve(did)
    }
}
```

### 3.2 Rust 코딩 규칙

```rust
// 명확한 lifetime 표시
pub fn verify<'a>(
    message: &'a [u8],
    signature: &[u8],
    public_key: &[u8]
) -> Result<bool, CryptoError> {
    // 구현
}

// FFI 안전성
#[no_mangle]
pub extern "C" fn sage_verify(
    msg_ptr: *const u8,
    msg_len: usize,
    sig_ptr: *const u8,
    sig_len: usize,
) -> bool {
    // NULL 체크 및 안전한 변환
    if msg_ptr.is_null() || sig_ptr.is_null() {
        return false;
    }
    
    unsafe {
        let msg = std::slice::from_raw_parts(msg_ptr, msg_len);
        let sig = std::slice::from_raw_parts(sig_ptr, sig_len);
        verify(msg, sig).unwrap_or(false)
    }
}
```

### 3.3 Clean Code 원칙

1. **함수는 한 가지 일만**: 30줄 이내 유지
2. **명확한 이름**: `VerifySignature` > `CheckSig`
3. **주석은 Why를 설명**: 무엇(What)이 아닌 왜(Why)
4. **DRY 원칙**: 중복 코드 제거
5. **일관된 추상화 수준**: 고수준과 저수준 로직 분리

## 4. 빌드 및 테스트

### 4.1 빌드

```bash
# Go 빌드
go build -o bin/agent ./cmd/agent-peer
go build -o bin/gateway ./cmd/gateway

# Rust 라이브러리 빌드
cd rust/sage_crypto
cargo build --release

# WASM 빌드
wasm-pack build --target bundler --out-dir pkg

# Docker 이미지 빌드
docker build -t sage/agent:latest -f docker/Dockerfile.agent .
docker build -t sage/gateway:latest -f docker/Dockerfile.gateway .
```

### 4.2 테스트

```bash
# 전체 테스트 실행
make test

# Go 단위 테스트
go test ./...
go test -v ./core/signature  # 특정 패키지

# 커버리지 확인
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Rust 테스트
cd rust/sage_crypto && cargo test

# 통합 테스트
go test -tags=integration ./tests/

# 벤치마크
go test -bench=. ./core/signature
```

### 4.3 Lint 및 포맷팅

```bash
# Go lint
golangci-lint run

# Go 포맷팅
go fmt ./...
goimports -w .

# Rust lint
cargo clippy -- -D warnings

# Rust 포맷팅
cargo fmt
```

## 5. 디버깅

### 5.1 로깅

```go
// 구조화된 로깅 사용
import "github.com/sirupsen/logrus"

log := logrus.WithFields(logrus.Fields{
    "did": agentDID,
    "method": "VerifyMessage",
})
log.Debug("Starting signature verification")
```

### 5.2 디버거 설정

**VS Code launch.json**:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Agent",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/agent-peer",
            "args": ["-config", "config/debug.yaml"]
        }
    ]
}
```

### 5.3 성능 프로파일링

```bash
# CPU 프로파일
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# 메모리 프로파일
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# 실행 중인 서버 프로파일
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
```

## 6. 기여 가이드

### 6.1 브랜치 전략

```bash
main              # 안정 버전
├── develop       # 개발 브랜치
└── feature/*     # 기능 개발
    fix/*         # 버그 수정
    release/*     # 릴리즈 준비
```

### 6.2 커밋 메시지

```
<type>(<scope>): <subject>

<body>

<footer>
```

**예시**:
```
feat(core): implement RFC 9421 signature verification

- Add canonicalization function
- Implement Ed25519 verification
- Add comprehensive tests

Closes #123
```

**타입**:
- feat: 새 기능
- fix: 버그 수정
- docs: 문서 변경
- style: 코드 스타일 변경
- refactor: 리팩토링
- test: 테스트 추가/수정
- chore: 빌드/도구 변경

### 6.3 Pull Request 체크리스트

- [ ] 모든 테스트 통과
- [ ] 코드 커버리지 80% 이상
- [ ] Lint 경고 없음
- [ ] 문서 업데이트
- [ ] CHANGELOG 업데이트
- [ ] 커밋 메시지 규칙 준수

### 6.4 코드 리뷰 가이드

**리뷰어 체크포인트**:
1. 기능 요구사항 충족
2. 테스트 적절성
3. 에러 처리
4. 성능 고려사항
5. 보안 취약점
6. 문서화

**건설적인 피드백**:
```
// Good
"이 부분에서 context를 사용하면 timeout 처리가 가능할 것 같습니다."

// Bad
"이 코드는 잘못되었습니다."
```

## 개발 로드맵

| 단계 | 기간 | 목표 |
|------|------|------|
| M1 | 2주 | 프로젝트 구조 및 CI/CD 설정 |
| M2 | 3주 | Rust 암호화 모듈 구현 |
| M3 | 3주 | Go SDK 및 기본 Agent 구현 |
| M4 | 2주 | DID Resolver 및 블록체인 연동 |
| M5 | 3주 | Gateway 서버 구현 |
| M6 | 2주 | TypeScript SDK 및 WASM 빌드 |
| M7 | 3주 | 통합 테스트 및 성능 최적화 |
| M8 | 1주 | 문서화 및 베타 릴리즈 |

## 추가 리소스

- [Go 공식 문서](https://golang.org/doc/)
- [Rust 공식 문서](https://doc.rust-lang.org/)
- [RFC 9421 명세](https://datatracker.ietf.org/doc/html/rfc9421)
- [W3C DID 명세](https://www.w3.org/TR/did-core/)
- [프로젝트 Wiki](https://github.com/sage-project/sage/wiki)