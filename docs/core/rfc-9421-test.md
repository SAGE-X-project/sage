### 1\. 단위 테스트 (Unit Tests)

#### 1.1. 헤더 파서 (`parser.go`)

* **`ParseSignatureInput` - 정상 케이스**

    * [ ] **Test 1.1.1 (기본 파싱)**
        * **Input**: `sig1=("@method" "host");keyid="did:key:z6Mk...";alg="ed25519";created=1719234000`
        * **Assert**: `map["sig1"]`이 존재하고, `CoveredComponents`에 `@method`, `host`가 포함되며, `KeyID`, `Algorithm`, `Created` 필드가 정확히 파싱되는지 확인.
    * [ ] **Test 1.1.2 (다중 서명 및 파라미터)**
        * **Input**: `sig-a=("@method");expires=1719237600, sig-b=("host" "date");keyid="test-key-2";nonce="abcdef"`
        * **Assert**: `map["sig-a"]`와 `map["sig-b"]`가 모두 존재하고, 각각의 파라미터(`expires`, `keyid`, `nonce`)가 정확히 파싱되는지 확인.
    * [ ] **Test 1.1.3 (공백 및 대소문자)**
        * **Input**: `sig1 = ( "@path"  "@query" ); KeyId = "test-key" ; Alg = "ecdsa-p256"`
        * **Assert**: 파라미터 이름의 대소문자가 달라도(`KeyId`, `Alg`) 정상적으로 파싱되고, 불필요한 공백이 무시되는지 확인.

* **`ParseSignature` - 정상 케이스**

    * [ ] **Test 1.2.1 (기본 파싱)**
        * **Input**: `sig1=:MEUCIQDkjN/g30k+A5U9F+a9ZcR6s5wzO8Y8Z8Y8Z8Y8Z8Y8Z8=:`
        * **Assert**: `map["sig1"]`의 값이 Base64 디코딩된 바이트 슬라이스와 일치하는지 확인.

* **파서 - 예외 케이스**

    * [ ] **Test 1.3.1 (잘못된 구조)**
        * **Input**: `sig1=("@method"`, `sig1="key=val"`
        * **Assert**: RFC 8941 형식이 아니므로 `ErrMalformedHeader` 반환.
    * [ ] **Test 1.3.2 (잘못된 Base64)**
        * **Input**: `sig1=:invalid-base64:=`
        * **Assert**: Base64 디코딩 실패 에러 반환.

-----

#### 1.2. 서명 기반 생성기 (`canonicalizer.go`)

* **Test 1.4.1 (기본 GET 요청)**
    * **Setup**: `req`, \_ := `http.NewRequest("GET", "https://example.com/foo?bar=baz", nil)`
    * **Components**: `["@method", "@authority", "@path", "@query"]`
    * **Assert**: 생성된 문자열이 정확히 다음과 같은지 확인:
      ```
      ("@method"): GET
      ("@authority"): example.com
      ("@path"): /foo
      ("@query"): ?bar=baz
      ("@signature-params"): ("@method" "@authority" "@path" "@query");...
      ```
* **Test 1.4.2 (POST 요청과 Content-Digest)**
    * **Setup**: `req` with `Body: '{"hello": "world"}'`, `Header["Content-Digest"] = "sha-256=:X48E9q...=:"`
    * **Components**: `["content-digest", "date"]`
    * **Assert**: `("content-digest"): sha-256=:X48E9q...=:` 라인이 정확히 포함되는지 확인.
* **Test 1.4.3 (헤더 값 공백 처리)**
    * **Setup**: `req.Header.Set("X-Custom", "  value with spaces  ")`
    * **Components**: `["x-custom"]` (소문자)
    * **Assert**: `("x-custom"): value with spaces` 와 같이 양쪽 공백은 제거되지만 내부 공백은 유지되는지 확인.
* **Test 1.4.4 (동일 이름의 다중 헤더)**
    * **Setup**: `req.Header.Add("Via", "1.1 proxy-a")`, `req.Header.Add("Via", "1.1 proxy-b")`
    * **Components**: `["via"]`
    * **Assert**: `("via"): 1.1 proxy-a, 1.1 proxy-b` 와 같이 쉼표와 공백으로 연결되는지 확인.
* **Test 1.4.5 (컴포넌트 부재)**
    * **Setup**: `req`에 `Content-Digest` 헤더가 없음.
    * **Components**: `["content-digest"]`
    * **Assert**: `ErrComponentNotFound` 에러 반환.

-----

### 2\. 통합 테스트 (End-to-End)

* **Test 2.1.1 (Ed25519 서명/검증)**
    * **Given**: `Ed25519` 키 쌍, `GET /resource/123?user=alice HTTP/1.1`, `Host: sage.dev`, `Date: ...`
    * **When**: `["@method", "host", "date", "@path", "@query"]` 컴포넌트로 요청에 서명.
    * **Then**: `VerifyRequest` 호출 시 에러 없이 `nil`을 반환.
