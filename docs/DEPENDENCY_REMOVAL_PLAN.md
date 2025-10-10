# a2a-go 의존성 제거 계획

**Date:** January 2025
**Status:** 아키텍처 리팩토링 완료, 의존성 제거 진행 중
**Goal:** Go 버전 1.24.4 → 1.23.0 복원, a2a-go 의존성 완전 제거

---

##  현재 상황

### 완료된 작업 
-  Transport Interface 추상화 완료
-  A2A Adapter 구현 완료 (`pkg/agent/transport/a2a/`)
-  Unit tests MockTransport로 전환 완료
-  코드 리팩토링 완료 (handshake, hpke)

### 남은 문제 
```bash
# go.mod 현재 상태:
go 1.24.4                              #  목표: 1.23.0
require (
    github.com/a2aproject/a2a v0.2.6   #  목표: 제거
)
```

### a2a 의존성 사용 현황

**총 5개 파일:**
1. `pkg/agent/transport/a2a/client.go` - **유지 필요** (Adapter)
2. `pkg/agent/transport/a2a/server.go` - **유지 필요** (Adapter)
3. `pkg/agent/transport/a2a/adapter_test.go` - **유지 필요** (Adapter 테스트)
4. `test/integration/tests/session/handshake/server/main.go` - **분리 필요**
5. `test/integration/tests/session/hpke/server/main.go` - **분리 필요**

**핵심 문제:**
- A2A Adapter는 a2a-go가 필요 (정상)
- Integration tests도 a2a-go가 필요 (문제)
- Integration tests가 메인 모듈에 포함됨 → go.mod에 a2a 필요

**해결 방법:**
- Integration tests를 별도 모듈로 분리
- 메인 go.mod에서 a2a 제거
- Go 버전 1.23.0으로 복원

---

##  작업 우선순위

### Priority 1: 핵심 목표 달성 (Critical) 🔴

#### Task 1-1: Integration Tests 별도 모듈 분리
**목표:** Integration tests를 독립 모듈로 분리하여 메인 모듈의 a2a 의존성 제거
**소요 시간:** 2시간
**우선순위:** P0 (최고)

**현재 구조:**
```
sage/
├── go.mod (메인 모듈, a2a 의존)
└── test/integration/tests/session/
    ├── handshake/server/main.go (a2a 사용)
    └── hpke/server/main.go (a2a 사용)
```

**목표 구조:**
```
sage/
├── go.mod (메인 모듈, a2a 제거!) 
└── test/integration/
    ├── go.mod (별도 모듈, a2a 의존) 
    └── tests/session/
        ├── handshake/server/main.go
        └── hpke/server/main.go
```

**실행 계획:**

1. **Integration tests용 go.mod 생성** (30분)
   ```bash
   cd test/integration
   go mod init github.com/sage-x-project/sage/test/integration
   ```

2. **의존성 추가** (30분)
   ```bash
   # Integration tests 의존성
   go get github.com/a2aproject/a2a@v0.2.6
   go get github.com/sage-x-project/sage@latest  # 메인 모듈 참조
   go get google.golang.org/grpc
   go get google.golang.org/protobuf
   go mod tidy
   ```

3. **Replace directive 추가** (15분)
   ```go
   // test/integration/go.mod
   replace github.com/sage-x-project/sage => ../..
   ```

4. **빌드 확인** (45분)
   ```bash
   # Integration tests 빌드 (별도 모듈)
   cd test/integration
   go build -tags="integration,a2a" ./tests/session/handshake/server
   go build -tags="integration,a2a" ./tests/session/hpke/server
   ```

**성공 기준:**
- [ ] `test/integration/go.mod` 존재
- [ ] Integration tests가 별도 모듈에서 빌드됨
- [ ] 메인 모듈과 독립적으로 작동

---

#### Task 1-2: 메인 go.mod에서 a2a 의존성 제거
**목표:** sage 메인 모듈에서 a2a-go 완전 제거
**소요 시간:** 1시간
**우선순위:** P0 (최고)
**의존성:** Task 1-1 완료 후

**실행 계획:**

1. **a2a 의존성 확인** (15분)
   ```bash
   # 메인 모듈에서 a2a 사용 확인
   grep -r "github.com/a2aproject/a2a" --include="*.go" | \
       grep -v test/integration | \
       grep -v vendor

   # 예상 결과: pkg/agent/transport/a2a/ 파일만 나와야 함
   ```

