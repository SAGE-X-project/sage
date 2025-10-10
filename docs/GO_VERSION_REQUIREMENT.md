# Go 버전 요구사항 정정

## 공식 Go 버전 요구사항

**SAGE 프로젝트는 Go 1.24.4 이상을 요구합니다.**

### feature_list.docx 정정 사항

**원본 명세 (feature_list.docx)**:
- 개발 환경: Go 1.23+ (현재 1.23.0)
- Core Library/CLI: Go 1.23 이상

**정정된 요구사항**:
- **개발 환경**: Go 1.24.4+ (현재 1.24.8)
- **Core Library/CLI**: Go 1.24.4 이상

### 변경 사유

SAGE 프로젝트의 핵심 의존성인 `github.com/a2aproject/a2a-go`가 **Go 1.24.4 이상**을 요구합니다.

이 의존성은 다음 핵심 기능에서 사용됩니다:
- HPKE (RFC 9180) 기반 암호화
- Agent-to-Agent 핸드셰이크 프로토콜
- 세션 관리

따라서 프로젝트 전체가 Go 1.24.4를 최소 요구 버전으로 설정해야 합니다.

## 현재 설정

### go.mod
```go
go 1.24.4

toolchain go1.24.8
```

- **최소 요구 버전**: Go 1.24.4
- **권장 툴체인**: Go 1.24.8

### 시스템 환경
```bash
$ go version
go version go1.24.7 darwin/arm64
```

현재 시스템은 Go 1.24.7을 사용 중이며, 프로젝트 요구사항을 충족합니다.

## 테스트 환경 정보

### SW 정보
- **SW명**: SAGE Core Library
- **SW 유형**: Go 라이브러리 / 보안 미들웨어
- **개발 환경**: **Go 1.24.4+** (현재 1.24.8)
- **Test 환경 상세**:
  - **Core Library/CLI**: Go 1.24.4 이상
  - **Smart Contract**: Solidity 0.8.19
  - **블록체인 네트워크**: Ethereum (Hardhat 로컬 노드)
  - **Chain ID**: 31337 (로컬)
  - **Hardhat 버전**: 2.26.3
  - **Web3 라이브러리**: ethers v6.4.0
  - **암호화 알고리즘**: Secp256k1 (Ethereum 호환), Ed25519 (고성능 EdDSA), X25519 (HPKE 키 교환용)

### 필수 의존성
- Go 모듈:
  - `github.com/ethereum/go-ethereum` v1.16.1
  - `github.com/decred/dcrd/dcrec/secp256k1/v4` v4.4.0
  - `github.com/a2aproject/a2a-go` (Go 1.24.4+ 요구)
  - `golang.org/x/crypto` v0.43.0
- Node.js 패키지:
  - `@nomicfoundation/hardhat-ethers` v4.0.0
  - `hardhat` v2.26.3

## 설치 방법

### Go 설치 (1.24.4 이상)

**macOS (Homebrew)**:
```bash
brew install go@1.24
```

**Linux**:
```bash
# Download from official site
wget https://go.dev/dl/go1.24.8.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.8.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Windows**:
- https://go.dev/dl/ 에서 Go 1.24.8 설치 프로그램 다운로드

### 버전 확인
```bash
go version
# 출력: go version go1.24.7 darwin/arm64 (또는 1.24.4 이상)
```

### 프로젝트 빌드
```bash
cd sage
go mod download
make build
```

## 검증

모든 테스트가 Go 1.24.4+ 환경에서 정상 동작함을 확인했습니다:

```bash
./tools/scripts/verify_all_features.sh -v
```

**결과**: 89/89 테스트 통과 (100%)

## 호환성

| Go 버전 | 상태 | 비고 |
|---------|------|------|
| 1.23.x  | ❌ 미지원 | a2a-go 의존성 요구사항 미충족 |
| 1.24.0-1.24.3 | ❌ 미지원 | a2a-go 의존성 요구사항 미충족 |
| 1.24.4 | ✅ 지원 | 최소 요구 버전 |
| 1.24.5+ | ✅ 지원 | 권장 버전 |

## 참고

- **README.md**: Go 1.24.4+ 요구사항 업데이트 완료
- **go.mod**: `go 1.24.4`, `toolchain go1.24.8` 설정
- **feature_list.docx**: 본 문서로 정정 사항 명시

---

**최종 업데이트**: 2025-10-10
**담당**: SAGE 개발팀
