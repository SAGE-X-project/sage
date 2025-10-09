# 🔴 모니터링 즉시 조치 사항

> **작성일:** 2025-10-10
> **우선순위:** Critical
> **예상 시간:** 90분 (1.5시간)
> **관련 문서:** `docs/MONITORING-REVIEW.md`

---

## 📊 현황 요약

### ✅ 이미 구현된 사항 (70%)
- Prometheus 메트릭 정의 완료 (`internal/metrics/`)
- 메트릭 서버 구현 (`/metrics` 엔드포인트)
- Grafana 대시보드 (`docker/grafana/dashboards/sage-overview.json`)
- Docker Compose 설정 (Prometheus, Grafana)

### ❌ 즉시 해결 필요 (30%)
1. **Prometheus 설정 불일치** - 존재하지 않는 엔드포인트 스크랩 시도
2. **메트릭 미통합** - 정의된 메트릭이 실제 코드에서 호출되지 않음
3. **검증 부재** - 메트릭 동작 확인 필요

---

## 🎯 즉시 조치 1: Prometheus 설정 수정

**예상 시간:** 15분
**우선순위:** 🔴 Critical
**종속성:** 없음

### 문제점

Prometheus가 미구현된 엔드포인트를 스크랩하려고 시도:

```yaml
# docker/prometheus/prometheus.yml (라인 72-100)
- job_name: 'sage-sessions'
  metrics_path: '/metrics/sessions'  # ❌ 미구현

- job_name: 'sage-handshakes'
  metrics_path: '/metrics/handshakes'  # ❌ 미구현

- job_name: 'sage-crypto'
  metrics_path: '/metrics/crypto'  # ❌ 미구현
```

**실제 구현:** `/metrics` 엔드포인트만 존재 (`internal/metrics/server.go:38`)

### 해결 방법

**파일:** `docker/prometheus/prometheus.yml`

**작업:**
```bash
# 1. 파일 열기
vim docker/prometheus/prometheus.yml

# 2. 라인 72-100 삭제
# 3개의 개별 job (sage-sessions, sage-handshakes, sage-crypto) 제거

# 3. 라인 26-39는 유지 (sage-backend job)
# 이 job이 /metrics 엔드포인트를 올바르게 스크랩함
```

**최종 설정 (유지할 부분):**
```yaml
- job_name: 'sage-backend'
  scrape_interval: 10s
  metrics_path: '/metrics'  # ✅ 실제 구현됨
  static_configs:
    - targets:
        - 'sage-backend:9090'
      labels:
        service: 'sage-backend'
        component: 'core'
```

### 검증

```bash
# Prometheus 재시작
docker-compose restart prometheus

# 타겟 확인 (브라우저)
open http://localhost:9091/targets
# sage-backend가 UP 상태여야 함

# 로그 확인
docker-compose logs prometheus | grep -i error
# 에러 없어야 함
```

**체크리스트:**
- [ ] `docker/prometheus/prometheus.yml` 파일 편집
- [ ] 라인 72-100 삭제 완료
- [ ] Prometheus 재시작 완료
- [ ] `/targets` 페이지에서 UP 상태 확인

---

## 🎯 즉시 조치 2: 핸드셰이크 메트릭 통합

**예상 시간:** 30분
**우선순위:** 🔴 Critical
**종속성:** 조치 1 완료 후

### 통합 위치

- `handshake/client.go`
- `handshake/server.go`
- `hpke/client.go`
- `hpke/server.go`

### 코드 예시

#### handshake/client.go