2. **go.mod 수정** (15분)
   ```bash
   # go.mod 백업
   cp go.mod go.mod.backup

   # a2a 의존성 제거
   # require 섹션에서 다음 라인 삭제:
   # github.com/a2aproject/a2a v0.2.6

   # replace 섹션에서 다음 라인 삭제 (있다면):
   # replace github.com/a2aproject/a2a => github.com/a2aproject/a2a-go ...
   ```

3. **go.mod 정리** (15분)
   ```bash
   go mod tidy
   ```

4. **빌드 확인** (15분)
   ```bash
   # A2A adapter는 빌드 실패 예상 (정상)
   # 이유: a2a-go가 없어서

   # 메인 코드 빌드 (A2A adapter 제외)
   go build ./cmd/...
   go build ./pkg/agent/handshake/...
   go build ./pkg/agent/hpke/...

   # 예상 결과: 성공 
   ```

**성공 기준:**
- [ ] go.mod에 a2a 의존성 없음
- [ ] 메인 코드 빌드 성공
- [ ] Unit tests 실행 성공

---

#### Task 1-3: Go 버전 1.23.0으로 복원
**목표:** Go 버전 요구사항을 1.23.0으로 낮춤
**소요 시간:** 30분
**우선순위:** P0 (최고)
**의존성:** Task 1-2 완료 후

**실행 계획:**

1. **go.mod 수정** (10분)
   ```bash
   # Before:
   go 1.24.4
   toolchain go1.24.8

   # After:
   go 1.23.0
   # toolchain 라인 제거
   ```

2. **빌드 확인** (10분)
   ```bash
   # Go 1.23.0으로 빌드
   go build ./...
   ```

3. **테스트 확인** (10분)
   ```bash
   # Unit tests (MockTransport)
   go test ./pkg/agent/handshake/...
   go test ./pkg/agent/hpke/...
   go test ./pkg/agent/session/...
   ```

**성공 기준:**
- [ ] go.mod에 `go 1.23.0`
- [ ] 빌드 성공 (Go 1.23.0)
- [ ] 모든 unit tests 통과

---

#### Task 1-4: A2A Adapter를 Optional로 만들기
**목표:** A2A adapter를 선택적으로 사용 가능하게 만들기
**소요 시간:** 1.5시간
**우선순위:** P1 (높음)
**의존성:** Task 1-1, 1-2 완료 후

**전략:** Build tags 사용

**실행 계획:**

1. **Build tags 추가** (45분)
   ```go
   // pkg/agent/transport/a2a/client.go
   //go:build a2a
   // +build a2a

   package a2a
   // ... (기존 코드)

   // pkg/agent/transport/a2a/server.go
   //go:build a2a
   // +build a2a

   package a2a
   // ... (기존 코드)
   ```

2. **A2A 없이 빌드 가능한지 확인** (30분)
   ```bash
   # A2A 없이 빌드 (기본)
   go build ./...
   # 예상: pkg/agent/transport/a2a/ 제외하고 빌드 성공

   # A2A 포함 빌드
   go build -tags=a2a ./...
   # 예상: a2a-go 의존성 필요로 실패 (정상)
   ```

3. **README 업데이트** (15분)
   - A2A adapter 사용 시 build tags 필요 명시
   - go.mod에 a2a 추가 방법 설명

**성공 기준:**
- [ ] 기본 빌드에 A2A 제외됨
- [ ] `-tags=a2a`로 A2A 포함 가능
- [ ] 문서 업데이트 완료

---

### Priority 2: 문서 업데이트 (High) 🟠

#### Task 2-1: 메인 README.md 업데이트
**목표:** Transport 추상화 반영, 사용법 업데이트
**소요 시간:** 2시간
**우선순위:** P1

**업데이트 내용:**
1. Transport Layer 소개 섹션 추가
2. 사용 예제 업데이트 (A2A adapter 사용법)
3. Go 버전 요구사항 변경 (1.23.0+)
4. Build tags 설명 추가

---

#### Task 2-2: 마이그레이션 가이드 작성
**목표:** 기존 사용자를 위한 마이그레이션 가이드
**소요 시간:** 3시간
**우선순위:** P1

**파일:** `docs/MIGRATION_GUIDE.md`

