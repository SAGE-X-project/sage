# 모니터링 및 관찰성 작업 검토

**최초 검토일:** 2025-10-08
**업데이트:** 2025-10-10 (실제 코드 검증 완료)
**문서:** `docs/REMAINING-TASKS-DETAILED.md` - Task 7 분석

---

## ⚠️ 중요 업데이트 (2025-10-10)

**검증 결과:** 문서 작성 이후 또는 발견하지 못한 구현이 존재합니다.

### 이미 구현된 사항:
- ✅ **Prometheus 메트릭 완전 구현** (`internal/metrics/` 패키지)
- ✅ **메트릭 서버 구현** (`/metrics` 엔드포인트)
- ✅ **Grafana 대시보드** (`docker/grafana/dashboards/sage-overview.json`)
- ✅ **Docker Compose 설정** (Prometheus, Grafana, Redis)
- ✅ **Zap 의존성 설치** (`go.uber.org/zap v1.21.0`)

### 실제 필요 작업:
- ❌ Prometheus 설정 수정 (엔드포인트 불일치)
- ❌ 메트릭을 실제 코드에 통합 (handshake, session, crypto)
- ❌ Zap 로거 적용 (설치되었지만 미사용)
- ❌ OpenTelemetry/Jaeger 추적 추가

**예상 작업 시간:** 3일 → **1-2일로 단축 가능** (인프라 70% 완성)

---

## 1. 작업의 필요성 분석

### 1.1 현재 SAGE 프로젝트의 상태

#### 로깅 현황
- **비구조화된 로깅 사용**: 코드베이스 전체에서 `log.Print*`, `fmt.Print*` 사용
  - `log.*`: 32회 사용 (실제 측정값)
  - `fmt.*`: 304회 사용 (실제 측정값)
- **Zap 설치 완료**: `go.uber.org/zap v1.21.0` 의존성 존재, 하지만 미활용
- **문제점**:
  - 로그 파싱이 어려움 (텍스트 기반)
  - 문맥 정보 부족 (request ID, session ID 등)
  - 로그 레벨 구분 불가
  - 프로덕션 환경에서 디버깅 어려움
  - 로그 집계 및 검색 불가능

#### 메트릭 현황
- **Prometheus 메트릭 구현 완료**: `internal/metrics/` 패키지 (9개 파일)
  - ✅ `handshake.go` - 핸드셰이크 메트릭 (Initiated, Completed, Failed, Duration)
  - ✅ `session.go` - 세션 메트릭 (Created, Active, Expired, Duration)
  - ✅ `crypto.go` - 암호화 메트릭 (Operations, Errors, Duration)
  - ✅ `message.go` - 메시지 메트릭 (Processed, ReplayAttacks, Nonce)
  - ✅ `collector.go` - 메트릭 수집기 (Snapshot, Statistics)
  - ✅ `server.go` - HTTP 핸들러 (`/metrics` 엔드포인트)
  - ✅ `registry.go` - Prometheus 레지스트리

- **Grafana 대시보드 구현**: `docker/grafana/dashboards/sage-overview.json`
  - 7개 패널: Active Sessions, Handshake Success Rate, Signature Latency 등

- **Docker Compose 설정**: Prometheus, Grafana 서비스 구성 완료
  - Prometheus: `docker/prometheus/prometheus.yml`
  - Grafana: `docker/grafana/` (datasources, dashboards)

- **문제점**:
  - ⚠️ **Prometheus 설정과 코드 불일치**:
    - 설정: `/metrics/sessions`, `/metrics/handshakes`, `/metrics/crypto` (3개 개별 엔드포인트)
    - 실제: `/metrics` (단일 표준 엔드포인트만 구현)
  - ⚠️ **메트릭 코드 통합 부족**: 정의된 메트릭이 실제 비즈니스 로직에서 호출되지 않음
  - ⚠️ **health/server.go**만 메트릭 사용, 핵심 모듈(handshake, session)에서 미사용

#### 추적(Tracing) 현황
- **분산 추적 없음**: OpenTelemetry나 Jaeger 통합 없음
- **문제점**:
  - 요청 흐름 추적 불가 (handshake → session → encryption)
  - 성능 병목 지점 파악 어려움
  - 마이크로서비스 간 호출 관계 파악 불가
  - 디버깅 시 전체 트랜잭션 컨텍스트 부재

#### 알림(Alerting) 현황
- **알림 규칙 없음**: Prometheus 알림 규칙 미구성
- **문제점**:
  - 프로덕션 문제 사전 감지 불가
  - 수동 모니터링 필요
  - SLA 위반 감지 불가
  - 장애 대응 지연