```go
package handshake

import (
    "time"
    "github.com/sage-x-project/sage/internal/metrics"
)

func (c *Client) InitiateHandshake() error {
    // 시작 시간 기록
    start := time.Now()

    // 함수 종료 시 duration 메트릭 기록
    defer func() {
        metrics.HandshakeDuration.WithLabelValues("init").Observe(
            time.Since(start).Seconds(),
        )
    }()

    // 핸드셰이크 시작 카운터 증가
    metrics.HandshakesInitiated.WithLabelValues("client").Inc()

    // 기존 로직...
    err := c.performHandshake()

    if err != nil {
        // 실패 카운터 증가
        metrics.HandshakesFailed.WithLabelValues("network_error").Inc()
        return err
    }

    // 성공 카운터 증가
    metrics.HandshakesCompleted.WithLabelValues("success").Inc()
    return nil
}
```

#### handshake/server.go

```go
func (s *Server) AcceptHandshake() error {
    start := time.Now()
    defer metrics.HandshakeDuration.WithLabelValues("accept").Observe(
        time.Since(start).Seconds(),
    )

    metrics.HandshakesInitiated.WithLabelValues("server").Inc()

    // 기존 로직...

    if err != nil {
        metrics.HandshakesFailed.WithLabelValues("validation_error").Inc()
        return err
    }

    metrics.HandshakesCompleted.WithLabelValues("success").Inc()
    return nil
}
```

### 검증

```bash
# 테스트 실행
go test ./handshake/... -v

# 메트릭 확인 (테스트 후)
curl -s localhost:9090/metrics | grep handshakes

# 예상 출력:
# sage_handshakes_initiated_total{role="client"} 1
# sage_handshakes_completed_total{status="success"} 1
# sage_handshakes_duration_seconds_bucket{stage="init",le="+Inf"} 1
```

**체크리스트:**
- [ ] `handshake/client.go` - InitiateHandshake() 수정
- [ ] `handshake/server.go` - AcceptHandshake() 수정
- [ ] `hpke/client.go` - Initialize() 수정
- [ ] `hpke/server.go` - ProcessInitialize() 수정
- [ ] 테스트 실행 및 통과 확인
- [ ] 메트릭 출력 확인

---

## 🎯 즉시 조치 3: 세션 메트릭 통합

**예상 시간:** 30분
**우선순위:** 🔴 Critical
**종속성:** 조치 1 완료 후

### 통합 위치

- `session/manager.go`
- `session/session.go`

### 코드 예시

#### session/manager.go

```go
package session

import (
    "time"
    "github.com/sage-x-project/sage/internal/metrics"
)

func (m *Manager) CreateSession(id string, sharedSecret []byte) (*Session, error) {
    start := time.Now()
    defer metrics.SessionDuration.WithLabelValues("create").Observe(
        time.Since(start).Seconds(),
    )

    sess, err := m.createSession(id, sharedSecret)
    if err != nil {
        metrics.SessionsCreated.WithLabelValues("failure").Inc()
        return nil, err
    }

    // 성공 카운터 및 활성 세션 게이지 증가
    metrics.SessionsCreated.WithLabelValues("success").Inc()
    metrics.SessionsActive.Inc()

    return sess, nil
}

func (m *Manager) cleanup() {
    expired := m.removeExpiredSessions()
    if expired > 0 {
        // 만료된 세션 카운터 증가
        metrics.SessionsExpired.Add(float64(expired))
        // 활성 세션 게이지 감소
        metrics.SessionsActive.Sub(float64(expired))
    }
}

func (m *Manager) CloseSession(id string) error {
    if err := m.closeSession(id); err != nil {
        return err
    }

    // 세션 닫기 카운터 증가
    metrics.SessionsClosed.Inc()
    metrics.SessionsActive.Dec()
    return nil
}
```

#### session/session.go

