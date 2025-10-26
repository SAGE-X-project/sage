# SAGE 프로젝트 작업 우선순위 리스트

마지막 업데이트: 2025-10-26

## 📊 작업 개요

프로젝트 정리 및 문서화 작업 완료 후, 다음 단계로 진행해야 할 작업들을 우선순위별로 정리했습니다.

---

## 🔴 최우선 작업 (High Priority)

### 1. 핵심 패키지 README 작성 ✅ **완료**

**대상 파일:**
- [x] `pkg/agent/crypto/README.md` - 암호화 키 관리 문서 (2,000+ 줄)
- [x] `pkg/agent/did/README.md` - DID 관리 문서 (1,600+ 줄)
- [x] `pkg/agent/session/README.md` - 세션 관리 문서 (1,400+ 줄)

**완료 상태:**
- ✅ 세 개의 핵심 패키지 README 모두 작성 완료
- ✅ 15개 이상의 실용 예제 포함
- ✅ 아키텍처 다이어그램, 성능 벤치마크, FAQ 포함

**작업 내용:**
각 README에 다음 내용 포함:
- 패키지 개요 및 목적
- 주요 기능 및 API
- 사용 예제 (코드 포함)
- 아키텍처 다이어그램
- 테스트 방법
- 참고 문서 링크

**영향도:** ⭐⭐⭐⭐⭐ (매우 높음)
**소요시간:** 각 2-3시간
**ROI:** 매우 높음

---

### 2. README.md 버전 참조 수정 ✅ **완료**

**파일:** `README.md` 라인 35, 37

**문제:**
- "V4 Update" 문구가 프로젝트 버전 1.3.0과 혼동 가능

**완료 작업:**
- [x] "V4 Update" → "SageRegistryV4"로 명확히 변경
- [x] 스마트 컨트랙트 버전임을 명시
- [x] 링크 텍스트 개선 ("SageRegistryV4 Deployment Guide")

**영향도:** ⭐⭐⭐⭐ (높음)
**소요시간:** 15분
**ROI:** 매우 높음 (빠른 수정으로 큰 개선)

---

### 3. DID 통합 테스트 완성 ✅ **완료**

**파일:** `tests/integration/did_integration_enhanced_test.go`

**완료 상태:**
- ✅ TestDIDRegistrationEnhanced 모두 통과 (4개 서브테스트)
- ✅ Mock Ethereum 서버 자동 연동 완료
- ✅ testutil 패키지로 환경 자동 감지

**테스트 결과:**
- ✅ Register DID on blockchain or mock
- ✅ Verify DID resolution
- ✅ Update DID document
- ✅ Revoke DID
- 실행 시간: 0.02s

**영향도:** ⭐⭐⭐⭐ (높음)
**소요시간:** 30분
**ROI:** 높음

---

## 🟡 중요 작업 (Medium Priority)

### 4. TODO/FIXME 주석 처리

코드베이스에 남아있는 TODO/FIXME 주석들:

**4.1 P-256 알고리즘 지원 추가** ✅ **완료**
- [x] 파일: `pkg/agent/crypto/keys/algorithms.go:59`
- [x] 내용: "TODO: Add support for distinguishing between different ECDSA curves (P-256, Secp256k1, etc.)"
- [x] 소요시간: 2-3시간
- [x] 완료 일자: 2025-10-26
- [x] 참고: P-256 (NIST prime256v1, secp256r1) ECDSA 곡선 지원 추가 완료
  - KeyTypeP256 상수 추가
  - P-256 키 생성, 서명, 검증 구현
  - PEM/JWK 직렬화 지원
  - RFC9421 알고리즘 등록 (ecdsa-p256-sha256)
  - 포괄적인 테스트 추가 (8개 테스트 시나리오)