* **Test 2.1.2 (ECDSA P-256 서명/검증)**
    * **Given**: `P-256` 키 쌍, `POST /data`, `Body: '{"a":1}'`, `Content-Digest` 헤더 포함.
    * **When**: `["date", "content-digest"]` 컴포넌트로 요청에 서명.
    * **Then**: `VerifyRequest` 호출 시 `nil`을 반환.

-----

### 3\. 예외 및 실패 테스트 (Negative Cases)

* **Test 3.1.1 (서명 값 1바이트 변경)**
    * **Given**: 유효하게 서명된 요청.
    * **When**: `Signature` 헤더의 Base64 값 중 마지막 글자를 변경하여 `VerifyRequest` 호출.
    * **Then**: `ErrInvalidSignature` 반환.
* **Test 3.1.2 (서명된 헤더 값 변경)**
    * **Given**: `Date` 헤더를 포함하여 유효하게 서명된 요청.
    * **When**: 검증 전 `req.Header.Set("Date", ...)`로 시간을 1초 뒤로 변경하여 `VerifyRequest` 호출.
    * **Then**: `ErrInvalidSignature` 반환.
* **Test 3.1.3 (서명되지 않은 헤더 값 변경)**
    * **Given**: `Date` 헤더는 포함했지만, `Accept` 헤더는 포함하지 않고 유효하게 서명된 요청.
    * **When**: 검증 전 `req.Header.Set("Accept", "application/xml")`로 변경하여 `VerifyRequest` 호출.
    * **Then**: **성공적으로 검증**되어야 함 (`nil` 반환).
* **Test 3.2.1 (서명 만료 - `created`와 `MaxAge`)**
    * **Given**: `created` 타임스탬프가 10분 전인 서명.
    * **When**: `VerificationOptions{MaxAge: 5 * time.Minute}`로 `VerifyRequest` 호출.
    * **Then**: `ErrSignatureExpired` 반환.
* **Test 3.2.2 (서명 만료 - `expires`)**
    * **Given**: `expires` 타임스탬프가 1분 전인 서명.
    * **When**: `VerifyRequest` 호출.
    * **Then**: `ErrSignatureExpired` 반환.

-----

### 4\. RFC 9421 고급 기능 및 엣지 케이스 테스트

* **`@query-param` 상세 테스트**
    * **Given**: Request URL `"/api/v1/users?id=123&format=json&cache=false"`
    * **Test 4.1.1 (특정 파라미터 보호)**
        * **When**: `["@query-param";name="id"]` 컴포넌트로 서명 후 검증.
        * **Then**: 성공.
    * **Test 4.1.2 (보호된 파라미터 변조)**
        * **When**: 위 요청의 URL을 `...id=456...`으로 변경 후 검증.
        * **Then**: `ErrInvalidSignature` 반환.
    * **Test 4.1.3 (보호되지 않은 파라미터 변경)**
        * **When**: 위 요청의 URL을 `...format=xml...`으로 변경 후 검증.
        * **Then**: **성공**.
    * **Test 4.1.4 (파라미터 이름의 대소문자 구분)**
        * **When**: 서명은 `["@query-param";name="id"]`로 하고, 검증 시 URL을 `...ID=123...`으로 변경.
        * **Then**: `ErrInvalidSignature` 반환 (파라미터 이름은 대소문자를 구분함).
    * **Test 4.1.5 (존재하지 않는 파라미터 서명 시도)**
        * **When**: URL에 `status` 파라미터가 없을 때, `["@query-param";name="status"]`로 서명 시도.
        * **Then**: `buildSignatureBase` 함수가 `ErrComponentNotFound` 에러 반환.
* **기타 엣지 케이스**
    * [ ] **Test 4.2.1 (빈 경로)**
        * **Given**: `GET https://example.com` (경로가 없음)
        * **When**: `["@path"]` 컴포넌트로 서명 (`/`로 처리되어야 함).
        * **Then**: 성공적으로 서명 및 검증.
    * [ ] **Test 4.2.2 (특수문자 포함 경로/쿼리)**
        * **Given**: URL `"/users/שלום?q=a%20b+c"`
        * **When**: `["@path", "@query"]` 컴포넌트로 서명.
        * **Then**: 인코딩된 상태 그대로 서명 기반에 포함되어 성공적으로 검증되는지 확인.
    * [ ] **Test 4.2.3 (프록시 요청 타겟)**
        * **Given**: `GET http://example.com/foo HTTP/1.1` (절대 경로 형식)
        * **When**: `["@request-target"]` 컴포넌트로 서명.
        * **Then**: `http://example.com/foo` 전체가 서명되어 검증되는지 확인.