**내용:**
1. Breaking Changes 설명
2. Before/After 코드 비교
3. 단계별 마이그레이션 절차
4. FAQ

---

#### Task 2-3: API 문서 생성
**목표:** godoc 호환 문서 완성
**소요 시간:** 2시간
**우선순위:** P2

**작업:**
- 모든 exported 심볼에 godoc 주석 추가
- Package-level 문서 추가
- 예제 코드 추가

---

### Priority 3: 검증 및 배포 (Medium) 🟡

#### Task 3-1: 전체 빌드 및 테스트
**목표:** 모든 변경사항 검증
**소요 시간:** 2시간
**우선순위:** P2

**검증 항목:**
- [ ] 메인 모듈 빌드 (`go build ./...`)
- [ ] Unit tests (`go test ./...`)
- [ ] Integration tests (별도 모듈)
- [ ] 예제 코드 빌드

---

#### Task 3-2: CI/CD 파이프라인 업데이트
**목표:** CI에서 별도 모듈 빌드 추가
**소요 시간:** 1시간
**우선순위:** P3

**업데이트:**
```yaml
# .github/workflows/test.yml
- name: Test main module
  run: go test ./...

- name: Test integration module
  run: |
    cd test/integration
    go test -tags="integration,a2a" ./...
```

---

## 📋 실행 순서

### Step 1: 의존성 분리 (P0 작업)
```
1. Task 1-1: Integration tests 별도 모듈 분리 (2h)
2. Task 1-2: 메인 go.mod에서 a2a 제거 (1h)
3. Task 1-3: Go 버전 1.23.0으로 복원 (0.5h)
4. Task 1-4: A2A adapter optional 만들기 (1.5h)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
총 소요 시간: 5시간
```

### Step 2: 문서 업데이트 (P1 작업)
```
5. Task 2-1: README.md 업데이트 (2h)
6. Task 2-2: 마이그레이션 가이드 (3h)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
총 소요 시간: 5시간
```

### Step 3: 검증 (P2-P3 작업)
```
7. Task 3-1: 전체 테스트 (2h)
8. Task 2-3: API 문서 (2h)
9. Task 3-2: CI/CD 업데이트 (1h)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
총 소요 시간: 5시간
```

**전체 예상 시간: 15시간 (약 2일)**

---

##  성공 기준

### 최종 목표 달성 확인

```bash
# 1. Go 버전 확인
grep "^go " go.mod
# 예상: go 1.23.0 

# 2. a2a 의존성 확인
grep "github.com/a2aproject/a2a" go.mod
# 예상: 결과 없음 

# 3. 메인 모듈 빌드
go build ./...
# 예상: 성공 

# 4. Unit tests
go test ./pkg/agent/...
# 예상: 모두 통과 

# 5. Integration tests (별도 모듈)
cd test/integration
go test -tags="integration,a2a" ./...
# 예상: 모두 통과 
```

---

##  즉시 시작

**우선순위 1 작업부터 시작:**

```bash
# Task 1-1: Integration tests 별도 모듈 생성
cd test/integration
go mod init github.com/sage-x-project/sage/test/integration
```

**다음 명령으로 진행 여부 확인:**
```bash
# 현재 위치 확인
pwd
# 예상: /Users/kevin/work/github/sage-x-project/sage

# 시작할까요?
cd test/integration
```

---

##  진행 상황 추적

| Task | 상태 | 소요 시간 | 완료 시간 |
|------|------|----------|----------|
| 1-1: Integration tests 분리 | ⏳ Pending | 2h | - |
| 1-2: go.mod a2a 제거 | ⏳ Pending | 1h | - |
| 1-3: Go 1.23.0 복원 | ⏳ Pending | 0.5h | - |
| 1-4: A2A optional | ⏳ Pending | 1.5h | - |
| 2-1: README 업데이트 | ⏳ Pending | 2h | - |
| 2-2: 마이그레이션 가이드 | ⏳ Pending | 3h | - |
| 3-1: 전체 테스트 | ⏳ Pending | 2h | - |
| 2-3: API 문서 | ⏳ Pending | 2h | - |
| 3-2: CI/CD 업데이트 | ⏳ Pending | 1h | - |

---

**Status:** Ready to Start
**First Task:** Task 1-1 (Integration tests 별도 모듈 분리)
**Expected Completion:** 2일 (15시간)
