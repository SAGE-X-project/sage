# Simple Agent Initialization

SAGE 에이전트의 간단한 초기화 예시입니다. 이 예시는 에이전트가 프로그램 시작 시 키를 자동으로 관리하는 방법을 보여줍니다.

##  주요 기능

-  **자동 키 감지**: 프로그램 시작 시 기존 키 파일 확인
-  **키 생성**: 키가 없으면 자동으로 새로 생성
-  **키 로드**: 기존 키가 있으면 자동으로 로드
-  **키 저장**: 생성된 키를 파일로 저장
-  **메시지 서명**: Ed25519를 사용한 메시지 서명/검증

##  빠른 시작

```bash
# 예시 실행
go run main.go

# 두 번째 실행 (기존 키 로드)
go run main.go

# 키 파일 삭제 후 실행 (새 키 생성)
rm -rf keys/
go run main.go
```

##  생성되는 파일들

```
keys/
├── ecdsa.key      # ECDSA 개인키 (PEM 형식)
├── ed25519.key    # Ed25519 개인키 (바이너리)
└── x25519.key     # X25519 개인키 (바이너리)
```

##  코드 예시

### 에이전트 생성

```go
// 에이전트 생성 (키 자동 관리)
agent, err := NewSimpleAgent("my-agent", "./keys")
if err != nil {
    log.Fatal(err)
}

// 에이전트 정보 출력
agent.PrintInfo()
```

### 메시지 서명

```go
// 메시지 서명
message := []byte("Hello, SAGE!")
signature := agent.SignMessage(message)

// 서명 검증
valid := agent.VerifyMessage(message, signature)
fmt.Printf("Signature valid: %t\n", valid)
```

### 공개키 가져오기

```go
// 공개키들 가져오기
ecdsaPub, ed25519Pub, x25519Pub, err := agent.GetPublicKeys()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("ECDSA public key: %d bytes\n", len(ecdsaPub))
fmt.Printf("Ed25519 public key: %d bytes\n", len(ed25519Pub))
fmt.Printf("X25519 public key: %d bytes\n", len(x25519Pub))
```

##  동작 플로우

1. **프로그램 시작**
   - 에이전트 이름과 키 디렉토리 설정

2. **키 디렉토리 확인**
   - 키 디렉토리가 없으면 생성

3. **키 파일 확인**
   - 각 키 타입별로 파일 존재 여부 확인

4. **키 로드 또는 생성**
   - 기존 키가 있으면 로드
   - 없으면 새로 생성하고 저장

5. **에이전트 초기화 완료**
   - 모든 키가 준비된 상태

##  보안

- 키 파일은 `0600` 권한으로 저장
- 키 디렉토리는 `0700` 권한으로 생성
- 프로덕션에서는 안전한 위치에 키 저장

##  테스트 시나리오

### 시나리오 1: 첫 실행
```bash
go run main.go
# 출력: 새로운 키들이 생성됩니다
```

### 시나리오 2: 재실행
```bash
go run main.go
# 출력: 기존 키들이 로드됩니다
```

### 시나리오 3: 키 파일 삭제 후 실행
```bash
rm -rf keys/
go run main.go
# 출력: 새로운 키들이 다시 생성됩니다
```

##  다음 단계

이 예시를 이해한 후 다음을 시도해보세요:

1. **[완전한 에이전트 초기화](../agent-initialization/)**: 블록체인 등록 포함
2. **[A2A 통합 예시](../a2a-integration/)**: 에이전트 간 통신
3. **[MCP 통합 예시](../mcp-integration/)**: MCP 도구 보안

##  기여하기

개선 사항이나 버그 리포트는 [GitHub Issues](https://github.com/sage-x-project/sage/issues)에 제출해주세요.

##  라이선스

LGPL-3.0 - 자세한 내용은 [LICENSE](../../../LICENSE)를 참조하세요.