**구현 내용:**
- `pkg/agent/crypto/types.go`: KeyTypeP256 상수 추가
- `pkg/agent/crypto/keys/p256.go` (신규): 완전한 P-256 구현 (170줄)
- `pkg/agent/crypto/keys/p256_test.go` (신규): 포괄적인 테스트 (270줄)
- `pkg/agent/crypto/keys/algorithms.go`: RFC9421 "ecdsa-p256-sha256" 등록
- `pkg/agent/crypto/formats/pem.go`: P-256 PEM 직렬화 지원
- `pkg/agent/crypto/formats/jwk.go`: P-256 JWK 직렬화 지원 (EC/P-256/ES256)
- 모든 관련 테스트 파일 업데이트

**4.2 WebSocket Origin 체크 구현** ✅ **완료**
- [x] 파일: `pkg/agent/transport/websocket/server.go:75`
- [x] 내용: "TODO: Implement proper origin checking in production"
- [x] 소요시간: 1시간
- [x] 완료 일자: 2025-10-26

**구현 내용:**
- Development mode (기본): 모든 origin 허용
- Production mode: 명시적으로 허용된 origin만 허용
- `NewWSServerWithOrigins()` 생성자 추가
- 동적 origin 관리 메서드 (`AddAllowedOrigin`, `RemoveAllowedOrigin`)
- Origin 체크 활성화/비활성화 기능
- 포괄적인 테스트 추가 (4개 서브테스트)

**4.3 불안정한 테스트 수정** ❌ **해당 없음**
- 파일: `pkg/agent/core/rfc9421/verifier_test.go:289`
- 내용: "FIXME: This test is flaky"
- 상태: 코드베이스에 해당 FIXME 코멘트 없음 (이미 해결됨 또는 문서 오류)

**4.4 DID 파싱 캐시 구현** ❌ **해당 없음**
- 파일: `pkg/agent/did/utils.go:67`
- 내용: "TODO: Cache parsed DIDs"
- 상태: 코드베이스에 해당 TODO 코멘트 없음

**4.5 에러 케이스 테스트 추가** ❌ **해당 없음**
- 파일: `pkg/agent/did/ethereum/client_test.go:234`
- 내용: "TODO: Add error case tests"
- 상태: 코드베이스에 해당 TODO 코멘트 없음

**영향도:** ⭐⭐⭐⭐ (높음)
**총 소요시간:** 6-9시간
**ROI:** 높음 (기술 부채 감소)

---

### 5. A2A/gRPC Transport 구현

**현재 상태:**
- `pkg/agent/transport/README.md`에 "🚧 Planned" 상태로 문서화됨
- HTTP, WebSocket transport는 구현 완료
- gRPC/A2A는 미구현

**작업 내용:**
- [ ] A2A 프로토콜 명세 검토
- [ ] gRPC transport 인터페이스 구현
- [ ] 클라이언트/서버 구현
- [ ] 테스트 작성
- [ ] 문서 업데이트
- [ ] Transport selector에 `grpc://` scheme 등록

**영향도:** ⭐⭐⭐⭐⭐ (매우 높음 - 프로덕션 성능)
**소요시간:** 1-2주
**ROI:** 높음 (장기적 성능 개선)

**참고:**
- 현재 HTTP/WebSocket만으로도 동작하므로 급하지 않음
- 대규모 에이전트 네트워크 배포 시 필수

---

### 6. SDK 문서 개선 ✅ **완료**

**대상:**
- [x] `sdk/python/README.md` - 321 → 908 lines (+587 lines, +183%)
- [x] `sdk/typescript/README.md` - 517 → 1,396 lines (+879 lines, +170%)
- [x] `sdk/java/sage-client/README.md` - 358 → 1,224 lines (+866 lines, +242%)
- [x] `sdk/rust/sage-client/README.md` - 293 → 1,126 lines (+833 lines, +284%)

**완료 일자:** 2025-10-26

**완료 내용:**
각 SDK에 다음 섹션 추가:

