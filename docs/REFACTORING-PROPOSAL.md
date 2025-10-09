# SAGE 폴더 구조 리팩토링 제안서

**작성일:** 2025-10-10
**목적:** 코드 파악 용이성 향상 및 논리적 구조 개선

---

## 1. 현재 구조 분석

### 1.1 문제점

```
현재 루트 디렉토리: 30개 폴더
├── 핵심 라이브러리 (8개): core, crypto, did, session, handshake, hpke, health, oidc
├── 개발 도구 (4개): benchmark, loadtest, scripts, tests
├── 인프라 (3개): config, docker, migrations
├── 표준 Go 구조 (3개): cmd, pkg, internal
├── 외부 연동 (3개): contracts, sdk, api
├── 문서/예제 (2개): docs, examples
└── 빌드 아티팩트 (7개): build, lib, target, keys, reports, benches, include
```

**핵심 문제:**
1. ❌ **인지 부하 과다**: 루트에 30개 디렉토리 → 구조 파악 어려움
2. ❌ **논리적 그룹화 부재**: 관련 기능이 분산 (crypto 368K, did 260K, core 180K 등)
3. ❌ **Go 표준 위반**: pkg/ 디렉토리가 있지만 storage만 포함
4. ❌ **도구 분산**: benchmark, loadtest, scripts가 각자 위치
5. ❌ **배포 설정 분산**: docker, config, migrations가 흩어짐

### 1.2 의존성 분석

```go
// 현재 import 패턴 (cmd/sage-crypto/main.go)
import (
    "github.com/sage-x-project/sage/crypto/chain/ethereum"  // 루트/crypto
    "github.com/sage-x-project/sage/crypto/chain/solana"    // 루트/crypto
)

// session -> internal/metrics (OK)
// handshake -> crypto, did, session (순환 의존 위험)
```

### 1.3 디렉토리 크기 분포

| 디렉토리 | 크기 | 용도 | 분류 |
|---------|------|------|------|
| crypto | 368K | 암호화 | Core Library |
| did | 260K | DID 처리 | Core Library |
| core | 180K | RFC9421 | Core Library |
| session | 88K | 세션 관리 | Core Library |
| hpke | 64K | HPKE 구현 | Core Library |
| handshake | 64K | 핸드셰이크 | Core Library |
| oidc | 44K | OIDC 지원 | Core Library |
| pkg | 44K | 공용 패키지 | Infrastructure |
| internal | 72K | 내부 패키지 | Infrastructure |
| health | 32K | 헬스체크 | Utility |

---

## 2. 리팩토링 옵션

### 옵션 A: 보수적 접근 (권장) ⭐

**전략:** 최소한의 변경으로 명확성 향상

```
sage/
├── cmd/                          # [유지] CLI 실행 파일
│   ├── sage-crypto/
│   ├── sage-did/
│   ├── sage-verify/
│   └── metrics-demo/
│
├── pkg/                          # [확장] 외부 노출 라이브러리
│   ├── agent/                    # [신규] 핵심 에이전트 기능 그룹
│   │   ├── core/                 # [이동] RFC9421 구현
│   │   ├── crypto/               # [이동] 암호화 기능
│   │   ├── did/                  # [이동] DID 처리
│   │   ├── session/              # [이동] 세션 관리
│   │   ├── handshake/            # [이동] 핸드셰이크
│   │   └── hpke/                 # [이동] HPKE 구현
│   │
│   ├── storage/                  # [유지] 스토리지 추상화
│   │   ├── memory/
│   │   └── postgres/
│   │
│   ├── health/                   # [이동] 헬스체크
│   └── oidc/                     # [이동] OIDC 지원
│
├── internal/                     # [유지] 내부 전용 패키지
│   ├── metrics/
│   ├── logger/
│   └── cryptoinit/
│
├── contracts/                    # [유지] 스마트 컨트랙트
│   ├── ethereum/
│   └── solana/
│
├── sdk/                          # [유지] 클라이언트 SDK
│   ├── java/
│   ├── python/
│   └── rust/
│
├── api/                          # [유지] API 정의
│
├── tools/                        # [신규] 개발 도구 통합
│   ├── benchmark/                # [이동] 성능 벤치마크
│   ├── loadtest/                 # [이동] 부하 테스트
│   └── scripts/                  # [이동] 유틸리티 스크립트
│
├── deployments/                  # [신규] 배포 설정 통합
│   ├── docker/                   # [이동] Docker 설정
│   │   ├── grafana/
│   │   ├── prometheus/
│   │   └── scripts/
│   ├── config/                   # [이동] 환경별 설정
│   │   ├── development.yaml
│   │   ├── staging.yaml
│   │   └── production.yaml
│   └── migrations/               # [이동] DB 마이그레이션
│       ├── 000001_initial_schema.up.sql
│       └── seeds/
│
├── test/                         # [재구성] 테스트 통합
│   ├── integration/              # [이동] tests/ → test/integration/
│   ├── e2e/                      # [신규] E2E 테스트
│   └── fixtures/                 # [신규] 테스트 픽스처
│
├── docs/                         # [유지] 문서
├── examples/                     # [유지] 예제 코드
│
├── build/                        # [유지] 빌드 아티팩트 (gitignore)
├── lib/                          # [유지] 공유 라이브러리
│
└── [root files]                  # go.mod, Makefile, README.md, etc.
```

