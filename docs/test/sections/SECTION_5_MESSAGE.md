## 5. 메시지 처리

### 5.1 Nonce 관리

#### 5.1.1 생성/검증

##### 5.1.1.1 중복된 Nonce 생성 없음 확인

**시험항목**: Nonce 생성 시 중복 방지 (Cryptographically Secure)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/GenerateNonce'
```

**예상 결과**:

```
=== RUN   TestNonceManager/GenerateNonce
    manager_test.go:37: ===== 5.1.1 Nonce Generation (Cryptographically Secure) =====
    manager_test.go:43: [PASS] Nonce generation successful
    manager_test.go:44:   Nonce value: 6rKHp5eJt6Z0NDwsvojHBA
    manager_test.go:45:   Nonce length: 22 characters
    manager_test.go:61: [PASS] Nonce uniqueness verified
--- PASS: TestNonceManager/GenerateNonce (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `nonce.GenerateNonce()` - 암호학적으로 안전한 Nonce 생성
- Nonce 생성 시 고유성 보장
- 두 개의 Nonce 생성 후 중복 검사
- Nonce 길이 검증 (최소 16 bytes)

**통과 기준**:

- ✅ Nonce 생성 성공
- ✅ 생성된 Nonce 길이 충분
- ✅ 두 Nonce가 서로 다름 (중복 없음)
- ✅ 암호학적으로 안전한 생성

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestNonceManager/GenerateNonce
    manager_test.go:37: ===== 5.1.1 Nonce Generation (Cryptographically Secure) =====
    manager_test.go:43: [PASS] Nonce generation successful
    manager_test.go:44:   Nonce value: 6rKHp5eJt6Z0NDwsvojHBA
    manager_test.go:45:   Nonce length: 22 characters
    manager_test.go:54:   Nonce encoding: non-hex format
    manager_test.go:61: [PASS] Nonce uniqueness verified
    manager_test.go:62:   Second nonce: Uqe7BR5Wxijp0AM1ZU9oyA
    manager_test.go:82:   Test data saved: testdata/verification/nonce/nonce_generation.json
--- PASS: TestNonceManager/GenerateNonce (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/nonce/manager_test.go:35-83`
- 테스트 데이터: `testdata/verification/nonce/nonce_generation.json`
- 상태: ✅ PASS
- SAGE 함수: `nonce.GenerateNonce()`
- Nonce 1: 22 characters (base64url 인코딩)
- Nonce 2: 22 characters (중복 없음 확인)
- 고유성: ✅ 검증 완료

---

##### 5.1.1.2 사용된 Nonce 재사용 방지

**시험항목**: Nonce 재사용 탐지 및 Replay 공격 방어

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/CheckReplay'
```

**예상 결과**:

```
=== RUN   TestNonceManager/CheckReplay
    manager_test.go:244: ===== 1.2.2 Nonce Duplicate Detection (CheckReplay) =====
    manager_test.go:256: [PASS] First use: nonce not marked as used
    manager_test.go:266: [PASS] Duplicate nonce detected successfully
    manager_test.go:272: [PASS] Replay attack prevention working
--- PASS: TestNonceManager/CheckReplay (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `nonce.Manager.MarkNonceUsed()` - Nonce 사용 표시
- **SAGE 함수 사용**: `nonce.Manager.IsNonceUsed()` - Nonce 사용 여부 확인
- 첫 사용 시 정상 처리
- 두 번째 사용 시 중복 탐지
- Replay 공격 방어 확인

**통과 기준**:

- ✅ 첫 사용 정상 처리
- ✅ 중복 Nonce 탐지
- ✅ Replay 공격 방어
- ✅ 사용된 Nonce 추적

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestNonceManager/CheckReplay
    manager_test.go:244: ===== 1.2.2 Nonce Duplicate Detection (CheckReplay) =====
    manager_test.go:251:   Generated nonce: KpRith5a2Xv0lSmakGerow
    manager_test.go:256: [PASS] First use: nonce not marked as used
    manager_test.go:257:   Is used before marking: false
    manager_test.go:261: [PASS] Nonce marked as used
    manager_test.go:266: [PASS] Duplicate nonce detected successfully
    manager_test.go:267:   Is used after marking: true
    manager_test.go:272: [PASS] Replay attack prevention working
    manager_test.go:293:   Test data saved: testdata/verification/nonce/nonce_check_replay.json
--- PASS: TestNonceManager/CheckReplay (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/nonce/manager_test.go:242-294`
- 테스트 데이터: `testdata/verification/nonce/nonce_check_replay.json`
- 상태: ✅ PASS
- SAGE 함수:
  - `nonce.GenerateNonce()` - Nonce 생성
  - `nonce.Manager.MarkNonceUsed()` - 사용 표시
  - `nonce.Manager.IsNonceUsed()` - 사용 여부 확인
- 첫 사용: false → 정상 처리
- 두 번째 사용: true → Replay 탐지
- 보안: ✅ Replay 공격 방어

---

##### 5.1.1.3 Nonce TTL(5분) 준수 확인

**시험항목**: Nonce TTL 기반 만료 및 자동 정리

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/Expiration'
```

**예상 결과**:

```
=== RUN   TestNonceManager/Expiration
    manager_test.go:299: ===== 10.1.10 Nonce Expiration (TTL-based) =====
    manager_test.go:313: [PASS] Nonce marked as used
    manager_test.go:319: [PASS] Nonce tracked before expiry
    manager_test.go:329: [PASS] Expired nonce correctly identified as unused
    manager_test.go:335: [PASS] Expired nonce removed from tracking
--- PASS: TestNonceManager/Expiration (0.07s)
```

**검증 방법**:

- **SAGE 함수 사용**: `nonce.NewManager(ttl, cleanupInterval)` - TTL 기반 Nonce 관리자 생성
- **SAGE 함수 사용**: `nonce.Manager.MarkNonceUsed()` - Nonce 사용 표시
- **SAGE 함수 사용**: `nonce.Manager.IsNonceUsed()` - 만료 확인 포함
- TTL 설정 (테스트: 50ms, 실제: 5분)
- TTL 경과 후 만료 확인
- 만료된 Nonce 제거 확인

**통과 기준**:

- ✅ TTL 설정 가능
- ✅ TTL 경과 전 Nonce 추적
- ✅ TTL 경과 후 만료 처리
- ✅ 만료 Nonce 자동 제거
- ✅ 메모리 효율적 관리

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestNonceManager/Expiration
    manager_test.go:299: ===== 10.1.10 Nonce Expiration (TTL-based) =====
    manager_test.go:306:   Generated nonce: Jk7Vn73IwhqvpBfhKleCOA
    manager_test.go:307:   TTL: 50ms
    manager_test.go:313: [PASS] Nonce marked as used
    manager_test.go:314:   Initial count: 1
    manager_test.go:319: [PASS] Nonce tracked before expiry
    manager_test.go:323:   Waiting 70ms for nonce to expire
    manager_test.go:329: [PASS] Expired nonce correctly identified as unused
    manager_test.go:330:   Is used after expiry: false
    manager_test.go:335: [PASS] Expired nonce removed from tracking
    manager_test.go:336:   Final count: 0
    manager_test.go:360:   Test data saved: testdata/verification/nonce/nonce_expiration.json
--- PASS: TestNonceManager/Expiration (0.07s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/nonce/manager_test.go:297-361`
- 테스트 데이터: `testdata/verification/nonce/nonce_expiration.json`
- 상태: ✅ PASS
- SAGE 함수:
  - `nonce.NewManager(ttl, cleanupInterval)` - TTL 기반 관리자
  - `nonce.Manager.MarkNonceUsed()` - Nonce 사용 표시
  - `nonce.Manager.IsNonceUsed()` - 만료 시 자동 제거
- 테스트 TTL: 50ms (실제는 5분 = 300,000ms)
- 만료 전: 추적됨 (count=1)
- 만료 후: 제거됨 (count=0)
- 메모리: ✅ 효율적 관리

---

### 5.2 메시지 순서

#### 5.2.1 순서 보장

##### 5.2.1.1 메시지 ID 규칙성 확인

**시험항목**: 메시지 Sequence Number 단조 증가 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/SeqMonotonicity'
```

**예상 결과**:

```
=== RUN   TestOrderManager/SeqMonotonicity
    manager_test.go:135: ===== 8.1.1 Message Sequence Number Monotonicity =====
    manager_test.go:147: [PASS] First message (seq=1) accepted
    manager_test.go:154: [PASS] Replay attack detected: Duplicate sequence rejected
    manager_test.go:162: [PASS] Higher sequence (seq=2) accepted
--- PASS: TestOrderManager/SeqMonotonicity (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `order.Manager.ProcessMessage()` - 메시지 순서 검증
- Sequence number 단조 증가 확인
- 중복 Sequence 거부
- Replay 공격 방어

**통과 기준**:

- ✅ 첫 메시지 수락 (seq=1)
- ✅ 중복 Sequence 거부
- ✅ 증가하는 Sequence 수락 (seq=2)
- ✅ Replay 공격 방어

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestOrderManager/SeqMonotonicity
    manager_test.go:135: ===== 8.1.1 Message Sequence Number Monotonicity =====
    manager_test.go:139:   Session ID: sess2
    manager_test.go:140:   Base timestamp: 2025-10-24T02:33:53.302575+09:00
    manager_test.go:144:   Processing message with sequence: 1
    manager_test.go:147: [PASS] First message (seq=1) accepted
    manager_test.go:150:   Attempting replay with same sequence: 1
    manager_test.go:154: [PASS] Replay attack detected: Duplicate sequence rejected
    manager_test.go:155:   Error message: invalid sequence: 1 >= last 1
    manager_test.go:159:   Processing message with higher sequence: 2
    manager_test.go:162: [PASS] Higher sequence (seq=2) accepted
    manager_test.go:192:   Test data saved: testdata/verification/message/order/sequence_monotonicity.json
--- PASS: TestOrderManager/SeqMonotonicity (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/order/manager_test.go:133-193`
- 테스트 데이터: `testdata/verification/message/order/sequence_monotonicity.json`
- 상태: ✅ PASS
- SAGE 함수: `order.Manager.ProcessMessage()`
- Sequence 1: ✅ 수락
- Sequence 1 (중복): ✅ 거부
- Sequence 2: ✅ 수락
- 단조 증가: ✅ 검증 완료

---

##### 5.2.1.2 타임스탬프 순서 2024 검증 확인

**시험항목**: 타임스탬프 순서 검증 (Temporal Consistency)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/TimestampOrder'
```

**예상 결과**:

```
=== RUN   TestOrderManager/TimestampOrder
    manager_test.go:197: ===== 8.1.2 Message Timestamp Ordering =====
    manager_test.go:209: [PASS] Baseline timestamp established
    manager_test.go:218: [PASS] Out-of-order timestamp rejected
    manager_test.go:227: [PASS] Later timestamp accepted
--- PASS: TestOrderManager/TimestampOrder (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `order.Manager.ProcessMessage()` - 타임스탬프 순서 검증
- 첫 메시지로 기준 타임스탬프 설정
- 이전 타임스탬프 거부 (out-of-order)
- 이후 타임스탬프 수락

**통과 기준**:

- ✅ 기준 타임스탬프 설정
- ✅ 이전 타임스탬프 거부
- ✅ 이후 타임스탬프 수락
- ✅ 시간 순서 일관성 유지

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestOrderManager/TimestampOrder
    manager_test.go:197: ===== 8.1.2 Message Timestamp Ordering =====
    manager_test.go:201:   Session ID: sess3
    manager_test.go:202:   Base timestamp: 2025-10-24T02:33:53.30394+09:00
    manager_test.go:206:   First message - seq=10, timestamp=2025-10-24T02:33:53.30394+09:00
    manager_test.go:209: [PASS] Baseline timestamp established
    manager_test.go:214:   Second message - seq=11, timestamp=2025-10-24T02:33:52.30394+09:00 (1 second earlier)
    manager_test.go:218: [PASS] Out-of-order timestamp rejected
    manager_test.go:219:   Error message: out-of-order: 2025-10-24 02:33:52.30394 +0900 KST m=-0.996442999 before 2025-10-24 02:33:53.30394 +0900 KST m=+0.003557001
    manager_test.go:224:   Third message - seq=12, timestamp=2025-10-24T02:33:54.30394+09:00 (1 second later)
    manager_test.go:227: [PASS] Later timestamp accepted
    manager_test.go:261:   Test data saved: testdata/verification/message/order/timestamp_ordering.json
--- PASS: TestOrderManager/TimestampOrder (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/order/manager_test.go:195-262`
- 테스트 데이터: `testdata/verification/message/order/timestamp_ordering.json`
- 상태: ✅ PASS
- SAGE 함수: `order.Manager.ProcessMessage()`
- 기준 타임스탬프: 2025-10-24T02:33:53
- 이전 타임스탬프 (-1초): ✅ 거부
- 이후 타임스탬프 (+1초): ✅ 수락
- 시간 순서: ✅ 일관성 유지

**참고**: 타임스탬프는 메시지 생성 시점의 현재 시간을 사용하며, 테스트는 2025년에 실행되었습니다. 시간 순서 검증 로직 자체는 연도에 무관하게 동작합니다.

---

##### 5.2.1.3 중복 메시지 거부 자동 거부

**시험항목**: 순서 불일치 및 중복 메시지 탐지

**Go 테스트**:

```bash
# Sequence 검증
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/ValidateSeq'

# Out-of-order 탐지
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/OutOfOrder'
```

**예상 결과**:

```
=== RUN   TestOrderManager/ValidateSeq
    manager_test.go:373: ===== 5.2.2 Sequence Number Validation =====
    manager_test.go:385: [PASS] Valid sequence accepted (seq=1)
    manager_test.go:393: [PASS] Valid sequence accepted (seq=2)
    manager_test.go:402: [PASS] Invalid sequence rejected (same as previous)
    manager_test.go:412: [PASS] Invalid sequence rejected (lower than current)
--- PASS: TestOrderManager/ValidateSeq (0.00s)

=== RUN   TestOrderManager/OutOfOrder
    manager_test.go:452: ===== 5.2.3 Out-of-Order Message Detection =====
    manager_test.go:465: [PASS] Baseline established (seq=5)
    manager_test.go:473: [PASS] Normal progression accepted (seq=6)
    manager_test.go:481: [PASS] Out-of-order message detected and rejected
    manager_test.go:491: [PASS] Out-of-order timestamp detected and rejected
--- PASS: TestOrderManager/OutOfOrder (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `order.Manager.ProcessMessage()` - 순서 검증 및 중복 탐지
- 올바른 Sequence 수락
- 잘못된 Sequence 거부 (중복, 역행)
- Out-of-order 메시지 거부

**통과 기준**:

- ✅ 올바른 순서 수락
- ✅ 잘못된 순서 거부
- ✅ Sequence 역행 탐지
- ✅ 타임스탬프 역행 탐지
- ✅ 중복 메시지 자동 거부

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestOrderManager/ValidateSeq
    manager_test.go:373: ===== 5.2.2 Sequence Number Validation =====
    manager_test.go:385: [PASS] Valid sequence accepted (seq=1)
    manager_test.go:393: [PASS] Valid sequence accepted (seq=2)
    manager_test.go:402: [PASS] Invalid sequence rejected (same as previous)
    manager_test.go:412: [PASS] Invalid sequence rejected (lower than current)
    manager_test.go:421: [PASS] Valid sequence accepted (seq=10, forward jump)
    manager_test.go:446:   Test data saved: testdata/verification/message/order/sequence_validation.json
--- PASS: TestOrderManager/ValidateSeq (0.00s)

=== RUN   TestOrderManager/OutOfOrder
    manager_test.go:452: ===== 5.2.3 Out-of-Order Message Detection =====
    manager_test.go:465: [PASS] Baseline established (seq=5)
    manager_test.go:473: [PASS] Normal progression accepted (seq=6)
    manager_test.go:481: [PASS] Out-of-order message detected and rejected
    manager_test.go:491: [PASS] Out-of-order timestamp detected and rejected
    manager_test.go:500: [PASS] Correct order accepted after rejections
    manager_test.go:524:   Test data saved: testdata/verification/message/order/out_of_order_detection.json
--- PASS: TestOrderManager/OutOfOrder (0.00s)
```

**검증 데이터**:
- 테스트 파일:
  - `pkg/agent/core/message/order/manager_test.go:371-447` (ValidateSeq)
  - `pkg/agent/core/message/order/manager_test.go:450-525` (OutOfOrder)
- 테스트 데이터:
  - `testdata/verification/message/order/sequence_validation.json`
  - `testdata/verification/message/order/out_of_order_detection.json`
- 상태: ✅ PASS
- SAGE 함수: `order.Manager.ProcessMessage()`
- Sequence 검증: ✅ 동일/역행 거부
- Out-of-order 탐지: ✅ 메시지 거부
- 보안: ✅ 중복 메시지 자동 거부

---

### 5.3 중복 서비스

#### 5.3.1 통합 검증

##### 5.3.1.1 DID 중복 상태 확인 테스트

**시험항목**: 중복 메시지 탐지 (Deduplication)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/dedupe -run 'TestDetector/MarkAndDetectDuplicate'
```

**예상 결과**:

```
=== RUN   TestDetector/MarkAndDetectDuplicate
    detector_test.go:108: ===== 8.2.1 Message Deduplication Detection =====
    detector_test.go:130: [PASS] Packet marked as seen
    detector_test.go:139: [PASS] Duplicate detected: Replay attack prevented
--- PASS: TestDetector/MarkAndDetectDuplicate (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `dedupe.Detector.MarkPacketSeen()` - 메시지 추적
- **SAGE 함수 사용**: `dedupe.Detector.IsDuplicate()` - 중복 탐지
- 메시지 해시 기반 중복 탐지
- Replay 공격 방어

**통과 기준**:

- ✅ 메시지 추적 성공
- ✅ 중복 메시지 탐지
- ✅ Replay 공격 방어
- ✅ 메시지 카운트 정확

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestDetector/MarkAndDetectDuplicate
    detector_test.go:108: ===== 8.2.1 Message Deduplication Detection =====
    detector_test.go:114:   Detector TTL: 1s
    detector_test.go:115:   Cleanup interval: 1s
    detector_test.go:123:   Message header:
    detector_test.go:124:     Sequence: 1
    detector_test.go:125:     Nonce: n1
    detector_test.go:126:     Timestamp: 2025-10-24T02:34:07.703312+09:00
    detector_test.go:130: [PASS] Packet marked as seen
    detector_test.go:134:   Seen packet count: 1
    detector_test.go:139: [PASS] Duplicate detected: Replay attack prevented
    detector_test.go:140:   Is duplicate: true
    detector_test.go:170:   Test data saved: testdata/verification/message/dedupe/deduplication_detection.json
--- PASS: TestDetector/MarkAndDetectDuplicate (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/dedupe/detector_test.go:106-171`
- 테스트 데이터: `testdata/verification/message/dedupe/deduplication_detection.json`
- 상태: ✅ PASS
- SAGE 함수:
  - `dedupe.NewDetector()` - 중복 탐지기 생성
  - `dedupe.Detector.MarkPacketSeen()` - 메시지 추적
  - `dedupe.Detector.IsDuplicate()` - 중복 확인
- 첫 메시지: 추적됨 (count=1)
- 중복 메시지: ✅ 탐지됨
- Replay 방어: ✅ 성공

---

##### 5.3.1.2 공개키와 서명 검증

**시험항목**: Nonce 재사용 탐지 (Replay Detection)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/ReplayDetection'
```

**예상 결과**:

```
=== RUN   TestValidateMessage/ReplayDetection
    validator_test.go:234: ===== 8.3.1 Message Validator Replay Detection =====
    validator_test.go:262: [PASS] First message validated successfully
    validator_test.go:279: [PASS] Replay attack detected and prevented
--- PASS: TestValidateMessage/ReplayDetection (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `validator.MessageValidator.ValidateMessage()` - 메시지 종합 검증
- Nonce 재사용 탐지
- Replay 공격 방어
- 검증 통계 확인

**통과 기준**:

- ✅ 첫 메시지 검증 성공
- ✅ Replay 탐지 (같은 Nonce)
- ✅ 에러 메시지 정확
- ✅ 통계 추적 정확

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestValidateMessage/ReplayDetection
    validator_test.go:234: ===== 8.3.1 Message Validator Replay Detection =====
    validator_test.go:237: [PASS] Message validator initialized
    validator_test.go:246:   Test message:
    validator_test.go:247:     Sequence: 1
    validator_test.go:248:     Nonce: f91b40e9-4a2a-4a31-a586-5080ef5bd4b0
    validator_test.go:262: [PASS] First message validated successfully
    validator_test.go:271:   Attempting replay with same nonce
    validator_test.go:279: [PASS] Replay attack detected and prevented
    validator_test.go:283:     Error: nonce has been used before (replay attack detected)
    validator_test.go:332:   Test data saved: testdata/verification/message/validator/replay_detection.json
--- PASS: TestValidateMessage/ReplayDetection (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/validator/validator_test.go:232-333`
- 테스트 데이터: `testdata/verification/message/validator/replay_detection.json`
- 상태: ✅ PASS
- SAGE 함수: `validator.MessageValidator.ValidateMessage()`
- 첫 메시지: ✅ 검증 성공
- Replay 시도: ✅ 탐지 및 거부
- 에러: "nonce has been used before (replay attack detected)"
- 보안: ✅ Replay 공격 방어

---

##### 5.3.1.3 타임스탬프 & Nonce 검증

**시험항목**: 메시지 종합 검증 및 통계 (Valid Message and Statistics)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/ValidAndStats'
```

**예상 결과**:

```
=== RUN   TestValidateMessage/ValidAndStats
    validator_test.go:46: ===== 8.3.2 Message Validator Valid Message and Statistics =====
    validator_test.go:62: [PASS] Message validator initialized
    validator_test.go:86: [PASS] Message validated successfully
    validator_test.go:98: [PASS] Statistics verified
--- PASS: TestValidateMessage/ValidAndStats (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `validator.NewMessageValidator()` - 검증자 생성
- **SAGE 함수 사용**: `validator.MessageValidator.ValidateMessage()` - 종합 검증
- **SAGE 함수 사용**: `validator.MessageValidator.GetStats()` - 통계 조회
- 타임스탬프, Nonce, Sequence 종합 검증
- 통계 추적 확인

**통과 기준**:

- ✅ 검증자 초기화 성공
- ✅ 유효한 메시지 검증 성공
- ✅ Replay, Duplicate, Out-of-order 플래그 확인
- ✅ 통계 추적 정확 (tracked_nonces, tracked_packets)

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestValidateMessage/ValidAndStats
    validator_test.go:46: ===== 8.3.2 Message Validator Valid Message and Statistics =====
    validator_test.go:55:   Validator configuration:
    validator_test.go:56:     Timestamp tolerance: 1s
    validator_test.go:57:     Nonce TTL: 1m0s
    validator_test.go:58:     Duplicate TTL: 1m0s
    validator_test.go:59:     Max out-of-order window: 1s
    validator_test.go:62: [PASS] Message validator initialized
    validator_test.go:86: [PASS] Message validated successfully
    validator_test.go:87:   Validation result:
    validator_test.go:88:     Is valid: true
    validator_test.go:89:     Is replay: false
    validator_test.go:90:     Is duplicate: false
    validator_test.go:91:     Is out-of-order: false
    validator_test.go:98: [PASS] Statistics verified
    validator_test.go:99:   Validator statistics:
    validator_test.go:100:     Tracked nonces: 1
    validator_test.go:101:     Tracked packets: 1
    validator_test.go:136:   Test data saved: testdata/verification/message/validator/valid_stats.json
--- PASS: TestValidateMessage/ValidAndStats (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/validator/validator_test.go:44-137`
- 테스트 데이터: `testdata/verification/message/validator/valid_stats.json`
- 상태: ✅ PASS
- SAGE 함수:
  - `validator.NewMessageValidator()` - 검증자 생성
  - `validator.MessageValidator.ValidateMessage()` - 종합 검증
  - `validator.MessageValidator.GetStats()` - 통계 조회
- 검증 결과: ✅ Valid, No replay, No duplicate, In order
- 통계: tracked_nonces=1, tracked_packets=1
- 종합 검증: ✅ 성공

---

##### 5.3.1.4 메시지 검증 종합

**시험항목**: Out-of-Order 메시지 탐지 및 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/OutOfOrderError'
```

**예상 결과**:

```
=== RUN   TestValidateMessage/OutOfOrderError
    validator_test.go:337: ===== 8.3.4 Message Validator Out-of-Order Detection =====
    validator_test.go:352: [PASS] Message validator initialized with strict order window
    validator_test.go:370: [PASS] First message validated successfully
    validator_test.go:391: [PASS] Out-of-order message correctly rejected
--- PASS: TestValidateMessage/OutOfOrderError (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `validator.MessageValidator.ValidateMessage()` - Order 검증 포함
- MaxOutOfOrderWindow 설정 (50ms)
- 기준 메시지 설정
- 순서 어긋난 메시지 거부 확인

**통과 기준**:

- ✅ 검증자 초기화 (strict order window)
- ✅ 첫 메시지 기준 설정
- ✅ Out-of-order 메시지 거부
- ✅ 에러 메시지 정확
- ✅ Order 보호 동작

**실제 테스트 결과** (2025-10-24):

```
=== RUN   TestValidateMessage/OutOfOrderError
    validator_test.go:337: ===== 8.3.4 Message Validator Out-of-Order Detection =====
    validator_test.go:346:   Validator configuration:
    validator_test.go:347:     Timestamp tolerance: 1s
    validator_test.go:348:     Max out-of-order window: 50ms (strict)
    validator_test.go:352: [PASS] Message validator initialized with strict order window
    validator_test.go:370: [PASS] First message validated successfully
    validator_test.go:379:   Second message (out-of-order):
    validator_test.go:382:     Timestamp: 100ms earlier
    validator_test.go:384:     Time difference: 100ms (exceeds 50ms window)
    validator_test.go:391: [PASS] Out-of-order message correctly rejected
    validator_test.go:394:     Error: order validation failed: out-of-order
    validator_test.go:448:   Test data saved: testdata/verification/message/validator/out_of_order.json
--- PASS: TestValidateMessage/OutOfOrderError (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/core/message/validator/validator_test.go:335-448`
- 테스트 데이터: `testdata/verification/message/validator/out_of_order.json`
- 상태: ✅ PASS
- SAGE 함수: `validator.MessageValidator.ValidateMessage()`
- Order window: 50ms (strict)
- 첫 메시지: ✅ 기준 설정
- Out-of-order (100ms 차이): ✅ 거부
- 에러: "order validation failed: out-of-order"
- 종합 검증: ✅ 메시지 검증 완료

---