**Troubleshooting (문제 해결):**
- Common Issues (일반적인 문제들)
- Debug Mode (디버그 모드)
- Performance Issues (성능 문제)
- Language-specific issues (언어별 특정 문제)

**Best Practices (모범 사례):**
- Security (보안)
  - Private key management (개인키 관리)
  - Input validation (입력 검증)
  - Error handling (에러 처리)
- Performance (성능)
  - Session reuse (세션 재사용)
  - Parallel processing (병렬 처리)
  - Memory optimization (메모리 최적화)
- Code Organization (코드 구조)
- Testing (테스팅)

**Advanced Usage (고급 사용):**
- Multi-agent coordination (다중 에이전트 조정)
- Connection pooling (연결 풀링)
- Monitoring and metrics (모니터링 및 메트릭)
- Custom implementations (커스텀 구현)

**Language-specific Content:**
- **Python**: Async/await patterns, context managers, pytest examples
- **TypeScript**: React hooks, Vite/Webpack config, Jest/RTL testing
- **Java**: Spring Boot integration, Micrometer metrics, Circuit breaker patterns
- **Rust**: Ownership/lifetime patterns, Tokio best practices, WASM support

**통계:**
- 총 라인 수: 1,489 → 4,654 lines
- 증가량: +3,165 lines (+212%)
- 평균 SDK당: +791 lines

**영향도:** ⭐⭐⭐⭐ (높음 - SDK 채택률 대폭 향상)
**실제 소요시간:** 5시간 (SDK 당 ~1.25시간)
**ROI:** 높음

---

## 🟢 일반 작업 (Low Priority)

### 7. Internal 패키지 문서화 ✅ **완료**

**대상:**
- [x] `internal/cryptoinit/README.md` - 암호화 서브시스템 초기화 문서 (347줄)
- [x] `internal/logger/README.md` - 구조화된 로깅 시스템 문서 (750줄)
- [x] `internal/metrics/README.md` - Prometheus 메트릭 수집 문서 (900줄)
- [x] `internal/sessioninit/README.md` - 핸드셰이크-세션 통합 문서 (620줄)

**완료 일자:** 2025-10-26

**완료 내용:**

**internal/cryptoinit (347줄):**
- 패키지 목적 및 아키텍처 설명
- 등록된 컴포넌트 (Ed25519, Secp256k1, P-256 키 생성기)
- Storage 및 Format constructors (JWK, PEM)
- 초기화 검증 및 사용 예제
- 설계 결정 사항 (init() 함수 사용 이유)

**internal/logger (750줄):**
- 구조화된 로깅 인터페이스 (Logger, Field, Level)
- Context 통합 및 영구 필드 관리
- SageError 타입 사용법
- 로그 레벨 제어 및 필드 타입
- Best practices (적절한 로그 레벨, 타입 안전 필드, 컨텍스트 전파)
- 성능 고려사항 (조건부 디버그 로깅, 필드 할당 최적화)
- HTTP/Crypto/Session 로깅 패턴

**internal/metrics (900줄):**
- Prometheus 메트릭 카테고리 (crypto, handshake, message, session)
- 메트릭 기록 예제 (Counter, Gauge, Histogram)
- 커스텀 메트릭 수집기 (타이밍, 퍼센타일)
- HTTP /metrics 엔드포인트 설정
- Prometheus 쿼리 예제 및 Grafana 대시보드
- 경고 규칙 (에러율, 리플레이 공격, 느린 연산)
- Best practices (일관된 라벨링, 낮은 카디널리티)

**internal/sessioninit (620줄):**
- handshake.Events 인터페이스 구현
- 임시 X25519 키 관리 (AskEphemeral, OnComplete)
- 세션 생성 플로우 (핸드셰이크 → 세션)
- Key ID 생성 및 바인딩 (IssueKeyID)
- 설계 결정 사항 (결정론적 세션, 임시 키 저장)
- 보안 고려사항 (키 수명주기, 컨텍스트 고유성)
- 완전한 통합 예제 및 커스텀 이벤트 핸들링

