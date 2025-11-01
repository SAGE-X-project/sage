## 7. 세션 관리

### 7.1 세션 생성

#### 7.1.1 초기화

##### 7.1.1.1 중복된 세션 ID 생성 방지

**시험항목**: 중복 세션 ID 생성 방지 및 EnsureSessionWithParams 멱등성 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_1_DuplicateSessionIDPrevention'
```

**예상 결과**:

```
=== RUN   Test_7_1_1_1_DuplicateSessionIDPrevention
    session_test.go:474: ===== 7.1.1.1 중복된 세션 ID 생성 방지 =====
    session_test.go:493: [PASS] 첫 번째 세션 생성 성공
    session_test.go:500: [PASS] 중복 세션 ID 생성 방지 확인 (에러 발생)
    session_test.go:506: [PASS] 세션 카운트 검증 (중복 생성 안 됨)
    session_test.go:531: [PASS] EnsureSessionWithParams 중복 방지 확인 (기존 세션 반환)
--- PASS: Test_7_1_1_1_DuplicateSessionIDPrevention (0.00s)
```

**검증 방법**:

1. SAGE ComputeSessionIDFromSeed로 세션 ID 생성
2. 동일 ID로 중복 생성 시도 시 에러 발생 확인
3. 세션 카운트가 증가하지 않음 확인
4. EnsureSessionWithParams 멱등성 확인 (동일 파라미터 → 동일 세션 반환)
5. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_1_1_1_duplicate_prevention.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "duplicate_prevented": true,
    "ensure_params_idempotent": true,
    "session_count": 1,
    "session_id": "EhgtcpeC8ybpKUyf2Km6eA",
    "test_case": "7.1.1.1_Duplicate_Session_ID_Prevention"
  },
  "test_name": "Test_7_1_1_1_DuplicateSessionIDPrevention"
}
```

**통과 기준**:

-  SAGE ComputeSessionIDFromSeed 사용
-  중복 세션 ID 생성 시 에러 발생
-  세션 카운트 증가하지 않음
-  EnsureSessionWithParams 멱등성 확인

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_1_1_1_DuplicateSessionIDPrevention
    session_test.go:474: ===== 7.1.1.1 중복된 세션 ID 생성 방지 =====
    session_test.go:485:   세션 ID 생성:
    session_test.go:486:     SAGE ComputeSessionIDFromSeed 사용
    session_test.go:487:     Generated ID: EhgtcpeC8ybpKUyf2Km6eA
    session_test.go:493: [PASS] 첫 번째 세션 생성 성공
    session_test.go:494:     Session ID: EhgtcpeC8ybpKUyf2Km6eA
    session_test.go:500: [PASS] 중복 세션 ID 생성 방지 확인 (에러 발생)
    session_test.go:501:     Error: session EhgtcpeC8ybpKUyf2Km6eA already exists
    session_test.go:506: [PASS] 세션 카운트 검증 (중복 생성 안 됨)
    session_test.go:507:     Active sessions: 1
    session_test.go:522:   EnsureSessionWithParams 중복 검사:
    session_test.go:523:     Generated ID: w5A-Nkr8vQiqwyPdRwvG_g
    session_test.go:531: [PASS] EnsureSessionWithParams 중복 방지 확인 (기존 세션 반환)
    session_test.go:532:     First call existed: false
    session_test.go:533:     Second call existed: true
    session_test.go:534:     IDs match: true
    session_test.go:550:   Test data saved: testdata/verification/session/7_1_1_1_duplicate_prevention.json
--- PASS: Test_7_1_1_1_DuplicateSessionIDPrevention (0.00s)
```

---

##### 7.1.1.2 세션 ID 포맷 검증 확인

**시험항목**: SAGE 세션 ID 포맷 (base64url, 22 characters, 결정론적 생성) 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_2_SessionIDFormatValidation'
```

**예상 결과**:

```
=== RUN   Test_7_1_1_2_SessionIDFormatValidation
    session_test.go:555: ===== 7.1.1.2 세션 ID 포맷 검증 확인 =====
    session_test.go:569: [PASS] ComputeSessionIDFromSeed로 세션 ID 생성
    session_test.go:575: [PASS] 세션 ID 포맷 검증: base64url (RFC 4648)
    session_test.go:581: [PASS] 세션 ID 길이 검증: 22 characters
    session_test.go:589: [PASS] 검증된 세션 ID로 세션 생성 성공
    session_test.go:595: [PASS] 결정론적 생성 확인 (동일 입력 → 동일 ID)
    session_test.go:604: [PASS] 다른 입력으로 다른 ID 생성 (포맷 동일)
--- PASS: Test_7_1_1_2_SessionIDFormatValidation (0.00s)
```