---

## 2. 설계 방향의 적절성 평가

### 2.1 구조화된 로깅 (Zap)

#### 제안된 설계
```go
package logging

type Logger struct {
    zap *zap.Logger
}

func (l *Logger) WithContext(ctx context.Context) *Logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger
```

#### 평가: Yes **적절함**

**장점:**
1. **성능**: Zap은 Go에서 가장 빠른 로거 (zero allocation)
2. **구조화**: JSON 포맷으로 로그 집계 도구와 통합 용이
3. **문맥 전파**: `WithContext`로 request ID, session ID 자동 추가
4. **레벨 관리**: Debug/Info/Warn/Error 구분으로 프로덕션/개발 환경 분리

**실제 SAGE 코드와의 통합:**
- `tests/handshake/server/main.go`: 현재 21개 `log.*` 호출
  - 변환 후: 구조화된 필드로 session ID, agent DID, 암호화 상태 추가 가능
  - 예: `logger.Info("handshake completed", zap.String("sessionID", sid), zap.String("clientDID", did))`

- `examples/mcp-integration/*/main.go`: HTTP 핸들러에서 로깅
  - 변환 후: 미들웨어로 자동 request 로깅
  - 예: request ID, method, path, duration, status code 자동 기록

**개선 제안:**
1. **표준 필드 정의**: `pkg/logging/fields.go`에 SAGE 특화 필드 정의
   ```go
   const (
       FieldSessionID = "session_id"
       FieldClientDID = "client_did"
       FieldServerDID = "server_did"
       FieldKeyType   = "key_type"
       FieldOperation = "operation" // handshake, encrypt, decrypt, sign
   )
   ```

2. **에러 로깅 표준화**:
   ```go
   func (l *Logger) Error(msg string, err error, fields ...zap.Field) {
       l.zap.Error(msg, append(fields, zap.Error(err), zap.Stack("stacktrace"))...)
   }
   ```

---

### 2.2 분산 추적 (OpenTelemetry + Jaeger)

#### 제안된 설계
```go
package tracing

func SpanHandshake(ctx context.Context, clientDID, serverDID string) (context.Context, trace.Span)
func SpanEncryption(ctx context.Context, sessionID string) (context.Context, trace.Span)
func SpanSignature(ctx context.Context, keyType string) (context.Context, trace.Span)
```

#### 평가: Yes **적절하며 필수적**

**장점:**
1. **표준 기술**: OpenTelemetry는 CNCF 표준, 벤더 중립적
2. **전체 흐름 추적**: SAGE의 복잡한 흐름 시각화
   - Client: 키 생성 → 핸드셰이크 시작 → 서명 생성
   - Server: 핸드셰이크 검증 → 세션 생성 → 메시지 암호화/복호화
3. **성능 분석**: 각 단계 소요 시간 측정으로 병목 지점 파악

**SAGE 특화 스팬 설계:**
```go
// 핸드셰이크 전체 플로우
func TraceHandshake(ctx context.Context, clientDID, serverDID string) (context.Context, trace.Span) {
    ctx, span := tracer.Start(ctx, "sage.handshake")
    span.SetAttributes(
        attribute.String("sage.client_did", clientDID),
        attribute.String("sage.server_did", serverDID),
        attribute.String("sage.version", "1.0"),
    )
    return ctx, span
}

// 암호화 작업
func TraceEncryption(ctx context.Context, sessionID string, payloadSize int) (context.Context, trace.Span) {
    ctx, span := tracer.Start(ctx, "sage.session.encrypt")
    span.SetAttributes(
        attribute.String("sage.session_id", sessionID),
        attribute.Int("sage.payload_size", payloadSize),
    )
    return ctx, span
}

// RFC 9421 서명
func TraceSignature(ctx context.Context, keyType, algorithm string) (context.Context, trace.Span) {
    ctx, span := tracer.Start(ctx, "sage.signature.create")
    span.SetAttributes(
        attribute.String("sage.key_type", keyType),
        attribute.String("sage.algorithm", algorithm),
    )
    return ctx, span
}
```

