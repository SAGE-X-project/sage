# Optional Dependency 전략

**Date:** January 2025
**Status:** 새로운 접근 방법
**Goal:** a2a를 optional dependency로 만들기

---

##  전략 변경

### 기존 계획 (DEPENDENCY_REMOVAL_PLAN.md)
- Integration tests를 별도 모듈로 분리
- go.mod에서 a2a 완전 제거

### 문제점 발견
```
cmd/random-test → tests/random
tests/integration → sage (replace directive)
→ 순환 의존성 발생
```

### 새로운 전략: Build Tags + Optional Dependency

**핵심 아이디어:**
- go.mod에는 a2a 유지 (호환성)
- Build tags로 a2a 사용을 선택적으로 만듦
- 기본 빌드에서는 a2a 코드 제외
- 문서로 optional dependency임을 명시

---

## 📋 새로운 실행 계획

### Step 1: A2A Adapter에 Build Tags 추가

**파일들:**
1. `pkg/agent/transport/a2a/client.go`
2. `pkg/agent/transport/a2a/server.go`
3. `pkg/agent/transport/a2a/adapter_test.go`

**추가할 태그:**
```go
//go:build a2a
// +build a2a

package a2a
// ...
```

---

### Step 2: cmd/random-test에 Build Tags 추가

**파일:** `cmd/random-test/main.go`

**추가할 태그:**
```go
//go:build integration
// +build integration

package main
// ...
```

---

### Step 3: 빌드 검증

**기본 빌드 (a2a 없이):**
```bash
go build ./...
# pkg/agent/transport/a2a/ 제외됨
# cmd/random-test 제외됨
# Integration tests 제외됨
```

**A2A 포함 빌드:**
```bash
go build -tags=a2a ./pkg/agent/transport/a2a/...
```

**Integration tests 빌드:**
```bash
go build -tags="integration,a2a" ./tests/integration/...
go build -tags=integration ./cmd/random-test/...
```

---

### Step 4: 문서 업데이트

**README.md에 추가:**

````markdown
## Transport Layer

SAGE는 transport-agnostic 아키텍처를 사용합니다.

### A2A Transport (Optional)

A2A transport를 사용하려면:

1. go.mod에 a2a 의존성 추가 (이미 포함됨)
2. Build tags로 빌드:
   ```bash
   go build -tags=a2a ./...
   ```

3. 코드에서 사용:
   ```go
   import "github.com/sage-x-project/sage/pkg/agent/transport/a2a"

   transport := a2a.NewA2ATransport(conn)
   ```

### Other Transports

- HTTP/REST (계획 중)
- WebSocket (계획 중)
````

---

##  성공 기준

### 기본 빌드 (a2a 없이)
```bash
# 1. 빌드 성공
go build ./cmd/sage-crypto
go build ./cmd/sage-did
# 예상: 성공 

# 2. Unit tests 성공
go test ./pkg/agent/handshake/...
go test ./pkg/agent/hpke/...
# 예상: 모두 통과  (MockTransport 사용)

# 3. A2A adapter 제외 확인
go build ./pkg/agent/transport/a2a/
# 예상: 빌드 안 됨 (build tag 필요) 
```

### A2A 포함 빌드
```bash
# 1. A2A adapter 빌드
go build -tags=a2a ./pkg/agent/transport/a2a/
# 예상: 성공 

# 2. Integration tests 빌드
go build -tags="integration,a2a" ./tests/integration/session/handshake/server
# 예상: 성공 
```

---

##  이 전략의 장점

### 1. 순환 의존성 해결 
- 모듈 분리 불필요
- 복잡도 감소

### 2. 호환성 유지 
- 기존 사용자 영향 최소화
- go.mod 변경 불필요

### 3. 선택적 사용 
- A2A 필요 없는 사용자: 기본 빌드
- A2A 필요한 사용자: `-tags=a2a`

### 4. 깔끔한 의존성 
- 기본 빌드는 a2a import 안 함
- go list로 확인 가능

---

## 🤔 Go 버전 문제

### 문제
- go.mod에 a2a가 있으면 Go 1.24.4+ 필요
- 제거하면 1.23.0으로 복원 가능

### 해결책 (2가지 옵션)

#### Option A: Go 1.24.4 유지
- go.mod에 a2a 유지
- Build tags로 선택적 사용
- **장점:** 안정성, 호환성
- **단점:** Go 버전 높음

#### Option B: Go 1.23.0 복원
- go.mod에서 a2a 제거
- A2A 사용자가 직접 추가
- **장점:** 낮은 Go 버전
- **단점:** 사용자 부담 증가

### 권장: Option A (Go 1.24.4 유지)

**이유:**
1. Go 1.24.4는 충분히 합리적 (2024년 릴리스)
2. 사용자 편의성 우선
3. 호환성 문제 최소화
4. Build tags로 충분히 선택적 사용 가능

---

##  제안서 목표 재검토

### 원래 목표 (ARCHITECTURE_REFACTORING_PROPOSAL.md)
1.  Transport Interface 추상화 (완료)
2.  A2A Adapter 구현 (완료)
3.  a2a-go 의존성 제거 (부분 달성)
4.  Go 1.23.0 복원 (미달성)

### 새로운 목표 (Optional Dependency 전략)
1.  Transport Interface 추상화 (완료)
2.  A2A Adapter 구현 (완료)
3.  a2a를 optional로 만들기 (build tags)
4.  Go 1.24.4 유지 (호환성 우선)

---

##  즉시 실행

**Task 1: A2A Adapter에 build tags 추가**

파일별 수정:
1. pkg/agent/transport/a2a/client.go
2. pkg/agent/transport/a2a/server.go
3. pkg/agent/transport/a2a/adapter_test.go

각 파일 맨 위에 추가:
```go
//go:build a2a
// +build a2a
```

**진행할까요?**