**총 통계:**
- 총 라인 수: 2,617 lines
- 평균 패키지당: ~654 lines
- 모든 패키지에 아키텍처 다이어그램, 사용 예제, Best practices 포함

**참고:**
- 원래 계획된 `internal/utils`와 `internal/testutils`는 존재하지 않음
- 실제 internal 디렉토리 구조를 분석하여 문서화

**영향도:** ⭐⭐⭐ (중간 - 내부 개발자 온보딩 및 유지보수)
**실제 소요시간:** 2시간
**ROI:** 중간

---

### 8. 아키텍처 의사결정 기록 (ADR) 작성 ✅ **완료**

**완료 상태:**
- ✅ `docs/adr/001-transport-layer-abstraction.md` (507줄)
- ✅ `docs/adr/002-hpke-selection-rationale.md` (539줄)
- ✅ `docs/adr/003-did-method-selection.md` (459줄)
- ✅ 완료 일자: 2025-10-26

**완료 내용:**
각 ADR에 포함된 내용:
- [x] Context (상황) - 상세한 문제 진술 및 요구사항
- [x] Decision (결정) - 선택한 아키텍처 및 구현 상세
- [x] Consequences (결과) - 장단점 및 트레이드오프 분석
- [x] Alternatives Considered (고려한 대안들) - 5-7개 대안 비교 분석
- [x] Implementation Notes - 실제 사용 예제 및 가이드
- [x] Related Documents - 관련 문서 링크
- [x] References - 표준 및 참고 문헌

**ADR 요약:**

**ADR-001: Transport Layer Abstraction**
- MessageTransport 인터페이스 기반 추상화
- HTTP, WebSocket, gRPC/A2A 지원
- 보안 로직과 전송 계층 분리
- 대안 비교: 직접 gRPC, HTTP 전용, Noise 프레임워크, 메시지 브로커, 기능별 구현

**ADR-002: HPKE Selection for E2E Encryption**
- RFC 9180 HPKE 채택 (IETF 표준)
- X25519 + ChaCha20-Poly1305 암호 스위트
- 단일 RTT 키 합의, 전방향 보안성
- 대안 비교: TLS 1.3, Signal Protocol, Noise Protocol, NaCl/libsodium, Age, 커스텀 암호화
- Post-quantum 마이그레이션 경로 포함

**ADR-003: DID Method Selection**
- did:sage 커스텀 DID 메소드
- 멀티체인 지원 (Ethereum, Solana)
- W3C DID Core 1.0 준수
- 블록체인 기반 에이전트 레지스트리
- 대안 비교: X.509 PKI, OAuth/OIDC, did:key, did:web, did:ethr, Self-signed 인증서, IPFS/IPNS

**영향도:** ⭐⭐⭐⭐ (높음 - 장기 유지보수 및 신규 개발자 온보딩)
**실제 소요시간:** 7시간 (ADR 당 2-2.5시간)
**ROI:** 높음

---

### 9. 벤치마크 성능 목표 문서화 ✅ **완료**

**파일:** `tools/benchmark/README.md`

**완료 상태:**
- ✅ 각 성능 목표의 근거 설명 추가
- ✅ 산업 표준과 비교 (TLS 1.3, WireGuard, AWS KMS, HSM 등)
- ✅ 프로덕션 환경 요구사항 연결
- ✅ 완료 일자: 2025-10-26

**완료 내용:**
- [x] Target Rationale & Industry Standards 섹션 추가
  - 각 벤치마크 목표에 대한 산업 표준 비교
  - AWS KMS, HSM, TLS 1.3, WireGuard, Noise Protocol 성능 데이터
  - SAGE 목표 설정 근거 상세 설명
- [x] Acceptable Overhead vs. No Security 섹션 추가
  - 보안 오버헤드 정당성 분석
  - 암호화/복호화, 메시지 서명, 세션 설정 오버헤드 설명