**실제 코드 적용 예시** (`tests/handshake/server/main.go`):
```go
// 기존 코드
func handleProtected(w http.ResponseWriter, r *http.Request) {
    // 1. 서명 검증
    if err := verifier.VerifyRequest(r, pubKey, opts); err != nil {
        http.Error(w, "signature verify failed", http.StatusUnauthorized)
        return
    }

    // 2. 복호화
    plain, err := sess.Decrypt(cipherBody)
    // ...
}

// 추적 추가 후
func handleProtected(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracing.TraceRequest(r.Context(), "protected_endpoint")
    defer span.End()

    // 1. 서명 검증 추적
    ctx, verifySpan := tracing.TraceSignatureVerification(ctx, "ed25519")
    if err := verifier.VerifyRequest(r, pubKey, opts); err != nil {
        verifySpan.RecordError(err)
        verifySpan.End()
        http.Error(w, "signature verify failed", http.StatusUnauthorized)
        return
    }
    verifySpan.End()

    // 2. 복호화 추적
    ctx, decryptSpan := tracing.TraceDecryption(ctx, params.KeyID, len(cipherBody))
    plain, err := sess.Decrypt(cipherBody)
    if err != nil {
        decryptSpan.RecordError(err)
    }
    decryptSpan.End()
    // ...
}
```

**개선 제안:**
1. **자동 계측**: HTTP 미들웨어로 모든 요청 자동 추적
2. **스팬 이벤트**: 중요 체크포인트 기록
   ```go
   span.AddEvent("nonce_validated", trace.WithAttributes(
       attribute.String("nonce", nonce),
   ))
   ```

---

### 2.3 커스텀 메트릭 (Prometheus)

#### 제안된 설계
```go
var (
    HandshakesTotal = promauto.NewCounterVec(...)
    SessionsActive = promauto.NewGauge(...)
    HandshakeDuration = promauto.NewHistogram(...)
)
```

#### 평가: Yes **적절하나 확장 필요**

**현재 문제:**
- Prometheus 설정은 있으나 실제 메트릭 코드 없음
- 엔드포인트 `/metrics/sessions`, `/metrics/handshakes` 구현 필요

**제안된 메트릭의 적절성:**
1. **HandshakesTotal (Counter)**: Yes 필수
   - 레이블: `status` (success/error), `key_type` (ed25519/secp256k1)
   - 용도: 핸드셰이크 성공률, 에러율 추적

2. **SessionsActive (Gauge)**: Yes 필수
   - 용도: 현재 활성 세션 수 모니터링
   - 메모리 사용량 예측에 중요

3. **HandshakeDuration (Histogram)**: Yes 필수
   - 용도: 성능 SLA 측정 (p50, p95, p99)
   - 레이턴시 문제 감지

**추가 필요 메트릭:**

```go
// 세션 관련
SessionsExpired = promauto.NewCounter(
    prometheus.CounterOpts{
        Name: "sage_sessions_expired_total",
        Help: "Total number of expired sessions",
    },
)

SessionLifetime = promauto.NewHistogram(
    prometheus.HistogramOpts{
        Name: "sage_session_lifetime_seconds",
        Help: "Session lifetime in seconds",
        Buckets: []float64{60, 300, 600, 1800, 3600}, // 1m, 5m, 10m, 30m, 1h
    },
)

// 암호화 작업
EncryptionErrors = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "sage_encryption_errors_total",
        Help: "Total encryption errors",
    },
    []string{"operation", "error_type"}, // operation: encrypt/decrypt
)

MessageSize = promauto.NewHistogram(
    prometheus.HistogramOpts{
        Name: "sage_message_size_bytes",
        Help: "Message size in bytes",
        Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100B to 102KB
    },
)

// Nonce/Replay 방어
ReplayAttempts = promauto.NewCounter(
    prometheus.CounterOpts{
        Name: "sage_replay_attempts_total",
        Help: "Total replay attack attempts detected",
    },
)

// DID 해결
DIDResolutionDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "sage_did_resolution_duration_seconds",
        Help: "DID resolution duration",
    },
    []string{"chain"}, // ethereum, solana
)

DIDResolutionErrors = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "sage_did_resolution_errors_total",
        Help: "DID resolution errors",
    },
    []string{"chain", "error_type"},
)

// 블록체인 상호작용
BlockchainTransactions = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "sage_blockchain_transactions_total",
        Help: "Blockchain transactions",
    },
    []string{"chain", "operation", "status"}, // operation: register/update/revoke
)
```