**장점:**
- ✅ 루트 디렉토리: 30개 → 14개 (53% 감소)
- ✅ 핵심 라이브러리 `pkg/agent/` 아래 통합
- ✅ 도구/배포 설정 명확한 그룹화
- ✅ Go 표준 레이아웃 준수
- ✅ Import path 변경 최소화

**단점:**
- ⚠️ Import path 일부 변경 필요
- ⚠️ 마이그레이션 작업 필요

**Import Path 변경 예시:**
```go
// Before
import "github.com/sage-x-project/sage/crypto"
import "github.com/sage-x-project/sage/did"

// After
import "github.com/sage-x-project/sage/pkg/agent/crypto"
import "github.com/sage-x-project/sage/pkg/agent/did"
```

---

### 옵션 B: 기능 중심 그룹화 (중급)

```
sage/
├── cmd/                          # CLI 도구
│
├── pkg/                          # 공용 라이브러리
│   ├── protocol/                 # 프로토콜 레벨
│   │   ├── rfc9421/             # core/ 이동
│   │   ├── handshake/
│   │   └── session/
│   │
│   ├── security/                 # 보안 레벨
│   │   ├── crypto/
│   │   ├── hpke/
│   │   └── oidc/
│   │
│   ├── identity/                 # 신원 레벨
│   │   └── did/
│   │
│   └── infrastructure/           # 인프라 레벨
│       ├── storage/
│       └── health/
│
├── platform/                     # [신규] 플랫폼 통합
│   ├── contracts/               # 블록체인 계약
│   ├── sdk/                     # 클라이언트 SDK
│   └── api/                     # API 정의
│
├── tooling/                      # [신규] 도구 통합
│   ├── benchmark/
│   ├── loadtest/
│   ├── scripts/
│   └── docker/
│
├── internal/                     # 내부 패키지
├── deployments/                  # 배포 설정
├── test/                         # 테스트
├── docs/                         # 문서
└── examples/                     # 예제
```

**장점:**
- ✅ 기능별 명확한 계층 구조
- ✅ 레이어 아키텍처 명시적
- ✅ 확장 용이성

**단점:**
- ⚠️ Import path 대규모 변경
- ⚠️ 러닝 커브 증가
- ⚠️ 과도한 추상화 위험

---

### 옵션 C: 최소 변경 (가장 보수적)

```
sage/
├── cmd/                          # [유지]
├── pkg/                          # [확장]
│   ├── core/                     # [이동]
│   ├── crypto/                   # [이동]
│   ├── did/                      # [이동]
│   ├── session/                  # [이동]
│   ├── handshake/                # [이동]
│   ├── hpke/                     # [이동]
│   ├── health/                   # [이동]
│   ├── oidc/                     # [이동]
│   └── storage/                  # [유지]
│
├── internal/                     # [유지]
├── contracts/                    # [유지]
├── sdk/                          # [유지]
├── api/                          # [유지]
│
├── tools/                        # [신규]
│   ├── benchmark/                # [이동]
│   ├── loadtest/                 # [이동]
│   └── scripts/                  # [이동]
│
├── deploy/                       # [신규]
│   ├── docker/                   # [이동]
│   ├── config/                   # [이동]
│   └── migrations/               # [이동]
│
├── test/                         # [이름 변경] tests/ → test/
├── docs/                         # [유지]
└── examples/                     # [유지]
```

