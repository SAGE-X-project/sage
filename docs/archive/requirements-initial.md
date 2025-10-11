# SAGE 요구사항 명세서

## 목차

- [1. 기능 요구사항](#1-기능-요구사항)
- [2. 비기능 요구사항](#2-비기능-요구사항)
- [3. 사용자 시나리오](#3-사용자-시나리오)

## 1. 기능 요구사항

### FR-01: Agent Gateway
**목적**: A2A·MCP 호출을 수신하여 검증·포워딩하는 단일 진입점 제공

**상세 요구사항**:
- HTTP/2 TLS(h2c 옵션) 기반 REST/JSON 엔드포인트 제공
  - `/invoke`: 에이전트 간 메시지 전달
  - `/tools/{id}`: MCP 도구 호출
- 기본 Rate-Limit: 1,000 req/agent/min (정책 엔진에서 조정 가능)
- 내부 서비스(Policy, Resolver) 호출은 gRPC 사용

**수용 기준**:
- AC-01: TLS 및 mTLS handshake 실패 시 HTTP 495 반환
- AC-02: 지원하지 않는 Path 요청 시 404 JSON 오류 반환
- AC-03: RFC 9421 검증 성공 후에만 다음 단계(Policy)로 전달

**우선순위**: 필수 ()

### FR-02: RFC 9421 Partial Signature
**목적**: HTTP 헤더 일부만 선택 서명·검증하여 전송 구간 변조 방지

**상세 요구사항**:
- 서명 알고리즘: `ed25519` 기본, `secp256k1`, `p-256` 옵션
- 헤더 선택 규칙: `(request-target) host date content-digest` 고정 필수
- Signer 키 검색: DID 문서의 `verificationMethod` 지정 키 사용

**수용 기준**:
- AC-01: 서명·검증 실패 시 401 `invalid_signature` 코드 반환
- AC-02: 성공 시 Gateway Context에 `X-Sig-Verified: true` 주입

**우선순위**: 필수 ()

### FR-03: 블록체인 DID 해석
**목적**: 에이전트가 제시한 DID → PublicKey·메타데이터 조회

**상세 요구사항**:
- 지원 Method: `did:key`, `did:ethr`, `did:sol` v0.1
- 캐싱: 5분 LRU 인메모리 + Redis(선택)
- DID Document 파싱 및 검증

**수용 기준**:
- AC-01: DID 문서 404 → Gateway 401 반환
- AC-02: 키 파싱 오류 → `invalid_did_doc` 로그

**우선순위**: 필수 ()

### FR-04: Rust Crypto 핵심 모듈화
**목적**: 서명 Canonicalization·Verify/Sign 로직을 memory-safe Rust로 통합

**상세 요구사항**:
- Cargo crate `libsage_crypto` (cdylib + wasm32)
- Core 함수: `sign`, `verify`, `hash` (Ed25519, SHA-256)
- FFI 및 WASM 바인딩 제공

**수용 기준**:
- AC-01: 벤치마크 `verify` 50 µs 이하
- AC-02: 패닉 발생 시 FFI error code 반환

**우선순위**: 필수 ()

### FR-05: Policy Engine (Access Control)
**목적**: `allowed_agents`, Rate-Limit, RBAC 정책으로 호출권 제한

**상세 요구사항**:
- 정책 DSL (YAML) 지원
- PDP(Policy Decision) → PEP(Gateway 미들웨어) 구조
- 동적 정책 업데이트 지원

**수용 기준**:
- AC-01: 정책 위반 시 403 `policy_denied` 반환
- AC-02: Rate-Limit 초과 시 429 반환

**우선순위**: 중요 ()

### FR-06: Audit Logging & Event Sourcing
**목적**: 모든 인증·정책 판정·도구 호출을 불변 로그로 기록

**상세 요구사항**:
- 구조적 JSON 로그 (1 레코드/호출)
- 옵션: IPFS / AWS QLDB로 해시 체인 저장
- SHA-256 체인 무결성 검증

**수용 기준**:
- AC-01: 로그 레코드 누락률 0%
- AC-02: SHA-256 체인 무결성 주기적 검증(1h)

**우선순위**: 중요 ()

## 2. 비기능 요구사항

### NFR-01: 성능
**요구사항**:
- 메시지 왕복 지연 시간(End-to-End Latency)
  - 이상적: 200ms 이내
  - 기준선: 400ms 이내
- 처리량: 초당 1,000건 이상의 요청 처리
- 리소스 사용률: 일반 부하에서 CPU 70% 이하, 메모리 500MB 이하

**검증 방안**:
- 부하 테스트(Load Test), 응답시간 p95, p99 측정
- DID 조회 캐싱 전후 성능 비교
- 모니터링 시스템 구성 및 지연 경보 설정

### NFR-02: 확장성
**요구사항**:
- 1,000개 이상의 에이전트 동시 연결 지원
- 인스턴스 수 증가 시 처리량 선형 증가
- DID 등록 수 만 건 이상에서도 성능 유지

**검증 방안**:
- 가상 에이전트 수 증가 시 처리량 변화 측정
- Auto Scaling 시나리오 실험(Kubernetes)
- 상태 저장 요소의 외부화 점검

### NFR-03: 보안
**요구사항**:
- 모든 메시지 TLS 암호화 및 RFC 9421 서명 적용
- DID 기반 접근 제어 → 비인가 에이전트 요청 100% 차단
- 민감 데이터(개인키 등) 저장 금지
- 치명적 보안 취약점(Critical CVE) 0건

**검증 방안**:
- 정기 보안 테스트(SAST, DAST, 펜테스트)
- 위협 모델링 및 대응책 문서화
- 서명 위조 시도 감지 및 로그 확인

### NFR-04: 가용성
**요구사항**:
- 가용성 99.9% 이상 (연간 최대 9시간 이하 중단)
- 단일 인스턴스 장애 시 자동 Failover
- 장애 복구 시간(RTO) 30초 이내
- 메시지 손실 없이 정확히 한 번 처리 보장

**검증 방안**:
- Chaos Testing: 컴포넌트 중단 시나리오
- 실제 장애 복구 시간 측정
- 메시지 ID 추적을 통한 손실 여부 확인

### NFR-05: 유지보수성
**요구사항**:
- 코드 커버리지 80% 이상
- Lint 및 정적 분석 도구 경고 0건
- 신규 기여자가 1일 이내 구조 이해 가능
- 중대 오류 발생 시 1일 이내 패치 배포

**검증 방안**:
- 다중 언어 바인딩 유지 테스트
- CI에서 lint 도구 통합 실행
- 문서 최신성 감사

### NFR-06: 가시성
**요구사항**:
- 주요 이벤트 로그 기록율 100%
- 메트릭 노출: 처리 시간(p95), TPS, 에러율, DID 조회 시간
- 요청별 Trace ID로 전 구간 추적 가능
- 지연 발생 시 5분 이내 경보 발생

**검증 방안**:
- 서명 검증 실패 시 로그 포맷 검사
- Prometheus/Grafana 대시보드 검토
- OpenTelemetry 통합 테스트

### NFR-07: 이식성
**요구사항**:
- Linux, Windows, macOS에서 Rust 라이브러리 컴파일 가능
- Docker 이미지 크기 ≤ 500MB, 부팅 시간 ≤ 10초
- FFI (Go), WASM (TS) 바인딩 제공
- 환경 구성은 모두 환경변수 또는 설정파일로 관리

**검증 방안**:
- CI에서 멀티 플랫폼 빌드 테스트
- Docker multi-arch 빌드 검증
- 설정값 변경 테스트

### NFR-08: 준수성
**요구사항**:
- W3C DID Core 규격 완전 준수
- RFC 9421 서명 방식 완전 호환
- 타 DIDComm 시스템과 상호운용 가능
- GDPR 준수 (개인정보 무저장 또는 삭제 API 제공)

**검증 방안**:
- DID 테스트 벡터 통과 확인
- 타 벤더와의 상호운용 테스트
- 개인정보 처리 흐름도 점검

## 3. 사용자 시나리오

### 시나리오 1: Agent 간 직접 통신

1. **DID 등록**
   - Agent A와 B는 각각 블록체인에 DID와 공개키를 등록
   - 트랜잭션 서명은 각 Agent가 직접 수행

2. **메시지 전송**
   - Agent A는 메시지에 RFC 9421 서명을 포함하여 B에게 전송
   - 서명에는 메시지 헤더와 본문의 해시가 포함됨

3. **메시지 검증**
   - Agent B는 A의 DID를 블록체인에서 조회
   - 조회된 공개키로 서명을 검증

4. **응답 처리**
   - B는 응답 메시지에 자신의 서명을 포함하여 A에게 전송
   - A는 동일한 방식으로 B의 서명을 검증

### 시나리오 2: Gateway를 통한 통신

1. **Gateway 경유 요청**
   - Agent A는 Gateway를 통해 B에게 메시지 전송
   - Gateway는 A의 서명을 검증하고 정책을 확인

2. **정책 기반 라우팅**
   - Gateway는 접근 제어 정책에 따라 요청을 필터링
   - Rate limiting 및 RBAC 규칙 적용

3. **감사 로그**
   - 모든 통신 내역이 Gateway에 기록됨
   - 블록체인 또는 불변 저장소에 로그 해시 저장

### 시나리오 3: 오류 처리

1. **서명 검증 실패**
   - 잘못된 서명이 포함된 메시지 수신
   - 401 Unauthorized 오류 반환 및 로그 기록

2. **DID 조회 실패**
   - 블록체인에 등록되지 않은 DID
   - 422 Unprocessable Entity 반환

3. **정책 위반**
   - 비허가 Agent의 접근 시도
   - 403 Forbidden 반환 및 보안 이벤트 기록

## 우선순위 정의

| 등급 | 정의 | 항목 |
|------|------|------|
|  | MVP 필수 | FR-01, FR-02 |
|  | MVP 범위 | FR-03, FR-04 |
|  | 후속 릴리즈 | FR-05, FR-06 |

## 적용 방법

1. **Issue Template**
   ```yaml
   - id: FR-02-AC-01
     description: "RFC 9421 검증 실패 시 401 반환"
     tests:
       - "서명 오류 시 status==401 확인"
   ```

2. **Sprint Planning**: FR → Epic → User Story → Task
3. **Definition of Done**:
   - 모든 AC 통과
   - 코드 리뷰 2인 이상
   - 단위 테스트 커버리지 ≥ 80%
   - CI 경고 0건