- [x] Production Environment Requirements 섹션 추가
  - 고처리량 API 게이트웨이 시나리오 (10,000 req/sec)
  - 실시간 에이전트 통신 시나리오 (sub-10ms 지연)
  - 대규모 에이전트 네트워크 시나리오 (1,000+ 동시 연결)
  - 데이터 집약적 워크로드 시나리오 (1 GB/s 처리량)

**영향도:** ⭐⭐⭐ (중간 - 성능 기준 명확화)
**실제 소요시간:** 1시간
**ROI:** 중간

---

### 10. 부하 테스트 시나리오 확장 ✅ **완료**

**기존 시나리오:**
- baseline (기본 부하)
- stress (스트레스 테스트)
- spike (급증 테스트)
- soak (장시간 테스트)

**완료 상태:**
- ✅ concurrent-sessions (동시 세션 테스트) - 60 VUs, 각 3-5 세션 유지
- ✅ did-operations (DID 연산 집중 테스트) - 100 VUs, DID 레지스트리 성능 검증
- ✅ hpke-operations (암호화 연산 집중 테스트) - 80 VUs, CPU 집약적 암호화 작업
- ✅ mixed-workload (복합 워크로드) - 75 VUs, 실제 프로덕션 트래픽 시뮬레이션
- ✅ 완료 일자: 2025-10-26

**완료 내용:**
- [x] 4개의 새로운 k6 시나리오 스크립트 작성 (~1,000줄)
- [x] config.js에 각 시나리오별 VUs 및 thresholds 설정 추가
- [x] run-loadtest.sh 스크립트 업데이트 (새 시나리오 지원)
- [x] README.md에 각 시나리오 상세 문서화
  - Purpose, Profile, Phases, Thresholds, Use Cases
  - 실행 방법 및 사용 시나리오 설명

**시나리오별 특징:**
1. **concurrent-sessions**: 세션 격리 및 동시성 검증
2. **did-operations**: DID 레지스트리 용량 계획
3. **hpke-operations**: 암호화 성능 및 CPU 용량 계획
4. **mixed-workload**: 실제 환경 시뮬레이션 (5가지 작업 유형 믹스)

**영향도:** ⭐⭐⭐⭐ (높음 - 성능 테스트 커버리지 대폭 개선)
**실제 소요시간:** 2.5시간
**ROI:** 높음

---

## ⚪ 유지보수 작업 (Maintenance)

### 11. 버전 동기화 자동화 스크립트 ✅ **완료**

**배경:**
현재 6개 파일에 버전 정보 분산:
- `VERSION`
- `README.md`
- `contracts/ethereum/package.json`
- `contracts/ethereum/package-lock.json`
- `pkg/version/version.go`
- `lib/export.go`

**완료 작업:**
- [x] `tools/scripts/update-version.sh` 스크립트 생성
- [x] Semantic versioning 검증 추가
- [x] 대화형 확인 프롬프트 추가
- [x] 자동 검증 기능 포함
- [x] 완전한 사용 가이드 (README_VERSION.md) 작성
- [x] CI/CD 통합 예제 포함

**사용법:**
```bash
./tools/scripts/update-version.sh 1.4.0
# 6개 파일의 버전을 자동으로 1.4.0으로 업데이트
```

**영향도:** ⭐⭐⭐⭐ (높음 - 반복 작업 방지)
**소요시간:** 1.5시간
**ROI:** 높음

---

### 12. 문서 정확성 정기 감사

**작업:**
- [ ] 월 1회 문서 검토 프로세스 수립
- [ ] 코드 예제가 컴파일되는지 확인
- [ ] 모든 경로 참조 확인
- [ ] 존재하지 않는 기능 문서화 방지

**체크리스트:**
```markdown
- [ ] README.md의 모든 링크 동작 확인
- [ ] 코드 예제 컴파일 테스트
- [ ] 스크린샷 최신 상태 확인
- [ ] 버전 번호 일관성 확인
- [ ] 계획된 기능과 구현된 기능 구분 명확히
```