**검증 방법**:

1. SAGE ComputeSessionIDFromSeed로 세션 ID 생성
2. Base64url 포맷 검증 (RFC 4648: A-Z, a-z, 0-9, _, -)
3. 고정 길이 22 characters 확인 (SHA256 해시 16바이트 → base64url 인코딩)
4. 결정론적 생성 확인 (동일 입력 → 동일 ID)
5. 다른 입력으로 다른 ID 생성 확인
6. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_1_1_2_id_format_validation.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "deterministic": true,
    "different_input_different_id": true,
    "format": "base64url",
    "session_id": "TQdv4I4R1teu6cw8cNsj7g",
    "session_id_length": 22,
    "test_case": "7.1.1.2_Session_ID_Format_Validation"
  },
  "test_name": "Test_7_1_1_2_SessionIDFormatValidation"
}
```

**통과 기준**:

-  SAGE ComputeSessionIDFromSeed 사용
-  Base64url 포맷 검증 (RFC 4648)
-  고정 길이 22 characters
-  결정론적 생성 확인
-  세션 생성 성공

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_1_1_2_SessionIDFormatValidation
    session_test.go:555: ===== 7.1.1.2 세션 ID 포맷 검증 확인 =====
    session_test.go:560:   SAGE 세션 ID 생성 함수 테스트:
    session_test.go:569: [PASS] ComputeSessionIDFromSeed로 세션 ID 생성
    session_test.go:570:     Generated ID: TQdv4I4R1teu6cw8cNsj7g
    session_test.go:571:     ID Length: 22 characters
    session_test.go:575: [PASS] 세션 ID 포맷 검증: base64url (RFC 4648)
    session_test.go:576:     Allowed characters: A-Z, a-z, 0-9, _, -
    session_test.go:577:     No padding (=) characters
    session_test.go:581: [PASS] 세션 ID 길이 검증: 22 characters
    session_test.go:582:     Source: SHA256 hash (16 bytes)
    session_test.go:583:     Encoding: base64url (22 chars)
    session_test.go:589: [PASS] 검증된 세션 ID로 세션 생성 성공
    session_test.go:595: [PASS] 결정론적 생성 확인 (동일 입력 → 동일 ID)
    session_test.go:604: [PASS] 다른 입력으로 다른 ID 생성 (포맷 동일)
    session_test.go:605:     Original ID:  TQdv4I4R1teu6cw8cNsj7g
    session_test.go:606:     Different ID: weF_WE614ug_84QUJ789_A
    session_test.go:624:   Test data saved: testdata/verification/session/7_1_1_2_id_format_validation.json
--- PASS: Test_7_1_1_2_SessionIDFormatValidation (0.00s)
```

---

##### 7.1.1.3 세션 데이터 메타데이터 설정 확인

**시험항목**: 세션 메타데이터 (ID, CreatedAt, LastUsedAt, MessageCount, Config, IsExpired) 설정 및 자동 갱신 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_3_SessionMetadataSetup'
```

**예상 결과**:

```
=== RUN   Test_7_1_1_3_SessionMetadataSetup
    session_test.go:629: ===== 7.1.1.3 세션 데이터 메타데이터 설정 확인 =====
    session_test.go:646: [PASS] 세션 생성 완료
    session_test.go:650: [PASS] 세션 ID 메타데이터 확인
    session_test.go:658: [PASS] 생성 시간 메타데이터 확인
    session_test.go:666: [PASS] 마지막 사용 시간 메타데이터 확인
    session_test.go:673: [PASS] 메시지 카운트 메타데이터 확인
    session_test.go:681: [PASS] 세션 설정 메타데이터 확인
    session_test.go:688: [PASS] 만료 상태 메타데이터 확인
    session_test.go:700: [PASS] 활동 후 메타데이터 자동 갱신 확인