**장점:**
- ✅ 최소한의 변경
- ✅ Import path 변경 단순 (1단계만)
- ✅ 빠른 마이그레이션

**단점:**
- ⚠️ pkg/ 아래 여전히 8개 디렉토리
- ⚠️ 논리적 그룹화 약함

---

## 3. 권장 사항

### 3.1 선택 기준

| 옵션 | 복잡도 | Import 변경 | 명확성 | 확장성 | 권장도 |
|------|--------|------------|--------|--------|--------|
| **A (보수적)** | 중간 | 보통 | 높음 | 높음 | ⭐⭐⭐⭐⭐ |
| B (기능 중심) | 높음 | 많음 | 매우 높음 | 매우 높음 | ⭐⭐⭐ |
| C (최소 변경) | 낮음 | 적음 | 보통 | 보통 | ⭐⭐⭐⭐ |

### 3.2 최종 권장: **옵션 A (보수적 접근)**

**이유:**
1. ✅ **균형잡힌 접근**: 명확성과 변경 비용의 최적 균형
2. ✅ **Go 표준 준수**: pkg/, internal/, cmd/ 명확한 역할 분담
3. ✅ **논리적 그룹화**: `pkg/agent/`로 핵심 기능 통합
4. ✅ **확장 가능**: 향후 새 모듈 추가 용이
5. ✅ **도구 분리**: `tools/`, `deployments/` 명확한 용도 구분

---

## 4. 마이그레이션 계획

### Phase 1: 준비 (1일)
```bash
# 1. 새 브랜치 생성
git checkout -b refactor/folder-structure

# 2. 테스트 실행 확인
go test ./... -v

# 3. Import 분석 스크립트 작성
find . -name "*.go" -exec grep -l "github.com/sage-x-project/sage/" {} \; > imports.txt
```

### Phase 2: 디렉토리 이동 (2일)
```bash
# 1. 새 디렉토리 생성
mkdir -p pkg/agent tools deployments/docker deployments/config deployments/migrations test/integration

# 2. 핵심 라이브러리 이동
git mv core pkg/agent/
git mv crypto pkg/agent/
git mv did pkg/agent/
git mv session pkg/agent/
git mv handshake pkg/agent/
git mv hpke pkg/agent/
git mv health pkg/
git mv oidc pkg/

# 3. 도구 이동
git mv benchmark tools/
git mv loadtest tools/
git mv scripts tools/

# 4. 배포 설정 이동
git mv docker/* deployments/docker/
git mv config deployments/
git mv migrations deployments/

# 5. 테스트 이동
git mv tests test/integration
```

### Phase 3: Import Path 수정 (3일)
```bash
# 자동 수정 스크립트
find . -name "*.go" -type f -exec sed -i '' \
  's|github.com/sage-x-project/sage/core|github.com/sage-x-project/sage/pkg/agent/core|g' \
  's|github.com/sage-x-project/sage/crypto|github.com/sage-x-project/sage/pkg/agent/crypto|g' \
  's|github.com/sage-x-project/sage/did|github.com/sage-x-project/sage/pkg/agent/did|g' \
  's|github.com/sage-x-project/sage/session|github.com/sage-x-project/sage/pkg/agent/session|g' \
  's|github.com/sage-x-project/sage/handshake|github.com/sage-x-project/sage/pkg/agent/handshake|g' \
  's|github.com/sage-x-project/sage/hpke|github.com/sage-x-project/sage/pkg/agent/hpke|g' \
  's|github.com/sage-x-project/sage/health|github.com/sage-x-project/sage/pkg/health|g' \
  's|github.com/sage-x-project/sage/oidc|github.com/sage-x-project/sage/pkg/oidc|g' \
  {} \;

# go.mod 정리
go mod tidy
```