**실제 코드 통합:**
```go
// session/manager.go 수정
func (m *Manager) Create(clientKey, serverKey interface{}) (*Session, error) {
    start := time.Now()
    defer func() {
        metrics.HandshakeDuration.Observe(time.Since(start).Seconds())
    }()

    sess, err := m.createSession(clientKey, serverKey)
    if err != nil {
        metrics.HandshakesTotal.WithLabelValues("error", "unknown").Inc()
        return nil, err
    }

    metrics.HandshakesTotal.WithLabelValues("success", sess.KeyType()).Inc()
    metrics.SessionsActive.Inc()
    return sess, nil
}

func (m *Manager) cleanup() {
    expired := m.removeExpiredSessions()
    metrics.SessionsExpired.Add(float64(expired))
    metrics.SessionsActive.Sub(float64(expired))
}
```

---

### 2.4 알림 규칙 (Prometheus Alerts)

#### 제안된 알림
1. **HighHandshakeErrorRate**: 5분간 에러율 > 10%
2. **SessionExpirationHigh**: 5분간 만료율 > 10/sec
3. **SlowHandshakes**: p95 > 1초
4. **HighMemoryUsage**: > 1GB

#### 평가: Warning **기본적이나 SAGE 특화 알림 추가 필요**

**적절한 알림:**
- Yes HighHandshakeErrorRate: 보안 공격 감지에 중요
- Yes SlowHandshakes: 성능 SLA 모니터링

**부족한 부분:**

```yaml
groups:
  - name: sage_security_alerts
    interval: 30s
    rules:
      # 보안 관련
      - alert: ReplayAttackDetected
        expr: rate(sage_replay_attempts_total[1m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Replay attack detected"
          description: "Replay attacks detected: {{ $value }} attempts/sec"

      - alert: SignatureVerificationFailureHigh
        expr: rate(sage_handshakes_total{status="error"}[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High signature verification failures"

      # DID 해결 문제
      - alert: DIDResolutionFailure
        expr: rate(sage_did_resolution_errors_total[5m]) > 0.5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "DID resolution failures"
          description: "Cannot resolve DIDs from blockchain"

      # 블록체인 연결
      - alert: BlockchainUnavailable
        expr: rate(sage_blockchain_transactions_total{status="error"}[5m]) > 0.8
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Blockchain unavailable"

  - name: sage_performance_alerts
    interval: 30s
    rules:
      # 세션 리소스
      - alert: SessionLeakSuspected
        expr: sage_sessions_active > 10000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Possible session leak"
          description: "Active sessions: {{ $value }}"

      - alert: EncryptionSlowdown
        expr: histogram_quantile(0.95, sage_encryption_duration_seconds) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Encryption slowdown detected"
          description: "P95 encryption time: {{ $value }}s"

  - name: sage_availability_alerts
    interval: 30s
    rules:
      # 서비스 가용성
      - alert: SAGEBackendDown
        expr: up{job="sage-backend"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "SAGE backend is down"

      - alert: HighErrorRate
        expr: rate(sage_handshakes_total{status="error"}[5m]) / rate(sage_handshakes_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate: {{ $value | humanizePercentage }}"
```

---

## 3. 설계 방향의 문제점 및 개선안

### 3.1 현재 설계의 문제점

#### 문제 1: HTTP 서버 메트릭 엔드포인트 미구현
- Prometheus 설정에는 `/metrics/sessions`, `/metrics/handshakes` 정의
- 실제 코드에는 이런 엔드포인트 없음
- **해결책**: 표준 `/metrics` 엔드포인트 하나로 통합, Prometheus 레이블로 구분

```go
// cmd/sage-server/main.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

func main() {
    http.Handle("/metrics", promhttp.Handler())
    // ...
}
```

#### 문제 2: 로깅과 추적의 통합 부족
- 로그와 트레이스가 분리되면 디버깅 어려움
- **해결책**: Trace ID를 로그에 자동 포함

```go
// pkg/logging/logger.go
func (l *Logger) WithTrace(ctx context.Context) *Logger {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        return l.WithFields(map[string]interface{}{
            "trace_id": span.SpanContext().TraceID().String(),
            "span_id":  span.SpanContext().SpanID().String(),
        })
    }
    return l
}

// 사용
logger.WithTrace(ctx).Info("handshake completed")
// 출력: {"level":"info","msg":"handshake completed","trace_id":"abc123","span_id":"def456"}
```

#### 문제 3: 메트릭 수집 누락 지점
현재 제안은 핸드셰이크와 세션에 집중, 하지만 중요한 지점 누락:
- RFC 9421 서명 생성/검증 시간
- DID 블록체인 해결 시간
- HPKE 키 유도 시간