```go
func (s *Session) Encrypt(plaintext []byte) ([]byte, error) {
    start := time.Now()
    defer metrics.SessionDuration.WithLabelValues("encrypt").Observe(
        time.Since(start).Seconds(),
    )

    // 메시지 크기 기록
    metrics.SessionMessageSize.WithLabelValues("outbound").Observe(
        float64(len(plaintext)),
    )

    // 기존 암호화 로직...
    ciphertext, err := s.encryptData(plaintext)
    if err != nil {
        return nil, err
    }

    return ciphertext, nil
}

func (s *Session) Decrypt(ciphertext []byte) ([]byte, error) {
    start := time.Now()
    defer metrics.SessionDuration.WithLabelValues("decrypt").Observe(
        time.Since(start).Seconds(),
    )

    metrics.SessionMessageSize.WithLabelValues("inbound").Observe(
        float64(len(ciphertext)),
    )

    // 기존 복호화 로직...
    plaintext, err := s.decryptData(ciphertext)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}
```

### 검증

```bash
# 테스트 실행
go test ./session/... -v

# 메트릭 확인
curl -s localhost:9090/metrics | grep sessions

# 예상 출력:
# sage_sessions_created_total{status="success"} 5
# sage_sessions_active 2
# sage_sessions_expired_total 3
# sage_sessions_duration_seconds_bucket{operation="create",le="+Inf"} 5
```

**체크리스트:**
- [ ] `session/manager.go` - CreateSession() 수정
- [ ] `session/manager.go` - cleanup() 수정
- [ ] `session/manager.go` - CloseSession() 수정
- [ ] `session/session.go` - Encrypt() 수정
- [ ] `session/session.go` - Decrypt() 수정
- [ ] 테스트 실행 및 통과 확인
- [ ] 메트릭 출력 확인

---

## 🎯 즉시 조치 4: 통합 검증

**예상 시간:** 15분
**우선순위:** 🔴 Critical
**종속성:** 조치 2, 3 완료 후

### 검증 단계

#### 1. Docker 환경 시작

```bash
# 전체 스택 시작
docker-compose up -d prometheus grafana sage-backend

# 상태 확인
docker-compose ps

# 로그 확인 (에러 없어야 함)
docker-compose logs prometheus | tail -20
docker-compose logs grafana | tail -20
```

#### 2. Prometheus 타겟 확인

```bash
# 브라우저 열기
open http://localhost:9091/targets

# 확인 사항:
# - sage-backend (1/1 up) - 정상
# - 에러 메시지 없음
# - Last Scrape 시간이 최근
```

#### 3. 메트릭 직접 조회

```bash
# sage 메트릭 존재 확인
curl -s localhost:9090/metrics | grep -E "sage_(handshakes|sessions|crypto)" | head -20

# 핸드셰이크 메트릭 확인
curl -s localhost:9090/metrics | grep handshakes_

# 세션 메트릭 확인
curl -s localhost:9090/metrics | grep sessions_
```

#### 4. Prometheus 쿼리 테스트

```bash
# 활성 세션 수 조회
curl -G 'http://localhost:9091/api/v1/query' \
  --data-urlencode 'query=sage_sessions_active' | jq

# 핸드셰이크 성공률 조회
curl -G 'http://localhost:9091/api/v1/query' \
  --data-urlencode 'query=rate(sage_handshakes_completed_total{status="success"}[5m])' | jq
```

#### 5. Grafana 대시보드 확인

```bash
# Grafana 열기
open http://localhost:3000

# 로그인: admin / admin
# Dashboards → SAGE System Overview

# 확인 사항:
# - Active Sessions 패널에 데이터 표시
# - Handshake Success Rate 패널에 그래프 표시
# - No data 메시지가 없어야 함
```

#### 6. 실제 동작 테스트

```bash
# 핸드셰이크 테스트 실행
go test ./handshake/... -v -run TestHandshake

# 메트릭 증가 확인
curl -s localhost:9090/metrics | grep handshakes_completed_total

# 세션 테스트 실행
go test ./session/... -v -run TestSession

# 메트릭 증가 확인
curl -s localhost:9090/metrics | grep sessions_created_total
```

### 성공 기준

✅ 다음 조건이 모두 충족되어야 함:

