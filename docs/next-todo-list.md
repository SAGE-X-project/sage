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

**4.1 P-256 알고리즘 지원 추가**
- [ ] 파일: `pkg/agent/crypto/keys/algorithms.go:59`
- [ ] 내용: "TODO: Add support for distinguishing between different ECDSA curves (P-256, Secp256k1, etc.)"
- [ ] 소요시간: 2-3시간
- [ ] 참고: P-256 (NIST prime256v1, secp256r1) ECDSA 곡선 지원 추가 필요
  - KeyTypeP256 상수 추가
  - P-256 키 생성, 서명, 검증 구현
  - PEM/JWK 직렬화 지원
  - RFC9421 알고리즘 등록

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

### 6. SDK 문서 개선

**대상:**
- [ ] `sdk/python/README.md`
- [ ] `sdk/typescript/README.md`
- [ ] `sdk/java/README.md`
- [ ] `sdk/rust/README.md`

**현재 상태:**
- 모든 SDK README가 잘 작성되어 있음
- 기본 사용법 및 설치 방법 포함

**개선 사항:**
- 실제 사용 예제 추가 (완전한 예제 코드)
- API 레퍼런스 링크 추가
- 고급 사용 패턴 문서화
- 트러블슈팅 섹션 추가
- 각 언어별 Best Practice 추가

**영향도:** ⭐⭐⭐ (중간)
**소요시간:** SDK 당 1-2시간
**ROI:** 중간 (SDK 채택률 향상)

---

## 🟢 일반 작업 (Low Priority)

### 7. Internal 패키지 문서화

**대상:**
- [ ] `internal/utils/README.md`
- [ ] `internal/testutils/README.md`

**내용:**
- 각 유틸리티 함수 설명
- 사용 예제
- 테스트 헬퍼 사용법

**영향도:** ⭐⭐ (낮음 - 내부 유지보수용)
**소요시간:** 1시간
**ROI:** 낮음

---

### 8. 아키텍처 의사결정 기록 (ADR) 작성

**대상:**
- [ ] `docs/adr/001-transport-layer-abstraction.md`
- [ ] `docs/adr/002-hpke-selection-rationale.md`
- [ ] `docs/adr/003-did-method-selection.md`

**내용:**
각 ADR에 포함할 내용:
- Context (상황)
- Decision (결정)
- Consequences (결과)
- Alternatives Considered (고려한 대안들)

**영향도:** ⭐⭐⭐ (중간 - 장기 유지보수)
**소요시간:** ADR 당 2-3시간
**ROI:** 중간

---

### 9. 벤치마크 성능 목표 문서화

**파일:** `tools/benchmark/README.md`

**현재 상태:**
- 성능 목표 수치는 있음
- 목표 설정 근거 없음

**개선:**
- [ ] 각 성능 목표의 근거 설명
- [ ] 산업 표준과 비교
- [ ] 프로덕션 환경 요구사항 연결

**영향도:** ⭐⭐ (낮음)
**소요시간:** 1시간
**ROI:** 낮음

---

### 10. 부하 테스트 시나리오 확장

**현재 시나리오:**
- baseline (기본 부하)
- stress (스트레스 테스트)
- spike (급증 테스트)
- soak (장시간 테스트)

**추가 시나리오:**
- [ ] concurrent-sessions (동시 세션 테스트)
- [ ] did-operations (DID 연산 집중 테스트)
- [ ] hpke-operations (암호화 연산 집중 테스트)
- [ ] mixed-workload (복합 워크로드)

**영향도:** ⭐⭐⭐ (중간)
**소요시간:** 2-4시간
**ROI:** 중간

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

### 13. 테스트 커버리지 리포팅

**현재 상태:**
- 테스트는 잘 작성되어 있음
- 커버리지 메트릭 없음

**작업:**
- [ ] GitHub Actions에 coverage 리포팅 추가
- [ ] Codecov 또는 Coveralls 통합
- [ ] 최소 커버리지 임계값 설정 (예: 80%)
- [ ] PR에 커버리지 변경사항 표시

**예시 설정:**
```yaml
# .github/workflows/test.yml
- name: Test with coverage
  run: go test -coverprofile=coverage.out ./...

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

**영향도:** ⭐⭐⭐ (중간)
**소요시간:** 2-3시간
**ROI:** 중간

---

## 📊 우선순위 매트릭스

| 우선순위 | 작업 | 영향도 | 소요시간 | ROI |
|---------|------|--------|---------|-----|
| ✅ 완료 | 핵심 패키지 README | 매우 높음 | 6시간 | ⭐⭐⭐⭐⭐ |
| ✅ 완료 | 버전 참조 수정 | 높음 | 15분 | ⭐⭐⭐⭐⭐ |
| ✅ 완료 | DID 통합 테스트 | 높음 | 30분 | ⭐⭐⭐⭐ |
| 🟡 중요 | TODO/FIXME 처리 | 높음 | 6-9시간 | ⭐⭐⭐⭐ |
| 🟡 중요 | A2A/gRPC Transport | 매우 높음 | 1-2주 | ⭐⭐⭐ |
| 🟡 중요 | SDK 문서 개선 | 중간 | 4-8시간 | ⭐⭐⭐ |
| 🟢 일반 | Internal 문서 | 낮음 | 1시간 | ⭐⭐ |
| 🟢 일반 | ADR 작성 | 중간 | 6-9시간 | ⭐⭐ |
| 🟢 일반 | 벤치마크 문서 | 낮음 | 1시간 | ⭐⭐ |
| 🟢 일반 | 부하 테스트 확장 | 중간 | 2-4시간 | ⭐⭐ |
| ✅ 완료 | 버전 동기화 스크립트 | 높음 | 1.5시간 | ⭐⭐⭐⭐ |
| ⚪ 유지보수 | 문서 정기 감사 | 중간 | 월 30분 | ⭐⭐⭐ |
| ⚪ 유지보수 | 커버리지 리포팅 | 중간 | 2-3시간 | ⭐⭐⭐ |

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

### 다음 주 (Week 2) 🔄 **진행 중**
**목표:** 코드 품질 개선 및 기술 부채 해소

- [x] 모든 최우선 작업 완료 ✅
- [ ] TODO/FIXME 처리 (진행 중):
  - [ ] 불안정한 테스트 수정 (1시간)
  - [ ] DID 파싱 캐시 구현 (1-2시간)
  - [ ] WebSocket 우아한 종료 (1-2시간)
  - [ ] 에러 케이스 테스트 추가 (1시간)
  - [ ] P-256 알고리즘 지원 추가 (2-3시간)

**예상 소요시간:** 6-9시간

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