**해결책**: 핵심 함수에 메트릭 추가
```go
// core/rfc9421/verifier_http.go
func (v *HTTPVerifier) VerifyRequest(r *http.Request, publicKey interface{}, opts *HTTPVerificationOptions) error {
    start := time.Now()
    defer func() {
        metrics.SignatureVerificationDuration.Observe(time.Since(start).Seconds())
    }()
    // 기존 로직
}
```

#### 문제 4: 환경별 설정 부재
- 로깅 레벨이 환경별로 달라야 함 (dev: debug, prod: info)
- 메트릭 샘플링 비율 조정 필요
- **해결책**: 설정 파일 통합

```yaml
# configs/production.yaml
logging:
  level: info
  format: json
  output: stdout

tracing:
  enabled: true
  sample_rate: 0.1  # 10% 샘플링으로 오버헤드 감소
  jaeger_endpoint: http://jaeger:14268/api/traces

metrics:
  enabled: true
  port: 9090
  path: /metrics

# configs/dev.yaml
logging:
  level: debug
  format: console  # 개발자 친화적 포맷

tracing:
  sample_rate: 1.0  # 전체 추적
```

---

### 3.2 아키텍처 개선안

#### 통합 관찰성 레이어

```
┌─────────────────────────────────────────────────────┐
│                  SAGE Application                    │
│                                                       │
│  ┌─────────────────────────────────────────────┐   │
│  │         Observability Middleware             │   │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────────┐  │   │
│  │  │ Logging │ │ Tracing │ │   Metrics    │  │   │
│  │  │  (Zap)  │ │ (OTEL)  │ │(Prometheus)  │  │   │
│  │  └────┬────┘ └────┬────┘ └──────┬───────┘  │   │
│  │       │           │              │          │   │
│  │       └───────────┴──────────────┘          │   │
│  │              Correlation                    │   │
│  └─────────────────────────────────────────────┘   │
│                                                       │
│  ┌─────────┐  ┌─────────┐  ┌─────────────────┐    │
│  │Handshake│  │ Session │  │   RFC 9421      │    │
│  │         │  │ Manager │  │   Signature     │    │
│  └─────────┘  └─────────┘  └─────────────────┘    │
└─────────────────────────────────────────────────────┘
         │              │                │
         ▼              ▼                ▼
    ┌────────┐    ┌─────────┐     ┌──────────┐
    │ Loki/  │    │ Jaeger  │     │Prometheus│
    │FluentD │    │         │     │          │
    └────────┘    └─────────┘     └──────────┘
         │              │                │
         └──────────────┴────────────────┘
                        ▼
                   ┌─────────┐
                   │ Grafana │
                   │(통합 뷰)│
                   └─────────┘
```

**구현 예시:**
```go
// pkg/observability/middleware.go
type Middleware struct {
    logger  *logging.Logger
    tracer  *tracing.Tracer
    metrics *metrics.Collector
}

func (m *Middleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. 요청 시작
        start := time.Now()
        requestID := uuid.New().String()

        // 2. 트레이스 시작
        ctx, span := m.tracer.StartHTTPSpan(r.Context(), r)
        defer span.End()

        // 3. 로거에 컨텍스트 추가
        logger := m.logger.WithFields(map[string]interface{}{
            "request_id": requestID,
            "trace_id":   span.SpanContext().TraceID().String(),
            "method":     r.Method,
            "path":       r.URL.Path,
        })
        ctx = context.WithValue(ctx, loggerKey, logger)

        // 4. Response writer 래핑 (status code 캡처)
        rw := &responseWriter{ResponseWriter: w, statusCode: 200}

        // 5. 다음 핸들러 실행
        next.ServeHTTP(rw, r.WithContext(ctx))

        // 6. 완료 로깅 및 메트릭
        duration := time.Since(start)
        logger.Info("request completed",
            zap.Int("status", rw.statusCode),
            zap.Duration("duration", duration),
        )

        m.metrics.HTTPRequestDuration.WithLabelValues(
            r.Method, r.URL.Path, strconv.Itoa(rw.statusCode),
        ).Observe(duration.Seconds())

        m.metrics.HTTPRequestsTotal.WithLabelValues(
            r.Method, r.URL.Path, strconv.Itoa(rw.statusCode),
        ).Inc()
    })
}
```

---

## 4. 결론 및 권장사항

### 4.1 설계의 전반적 평가

**Yes 적절한 기술 선택:**
- Zap (로깅): 성능, 구조화, 에코시스템
- OpenTelemetry (추적): 표준, 벤더 중립, 미래 지향
- Prometheus (메트릭): 클라우드 네이티브 표준, Grafana 통합