1. Prometheus `/targets` 페이지에서 sage-backend가 UP 상태
2. `/metrics` 엔드포인트에서 `sage_*` 메트릭 출력
3. Grafana 대시보드에 실시간 데이터 표시
4. 테스트 실행 후 메트릭 카운터 증가 확인
5. 에러 로그 없음

**체크리스트:**
- [ ] Docker 환경 시작 완료
- [ ] Prometheus 타겟 UP 확인
- [ ] 메트릭 엔드포인트 응답 확인
- [ ] Prometheus 쿼리 성공
- [ ] Grafana 대시보드 데이터 표시
- [ ] 테스트 실행 후 메트릭 증가 확인

---

## 📋 전체 체크리스트 (Copy & Paste)

```markdown
## 모니터링 즉시 조치 체크리스트

### Phase 1: Prometheus 설정 수정 (15분)
- [ ] `docker/prometheus/prometheus.yml` 파일 열기
- [ ] 라인 72-100 삭제 (sage-sessions, sage-handshakes, sage-crypto jobs)
- [ ] 라인 26-39 확인 (sage-backend job 유지)
- [ ] 파일 저장
- [ ] Prometheus 재시작: `docker-compose restart prometheus`
- [ ] 타겟 확인: http://localhost:9091/targets
- [ ] sage-backend UP 상태 확인

### Phase 2: 핸드셰이크 메트릭 통합 (30분)
- [ ] `handshake/client.go` 수정
  - [ ] `internal/metrics` import 추가
  - [ ] `HandshakesInitiated.Inc()` 추가
  - [ ] `HandshakeDuration.Observe()` 추가
  - [ ] `HandshakesCompleted.Inc()` 추가
  - [ ] 에러 처리에 `HandshakesFailed.Inc()` 추가
- [ ] `handshake/server.go` 수정 (동일 패턴)
- [ ] `hpke/client.go` 수정 (선택적)
- [ ] `hpke/server.go` 수정 (선택적)
- [ ] 테스트 실행: `go test ./handshake/... -v`
- [ ] 테스트 통과 확인

### Phase 3: 세션 메트릭 통합 (30분)
- [ ] `session/manager.go` 수정
  - [ ] `internal/metrics` import 추가
  - [ ] `CreateSession()` - SessionsCreated, SessionsActive
  - [ ] `cleanup()` - SessionsExpired
  - [ ] `CloseSession()` - SessionsClosed
- [ ] `session/session.go` 수정
  - [ ] `Encrypt()` - SessionDuration, MessageSize
  - [ ] `Decrypt()` - SessionDuration, MessageSize
- [ ] 테스트 실행: `go test ./session/... -v`
- [ ] 테스트 통과 확인

### Phase 4: 통합 검증 (15분)
- [ ] Docker 시작: `docker-compose up -d`
- [ ] Prometheus 타겟 확인: http://localhost:9091/targets
- [ ] 메트릭 확인: `curl localhost:9090/metrics | grep sage_`
- [ ] Prometheus 쿼리 테스트
- [ ] Grafana 대시보드 확인: http://localhost:3000
- [ ] 대시보드에 데이터 표시 확인
- [ ] 핸드셰이크 테스트 후 메트릭 증가 확인
- [ ] 세션 테스트 후 메트릭 증가 확인
```

---

## 🚀 후속 작업 (선택적)

### RFC 9421 서명 메트릭 추가 (20분)

**파일:** `core/rfc9421/verifier_http.go`

```go
func (v *HTTPVerifier) VerifyRequest(r *http.Request, publicKey interface{}, opts *HTTPVerificationOptions) error {
    start := time.Now()
    defer metrics.CryptoOperationDuration.WithLabelValues(
        "verify", "ed25519",
    ).Observe(time.Since(start).Seconds())

    metrics.CryptoOperations.WithLabelValues("verify", "ed25519").Inc()

    // 기존 검증 로직...

    if err != nil {
        metrics.CryptoErrors.WithLabelValues("verify").Inc()
        return err
    }
    return nil
}
```