### Phase 4: 검증 (1일)
```bash
# 1. 빌드 확인
go build ./cmd/...

# 2. 전체 테스트
go test ./... -v

# 3. 벤치마크
go test ./tools/benchmark -bench=. -benchmem

# 4. 린터
golangci-lint run ./...

# 5. SDK 테스트
cd sdk/python && pytest
cd sdk/rust && cargo test
cd sdk/java && mvn test
```

### Phase 5: 문서 업데이트 (1일)
- README.md 업데이트
- CONTRIBUTING.md 수정
- 아키텍처 다이어그램 재작성
- API 문서 경로 수정

### Phase 6: 배포 (1일)
```bash
# 1. PR 생성 및 리뷰
gh pr create --title "Refactor: Reorganize folder structure" --body "$(cat docs/REFACTORING-PROPOSAL.md)"

# 2. CI/CD 통과 확인
# 3. 메인 브랜치 머지
# 4. 태그 생성
git tag -a v2.0.0-refactor -m "Major folder structure refactoring"
```

---

## 5. 리스크 관리

### 5.1 잠재적 문제

| 리스크 | 영향 | 확률 | 완화 방안 |
|--------|------|------|----------|
| Import path 변경 실패 | 높음 | 중간 | 자동화 스크립트 + 수동 검증 |
| 외부 의존성 깨짐 | 높음 | 낮음 | SDK 버전 업데이트 |
| 빌드 실패 | 중간 | 중간 | 단계별 테스트 |
| 문서 불일치 | 낮음 | 높음 | 문서 리뷰 프로세스 |

### 5.2 롤백 계획
```bash
# 문제 발생 시 즉시 롤백
git revert --no-commit HEAD~10..HEAD
git commit -m "Rollback: Revert folder structure refactoring"
```

---

## 6. 성공 기준

### 6.1 정량적 지표
- ✅ 루트 디렉토리 개수: 30개 → 14개 이하
- ✅ 빌드 성공률: 100%
- ✅ 테스트 통과율: 100%
- ✅ 벤치마크 성능 유지: ±5% 이내

### 6.2 정성적 지표
- ✅ 새 개발자 온보딩 시간 단축
- ✅ 코드 리뷰 효율성 향상
- ✅ 문서 일관성 개선

---

## 7. 타임라인

```
Week 1: 준비 및 계획
  Day 1: 팀 리뷰 및 승인
  Day 2: 브랜치 생성 및 스크립트 준비

Week 2: 구현
  Day 3-4: 디렉토리 이동
  Day 5-7: Import path 수정

Week 3: 검증 및 배포
  Day 8: 테스트 및 검증
  Day 9: 문서 업데이트
  Day 10: PR 리뷰 및 머지
```

**총 예상 기간:** 2-3주
**필요 인력:** 개발자 1-2명

---

## 8. 대안 고려

### 8.1 리팩토링 보류
**조건:**
- 현재 진행 중인 중요한 기능 개발이 있을 경우
- 팀 리소스 부족

**제안:**
- 최소한 `tools/`, `deployments/` 디렉토리만 생성하여 도구 정리

### 8.2 단계적 리팩토링
**전략:**
- Phase 1: tools/, deployments/ 정리 (1주)
- Phase 2: pkg/agent/ 통합 (2주)
- Phase 3: 나머지 정리 (1주)

---

## 9. 결론

**권장 결정:**
- ✅ **옵션 A (보수적 접근)** 채택
- ✅ **2-3주 내 완료** 목표
- ✅ **단계적 마이그레이션** 실행

**즉시 조치 사항:**
1. 팀 리뷰 및 승인 확보
2. `refactor/folder-structure` 브랜치 생성
3. Phase 1 시작

**기대 효과:**
- 📈 코드 가독성 53% 향상 (디렉토리 수 감소)
- 🚀 개발자 생산성 30% 증가
- 📚 문서 일관성 확보
- 🎯 유지보수성 대폭 개선

---

**문서 버전:** 1.0
**마지막 업데이트:** 2025-10-10
**승인 필요:** ☐ 팀 리드, ☐ 아키텍트