--- PASS: Test_7_1_1_3_SessionMetadataSetup (0.00s)
```

**검증 방법**:

1. 세션 생성 후 모든 메타데이터 필드 검증
   - Session ID
   - CreatedAt (생성 시간)
   - LastUsedAt (마지막 사용 시간)
   - MessageCount (메시지 카운트, 초기값 0)
   - Config (MaxAge, IdleTimeout, MaxMessages)
   - IsExpired (만료 상태, 초기값 false)
2. 세션 활동 후 메타데이터 자동 갱신 확인
   - LastUsedAt 업데이트
   - MessageCount 증가
3. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_1_1_3_metadata_setup.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "created_at": "2025-10-24T01:48:20+09:00",
    "initial_message_count": 0,
    "is_expired": false,
    "last_used_at": "2025-10-24T01:48:20+09:00",
    "max_age_minutes": 60,
    "metadata_auto_update": true,
    "session_id": "JNIzi8APg6XHlXAv5NQ11A",
    "test_case": "7.1.1.3_Session_Metadata_Setup"
  },
  "test_name": "Test_7_1_1_3_SessionMetadataSetup"
}
```

**통과 기준**:

-  세션 ID 메타데이터 설정
-  생성 시간 (CreatedAt) 설정
-  마지막 사용 시간 (LastUsedAt) 설정
-  메시지 카운트 초기화
-  세션 설정 (Config) 저장
-  만료 상태 초기화
-  활동 시 메타데이터 자동 갱신

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_1_1_3_SessionMetadataSetup
    session_test.go:629: ===== 7.1.1.3 세션 데이터 메타데이터 설정 확인 =====
    session_test.go:638:   세션 생성:
    session_test.go:639:     Session ID: JNIzi8APg6XHlXAv5NQ11A
    session_test.go:646: [PASS] 세션 생성 완료
    session_test.go:650: [PASS] 세션 ID 메타데이터 확인
    session_test.go:651:     Session ID: JNIzi8APg6XHlXAv5NQ11A
    session_test.go:658: [PASS] 생성 시간 메타데이터 확인
    session_test.go:659:     Created At: 2025-10-24T01:48:20.374062+09:00
    session_test.go:666: [PASS] 마지막 사용 시간 메타데이터 확인
    session_test.go:667:     Last Used At: 2025-10-24T01:48:20.374062+09:00
    session_test.go:673: [PASS] 메시지 카운트 메타데이터 확인
    session_test.go:674:     Initial message count: 0
    session_test.go:681: [PASS] 세션 설정 메타데이터 확인
    session_test.go:682:     Max Age: 1h0m0s
    session_test.go:683:     Idle Timeout: 10m0s
    session_test.go:684:     Max Messages: 1000
    session_test.go:688: [PASS] 만료 상태 메타데이터 확인
    session_test.go:689:     Is Expired: false
    session_test.go:700: [PASS] 활동 후 메타데이터 자동 갱신 확인
    session_test.go:701:     New Last Used At: 2025-10-24T01:48:20.374252+09:00
    session_test.go:706:     Updated message count: 1
    session_test.go:732:   Test data saved: testdata/verification/session/7_1_1_3_metadata_setup.json
--- PASS: Test_7_1_1_3_SessionMetadataSetup (0.00s)
```

---

### 7.2 세션 관리

#### 7.2.1 조회/삭제

##### 7.2.1.1 세션 생성 ID TTL 시간 확인

**시험항목**: 세션 TTL (MaxAge) 설정 및 만료 시간 자동 무효화 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_1_SessionTTLTime'
```

**예상 결과**:

```
=== RUN   Test_7_2_1_1_SessionTTLTime
    session_test.go:737: ===== 7.2.1.1 세션 TTL 시간 확인 =====
    session_test.go:764: [PASS] TTL 설정된 세션 생성 완료
    session_test.go:772: [PASS] TTL 설정값 확인
    session_test.go:779: [PASS] TTL 절반 경과 - 세션 유효
    session_test.go:786: [PASS] TTL 만료 - 세션 무효
    session_test.go:795: [PASS] 만료된 세션 조회 실패 (자동 무효화)
--- PASS: Test_7_2_1_1_SessionTTLTime (0.12s)
```

**검증 방법**:

1. TTL 100ms로 설정된 세션 생성
2. TTL 설정값 확인 (Config.MaxAge)
3. TTL 절반 경과 후 세션 유효 확인 (IsExpired = false)
4. TTL 전체 경과 후 세션 만료 확인 (IsExpired = true)
5. 만료된 세션 조회 실패 확인 (자동 무효화)
6. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_2_1_1_ttl_time.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "auto_invalidated": true,
    "full_ttl_expired": true,
    "half_ttl_valid": true,
    "session_id": "iZuFU5ybnv7cKLeIniMMWw",
    "test_case": "7.2.1.1_Session_TTL_Time",
    "ttl_ms": 100
  },
  "test_name": "Test_7_2_1_1_SessionTTLTime"
}
```

**통과 기준**:

-  세션 TTL (MaxAge) 설정 가능
-  TTL 설정값 확인 가능
-  TTL 경과 전 세션 유효
-  TTL 경과 후 세션 만료
-  만료 세션 자동 무효화

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_2_1_1_SessionTTLTime
    session_test.go:737: ===== 7.2.1.1 세션 TTL 시간 확인 =====
    session_test.go:754:   세션 TTL 설정:
    session_test.go:755:     Session ID: iZuFU5ybnv7cKLeIniMMWw
    session_test.go:756:     Max Age (TTL): 100ms
    session_test.go:757:     Idle Timeout: 1h0m0s
    session_test.go:764: [PASS] TTL 설정된 세션 생성 완료
    session_test.go:765:     Created at: 2025-10-24T01:48:20+09:00
    session_test.go:766:     Expected expiry: 2025-10-24T01:48:20+09:00
    session_test.go:767:     Initial expired status: false
    session_test.go:772: [PASS] TTL 설정값 확인
    session_test.go:773:     Configured Max Age: 100ms
    session_test.go:779: [PASS] TTL 절반 경과 - 세션 유효
    session_test.go:780:     Waited: 50ms
    session_test.go:781:     Expired: false
    session_test.go:786: [PASS] TTL 만료 - 세션 무효
    session_test.go:788:     Total waited: ~121.40175ms
    session_test.go:789:     Expired: true
    session_test.go:795: [PASS] 만료된 세션 조회 실패 (자동 무효화)
    session_test.go:813:   Test data saved: testdata/verification/session/7_2_1_1_ttl_time.json
--- PASS: Test_7_2_1_1_SessionTTLTime (0.12s)
```

---

##### 7.2.1.2 세션 정보 조회 성공

**시험항목**: 세션 정보 조회 (GetSession) 및 모든 메타데이터 접근 가능성 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_2_SessionInfoRetrieval'
```

**예상 결과**:

```
=== RUN   Test_7_2_1_2_SessionInfoRetrieval
    session_test.go:818: ===== 7.2.1.2 세션 정보 조회 성공 =====
    session_test.go:830: [PASS] 세션 생성 완료
    session_test.go:837: [PASS] 세션 조회 성공
    session_test.go:872: [PASS] 모든 세션 정보 조회 가능
    session_test.go:882: [PASS] 존재하지 않는 세션 조회 처리 확인
--- PASS: Test_7_2_1_2_SessionInfoRetrieval (0.00s)
```

**검증 방법**:

1. 세션 생성 후 GetSession으로 조회
2. 조회된 세션의 모든 정보 접근 확인:
   - Session ID
   - Created At
   - Last Used At
   - Message Count
   - Is Expired
   - Config (MaxAge, IdleTimeout, MaxMessages)
3. 존재하지 않는 세션 조회 시 적절한 처리 확인
4. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_2_1_2_info_retrieval.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "created_at": "2025-10-24T01:48:20+09:00",
    "is_expired": false,
    "last_used_at": "2025-10-24T01:48:20+09:00",
    "message_count": 0,
    "retrieval_success": true,
    "session_id": "_jCZ-xG8yY8QJnCi3qINiw",
    "test_case": "7.2.1.2_Session_Info_Retrieval"
  },
  "test_name": "Test_7_2_1_2_SessionInfoRetrieval"
}
```

**통과 기준**:

-  세션 조회 성공 (GetSession)
-  세션 ID 조회 가능
-  생성 시간 조회 가능
-  마지막 사용 시간 조회 가능
-  메시지 카운트 조회 가능
-  만료 상태 조회 가능
-  세션 설정 조회 가능
-  존재하지 않는 세션 처리

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_2_1_2_SessionInfoRetrieval
    session_test.go:818: ===== 7.2.1.2 세션 정보 조회 성공 =====
    session_test.go:830: [PASS] 세션 생성 완료
    session_test.go:831:     Session ID: _jCZ-xG8yY8QJnCi3qINiw
    session_test.go:837: [PASS] 세션 조회 성공
    session_test.go:840:   조회된 세션 정보:
    session_test.go:845:     [1] ID: _jCZ-xG8yY8QJnCi3qINiw
    session_test.go:850:     [2] Created At: 2025-10-24T01:48:20+09:00
    session_test.go:855:     [3] Last Used At: 2025-10-24T01:48:20+09:00
    session_test.go:859:     [4] Message Count: 0
    session_test.go:863:     [5] Is Expired: false
    session_test.go:867:     [6] Config:
    session_test.go:868:         - Max Age: 1h0m0s
    session_test.go:869:         - Idle Timeout: 10m0s
    session_test.go:870:         - Max Messages: 1000
    session_test.go:872: [PASS] 모든 세션 정보 조회 가능
    session_test.go:877:     Manager session count: 1
    session_test.go:882: [PASS] 존재하지 않는 세션 조회 처리 확인
    session_test.go:909:   Test data saved: testdata/verification/session/7_2_1_2_info_retrieval.json