### DID 해결 메트릭 추가 (20분)

**파일:** `did/resolver.go`

```go
func (r *MultiChainResolver) Resolve(did string) (*Document, error) {
    start := time.Now()
    chain := extractChain(did)

    defer metrics.GetGlobalCollector().RecordDIDResolution(
        false, // cached 여부
        time.Since(start),
    )

    // 기존 해결 로직...
}
```

### Replay 공격 메트릭 추가 (15분)

**파일:** `core/rfc9421/nonce.go`

```go
func (nc *NonceCache) ValidateNonce(nonce string) bool {
    if nc.IsSeen(nonce) {
        metrics.ReplayAttacksDetected.Inc()
        metrics.NonceValidations.WithLabelValues("replay_detected").Inc()
        return false
    }

    metrics.NonceValidations.WithLabelValues("valid").Inc()
    return true
}
```

---

## 📊 예상 결과

### 완료 후 Grafana 대시보드

**Active Sessions 패널:**
- 실시간으로 활성 세션 수 표시
- 세션 생성/만료에 따라 그래프 변동

**Handshake Success Rate 패널:**
- 성공/실패 핸드셰이크 비율 표시
- 5분 단위 평균 성공률

**Signature Verification Latency 패널:**
- p95, p99 레이턴시 표시
- 성능 병목 지점 파악 가능

### 메트릭 쿼리 예시

```promql
# 활성 세션 수
sage_sessions_active

# 핸드셰이크 성공률 (5분)
rate(sage_handshakes_completed_total{status="success"}[5m])
/
rate(sage_handshakes_initiated_total[5m]) * 100

# 세션 생성 레이턴시 p95
histogram_quantile(0.95, rate(sage_sessions_duration_seconds_bucket{operation="create"}[5m]))

# Replay 공격 감지율
rate(sage_messages_replay_attacks_detected_total[1m])
```

---

## 🆘 문제 해결

### Prometheus가 메트릭을 수집하지 않음

```bash
# 1. sage-backend 포트 확인
docker-compose ps sage-backend
# 9090 포트가 매핑되어 있는지 확인

# 2. 메트릭 엔드포인트 직접 확인
curl http://localhost:9090/metrics
# sage_ 메트릭이 출력되는지 확인

# 3. Prometheus 로그 확인
docker-compose logs prometheus | grep sage-backend
```

### Grafana 대시보드에 데이터가 없음

```bash
# 1. Datasource 확인
# Grafana → Configuration → Data Sources
# Prometheus URL이 http://prometheus:9090인지 확인

# 2. 쿼리 테스트
# Grafana → Explore
# Query: sage_sessions_active
# Run Query 클릭

# 3. 시간 범위 확인
# 대시보드 우측 상단 시간 범위를 "Last 5 minutes"로 설정
```

### 메트릭이 증가하지 않음

```bash
# 1. 코드에 메트릭 호출이 있는지 확인
grep -r "metrics\\.Handshakes" handshake/
grep -r "metrics\\.Sessions" session/

# 2. import 확인
# internal/metrics 패키지가 import되었는지 확인

# 3. 테스트 실행 확인
go test ./handshake/... -v
# 테스트가 실제로 실행되는지 확인
```

---

## 📞 지원

**문제 발생 시:**
1. 체크리스트의 각 항목을 순서대로 확인
2. 로그 확인: `docker-compose logs [service]`
3. 관련 문서: `docs/MONITORING-REVIEW.md` 섹션 5 참조

**완료 후:**
- [ ] 이 문서의 내용을 `docs/MONITORING-REVIEW.md`에 반영
- [ ] 작업 시간 기록 (실제 소요 시간)
- [ ] 발견된 추가 이슈 문서화

---

**작성자:** SAGE Development Team
**최종 업데이트:** 2025-10-10
**다음 리뷰:** 메트릭 통합 완료 후