**Warning 개선 필요 영역:**
1. SAGE 특화 메트릭 확장 (DID, 블록체인, 보안)
2. 로깅-추적-메트릭 상관관계 강화
3. 환경별 설정 관리
4. 보안 관련 알림 강화

### 4.2 우선순위 재조정 (실제 구현 반영)

**원래 예상:** 3일 (0일부터 시작 가정)
**실제 상황:** 인프라 70% 구현 완료
**수정된 예상:** 1-2일

#### ✅ Phase 1: 기본 인프라 - **이미 완료**
- ✅ Prometheus 메트릭 정의 (`internal/metrics/*.go`)
- ✅ 메트릭 서버 구현 (`internal/metrics/server.go`)
- ✅ Grafana 대시보드 (`docker/grafana/dashboards/sage-overview.json`)
- ✅ Docker Compose 설정 (Prometheus, Grafana)
- ✅ Zap 의존성 설치

#### 🔴 Phase 2 (즉시): 설정 수정 및 통합 - **0.5일**
- [ ] **즉시 조치 1**: Prometheus 설정 수정 (엔드포인트 통합)
- [ ] **즉시 조치 2**: 핸드셰이크 코드에 메트릭 통합
- [ ] **즉시 조치 3**: 세션 매니저에 메트릭 통합
- [ ] 메트릭 동작 검증 (Grafana 확인)

#### 🟡 Phase 3 (단기): 로깅 개선 - **0.5-1일**
- [ ] Zap 로거 래퍼 구현 (`pkg/logging/`)
- [ ] 표준 필드 정의 (`pkg/logging/fields.go`)
- [ ] HTTP 미들웨어 추가
- [ ] 핵심 모듈에 Zap 적용 (점진적)

#### 🟢 Phase 4 (중기): 추적 및 알림 - **0.5-1일**
- [ ] OpenTelemetry 설정
- [ ] Jaeger를 Docker Compose에 추가
- [ ] HTTP 추적 미들웨어
- [ ] Prometheus 알림 규칙 추가

**총 예상 시간:** 1.5-2.5일 (인프라 재사용으로 단축)

### 4.3 즉시 적용 가능한 Quick Wins

1. **기존 로그 마이그레이션 자동화:**
   ```bash
   # 간단한 sed 스크립트로 log.Printf -> logger.Info 변환
   find . -name "*.go" -type f -exec sed -i '' 's/log.Printf/logger.Info/g' {} \;
   ```

2. **핵심 메트릭만 우선 구현:**
   - HandshakesTotal, SessionsActive, HandshakeDuration
   - 나머지는 점진적 추가

3. **Docker Compose에 Jaeger 추가 (1분):**
   ```yaml
   services:
     jaeger:
       image: jaegertracing/all-in-one:latest
       ports:
         - "16686:16686"  # UI
         - "14268:14268"  # HTTP collector
   ```

### 4.4 최종 권장사항

**Yes 진행 승인:**
- 제안된 모니터링 및 관찰성 작업은 프로덕션에 **필수적**
- 기술 선택과 설계 방향은 **적절함**
- 제시된 타임라인(2-3일)은 **달성 가능**

**Warning 주의사항:**
1. **점진적 마이그레이션**: 모든 로그를 한번에 변경하지 말고 모듈별 진행
2. **성능 오버헤드 모니터링**: 추적 샘플링 비율 조정 (프로덕션: 10%, 개발: 100%)
3. **알림 피로도 방지**: 초기에는 critical 알림만, 점진적 확장
4. **문서화 우선**: 다른 개발자가 메트릭/로그를 쉽게 추가할 수 있도록 가이드 작성

**Note 다음 단계:**
1. `pkg/logging/`, `pkg/metrics/`, `pkg/tracing/` 패키지 구현
2. 핵심 모듈(`handshake`, `session`, `rfc9421`)에 통합
3. 기존 예제 코드 업데이트 (best practice 시연)
4. 통합 테스트로 검증

---

**결론:** 제안된 모니터링 및 관찰성 작업은 SAGE의 프로덕션 준비에 **절대적으로 필요**하며, 설계 방향은 **적절**합니다. 다만 SAGE 특화 메트릭과 보안 알림을 강화하고, 관찰성 도구 간 상관관계를 명확히 하면 더욱 효과적일 것입니다.

---

## 5. 🔴 즉시 조치 사항 (Immediate Actions Required)

> **업데이트:** 2025-10-10
> **우선순위:** Critical - 설정 불일치 해소 및 메트릭 활성화