**영향도:** ⭐⭐⭐ (중간)
**소요시간:** 월 30분
**ROI:** 중간

---

### 13. 테스트 커버리지 리포팅 ✅ **완료**

**완료 상태:**
- ✅ GitHub Actions에 coverage 리포팅 이미 구현됨
- ✅ Codecov 통합 완료 (README 배지 포함)
- ✅ codecov.yml 설정 파일 추가
- ✅ 최소 커버리지 임계값 설정 (project: 80%, patch: 75%)
- ✅ PR 코멘트 자동 설정
- ✅ 완료 일자: 2025-10-26

**구현 내용:**
- GitHub Actions workflow (.github/workflows/test.yml)
  - 커버리지 수집: `go test -coverprofile=coverage.out -covermode=atomic`
  - Codecov 업로드: codecov-action@v5
  - HTML 리포트 생성 및 artifact 업로드
- Codecov 설정 (codecov.yml)
  - Project 커버리지 목표: 80%
  - PR diff 커버리지 목표: 75%
  - 테스트 파일, examples, tools, contracts 제외
  - PR 코멘트 자동 생성
  - GitHub checks annotations 활성화

**현재 커버리지:**
- 핵심 crypto 패키지: 60-90%
- Transport layer: 80-93%
- Session management: 54.5%
- DID operations: 70.1%

**영향도:** ⭐⭐⭐ (중간)
**실제 소요시간:** 30분 (이미 대부분 구현됨, 설정 파일만 추가)
**ROI:** 높음

---

## 📊 우선순위 매트릭스

| 우선순위 | 작업 | 영향도 | 소요시간 | ROI |
|---------|------|--------|---------|-----|
| ✅ 완료 | 핵심 패키지 README | 매우 높음 | 6시간 | ⭐⭐⭐⭐⭐ |
| ✅ 완료 | 버전 참조 수정 | 높음 | 15분 | ⭐⭐⭐⭐⭐ |
| ✅ 완료 | DID 통합 테스트 | 높음 | 30분 | ⭐⭐⭐⭐ |
| ✅ 완료 | TODO/FIXME 처리 | 높음 | 3-4시간 | ⭐⭐⭐⭐ |
| 🟡 중요 | A2A/gRPC Transport | 매우 높음 | 1-2주 | ⭐⭐⭐ |
| ✅ 완료 | SDK 문서 개선 | 높음 | 5시간 | ⭐⭐⭐⭐ |
| ✅ 완료 | Internal 문서 | 중간 | 2시간 | ⭐⭐⭐ |
| ✅ 완료 | ADR 작성 | 높음 | 7시간 | ⭐⭐⭐⭐ |
| ✅ 완료 | 벤치마크 문서 | 중간 | 1시간 | ⭐⭐⭐ |
| ✅ 완료 | 부하 테스트 확장 | 높음 | 2.5시간 | ⭐⭐⭐⭐ |
| ✅ 완료 | 버전 동기화 스크립트 | 높음 | 1.5시간 | ⭐⭐⭐⭐ |
| ✅ 완료 | 커버리지 리포팅 | 중간 | 30분 | ⭐⭐⭐⭐ |
| ⚪ 유지보수 | 문서 정기 감사 | 중간 | 월 30분 | ⭐⭐⭐ |

---

## 🎯 추천 실행 계획

### 이번 주 (Week 1) ✅ **완료**
**목표:** 빠르게 완료 가능한 고효과 작업

- [x] 프로젝트 정리 및 버전 동기화 (완료)
- [x] README.md 버전 참조 수정 (15분)
- [x] `pkg/agent/crypto/README.md` 작성 (2시간)
- [x] `pkg/agent/did/README.md` 작성 (2시간)
- [x] `pkg/agent/session/README.md` 작성 (2시간)
- [x] 버전 동기화 스크립트 작성 (1.5시간)
- [x] DID 통합 테스트 완성 (30분)