--- PASS: Test_7_2_1_2_SessionInfoRetrieval (0.00s)
```

---

##### 7.2.1.3 만료 세션 삭제

**시험항목**: 만료 세션 자동 정리 (cleanupExpiredSessions) 및 수동 삭제 (RemoveSession) 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_3_ExpiredSessionDeletion'
```

**예상 결과**:

```
=== RUN   Test_7_2_1_3_ExpiredSessionDeletion
    session_test.go:914: ===== 7.2.1.3 만료 세션 삭제 =====
    session_test.go:945: [PASS] 3개 세션 생성 완료
    session_test.go:954: [PASS] 만료 세션 정리 실행
    session_test.go:959: [PASS] 만료 세션 모두 삭제 확인
    session_test.go:968: [PASS] 모든 만료 세션 조회 불가 확인
    session_test.go:987: [PASS] 수동 삭제 성공
    session_test.go:992: [PASS] 수동 삭제된 세션 조회 불가 확인
--- PASS: Test_7_2_1_3_ExpiredSessionDeletion (0.07s)
```

**검증 방법**:

1. TTL 50ms로 3개 세션 생성
2. TTL 만료 대기
3. cleanupExpiredSessions() 실행
4. 세션 카운트 0 확인
5. 모든 만료 세션 조회 불가 확인
6. 수동 삭제 (RemoveSession) 테스트
7. 수동 삭제된 세션 조회 불가 확인
8. 검증 데이터 확인: `pkg/agent/session/testdata/verification/session/7_2_1_3_expired_deletion.json`

**검증 데이터 예시**:

```json
{
  "data": {
    "auto_cleanup_count": 3,
    "manual_deletion_success": true,
    "session_count_after_cleanup": 0,
    "test_case": "7.2.1.3_Expired_Session_Deletion"
  },
  "test_name": "Test_7_2_1_3_ExpiredSessionDeletion"
}
```

**통과 기준**:

-  만료 세션 자동 감지
-  cleanupExpiredSessions 실행
-  만료 세션 모두 삭제
-  삭제된 세션 조회 불가
-  수동 삭제 (RemoveSession) 동작

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_7_2_1_3_ExpiredSessionDeletion
    session_test.go:914: ===== 7.2.1.3 만료 세션 삭제 =====
    session_test.go:927:   만료 세션 자동 삭제 테스트:
    session_test.go:928:     TTL: 50ms
    session_test.go:940:     Session 1 created: qMzpWqpA9pD8JA4ArZrFgg
    session_test.go:940:     Session 2 created: Z9NvIIIZHgga2sadJOW5CQ
    session_test.go:940:     Session 3 created: xMHigPD91O9HzWvbfVXk-Q
    session_test.go:945: [PASS] 3개 세션 생성 완료
    session_test.go:946:     삭제 전 세션 수: 3
    session_test.go:950:     TTL 만료 대기 완료
Cleaned up 3 expired sessions
    session_test.go:954: [PASS] 만료 세션 정리 실행
    session_test.go:959: [PASS] 만료 세션 모두 삭제 확인
    session_test.go:960:     삭제 후 세션 수: 0
    session_test.go:966:     Session 1 삭제 확인: qMzpWqpA9pD8JA4ArZrFgg
    session_test.go:966:     Session 2 삭제 확인: Z9NvIIIZHgga2sadJOW5CQ
    session_test.go:966:     Session 3 삭제 확인: xMHigPD91O9HzWvbfVXk-Q
    session_test.go:968: [PASS] 모든 만료 세션 조회 불가 확인
    session_test.go:983:   수동 삭제 테스트 세션 생성: x32t1FWvYp0JF2xx4uRDPw
    session_test.go:987: [PASS] 수동 삭제 성공
    session_test.go:992: [PASS] 수동 삭제된 세션 조회 불가 확인
    session_test.go:1011:   Test data saved: testdata/verification/session/7_2_1_3_expired_deletion.json
--- PASS: Test_7_2_1_3_ExpiredSessionDeletion (0.07s)
```

---