### 5.1 즉시 조치 1: Prometheus 설정 수정 ⏱️ 15분

**문제:** Prometheus가 존재하지 않는 엔드포인트를 스크랩하려고 시도

**현재 설정** (`docker/prometheus/prometheus.yml:72-100`):
```yaml
- job_name: 'sage-sessions'
  metrics_path: '/metrics/sessions'  # ❌ 미구현

- job_name: 'sage-handshakes'
  metrics_path: '/metrics/handshakes'  # ❌ 미구현

- job_name: 'sage-crypto'
  metrics_path: '/metrics/crypto'  # ❌ 미구현
```

**해결 방법:**
```yaml
# 위 3개 job 삭제, 아래 job만 유지 (라인 26-39 유지)
- job_name: 'sage-backend'
  scrape_interval: 10s
  metrics_path: '/metrics'  # ✅ 실제 구현됨
  static_configs:
    - targets:
        - 'sage-backend:9090'
      labels:
        service: 'sage-backend'
```

**액션:**
```bash
# docker/prometheus/prometheus.yml 편집
# 라인 72-100 삭제 (sage-sessions, sage-handshakes, sage-crypto jobs)
# Prometheus 재시작
docker-compose restart prometheus
```

---

### 5.2 즉시 조치 2: 핸드셰이크 메트릭 통합 ⏱️ 30분

**문제:** 메트릭이 정의되어 있지만 실제 코드에서 호출되지 않음

**통합 위치:**
- `handshake/client.go` - 클라이언트 핸드셰이크
- `handshake/server.go` - 서버 핸드셰이크
- `hpke/client.go`, `hpke/server.go` - HPKE 핸드셰이크

**예시 코드 추가:**
```go
// handshake/client.go
import "github.com/sage-x-project/sage/internal/metrics"

func (c *Client) InitiateHandshake() error {
    start := time.Now()
    defer func() {
        metrics.HandshakeDuration.WithLabelValues("init").Observe(
            time.Since(start).Seconds(),
        )
    }()

    metrics.HandshakesInitiated.WithLabelValues("client").Inc()

    // 기존 로직...

    if err != nil {
        metrics.HandshakesFailed.WithLabelValues("network_error").Inc()
        return err
    }

    metrics.HandshakesCompleted.WithLabelValues("success").Inc()
    return nil
}
```

**통합 체크리스트:**
- [ ] `handshake/client.go` - InitiateHandshake()
- [ ] `handshake/server.go` - AcceptHandshake()
- [ ] `hpke/client.go` - Initialize()
- [ ] `hpke/server.go` - ProcessInitialize()

---

### 5.3 즉시 조치 3: 세션 메트릭 통합 ⏱️ 30분

**문제:** 세션 생성/만료 메트릭이 기록되지 않음

**통합 위치:** `session/manager.go`

**예시 코드:**
```go
// session/manager.go
import "github.com/sage-x-project/sage/internal/metrics"

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

    metrics.SessionsCreated.WithLabelValues("success").Inc()
    metrics.SessionsActive.Inc()
    return sess, nil
}

func (m *Manager) cleanup() {
    expired := m.removeExpiredSessions()
    if expired > 0 {
        metrics.SessionsExpired.Add(float64(expired))
        metrics.SessionsActive.Sub(float64(expired))
    }
}

func (s *Session) Encrypt(plaintext []byte) ([]byte, error) {
    start := time.Now()
    defer metrics.SessionDuration.WithLabelValues("encrypt").Observe(
        time.Since(start).Seconds(),
    )

    metrics.SessionMessageSize.WithLabelValues("outbound").Observe(
        float64(len(plaintext)),
    )

    // 기존 암호화 로직...
}
```

**통합 체크리스트:**
- [ ] `session/manager.go` - CreateSession()
- [ ] `session/manager.go` - cleanup()
- [ ] `session/session.go` - Encrypt()
- [ ] `session/session.go` - Decrypt()

---

### 5.4 즉시 조치 4: 메트릭 검증 ⏱️ 15분

**액션:**
```bash
# 1. Docker Compose 시작
docker-compose up -d prometheus grafana sage-backend

# 2. Prometheus 타겟 확인
open http://localhost:9091/targets
# sage-backend가 UP 상태인지 확인

# 3. 메트릭 조회 테스트
curl http://localhost:9090/metrics | grep sage_

# 4. Grafana 대시보드 확인
open http://localhost:3000
# 로그인: admin / admin
# Dashboards → SAGE System Overview
```

