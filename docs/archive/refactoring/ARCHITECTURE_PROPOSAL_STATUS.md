# ARCHITECTURE_REFACTORING_PROPOSAL.md 진행 상황 체크리스트

**Date:** 2025년 1월
**Status:** Phase 1-2 완료, Phase 3 보류, Phase 4 진행 중

---

##  제안서 Phase 별 진행 상황

###  Phase 1: sage 리팩토링 (완료 100%)

**Day 1-2: 인터페이스 설계**
-  `pkg/agent/transport/interface.go` 생성
-  `SecureMessage`, `Response` 타입 정의
-  `MessageTransport` 인터페이스 정의

**Day 3-4: 코드 리팩토링**
-  `handshake/client.go` 리팩토링
-  `handshake/server.go` 리팩토링
-  `hpke/client.go` 리팩토링
-  `hpke/server.go` 리팩토링
-  `hpke/common.go` 리팩토링

**Day 5: 테스트 및 정리**
-  테스트 코드 업데이트 (MockTransport로 전환)
-  go.mod에서 a2a-go 제거 → Build tags로 대체
-  Go 1.23.0 복원 확인 → 1.24.4 유지 (호환성 우선)
-  전체 테스트 실행 (12/12 통과)

**추가 완료 (제안서 이상):**
-  MockTransport 구현
-  Build tags 전략 적용
-  Unit tests 재작성 (5배 빠름)

---

###  Phase 2: A2A Adapter (완료 100%)

**Day 1: 프로젝트 설정**
-  sage 내부 `pkg/agent/transport/a2a` 패키지 생성 (별도 저장소 대신)
-  Build tags로 optional dependency 구현

**Day 2: Adapter 구현**
-  `client.go` 구현 (A2ATransport)
-  `server.go` 구현 (A2AServerAdapter)
-  `adapter_test.go` 작성
-  Integration tests 검증

**Day 3 (선택): HTTP Adapter**
- ⏳ 미구현 → **Option 2에서 진행 예정**

---

### ⏸️ Phase 3: sage-adk 통합 (보류)

**현재 상태:** sage-adk는 별도 프로젝트로, 현재 sage 프로젝트 범위 밖

**보류 사유:**
- sage-adk 프로젝트 미착수
- sage 라이브러리 리팩토링 먼저 완료 필요
- 향후 별도로 진행

**대안:**
- sage 프로젝트 내 예제 작성 가능
- Transport 사용 예제 추가 (Option 1-3 완료 후)

---

###  Phase 4: 문서 업데이트 (진행 중 60%)

**완료된 문서:**
-  `pkg/agent/transport/README.md` - Transport 가이드
-  `docs/TRANSPORT_REFACTORING.md` - 리팩토링 문서
-  `docs/EXAMPLES_MIGRATION_PLAN.md` - 예제 분석
-  `docs/NEXT_TASKS_PRIORITY.md` - 향후 작업
-  `docs/BUILD_TAGS_SUCCESS.md` - Build tags 성공
-  `docs/FINAL_SUMMARY_KO.md` - 최종 요약

**남은 작업:**
- ⏳ README.md 업데이트 (메인 프로젝트)
- ⏳ docs/handshake/*.md 업데이트
- ⏳ 마이그레이션 가이드 작성
- ⏳ API 문서 업데이트 (godoc)

---

### ⏳ Phase 5: 배포 (대기)

**대기 사유:**
- Option 1-3 완료 후 배포 예정
- 안정성 검증 필요

**예정 작업:**
- [ ] sage v2.0.0 릴리스
- [ ] Release notes 작성
- [ ] Breaking changes 문서

---

##  현재 sage 프로젝트에서 진행 가능한 작업

### 즉시 진행 가능 (제안서 Phase 4 남은 부분)

1. **README.md 업데이트** (1시간)
   - Transport 추상화 설명
   - Build tags 사용법
   - 예제 코드 업데이트

2. **마이그레이션 가이드** (2시간)
   - Before/After 비교
   - 단계별 마이그레이션
   - FAQ

3. **API 문서** (1시간)
   - godoc 주석 추가
   - Package-level 문서

---

##  다음 단계: Option 1, 2, 3 진행

### Option 1: 성능 최적화 (P0, 12시간)

**목표:** 세션 생성 최적화 (38 allocations → <10)

**작업:**
1. **P0-1: 키 버퍼 사전 할당** (2시간)
   - 파일: `pkg/agent/session/session.go`
   - 현재: 6번 별도 할당
   - 목표: 1번 할당 후 슬라이싱

2. **P0-2: 단일 HKDF Expand** (4시간)
   - 파일: `pkg/agent/hpke/client.go`, `server.go`
   - 현재: 6번 HKDF 인스턴스
   - 목표: 1번 HKDF로 모든 키 유도

3. **P0-3: 세션 풀** (6시간)
   - 파일: `pkg/agent/session/manager.go`
   - sync.Pool로 세션 재활용
   - GC 압력 80% 감소

---

### Option 2: HTTP Transport (P1, 18시간)

**목표:** HTTP/REST 기반 transport 구현

**작업:**
1. **HTTP Transport 구현** (16시간)
   - `pkg/agent/transport/http/client.go`
   - `pkg/agent/transport/http/server.go`
   - `pkg/agent/transport/http/handler.go`

2. **Transport Selector** (6시간)
   - `pkg/agent/transport/selector.go`
   - 런타임에 transport 선택

3. **문서 업데이트** (2시간)
   - HTTP 사용 예제
   - API 문서

---

### Option 3: WebSocket Transport (P1, 12시간)

**목표:** WebSocket 기반 양방향 통신

**작업:**
1. **WebSocket 구현** (12시간)
   - `pkg/agent/transport/websocket/client.go`
   - `pkg/agent/transport/websocket/server.go`

2. **문서** (4시간)
   - WebSocket 예제
   - 마이그레이션 가이드 완성

---

## 📋 실행 계획

### 1단계: Phase 4 남은 작업 완료 (4시간)
- README.md 업데이트
- 마이그레이션 가이드
- API 문서

### 2단계: Option 1 - 성능 최적화 (12시간)
- P0-1: 키 버퍼 사전 할당
- P0-2: 단일 HKDF
- P0-3: 세션 풀

### 3단계: Option 2 - HTTP Transport (18시간)
- HTTP Transport 구현
- Transport Selector
- 문서

### 4단계: Option 3 - WebSocket Transport (12시간)
- WebSocket 구현
- 최종 문서

### 5단계: Phase 5 - 배포 (2시간)
- Release notes
- 버전 태깅

**총 예상 시간: 48시간 (약 6일)**

---

##  즉시 시작

**현재 위치:** Phase 4 일부 완료
**다음 작업:** Phase 4 남은 작업 → Option 1 → Option 2 → Option 3

**시작 작업:**
1. Phase 4 남은 문서 작업 (4시간)
2. Option 1 - P0-1 키 버퍼 사전 할당 (2시간)

진행할까요?