**실제 소요시간:** ~8시간

---

### 다음 주 (Week 2) ✅ **완료**
**목표:** 코드 품질 개선 및 기술 부채 해소

- [x] 모든 최우선 작업 완료 ✅
- [x] TODO/FIXME 처리 완료:
  - [x] WebSocket Origin 체크 구현 (1시간) ✅
  - [x] P-256 알고리즘 지원 추가 (2-3시간) ✅
  - ❌ 불안정한 테스트 수정 (해당 없음 - 이미 해결됨)
  - ❌ DID 파싱 캐시 구현 (해당 없음 - 코드베이스에 TODO 없음)
  - ❌ 에러 케이스 테스트 추가 (해당 없음 - 코드베이스에 TODO 없음)

**실제 소요시간:** 3-4시간

---

### 이번 달 (Month 1)
**목표:** 코드 품질 개선 및 자동화

- [ ] 나머지 TODO/FIXME 처리 (3-4시간)
- [ ] 테스트 커버리지 리포팅 설정 (2-3시간)
- [ ] SDK 문서 개선 (4-8시간)
- [ ] ADR 작성 (Transport layer) (2-3시간)

**예상 소요시간:** 11-18시간

---

### 장기 목표 (3-6개월)
**목표:** 대규모 기능 개발

- [ ] A2A/gRPC Transport 구현 (1-2주)
- [ ] 부하 테스트 시나리오 확장 (2-4시간)
- [ ] 나머지 ADR 작성 (4-6시간)
- [ ] 문서 정기 감사 프로세스 확립

---

## 💡 작업 시 고려사항

### 문서 작성 원칙
1. **사용자 관점 우선**: 코드 구조보다 사용 방법 먼저 설명
2. **실행 가능한 예제**: 모든 예제는 복사-붙여넣기로 실행 가능해야 함
3. **다이어그램 활용**: 복잡한 개념은 시각적으로 표현
4. **일관성 유지**: 기존 README (hpke, transport) 스타일 따르기

### 코드 수정 원칙
1. **테스트 우선**: 모든 수정은 테스트 추가/업데이트 포함
2. **하위 호환성**: 기존 API 깨지지 않도록 주의
3. **문서 동시 업데이트**: 코드 변경 시 관련 문서도 함께 수정
4. **리뷰 요청**: 중요한 변경은 팀 리뷰 필수

### 우선순위 조정 기준
- **보안 이슈**: 발견 시 즉시 최우선으로 상향
- **사용자 피드백**: 실제 사용자 문제 발생 시 우선순위 상향
- **블로킹 이슈**: 다른 작업을 막는 경우 우선 처리
- **기술 부채**: 누적되지 않도록 정기적으로 처리

---

## 📝 진행 상황 추적

작업 시작 시:
```bash
# 브랜치 생성
git checkout -b docs/작업명

# 작업 완료 후
git add .
git commit -m "docs: 작업 설명"
git push -u origin docs/작업명

# PR 생성 및 체크리스트 업데이트
```

이 문서의 체크박스 업데이트:
```bash
# 작업 완료 시 - [ ]를 - [x]로 변경
vim docs/next-todo-list.md
git add docs/next-todo-list.md
git commit -m "docs: Update next-todo-list progress"
```

---

## 🔗 참고 자료

- [CLAUDE.md](../CLAUDE.md) - AI 지원 개발 가이드라인
- [CONTRIBUTING.md](../CONTRIBUTING.md) - 기여 가이드
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 아키텍처 문서
- [SPECIFICATION_VERIFICATION_MATRIX.md](./test/SPECIFICATION_VERIFICATION_MATRIX.md) - 명세 검증 매트릭스

---

**마지막 업데이트:** 2025-10-26
**작성자:** Claude (AI Assistant)
**다음 리뷰:** 작업 진행에 따라 수시 업데이트