**예상 메트릭:**
```
sage_handshakes_initiated_total{role="client"} 5
sage_handshakes_completed_total{status="success"} 4
sage_sessions_active 2
sage_crypto_operations_total{operation="sign",algorithm="ed25519"} 10
```

---

### 5.5 작업 우선순위 및 예상 시간

| 번호 | 작업 | 우선순위 | 예상 시간 | 종속성 |
|------|------|----------|-----------|--------|
| 5.1 | Prometheus 설정 수정 | 🔴 Critical | 15분 | 없음 |
| 5.2 | 핸드셰이크 메트릭 통합 | 🔴 Critical | 30분 | 5.1 |
| 5.3 | 세션 메트릭 통합 | 🔴 Critical | 30분 | 5.1 |
| 5.4 | 메트릭 검증 | 🔴 Critical | 15분 | 5.2, 5.3 |
| **합계** | | | **90분 (1.5시간)** | |

---

### 5.6 후속 조치 (단기 - 이번 주)

#### RFC 9421 서명 메트릭 추가 ⏱️ 20분
```go
// core/rfc9421/verifier_http.go
func (v *HTTPVerifier) VerifyRequest(r *http.Request, ...) error {
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

#### DID 해결 메트릭 추가 ⏱️ 20분
```go
// did/resolver.go
func (r *MultiChainResolver) Resolve(did string) (*Document, error) {
    start := time.Now()
    chain := extractChain(did) // ethereum, solana

    defer metrics.GetGlobalCollector().RecordDIDResolution(
        false, // cached 여부
        time.Since(start),
    )

    // 기존 해결 로직...
}
```

#### Nonce/Replay 공격 메트릭 추가 ⏱️ 15분
```go
// core/rfc9421/nonce.go
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

### 5.7 체크리스트 (Copy & Paste)

```markdown
## 즉시 조치 체크리스트

### Phase 1: 설정 수정 (15분)
- [ ] Prometheus 설정 편집 (`docker/prometheus/prometheus.yml`)
  - [ ] 라인 72-100 삭제 (sage-sessions, sage-handshakes, sage-crypto)
  - [ ] 라인 26-39 확인 (sage-backend job 유지)
- [ ] Prometheus 재시작: `docker-compose restart prometheus`
- [ ] 타겟 확인: http://localhost:9091/targets

### Phase 2: 핸드셰이크 메트릭 (30분)
- [ ] `handshake/client.go` 수정
  - [ ] metrics import 추가
  - [ ] HandshakesInitiated.Inc() 추가
  - [ ] HandshakeDuration.Observe() 추가
- [ ] `handshake/server.go` 수정
  - [ ] 동일 패턴 적용
- [ ] 테스트: `go test ./handshake/...`

### Phase 3: 세션 메트릭 (30분)
- [ ] `session/manager.go` 수정
  - [ ] CreateSession() - SessionsCreated, SessionsActive
  - [ ] cleanup() - SessionsExpired
- [ ] `session/session.go` 수정
  - [ ] Encrypt() - SessionDuration, MessageSize
  - [ ] Decrypt() - 동일 패턴
- [ ] 테스트: `go test ./session/...`

### Phase 4: 검증 (15분)
- [ ] 서비스 시작: `docker-compose up -d`
- [ ] 메트릭 확인: `curl localhost:9090/metrics | grep sage_`
- [ ] Grafana 확인: http://localhost:3000
- [ ] 대시보드 데이터 표시 확인
```

---

## 6. 📋 작업 완료 기준

### 성공 지표:
1. ✅ Prometheus가 `/metrics` 엔드포인트에서 메트릭 수집
2. ✅ Grafana 대시보드에 실시간 데이터 표시
3. ✅ 핸드셰이크 카운터 증가 확인
4. ✅ 세션 활성 게이지 변화 확인
5. ✅ 에러 없이 모든 테스트 통과

### 검증 명령어:
```bash
# 메트릭 존재 확인
curl -s localhost:9090/metrics | grep -E "sage_(handshakes|sessions|crypto)" | head -20

# Prometheus 쿼리 테스트
curl -G 'http://localhost:9091/api/v1/query' \
  --data-urlencode 'query=sage_sessions_active' | jq

# 핸드셰이크 테스트 실행 후 메트릭 확인
go test ./handshake/... -v
curl -s localhost:9090/metrics | grep handshakes_completed_total
```

---

**최종 업데이트:** 2025-10-10
**다음 리뷰:** 메트릭 통합 완료 후 (예상: 2025-10-11)
